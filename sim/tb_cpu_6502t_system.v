`timescale 1ns / 1ps

module tb_cpu_6502t_system;

    reg         clk;
    reg         rst_n;

    wire [19:0] addr_bus;
    reg  [11:0] mem_data_to_cpu;
    wire [11:0] cpu_data_to_mem;
    wire        read_en;
    wire        write_en;

    wire [11:0] dbg_acc;
    wire [11:0] dbg_reg_x;
    wire [11:0] dbg_reg_y;
    wire        dbg_flag_z;
    wire [1:0]  dbg_flag_s;
    wire        dbg_flag_o;

    localparam [1:0] T_NEG = 2'b00; // -1
    localparam [1:0] T_ZER = 2'b01; //  0
    localparam [1:0] T_POS = 2'b10; // +1

    // Instanciação da CPU Top-Level
    cpu_6502t_top dut (
        .clk          (clk),
        .rst_n        (rst_n),
        .addr_bus     (addr_bus),
        .data_in      (mem_data_to_cpu),
        .data_out     (cpu_data_to_mem),
        .read_en      (read_en),
        .write_en     (write_en),
        .dbg_acc      (dbg_acc),
        .dbg_reg_x    (dbg_reg_x),
        .dbg_reg_y    (dbg_reg_y),
        .dbg_flag_z   (dbg_flag_z),
        .dbg_flag_s   (dbg_flag_s),
        .dbg_flag_o   (dbg_flag_o)
    );

    // Clock com período de 10ns (100 MHz)
    always #5 clk = ~clk;

    // Memória Ternária Simulatória (64 palavras de 6 trits)
    reg [11:0] ram [0:63];

    // Leitura e Escrita Síncrona na Memória
    always @(posedge clk) begin
        if (write_en) begin
            ram[addr_bus[5:0]] <= cpu_data_to_mem;
            $display("[MEMORIA] Escrita no endereco %0d -> Dado: %b", addr_bus[5:0], cpu_data_to_mem);
        end
    end

    always @(*) begin
        if (read_en)
            mem_data_to_cpu = ram[addr_bus[5:0]];
        else
            mem_data_to_cpu = {6{T_ZER}};
    end

    initial begin
        clk   = 0;
        rst_n = 0;

        // Limpeza da Memória
        for (integer k = 0; k < 64; k = k + 1)
            ram[k] = {6{T_ZER}};

        // -------------------------------------------------------------
        // CARGA DO PROGRAMA TERNÁRIO NA MEMÓRIA
        // -------------------------------------------------------------
        // Endereço 0: LDA #+13 (Grupo=+1+1, Modo=0+1, Op=+1+1)
        ram[0] = {T_POS, T_POS, T_ZER, T_POS, T_POS, T_POS};
        // Endereço 1: Operando Imediato (+13: 0 0 0 + + +)
        ram[1] = {T_ZER, T_ZER, T_ZER, T_POS, T_POS, T_POS};

        // Endereço 2: TAX (Grupo=-1 0, Modo=-1-1, Op=-1-1)
        ram[2] = {T_NEG, T_ZER, T_NEG, T_NEG, T_NEG, T_NEG};

        // Endereço 3: ADD #+5 (Grupo=+1+1, Modo=0+1, Op=-1-1)
        ram[3] = {T_POS, T_POS, T_ZER, T_POS, T_NEG, T_NEG};
        // Endereço 4: Operando Imediato (+5: 0 0 0 + - -)
        ram[4] = {T_ZER, T_ZER, T_ZER, T_POS, T_NEG, T_NEG};

        // Endereço 5: STA Absoluto $008 (Grupo=0+1, Modo=+1+1, Op=+1+1)
        ram[5] = {T_ZER, T_POS, T_POS, T_POS, T_POS, T_POS};
        // Endereço 6: Endereço Alvo (Absoluto: palavra 8)
        ram[6] = {T_ZER, T_ZER, T_ZER, T_POS, T_ZER, T_NEG}; // +8 decimal: 0 0 0 + 0 - (9 - 1)

        // Reset inicial
        #25 rst_n = 1;

        $display("=== INICIANDO EXECUCAO DO PROGRAMA NO 6502-T ===");

        // Observa a execução ao longo dos ciclos de clock
        #350;

        $display("=== RESULTADOS FINAIS DE EXECUÇÃO ===");
        $display("Acumulador A: %b", dbg_acc);
        $display("Registrador X: %b", dbg_reg_x);
        $display("Dado gravado na RAM no endereco 8: %b", ram[8]);
        $display("Flag Zero: %0d | Flag Sign: %b", dbg_flag_z, dbg_flag_s);

        $finish;
    end

endmodule
