`timescale 1ns / 1ps

module tfa_1trit (
    input  wire [1:0] a,      // Trit A   (2'b00 = -1, 2'b01 = 0, 2'b10 = +1)
    input  wire [1:0] b,      // Trit B   (2'b00 = -1, 2'b01 = 0, 2'b10 = +1)
    input  wire [1:0] cin,    // Trit Cin (2'b00 = -1, 2'b01 = 0, 2'b10 = +1)
    output reg  [1:0] s,      // Trit Soma
    output reg  [1:0] cout    // Trit Transporte (Cout)
);

    // Codificação de constantes de estado
    localparam [1:0] TRIT_NEG  = 2'b00; // -1
    localparam [1:0] TRIT_ZERO = 2'b01; //  0
    localparam [1:0] TRIT_POS  = 2'b10; // +1

    // Sinais internos com sinal (largura suficiente para -3 até +3)
    reg signed [1:0] val_a;
    reg signed [1:0] val_b;
    reg signed [1:0] val_cin;
    reg signed [3:0] s_int;
    reg signed [1:0] s_calc;

    // 1. Decodificação dos 2 bits para inteiro com sinal
    always @(*) begin
        case (a)
            TRIT_NEG:  val_a = -2'sd1;
            TRIT_ZERO: val_a =  2'sd0;
            TRIT_POS:  val_a =  2'sd1;
            default:   val_a =  2'sd0;
        endcase

        case (b)
            TRIT_NEG:  val_b = -2'sd1;
            TRIT_ZERO: val_b =  2'sd0;
            TRIT_POS:  val_b =  2'sd1;
            default:   val_b =  2'sd0;
        endcase

        case (cin)
            TRIT_NEG:  val_cin = -2'sd1;
            TRIT_ZERO: val_cin =  2'sd0;
            TRIT_POS:  val_cin =  2'sd1;
            default:   val_cin =  2'sd0;
        endcase
    end

    // 2. Aritmética de Limiar Ternário e Recodificação para 2 bits
    always @(*) begin
        // Soma aritmética total (-3 <= s_int <= +3)
        s_int = val_a + val_b + val_cin;

        // Determinação do transporte e da soma balanceada
        if (s_int >= 4'sd2) begin
            cout   = TRIT_POS;           // Cout = +1
            s_calc = s_int - 4'sd3;      // S = S_int - 3
        end else if (s_int <= -4'sd2) begin
            cout   = TRIT_NEG;           // Cout = -1
            s_calc = s_int + 4'sd3;      // S = S_int - (-3) = S_int + 3
        end else begin
            cout   = TRIT_ZERO;          // Cout = 0
            s_calc = s_int[1:0];         // S = S_int
        end

        // Mapeamento da soma resultante para 2 bits
        case (s_calc)
            -2'sd1:  s = TRIT_NEG;
             2'sd0:  s = TRIT_ZERO;
             2'sd1:  s = TRIT_POS;
            default: s = TRIT_ZERO;
        endcase
    end

endmodule
