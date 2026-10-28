`timescale 1ns / 1ps

module alu_6trit (
    input  wire        clk,
    input  wire        rst_n,          // Reset assíncrono ativo em nível baixo
    input  wire        load_acc,       // Habilita escrita no Acumulador
    input  wire [2:0]  opcode,         // Código da operação
    input  wire [11:0] operand_b,      // Entrada externa B (6 trits = 12 bits)
    input  wire [1:0]  cin_ext,        // Carry-in externo
    output reg  [11:0] acc_out,        // Saída direta do Acumulador A
    output wire [1:0]  cout,           // Carry-out final da cadeia de soma
    output wire        flag_zero,      // Flag Z: 1 se todos os trits forem 0
    output wire [1:0]  flag_sign,      // Flag S: sinal do trit não-nulo mais significativo
    output wire        flag_overflow   // Flag O: 1 se houver transporte para fora
);

    // Codificação de constantes de estado do Trit
    localparam [1:0] TRIT_NEG  = 2'b00; // -1
    localparam [1:0] TRIT_ZERO = 2'b01; //  0
    localparam [1:0] TRIT_POS  = 2'b10; // +1

    // Definição dos Opcodes
    localparam [2:0] ALU_ADD   = 3'b000; // A + B
    localparam [2:0] ALU_SUB   = 3'b001; // A - B = A + (-B)
    localparam [2:0] ALU_NEG_A = 3'b010; // -A
    localparam [2:0] ALU_LOAD  = 3'b011; // A = B (Pass-through de B)
    localparam [2:0] ALU_CLR   = 3'b100; // A = 0
    localparam [2:0] ALU_INC   = 3'b101; // A + 1
    localparam [2:0] ALU_DEC   = 3'b110; // A - 1

    // Sinais internos dos operandos efetivos para o somador
    reg  [11:0] op_a_eff;
    reg  [11:0] op_b_eff;
    reg  [1:0]  cin_eff;
    wire [11:0] alu_result;

    // Função para inversão simétrica de 1 trit (-1 <-> +1, 0 <-> 0)
    function [1:0] invert_trit;
        input [1:0] t;
        begin
            case (t)
                TRIT_NEG:  invert_trit = TRIT_POS;
                TRIT_POS:  invert_trit = TRIT_NEG;
                default:   invert_trit = TRIT_ZERO;
            endcase
        end
    endfunction

    // 1. Condicionamento dos Operandos Efetivos conforme o Opcode
    integer i;
    always @(*) begin
        case (opcode)
            ALU_ADD: begin
                op_a_eff = acc_out;
                op_b_eff = operand_b;
                cin_eff  = cin_ext;
            end

            ALU_SUB: begin
                op_a_eff = acc_out;
                cin_eff  = cin_ext;
                // Negação trit a trit de B a custo de inversão de estado
                for (i = 0; i < 6; i = i + 1)
                    op_b_eff[i*2 +: 2] = invert_trit(operand_b[i*2 +: 2]);
            end

            ALU_NEG_A: begin
                // Soma (-A) + 0
                for (i = 0; i < 6; i = i + 1)
                    op_a_eff[i*2 +: 2] = invert_trit(acc_out[i*2 +: 2]);
                op_b_eff = {6{TRIT_ZERO}};
                cin_eff  = TRIT_ZERO;
            end

            ALU_LOAD: begin
                // Soma 0 + B
                op_a_eff = {6{TRIT_ZERO}};
                op_b_eff = operand_b;
                cin_eff  = TRIT_ZERO;
            end

            ALU_CLR: begin
                // Soma 0 + 0
                op_a_eff = {6{TRIT_ZERO}};
                op_b_eff = {6{TRIT_ZERO}};
                cin_eff  = TRIT_ZERO;
            end

            ALU_INC: begin
                // A + 1
                op_a_eff = acc_out;
                op_b_eff = {5{TRIT_ZERO}, TRIT_POS};
                cin_eff  = TRIT_ZERO;
            end

            ALU_DEC: begin
                // A - 1
                op_a_eff = acc_out;
                op_b_eff = {5{TRIT_ZERO}, TRIT_NEG};
                cin_eff  = TRIT_ZERO;
            end

            default: begin
                op_a_eff = acc_out;
                op_b_eff = {6{TRIT_ZERO}};
                cin_eff  = TRIT_ZERO;
            end
        endcase
    end

    // 2. Encadeamento dos 6 Somadores Completos Ternários (Ripple-Carry)
    wire [1:0] carry_wire [0:6];
    assign carry_wire[0] = cin_eff;

    genvar g;
    generate
        for (g = 0; g < 6; g = g + 1) begin : gen_tfa_chain
            tfa_1trit stage (
                .a   (op_a_eff[g*2 +: 2]),
                .b   (op_b_eff[g*2 +: 2]),
                .cin (carry_wire[g]),
                .s   (alu_result[g*2 +: 2]),
                .cout(carry_wire[g+1])
            );
        end
    endgenerate

    assign cout = carry_wire[6];

    // 3. Registrador Acumulador (A)
    always @(posedge clk or negedge rst_n) begin
        if (!rst_n) begin
            acc_out <= {6{TRIT_ZERO}}; // Inicializa com 0 ternário
        end else if (load_acc) begin
            acc_out <= alu_result;
        end
    end

    // 4. Lógica de Flags de Status
    // Flag Zero: ativo se todos os 6 trits do resultado forem TRIT_ZERO (0)
    assign flag_zero = (alu_result == {6{TRIT_ZERO}});

    // Flag Overflow: ativo se houver carry de saída diferente de zero (+1 ou -1)
    assign flag_overflow = (cout != TRIT_ZERO);

    // Flag Sign: determinada pelo trit não-nulo mais significativo
    function [1:0] detect_sign;
        input [11:0] val;
        begin
            if (val[11:10] != TRIT_ZERO)
                detect_sign = val[11:10];
            else if (val[9:8] != TRIT_ZERO)
                detect_sign = val[9:8];
            else if (val[7:6] != TRIT_ZERO)
                detect_sign = val[7:6];
            else if (val[5:4] != TRIT_ZERO)
                detect_sign = val[5:4];
            else if (val[3:2] != TRIT_ZERO)
                detect_sign = val[3:2];
            else
                detect_sign = val[1:0]; // Se todos forem zero, retorna o último
        end
    endfunction

    assign flag_sign = detect_sign(alu_result);

endmodule
