`timescale 1ns / 1ps

module cpu_6502t_top (
    input  wire        clk,
    input  wire        rst_n,
    input  wire        clk_en,        // Habilitação síncrona de clock

    // Barramento Externo de Memória / Periféricos
    output wire [19:0] addr_bus,      // 10 trits de endereço (20 bits)
    input  wire [11:0] data_in,       // 6 trits de dados lidos (12 bits)
    output wire [11:0] data_out,      // 6 trits de dados gravados (12 bits)
    output wire        read_en,       // Habilitação de leitura externa
    output wire        write_en,      // Habilitação de escrita externa

    // Pinos de Depuração / Monitoramento Interno
    output wire [11:0] dbg_acc,       // Conteúdo atual do Acumulador A
    output wire [11:0] dbg_reg_x,     // Conteúdo atual do Registrador X
    output wire [11:0] dbg_reg_y,     // Conteúdo atual do Registrador Y
    output wire        dbg_flag_z,    // Flag Zero
    output wire [1:0]  dbg_flag_s,    // Flag Sign (+1, 0, -1)
    output wire        dbg_flag_o     // Flag Overflow
);

    // Codificação de constantes do Trit
    localparam [1:0] T_NEG = 2'b00; // -1
    localparam [1:0] T_ZER = 2'b01; //  0
    localparam [1:0] T_POS = 2'b10; // +1

    // Sinais de Interconexão entre o Decodificador e a ALU
    wire [2:0]  ctrl_alu_opcode;
    wire        ctrl_alu_load_acc;
    wire [11:0] ctrl_alu_operand_b;
    wire [1:0]  ctrl_alu_cin_ext;

    // Sinais de Interconexão com o Banco de Registradores
    wire        ctrl_reg_x_load;
    wire        ctrl_reg_y_load;
    reg  [11:0] reg_x;
    reg  [11:0] reg_y;

    // Sinais de Status gerados pela ALU
    wire [11:0] acc_wire;
    wire [1:0]  alu_cout;
    wire        alu_flag_zero;
    wire [1:0]  alu_flag_sign;
    wire        alu_flag_overflow;

    // 1. Instanciação da ALU de 6 Trits Balanceados
    alu_6trit u_alu (
        .clk          (clk),
        .rst_n        (rst_n),
        .load_acc     (ctrl_alu_load_acc),
        .opcode       (ctrl_alu_opcode),
        .operand_b    (ctrl_alu_operand_b),
        .cin_ext      (ctrl_alu_cin_ext),
        .acc_out      (acc_wire),
        .cout         (alu_cout),
        .flag_zero    (alu_flag_zero),
        .flag_sign    (alu_flag_sign),
        .flag_overflow(alu_flag_overflow)
    );

    // 2. Instanciação da Unidade de Busca e Decodificação de Instruções
    instruction_decoder_6502t u_decoder (
        .clk          (clk),
        .rst_n        (rst_n),
        .clk_en       (clk_en),

        // Interface do Barramento Externo
        .mem_data_in  (data_in),
        .mem_addr     (addr_bus),
        .mem_data_out (data_out),
        .mem_read     (read_en),
        .mem_write    (write_en),

        // Realimentação de Flags da ALU
        .flag_zero    (alu_flag_zero),
        .flag_sign    (alu_flag_sign),
        .flag_overflow(alu_flag_overflow),

        // Controle para a ALU
        .alu_opcode   (ctrl_alu_opcode),
        .alu_load_acc (ctrl_alu_load_acc),
        .alu_operand_b(ctrl_alu_operand_b),
        .alu_cin_ext  (ctrl_alu_cin_ext),

        // Controle e Dados dos Registradores
        .reg_x_load   (ctrl_reg_x_load),
        .reg_y_load   (ctrl_reg_y_load),
        .reg_x_val    (reg_x),
        .reg_y_val    (reg_y),
        .acc_val      (acc_wire)
    );

    // 3. Banco de Registradores de Índice (X e Y)
    always @(posedge clk or negedge rst_n) begin
        if (!rst_n) begin
            reg_x <= {6{T_ZER}}; // Reset em zero balanceado
            reg_y <= {6{T_ZER}};
        end else begin
            if (ctrl_reg_x_load) begin
                reg_x <= acc_wire; // TAX: transfere Acumulador para X
            end
            if (ctrl_reg_y_load) begin
                reg_y <= acc_wire; // TAY: transfere Acumulador para Y
            end
        end
    end

    // 4. Mapeamento das Portas de Depuração
    assign dbg_acc    = acc_wire;
    assign dbg_reg_x  = reg_x;
    assign dbg_reg_y  = reg_y;
    assign dbg_flag_z = alu_flag_zero;
    assign dbg_flag_s = alu_flag_sign;
    assign dbg_flag_o = alu_flag_overflow;

endmodule
