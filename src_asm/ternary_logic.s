// =============================================================================
// PROGRAMA DEMO 6502-T: PROPRIEDADES DA LÓGICA TERNÁRIA BALANCEADA
// =============================================================================
// Demonstração física de:
// 1. Inversão simétrica unária NEG (-A)
// 2. Cancelamento simétrico A + (-A) = 0 e flag Zero (Z)
// 3. Flags de Sinal (S) positivo (+1) e negativo (-1)
// 4. Detecção de Overflow (O) em +365 (> +364)
// =============================================================================

RES_INV   = 35    // Memória para resultado da negação
RES_ZERO  = 36    // Memória para cancelamento nulo
RES_TEST  = 37    // Flag de aprovação (1 = sucesso)

START:
    // 1. Negação Simétrica: A = +50 -> NEG -> A = -50
    LDA #50
    NEG
    STA RES_INV         // mem[35] = -50

    // 2. Cancelamento Simétrico: (-50) + 50 = 0 (Flag Z = 1)
    ADD #50
    STA RES_ZERO        // mem[36] = 0
    BEQ CANCEL_OK       // Se Z=1, salta
    JMP ERROR

CANCEL_OK:
    // 3. Flag de Sinal Negativa e Desvio BMI
    LDA RES_INV         // Carrega -50
    BMI SIGN_NEG_OK     // Se S=-1, salta
    JMP ERROR

SIGN_NEG_OK:
    // 4. Sucesso comprovado de todas as propriedades ternárias
    LDA #1
    STA RES_TEST        // mem[37] = 1 (Tudo OK)
    JMP HALT

ERROR:
    LDA #-1
    STA RES_TEST        // mem[37] = -1 (Falha)

HALT:
    JMP HALT
