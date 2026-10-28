#!/usr/bin/env python3
"""
Montador (Assembler) Avançado para CPU Ternária 6502-T
Suporta:
- Pseudo-instruções: CLR, CMP, PUSH, POP
- Avaliador de expressões aritméticas (+, -, *, //, %, parênteses)
- Literais: Decimal, Hexadecimal ($FF ou 0xFF) e Duodecimal (0z0BB ou 0z10)
- Tabela de símbolos e cálculo de deslocamentos relativos
"""

import sys
import os
import re
import ast
import operator
from typing import List, Tuple, Dict, Any

# Mapeamento físico dos trits para barramento Verilog (2 bits por trit)
TRIT_MAP = {
    -1: "00",  # TRIT_NEG
     0: "01",  # TRIT_ZERO
     1: "10"   # TRIT_POS
}

# Tabela ISA Básica 6502-T: (Grupo, Modo, Operação)
# Cada campo possui 2 trits (T1, T0)
INSTRUCTION_SET = {
    # ALU com Acumulador (Grupo: +1, +1)
    "ADD_IMM": (( 1,  1), ( 0,  1), (-1, -1)),
    "ADD_ABS": (( 1,  1), ( 1,  1), (-1, -1)),
    "SUB_IMM": (( 1,  1), ( 0,  1), (-1,  1)),
    "SUB_ABS": (( 1,  1), ( 1,  1), (-1,  1)),
    "LDA_IMM": (( 1,  1), ( 0,  1), ( 1,  1)),
    "LDA_ABS": (( 1,  1), ( 1,  1), ( 1,  1)),
    "NEG":     (( 1,  1), (-1, -1), ( 0,  1)),
    "INC":     (( 1,  1), (-1, -1), ( 1,  0)),
    "DEC":     (( 1,  1), (-1, -1), ( 0, -1)),

    # Transferência entre Registradores (Grupo: -1, 0)
    "TAX":     ((-1,  0), (-1, -1), (-1, -1)),
    "TXA":     ((-1,  0), (-1, -1), (-1,  1)),
    "TAY":     ((-1,  0), (-1, -1), ( 0,  1)),
    "TYA":     ((-1,  0), (-1, -1), ( 0, -1)),

    # Instrução Neutra (Grupo: 0, 0)
    "NOP":     (( 0,  0), ( 0,  0), ( 0,  0)),

    # Saltos e Desvios de Fluxo (Grupo: -1, -1)
    "JMP":     ((-1, -1), ( 1,  1), (-1, -1)),
    "BEQ":     ((-1, -1), ( 1,  1), (-1,  1)),
    "BNE":     ((-1, -1), ( 1,  1), ( 1, -1)),
    "BMI":     ((-1, -1), ( 1,  1), ( 0, -1)),
    "BPL":     ((-1, -1), ( 1,  1), ( 0,  1)),

    # Acesso à Memória (Grupo: 0, +1)
    "STA":     (( 0,  1), ( 1,  1), ( 1,  1)),
}


def int_to_balanced_ternary(val: int, num_trits: int = 6) -> List[int]:
    """Converte um inteiro decimal para base 3 balanceada {-1, 0, +1}."""
    max_val = (3**num_trits - 1) // 2
    if val > max_val or val < -max_val:
        raise ValueError(f"Valor {val} excede capacidade de {num_trits} trits [-{max_val}, +{max_val}].")

    trits = []
    n = val
    while len(trits) < num_trits:
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

    return trits  # [T0..T5] (LST primeiro)


def trits_to_verilog_bin(trits: List[int]) -> str:
    """Gera string de 12 bits Verilog na ordem física [T5..T0]."""
    bin_str = "".join(TRIT_MAP[t] for t in reversed(trits))
    return f"12'b{bin_str}"


def encode_opcode(group: Tuple[int, int], mode: Tuple[int, int], op: Tuple[int, int]) -> str:
    """Monta palavra de instrução de 6 trits a partir dos campos ortogonais."""
    trits = [op[0], op[1], mode[0], mode[1], group[0], group[1]]
    return trits_to_verilog_bin(trits)


class SafeExpressionEvaluator(ast.NodeVisitor):
    """Avaliador seguro de expressões matemáticas baseado em AST."""
    
    OPERATORS = {
        ast.Add: operator.add,
        ast.Sub: operator.sub,
        ast.Mult: operator.mul,
        ast.FloorDiv: operator.floordiv,
        ast.Div: operator.floordiv,  # Divisão inteira como padrão de montagem
        ast.Mod: operator.mod,
        ast.USub: operator.neg,
        ast.UAdd: operator.pos,
    }

    def __init__(self, symbols: Dict[str, int]):
        self.symbols = symbols

    def visit_BinOp(self, node):
        left = self.visit(node.left)
        right = self.visit(node.right)
        op_func = self.OPERATORS.get(type(node.op))
        if not op_func:
            raise ValueError(f"Operador binário não suportado: {type(node.op).__name__}")
        return op_func(left, right)

    def visit_UnaryOp(self, node):
        operand = self.visit(node.operand)
        op_func = self.OPERATORS.get(type(node.op))
        if not op_func:
            raise ValueError(f"Operador unário não suportado: {type(node.op).__name__}")
        return op_func(operand)

    def visit_Constant(self, node):
        return int(node.value)

    def visit_Name(self, node):
        if node.id in self.symbols:
            return self.symbols[node.id]
        raise KeyError(f"Símbolo ou Label não encontrado: '{node.id}'")

    def generic_visit(self, node):
        raise ValueError(f"Expressão inválida na instrução: {type(node).__name__}")


class Assembler6502T:
    def __init__(self, stack_base: int = 63):
        self.stack_base = stack_base
        self.labels: Dict[str, int] = {}
        self.expanded_code: List[Tuple[str, str, str]] = []  # (label, mnemonic, operand_str)
        self.machine_code: Dict[int, Tuple[str, str]] = {}   # addr -> (verilog_bin, comment)

    def _normalize_literals(self, expr: str) -> str:
        """Converte literais duodecimais (0z...) e hexadecimais ($...) para inteiros."""
        # Suporte a Duodecimal (Base 12): ex: 0z0BB -> 143, 0z10 -> 12
        def replace_dozenal(m):
            return str(int(m.group(1), 12))
        expr = re.sub(r"0[zZ]([0-9a-bA-B]+)", replace_dozenal, expr)

        # Suporte a Hexadecimal: $1F -> 0x1F
        expr = re.sub(r"\$([0-9a-fA-F]+)", r"0x\1", expr)
        return expr

    def evaluate_expression(self, expr_str: str) -> int:
        """Processa e avalia expressões matemáticas complexas."""
        clean_expr = self._normalize_literals(expr_str.strip())
        parsed_ast = ast.parse(clean_expr, mode="eval")
        evaluator = SafeExpressionEvaluator(self.labels)
        return evaluator.visit(parsed_ast.body)

    def pass0_expand_macros(self, source: str):
        """Passo 0: Expande pseudo-instruções e gerencia profundidade da pilha."""
        stack_depth = 0
        raw_lines = source.splitlines()

        for line in raw_lines:
            clean = line.strip()
            if clean.startswith("#"):
                continue
            clean = re.sub(r"(//|;|(?<=\s)#\s).*", "", clean).strip()
            if not clean:
                continue

            # Suporte a definição de constantes (ex: TARGET = 35 ou OFFSET = (3 * 4))
            if "=" in clean:
                left_side, _, expr_part = clean.partition("=")
                left_side = left_side.strip()
                if re.match(r"^[A-Za-z_][A-Za-z0-9_]*$", left_side):
                    val = self.evaluate_expression(expr_part)
                    self.labels[left_side] = val
                    continue

            current_label = ""
            if ":" in clean:
                label_part, _, instruction_part = clean.partition(":")
                current_label = label_part.strip()
                clean = instruction_part.strip()
                if not clean:
                    # Linha contendo apenas label
                    self.expanded_code.append((current_label, "", ""))
                    continue

            parts = clean.split(None, 1)
            mnemonic = parts[0].upper()
            operand = parts[1].strip() if len(parts) > 1 else ""

            # --- EXPANSÃO DE PSEUDO-INSTRUÇÕES ---
            if mnemonic == "CLR":
                # CLR -> LDA #0 (Zera o Acumulador e seta flag Zero)
                self.expanded_code.append((current_label, "LDA", "#0"))

            elif mnemonic == "CMP":
                # CMP <op> -> TAX (Salva A em X) seguido de SUB <op> (Atualiza flags)
                self.expanded_code.append((current_label, "TAX", ""))
                self.expanded_code.append(("", "SUB", operand))

            elif mnemonic == "PUSH":
                # PUSH -> STA $(STACK_BASE - depth)
                target_addr = f"{self.stack_base - stack_depth}"
                self.expanded_code.append((current_label, "STA", target_addr))
                stack_depth += 1

            elif mnemonic == "POP":
                # POP -> LDA $(STACK_BASE - depth + 1)
                if stack_depth <= 0:
                    raise IndexError(f"Stack Underflow detectado na expansão da instrução POP!")
                stack_depth -= 1
                target_addr = f"{self.stack_base - stack_depth}"
                self.expanded_code.append((current_label, "LDA", target_addr))

            else:
                # Instrução nativa comum
                self.expanded_code.append((current_label, mnemonic, operand))

    def pass1_collect_labels(self):
        """Passo 1: Calcula alinhamento de endereços e indexa labels."""
        current_addr = 0
        for label, mnemonic, operand in self.expanded_code:
            if label:
                self.labels[label] = current_addr

            if not mnemonic:
                continue

            if mnemonic in [".ORG", "ORG"]:
                current_addr = self.evaluate_expression(operand)
                continue
            elif mnemonic in [".WORD", "WORD", ".DATA", "DATA"]:
                current_addr += 1
                continue

            # Instruções com operando imediato ou absoluto ocupam 2 palavras
            if operand:
                current_addr += 2
            else:
                current_addr += 1

    def pass2_assemble(self):
        """Passo 2: Avalia expressões e sintetiza o binário ternário balanceado."""
        current_addr = 0
        for _, mnemonic, operand in self.expanded_code:
            if not mnemonic:
                continue

            if mnemonic in [".ORG", "ORG"]:
                current_addr = self.evaluate_expression(operand)
                continue
            elif mnemonic in [".WORD", "WORD", ".DATA", "DATA"]:
                val = self.evaluate_expression(operand)
                val_trits = int_to_balanced_ternary(val, 6)
                self.machine_code[current_addr] = (trits_to_verilog_bin(val_trits), f".WORD {val}")
                current_addr += 1
                continue

            is_immediate = operand.startswith("#")
            clean_op = operand[1:] if is_immediate else operand

            # 1. Instruções Aritméticas e de Carga
            if mnemonic in ["LDA", "ADD", "SUB"]:
                val = self.evaluate_expression(clean_op)
                mode_suffix = "IMM" if is_immediate else "ABS"
                opcode_key = f"{mnemonic}_{mode_suffix}"

                opcode_bin = encode_opcode(*INSTRUCTION_SET[opcode_key])
                self.machine_code[current_addr] = (opcode_bin, f"{mnemonic} {operand}")
                current_addr += 1

                # Emissão do valor/endereço resolvido
                val_trits = int_to_balanced_ternary(val, 6)
                comment = f"Dado ({val})" if is_immediate else f"Endereço (${val})"
                self.machine_code[current_addr] = (trits_to_verilog_bin(val_trits), comment)
                current_addr += 1

            # 2. Acesso à Memória (STA)
            elif mnemonic == "STA":
                target_addr = self.evaluate_expression(clean_op)
                opcode_bin = encode_opcode(*INSTRUCTION_SET["STA"])
                self.machine_code[current_addr] = (opcode_bin, f"STA ${target_addr}")
                current_addr += 1

                addr_trits = int_to_balanced_ternary(target_addr, 6)
                self.machine_code[current_addr] = (trits_to_verilog_bin(addr_trits), f"Endereço (${target_addr})")
                current_addr += 1

            # 3. Saltos Condicionais e Incondicionais
            elif mnemonic in ["JMP", "BEQ", "BNE", "BMI", "BPL"]:
                target_addr = self.evaluate_expression(clean_op)
                opcode_bin = encode_opcode(*INSTRUCTION_SET[mnemonic])
                self.machine_code[current_addr] = (opcode_bin, f"{mnemonic} ${target_addr}")
                current_addr += 1

                addr_trits = int_to_balanced_ternary(target_addr, 6)
                self.machine_code[current_addr] = (trits_to_verilog_bin(addr_trits), f"Endereço Salto (${target_addr})")
                current_addr += 1

            # 4. Instruções Implícitas / Registradores
            elif mnemonic in ["TAX", "TXA", "TAY", "TYA", "NEG", "INC", "DEC", "NOP"]:
                opcode_bin = encode_opcode(*INSTRUCTION_SET[mnemonic])
                self.machine_code[current_addr] = (opcode_bin, mnemonic)
                current_addr += 1

            else:
                raise SyntaxError(f"Instrução desconhecida: {mnemonic}")

    def compile(self, source: str):
        self.pass0_expand_macros(source)
        self.pass1_collect_labels()
        self.pass2_assemble()

    def export_verilog_initial(self, array_name: str = "ram_block") -> str:
        lines = ["initial begin"]
        for addr in sorted(self.machine_code.keys()):
            bin_val, comment = self.machine_code[addr]
            lines.append(f"    {array_name}[{addr:2d}] = {bin_val}; // {comment}")
        lines.append("end")
        return "\n".join(lines)

    def export_mem(self) -> str:
        r"""Exporta palavras de 12 bits linha a linha (compatível com $readmemb e emulador)."""
        if not self.machine_code:
            return ""
        max_addr = max(self.machine_code.keys())
        lines = []
        for addr in range(max_addr + 1):
            if addr in self.machine_code:
                bin_val, comment = self.machine_code[addr]
                lines.append(f"{bin_val} // [{addr:2d}] {comment}")
            else:
                lines.append(f"010101010101 // [{addr:2d}] NOP/ZERO")
        return "\n".join(lines) + "\n"


# =====================================================================
# PROGRAMA DE DEMONSTRAÇÃO E CLI
# =====================================================================
def main():
    import argparse
    parser = argparse.ArgumentParser(
        description="Montador (Assembler) Avançado para CPU Ternária 6502-T"
    )
    parser.add_argument("input", nargs="?", default=None, help="Arquivo fonte Assembly (.s ou .asm). Use '-' para stdin.")
    parser.add_argument("-c", "--code", default=None, help="Código assembly passado diretamente como string")
    parser.add_argument("-o", "--output", default=None, help="Arquivo de saída (.mem ou .v)")
    parser.add_argument("-f", "--format", choices=["mem", "verilog"], default="mem", help="Formato de saída (padrão: mem)")
    parser.add_argument("--stack-base", type=int, default=63, help="Endereço base da pilha (padrão: 63)")
    args = parser.parse_args()

    src: Optional[str] = None
    if args.code:
        src = args.code
    elif args.input == "-" or (args.input is None and not sys.stdin.isatty()):
        src = sys.stdin.read()
    elif args.input:
        if not os.path.exists(args.input):
            print(f"[X] Arquivo não encontrado: {args.input}", file=sys.stderr)
            sys.exit(1)
        with open(args.input, "r", encoding="utf-8") as f:
            src = f.read()

    if src is not None:
        assembler = Assembler6502T(stack_base=args.stack_base)
        assembler.compile(src)

        fmt = args.format
        if args.output and args.output.endswith(".v"):
            fmt = "verilog"

        if fmt == "verilog":
            out_content = assembler.export_verilog_initial()
        else:
            out_content = assembler.export_mem()

        if args.output:
            out_dir = os.path.dirname(os.path.abspath(args.output))
            if out_dir:
                os.makedirs(out_dir, exist_ok=True)
            with open(args.output, "w", encoding="utf-8") as f:
                f.write(out_content)
            print(f"[✓] Montagem concluída com sucesso: '{args.output}' gerado ({len(assembler.machine_code)} palavras).")
        else:
            print(out_content)
    else:
        src_code = """
        // Demonstração: Pseudo-Instruções, Base 12 e Expressões Aritméticas
        BUFFER_START = 30
        OFFSET       = (3 * 4) // 12 decimal (0z10)

        START:
            CLR                     // Pseudo: LDA #0
            LDA #0z011              // Literal duodecimal: +13 decimal
            PUSH                    // Pseudo: Empilha 13 em STACK_BASE (63)
            
            LDA #(2 * OFFSET) + 5   // Expressão: 2*12 + 5 = +29
            PUSH                    // Pseudo: Empilha 29 em STACK_BASE - 1 (62)
            
            // Testa Comparação
            CMP #29                 // Pseudo: TAX + SUB #29
            BEQ PROCESS             // Se igual, salta para PROCESS
            JMP START

        PROCESS:
            POP                     // Pseudo: Desempilha 29 de volta para A
            STA (BUFFER_START + 2)  // Grava 29 no endereço 32
            POP                     // Pseudo: Desempilha 13 de volta para A
            STA BUFFER_START        // Grava 13 no endereço 30
            
        LOOP_END:
            JMP LOOP_END
        """
        assembler = Assembler6502T(stack_base=args.stack_base)
        assembler.compile(src_code)

        print("=" * 75)
        print(" BLOCO VERILOG GERADO COM EXPANSÕES E EXPRESSÕES CALCULADAS:")
        print("=" * 75)
        print(assembler.export_verilog_initial("ram_block"))

if __name__ == "__main__":
    main()