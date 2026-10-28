package main

import (
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

//go:embed static/*
var staticFS embed.FS

type EmuManager struct {
	mu         sync.Mutex
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	scriptPath string
	memPath    string
	lastOutput string
	running    bool
}

func NewEmuManager(scriptPath, memPath string) *EmuManager {
	return &EmuManager{
		scriptPath: scriptPath,
		memPath:    memPath,
	}
}

func (em *EmuManager) startLocked() error {
	if em.cmd != nil && em.cmd.Process != nil {
		_ = em.cmd.Process.Kill()
		if em.stdin != nil {
			_ = em.stdin.Close()
		}
		if em.stdout != nil {
			_ = em.stdout.Close()
		}
		_ = em.cmd.Wait()
		em.running = false
	}

	var cmdArgs []string
	cmdArgs = append(cmdArgs, "-u", em.scriptPath)
	if em.memPath != "" {
		if _, err := os.Stat(em.memPath); err == nil {
			cmdArgs = append(cmdArgs, em.memPath)
		} else {
			log.Printf("[WARN] Arquivo de memória '%s' não encontrado. Usando demo padrão.", em.memPath)
		}
	}

	cmd := exec.Command("python3", cmdArgs...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("falha ao criar stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("falha ao criar stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return fmt.Errorf("falha ao iniciar python3: %w", err)
	}

	em.cmd = cmd
	em.stdin = stdin
	em.stdout = stdout
	em.running = true

	// Lê o banner inicial e tela de boas vindas até o prompt (6502-T) >
	initOut, err := em.readUntilPromptLocked(5 * time.Second)
	if err != nil {
		log.Printf("[EMU] Aviso lendo banner inicial: %v", err)
	}
	em.lastOutput = initOut
	log.Printf("[EMU] CPU 6502-T inicializada com sucesso (PID %d).", cmd.Process.Pid)
	return nil
}

func (em *EmuManager) readUntilPromptLocked(timeout time.Duration) (string, error) {
	if f, ok := em.stdout.(*os.File); ok {
		_ = f.SetReadDeadline(time.Now().Add(timeout))
	}

	var sb strings.Builder
	buf := make([]byte, 512)
	promptMarker := "(6502-T) >"

	for {
		n, err := em.stdout.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
			if strings.Contains(sb.String(), promptMarker) {
				return sb.String(), nil
			}
		}
		if err != nil {
			if sb.Len() > 0 {
				return sb.String(), err
			}
			return "", err
		}
	}
}

func (em *EmuManager) SendCommand(command string) (string, error) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if !em.running || em.cmd == nil || em.cmd.Process == nil {
		if err := em.startLocked(); err != nil {
			return "", err
		}
	}

	command = strings.TrimSpace(command)
	// Se comando for vazio, envia nova linha (avança 1 ciclo no emu6502t.py)
	toSend := command + "\n"

	if _, err := io.WriteString(em.stdin, toSend); err != nil {
		log.Printf("[EMU] Falha de escrita no stdin: %v. Reiniciando processo...", err)
		if rerr := em.startLocked(); rerr != nil {
			return "", fmt.Errorf("falha ao reiniciar emulador: %w", rerr)
		}
		if _, err2 := io.WriteString(em.stdin, toSend); err2 != nil {
			return "", fmt.Errorf("falha ao enviar comando após reinicialização: %w", err2)
		}
	}

	out, err := em.readUntilPromptLocked(6 * time.Second)
	if err != nil {
		// Se atingiu EOF, emulador foi finalizado (ex: comando 'q')
		if err == io.EOF {
			log.Printf("[EMU] Processo finalizado (EOF detectado). Reiniciando...")
			_ = em.startLocked()
			out = out + "\n[*] Processo do emulador finalizado e reiniciado automaticamente.\n" + em.lastOutput
			return out, nil
		}
		log.Printf("[EMU] Erro ao ler resposta: %v", err)
	}

	em.lastOutput = out
	return out, nil
}

func (em *EmuManager) Reset() (string, error) {
	em.mu.Lock()
	defer em.mu.Unlock()

	log.Println("[EMU] Reiniciando processo da CPU 6502-T...")
	if err := em.startLocked(); err != nil {
		return "", err
	}
	return em.lastOutput, nil
}

func (em *EmuManager) GetLastOutput() string {
	em.mu.Lock()
	defer em.mu.Unlock()
	return em.lastOutput
}

func (em *EmuManager) Kill() {
	em.mu.Lock()
	defer em.mu.Unlock()
	if em.cmd != nil && em.cmd.Process != nil {
		_ = em.cmd.Process.Kill()
		if em.stdin != nil {
			_ = em.stdin.Close()
		}
		if em.stdout != nil {
			_ = em.stdout.Close()
		}
		_ = em.cmd.Wait()
	}
	em.running = false
}

// ==============================================================================
// GERENCIADOR DE SESSÕES DINÂMICAS E ISOLAMENTO DE PROCESSOS
// ==============================================================================

var reSessionID = regexp.MustCompile(`^[a-zA-Z0-9_\-]{4,64}$`)

func generateSessionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("s_%d_%x", time.Now().Unix(), b)
}

type SessionEntry struct {
	ID         string
	Emu        *EmuManager
	LastAccess time.Time
	MemPath    string
}

type SessionManager struct {
	mu         sync.RWMutex
	sessions   map[string]*SessionEntry
	scriptPath string
	defaultMem string
	sessionDir string
	maxAge     time.Duration
}

func NewSessionManager(scriptPath, defaultMem, sessionDir string, maxAge time.Duration) *SessionManager {
	_ = os.MkdirAll(sessionDir, 0755)
	sm := &SessionManager{
		sessions:   make(map[string]*SessionEntry),
		scriptPath: scriptPath,
		defaultMem: defaultMem,
		sessionDir: sessionDir,
		maxAge:     maxAge,
	}

	// Inicializa a sessão padrão global ("default")
	defaultEntry := &SessionEntry{
		ID:         "default",
		Emu:        NewEmuManager(scriptPath, defaultMem),
		LastAccess: time.Now(),
		MemPath:    defaultMem,
	}
	if err := defaultEntry.Emu.startLocked(); err != nil {
		log.Printf("[SESSION] Aviso ao iniciar sessão padrão: %v", err)
	}
	sm.sessions["default"] = defaultEntry

	// Ticker de limpeza de processos inativos em background
	go sm.reapLoop(1 * time.Minute)
	return sm
}

func (sm *SessionManager) reapLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for id, entry := range sm.sessions {
			if id == "default" {
				continue
			}
			if now.Sub(entry.LastAccess) > sm.maxAge {
				log.Printf("[SESSION] Encerrando sessão inativa: %s (inatividade > %v)", id, sm.maxAge)
				entry.Emu.Kill()
				if entry.MemPath != "" && entry.MemPath != sm.defaultMem {
					_ = os.Remove(entry.MemPath)
				}
				delete(sm.sessions, id)
			}
		}
		sm.mu.Unlock()
	}
}

func (sm *SessionManager) GetOrCreate(r *http.Request, w http.ResponseWriter) *SessionEntry {
	var sessionID string
	if sid := r.Header.Get("X-Session-ID"); sid != "" && reSessionID.MatchString(sid) {
		sessionID = sid
	} else if c, err := r.Cookie("session_id"); err == nil && reSessionID.MatchString(c.Value) {
		sessionID = c.Value
	}

	if sessionID == "" {
		if strings.Contains(r.Header.Get("Accept"), "text/html") {
			sessionID = generateSessionID()
			if w != nil {
				http.SetCookie(w, &http.Cookie{
					Name:     "session_id",
					Value:    sessionID,
					Path:     "/",
					HttpOnly: false,
					SameSite: http.SameSiteLaxMode,
				})
			}
		} else {
			sessionID = "default"
		}
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if entry, ok := sm.sessions[sessionID]; ok {
		entry.LastAccess = time.Now()
		return entry
	}

	// Criação de nova sessão com arquivo de memória isolado
	sessionMem := filepath.Join(sm.sessionDir, fmt.Sprintf("session_%s.mem", sessionID))
	if data, err := os.ReadFile(sm.defaultMem); err == nil {
		_ = os.WriteFile(sessionMem, data, 0644)
	}

	emu := NewEmuManager(sm.scriptPath, sessionMem)
	if err := emu.startLocked(); err != nil {
		log.Printf("[SESSION] Erro ao iniciar subprocesso para sessão %s: %v", sessionID, err)
	}

	entry := &SessionEntry{
		ID:         sessionID,
		Emu:        emu,
		LastAccess: time.Now(),
		MemPath:    sessionMem,
	}
	sm.sessions[sessionID] = entry
	log.Printf("[SESSION] Nova sessão criada: %s", sessionID)
	return entry
}

// ==============================================================================
// HANDLERS HTTP
// ==============================================================================

type Server struct {
	sessions *SessionManager
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	// Garante que o cookie de sessão seja configurado
	_ = s.sessions.GetOrCreate(r, w)

	content, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Arquivo index.html não encontrado", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *Server) handleCmd(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.GetOrCreate(r, w)

	if r.Method == http.MethodGet {
		// GET retorna a última saída em buffer sem avançar a CPU
		s.respond(w, r, session.Emu.GetLastOutput())
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido (use POST)", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		http.Error(w, "Erro ao ler corpo da requisição", http.StatusBadRequest)
		return
	}

	cmdStr := strings.TrimSpace(string(body))
	if strings.HasPrefix(cmdStr, "{") {
		var req struct {
			Cmd string `json:"cmd"`
		}
		if err := json.Unmarshal(body, &req); err == nil {
			cmdStr = req.Cmd
		}
	}

	out, err := session.Emu.SendCommand(cmdStr)
	if err != nil && out == "" {
		http.Error(w, fmt.Sprintf("Erro no emulador: %v", err), http.StatusInternalServerError)
		return
	}

	s.respond(w, r, out)
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Método não permitido (use POST)", http.StatusMethodNotAllowed)
		return
	}

	session := s.sessions.GetOrCreate(r, w)
	out, err := session.Emu.Reset()
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao reiniciar CPU: %v", err), http.StatusInternalServerError)
		return
	}

	s.respond(w, r, out)
}

func (s *Server) handleVamos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido (use POST)", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		http.Error(w, "Erro ao ler código VAMOS", http.StatusBadRequest)
		return
	}

	source := string(body)
	if strings.HasPrefix(strings.TrimSpace(source), "{") {
		var req struct {
			Source string `json:"source"`
			Code   string `json:"code"`
		}
		if err := json.Unmarshal(body, &req); err == nil {
			if req.Source != "" {
				source = req.Source
			} else if req.Code != "" {
				source = req.Code
			}
		}
	}

	session := s.sessions.GetOrCreate(r, w)

	vamosScript := findFile([]string{
		"tools/vamos6502t.py",
		"../tools/vamos6502t.py",
		"/app/tools/vamos6502t.py",
	})

	outMem := session.MemPath
	if outMem == "" {
		outMem = s.sessions.defaultMem
	}
	_ = os.MkdirAll(filepath.Dir(outMem), 0755)

	cmd := exec.Command("python3", vamosScript, "--json", "-o", outMem)
	cmd.Stdin = strings.NewReader(source)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Falha de compilação VAMOS-T: %s (%v)", string(outBytes), err),
		})
		return
	}

	// Reinicia o emulador da sessão atual com o novo programa compilado
	session.Emu.mu.Lock()
	_ = session.Emu.startLocked()
	session.Emu.mu.Unlock()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(outBytes)
}

func (s *Server) respond(w http.ResponseWriter, r *http.Request, text string) {
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"output": text,
			"status": "ok",
		})
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(text))
}

// ==============================================================================
// LOCALIZAÇÃO DE ARQUIVOS E MAIN
// ==============================================================================

func findFile(candidates []string) string {
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, err := filepath.Abs(p)
			if err == nil {
				return abs
			}
			return p
		}
	}
	return candidates[0]
}

// RateLimiter armazena histórico de requisições por IP para proteção DoS
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	times := rl.requests[ip]
	var valid []time.Time
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.requests[ip] = valid
		return false
	}

	rl.requests[ip] = append(valid, now)
	return true
}

func getClientIP(r *http.Request) string {
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return strings.TrimSpace(cfIP)
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func securityMiddleware(next http.Handler, rl *RateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cabeçalhos HTTP de Segurança para exposição na Web / Cloudflare
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline' data:; connect-src 'self';")

		// Proteção por Rate Limit nas rotas de API
		if strings.HasPrefix(r.URL.Path, "/api/") {
			clientIP := getClientIP(r)
			if !rl.Allow(clientIP) {
				w.Header().Set("Retry-After", "60")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"Muitas requisições (Rate limit ativo). Aguarde alguns segundos.","success":false}`))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "27005"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	emuScript := os.Getenv("EMU_PATH")
	if emuScript == "" {
		emuScript = findFile([]string{
			"tools/emu6502t.py",
			"../tools/emu6502t.py",
			"/app/tools/emu6502t.py",
		})
	}

	memFile := os.Getenv("MEM_PATH")
	if memFile == "" {
		memFile = findFile([]string{
			"build/program.mem",
			"../build/program.mem",
			"/app/build/program.mem",
		})
	}

	log.Printf("[INIT] Iniciando 6502-T Web Frontend...")
	log.Printf("[INIT] Emulador Script: %s", emuScript)
	log.Printf("[INIT] Arquivo Memória: %s", memFile)

	sessionDir := filepath.Join(filepath.Dir(memFile), "sessions")
	sessions := NewSessionManager(emuScript, memFile, sessionDir, 15*time.Minute)
	server := &Server{sessions: sessions}

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleIndex)
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))
	mux.HandleFunc("/api/cmd", server.handleCmd)
	mux.HandleFunc("/api/reset", server.handleReset)
	mux.HandleFunc("/api/vamos", server.handleVamos)

	rateLimiter := NewRateLimiter(60, time.Minute)
	handler := securityMiddleware(mux, rateLimiter)

	log.Printf("[HTTP] Servidor ouvindo na porta %s (Tokyo Night Theme)", port)
	log.Printf("[HTTP] Interface Web disponível em: http://localhost%s", port)
	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatalf("[FATAL] Servidor HTTP finalizado com erro: %v", err)
	}
}

