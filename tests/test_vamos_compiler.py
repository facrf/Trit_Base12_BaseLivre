#!/usr/bin/env python3
"""
Suite de Testes Automatizada para o Compilador VAMOS-T (6502-T)
Valida compilação, literais duodecimais, condicionais relacionais,
laços de repetição, negação unária e MMIO TTY.
"""

import sys
import os
import time
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass, field

# Garante acesso às ferramentas do repositório
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "tools")))
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

from vamos6502t import VamosCompiler6502T
from asm6502t import Assembler6502T
from emu6502t import CPU6502T, bin12_to_trits, trits_to_int, int_to_dozenal


@dataclass
class VamosTestCase:
    name: str
    source_code: str
    expected_vars: Dict[str, int] = field(default_factory=dict)
    expected_stdout: Optional[str] = None
    expected_acc: Optional[int] = None
    max_cycles: int = 1500


class VamosTestRunner:
    def __init__(self, mem_size: int = 1024):
        self.mem_size = mem_size
        self.tests: List[VamosTestCase] = []

    def add_test(self, test: VamosTestCase):
        self.tests.append(test)

    def run_all(self) -> bool:
        print("=" * 80)
        print(" SUITE DE TESTES AUTOMATIZADA - COMPILADOR VAMOS-T (6502-T)")
        print("=" * 80)

        passed_count = 0
        total_tests = len(self.tests)
        start_time = time.perf_counter()

        for idx, test in enumerate(self.tests, 1):
            success, failure_reasons, elapsed_cycles = self._run_single_test(test)
            if success:
                passed_count += 1
                print(f" [{idx:02d}/{total_tests:02d}] PASS: {test.name:<45} ({elapsed_cycles:3d} ciclos)")
            else:
                print(f" [{idx:02d}/{total_tests:02d}] FAIL: {test.name}")
                for reason in failure_reasons:
                    print(f"        └─► {reason}")

        elapsed_total = (time.perf_counter() - start_time) * 1000.0
        print("-" * 80)
        print(f" RESULTADO FINAL VAMOS-T: {passed_count}/{total_tests} testes passaram com sucesso.")
        print(f" Tempo de Execução: {elapsed_total:.2f} ms")
        print("=" * 80)

        return passed_count == total_tests

    def _run_single_test(self, test: VamosTestCase) -> Tuple[bool, List[str], int]:
        failures: List[str] = []

        # 1. Compilação de VAMOS-LANG para Assembly 6502-T
        compiler = VamosCompiler6502T(var_base_addr=180, tty_port=300)
        try:
            asm_code = compiler.compile(test.source_code)
        except Exception as e:
            return False, [f"Falha de compilação VAMOS-T: {e}"], 0

        # 2. Montagem do Assembly para binário da CPU
        assembler = Assembler6502T()
        try:
            assembler.compile(asm_code)
        except Exception as e:
            return False, [f"Falha de montagem (asm6502t): {e}"], 0

        # 3. Carga e execução no emulador
        cpu = CPU6502T(mem_size=self.mem_size)
        for addr, (b_str, _) in assembler.machine_code.items():
            if addr < self.mem_size:
                cpu.memory[addr] = trits_to_int(bin12_to_trits(b_str))

        cycles = 0
        halted = False
        tty_chars: List[str] = []

        while cycles < test.max_cycles:
            # Detecta instrução de parada (HLT: JMP HLT)
            if cpu.state == CPU6502T.S_FETCH_OPCODE:
                current_word = cpu.memory[cpu.pc % self.mem_size]
                if current_word == -292:  # JMP
                    target_addr = cpu.memory[(cpu.pc + 1) % self.mem_size]
                    if target_addr == cpu.pc:
                        halted = True
                        break

            cpu.step_cycle()
            cycles += 1

            # Captura escrita MMIO no TTY (Porta 300)
            if cpu.write_en and (cpu.addr_bus % self.mem_size) == compiler.tty_port:
                ch_code = cpu.data_out
                if 0 <= ch_code <= 255:
                    tty_chars.append(chr(ch_code))

        if not halted and cycles >= test.max_cycles:
            failures.append(f"Timeout: execução excedeu {test.max_cycles} ciclos sem atingir HLT.")

        # 4. Asserção das variáveis de memória
        for vname, expected_val in test.expected_vars.items():
            norm_name = vname.lower()
            if norm_name not in compiler.variables:
                failures.append(f"Variável '{vname}' não foi encontrada na tabela de símbolos.")
                continue

            vaddr = compiler.variables[norm_name]
            actual_val = cpu.memory[vaddr % self.mem_size]
            if actual_val != expected_val:
                failures.append(
                    f"Variável '{vname}' [${vaddr:02d}] inválida: esperado {expected_val:+d} "
                    f"({int_to_dozenal(expected_val).strip()}), obtido {actual_val:+d} "
                    f"({int_to_dozenal(actual_val).strip()})"
                )

        # 5. Asserção de saída TTY
        if test.expected_stdout is not None:
            actual_stdout = "".join(tty_chars)
            if actual_stdout != test.expected_stdout:
                failures.append(f"Saída TTY divergente: esperado '{test.expected_stdout}', obtido '{actual_stdout}'")

        # 6. Asserção do Acumulador (opcional)
        if test.expected_acc is not None and cpu.acc != test.expected_acc:
            failures.append(
                f"Acumulador A inválido: esperado {test.expected_acc}, obtido {cpu.acc}"
            )

        return len(failures) == 0, failures, cycles


def build_vamos_test_suite() -> VamosTestRunner:
    runner = VamosTestRunner(mem_size=1024)

    # -----------------------------------------------------------------
    # Teste 1: Aritmética Duodecimal e Atribuição
    # -----------------------------------------------------------------
    runner.add_test(VamosTestCase(
        name="Aritmética Duodecimal e Atribuição Simples",
        source_code="""
        pacote principal
        funcao principal() {
            a := 0z011       // +13
            b := 0z0BB       // +143
            soma := a + b    // +156 (0z110)
            parar()
        }
        """,
        expected_vars={"a": 13, "b": 143, "soma": 156}
    ))

    # -----------------------------------------------------------------
    # Teste 2: Negação Unária de Variáveis e Literais
    # -----------------------------------------------------------------
    runner.add_test(VamosTestCase(
        name="Negação Unária via Instrução Nativa NEG",
        source_code="""
        pacote principal
        funcao principal() {
            positivo := 50
            negativo := -positivo // -50
            offset := -negativo + 10 // +60
            parar()
        }
        """,
        expected_vars={"positivo": 50, "negativo": -50, "offset": 60}
    ))

    # -----------------------------------------------------------------
    # Teste 3: Condicionais de Igualdade (== e !=)
    # -----------------------------------------------------------------
    runner.add_test(VamosTestCase(
        name="Condicionais de Igualdade (== e != com senao)",
        source_code="""
        pacote principal
        funcao principal() {
            x := 10
            r1 := 0
            r2 := 0

            se x == 10 {
                r1 := 1
            } senao {
                r1 := -1
            }

            se x != 10 {
                r2 := -1
            } senao {
                r2 := 1
            }
            parar()
        }
        """,
        expected_vars={"r1": 1, "r2": 1}
    ))

    # -----------------------------------------------------------------
    # Teste 4: Condicionais Estritas (< e >)
    # -----------------------------------------------------------------
    runner.add_test(VamosTestCase(
        name="Condicionais de Desigualdade Estrita (< e >)",
        source_code="""
        pacote principal
        funcao principal() {
            a := 5
            b := 12
            menor := 0
            maior := 0

            se a < b {
                menor := 1
            }
            se b > a {
                maior := 1
            }
            se a > b {
                menor := 99
            }
            parar()
        }
        """,
        expected_vars={"menor": 1, "maior": 1}
    ))

    # -----------------------------------------------------------------
    # Teste 5: Operadores Não-Estritos (<= e >=)
    # -----------------------------------------------------------------
    runner.add_test(VamosTestCase(
        name="Operadores Relacionais Compostos (<= e >=)",
        source_code="""
        pacote principal
        funcao principal() {
            val := 20
            ok_le1 := 0
            ok_le2 := 0
            ok_ge1 := 0
            ok_ge2 := 0

            se val <= 20 {
                ok_le1 := 1 // Igualdade: deve entrar
            }
            se val <= 25 {
                ok_le2 := 1 // Menor: deve entrar
            }
            se val >= 20 {
                ok_ge1 := 1 // Igualdade: deve entrar
            }
            se val >= 15 {
                ok_ge2 := 1 // Maior: deve entrar
            }
            parar()
        }
        """,
        expected_vars={"ok_le1": 1, "ok_le2": 1, "ok_ge1": 1, "ok_ge2": 1}
    ))

    # -----------------------------------------------------------------
    # Teste 6: Laço 'para' com Decremento e Acumulador
    # -----------------------------------------------------------------
    runner.add_test(VamosTestCase(
        name="Laço de Repetição 'para' com Decremento",
        source_code="""
        pacote principal
        funcao principal() {
            acumulador := 0
            i := 5
            para i != 0 {
                acumulador := acumulador + 10
                i--
            }
            parar()
        }
        """,
        expected_vars={"acumulador": 50, "i": 0}
    ))

    # -----------------------------------------------------------------
    # Teste 7: Saída MMIO para Terminal CRT TTY
    # -----------------------------------------------------------------
    runner.add_test(VamosTestCase(
        name="Saída Formatada MMIO TTY (Porta 300)",
        source_code="""
        pacote principal
        funcao principal() {
            escrever("6502-T OK!")
            parar()
        }
        """,
        expected_stdout="6502-T OK!"
    ))

    # -----------------------------------------------------------------
    # Teste 8: Indexação Bidirecional de Memória (Leitura, Escrita e Condição)
    # -----------------------------------------------------------------
    runner.add_test(VamosTestCase(
        name="Acesso e Aritmética Direta de Memória mem[addr]",
        source_code="""
        pacote principal
        funcao principal() {
            mem[32] = 100
            mem[33] = 43
            lido := mem[32]
            soma := mem[32] + mem[33]
            cond_ok := 0

            se mem[32] == 100 {
                cond_ok := 1
            }
            parar()
        }
        """,
        expected_vars={"lido": 100, "soma": 143, "cond_ok": 1}
    ))

    # -----------------------------------------------------------------
    # Teste 9: Declaração Explícita de Constantes e Variáveis (var, const, constante)
    # -----------------------------------------------------------------
    runner.add_test(VamosTestCase(
        name="Declaração de Constantes e Variáveis (var, const, constante)",
        source_code="""
        pacote principal
        funcao principal() {
            var taxa = 12
            const fator = 5
            constante offset := 20
            resultado := taxa + fator + offset
            parar()
        }
        """,
        expected_vars={"taxa": 12, "fator": 5, "offset": 20, "resultado": 37}
    ))

    return runner


if __name__ == "__main__":
    runner = build_vamos_test_suite()
    success = runner.run_all()
    sys.exit(0 if success else 1)
