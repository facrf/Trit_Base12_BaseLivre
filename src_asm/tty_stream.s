// =============================================================================
// PROGRAMA DEMO 6502-T: STREAMING DE TEXTO VIA MMIO TTY (PORTA 300)
// =============================================================================
// Transmite a sequência de caracteres ASCII "6502-T OK!\n" para a porta MMIO.
// =============================================================================

TTY_PORT = 300   // Endereço de saída de telemetria CRT / Terminal TTY (0z210)

START:
    LDA #54      // '6' (ASCII 54)
    STA TTY_PORT
    LDA #53      // '5' (ASCII 53)
    STA TTY_PORT
    LDA #48      // '0' (ASCII 48)
    STA TTY_PORT
    LDA #50      // '2' (ASCII 50)
    STA TTY_PORT
    LDA #45      // '-' (ASCII 45)
    STA TTY_PORT
    LDA #84      // 'T' (ASCII 84)
    STA TTY_PORT
    LDA #32      // ' ' (ASCII 32)
    STA TTY_PORT
    LDA #79      // 'O' (ASCII 79)
    STA TTY_PORT
    LDA #75      // 'K' (ASCII 75)
    STA TTY_PORT
    LDA #33      // '!' (ASCII 33)
    STA TTY_PORT
    LDA #10      // '\n' (ASCII 10)
    STA TTY_PORT

HALT:
    JMP HALT
