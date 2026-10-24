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

  <!-- Main Content Area -->
  <main>
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
    <div class="toolbar-card">
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
    <div class="terminal-window">
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
        const sMatch = text.match(/Sinal\s*\(S\):\s*([+\-]?\d+)/i);
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
        if (document.activeElement === cmdInput) return;
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
    })();
  </script>
</body>
</html>
`
