`timescale 1ns / 1ps

module button_debounce #(
    parameter CLK_FREQ_HZ = 100_000_000, // 100 MHz
    parameter DEBOUNCE_MS = 20           // Janela de estabilização de 20ms
)(
    input  wire clk,
    input  wire rst_n,
    input  wire btn_in,
    output reg  btn_state, // Nível lógico estável filtrado
    output reg  btn_pulse  // Pulso de ciclo único na ativação
);

    localparam integer THRESHOLD = (CLK_FREQ_HZ / 1000) * DEBOUNCE_MS;

    // Sincronizador de 2 estágios para eliminar metaestabilidade
    reg sync_0, sync_1;
    always @(posedge clk or negedge rst_n) begin
        if (!rst_n) begin
            sync_0 <= 1'b0;
            sync_1 <= 1'b0;
        end else begin
            sync_0 <= btn_in;
            sync_1 <= sync_0;
        end
    end

    reg [31:0] timer;
    reg        state_reg;

    always @(posedge clk or negedge rst_n) begin
        if (!rst_n) begin
            timer     <= 32'd0;
            state_reg <= 1'b0;
            btn_state <= 1'b0;
            btn_pulse <= 1'b0;
        end else begin
            btn_pulse <= 1'b0; // Padrão

            // Se o sinal mudou em relação ao estado consolidado, incrementa o filtro
            if (sync_1 != state_reg) begin
                timer <= timer + 1'b1;
                if (timer >= THRESHOLD) begin
                    state_reg <= sync_1;
                    btn_state <= sync_1;
                    timer     <= 32'd0;
                    if (sync_1 == 1'b1) begin
                        btn_pulse <= 1'b1; // Dispara pulso de borda de subida
                    end
                end
            end else begin
                timer <= 32'd0;
            end
        end
    end

endmodule
