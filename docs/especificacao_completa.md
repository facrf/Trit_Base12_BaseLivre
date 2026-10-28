# ESPECIFICAÇÃO TÉCNICA E ARQUITETURAL: PROJETO TRIT + BASE 12 + CHIP (6502-T)

---

## 1. Visão Geral do Projeto

O projeto **TRIT + base 12 + chip** estabelece uma arquitetura computacional baseada em três pilares fundamentais:
1. **Camada Física e Lógica Primitiva:** Aritmética ternária balanceada $\{-1, 0, +1\}$ operada por células chaveadoras BJT em três níveis de potencial elétrico ($+5\text{ V}$, $0\text{ V}$, $-5\text{ V}$).
2. **Camada de Abstração Algébrica e Interfacing:** Sistema numérico duodecimal (base 12), aproveitando sua alta divisibilidade e representação fracionária exata para endereçamento, barramentos e interação externa.
3. **Núcleo de Processamento:** Derivação em silício (*fork*) da consagrada arquitetura do microprocessador **MOS 6502**, adaptada para registradores de 6 trits balanceados ($[-364, +364]$) e barramento de endereçamento de 10 trits ($59.049$ endereços).

---

## 2. O Conceito do TRIT e o BJT de Três Estágios

O **trit** (*ternary digit*) é a unidade elementar da computação ternária, assumindo os estados balanceados $\{-1, 0, +1\}$ (notação gráfica $\{-, 0, +\}$). Essa abordagem foi validada pelo computador soviético **Setun** (1958, Universidade Estatal de Moscou), desenvolvido sob liderança de Nikolai P. Brusentsov, operando com amplificadores magnéticos de ferrite saturável.

### 2.1. Implementação Física com BJT ($\pm 5\text{ V}$, $0\text{ V}$)
A implementação física utiliza pares complementares de Transistores de Junção Bipolar (BJT) conectados a três barramentos de alimentação simétricos: $+5\text{ V}$, $0\text{ V}$ (GND/referência central) e $-5\text{ V}$.

```
                 +5V (+1 Lógico)
                    │
                 ┌──┴──┐
                 │ NPN │ (Conduz quando V_in > +V_th)
                 └──┬──┘
                    ├────── V_out (Nível do Trit)
                 ┌──┴──┐
                 │ PNP │ (Conduz quando V_in < -V_th)
                 └──┬──┘
                    │
                 -5V (-1 Lógico)
                    
  (Em V_in = 0V, ambos os transistores cortam e resistores
   de centralização forçam V_out para 0V estável)
```

* **Estado $+1$ ($+5\text{ V}$):** O transistor NPN satura, conectando o nó de saída diretamente ao trilho de $+5\text{ V}$.
* **Estado $0$ ($0\text{ V}$):** Ambos os transistores (NPN e PNP) entram em **corte**. Um divisor de tensão balanceado de precisão ou resistor de *pull-to-ground* ancora o nó em $0\text{ V}$ sem flutuações de alta impedância.
* **Estado $-1$ ($-5\text{ V}$):** O transistor PNP satura, conectando o nó de saída ao trilho inferior de $-5\text{ V}$.

### 2.2. Vantagens Aritméticas da Lógica Balanceada
1. **Sinal Intrínseco:** Dispensa circuitos adicionais ou convenções de bit de sinal (como o complemento de dois binário). O sinal do número é determinado unicamente pelo primeiro trit não-nulo mais significativo.
2. **Negação a Custo Zero:** A operação de inversão de sinal ($A \to -A$) exige apenas a transposição dos potenciais elétricos (troca direta de $+5\text{ V}$ por $-5\text{ V}$ e vice-versa), sem necessidade de somadores de incremento ($+1$).
3. **Subtração como Adição:** A operação de subtração $A - B$ reduz-se a somar $A + (-B)$ na mesma célula de hardware.
4. **Arredondamento por Truncamento:** O truncamento de trits menos significativos resulta naturalmente em arredondamento simétrico ao inteiro mais próximo, eliminando o viés cumulativo típico de truncamentos em base binária.

---

## 3. Fundamentação Matemática: Euler, Pi, Gauss e Eficiência de Radix

### 3.1. A Constante de Euler ($e$) e o Custo de Radix
A complexidade de hardware de uma arquitetura digital é modelada pelo **Custo de Radix** ($E$), definido como o produto da base ($R$) pela quantidade de dígitos ($d$) necessários para representar um intervalo numérico $N$:

$$N = R^d \implies d = \frac{\ln N}{\ln R}$$

$$E(R) = R \cdot d = R \cdot \frac{\ln N}{\ln R} = \ln N \cdot \left( \frac{R}{\ln R} \right)$$

Minimizando a função $f(R) = \frac{R}{\ln R}$ no domínio real contínuo:

$$f'(R) = \frac{\ln R - 1}{(\ln R)^2} = 0 \implies \ln R = 1 \implies R = e \approx 2{,}71828$$

| Base ($R$) | Nome | $f(R) = \frac{R}{\ln R}$ | Eficiência Relativa em Silício |
| :---: | :---: | :---: | :---: |
| **2** | Binário | $\frac{2}{\ln 2} \approx 2{,}885$ | $94{,}2\%$ |
| **3** | **Ternário** | **$\frac{3}{\ln 3} \approx 2{,}731$** | **$99{,}5\%$** |
| **$e$** | Ideal Matemático | $\frac{e}{\ln e} \approx 2{,}718$ | $100{,}0\%$ |
| **4** | Quaternário | $\frac{4}{\ln 4} \approx 2{,}885$ | $94{,}2\%$ |
| **10** | Decimal | $\frac{10}{\ln 10} \approx 4{,}343$ | $62{,}6\%$ |
| **12** | Duodecimal | $\frac{12}{\ln 12} \approx 4{,}829$ | $56{,}3\%$ |

A base 3 é o inteiro físico mais próximo do número de Euler, provendo a máxima densidade informacional por condutor de barramento e porta lógica.

A **Identidade de Euler** conecta a lógica ternária às grandezas de fase no plano complexo:

$$e^{i\pi} + 1 = 0 \iff e^{i\pi} = -1$$

Ao estender os três estados discretos para fasores harmônicos de rotação:

$$\omega_k = e^{i \frac{2\pi k}{3}} \quad \text{para } k \in \{-1, 0, +1\}$$

$$\omega_0 = 1 \quad (0\text{ V}), \quad \omega_{+1} = -\frac{1}{2} + i\frac{\sqrt{3}}{2} \quad (+5\text{ V}), \quad \omega_{-1} = -\frac{1}{2} - i\frac{\sqrt{3}}{2} \quad (-5\text{ V})$$

A soma vetorial nula ($\sum \omega_k = 0$) traduz em hardware a conservação nodal das correntes nos circuitos somadores.

No nível duodecimal, a **Função Totiente de Euler** ($\phi$) explicita a alta densidade de divisores da base 12:

$$\phi(12) = 12 \left(1 - \frac{1}{2}\right)\left(1 - \frac{1}{3}\right) = 4$$

Apenas quatro inteiros menores que 12 são coprimos com ele ($\{1, 5, 7, 11\}$). Todos os outros oito inteiros compartilham fatores com a base, facilitando reduções algébricas em ponto fixo.

### 3.2. A Constante Pi ($\pi$) e a Geometria Modular
A base 12 divide o círculo trigonométrico de comprimento $2\pi$ em arcos exatos:

$$\theta_n = n \cdot \frac{2\pi}{12} = n \cdot \frac{\pi}{6} \quad (n \in \{0, 1, 2, \dots, 11\})$$

Cada unidade fracionária em base 12 ($0{,}1_{12}$) corresponde exatamente a $30^\circ$ ($\frac{\pi}{6}\text{ rad}$), permitindo cálculos trigonométricos sem truncamento irracional nos ângulos notáveis.

A expansão de $\pi$ em notação duodecimal apresenta convergência rápida:

$$\pi_{10} \approx 3{,}1415926535... \implies \pi_{12} = 3{,}184809493B91..._{12}$$

$$\left(\frac{\pi}{4}\right)_{12} \approx 0{,}9412..._{12}$$

### 3.3. Gauss: Resíduos Mínimos e Simetria de Ruído
Em *Disquisitiones Arithmeticae* (1801), Carl Friedrich Gauss definiu o **sistema de resíduos de menor valor absoluto** para congruências $x \equiv a \pmod m$, onde os restos são tomados dentro do intervalo simétrico:

$$-\frac{m}{2} < r \le \frac{m}{2}$$

Para o módulo $m = 3$, o sistema de Gauss é o conjunto $\{-1, 0, +1\}$. A lógica ternária balanceada é a realização em silício da teoria de resíduos mínimos de Gauss.

Nas comutações com BJTs, o ruído térmico de Johnson-Nyquist nos resistores de entrada segue a distribuição normal gaussiana:

$$p(V_n) = \frac{1}{\sigma \sqrt{2\pi}} \exp\left(-\frac{V_n^2}{2\sigma^2}\right), \quad \sigma^2 = 4 k_B T R \Delta f$$

Com os limiares de transição simétricos em $\pm 2{,}5\text{ V}$, a integral das caudas gaussianas é rigorosamente balanceada:

$$P(\text{erro } 0 \to +1) = P(\text{erro } 0 \to -1) = \frac{1}{2} \operatorname{erfc}\left(\frac{V_{th}}{\sigma \sqrt{2}}\right)$$

---

## 4. A Camada Duodecimal (Base 12)

Enquanto a base 3 rege a física de comutação do silício, a base 12 estrutura a interface de dados, representação e endereçamento da ISA.

### 4.1. Vantagens Operacionais
* **Alta Divisibilidade:** O número 12 possui divisores próprios $\{2, 3, 4, 6\}$ (o número 10 possui apenas $\{2, 5\}$).
* **Frações Finitas:**
  * $\frac{1}{2} = 0{,}6_{12}$
  * $\frac{1}{3} = 0{,}4_{12}$
  * $\frac{1}{4} = 0{,}3_{12}$
  * $\frac{1}{6} = 0{,}2_{12}$
* **Casamento de Barramento:** Subdivisão limpa de campos em metades, terços e quartos sem erros de arredondamento.

### 4.2. Desafios Arquiteturais
* **Mapeamento Trit $\leftrightarrow$ Dozenal:** Três trits balanceados fornecem $3^3 = 27$ estados (faixa $-13$ a $+13$), enquanto 1 dígito de base 12 necessita de apenas 12 estados ($0$ a $11$, rotulados $0\text{--}9, \text{A}, \text{B}$). Os estados redundantes são aproveitados em instruções especiais, flags e controle de fluxo.

---

## 5. O Somador Completo Ternário (TFA) Balanceado

O TFA recebe três entradas $A, B, C_{in} \in \{-1, 0, +1\}$ e gera a soma $S \in \{-1, 0, +1\}$ e o transporte $C_{out} \in \{-1, 0, +1\}$:

$$S_{int} = A + B + C_{in} = 3 \cdot C_{out} + S, \quad S_{int} \in [-3, +3]$$

$$C_{out} = \begin{cases} +1, & \text{se } S_{int} \ge +2 \\ 0, & \text{se } -1 \le S_{int} \le +1 \\ -1, & \text{se } S_{int} \le -2 \end{cases}, \qquad S = S_{int} - 3 \cdot C_{out}$$

### 5.1. Tabela-Verdade Completa ($3^3 = 27$ Estados)

| $A$ | $B$ | $C_{in}$ | Tensões ($V_A, V_B, V_{Cin}$) | $S_{int}$ | $C_{out}$ | $S$ | $V_{Cout}$ | $V_S$ |
| :-: | :-: | :------: | :---------------------------: | :-------: | :-------: | :-: | :--------: | :---: |
| $-$ | $-$ |   $-$    |        $-5V, -5V, -5V$        |   $-3$    |    $-$    | $0$ |   $-5V$    | $0V$  |
| $-$ | $-$ |   $0$    |        $-5V, -5V, 0V$         |   $-2$    |    $-$    | $+$ |   $-5V$    | $+5V$ |
| $-$ | $-$ |   $+$    |        $-5V, -5V, +5V$        |   $-1$    |    $0$    | $-$ |    $0V$    | $-5V$ |
| $-$ | $0$ |   $-$    |        $-5V, 0V, -5V$         |   $-2$    |    $-$    | $+$ |   $-5V$    | $+5V$ |
| $-$ | $0$ |   $0$    |         $-5V, 0V, 0V$         |   $-1$    |    $0$    | $-$ |    $0V$    | $-5V$ |
| $-$ | $0$ |   $+$    |        $-5V, 0V, +5V$         |    $0$    |    $0$    | $0$ |    $0V$    | $0V$  |
| $-$ | $+$ |   $-$    |        $-5V, +5V, -5V$        |   $-1$    |    $0$    | $-$ |    $0V$    | $-5V$ |
| $-$ | $+$ |   $0$    |        $-5V, +5V, 0V$         |    $0$    |    $0$    | $0$ |    $0V$    | $0V$  |
| $-$ | $+$ |   $+$    |        $-5V, +5V, +5V$        |   $+1$    |    $0$    | $+$ |    $0V$    | $+5V$ |
| $0$ | $-$ |   $-$    |         $0V, -5V, -5V$        |   $-2$    |    $-$    | $+$ |   $-5V$    | $+5V$ |
| $0$ | $-$ |   $0$    |         $0V, -5V, 0V$         |   $-1$    |    $0$    | $-$ |    $0V$    | $-5V$ |
| $0$ | $-$ |   $+$    |         $0V, -5V, +5V$        |    $0$    |    $0$    | $0$ |    $0V$    | $0V$  |
| $0$ | $0$ |   $-$    |         $0V, 0V, -5V$         |   $-1$    |    $0$    | $-$ |    $0V$    | $-5V$ |
| $0$ | $0$ |   $0$    |          $0V, 0V, 0V$         |    $0$    |    $0$    | $0$ |    $0V$    | $0V$  |
| $0$ | $0$ |   $+$    |         $0V, 0V, +5V$         |   $+1$    |    $0$    | $+$ |    $0V$    | $+5V$ |
| $0$ | $+$ |   $-$    |         $0V, +5V, -5V$        |    $0$    |    $0$    | $0$ |    $0V$    | $0V$  |
| $0$ | $+$ |   $0$    |         $0V, +5V, 0V$         |   $+1$    |    $0$    | $+$ |    $0V$    | $+5V$ |
| $0$ | $+$ |   $+$    |         $0V, +5V, +5V$        |   $+2$    |    $+$    | $-$ |   $+5V$    | $-5V$ |
| $+$ | $-$ |   $-$    |        $+5V, -5V, -5V$        |   $-1$    |    $0$    | $-$ |    $0V$    | $-5V$ |
| $+$ | $-$ |   $0$    |        $+5V, -5V, 0V$         |    $0$    |    $0$    | $0$ |    $0V$    | $0V$  |
| $+$ | $-$ |   $+$    |        $+5V, -5V, +5V$        |   $+1$    |    $0$    | $+$ |    $0V$    | $+5V$ |
| $+$ | $0$ |   $-$    |        $+5V, 0V, -5V$         |    $0$    |    $0$    | $0$ |    $0V$    | $0V$  |
| $+$ | $0$ |   $0$    |         $+5V, 0V, 0V$         |   $+1$    |    $0$    | $+$ |    $0V$    | $+5V$ |
| $+$ | $0$ |   $+$    |        $+5V, 0V, +5V$         |   $+2$    |    $+$    | $-$ |   $+5V$    | $-5V$ |
| $+$ | $+$ |   $-$    |        $+5V, +5V, -5V$        |   $+1$    |    $0$    | $+$ |    $0V$    | $+5V$ |
| $+$ | $+$ |   $0$    |        $+5V, +5V, 0V$         |   $+2$    |    $+$    | $-$ |   $+5V$    | $-5V$ |
| $+$ | $+$ |   $+$    |        $+5V, +5V, +5V$        |   $+3$    |    $+$    | $0$ |   $+5V$    | $0V$  |

### 5.2. Esquemático Elétrico BJT ($\pm 5\text{ V}$, $0\text{ V}$)

```
========================================================================================
                          ESTÁGIO 1: SOMA ANALÓGICA POR NÓ KCL
========================================================================================
    A   o───[ R1: 10k ]───┐
                          │
    B   o───[ R2: 10k ]───┼───────o Nó V_sigma (Tensão proporcional a S_int)
                          │         (Escala: -5V a +5V mapeando S_int de -3 a +3)
    Cin o───[ R3: 10k ]───┘

========================================================================================
                  ESTÁGIO 2: DISCRIMINADOR DE JANELA E GERAÇÃO DE C_out
========================================================================================
         +5V
          │
         [Rc1]
          │
          ├─── Base Q3 (Driver NPN +5V)
          │
          c
   V_sigma ──b  Q1 (NPN - 2N3904)   [Threshold Superior: Conduz se V_sigma > +2.5V]
          e
          │
         [R_ref1] ───o V_ref+ (+2.5V gerado por divisor 10k/10k entre +5V e GND)
          
----------------------------------------------------------------------------------------
         [R_ref2] ───o V_ref- (-2.5V gerado por divisor 10k/10k entre GND e -5V)
          │
          e
   V_sigma ──b  Q2 (PNP - 2N3906)   [Threshold Inferior: Conduz se V_sigma < -2.5V]
          c
          │
          ├─── Base Q4 (Driver PNP -5V)
          │
         [Rc2]
          │
         -5V

========================================================================================
                ESTÁGIO 3: BUFFER PUSH-PULL TERNÁRIO DE SAÍDA (C_out)
========================================================================================
                      +5V
                       │
                       c
      (De Q1) ───────b  Q3 (NPN - 2N3904)
                       e
                       ├───────────────o Saída C_out (+5V, 0V, -5V)
                       │
                      [R_center: 2.2k]
                       │
                      GND (0V)
                       │
                       e
      (De Q2) ───────b  Q4 (PNP - 2N3906)
                       c
                       │
                      -5V

========================================================================================
             ESTÁGIO 4: RECONSTRUÇÃO DA SOMA (S = S_int - 3 * C_out)
========================================================================================
                       
    V_sigma ────[ R_sum1: 10k ]───┐
                                  │
    C_out   ────[ R_sum2: 30k ]───┼───────o Nó V_S_raw (Injeção de corrente oposta)
       (Invertido)                │
                                 [Rg: 10k]
                                  │
                                 GND (0V)

    * O sinal V_S_raw passa por discriminadores idênticos e estágio de saída push-pull
      com centralização em GND, restaurando os níveis plenos para a saída S.
```

---

## 6. Arquitetura do Processador 6502-T

O **MOS 6502** foi escolhido para o fork ternário devido à sua contagem reduzida de componentes (~3.510 transistores no original) e topologia de barramento aberta documentada.

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

### 6.1. Especificações do 6502-T
* **Palavra de Dados:** 6 trits balanceados ($3^6 = 729$ estados: $-364$ a $+364$). Mapeados em 12 bits de simulação (`2'b00` = $-1$, `2'b01` = $0$, `2'b10` = $+1$).
* **Barramento de Endereçamento:** 10 trits ($3^{10} = 59.049$ posições de memória).
* **Campos da Instrução (6 trits):**
  * $T_5, T_4$: Grupo de Instrução (ALU, Transferência, Salto, Memória).
  * $T_3, T_2$: Modo de Endereçamento (Implícito, Imediato, Absoluto).
  * $T_1, T_0$: Operação Específica (ADD, SUB, LOAD, NEG, INC, DEC).

---

## 7. Módulos Verilog Sintetizáveis

### 7.1. Somador Ternário de 1 Trit (`rtl/tfa_1trit.v`)
```verilog
`timescale 1ns / 1ps

module tfa_1trit (
    input  wire [1:0] a,      // 2'b00 = -1, 2'b01 = 0, 2'b10 = +1
    input  wire [1:0] b,
    input  wire [1:0] cin,
    output reg  [1:0] s,
    output reg  [1:0] cout
);

    localparam [1:0] TRIT_NEG  = 2'b00;
    localparam [1:0] TRIT_ZERO = 2'b01;
    localparam [1:0] TRIT_POS  = 2'b10;

    reg signed [1:0] val_a, val_b, val_cin;
    reg signed [3:0] s_int;
    reg signed [1:0] s_calc;

    always @(*) begin
        case (a)
            TRIT_NEG: val_a = -2'sd1;
            TRIT_POS: val_a =  2'sd1;
            default:  val_a =  2'sd0;
        endcase

        case (b)
            TRIT_NEG: val_b = -2'sd1;
            TRIT_POS: val_b =  2'sd1;
            default:  val_b =  2'sd0;
        endcase

        case (cin)
            TRIT_NEG: val_cin = -2'sd1;
            TRIT_POS: val_cin =  2'sd1;
            default:  val_cin =  2'sd0;
        endcase

        s_int = val_a + val_b + val_cin;

        if (s_int >= 4'sd2) begin
            cout   = TRIT_POS;
            s_calc = s_int - 4'sd3;
        end else if (s_int <= -4'sd2) begin
            cout   = TRIT_NEG;
            s_calc = s_int + 4'sd3;
        end else begin
            cout   = TRIT_ZERO;
            s_calc = s_int[1:0];
        end

        case (s_calc)
            -2'sd1:  s = TRIT_NEG;
             2'sd0:  s = TRIT_ZERO;
             2'sd1:  s = TRIT_POS;
            default: s = TRIT_ZERO;
        endcase
    end

endmodule
```

### 7.2. ALU de 6 Trits com Registrador Acumulador (`rtl/alu_6trit.v`)
```verilog
`timescale 1ns / 1ps

module alu_6trit (
    input  wire        clk,
    input  wire        rst_n,
    input  wire        load_acc,
    input  wire [2:0]  opcode,
    input  wire [11:0] operand_b,
    input  wire [1:0]  cin_ext,
    output reg  [11:0] acc_out,
    output wire [1:0]  cout,
    output wire        flag_zero,
    output wire [1:0]  flag_sign,
    output wire        flag_overflow
);

    localparam [1:0] TRIT_NEG  = 2'b00;
    localparam [1:0] TRIT_ZERO = 2'b01;
    localparam [1:0] TRIT_POS  = 2'b10;

    localparam [2:0] ALU_ADD   = 3'b000;
    localparam [2:0] ALU_SUB   = 3'b001;
    localparam [2:0] ALU_NEG_A = 3'b010;
    localparam [2:0] ALU_LOAD  = 3'b011;
    localparam [2:0] ALU_CLR   = 3'b100;
    localparam [2:0] ALU_INC   = 3'b101;
    localparam [2:0] ALU_DEC   = 3'b110;

    reg  [11:0] op_a_eff;
    reg  [11:0] op_b_eff;
    reg  [1:0]  cin_eff;
    wire [11:0] alu_result;

    function [1:0] invert_trit;
        input [1:0] t;
        begin
            case (t)
                TRIT_NEG: invert_trit = TRIT_POS;
                TRIT_POS: invert_trit = TRIT_NEG;
                default:  invert_trit = TRIT_ZERO;
            endcase
        end
    endfunction

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
                for (i = 0; i < 6; i = i + 1)
                    op_b_eff[i*2 +: 2] = invert_trit(operand_b[i*2 +: 2]);
            end
            ALU_NEG_A: begin
                for (i = 0; i < 6; i = i + 1)
                    op_a_eff[i*2 +: 2] = invert_trit(acc_out[i*2 +: 2]);
                op_b_eff = {6{TRIT_ZERO}};
                cin_eff  = TRIT_ZERO;
            end
            ALU_LOAD: begin
                op_a_eff = {6{TRIT_ZERO}};
                op_b_eff = operand_b;
                cin_eff  = TRIT_ZERO;
            end
            ALU_CLR: begin
                op_a_eff = {6{TRIT_ZERO}};
                op_b_eff = {6{TRIT_ZERO}};
                cin_eff  = TRIT_ZERO;
            end
            ALU_INC: begin
                op_a_eff = acc_out;
                op_b_eff = {5{TRIT_ZERO}, TRIT_POS};
                cin_eff  = TRIT_ZERO;
            end
            ALU_DEC: begin
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

    always @(posedge clk or negedge rst_n) begin
        if (!rst_n)
            acc_out <= {6{TRIT_ZERO}};
        else if (load_acc)
            acc_out <= alu_result;
    end

    assign flag_zero     = (alu_result == {6{TRIT_ZERO}});
    assign flag_overflow = (cout != TRIT_ZERO);

    function [1:0] detect_sign;
        input [11:0] val;
        begin
            if (val[11:10] != TRIT_ZERO)      detect_sign = val[11:10];
            else if (val[9:8] != TRIT_ZERO)   detect_sign = val[9:8];
            else if (val[7:6] != TRIT_ZERO)   detect_sign = val[7:6];
            else if (val[5:4] != TRIT_ZERO)   detect_sign = val[5:4];
            else if (val[3:2] != TRIT_ZERO)   detect_sign = val[3:2];
            else                              detect_sign = val[1:0];
        end
    endfunction

    assign flag_sign = detect_sign(alu_result);

endmodule
```

### 7.3. Decodificador de Instruções e FSM (`rtl/instruction_decoder_6502t.v`)
```verilog
`timescale 1ns / 1ps

module instruction_decoder_6502t (
    input  wire        clk,
    input  wire        rst_n,
    input  wire [11:0] mem_data_in,
    output reg  [19:0] mem_addr,
    output reg  [11:0] mem_data_out,
    output reg         mem_read,
    output reg         mem_write,
    input  wire        flag_zero,
    input  wire [1:0]  flag_sign,
    input  wire        flag_overflow,
    output reg  [2:0]  alu_opcode,
    output reg         alu_load_acc,
    output reg  [11:0] alu_operand_b,
    output reg  [1:0]  alu_cin_ext,
    output reg         reg_x_load,
    output reg         reg_y_load,
    input  wire [11:0] reg_x_val,
    input  wire [11:0] reg_y_val,
    input  wire [11:0] acc_val
);

    localparam [1:0] T_NEG = 2'b00;
    localparam [1:0] T_ZER = 2'b01;
    localparam [1:0] T_POS = 2'b10;

    localparam [2:0] S_FETCH_OPCODE = 3'b000;
    localparam [2:0] S_DECODE       = 3'b001;
    localparam [2:0] S_FETCH_OPERAND= 3'b010;
    localparam [2:0] S_FETCH_ADDR_H = 3'b011;
    localparam [2:0] S_EXECUTE      = 3'b100;

    reg [2:0]  state;
    reg [19:0] pc;
    reg [11:0] ir;
    reg [11:0] operand_latched;
    reg [19:0] target_addr;

    wire [3:0] field_group = ir[11:8];
    wire [3:0] field_mode  = ir[7:4];
    wire [3:0] field_op    = ir[3:0];

    function [19:0] inc_pc_ternary;
        input [19:0] cur;
        integer k;
        reg [1:0] c, sum_d;
        begin
            c = T_POS;
            for (k = 0; k < 10; k = k + 1) begin
                case ({cur[k*2 +: 2], c})
                    {T_ZER, T_POS}: begin sum_d = T_POS; c = T_ZER; end
                    {T_POS, T_POS}: begin sum_d = T_NEG; c = T_POS; end
                    {T_NEG, T_POS}: begin sum_d = T_ZER; c = T_ZER; end
                    default:        begin sum_d = cur[k*2 +: 2]; c = T_ZER; end
                endcase
                inc_pc_ternary[k*2 +: 2] = sum_d;
            end
        end
    endfunction

    always @(posedge clk or negedge rst_n) begin
        if (!rst_n) begin
            state         <= S_FETCH_OPCODE;
            pc            <= {10{T_ZER}};
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
            alu_load_acc <= 1'b0;
            reg_x_load   <= 1'b0;
            reg_y_load   <= 1'b0;
            mem_write    <= 1'b0;

            case (state)
                S_FETCH_OPCODE: begin
                    mem_addr <= pc;
                    mem_read <= 1'b1;
                    state    <= S_DECODE;
                end

                S_DECODE: begin
                    ir       <= mem_data_in;
                    mem_read <= 1'b0;
                    pc       <= inc_pc_ternary(pc);

                    if (mem_data_in[7:4] == {T_ZER, T_POS}) begin
                        mem_addr <= inc_pc_ternary(pc);
                        mem_read <= 1'b1;
                        state    <= S_FETCH_OPERAND;
                    end else if (mem_data_in[7:4] == {T_POS, T_POS}) begin
                        mem_addr <= inc_pc_ternary(pc);
                        mem_read <= 1'b1;
                        state    <= S_FETCH_ADDR_H;
                    end else begin
                        state    <= S_EXECUTE;
                    end
                end

                S_FETCH_OPERAND: begin
                    operand_latched <= mem_data_in;
                    mem_read        <= 1'b0;
                    pc              <= inc_pc_ternary(pc);
                    state           <= S_EXECUTE;
                end

                S_FETCH_ADDR_H: begin
                    target_addr[11:0]  <= mem_data_in;
                    target_addr[19:12] <= {4{T_ZER}};
                    mem_read           <= 1'b0;
                    pc                 <= inc_pc_ternary(pc);
                    state              <= S_EXECUTE;
                end

                S_EXECUTE: begin
                    case (field_group)
                        {T_POS, T_POS}: begin
                            alu_operand_b <= (field_mode == {T_ZER, T_POS}) ? operand_latched : reg_x_val;
                            alu_cin_ext   <= T_ZER;
                            case (field_op)
                                {T_NEG, T_NEG}: alu_opcode <= 3'b000;
                                {T_NEG, T_POS}: alu_opcode <= 3'b001;
                                {T_ZER, T_POS}: alu_opcode <= 3'b010;
                                {T_POS, T_POS}: alu_opcode <= 3'b011;
                                default:        alu_opcode <= 3'b000;
                            endcase
                            alu_load_acc <= 1'b1;
                            state        <= S_FETCH_OPCODE;
                        end

                        {T_NEG, T_ZER}: begin
                            case (field_op)
                                {T_NEG, T_NEG}: reg_x_load <= 1'b1;
                                {T_NEG, T_POS}: begin
                                    alu_operand_b <= reg_x_val;
                                    alu_opcode    <= 3'b011;
                                    alu_load_acc  <= 1'b1;
                                end
                                {T_ZER, T_POS}: reg_y_load <= 1'b1;
                                default: ;
                            endcase
                            state <= S_FETCH_OPCODE;
                        end

                        {T_NEG, T_NEG}: begin
                            case (field_op)
                                {T_NEG, T_NEG}: pc <= target_addr;
                                {T_NEG, T_POS}: if (flag_zero) pc <= target_addr;
                                {T_ZER, T_NEG}: if (flag_sign == T_NEG) pc <= target_addr;
                                {T_ZER, T_POS}: if (flag_sign == T_POS) pc <= target_addr;
                                default: ;
                            endcase
                            state <= S_FETCH_OPCODE;
                        end

                        {T_ZER, T_POS}: begin
                            if (field_op == {T_POS, T_POS}) begin
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
    end

endmodule
```

### 7.4. Módulo Top-Level da CPU (`rtl/cpu_6502t_top.v`)
```verilog
`timescale 1ns / 1ps

module cpu_6502t_top (
    input  wire        clk,
    input  wire        rst_n,
    output wire [19:0] addr_bus,
    input  wire [11:0] data_in,
    output wire [11:0] data_out,
    output wire        read_en,
    output wire        write_en,
    output wire [11:0] dbg_acc,
    output wire [11:0] dbg_reg_x,
    output wire [11:0] dbg_reg_y,
    output wire        dbg_flag_z,
    output wire [1:0]  dbg_flag_s,
    output wire        dbg_flag_o
);

    localparam [1:0] T_ZER = 2'b01;

    wire [2:0]  ctrl_alu_opcode;
    wire        ctrl_alu_load_acc;
    wire [11:0] ctrl_alu_operand_b;
    wire [1:0]  ctrl_alu_cin_ext;
    wire        ctrl_reg_x_load;
    wire        ctrl_reg_y_load;
    reg  [11:0] reg_x;
    reg  [11:0] reg_y;

    wire [11:0] acc_wire;
    wire [1:0]  alu_cout;
    wire        alu_flag_zero;
    wire [1:0]  alu_flag_sign;
    wire        alu_flag_overflow;

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

    instruction_decoder_6502t u_decoder (
        .clk          (clk),
        .rst_n        (rst_n),
        .mem_data_in  (data_in),
        .mem_addr     (addr_bus),
        .mem_data_out (data_out),
        .mem_read     (read_en),
        .mem_write    (write_en),
        .flag_zero    (alu_flag_zero),
        .flag_sign    (alu_flag_sign),
        .flag_overflow(alu_flag_overflow),
        .alu_opcode   (ctrl_alu_opcode),
        .alu_load_acc (ctrl_alu_load_acc),
        .alu_operand_b(ctrl_alu_operand_b),
        .alu_cin_ext  (ctrl_alu_cin_ext),
        .reg_x_load   (ctrl_reg_x_load),
        .reg_y_load   (ctrl_reg_y_load),
        .reg_x_val    (reg_x),
        .reg_y_val    (reg_y),
        .acc_val      (acc_wire)
    );

    always @(posedge clk or negedge rst_n) begin
        if (!rst_n) begin
            reg_x <= {6{T_ZER}};
            reg_y <= {6{T_ZER}};
        end else begin
            if (ctrl_reg_x_load) reg_x <= acc_wire;
            if (ctrl_reg_y_load) reg_y <= acc_wire;
        end
    end

    assign dbg_acc    = acc_wire;
    assign dbg_reg_x  = reg_x;
    assign dbg_reg_y  = reg_y;
    assign dbg_flag_z = alu_flag_zero;
    assign dbg_flag_s = alu_flag_sign;
    assign dbg_flag_o = alu_flag_overflow;

endmodule
```

### 7.5. Decodificador Duodecimal (`rtl/trits_to_dozenal.v`)
```verilog
`timescale 1ns / 1ps

module trits_to_dozenal (
    input  wire [11:0] trits_in,
    output reg         sign,          // 0 = Positivo/Zero, 1 = Negativo
    output reg  [3:0]  d2,            // Grosas (peso 144)
    output reg  [3:0]  d1,            // Dúzias (peso 12)
    output reg  [3:0]  d0,            // Unidades (peso 1)
    output wire        is_zero
);

    localparam [1:0] TRIT_NEG  = 2'b00;
    localparam [1:0] TRIT_POS  = 2'b10;

    function signed [1:0] trit_val;
        input [1:0] t;
        begin
            case (t)
                TRIT_NEG: trit_val = -2'sd1;
                TRIT_POS: trit_val =  2'sd1;
                default:  trit_val =  2'sd0;
            endcase
        end
    endfunction

    reg signed [9:0] v_signed;
    reg        [8:0] v_abs;
    reg        [8:0] rem_144;

    always @(*) begin
        v_signed = trit_val(trits_in[1:0])   * 10'sd1   +
                   trit_val(trits_in[3:2])   * 10'sd3   +
                   trit_val(trits_in[5:4])   * 10'sd9   +
                   trit_val(trits_in[7:6])   * 10'sd27  +
                   trit_val(trits_in[9:8])   * 10'sd81  +
                   trit_val(trits_in[11:10]) * 10'sd243;

        if (v_signed < 0) begin
            sign  = 1'b1;
            v_abs = -v_signed;
        end else begin
            sign  = 1'b0;
            v_abs = v_signed[8:0];
        end

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

        d1 = rem_144 / 12;
        d0 = rem_144 % 12;
    end

    assign is_zero = (v_signed == 10'sd0);

endmodule
```

### 7.6. Controlador Multiplexado de Displays (`rtl/display_controller_4digit.v`)
```verilog
`timescale 1ns / 1ps

module display_controller_4digit (
    input  wire        clk,
    input  wire        rst_n,
    input  wire        sign,
    input  wire [3:0]  d2,
    input  wire [3:0]  d1,
    input  wire [3:0]  d0,
    output reg  [6:0]  segments,
    output reg  [3:0]  anodes
);

    reg [15:0] clk_div;
    wire refresh_tick = (clk_div == 16'd49_999);

    always @(posedge clk or negedge rst_n) begin
        if (!rst_n) clk_div <= 16'd0;
        else if (refresh_tick) clk_div <= 16'd0;
        else clk_div <= clk_div + 1'b1;
    end

    reg [1:0] digit_sel;
    always @(posedge clk or negedge rst_n) begin
        if (!rst_n) digit_sel <= 2'b00;
        else if (refresh_tick) digit_sel <= digit_sel + 1'b1;
    end

    reg [3:0] cur_digit;
    reg       cur_is_sign;
    reg       cur_blank;

    always @(*) begin
        case (digit_sel)
            2'b00: begin anodes = 4'b1110; cur_digit = d0; cur_is_sign = 1'b0; cur_blank = 1'b0; end
            2'b01: begin anodes = 4'b1101; cur_digit = d1; cur_is_sign = 1'b0; cur_blank = 1'b0; end
            2'b10: begin anodes = 4'b1011; cur_digit = d2; cur_is_sign = 1'b0; cur_blank = (d2 == 4'd0); end
            2'b11: begin anodes = 4'b0111; cur_digit = 4'd0; cur_is_sign = sign; cur_blank = ~sign; end
        endcase
    end

    always @(*) begin
        if (cur_blank)
            segments = 7'b111_1111;
        else if (cur_is_sign)
            segments = 7'b011_1111;
        else begin
            case (cur_digit)
                4'd0:    segments = 7'b100_0000;
                4'd1:    segments = 7'b111_1001;
                4'd2:    segments = 7'b010_0100;
                4'd3:    segments = 7'b011_0000;
                4'd4:    segments = 7'b001_1001;
                4'd5:    segments = 7'b001_0010;
                4'd6:    segments = 7'b000_0010;
                4'd7:    segments = 7'b111_1000;
                4'd8:    segments = 7'b000_0000;
                4'd9:    segments = 7'b001_0000;
                4'd10:   segments = 7'b000_1000; // 'A'
                4'd11:   segments = 7'b000_0011; // 'b'
                default: segments = 7'b111_1111;
            endcase
        end
    end

endmodule
```

### 7.7. Decodificador de Endereçamento Linear (`rtl/trits_to_addr.v`)
Converte vetores ternários balanceados para índice linear inteiro sem sinal (0 a $3^k - 1$), permitindo mapeamento direto em blocos de memória RAM/BRAM física:

```verilog
`timescale 1ns / 1ps

module trits_to_addr #(
    parameter NUM_TRITS  = 6,
    parameter ADDR_WIDTH = 10
) (
    input  wire [NUM_TRITS*2-1:0] trits_in,
    output wire [ADDR_WIDTH-1:0]  addr_out
);

    localparam [1:0] TRIT_NEG  = 2'b00; // -1
    localparam [1:0] TRIT_ZERO = 2'b01; //  0
    localparam [1:0] TRIT_POS  = 2'b10; // +1

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

    reg signed [15:0] addr_calc;

    always @(*) begin
        addr_calc = 16'sd0;
        if (NUM_TRITS >= 1)  addr_calc = addr_calc + trit_val(trits_in[1:0])   * 16'sd1;
        if (NUM_TRITS >= 2)  addr_calc = addr_calc + trit_val(trits_in[3:2])   * 16'sd3;
        if (NUM_TRITS >= 3)  addr_calc = addr_calc + trit_val(trits_in[5:4])   * 16'sd9;
        if (NUM_TRITS >= 4)  addr_calc = addr_calc + trit_val(trits_in[7:6])   * 16'sd27;
        if (NUM_TRITS >= 5)  addr_calc = addr_calc + trit_val(trits_in[9:8])   * 16'sd81;
        if (NUM_TRITS >= 6)  addr_calc = addr_calc + trit_val(trits_in[11:10]) * 16'sd243;
        if (NUM_TRITS >= 7)  addr_calc = addr_calc + trit_val(trits_in[13:12]) * 16'sd729;
        if (NUM_TRITS >= 8)  addr_calc = addr_calc + trit_val(trits_in[15:14]) * 16'sd2187;
        if (NUM_TRITS >= 9)  addr_calc = addr_calc + trit_val(trits_in[17:16]) * 16'sd6561;
        if (NUM_TRITS >= 10) addr_calc = addr_calc + trit_val(trits_in[19:18]) * 16'sd19683;
    end

    assign addr_out = (addr_calc >= 0) ? addr_calc[ADDR_WIDTH-1:0] : {ADDR_WIDTH{1'b0}};

endmodule
```

---

## 8. Cadeia de Ferramentas em Python

### 8.1. Montador Avançado (`asm6502t.py`)
```python
#!/usr/bin/env python3
"""
Montador para CPU Ternária 6502-T
Suporta pseudo-instruções (CLR, CMP, PUSH, POP), literais duodecimais (0z...)
e avaliação de expressões aritméticas.
"""

import sys
import re
import ast
import operator
from typing import List, Tuple, Dict

TRIT_MAP = {-1: "00", 0: "01", 1: "10"}

INSTRUCTION_SET = {
    "ADD_IMM": (( 1,  1), ( 0,  1), (-1, -1)),
    "ADD_ABS": (( 1,  1), ( 1,  1), (-1, -1)),
    "SUB_IMM": (( 1,  1), ( 0,  1), (-1,  1)),
    "SUB_ABS": (( 1,  1), ( 1,  1), (-1,  1)),
    "LDA_IMM": (( 1,  1), ( 0,  1), ( 1,  1)),
    "LDA_ABS": (( 1,  1), ( 1,  1), ( 1,  1)),
    "NEG":     (( 1,  1), (-1, -1), ( 0,  1)),
    "INC":     (( 1,  1), (-1, -1), ( 1,  0)),
    "DEC":     (( 1,  1), (-1, -1), ( 0, -1)),
    "TAX":     ((-1,  0), (-1, -1), (-1, -1)),
    "TXA":     ((-1,  0), (-1, -1), (-1,  1)),
    "TAY":     ((-1,  0), (-1, -1), ( 0,  1)),
    "TYA":     ((-1,  0), (-1, -1), ( 0, -1)),
    "JMP":     ((-1, -1), ( 1,  1), (-1, -1)),
    "BEQ":     ((-1, -1), ( 1,  1), (-1,  1)),
    "BMI":     ((-1, -1), ( 1,  1), ( 0, -1)),
    "BPL":     ((-1, -1), ( 1,  1), ( 0,  1)),
    "STA":     (( 0,  1), ( 1,  1), ( 1,  1)),
}

def int_to_balanced_ternary(val: int, num_trits: int = 6) -> List[int]:
    max_val = (3**num_trits - 1) // 2
    if val > max_val or val < -max_val:
        raise ValueError(f"Valor {val} excede capacidade de {num_trits} trits.")
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
    return trits

def trits_to_verilog_bin(trits: List[int]) -> str:
    return "12'b" + "".join(TRIT_MAP[t] for t in reversed(trits))

def encode_opcode(group: Tuple[int, int], mode: Tuple[int, int], op: Tuple[int, int]) -> str:
    return trits_to_verilog_bin([op[0], op[1], mode[0], mode[1], group[0], group[1]])

class SafeEvaluator(ast.NodeVisitor):
    OPS = {ast.Add: operator.add, ast.Sub: operator.sub, ast.Mult: operator.mul,
           ast.FloorDiv: operator.floordiv, ast.Div: operator.floordiv, ast.USub: operator.neg}
    def __init__(self, syms): self.syms = syms
    def visit_BinOp(self, n): return self.OPS[type(n.op)](self.visit(n.left), self.visit(n.right))
    def visit_UnaryOp(self, n): return self.OPS[type(n.op)](self.visit(n.operand))
    def visit_Constant(self, n): return int(n.value)
    def visit_Name(self, n): return self.syms[n.id]

class Assembler6502T:
    def __init__(self, stack_base: int = 63):
        self.stack_base = stack_base
        self.labels: Dict[str, int] = {}
        self.expanded_code = []
        self.machine_code: Dict[int, Tuple[str, str]] = {}

    def _normalize(self, expr: str) -> str:
        expr = re.sub(r"0[zZ]([0-9a-bA-B]+)", lambda m: str(int(m.group(1), 12)), expr)
        return re.sub(r"\$([0-9a-fA-F]+)", r"0x\1", expr)

    def evaluate(self, expr: str) -> int:
        return SafeEvaluator(self.labels).visit(ast.parse(self._normalize(expr.strip()), mode="eval").body)

    def compile(self, source: str):
        stack_depth = 0
        for line in source.splitlines():
            clean = re.sub(r"(//|;).*", "", line).strip()
            if not clean: continue
            cur_label = ""
            if ":" in clean:
                lpart, _, ipart = clean.partition(":")
                cur_label = lpart.strip()
                clean = ipart.strip()
                if not clean:
                    self.expanded_code.append((cur_label, "", ""))
                    continue
            tokens = clean.replace(",", " ").split()
            m = tokens[0].upper()
            op = tokens[1] if len(tokens) > 1 else ""

            if m == "CLR":
                self.expanded_code.append((cur_label, "LDA", "#0"))
            elif m == "CMP":
                self.expanded_code.append((cur_label, "TAX", ""))
                self.expanded_code.append(("", "SUB", op))
            elif m == "PUSH":
                self.expanded_code.append((cur_label, "STA", str(self.stack_base - stack_depth)))
                stack_depth += 1
            elif m == "POP":
                stack_depth -= 1
                self.expanded_code.append((cur_label, "LDA", str(self.stack_base - stack_depth)))
            else:
                self.expanded_code.append((cur_label, m, op))

        cur_addr = 0
        for label, m, op in self.expanded_code:
            if label: self.labels[label] = cur_addr
            if m: cur_addr += 2 if op else 1

        cur_addr = 0
        for _, m, op in self.expanded_code:
            if not m: continue
            imm = op.startswith("#")
            clean_op = op[1:] if imm else op

            if m in ["LDA", "ADD", "SUB"]:
                val = self.evaluate(clean_op)
                mode = "IMM" if imm else "ABS"
                self.machine_code[cur_addr] = (encode_opcode(*INSTRUCTION_SET[f"{m}_{mode}"]), f"{m} {op}")
                self.machine_code[cur_addr+1] = (trits_to_verilog_bin(int_to_balanced_ternary(val, 6)), f"Val: {val}")
                cur_addr += 2
            elif m in ["STA", "JMP", "BEQ", "BMI", "BPL"]:
                val = self.evaluate(clean_op)
                self.machine_code[cur_addr] = (encode_opcode(*INSTRUCTION_SET[m]), f"{m} ${val}")
                self.machine_code[cur_addr+1] = (trits_to_verilog_bin(int_to_balanced_ternary(val, 6)), f"Addr: {val}")
                cur_addr += 2
            elif m in ["TAX", "TXA", "TAY", "TYA", "NEG", "INC", "DEC"]:
                self.machine_code[cur_addr] = (encode_opcode(*INSTRUCTION_SET[m]), m)
                cur_addr += 1

    def export_verilog_initial(self, array_name: str = "ram_block") -> str:
        lines = ["initial begin"]
        for addr in sorted(self.machine_code.keys()):
            b, c = self.machine_code[addr]
            lines.append(f"    {array_name}[{addr:2d}] = {b}; // {c}")
        lines.append("end")
        return "\n".join(lines)
```

### 8.2. Desassemblador Duodecimal (`disasm6502t.py`)
```python
#!/usr/bin/env python3
"""
Desassemblador para CPU Ternária 6502-T
Reverte vetores de 12 bits para mnemônicos e bases decimal/duodecimal.
"""

import re
from typing import Dict, Tuple, List

BIN_TO_TRIT = {"00": -1, "01": 0, "10": 1}

REVERSE_ISA = {
    (( 1,  1), ( 0,  1), (-1, -1)): ("ADD", "IMM"),
    (( 1,  1), ( 1,  1), (-1, -1)): ("ADD", "ABS"),
    (( 1,  1), ( 0,  1), (-1,  1)): ("SUB", "IMM"),
    (( 1,  1), ( 1,  1), (-1,  1)): ("SUB", "ABS"),
    (( 1,  1), ( 0,  1), ( 1,  1)): ("LDA", "IMM"),
    (( 1,  1), ( 1,  1), ( 1,  1)): ("LDA", "ABS"),
    (( 1,  1), (-1, -1), ( 0,  1)): ("NEG", "NONE"),
    (( 1,  1), (-1, -1), ( 1,  0)): ("INC", "NONE"),
    (( 1,  1), (-1, -1), ( 0, -1)): ("DEC", "NONE"),
    ((-1,  0), (-1, -1), (-1, -1)): ("TAX", "NONE"),
    ((-1,  0), (-1, -1), (-1,  1)): ("TXA", "NONE"),
    ((-1,  0), (-1, -1), ( 0,  1)): ("TAY", "NONE"),
    ((-1,  0), (-1, -1), ( 0, -1)): ("TYA", "NONE"),
    ((-1, -1), ( 1,  1), (-1, -1)): ("JMP", "ABS"),
    ((-1, -1), ( 1,  1), (-1,  1)): ("BEQ", "ABS"),
    ((-1, -1), ( 1,  1), ( 0, -1)): ("BMI", "ABS"),
    ((-1, -1), ( 1,  1), ( 0,  1)): ("BPL", "ABS"),
    (( 0,  1), ( 1,  1), ( 1,  1)): ("STA", "ABS"),
}

def bin12_to_trits(b: str) -> List[int]:
    b = b.replace("12'b", "").replace("_", "").strip()
    return [BIN_TO_TRIT[b[i:i+2]] for i in range(10, -1, -2)]

def trits_to_int(t: List[int]) -> int:
    return sum(v * (3**i) for i, v in enumerate(t))

def int_to_dozenal(val: int) -> str:
    if val == 0: return "0z0"
    s = "-" if val < 0 else "+"
    v = abs(val)
    chars = "0123456789AB"
    d = []
    while v > 0:
        d.append(chars[v % 12])
        v //= 12
    return f"{s}0z" + "".join(reversed(d))

class Disassembler6502T:
    def __init__(self):
        self.memory: Dict[int, str] = {}
        self.labels: Dict[int, str] = {}
        self.instructions = {}

    def load(self, text: str):
        for line in text.splitlines():
            line = re.sub(r"//.*", "", line).strip()
            m = re.search(r"\[\s*(\d+)\s*\]\s*=\s*(12'b[01]{12}|[01]{12})", line)
            if m:
                self.memory[int(m.group(1))] = m.group(2).replace("12'b", "")

    def disassemble(self, text: str) -> str:
        self.load(text)
        pc = 0
        max_a = max(self.memory.keys()) if self.memory else 0
        while pc <= max_a:
            if pc not in self.memory: pc += 1; continue
            trits = bin12_to_trits(self.memory[pc])
            key = ((trits[4], trits[5]), (trits[2], trits[3]), (trits[0], trits[1]))
            if key in REVERSE_ISA:
                m, op_t = REVERSE_ISA[key]
                if op_t in ["IMM", "ABS"]:
                    val = trits_to_int(bin12_to_trits(self.memory[pc+1]))
                    self.instructions[pc] = (m, op_t, val)
                    if op_t == "ABS": self.labels[val] = f"L_{val:02d}"
                    pc += 2
                else:
                    self.instructions[pc] = (m, op_t, None)
                    pc += 1
            else: pc += 1

        out = []
        pc = 0
        while pc <= max_a:
            if pc in self.labels: out.append(f"\n{self.labels[pc]}:")
            if pc in self.instructions:
                m, op_t, v = self.instructions[pc]
                if op_t == "IMM":
                    out.append(f"[{pc:02d}]  {m:<4} #{v:<4}  ; B12: {int_to_dozenal(v)}")
                    pc += 2
                elif op_t == "ABS":
                    out.append(f"[{pc:02d}]  {m:<4} ${v:<4}  ; B12: {int_to_dozenal(v)}")
                    pc += 2
                else:
                    out.append(f"[{pc:02d}]  {m}")
                    pc += 1
            else: pc += 1
        return "\n".join(out)
```

### 8.3. Emulador Ciclo a Ciclo (`emu6502t.py`)
```python
#!/usr/bin/env python3
"""
Emulador Ciclo a Ciclo da CPU Ternária 6502-T
Suporta carga via CLI de arquivos .mem, .hex e arquivos Verilog.
"""

import sys
import os
import re
import argparse
from typing import List, Dict

TRIT_CHARS = {-1: "-", 0: "0", 1: "+"}

def int_to_trits(val: int, length: int = 6) -> List[int]:
    trits = []
    n = val
    for _ in range(length):
        rem = n % 3
        if rem == 2:   trits.append(-1); n = (n // 3) + 1
        elif rem == 1: trits.append(1);  n = n // 3
        else:          trits.append(0);  n = n // 3
    return trits

def trits_to_int(trits: List[int]) -> int:
    return sum(t * (3**i) for i, t in enumerate(trits))

def trits_to_str(trits: List[int]) -> str:
    return "".join(TRIT_CHARS[t] for t in reversed(trits))

def int_to_dozenal(val: int) -> str:
    if val == 0: return " 0z000"
    s = "-" if val < 0 else "+"
    v = abs(val)
    chars = "0123456789AB"
    return f"{s}0z{chars[(v//144)%12]}{chars[(v//12)%12]}{chars[v%12]}"

def bin12_to_trits(bin_str: str) -> List[int]:
    bin_str = bin_str.replace("12'b", "").replace("_", "").strip()
    m = {"00": -1, "01": 0, "10": 1}
    return [m[bin_str[i:i+2]] for i in range(10, -1, -2)]

class CPU6502T:
    S_FETCH_OPCODE, S_DECODE, S_FETCH_OPERAND, S_FETCH_ADDR_H, S_EXECUTE = range(5)
    STATE_NAMES = {0: "FETCH_OPCODE", 1: "DECODE", 2: "FETCH_OPERAND", 3: "FETCH_ADDR", 4: "EXECUTE"}

    def __init__(self, mem_size: int = 64):
        self.mem_size = mem_size
        self.memory = [0] * mem_size
        self.reset()

    def reset(self):
        self.pc = self.acc = self.reg_x = self.reg_y = self.ir = 0
        self.operand_latch = self.target_addr = 0
        self.state = self.S_FETCH_OPCODE
        self.cycle_count = 0
        self.flag_zero = True
        self.flag_sign = 0
        self.flag_overflow = False
        self.addr_bus = self.data_out = 0
        self.read_en = self.write_en = False

    def alu_add(self, a_val: int, b_val: int, cin: int = 0) -> int:
        s_int = a_val + b_val + cin
        cout = 0
        if s_int > 364:    cout = 1;  res = s_int - 729
        elif s_int < -364: cout = -1; res = s_int + 729
        else:              cout = 0;  res = s_int
        self.flag_overflow = (cout != 0)
        self.flag_zero = (res == 0)
        self.flag_sign = 0 if res == 0 else (1 if res > 0 else -1)
        return res

    def step_cycle(self):
        self.cycle_count += 1
        self.read_en = self.write_en = False

        if self.state == self.S_FETCH_OPCODE:
            self.addr_bus = self.pc
            self.read_en = True
            self.state = self.S_DECODE

        elif self.state == self.S_DECODE:
            self.ir = self.memory[self.addr_bus % self.mem_size]
            self.pc = (self.pc + 1) % 59049
            tr = int_to_trits(self.ir, 6)
            mode = (tr[2], tr[3])
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
            tr = int_to_trits(self.ir, 6)
            group, mode, op = (tr[4], tr[5]), (tr[2], tr[3]), (tr[0], tr[1])
            if group == (1, 1):
                op_b = self.operand_latch if mode == (0, 1) else self.reg_x
                if op == (-1, -1):    self.acc = self.alu_add(self.acc, op_b)
                elif op == (-1, 1):   self.acc = self.alu_add(self.acc, -op_b)
                elif op == (0, 1):    self.acc = self.alu_add(0, -self.acc)
                elif op == (1, 1):    self.acc = self.alu_add(0, op_b)
                elif op == (1, 0):    self.acc = self.alu_add(self.acc, 1)
                elif op == (0, -1):   self.acc = self.alu_add(self.acc, -1)
            elif group == (-1, 0):
                if op == (-1, -1):    self.reg_x = self.acc
                elif op == (-1, 1):
                    self.acc = self.reg_x
                    self.flag_zero = (self.acc == 0)
                    self.flag_sign = 0 if self.acc == 0 else (1 if self.acc > 0 else -1)
                elif op == (0, 1):    self.reg_y = self.acc
            elif group == (-1, -1):
                if op == (-1, -1):    self.pc = self.target_addr
                elif op == (-1, 1) and self.flag_zero: self.pc = self.target_addr
                elif op == (0, -1) and self.flag_sign == -1: self.pc = self.target_addr
                elif op == (0, 1) and self.flag_sign == 1: self.pc = self.target_addr
            elif group == (0, 1) and op == (1, 1):
                self.addr_bus = self.target_addr
                self.data_out = self.acc
                self.memory[self.target_addr % self.mem_size] = self.acc
                self.write_en = True
            self.state = self.S_FETCH_OPCODE

    def step_instruction(self):
        while True:
            self.step_cycle()
            if self.state == self.S_FETCH_OPCODE: break

    def load_file(self, filepath: str) -> int:
        with open(filepath, "r", encoding="utf-8") as f:
            lines = f.readlines()
        count = cur = 0
        for line in lines:
            clean = re.sub(r"(//|;|#).*", "", line).strip()
            if not clean: continue
            m = re.search(r"\[\s*(\d+)\s*\]\s*=\s*(12'b[01]{12}|[01]{12})", clean)
            if m:
                a, b = int(m.group(1)), m.group(2).replace("12'b", "")
                if a < self.mem_size:
                    self.memory[a] = trits_to_int(bin12_to_trits(b))
                    count += 1
                continue
            toks = [t for t in clean.split() if re.fullmatch(r"[01]{12}", t)]
            for t in toks:
                if cur < self.mem_size:
                    self.memory[cur] = trits_to_int(bin12_to_trits(t))
                    cur += 1
                    count += 1
        return count
```

### 8.4. Suite de Testes Automatizada (`test_6502t_suite.py`)
```python
#!/usr/bin/env python3
"""
Test Suite Integrado para CPU Ternária 6502-T
Valida montagem, execução ciclo a ciclo, flags e memória.
"""

import sys
from asm6502t import Assembler6502T
from emu6502t import CPU6502T, bin12_to_trits, trits_to_int

def run_tests():
    tests = [
        {
            "name": "Aritmética Balanceada em Base 12",
            "src": "START: LDA #0z011\n ADD #0z0BB\nHLT: JMP HLT",
            "acc": 156, "zero": False, "sign": 1
        },
        {
            "name": "Negação Unária e Flag Sign Negativa",
            "src": "START: LDA #40\n NEG\n SUB #10\nHLT: JMP HLT",
            "acc": -50, "zero": False, "sign": -1
        },
        {
            "name": "Branch Condicional BEQ",
            "src": "START: LDA #20\n SUB #20\n BEQ OK\n LDA #99\nHLT: JMP HLT\nOK: LDA #42\n JMP HLT",
            "acc": 42, "zero": False, "sign": 1
        },
        {
            "name": "Pilha e Pseudo-Instruções (PUSH, POP, CMP)",
            "src": "START: LDA #100\n PUSH\n LDA #50\n CMP #50\n BEQ POP_VAL\n JMP HLT\nPOP_VAL: POP\nHLT: JMP HLT",
            "acc": 100, "zero": False, "sign": 1
        }
    ]

    passed = 0
    for idx, t in enumerate(tests, 1):
        asm = Assembler6502T()
        asm.compile(t["src"])
        cpu = CPU6502T(mem_size=64)
        for addr, (bin_str, _) in asm.machine_code.items():
            cpu.memory[addr] = trits_to_int(bin12_to_trits(bin_str))

        for _ in range(300):
            if cpu.state == CPU6502T.S_FETCH_OPCODE and cpu.memory[cpu.pc] == -292:
                if cpu.memory[(cpu.pc + 1) % 64] == cpu.pc: break
            cpu.step_cycle()

        fail = False
        if "acc" in t and cpu.acc != t["acc"]: fail = True
        if "zero" in t and cpu.flag_zero != t["zero"]: fail = True
        if "sign" in t and cpu.flag_sign != t["sign"]: fail = True

        if not fail:
            passed += 1
            print(f"[{idx:02d}/{len(tests):02d}] PASS: {t['name']}")
        else:
            print(f"[{idx:02d}/{len(tests):02d}] FAIL: {t['name']} (Acc: {cpu.acc}, Z: {cpu.flag_zero}, S: {cpu.flag_sign})")

    print(f"\nResultado: {passed}/{len(tests)} testes passaram.")
    return passed == len(tests)

if __name__ == "__main__":
    sys.exit(0 if run_tests() else 1)
```

### 8.5. Compilador VAMOS-T (`tools/vamos6502t.py`)
O compilador VAMOS-T traduz a linguagem de alto nível VAMOS-LANG ([facrf/VAMOSLANG](https://github.com/facrf/VAMOSLANG)) diretamente para instruções assembly do 6502-T.

* **Layout de Memória Padrão:**
  * Segmento de Código: `$000` a `$179` ($0$ a $179$).
  * Variáveis na RAM: `$180` a `$299` ($180$ a $299$).
  * Porta MMIO TTY Virtual: `$300` / `0z210`.
* **Funcionalidades:**
  * Suporte a literais em Base 12 (`0z...`) e inteiros decimais.
  * Alocação automática de variáveis escalares na RAM.
  * Estruturas de controle `se / senao` (mapeadas para `BEQ`, `BNE`, `BMI`, `BPL`) e `para` (laços iterativos).
  * Streaming de strings via MMIO na porta 300 através da chamada nativa `escrever(...)`.
  * Emissão estruturada em JSON com trace ciclo a ciclo para integração com a interface web.

### 8.6. Interface Web HTTP, Emulação Visual e Telemetria (`web/main.go`)
Frontend web moderno desenvolvido em Go puro (sem dependências externas) estilizado sob a paleta **Tokyo Night**:
* **Servidor HTTP:** Escuta na porta `27005`, servindo uma SPA autossuficiente.
* **Quatro Ambientes Dedicados:**
  1. *Laboratório VAMOS-T:* Editor de código, reprodução animada passo a passo, console CRT TTY com fósforo verde, medidor LED de 6 trits físicos e rastreamento síncrono do Program Counter.
  2. *Manual & Didática:* Guia de conversão duodecimal, tabela de pesos e cards didáticos interativos com carregamento em um clique.
  3. *Desafios Interativos (Trit Challenges):* Quizzes práticos com validação automática das flags e variáveis.
  4. *Terminal & Telemetria:* Sessão interativa com a CPU e o monitor de memória.
* **Segurança e Cloudflare Readiness:**
  * Injeção de headers defensivos (`Content-Security-Policy`, `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`).
  * Rate limiter ativo em memória (60 req/min por IP) compatível com cabeçalhos `CF-Connecting-IP` e `X-Forwarded-For`.
  * Compartilhamento de snippets de código via URL hash (`#code=base64...`).

---

## 9. Infraestrutura de Hardware e Síntese FPGA

### 9.1. Arquivo de Restrições Físicas (`constr/basys3_6502t.xdc`)
```tcl
## CLOCK PRINCIPAL (100 MHz Onboard)
set_property PACKAGE_PIN W5 [get_ports clk]
set_property IOSTANDARD LVCMOS33 [get_ports clk]
create_clock -add -name sys_clk_pin -period 10.00 -waveform {0 5} [get_ports clk]

## BOTÕES DE ENTRADA
set_property PACKAGE_PIN U18 [get_ports btnC]
set_property IOSTANDARD LVCMOS33 [get_ports btnC]
set_property PACKAGE_PIN T18 [get_ports btnU]
set_property IOSTANDARD LVCMOS33 [get_ports btnU]

## CHAVES SELETORAS
set_property PACKAGE_PIN V17 [get_ports {sw[0]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[0]}]
set_property PACKAGE_PIN V16 [get_ports {sw[1]}]
set_property IOSTANDARD LVCMOS33 [get_ports {sw[1]}]

## LEDS DE STATUS
set_property PACKAGE_PIN U16 [get_ports {led[0]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[0]}]
set_property PACKAGE_PIN E19 [get_ports {led[1]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[1]}]
set_property PACKAGE_PIN U19 [get_ports {led[2]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[2]}]
set_property PACKAGE_PIN V19 [get_ports {led[3]}]
set_property IOSTANDARD LVCMOS33 [get_ports {led[3]}]

## DISPLAYS DE 7 SEGMENTOS
set_property PACKAGE_PIN W7 [get_ports {seg[0]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[0]}]
set_property PACKAGE_PIN W6 [get_ports {seg[1]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[1]}]
set_property PACKAGE_PIN U8 [get_ports {seg[2]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[2]}]
set_property PACKAGE_PIN V8 [get_ports {seg[3]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[3]}]
set_property PACKAGE_PIN U5 [get_ports {seg[4]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[4]}]
set_property PACKAGE_PIN V5 [get_ports {seg[5]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[5]}]
set_property PACKAGE_PIN U7 [get_ports {seg[6]}]
set_property IOSTANDARD LVCMOS33 [get_ports {seg[6]}]

## ANODOS
set_property PACKAGE_PIN U2 [get_ports {an[0]}]
set_property IOSTANDARD LVCMOS33 [get_ports {an[0]}]
set_property PACKAGE_PIN U4 [get_ports {an[1]}]
set_property IOSTANDARD LVCMOS33 [get_ports {an[1]}]
set_property PACKAGE_PIN V4 [get_ports {an[2]}]
set_property IOSTANDARD LVCMOS33 [get_ports {an[2]}]
set_property PACKAGE_PIN W4 [get_ports {an[3]}]
set_property IOSTANDARD LVCMOS33 [get_ports {an[3]}]

set_property CONFIG_VOLTAGE 3.3 [current_design]
set_property CFGBVS VCCO [current_design]
```

### 9.2. Script TCL de Síntese em Lote (`scripts/build_bitstream.tcl`)
```tcl
set PART      "xc7a35tcpg236-1"
set TOP_MOD   "fpga_test_top"
set OUTPUT_DIR "build"

file mkdir $OUTPUT_DIR
read_verilog [ glob rtl/*.v ]
read_xdc constr/basys3_6502t.xdc

synth_design -top $TOP_MOD -part $PART -flatten_hierarchy rebuilt
write_checkpoint -force $OUTPUT_DIR/post_synth.dcp
report_utilization -file $OUTPUT_DIR/post_synth_util.rpt

opt_design
place_design
phys_opt_design
route_design
write_checkpoint -force $OUTPUT_DIR/post_route.dcp
report_timing_summary -file $OUTPUT_DIR/post_route_timing.rpt
write_bitstream -force $OUTPUT_DIR/${TOP_MOD}.bit
exit
```

### 9.3. Makefile Unificado (`Makefile`)
```makefile
PYTHON     := python3
VIVADO     := vivado
BUILD_DIR  := build
RTL_SRCS   := $(wildcard rtl/*.v)
BITSTREAM  := $(BUILD_DIR)/fpga_test_top.bit

.PHONY: all test asm sim bitstream clean

all: test asm

test:
	@$(PYTHON) test_6502t_suite.py

asm: $(BUILD_DIR)/program.mem

$(BUILD_DIR)/program.mem: src_asm/program.s | $(BUILD_DIR)
	@$(PYTHON) asm6502t.py $< -o $@

bitstream: $(BITSTREAM)

$(BITSTREAM): $(RTL_SRCS) constr/basys3_6502t.xdc scripts/build_bitstream.tcl | $(BUILD_DIR)
	@$(VIVADO) -mode batch -nojournal -nolog -source scripts/build_bitstream.tcl

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

clean:
	@rm -rf $(BUILD_DIR) *.log *.jou .Xil
```

---

## 10. Referências Bibliográficas e Materiais Acadêmicos

1. **BRUSENTSOV, N. P. et al.** *Small Digital Computing Machine "Setun"*. Moscou: Moscow State University Press, 1962.
2. **BRUSENTSOV, N. P.** "Ternary Principle: In Search of Natural Simplicity". *Informatika i Eyo Primeneniya* (Informatics and Its Applications), v. 2, n. 2, p. 104–112, 2008.
3. **CAUCHY, A.-L.** "Sur les signes d'inégalité et sur la notation abrégée des nombres". *Comptes Rendus de l'Académie des Sciences*, v. 11, p. 789–798, Paris, 1840.
4. **EULER, L.** *Introductio in analysin infinitorum*. Tomus Primus. Lausanne: Marc-Michel Bousquet & Co., 1748.
5. **GAUSS, C. F.** *Disquisitiones Arithmeticae*. Leipzig: Gerh. Fleischer, Jun., 1801.
6. **HURST, S. L.** "Multiple-Valued Logic: Its Status and Its Future". *IEEE Transactions on Computers*, v. C-33, n. 12, p. 1160–1179, dez. 1984. DOI: 10.1109/TC.1984.1676392.
7. **KNUTH, D. E.** *The Art of Computer Programming, Volume 2: Seminumerical Algorithms*. 3. ed. Boston: Addison-Wesley, 1997. Cap. 4.1: "Positional Number Systems", p. 192–212.
8. **SHANNON, C. E.** "A Mathematical Theory of Communication". *Bell System Technical Journal*, v. 27, p. 379–423, 623–656, 1948.
9. **VASILYEV, A. S.; SARKISOV, V. G.** "Design of Multi-Valued Current-Mode and Voltage-Mode Circuits Using BJT Technology". *IEEE International Symposium on Multiple-Valued Logic (ISMVL)*, 1999.
10. **VISUAL6502 TEAM.** *Visual 6502: Transistor-level simulation and reverse engineering of the MOS 6502*. 2010. Disponível em: `<http://visual6502.org/>`.