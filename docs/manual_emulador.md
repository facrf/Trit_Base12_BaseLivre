#!/usr/bin/env python3
"""
Emulador Ciclo a Ciclo da CPU Ternária 6502-T
Suporte a carga via CLI de arquivos .mem, .hex e arquivos Verilog.
Mapeamento: -1 -> 2'b00, 0 -> 2'b01, +1 -> 2'b10
"""

import sys
import os
import re
import argparse
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

# =====================================================================
# NÚCLEO DA CPU 6502-T
# =====================================================================

class CPU6502T:
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
        self.memory: List[int] = [0] * mem_size
        
        self.pc = 0
        self.acc = 0
        self.reg_x = 0
        self.reg_y = 0
        self.ir = 0
        
        self.operand_latch = 0
        self.target_addr = 0
        self.state = self.S_FETCH_OPCODE
        self.cycle_count = 0
        
        self.flag_zero = True
        self.flag_sign = 0
        self.flag_overflow = False

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
        s_int = a_val + b_val + cin
        cout = 0
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
        self.cycle_count += 1
        self.read_en = False
        self.write_en = False

        if self.state == self.S_FETCH_OPCODE:
            self.addr_bus = self.pc
            self.read_en = True
            self.state = self.S_DECODE

        elif self.state == self.S_DECODE:
            self.ir = self.memory[self.addr_bus % self.mem_size]
            self.pc = (self.pc + 1) % 59049
            
            trits = int_to_trits(self.ir, 6)
            mode = (trits[2], trits[3])

            if mode == (0, 1):
                self.addr_bus = self.pc
                self.read_en = True
                self.state = self.S_FETCH_OPERAND
            elif mode == (1, 1):
                self.addr_bus = self.pc
                self.read_en = True
                self.state = self.S_FETCH_ADDR_H
            else:
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

            if group == (1, 1):
                op_b = self.operand_latch if mode == (0, 1) else self.reg_x
                if op == (-1, -1):    self.acc = self.alu_add(self.acc, op_b)
                elif op == (-1, 1):   self.acc = self.alu_add(self.acc, -op_b)
                elif op == (0, 1):    self.acc = self.alu_add(0, -self.acc)
                elif op == (1, 1):    self.acc = self.alu_add(0, op_b)
                elif op == (1, 0):    self.acc = self.alu_add(self.acc, 1)
                elif op == (0, -1):   self.acc = self.alu_add(self.acc, -1)

            elif group == (-1, 0):
                if op == (-1, -1):
                    self.reg_x = self.acc
                elif op == (-1, 1):
                    self.acc = self.reg_x
                    self.flag_zero = (self.acc == 0)
                    self.flag_sign = 0 if self.acc == 0 else (1 if self.acc > 0 else -1)
                elif op == (0, 1):
                    self.reg_y = self.acc

            elif group == (-1, -1):
                if op == (-1, -1):    self.pc = self.target_addr
                elif op == (-1, 1):   (self.flag_zero and setattr(self, 'pc', self.target_addr))
                elif op == (0, -1):   (self.flag_sign == -1 and setattr(self, 'pc', self.target_addr))
                elif op == (0, 1):    (self.flag_sign == 1 and setattr(self, 'pc', self.target_addr))

            elif group == (0, 1):
                if op == (1, 1):
                    self.addr_bus = self.target_addr
                    self.data_out = self.acc
                    self.memory[self.target_addr % self.mem_size] = self.acc
                    self.write_en = True

            self.state = self.S_FETCH_OPCODE

    def step_instruction(self):
        while True:
            self.step_cycle()
            if self.state == self.S_FETCH_OPCODE:
                break

    def load_program_from_file(self, filepath: str) -> int:
        """
        Carrega programa de arquivo .mem, .hex ou Verilog.
        Retorna a quantidade de palavras carregadas com sucesso.
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

            # Formato .mem (vetor binário plano de 12 bits)
            bin_tokens = [tok for tok in clean.split() if re.fullmatch(r"[01]{12}", tok)]
            if bin_tokens:
                for b_tok in bin_tokens:
                    if current_addr < self.mem_size:
                        val = trits_to_int(bin12_to_trits(b_tok))
                        self.memory[current_addr] = val
                        current_addr += 1
                        words_loaded += 1
                continue

            # Formato numérico decimal ou hexadecimal puro por token
            num_tokens = clean.split()
            for token in num_tokens:
                val = None
                if token.startswith("$") or token.startswith("0x"):
                    val = int(token.replace("$", "").replace("0x", ""), 16)
                elif re.fullmatch(r"[+-]?\d+", token):
                    val = int(token)

                if val is not None and current_addr < self.mem_size:
                    self.memory[current_addr] = val
                    current_addr += 1
                    words_loaded += 1

        return words_loaded

# =====================================================================
# CLI DE DEPURAÇÃO
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

    def run_interactive(self):
        print("=" * 78)
        print(" EMULADOR 6502-T - DEPURAÇÃO INTERATIVA")
        print(" Comandos: [Enter]=Ciclo | 'i'=Instrução | 'c'=Continuar | 'm'=Memória | 'q'=Sair")
        print("=" * 78)
        self.print_status()

        while True:
            try:
                cmd = input("(6502-T) > ").strip().lower()
            except (KeyboardInterrupt, EOFError):
                break

            if cmd in ["", "s"]:
                self.cpu.step_cycle()
                self.print_status()
            elif cmd == "i":
                self.cpu.step_instruction()
                self.print_status()
            elif cmd == "c":
                steps = 0
                while steps < 10000:
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
                    print(f"Breakpoints: {sorted(list(self.breakpoints))}")
            elif cmd == "r":
                self.cpu.reset()
                print("[*] CPU Resetada.")
                self.print_status()
            elif cmd == "q":
                break


# =====================================================================
# ENTRADA DE EXECUÇÃO PRINCIPAL (CLI ARGPARSE)
# =====================================================================

def main():
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
        # Programa básico de demonstração caso nenhum arquivo seja passado
        demo = {0: 337, 1: 13, 2: -283, 3: 329, 4: 130, 5: 121, 6: 32}
        for k, v in demo.items():
            cpu.memory[k] = v
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
        debugger.run_interactive()


if __name__ == "__main__":
    main()