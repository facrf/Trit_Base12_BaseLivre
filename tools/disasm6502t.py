#!/usr/bin/env python3
"""
Desassemblador (Disassembler) para a CPU Ternária 6502-T.
Reconstrói código Assembly a partir de vetores de 12 bits (6 trits balanceados).
Identifica instruções, modos de endereçamento, literais em Base 12 e reconstrói labels.
"""

import sys
import re
from typing import Dict, Tuple, List, Optional

# Decodificação de 2 bits para trit balanceado (-1, 0, +1)
BIN_TO_TRIT = {
    "00": -1,  # TRIT_NEG
    "01":  0,  # TRIT_ZERO
    "10":  1   # TRIT_POS
}

# Tabela reversa de decodificação: ((Grupo), (Modo), (Operação)) -> (Mnemônico, Tipo de Operando)
# Tipos: 'NONE' (implícito), 'IMM' (imediato), 'ABS' (endereço absoluto)
REVERSE_ISA: Dict[Tuple[Tuple[int, int], Tuple[int, int], Tuple[int, int]], Tuple[str, str]] = {
    # ALU com Acumulador (Grupo: +1, +1)
    (( 1,  1), ( 0,  1), (-1, -1)): ("ADD", "IMM"),
    (( 1,  1), ( 1,  1), (-1, -1)): ("ADD", "ABS"),
    (( 1,  1), ( 0,  1), (-1,  1)): ("SUB", "IMM"),
    (( 1,  1), ( 1,  1), (-1,  1)): ("SUB", "ABS"),
    (( 1,  1), ( 0,  1), ( 1,  1)): ("LDA", "IMM"),
    (( 1,  1), ( 1,  1), ( 1,  1)): ("LDA", "ABS"),
    (( 1,  1), (-1, -1), ( 0,  1)): ("NEG", "NONE"),
    (( 1,  1), (-1, -1), ( 1,  0)): ("INC", "NONE"),
    (( 1,  1), (-1, -1), ( 0, -1)): ("DEC", "NONE"),

    # Transferências (Grupo: -1, 0)
    ((-1,  0), (-1, -1), (-1, -1)): ("TAX", "NONE"),
    ((-1,  0), (-1, -1), (-1,  1)): ("TXA", "NONE"),
    ((-1,  0), (-1, -1), ( 0,  1)): ("TAY", "NONE"),
    ((-1,  0), (-1, -1), ( 0, -1)): ("TYA", "NONE"),

    # Fluxo / Saltos (Grupo: -1, -1)
    ((-1, -1), ( 1,  1), (-1, -1)): ("JMP", "ABS"),
    ((-1, -1), ( 1,  1), (-1,  1)): ("BEQ", "ABS"),
    ((-1, -1), ( 1,  1), ( 0, -1)): ("BMI", "ABS"),
    ((-1, -1), ( 1,  1), ( 0,  1)): ("BPL", "ABS"),

    # Memória (Grupo: 0, +1)
    (( 0,  1), ( 1,  1), ( 1,  1)): ("STA", "ABS"),
}


def bin12_to_trits(bin_str: str) -> List[int]:
    """Converte string de 12 bits para lista de 6 trits [T0..T5]."""
    bin_str = bin_str.strip().replace("12'b", "").replace("_", "")
    if len(bin_str) != 12:
        raise ValueError(f"Vetor binário deve conter exatamente 12 bits: '{bin_str}'")

    # Mapeamento: bit 11..10 = T5, ..., bit 1..0 = T0
    # Na leitura sequencial: bin_str[0:2] = T5 ... bin_str[10:12] = T0
    t5 = BIN_TO_TRIT[bin_str[0:2]]
    t4 = BIN_TO_TRIT[bin_str[2:4]]
    t3 = BIN_TO_TRIT[bin_str[4:6]]
    t2 = BIN_TO_TRIT[bin_str[6:8]]
    t1 = BIN_TO_TRIT[bin_str[8:10]]
    t0 = BIN_TO_TRIT[bin_str[10:12]]

    return [t0, t1, t2, t3, t4, t5]


def trits_to_int(trits: List[int]) -> int:
    """Calcula o inteiro decimal correspondente aos 6 trits balanceados."""
    return sum(t * (3**i) for i, t in enumerate(trits))


def int_to_dozenal_str(val: int) -> str:
    """Converte inteiro decimal para representação literal em base 12 (0z...)."""
    if val == 0:
        return "0z0"
    sign_str = "-" if val < 0 else "+"
    v = abs(val)
    digits = []
    while v > 0:
        rem = v % 12
        if rem == 10:
            digits.append("A")
        elif rem == 11:
            digits.append("B")
        else:
            digits.append(str(rem))
        v //= 12
    return f"{sign_str}0z" + "".join(reversed(digits))


class Disassembler6502T:
    def __init__(self):
        self.memory: Dict[int, str] = {}
        self.labels: Dict[int, str] = {}
        self.instruction_map: Dict[int, Tuple[str, str, Optional[int]]] = {}

    def load_verilog_code(self, text: str):
        """Carrega dados a partir de blocos Verilog ou linhas .mem."""
        for line in text.splitlines():
            line = re.sub(r"//.*", "", line).strip()
            if not line:
                continue

            # Formato 1: ram_block[ 0] = 12'b101001101010;
            match_assign = re.search(r"\[\s*(\d+)\s*\]\s*=\s*(12'b[01]{12}|[01]{12})", line)
            if match_assign:
                addr = int(match_assign.group(1))
                bin_raw = match_assign.group(2).replace("12'b", "")
                self.memory[addr] = bin_raw
                continue

            # Formato 2: Arquivo .mem direto (12 bits por linha, com ou sem 12'b)
            clean_bits = line.replace("12'b", "").strip()
            if re.fullmatch(r"[01]{12}", clean_bits):
                addr = len(self.memory)
                self.memory[addr] = clean_bits

    def pass1_analyze_and_create_labels(self):
        """Passo 1: Percorre as instruções para mapear alvos de saltos e criar labels."""
        pc = 0
        max_addr = max(self.memory.keys()) if self.memory else 0

        while pc <= max_addr:
            if pc not in self.memory:
                pc += 1
                continue

            bin_word = self.memory[pc]
            trits = bin12_to_trits(bin_word)

            # Decomposição dos campos ortogonais
            # trits = [T0, T1, T2, T3, T4, T5]
            group = (trits[4], trits[5])
            mode  = (trits[2], trits[3])
            op    = (trits[0], trits[1])

            key = (group, mode, op)
            if key in REVERSE_ISA:
                mnemonic, op_type = REVERSE_ISA[key]

                if op_type in ["IMM", "ABS"]:
                    # Próxima palavra é operando
                    operand_addr = pc + 1
                    if operand_addr in self.memory:
                        op_val = trits_to_int(bin12_to_trits(self.memory[operand_addr]))
                        self.instruction_map[pc] = (mnemonic, op_type, op_val)

                        # Se for salto ou instrução de escrita absoluta, gera label
                        if op_type == "ABS":
                            if mnemonic in ["JMP", "BEQ", "BMI", "BPL"]:
                                self.labels[op_val] = f"L_{op_val:02d}"
                            elif mnemonic == "STA":
                                if op_val not in self.labels:
                                    self.labels[op_val] = f"DATA_{op_val:02d}"

                    pc += 2
                else:
                    self.instruction_map[pc] = (mnemonic, op_type, None)
                    pc += 1
            else:
                # Palavra desconhecida tratada como dado bruto
                pc += 1

    def pass2_generate_assembly(self) -> str:
        """Passo 2: Gera a listagem final formatada com mnemônicos e comentários."""
        pc = 0
        max_addr = max(self.memory.keys()) if self.memory else 0
        output_lines = []

        output_lines.append("; ===========================================================================")
        output_lines.append("; DESASSEMBLAGEM 6502-T (TRIT + BASE 12)")
        output_lines.append("; Endr  Vetor Binário  Instrução          Equiv. Decimal / Base 12")
        output_lines.append("; ===========================================================================")

        while pc <= max_addr:
            if pc not in self.memory:
                pc += 1
                continue

            # Emite Label caso o endereço seja referenciado
            if pc in self.labels:
                output_lines.append(f"\n{self.labels[pc]}:")

            bin_word = self.memory[pc]

            if pc in self.instruction_map:
                mnemonic, op_type, op_val = self.instruction_map[pc]

                if op_type == "IMM":
                    operand_bin = self.memory[pc + 1]
                    dozenal = int_to_dozenal_str(op_val)
                    pseudo_hint = " ; (Possível CLR)" if (mnemonic == "LDA" and op_val == 0) else ""

                    inst_str = f"{mnemonic:<4} #{op_val}"
                    output_lines.append(f"[{pc:02d}]   12'b{bin_word}   {inst_str:<18} ; Dec: {op_val:4d} | B12: {dozenal}{pseudo_hint}")
                    output_lines.append(f"[{pc+1:02d}]   12'b{operand_bin}   ; (Operando Imediato)")
                    pc += 2

                elif op_type == "ABS":
                    operand_bin = self.memory[pc + 1]
                    target_label = self.labels.get(op_val, f"${op_val}")
                    inst_str = f"{mnemonic:<4} {target_label}"
                    dozenal = int_to_dozenal_str(op_val)

                    output_lines.append(f"[{pc:02d}]   12'b{bin_word}   {inst_str:<18} ; Alvo: ${op_val} | B12: {dozenal}")
                    output_lines.append(f"[{pc+1:02d}]   12'b{operand_bin}   ; (Endereço Absoluto)")
                    pc += 2

                else:
                    output_lines.append(f"[{pc:02d}]   12'b{bin_word}   {mnemonic:<18}")
                    pc += 1
            else:
                # Palavra isolada de memória de dados
                val = trits_to_int(bin12_to_trits(bin_word))
                dozenal = int_to_dozenal_str(val)
                output_lines.append(f"[{pc:02d}]   12'b{bin_word}   .WORD {val:<12} ; Dado bruto: {dozenal}")
                pc += 1

        return "\n".join(output_lines)

    def disassemble(self, text: str) -> str:
        self.load_verilog_code(text)
        self.pass1_analyze_and_create_labels()
        return self.pass2_generate_assembly()


# =====================================================================
# DEMONSTRAÇÃO E CLI
# =====================================================================
def main():
    import argparse
    import os
    parser = argparse.ArgumentParser(
        description="Desassemblador (Disassembler) para a CPU Ternária 6502-T"
    )
    parser.add_argument("file", nargs="?", default=None, help="Arquivo de entrada (.mem, .v, .hex contendo palavras de 12 bits)")
    args = parser.parse_args()

    disasm = Disassembler6502T()
    if args.file:
        if not os.path.exists(args.file):
            print(f"[X] Arquivo não encontrado: {args.file}", file=sys.stderr)
            sys.exit(1)
        with open(args.file, "r", encoding="utf-8") as f:
            content = f.read()
        print(disasm.disassemble(content))
    else:
        # Teste com os vetores de demonstração
        sample_verilog_code = """
        initial begin
            ram_block[ 0] = 12'b101001101010; // LDA #0
            ram_block[ 1] = 12'b010101010101; // Dado (0)
            ram_block[ 2] = 12'b101001101010; // LDA #13
            ram_block[ 3] = 12'b010101101010; // Dado (13)
            ram_block[ 4] = 12'b011010101010; // STA $63
            ram_block[ 5] = 12'b010010010100; // Endereço ($63)
            ram_block[ 6] = 12'b101001101010; // LDA #29
            ram_block[ 7] = 12'b010110010100; // Dado (29)
            ram_block[ 8] = 12'b011010101010; // STA $62
            ram_block[ 9] = 12'b010010010110; // Endereço ($62)
            ram_block[10] = 12'b000100000000; // TAX
            ram_block[11] = 12'b101001100010; // SUB #29
            ram_block[12] = 12'b010110010100; // Dado (29)
            ram_block[13] = 12'b000010100010; // BEQ $16
            ram_block[14] = 12'b010101100010; // Endereço Salto ($16)
            ram_block[15] = 12'b000010100000; // JMP $0
            ram_block[16] = 12'b010101010101; // Endereço Salto ($0)
            ram_block[17] = 12'b101010101010; // LDA $62
            ram_block[18] = 12'b010010010110; // Endereço ($62)
            ram_block[19] = 12'b011010101010; // STA $32
            ram_block[20] = 12'b010110100000; // Endereço ($32)
            ram_block[25] = 12'b000010100000; // JMP $25
            ram_block[26] = 12'b010110010010; // Endereço Salto ($25)
        end
        """
        result = disasm.disassemble(sample_verilog_code)
        print(result)

if __name__ == "__main__":
    main()