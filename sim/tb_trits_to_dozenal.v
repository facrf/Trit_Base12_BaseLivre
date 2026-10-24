`timescale 1ns / 1ps

module tb_trits_to_dozenal;

    reg  [11:0] trits_in;
    wire        sign;
    wire [3:0]  d2, d1, d0;
    wire        is_zero;

    localparam [1:0] T_NEG = 2'b00;
    localparam [1:0] T_ZER = 2'b01;
    localparam [1:0] T_POS = 2'b10;

    trits_to_dozenal uut (
        .trits_in(trits_in),
        .sign    (sign),
        .d2      (d2),
        .d1      (d1),
        .d0      (d0),
        .is_zero (is_zero)
    );

    // Função para imprimir dígitos 10 e 11 como A e B (notação duodecimal padrão)
    task print_dozenal;
        input [80:0] desc;
        begin
            $write("%s -> Sinal: %s | Base 12: %0d", desc, sign ? "-" : "+", d2);
            if (d1 == 10) $write("A");
            else if (d1 == 11) $write("B");
            else $write("%0d", d1);

            if (d0 == 10) $write("A");
            else if (d0 == 11) $write("B");
            else $write("%0d", d0);

            $display(" (is_zero=%0d)", is_zero);
        end
    endtask

    initial begin
        $display("=== INICIANDO VALIDACAO DO DECODIFICADOR DUODECIMAL ===");

        // Teste 1: Valor 0 (todos os trits em zero) -> Esperado: +000
        trits_in = {6{T_ZER}};
        #10 print_dozenal("Valor 0");

        // Teste 2: Valor +13 (9 + 3 + 1 = 13 decimal) -> Esperado: +011 (1*12 + 1)
        trits_in = {T_ZER, T_ZER, T_ZER, T_POS, T_POS, T_POS};
        #10 print_dozenal("Valor +13");

        // Teste 3: Valor +18 (27 - 9 = 18 decimal) -> Esperado: +016 (1*12 + 6)
        trits_in = {T_ZER, T_ZER, T_POS, T_NEG, T_ZER, T_ZER};
        #10 print_dozenal("Valor +18");

        // Teste 4: Valor +143 decimal -> Esperado: +0BB (11*12 + 11)
        // 143 = 243 - 81 - 27 + 9 - 1 (+ - - + 0 -)
        trits_in = {T_POS, T_NEG, T_NEG, T_POS, T_ZER, T_NEG};
        #10 print_dozenal("Valor +143");

        // Teste 5: Fundo de escala positivo +364 -> Esperado: +264 (2*144 + 6*12 + 4)
        trits_in = {6{T_POS}};
        #10 print_dozenal("Maximo Positivo (+364)");

        // Teste 6: Fundo de escala negativo -364 -> Esperado: -264
        trits_in = {6{T_NEG}};
        #10 print_dozenal("Minimo Negativo (-364)");

        $finish;
    end

endmodule
