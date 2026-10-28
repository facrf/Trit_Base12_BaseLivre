// =============================================================================
// PROGRAMA DEMO 6502-T: SÉRIE DE FIBONACCI EM LAÇO COMPACTO
// =============================================================================
// Calcula termos sucessivos de Fibonacci no registrador A e na RAM
// utilizando registradores A e Y e desvio condicional BNE.
// =============================================================================

FIB_FN2 = 45    // F(n-2)
FIB_FN1 = 46    // F(n-1)
FIB_RES = 47    // F(n)

START:
    LDA #0
    STA FIB_FN2         // F(0) = 0
    LDA #1
    STA FIB_FN1         // F(1) = 1
    
    // Y atua como contador de iterações: 6 ciclos adicionais
    LDA #6
    TAY                 // Y = 6

LOOP:
    // F(n) = F(n-2) + F(n-1)
    LDA FIB_FN2
    ADD FIB_FN1
    STA FIB_RES

    // Rotaciona variáveis
    LDA FIB_FN1
    STA FIB_FN2
    LDA FIB_RES
    STA FIB_FN1

    // Decrementa contador Y
    TYA
    DEC
    TAY
    BNE LOOP            // Repete enquanto Y != 0

HALT:
    LDA FIB_RES         // Coloca termo final de Fibonacci no Acumulador (+13 / 0z011)

DONE:
    JMP DONE
