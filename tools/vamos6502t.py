#!/usr/bin/env python3
"""
Compilador VAMOS-T (VAMOS-LANG -> 6502-T Assembly)
Converte código VAMOS-LANG (PT-BR) com literais Duodecimais e lógicas ternárias
em código assembly 6502-T nativo e executável no emulador e no silício.
"""

import sys
import os
import re
import json
import argparse
from typing import List, Dict, Tuple, Optional

# Garante acesso às ferramentas do repositório
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from asm6502t import Assembler6502T
from emu6502t import CPU6502T, bin12_to_trits, trits_to_int, int_to_dozenal, int_to_trits, trits_to_str


class Token:
    def __init__(self, type_: str, value: str, line: int):
        self.type = type_
        self.value = value
        self.line = line

    def __repr__(self):
        return f"Token({self.type}, {repr(self.value)})"


class VamosLexer:
    TOKEN_SPECS = [
        ("COMMENT",   r"//.*|/\*[\s\S]*?\*/|#.*"),
        ("DOZENAL",   r"[+-]?0[zZ][0-9a-bA-B]+"),
        ("HEX",       r"0[xX][0-9a-fA-F]+|\$[0-9a-fA-F]+"),
        ("NUMBER",    r"[+-]?\d+"),
        ("OP",        r":=|==|!=|<=|>=|\+\+|--|[+\-=<>!]"),
        ("IDENT",     r"[a-zA-Z_]\w*"),
        ("LBRACKET",  r"\["),
        ("RBRACKET",  r"\]"),
        ("LBRACE",    r"\{"),
        ("RBRACE",    r"\}"),
        ("LPAREN",    r"\("),
        ("RPAREN",    r"\)"),
        ("STRING",    r'"[^"\\]*(?:\\.[^"\\]*)*"'),
        ("SEMICOLON", r";"),
        ("NEWLINE",   r"\n"),
        ("SKIP",      r"[ \t\r]+"),
    ]

    def __init__(self, source: str):
        self.source = source
        regex_parts = [f"(?P<{name}>{pattern})" for name, pattern in self.TOKEN_SPECS]
        self.regex = re.compile("|".join(regex_parts))

    def tokenize(self) -> List[Token]:
        tokens = []
        line_num = 1
        for match in self.regex.finditer(self.source):
            kind = match.lastgroup
            val = match.group()
            if kind == "NEWLINE":
                tokens.append(Token("NEWLINE", "\n", line_num))
                line_num += 1
            elif kind == "SKIP" or kind == "COMMENT":
                pass
            else:
                tokens.append(Token(kind, val, line_num))
        tokens.append(Token("EOF", "", line_num))
        return tokens


class VamosCompiler6502T:
    def __init__(self, var_base_addr: int = 180, tty_port: int = 300):
        self.var_base = var_base_addr
        self.tty_port = tty_port
        self.variables: Dict[str, int] = {}
        self.next_var_addr = var_base_addr
        self.label_counter = 0
        self.asm: List[str] = []
        self.tokens: List[Token] = []
        self.pos = 0

    def _get_label(self, prefix: str = "L") -> str:
        self.label_counter += 1
        return f"{prefix}_{self.label_counter}"

    def _get_var_addr(self, name: str) -> int:
        norm = name.lower()
        if norm not in self.variables:
            self.variables[norm] = self.next_var_addr
            self.next_var_addr += 1
        return self.variables[norm]

    def _cur(self) -> Token:
        return self.tokens[self.pos] if self.pos < len(self.tokens) else Token("EOF", "", -1)

    def _peek(self, offset: int = 1) -> Token:
        idx = self.pos + offset
        return self.tokens[idx] if idx < len(self.tokens) else Token("EOF", "", -1)

    def _consume(self, expected_type: Optional[str] = None, expected_val: Optional[str] = None) -> Token:
        tok = self._cur()
        if expected_type and tok.type != expected_type:
            raise SyntaxError(f"Linha {tok.line}: Esperava token '{expected_type}', obteve '{tok.type}' ({tok.value})")
        if expected_val and tok.value.lower() != expected_val.lower():
            raise SyntaxError(f"Linha {tok.line}: Esperava '{expected_val}', obteve '{tok.value}'")
        self.pos += 1
        return tok

    def _skip_terminators(self):
        while self._cur().type in ("NEWLINE", "SEMICOLON"):
            self.pos += 1

    def compile(self, source: str) -> str:
        lexer = VamosLexer(source)
        self.tokens = lexer.tokenize()
        self.pos = 0
        self.variables.clear()
        self.next_var_addr = self.var_base
        self.label_counter = 0
        self.asm = [
            "// =============================================================================",
            "// CÓDIGO MONTADO AUTOMATICAMENTE VIA VAMOS-T COMPILER (6502-T)",
            "// =============================================================================",
            "START:"
        ]

        self._parse_program()

        self.asm.append("\nHLT:")
        self.asm.append("    JMP HLT")
        return "\n".join(self.asm)

    def _parse_program(self):
        self._skip_terminators()
        if self._cur().type == "IDENT" and self._cur().value == "pacote":
            self._consume()
            self._consume("IDENT")
            self._skip_terminators()

        if self._cur().type == "IDENT" and self._cur().value == "funcao":
            self._consume()
            self._consume("IDENT")
            if self._cur().type == "LPAREN":
                self._consume("LPAREN")
                self._consume("RPAREN")
            self._consume("LBRACE")
            self._parse_block(until_rbrace=True)
            if self._cur().type == "RBRACE":
                self._consume("RBRACE")
        else:
            self._parse_block(until_rbrace=False)

    def _parse_block(self, until_rbrace: bool):
        while True:
            self._skip_terminators()
            tok = self._cur()
            if tok.type == "EOF":
                break
            if until_rbrace and tok.type == "RBRACE":
                break

            self._parse_statement()

    def _parse_statement(self):
        tok = self._cur()

        # 1. Comandos embutidos: parar(), zerar(), nop()
        if tok.type == "IDENT" and tok.value in ("parar", "pausar"):
            self._consume()
            if self._cur().type == "LPAREN":
                self._consume("LPAREN"); self._consume("RPAREN")
            self.asm.append("    JMP HLT")
            return

        if tok.type == "IDENT" and tok.value in ("zerar", "limpar"):
            self._consume()
            if self._cur().type == "LPAREN":
                self._consume("LPAREN"); self._consume("RPAREN")
            self.asm.append("    CLR")
            return

        if tok.type == "IDENT" and tok.value == "nop":
            self._consume()
            if self._cur().type == "LPAREN":
                self._consume("LPAREN"); self._consume("RPAREN")
            self.asm.append("    NOP")
            return

        # 2. asm("...")
        if tok.type == "IDENT" and tok.value == "asm":
            self._consume()
            self._consume("LPAREN")
            raw_str = self._consume("STRING").value[1:-1]
            self._consume("RPAREN")
            self.asm.append(f"    {raw_str}")
            return

        # 2.1. Saída TTY: escrever("Olá Mundo!") / imprimir(...) / print(...)
        if tok.type == "IDENT" and tok.value.lower() in ("escrever", "imprimir", "print", "mostrar"):
            self._consume()
            self._consume("LPAREN")
            arg_tok = self._cur()
            self.pos += 1

            if arg_tok.type == "STRING":
                raw_text = arg_tok.value[1:-1].replace("\\n", "\n").replace("\\t", "\t").replace('\\"', '"').replace("\\\\", "\\")
                for ch in raw_text:
                    code = ord(ch)
                    ch_disp = repr(ch)[1:-1]
                    self.asm.append(f"    LDA #{code} // char '{ch_disp}'")
                    self.asm.append(f"    STA {self.tty_port} // TTY_OUT")
            elif arg_tok.type == "IDENT":
                addr = self._get_var_addr(arg_tok.value)
                self.asm.append(f"    LDA {addr} // {arg_tok.value}")
                self.asm.append(f"    STA {self.tty_port} // TTY_OUT")
            else:
                self.asm.append(f"    LDA #{arg_tok.value}")
                self.asm.append(f"    STA {self.tty_port} // TTY_OUT")

            self._consume("RPAREN")
            return

        # 3. Transferências / Atribuições de Registradores
        if tok.type == "IDENT" and tok.value.upper() in ("A", "X", "Y", "REG_A", "REG_X", "REG_Y") and self._peek().value == "=":
            dest = tok.value.upper()
            self._consume()
            self._consume("OP", "=")
            src_tok = self._cur()
            self.pos += 1

            src = src_tok.value.upper() if src_tok.type == "IDENT" else src_tok.value

            if dest in ("X", "REG_X") and src in ("A", "REG_A"):
                self.asm.append("    TAX")
                return
            elif dest in ("A", "REG_A") and src in ("X", "REG_X"):
                self.asm.append("    TXA")
                return
            elif dest in ("Y", "REG_Y") and src in ("A", "REG_A"):
                self.asm.append("    TAY")
                return
            elif dest in ("A", "REG_A") and src in ("Y", "REG_Y"):
                self.asm.append("    TYA")
                return
            elif dest in ("X", "REG_X"):
                addr = self._get_var_addr(src_tok.value)
                self.asm.append(f"    LDA {addr} // {src_tok.value}")
                self.asm.append("    TAX")
                return
            elif dest in ("Y", "REG_Y"):
                addr = self._get_var_addr(src_tok.value)
                self.asm.append(f"    LDA {addr} // {src_tok.value}")
                self.asm.append("    TAY")
                return
            elif dest in ("A", "REG_A"):
                if src_tok.type == "IDENT":
                    addr = self._get_var_addr(src_tok.value)
                    self.asm.append(f"    LDA {addr} // {src_tok.value}")
                else:
                    self.asm.append(f"    LDA #{src_tok.value}")
                return

        # 4. Incremento / Decremento: var++ ou var--
        if tok.type == "IDENT" and self._peek().value in ("++", "--"):
            var_name = self._consume().value
            op = self._consume().value
            addr = self._get_var_addr(var_name)
            self.asm.append(f"    LDA {addr} // {var_name}")
            self.asm.append("    INC" if op == "++" else "    DEC")
            self.asm.append(f"    STA {addr} // {var_name}")
            return

        # 5. Condicional: se <cond> { ... } [senao { ... }]
        if tok.type == "IDENT" and tok.value == "se":
            self._consume()
            self._parse_if()
            return

        # 6. Laço: para <cond> { ... }
        if tok.type == "IDENT" and tok.value == "para":
            self._consume()
            self._parse_for()
            return

        # 7. Acesso à Memória indexada: mem[addr] = expr
        if tok.type == "IDENT" and tok.value.lower() in ("mem", "memoria") and self._peek().type == "LBRACKET":
            self._consume()
            self._consume("LBRACKET")
            addr_tok = self._cur()
            self.pos += 1
            self._consume("RBRACKET")
            self._consume("OP", "=")
            target_addr = addr_tok.value
            self._parse_expression()
            self.asm.append(f"    STA {target_addr}")
            return

        # 8. Declaração / Atribuição: var := expr ou var = expr
        if tok.type == "IDENT" and self._peek().value in (":=", "="):
            var_name = self._consume().value
            self._consume("OP")
            target_addr = self._get_var_addr(var_name)
            self._parse_expression()
            self.asm.append(f"    STA {target_addr} // {var_name}")
            return

        # 9. Palavra 'var' explícita: var x = 10
        if tok.type == "IDENT" and tok.value in ("var", "variavel"):
            self._consume()
            var_name = self._consume("IDENT").value
            self._consume("OP", "=")
            target_addr = self._get_var_addr(var_name)
            self._parse_expression()
            self.asm.append(f"    STA {target_addr} // {var_name}")
            return

        raise SyntaxError(f"Linha {tok.line}: Comando não reconhecido próximo a '{tok.value}'")

    def _parse_expression(self):
        left_tok = self._cur()
        self.pos += 1

        if self._cur().type == "OP" and self._cur().value in ("+", "-"):
            op_tok = self._consume()
            right_tok = self._cur()
            self.pos += 1

            if left_tok.type == "IDENT":
                addr = self._get_var_addr(left_tok.value)
                self.asm.append(f"    LDA {addr} // {left_tok.value}")
            else:
                self.asm.append(f"    LDA #{left_tok.value}")

            inst = "ADD" if op_tok.value == "+" else "SUB"
            if right_tok.type == "IDENT":
                addr = self._get_var_addr(right_tok.value)
                self.asm.append(f"    {inst} {addr} // {right_tok.value}")
            else:
                self.asm.append(f"    {inst} #{right_tok.value}")
        else:
            if left_tok.type == "IDENT":
                addr = self._get_var_addr(left_tok.value)
                self.asm.append(f"    LDA {addr} // {left_tok.value}")
            else:
                self.asm.append(f"    LDA #{left_tok.value}")

    def _parse_if(self):
        lbl_else = self._get_label("L_ELSE")
        lbl_endif = self._get_label("L_ENDIF")

        cond_tokens = []
        while self._cur().type != "LBRACE" and self._cur().type != "EOF":
            cond_tokens.append(self._cur())
            self.pos += 1

        self._consume("LBRACE")
        self._compile_condition_tokens(cond_tokens, lbl_else)

        self._parse_block(until_rbrace=True)
        self._consume("RBRACE")

        self._skip_terminators()
        has_else = False
        if self._cur().type == "IDENT" and self._cur().value == "senao":
            has_else = True
            self._consume()
            self._consume("LBRACE")
            self.asm.append(f"    JMP {lbl_endif}")
            self.asm.append(f"\n{lbl_else}:")
            self._parse_block(until_rbrace=True)
            self._consume("RBRACE")
            self.asm.append(f"\n{lbl_endif}:")
        else:
            self.asm.append(f"\n{lbl_else}:")

    def _parse_for(self):
        lbl_start = self._get_label("L_LOOP")
        lbl_end = self._get_label("L_ENDLOOP")

        cond_tokens = []
        while self._cur().type != "LBRACE" and self._cur().type != "EOF":
            cond_tokens.append(self._cur())
            self.pos += 1

        self._consume("LBRACE")
        self.asm.append(f"\n{lbl_start}:")
        self._compile_condition_tokens(cond_tokens, lbl_end)

        self._parse_block(until_rbrace=True)
        self._consume("RBRACE")

        self.asm.append(f"    JMP {lbl_start}")
        self.asm.append(f"\n{lbl_end}:")

    def _compile_condition_tokens(self, tokens: List[Token], jump_if_false: str):
        if len(tokens) == 3:
            left, op, right = tokens[0], tokens[1], tokens[2]

            if left.type == "IDENT":
                addr = self._get_var_addr(left.value)
                self.asm.append(f"    LDA {addr} // {left.value}")
            else:
                self.asm.append(f"    LDA #{left.value}")

            if right.type == "IDENT":
                addr = self._get_var_addr(right.value)
                self.asm.append(f"    SUB {addr} // {right.value}")
            else:
                self.asm.append(f"    SUB #{right.value}")

            if op.value == "==":
                self.asm.append(f"    BNE {jump_if_false}")
            elif op.value == "!=":
                self.asm.append(f"    BEQ {jump_if_false}")
            elif op.value == "<":
                self.asm.append(f"    BEQ {jump_if_false}")
                self.asm.append(f"    BPL {jump_if_false}")
            elif op.value == ">":
                self.asm.append(f"    BEQ {jump_if_false}")
                self.asm.append(f"    BMI {jump_if_false}")
        elif len(tokens) == 1:
            tok = tokens[0]
            if tok.type == "IDENT":
                addr = self._get_var_addr(tok.value)
                self.asm.append(f"    LDA {addr} // {tok.value}")
            else:
                self.asm.append(f"    LDA #{tok.value}")
            self.asm.append(f"    BEQ {jump_if_false}")


def main():
    parser = argparse.ArgumentParser(
        description="Compilador VAMOS-T: Converte VAMOS-LANG para Assembly 6502-T"
    )
    parser.add_argument("file", nargs="?", default=None, help="Arquivo-fonte .vamos")
    parser.add_argument("-o", "--output", default=None, help="Arquivo de saída (.s, .asm ou .mem)")
    parser.add_argument("-f", "--format", choices=["asm", "mem"], default="asm", help="Formato de saída (padrão: asm)")
    parser.add_argument("--var-base", type=int, default=180, help="Endereço inicial da memória RAM para variáveis (padrão: 180)")
    parser.add_argument("--tty-port", type=int, default=300, help="Endereço MMIO para escrita TTY (padrão: 300)")
    parser.add_argument("-r", "--run", action="store_true", help="Executa o programa compilado no emulador 6502-T")
    parser.add_argument("--cycles", type=int, default=1000, help="Limite de ciclos em execução (padrão: 1000)")
    parser.add_argument("--json", action="store_true", help="Retorna saída estruturada em JSON")

    args = parser.parse_args()

    if args.file:
        if not os.path.exists(args.file):
            print(f"[X] Arquivo não encontrado: {args.file}", file=sys.stderr)
            sys.exit(1)
        with open(args.file, "r", encoding="utf-8") as f:
            src = f.read()
    elif not sys.stdin.isatty():
        src = sys.stdin.read()
    else:
        # Exemplo padrão demonstrativo
        src = """
        pacote principal

        funcao principal() {
            a := 0z011       // +13
            b := 0z0BB       // +143
            soma := a + b    // +156

            se soma == 0z110 {
                resultado := 0z042 // +50
                A = resultado
                X = A
            } senao {
                resultado := -1
            }

            contador := 3
            para contador != 0 {
                contador--
            }

            parar()
        }
        """

    compiler = VamosCompiler6502T(var_base_addr=args.var_base, tty_port=args.tty_port)
    try:
        asm_code = compiler.compile(src)
    except Exception as e:
        if args.json:
            print(json.dumps({"success": False, "error": str(e)}))
        else:
            print(f"[ERRO DE COMPILAÇÃO]: {e}", file=sys.stderr)
        sys.exit(1)

    # Monta com o Assembler 6502-T
    assembler = Assembler6502T()
    assembler.compile(asm_code)
    mem_code = assembler.export_mem()

    if args.output:
        out_fmt = args.format
        if args.output.endswith(".mem"):
            out_fmt = "mem"
        out_content = mem_code if out_fmt == "mem" else asm_code
        os.makedirs(os.path.dirname(os.path.abspath(args.output)) or ".", exist_ok=True)
        with open(args.output, "w", encoding="utf-8") as f:
            f.write(out_content)
        if not args.json:
            print(f"[✓] Compilado com sucesso para '{args.output}' ({len(assembler.machine_code)} palavras).")

    # Execução opcional no emulador
    exec_info = {}
    if args.run or args.json:
        cpu = CPU6502T(mem_size=1024)
        for addr, (b_str, _) in assembler.machine_code.items():
            cpu.memory[addr] = trits_to_int(bin12_to_trits(b_str))

        cycles_run = 0
        tty_out: List[str] = []
        trace_steps: List[Dict[str, Any]] = []

        for _ in range(args.cycles):
            # Captura snapshot no início de cada instrução
            if cpu.state == CPU6502T.S_FETCH_OPCODE:
                # Condição de parada (HLT: JMP HLT)
                if cpu.memory[cpu.pc % 1024] == -292 and cpu.memory[(cpu.pc + 1) % 1024] == cpu.pc:
                    break

                if len(trace_steps) < 250:
                    trace_steps.append({
                        "cycle": cycles_run,
                        "pc": cpu.pc,
                        "acc": cpu.acc,
                        "acc_dozenal": int_to_dozenal(cpu.acc).strip(),
                        "acc_trits": int_to_trits(cpu.acc, 6),
                        "reg_x": cpu.reg_x,
                        "reg_y": cpu.reg_y,
                        "flag_zero": bool(cpu.flag_zero),
                        "flag_sign": int(cpu.flag_sign),
                        "flag_overflow": bool(cpu.flag_overflow),
                        "stdout": "".join(tty_out)
                    })

            cpu.step_cycle()
            cycles_run += 1

            # Captura escrita no terminal TTY (MMIO Porta configurada)
            if cpu.write_en and (cpu.addr_bus % 1024) == compiler.tty_port:
                char_code = cpu.data_out
                if 0 <= char_code <= 255:
                    tty_out.append(chr(char_code))
                else:
                    try:
                        tty_out.append(chr(char_code))
                    except Exception:
                        tty_out.append("?")

        # Snapshot final
        trace_steps.append({
            "cycle": cycles_run,
            "pc": cpu.pc,
            "acc": cpu.acc,
            "acc_dozenal": int_to_dozenal(cpu.acc).strip(),
            "acc_trits": int_to_trits(cpu.acc, 6),
            "reg_x": cpu.reg_x,
            "reg_y": cpu.reg_y,
            "flag_zero": bool(cpu.flag_zero),
            "flag_sign": int(cpu.flag_sign),
            "flag_overflow": bool(cpu.flag_overflow),
            "stdout": "".join(tty_out)
        })

        # Mapeia variáveis
        var_dump = {}
        for vname, vaddr in compiler.variables.items():
            val = cpu.memory[vaddr % 1024]
            var_dump[vname] = {
                "address": vaddr,
                "decimal": val,
                "dozenal": int_to_dozenal(val).strip(),
                "trits": trits_to_str(int_to_trits(val, 6))
            }

        exec_info = {
            "cycles": cycles_run,
            "acc": cpu.acc,
            "acc_dozenal": int_to_dozenal(cpu.acc).strip(),
            "acc_trits": int_to_trits(cpu.acc, 6),
            "reg_x": cpu.reg_x,
            "reg_y": cpu.reg_y,
            "pc": cpu.pc,
            "flag_zero": bool(cpu.flag_zero),
            "flag_sign": int(cpu.flag_sign),
            "flag_overflow": bool(cpu.flag_overflow),
            "stdout": "".join(tty_out),
            "trace": trace_steps,
            "variables": var_dump
        }

    if args.json:
        result = {
            "success": True,
            "assembly": asm_code,
            "words_count": len(assembler.machine_code),
            "execution": exec_info
        }
        print(json.dumps(result, indent=2))
    elif not args.output and not args.run:
        print(asm_code)
    elif args.run and not args.output:
        print("=" * 75)
        print(" EXECUÇÃO VAMOS-T NA CPU 6502-T:")
        print("=" * 75)
        print(f" Ciclos executados: {exec_info['cycles']}")
        print(f" Acumulador (A)   : {exec_info['acc']:+4d} ({int_to_dozenal(exec_info['acc'])})")
        print(f" Índice X         : {exec_info['reg_x']:+4d} ({int_to_dozenal(exec_info['reg_x'])})")
        print(f" Flags Ternárias  : [Z: {int(exec_info['flag_zero'])}] [S: {exec_info['flag_sign']}] [O: {int(exec_info['flag_overflow'])}]")
        if exec_info["stdout"]:
            print(f" Saída TTY MMIO   : {exec_info['stdout']}")
        print("\n Variáveis em Memória:")
        for vname, vdata in exec_info["variables"].items():
            print(f"   * {vname:<12} [${vdata['address']:02d}] = {vdata['decimal']:+4d} ({vdata['dozenal']}) [{vdata['trits']}]")
        print("=" * 75)


if __name__ == "__main__":
    main()
