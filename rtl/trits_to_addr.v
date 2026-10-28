`timescale 1ns / 1ps

module trits_to_addr #(
    parameter NUM_TRITS  = 6,
    parameter ADDR_WIDTH = 10
) (
    input  wire [NUM_TRITS*2-1:0] trits_in,
    output wire [ADDR_WIDTH-1:0]  addr_out
);

    localparam [1:0] TRIT_NEG  = 2'b00; // -1
    localparam [1:0] TRIT_ZERO = 2'b01; //  0
    localparam [1:0] TRIT_POS  = 2'b10; // +1

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

    reg signed [15:0] addr_calc;

    always @(*) begin
        addr_calc = 16'sd0;
        if (NUM_TRITS >= 1)  addr_calc = addr_calc + trit_val(trits_in[1:0])   * 16'sd1;
        if (NUM_TRITS >= 2)  addr_calc = addr_calc + trit_val(trits_in[3:2])   * 16'sd3;
        if (NUM_TRITS >= 3)  addr_calc = addr_calc + trit_val(trits_in[5:4])   * 16'sd9;
        if (NUM_TRITS >= 4)  addr_calc = addr_calc + trit_val(trits_in[7:6])   * 16'sd27;
        if (NUM_TRITS >= 5)  addr_calc = addr_calc + trit_val(trits_in[9:8])   * 16'sd81;
        if (NUM_TRITS >= 6)  addr_calc = addr_calc + trit_val(trits_in[11:10]) * 16'sd243;
        if (NUM_TRITS >= 7)  addr_calc = addr_calc + trit_val(trits_in[13:12]) * 16'sd729;
        if (NUM_TRITS >= 8)  addr_calc = addr_calc + trit_val(trits_in[15:14]) * 16'sd2187;
        if (NUM_TRITS >= 9)  addr_calc = addr_calc + trit_val(trits_in[17:16]) * 16'sd6561;
        if (NUM_TRITS >= 10) addr_calc = addr_calc + trit_val(trits_in[19:18]) * 16'sd19683;
    end

    // Mapeamento para faixa positiva de endereçamento sem sinal
    assign addr_out = (addr_calc >= 0) ? addr_calc[ADDR_WIDTH-1:0] : {ADDR_WIDTH{1'b0}};

endmodule
