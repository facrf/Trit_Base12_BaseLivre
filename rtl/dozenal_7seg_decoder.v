`timescale 1ns / 1ps

module dozenal_7seg_decoder (
    input  wire [3:0] digit,      // Dígito 0 a 11 (0 a B)
    input  wire       is_sign,     // 1: Força exibição de caractere de sinal
    input  wire       blank,       // 1: Apaga display (blanking)
    output reg  [6:0] seg          // Segmentos {g, f, e, d, c, b, a}
);

    // Mapeamento Anodo Comum (0 = Aceso, 1 = Apagado)
    localparam [6:0] SEG_0     = 7'b100_0000;
    localparam [6:0] SEG_1     = 7'b111_1001;
    localparam [6:0] SEG_2     = 7'b010_0100;
    localparam [6:0] SEG_3     = 7'b011_0000;
    localparam [6:0] SEG_4     = 7'b001_1001;
    localparam [6:0] SEG_5     = 7'b001_0010;
    localparam [6:0] SEG_6     = 7'b000_0010;
    localparam [6:0] SEG_7     = 7'b111_1000;
    localparam [6:0] SEG_8     = 7'b000_0000;
    localparam [6:0] SEG_9     = 7'b001_0000;
    localparam [6:0] SEG_A     = 7'b000_1000; // 'A' Maiúsculo
    localparam [6:0] SEG_B     = 7'b000_0011; // 'b' Minúsculo
    localparam [6:0] SEG_MINUS = 7'b011_1111; // Apenas segmento 'g' aceso
    localparam [6:0] SEG_BLANK = 7'b111_1111; // Display totalmente apagado

    always @(*) begin
        if (blank) begin
            seg = SEG_BLANK;
        end else if (is_sign) begin
            seg = SEG_MINUS;
        end else begin
            case (digit)
                4'd0:    seg = SEG_0;
                4'd1:    seg = SEG_1;
                4'd2:    seg = SEG_2;
                4'd3:    seg = SEG_3;
                4'd4:    seg = SEG_4;
                4'd5:    seg = SEG_5;
                4'd6:    seg = SEG_6;
                4'd7:    seg = SEG_7;
                4'd8:    seg = SEG_8;
                4'd9:    seg = SEG_9;
                4'd10:   seg = SEG_A;
                4'd11:   seg = SEG_B;
                default: seg = SEG_BLANK;
            endcase
        end
    end

endmodule
