`timescale 1ns / 1ps

module cpu_6502t_debugger (
    input  wire        clk,
    input  wire        rst_n,
    input  wire        clk_en,

    // Interface de Memória Externa
    output wire [19:0] ext_addr_bus,
    input  wire [11:0] ext_data_in,
    output wire [11:0] ext_data_out,
    output wire        ext_read_en,
    output wire        ext_write_en,

    // Barramento de Telemetria Duodecimal do Acumulador (A)
    output wire        acc_sign,       // 0 = +, 1 = -
    output wire [3:0]  acc_d2,         // Grosas (peso 144)
    output wire [3:0]  acc_d1,         // Dúzias (peso 12)
    output wire [3:0]  acc_d0,         // Unidades (peso 1)
    output wire        acc_is_zero,

    // Barramento de Telemetria Duodecimal da Escrita em Memória
    output wire        mem_wr_sign,
    output wire [3:0]  mem_wr_d2,
    output wire [3:0]  mem_wr_d1,
    output wire [3:0]  mem_wr_d0,
    output wire        mem_wr_is_zero,
    output wire        mem_wr_active   // Pulso de escrita ativo
);

    // Sinais de depuração do núcleo
    wire [11:0] dbg_acc_raw;
    wire [11:0] dbg_reg_x_raw;
    wire [11:0] dbg_reg_y_raw;
    wire        dbg_flag_z;
    wire [1:0]  dbg_flag_s;
    wire        dbg_flag_o;

    // 1. Instanciação do Núcleo da CPU 6502-T
    cpu_6502t_top u_cpu (
        .clk          (clk),
        .rst_n        (rst_n),
        .clk_en       (clk_en),
        .addr_bus     (ext_addr_bus),
        .data_in      (ext_data_in),
        .data_out     (ext_data_out),
        .read_en      (ext_read_en),
        .write_en     (ext_write_en),
        .dbg_acc      (dbg_acc_raw),
        .dbg_reg_x    (dbg_reg_x_raw),
        .dbg_reg_y    (dbg_reg_y_raw),
        .dbg_flag_z   (dbg_flag_z),
        .dbg_flag_s   (dbg_flag_s),
        .dbg_flag_o   (dbg_flag_o)
    );

    // 2. Decodificador Duodecimal para o Acumulador A
    trits_to_dozenal u_dec_acc (
        .trits_in (dbg_acc_raw),
        .sign     (acc_sign),
        .d2       (acc_d2),
        .d1       (acc_d1),
        .d0       (acc_d0),
        .is_zero  (acc_is_zero)
    );

    // 3. Decodificador Duodecimal para o Barramento de Escrita (data_out)
    trits_to_dozenal u_dec_mem (
        .trits_in (ext_data_out),
        .sign     (mem_wr_sign),
        .d2       (mem_wr_d2),
        .d1       (mem_wr_d1),
        .d0       (mem_wr_d0),
        .is_zero  (mem_wr_is_zero)
    );

    assign mem_wr_active = ext_write_en;

endmodule
