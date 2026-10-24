#!/usr/bin/env python3
"""
Emulador Ciclo a Ciclo da CPU Ternária 6502-T
Simula barramentos físicos, registradores, ALU balanceada e FSM interna.
Apresenta telemetria em Trits, Decimal e Base 12 (Duodecimal).
"""

import sys
import os
import re
from typing import List, Tuple, Dict, Optional

# =====================================================================
# MATEMÁTICA E CONVERSÕES TERNÁRIAS BALANCEADAS / BASE 12
# =====================================================================

TRIT_CHARS = {-1: "-", 0: "0", 1: "+"}
CHAR_TO_TRIT = {"-": -1, "0": 0, "+": 1}

def int_to_trits(val: int, length: int = 6) -> List[int]:
    """Converte inteiro decimal para lista de trits balanceados [T0..T(n-1)]."""
    trits = []
    n = val
    for _ in range(length):
        rem = n % 3
        if rem == 2:
            trits.append(-1)
            n = (n // 3) + 1
        elif rem == 1:
            trits.append(1)
            n = n // 3
        else:
            trits.append(0)
            n = n // 3
    return trits

def trits_to_int(trits: List[int]) -> int:
    """Calcula o inteiro decimal correspondente à lista de trits."""
    return sum(t * (3**i) for i, t in enumerate(trits))

def trits_to_str(trits: List[int]) -> str:
    """Gera string de visualização física (MST -> LST)."""
    return "".join(TRIT_CHARS[t] for t in reversed(trits))

def int_to_dozenal(val: int) -> str:
    """Converte inteiro para representação duodecimal formatada com sinal."""
    if val == 0:
        return " 0z000"
    sign_str = "-" if val < 0 else "+"
    v = abs(val)
    d2 = (v // 144) % 12
    d1 = (v // 12) % 12
    d0 = v % 12
    chars = "0123456789AB"
    return f"{sign_str}0z{chars[d2]}{chars[d1]}{chars[d0]}"

def bin12_to_trits(bin_str: str) -> List[int]:
    """Decodifica palavra Verilog de 12 bits para trits [T0..T5]."""
    bin_str = bin_str.replace("12'b", "").replace("_", "").strip()
    map_2bit = {"00": -1, "01": 0, "10": 1}
    t = [map_2bit[bin_str[i:i+2]] for i in range(0, 12, 2)]
    return list(reversed(t))

def trits_to_bin12(trits: List[int]) -> str:
    """Codifica lista de 6 trits para string de 12 bits Verilog."""
    map_trit = {-1: "00", 0: "01", 1: "10"}
    return "".join(map_trit[t] for t in reversed(trits))

# =====================================================================
# NÚCLEO DA CPU 6502-T
# =====================================================================

class CPU6502T:
    # Estados da FSM (idênticos ao Verilog)
    S_FETCH_OPCODE  = 0
    S_DECODE        = 1
    S_FETCH_OPERAND = 2
    S_FETCH_ADDR_H  = 3
    S_EXECUTE       = 4

    STATE_NAMES = {
        0: "FETCH_OPCODE",
        1: "DECODE",
        2: "FETCH_OPERAND",
        3: "FETCH_ADDR",
        4: "EXECUTE"
    }

    def __init__(self, mem_size: int = 64):
        self.mem_size = mem_size
        self.memory: List[int] = [0] * mem_size  # Palavras de 6 trits (-364 a +364)
        
        # Registradores Internos
        self.pc = 0           # Program Counter (10 trits)
        self.acc = 0          # Acumulador A (6 trits)
        self.reg_x = 0        # Registrador de Índice X (6 trits)
        self.reg_y = 0        # Registrador de Índice Y (6 trits)
        self.ir = 0           # Instruction Register (6 trits)
        
        # Latches temporários da FSM
        self.operand_latch = 0
        self.target_addr = 0
        self.state = self.S_FETCH_OPCODE
        self.cycle_count = 0
        
        # Flags
        self.flag_zero = False
        self.flag_sign = 0    # -1, 0, +1
        self.flag_overflow = False

        # Linhas do Barramento Externo
        self.addr_bus = 0
        self.data_out = 0
        self.read_en = False
        self.write_en = False

    def reset(self):
        self.pc = 0
        self.acc = 0
        self.reg_x = 0
        self.reg_y = 0
        self.ir = 0
        self.state = self.S_FETCH_OPCODE
        self.cycle_count = 0
        self.flag_zero = True
        self.flag_sign = 0
        self.flag_overflow = False
        self.read_en = False
        self.write_en = False

    def alu_add(self, a_val: int, b_val: int, cin: int = 0) -> int:
        """Soma ternária de 6 trits com saturação e flags."""
        s_int = a_val + b_val + cin
        cout = 0
        
        # Detecção de Overflow além de 6 trits [-364, +364]
        if s_int > 364:
            cout = 1
            res = s_int - 729
        elif s_int < -364:
            cout = -1
            res = s_int + 729
        else:
            res = s_int
            cout = 0

        self.flag_overflow = (cout != 0)
        self.flag_zero = (res == 0)
        self.flag_sign = 0 if res == 0 else (1 if res > 0 else -1)
        return res

    def step_cycle(self):
        """Avança exatamente 1 ciclo de clock na máquina de estados."""
        self.cycle_count += 1
        self.read_en = False
        self.write_en = False

        if self.state == self.S_FETCH_OPCODE:
            self.addr_bus = self.pc
            self.read_en = True
            self.state = self.S_DECODE

        elif self.state == self.S_DECODE:
            # Captura a instrução do barramento
            self.ir = self.memory[self.addr_bus % self.mem_size]
            self.pc = (self.pc + 1) % 59049
            
            trits = int_to_trits(self.ir, 6)
            mode = (trits[2], trits[3]) # Modo de endereçamento

            if mode == (0, 1): # Imediato
                self.addr_bus = self.pc
                self.read_en = True
                self.state = self.S_FETCH_OPERAND
            elif mode == (1, 1): # Absoluto
                self.addr_bus = self.pc
                self.read_en = True
                self.state = self.S_FETCH_ADDR_H
            else: # Implícito / Registrador
                self.state = self.S_EXECUTE

        elif self.state == self.S_FETCH_OPERAND:
            self.operand_latch = self.memory[self.addr_bus % self.mem_size]
            self.pc = (self.pc + 1) % 59049
            self.state = self.S_EXECUTE

        elif self.state == self.S_FETCH_ADDR_H:
            self.target_addr = self.memory[self.addr_bus % self.mem_size]
            self.pc = (self.pc + 1) % 59049
            self.state = self.S_EXECUTE

        elif self.state == self.S_EXECUTE:
            trits = int_to_trits(self.ir, 6)
            group = (trits[4], trits[5])
            mode  = (trits[2], trits[3])
            op    = (trits[0], trits[1])

            # 1. Aritmética e Lógica da ALU (Grupo +1, +1)
            if group == (1, 1):
                if mode == (0, 1):      # Imediato
                    op_b = self.operand_latch
                elif mode == (1, 1):    # Absoluto (Memória)
                    op_b = self.memory[self.target_addr % self.mem_size]
                else:                   # Registrador X
                    op_b = self.reg_x
                if op == (-1, -1):    # ADD
                    self.acc = self.alu_add(self.acc, op_b)
                elif op == (-1, 1):   # SUB: A + (-B)
                    self.acc = self.alu_add(self.acc, -op_b)
                elif op == (0, 1):    # NEG
                    self.acc = self.alu_add(0, -self.acc)
                elif op == (1, 1):    # LOAD (LDA)
                    self.acc = self.alu_add(0, op_b)
                elif op == (1, 0):    # INC
                    self.acc = self.alu_add(self.acc, 1)
                elif op == (0, -1):   # DEC
                    self.acc = self.alu_add(self.acc, -1)

            # 2. Transferências entre Registradores (Grupo -1, 0)
            elif group == (-1, 0):
                if op == (-1, -1):    # TAX
                    self.reg_x = self.acc
                elif op == (-1, 1):   # TXA
                    self.acc = self.reg_x
                    self.flag_zero = (self.acc == 0)
                    self.flag_sign = 0 if self.acc == 0 else (1 if self.acc > 0 else -1)
                elif op == (0, 1):    # TAY
                    self.reg_y = self.acc

            # 3. Saltos de Fluxo (Grupo -1, -1)
            elif group == (-1, -1):
                if op == (-1, -1):    # JMP
                    self.pc = self.target_addr
                elif op == (-1, 1):   # BEQ
                    if self.flag_zero: self.pc = self.target_addr
                elif op == (0, -1):   # BMI
                    if self.flag_sign == -1: self.pc = self.target_addr
                elif op == (0, 1):    # BPL
                    if self.flag_sign == 1: self.pc = self.target_addr

            # 4. Acesso à Memória (Grupo 0, +1)
            elif group == (0, 1):
                if op == (1, 1):      # STA
                    self.addr_bus = self.target_addr
                    self.data_out = self.acc
                    self.memory[self.target_addr % self.mem_size] = self.acc
                    self.write_en = True

            self.state = self.S_FETCH_OPCODE

    def step_instruction(self):
        """Executa ciclos suficientes até completar uma instrução inteira."""
        initial_state = self.state
        while True:
            self.step_cycle()
            if self.state == self.S_FETCH_OPCODE:
                break

    def load_program(self, program: Dict[int, int]):
        """Carrega dicionário {endereco: valor_decimal} na memória."""
        for addr, val in program.items():
            if addr < self.mem_size:
                self.memory[addr] = val

    def load_program_from_file(self, filepath: str) -> int:
        """
        Carrega programa de arquivo .mem, .hex, .txt ou bloco Verilog.
        Retorna a quantidade de palavras de 6 trits carregadas com sucesso.
        """
        if not os.path.exists(filepath):
            raise FileNotFoundError(f"Arquivo não encontrado: {filepath}")

        with open(filepath, "r", encoding="utf-8") as f:
            content = f.read()

        words_loaded = 0
        current_addr = 0

        for line in content.splitlines():
            clean = re.sub(r"(//|;|#).*", "", line).strip()
            if not clean:
                continue

            # Diretiva de endereço estilo @addr (ex: @0A ou @12)
            if clean.startswith("@"):
                addr_token = clean[1:].strip()
                current_addr = int(addr_token, 16) if any(c in addr_token.lower() for c in "abcdef") else int(addr_token)
                continue

            # Formato de atribuição Verilog: ram_block[ 0] = 12'b101001101010;
            assign_match = re.search(r"\[\s*(\d+)\s*\]\s*=\s*(12'b[01]{12}|[01]{12})", clean)
            if assign_match:
                addr = int(assign_match.group(1))
                bin_str = assign_match.group(2).replace("12'b", "")
                val = trits_to_int(bin12_to_trits(bin_str))
                if addr < self.mem_size:
                    self.memory[addr] = val
                    words_loaded += 1
                continue

            # Formato .mem (vetor binário plano de 12 bits, com ou sem prefixo 12'b)
            bin_tokens = [tok.replace("12'b", "") for tok in clean.split() if re.fullmatch(r"(12'b)?[01]{12}", tok)]
            if bin_tokens:
                for b_tok in bin_tokens:
                    if current_addr < self.mem_size:
                        val = trits_to_int(bin12_to_trits(b_tok))
                        self.memory[current_addr] = val
                        current_addr += 1
                        words_loaded += 1
                continue

        return words_loaded

# =====================================================================
# INTERFACE DE DEPURAÇÃO INTERATIVA (CLI)
# =====================================================================

class DebuggerCLI:
    def __init__(self, cpu: CPU6502T):
        self.cpu = cpu
        self.breakpoints = set()

    def print_status(self):
        c = self.cpu
        tr_acc = trits_to_str(int_to_trits(c.acc, 6))
        tr_x   = trits_to_str(int_to_trits(c.reg_x, 6))
        tr_y   = trits_to_str(int_to_trits(c.reg_y, 6))
        tr_pc  = trits_to_str(int_to_trits(c.pc, 10))

        sign_sym = "-" if c.flag_sign == -1 else ("+" if c.flag_sign == 1 else "0")

        print("┌" + "─" * 76 + "┐")
        print(f"│ CICLO: {c.cycle_count:<6} | FSM: {c.STATE_NAMES[c.state]:<14} | BARRAMENTO: ADDR=${c.addr_bus:02d} "
              f"DATA_OUT={c.data_out:+04d} │")
        print("├" + "─" * 76 + "┤")
        print(f"│ REGISTRADORES       TRITS (MST->LST)    DECIMAL     BASE 12                │")
        print(f"│  Acumulador (A) :   [{tr_acc}]           {c.acc:+4d}        {int_to_dozenal(c.acc)}                │")
        print(f"│  Índice X       :   [{tr_x}]           {c.reg_x:+4d}        {int_to_dozenal(c.reg_x)}                │")
        print(f"│  Índice Y       :   [{tr_y}]           {c.reg_y:+4d}        {int_to_dozenal(c.reg_y)}                │")
        print(f"│  Prog. Counter  :   [{tr_pc}]       ${c.pc:04d}        {int_to_dozenal(c.pc)}                │")
        print("├" + "─" * 76 + "┤")
        print(f"│ FLAGS TERNÁRIAS: [Zero (Z): {int(c.flag_zero)}] [Sinal (S): {sign_sym}] "
              f"[Overflow (O): {int(c.flag_overflow)}]               │")
        print("└" + "─" * 76 + "┘")

    def dump_memory(self, start: int = 0, length: int = 16):
        print(f"\n--- MAPA DE MEMÓRIA (Palavras {start} a {min(start+length-1, self.cpu.mem_size-1)}) ---")
        for i in range(start, min(start + length, self.cpu.mem_size)):
            val = self.cpu.memory[i]
            tr_str = trits_to_str(int_to_trits(val, 6))
            doz_str = int_to_dozenal(val)
            marker = " <- PC" if i == self.cpu.pc else ""
            print(f"  [{i:02d}] : {val:+4d} | {doz_str} | Trits: [{tr_str}]{marker}")
        print()

    def run(self):
        print("=" * 78)
        print(" EMULADOR 6502-T (TRIT + BASE 12) - DEPURAÇÃO INTERATIVA CICLO A CICLO")
        print(" Comandos: [Enter]=Passo de Ciclo | 'i'=Passo de Instrução | 'c'=Continuar")
        print("           'm'=Ver Memória | 'b <end>'=Breakpoint | 'r'=Reset | 'q'=Sair")
        print("=" * 78)
        self.print_status()

        while True:
            try:
                cmd = input("(6502-T) > ").strip().lower()
            except (KeyboardInterrupt, EOFError):
                break

            if cmd == "" or cmd == "s":
                self.cpu.step_cycle()
                self.print_status()

            elif cmd == "i":
                self.cpu.step_instruction()
                self.print_status()

            elif cmd == "c":
                steps = 0
                while steps < 2000:
                    self.cpu.step_cycle()
                    steps += 1
                    if self.cpu.pc in self.breakpoints and self.cpu.state == CPU6502T.S_FETCH_OPCODE:
                        print(f"\n[!] Breakpoint atingido no endereço {self.cpu.pc}!")
                        break
                self.print_status()

            elif cmd.startswith("m"):
                parts = cmd.split()
                start = int(parts[1]) if len(parts) > 1 else 0
                length = int(parts[2]) if len(parts) > 2 else 16
                self.dump_memory(start, length)

            elif cmd.startswith("b"):
                parts = cmd.split()
                if len(parts) > 1:
                    bp = int(parts[1])
                    if bp in self.breakpoints:
                        self.breakpoints.remove(bp)
                        print(f"[-] Breakpoint removido de ${bp:02d}")
                    else:
                        self.breakpoints.add(bp)
                        print(f"[+] Breakpoint adicionado em ${bp:02d}")
                else:
                    print(f"Breakpoints ativos: {sorted(list(self.breakpoints))}")

            elif cmd == "r":
                self.cpu.reset()
                print("[*] CPU Resetada.")
                self.print_status()

            elif cmd == "q":
                break
            else:
                print("Comando não reconhecido. Use: [s, i, c, m, b, r, q]")

    def run_interactive(self):
        """Alias para o método run()."""
        self.run()

# =====================================================================
# PROGRAMA DEMO PRÉ-COMPILADO E PONTO DE ENTRADA CLI
# =====================================================================

def build_demo_program() -> Dict[int, int]:
    """
    Programa carregado na memória:
      [0]  LDA #+13    (0z011)
      [2]  TAX
      [3]  ADD #+130   (0z0BB) -> A vira +143
      [5]  STA $20     (Grava 143 no endereço 32)
      [7]  SUB #+143   -> A vira 0, ativa flag Zero
      [9]  BEQ $13     -> Salta para endereço 13
      [11] JMP $0
      [13] LDA #0z025  (+29 decimal)
      [15] JMP $15     -> Loop infinito de término
    """
    p = {}
    p[0] = 337;   p[1] = 13
    p[2] = -283
    p[3] = 329;   p[4] = 130
    p[5] = 121;   p[6] = 32
    p[7] = 331;   p[8] = 143
    p[9] = -290;  p[10] = 13
    p[11] = -292; p[12] = 0
    p[13] = 337;  p[14] = 29
    p[15] = -292; p[16] = 15
    return p

def main():
    import argparse
    parser = argparse.ArgumentParser(
        description="Emulador Ciclo a Ciclo da CPU Ternária 6502-T (TRIT + Base 12)"
    )
    parser.add_argument(
        "file",
        nargs="?",
        default=None,
        help="Caminho do arquivo binário (.mem, .hex, .txt ou bloco Verilog) para carregar na RAM"
    )
    parser.add_argument(
        "-s", "--size",
        type=int,
        default=64,
        help="Tamanho da memória RAM em palavras de 6 trits (padrão: 64)"
    )
    parser.add_argument(
        "-a", "--autorun",
        action="store_true",
        help="Executa diretamente o programa sem abrir a interface interativa"
    )
    parser.add_argument(
        "-n", "--cycles",
        type=int,
        default=500,
        help="Limite de ciclos em modo autorun (padrão: 500)"
    )
    parser.add_argument(
        "-d", "--dump",
        action="store_true",
        help="Exibe o dump de memória ao final da execução"
    )

    args = parser.parse_args()

    cpu = CPU6502T(mem_size=args.size)

    if args.file:
        try:
            count = cpu.load_program_from_file(args.file)
            print(f"[✓] {count} palavras de 6 trits carregadas de '{args.file}'.")
        except Exception as e:
            print(f"[X] Erro ao carregar arquivo: {e}", file=sys.stderr)
            sys.exit(1)
    else:
        demo = build_demo_program()
        cpu.load_program(demo)
        print("[i] Nenhum arquivo especificado. Programa de teste padrão carregado.")

    debugger = DebuggerCLI(cpu)

    if args.autorun:
        print(f"[*] Executando até {args.cycles} ciclos de clock...")
        for _ in range(args.cycles):
            cpu.step_cycle()
        debugger.print_status()
        if args.dump:
            debugger.dump_memory(0, min(args.size, 32))
    else:
        debugger.run()

if __name__ == "__main__":
    main()