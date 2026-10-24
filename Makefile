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

.PHONY: all test asm disasm emu sim bitstream program clean help web-build web-run

# Alvo padrão: executa a suite de testes e monta o programa de exemplo
all: test asm

# ==============================================================================
# SUITE DE TESTES EM PYTHON (EMULADOR + MONTADOR)
# ==============================================================================
test:
	@echo "===> [TEST] Executando Test Suite Automatizada (6502-T)..."
	@$(PYTHON) $(TEST_SUITE)

# ==============================================================================
# MONTAGEM E DESMONTAGEM DE CÓDIGO ASSEMBLY
# ==============================================================================
asm: $(MEM_OUTPUT)

$(MEM_OUTPUT): $(ASM_INPUT) $(ASM_TOOL) | $(BUILD_DIR)
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
sim: $(BUILD_DIR)/sim.vvp
	@echo "===> [SIM] Executando simulação de RTL..."
	@$(VVP) $(BUILD_DIR)/sim.vvp

$(BUILD_DIR)/sim.vvp: $(RTL_SRCS) $(SIM_TB) | $(BUILD_DIR)
	@echo "===> [SIM-BUILD] Compilando testbench com Icarus Verilog..."
	@$(IVERILOG) -o $@ -I $(RTL_DIR) $(RTL_SRCS) $(SIM_TB)

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
	@echo "  make test       - Roda a suite automatizada de testes em Python"
	@echo "  make asm        - Compila src_asm/program.s para build/program.mem"
	@echo "  make disasm     - Desassembla o binário build/program.mem"
	@echo "  make emu        - Inicia o emulador ciclo a ciclo interativo"
	@echo "  make web-build  - Constrói imagem Docker facrf/6502t-web:latest"
	@echo "  make web-run    - Executa container Docker na porta 27005"
	@echo "  make sim        - Compila e roda o testbench RTL no Icarus Verilog"
	@echo "  make bitstream  - Executa síntese, place, route e gera .bit no Vivado"
	@echo "  make program    - Grava o bitstream diretamente na FPGA via JTAG"
	@echo "  make clean      - Exclui diretório build/ e logs temporários"
