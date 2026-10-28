`timescale 1ns / 1ps

module instruction_decoder_6502t (
    input  wire        clk,
    input  wire        rst_n,
    input  wire        clk_en,

    // Interface com o Barramento de Memória
    input  wire [11:0] mem_data_in,    // Dado lido da memória (6 trits)
    output reg  [19:0] mem_addr,       // Barramento de endereçamento (10 trits = 20 bits)
    output reg  [11:0] mem_data_out,   // Dado para escrita (6 trits)
    output reg         mem_read,
    output reg         mem_write,

    // Status vindo da ALU
    input  wire        flag_zero,
    input  wire [1:0]  flag_sign,
    input  wire        flag_overflow,

    // Sinais de Controle para a ALU e Registradores
    output reg  [2:0]  alu_opcode,
    output reg         alu_load_acc,
    output reg  [11:0] alu_operand_b,
    output reg  [1:0]  alu_cin_ext,

    // Sinais para Registradores de Índice (X e Y)
    output reg         reg_x_load,
    output reg         reg_y_load,
    input  wire [11:0] reg_x_val,
    input  wire [11:0] reg_y_val,
    input  wire [11:0] acc_val
);

    // Codificação de constantes do Trit
    localparam [1:0] T_NEG = 2'b00; // -1
    localparam [1:0] T_ZER = 2'b01; //  0
    localparam [1:0] T_POS = 2'b10; // +1

    // Estados da Máquina FSM (Ciclos de Instrução)
    localparam [2:0] S_FETCH_OPCODE = 3'b000;
    localparam [2:0] S_DECODE       = 3'b001;
    localparam [2:0] S_FETCH_OPERAND= 3'b010;
    localparam [2:0] S_FETCH_ADDR_H = 3'b011;
    localparam [2:0] S_READ_MEM     = 3'b101;
    localparam [2:0] S_EXECUTE      = 3'b100;

    reg [2:0]  state;
    reg [19:0] pc;                     // Program Counter (10 trits balanceados)
    reg [11:0] ir;                     // Instruction Register (6 trits)
    reg [11:0] operand_latched;        // Operando intermediário
    reg [19:0] target_addr;

    // Campos do IR decodificados
    wire [3:0] field_group = ir[11:8];
    wire [3:0] field_mode  = ir[7:4];
    wire [3:0] field_op    = ir[3:0];

    // Função de incremento de PC de 10 trits (+1 balanceado)
    function [19:0] inc_pc_ternary;
        input [19:0] current_pc;
        integer i;
        reg [1:0] c;
        reg [1:0] sum_digit;
        begin
            c = T_POS; // Injeta +1 no estágio 0
            for (i = 0; i < 10; i = i + 1) begin
                case ({current_pc[i*2 +: 2], c})
                    {T_ZER, T_POS}: begin sum_digit = T_POS; c = T_ZER; end
                    {T_POS, T_POS}: begin sum_digit = T_NEG; c = T_POS; end
                    {T_NEG, T_POS}: begin sum_digit = T_ZER; c = T_ZER; end
                    default:        begin sum_digit = current_pc[i*2 +: 2]; c = T_ZER; end
                endcase
                inc_pc_ternary[i*2 +: 2] = sum_digit;
            end
        end
    endfunction

    // Ciclo de Busca, Decodificação e Execução
    always @(posedge clk or negedge rst_n) begin
        if (!rst_n) begin
            state         <= S_FETCH_OPCODE;
            pc            <= {10{T_ZER}}; // Reset vector em 0 balanceado
            ir            <= {6{T_ZER}};
            mem_read      <= 1'b0;
            mem_write     <= 1'b0;
            alu_load_acc  <= 1'b0;
            reg_x_load    <= 1'b0;
            reg_y_load    <= 1'b0;
            alu_opcode    <= 3'b000;
            alu_cin_ext   <= T_ZER;
            alu_operand_b <= {6{T_ZER}};
            mem_addr      <= {10{T_ZER}};
            mem_data_out  <= {6{T_ZER}};
        end else if (clk_en) begin
            // Pulso padrão de strobes
            alu_load_acc <= 1'b0;
            reg_x_load   <= 1'b0;
            reg_y_load   <= 1'b0;
            mem_write    <= 1'b0;

            case (state)
                // 1. Ciclo de Busca de Opcode (T0)
                S_FETCH_OPCODE: begin
                    mem_addr <= pc;
                    mem_read <= 1'b1;
                    state    <= S_DECODE;
                end

                // 2. Ciclo de Decodificação (T1)
                S_DECODE: begin
                    ir       <= mem_data_in;
                    mem_read <= 1'b0;
                    pc       <= inc_pc_ternary(pc);

                    // Analisa se a instrução requer busca de operando externo
                    if (mem_data_in[7:4] == {T_ZER, T_POS}) begin
                        // Modo Imediato: busca próximo dado
                        mem_addr <= inc_pc_ternary(pc);
                        mem_read <= 1'b1;
                        state    <= S_FETCH_OPERAND;
                    end else if (mem_data_in[7:4] == {T_POS, T_POS}) begin
                        // Modo Absoluto: busca endereço
                        mem_addr <= inc_pc_ternary(pc);
                        mem_read <= 1'b1;
                        state    <= S_FETCH_ADDR_H;
                    end else begin
                        // Modo Implícito ou Registrador: executa direto
                        state    <= S_EXECUTE;
                    end
                end

                // 3. Busca de Operando Imediato
                S_FETCH_OPERAND: begin
                    operand_latched <= mem_data_in;
                    mem_read        <= 1'b0;
                    pc              <= inc_pc_ternary(pc);
                    state           <= S_EXECUTE;
                end

                // 4. Busca de Endereço Absoluto
                S_FETCH_ADDR_H: begin
                    target_addr[11:0] <= mem_data_in;
                    target_addr[19:12]<= {4{T_ZER}};
                    pc                <= inc_pc_ternary(pc);
                    if (field_group == {T_POS, T_POS}) begin
                        // Se for operação de ALU (LDA, ADD, SUB), busca dado na memória
                        mem_addr <= { {4{T_ZER}}, mem_data_in };
                        mem_read <= 1'b1;
                        state    <= S_READ_MEM;
                    end else begin
                        mem_read <= 1'b0;
                        state    <= S_EXECUTE;
                    end
                end

                // 4.1. Leitura de Memória (Modo Absoluto da ALU)
                S_READ_MEM: begin
                    operand_latched <= mem_data_in;
                    mem_read        <= 1'b0;
                    state           <= S_EXECUTE;
                end

                // 5. Estágio de Execução e Emissão de Controle (T2)
                S_EXECUTE: begin
                    case (field_group)
                        // Aritmética e Lógica da ALU
                        {T_POS, T_POS}: begin
                            alu_operand_b <= (field_mode == {T_ZER, T_POS} || field_mode == {T_POS, T_POS}) ? operand_latched : reg_x_val;
                            alu_cin_ext   <= T_ZER;

                            case (field_op)
                                {T_NEG, T_NEG}: alu_opcode <= 3'b000; // ADD
                                {T_NEG, T_POS}: alu_opcode <= 3'b001; // SUB
                                {T_ZER, T_POS}: alu_opcode <= 3'b010; // NEG
                                {T_POS, T_POS}: alu_opcode <= 3'b011; // LOAD
                                {T_POS, T_ZER}: alu_opcode <= 3'b101; // INC
                                {T_ZER, T_NEG}: alu_opcode <= 3'b110; // DEC
                                default:        alu_opcode <= 3'b000;
                            endcase

                            alu_load_acc <= 1'b1;
                            state        <= S_FETCH_OPCODE;
                        end

                        // Transferências Internas entre Registradores
                        {T_NEG, T_ZER}: begin
                            case (field_op)
                                {T_NEG, T_NEG}: begin // TAX: A -> X
                                    reg_x_load <= 1'b1;
                                end
                                {T_NEG, T_POS}: begin // TXA: X -> A
                                    alu_operand_b <= reg_x_val;
                                    alu_opcode    <= 3'b011; // LOAD
                                    alu_load_acc  <= 1'b1;
                                end
                                {T_ZER, T_POS}: begin // TAY: A -> Y
                                    reg_y_load <= 1'b1;
                                end
                                {T_ZER, T_NEG}: begin // TYA: Y -> A
                                    alu_operand_b <= reg_y_val;
                                    alu_opcode    <= 3'b011; // LOAD
                                    alu_load_acc  <= 1'b1;
                                end
                                default: ;
                            endcase
                            state <= S_FETCH_OPCODE;
                        end

                        // Desvios e Saltos de Fluxo (Branch / Jump)
                        {T_NEG, T_NEG}: begin
                            case (field_op)
                                {T_NEG, T_NEG}: begin // JMP Absoluto
                                    pc <= target_addr;
                                end
                                {T_NEG, T_POS}: begin // BEQ: Salta se Zero
                                    if (flag_zero) pc <= target_addr;
                                end
                                {T_POS, T_NEG}: begin // BNE: Salta se Não-Zero
                                    if (!flag_zero) pc <= target_addr;
                                end
                                {T_ZER, T_NEG}: begin // BMI: Salta se Negativo (-1)
                                    if (flag_sign == T_NEG) pc <= target_addr;
                                end
                                {T_ZER, T_POS}: begin // BPL: Salta se Positivo (+1)
                                    if (flag_sign == T_POS) pc <= target_addr;
                                end
                                default: ;
                            endcase
                            state <= S_FETCH_OPCODE;
                        end

                        // Memória (Load / Store)
                        {T_ZER, T_POS}: begin
                            if (field_op == {T_POS, T_POS}) begin
                                // STA: Escreve A na memória
                                mem_addr     <= target_addr;
                                mem_data_out <= acc_val;
                                mem_write    <= 1'b1;
                            end
                            state <= S_FETCH_OPCODE;
                        end

                        default: state <= S_FETCH_OPCODE;
                    endcase
                end

                default: state <= S_FETCH_OPCODE;
            endcase
        end else begin
            // Se clk_en = 0, zera strobes de pulso único mantendo registradores
            alu_load_acc <= 1'b0;
            reg_x_load   <= 1'b0;
            reg_y_load   <= 1'b0;
            mem_write    <= 1'b0;
        end
    end

endmodule
