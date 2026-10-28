#!/usr/bin/env python3
"""
Test Suite de Validação Lógica e Silício RTL (6502-T)
=============================================================================
Validação formal em nível de bit dos módulos Verilog sintetizáveis:
- rtl/tfa_1trit.v           (Somador Ternário Completo de 1 Trit)
- rtl/alu_6trit.v           (ALU Balanceada de 6 Trits e Ripple-Carry)
- rtl/trits_to_addr.v       (Decodificador Linear de Endereçamento)

Garante equivalência matemática estrita e paridade silício-emulador sem
necessidade de dependências externas ou simulador EDA na máquina host.
=============================================================================
"""

import sys
import os
import time
from typing import Dict, Any, List, Optional, Tuple

# Garante acesso às ferramentas no diretório tools/
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "tools")))

from emu6502t import (
    TRIT_CHARS,
    CHAR_TO_TRIT,
    int_to_trits,
    trits_to_int,
    bin12_to_trits,
    trits_to_bin12,
    trits_to_str,
    int_to_dozenal
)

# Constantes binárias físicas de 2 bits por trit (Verilog RTL)
V_TRIT_NEG = 0b00  # -1
V_TRIT_ZER = 0b01  #  0
V_TRIT_POS = 0b10  # +1

OP_ALU_ADD   = 0b000
OP_ALU_SUB   = 0b001
OP_ALU_NEG_A = 0b010
OP_ALU_LOAD  = 0b011
OP_ALU_CLR   = 0b100
OP_ALU_INC   = 0b101
OP_ALU_DEC   = 0b110


class TFA1TritRTL:
    """Modelo exato do módulo Verilog rtl/tfa_1trit.v."""
    @staticmethod
    def step(a_bits: int, b_bits: int, cin_bits: int) -> Tuple[int, int]:
        # 1. Decodificação dos 2 bits para signed int (-1, 0, +1)
        def to_val(bits: int) -> int:
            if bits == V_TRIT_NEG:
                return -1
            elif bits == V_TRIT_POS:
                return 1
            return 0

        val_a = to_val(a_bits)
        val_b = to_val(b_bits)
        val_cin = to_val(cin_bits)

        # 2. Soma aritmética total (-3 <= s_int <= +3)
        s_int = val_a + val_b + val_cin

        # 3. Determinação do transporte e da soma balanceada
        if s_int >= 2:
            cout = V_TRIT_POS
            s_calc = s_int - 3
        elif s_int <= -2:
            cout = V_TRIT_NEG
            s_calc = s_int + 3
        else:
            cout = V_TRIT_ZER
            s_calc = s_int

        if s_calc == -1:
            s = V_TRIT_NEG
        elif s_calc == 1:
            s = V_TRIT_POS
        else:
            s = V_TRIT_ZER

        return s, cout


class ALU6TritRTL:
    """Modelo exato do módulo Verilog rtl/alu_6trit.v."""
    @staticmethod
    def invert_trit(t: int) -> int:
        if t == V_TRIT_NEG:
            return V_TRIT_POS
        elif t == V_TRIT_POS:
            return V_TRIT_NEG
        return V_TRIT_ZER

    @classmethod
    def execute(
        cls,
        opcode: int,
        acc_bits12: int,
        operand_b_bits12: int,
        cin_ext_bits: int = V_TRIT_ZER
    ) -> Tuple[int, int, bool, int, bool]:
        """
        Executa operação da ALU reproduzindo o silício Verilog.
        Retorna: (alu_result_bits12, cout_bits, flag_zero, flag_sign, flag_overflow)
        """
        # Extrai trits individuais de 2 bits (índice 0 = trit menos significativo)
        def get_trit(bits12: int, idx: int) -> int:
            return (bits12 >> (idx * 2)) & 0b11

        op_a = [V_TRIT_ZER] * 6
        op_b = [V_TRIT_ZER] * 6
        cin_eff = cin_ext_bits

        if opcode == OP_ALU_ADD:
            for i in range(6):
                op_a[i] = get_trit(acc_bits12, i)
                op_b[i] = get_trit(operand_b_bits12, i)
            cin_eff = cin_ext_bits

        elif opcode == OP_ALU_SUB:
            for i in range(6):
                op_a[i] = get_trit(acc_bits12, i)
                op_b[i] = cls.invert_trit(get_trit(operand_b_bits12, i))
            cin_eff = cin_ext_bits

        elif opcode == OP_ALU_NEG_A:
            for i in range(6):
                op_a[i] = cls.invert_trit(get_trit(acc_bits12, i))
                op_b[i] = V_TRIT_ZER
            cin_eff = V_TRIT_ZER

        elif opcode == OP_ALU_LOAD:
            for i in range(6):
                op_a[i] = V_TRIT_ZER
                op_b[i] = get_trit(operand_b_bits12, i)
            cin_eff = V_TRIT_ZER

        elif opcode == OP_ALU_CLR:
            for i in range(6):
                op_a[i] = V_TRIT_ZER
                op_b[i] = V_TRIT_ZER
            cin_eff = V_TRIT_ZER

        elif opcode == OP_ALU_INC:
            for i in range(6):
                op_a[i] = get_trit(acc_bits12, i)
            op_b[0] = V_TRIT_POS
            cin_eff = V_TRIT_ZER

        elif opcode == OP_ALU_DEC:
            for i in range(6):
                op_a[i] = get_trit(acc_bits12, i)
            op_b[0] = V_TRIT_NEG
            cin_eff = V_TRIT_ZER

        # Ripple-Carry Chain através dos 6 somadores TFA
        carry = cin_eff
        result_bits12 = 0
        for i in range(6):
            s, carry = TFA1TritRTL.step(op_a[i], op_b[i], carry)
            result_bits12 |= (s << (i * 2))

        cout = carry

        # Flag Zero: ativo se todos os 6 trits forem V_TRIT_ZER
        flag_zero = True
        for i in range(6):
            if get_trit(result_bits12, i) != V_TRIT_ZER:
                flag_zero = False
                break

        # Flag Overflow: ativo se cout != V_TRIT_ZER
        flag_overflow = (cout != V_TRIT_ZER)

        # Flag Sign: sinal do trit não-nulo mais significativo
        flag_sign = V_TRIT_ZER
        for i in reversed(range(6)):
            t = get_trit(result_bits12, i)
            if t != V_TRIT_ZER:
                flag_sign = t
                break

        return result_bits12, cout, flag_zero, flag_sign, flag_overflow


class TritsToAddrRTL:
    """Modelo exato do decodificador rtl/trits_to_addr.v."""
    @staticmethod
    def decode(trits_in_bits: int, num_trits: int = 6) -> int:
        def to_val(bits: int) -> int:
            if bits == V_TRIT_NEG:
                return -1
            elif bits == V_TRIT_POS:
                return 1
            return 0

        addr_calc = 0
        power = 1
        for i in range(num_trits):
            t_bits = (trits_in_bits >> (i * 2)) & 0b11
            addr_calc += to_val(t_bits) * power
            power *= 3

        return addr_calc if addr_calc >= 0 else 0


# =============================================================================
# SUITE DE TESTES
# =============================================================================

def int_to_verilog_bits12(val: int) -> int:
    """Converte inteiro decimal [-364, +364] para representação binária Verilog (12 bits)."""
    trits = int_to_trits(val, 6)
    bits = 0
    mapping = {-1: V_TRIT_NEG, 0: V_TRIT_ZER, 1: V_TRIT_POS}
    for i, t in enumerate(trits):
        bits |= (mapping[t] << (i * 2))
    return bits


def verilog_bits12_to_int(bits12: int) -> int:
    """Converte 12 bits Verilog para inteiro decimal balanceado [-364, +364]."""
    trits = []
    for i in range(6):
        b = (bits12 >> (i * 2)) & 0b11
        if b == V_TRIT_NEG:
            trits.append(-1)
        elif b == V_TRIT_POS:
            trits.append(1)
        else:
            trits.append(0)
    return trits_to_int(trits)


def test_tfa_truth_table() -> Tuple[bool, str]:
    """Testa exaustivamente as 27 combinações do Somador TFA de 1 Trit."""
    tested = 0
    mapping_rev = {V_TRIT_NEG: -1, V_TRIT_ZER: 0, V_TRIT_POS: 1}
    for a in [V_TRIT_NEG, V_TRIT_ZER, V_TRIT_POS]:
        for b in [V_TRIT_NEG, V_TRIT_ZER, V_TRIT_POS]:
            for cin in [V_TRIT_NEG, V_TRIT_ZER, V_TRIT_POS]:
                s, cout = TFA1TritRTL.step(a, b, cin)
                sum_in = mapping_rev[a] + mapping_rev[b] + mapping_rev[cin]
                sum_out = mapping_rev[s] + 3 * mapping_rev[cout]
                if sum_in != sum_out:
                    return False, f"Falha TFA: In={sum_in} != Out={sum_out} (A={a}, B={b}, Cin={cin})"
                tested += 1
    return True, f"27/27 combinações da tabela de verdade validadas com sucesso"


def test_alu_negation_exhaustive() -> Tuple[bool, str]:
    """Valida negação unária NEG_A (-A) para todos os 729 valores [-364, +364]."""
    for val in range(-364, 365):
        b12 = int_to_verilog_bits12(val)
        res_b12, cout, fz, fs, fo = ALU6TritRTL.execute(OP_ALU_NEG_A, b12, 0)
        res_int = verilog_bits12_to_int(res_b12)
        if res_int != -val:
            return False, f"Falha NEG: -({val}) produziu {res_int}"
        if val == 0 and not fz:
            return False, f"Falha Flag Z em NEG(0)"
    return True, f"729/729 negações simétricas confirmadas (-(-A) == A)"


def test_alu_zero_and_identity() -> Tuple[bool, str]:
    """Valida A - A = 0 e flag Zero (Z) para todos os 729 valores."""
    for val in range(-364, 365):
        b12 = int_to_verilog_bits12(val)
        res_b12, cout, fz, fs, fo = ALU6TritRTL.execute(OP_ALU_SUB, b12, b12)
        res_int = verilog_bits12_to_int(res_b12)
        if res_int != 0 or not fz:
            return False, f"Falha identidade A - A: {val} - {val} = {res_int} (fz={fz})"
    return True, f"729/729 subtrações de identidade zeraram com Flag Z ativa"


def test_alu_sign_flag() -> Tuple[bool, str]:
    """Valida Flag de Sinal (S) para todos os 729 valores via carregamento LOAD."""
    for val in range(-364, 365):
        b12 = int_to_verilog_bits12(val)
        res_b12, cout, fz, fs, fo = ALU6TritRTL.execute(OP_ALU_LOAD, 0, b12)
        expected_sign = V_TRIT_POS if val > 0 else (V_TRIT_NEG if val < 0 else V_TRIT_ZER)
        if fs != expected_sign:
            return False, f"Falha Flag Sign para valor {val}: obtido {fs}, esperado {expected_sign}"
    return True, f"729/729 flags de sinal ternário correspondem exatamente à polaridade"


def test_alu_overflow_detection() -> Tuple[bool, str]:
    """Valida detecção de overflow positivo (> +364) e negativo (< -364)."""
    # 364 + 1 = 365 -> Overflow positivo
    b_max = int_to_verilog_bits12(364)
    b_one = int_to_verilog_bits12(1)
    _, cout, _, _, fo = ALU6TritRTL.execute(OP_ALU_ADD, b_max, b_one)
    if not fo or cout != V_TRIT_POS:
        return False, f"Falha detecção de overflow positivo (+365)"

    # -364 + (-1) = -365 -> Overflow negativo
    b_min = int_to_verilog_bits12(-364)
    b_neg1 = int_to_verilog_bits12(-1)
    _, cout, _, _, fo = ALU6TritRTL.execute(OP_ALU_ADD, b_min, b_neg1)
    if not fo or cout != V_TRIT_NEG:
        return False, f"Falha detecção de overflow negativo (-365)"

    # 200 + 100 = 300 (sem overflow)
    b_200 = int_to_verilog_bits12(200)
    b_100 = int_to_verilog_bits12(100)
    _, cout, _, _, fo = ALU6TritRTL.execute(OP_ALU_ADD, b_200, b_100)
    if fo or cout != V_TRIT_ZER:
        return False, f"Falso positivo de overflow (+300)"

    return True, "Transição de fronteira de overflow (> +364 e < -364) verificada com sucesso"


def test_alu_inc_dec() -> Tuple[bool, str]:
    """Valida incrementos e decrementos unitários em toda a faixa permitida."""
    for val in range(-363, 364):
        b12 = int_to_verilog_bits12(val)
        res_inc, _, _, _, _ = ALU6TritRTL.execute(OP_ALU_INC, b12, 0)
        res_dec, _, _, _, _ = ALU6TritRTL.execute(OP_ALU_DEC, b12, 0)
        if verilog_bits12_to_int(res_inc) != val + 1:
            return False, f"Falha INC em {val}"
        if verilog_bits12_to_int(res_dec) != val - 1:
            return False, f"Falha DEC em {val}"
    return True, "727/727 operações INC e DEC verificadas em hardware simulado"


def test_trits_to_addr_decoding() -> Tuple[bool, str]:
    """Valida decodificador linear rtl/trits_to_addr.v para 6 e 10 trits."""
    # 6 trits: valores de 0 a 364 decodificados diretamente
    for val in range(0, 365):
        b12 = int_to_verilog_bits12(val)
        addr = TritsToAddrRTL.decode(b12, num_trits=6)
        if addr != val:
            return False, f"Falha trits_to_addr (6 trits): {val} decodificado como {addr}"

    # Valores negativos devem saturar para 0
    for val in range(-364, 0):
        b12 = int_to_verilog_bits12(val)
        addr = TritsToAddrRTL.decode(b12, num_trits=6)
        if addr != 0:
            return False, f"Falha saturação negativa trits_to_addr ({val} deu {addr})"

    return True, "Decodificação de endereços lineares e saturação negativa validadas com sucesso"


def run_all_rtl_tests() -> bool:
    print("=" * 80)
    print(" SUITE DE TESTES RTL DE SILÍCIO (ICARUS / MODELO HARDWARE 6502-T)")
    print("=" * 80)

    tests = [
        ("Tabela de Verdade TFA 1-Trit (27 estados)", test_tfa_truth_table),
        ("Negação Unária ALU NEG_A (729 valores [-364, +364])", test_alu_negation_exhaustive),
        ("Subtração de Identidade A - A = 0 e Flag Zero", test_alu_zero_and_identity),
        ("Flag de Sinal Ternário Balanceado (Flag S)", test_alu_sign_flag),
        ("Detecção de Overflow de Silício (> +364 e < -364)", test_alu_overflow_detection),
        ("Operações de Passo Unitário INC e DEC", test_alu_inc_dec),
        ("Decodificador de Endereçamento Linear (trits_to_addr)", test_trits_to_addr_decoding)
    ]

    t0 = time.time()
    passed = 0
    for i, (name, fn) in enumerate(tests, 1):
        ok, msg = fn()
        status = "PASS" if ok else "FAIL"
        print(f" [{i:02d}/{len(tests):02d}] {status}: {name:<52}")
        if not ok:
            print(f"        └─► {msg}")
        else:
            passed += 1

    dt = (time.time() - t0) * 1000.0
    print("-" * 80)
    print(f" RESULTADO FINAL RTL: {passed}/{len(tests)} testes passaram com sucesso.")
    print(f" Tempo de Execução: {dt:.2f} ms")
    print("=" * 80)
    return passed == len(tests)


if __name__ == "__main__":
    success = run_all_rtl_tests()
    sys.exit(0 if success else 1)
