`timescale 1ns / 1ps

module fpga_test_top (
    input  wire        clk,        // Oscilador onboard de 100 MHz
    input  wire        btnC,       // Reset (ativo em nível alto na placa)
    input  wire        btnU,       // Passo Único (Single-Step)
    input  wire [15:0] sw,         // Chaves de controle
    output wire [15:0] led,        // LEDs de diagnóstico
    output wire [6:0]  seg,        // Segmentos {g, f, e, d, c, b, a}
    output wire [3:0]  an          // Anodos de seleção {an3, an2, an1, an0}
);

    // Inversão do botão central para reset interno ativo baixo
    wire rst_n = ~btnC;

    // 1. Debounce do botão de passo único (btnU)
    wire step_pulse;
    wire step_state;

    button_debounce #(
        .CLK_FREQ_HZ(100_000_000),
        .DEBOUNCE_MS(20)
    ) u_deb_step (
        .clk      (clk),
        .rst_n    (rst_n),
        .btn_in   (btnU),
        .btn_state(step_state),
        .btn_pulse(step_pulse)
    );

    // 2. Divisor de clock para modo automático lento (~2 Hz)
    reg [25:0] auto_clk_div;
    wire       auto_tick = (auto_clk_div == 26'd49_999_999);

    always @(posedge clk or negedge rst_n) begin
        if (!rst_n)
            auto_clk_div <= 26'd0;
        else if (auto_tick)
            auto_clk_div <= 26'd0;
        else
            auto_clk_div <= auto_clk_div + 1'b1;
    end

    // Habilitação do clock da CPU (sw[1]: 0 = Manual via btnU, 1 = Auto 2 Hz)
    wire cpu_clk_enable = sw[1] ? auto_tick : step_pulse;

    // 3. Sinais e Barramentos da CPU 6502-T
    wire [19:0] addr_bus;
    wire [11:0] data_to_cpu;
    wire [11:0] data_from_cpu;
    wire        read_en;
    wire        write_en;

    wire        acc_sign;
    wire [3:0]  acc_d2, acc_d1, acc_d0;
    wire        acc_is_zero;

    wire        mem_wr_sign;
    wire [3:0]  mem_wr_d2, mem_wr_d1, mem_wr_d0;
    wire        mem_wr_is_zero;
    wire        mem_wr_active;

    // Instanciação do Wrapper CPU + Decodificador Duodecimal
    // Note: cpu_clk_enable é sincronizado via porta de clock direto habilitada
    cpu_6502t_debugger u_cpu_debugger (
        .clk           (clk),
        .rst_n         (rst_n),
        .ext_addr_bus  (addr_bus),
        .ext_data_in   (data_to_cpu),
        .ext_data_out  (data_from_cpu),
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

    // 4. Memória RAM/ROM Interna de Bloco (SRAM Síncrona de 64 palavras de 6 trits)
    localparam [1:0] T_NEG = 2'b00;
    localparam [1:0] T_ZER = 2'b01;
    localparam [1:0] T_POS = 2'b10;

    reg [11:0] ram_block [0:63];

    initial begin
        // Programa pré-gravado para demonstração no display:
        // Endereço 0: LDA #+13 (+13 dec = 011 em base 12)
        ram_block[0] = {T_POS, T_POS, T_ZER, T_POS, T_POS, T_POS};
        ram_block[1] = {T_ZER, T_ZER, T_ZER, T_POS, T_POS, T_POS};
        // Endereço 2: TAX
        ram_block[2] = {T_NEG, T_ZER, T_NEG, T_NEG, T_NEG, T_NEG};
        // Endereço 3: ADD #+130 (+130 dec -> Total: 143 dec = 0BB em base 12)
        ram_block[3] = {T_POS, T_POS, T_ZER, T_POS, T_NEG, T_NEG};
        ram_block[4] = {T_POS, T_NEG, T_NEG, T_ZER, T_NEG, T_POS};
        // Endereço 5: STA Absoluto $00A (Grava no endereço 10)
        ram_block[5] = {T_ZER, T_POS, T_POS, T_POS, T_POS, T_POS};
        ram_block[6] = {T_ZER, T_ZER, T_ZER, T_POS, T_ZER, T_POS};
    end

    always @(posedge clk) begin
        if (write_en)
            ram_block[addr_bus[5:0]] <= data_from_cpu;
    end

    assign data_to_cpu = read_en ? ram_block[addr_bus[5:0]] : {6{T_ZER}};

    // 5. Multiplexação dos Displays (sw[0]: 0 = Acumulador, 1 = Memória Escrita)
    wire       disp_sign = sw[0] ? mem_wr_sign : acc_sign;
    wire [3:0] disp_d2   = sw[0] ? mem_wr_d2   : acc_d2;
    wire [3:0] disp_d1   = sw[0] ? mem_wr_d1   : acc_d1;
    wire [3:0] disp_d0   = sw[0] ? mem_wr_d0   : acc_d0;

    display_controller_4digit u_display (
        .clk     (clk),
        .rst_n   (rst_n),
        .sign    (disp_sign),
        .d2      (disp_d2),
        .d1      (disp_d1),
        .d0      (disp_d0),
        .segments(seg),
        .anodes  (an)
    );

    // 6. Diagnóstico nos LEDs
    assign led[0]     = acc_is_zero;
    assign led[2:1]   = u_cpu_debugger.u_cpu.dbg_flag_s; // Sinal ternário (+1, 0, -1)
    assign led[3]     = u_cpu_debugger.u_cpu.dbg_flag_o; // Overflow
    assign led[4]     = write_en;                        // Pulso de escrita ativo
    assign led[5]     = read_en;                         // Pulso de leitura ativo
    assign led[9:6]   = addr_bus[3:0];                   // 4 bits inferiores do barramento
    assign led[11:10] = u_cpu_debugger.u_cpu.dbg_acc[1:0]; // Trit 0 do Acumulador
    assign led[14]    = sw[0];                           // Modo de display
    assign led[15]    = sw[1] ? auto_clk_div[25] : step_state; // Heartbeat / Clock monitor

endmodule
