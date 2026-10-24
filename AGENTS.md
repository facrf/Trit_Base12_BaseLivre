# AGENTS.md - Diretrizes para Modelos de Linguagem e Agentes Autônomos

> **Repositório:** `6502-T` (`Trit_Base12_BaseLivre`)  
> **Arquitetura:** Microprocessador MOS 6502 modificado para Lógica Ternária Balanceada $\{-1, 0, +1\}$ e Abstração Aritmética Duodecimal (Base 12).  
> **Mantenedor:** Fabiano Augusto Cesar da Rocha Filho e Colaboradores  
> **Licença:** GNU General Public License v3.0 (GPLv3)

---

## 1. Missão e Contexto Essencial

Ao atuar neste repositório, o agente deve assumir o papel de **Engenheiro Sênior de Hardware e Sistemas Embarcados**. O projeto **6502-T** não é uma CPU binária convencional. Todo o pipeline computacional obedece a regras físico-matemáticas específicas:

1. **Lógica Ternária Balanceada:**
   * Unidade básica: **Trit** $\in \{-1, 0, +1\}$ (denotado abreviadamente por $\{-, 0, +\}$).
   * Mapeamento elétrico físico: $+5\text{ V}$ ($+1$), $0\text{ V}$ ($0$), $-5\text{ V}$ ($-1$) via chaveamento de transistores bipolares push-pull (NPN/PNP).
   * Emulação/Verilog: cada trit é transportado em **2 bits binários**:
     * `2'b00` = Trit $-1$ (`TRIT_NEG` / `-`)
     * `2'b01` = Trit $0$ (`TRIT_ZER` / `0`)
     * `2'b10` = Trit $+1$ (`TRIT_POS` / `+`)
     * `2'b11` = *Inválido / Trap state* (não deve ocorrer em condições operacionais normais).

2. **Abstração Aritmética Duodecimal (Base 12):**
   * Usada na notação de instruções assembly e telemetria para leitura humana.
   * Prefixo literal: `0z` (ex.: `0z0BB` = $+143$, `0z011` = $+13$, `0z110` = $+156$).
   * Algarismos duodecimais: `0, 1, 2, 3, 4, 5, 6, 7, 8, 9, A (=10), B (=11)`.

3. **Geometria de Barramentos:**
   * **Barramento de Dados:** 6 trits balanceados ($12\text{ bits}$ em Verilog/emulação). Faixa inteira: $[-364, +364]$ ($3^6 = 729$ valores possíveis).
   * **Barramento de Endereço:** 10 trits ($20\text{ bits}$ em Verilog). Espaço endereçável linear: $3^{10} = \mathbf{59.049}$ posições de memória ($0$ a $59.048$).
   * **Flags de Status Balanceadas:**
     * `Zero (Z)`: booleano ($1$ quando o valor é zero, $0$ caso contrário).
     * `Sign (S)`: trit balanceado ($-1$ negativo, $0$ nulo, $+1$ positivo).
     * `Overflow (O)`: booleano ($1$ se a operação na ALU extrapolou $[-364, +364]$).

---

## 2. Estrutura de Diretórios e Papel dos Módulos

```text
Trit_Base12_BaseLivre/
├── constr/                      # Restrições físicas para FPGA Digilent Basys 3
│   └── basys3_6502t.xdc         # Mapeamento de pinos e clocks para Artix-7
├── rtl/                         # 10 módulos Verilog sintetizáveis (silício)
│   ├── alu_6trit.v              # Unidade Lógica e Aritmética ternária balanceada
│   ├── button_debounce.v        # Debouncer de botões para FPGA
│   ├── cpu_6502t_debugger.v     # Controlador de debug e interface do usuário
│   ├── cpu_6502t_top.v          # Top-level da CPU MOS 6502-T
│   ├── display_controller_4digit.v # Varredura multiplexada de displays de 7 segmentos
│   ├── dozenal_7seg_decoder.v   # Decodificador de base 12 para 7 segmentos
│   ├── fpga_test_top.v          # Top-level para sintetização em FPGA Basys 3
│   ├── instruction_decoder_6502t.v # Decodificador de instruções e FSM síncrona
│   ├── tfa_1trit.v              # Ternary Full Adder de 1 trit balanceado
│   └── trits_to_dozenal.v       # Conversor físico de 6 trits para 2 dígitos duodecimais
├── sim/                         # 7 testbenches para simulação Icarus Verilog
│   ├── tb_alu_6trit.v           # Verificação da ALU e detecção de overflow
│   ├── tb_cpu_6502t_system.v    # Simulação integrada da CPU executando programa
│   ├── tb_display_controller.v  # Testbench dos controladores de display
│   ├── tb_dozenal_display.v     # Verificação dos decodificadores de display
│   ├── tb_instruction_decoder_6502t.v # Verificação do decodificador de instruções
│   ├── tb_system_dozenal_monitor.v # Monitor de barramento e telemetria duodecimal
│   └── tb_tfa_1trit.v           # Verificação da tabela de verdade do somador
├── docs/                        # Documentação técnica e especificações completas
│   ├── base_conceitual.md       # Fundamentos da física do trit e constante de Euler (e)
│   ├── especificacao_completa.md# Especificação formal da CPU, ISA e RTL
│   ├── manual_emulador.md       # Manual de operação do emulador ciclo a ciclo
│   ├── simulacao.md             # Guia de simulação de hardware
│   ├── tese_antitese.md         # Confronto técnico binário vs. ternário
│   └── assets/                  # Esquemas e diagramas
├── tools/                       # Toolchain CLI em Python
│   ├── asm6502t.py              # Montador com suporte a NOP, BNE, .ORG e .WORD
│   ├── disasm6502t.py           # Desassemblador com modo limpo (-c) e suporte a @addr
│   ├── emu6502t.py              # Emulador ciclo a ciclo interativo (TYA, BNE, dump 64p)
│   └── vamos6502t.py            # Compilador VAMOS-LANG para 6502-T (CLI, JSON e trace)
├── tests/                       # Testes automatizados de regressão
│   ├── test_6502t_suite.py      # Suite ponta a ponta obrigatória (7/7 PASS)
│   └── test_6502t_extended.py   # Suite estendida e reversibilidade round-trip
├── scripts/                     # Scripts de automação EDA
│   └── build_bitstream.tcl      # Síntese em lote para Xilinx Vivado (FPGA Artix-7)
├── src_asm/                     # Programas Assembly de demonstração
│   └── program.s                # Código fonte de exemplo
├── web/                         # Frontend Web em Go puro (Tema Tokyo Night)
│   ├── go.mod                   # Módulo Go
│   └── main.go                  # Servidor HTTP na porta 27005, Terminal e Laboratório VAMOS-T
├── Dockerfile.web               # Multi-stage Docker (golang:alpine -> python:3.12-alpine)
├── Makefile                     # Orquestrador padrão de comandos
├── AGENTS.md                    # Este guia operacional para LLMs
├── CITATION.cff                 # Citação acadêmica formal
├── LICENSE                      # Licença GPLv3
└── README.md                    # Documentação principal
```

---

## 3. Convenções de Código

### 3.1. Scripts Python (`tools/` e `tests/`)
* **Versão Mínima:** Python 3.10+.
* **Tipagem Estática:** Usar sempre anotações de tipo completas (`typing.List`, `Tuple`, `Dict`, `Optional`, `Any`).
* **Dependências Externas:** Não adicione pacotes externos ao ecossistema padrão (`requirements.txt` desnecessário). Use módulos nativos: `sys`, `os`, `re`, `ast`, `operator`, `argparse`, `dataclasses`, `time`.
* **Interface CLI Padronizada:**
  * Toda ferramenta em `tools/` deve ter bloco `if __name__ == "__main__":` com `argparse` funcional.
  * Suportar flags comuns: `-o/--output`, `-h/--help`, `-a/--autorun`, `-n/--cycles`, `-d/--dump`.
  * Se invocada sem argumentos, a ferramenta deve exibir ajuda amigável ou executar a demonstração padrão embutida.
* **Resiliência de Caminhos:**
  * Qualquer script em `tests/` deve garantir acesso a `tools/` injetando `sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "tools")))`.

### 3.2. RTL Verilog Sintetizável (`rtl/` e `sim/`)
* **Padrão:** Verilog-2001 sintetizável e testável via **Icarus Verilog** (`iverilog`) e **Xilinx Vivado**.
* **Largura de Trits:** Barramentos ternários devem explicitar a largura física de 2 bits por trit:
  ```verilog
  // Dado de 6 trits balanceados = 12 bits
  input  wire [11:0] data_in;
  output wire [11:0] data_out;
  
  // Endereço de 10 trits balanceados = 20 bits
  output wire [19:0] addr_bus;
  ```
* **Constantes:** Sempre utilize parâmetros simbólicos para os trits:
  ```verilog
  localparam [1:0] TRIT_NEG = 2'b00; // -1
  localparam [1:0] TRIT_ZER = 2'b01; //  0
  localparam [1:0] TRIT_POS = 2'b10; // +1
  ```
* **FSM Síncrona:** FSMs devem ser acionadas na borda de subida do clock (`always @(posedge clk or posedge rst)`) com reset previsível.

### 3.3. Servidor Web Go (`web/`)
* **Padrão:** Go 1.22+ puro (biblioteca padrão `net/http`, `os/exec`, `sync`). Evitar frameworks pesados externos.
* **Tema Visual:** Estilização baseada na paleta oficial **Tokyo Night** (`#1a1b26`, `#24283b`, `#7dcfff`, `#bb9af7`), tipografia monoespaçada e telemetria dinâmica.
* **Comunicação Subprocesso:** Sempre sincronizar acesso ao stdin/stdout do emulador Python via `sync.Mutex` com deadlines (`SetReadDeadline`) para evitar travamento em concorrência.

---

## 4. Comandos Padrão para Teste e Validação

Antes de concluir qualquer alteração no código-fonte, o agente **DEVE OBRIGATORIAMENTE** executar e validar os seguintes comandos no terminal:

### 4.1. Test Suite de Software (Obrigatório)
```bash
make test
# Ou diretamente:
python3 tests/test_6502t_suite.py
```
* **Critério de Aceite:** O resultado final deve ser impreterivelmente:
  ```text
  RESULTADO FINAL: 7/7 testes passaram com sucesso.
  ```
  Se qualquer teste falhar, analise a FSM, o montador ou a ALU e corrija antes de prosseguir.

### 4.2. Pipeline de Montagem e Desassemblagem
```bash
# Compila src_asm/program.s para build/program.mem
make asm

# Desassembla o binário gerado para inspeção
make disasm
```

### 4.3. Emulação Automatizada
```bash
# Executa 60 ciclos com dump de memória final
python3 tools/emu6502t.py build/program.mem --autorun --cycles 60 --dump
```

### 4.4. Frontend Web e Containerização Docker
```bash
# Constrói a imagem Docker facrf/6502t-web:latest
make web-build

# Sobe o container em background na porta 27005
make web-run

# Valida endpoint HTTP
curl -s http://localhost:27005/api/cmd
```

### 4.5. Limpeza de Artefatos Temporários
```bash
make clean
```

---

## 5. Regras Cardeais para Agentes de IA

1. **Nunca quebre a suíte de testes:** Qualquer refatoração deve manter os 7 testes de regressão passando.
2. **Preserve a lógica balanceada:** Não substitua a aritmética simétrica por lógicas binárias padrão (como tentar usar complemento de dois convencional). A negação em 6502-T é simples inversão de polaridade: $-(-1) = +1$, $-(+1) = -1$, $-(0) = 0$.
3. **Mantenha a compatibilidade cruzada:** Alterações na ISA do montador (`tools/asm6502t.py`) devem ser imediatamente refletidas no desassemblador (`tools/disasm6502t.py`), no emulador (`tools/emu6502t.py`) e na especificação formal (`docs/especificacao_completa.md`).
4. **Git e Artefatos:** Não adicione ao repositório git diretórios `build/`, `.Xil/`, arquivos de log do Vivado (`vivado*.log`, `vivado*.jou`), bitstreams (`*.bit`), nem caches Python (`__pycache__`).
