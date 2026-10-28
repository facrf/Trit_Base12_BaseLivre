#!/usr/bin/env python3
"""
Suite de Testes Estendida para a CPU Ternária 6502-T
Valida novos opcodes (TYA, NOP, BNE), diretivas (.ORG, .WORD),
comportamento estrito de flags ternárias e reversibilidade do disassembler.
"""

import sys
import os
import time
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass, field

# Garante acesso às ferramentas no diretório tools/
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "tools")))
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

from asm6502t import Assembler6502T
from emu6502t import CPU6502T, bin12_to_trits, trits_to_int, int_to_dozenal
from disasm6502t import Disassembler6502T
from test_6502t_suite import TestCase6502T, TestRunner6502T


def test_roundtrip_verification() -> Tuple[bool, str]:
    """Testa a simetria completa entre Assembler e Disassembler."""
    source_program = """
    START:
        LDA #13
        TAX
        ADD #143
        STA 32
        SUB #156
        BEQ L_15
        LDA #-1
        JMP L_19
    L_15:
        LDA #50
        STA 33
    L_19:
        JMP L_19
    """
    try:
        asm1 = Assembler6502T()
        asm1.compile(source_program)
        mem1 = asm1.export_mem()

        dis = Disassembler6502T()
        clean_src = dis.disassemble(mem1, clean=True)

        asm2 = Assembler6502T()
        asm2.compile(clean_src)
        mem2 = asm2.export_mem()

        words1 = [line.split("//")[0].strip() for line in mem1.splitlines() if line.strip()]
        words2 = [line.split("//")[0].strip() for line in mem2.splitlines() if line.strip()]

        if words1 != words2:
            return False, f"Divergência de palavras: {len(words1)} originais vs {len(words2)} remontadas"
        return True, "Simetria perfeita confirmada (Round-trip)"
    except Exception as e:
        return False, f"Exceção no teste de reversibilidade: {e}"


def build_extended_test_suite() -> TestRunner6502T:
    suite = TestRunner6502T(mem_size=64)

    # 1. Transferência Y/A (TAY e TYA)
    suite.add_test(TestCase6502T(
        name="Transferência de Registrador Y (TAY e TYA)",
        source_code="""
        START:
            LDA #15
            TAY              // Y = 15
            LDA #0           // A = 0
            TYA              // A = 15
        HLT:
            JMP HLT
        """,
        expected_acc=15,
        expected_y=15,
        expected_zero=False,
        expected_sign=1
    ))

    # 2. Operações Unárias e NOP
    suite.add_test(TestCase6502T(
        name="Operações Unárias e Instrução Neutra (INC, DEC, NOP)",
        source_code="""
        START:
            LDA #10
            INC              // A = 11
            NOP              // Sem efeito
            NOP
            INC              // A = 12
            DEC              // A = 11
            DEC              // A = 10
        HLT:
            JMP HLT
        """,
        expected_acc=10,
        expected_zero=False,
        expected_sign=1
    ))

    # 3. Desvio Condicional BNE (Branch on Not Equal / Não-Zero)
    suite.add_test(TestCase6502T(
        name="Desvio Condicional BNE (Salto se Não-Zero)",
        source_code="""
        START:
            LDA #5
            BNE BRANCH_OK
            LDA #(-1)        // Não deve executar
            JMP HLT
        BRANCH_OK:
            LDA #42          // Deve executar
        HLT:
            JMP HLT
        """,
        expected_acc=42,
        expected_zero=False,
        expected_sign=1
    ))

    # 4. Desvio BPL com Zero Estrito (Zero não é positivo em lógica balanceada)
    suite.add_test(TestCase6502T(
        name="Desvio BPL e Validação de Zero Estrito (S=0 não salta)",
        source_code="""
        START:
            LDA #0
            BPL SHOULD_NOT_BRANCH  // 0 tem Sign=0, NÃO deve saltar
            LDA #99                // Deve executar
            JMP HLT
        SHOULD_NOT_BRANCH:
            LDA #(-1)
        HLT:
            JMP HLT
        """,
        expected_acc=99,
        expected_zero=False,
        expected_sign=1
    ))

    # 5. Detecção de Subfluxo e Overflow Negativo (< -364)
    suite.add_test(TestCase6502T(
        name="Detecção de Overflow Negativo em Silício (< -364)",
        source_code="""
        START:
            LDA #(-360)      // Fundo de escala negativo
            SUB #10          // -360 - 10 = -370 -> Transborda (-370 + 729 = +359)
        HLT:
            JMP HLT
        """,
        expected_acc=359,
        expected_overflow=True,
        expected_sign=1
    ))

    # 6. Diretivas .ORG e .WORD
    suite.add_test(TestCase6502T(
        name="Diretivas de Montador (.ORG e .WORD)",
        source_code="""
            .ORG 10
            .WORD 100
            .WORD (-50)
            .ORG 0
        START:
            LDA 10           // Lê valor da tabela no endereço 10
        HLT:
            JMP HLT
        """,
        expected_acc=100,
        expected_memory={10: 100, 11: -50},
        expected_zero=False,
        expected_sign=1
    ))

    # 7. Demonstração de Fibonacci (src_asm/fibonacci.s)
    fib_path = os.path.join(os.path.dirname(__file__), "..", "src_asm", "fibonacci.s")
    if os.path.exists(fib_path):
        with open(fib_path, "r", encoding="utf-8") as f:
            fib_code = f.read()
        suite.add_test(TestCase6502T(
            name="Execução de Fibonacci em Laço (fibonacci.s)",
            source_code=fib_code,
            expected_acc=13,
            expected_memory={45: 8, 46: 13, 47: 13},
            expected_zero=False,
            expected_sign=1,
            max_cycles=350
        ))

    # 8. Demonstração de Lógica Ternária (src_asm/ternary_logic.s)
    ternary_path = os.path.join(os.path.dirname(__file__), "..", "src_asm", "ternary_logic.s")
    if os.path.exists(ternary_path):
        with open(ternary_path, "r", encoding="utf-8") as f:
            ternary_code = f.read()
        suite.add_test(TestCase6502T(
            name="Lógica Ternária e Flags Físicas (ternary_logic.s)",
            source_code=ternary_code,
            expected_acc=1,
            expected_memory={35: -50, 36: 0, 37: 1},
            expected_zero=False,
            expected_sign=1,
            max_cycles=100
        ))

    return suite


def main():
    print("=" * 80)
    print(" SUITE DE TESTES ESTENDIDA - CPU TERNÁRIA 6502-T")
    print("=" * 80)

    # 1. Executa suite estendida de CPU
    runner = build_extended_test_suite()
    cpu_ok = runner.run_all()

    # 2. Executa teste de reversibilidade do disassembler
    print("\n--- TESTE ADICIONAL DE REVERSIBILIDADE DO TOOLCHAIN ---")
    rt_ok, rt_msg = test_roundtrip_verification()
    if rt_ok:
        print(f" [PASS] Reversibilidade Assembler <-> Disassembler: {rt_msg}")
    else:
        print(f" [FAIL] Reversibilidade Assembler <-> Disassembler: {rt_msg}")

    all_passed = cpu_ok and rt_ok
    print("=" * 80)
    print(f" RESULTADO FINAL ESTENDIDO: {'TODOS OS TESTES PASSARAM' if all_passed else 'FALHA EM TESTES'}")
    print("=" * 80)
    sys.exit(0 if all_passed else 1)


if __name__ == "__main__":
    main()
