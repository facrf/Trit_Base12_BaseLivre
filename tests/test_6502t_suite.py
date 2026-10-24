#!/usr/bin/env python3
"""
Suite de Testes Automatizados para a CPU Ternária 6502-T
Integra asm6502t.py e emu6502t.py para validação ponta a ponta.
"""

import sys
import os
import time
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass, field

# Garante acesso às ferramentas no diretório tools/
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "tools")))
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

# Importa os módulos desenvolvidos anteriormente
try:
    from asm6502t import Assembler6502T
    from emu6502t import CPU6502T, bin12_to_trits, trits_to_int, int_to_dozenal
except ImportError:
    print("[ERRO] Certifique-se de que 'asm6502t.py' e 'emu6502t.py' estejam no diretório 'tools/' ou no PYTHONPATH.", file=sys.stderr)
    sys.exit(1)


@dataclass
class TestCase6502T:
    name: str
    source_code: str
    expected_acc: Optional[int] = None
    expected_x: Optional[int] = None
    expected_y: Optional[int] = None
    expected_zero: Optional[bool] = None
    expected_sign: Optional[int] = None      # -1, 0, +1
    expected_overflow: Optional[bool] = None
    expected_memory: Dict[int, int] = field(default_factory=dict)
    max_cycles: int = 400


class TestRunner6502T:
    def __init__(self, mem_size: int = 64):
        self.mem_size = mem_size
        self.tests: List[TestCase6502T] = []

    def add_test(self, test: TestCase6502T):
        self.tests.append(test)

    def run_all(self) -> bool:
        print("=" * 80)
        print(" TEST SUITE INTEGRADO - CPU TERNÁRIA 6502-T (TRIT + BASE 12)")
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
        print(f" RESULTADO FINAL: {passed_count}/{total_tests} testes passaram com sucesso.")
        print(f" Tempo de Execução: {elapsed_total:.2f} ms")
        print("=" * 80)

        return passed_count == total_tests

    def _run_single_test(self, test: TestCase6502T) -> Tuple[bool, List[str], int]:
        failures = []

        # 1. Montagem com o Assembler
        assembler = Assembler6502T(stack_base=self.mem_size - 1)
        try:
            assembler.compile(test.source_code)
        except Exception as e:
            return False, [f"Falha de Montagem (Assembly): {e}"], 0

        # 2. Inicialização da CPU e Injeção de Código
        cpu = CPU6502T(mem_size=self.mem_size)
        for addr, (bin_str, _) in assembler.machine_code.items():
            if addr < self.mem_size:
                val = trits_to_int(bin12_to_trits(bin_str))
                cpu.memory[addr] = val

        # 3. Execução até HLT (Loop de salto para si mesmo) ou estouro de ciclos
        cycles = 0
        last_pc = -1
        halted = False

        while cycles < test.max_cycles:
            # Detecta auto-loop (ex: HALT: JMP HALT) na fase FETCH_OPCODE
            if cpu.state == CPU6502T.S_FETCH_OPCODE:
                current_word = cpu.memory[cpu.pc % self.mem_size]
                # Se opcode for JMP (-292 decimal) e o alvo for o próprio PC
                if current_word == -292:
                    target_addr = cpu.memory[(cpu.pc + 1) % self.mem_size]
                    if target_addr == cpu.pc:
                        halted = True
                        break

            cpu.step_cycle()
            cycles += 1

        if not halted and cycles >= test.max_cycles:
            failures.append(f"Timeout: execução excedeu o limite de {test.max_cycles} ciclos sem atingir HLT.")

        # 4. Verificação de Assertions
        if test.expected_acc is not None and cpu.acc != test.expected_acc:
            failures.append(
                f"Acumulador inválido: esperado {test.expected_acc:+d} ({int_to_dozenal(test.expected_acc)}), "
                f"obtido {cpu.acc:+d} ({int_to_dozenal(cpu.acc)})"
            )

        if test.expected_x is not None and cpu.reg_x != test.expected_x:
            failures.append(f"Registrador X inválido: esperado {test.expected_x}, obtido {cpu.reg_x}")

        if test.expected_y is not None and cpu.reg_y != test.expected_y:
            failures.append(f"Registrador Y inválido: esperado {test.expected_y}, obtido {cpu.reg_y}")

        if test.expected_zero is not None and cpu.flag_zero != test.expected_zero:
            failures.append(f"Flag Zero (Z) inválida: esperada {test.expected_zero}, obtida {cpu.flag_zero}")

        if test.expected_sign is not None and cpu.flag_sign != test.expected_sign:
            failures.append(f"Flag Sign (S) inválida: esperada {test.expected_sign}, obtida {cpu.flag_sign}")

        if test.expected_overflow is not None and cpu.flag_overflow != test.expected_overflow:
            failures.append(f"Flag Overflow (O) inválida: esperada {test.expected_overflow}, obtida {cpu.flag_overflow}")

        for mem_addr, expected_val in test.expected_memory.items():
            actual_val = cpu.memory[mem_addr]
            if actual_val != expected_val:
                failures.append(
                    f"Memória em [${mem_addr:02d}] divergente: esperava {expected_val:+d} "
                    f"({int_to_dozenal(expected_val)}), encontrou {actual_val:+d} ({int_to_dozenal(actual_val)})"
                )

        return len(failures) == 0, failures, cycles


# =====================================================================
# DEFINIÇÃO DOS CENÁRIOS DE TESTE
# =====================================================================

def build_test_suite() -> TestRunner6502T:
    suite = TestRunner6502T(mem_size=64)

    # -----------------------------------------------------------------
    # Teste 1: Aritmética Balanceada e Literais Duodecimais
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Aritmética Básica (LDA + ADD em Base 12)",
        source_code="""
        START:
            LDA #0z011       // +13 decimal
            ADD #0z0BB       // +143 decimal
        HLT:
            JMP HLT          // Total esperado: 156 (0z110)
        """,
        expected_acc=156,
        expected_sign=1,
        expected_zero=False,
        expected_overflow=False
    ))

    # -----------------------------------------------------------------
    # Teste 2: Negação Direta e Aritmética com Números Negativos
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Negação Unária e Flag Sign Negativa (NEG + SUB)",
        source_code="""
        START:
            LDA #40
            NEG              // A vira -40
            SUB #10          // A vira -50 (-40 - 10)
        HLT:
            JMP HLT
        """,
        expected_acc=-50,
        expected_sign=-1,
        expected_zero=False,
        expected_overflow=False
    ))

    # -----------------------------------------------------------------
    # Teste 3: Transferências e Escrita/Leitura de Memória (TAX, STA, LDA)
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Transferência de Registradores e Memória (TAX, STA, LDA)",
        source_code="""
        TARGET = 35
        START:
            LDA #42
            TAX              // X = 42
            STA TARGET       // Memória[35] = 42
            LDA #0           // A = 0
            LDA TARGET       // Recarrega A da Memória[35]
        HLT:
            JMP HLT
        """,
        expected_acc=42,
        expected_x=42,
        expected_memory={35: 42},
        expected_zero=False,
        expected_sign=1
    ))

    # -----------------------------------------------------------------
    # Teste 4: Desvio Condicional BEQ (Branch on Equal / Zero)
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Desvio Condicional BEQ com Ativação de Flag Zero",
        source_code="""
        START:
            LDA #25
            SUB #25          // A = 0 -> Flag Zero = 1
            BEQ BRANCH_OK
            LDA #(-1)        // Não deve executar
            JMP HLT
        BRANCH_OK:
            LDA #77          // Deve executar
        HLT:
            JMP HLT
        """,
        expected_acc=77,
        expected_zero=False,
        expected_sign=1
    ))

    # -----------------------------------------------------------------
    # Teste 5: Desvio Condicional BMI (Branch on Minus / Sign -1)
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Desvio Condicional BMI com Número Negativo",
        source_code="""
        START:
            LDA #10
            SUB #30          // A = -20 -> Flag Sign = -1
            BMI BRANCH_OK
            LDA #99          // Não deve executar
            JMP HLT
        BRANCH_OK:
            LDA #(-77)       // Deve executar
        HLT:
            JMP HLT
        """,
        expected_acc=-77,
        expected_sign=-1,
        expected_zero=False
    ))

    # -----------------------------------------------------------------
    # Teste 6: Pseudo-Instruções e Pilha (CLR, CMP, PUSH, POP)
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Pseudo-Instruções (CLR, CMP, PUSH, POP)",
        source_code="""
        DATA_LOC = 30
        START:
            CLR              // Pseudo: A = 0
            LDA #100
            PUSH             // Salva 100 na pilha (Stack base = 63)
            LDA #50
            PUSH             // Salva 50 na pilha (Stack base - 1 = 62)
            
            CMP #50          // Pseudo: TAX + SUB #50 -> Zera flag sem perder X
            BEQ RESTORE
            JMP HLT
        RESTORE:
            POP              // Desempilha 50 em A
            STA (DATA_LOC)   // Memória[30] = 50
            POP              // Desempilha 100 em A
            STA (DATA_LOC+1) // Memória[31] = 100
        HLT:
            JMP HLT
        """,
        expected_acc=100,
        expected_memory={63: 100, 62: 50, 30: 50, 31: 100},
        expected_zero=False,
        expected_sign=1
    ))

    # -----------------------------------------------------------------
    # Teste 7: Detecção de Overflow Físico (Além de 6 Trits: > +364)
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Detecção de Overflow Ternário (> +364)",
        source_code="""
        START:
            LDA #360         // Próximo do fundo de escala (+364)
            ADD #10          // 360 + 10 = 370 -> Transborda capacidade de 6 trits
        HLT:
            JMP HLT
        """,
        expected_acc=-359,   # 370 - 729 = -359 (comportamento de saturação do trit)
        expected_overflow=True,
        expected_sign=-1
    ))

    return suite


if __name__ == "__main__":
    test_runner = build_test_suite()
    all_passed = test_runner.run_all()
    sys.exit(0 if all_passed else 1)