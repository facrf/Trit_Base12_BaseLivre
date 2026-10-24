`timescale 1ns / 1ps

module tb_alu_6trit;

    reg         clk;
    reg         rst_n;
    reg         load_acc;
    reg  [2:0]  opcode;
    reg  [11:0] operand_b;
    reg  [1:0]  cin_ext;
    wire [11:0] acc_out;
    wire [1:0]  cout;
    wire        flag_zero;
    wire [1:0]  flag_sign;
    wire        flag_overflow;

    localparam [1:0] T_NEG = 2'b00;
    localparam [1:0] T_ZER = 2'b01;
    localparam [1:0] T_POS = 2'b10;

    alu_6trit uut (
        .clk          (clk),
        .rst_n        (rst_n),
        .load_acc     (load_acc),
        .opcode       (opcode),
        .operand_b    (operand_b),
        .cin_ext      (cin_ext),
        .acc_out      (acc_out),
        .cout         (cout),
        .flag_zero    (flag_zero),
        .flag_sign    (flag_sign),
        .flag_overflow(flag_overflow)
    );

    // Geração do Clock
    always #5 clk = ~clk;

    initial begin
        clk       = 0;
        rst_n     = 0;
        load_acc  = 0;
        opcode    = 3'b000;
        operand_b = {6{T_ZER}};
        cin_ext   = T_ZER;

        #15 rst_n = 1; // Sai do Reset
        #10;

        // Teste 1: Carregar A com o valor +13 em base 3 balanceada (+ + + 0 0 0 = 9 + 3 + 1 = 13)
        // [Trit 5..0] = {T_ZER, T_ZER, T_ZER, T_POS, T_POS, T_POS}
        $display("=== TESTE 1: LOAD NO ACUMULADOR (A = +13) ===");
        opcode    = 3'b011; // ALU_LOAD
        operand_b = {T_ZER, T_ZER, T_ZER, T_POS, T_POS, T_POS};
        load_acc  = 1;
        #10 load_acc = 0;
        #5;
        $display("A = %b (Flag Zero=%0d, Flag Sign=%b)", acc_out, flag_zero, flag_sign);

        // Teste 2: Somar B = +5 (+ - - = 9 - 3 - 1 = 5) -> A esperado = +18
        // 18 em base 3 balanceada: (+ - 0 0 = 27 - 9 = 18)
        $display("=== TESTE 2: SOMA (A = 13 + 5 = 18) ===");
        opcode    = 3'b000; // ALU_ADD
        operand_b = {T_ZER, T_ZER, T_ZER, T_POS, T_NEG, T_NEG};
        load_acc  = 1;
        #10 load_acc = 0;
        #5;
        $display("A = %b (Cout=%b, Sign=%b)", acc_out, cout, flag_sign);

        // Teste 3: Subtrair B = +20 (A = 18 - 20 = -2)
        // 20 balanceado: (+ - - - = 27 - 9 - 3 + 5... ou + - 0 - = 27 - 9 - 1 = 17) -> Usando valor direto
        $display("=== TESTE 3: SUBTRACAO (A = 18 - 18 = 0) ===");
        opcode    = 3'b001; // ALU_SUB
        operand_b = acc_out; // Subtrai A de si mesmo
        load_acc  = 1;
        #10 load_acc = 0;
        #5;
        $display("A = %b (Flag Zero=%0d)", acc_out, flag_zero);

        // Teste 4: Inversão unária (-A)
        $display("=== TESTE 4: INVERSAO (-A) ===");
        // Carrega A com -1
        opcode    = 3'b011;
        operand_b = {T_ZER, T_ZER, T_ZER, T_ZER, T_ZER, T_NEG};
        load_acc  = 1;
        #10;
        // Inverte A -> Deve virar +1
        opcode    = 3'b010; // ALU_NEG_A
        #10 load_acc = 0;
        #5;
        $display("A invertido = %b (Sign=%b)", acc_out, flag_sign);

        $finish;
    end

endmodule
