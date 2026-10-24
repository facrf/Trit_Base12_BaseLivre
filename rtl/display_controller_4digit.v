`timescale 1ns / 1ps

module display_controller_4digit (
    input  wire        clk,           // Clock base de hardware (ex: 50 MHz ou 100 MHz)
    input  wire        rst_n,

    // Dados vindo do módulo trits_to_dozenal
    input  wire        sign,          // 1 = Negativo (-), 0 = Positivo
    input  wire [3:0]  d2,            // Grosas (0..2)
    input  wire [3:0]  d1,            // Dúzias (0..11)
    input  wire [3:0]  d0,            // Unidades (0..11)

    // Interface física com os Displays
    output wire [6:0]  segments,      // Linhas {g, f, e, d, c, b, a}
    output reg  [3:0]  anodes         // Habilitação dos 4 anodos (ativo baixo: 0 = aceso)
);

    // Divisor de clock para multiplexação (~1 kHz de taxa de varredura global)
    // Para clock de 50 MHz: 50_000_000 / 50_000 = 1 kHz (250 Hz por dígito)
    reg [15:0] clk_div;
    wire       refresh_tick = (clk_div == 16'd49_999);

    always @(posedge clk or negedge rst_n) begin
        if (!rst_n)
            clk_div <= 16'd0;
        else if (refresh_tick)
            clk_div <= 16'd0;
        else
            clk_div <= clk_div + 1'b1;
    end

    // Contador de anodo (0 a 3)
    reg [1:0] digit_select;
    always @(posedge clk or negedge rst_n) begin
        if (!rst_n)
            digit_select <= 2'b00;
        else if (refresh_tick)
            digit_select <= digit_select + 1'b1;
    end

    // Multiplexação dos sinais de controle para o decodificador
    reg [3:0] cur_digit;
    reg       cur_is_sign;
    reg       cur_blank;

    always @(*) begin
        case (digit_select)
            2'b00: begin // Dígito 0: Unidades (D0)
                anodes      = 4'b1110;
                cur_digit   = d0;
                cur_is_sign = 1'b0;
                cur_blank   = 1'b0;
            end

            2'b01: begin // Dígito 1: Dúzias (D1)
                anodes      = 4'b1101;
                cur_digit   = d1;
                cur_is_sign = 1'b0;
                cur_blank   = 1'b0;
            end

            2'b10: begin // Dígito 2: Grosas (D2)
                anodes      = 4'b1011;
                cur_digit   = d2;
                cur_is_sign = 1'b0;
                // Supressão de zero à esquerda se D2 == 0
                cur_blank   = (d2 == 4'd0);
            end

            2'b11: begin // Dígito 3: Sinal (-)
                anodes      = 4'b0111;
                cur_digit   = 4'd0;
                cur_is_sign = sign;   // Exibe traço apenas se sign == 1
                cur_blank   = ~sign;  // Apaga display se positivo
            end
        endcase
    end

    // Instanciação da Célula Decodificadora
    dozenal_7seg_decoder u_decoder (
        .digit  (cur_digit),
        .is_sign(cur_is_sign),
        .blank  (cur_blank),
        .seg    (segments)
    );

endmodule
