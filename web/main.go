package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

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

// ==============================================================================
// HANDLERS HTTP
// ==============================================================================

type Server struct {
	emu *EmuManager
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(indexHTML))
}

func (s *Server) handleCmd(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// GET retorna a última saída em buffer sem avançar a CPU
		s.respond(w, r, s.emu.GetLastOutput())
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

	out, err := s.emu.SendCommand(cmdStr)
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

	out, err := s.emu.Reset()
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

	vamosScript := findFile([]string{
		"tools/vamos6502t.py",
		"../tools/vamos6502t.py",
		"/app/tools/vamos6502t.py",
	})

	outMem := findFile([]string{
		"build/program.mem",
		"../build/program.mem",
		"/app/build/program.mem",
	})
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

	// Reinicia o emulador com o novo programa na memória
	s.emu.mu.Lock()
	_ = s.emu.startLocked()
	s.emu.mu.Unlock()

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

	emu := NewEmuManager(emuScript, memFile)
	if err := emu.startLocked(); err != nil {
		log.Fatalf("[FATAL] Erro ao iniciar subprocesso do emulador: %v", err)
	}

	server := &Server{emu: emu}

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleIndex)
	mux.HandleFunc("/api/cmd", server.handleCmd)
	mux.HandleFunc("/api/reset", server.handleReset)
	mux.HandleFunc("/api/vamos", server.handleVamos)

	log.Printf("[HTTP] Servidor ouvindo na porta %s (Tokyo Night Theme)", port)
	log.Printf("[HTTP] Interface Web disponível em: http://localhost%s", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("[FATAL] Servidor HTTP finalizado com erro: %v", err)
	}
}

// ==============================================================================
// SPA HTML/CSS (TEMA TOKYO NIGHT)
// ==============================================================================

const indexHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>6502-T // Emulador Ternário Balanceado (Tokyo Night)</title>
  <style>
    :root {
      --bg-darker: #16161e;
      --bg-main: #1a1b26;
      --bg-card: #24283b;
      --bg-card-hover: #292e42;
      --border-color: #414868;
      --border-subtle: #292e42;
      --text-main: #c0caf5;
      --text-muted: #565f89;
      --text-secondary: #a9b1d6;
      
      --accent-cyan: #7dcfff;
      --accent-purple: #bb9af7;
      --accent-blue: #7aa2f7;
      --accent-green: #9ece6a;
      --accent-red: #f7768e;
      --accent-orange: #ff9e64;
      --accent-yellow: #e0af68;
      
      --font-mono: ui-monospace, 'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'SF Mono', Menlo, Consolas, monospace;
    }

    * {
      box-sizing: border-box;
      margin: 0;
      padding: 0;
    }

    body {
      background-color: var(--bg-main);
      color: var(--text-main);
      font-family: var(--font-mono);
      min-height: 100vh;
      display: flex;
      flex-direction: column;
      line-height: 1.5;
    }

    /* Top Navigation */
    header {
      background: var(--bg-darker);
      border-bottom: 1px solid var(--border-color);
      padding: 12px 24px;
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: 12px;
    }

    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .brand-logo {
      width: 32px;
      height: 32px;
      border-radius: 6px;
      background: linear-gradient(135deg, var(--accent-purple), var(--accent-cyan));
      display: flex;
      align-items: center;
      justify-content: center;
      font-weight: 900;
      color: var(--bg-darker);
      font-size: 16px;
      box-shadow: 0 0 10px rgba(125, 207, 255, 0.4);
    }

    .brand-title {
      font-size: 18px;
      font-weight: 700;
      color: var(--accent-cyan);
      letter-spacing: 0.5px;
    }

    .brand-subtitle {
      font-size: 12px;
      color: var(--accent-purple);
      margin-left: 6px;
    }

    .header-badges {
      display: flex;
      align-items: center;
      gap: 10px;
      font-size: 12px;
    }

    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 4px 10px;
      border-radius: 20px;
      border: 1px solid var(--border-color);
      background: var(--bg-card);
      color: var(--text-secondary);
    }

    .badge-status {
      border-color: var(--accent-green);
      color: var(--accent-green);
    }

    .status-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background-color: var(--accent-green);
      box-shadow: 0 0 8px var(--accent-green);
      animation: pulse 2s infinite ease-in-out;
    }

    @keyframes pulse {
      0%, 100% { opacity: 1; transform: scale(1); }
      50% { opacity: 0.4; transform: scale(0.85); }
    }

    /* Main Container */
    main {
      flex: 1;
      max-width: 1300px;
      width: 100%;
      margin: 0 auto;
      padding: 20px 24px;
      display: flex;
      flex-direction: column;
      gap: 16px;
    }

    /* Telemetry Bar */
    .telemetry-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
      gap: 10px;
    }

    .stat-card {
      background: var(--bg-card);
      border: 1px solid var(--border-subtle);
      border-radius: 8px;
      padding: 8px 12px;
      display: flex;
      flex-direction: column;
      transition: border-color 0.2s;
    }

    .stat-card:hover {
      border-color: var(--border-color);
    }

    .stat-label {
      font-size: 10px;
      text-transform: uppercase;
      letter-spacing: 0.5px;
      color: var(--text-muted);
    }

    .stat-value {
      font-size: 14px;
      font-weight: 700;
      color: var(--accent-cyan);
      margin-top: 2px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .stat-value.purple { color: var(--accent-purple); }
    .stat-value.green  { color: var(--accent-green); }
    .stat-value.yellow { color: var(--accent-yellow); }
    .stat-value.orange { color: var(--accent-orange); }

    /* Control Toolbar */
    .toolbar-card {
      background: var(--bg-card);
      border: 1px solid var(--border-color);
      border-radius: 10px;
      padding: 12px 16px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      flex-wrap: wrap;
      gap: 12px;
    }

    .btn-group {
      display: flex;
      align-items: center;
      flex-wrap: wrap;
      gap: 8px;
    }

    .btn {
      font-family: var(--font-mono);
      font-size: 12px;
      font-weight: 600;
      padding: 8px 14px;
      border-radius: 6px;
      border: 1px solid transparent;
      cursor: pointer;
      display: inline-flex;
      align-items: center;
      gap: 6px;
      transition: all 0.15s ease-in-out;
      user-select: none;
    }

    .btn-step {
      background: rgba(125, 207, 255, 0.15);
      border-color: var(--accent-cyan);
      color: var(--accent-cyan);
    }
    .btn-step:hover {
      background: var(--accent-cyan);
      color: var(--bg-darker);
      box-shadow: 0 0 12px rgba(125, 207, 255, 0.4);
    }

    .btn-inst {
      background: rgba(187, 154, 247, 0.15);
      border-color: var(--accent-purple);
      color: var(--accent-purple);
    }
    .btn-inst:hover {
      background: var(--accent-purple);
      color: var(--bg-darker);
      box-shadow: 0 0 12px rgba(187, 154, 247, 0.4);
    }

    .btn-run {
      background: rgba(158, 206, 106, 0.15);
      border-color: var(--accent-green);
      color: var(--accent-green);
    }
    .btn-run:hover {
      background: var(--accent-green);
      color: var(--bg-darker);
      box-shadow: 0 0 12px rgba(158, 206, 106, 0.4);
    }

    .btn-dump {
      background: rgba(224, 175, 104, 0.15);
      border-color: var(--accent-yellow);
      color: var(--accent-yellow);
    }
    .btn-dump:hover {
      background: var(--accent-yellow);
      color: var(--bg-darker);
      box-shadow: 0 0 12px rgba(224, 175, 104, 0.4);
    }

    .btn-reset {
      background: rgba(247, 118, 142, 0.15);
      border-color: var(--accent-red);
      color: var(--accent-red);
    }
    .btn-reset:hover {
      background: var(--accent-red);
      color: var(--bg-darker);
      box-shadow: 0 0 12px rgba(247, 118, 142, 0.4);
    }

    .btn-secondary {
      background: var(--bg-darker);
      border-color: var(--border-color);
      color: var(--text-secondary);
    }
    .btn-secondary:hover {
      border-color: var(--accent-cyan);
      color: var(--accent-cyan);
    }

    .btn-kbd {
      background: rgba(255, 255, 255, 0.08);
      font-size: 10px;
      padding: 2px 5px;
      border-radius: 3px;
      margin-left: 2px;
      border: 1px solid rgba(255, 255, 255, 0.1);
    }

    /* Terminal Container */
    .terminal-window {
      background: var(--bg-darker);
      border: 1px solid var(--border-color);
      border-radius: 10px;
      display: flex;
      flex-direction: column;
      flex: 1;
      min-height: 480px;
      overflow: hidden;
      box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    }

    .terminal-header {
      background: #13141c;
      padding: 10px 16px;
      border-bottom: 1px solid var(--border-subtle);
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .terminal-dots {
      display: flex;
      gap: 6px;
      align-items: center;
    }

    .terminal-dot {
      width: 10px;
      height: 10px;
      border-radius: 50%;
    }
    .dot-red    { background-color: var(--accent-red); }
    .dot-yellow { background-color: var(--accent-yellow); }
    .dot-green  { background-color: var(--accent-green); }

    .terminal-title {
      font-size: 12px;
      color: var(--accent-purple);
      font-weight: 500;
    }

    .terminal-actions {
      display: flex;
      gap: 12px;
      font-size: 11px;
      color: var(--text-muted);
    }

    .terminal-link {
      cursor: pointer;
      color: var(--text-muted);
      transition: color 0.15s;
    }
    .terminal-link:hover {
      color: var(--accent-cyan);
      text-decoration: underline;
    }

    .terminal-body {
      flex: 1;
      padding: 16px;
      overflow-y: auto;
      font-size: 13px;
      color: var(--text-main);
      white-space: pre-wrap;
      word-break: break-all;
      background: var(--bg-darker);
      scrollbar-width: thin;
      scrollbar-color: var(--border-color) var(--bg-darker);
    }

    .terminal-body::-webkit-scrollbar {
      width: 8px;
    }
    .terminal-body::-webkit-scrollbar-track {
      background: var(--bg-darker);
    }
    .terminal-body::-webkit-scrollbar-thumb {
      background: var(--border-color);
      border-radius: 4px;
    }

    /* Terminal Input Form */
    .terminal-footer {
      background: #181924;
      border-top: 1px solid var(--border-subtle);
      padding: 10px 16px;
      display: flex;
      align-items: center;
      gap: 10px;
    }

    .prompt-label {
      color: var(--accent-cyan);
      font-weight: 700;
      font-size: 13px;
      white-space: nowrap;
      user-select: none;
    }

    .cmd-input {
      flex: 1;
      background: transparent;
      border: none;
      outline: none;
      font-family: var(--font-mono);
      font-size: 13px;
      color: var(--text-main);
      caret-color: var(--accent-cyan);
    }

    .cmd-input::placeholder {
      color: var(--text-muted);
      font-size: 12px;
    }

    /* Footer info */
    footer {
      background: var(--bg-darker);
      border-top: 1px solid var(--border-subtle);
      padding: 10px 24px;
      font-size: 11px;
      color: var(--text-muted);
      display: flex;
      justify-content: space-between;
      flex-wrap: wrap;
      gap: 10px;
    }

    .footer-left {
      display: flex;
      gap: 16px;
    }

    .footer-highlight {
      color: var(--accent-cyan);
    }

    /* Tabs Navigation */
    .nav-tabs {
      display: flex;
      gap: 10px;
      padding: 0 24px;
      background: var(--bg-darker);
      border-bottom: 1px solid var(--border-color);
    }
    .nav-tab {
      background: transparent;
      border: none;
      border-bottom: 2px solid transparent;
      color: var(--text-muted);
      padding: 12px 18px;
      font-family: var(--font-mono);
      font-size: 13px;
      font-weight: 600;
      cursor: pointer;
      display: inline-flex;
      align-items: center;
      gap: 8px;
      transition: all 0.2s;
    }
    .nav-tab:hover {
      color: var(--text-main);
    }
    .nav-tab.active {
      color: var(--accent-cyan);
      border-bottom-color: var(--accent-cyan);
    }

    /* VAMOS-T Playground Layout */
    .vamos-container {
      display: grid;
      grid-template-columns: 1.1fr 0.9fr;
      gap: 16px;
      margin-top: 16px;
    }
    @media (max-width: 1024px) {
      .vamos-container {
        grid-template-columns: 1fr;
      }
    }
    .vamos-pane {
      background: var(--bg-card);
      border: 1px solid var(--border-color);
      border-radius: 8px;
      display: flex;
      flex-direction: column;
      overflow: hidden;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
    }
    .vamos-pane-header {
      background: var(--bg-darker);
      padding: 10px 16px;
      border-bottom: 1px solid var(--border-color);
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: 10px;
    }
    .vamos-pane-title {
      font-size: 13px;
      font-weight: 700;
      color: var(--accent-purple);
      display: flex;
      align-items: center;
      gap: 6px;
    }
    .vamos-select {
      background: var(--bg-card);
      color: var(--text-main);
      border: 1px solid var(--border-color);
      border-radius: 4px;
      padding: 4px 8px;
      font-family: var(--font-mono);
      font-size: 11px;
      outline: none;
    }
    .vamos-textarea {
      flex: 1;
      width: 100%;
      min-height: 380px;
      background: #14151e;
      color: var(--accent-cyan);
      border: none;
      padding: 16px;
      font-family: var(--font-mono);
      font-size: 13px;
      line-height: 1.6;
      resize: vertical;
      outline: none;
      scrollbar-width: thin;
      scrollbar-color: var(--border-color) #14151e;
    }
    .vamos-output-pre {
      min-height: 180px;
      max-height: 240px;
      background: #14151e;
      color: var(--accent-green);
      padding: 12px 16px;
      font-family: var(--font-mono);
      font-size: 12px;
      overflow-y: auto;
      white-space: pre;
      border-bottom: 1px solid var(--border-subtle);
      scrollbar-width: thin;
      scrollbar-color: var(--border-color) #14151e;
    }
    .vamos-vars-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 12px;
      text-align: left;
    }
    .vamos-vars-table th {
      background: var(--bg-darker);
      padding: 6px 12px;
      color: var(--accent-cyan);
      border-bottom: 1px solid var(--border-color);
      font-weight: 600;
    }
    .vamos-vars-table td {
      padding: 6px 12px;
      border-bottom: 1px solid var(--border-subtle);
      color: var(--text-main);
    }
    .vamos-vars-table tr:hover td {
      background: var(--bg-card-hover);
    }

    /* Virtual TTY CRT Console Screen */
    .tty-screen {
      background: #090a10;
      border: 1px solid #1f2335;
      border-radius: 6px;
      padding: 12px 14px;
      font-family: var(--font-mono);
      font-size: 13px;
      color: var(--accent-green);
      min-height: 80px;
      max-height: 120px;
      overflow-y: auto;
      white-space: pre-wrap;
      box-shadow: inset 0 0 16px rgba(0, 0, 0, 0.8);
      position: relative;
      line-height: 1.5;
    }
    .tty-cursor {
      display: inline-block;
      width: 8px;
      height: 14px;
      background: var(--accent-green);
      vertical-align: middle;
      animation: blink 1s step-end infinite;
      box-shadow: 0 0 6px var(--accent-green);
    }
    @keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }

    /* 6-Trit LED Bus Meter */
    .trit-meter {
      display: flex;
      gap: 8px;
      align-items: center;
      justify-content: center;
      padding: 8px 12px;
      background: var(--bg-darker);
      border-radius: 6px;
      border: 1px solid var(--border-subtle);
    }
    .trit-led {
      width: 38px;
      height: 38px;
      border-radius: 6px;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      font-size: 11px;
      font-weight: 700;
      transition: all 0.15s ease-in-out;
      border: 1px solid transparent;
      user-select: none;
    }
    .trit-led.pos {
      background: rgba(125, 207, 255, 0.2);
      border-color: var(--accent-cyan);
      color: var(--accent-cyan);
      box-shadow: 0 0 10px rgba(125, 207, 255, 0.5);
    }
    .trit-led.zer {
      background: rgba(41, 46, 66, 0.5);
      border-color: var(--border-color);
      color: var(--text-muted);
    }
    .trit-led.neg {
      background: rgba(247, 118, 142, 0.2);
      border-color: var(--accent-red);
      color: var(--accent-red);
      box-shadow: 0 0 10px rgba(247, 118, 142, 0.5);
    }
    .trit-label {
      font-size: 8px;
      color: var(--text-muted);
      margin-top: 1px;
    }

    /* Execution Player Toolbar */
    .player-toolbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 8px;
      background: #141520;
      padding: 8px 14px;
      border-bottom: 1px solid var(--border-color);
      flex-wrap: wrap;
    }
    .player-slider {
      flex: 1;
      height: 6px;
      accent-color: var(--accent-cyan);
      cursor: pointer;
      min-width: 120px;
    }
    .btn-player {
      background: var(--bg-card);
      border: 1px solid var(--border-color);
      color: var(--text-main);
      padding: 4px 8px;
      border-radius: 4px;
      font-size: 11px;
      font-family: var(--font-mono);
      cursor: pointer;
      display: inline-flex;
      align-items: center;
      gap: 4px;
      transition: all 0.15s;
    }
    .btn-player:hover {
      border-color: var(--accent-cyan);
      color: var(--accent-cyan);
    }
    .btn-player.active {
      background: var(--accent-cyan);
      color: var(--bg-darker);
      font-weight: 700;
    }

    /* Helper snippet button */
    .btn-snippet {
      background: rgba(255, 255, 255, 0.05);
      border: 1px solid var(--border-subtle);
      color: var(--text-muted);
      padding: 3px 8px;
      border-radius: 4px;
      font-size: 10px;
      font-family: var(--font-mono);
      cursor: pointer;
      transition: all 0.15s;
    }
    .btn-snippet:hover {
      color: var(--accent-cyan);
      border-color: var(--accent-cyan);
      background: rgba(125, 207, 255, 0.1);
    }

    .asm-line {
      display: block;
      padding: 1px 6px;
      border-radius: 3px;
      line-height: 1.4;
    }
    .asm-line.active {
      background: rgba(125, 207, 255, 0.25);
      color: #ffffff;
      font-weight: 700;
      border-left: 3px solid var(--accent-cyan);
    }

    /* Manual & Educational Guide Tab */
    .manual-hero {
      background: linear-gradient(135deg, rgba(187, 154, 247, 0.12), rgba(125, 207, 255, 0.08));
      border: 1px solid var(--border-color);
      border-radius: 8px;
      padding: 18px 24px;
      display: flex;
      flex-direction: column;
      gap: 6px;
    }
    .manual-hero-title {
      font-size: 18px;
      font-weight: 800;
      color: var(--accent-cyan);
      display: flex;
      align-items: center;
      gap: 10px;
    }
    .manual-hero-desc {
      font-size: 13px;
      color: var(--text-secondary);
      line-height: 1.6;
    }
    .manual-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
      gap: 16px;
      margin-top: 14px;
    }
    .manual-card {
      background: var(--bg-card);
      border: 1px solid var(--border-color);
      border-radius: 8px;
      padding: 16px 18px;
      display: flex;
      flex-direction: column;
      gap: 10px;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
    }
    .manual-card-title {
      font-size: 14px;
      font-weight: 700;
      color: var(--accent-purple);
      display: flex;
      align-items: center;
      justify-content: space-between;
      border-bottom: 1px solid var(--border-subtle);
      padding-bottom: 8px;
    }
    .manual-card p, .manual-card li {
      font-size: 12px;
      color: var(--text-secondary);
      line-height: 1.6;
    }
    .manual-card code {
      background: var(--bg-darker);
      color: var(--accent-purple);
      padding: 2px 5px;
      border-radius: 3px;
      font-size: 11px;
      border: 1px solid var(--border-subtle);
    }
    .manual-code-block {
      background: #12131c;
      border: 1px solid var(--border-subtle);
      border-radius: 6px;
      padding: 10px 12px;
      font-family: var(--font-mono);
      font-size: 11.5px;
      color: var(--accent-green);
      overflow-x: auto;
      white-space: pre;
      line-height: 1.5;
    }
    .manual-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 11px;
      margin-top: 6px;
    }
    .manual-table th {
      background: var(--bg-darker);
      padding: 5px 8px;
      color: var(--accent-cyan);
      border-bottom: 1px solid var(--border-color);
      text-align: left;
    }
    .manual-table td {
      padding: 5px 8px;
      border-bottom: 1px solid var(--border-subtle);
      color: var(--text-main);
    }
  </style>
</head>
<body>

  <!-- Top Navigation Header -->
  <header>
    <div class="brand">
      <div class="brand-logo">3T</div>
      <div>
        <span class="brand-title">6502-T</span>
        <span class="brand-subtitle">Ternary Microprocessor (Base 12)</span>
      </div>
    </div>
    <div class="header-badges">
      <div class="badge badge-status">
        <span class="status-dot"></span>
        <span id="conn-status">CPU ONLINE</span>
      </div>
      <div class="badge">
        <span>Port: <strong class="footer-highlight">27005</strong></span>
      </div>
      <div class="badge">
        <span>Theme: <strong style="color:var(--accent-purple)">Tokyo Night</strong></span>
      </div>
    </div>
  </header>

  <!-- Navigation Tabs -->
  <nav class="nav-tabs">
    <button class="nav-tab active" id="tab-vamos">🧪 Laboratório VAMOS-T <span style="font-size: 10px; background: var(--accent-purple); color: var(--bg-darker); padding: 1px 6px; border-radius: 10px; font-weight: 700;">VISUAL</span></button>
    <button class="nav-tab" id="tab-manual">📖 Manual &amp; Exemplos</button>
    <button class="nav-tab" id="tab-terminal">💻 Terminal &amp; Telemetria</button>
  </nav>

  <!-- Main Content Area -->
  <main>
    <!-- Tab 1: Terminal & Telemetry View -->
    <div id="view-terminal" style="display: none;">
      <!-- Live Telemetry Quick Cards -->
      <div class="telemetry-grid">
        <div class="stat-card">
          <span class="stat-label">Ciclo Clock</span>
          <span class="stat-value" id="stat-cycle">0</span>
        </div>
        <div class="stat-card">
          <span class="stat-label">Estado FSM</span>
          <span class="stat-value purple" id="stat-state">FETCH_OPCODE</span>
        </div>
        <div class="stat-card">
          <span class="stat-label">Prog. Counter (PC)</span>
          <span class="stat-value" id="stat-pc">$0000</span>
        </div>
        <div class="stat-card">
          <span class="stat-label">Acumulador (A)</span>
          <span class="stat-value green" id="stat-acc">+0 (0z000)</span>
        </div>
        <div class="stat-card">
          <span class="stat-label">Registrador X</span>
          <span class="stat-value yellow" id="stat-regx">+0 (0z000)</span>
        </div>
        <div class="stat-card">
          <span class="stat-label">Registrador Y</span>
          <span class="stat-value orange" id="stat-regy">+0 (0z000)</span>
        </div>
        <div class="stat-card">
          <span class="stat-label">Flags [Z | S | O]</span>
          <span class="stat-value" id="stat-flags">[0 | 0 | 0]</span>
        </div>
      </div>

      <!-- Quick Control Buttons Toolbar -->
      <div class="toolbar-card" style="margin-top: 14px;">
        <div class="btn-group">
          <button class="btn btn-step" id="btn-step" title="Avança 1 ciclo de clock (s)">
            <span>▶ Step (Ciclo)</span>
            <span class="btn-kbd">Space</span>
          </button>
          <button class="btn btn-inst" id="btn-inst" title="Avança até a próxima instrução (i)">
            <span>⏩ Instrução</span>
            <span class="btn-kbd">I</span>
          </button>
          <button class="btn btn-run" id="btn-run" title="Executa em velocidade contínua (c)">
            <span>⚡ Run (Cont)</span>
            <span class="btn-kbd">C</span>
          </button>
          <button class="btn btn-dump" id="btn-dump" title="Exibe o conteúdo da memória RAM (m)">
            <span>📋 Dump RAM</span>
            <span class="btn-kbd">M</span>
          </button>
          <button class="btn btn-reset" id="btn-reset" title="Reinicia o processo da CPU (POST /api/reset)">
            <span>🔄 Reset CPU</span>
            <span class="btn-kbd">R</span>
          </button>
        </div>

        <div class="btn-group">
          <button class="btn btn-secondary" id="btn-clear" title="Limpar histórico do terminal">
            <span>🧹 Limpar</span>
          </button>
          <button class="btn btn-secondary" id="btn-copy" title="Copiar saída do terminal">
            <span>📄 Copiar</span>
          </button>
        </div>
      </div>

      <!-- Interactive Monospace Terminal Window -->
      <div class="terminal-window" style="margin-top: 14px;">
        <div class="terminal-header">
          <div class="terminal-dots">
            <div class="terminal-dot dot-red"></div>
            <div class="terminal-dot dot-yellow"></div>
            <div class="terminal-dot dot-green"></div>
          </div>
          <div class="terminal-title">emu6502t.py (Sessão Interativa - build/program.mem)</div>
          <div class="terminal-actions">
            <span class="terminal-link" id="link-autoscroll">Auto-scroll: ON</span>
          </div>
        </div>

        <div class="terminal-body" id="term-output">Carregando emulador ternário 6502-T...</div>

        <form class="terminal-footer" id="terminal-form" autocomplete="off">
          <span class="prompt-label">(6502-T) &gt;</span>
          <input 
            type="text" 
            id="cmd-input" 
            class="cmd-input" 
            placeholder="Digite um comando (s, i, c, m [end] [len], b [end], r, q) e pressione Enter..."
            autofocus
          />
          <button type="submit" class="btn btn-step" style="padding: 4px 10px; font-size: 11px;">Enviar</button>
        </form>
      </div>
    </div>

    <!-- Tab 2: VAMOS-LANG Playground View -->
    <div id="view-vamos" style="display: block;">
      <div class="toolbar-card">
        <div class="btn-group">
          <button class="btn btn-step" id="btn-vamos-run" title="Compilar VAMOS-LANG e executar na CPU 6502-T">
            <span>⚡ Compilar Código</span>
            <span class="btn-kbd">Ctrl+Enter</span>
          </button>
          <select id="vamos-presets" class="vamos-select">
            <option value="ola">🌟 Exemplo 1: Olá Mundo! (Texto &amp; TTY MMIO)</option>
            <option value="calc">🧮 Exemplo 2: Calculadora Duodecimal (Base 12)</option>
            <option value="trit">💠 Exemplo 3: Trit Balanceado &amp; Negação Simétrica</option>
            <option value="loop">🔄 Exemplo 4: Loop &amp; Acúmulo (Gauss 1..5)</option>
            <option value="cond">⚖️ Exemplo 5: Flags &amp; Condicionais Ternários</option>
            <option value="olamindo">👾 Exemplo 6: olá mindo! (Solicitação literal)</option>
          </select>
        </div>
        <div class="header-badges">
          <span class="badge" id="vamos-compile-status">Pronto para compilar</span>
        </div>
      </div>

      <div class="vamos-container">
        <!-- Editor Pane -->
        <div class="vamos-pane">
          <div class="vamos-pane-header">
            <span class="vamos-pane-title">📝 Editor VAMOS-LANG (facrf/VAMOSLANG)</span>
            <span style="font-size: 11px; color: var(--text-muted);">Sintaxe Go em Português</span>
          </div>
          <textarea id="vamos-code" class="vamos-textarea" spellcheck="false" placeholder="Digite seu código VAMOS-LANG aqui..."></textarea>

          <!-- Quick Code Snippets Toolbar -->
          <div style="background: var(--bg-darker); padding: 8px 12px; border-top: 1px solid var(--border-color); display: flex; gap: 6px; flex-wrap: wrap; align-items: center;">
            <span style="font-size: 10px; color: var(--text-muted); text-transform: uppercase;">Inserir:</span>
            <button type="button" class="btn-snippet" data-snip="escrever(&quot;Olá Mundo!\n&quot;)">escrever("...")</button>
            <button type="button" class="btn-snippet" data-snip="se x &lt; 0 {&#10;    A = -1&#10;} senao {&#10;    A = 1&#10;}">se ... senao</button>
            <button type="button" class="btn-snippet" data-snip="para i &lt; 5 {&#10;    total = total + i&#10;    i = i + 1&#10;}">para ...</button>
            <button type="button" class="btn-snippet" data-snip="x := 0z011 // +13">0z (Base 12)</button>
            <button type="button" class="btn-snippet" data-snip="parar()">parar()</button>
            <span style="flex:1;"></span>
            <button type="button" class="btn-snippet" id="btn-vamos-clear" style="color:var(--accent-red);">Limpar</button>
          </div>
        </div>

        <!-- Output & Visual State Pane -->
        <div class="vamos-pane">
          <!-- Interactive Execution Player Toolbar -->
          <div class="player-toolbar">
            <div style="display:flex; align-items:center; gap:6px;">
              <button class="btn-player" id="v-btn-play" title="Reproduzir execução passo a passo animada">
                <span id="v-play-icon">▶</span> <span id="v-play-text">Animar</span>
              </button>
              <button class="btn-player" id="v-btn-prev" title="Passo anterior">⏮</button>
              <button class="btn-player" id="v-btn-next" title="Próximo passo">⏭</button>
              <button class="btn-player" id="v-btn-start" title="Reiniciar do primeiro passo">🔄 Início</button>
              <button class="btn-player" id="v-btn-end" title="Ir para o final da execução">⚡ Fim</button>
            </div>
            <div class="player-progress">
              <span id="v-step-label" style="font-size: 11px; color: var(--accent-cyan); white-space: nowrap;">Passo 0 / 0</span>
              <input type="range" id="v-step-slider" min="0" max="0" value="0" class="player-slider" title="Arraste para navegar no tempo de execução" />
              <select id="v-speed-select" class="vamos-select" style="font-size:10px;">
                <option value="150">150ms</option>
                <option value="50">50ms (Rápido)</option>
                <option value="350">350ms (Lento)</option>
              </select>
            </div>
          </div>

          <!-- Virtual TTY CRT Console Screen -->
          <div class="vamos-pane-header">
            <span class="vamos-pane-title" style="color:var(--accent-green);">🟢 Saída TTY MMIO (Console Port 300 / 0z210)</span>
            <span id="v-tty-badge" class="badge" style="font-size: 10px; color: var(--accent-green);">0 chars</span>
          </div>
          <div style="padding: 10px 14px; background: var(--bg-darker);">
            <div class="tty-screen" id="v-tty-console">
              <span id="v-tty-text">&gt; Clique em "Compilar Código" para iniciar...</span><span class="tty-cursor"></span>
            </div>
          </div>

          <!-- 6-Trit LED Visualizer Bus -->
          <div class="vamos-pane-header">
            <span class="vamos-pane-title">💠 Barramento de Dados (6 Trits Balanceados do Acumulador)</span>
            <span id="v-trit-value" class="badge" style="font-size: 10px;">+0 (0z000)</span>
          </div>
          <div style="padding: 8px 14px; background: var(--bg-darker);">
            <div class="trit-meter">
              <div class="trit-led zer" id="trit-5"><span class="trit-val">0</span><span class="trit-label">T5 (243)</span></div>
              <div class="trit-led zer" id="trit-4"><span class="trit-val">0</span><span class="trit-label">T4 (81)</span></div>
              <div class="trit-led zer" id="trit-3"><span class="trit-val">0</span><span class="trit-label">T3 (27)</span></div>
              <div class="trit-led zer" id="trit-2"><span class="trit-val">0</span><span class="trit-label">T2 (9)</span></div>
              <div class="trit-led zer" id="trit-1"><span class="trit-val">0</span><span class="trit-label">T1 (3)</span></div>
              <div class="trit-led zer" id="trit-0"><span class="trit-val">0</span><span class="trit-label">T0 (1)</span></div>
            </div>
          </div>

          <!-- Registers & Flags Grid -->
          <div style="padding: 10px; display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; background: var(--bg-darker); border-top: 1px solid var(--border-color);">
            <div class="stat-card">
              <span class="stat-label">Ciclo Atual</span>
              <span class="stat-value" id="v-stat-cycles">0</span>
            </div>
            <div class="stat-card">
              <span class="stat-label">Prog. Counter (PC)</span>
              <span class="stat-value" id="v-stat-pc">$0000</span>
            </div>
            <div class="stat-card">
              <span class="stat-label">Acumulador (A)</span>
              <span class="stat-value green" id="v-stat-acc">+0</span>
            </div>
            <div class="stat-card">
              <span class="stat-label">Registrador X</span>
              <span class="stat-value yellow" id="v-stat-regx">+0</span>
            </div>
            <div class="stat-card">
              <span class="stat-label">Registrador Y</span>
              <span class="stat-value orange" id="v-stat-regy">+0</span>
            </div>
            <div class="stat-card">
              <span class="stat-label">Flags [Z|S|O]</span>
              <span class="stat-value" id="v-stat-flags">[0 | 0 | 0]</span>
            </div>
          </div>

          <!-- Assembly 6502-T Inspector with Highlighting -->
          <div class="vamos-pane-header" style="border-top: 1px solid var(--border-color);">
            <span class="vamos-pane-title">⚙️ Assembly 6502-T Gerado &amp; Rastreamento</span>
            <span id="vamos-asm-badge" class="badge" style="font-size: 10px;">0 palavras</span>
          </div>
          <div id="vamos-asm-output" class="vamos-output-pre">; O código assembly compilado aparecerá aqui...</div>

          <!-- Variables in RAM Table -->
          <div class="vamos-pane-header" style="border-top: 1px solid var(--border-color);">
            <span class="vamos-pane-title">📦 Variáveis Alocadas na RAM</span>
          </div>
          <div style="max-height: 140px; overflow-y: auto;">
            <table class="vamos-vars-table">
              <thead>
                <tr>
                  <th>Variável</th>
                  <th>Endereço RAM</th>
                  <th>Decimal</th>
                  <th>Duodecimal</th>
                  <th>Trits (6T)</th>
                </tr>
              </thead>
              <tbody id="vamos-vars-tbody">
                <tr><td colspan="5" style="text-align: center; color: var(--text-muted); padding: 10px;">Nenhuma variável declarada</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- Tab 3: Manual & Exemplos Didáticos (VAMOS-LANG) -->
    <div id="view-manual" style="display: none;">
      <!-- Hero Banner -->
      <div class="manual-hero">
        <div class="manual-hero-title">
          <span>📚</span> Manual Rápido VAMOS-LANG // CPU 6502-T
        </div>
        <div class="manual-hero-desc">
          Guia introdutório para cursos e laboratórios: aprenda como funcionam a <strong>Lógica Ternária Balanceada {-1, 0, +1}</strong> e o <strong>Sistema Duodecimal (Base 12)</strong> no clássico microprocessador MOS 6502 reinventado.
        </div>
      </div>

      <!-- Fundamentos Teóricos (3 Cards) -->
      <div class="manual-grid">
        <!-- Card 1: O que é o Trit -->
        <div class="manual-card">
          <div class="manual-card-title">
            <span>💠 O que é o Trit Balanceado?</span>
            <span style="font-size: 11px; color: var(--accent-cyan);">{-1, 0, +1}</span>
          </div>
          <p>
            Diferente do <em>bit</em> binário ({0, 1}), o <strong>Trit</strong> possui 3 estados físicos comutados por transistores NPN/PNP em push-pull:
          </p>
          <ul style="padding-left: 18px; display: flex; flex-direction: column; gap: 4px;">
            <li><strong>+1 (POS):</strong> +5V (Transistor NPN saturado)</li>
            <li><strong> 0 (ZER):</strong>  0V (Ambos transistores em corte)</li>
            <li><strong>-1 (NEG):</strong> -5V (Transistor PNP saturado)</li>
          </ul>
          <p>
            No barramento de dados de <strong>6 trits</strong>, cada posição possui peso exponencial de potências de 3:
          </p>
          <table class="manual-table">
            <thead>
              <tr><th>Trit</th><th>T5</th><th>T4</th><th>T3</th><th>T2</th><th>T1</th><th>T0</th></tr>
            </thead>
            <tbody>
              <tr><td>Peso</td><td>243</td><td>81</td><td>27</td><td>9</td><td>3</td><td>1</td></tr>
            </tbody>
          </table>
          <p style="font-size: 11px; color: var(--accent-green); margin-top: 4px;">
            ★ <strong>Vantagem Revolucionária:</strong> A negação simétrica é direta: para inverter o sinal de um número, basta inverter cada trit (+ vira -, - vira +). Não há necessidade de bit de sinal nem complemento de dois assimétrico!
          </p>
        </div>

        <!-- Card 2: Aritmética Duodecimal (Base 12) -->
        <div class="manual-card">
          <div class="manual-card-title">
            <span>🔢 A Mágica da Base 12 (0z...)</span>
            <span style="font-size: 11px; color: var(--accent-purple);">Radix 12</span>
          </div>
          <p>
            Para facilitar a leitura humana da telemetria e das instruções, o 6502-T utiliza numeração <strong>Duodecimal (Base 12)</strong> com prefixo <code>0z</code>.
          </p>
          <p>
            O número 12 possui quatro divisores limpos {2, 3, 4, 6}, gerando frações muito mais naturais que a base 10. Seus algarismos são:
            <code>0, 1, 2, 3, 4, 5, 6, 7, 8, 9, A (=10), B (=11)</code>.
          </p>
          <table class="manual-table">
            <thead>
              <tr><th>Duodecimal</th><th>Cálculo</th><th>Decimal</th></tr>
            </thead>
            <tbody>
              <tr><td><code>0z00A</code></td><td>10</td><td>+10</td></tr>
              <tr><td><code>0z00B</code></td><td>11</td><td>+11</td></tr>
              <tr><td><code>0z010</code></td><td>1×12 + 0</td><td>+12</td></tr>
              <tr><td><code>0z011</code></td><td>1×12 + 1</td><td>+13</td></tr>
              <tr><td><code>0z042</code></td><td>4×12 + 2</td><td>+50</td></tr>
              <tr><td><code>0z0BB</code></td><td>11×12 + 11</td><td>+143</td></tr>
              <tr><td><code>0z110</code></td><td>1×144 + 1×12 + 0</td><td>+156</td></tr>
            </tbody>
          </table>
        </div>

        <!-- Card 3: Sintaxe VAMOS-LANG -->
        <div class="manual-card">
          <div class="manual-card-title">
            <span>⚡ Sintaxe Essencial de VAMOS</span>
            <span style="font-size: 11px; color: var(--accent-yellow);">Go em Português</span>
          </div>
          <p>
            VAMOS-LANG é uma linguagem simplificada estilo Go com palavras-chave em Português, transpilada diretamente para Assembly 6502-T:
          </p>
          <ul style="padding-left: 18px; display: flex; flex-direction: column; gap: 4px;">
            <li><code>pacote principal</code> / <code>funcao principal() { ... }</code></li>
            <li><code>a := 0z010</code> ou <code>var x = 5</code> (alocadas na RAM)</li>
            <li><code>soma := a + b</code> e <code>sub := a - b</code></li>
            <li><code>se cond { ... } senao { ... }</code> (desvios condicionais)</li>
            <li><code>para cond { ... }</code> (laços de repetição)</li>
            <li><code>A = resultado</code> e <code>X = A</code> (registradores)</li>
            <li><code>escrever("Olá!\n")</code> (MMIO porta 300 / 0z210)</li>
            <li><code>parar()</code> (termina o programa)</li>
          </ul>
        </div>
      </div>

      <!-- Galeria de Exemplos Práticos Interativos -->
      <div style="margin-top: 20px;">
        <h3 style="font-size: 16px; color: var(--accent-cyan); display: flex; align-items: center; gap: 8px;">
          <span>🧪</span> Exemplos Didáticos Prontos para Testar
        </h3>
        <p style="font-size: 12px; color: var(--text-muted); margin-top: 2px;">
          Clique em "▶ Carregar no Laboratório" para enviar o código diretamente ao editor, compilar e assistir à execução ciclo a ciclo.
        </p>

        <div class="manual-grid" style="margin-top: 12px;">
          <!-- Exemplo 1: Calculadora Duodecimal -->
          <div class="manual-card">
            <div class="manual-card-title">
              <span>🧮 1. Calculadora Duodecimal (Base 12)</span>
              <button class="btn btn-step btn-load-code" data-preset="calc">▶ Carregar no Laboratório</button>
            </div>
            <p>Realiza operações aritméticas diretamente usando numeração duodecimal <code>0z</code> e reflete o resultado no Acumulador e na tela CRT.</p>
            <div class="manual-code-block">num1 := 0z010       // +12 decimal
num2 := 0z005       // +5 decimal
soma := num1 + num2 // +17 decimal (0z015)
resultado := soma - 0z008 // +9 decimal</div>
          </div>

          <!-- Exemplo 2: Trit Balanceado e Inversão de Sinal -->
          <div class="manual-card">
            <div class="manual-card-title">
              <span>💠 2. O Poder do Trit (Inversão Simétrica)</span>
              <button class="btn btn-step btn-load-code" data-preset="trit">▶ Carregar no Laboratório</button>
            </div>
            <p>Inverte o sinal de um número positivo para negativo sem complicação matemática. Observe a iluminação dos LEDs rosa/vermelho (-1) no barramento!</p>
            <div class="manual-code-block">valor := 0z042       // +50 decimal
invertido := 0 - valor // -50 decimal (Simetria pura!)
A = invertido</div>
          </div>

          <!-- Exemplo 3: Contador e Loop de Gauss -->
          <div class="manual-card">
            <div class="manual-card-title">
              <span>🔄 3. Contador &amp; Somatório de Gauss (1..5)</span>
              <button class="btn btn-step btn-load-code" data-preset="loop">▶ Carregar no Laboratório</button>
            </div>
            <p>Utiliza a instrução de repetição <code>para i &lt; limite</code> para iterar 5 vezes, somando valores progressivamente na RAM.</p>
            <div class="manual-code-block">para i &lt; limite {
    total = total + i
    i = i + 1
}</div>
          </div>

          <!-- Exemplo 4: Tomada de Decisão com se / senao -->
          <div class="manual-card">
            <div class="manual-card-title">
              <span>⚖️ 4. Decisão &amp; Flags Ternárias (se / senao)</span>
              <button class="btn btn-step btn-load-code" data-preset="cond">▶ Carregar no Laboratório</button>
            </div>
            <p>Testa o sinal do número com desvio condicional ternário, ativando as flags de Status Zero (Z) e Sign (S).</p>
            <div class="manual-code-block">se alvo &lt; 0 {
    status = -1
} senao {
    status = 1
}</div>
          </div>

          <!-- Exemplo 5: Olá Mundo no Console CRT TTY -->
          <div class="manual-card">
            <div class="manual-card-title">
              <span>🌟 5. Olá Mundo! (Console CRT MMIO)</span>
              <button class="btn btn-step btn-load-code" data-preset="ola">▶ Carregar no Laboratório</button>
            </div>
            <p>Escreve uma cadeia de caracteres na porta de terminal mapeada em memória (MMIO porta 300 / 0z210), simulando um terminal CRT com fósforo verde.</p>
            <div class="manual-code-block">escrever("Olá Mundo!\n")
parar()</div>
          </div>
        </div>
      </div>
    </div>
  </main>

  <!-- Technical Footer -->
  <footer>
    <div class="footer-left">
      <span>Barramento Dados: <strong class="footer-highlight">6 trits</strong> [-364, +364]</span>
      <span>Barramento Endereço: <strong class="footer-highlight">10 trits</strong> (59.049 palavras)</span>
      <span>Aritmética: <strong style="color:var(--accent-purple)">Duodecimal (Base 12)</strong></span>
    </div>
    <div>
      <span>MOS 6502 Ternário Balanceado &copy; Fabiano Augusto Cesar da Rocha Filho</span>
    </div>
  </footer>

  <script>
    (function() {
      const termOutput = document.getElementById('term-output');
      const cmdInput = document.getElementById('cmd-input');
      const termForm = document.getElementById('terminal-form');
      const connStatus = document.getElementById('conn-status');
      const linkAutoscroll = document.getElementById('link-autoscroll');

      // Telemetry elements
      const elCycle = document.getElementById('stat-cycle');
      const elState = document.getElementById('stat-state');
      const elPc = document.getElementById('stat-pc');
      const elAcc = document.getElementById('stat-acc');
      const elRegX = document.getElementById('stat-regx');
      const elRegY = document.getElementById('stat-regy');
      const elFlags = document.getElementById('stat-flags');

      let autoscroll = true;
      let cmdHistory = [];
      let historyIndex = -1;

      linkAutoscroll.addEventListener('click', () => {
        autoscroll = !autoscroll;
        linkAutoscroll.textContent = 'Auto-scroll: ' + (autoscroll ? 'ON' : 'OFF');
      });

      function parseTelemetry(text) {
        if (!text) return;

        // CICLO: 0 | FSM: FETCH_OPCODE
        const cycleMatch = text.match(/CICLO:\s*(\d+)/i);
        if (cycleMatch) elCycle.textContent = cycleMatch[1];

        const fsmMatch = text.match(/FSM:\s*(\w+)/i);
        if (fsmMatch) elState.textContent = fsmMatch[1];

        // Prog. Counter : [0000000000] $0000 0z000
        const pcMatch = text.match(/Prog\.\s*Counter\s*:\s*\[[^\]]+\]\s*(\S+)\s*(\S+)/i);
        if (pcMatch) elPc.textContent = pcMatch[1] + ' (' + pcMatch[2] + ')';

        // Acumulador (A) : [000000] +0 0z000
        const accMatch = text.match(/Acumulador\s*\(A\)\s*:\s*\[[^\]]+\]\s*([+\-]?\d+)\s*(\S+)/i);
        if (accMatch) elAcc.textContent = accMatch[1] + ' (' + accMatch[2] + ')';

        // Índice X : [000000] +0 0z000
        const xMatch = text.match(/Índice\s*X\s*:\s*\[[^\]]+\]\s*([+\-]?\d+)\s*(\S+)/i);
        if (xMatch) elRegX.textContent = xMatch[1] + ' (' + xMatch[2] + ')';

        // Índice Y : [000000] +0 0z000
        const yMatch = text.match(/Índice\s*Y\s*:\s*\[[^\]]+\]\s*([+\-]?\d+)\s*(\S+)/i);
        if (yMatch) elRegY.textContent = yMatch[1] + ' (' + yMatch[2] + ')';

        // FLAGS TERNÁRIAS: [Zero (Z): 0] [Sinal (S): 0] [Overflow (O): 0]
        const zMatch = text.match(/Zero\s*\(Z\):\s*([+\-]?\d+)/i);
        const sMatch = text.match(/Sinal\s*\(S\):\s*([+\-0])/i);
        const oMatch = text.match(/Overflow\s*\(O\):\s*([+\-]?\d+)/i);
        if (zMatch && sMatch && oMatch) {
          elFlags.textContent = '[' + zMatch[1] + ' | ' + sMatch[1] + ' | ' + oMatch[1] + ']';
        }
      }

      function updateTerminal(text, append) {
        if (append) {
          termOutput.textContent += text;
        } else {
          termOutput.textContent = text;
        }
        parseTelemetry(text);
        if (autoscroll) {
          termOutput.scrollTop = termOutput.scrollHeight;
        }
      }

      async function sendCommand(cmd) {
        try {
          connStatus.textContent = 'EXECUTANDO...';
          const res = await fetch('/api/cmd', {
            method: 'POST',
            headers: { 'Content-Type': 'text/plain' },
            body: cmd
          });
          if (!res.ok) {
            throw new Error('HTTP ' + res.status + ': ' + (await res.text()));
          }
          const text = await res.text();
          updateTerminal(text, false);
          connStatus.textContent = 'CPU ONLINE';
        } catch (err) {
          updateTerminal('\n[ERRO DE CONEXÃO]: ' + err.message + '\n', true);
          connStatus.textContent = 'ERRO';
        }
      }

      async function resetCPU() {
        try {
          connStatus.textContent = 'REINICIANDO...';
          const res = await fetch('/api/reset', { method: 'POST' });
          if (!res.ok) {
            throw new Error('HTTP ' + res.status + ': ' + (await res.text()));
          }
          const text = await res.text();
          updateTerminal(text, false);
          connStatus.textContent = 'CPU ONLINE';
        } catch (err) {
          updateTerminal('\n[ERRO AO REINICIAR]: ' + err.message + '\n', true);
          connStatus.textContent = 'ERRO';
        }
      }

      // Initial Fetch
      async function init() {
        try {
          const res = await fetch('/api/cmd');
          if (res.ok) {
            const text = await res.text();
            updateTerminal(text, false);
          }
        } catch (e) {
          console.warn('Erro ao carregar estado inicial:', e);
        }
      }
      init();

      // Form Submit
      termForm.addEventListener('submit', (e) => {
        e.preventDefault();
        const cmd = cmdInput.value.trim();
        cmdInput.value = '';
        if (cmd) {
          cmdHistory.push(cmd);
          historyIndex = cmdHistory.length;
        }
        sendCommand(cmd);
      });

      // History navigation (Up / Down)
      cmdInput.addEventListener('keydown', (e) => {
        if (e.key === 'ArrowUp') {
          if (cmdHistory.length > 0 && historyIndex > 0) {
            historyIndex--;
            cmdInput.value = cmdHistory[historyIndex];
          }
          e.preventDefault();
        } else if (e.key === 'ArrowDown') {
          if (historyIndex < cmdHistory.length - 1) {
            historyIndex++;
            cmdInput.value = cmdHistory[historyIndex];
          } else {
            historyIndex = cmdHistory.length;
            cmdInput.value = '';
          }
          e.preventDefault();
        }
      });

      // Quick Control Buttons
      document.getElementById('btn-step').addEventListener('click', () => sendCommand('s'));
      document.getElementById('btn-inst').addEventListener('click', () => sendCommand('i'));
      document.getElementById('btn-run').addEventListener('click', () => sendCommand('c'));
      document.getElementById('btn-dump').addEventListener('click', () => sendCommand('m 0 32'));
      document.getElementById('btn-reset').addEventListener('click', () => resetCPU());

      document.getElementById('btn-clear').addEventListener('click', () => {
        termOutput.textContent = '';
      });

      document.getElementById('btn-copy').addEventListener('click', () => {
        navigator.clipboard.writeText(termOutput.textContent).then(() => {
          const original = document.getElementById('btn-copy').textContent;
          document.getElementById('btn-copy').textContent = '✔ Copiado!';
          setTimeout(() => {
            document.getElementById('btn-copy').textContent = original;
          }, 1500);
        });
      });

      // Global hotkeys when input is not focused
      window.addEventListener('keydown', (e) => {
        if (document.activeElement === cmdInput || document.activeElement === document.getElementById('vamos-code')) return;
        if (e.ctrlKey || e.metaKey || e.altKey) return;

        const key = e.key.toLowerCase();
        if (key === ' ' || key === 's') {
          e.preventDefault();
          sendCommand('s');
        } else if (key === 'i') {
          e.preventDefault();
          sendCommand('i');
        } else if (key === 'c') {
          e.preventDefault();
          sendCommand('c');
        } else if (key === 'm') {
          e.preventDefault();
          sendCommand('m 0 32');
        } else if (key === 'r') {
          e.preventDefault();
          resetCPU();
        }
      });

      // Tab Navigation
      const tabVamos = document.getElementById('tab-vamos');
      const tabManual = document.getElementById('tab-manual');
      const tabTerminal = document.getElementById('tab-terminal');
      const viewVamos = document.getElementById('view-vamos');
      const viewManual = document.getElementById('view-manual');
      const viewTerminal = document.getElementById('view-terminal');

      function switchTab(target) {
        tabVamos.classList.toggle('active', target === 'vamos');
        tabManual.classList.toggle('active', target === 'manual');
        tabTerminal.classList.toggle('active', target === 'terminal');

        viewVamos.style.display = (target === 'vamos') ? 'block' : 'none';
        viewManual.style.display = (target === 'manual') ? 'block' : 'none';
        viewTerminal.style.display = (target === 'terminal') ? 'block' : 'none';
      }

      tabVamos.addEventListener('click', () => switchTab('vamos'));
      tabManual.addEventListener('click', () => switchTab('manual'));
      tabTerminal.addEventListener('click', () => switchTab('terminal'));

      // VAMOS-LANG Playground & Visualizer
      const vamosCode = document.getElementById('vamos-code');
      const vamosPresets = document.getElementById('vamos-presets');
      const btnVamosRun = document.getElementById('btn-vamos-run');
      const vamosCompileStatus = document.getElementById('vamos-compile-status');
      const vamosAsmOutput = document.getElementById('vamos-asm-output');
      const vamosAsmBadge = document.getElementById('vamos-asm-badge');
      const vStatCycles = document.getElementById('v-stat-cycles');
      const vStatPc = document.getElementById('v-stat-pc');
      const vStatAcc = document.getElementById('v-stat-acc');
      const vStatRegx = document.getElementById('v-stat-regx');
      const vStatRegy = document.getElementById('v-stat-regy');
      const vStatFlags = document.getElementById('v-stat-flags');
      const vamosVarsTbody = document.getElementById('vamos-vars-tbody');

      const vBtnPlay = document.getElementById('v-btn-play');
      const vPlayIcon = document.getElementById('v-play-icon');
      const vPlayText = document.getElementById('v-play-text');
      const vBtnPrev = document.getElementById('v-btn-prev');
      const vBtnNext = document.getElementById('v-btn-next');
      const vBtnStart = document.getElementById('v-btn-start');
      const vBtnEnd = document.getElementById('v-btn-end');
      const vStepLabel = document.getElementById('v-step-label');
      const vStepSlider = document.getElementById('v-step-slider');
      const vSpeedSelect = document.getElementById('v-speed-select');
      const vTtyText = document.getElementById('v-tty-text');
      const vTtyBadge = document.getElementById('v-tty-badge');
      const vTritValue = document.getElementById('v-trit-value');

      const presets = {
        ola: "pacote principal\n\n// Exemplo VAMOS-T: Olá Mundo na CPU Ternária 6502-T\n// Emite cada caractere via MMIO (Porta 300 / 0z210) no Barramento de 6 Trits\nfuncao principal() {\n    escrever(\"Olá Mundo!\\n\")\n    parar()\n}",
        calc: "pacote principal\n\n// Exemplo VAMOS-T: Calculadora Duodecimal (Base 12) no 6502-T\n// Demonstra soma e subtração com literais duodecimais 0z...\nfuncao principal() {\n    // 0z010 = 12 em decimal, 0z005 = 5 em decimal\n    num1 := 0z010\n    num2 := 0z005\n\n    // Soma: 12 + 5 = 17 (em base 12: 0z015)\n    soma := num1 + num2\n\n    // Subtração: 17 - 8 = 9 (0z009)\n    resultado := soma - 0z008\n\n    A = resultado\n    escrever(\"Calc Base 12: OK!\\n\")\n    parar()\n}",
        trit: "pacote principal\n\n// Exemplo VAMOS-T: Lógica Ternária Balanceada {-1, 0, +1}\n// Demonstra negação unária simétrica e flags de sinal\nfuncao principal() {\n    // Carrega um valor positivo (+50 decimal / 0z042)\n    valor := 0z042\n\n    // Inversão simétrica sem complemento de dois!\n    invertido := 0 - valor\n\n    A = invertido\n    escrever(\"Trit Balanceado Simetrico!\\n\")\n    parar()\n}",
        loop: "pacote principal\n\n// Exemplo VAMOS-T: Loop de Contagem e Acúmulo no 6502-T\nfuncao principal() {\n    limite := 5\n    total := 0\n    i := 0\n\n    para i < limite {\n        total = total + i\n        i = i + 1\n    }\n\n    A = total\n    parar()\n}",
        cond: "pacote principal\n\n// Exemplo VAMOS-T: Lógica Condicional e Flags Ternárias\nfuncao principal() {\n    alvo := -12\n    status := 0\n\n    se alvo < 0 {\n        status = -1\n    } senao {\n        status = 1\n    }\n\n    A = status\n    parar()\n}",
        olamindo: "pacote principal\n\n// Exemplo VAMOS-T: olá mindo! (Solicitação literal)\nfuncao principal() {\n    escrever(\"olá mindo!\\n\")\n    parar()\n}"
      };

      // Load code from manual cards
      document.querySelectorAll('.btn-load-code').forEach(btn => {
        btn.addEventListener('click', () => {
          const pName = btn.getAttribute('data-preset');
          if (pName && presets[pName]) {
            vamosPresets.value = pName;
            vamosCode.value = presets[pName];
            switchTab('vamos');
            window.scrollTo({ top: 0, behavior: 'smooth' });
            runVamos(true);
          }
        });
      });

      vamosPresets.addEventListener('change', () => {
        const val = vamosPresets.value;
        if (presets[val]) {
          vamosCode.value = presets[val];
          runVamos(false);
        }
      });
      // Set initial code to "ola"
      vamosCode.value = presets.ola;

      // Snippet helpers
      document.querySelectorAll('.btn-snippet').forEach(btn => {
        btn.addEventListener('click', () => {
          const snip = btn.getAttribute('data-snip');
          if (snip) {
            const code = snip.replace(/\\n/g, '\n');
            const start = vamosCode.selectionStart;
            const end = vamosCode.selectionEnd;
            vamosCode.value = vamosCode.value.substring(0, start) + code + vamosCode.value.substring(end);
            vamosCode.focus();
            vamosCode.selectionStart = vamosCode.selectionEnd = start + code.length;
          }
        });
      });
      const btnClear = document.getElementById('btn-vamos-clear');
      if (btnClear) {
        btnClear.addEventListener('click', () => {
          vamosCode.value = '';
          vamosCode.focus();
        });
      }

      // Step & Timeline Execution State
      let currentTrace = [];
      let currentStepIndex = 0;
      let playTimer = null;

      function renderStep(idx) {
        if (!currentTrace || currentTrace.length === 0) return;
        if (idx < 0) idx = 0;
        if (idx >= currentTrace.length) idx = currentTrace.length - 1;
        currentStepIndex = idx;

        const step = currentTrace[idx];
        vStepSlider.value = idx;
        vStepLabel.textContent = 'Passo ' + (idx + 1) + ' / ' + currentTrace.length + ' (Ciclo ' + step.cycle + ')';

        vStatCycles.textContent = step.cycle;
        vStatPc.textContent = '$' + (step.pc || 0).toString(16).padStart(4, '0');
        vStatAcc.textContent = (step.acc >= 0 ? '+' : '') + step.acc + ' (' + (step.acc_dozenal || '0z000') + ')';
        vStatRegx.textContent = (step.reg_x >= 0 ? '+' : '') + step.reg_x;
        vStatRegy.textContent = (step.reg_y >= 0 ? '+' : '') + step.reg_y;
        vStatFlags.textContent = '[' + (step.flag_zero ? 1 : 0) + ' | ' + (step.flag_sign || 0) + ' | ' + (step.flag_overflow ? 1 : 0) + ']';

        // TTY Console
        if (step.stdout && step.stdout.length > 0) {
          vTtyText.textContent = '> ' + step.stdout;
          vTtyBadge.textContent = step.stdout.length + ' chars';
        } else {
          vTtyText.textContent = '> [Aguardando saída TTY MMIO...]';
          vTtyBadge.textContent = '0 chars';
        }

        // 6-Trit LEDs
        const trits = step.acc_trits || [0, 0, 0, 0, 0, 0];
        vTritValue.textContent = (step.acc >= 0 ? '+' : '') + step.acc + ' (' + (step.acc_dozenal || '0z000') + ')';
        for (let i = 0; i < 6; i++) {
          const led = document.getElementById('trit-' + i);
          if (led) {
            const val = trits[i] || 0;
            led.className = 'trit-led ' + (val > 0 ? 'pos' : (val < 0 ? 'neg' : 'zer'));
            const valSpan = led.querySelector('.trit-val');
            if (valSpan) valSpan.textContent = (val > 0 ? '+1' : (val < 0 ? '-1' : '0'));
          }
        }

        // Assembly highlighting
        const lines = vamosAsmOutput.querySelectorAll('.asm-line');
        lines.forEach(lineEl => {
          const lpc = parseInt(lineEl.getAttribute('data-pc'), 10);
          if (lpc === step.pc) {
            lineEl.classList.add('active');
            lineEl.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
          } else {
            lineEl.classList.remove('active');
          }
        });
      }

      function stopPlay() {
        if (playTimer) {
          clearInterval(playTimer);
          playTimer = null;
          vBtnPlay.classList.remove('active');
          vPlayIcon.textContent = '▶';
          vPlayText.textContent = 'Animar';
        }
      }

      function startPlay() {
        stopPlay();
        if (!currentTrace || currentTrace.length === 0) return;
        if (currentStepIndex >= currentTrace.length - 1) {
          currentStepIndex = 0;
          renderStep(0);
        }
        vBtnPlay.classList.add('active');
        vPlayIcon.textContent = '⏸';
        vPlayText.textContent = 'Pausar';

        const speed = parseInt(vSpeedSelect.value, 10) || 150;
        playTimer = setInterval(() => {
          if (currentStepIndex < currentTrace.length - 1) {
            renderStep(currentStepIndex + 1);
          } else {
            stopPlay();
          }
        }, speed);
      }

      vBtnPlay.addEventListener('click', () => {
        if (playTimer) stopPlay();
        else startPlay();
      });
      vBtnPrev.addEventListener('click', () => {
        stopPlay();
        renderStep(currentStepIndex - 1);
      });
      vBtnNext.addEventListener('click', () => {
        stopPlay();
        renderStep(currentStepIndex + 1);
      });
      vBtnStart.addEventListener('click', () => {
        stopPlay();
        renderStep(0);
      });
      vBtnEnd.addEventListener('click', () => {
        stopPlay();
        renderStep(currentTrace.length - 1);
      });
      vStepSlider.addEventListener('input', () => {
        stopPlay();
        renderStep(parseInt(vStepSlider.value, 10));
      });

      function renderAssemblyWithPC(asmText) {
        let pc = 0;
        const lines = asmText.split('\n');
        let html = '';
        lines.forEach((line) => {
          const trimmed = line.trim();
          let linePC = pc;
          let isInst = false;
          if (trimmed && !trimmed.startsWith('//') && !trimmed.startsWith(';') && !trimmed.endsWith(':')) {
            isInst = true;
            const parts = trimmed.split(/\s+/);
            if (parts.length > 1 && !parts[1].startsWith('//') && !parts[1].startsWith(';')) {
              pc += 2;
            } else {
              pc += 1;
            }
          }
          const escaped = line.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
          html += '<span class="asm-line" data-pc="' + (isInst ? linePC : -1) + '">' + escaped + '</span>';
        });
        vamosAsmOutput.innerHTML = html;
      }

      async function runVamos(autoPlay) {
        const code = vamosCode.value.trim();
        if (!code) return;

        stopPlay();
        vamosCompileStatus.textContent = 'COMPILANDO...';
        vamosCompileStatus.style.borderColor = 'var(--accent-yellow)';
        vamosCompileStatus.style.color = 'var(--accent-yellow)';

        try {
          const res = await fetch('/api/vamos', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ source: code })
          });
          const data = await res.json();

          if (!res.ok || !data.success) {
            const err = data.error || 'Erro na compilação';
            vamosCompileStatus.textContent = 'ERRO';
            vamosCompileStatus.style.borderColor = 'var(--accent-red)';
            vamosCompileStatus.style.color = 'var(--accent-red)';
            vamosAsmOutput.textContent = '; [ERRO DE COMPILAÇÃO]\n; ' + err;
            vamosAsmBadge.textContent = 'Erro';
            return;
          }

          vamosCompileStatus.textContent = 'SUCESSO';
          vamosCompileStatus.style.borderColor = 'var(--accent-green)';
          vamosCompileStatus.style.color = 'var(--accent-green)';

          renderAssemblyWithPC(data.assembly || '; Sem código gerado');
          vamosAsmBadge.textContent = (data.words_count || 0) + ' palavras';

          const ex = data.execution;
          if (ex) {
            currentTrace = ex.trace || [];
            vStepSlider.min = 0;
            vStepSlider.max = Math.max(0, currentTrace.length - 1);
            vStepSlider.value = 0;

            // Variables table
            const vars = ex.variables || {};
            const keys = Object.keys(vars);
            if (keys.length === 0) {
              vamosVarsTbody.innerHTML = '<tr><td colspan="5" style="text-align: center; color: var(--text-muted); padding: 10px;">Nenhuma variável declarada</td></tr>';
            } else {
              let rowsHtml = '';
              for (const k of keys) {
                const item = vars[k];
                const decSign = item.decimal >= 0 ? '+' : '';
                rowsHtml += '<tr>' +
                  '<td><strong style="color:var(--accent-cyan);">' + k + '</strong></td>' +
                  '<td>$' + item.address.toString(16).padStart(4, '0') + ' (' + item.address + ')</td>' +
                  '<td>' + decSign + item.decimal + '</td>' +
                  '<td style="color:var(--accent-purple);">' + item.dozenal + '</td>' +
                  '<td style="color:var(--accent-cyan);">' + (item.trits || '') + '</td>' +
                '</tr>';
              }
              vamosVarsTbody.innerHTML = rowsHtml;
            }

            if (autoPlay) {
              startPlay();
            } else {
              renderStep(currentTrace.length - 1);
            }
          }
        } catch (err) {
          vamosCompileStatus.textContent = 'ERRO HTTP';
          vamosCompileStatus.style.borderColor = 'var(--accent-red)';
          vamosCompileStatus.style.color = 'var(--accent-red)';
          vamosAsmOutput.textContent = '; [ERRO HTTP]\n; ' + err.message;
        }
      }

      btnVamosRun.addEventListener('click', () => runVamos(true));

      vamosCode.addEventListener('keydown', (e) => {
        if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
          e.preventDefault();
          runVamos(true);
        }
      });

      // Auto-compile default preset on startup
      runVamos(false);
    })();
  </script>
</body>
</html>
`
