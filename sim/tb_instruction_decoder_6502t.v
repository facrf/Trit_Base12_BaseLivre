`timescale 1ns / 1ps

module tb_instruction_decoder_6502t;

    reg         clk;
    reg         rst_n;
    reg  [11:0] mem_data_in;
    wire [19:0] mem_addr;
    wire [11:0] mem_data_out;
    wire        mem_read;
    wire        mem_write;

    reg         flag_zero;
    reg  [1:0]  flag_sign;
    reg         flag_overflow;

    wire [2:0]  alu_opcode;
    wire        alu_load_acc;
    wire [11:0] alu_operand_b;
    wire [1:0]  alu_cin_ext;
    wire        reg_x_load;
    wire        reg_y_load;

    reg  [11:0] acc_val;
    reg  [11:0] reg_x_val;
    reg  [11:0] reg_y_val;

    localparam [1:0] T_NEG = 2'b00;
    localparam [1:0] T_ZER = 2'b01;
    localparam [1:0] T_POS = 2'b10;

    instruction_decoder_6502t uut (
        .clk          (clk),
        .rst_n        (rst_n),
        .clk_en       (1'b1),
        .mem_data_in  (mem_data_in),
        .mem_addr     (mem_addr),
        .mem_data_out (mem_data_out),
        .mem_read     (mem_read),
        .mem_write    (mem_write),
        .flag_zero    (flag_zero),
        .flag_sign    (flag_sign),
        .flag_overflow(flag_overflow),
        .alu_opcode   (alu_opcode),
        .alu_load_acc (alu_load_acc),
        .alu_operand_b(alu_operand_b),
        .alu_cin_ext  (alu_cin_ext),
        .reg_x_load   (reg_x_load),
        .reg_y_load   (reg_y_load),
        .reg_x_val    (reg_x_val),
        .reg_y_val    (reg_y_val),
        .acc_val      (acc_val)
    );

    always #5 clk = ~clk;

    // Decodificador de endereço ternário (4 trits balanceados -> índice 0..15)
    wire [3:0] linear_addr;
    trits_to_addr #(
        .NUM_TRITS(4),
        .ADDR_WIDTH(4)
    ) u_addr_dec (
        .trits_in(mem_addr[7:0]),
        .addr_out(linear_addr)
    );

    // Modelo de memória assíncrona simulada de 6 trits
    reg [11:0] rom [0:15];

    always @(*) begin
        if (mem_read)
            mem_data_in = rom[linear_addr];
        else
            mem_data_in = {6{T_ZER}};
    end

    initial begin
        clk           = 0;
        rst_n         = 0;
        flag_zero     = 0;
        flag_sign     = T_ZER;
        flag_overflow = 0;
        acc_val       = {6{T_ZER}};
        reg_x_val     = {6{T_ZER}};
        reg_y_val     = {6{T_ZER}};

        // Inicializa a memória com o programa de teste:
        // Endereço 0: Opcode LOAD Imediato (Grupo=+1+1, Modo=0+1, Op=+1+1)
        rom[0] = {T_POS, T_POS, T_ZER, T_POS, T_POS, T_POS};
        // Endereço 1: Operando imediato (+13 decimal: + + + 0 0 0)
        rom[1] = {T_ZER, T_ZER, T_ZER, T_POS, T_POS, T_POS};
        // Endereço 2: Opcode ADD Imediato (Grupo=+1+1, Modo=0+1, Op=-1-1)
        rom[2] = {T_POS, T_POS, T_ZER, T_POS, T_NEG, T_NEG};
        // Endereço 3: Operando imediato (+5 decimal)
        rom[3] = {T_ZER, T_ZER, T_ZER, T_POS, T_NEG, T_NEG};

        #15 rst_n = 1;

        $display("=== SIMULANDO CICLOS DE INSTRUÇÃO DO 6502-T ===");
        #80;

        $display("Operando carregado na ALU: %b | Opcode ALU emitido: %b | Strobe Load: %b",
                 alu_operand_b, alu_opcode, alu_load_acc);

        #80;
        $finish;
    end

endmodule
