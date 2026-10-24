// =============================================================================
// PROGRAMA DEMONSTRATIVO 6502-T: ARITMÉTICA DUODECIMAL E BALANCEADA
// =============================================================================

BUFFER_RES = 32

START:
    CLR                     // Acumulador A = 0
    LDA #0z011              // Literal duodecimal: +13 decimal
    TAX                     // Transfere para X
    ADD #0z0BB              // +143 decimal -> A vira +156 (0z110)
    STA BUFFER_RES          // Grava resultado na memória [32]
    
    // Teste de comparação e salto condicional
    SUB #0z110              // 156 - 156 = 0 -> Flag Zero = 1
    BEQ SUCCESS
    LDA #(-1)
    JMP HLT

SUCCESS:
    LDA #0z042              // +50 decimal
    STA (BUFFER_RES + 1)    // Memória [33] = 50

HLT:
    JMP HLT                 // Loop infinito de término
