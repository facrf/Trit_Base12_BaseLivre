`timescale 1ns / 1ps

module tb_system_dozenal_monitor;

    reg         clk;
    reg         rst_n;

    wire [19:0] addr_bus;
    reg  [11:0] mem_data_to_cpu;
    wire [11:0] cpu_data_to_mem;
    wire        read_en;
    wire        write_en;

    wire        acc_sign;
    wire [3:0]  acc_d2, acc_d1, acc_d0;
    wire        acc_is_zero;

    wire        mem_wr_sign;
    wire [3:0]  mem_wr_d2, mem_wr_d1, mem_wr_d0;
    wire        mem_wr_is_zero;
    wire        mem_wr_active;

    localparam [1:0] T_NEG = 2'b00; // -1
    localparam [1:0] T_ZER = 2'b01; //  0
    localparam [1:0] T_POS = 2'b10; // +1

    // Instanciação do Wrapper de Depuração
    cpu_6502t_debugger dut (
        .clk           (clk),
        .rst_n         (rst_n),
        .clk_en        (1'b1),
        .ext_addr_bus  (addr_bus),
        .ext_data_in   (mem_data_to_cpu),
        .ext_data_out  (cpu_data_to_mem),
        .ext_read_en   (read_en),
        .ext_write_en  (write_en),
        .acc_sign      (acc_sign),
        .acc_d2        (acc_d2),
        .acc_d1        (acc_d1),
        .acc_d0        (acc_d0),
        .acc_is_zero   (acc_is_zero),
        .mem_wr_sign   (mem_wr_sign),
        .mem_wr_d2     (mem_wr_d2),
        .mem_wr_d1     (mem_wr_d1),
        .mem_wr_d0     (mem_wr_d0),
        .mem_wr_is_zero(mem_wr_is_zero),
        .mem_wr_active (mem_wr_active)
    );

    always #5 clk = ~clk;

    // Função de formatação para exibir dígitos 10 e 11 como 'A' e 'B'
    function [7:0] fmt_dozenal_char;
        input [3:0] digit;
        begin
            case (digit)
                4'd10:   fmt_dozenal_char = "A";
                4'd11:   fmt_dozenal_char = "B";
                default: fmt_dozenal_char = 8'd48 + digit; // '0'..'9' em ASCII
            endcase
        end
    endfunction

    // Decodificador de endereço ternário (5 trits balanceados -> índice 0..63)
    wire [5:0] linear_addr;
    trits_to_addr #(
        .NUM_TRITS(5),
        .ADDR_WIDTH(6)
    ) u_addr_dec (
        .trits_in(addr_bus[9:0]),
        .addr_out(linear_addr)
    );

    // Memória Ternária Simulatória (RAM/ROM)
    reg [11:0] memory [0:63];

    always @(posedge clk) begin
        if (write_en) begin
            memory[linear_addr] <= cpu_data_to_mem;
            $display("[T=%0t ns] [MEM WRITE] Endereco: %0d | Valor Gravado: %s%c%c%c (Base 12)",
                     $time, linear_addr,
                     mem_wr_sign ? "-" : "+",
                     fmt_dozenal_char(mem_wr_d2),
                     fmt_dozenal_char(mem_wr_d1),
                     fmt_dozenal_char(mem_wr_d0));
        end
    end

    always @(*) begin
        if (read_en)
            mem_data_to_cpu = memory[linear_addr];
        else
            mem_data_to_cpu = {6{T_ZER}};
    end

    // Monitoramento do Acumulador a cada borda de subida
    always @(posedge clk) begin
        if (rst_n) begin
            $display("[T=%0t ns] [CPU MONITOR] Acumulador: %s%c%c%c | Zero: %0d",
                     $time,
                     acc_sign ? "-" : "+",
                     fmt_dozenal_char(acc_d2),
                     fmt_dozenal_char(acc_d1),
                     fmt_dozenal_char(acc_d0),
                     acc_is_zero);
        end
    end

    initial begin
        clk   = 0;
        rst_n = 0;

        for (integer i = 0; i < 64; i = i + 1)
            memory[i] = {6{T_ZER}};

        // PROGRAMA DE TESTE:
        // 1. LDA #+13 (+13 decimal = 011 em base 12)
        memory[0] = {T_POS, T_POS, T_ZER, T_POS, T_POS, T_POS};
        memory[1] = {T_ZER, T_ZER, T_ZER, T_POS, T_POS, T_POS}; // +13

        // 2. ADD #+130 (+130 decimal = 143 total -> 0BB em base 12)
        // 130 = +243 - 81 - 27 - 5 = (+ - - 0 - +)
        memory[2] = {T_POS, T_POS, T_ZER, T_POS, T_NEG, T_NEG};
        memory[3] = {T_POS, T_NEG, T_NEG, T_ZER, T_NEG, T_POS}; // +130

        // 3. STA Absoluto Endereço 12
        memory[4] = {T_ZER, T_POS, T_POS, T_POS, T_POS, T_POS};
        memory[5] = {T_ZER, T_ZER, T_ZER, T_POS, T_POS, T_ZER}; // +12 decimal (0 0 0 + + 0)

        #25 rst_n = 1;

        #200;
        $finish;
    end

endmodule
