# ==============================================================================
# MAKEFILE UNIFICADO - CPU 6502-T (TRIT + BASE 12)
# ==============================================================================

PYTHON        := python3
VIVADO        := vivado
IVERILOG      := iverilog
VVP           := vvp

# Diretórios do Projeto
BUILD_DIR     := build
RTL_DIR       := rtl
CONSTR_DIR    := constr
SRC_ASM       := src_asm
TOOLS_DIR     := tools
TESTS_DIR     := tests
SCRIPTS_DIR   := scripts
DOCS_DIR      := docs

# Ferramentas CLI
ASM_TOOL      := $(TOOLS_DIR)/asm6502t.py
DISASM_TOOL   := $(TOOLS_DIR)/disasm6502t.py
EMU_TOOL      := $(TOOLS_DIR)/emu6502t.py
TEST_SUITE    := $(TESTS_DIR)/test_6502t_suite.py
TEST_EXTENDED := $(TESTS_DIR)/test_6502t_extended.py
TEST_VAMOS    := $(TESTS_DIR)/test_vamos_compiler.py
TEST_RTL      := $(TESTS_DIR)/test_rtl_logic.py
TCL_SCRIPT    := $(SCRIPTS_DIR)/build_bitstream.tcl

# Fontes e Alvos
ASM_INPUT     := $(SRC_ASM)/program.s
MEM_OUTPUT    := $(BUILD_DIR)/program.mem
BITSTREAM     := $(BUILD_DIR)/fpga_test_top.bit

# Listagem de RTL Verilog e Simulação
RTL_SRCS      := $(wildcard $(RTL_DIR)/*.v)
SIM_TB        := sim/tb_system_dozenal_monitor.v

# Configurações Docker Web
DOCKER_IMAGE  := facrf/6502t-web:latest
DOCKER_NAME   := 6502t_web
WEB_PORT      := 27005

.PHONY: all test test-all test-vamos test-rtl asm asm-all disasm emu sim bitstream program clean help web-build web-run

# Alvo padrão: executa a suite de testes e monta o programa de exemplo
all: test asm

# ==============================================================================
# SUITE DE TESTES EM PYTHON (EMULADOR + MONTADOR + VAMOS-T + RTL SILÍCIO)
# ==============================================================================
test:
	@echo "===> [TEST] Executando Test Suite Automatizada (6502-T)..."
	@$(PYTHON) $(TEST_SUITE)

test-vamos:
	@echo "===> [TEST-VAMOS] Executando Test Suite VAMOS-T..."
	@$(PYTHON) $(TEST_VAMOS)

test-rtl:
	@echo "===> [TEST-RTL] Executando Test Suite de Silício RTL (6502-T)..."
	@$(PYTHON) $(TEST_RTL)

test-all: test test-vamos test-rtl
	@echo "===> [TEST-EXTENDED] Executando Test Suite Estendida (6502-T)..."
	@$(PYTHON) $(TEST_EXTENDED)

# ==============================================================================
# MONTAGEM E DESMONTAGEM DE CÓDIGO ASSEMBLY
# ==============================================================================
asm: $(MEM_OUTPUT)

asm-all: $(MEM_OUTPUT) $(BUILD_DIR)/fibonacci.mem $(BUILD_DIR)/ternary_logic.mem $(BUILD_DIR)/tty_stream.mem

$(MEM_OUTPUT): $(ASM_INPUT) $(ASM_TOOL) | $(BUILD_DIR)
	@echo "===> [ASM] Montando $< para $@"
	@$(PYTHON) $(ASM_TOOL) $< -o $@

$(BUILD_DIR)/fibonacci.mem: $(SRC_ASM)/fibonacci.s $(ASM_TOOL) | $(BUILD_DIR)
	@echo "===> [ASM] Montando $< para $@"
	@$(PYTHON) $(ASM_TOOL) $< -o $@

$(BUILD_DIR)/ternary_logic.mem: $(SRC_ASM)/ternary_logic.s $(ASM_TOOL) | $(BUILD_DIR)
	@echo "===> [ASM] Montando $< para $@"
	@$(PYTHON) $(ASM_TOOL) $< -o $@

$(BUILD_DIR)/tty_stream.mem: $(SRC_ASM)/tty_stream.s $(ASM_TOOL) | $(BUILD_DIR)
	@echo "===> [ASM] Montando $< para $@"
	@$(PYTHON) $(ASM_TOOL) $< -o $@

disasm: $(MEM_OUTPUT) $(DISASM_TOOL)
	@echo "===> [DISASM] Desassemblando $(MEM_OUTPUT)..."
	@$(PYTHON) $(DISASM_TOOL) $(MEM_OUTPUT)

# Executa o emulador interativo sobre o binário montado
emu: asm
	@$(PYTHON) $(EMU_TOOL) $(MEM_OUTPUT)

# ==============================================================================
# SIMULAÇÃO RTL (ICARUS VERILOG)
# ==============================================================================
sim: | $(BUILD_DIR)
	@if command -v $(IVERILOG) >/dev/null 2>&1 && command -v $(VVP) >/dev/null 2>&1; then \
		echo "===> [SIM-BUILD] Compilando testbench com Icarus Verilog..."; \
		$(IVERILOG) -o $(BUILD_DIR)/sim.vvp -I $(RTL_DIR) $(RTL_SRCS) $(SIM_TB) && \
		echo "===> [SIM] Executando simulação de RTL..." && \
		$(VVP) $(BUILD_DIR)/sim.vvp; \
	else \
		echo "[AVISO] 'iverilog' e/ou 'vvp' não encontrados no sistema."; \
		echo "        Para simulação completa de silício em FPGA, instale: sudo apt install iverilog"; \
		echo "        A CPU 6502-T e VAMOS-T podem ser validados via software com: make test-all"; \
	fi

# ==============================================================================
# SÍNTESE E GERAÇÃO DE BITSTREAM (XILINX VIVADO)
# ==============================================================================
bitstream: $(BITSTREAM)

$(BITSTREAM): $(RTL_SRCS) $(CONSTR_DIR)/basys3_6502t.xdc $(TCL_SCRIPT) | $(BUILD_DIR)
	@echo "===> [VIVADO] Iniciando síntese e roteamento em lote..."
	@$(VIVADO) -mode batch -nojournal -nolog -source $(TCL_SCRIPT)

# Gravação direta na FPGA conectada via USB
program: $(BITSTREAM)
	@echo "===> [FPGA] Gravando bitstream na FPGA Basys 3..."
	@$(VIVADO) -mode batch -nojournal -nolog -eval \
	  "open_hw_manager; \
	   connect_hw_server; \
	   open_hw_target; \
	   set_property PROGRAM.FILE {$(BITSTREAM)} [current_hw_device]; \
	   program_hw_devices [current_hw_device]; \
	   close_hw_target; \
	   disconnect_hw_server;"

# ==============================================================================
# FRONTEND WEB & CONTAINER DOCKER (PORTA 27005)
# ==============================================================================
web-build: asm
	@echo "===> [DOCKER] Construindo imagem $(DOCKER_IMAGE)..."
	docker build -f Dockerfile.web -t $(DOCKER_IMAGE) .

web-run:
	@echo "===> [DOCKER] Subindo container $(DOCKER_NAME)..."
	@docker rm -f $(DOCKER_NAME) 2>/dev/null || true
	docker run -d --name $(DOCKER_NAME) -p $(WEB_PORT):$(WEB_PORT) $(DOCKER_IMAGE)

# ==============================================================================
# DIRETÓRIOS E LIMPEZA
# ==============================================================================
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

clean:
	@echo "===> [CLEAN] Removendo arquivos temporários e artefatos de build..."
	@rm -rf $(BUILD_DIR) *.log *.jou .Xil vivado*.str fsm_encoding.os

help:
	@echo "Comandos disponíveis no projeto 6502-T:"
	@echo "  make test       - Roda a suite automatizada de testes em Python (7/7)"
	@echo "  make test-vamos - Roda a suite automatizada do compilador VAMOS-T (7/7)"
	@echo "  make test-all   - Roda todas as suites (padrão, estendida e VAMOS-T)"
	@echo "  make asm        - Compila src_asm/program.s para build/program.mem"
	@echo "  make disasm     - Desassembla o binário build/program.mem"
	@echo "  make emu        - Inicia o emulador ciclo a ciclo interativo"
	@echo "  make web-build  - Constrói imagem Docker facrf/6502t-web:latest"
	@echo "  make web-run    - Executa container Docker na porta 27005"
	@echo "  make sim        - Compila e roda o testbench RTL no Icarus Verilog"
	@echo "  make bitstream  - Executa síntese, place, route e gera .bit no Vivado"
	@echo "  make program    - Grava o bitstream diretamente na FPGA via JTAG"
	@echo "  make clean      - Exclui diretório build/ e logs temporários"
