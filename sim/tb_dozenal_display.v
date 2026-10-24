`timescale 1ns / 1ps

module tb_dozenal_display;

    reg        clk;
    reg        rst_n;
    reg        sign;
    reg  [3:0] d2, d1, d0;

    wire [6:0] segments;
    wire [3:0] anodes;

    display_controller_4digit uut (
        .clk     (clk),
        .rst_n   (rst_n),
        .sign    (sign),
        .d2      (d2),
        .d1      (d1),
        .d0      (d0),
        .segments(segments),
        .anodes  (anodes)
    );

    always #10 clk = ~clk; // Clock de 50 MHz (período de 20ns)

    initial begin
        clk   = 0;
        rst_n = 0;
        sign  = 1;      // Valor negativo
        d2    = 4'd0;   // 0 grosas (deve ser suprimido)
        d1    = 4'd10;  // 'A' dúzias
        d0    = 4'd11;  // 'b' unidades -> Valor: -0Ab (Base 12)

        #40 rst_n = 1;

        $display("=== MONITOR DE VARREDURA DOS 4 DISPLAYS (ANODO COMUM) ===");

        // Simula ciclos rápidos forçando o avanço do tick
        repeat (4) begin
            @(negedge clk);
            force uut.refresh_tick = 1'b1;
            @(posedge clk);
            #1;
            release uut.refresh_tick;

            case (anodes)
                4'b1110: $display("Display 0 (Unidades): seg=%b ('b')", segments);
                4'b1101: $display("Display 1 (Dúzias):   seg=%b ('A')", segments);
                4'b1011: $display("Display 2 (Grosas):   seg=%b (Blank/Apagado)", segments);
                4'b0111: $display("Display 3 (Sinal):    seg=%b ('-')", segments);
            endcase
        end

        $finish;
    end

endmodule
