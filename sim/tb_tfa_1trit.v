`timescale 1ns / 1ps

module tb_tfa_1trit;

    reg  [1:0] a, b, cin;
    wire [1:0] s, cout;

    // Instanciação da Unidade sob Teste (UUT)
    tfa_1trit uut (
        .a(a),
        .b(b),
        .cin(cin),
        .s(s),
        .cout(cout)
    );

    // Arrays auxiliares para iteração
    reg [1:0] trits [0:2];
    integer i, j, k;
    integer val_a, val_b, val_cin, val_s, val_cout;
    integer errors;

    function integer to_int;
        input [1:0] t;
        case (t)
            2'b00: to_int = -1;
            2'b01: to_int =  0;
            2'b10: to_int =  1;
            default: to_int = 0;
        endcase
    endfunction

    initial begin
        trits[0] = 2'b00; // -1
        trits[1] = 2'b01; //  0
        trits[2] = 2'b10; // +1
        errors = 0;

        $display("=== INICIANDO TESTE DO TERNARY FULL ADDER (27 COMBINACOES) ===");

        for (i = 0; i < 3; i = i + 1) begin
            for (j = 0; j < 3; j = j + 1) begin
                for (k = 0; k < 3; k = k + 1) begin
                    a   = trits[i];
                    b   = trits[j];
                    cin = trits[k];
                    #10;

                    val_a    = to_int(a);
                    val_b    = to_int(b);
                    val_cin  = to_int(cin);
                    val_s    = to_int(s);
                    val_cout = to_int(cout);

                    // Validação: A + B + Cin == 3*Cout + S
                    if ((val_a + val_b + val_cin) !== (3 * val_cout + val_s)) begin
                        $display("[FALHA] A=%0d B=%0d Cin=%0d | Cout=%0d S=%0d", 
                                 val_a, val_b, val_cin, val_cout, val_s);
                        errors = errors + 1;
                    end else begin
                        $display("[OK] A=%2d B=%2d Cin=%2d -> Cout=%2d S=%2d", 
                                 val_a, val_b, val_cin, val_cout, val_s);
                    end
                end
            end
        end

        if (errors == 0)
            $display("=== SUCESSO: TODAS AS 27 COMBINACOES PASSARAM ===");
        else
            $display("=== FINALIZADO COM %0d ERROS ===", errors);

        $finish;
    end

endmodule
