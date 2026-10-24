`timescale 1ns / 1ps

module trits_to_dozenal (
    input  wire [11:0] trits_in,      // 6 trits balanceados (2 bits por trit)
    output reg         sign,          // 0 = Positivo/Zero, 1 = Negativo
    output reg  [3:0]  d2,            // Dígito de peso 144 (0 a 2)
    output reg  [3:0]  d1,            // Dígito de peso 12  (0 a 11 / 0 a B)
    output reg  [3:0]  d0,            // Dígito de peso 1   (0 a 11 / 0 a B)
    output wire        is_zero        // Flag indicativa de valor nulo
);

    // Codificação de constantes do trit
    localparam [1:0] TRIT_NEG  = 2'b00; // -1
    localparam [1:0] TRIT_ZERO = 2'b01; //  0
    localparam [1:0] TRIT_POS  = 2'b10; // +1

    // Função de extração de valor com sinal de cada trit
    function signed [1:0] trit_val;
        input [1:0] t;
        begin
            case (t)
                TRIT_NEG:  trit_val = -2'sd1;
                TRIT_POS:  trit_val =  2'sd1;
                default:   trit_val =  2'sd0;
            endcase
        end
    endfunction

    reg signed [9:0] v_signed;   // Faixa: -364 a +364 (requer 10 bits com sinal)
    reg        [8:0] v_abs;      // Faixa: 0 a 364 (9 bits sem sinal)
    reg        [8:0] rem_144;

    integer k;

    // 1. Recomposição Ponderada em Base 3 Balanceada
    always @(*) begin
        v_signed = trit_val(trits_in[1:0])   * 10'sd1   +
                   trit_val(trits_in[3:2])   * 10'sd3   +
                   trit_val(trits_in[5:4])   * 10'sd9   +
                   trit_val(trits_in[7:6])   * 10'sd27  +
                   trit_val(trits_in[9:8])   * 10'sd81  +
                   trit_val(trits_in[11:10]) * 10'sd243;

        // 2. Extração de Sinal e Magnitude
        if (v_signed < 0) begin
            sign  = 1'b1;
            v_abs = -v_signed;
        end else begin
            sign  = 1'b0;
            v_abs = v_signed[8:0];
        end

        // 3. Decomposição Duodecimal (Base 12) por Subtração de Múltiplos
        // Dígito D2 (Grosas: peso 144)
        if (v_abs >= 9'd288) begin
            d2      = 4'd2;
            rem_144 = v_abs - 9'd288;
        end else if (v_abs >= 9'd144) begin
            d2      = 4'd1;
            rem_144 = v_abs - 9'd144;
        end else begin
            d2      = 4'd0;
            rem_144 = v_abs;
        end

        // Dígito D1 (Dúzias: peso 12) e Dígito D0 (Unidades: peso 1)
        d1 = rem_144 / 12;
        d0 = rem_144 % 12;
    end

    assign is_zero = (v_signed == 10'sd0);

endmodule
