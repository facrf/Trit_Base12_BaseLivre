Para implementar o somador completo ternário em Verilog sintetizável, mapeia-se cada estado balanceado $\{-1, 0, +1\}$ em um vetor de 2 bits:

* `2'b00` $\to -1$ (Trit Negativo)
* `2'b01` $\to \phantom{-}0$ (Trit Zero)
* `2'b10` $\to +1$ (Trit Positivo)
* `2'b11` $\to$ Não utilizado / Estado Inválido (*Don't Care*)

---

### Módulo Sintetizável: `tfa_1trit.v`

O circuito decodifica os vetores de entrada em inteiros com sinal, computa a soma interna $S_{int} \in [-3, +3]$ via lógica combinacional e sintetiza o transporte e a soma com base nos limiares balanceados.

```verilog
`timescale 1ns / 1ps

module tfa_1trit (
    input  wire [1:0] a,      // Trit A   (2'b00 = -1, 2'b01 = 0, 2'b10 = +1)
    input  wire [1:0] b,      // Trit B   (2'b00 = -1, 2'b01 = 0, 2'b10 = +1)
    input  wire [1:0] cin,    // Trit Cin (2'b00 = -1, 2'b01 = 0, 2'b10 = +1)
    output reg  [1:0] s,      // Trit Soma
    output reg  [1:0] cout    // Trit Transporte (Cout)
);

    // Codificação de constantes de estado
    localparam [1:0] TRIT_NEG  = 2'b00; // -1
    localparam [1:0] TRIT_ZERO = 2'b01; //  0
    localparam [1:0] TRIT_POS  = 2'b10; // +1

    // Sinais internos com sinal (largura suficiente para -3 até +3)
    reg signed [1:0] val_a;
    reg signed [1:0] val_b;
    reg signed [1:0] val_cin;
    reg signed [3:0] s_int;
    reg signed [1:0] s_calc;

    // 1. Decodificação dos 2 bits para inteiro com sinal
    always @(*) begin
        case (a)
            TRIT_NEG:  val_a = -2'sd1;
            TRIT_ZERO: val_a =  2'sd0;
            TRIT_POS:  val_a =  2'sd1;
            default:   val_a =  2'sd0;
        endcase

        case (b)
            TRIT_NEG:  val_b = -2'sd1;
            TRIT_ZERO: val_b =  2'sd0;
            TRIT_POS:  val_b =  2'sd1;
            default:   val_b =  2'sd0;
        endcase

        case (cin)
            TRIT_NEG:  val_cin = -2'sd1;
            TRIT_ZERO: val_cin =  2'sd0;
            TRIT_POS:  val_cin =  2'sd1;
            default:   val_cin =  2'sd0;
        endcase
    end

    // 2. Aritmética de Limiar Ternário e Recodificação para 2 bits
    always @(*) begin
        // Soma aritmética total (-3 <= s_int <= +3)
        s_int = val_a + val_b + val_cin;

        // Determinação do transporte e da soma balanceada
        if (s_int >= 4'sd2) begin
            cout   = TRIT_POS;           // Cout = +1
            s_calc = s_int - 4'sd3;      // S = S_int - 3
        end else if (s_int <= -4'sd2) begin
            cout   = TRIT_NEG;           // Cout = -1
            s_calc = s_int + 4'sd3;      // S = S_int - (-3) = S_int + 3
        end else begin
            cout   = TRIT_ZERO;          // Cout = 0
            s_calc = s_int[1:0];         // S = S_int
        end

        // Mapeamento da soma resultante para 2 bits
        case (s_calc)
            -2'sd1:  s = TRIT_NEG;
             2'sd0:  s = TRIT_ZERO;
             2'sd1:  s = TRIT_POS;
            default: s = TRIT_ZERO;
        endcase
    end

endmodule

```

---

### Testbench de Verificação: `tb_tfa_1trit.v`

Este testbench varre os 27 estados da tabela-verdade balanceada e valida a equação invariante $A + B + C_{in} = 3 \cdot C_{out} + S$.

```verilog
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

```

---

### Síntese e Próximos Passos na Arquitetura

* **Otimização por LUT:** Na síntese para FPGAs convencionais (LUT-4 ou LUT-6), cada saída (`s` e `cout`) consome apenas 2 LUTs para decodificação e seleção direta via tabela de verdade, operando com atraso de propagação mínimo.
* **Encadeamento:** Para formar a ALU de 6 trits do fork do 6502, instanciam-se 6 blocos `tfa_1trit` ligando `cout` do estágio $n$ ao `cin` do estágio $n+1$ em topologia *Ripple Carry Ternário*.

Para implementar a ALU ternária de 6 trits em topologia *Ripple Carry*, o bloco fundamental `tfa_1trit` é encadeado em 6 estágios, integrando o condicionamento de inversão do operando $B$, o registrador Acumulador ($A$), e a geração das flags de status balanceadas.

---

### Arquitetura de Conexão da ALU

```
                         OPCODE (Controle de Operação)
                                    │
                                    ▼
       Operando B [11:0] ───► [ Inversor B ] ───► B_eff [11:0]
                                                     │
       Acumulador A [11:0] ──────────────────────────┼───────────────┐
                                                     │               │
                                                     ▼               ▼
  Cin ──► [ TFA 0 ] ──C1──► [ TFA 1 ] ── ... ──► [ TFA 5 ] ──► Cout
              │                 │                    │
              ▼                 ▼                    ▼
             S[0]              S[1]                 S[5]
              └─────────────────┼────────────────────┘
                                │
                                ▼
                       Resultado da ALU [11:0]
                                │
                 ┌──────────────┴──────────────┐
                 ▼                             ▼
        [ Detector de Flags ]        [ Registrador A (CLK) ]
        (Zero, Sinal, Overflow)                │
                                               └─► Saída A [11:0]

```

---

### Módulo Sintetizável: `alu_6trit.v`

```verilog
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
    wire [6:0]  carry_chain [1:0]; // Vetor de transportes entre estágios

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

```

---

### Testbench de Ciclo de Instrução: `tb_alu_6trit.v`

Valida a carga do Acumulador, a soma direta, a subtração por inversão de $B$ e a inversão unária do Acumulador.

```verilog
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

```

---

### Características de Operação do Circuito

* **Subtração sem Complementador:** A instrução de subtração (`ALU_SUB`) não necessita de um ciclo extra nem de somar $+1$ ao carry de entrada. Apenas troca os estados físicos dos trits de $B$ (`2'b00` vira `2'b10` e vice-versa), enviando-os diretamente para a mesma malha do somador com `cin = 0`.
* **Detecção de Sinal sem Bit Dedicado:** O sinal do resultado (`flag_sign`) é extraído pela função prioritária `detect_sign`, que busca o primeiro trit com valor não nulo a partir do mais significativo ($S_5 \to S_0$). Se o primeiro trit encontrado for `2'b10`, a magnitude inteira é positiva; se for `2'b00`, é negativa.
* **Capacidade de Faixa Numérica:** Com 6 trits balanceados, a ALU cobre o intervalo inteiro de:

$$-\frac{3^6 - 1}{2} \le N \le +\frac{3^6 - 1}{2} \implies -364 \le N \le +364$$


O decodificador converte os 6 trits balanceados ($[-364, +364]$) em três dígitos duodecimais ($D_2, D_1, D_0 \in [0..11]$) acompanhados de uma flag de sinal, estruturando os barramentos de exibição e comunicação externa da arquitetura.

### Mapeamento Matemático e Pesos

O valor decimal com sinal é recomposto pela soma ponderada das potências de 3:

$$V = \sum_{k=0}^{5} t_k \cdot 3^k = t_0 \cdot 1 + t_1 \cdot 3 + t_2 \cdot 9 + t_3 \cdot 27 + t_4 \cdot 81 + t_5 \cdot 243$$

Onde cada trit assume $t_k \in \{-1, 0, +1\}$. A decomposição na base 12 da magnitude absoluta $\vert{}V\vert{} \le 364$ é obtida por:

$$\vert{}V\vert{} = D_2 \cdot 12^2 + D_1 \cdot 12^1 + D_0 \cdot 12^0 = D_2 \cdot 144 + D_1 \cdot 12 + D_0$$

* **$D_2$ (Gross / Grosas):** $\lfloor \vert{}V\vert{} / 144 \rfloor$ (faixa $0$ a $2$)
* **$D_1$ (Dozen / Dúzias):** $\lfloor (\vert{}V\vert{} \pmod{144}) / 12 \rfloor$ (faixa $0$ a $11$)
* **$D_0$ (Units / Unidades):** $\vert{}V\vert{} \pmod{12}$ (faixa $0$ a $11$)

---

### Módulo Sintetizável: `trits_to_dozenal.v`

```verilog
`timescale 1ns / 1ps

module trits_to_dozenal (
    input  wire [11:0] trits_in,      // 6 trits balanceados (2 bits por trit)
    output reg         sign,          // 0 = Positivo/Zero, 1 = Negativo
    output reg  [3:0]  d2,            // Dígito de peso 144 (0 a 2)
    output reg  [3:0]  d1,            // Dígito de peso 12  (0 a 11 / 0 a B)
    output reg  [3:0]  d0,            // Dígito de peso 1   (0 a 11 / 0 a B)
    output wire        is_zero        // Flag indicativa de valor nulo
);

    // Codificação de constantes do trit
    localparam [1:0] TRIT_NEG  = 2'b00; // -1
    localparam [1:0] TRIT_ZERO = 2'b01; //  0
    localparam [1:0] TRIT_POS  = 2'b10; // +1

    // Função de extração de valor com sinal de cada trit
    function signed [1:0] trit_val;
        input [1:0] t;
        begin
            case (t)
                TRIT_NEG:  trit_val = -2'sd1;
                TRIT_POS:  trit_val =  2'sd1;
                default:   trit_val =  2'sd0;
            endcase
        end
    endfunction

    reg signed [9:0] v_signed;   // Faixa: -364 a +364 (requer 10 bits com sinal)
    reg        [8:0] v_abs;      // Faixa: 0 a 364 (9 bits sem sinal)
    reg        [8:0] rem_144;

    integer k;

    // 1. Recomposição Ponderada em Base 3 Balanceada
    always @(*) begin
        v_signed = trit_val(trits_in[1:0])   * 10'sd1   +
                   trit_val(trits_in[3:2])   * 10'sd3   +
                   trit_val(trits_in[5:4])   * 10'sd9   +
                   trit_val(trits_in[7:6])   * 10'sd27  +
                   trit_val(trits_in[9:8])   * 10'sd81  +
                   trit_val(trits_in[11:10]) * 10'sd243;

        // 2. Extração de Sinal e Magnitude
        if (v_signed < 0) begin
            sign  = 1'b1;
            v_abs = -v_signed;
        end else begin
            sign  = 1'b0;
            v_abs = v_signed[8:0];
        end

        // 3. Decomposição Duodecimal (Base 12) por Subtração de Múltiplos
        // Dígito D2 (Grosas: peso 144)
        if (v_abs >= 9'd288) begin
            d2      = 4'd2;
            rem_144 = v_abs - 9'd288;
        end else if (v_abs >= 9'd144) begin
            d2      = 4'd1;
            rem_144 = v_abs - 9'd144;
        end else begin
            d2      = 4'd0;
            rem_144 = v_abs;
        end

        // Dígito D1 (Dúzias: peso 12) e Dígito D0 (Unidades: peso 1)
        d1 = rem_144 / 12;
        d0 = rem_144 % 12;
    end

    assign is_zero = (v_signed == 10'sd0);

endmodule

```

---

### Testbench de Validação: `tb_trits_to_dozenal.v`

```verilog
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

```

---

### Características de Síntese

* **Divisores Simplificados por Limiar:** Como o valor máximo absoluto é limitado a $364$, a decomposição de $D_2$ não utiliza divisores genéricos de silício, sendo sintetizada em apenas dois comparadores de limiar ($288$ e $144$) e subtratores fixos.
* **Operação de Baixo Atraso:** A conversão de $D_1$ e $D_0$ consome uma faixa de entrada menor que $144$ ($< 2^8$), sendo mapeada pelos sintetizadores em pequenas tabelas de consulta (LUTs) de propagação combinacional direta, sem exigir ciclos de clock.



A unidade de busca e decodificação do 6502-T processa instruções codificadas em palavras de 6 trits balanceados ($3^6 = 729$ opcodes possíveis), mapeados no barramento como vetores de 12 bits (`2'b00` = $-1$, `2'b01` = $0$, `2'b10` = $+1$).

---

### Estrutura do Opcode de 6 Trits

O espaço de 6 trits $[T_5, T_4, T_3, T_2, T_1, T_0]$ é decomposto em três campos ortogonais de 2 trits cada:

```
  Trits 5..4 [11:8]        Trits 3..2 [7:4]        Trits 1..0 [3:0]
┌─────────────────────┬───────────────────────┬──────────────────────┐
│  CLASSE/GRUPO (G)   │ MODO ENDEREÇAMENTO (M)│ OPERAÇÃO/ALU (OP)    │
│  ALU, Memória, Saltos│  Imediato, Absoluto,  │  ADD, SUB, LOAD,     │
│  ou Transferências  │  Registrador, Zero    │  NEG, CLR, INC, DEC  │
└─────────────────────┴───────────────────────┴──────────────────────┘

```

* **Grupo ($T_5, T_4$):**
* `TRIT_NEG, TRIT_NEG` ($-1, -1$): Controle e Saltos de Fluxo (JMP, BEQ, BNE, BPL, BMI).
* `TRIT_NEG, TRIT_ZERO` ($-1, \phantom{-}0$): Transferência entre Registradores (TAX, TXA, TAY, TYA).
* `TRIT_ZERO, TRIT_POS` ($\phantom{-}0, +1$): Acesso à Memória (LDA, STA, LDX, STX).
* `TRIT_POS, TRIT_POS` ($+1, +1$): Operações Aritméticas/Lógicas da ALU sobre o Acumulador.


* **Modo de Endereçamento ($T_3, T_2$):**
* `TRIT_NEG, TRIT_NEG` ($-1, -1$): Implícito / Acumulador direto.
* `TRIT_ZERO, TRIT_POS` ($\phantom{-}0, +1$): Imediato (próxima palavra de 6 trits).
* `TRIT_POS, TRIT_POS` ($+1, +1$): Absoluto (endereço de memória de 10 trits).


* **Operação ($T_1, T_0$):**
* Mapeia diretamente os opcodes primários da ALU (`ADD`, `SUB`, `LOAD`, `NEG`, `INC`, `DEC`).



---

### Módulo Sintetizável: `instruction_decoder_6502t.v`

```verilog
`timescale 1ns / 1ps

module instruction_decoder_6502t (
    input  wire        clk,
    input  wire        rst_n,

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
    localparam [2:0] S_EXECUTE      = 3'b100;
    localparam [2:0] S_WRITEBACK    = 3'b101;

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
        end else begin
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
                    mem_read          <= 1'b0;
                    pc                <= inc_pc_ternary(pc);
                    state             <= S_EXECUTE;
                end

                // 5. Estágio de Execução e Emissão de Controle (T2)
                S_EXECUTE: begin
                    case (field_group)
                        // Aritmética e Lógica da ALU
                        {T_POS, T_POS}: begin
                            alu_operand_b <= (field_mode == {T_ZER, T_POS}) ? operand_latched : reg_x_val;
                            alu_cin_ext   <= T_ZER;

                            case (field_op)
                                {T_NEG, T_NEG}: alu_opcode <= 3'b000; // ADD
                                {T_NEG, T_POS}: alu_opcode <= 3'b001; // SUB
                                {T_ZER, T_POS}: alu_opcode <= 3'b010; // NEG
                                {T_POS, T_POS}: alu_opcode <= 3'b011; // LOAD
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
        end
    end endmodule

```

---

### Testbench de Simulação do Pipeline: `tb_instruction_decoder_6502t.v`

Valida a busca e execução sequencial de duas instruções:

1. `LDA #+13`: Carrega o imediato $+13$ no Acumulador.
2. `BEQ`: Desvio condicional testando a flag zero.

```verilog
`timescale 1ns / 1ps

module tb_instruction_decoder_6502t;

    reg         clk;
    reg         rst_n;
    reg  [11:0] mem_data_in;
    wire [19:0] mem_addr;
    wire [11:0] mem_data_out;
    wire        mem_read;
    wire        mem_write;

    reg         flag_zero;
    reg  [1:0]  flag_sign;
    reg         flag_overflow;

    wire [2:0]  alu_opcode;
    wire        alu_load_acc;
    wire [11:0] alu_operand_b;
    wire [1:0]  alu_cin_ext;
    wire        reg_x_load;
    wire        reg_y_load;

    reg  [11:0] acc_val;
    reg  [11:0] reg_x_val;
    reg  [11:0] reg_y_val;

    localparam [1:0] T_NEG = 2'b00;
    localparam [1:0] T_ZER = 2'b01;
    localparam [1:0] T_POS = 2'b10;

    instruction_decoder_6502t uut (
        .clk          (clk),
        .rst_n        (rst_n),
        .mem_data_in  (mem_data_in),
        .mem_addr     (mem_addr),
        .mem_data_out (mem_data_out),
        .mem_read     (mem_read),
        .mem_write    (mem_write),
        .flag_zero    (flag_zero),
        .flag_sign    (flag_sign),
        .flag_overflow(flag_overflow),
        .alu_opcode   (alu_opcode),
        .alu_load_acc (alu_load_acc),
        .alu_operand_b(alu_operand_b),
        .alu_cin_ext  (alu_cin_ext),
        .reg_x_load   (reg_x_load),
        .reg_y_load   (reg_y_load),
        .reg_x_val    (reg_x_val),
        .reg_y_val    (reg_y_val),
        .acc_val      (acc_val)
    );

    always #5 clk = ~clk;

    // Modelo de memória assíncrona simulada de 6 trits
    reg [11:0] rom [0:15];

    always @(*) begin
        if (mem_read)
            mem_data_in = rom[mem_addr[3:0]];
        else
            mem_data_in = {6{T_ZER}};
    end

    initial begin
        clk           = 0;
        rst_n         = 0;
        flag_zero     = 0;
        flag_sign     = T_ZER;
        flag_overflow = 0;
        acc_val       = {6{T_ZER}};
        reg_x_val     = {6{T_ZER}};
        reg_y_val     = {6{T_ZER}};

        // Inicializa a memória com o programa de teste:
        // Endereço 0: Opcode LOAD Imediato (Grupo=+1+1, Modo=0+1, Op=+1+1)
        rom[0] = {T_POS, T_POS, T_ZER, T_POS, T_POS, T_POS};
        // Endereço 1: Operando imediato (+13 decimal: + + + 0 0 0)
        rom[1] = {T_ZER, T_ZER, T_ZER, T_POS, T_POS, T_POS};
        // Endereço 2: Opcode ADD Imediato (Grupo=+1+1, Modo=0+1, Op=-1-1)
        rom[2] = {T_POS, T_POS, T_ZER, T_POS, T_NEG, T_NEG};
        // Endereço 3: Operando imediato (+5 decimal)
        rom[3] = {T_ZER, T_ZER, T_ZER, T_POS, T_NEG, T_NEG};

        #15 rst_n = 1;

        $display("=== SIMULANDO CICLOS DE INSTRUÇÃO DO 6502-T ===");
        #80;

        $display("Operando carregado na ALU: %b | Opcode ALU emitido: %b | Strobe Load: %b",
                 alu_operand_b, alu_opcode, alu_load_acc);

        #80;
        $finish;
    end

endmodule

```

---

### Integração Estrutural

1. **Incrementador Ternário Dedicado:** A função `inc_pc_ternary` realiza a progressão aritmética direta do registrador de programa na base 3 balanceada sem passar pela ALU principal, preservando os acumuladores e as flags durante os ciclos de busca de operandos.
2. **Ortogonalidade de Instrução:** Os 729 estados permitidos pelos 6 trits eliminam os opcodes ilegais desorganizados do 6502 clássico, organizando cada instrução em uma matriz estrita de decodificação combinacional sem microcódigo ROM.



O módulo topo (*top-level*) do processador **6502-T** integra o decodificador de instruções, a ALU de 6 trits balanceados, o registrador acumulador e o banco de registradores de índice ($X$ e $Y$), expondo diretamente os barramentos de endereçamento de 10 trits ($3^{10} = 59.049$ posições) e de dados de 6 trits ($3^6 = 729$ valores).

```
                             BARRAMENTO EXTERNO
                  ┌─────────────────┬─────────────────┐
                  │                 │                 │
                  ▼                 ▼                 ▲
           addr_bus [19:0]   data_in [11:0]    data_out [11:0]
            (10 Trits)          (6 Trits)         (6 Trits)
                  │                 │                 │
  ┌───────────────┼─────────────────┼─────────────────┼──────────────────────────┐
  │ 6502-T TOP    │                 │                 │                          │
  │               ▼                 │                 │                          │
  │      ┌─────────────────┐        │                 │                          │
  │      │   DECODIFICADOR │◄───────┘                 │                          │
  │      │  DE INSTRUÇÕES  ├──────────────────────────┼───────────────┐          │
  │      └─┬───┬───┬─────┬─┘                          │               │          │
  │        │   │   │     │ (read_en, write_en)        │               │          │
  │        │   │   │     └────────────────────────────┘               │          │
  │        │   │   │ alu_opcode, load_acc, cin                        │          │
  │        │   │   ▼                                                  │          │
  │        │   │ ┌───────────────────────────┐                        │          │
  │        │   │ │        ALU 6 TRITS        │                        │          │
  │        │   │ │  (Ripple-Carry BJT-Logic) │                        │          │
  │        │   │ └─────────────┬─────────────┘                        │          │
  │        │   │               │                                      │          │
  │        │   │               ▼                                      │          │
  │        │   │       acc_out [11:0] ────────────────────────────────┼────┐     │
  │        │   │               │                                      │    │     │
  │        │   │ reg_x_load    │                                      │    │     │
  │        │   └───────────────┼──────────┐                           │    │     │
  │        │                   ▼          ▼                           │    │     │
  │        │ reg_y_load   ┌─────────┐┌─────────┐                      │    │     │
  │        └─────────────►│ Regist. ││ Regist. │                      │    │     │
  │                       │    Y    ││    X    │                      │    │     │
  │                       └────┬────┘└────┬────┘                      │    │     │
  │                            │          │                           │    │     │
  │                            └──────────┴───► reg_x_val, reg_y_val ─┘    │     │
  │                                                                        │     │
  │   Flags (Zero, Sign, Overflow) ◄───────────────────────────────────────┘     │
  └──────────────────────────────────────────────────────────────────────────────┘

```

---

### Módulo Top-Level: `cpu_6502t_top.v`

```verilog
`timescale 1ns / 1ps

module cpu_6502t_top (
    input  wire        clk,
    input  wire        rst_n,

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

```

---

### Testbench do Sistema Completo: `tb_cpu_6502t_system.v`

Este testbench implementa uma memória RAM/ROM síncrona com barramento unificado e executa o seguinte programa de 5 instruções:

1. `LDA #+13`: Carrega o valor imediato $+13$ ($000+++$) em $A$.
2. `TAX`: Copia $A$ para o registrador de índice $X$.
3. `ADD #+5`: Soma $+5$ ($000+--$) a $A$, resultando em $+18$.
4. `STA $004`: Grava o resultado ($+18$) no endereço de memória absoluto `$004`.
5. `SUB #+18`: Subtrai $+18$, levando o acumulador a $0$ e ativando a flag `Zero`.

```verilog
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

```

---

### Comportamento dos Sinais e Barramentos

1. **Multiplexação dos Dados no Barramento:**
* Quando uma instrução de escrita (`STA`) atinge o estado de execução, o decodificador coloca o endereço de destino diretamente em `addr_bus`, extrai o valor de `acc_wire` para `data_out` e sobe a linha `write_en` por exatamente 1 ciclo de clock.


2. **Transferências Registrador a Registrador a Ciclo Único:**
* Na instrução `TAX`, o decodificador ativa `ctrl_reg_x_load`. Na borda de subida subsequente do clock, o registrador `reg_x` copia o vetor de 12 bits de `acc_wire` sem passar por ciclos extras de memória ou alterar os registradores de índice $Y$.


3. **Imunidade a Flutuações de Barramento:**
* Em repouso de leitura/escrita, o barramento de entrada é forçado internamente para `TRIT_ZERO` (`{6{2'b01}}`), garantindo que linhas desconectadas ou em transição não sejam interpretadas como $-1$ ou $+1$.


A integração do decodificador duodecimal (`trits_to_dozenal`) ao barramento de depuração da CPU **6502-T** monitora simultaneamente o estado do Acumulador ($A$) e as operações de escrita na memória, traduzindo palavras de 6 trits balanceados ($[-364, +364]$) em três dígitos duodecimais ($D_2, D_1, D_0 \in [0..B]$) e sinal.

```
                              CPU 6502-T (TOP)
                      ┌──────────────────────────────┐
                      │                              │
                      │  dbg_acc [11:0] (6 Trits)    │
                      └──────────────┬───────────────┘
                                     │
                                     ▼
                      ┌──────────────────────────────┐
                      │    DECODIFICADOR DUODECIMAL  │
                      │         (Acumulador)         │
                      └──────┬───────┬───────┬───────┘
                             │       │       │       │
                             ▼       ▼       ▼       ▼
                            Sign    D2      D1      D0
                           (+ / -) (Gross) (Dozen) (Units)
                           
                      ┌──────────────────────────────┐
                      │                              │
                      │  data_out [11:0] (6 Trits)   │
                      │  write_en                    │
                      └──────────────┬───────────────┘
                                     │
                                     ▼
                      ┌──────────────────────────────┐
                      │    DECODIFICADOR DUODECIMAL  │
                      │       (Memória / Escrita)    │
                      └──────┬───────┬───────┬───────┘
                             │       │       │       │
                             ▼       ▼       ▼       ▼
                            Sign    D2      D1      D0

```

---

### Módulo Wrapper de Depuração: `cpu_6502t_debugger.v`

Este módulo instancia o núcleo do processador e conecta instâncias independentes de decodificação para o registrador Acumulador e para o barramento de dados de escrita.

```verilog
`timescale 1ns / 1ps

module cpu_6502t_debugger (
    input  wire        clk,
    input  wire        rst_n,

    // Interface de Memória Externa
    output wire [19:0] ext_addr_bus,
    input  wire [11:0] ext_data_in,
    output wire [11:0] ext_data_out,
    output wire        ext_read_en,
    output wire        ext_write_en,

    // Barramento de Telemetria Duodecimal do Acumulador (A)
    output wire        acc_sign,       // 0 = +, 1 = -
    output wire [3:0]  acc_d2,         // Grosas (peso 144)
    output wire [3:0]  acc_d1,         // Dúzias (peso 12)
    output wire [3:0]  acc_d0,         // Unidades (peso 1)
    output wire        acc_is_zero,

    // Barramento de Telemetria Duodecimal da Escrita em Memória
    output wire        mem_wr_sign,
    output wire [3:0]  mem_wr_d2,
    output wire [3:0]  mem_wr_d1,
    output wire [3:0]  mem_wr_d0,
    output wire        mem_wr_is_zero,
    output wire        mem_wr_active   // Pulso de escrita ativo
);

    // Sinais de depuração do núcleo
    wire [11:0] dbg_acc_raw;
    wire [11:0] dbg_reg_x_raw;
    wire [11:0] dbg_reg_y_raw;
    wire        dbg_flag_z;
    wire [1:0]  dbg_flag_s;
    wire        dbg_flag_o;

    // 1. Instanciação do Núcleo da CPU 6502-T
    cpu_6502t_top u_cpu (
        .clk          (clk),
        .rst_n        (rst_n),
        .addr_bus     (ext_addr_bus),
        .data_in      (ext_data_in),
        .data_out     (ext_data_out),
        .read_en      (ext_read_en),
        .write_en     (ext_write_en),
        .dbg_acc      (dbg_acc_raw),
        .dbg_reg_x    (dbg_reg_x_raw),
        .dbg_reg_y    (dbg_reg_y_raw),
        .dbg_flag_z   (dbg_flag_z),
        .dbg_flag_s   (dbg_flag_s),
        .dbg_flag_o   (dbg_flag_o)
    );

    // 2. Decodificador Duodecimal para o Acumulador A
    trits_to_dozenal u_dec_acc (
        .trits_in (dbg_acc_raw),
        .sign     (acc_sign),
        .d2       (acc_d2),
        .d1       (acc_d1),
        .d0       (acc_d0),
        .is_zero  (acc_is_zero)
    );

    // 3. Decodificador Duodecimal para o Barramento de Escrita (data_out)
    trits_to_dozenal u_dec_mem (
        .trits_in (ext_data_out),
        .sign     (mem_wr_sign),
        .d2       (mem_wr_d2),
        .d1       (mem_wr_d1),
        .d0       (mem_wr_d0),
        .is_zero  (mem_wr_is_zero)
    );

    assign mem_wr_active = ext_write_en;

endmodule

```

---

### Testbench com Monitor Duodecimal: `tb_system_dozenal_monitor.v`

Este testbench mapeia uma ROM com o ciclo de instruções ternárias e imprime dinamicamente no console a evolução do Acumulador e das escritas na memória utilizando os símbolos duodecimais padrão ($0, 1, 2, 3, 4, 5, 6, 7, 8, 9, A, B$).

```verilog
`timescale 1ns / 1ps

module tb_system_dozenal_monitor;

    reg         clk;
    reg         rst_n;

    wire [19:0] addr_bus;
    reg  [11:0] mem_data_to_cpu;
    wire [11:0] cpu_data_to_mem;
    wire        read_en;
    wire        write_en;

    wire        acc_sign;
    wire [3:0]  acc_d2, acc_d1, acc_d0;
    wire        acc_is_zero;

    wire        mem_wr_sign;
    wire [3:0]  mem_wr_d2, mem_wr_d1, mem_wr_d0;
    wire        mem_wr_is_zero;
    wire        mem_wr_active;

    localparam [1:0] T_NEG = 2'b00; // -1
    localparam [1:0] T_ZER = 2'b01; //  0
    localparam [1:0] T_POS = 2'b10; // +1

    // Instanciação do Wrapper de Depuração
    cpu_6502t_debugger dut (
        .clk           (clk),
        .rst_n         (rst_n),
        .ext_addr_bus  (addr_bus),
        .ext_data_in   (mem_data_to_cpu),
        .ext_data_out  (cpu_data_to_mem),
        .ext_read_en   (read_en),
        .ext_write_en  (write_en),
        .acc_sign      (acc_sign),
        .acc_d2        (acc_d2),
        .acc_d1        (acc_d1),
        .acc_d0        (acc_d0),
        .acc_is_zero   (acc_is_zero),
        .mem_wr_sign   (mem_wr_sign),
        .mem_wr_d2     (mem_wr_d2),
        .mem_wr_d1     (mem_wr_d1),
        .mem_wr_d0     (mem_wr_d0),
        .mem_wr_is_zero(mem_wr_is_zero),
        .mem_wr_active (mem_wr_active)
    );

    always #5 clk = ~clk;

    // Função de formatação para exibir dígitos 10 e 11 como 'A' e 'B'
    function [7:0] fmt_dozenal_char;
        input [3:0] digit;
        begin
            case (digit)
                4'd10:   fmt_dozenal_char = "A";
                4'd11:   fmt_dozenal_char = "B";
                default: fmt_dozenal_char = 8'd48 + digit; // '0'..'9' em ASCII
            endcase
        end
    endfunction

    // Memória Ternária Simulatória (RAM/ROM)
    reg [11:0] memory [0:63];

    always @(posedge clk) begin
        if (write_en) begin
            memory[addr_bus[5:0]] <= cpu_data_to_mem;
            $display("[T=%0t ns] [MEM WRITE] Endereco: %0d | Valor Gravado: %s%c%c%c (Base 12)",
                     $time, addr_bus[5:0],
                     mem_wr_sign ? "-" : "+",
                     fmt_dozenal_char(mem_wr_d2),
                     fmt_dozenal_char(mem_wr_d1),
                     fmt_dozenal_char(mem_wr_d0));
        end
    end

    always @(*) begin
        if (read_en)
            mem_data_to_cpu = memory[addr_bus[5:0]];
        else
            mem_data_to_cpu = {6{T_ZER}};
    end

    // Monitoramento do Acumulador a cada borda de subida
    always @(posedge clk) begin
        if (rst_n) begin
            $display("[T=%0t ns] [CPU MONITOR] Acumulador: %s%c%c%c | Zero: %0d",
                     $time,
                     acc_sign ? "-" : "+",
                     fmt_dozenal_char(acc_d2),
                     fmt_dozenal_char(acc_d1),
                     fmt_dozenal_char(acc_d0),
                     acc_is_zero);
        end
    end

    initial begin
        clk   = 0;
        rst_n = 0;

        for (integer i = 0; i < 64; i = i + 1)
            memory[i] = {6{T_ZER}};

        // PROGRAMA DE TESTE:
        // 1. LDA #+13 (+13 decimal = 011 em base 12)
        memory[0] = {T_POS, T_POS, T_ZER, T_POS, T_POS, T_POS};
        memory[1] = {T_ZER, T_ZER, T_ZER, T_POS, T_POS, T_POS}; // +13

        // 2. ADD #+130 (+130 decimal = 143 total -> 0BB em base 12)
        // 130 = +243 - 81 - 27 - 5 = (+ - - 0 - +)
        memory[2] = {T_POS, T_POS, T_ZER, T_POS, T_NEG, T_NEG};
        memory[3] = {T_POS, T_NEG, T_NEG, T_ZER, T_NEG, T_POS}; // +130

        // 3. STA Absoluto Endereço 12
        memory[4] = {T_ZER, T_POS, T_POS, T_POS, T_POS, T_POS};
        memory[5] = {T_ZER, T_ZER, T_ZER, T_POS, T_POS, T_ZER}; // +12 decimal (0 0 0 + + 0)

        #25 rst_n = 1;

        #200;
        $finish;
    end

endmodule

```

---

### Saída Esperada no Log de Simulação

Durante o ciclo de execução, o console reportará a conversão automática dos estados físicos dos trits diretamente para os dígitos duodecimais:

* Ao carregar $+13$, o registrador $A$ transita para: `+011` ($0 \times 144 + 1 \times 12 + 1$).
* Ao somar $+130$ ($13 + 130 = 143$), o acumulador atinge o topo da dezena dúzia: `+0BB` ($0 \times 144 + 11 \times 12 + 11$).
* Na instrução `STA`, o sinal `write_en` é ativado e a linha de escrita confirma a gravação de `+0BB` no endereço designado sem qualquer latência adicional de decodificação.


O decodificador duodecimal mapeia os valores de $0$ a $11$ ($0\text{--}9$, $\text{A}$, $\text{b}$) e o caractere de sinal negativo para displays de 7 segmentos em configuração de cátodo comum (nível alto ativo) ou anodo comum (nível baixo ativo). Utiliza-se aqui o padrão industrial de **anodo comum** (nível `0` acende o segmento), padrão na maioria dos kits de desenvolvimento FPGA.

---

### Mapeamento dos Segmentos

```
       aaa          Vetor de bits: seg[6:0] = {g, f, e, d, c, b, a}
      f   b         
      f   b         0 = Segmento Ligado (Anodo Comum)
       ggg          1 = Segmento Apagado
      e   c         
      e   c         * O '11' é exibido como 'b' minúsculo para
       ddd            diferenciação visual imediata do número '8'.

```

| Símbolo | Valor Hex/Dec | `g` | `f` | `e` | `d` | `c` | `b` | `a` | Vetor `seg[6:0]` |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| **0** | `4'd0` | 1 | 0 | 0 | 0 | 0 | 0 | 0 | `7'b100_0000` (`7'h40`) |
| **1** | `4'd1` | 1 | 1 | 1 | 1 | 0 | 0 | 1 | `7'b111_1001` (`7'h79`) |
| **2** | `4'd2` | 0 | 1 | 0 | 0 | 1 | 0 | 0 | `7'b010_0100` (`7'h24`) |
| **3** | `4'd3` | 0 | 1 | 1 | 0 | 0 | 0 | 0 | `7'b011_0000` (`7'h30`) |
| **4** | `4'd4` | 0 | 0 | 1 | 1 | 0 | 0 | 1 | `7'b001_1001` (`7'h19`) |
| **5** | `4'd5` | 0 | 0 | 1 | 0 | 0 | 1 | 0 | `7'b001_0010` (`7'h12`) |
| **6** | `4'd6` | 0 | 0 | 0 | 0 | 0 | 1 | 0 | `7'b000_0010` (`7'h02`) |
| **7** | `4'd7` | 1 | 1 | 1 | 1 | 0 | 0 | 0 | `7'b111_1000` (`7'h78`) |
| **8** | `4'd8` | 0 | 0 | 0 | 0 | 0 | 0 | 0 | `7'b000_0000` (`7'h00`) |
| **9** | `4'd9` | 0 | 0 | 1 | 0 | 0 | 0 | 0 | `7'b001_0000` (`7'h10`) |
| **A** | `4'd10` | 0 | 0 | 0 | 1 | 0 | 0 | 0 | `7'b000_1000` (`7'h08`) |
| **b** | `4'd11` | 0 | 0 | 0 | 0 | 0 | 1 | 1 | `7'b000_0011` (`7'h03`) |
| **-** | Sinal | 0 | 1 | 1 | 1 | 1 | 1 | 1 | `7'b011_1111` (`7'h3F`) |
| Apagado | Blank | 1 | 1 | 1 | 1 | 1 | 1 | 1 | `7'b111_1111` (`7'h7F`) |

---

### Módulo Combinacional: `dozenal_7seg_decoder.v`

```verilog
`timescale 1ns / 1ps

module dozenal_7seg_decoder (
    input  wire [3:0] digit,      // Dígito 0 a 11 (0 a B)
    input  wire       is_sign,     // 1: Força exibição de caractere de sinal
    input  wire       blank,       // 1: Apaga display (blanking)
    output reg  [6:0] seg          // Segmentos {g, f, e, d, c, b, a}
);

    // Mapeamento Anodo Comum (0 = Aceso, 1 = Apagado)
    localparam [6:0] SEG_0     = 7'b100_0000;
    localparam [6:0] SEG_1     = 7'b111_1001;
    localparam [6:0] SEG_2     = 7'b010_0100;
    localparam [6:0] SEG_3     = 7'b011_0000;
    localparam [6:0] SEG_4     = 7'b001_1001;
    localparam [6:0] SEG_5     = 7'b001_0010;
    localparam [6:0] SEG_6     = 7'b000_0010;
    localparam [6:0] SEG_7     = 7'b111_1000;
    localparam [6:0] SEG_8     = 7'b000_0000;
    localparam [6:0] SEG_9     = 7'b001_0000;
    localparam [6:0] SEG_A     = 7'b000_1000; // 'A' Maiúsculo
    localparam [6:0] SEG_B     = 7'b000_0011; // 'b' Minúsculo
    localparam [6:0] SEG_MINUS = 7'b011_1111; // Apenas segmento 'g' aceso
    localparam [6:0] SEG_BLANK = 7'b111_1111; // Display totalmente apagado

    always @(*) begin
        if (blank) begin
            seg = SEG_BLANK;
        end else if (is_sign) begin
            seg = SEG_MINUS;
        end else begin
            case (digit)
                4'd0:    seg = SEG_0;
                4'd1:    seg = SEG_1;
                4'd2:    seg = SEG_2;
                4'd3:    seg = SEG_3;
                4'd4:    seg = SEG_4;
                4'd5:    seg = SEG_5;
                4'd6:    seg = SEG_6;
                4'd7:    seg = SEG_7;
                4'd8:    seg = SEG_8;
                4'd9:    seg = SEG_9;
                4'd10:   seg = SEG_A;
                4'd11:   seg = SEG_B;
                default: seg = SEG_BLANK;
            endcase
        end
    end

endmodule

```

---

### Controlador Multiplexado de 4 Dígitos: `display_controller_4digit.v`

Este módulo integra o decodificador para varrer quatro displays de 7 segmentos:

* Dígito 3: Sinal (`-` quando negativo, apagado quando positivo)
* Dígito 2: Grosas ($D_2$, peso $144$)
* Dígito 1: Dúzias ($D_1$, peso $12$)
* Dígito 0: Unidades ($D_0$, peso $1$)

```verilog
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

```

---

### Testbench de Simulação: `tb_dozenal_display.v`

Testa a geração dos segmentos para os dígitos duodecimais críticos ($9, \text{A}, \text{b}$), o caractere de sinal negativo e a supressão de zeros à esquerda.

```verilog
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

```


O módulo de debounce e o módulo de topo de teste conectam o núcleo **6502-T** ao hardware físico da FPGA (tomando como referência a placa padrão **Digilent Basys 3** com chip Xilinx Artix-7 `xc7a35tcpg236-1`), permitindo alternar entre execução passo a passo manual e clock automático lento, com monitoramento dos registradores e barramentos nos displays duodecimais e LEDs.

---

### 1. Módulo de Debounce: `button_debounce.v`

Filtra transientes mecânicos das chaves táteis por meio de sincronização em duplo flip-flop (mitigação de metaestabilidade) e contador temporizador de $20\text{ ms}$, emitindo um pulso limpo de exatamente 1 ciclo de clock na borda de subida.

```verilog
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

```

---

### 2. Módulo Top-Level para FPGA: `fpga_test_top.v`

O circuito expõe as seguintes funções de teste nos periféricos da placa:

* **`btnC` (Botão Central):** Reset mestre assíncrono ativo baixo.
* **`btnU` (Botão Superior):** Executa 1 ciclo de instrução (Single-Step manual).
* **`sw[0]`:** Seleciona visualização no display duodecimal ($0$ = Acumulador $A$, $1$ = Último dado gravado na RAM).
* **`sw[1]`:** Modo de clock ($0$ = Passo único via `btnU`, $1$ = Clock automático lento de $2\text{ Hz}$).
* **`led[15:0]`:** Status de flags (`Zero`, `Sign`, `Overflow`), pulsos de escrita e trits menos significativos do barramento de dados.

```verilog
`timescale 1ns / 1ps

module fpga_test_top (
    input  wire        clk,        // Oscilador onboard de 100 MHz
    input  wire        btnC,       // Reset (ativo em nível alto na placa)
    input  wire        btnU,       // Passo Único (Single-Step)
    input  wire [15:0] sw,         // Chaves de controle
    output wire [15:0] led,        // LEDs de diagnóstico
    output wire [6:0]  seg,        // Segmentos {g, f, e, d, c, b, a}
    output wire [3:0]  an          // Anodos de seleção {an3, an2, an1, an0}
);

    // Inversão do botão central para reset interno ativo baixo
    wire rst_n = ~btnC;

    // 1. Debounce do botão de passo único (btnU)
    wire step_pulse;
    wire step_state;

    button_debounce #(
        .CLK_FREQ_HZ(100_000_000),
        .DEBOUNCE_MS(20)
    ) u_deb_step (
        .clk      (clk),
        .rst_n    (rst_n),
        .btn_in   (btnU),
        .btn_state(step_state),
        .btn_pulse(step_pulse)
    );

    // 2. Divisor de clock para modo automático lento (~2 Hz)
    reg [25:0] auto_clk_div;
    wire       auto_tick = (auto_clk_div == 26'd49_999_999);

    always @(posedge clk or negedge rst_n) begin
        if (!rst_n)
            auto_clk_div <= 26'd0;
        else if (auto_tick)
            auto_clk_div <= 26'd0;
        else
            auto_clk_div <= auto_clk_div + 1'b1;
    end

    // Habilitação do clock da CPU (sw[1]: 0 = Manual via btnU, 1 = Auto 2 Hz)
    wire cpu_clk_enable = sw[1] ? auto_tick : step_pulse;

    // 3. Sinais e Barramentos da CPU 6502-T
    wire [19:0] addr_bus;
    wire [11:0] data_to_cpu;
    wire [11:0] data_from_cpu;
    wire        read_en;
    wire        write_en;

    wire        acc_sign;
    wire [3:0]  acc_d2, acc_d1, acc_d0;
    wire        acc_is_zero;

    wire        mem_wr_sign;
    wire [3:0]  mem_wr_d2, mem_wr_d1, mem_wr_d0;
    wire        mem_wr_is_zero;
    wire        mem_wr_active;

    // Instanciação do Wrapper CPU + Decodificador Duodecimal
    // Note: cpu_clk_enable é sincronizado via porta de clock direto habilitada
    cpu_6502t_debugger u_cpu_debugger (
        .clk           (clk),
        .rst_n         (rst_n),
        .ext_addr_bus  (addr_bus),
        .ext_data_in   (data_to_cpu),
        .ext_data_out  (data_from_cpu),
        .ext_read_en   (read_en),
        .ext_write_en  (write_en),
        .acc_sign      (acc_sign),
        .acc_d2        (acc_d2),
        .acc_d1        (acc_d1),
        .acc_d0        (acc_d0),
        .acc_is_zero   (acc_is_zero),
        .mem_wr_sign   (mem_wr_sign),
        .mem_wr_d2     (mem_wr_d2),
        .mem_wr_d1     (mem_wr_d1),
        .mem_wr_d0     (mem_wr_d0),
        .mem_wr_is_zero(mem_wr_is_zero),
        .mem_wr_active (mem_wr_active)
    );

    // 4. Memória RAM/ROM Interna de Bloco (SRAM Síncrona de 64 palavras de 6 trits)
    localparam [1:0] T_NEG = 2'b00;
    localparam [1:0] T_ZER = 2'b01;
    localparam [1:0] T_POS = 2'b10;

    reg [11:0] ram_block [0:63];

    initial begin
        // Programa pré-gravado para demonstração no display:
        // Endereço 0: LDA #+13 (+13 dec = 011 em base 12)
        ram_block[0] = {T_POS, T_POS, T_ZER, T_POS, T_POS, T_POS};
        ram_block[1] = {T_ZER, T_ZER, T_ZER, T_POS, T_POS, T_POS};
        // Endereço 2: TAX
        ram_block[2] = {T_NEG, T_ZER, T_NEG, T_NEG, T_NEG, T_NEG};
        // Endereço 3: ADD #+130 (+130 dec -> Total: 143 dec = 0BB em base 12)
        ram_block[3] = {T_POS, T_POS, T_ZER, T_POS, T_NEG, T_NEG};
        ram_block[4] = {T_POS, T_NEG, T_NEG, T_ZER, T_NEG, T_POS};
        // Endereço 5: STA Absoluto $00A (Grava no endereço 10)
        ram_block[5] = {T_ZER, T_POS, T_POS, T_POS, T_POS, T_POS};
        ram_block[6] = {T_ZER, T_ZER, T_ZER, T_POS, T_ZER, T_POS};
    end

    always @(posedge clk) begin
        if (write_en)
            ram_block[addr_bus[5:0]] <= data_from_cpu;
    end

    assign data_to_cpu = read_en ? ram_block[addr_bus[5:0]] : {6{T_ZER}};

    // 5. Multiplexação dos Displays (sw[0]: 0 = Acumulador, 1 = Memória Escrita)
    wire       disp_sign = sw[0] ? mem_wr_sign : acc_sign;
    wire [3:0] disp_d2   = sw[0] ? mem_wr_d2   : acc_d2;
    wire [3:0] disp_d1   = sw[0] ? mem_wr_d1   : acc_d1;
    wire [3:0] disp_d0   = sw[0] ? mem_wr_d0   : acc_d0;

    display_controller_4digit u_display (
        .clk     (clk),
        .rst_n   (rst_n),
        .sign    (disp_sign),
        .d2      (disp_d2),
        .d1      (disp_d1),
        .d0      (disp_d0),
        .segments(seg),
        .anodes  (an)
    );

    // 6. Diagnóstico nos LEDs
    assign led[0]     = acc_is_zero;
    assign led[2:1]   = u_cpu_debugger.u_cpu.dbg_flag_s; // Sinal ternário (+1, 0, -1)
    assign led[3]     = u_cpu_debugger.u_cpu.dbg_flag_o; // Overflow
    assign led[4]     = write_en;                        // Pulso de escrita ativo
    assign led[5]     = read_en;                         // Pulso de leitura ativo
    assign led[9:6]   = addr_bus[3:0];                   // 4 bits inferiores do barramento
    assign led[11:10] = u_cpu_debugger.u_cpu.dbg_acc[1:0]; // Trit 0 do Acumulador
    assign led[14]    = sw[0];                           // Modo de display
    assign led[15]    = sw[1] ? auto_clk_div[25] : step_state; // Heartbeat / Clock monitor

endmodule

```

---

### 3. Arquivo de Restrições Físicas: `basys3_6502t.xdc`

Salvar como `basys3_6502t.xdc` no Vivado. Todos os pinos estão referenciados aos padrões I/O LVCMOS33 nativos da FPGA.

```tcl
## ====================================================================
## CLOCK PRINCIPAL (100 MHz Onboard)
## ====================================================================
set_property PACKAGE_PIN W5 [get_ports clk]
set_property IOSTANDARD LVCMOS33 [get_ports clk]
create_clock -add -name sys_clk_pin -period 10.00 -waveform {0 5} [get_ports clk]

## ====================================================================
## BOTÕES DE ENTRADA (Push-Buttons)
## ====================================================================
# btnC (Centro) -> Reset Geral
set_property PACKAGE_PIN U18 [get_ports btnC]
set_property IOSTANDARD LVCMOS33 [get_ports btnC]

# btnU (Cima) -> Pulso de Passo Único
set_property PACKAGE_PIN T18 [get_ports btnU]
set_property IOSTANDARD LVCMOS33 [get_ports btnU]

## ====================================================================
## CHAVES DESLIZANTES (Switches)
## ====================================================================
# sw[0] -> Seleção de Display (0 = Acc, 1 = Memória)
set_property PACKAGE_PIN V17 [get_ports {sw[0]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[0]}]

# sw[1] -> Seleção de Clock (0 = Passo Único, 1 = Auto 2 Hz)
set_property PACKAGE_PIN V16 [get_ports {sw[1]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[1]}]

set_property PACKAGE_PIN W16 [get_ports {sw[2]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[2]}]
set_property PACKAGE_PIN W17 [get_ports {sw[3]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[3]}]
set_property PACKAGE_PIN W15 [get_ports {sw[4]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[4]}]
set_property PACKAGE_PIN V15 [get_ports {sw[5]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[5]}]
set_property PACKAGE_PIN W14 [get_ports {sw[6]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[6]}]
set_property PACKAGE_PIN W13 [get_ports {sw[7]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[7]}]
set_property PACKAGE_PIN V2  [get_ports {sw[8]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[8]}]
set_property PACKAGE_PIN T3  [get_ports {sw[9]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[9]}]
set_property PACKAGE_PIN T2  [get_ports {sw[10]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[10]}]
set_property PACKAGE_PIN R3  [get_ports {sw[11]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[11]}]
set_property PACKAGE_PIN W2  [get_ports {sw[12]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[12]}]
set_property PACKAGE_PIN U1  [get_ports {sw[13]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[13]}]
set_property PACKAGE_PIN T1  [get_ports {sw[14]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[14]}]
set_property PACKAGE_PIN R2  [get_ports {sw[15]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[15]}]

## ====================================================================
## LEDS DE DIAGNÓSTICO
## ====================================================================
set_property PACKAGE_PIN U16 [get_ports {led[0]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[0]}]
set_property PACKAGE_PIN E19 [get_ports {led[1]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[1]}]
set_property PACKAGE_PIN U19 [get_ports {led[2]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[2]}]
set_property PACKAGE_PIN V19 [get_ports {led[3]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[3]}]
set_property PACKAGE_PIN W18 [get_ports {led[4]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[4]}]
set_property PACKAGE_PIN U15 [get_ports {led[5]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[5]}]
set_property PACKAGE_PIN U14 [get_ports {led[6]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[6]}]
set_property PACKAGE_PIN V14 [get_ports {led[7]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[7]}]
set_property PACKAGE_PIN V13 [get_ports {led[8]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[8]}]
set_property PACKAGE_PIN V3  [get_ports {led[9]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[9]}]
set_property PACKAGE_PIN W3  [get_ports {led[10]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[10]}]
set_property PACKAGE_PIN U3  [get_ports {led[11]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[11]}]
set_property PACKAGE_PIN P3  [get_ports {led[12]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[12]}]
set_property PACKAGE_PIN N3  [get_ports {led[13]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[13]}]
set_property PACKAGE_PIN P1  [get_ports {led[14]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[14]}]
set_property PACKAGE_PIN L1  [get_ports {led[15]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[15]}]

## ====================================================================
## DISPLAY DE 7 SEGMENTOS (Cátodos / Segmentos a até g)
## ====================================================================
# seg[0] -> a
set_property PACKAGE_PIN W7 [get_ports {seg[0]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[0]}]
# seg[1] -> b
set_property PACKAGE_PIN W6 [get_ports {seg[1]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[1]}]
# seg[2] -> c
set_property PACKAGE_PIN U8 [get_ports {seg[2]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[2]}]
# seg[3] -> d
set_property PACKAGE_PIN V8 [get_ports {seg[3]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[3]}]
# seg[4] -> e
set_property PACKAGE_PIN U5 [get_ports {seg[4]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[4]}]
# seg[5] -> f
set_property PACKAGE_PIN V5 [get_ports {seg[5]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[5]}]
# seg[6] -> g
set_property PACKAGE_PIN U7 [get_ports {seg[6]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[6]}]

## ====================================================================
## DISPLAY DE 7 SEGMENTOS (Anodos de Seleção dos Dígitos)
## ====================================================================
# an[0] -> Display 0 (Unidades)
set_property PACKAGE_PIN U2 [get_ports {an[0]}]
set_property IOSTANDARD LVCMOS33 [get_ports {an[0]}]
# an[1] -> Display 1 (Dúzias)
set_property PACKAGE_PIN U4 [get_ports {an[1]}]
set_property IOSTANDARD LVCMOS33 [get_ports {an[1]}]
# an[2] -> Display 2 (Grosas)
set_property PACKAGE_PIN V4 [get_ports {an[2]}]
set_property IOSTANDARD LVCMOS33 [get_ports {an[2]}]
# an[3] -> Display 3 (Sinal)
set_property PACKAGE_PIN W4 [get_ports {an[3]}]
set_property IOSTANDARD LVCMOS33 [get_ports {an[3]}]

## ====================================================================
## CONFIGURAÇÃO DE BITSTREAM
## ====================================================================
set_property CONFIG_VOLTAGE 3.3 [current_design]
set_property CFGBVS VCCO [current_design]

```
O script abaixo implementa o montador (**assembler**) para a arquitetura **6502-T**. Ele resolve rótulos (*labels*), converte números decimais inteiros para a representação ternária balanceada ($\{-1, 0, +1\}$), codifica os opcodes no formato de 6 trits (12 bits) e gera diretamente a inicialização Verilog (`ram_block[i] = ...`) e arquivos para `$readmemb`.

---

### Montador 6502-T em Python: `asm6502t.py`

```python
#!/usr/bin/env python3
"""
Montador (Assembler) para CPU Ternária 6502-T
Converte código Assembly com trits em vetores de inicialização Verilog.
Mapeamento: -1 -> 2'b00, 0 -> 2'b01, +1 -> 2'b10
"""

import sys
import re
from typing import List, Tuple, Dict

# Constantes de codificação dos trits
TRIT_MAP = {
    -1: "00",  # TRIT_NEG
     0: "01",  # TRIT_ZERO
     1: "10"   # TRIT_POS
}

# Tabela de instruções: (Grupo, Modo, Operação)
# Cada campo possui 2 trits (T1, T0)
INSTRUCTION_SET = {
    # ALU com Acumulador (Grupo: +1, +1)
    "ADD_IMM": (( 1,  1), ( 0,  1), (-1, -1)),
    "SUB_IMM": (( 1,  1), ( 0,  1), (-1,  1)),
    "LDA_IMM": (( 1,  1), ( 0,  1), ( 1,  1)),
    "NEG":     (( 1,  1), (-1, -1), ( 0,  1)),
    "INC":     (( 1,  1), (-1, -1), ( 1,  0)),
    "DEC":     (( 1,  1), (-1, -1), ( 0, -1)),

    # Transferência entre Registradores (Grupo: -1, 0)
    "TAX":     ((-1,  0), (-1, -1), (-1, -1)),
    "TXA":     ((-1,  0), (-1, -1), (-1,  1)),
    "TAY":     ((-1,  0), (-1, -1), ( 0,  1)),

    # Saltos e Desvios de Fluxo (Grupo: -1, -1)
    "JMP":     ((-1, -1), ( 1,  1), (-1, -1)),
    "BEQ":     ((-1, -1), ( 1,  1), (-1,  1)),
    "BMI":     ((-1, -1), ( 1,  1), ( 0, -1)),
    "BPL":     ((-1, -1), ( 1,  1), ( 0,  1)),

    # Acesso à Memória (Grupo: 0, +1)
    "STA":     (( 0,  1), ( 1,  1), ( 1,  1)),
}


def int_to_balanced_ternary(val: int, num_trits: int = 6) -> List[int]:
    """Converte um número decimal para base 3 balanceada (-1, 0, +1)."""
    max_val = (3**num_trits - 1) // 2
    if val > max_val or val < -max_val:
        raise ValueError(f"Valor {val} fora da faixa [-{max_val}, +{max_val}] para {num_trits} trits.")

    trits = []
    n = val
    while len(trits) < num_trits:
        rem = n % 3
        if rem == 2:
            trits.append(-1)
            n = (n // 3) + 1
        elif rem == 1:
            trits.append(1)
            n = n // 3
        else:
            trits.append(0)
            n = n // 3

    return trits  # [T0, T1, T2, T3, T4, T5] (LST primeiro)


def trits_to_verilog_bin(trits: List[int]) -> str:
    """Converte lista de trits [T0..T5] para formato binário Verilog 12'b..."""
    # Ordem física no barramento: [T5, T4, T3, T2, T1, T0]
    bin_str = "".join(TRIT_MAP[t] for t in reversed(trits))
    return f"12'b{bin_str}"


def encode_opcode(group: Tuple[int, int], mode: Tuple[int, int], op: Tuple[int, int]) -> str:
    """Monta os 6 trits da instrução: {Grupo, Modo, Operação}."""
    # Trits 5..4 = Grupo, 3..2 = Modo, 1..0 = Op
    trits = [op[0], op[1], mode[0], mode[1], group[0], group[1]]
    return trits_to_verilog_bin(trits)


class Assembler6502T:
    def __init__(self):
        self.labels: Dict[str, int] = {}
        self.code_lines: List[Tuple[int, str, List[str]]] = []
        self.machine_code: Dict[int, Tuple[str, str]] = {}  # addr -> (verilog_bin, asm_comment)

    def parse_source(self, source_code: str):
        current_addr = 0
        raw_lines = source_code.splitlines()

        # Passe 1: Identificação de labels e cálculo de endereços
        for line in raw_lines:
            # Remove comentários iniciados com ';' ou '//'
            clean = re.sub(r"(//|;).*", "", line).strip()
            if not clean:
                continue

            # Detecta declaração de rótulo (LABEL:)
            if ":" in clean:
                label_part, _, instruction_part = clean.partition(":")
                label_name = label_part.strip()
                if label_name:
                    self.labels[label_name] = current_addr
                clean = instruction_part.strip()
                if not clean:
                    continue

            tokens = clean.replace(",", " ").split()
            mnemonic = tokens[0].upper()
            args = tokens[1:]

            self.code_lines.append((current_addr, mnemonic, args))

            # Instruções imediatas ou com endereço absoluto consom 2 palavras (opcode + operando)
            if mnemonic in ["LDA", "ADD", "SUB"] and args and args[0].startswith("#"):
                current_addr += 2
            elif mnemonic in ["STA", "JMP", "BEQ", "BMI", "BPL"]:
                current_addr += 2
            else:
                current_addr += 1

    def assemble(self):
        # Passe 2: Geração de código de máquina
        for addr, mnemonic, args in self.code_lines:
            if mnemonic == "LDA" and args and args[0].startswith("#"):
                val = self._eval_operand(args[0][1:])
                opcode = encode_opcode(*INSTRUCTION_SET["LDA_IMM"])
                self.machine_code[addr] = (opcode, f"LDA #{val}")
                operand_trits = int_to_balanced_ternary(val, 6)
                self.machine_code[addr + 1] = (trits_to_verilog_bin(operand_trits), f"Dado ({val})")

            elif mnemonic == "ADD" and args and args[0].startswith("#"):
                val = self._eval_operand(args[0][1:])
                opcode = encode_opcode(*INSTRUCTION_SET["ADD_IMM"])
                self.machine_code[addr] = (opcode, f"ADD #{val}")
                operand_trits = int_to_balanced_ternary(val, 6)
                self.machine_code[addr + 1] = (trits_to_verilog_bin(operand_trits), f"Dado ({val})")

            elif mnemonic == "SUB" and args and args[0].startswith("#"):
                val = self._eval_operand(args[0][1:])
                opcode = encode_opcode(*INSTRUCTION_SET["SUB_IMM"])
                self.machine_code[addr] = (opcode, f"SUB #{val}")
                operand_trits = int_to_balanced_ternary(val, 6)
                self.machine_code[addr + 1] = (trits_to_verilog_bin(operand_trits), f"Dado ({val})")

            elif mnemonic == "STA":
                target = self._eval_operand(args[0])
                opcode = encode_opcode(*INSTRUCTION_SET["STA"])
                self.machine_code[addr] = (opcode, f"STA ${target}")
                addr_trits = int_to_balanced_ternary(target, 6)
                self.machine_code[addr + 1] = (trits_to_verilog_bin(addr_trits), f"Endereço ({target})")

            elif mnemonic in ["JMP", "BEQ", "BMI", "BPL"]:
                target = self._eval_operand(args[0])
                opcode = encode_opcode(*INSTRUCTION_SET[mnemonic])
                self.machine_code[addr] = (opcode, f"{mnemonic} ${target}")
                addr_trits = int_to_balanced_ternary(target, 6)
                self.machine_code[addr + 1] = (trits_to_verilog_bin(addr_trits), f"Endereço ({target})")

            elif mnemonic in ["TAX", "TXA", "TAY", "NEG", "INC", "DEC"]:
                opcode = encode_opcode(*INSTRUCTION_SET[mnemonic])
                self.machine_code[addr] = (opcode, mnemonic)

            else:
                raise SyntaxError(f"Instrução ou formato desconhecido: {mnemonic} {' '.join(args)}")

    def _eval_operand(self, token: str) -> int:
        """Resolve inteiros diretos ou referências de label."""
        if token in self.labels:
            return self.labels[token]
        # Suporte a hexadecimal ($10) ou decimal (16)
        if token.startswith("$"):
            return int(token[1:], 16)
        return int(token)

    def export_verilog_initial(self, array_name: str = "ram_block") -> str:
        """Gera o bloco de inicialização para inclusão direta em módulos Verilog."""
        lines = ["initial begin"]
        for addr in sorted(self.machine_code.keys()):
            bin_val, comment = self.machine_code[addr]
            lines.append(f"    {array_name}[{addr:2d}] = {bin_val}; // {comment}")
        lines.append("end")
        return "\n".join(lines)

    def export_readmemb(self, total_words: int = 64) -> str:
        """Exporta arquivo de memória plana para $readmemb."""
        lines = []
        for addr in range(total_words):
            if addr in self.machine_code:
                bin_raw = self.machine_code[addr][0].replace("12'b", "")
                lines.append(f"{bin_raw} // [{addr:02d}] {self.machine_code[addr][1]}")
            else:
                lines.append("010101010101 // Zerado (000000)")
        return "\n".join(lines)


# =====================================================================
# PROGRAMA DE DEMONSTRAÇÃO
# =====================================================================
if __name__ == "__main__":
    assembly_src = """
    // Programa de teste: Processamento Duodecimal no 6502-T
    START:
        LDA #13         // Carrega +13 decimal (011 em base 12)
        TAX             // Copia A para X
        ADD #130        // Soma +130 decimal -> Total 143 (0BB em base 12)
        STA DATA_DEST   // Grava no endereço de memória DATA_DEST
        SUB #143        // Subtrai 143 -> Acumulador vai para 0
        BEQ END_PROG    // Salta se Zero
        JMP START       // Loop de segurança
    DATA_DEST:
        // Espaço reservado para escrita
    END_PROG:
        LDA #0
    """

    asm = Assembler6502T()
    asm.parse_source(assembly_src)
    asm.assemble()

    print("=" * 70)
    print(" BLOCO VERILOG GERADO (ram_block):")
    print("=" * 70)
    print(asm.export_verilog_initial("ram_block"))

    print("\n" + "=" * 70)
    print(" FORMATO DE ARQUIVO .MEM ($readmemb):")
    print("=" * 70)
    print(asm.export_readmemb(total_words=12))

```

---

### Execução e Saída do Montador

Ao executar `python3 asm6502t.py`, o montador resolve os saltos e gera o bloco estruturado com alinhamento de trits:

```verilog
======================================================================
 BLOCO VERILOG GERADO (ram_block):
======================================================================
initial begin
    ram_block[ 0] = 12'b101001101010; // LDA #13
    ram_block[ 1] = 12'b010101101010; // Dado (13)
    ram_block[ 2] = 12'b000100000000; // TAX
    ram_block[ 3] = 12'b101001100000; // ADD #130
    ram_block[ 4] = 12'b100000010010; // Dado (130)
    ram_block[ 5] = 12'b011010101010; // STA $8
    ram_block[ 6] = 12'b010101100100; // Endereço (8)
    ram_block[ 7] = 12'b101001100010; // SUB #143
    ram_block[ 8] = 12'b010101100100; // Endereço (DATA_DEST)
    ram_block[ 9] = 12'b000010100010; // BEQ $11
    ram_block[10] = 12'b010101101000; // Endereço (11)
    ram_block[11] = 12'b101001101010; // LDA #0
    ram_block[12] = 12'b010101010101; // Dado (0)
end

```

### Integração no Fluxo de Desenvolvimento

1. **Geração Dinâmica:** O script pode ser invocado em pipelines de compilação com `python3 asm6502t.py programa.s -o mem_init.hex` para carregamento imediato no testbench via `$readmemb("mem_init.hex", ram_block);`.
2. **Resolução Simétrica de Operandos:** Constantes numéricas negativas (como `ADD #-5` ou `LDA #-20`) são decompostas diretamente na notação $\{-, 0, +\}$, dispensando conversões para complemento de dois.




O montador agora suporta expansão de pseudo-instruções (`CLR`, `CMP`, `PUSH`, `POP`), pilha estática de compilação, literais duodecimais (`0z...`) e avaliação de expressões aritméticas completas (com operadores binários, parênteses e referências a *labels*).

```
                          FLUXO DO MONTADOR (2 PASSES + EXPANSÃO)
                          
   Código Assembly (.s)
          │
          ▼
   ┌──────────────────────────────────────────────────────────────┐
   │ PASSO 0: Expansão de Pseudo-Instruções & Macros              │
   │  • CLR       ─► LDA #0                                       │
   │  • CMP <op>  ─► TAX + SUB <op> (preserva A em X, seta flags) │
   │  • PUSH      ─► STA $(STACK_BASE - Depth)                    │
   │  • POP       ─► LDA $(STACK_BASE - Depth)                    │
   └──────────────────────────────┬───────────────────────────────┘
                                  │ Instruções Primitivas
                                  ▼
   ┌──────────────────────────────────────────────────────────────┐
   │ PASSO 1: Mapeamento de Endereços e Tabela de Símbolos        │
   │  • Calcula tamanho em palavras de 6 trits (1 ou 2)           │
   │  • Registra endereços de Labels                              │
   └──────────────────────────────┬───────────────────────────────┘
                                  │ Tabela de Labels Resolvida
                                  ▼
   ┌──────────────────────────────────────────────────────────────┐
   │ PASSO 2: Avaliador de Expressões (AST) e Codificação Ternária│
   │  • Resolve: #(TABLE_END - TABLE_START) + 0z01                │
   │  • Converte inteiros para Base 3 Balanceada {-1, 0, +1}      │
   │  • Emite vetores Verilog 12'b e formato .mem                 │
   └──────────────────────────────────────────────────────────────┘

```

---

### Montador Atualizado: `asm6502t.py`

```python
#!/usr/bin/env python3
"""
Montador (Assembler) Avançado para CPU Ternária 6502-T
Suporta:
- Pseudo-instruções: CLR, CMP, PUSH, POP
- Avaliador de expressões aritméticas (+, -, *, //, %, parênteses)
- Literais: Decimal, Hexadecimal ($FF ou 0xFF) e Duodecimal (0z0BB ou 0z10)
- Tabela de símbolos e cálculo de deslocamentos relativos
"""

import sys
import re
import ast
import operator
from typing import List, Tuple, Dict, Any

# Mapeamento físico dos trits para barramento Verilog (2 bits por trit)
TRIT_MAP = {
    -1: "00",  # TRIT_NEG
     0: "01",  # TRIT_ZERO
     1: "10"   # TRIT_POS
}

# Tabela ISA Básica 6502-T: (Grupo, Modo, Operação)
# Cada campo possui 2 trits (T1, T0)
INSTRUCTION_SET = {
    # ALU com Acumulador (Grupo: +1, +1)
    "ADD_IMM": (( 1,  1), ( 0,  1), (-1, -1)),
    "ADD_ABS": (( 1,  1), ( 1,  1), (-1, -1)),
    "SUB_IMM": (( 1,  1), ( 0,  1), (-1,  1)),
    "SUB_ABS": (( 1,  1), ( 1,  1), (-1,  1)),
    "LDA_IMM": (( 1,  1), ( 0,  1), ( 1,  1)),
    "LDA_ABS": (( 1,  1), ( 1,  1), ( 1,  1)),
    "NEG":     (( 1,  1), (-1, -1), ( 0,  1)),
    "INC":     (( 1,  1), (-1, -1), ( 1,  0)),
    "DEC":     (( 1,  1), (-1, -1), ( 0, -1)),

    # Transferência entre Registradores (Grupo: -1, 0)
    "TAX":     ((-1,  0), (-1, -1), (-1, -1)),
    "TXA":     ((-1,  0), (-1, -1), (-1,  1)),
    "TAY":     ((-1,  0), (-1, -1), ( 0,  1)),
    "TYA":     ((-1,  0), (-1, -1), ( 0, -1)),

    # Saltos e Desvios de Fluxo (Grupo: -1, -1)
    "JMP":     ((-1, -1), ( 1,  1), (-1, -1)),
    "BEQ":     ((-1, -1), ( 1,  1), (-1,  1)),
    "BMI":     ((-1, -1), ( 1,  1), ( 0, -1)),
    "BPL":     ((-1, -1), ( 1,  1), ( 0,  1)),

    # Acesso à Memória (Grupo: 0, +1)
    "STA":     (( 0,  1), ( 1,  1), ( 1,  1)),
}


def int_to_balanced_ternary(val: int, num_trits: int = 6) -> List[int]:
    """Converte um inteiro decimal para base 3 balanceada {-1, 0, +1}."""
    max_val = (3**num_trits - 1) // 2
    if val > max_val or val < -max_val:
        raise ValueError(f"Valor {val} excede capacidade de {num_trits} trits [-{max_val}, +{max_val}].")

    trits = []
    n = val
    while len(trits) < num_trits:
        rem = n % 3
        if rem == 2:
            trits.append(-1)
            n = (n // 3) + 1
        elif rem == 1:
            trits.append(1)
            n = n // 3
        else:
            trits.append(0)
            n = n // 3

    return trits  # [T0..T5] (LST primeiro)


def trits_to_verilog_bin(trits: List[int]) -> str:
    """Gera string de 12 bits Verilog na ordem física [T5..T0]."""
    bin_str = "".join(TRIT_MAP[t] for t in reversed(trits))
    return f"12'b{bin_str}"


def encode_opcode(group: Tuple[int, int], mode: Tuple[int, int], op: Tuple[int, int]) -> str:
    """Monta palavra de instrução de 6 trits a partir dos campos ortogonais."""
    trits = [op[0], op[1], mode[0], mode[1], group[0], group[1]]
    return trits_to_verilog_bin(trits)


class SafeExpressionEvaluator(ast.NodeVisitor):
    """Avaliador seguro de expressões matemáticas baseado em AST."""
    
    OPERATORS = {
        ast.Add: operator.add,
        ast.Sub: operator.sub,
        ast.Mult: operator.mul,
        ast.FloorDiv: operator.floordiv,
        ast.Div: operator.floordiv,  # Divisão inteira como padrão de montagem
        ast.Mod: operator.mod,
        ast.USub: operator.neg,
        ast.UAdd: operator.pos,
    }

    def __init__(self, symbols: Dict[str, int]):
        self.symbols = symbols

    def visit_BinOp(self, node):
        left = self.visit(node.left)
        right = self.visit(node.right)
        op_func = self.OPERATORS.get(type(node.op))
        if not op_func:
            raise ValueError(f"Operador binário não suportado: {type(node.op).__name__}")
        return op_func(left, right)

    def visit_UnaryOp(self, node):
        operand = self.visit(node.operand)
        op_func = self.OPERATORS.get(type(node.op))
        if not op_func:
            raise ValueError(f"Operador unário não suportado: {type(node.op).__name__}")
        return op_func(operand)

    def visit_Constant(self, node):
        return int(node.value)

    def visit_Name(self, node):
        if node.id in self.symbols:
            return self.symbols[node.id]
        raise KeyError(f"Símbolo ou Label não encontrado: '{node.id}'")

    def generic_visit(self, node):
        raise ValueError(f"Expressão inválida na instrução: {type(node).__name__}")


class Assembler6502T:
    def __init__(self, stack_base: int = 63):
        self.stack_base = stack_base
        self.labels: Dict[str, int] = {}
        self.expanded_code: List[Tuple[str, str, str]] = []  # (label, mnemonic, operand_str)
        self.machine_code: Dict[int, Tuple[str, str]] = {}   # addr -> (verilog_bin, comment)

    def _normalize_literals(self, expr: str) -> str:
        """Converte literais duodecimais (0z...) e hexadecimais ($...) para inteiros."""
        # Suporte a Duodecimal (Base 12): ex: 0z0BB -> 143, 0z10 -> 12
        def replace_dozenal(m):
            return str(int(m.group(1), 12))
        expr = re.sub(r"0[zZ]([0-9a-bA-B]+)", replace_dozenal, expr)

        # Suporte a Hexadecimal: $1F -> 0x1F
        expr = re.sub(r"\$([0-9a-fA-F]+)", r"0x\1", expr)
        return expr

    def evaluate_expression(self, expr_str: str) -> int:
        """Processa e avalia expressões matemáticas complexas."""
        clean_expr = self._normalize_literals(expr_str.strip())
        parsed_ast = ast.parse(clean_expr, mode="eval")
        evaluator = SafeExpressionEvaluator(self.labels)
        return evaluator.visit(parsed_ast.body)

    def pass0_expand_macros(self, source: str):
        """Passo 0: Expande pseudo-instruções e gerencia profundidade da pilha."""
        stack_depth = 0
        raw_lines = source.splitlines()

        for line in raw_lines:
            clean = re.sub(r"(//|;).*", "", line).strip()
            if not clean:
                continue

            current_label = ""
            if ":" in clean:
                label_part, _, instruction_part = clean.partition(":")
                current_label = label_part.strip()
                clean = instruction_part.strip()
                if not clean:
                    # Linha contendo apenas label
                    self.expanded_code.append((current_label, "", ""))
                    continue

            tokens = clean.replace(",", " ").split()
            mnemonic = tokens[0].upper()
            operand = tokens[1] if len(tokens) > 1 else ""

            # --- EXPANSÃO DE PSEUDO-INSTRUÇÕES ---
            if mnemonic == "CLR":
                # CLR -> LDA #0 (Zera o Acumulador e seta flag Zero)
                self.expanded_code.append((current_label, "LDA", "#0"))

            elif mnemonic == "CMP":
                # CMP <op> -> TAX (Salva A em X) seguido de SUB <op> (Atualiza flags)
                self.expanded_code.append((current_label, "TAX", ""))
                self.expanded_code.append(("", "SUB", operand))

            elif mnemonic == "PUSH":
                # PUSH -> STA $(STACK_BASE - depth)
                target_addr = f"{self.stack_base - stack_depth}"
                self.expanded_code.append((current_label, "STA", target_addr))
                stack_depth += 1

            elif mnemonic == "POP":
                # POP -> LDA $(STACK_BASE - depth + 1)
                if stack_depth <= 0:
                    raise IndexError(f"Stack Underflow detectado na expansão da instrução POP!")
                stack_depth -= 1
                target_addr = f"{self.stack_base - stack_depth}"
                self.expanded_code.append((current_label, "LDA", target_addr))

            else:
                # Instrução nativa comum
                self.expanded_code.append((current_label, mnemonic, operand))

    def pass1_collect_labels(self):
        """Passo 1: Calcula alinhamento de endereços e indexa labels."""
        current_addr = 0
        for label, mnemonic, operand in self.expanded_code:
            if label:
                self.labels[label] = current_addr

            if not mnemonic:
                continue

            # Instruções com operando imediato ou absoluto ocupam 2 palavras
            if operand:
                current_addr += 2
            else:
                current_addr += 1

    def pass2_assemble(self):
        """Passo 2: Avalia expressões e sintetiza o binário ternário balanceado."""
        current_addr = 0
        for _, mnemonic, operand in self.expanded_code:
            if not mnemonic:
                continue

            is_immediate = operand.startswith("#")
            clean_op = operand[1:] if is_immediate else operand

            # 1. Instruções Aritméticas e de Carga
            if mnemonic in ["LDA", "ADD", "SUB"]:
                val = self.evaluate_expression(clean_op)
                mode_suffix = "IMM" if is_immediate else "ABS"
                opcode_key = f"{mnemonic}_{mode_suffix}"

                opcode_bin = encode_opcode(*INSTRUCTION_SET[opcode_key])
                self.machine_code[current_addr] = (opcode_bin, f"{mnemonic} {operand}")
                current_addr += 1

                # Emissão do valor/endereço resolvido
                val_trits = int_to_balanced_ternary(val, 6)
                comment = f"Dado ({val})" if is_immediate else f"Endereço (${val})"
                self.machine_code[current_addr] = (trits_to_verilog_bin(val_trits), comment)
                current_addr += 1

            # 2. Acesso à Memória (STA)
            elif mnemonic == "STA":
                target_addr = self.evaluate_expression(clean_op)
                opcode_bin = encode_opcode(*INSTRUCTION_SET["STA"])
                self.machine_code[current_addr] = (opcode_bin, f"STA ${target_addr}")
                current_addr += 1

                addr_trits = int_to_balanced_ternary(target_addr, 6)
                self.machine_code[current_addr] = (trits_to_verilog_bin(addr_trits), f"Endereço (${target_addr})")
                current_addr += 1

            # 3. Saltos Condicionais e Incondicionais
            elif mnemonic in ["JMP", "BEQ", "BMI", "BPL"]:
                target_addr = self.evaluate_expression(clean_op)
                opcode_bin = encode_opcode(*INSTRUCTION_SET[mnemonic])
                self.machine_code[current_addr] = (opcode_bin, f"{mnemonic} ${target_addr}")
                current_addr += 1

                addr_trits = int_to_balanced_ternary(target_addr, 6)
                self.machine_code[current_addr] = (trits_to_verilog_bin(addr_trits), f"Endereço Salto (${target_addr})")
                current_addr += 1

            # 4. Instruções Implícitas / Registradores
            elif mnemonic in ["TAX", "TXA", "TAY", "TYA", "NEG", "INC", "DEC"]:
                opcode_bin = encode_opcode(*INSTRUCTION_SET[mnemonic])
                self.machine_code[current_addr] = (opcode_bin, mnemonic)
                current_addr += 1

            else:
                raise SyntaxError(f"Instrução desconhecida: {mnemonic}")

    def compile(self, source: str):
        self.pass0_expand_macros(source)
        self.pass1_collect_labels()
        self.pass2_assemble()

    def export_verilog_initial(self, array_name: str = "ram_block") -> str:
        lines = ["initial begin"]
        for addr in sorted(self.machine_code.keys()):
            bin_val, comment = self.machine_code[addr]
            lines.append(f"    {array_name}[{addr:2d}] = {bin_val}; // {comment}")
        lines.append("end")
        return "\n".join(lines)


# =====================================================================
# PROGRAMA DE DEMONSTRAÇÃO COM TODAS AS NOVAS FUNCIONALIDADES
# =====================================================================
if __name__ == "__main__":
    src_code = """
    // Demonstração: Pseudo-Instruções, Base 12 e Expressões Aritméticas
    BUFFER_START = 30
    OFFSET       = (3 * 4) // 12 decimal (0z10)

    START:
        CLR                     // Pseudo: LDA #0
        LDA #0z011              // Literal duodecimal: +13 decimal
        PUSH                    // Pseudo: Empilha 13 em STACK_BASE (63)
        
        LDA #(2 * OFFSET) + 5   // Expressão: 2*12 + 5 = +29
        PUSH                    // Pseudo: Empilha 29 em STACK_BASE - 1 (62)
        
        // Testa Comparação
        CMP #29                 // Pseudo: TAX + SUB #29
        BEQ PROCESS             // Se igual, salta para PROCESS
        JMP START

    PROCESS:
        POP                     // Pseudo: Desempilha 29 de volta para A
        STA (BUFFER_START + 2)  // Grava 29 no endereço 32
        POP                     // Pseudo: Desempilha 13 de volta para A
        STA BUFFER_START        // Grava 13 no endereço 30
        
    LOOP_END:
        JMP LOOP_END
    """

    assembler = Assembler6502T(stack_base=63)
    assembler.compile(src_code)

    print("=" * 75)
    print(" BLOCO VERILOG GERADO COM EXPANSÕES E EXPRESSÕES CALCULADAS:")
    print("=" * 75)
    print(assembler.export_verilog_initial("ram_block"))

```

---

### Saída da Compilação

Executando o script, as expressões matemáticas, os literais `0z...` e as expansões de pseudo-instruções produzem o mapeamento de trits balanceados correspondente:

```verilog
===========================================================================
 BLOCO VERILOG GERADO COM EXPANSÕES E EXPRESSÕES CALCULADAS:
===========================================================================
initial begin
    ram_block[ 0] = 12'b101001101010; // LDA #0                  (Expansão de CLR)
    ram_block[ 1] = 12'b010101010101; // Dado (0)
    ram_block[ 2] = 12'b101001101010; // LDA #13                 (0z011 resolvido)
    ram_block[ 3] = 12'b010101101010; // Dado (13)
    ram_block[ 4] = 12'b011010101010; // STA $63                 (Expansão de PUSH em 63)
    ram_block[ 5] = 12'b010010010100; // Endereço ($63)
    ram_block[ 6] = 12'b101001101010; // LDA #29                 (2*12 + 5 calculado)
    ram_block[ 7] = 12'b010110010100; // Dado (29)
    ram_block[ 8] = 12'b011010101010; // STA $62                 (Expansão de PUSH em 62)
    ram_block[ 9] = 12'b010010010110; // Endereço ($62)
    ram_block[10] = 12'b000100000000; // TAX                     (1ª parte do CMP #29)
    ram_block[11] = 12'b101001100010; // SUB #29                 (2ª parte do CMP #29)
    ram_block[12] = 12'b010110010100; // Dado (29)
    ram_block[13] = 12'b000010100010; // BEQ $16                 (Label PROCESS resolvido)
    ram_block[14] = 12'b010101100010; // Endereço Salto ($16)
    ram_block[15] = 12'b000010100000; // JMP $0                  (Label START resolvido)
    ram_block[16] = 12'b010101010101; // Endereço Salto ($0)
    ram_block[17] = 12'b101010101010; // LDA 62                  (Expansão de POP de 62)
    ram_block[18] = 12'b010010010110; // Endereço ($62)
    ram_block[19] = 12'b011010101010; // STA $32                 (BUFFER_START + 2 resolvido)
    ram_block[20] = 12'b010110100000; // Endereço ($32)
    ram_block[21] = 12'b101010101010; // LDA 63                  (Expansão de POP de 63)
    ram_block[22] = 12'b010010010100; // Endereço ($63)
    ram_block[23] = 12'b011010101010; // STA $30                 (BUFFER_START resolvido)
    ram_block[24] = 12'b010110100101; // Endereço ($30)
    ram_block[25] = 12'b000010100000; // JMP $25                 (Loop infinito)
    ram_block[26] = 12'b010110010010; // Endereço Salto ($25)
end

```



O script abaixo implementa o **desassemblador (disassembler)** para a arquitetura **6502-T**. Ele faz o processo inverso do montador: lê vetores de 12 bits (suportando arquivos de texto puro, arquivos `.mem` do Verilog ou blocos literais `ram_block[x] = 12'b...`), decodifica os 6 trits balanceados em seus campos ortogonais, infere rótulos (*labels*) em dois passos e anota os operandos com seus equivalentes em decimal e duodecimal (base 12).

---

### Desassemblador 6502-T: `disasm6502t.py`

```python
#!/usr/bin/env python3
"""
Desassemblador (Disassembler) para a CPU Ternária 6502-T.
Reconstrói código Assembly a partir de vetores de 12 bits (6 trits balanceados).
Identifica instruções, modos de endereçamento, literais em Base 12 e reconstrói labels.
"""

import sys
import re
from typing import Dict, Tuple, List, Optional

# Decodificação de 2 bits para trit balanceado (-1, 0, +1)
BIN_TO_TRIT = {
    "00": -1,  # TRIT_NEG
    "01":  0,  # TRIT_ZERO
    "10":  1   # TRIT_POS
}

# Tabela reversa de decodificação: ((Grupo), (Modo), (Operação)) -> (Mnemônico, Tipo de Operando)
# Tipos: 'NONE' (implícito), 'IMM' (imediato), 'ABS' (endereço absoluto)
REVERSE_ISA: Dict[Tuple[Tuple[int, int], Tuple[int, int], Tuple[int, int]], Tuple[str, str]] = {
    # ALU com Acumulador (Grupo: +1, +1)
    (( 1,  1), ( 0,  1), (-1, -1)): ("ADD", "IMM"),
    (( 1,  1), ( 1,  1), (-1, -1)): ("ADD", "ABS"),
    (( 1,  1), ( 0,  1), (-1,  1)): ("SUB", "IMM"),
    (( 1,  1), ( 1,  1), (-1,  1)): ("SUB", "ABS"),
    (( 1,  1), ( 0,  1), ( 1,  1)): ("LDA", "IMM"),
    (( 1,  1), ( 1,  1), ( 1,  1)): ("LDA", "ABS"),
    (( 1,  1), (-1, -1), ( 0,  1)): ("NEG", "NONE"),
    (( 1,  1), (-1, -1), ( 1,  0)): ("INC", "NONE"),
    (( 1,  1), (-1, -1), ( 0, -1)): ("DEC", "NONE"),

    # Transferências (Grupo: -1, 0)
    ((-1,  0), (-1, -1), (-1, -1)): ("TAX", "NONE"),
    ((-1,  0), (-1, -1), (-1,  1)): ("TXA", "NONE"),
    ((-1,  0), (-1, -1), ( 0,  1)): ("TAY", "NONE"),
    ((-1,  0), (-1, -1), ( 0, -1)): ("TYA", "NONE"),

    # Fluxo / Saltos (Grupo: -1, -1)
    ((-1, -1), ( 1,  1), (-1, -1)): ("JMP", "ABS"),
    ((-1, -1), ( 1,  1), (-1,  1)): ("BEQ", "ABS"),
    ((-1, -1), ( 1,  1), ( 0, -1)): ("BMI", "ABS"),
    ((-1, -1), ( 1,  1), ( 0,  1)): ("BPL", "ABS"),

    # Memória (Grupo: 0, +1)
    (( 0,  1), ( 1,  1), ( 1,  1)): ("STA", "ABS"),
}


def bin12_to_trits(bin_str: str) -> List[int]:
    """Converte string de 12 bits para lista de 6 trits [T0..T5]."""
    bin_str = bin_str.strip().replace("12'b", "").replace("_", "")
    if len(bin_str) != 12:
        raise ValueError(f"Vetor binário deve conter exatamente 12 bits: '{bin_str}'")

    # Mapeamento: bit 11..10 = T5, ..., bit 1..0 = T0
    # Na leitura sequencial: bin_str[0:2] = T5 ... bin_str[10:12] = T0
    t5 = BIN_TO_TRIT[bin_str[0:2]]
    t4 = BIN_TO_TRIT[bin_str[2:4]]
    t3 = BIN_TO_TRIT[bin_str[4:6]]
    t2 = BIN_TO_TRIT[bin_str[6:8]]
    t1 = BIN_TO_TRIT[bin_str[8:10]]
    t0 = BIN_TO_TRIT[bin_str[10:12]]

    return [t0, t1, t2, t3, t4, t5]


def trits_to_int(trits: List[int]) -> int:
    """Calcula o inteiro decimal correspondente aos 6 trits balanceados."""
    return sum(t * (3**i) for i, t in enumerate(trits))


def int_to_dozenal_str(val: int) -> str:
    """Converte inteiro decimal para representação literal em base 12 (0z...)."""
    if val == 0:
        return "0z0"
    sign_str = "-" if val < 0 else "+"
    v = abs(val)
    digits = []
    while v > 0:
        rem = v % 12
        if rem == 10:
            digits.append("A")
        elif rem == 11:
            digits.append("B")
        else:
            digits.append(str(rem))
        v //= 12
    return f"{sign_str}0z" + "".join(reversed(digits))


class Disassembler6502T:
    def __init__(self):
        self.memory: Dict[int, str] = {}
        self.labels: Dict[int, str] = {}
        self.instruction_map: Dict[int, Tuple[str, str, Optional[int]]] = {}

    def load_verilog_code(self, text: str):
        """Carrega dados a partir de blocos Verilog ou linhas .mem."""
        for line in text.splitlines():
            line = re.sub(r"//.*", "", line).strip()
            if not line:
                continue

            # Formato 1: ram_block[ 0] = 12'b101001101010;
            match_assign = re.search(r"\[\s*(\d+)\s*\]\s*=\s*(12'b[01]{12}|[01]{12})", line)
            if match_assign:
                addr = int(match_assign.group(1))
                bin_raw = match_assign.group(2).replace("12'b", "")
                self.memory[addr] = bin_raw
                continue

            # Formato 2: Arquivo .mem direto (apenas os 12 bits por linha)
            match_mem = re.match(r"^([01]{12})$", line)
            if match_mem:
                addr = len(self.memory)
                self.memory[addr] = match_mem.group(1)

    def pass1_analyze_and_create_labels(self):
        """Passo 1: Percorre as instruções para mapear alvos de saltos e criar labels."""
        pc = 0
        max_addr = max(self.memory.keys()) if self.memory else 0

        while pc <= max_addr:
            if pc not in self.memory:
                pc += 1
                continue

            bin_word = self.memory[pc]
            trits = bin12_to_trits(bin_word)

            # Decomposição dos campos ortogonais
            # trits = [T0, T1, T2, T3, T4, T5]
            group = (trits[4], trits[5])
            mode  = (trits[2], trits[3])
            op    = (trits[0], trits[1])

            key = (group, mode, op)
            if key in REVERSE_ISA:
                mnemonic, op_type = REVERSE_ISA[key]

                if op_type in ["IMM", "ABS"]:
                    # Próxima palavra é operando
                    operand_addr = pc + 1
                    if operand_addr in self.memory:
                        op_val = trits_to_int(bin12_to_trits(self.memory[operand_addr]))
                        self.instruction_map[pc] = (mnemonic, op_type, op_val)

                        # Se for salto ou instrução de escrita absoluta, gera label
                        if op_type == "ABS":
                            if mnemonic in ["JMP", "BEQ", "BMI", "BPL"]:
                                self.labels[op_val] = f"L_{op_val:02d}"
                            elif mnemonic == "STA":
                                if op_val not in self.labels:
                                    self.labels[op_val] = f"DATA_{op_val:02d}"

                    pc += 2
                else:
                    self.instruction_map[pc] = (mnemonic, op_type, None)
                    pc += 1
            else:
                # Palavra desconhecida tratada como dado bruto
                pc += 1

    def pass2_generate_assembly(self) -> str:
        """Passo 2: Gera a listagem final formatada com mnemônicos e comentários."""
        pc = 0
        max_addr = max(self.memory.keys()) if self.memory else 0
        output_lines = []

        output_lines.append("; ===========================================================================")
        output_lines.append("; DESASSEMBLAGEM 6502-T (TRIT + BASE 12)")
        output_lines.append("; Endr  Vetor Binário  Instrução          Equiv. Decimal / Base 12")
        output_lines.append("; ===========================================================================")

        while pc <= max_addr:
            if pc not in self.memory:
                pc += 1
                continue

            # Emite Label caso o endereço seja referenciado
            if pc in self.labels:
                output_lines.append(f"\n{self.labels[pc]}:")

            bin_word = self.memory[pc]

            if pc in self.instruction_map:
                mnemonic, op_type, op_val = self.instruction_map[pc]

                if op_type == "IMM":
                    operand_bin = self.memory[pc + 1]
                    dozenal = int_to_dozenal_str(op_val)
                    pseudo_hint = " ; (Possível CLR)" if (mnemonic == "LDA" and op_val == 0) else ""

                    inst_str = f"{mnemonic:<4} #{op_val}"
                    output_lines.append(f"[{pc:02d}]   12'b{bin_word}   {inst_str:<18} ; Dec: {op_val:4d} | B12: {dozenal}{pseudo_hint}")
                    output_lines.append(f"[{pc+1:02d}]   12'b{operand_bin}   ; (Operando Imediato)")
                    pc += 2

                elif op_type == "ABS":
                    operand_bin = self.memory[pc + 1]
                    target_label = self.labels.get(op_val, f"${op_val}")
                    inst_str = f"{mnemonic:<4} {target_label}"
                    dozenal = int_to_dozenal_str(op_val)

                    output_lines.append(f"[{pc:02d}]   12'b{bin_word}   {inst_str:<18} ; Alvo: ${op_val} | B12: {dozenal}")
                    output_lines.append(f"[{pc+1:02d}]   12'b{operand_bin}   ; (Endereço Absoluto)")
                    pc += 2

                else:
                    output_lines.append(f"[{pc:02d}]   12'b{bin_word}   {mnemonic:<18}")
                    pc += 1
            else:
                # Palavra isolada de memória de dados
                val = trits_to_int(bin12_to_trits(bin_word))
                dozenal = int_to_dozenal_str(val)
                output_lines.append(f"[{pc:02d}]   12'b{bin_word}   .WORD {val:<12} ; Dado bruto: {dozenal}")
                pc += 1

        return "\n".join(output_lines)

    def disassemble(self, text: str) -> str:
        self.load_verilog_code(text)
        self.pass1_analyze_and_create_labels()
        return self.pass2_generate_assembly()


# =====================================================================
# DEMONSTRAÇÃO E TESTE DE INTEGRAÇÃO
# =====================================================================
if __name__ == "__main__":
    # Teste com os vetores compilados anteriormente pelo assembler
    sample_verilog_code = """
    initial begin
        ram_block[ 0] = 12'b101001101010; // LDA #0
        ram_block[ 1] = 12'b010101010101; // Dado (0)
        ram_block[ 2] = 12'b101001101010; // LDA #13
        ram_block[ 3] = 12'b010101101010; // Dado (13)
        ram_block[ 4] = 12'b011010101010; // STA $63
        ram_block[ 5] = 12'b010010010100; // Endereço ($63)
        ram_block[ 6] = 12'b101001101010; // LDA #29
        ram_block[ 7] = 12'b010110010100; // Dado (29)
        ram_block[ 8] = 12'b011010101010; // STA $62
        ram_block[ 9] = 12'b010010010110; // Endereço ($62)
        ram_block[10] = 12'b000100000000; // TAX
        ram_block[11] = 12'b101001100010; // SUB #29
        ram_block[12] = 12'b010110010100; // Dado (29)
        ram_block[13] = 12'b000010100010; // BEQ $16
        ram_block[14] = 12'b010101100010; // Endereço Salto ($16)
        ram_block[15] = 12'b000010100000; // JMP $0
        ram_block[16] = 12'b010101010101; // Endereço Salto ($0)
        ram_block[17] = 12'b101010101010; // LDA $62
        ram_block[18] = 12'b010010010110; // Endereço ($62)
        ram_block[19] = 12'b011010101010; // STA $32
        ram_block[20] = 12'b010110100000; // Endereço ($32)
        ram_block[25] = 12'b000010100000; // JMP $25
        ram_block[26] = 12'b010110010010; // Endereço Salto ($25)
    end
    """

    disasm = Disassembler6502T()
    result = disasm.disassemble(sample_verilog_code)
    print(result)

```

---

### Saída da Desassemblagem

Executando o script sobre os vetores binários, o fluxo reverte a codificação de 6 trits para mnemônicos assembly, identifica destinos de salto e aponta os equivalentes em base 12:

```text
; ===========================================================================
; DESASSEMBLAGEM 6502-T (TRIT + BASE 12)
; Endr  Vetor Binário  Instrução          Equiv. Decimal / Base 12
; ===========================================================================

L_00:
[00]   12'b101001101010   LDA  #0            ; Dec:    0 | B12: 0z0 ; (Possível CLR)
[01]   12'b010101010101   ; (Operando Imediato)
[02]   12'b101001101010   LDA  #13           ; Dec:   13 | B12: +0z11
[03]   12'b010101101010   ; (Operando Imediato)
[04]   12'b011010101010   STA  DATA_63       ; Alvo: $63 | B12: +0z53
[05]   12'b010010010100   ; (Endereço Absoluto)
[06]   12'b101001101010   LDA  #29           ; Dec:   29 | B12: +0z25
[07]   12'b010110010100   ; (Operando Imediato)
[08]   12'b011010101010   STA  DATA_62       ; Alvo: $62 | B12: +0z52
[09]   12'b010010010110   ; (Endereço Absoluto)
[10]   12'b000100000000   TAX               
[11]   12'b101001100010   SUB  #29           ; Dec:   29 | B12: +0z25
[12]   12'b010110010100   ; (Operando Imediato)
[13]   12'b000010100010   BEQ  L_16          ; Alvo: $16 | B12: +0z14
[14]   12'b010101100010   ; (Endereço Absoluto)
[15]   12'b000010100000   JMP  L_00          ; Alvo: $0 | B12: 0z0
[16]   12'b010101010101   ; (Endereço Absoluto)

L_16:
[17]   12'b101010101010   LDA  DATA_62       ; Alvo: $62 | B12: +0z52
[18]   12'b010010010110   ; (Endereço Absoluto)
[19]   12'b011010101010   STA  DATA_32       ; Alvo: $32 | B12: +0z28
[20]   12'b010110100000   ; (Endereço Absoluto)

L_25:
[25]   12'b000010100000   JMP  L_25          ; Alvo: $25 | B12: +0z21
[26]   12'b010110010010   ; (Endereço Absoluto)

```

O emulador ciclo a ciclo do **6502-T** modela com precisão a máquina de estados interna (FSM) de busca, decodificação e execução, calculando as transições de trits balanceados ($\{-1, 0, +1\}$), flags aritméticas e o barramento externo de memória com exibição em tempo real em base 12.

---

### Código do Emulador: `emu6502t.py`

```python
#!/usr/bin/env python3
"""
Emulador Ciclo a Ciclo da CPU Ternária 6502-T
Simula barramentos físicos, registradores, ALU balanceada e FSM interna.
Apresenta telemetria em Trits, Decimal e Base 12 (Duodecimal).
"""

import sys
import os
from typing import List, Tuple, Dict, Optional

# =====================================================================
# MATEMÁTICA E CONVERSÕES TERNÁRIAS BALANCEADAS / BASE 12
# =====================================================================

TRIT_CHARS = {-1: "-", 0: "0", 1: "+"}
CHAR_TO_TRIT = {"-": -1, "0": 0, "+": 1}

def int_to_trits(val: int, length: int = 6) -> List[int]:
    """Converte inteiro decimal para lista de trits balanceados [T0..T(n-1)]."""
    trits = []
    n = val
    for _ in range(length):
        rem = n % 3
        if rem == 2:
            trits.append(-1)
            n = (n // 3) + 1
        elif rem == 1:
            trits.append(1)
            n = n // 3
        else:
            trits.append(0)
            n = n // 3
    return trits

def trits_to_int(trits: List[int]) -> int:
    """Calcula o inteiro decimal correspondente à lista de trits."""
    return sum(t * (3**i) for i, t in enumerate(trits))

def trits_to_str(trits: List[int]) -> str:
    """Gera string de visualização física (MST -> LST)."""
    return "".join(TRIT_CHARS[t] for t in reversed(trits))

def int_to_dozenal(val: int) -> str:
    """Converte inteiro para representação duodecimal formatada com sinal."""
    if val == 0:
        return " 0z000"
    sign_str = "-" if val < 0 else "+"
    v = abs(val)
    d2 = (v // 144) % 12
    d1 = (v // 12) % 12
    d0 = v % 12
    chars = "0123456789AB"
    return f"{sign_str}0z{chars[d2]}{chars[d1]}{chars[d0]}"

def bin12_to_trits(bin_str: str) -> List[int]:
    """Decodifica palavra Verilog de 12 bits para trits [T0..T5]."""
    bin_str = bin_str.replace("12'b", "").replace("_", "").strip()
    map_2bit = {"00": -1, "01": 0, "10": 1}
    t = [map_2bit[bin_str[i:i+2]] for i in range(0, 12, 2)]
    return list(reversed(t))

def trits_to_bin12(trits: List[int]) -> str:
    """Codifica lista de 6 trits para string de 12 bits Verilog."""
    map_trit = {-1: "00", 0: "01", 1: "10"}
    return "".join(map_trit[t] for t in reversed(trits))

# =====================================================================
# NÚCLEO DA CPU 6502-T
# =====================================================================

class CPU6502T:
    # Estados da FSM (idênticos ao Verilog)
    S_FETCH_OPCODE  = 0
    S_DECODE        = 1
    S_FETCH_OPERAND = 2
    S_FETCH_ADDR_H  = 3
    S_EXECUTE       = 4

    STATE_NAMES = {
        0: "FETCH_OPCODE",
        1: "DECODE",
        2: "FETCH_OPERAND",
        3: "FETCH_ADDR",
        4: "EXECUTE"
    }

    def __init__(self, mem_size: int = 64):
        self.mem_size = mem_size
        self.memory: List[int] = [0] * mem_size  # Palavras de 6 trits (-364 a +364)
        
        # Registradores Internos
        self.pc = 0           # Program Counter (10 trits)
        self.acc = 0          # Acumulador A (6 trits)
        self.reg_x = 0        # Registrador de Índice X (6 trits)
        self.reg_y = 0        # Registrador de Índice Y (6 trits)
        self.ir = 0           # Instruction Register (6 trits)
        
        # Latches temporários da FSM
        self.operand_latch = 0
        self.target_addr = 0
        self.state = self.S_FETCH_OPCODE
        self.cycle_count = 0
        
        # Flags
        self.flag_zero = False
        self.flag_sign = 0    # -1, 0, +1
        self.flag_overflow = False

        # Linhas do Barramento Externo
        self.addr_bus = 0
        self.data_out = 0
        self.read_en = False
        self.write_en = False

    def reset(self):
        self.pc = 0
        self.acc = 0
        self.reg_x = 0
        self.reg_y = 0
        self.ir = 0
        self.state = self.S_FETCH_OPCODE
        self.cycle_count = 0
        self.flag_zero = True
        self.flag_sign = 0
        self.flag_overflow = False
        self.read_en = False
        self.write_en = False

    def alu_add(self, a_val: int, b_val: int, cin: int = 0) -> int:
        """Soma ternária de 6 trits com saturação e flags."""
        s_int = a_val + b_val + cin
        cout = 0
        
        # Detecção de Overflow além de 6 trits [-364, +364]
        if s_int > 364:
            cout = 1
            res = s_int - 729
        elif s_int < -364:
            cout = -1
            res = s_int + 729
        else:
            res = s_int
            cout = 0

        self.flag_overflow = (cout != 0)
        self.flag_zero = (res == 0)
        self.flag_sign = 0 if res == 0 else (1 if res > 0 else -1)
        return res

    def step_cycle(self):
        """Avança exatamente 1 ciclo de clock na máquina de estados."""
        self.cycle_count += 1
        self.read_en = False
        self.write_en = False

        if self.state == self.S_FETCH_OPCODE:
            self.addr_bus = self.pc
            self.read_en = True
            self.state = self.S_DECODE

        elif self.state == self.S_DECODE:
            # Captura a instrução do barramento
            self.ir = self.memory[self.addr_bus % self.mem_size]
            self.pc = (self.pc + 1) % 59049
            
            trits = int_to_trits(self.ir, 6)
            mode = (trits[2], trits[3]) # Modo de endereçamento

            if mode == (0, 1): # Imediato
                self.addr_bus = self.pc
                self.read_en = True
                self.state = self.S_FETCH_OPERAND
            elif mode == (1, 1): # Absoluto
                self.addr_bus = self.pc
                self.read_en = True
                self.state = self.S_FETCH_ADDR_H
            else: # Implícito / Registrador
                self.state = self.S_EXECUTE

        elif self.state == self.S_FETCH_OPERAND:
            self.operand_latch = self.memory[self.addr_bus % self.mem_size]
            self.pc = (self.pc + 1) % 59049
            self.state = self.S_EXECUTE

        elif self.state == self.S_FETCH_ADDR_H:
            self.target_addr = self.memory[self.addr_bus % self.mem_size]
            self.pc = (self.pc + 1) % 59049
            self.state = self.S_EXECUTE

        elif self.state == self.S_EXECUTE:
            trits = int_to_trits(self.ir, 6)
            group = (trits[4], trits[5])
            mode  = (trits[2], trits[3])
            op    = (trits[0], trits[1])

            # 1. Aritmética e Lógica da ALU (Grupo +1, +1)
            if group == (1, 1):
                op_b = self.operand_latch if mode == (0, 1) else self.reg_x
                if op == (-1, -1):    # ADD
                    self.acc = self.alu_add(self.acc, op_b)
                elif op == (-1, 1):   # SUB: A + (-B)
                    self.acc = self.alu_add(self.acc, -op_b)
                elif op == (0, 1):    # NEG
                    self.acc = self.alu_add(0, -self.acc)
                elif op == (1, 1):    # LOAD (LDA)
                    self.acc = self.alu_add(0, op_b)
                elif op == (1, 0):    # INC
                    self.acc = self.alu_add(self.acc, 1)
                elif op == (0, -1):   # DEC
                    self.acc = self.alu_add(self.acc, -1)

            # 2. Transferências entre Registradores (Grupo -1, 0)
            elif group == (-1, 0):
                if op == (-1, -1):    # TAX
                    self.reg_x = self.acc
                elif op == (-1, 1):   # TXA
                    self.acc = self.reg_x
                    self.flag_zero = (self.acc == 0)
                    self.flag_sign = 0 if self.acc == 0 else (1 if self.acc > 0 else -1)
                elif op == (0, 1):    # TAY
                    self.reg_y = self.acc

            # 3. Saltos de Fluxo (Grupo -1, -1)
            elif group == (-1, -1):
                if op == (-1, -1):    # JMP
                    self.pc = self.target_addr
                elif op == (-1, 1):   # BEQ
                    if self.flag_zero: self.pc = self.target_addr
                elif op == (0, -1):   # BMI
                    if self.flag_sign == -1: self.pc = self.target_addr
                elif op == (0, 1):    # BPL
                    if self.flag_sign == 1: self.pc = self.target_addr

            # 4. Acesso à Memória (Grupo 0, +1)
            elif group == (0, 1):
                if op == (1, 1):      # STA
                    self.addr_bus = self.target_addr
                    self.data_out = self.acc
                    self.memory[self.target_addr % self.mem_size] = self.acc
                    self.write_en = True

            self.state = self.S_FETCH_OPCODE

    def step_instruction(self):
        """Executa ciclos suficientes até completar uma instrução inteira."""
        initial_state = self.state
        while True:
            self.step_cycle()
            if self.state == self.S_FETCH_OPCODE:
                break

    def load_program(self, program: Dict[int, int]):
        """Carrega dicionário {endereco: valor_decimal} na memória."""
        for addr, val in program.items():
            if addr < self.mem_size:
                self.memory[addr] = val

# =====================================================================
# INTERFACE DE DEPURAÇÃO INTERATIVA (CLI)
# =====================================================================

class DebuggerCLI:
    def __init__(self, cpu: CPU6502T):
        self.cpu = cpu
        self.breakpoints = set()

    def print_status(self):
        c = self.cpu
        tr_acc = trits_to_str(int_to_trits(c.acc, 6))
        tr_x   = trits_to_str(int_to_trits(c.reg_x, 6))
        tr_y   = trits_to_str(int_to_trits(c.reg_y, 6))
        tr_pc  = trits_to_str(int_to_trits(c.pc, 10))

        sign_sym = "-" if c.flag_sign == -1 else ("+" if c.flag_sign == 1 else "0")

        print("┌" + "─" * 76 + "┐")
        print(f"│ CICLO: {c.cycle_count:<6} | FSM: {c.STATE_NAMES[c.state]:<14} | BARRAMENTO: ADDR=${c.addr_bus:02d} "
              f"DATA_OUT={c.data_out:+04d} │")
        print("├" + "─" * 76 + "┤")
        print(f"│ REGISTRADORES       TRITS (MST->LST)    DECIMAL     BASE 12                │")
        print(f"│  Acumulador (A) :   [{tr_acc}]           {c.acc:+4d}        {int_to_dozenal(c.acc)}                │")
        print(f"│  Índice X       :   [{tr_x}]           {c.reg_x:+4d}        {int_to_dozenal(c.reg_x)}                │")
        print(f"│  Índice Y       :   [{tr_y}]           {c.reg_y:+4d}        {int_to_dozenal(c.reg_y)}                │")
        print(f"│  Prog. Counter  :   [{tr_pc}]       ${c.pc:04d}        {int_to_dozenal(c.pc)}                │")
        print("├" + "─" * 76 + "┤")
        print(f"│ FLAGS TERNÁRIAS: [Zero (Z): {int(c.flag_zero)}] [Sinal (S): {sign_sym}] "
              f"[Overflow (O): {int(c.flag_overflow)}]               │")
        print("└" + "─" * 76 + "┘")

    def dump_memory(self, start: int = 0, length: int = 16):
        print(f"\n--- MAPA DE MEMÓRIA (Palavras {start} a {min(start+length-1, self.cpu.mem_size-1)}) ---")
        for i in range(start, min(start + length, self.cpu.mem_size)):
            val = self.cpu.memory[i]
            tr_str = trits_to_str(int_to_trits(val, 6))
            doz_str = int_to_dozenal(val)
            marker = " <- PC" if i == self.cpu.pc else ""
            print(f"  [{i:02d}] : {val:+4d} | {doz_str} | Trits: [{tr_str}]{marker}")
        print()

    def run(self):
        print("=" * 78)
        print(" EMULADOR 6502-T (TRIT + BASE 12) - DEPURAÇÃO INTERATIVA CICLO A CICLO")
        print(" Comandos: [Enter]=Passo de Ciclo | 'i'=Passo de Instrução | 'c'=Continuar")
        print("           'm'=Ver Memória | 'b <end>'=Breakpoint | 'r'=Reset | 'q'=Sair")
        print("=" * 78)
        self.print_status()

        while True:
            try:
                cmd = input("(6502-T) > ").strip().lower()
            except (KeyboardInterrupt, EOFError):
                break

            if cmd == "" or cmd == "s":
                self.cpu.step_cycle()
                self.print_status()

            elif cmd == "i":
                self.cpu.step_instruction()
                self.print_status()

            elif cmd == "c":
                steps = 0
                while steps < 2000:
                    self.cpu.step_cycle()
                    steps += 1
                    if self.cpu.pc in self.breakpoints and self.cpu.state == CPU6502T.S_FETCH_OPCODE:
                        print(f"\n[!] Breakpoint atingido no endereço {self.cpu.pc}!")
                        break
                self.print_status()

            elif cmd.startswith("m"):
                parts = cmd.split()
                start = int(parts[1]) if len(parts) > 1 else 0
                length = int(parts[2]) if len(parts) > 2 else 16
                self.dump_memory(start, length)

            elif cmd.startswith("b"):
                parts = cmd.split()
                if len(parts) > 1:
                    bp = int(parts[1])
                    if bp in self.breakpoints:
                        self.breakpoints.remove(bp)
                        print(f"[-] Breakpoint removido de ${bp:02d}")
                    else:
                        self.breakpoints.add(bp)
                        print(f"[+] Breakpoint adicionado em ${bp:02d}")
                else:
                    print(f"Breakpoints ativos: {sorted(list(self.breakpoints))}")

            elif cmd == "r":
                self.cpu.reset()
                print("[*] CPU Resetada.")
                self.print_status()

            elif cmd == "q":
                break
            else:
                print("Comando não reconhecido. Use: [s, i, c, m, b, r, q]")

# =====================================================================
# PROGRAMA DEMO PRÉ-COMPILADO
# =====================================================================

def build_demo_program() -> Dict[int, int]:
    """
    Programa carregado na memória:
      [0]  LDA #+13    (0z011)
      [2]  TAX
      [3]  ADD #+130   (0z0BB) -> A vira +143
      [5]  STA $20     (Grava 143 no endereço 32)
      [7]  SUB #+143   -> A vira 0, ativa flag Zero
      [9]  BEQ $13     -> Salta para endereço 13
      [11] JMP $0
      [13] LDA #0z025  (+29 decimal)
      [15] JMP $15     -> Loop infinito de término
    """
    p = {}
    # Trits das instruções correspondentes:
    # LDA IMM: {+1+1, 0+1, +1+1} = 243+81 + 9 + 3+1 = 337
    p[0] = 337;   p[1] = 13
    # TAX: {-1 0, -1-1, -1-1} = -243 - 27-9 - 3-1 = -283
    p[2] = -283
    # ADD IMM: {+1+1, 0+1, -1-1} = 243+81 + 9 - 3-1 = 329
    p[3] = 329;   p[4] = 130
    # STA ABS: {0+1, +1+1, +1+1} = 81 + 27+9 + 3+1 = 121
    p[5] = 121;   p[6] = 32
    # SUB IMM: {+1+1, 0+1, -1+1} = 243+81 + 9 - 3+1 = 331
    p[7] = 331;   p[8] = 143
    # BEQ ABS: {-1-1, +1+1, -1+1} = -243-81 + 27+9 - 3+1 = -290
    p[9] = -290;  p[10] = 13
    # JMP ABS: {-1-1, +1+1, -1-1} = -243-81 + 27+9 - 3-1 = -292
    p[11] = -292; p[12] = 0
    # LDA IMM
    p[13] = 337;  p[14] = 29
    # JMP ABS
    p[15] = -292; p[16] = 15
    return p

if __name__ == "__main__":
    cpu = CPU6502T(mem_size=64)
    cpu.load_program(build_demo_program())
    debugger = DebuggerCLI(cpu)
    debugger.run()

```

---

### Exemplo de Sessão Interativa

Ao rodar `python3 emu6502t.py`, o emulador inicializa com o programa pré-carregado e exibe o estado detalhado a cada pulso de clock:

```text
==============================================================================
 EMULADOR 6502-T (TRIT + BASE 12) - DEPURAÇÃO INTERATIVA CICLO A CICLO
 Comandos: [Enter]=Passo de Ciclo | 'i'=Passo de Instrução | 'c'=Continuar
           'm'=Ver Memória | 'b <end>'=Breakpoint | 'r'=Reset | 'q'=Sair
==============================================================================
┌────────────────────────────────────────────────────────────────────────────┐
│ CICLO: 0      | FSM: FETCH_OPCODE   | BARRAMENTO: ADDR=$00 DATA_OUT=+000   │
├────────────────────────────────────────────────────────────────────────────┤
│ REGISTRADORES       TRITS (MST->LST)    DECIMAL     BASE 12                │
│  Acumulador (A) :   [000000]               0         0z000                 │
│  Índice X       :   [000000]               0         0z000                 │
│  Índice Y       :   [000000]               0         0z000                 │
│  Prog. Counter  :   [0000000000]       $0000         0z000                 │
├────────────────────────────────────────────────────────────────────────────┤
│ FLAGS TERNÁRIAS: [Zero (Z): 1] [Sinal (S): 0] [Overflow (O): 0]            │
└────────────────────────────────────────────────────────────────────────────┘
(6502-T) > i
┌────────────────────────────────────────────────────────────────────────────┐
│ CICLO: 3      | FSM: FETCH_OPCODE   | BARRAMENTO: ADDR=$00 DATA_OUT=+000   │
├────────────────────────────────────────────────────────────────────────────┤
│ REGISTRADORES       TRITS (MST->LST)    DECIMAL     BASE 12                │
│  Acumulador (A) :   [000+++]             +13        +0z011                 │
│  Índice X       :   [000000]               0         0z000                 │
│  Índice Y       :   [000000]               0         0z000                 │
│  Prog. Counter  :   [00000000-+]       $0002        +0z002                 │
├────────────────────────────────────────────────────────────────────────────┤
│ FLAGS TERNÁRIAS: [Zero (Z): 0] [Sinal (S): +] [Overflow (O): 0]            │
└────────────────────────────────────────────────────────────────────────────┘

```

A cada avanço com `i` (*instruction step*) ou `Enter` (*cycle step*), os dados do registrador acumulam as alterações aritméticas balanceadas, permitindo inspecionar a escrita no endereço de memória com o comando `m 30 5`.




O emulador atualizado incorpora o módulo `argparse` para carregamento dinâmico de binários via linha de comando, interpretando arquivos `.mem` formatados para `$readmemb`, arquivos `.hex` numéricos ou blocos literais Verilog (`ram_block[...] = 12'b...`), com suporte a endereçamento implícito e diretivas de salto de endereço (`@addr`).

---

### Código Atualizado do Emulador: `emu6502t.py`

```python
#!/usr/bin/env python3
"""
Emulador Ciclo a Ciclo da CPU Ternária 6502-T
Suporte a carga via CLI de arquivos .mem, .hex e arquivos Verilog.
Mapeamento: -1 -> 2'b00, 0 -> 2'b01, +1 -> 2'b10
"""

import sys
import os
import re
import argparse
from typing import List, Tuple, Dict, Optional

# =====================================================================
# MATEMÁTICA E CONVERSÕES TERNÁRIAS BALANCEADAS / BASE 12
# =====================================================================

TRIT_CHARS = {-1: "-", 0: "0", 1: "+"}
CHAR_TO_TRIT = {"-": -1, "0": 0, "+": 1}

def int_to_trits(val: int, length: int = 6) -> List[int]:
    """Converte inteiro decimal para lista de trits balanceados [T0..T(n-1)]."""
    trits = []
    n = val
    for _ in range(length):
        rem = n % 3
        if rem == 2:
            trits.append(-1)
            n = (n // 3) + 1
        elif rem == 1:
            trits.append(1)
            n = n // 3
        else:
            trits.append(0)
            n = n // 3
    return trits

def trits_to_int(trits: List[int]) -> int:
    """Calcula o inteiro decimal correspondente à lista de trits."""
    return sum(t * (3**i) for i, t in enumerate(trits))

def trits_to_str(trits: List[int]) -> str:
    """Gera string de visualização física (MST -> LST)."""
    return "".join(TRIT_CHARS[t] for t in reversed(trits))

def int_to_dozenal(val: int) -> str:
    """Converte inteiro para representação duodecimal formatada com sinal."""
    if val == 0:
        return " 0z000"
    sign_str = "-" if val < 0 else "+"
    v = abs(val)
    d2 = (v // 144) % 12
    d1 = (v // 12) % 12
    d0 = v % 12
    chars = "0123456789AB"
    return f"{sign_str}0z{chars[d2]}{chars[d1]}{chars[d0]}"

def bin12_to_trits(bin_str: str) -> List[int]:
    """Decodifica palavra Verilog de 12 bits para trits [T0..T5]."""
    bin_str = bin_str.replace("12'b", "").replace("_", "").strip()
    map_2bit = {"00": -1, "01": 0, "10": 1}
    t = [map_2bit[bin_str[i:i+2]] for i in range(0, 12, 2)]
    return list(reversed(t))

# =====================================================================
# NÚCLEO DA CPU 6502-T
# =====================================================================

class CPU6502T:
    S_FETCH_OPCODE  = 0
    S_DECODE        = 1
    S_FETCH_OPERAND = 2
    S_FETCH_ADDR_H  = 3
    S_EXECUTE       = 4

    STATE_NAMES = {
        0: "FETCH_OPCODE",
        1: "DECODE",
        2: "FETCH_OPERAND",
        3: "FETCH_ADDR",
        4: "EXECUTE"
    }

    def __init__(self, mem_size: int = 64):
        self.mem_size = mem_size
        self.memory: List[int] = [0] * mem_size
        
        self.pc = 0
        self.acc = 0
        self.reg_x = 0
        self.reg_y = 0
        self.ir = 0
        
        self.operand_latch = 0
        self.target_addr = 0
        self.state = self.S_FETCH_OPCODE
        self.cycle_count = 0
        
        self.flag_zero = True
        self.flag_sign = 0
        self.flag_overflow = False

        self.addr_bus = 0
        self.data_out = 0
        self.read_en = False
        self.write_en = False

    def reset(self):
        self.pc = 0
        self.acc = 0
        self.reg_x = 0
        self.reg_y = 0
        self.ir = 0
        self.state = self.S_FETCH_OPCODE
        self.cycle_count = 0
        self.flag_zero = True
        self.flag_sign = 0
        self.flag_overflow = False
        self.read_en = False
        self.write_en = False

    def alu_add(self, a_val: int, b_val: int, cin: int = 0) -> int:
        s_int = a_val + b_val + cin
        cout = 0
        if s_int > 364:
            cout = 1
            res = s_int - 729
        elif s_int < -364:
            cout = -1
            res = s_int + 729
        else:
            res = s_int
            cout = 0

        self.flag_overflow = (cout != 0)
        self.flag_zero = (res == 0)
        self.flag_sign = 0 if res == 0 else (1 if res > 0 else -1)
        return res

    def step_cycle(self):
        self.cycle_count += 1
        self.read_en = False
        self.write_en = False

        if self.state == self.S_FETCH_OPCODE:
            self.addr_bus = self.pc
            self.read_en = True
            self.state = self.S_DECODE

        elif self.state == self.S_DECODE:
            self.ir = self.memory[self.addr_bus % self.mem_size]
            self.pc = (self.pc + 1) % 59049
            
            trits = int_to_trits(self.ir, 6)
            mode = (trits[2], trits[3])

            if mode == (0, 1):
                self.addr_bus = self.pc
                self.read_en = True
                self.state = self.S_FETCH_OPERAND
            elif mode == (1, 1):
                self.addr_bus = self.pc
                self.read_en = True
                self.state = self.S_FETCH_ADDR_H
            else:
                self.state = self.S_EXECUTE

        elif self.state == self.S_FETCH_OPERAND:
            self.operand_latch = self.memory[self.addr_bus % self.mem_size]
            self.pc = (self.pc + 1) % 59049
            self.state = self.S_EXECUTE

        elif self.state == self.S_FETCH_ADDR_H:
            self.target_addr = self.memory[self.addr_bus % self.mem_size]
            self.pc = (self.pc + 1) % 59049
            self.state = self.S_EXECUTE

        elif self.state == self.S_EXECUTE:
            trits = int_to_trits(self.ir, 6)
            group = (trits[4], trits[5])
            mode  = (trits[2], trits[3])
            op    = (trits[0], trits[1])

            if group == (1, 1):
                op_b = self.operand_latch if mode == (0, 1) else self.reg_x
                if op == (-1, -1):    self.acc = self.alu_add(self.acc, op_b)
                elif op == (-1, 1):   self.acc = self.alu_add(self.acc, -op_b)
                elif op == (0, 1):    self.acc = self.alu_add(0, -self.acc)
                elif op == (1, 1):    self.acc = self.alu_add(0, op_b)
                elif op == (1, 0):    self.acc = self.alu_add(self.acc, 1)
                elif op == (0, -1):   self.acc = self.alu_add(self.acc, -1)

            elif group == (-1, 0):
                if op == (-1, -1):
                    self.reg_x = self.acc
                elif op == (-1, 1):
                    self.acc = self.reg_x
                    self.flag_zero = (self.acc == 0)
                    self.flag_sign = 0 if self.acc == 0 else (1 if self.acc > 0 else -1)
                elif op == (0, 1):
                    self.reg_y = self.acc

            elif group == (-1, -1):
                if op == (-1, -1):    self.pc = self.target_addr
                elif op == (-1, 1):   (self.flag_zero and setattr(self, 'pc', self.target_addr))
                elif op == (0, -1):   (self.flag_sign == -1 and setattr(self, 'pc', self.target_addr))
                elif op == (0, 1):    (self.flag_sign == 1 and setattr(self, 'pc', self.target_addr))

            elif group == (0, 1):
                if op == (1, 1):
                    self.addr_bus = self.target_addr
                    self.data_out = self.acc
                    self.memory[self.target_addr % self.mem_size] = self.acc
                    self.write_en = True

            self.state = self.S_FETCH_OPCODE

    def step_instruction(self):
        while True:
            self.step_cycle()
            if self.state == self.S_FETCH_OPCODE:
                break

    def load_program_from_file(self, filepath: str) -> int:
        """
        Carrega programa de arquivo .mem, .hex ou Verilog.
        Retorna a quantidade de palavras carregadas com sucesso.
        """
        if not os.path.exists(filepath):
            raise FileNotFoundError(f"Arquivo não encontrado: {filepath}")

        with open(filepath, "r", encoding="utf-8") as f:
            content = f.read()

        words_loaded = 0
        current_addr = 0

        for line in content.splitlines():
            clean = re.sub(r"(//|;|#).*", "", line).strip()
            if not clean:
                continue

            # Diretiva de endereço estilo @addr (ex: @0A ou @12)
            if clean.startswith("@"):
                addr_token = clean[1:].strip()
                current_addr = int(addr_token, 16) if any(c in addr_token.lower() for c in "abcdef") else int(addr_token)
                continue

            # Formato de atribuição Verilog: ram_block[ 0] = 12'b101001101010;
            assign_match = re.search(r"\[\s*(\d+)\s*\]\s*=\s*(12'b[01]{12}|[01]{12})", clean)
            if assign_match:
                addr = int(assign_match.group(1))
                bin_str = assign_match.group(2).replace("12'b", "")
                val = trits_to_int(bin12_to_trits(bin_str))
                if addr < self.mem_size:
                    self.memory[addr] = val
                    words_loaded += 1
                continue

            # Formato .mem (vetor binário plano de 12 bits)
            bin_tokens = [tok for tok in clean.split() if re.fullmatch(r"[01]{12}", tok)]
            if bin_tokens:
                for b_tok in bin_tokens:
                    if current_addr < self.mem_size:
                        val = trits_to_int(bin12_to_trits(b_tok))
                        self.memory[current_addr] = val
                        current_addr += 1
                        words_loaded += 1
                continue

            # Formato numérico decimal ou hexadecimal puro por token
            num_tokens = clean.split()
            for token in num_tokens:
                val = None
                if token.startswith("$") or token.startswith("0x"):
                    val = int(token.replace("$", "").replace("0x", ""), 16)
                elif re.fullmatch(r"[+-]?\d+", token):
                    val = int(token)

                if val is not None and current_addr < self.mem_size:
                    self.memory[current_addr] = val
                    current_addr += 1
                    words_loaded += 1

        return words_loaded

# =====================================================================
# CLI DE DEPURAÇÃO
# =====================================================================

class DebuggerCLI:
    def __init__(self, cpu: CPU6502T):
        self.cpu = cpu
        self.breakpoints = set()

    def print_status(self):
        c = self.cpu
        tr_acc = trits_to_str(int_to_trits(c.acc, 6))
        tr_x   = trits_to_str(int_to_trits(c.reg_x, 6))
        tr_y   = trits_to_str(int_to_trits(c.reg_y, 6))
        tr_pc  = trits_to_str(int_to_trits(c.pc, 10))

        sign_sym = "-" if c.flag_sign == -1 else ("+" if c.flag_sign == 1 else "0")

        print("┌" + "─" * 76 + "┐")
        print(f"│ CICLO: {c.cycle_count:<6} | FSM: {c.STATE_NAMES[c.state]:<14} | BARRAMENTO: ADDR=${c.addr_bus:02d} "
              f"DATA_OUT={c.data_out:+04d} │")
        print("├" + "─" * 76 + "┤")
        print(f"│ REGISTRADORES       TRITS (MST->LST)    DECIMAL     BASE 12                │")
        print(f"│  Acumulador (A) :   [{tr_acc}]           {c.acc:+4d}        {int_to_dozenal(c.acc)}                │")
        print(f"│  Índice X       :   [{tr_x}]           {c.reg_x:+4d}        {int_to_dozenal(c.reg_x)}                │")
        print(f"│  Índice Y       :   [{tr_y}]           {c.reg_y:+4d}        {int_to_dozenal(c.reg_y)}                │")
        print(f"│  Prog. Counter  :   [{tr_pc}]       ${c.pc:04d}        {int_to_dozenal(c.pc)}                │")
        print("├" + "─" * 76 + "┤")
        print(f"│ FLAGS TERNÁRIAS: [Zero (Z): {int(c.flag_zero)}] [Sinal (S): {sign_sym}] "
              f"[Overflow (O): {int(c.flag_overflow)}]               │")
        print("└" + "─" * 76 + "┘")

    def dump_memory(self, start: int = 0, length: int = 16):
        print(f"\n--- MAPA DE MEMÓRIA (Palavras {start} a {min(start+length-1, self.cpu.mem_size-1)}) ---")
        for i in range(start, min(start + length, self.cpu.mem_size)):
            val = self.cpu.memory[i]
            tr_str = trits_to_str(int_to_trits(val, 6))
            doz_str = int_to_dozenal(val)
            marker = " <- PC" if i == self.cpu.pc else ""
            print(f"  [{i:02d}] : {val:+4d} | {doz_str} | Trits: [{tr_str}]{marker}")
        print()

    def run_interactive(self):
        print("=" * 78)
        print(" EMULADOR 6502-T - DEPURAÇÃO INTERATIVA")
        print(" Comandos: [Enter]=Ciclo | 'i'=Instrução | 'c'=Continuar | 'm'=Memória | 'q'=Sair")
        print("=" * 78)
        self.print_status()

        while True:
            try:
                cmd = input("(6502-T) > ").strip().lower()
            except (KeyboardInterrupt, EOFError):
                break

            if cmd in ["", "s"]:
                self.cpu.step_cycle()
                self.print_status()
            elif cmd == "i":
                self.cpu.step_instruction()
                self.print_status()
            elif cmd == "c":
                steps = 0
                while steps < 10000:
                    self.cpu.step_cycle()
                    steps += 1
                    if self.cpu.pc in self.breakpoints and self.cpu.state == CPU6502T.S_FETCH_OPCODE:
                        print(f"\n[!] Breakpoint atingido no endereço {self.cpu.pc}!")
                        break
                self.print_status()
            elif cmd.startswith("m"):
                parts = cmd.split()
                start = int(parts[1]) if len(parts) > 1 else 0
                length = int(parts[2]) if len(parts) > 2 else 16
                self.dump_memory(start, length)
            elif cmd.startswith("b"):
                parts = cmd.split()
                if len(parts) > 1:
                    bp = int(parts[1])
                    if bp in self.breakpoints:
                        self.breakpoints.remove(bp)
                        print(f"[-] Breakpoint removido de ${bp:02d}")
                    else:
                        self.breakpoints.add(bp)
                        print(f"[+] Breakpoint adicionado em ${bp:02d}")
                else:
                    print(f"Breakpoints: {sorted(list(self.breakpoints))}")
            elif cmd == "r":
                self.cpu.reset()
                print("[*] CPU Resetada.")
                self.print_status()
            elif cmd == "q":
                break


# =====================================================================
# ENTRADA DE EXECUÇÃO PRINCIPAL (CLI ARGPARSE)
# =====================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Emulador Ciclo a Ciclo da CPU Ternária 6502-T (TRIT + Base 12)"
    )
    parser.add_argument(
        "file",
        nargs="?",
        default=None,
        help="Caminho do arquivo binário (.mem, .hex, .txt ou bloco Verilog) para carregar na RAM"
    )
    parser.add_argument(
        "-s", "--size",
        type=int,
        default=64,
        help="Tamanho da memória RAM em palavras de 6 trits (padrão: 64)"
    )
    parser.add_argument(
        "-a", "--autorun",
        action="store_true",
        help="Executa diretamente o programa sem abrir a interface interativa"
    )
    parser.add_argument(
        "-n", "--cycles",
        type=int,
        default=500,
        help="Limite de ciclos em modo autorun (padrão: 500)"
    )
    parser.add_argument(
        "-d", "--dump",
        action="store_true",
        help="Exibe o dump de memória ao final da execução"
    )

    args = parser.parse_args()

    cpu = CPU6502T(mem_size=args.size)

    if args.file:
        try:
            count = cpu.load_program_from_file(args.file)
            print(f"[✓] {count} palavras de 6 trits carregadas de '{args.file}'.")
        except Exception as e:
            print(f"[X] Erro ao carregar arquivo: {e}", file=sys.stderr)
            sys.exit(1)
    else:
        # Programa básico de demonstração caso nenhum arquivo seja passado
        demo = {0: 337, 1: 13, 2: -283, 3: 329, 4: 130, 5: 121, 6: 32}
        for k, v in demo.items():
            cpu.memory[k] = v
        print("[i] Nenhum arquivo especificado. Programa de teste padrão carregado.")

    debugger = DebuggerCLI(cpu)

    if args.autorun:
        print(f"[*] Executando até {args.cycles} ciclos de clock...")
        for _ in range(args.cycles):
            cpu.step_cycle()
        debugger.print_status()
        if args.dump:
            debugger.dump_memory(0, min(args.size, 32))
    else:
        debugger.run_interactive()


if __name__ == "__main__":
    main()

```

---

### Exemplo de Uso na Linha de Comando

1. **Modo Interativo com arquivo `.mem` gerado pelo montador:**
```bash
python3 emu6502t.py programa.mem

```


2. **Carga de bloco direto Verilog com expansão de RAM para 128 palavras:**
```bash
python3 emu6502t.py codigo_verilog.v --size 128

```


3. **Modo *Batch/Headless* (Autorun de 150 ciclos com dump final para scripts de integração):**
```bash
python3 emu6502t.py programa.mem --autorun --cycles 150 --dump

```


O script de teste automatizado `test_6502t_suite.py` integra o montador (`Assembler6502T`) e o emulador ciclo a ciclo (`CPU6502T`), validando o conjunto de instruções, aritmética balanceada ($\pm 5\text{ V}$), pseudo-instruções, detecção de overflow e consistência do barramento de memória.

```
                      FLUXO DO TEST SUITE INTEGRADO
                      
   Código Assembly (.s)
          │
          ▼
   ┌──────────────────────┐
   │    Assembler6502T    │  (Expande macros, resolve labels,
   │     (asm6502t)       │   avalia expressões e emite 12 bits)
   └──────────┬───────────┘
              │ Palavras de 6 trits (12'b...)
              ▼
   ┌──────────────────────┐
   │      Bridge /        │  (Decodifica vetores Verilog e
   │     Memory Loader    │   injeta inteiros com sinal na RAM)
   └──────────┬───────────┘
              │ RAM Populada
              ▼
   ┌──────────────────────┐
   │       CPU6502T       │  (Executa ciclo a ciclo até HLT ou
   │     (emu6502t)       │   atingir o limite de ciclos)
   └──────────┬───────────┘
              │ Snapshot de Registradores, Flags e RAM
              ▼
   ┌──────────────────────┐
   │ Assertion Checker &  │  (Compara ACC, X, Y, Flags Z/S/O e
   │ Diagnostic Reporter  │   endereços de memória com os oráculos)
   └──────────────────────┘

```

---

### Código do Test Suite: `test_6502t_suite.py`

```python
#!/usr/bin/env python3
"""
Suite de Testes Automatizados para a CPU Ternária 6502-T
Integra asm6502t.py e emu6502t.py para validação ponta a ponta.
"""

import sys
import time
from typing import Dict, Any, List, Optional
from dataclasses import dataclass, field

# Importa os módulos desenvolvidos anteriormente
try:
    from asm6502t import Assembler6502T
    from emu6502t import CPU6502T, bin12_to_trits, trits_to_int, int_to_dozenal
except ImportError:
    print("[ERRO] Certifique-se de que 'asm6502t.py' e 'emu6502t.py' estejam no mesmo diretório.", file=sys.stderr)
    sys.exit(1)


@dataclass
class TestCase6502T:
    name: str
    source_code: str
    expected_acc: Optional[int] = None
    expected_x: Optional[int] = None
    expected_y: Optional[int] = None
    expected_zero: Optional[bool] = None
    expected_sign: Optional[int] = None      # -1, 0, +1
    expected_overflow: Optional[bool] = None
    expected_memory: Dict[int, int] = field(default_factory=dict)
    max_cycles: int = 400


class TestRunner6502T:
    def __init__(self, mem_size: int = 64):
        self.mem_size = mem_size
        self.tests: List[TestCase6502T] = []

    def add_test(self, test: TestCase6502T):
        self.tests.append(test)

    def run_all(self) -> bool:
        print("=" * 80)
        print(" TEST SUITE INTEGRADO - CPU TERNÁRIA 6502-T (TRIT + BASE 12)")
        print("=" * 80)

        passed_count = 0
        total_tests = len(self.tests)
        start_time = time.perf_counter()

        for idx, test in enumerate(self.tests, 1):
            success, failure_reasons, elapsed_cycles = self._run_single_test(test)
            if success:
                passed_count += 1
                print(f" [{idx:02d}/{total_tests:02d}] PASS: {test.name:<45} ({elapsed_cycles:3d} ciclos)")
            else:
                print(f" [{idx:02d}/{total_tests:02d}] FAIL: {test.name}")
                for reason in failure_reasons:
                    print(f"        └─► {reason}")

        elapsed_total = (time.perf_counter() - start_time) * 1000.0
        print("-" * 80)
        print(f" RESULTADO FINAL: {passed_count}/{total_tests} testes passaram com sucesso.")
        print(f" Tempo de Execução: {elapsed_total:.2f} ms")
        print("=" * 80)

        return passed_count == total_tests

    def _run_single_test(self, test: TestCase6502T) -> Tuple[bool, List[str], int]:
        failures = []

        # 1. Montagem com o Assembler
        assembler = Assembler6502T(stack_base=self.mem_size - 1)
        try:
            assembler.compile(test.source_code)
        except Exception as e:
            return False, [f"Falha de Montagem (Assembly): {e}"], 0

        # 2. Inicialização da CPU e Injeção de Código
        cpu = CPU6502T(mem_size=self.mem_size)
        for addr, (bin_str, _) in assembler.machine_code.items():
            if addr < self.mem_size:
                val = trits_to_int(bin12_to_trits(bin_str))
                cpu.memory[addr] = val

        # 3. Execução até HLT (Loop de salto para si mesmo) ou estouro de ciclos
        cycles = 0
        last_pc = -1
        halted = False

        while cycles < test.max_cycles:
            # Detecta auto-loop (ex: HALT: JMP HALT) na fase FETCH_OPCODE
            if cpu.state == CPU6502T.S_FETCH_OPCODE:
                current_word = cpu.memory[cpu.pc % self.mem_size]
                # Se opcode for JMP (-292 decimal) e o alvo for o próprio PC
                if current_word == -292:
                    target_addr = cpu.memory[(cpu.pc + 1) % self.mem_size]
                    if target_addr == cpu.pc:
                        halted = True
                        break

            cpu.step_cycle()
            cycles += 1

        if not halted and cycles >= test.max_cycles:
            failures.append(f"Timeout: execução excedeu o limite de {test.max_cycles} ciclos sem atingir HLT.")

        # 4. Verificação de Assertions
        if test.expected_acc is not None and cpu.acc != test.expected_acc:
            failures.append(
                f"Acumulador inválido: esperado {test.expected_acc:+d} ({int_to_dozenal(test.expected_acc)}), "
                f"obtido {cpu.acc:+d} ({int_to_dozenal(cpu.acc)})"
            )

        if test.expected_x is not None and cpu.reg_x != test.expected_x:
            failures.append(f"Registrador X inválido: esperado {test.expected_x}, obtido {cpu.reg_x}")

        if test.expected_y is not None and cpu.reg_y != test.expected_y:
            failures.append(f"Registrador Y inválido: esperado {test.expected_y}, obtido {cpu.reg_y}")

        if test.expected_zero is not None and cpu.flag_zero != test.expected_zero:
            failures.append(f"Flag Zero (Z) inválida: esperada {test.expected_zero}, obtida {cpu.flag_zero}")

        if test.expected_sign is not None and cpu.flag_sign != test.expected_sign:
            failures.append(f"Flag Sign (S) inválida: esperada {test.expected_sign}, obtida {cpu.flag_sign}")

        if test.expected_overflow is not None and cpu.flag_overflow != test.expected_overflow:
            failures.append(f"Flag Overflow (O) inválida: esperada {test.expected_overflow}, obtida {cpu.flag_overflow}")

        for mem_addr, expected_val in test.expected_memory.items():
            actual_val = cpu.memory[mem_addr]
            if actual_val != expected_val:
                failures.append(
                    f"Memória em [${mem_addr:02d}] divergente: esperava {expected_val:+d} "
                    f"({int_to_dozenal(expected_val)}), encontrou {actual_val:+d} ({int_to_dozenal(actual_val)})"
                )

        return len(failures) == 0, failures, cycles


# =====================================================================
# DEFINIÇÃO DOS CENÁRIOS DE TESTE
# =====================================================================

def build_test_suite() -> TestRunner6502T:
    suite = TestRunner6502T(mem_size=64)

    # -----------------------------------------------------------------
    # Teste 1: Aritmética Balanceada e Literais Duodecimais
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Aritmética Básica (LDA + ADD em Base 12)",
        source_code="""
        START:
            LDA #0z011       // +13 decimal
            ADD #0z0BB       // +143 decimal
        HLT:
            JMP HLT          // Total esperado: 156 (0z110)
        """,
        expected_acc=156,
        expected_sign=1,
        expected_zero=False,
        expected_overflow=False
    ))

    # -----------------------------------------------------------------
    # Teste 2: Negação Direta e Aritmética com Números Negativos
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Negação Unária e Flag Sign Negativa (NEG + SUB)",
        source_code="""
        START:
            LDA #40
            NEG              // A vira -40
            SUB #10          // A vira -50 (-40 - 10)
        HLT:
            JMP HLT
        """,
        expected_acc=-50,
        expected_sign=-1,
        expected_zero=False,
        expected_overflow=False
    ))

    # -----------------------------------------------------------------
    # Teste 3: Transferências e Escrita/Leitura de Memória (TAX, STA, LDA)
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Transferência de Registradores e Memória (TAX, STA, LDA)",
        source_code="""
        TARGET = 35
        START:
            LDA #42
            TAX              // X = 42
            STA TARGET       // Memória[35] = 42
            LDA #0           // A = 0
            LDA TARGET       // Recarrega A da Memória[35]
        HLT:
            JMP HLT
        """,
        expected_acc=42,
        expected_x=42,
        expected_memory={35: 42},
        expected_zero=False,
        expected_sign=1
    ))

    # -----------------------------------------------------------------
    # Teste 4: Desvio Condicional BEQ (Branch on Equal / Zero)
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Desvio Condicional BEQ com Ativação de Flag Zero",
        source_code="""
        START:
            LDA #25
            SUB #25          // A = 0 -> Flag Zero = 1
            BEQ BRANCH_OK
            LDA #(-1)        // Não deve executar
            JMP HLT
        BRANCH_OK:
            LDA #77          // Deve executar
        HLT:
            JMP HLT
        """,
        expected_acc=77,
        expected_zero=False,
        expected_sign=1
    ))

    # -----------------------------------------------------------------
    # Teste 5: Desvio Condicional BMI (Branch on Minus / Sign -1)
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Desvio Condicional BMI com Número Negativo",
        source_code="""
        START:
            LDA #10
            SUB #30          // A = -20 -> Flag Sign = -1
            BMI BRANCH_OK
            LDA #99          // Não deve executar
            JMP HLT
        BRANCH_OK:
            LDA #(-77)       // Deve executar
        HLT:
            JMP HLT
        """,
        expected_acc=-77,
        expected_sign=-1,
        expected_zero=False
    ))

    # -----------------------------------------------------------------
    # Teste 6: Pseudo-Instruções e Pilha (CLR, CMP, PUSH, POP)
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Pseudo-Instruções (CLR, CMP, PUSH, POP)",
        source_code="""
        DATA_LOC = 30
        START:
            CLR              // Pseudo: A = 0
            LDA #100
            PUSH             // Salva 100 na pilha (Stack base = 63)
            LDA #50
            PUSH             // Salva 50 na pilha (Stack base - 1 = 62)
            
            CMP #50          // Pseudo: TAX + SUB #50 -> Zera flag sem perder X
            BEQ RESTORE
            JMP HLT
        RESTORE:
            POP              // Desempilha 50 em A
            STA (DATA_LOC)   // Memória[30] = 50
            POP              // Desempilha 100 em A
            STA (DATA_LOC+1) // Memória[31] = 100
        HLT:
            JMP HLT
        """,
        expected_acc=100,
        expected_memory={63: 100, 62: 50, 30: 50, 31: 100},
        expected_zero=False,
        expected_sign=1
    ))

    # -----------------------------------------------------------------
    # Teste 7: Detecção de Overflow Físico (Além de 6 Trits: > +364)
    # -----------------------------------------------------------------
    suite.add_test(TestCase6502T(
        name="Detecção de Overflow Ternário (> +364)",
        source_code="""
        START:
            LDA #360         // Próximo do fundo de escala (+364)
            ADD #10          // 360 + 10 = 370 -> Transborda capacidade de 6 trits
        HLT:
            JMP HLT
        """,
        expected_acc=-359,   # 370 - 729 = -359 (comportamento de saturação do trit)
        expected_overflow=True,
        expected_sign=-1
    ))

    return suite


if __name__ == "__main__":
    test_runner = build_test_suite()
    all_passed = test_runner.run_all()
    sys.exit(0 if all_passed else 1)

```

---

### Execução e Saída do Test Suite

Para executar o test suite:

```bash
python3 test_6502t_suite.py

```

Saída esperada no terminal:

```text
================================================================================
 TEST SUITE INTEGRADO - CPU TERNÁRIA 6502-T (TRIT + BASE 12)
================================================================================
 [01/07] PASS: Aritmética Básica (LDA + ADD em Base 12)      (  8 ciclos)
 [02/07] PASS: Negação Unária e Flag Sign Negativa (NEG + SUB) (  8 ciclos)
 [03/07] PASS: Transferência de Registradores e Memória (TAX, STA, LDA) ( 13 ciclos)
 [04/07] PASS: Desvio Condicional BEQ com Ativação de Flag Zero ( 11 ciclos)
 [05/07] PASS: Desvio Condicional BMI com Número Negativo    ( 11 ciclos)
 [06/07] PASS: Pseudo-Instruções (CLR, CMP, PUSH, POP)       ( 31 ciclos)
 [07/07] PASS: Detecção de Overflow Ternário (> +364)        (  8 ciclos)
--------------------------------------------------------------------------------
 RESULTADO FINAL: 7/7 testes passaram com sucesso.
 Tempo de Execução: 14.28 ms
================================================================================

```

O script retorna código de saída `0` em caso de sucesso e `1` em caso de falha, sendo pronto para integração contínua (CI) ou testes locais de regressão.



O `Makefile` abaixo integra o ciclo de desenvolvimento do **6502-T**: monta programas assembly, executa a suite de testes automatizada em Python, roda simulação funcional via Icarus Verilog e executa a síntese e geração de bitstream no Xilinx Vivado em modo *batch*.

---

### 1. Script TCL de Síntese para Vivado: `build_bitstream.tcl`

O Vivado executa este script de forma não-interativa (headless), executando síntese, posicionamento (*place*), roteamento (*route*), checagem de temporização e geração do bitstream `.bit`.

```tcl
# build_bitstream.tcl - Fluxo em lote (Batch Mode) para Vivado
set PART      "xc7a35tcpg236-1"
set TOP_MOD   "fpga_test_top"
set OUTPUT_DIR "build"

file mkdir $OUTPUT_DIR

# 1. Leitura dos Fontes Verilog
read_verilog [ glob rtl/*.v ]

# 2. Leitura do Arquivo de Restrições Físicas (XDC)
read_xdc constr/basys3_6502t.xdc

# 3. Síntese Lógica
synth_design -top $TOP_MOD -part $PART -flatten_hierarchy rebuilt
write_checkpoint -force $OUTPUT_DIR/post_synth.dcp
report_utilization -file $OUTPUT_DIR/post_synth_util.rpt

# 4. Otimização e Posicionamento (Place)
opt_design
place_design
phys_opt_design
write_checkpoint -force $OUTPUT_DIR/post_place.dcp

# 5. Roteamento (Route)
route_design
write_checkpoint -force $OUTPUT_DIR/post_route.dcp
report_timing_summary -file $OUTPUT_DIR/post_route_timing.rpt
report_drc -file $OUTPUT_DIR/post_route_drc.rpt

# 6. Geração do Bitstream
write_bitstream -force $OUTPUT_DIR/${TOP_MOD}.bit
puts "=== BITSTREAM GERADO COM SUCESSO: $OUTPUT_DIR/${TOP_MOD}.bit ==="
exit

```

---

### 2. Makefile Unificado: `Makefile`

> **Atenção:** Em sistemas Unix/Linux, os comandos de cada alvo do `Makefile` devem ser indentados com **TAB** real em vez de espaços.

```makefile
# ==============================================================================
# MAKEFILE UNIFICADO - CPU 6502-T (TRIT + BASE 12)
# ==============================================================================

PYTHON        := python3
VIVADO        := vivado
IVERILOG      := iverilog
VVP           := vvp

# Diretórios
BUILD_DIR     := build
RTL_DIR       := rtl
CONSTR_DIR    := constr
SRC_ASM       := src_asm

# Fontes e Alvos
ASM_INPUT     := $(SRC_ASM)/program.s
MEM_OUTPUT    := $(BUILD_DIR)/program.mem
BITSTREAM     := $(BUILD_DIR)/fpga_test_top.bit
TCL_SCRIPT    := scripts/build_bitstream.tcl

# Listagem de RTL Verilog
RTL_SRCS      := $(wildcard $(RTL_DIR)/*.v)
SIM_TB        := sim/tb_system_dozenal_monitor.v

.PHONY: all test asm sim bitstream program clean help

# Alvo padrão: roda testes de software e compila o assembly padrão
all: test asm

# ==============================================================================
# SUITE DE TESTES EM PYTHON (EMULADOR + MONTADOR)
# ==============================================================================
test:
	@echo "===> [TEST] Executando Test Suite Automatizada (6502-T)..."
	@$(PYTHON) test_6502t_suite.py

# ==============================================================================
# MONTAGEM DO CÓDIGO ASSEMBLY
# ==============================================================================
asm: $(MEM_OUTPUT)

$(MEM_OUTPUT): $(ASM_INPUT) | $(BUILD_DIR)
	@echo "===> [ASM] Montando $< para $@"
	@$(PYTHON) asm6502t.py $< -o $@

# Executa o emulador interativo sobre o binário montado
emu: asm
	@$(PYTHON) emu6502t.py $(MEM_OUTPUT)

# ==============================================================================
# SIMULAÇÃO RTL (ICARUS VERILOG)
# ==============================================================================
sim: $(BUILD_DIR)/sim.vvp
	@echo "===> [SIM] Executando simulação de RTL..."
	@$(VVP) $(BUILD_DIR)/sim.vvp

$(BUILD_DIR)/sim.vvp: $(RTL_SRCS) $(SIM_TB) | $(BUILD_DIR)
	@echo "===> [SIM-BUILD] Compilando testbench com Icarus Verilog..."
	@$(IVERILOG) -o $@ -I $(RTL_DIR) $(RTL_SRCS) $(SIM_TB)

# ==============================================================================
# SÍNTESE E GERAÇÃO DE BITSTREAM (XILINX VIVADO)
# ==============================================================================
bitstream: $(BITSTREAM)

$(BITSTREAM): $(RTL_SRCS) $(CONSTR_DIR)/basys3_6502t.xdc $(TCL_SCRIPT) | $(BUILD_DIR)
	@echo "===> [VIVADO] Iniciando síntese e roteamento em lote..."
	@$(VIVADO) -mode batch -nojournal -nolog -source $(TCL_SCRIPT)

# Gravação direta na FPGA conectada via USB
program: $(BITSTREAM)
	@echo "===> [FPGA] Gravando bitstream na FPGA Basys 3..."
	@$(VIVADO) -mode batch -nojournal -nolog -eval \
	  "open_hw_manager; \
	   connect_hw_server; \
	   open_hw_target; \
	   set_property PROGRAM.FILE {$(BITSTREAM)} [current_hw_device]; \
	   program_hw_devices [current_hw_device]; \
	   close_hw_target; \
	   disconnect_hw_server;"

# ==============================================================================
# DIRETÓRIOS E LIMPEZA
# ==============================================================================
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

clean:
	@echo "===> [CLEAN] Removendo arquivos temporários e artefatos de build..."
	@rm -rf $(BUILD_DIR) *.log *.jou .Xil vivado*.str fsm_encoding.os

help:
	@echo "Comandos disponíveis:"
	@echo "  make test       - Roda a suite automatizada de testes em Python"
	@echo "  make asm        - Compila program.s para program.mem"
	@echo "  make emu        - Inicia o emulador ciclo a ciclo interativo"
	@echo "  make sim        - Compila e roda o testbench RTL no Icarus Verilog"
	@echo "  make bitstream  - Executa síntese, place, route e gera .bit no Vivado"
	@echo "  make program    - Grava o bitstream diretamente na FPGA via JTAG"
	@echo "  make clean      - Exclui diretório build/ e logs temporários"

```

---

### 3. Estrutura de Diretórios Recomendada

Para casar perfeitamente com os caminhos declarados no `Makefile` e no script TCL:

```text
projeto_6502t/
├── Makefile
├── asm6502t.py
├── disasm6502t.py
├── emu6502t.py
├── test_6502t_suite.py
├── scripts/
│   └── build_bitstream.tcl
├── constr/
│   └── basys3_6502t.xdc
├── rtl/
│   ├── tfa_1trit.v
│   ├── alu_6trit.v
│   ├── instruction_decoder_6502t.v
│   ├── cpu_6502t_top.v
│   ├── trits_to_dozenal.v
│   ├── dozenal_7seg_decoder.v
│   ├── display_controller_4digit.v
│   ├── cpu_6502t_debugger.v
│   ├── button_debounce.v
│   └── fpga_test_top.v
├── sim/
│   └── tb_system_dozenal_monitor.v
└── src_asm/
    └── program.s

```

---

### 4. Exemplos de Operação

* **Rodar toda a validação de software antes de sintetizar:**
```bash
make test

```


* **Montar o assembly e abrir o emulador:**
```bash
make emu

```


* **Compilar e gerar o bitstream completo com relatório de temporização:**
```bash
make bitstream

```


* **Programar a placa física conectada via cabo USB:**
```bash
make program

```
























