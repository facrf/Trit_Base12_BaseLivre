O projeto **TRIT + base 12 + chip** une a teoria da computação ternária balanceada à matemática duodecimal e à simplificação de arquiteturas clássicas de silício. Abaixo está a estruturação conceitual, física e arquitetural para viabilizar o projeto.

---

## 1. O Conceito do TRIT e o BJT de Três Estágios

O **trit** (*ternary digit*) é a unidade fundamental de informação ternária, assumindo três estados: $\{-1, 0, +1\}$ (ou $\{-, 0, +\}$ na notação balanceada).

Historicamente, essa abordagem foi validada pelo computador soviético **Setun** (1958, Universidade Estatal de Moscou), que utilizava lógica ternária balanceada via amplificadores magnéticos de ferrite (núcleos de saturação magnética com polaridade positiva, neutra e negativa).

### Implementação Elétrica com BJT ($\pm 5\text{ V}$)

A implementação física com Transistores de Junção Bipolar (BJT) opera em três trilhos de alimentação simétricos: $+5\text{ V}$, $0\text{ V}$ (GND/referência) e $-5\text{ V}$.

```
                 +5V (+1 Lógico)
                    │
                 ┌──┴──┐
                 │ NPN │ (Conduz quando V_in > V_th)
                 └──┬──┘
                    ├────── V_out (Nível do Trit)
                 ┌──┴──┐
                 │ PNP │ (Conduz quando V_in < -V_th)
                 └──┬──┘
                    │
                 -5V (-1 Lógico)
                    
  (Em V_in = 0V, ambos os transistores cortam e resistores
   de pull-to-ground forçam V_out para o terra: 0 Lógico)

```

* **Estado $+1$ ($+5\text{ V}$):** O transistor NPN satura, conectando a saída ao trilho superior de $+5\text{ V}$.
* **Estado $0$ ($0\text{ V}$):** Ambos os transistores (NPN e PNP) entram em **corte**. Um divisor de tensão simétrico de alta precisão ou resistores de *pull-to-ground* garantem que a saída permaneça em $0\text{ V}$ estável sem flutuações de alta impedância.
* **Estado $-1$ ($-5\text{ V}$):** O transistor PNP satura, conectando a saída ao trilho inferior de $-5\text{ V}$.

### Vantagens Aritméticas da Lógica Balanceada

1. **Sinal Intrínseco:** Dispensa circuitos dedicados para bit de sinal (como complemento de dois). O sinal do número é determinado unicamente pelo trit não-nulo mais significativo.
2. **Negação a Custo Zero:** A operação de inversão de sinal ($A \to -A$) exige apenas a inversão física dos níveis de tensão (trocar $+5\text{ V}$ por $-5\text{ V}$ e vice-versa), sem necessidade de somar $+1$ como no complemento de dois binário.
3. **Subtração como Adição:** Subtrair $A - B$ reduz-se a somar $A + (-B)$ diretamente na mesma unidade lógica.
4. **Arredondamento por Truncamento:** O truncamento de dígitos menos significativos resulta naturalmente em arredondamento simétrico (para o inteiro mais próximo), eliminando o viés numérico típico de sistemas binários truncados.

---

## 2. A Constante de Euler ($e$) e a Eficiência Radix

A escolha da base 3 não é arbitrária; ela é o inteiro matematicamente mais eficiente do universo físico para processamento de informação.

### Formulação Matemática

A complexidade de hardware de um sistema digital é expressa pelo **Custo de Radix** ($E$), definido pelo produto da base ($R$) pela quantidade de dígitos ($d$) necessários para representar uma faixa numérica $N$:

$$N = R^d \implies d = \frac{\ln N}{\ln R}$$

$$E(R) = R \cdot d = R \cdot \frac{\ln N}{\ln R} = \ln N \cdot \left( \frac{R}{\ln R} \right)$$

Para encontrar a base economicamente ótima que minimiza o número de linhas de circuito, portas lógicas e conexões ativas, calcula-se a derivada de $f(R) = \frac{R}{\ln R}$ em relação a $R$ e iguala-se a zero:

$$f'(R) = \frac{\ln R - 1}{(\ln R)^2} = 0 \implies \ln R = 1 \implies R = e \approx 2{,}71828$$

### Comparação de Eficiência

| Base ($R$) | Nome | $f(R) = \frac{R}{\ln R}$ | Eficiência Relativa |
| --- | --- | --- | --- |
| **2** | Binário | $\frac{2}{\ln 2} \approx 2{,}885$ | $94{,}2\%$ |
| **3** | **Ternário** | **$\frac{3}{\ln 3} \approx 2{,}731$** | **$99{,}5\%$** |
| **$e$** | Ideal Matemático | $\frac{e}{\ln e} \approx 2{,}718$ | $100\%$ |
| **4** | Quaternário | $\frac{4}{\ln 4} \approx 2{,}885$ | $94{,}2\%$ |
| **10** | Decimal | $\frac{10}{\ln 10} \approx 4{,}343$ | $62{,}6\%$ |
| **12** | Duodecimal | $\frac{12}{\ln 12} \approx 4{,}829$ | $56{,}3\%$ |

A **base 3 é o inteiro mais próximo do número de Euler**. Isso significa que, do ponto de vista de densidade de informação por transistor e contagem de condutores de barramento, um sistema ternário consome menos silício/componentes para representar a mesma capacidade de cálculo que um sistema binário equivalente.

---

## 3. Conceito da Base 12 (Duodecimal) no Sistema

Enquanto a base 3 é a camada física ótima de hardware (nível de silício e comutação), a **base 12** atua idealmente como a camada de abstração de dados, endereçamento e aritmética de alta densidade para o desenvolvedor ou arquitetura da ISA.

### Vantagens em Relação ao Binário/Decimal

* **Alta Divisibilidade:** O número 12 possui quatro divisores próprios não triviais: $\{2, 3, 4, 6\}$ (o número 10 tem apenas $\{2, 5\}$; o número 2 tem apenas ele próprio).
* **Frações Limpas:** Frações operacionais comuns em engenharia e física ($\frac{1}{2}, \frac{1}{3}, \frac{1}{4}, \frac{1}{6}$) tornam-se dízimas finitas exatas na base 12:
* $\frac{1}{2} = 0{,}6_{12}$
* $\frac{1}{3} = 0{,}4_{12}$
* $\frac{1}{4} = 0{,}3_{12}$
* $\frac{1}{6} = 0{,}2_{12}$


* **Casamento de Barramento:** Um bloco de dados pode ser facilmente fracionado em metades, terços e quartos sem restos ou perdas de precisão em ponto fixo.

### Desvantagens e Gargalos

* **Mapeamento Trit $\leftrightarrow$ Dozenal:** $3^n$ não cai diretamente numa potência de 12 ($3^1=3, 3^2=9, 3^3=27, 3^4=81$). Para representar 1 dígito base 12 (0 a 11), são necessários **3 trits balanceados** (capacidade de $3^3 = 27$ estados, cobrindo a faixa de $-13$ a $+13$). Há uma folga de estados redundantes que precisa de tabelas de decodificação de instruções (lookup ou máscara de hardware).
* **Complexidade de Portas Lógicas:** Circuitos que lidam diretamente com 12 símbolos exigem maior profundidade lógica se não forem decompostos em trits primitivos na ALU.
* **Incompatibilidade de I/O:** Toda comunicação serial/paralela com telas, discos rígidos e redes legadas exige conversores de nível (transceivers) e transcodificadores de base.

---

## 4. Fork de Silício para Testes: MOS 6502 Ternário ("6502-T")

O processador mais viável para um fork experimental é o **MOS Technology 6502**.

### Por que o 6502?

1. **Contagem Mínima de Componentes:** O 6502 original contém apenas ~3.510 transistores NMOS (contra mais de 8.500 no Z80 e ~4.000 no Motorola 6800). Em um protótipo com BJTs discretos ou síntese em FPGA/ASIC, o esforço de reengenharia é reduzido pela metade.
2. **Topologia Inteiramente Documentada:** O projeto *Visual6502* reverteu o die do silício em nível de transistor, fornecendo netlists abertas e diagramas exatos de cada porta de passagem e registrador dinâmico.
3. **Conjunto de Registradores Compacto:** Apenas A (Acumulador), X, Y, SP (Stack Pointer), PC (Program Counter) e P (Status Register).

### Especificações do Fork "6502-T"

| Módulo Original (Binário) | Módulo Modificado 6502-T (Ternário) |
| --- | --- |
| **Tamanho da Palavra** | 8 bits (256 estados) $\to$ **6 trits** ($3^6 = 729$ estados) ou **1 tryte de 9 trits** ($3^9 = 19.683$ estados) |
| **Barramento de Endereço** | 16 bits (64 KB) $\to$ **10 trits** ($3^{10} = 59.049$ endereços) |
| **Flags de Status (P)** | Carry, Zero, Negative, Overflow $\to$ **Flag T (Trit Sign: $-$, $0$, $+$)** e **Flag O (Overflow Ternário)** |
| **ALU** | Somador de 8 bits com lógica Booleana $\to$ **Full Adder Ternário (TFA)** balanceado via lógica multiplexada de $\pm 5\text{ V}$ / $0\text{ V}$ |

```
                 SISTEMA MOS 6502-T (Visão de Blocos)
                 
     ┌─────────────────────────────────────────────────────────┐
     │                       6502-T CORE                       │
     │                                                         │
     │   ┌───────────────┐     ┌───────────────┐               │
     │   │ Registrador A │     │ Registrador X │               │
     │   │   (6 Trits)   │     │   (6 Trits)   │               │
     │   └───────┬───────┘     └───────┬───────┘               │
     │           │                     │                       │
     │           ▼                     ▼                       │
     │   ┌─────────────────────────────────────┐               │
     │   │     Ternary ALU (BJT Push-Pull)     │               │
     │   │       Níveis: +5V, 0V, -5V          │               │
     │   │   Soma, Inversão (-), Shifts Base 3 │               │
     │   └──────────────────┬──────────────────┘               │
     │                      │                                  │
     └──────────────────────┼──────────────────────────────────┘
                            │ Barramento Interno (6 Trits balanceados)
                            ▼
     ┌─────────────────────────────────────────────────────────┐
     │            DECODIFICADOR DUODECIMAL (BASE 12)           │
     │    Converte 6 Trits (-364 a +364) em 3 Dígitos Base 12  │
     │    Exibição e Barramento de Periféricos em Base 12      │
     └─────────────────────────────────────────────────────────┘

```

### Roteiro Prático de Prototipagem

1. **Fase 1 (Simulação Spice / Falstad):** Validar a célula básica do inversor ternário balanceado e a porta *Full Ternary Adder* (TFA) usando pares BJT complementares (como 2N3904 para NPN e 2N3906 para PNP) com os três trilhos (+5 V, 0 V, -5 V).
2. **Fase 2 (Verilog / Emulação Digital):** Mapear o 6502 em Verilog codificando cada trit em 2 bits de estado (`00` = -1, `01` = 0, `10` = +1) para verificar o pipeline e o conjunto de instruções duodecimal/ternário.
3. **Fase 3 (Hardware Discreto da ALU):** Construir a ALU de 6 trits em placas de circuito impresso com transistores BJT discretos para testes de velocidade de chaveamento, consumo estático no nível $0\text{ V}$ e estabilidade de transição entre $+5\text{ V}$ e $-5\text{ V}$.




Um somador completo ternário de 1 trit (*Ternary Full Adder* - TFA) opera sobre três entradas balanceadas $A, B, C_{in} \in \{-1, 0, +1\}$ e gera duas saídas balanceadas: a soma $S \in \{-1, 0, +1\}$ e o transporte $C_{out} \in \{-1, 0, +1\}$, mapeadas eletricamente nos níveis $+5\text{ V}$ ($+1$), $0\text{ V}$ ($0$) e $-5\text{ V}$ ($-1$).

---

## 1. Lógica Matemática do TFA Balanceado

A operação fundamental do somador completo ternário consiste em balancear a soma algébrica das entradas através da base 3:

$$S_{int} = A + B + C_{in}$$

Como cada entrada varia entre $-1$ e $+1$, a soma interna varia no intervalo discreto:

$$S_{int} \in \{-3, -2, -1, 0, +1, +2, +3\}$$

A relação entre as saídas e as entradas é dada pela identidade:

$$A + B + C_{in} = 3 \cdot C_{out} + S$$

Diferente do sistema binário, onde o transporte só ocorre para valores positivos ($1+1=2 \implies C=1, S=0$), no sistema balanceado o transporte é **simétrico e assinado**:

* **Se $S_{int} = +3$:** $C_{out} = +1$, $S = 0$ $\quad(3 \times 1 + 0 = 3)$
* **Se $S_{int} = +2$:** $C_{out} = +1$, $S = -1$ $\quad(3 \times 1 - 1 = 2)$
* **Se $S_{int} = +1$:** $C_{out} = 0$, $S = +1$ $\quad(3 \times 0 + 1 = 1)$
* **Se $S_{int} = 0$:** $C_{out} = 0$, $S = 0$ $\quad(3 \times 0 + 0 = 0)$
* **Se $S_{int} = -1$:** $C_{out} = 0$, $S = -1$ $\quad(3 \times 0 - 1 = -1)$
* **Se $S_{int} = -2$:** $C_{out} = -1$, $S = +1$ $\quad(3 \times -1 + 1 = -2)$
* **Se $S_{int} = -3$:** $C_{out} = -1$, $S = 0$ $\quad(3 \times -1 + 0 = -3)$

As equações analíticas diretas para $C_{out}$ e $S$ são:

$$C_{out} = \begin{cases} +1, & \text{se } S_{int} \ge +2 \\  0, & \text{se } -1 \le S_{int} \le +1 \\  -1, & \text{se } S_{int} \le -2 \end{cases}$$

$$S = S_{int} - 3 \cdot C_{out}$$

---

## 2. Tabela-Verdade Completa ($3^3 = 27$ Estados)

Na notação balanceada, utiliza-se comumente $-$ para representar $-1$, $0$ para $0$ e $+$ para $+1$.

| $A$ | $B$ | $C_{in}$ | Níveis de Tensão ($V_A, V_B, V_{Cin}$) | $S_{int}$ | $C_{out}$ | $S$ | Tensão $V_{Cout}$ | Tensão $V_S$ |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| $-$ | $-$ | $-$ | $-5V, -5V, -5V$ | $-3$ | $-$ | $0$ | $-5V$ | $0V$ |
| $-$ | $-$ | $0$ | $-5V, -5V, \phantom{-}0V$ | $-2$ | $-$ | $+$ | $-5V$ | $+5V$ |
| $-$ | $-$ | $+$ | $-5V, -5V, +5V$ | $-1$ | $0$ | $-$ | $\phantom{-}0V$ | $-5V$ |
| $-$ | $0$ | $-$ | $-5V, \phantom{-}0V, -5V$ | $-2$ | $-$ | $+$ | $-5V$ | $+5V$ |
| $-$ | $0$ | $0$ | $-5V, \phantom{-}0V, \phantom{-}0V$ | $-1$ | $0$ | $-$ | $\phantom{-}0V$ | $-5V$ |
| $-$ | $0$ | $+$ | $-5V, \phantom{-}0V, +5V$ | $0$ | $0$ | $0$ | $\phantom{-}0V$ | $0V$ |
| $-$ | $+$ | $-$ | $-5V, +5V, -5V$ | $-1$ | $0$ | $-$ | $\phantom{-}0V$ | $-5V$ |
| $-$ | $+$ | $0$ | $-5V, +5V, \phantom{-}0V$ | $0$ | $0$ | $0$ | $\phantom{-}0V$ | $0V$ |
| $-$ | $+$ | $+$ | $-5V, +5V, +5V$ | $+1$ | $0$ | $+$ | $\phantom{-}0V$ | $+5V$ |
| $0$ | $-$ | $-$ | $\phantom{-}0V, -5V, -5V$ | $-2$ | $-$ | $+$ | $-5V$ | $+5V$ |
| $0$ | $-$ | $0$ | $\phantom{-}0V, -5V, \phantom{-}0V$ | $-1$ | $0$ | $-$ | $\phantom{-}0V$ | $-5V$ |
| $0$ | $-$ | $+$ | $\phantom{-}0V, -5V, +5V$ | $0$ | $0$ | $0$ | $\phantom{-}0V$ | $0V$ |
| $0$ | $0$ | $-$ | $\phantom{-}0V, \phantom{-}0V, -5V$ | $-1$ | $0$ | $-$ | $\phantom{-}0V$ | $-5V$ |
| $0$ | $0$ | $0$ | $\phantom{-}0V, \phantom{-}0V, \phantom{-}0V$ | $0$ | $0$ | $0$ | $\phantom{-}0V$ | $0V$ |
| $0$ | $0$ | $+$ | $\phantom{-}0V, \phantom{-}0V, +5V$ | $+1$ | $0$ | $+$ | $\phantom{-}0V$ | $+5V$ |
| $0$ | $+$ | $-$ | $\phantom{-}0V, +5V, -5V$ | $0$ | $0$ | $0$ | $\phantom{-}0V$ | $0V$ |
| $0$ | $+$ | $0$ | $\phantom{-}0V, +5V, \phantom{-}0V$ | $+1$ | $0$ | $+$ | $\phantom{-}0V$ | $+5V$ |
| $0$ | $+$ | $+$ | $\phantom{-}0V, +5V, +5V$ | $+2$ | $+$ | $-$ | $+5V$ | $-5V$ |
| $+$ | $-$ | $-$ | $+5V, -5V, -5V$ | $-1$ | $0$ | $-$ | $\phantom{-}0V$ | $-5V$ |
| $+$ | $-$ | $0$ | $+5V, -5V, \phantom{-}0V$ | $0$ | $0$ | $0$ | $\phantom{-}0V$ | $0V$ |
| $+$ | $-$ | $+$ | $+5V, -5V, +5V$ | $+1$ | $0$ | $+$ | $\phantom{-}0V$ | $+5V$ |
| $+$ | $0$ | $-$ | $+5V, \phantom{-}0V, -5V$ | $0$ | $0$ | $0$ | $\phantom{-}0V$ | $0V$ |
| $+$ | $0$ | $0$ | $+5V, \phantom{-}0V, \phantom{-}0V$ | $+1$ | $0$ | $+$ | $\phantom{-}0V$ | $+5V$ |
| $+$ | $0$ | $+$ | $+5V, \phantom{-}0V, +5V$ | $+2$ | $+$ | $-$ | $+5V$ | $-5V$ |
| $+$ | $+$ | $-$ | $+5V, +5V, -5V$ | $+1$ | $0$ | $+$ | $\phantom{-}0V$ | $+5V$ |
| $+$ | $+$ | $0$ | $+5V, +5V, \phantom{-}0V$ | $+2$ | $+$ | $-$ | $+5V$ | $-5V$ |
| $+$ | $+$ | $+$ | $+5V, +5V, +5V$ | $+3$ | $+$ | $0$ | $+5V$ | $0V$ |

---

## 3. Arquitetura Modular em Blocos

A estrutura conceitual pode ser organizada em meio-somadores (*Ternary Half Adders*) e um gerador de transporte (*Carry Generator*), ou por uma abordagem de lógica por limiar (*Threshold Logic*), que reduz consideravelmente a quantidade de componentes discretos:

---

## 4. Esquemático Elétrico com BJTs ($\pm 5\text{ V}$, $0\text{ V}$)

A implementação em lógica de limiar analógica-digital aproveita as Leis de Kirchhoff para efetuar a soma ponderada de tensões, seguida por um estágio discriminador de janela e um estágio de restauração push-pull com BJTs.

### Diagrama do Circuito

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

    * O sinal V_S_raw passa por um par idêntico de limiares (Q5/Q6) e saída push-pull (Q7/Q8)
      com resistores de centralização em GND, restaurando os níveis plenos para a saída S.

```

### Detalhamento dos Componentes e Funcionamento

1. **Nó de Soma Linear ($V_\Sigma$):**
* Os resistores de precisão $R_1 = R_2 = R_3 = 10\text{ k}\Omega$ criam uma média aritmética direta das tensões de entrada:

$$V_\Sigma = \frac{V_A + V_B + V_{Cin}}{3}$$


* Para $S_{int} = +3 \implies V_\Sigma = +5{,}0\text{ V}$.
* Para $S_{int} = +2 \implies V_\Sigma = +3{,}33\text{ V}$.
* Para $S_{int} = +1 \implies V_\Sigma = +1{,}67\text{ V}$.
* Para $S_{int} = 0 \implies V_\Sigma = 0{,}0\text{ V}$.
* Para $S_{int} = -1 \implies V_\Sigma = -1{,}67\text{ V}$.
* Para $S_{int} = -2 \implies V_\Sigma = -3{,}33\text{ V}$.
* Para $S_{int} = -3 \implies V_\Sigma = -5{,}0\text{ V}$.


2. **Janela de Transporte ($C_{out}$):**
* O transistor **$Q_1$ (NPN)** tem seu emissor mantido em $+2{,}5\text{ V}$. Ele só conduz saturação quando $V_\Sigma > +2{,}5\text{ V} + V_{be} \approx +3{,}1\text{ V}$ (casos $S_{int} = +2$ e $+3$). Ao conduzir, aciona a base de $Q_3$, puxando a saída $C_{out}$ para $+5\text{ V}$.
* O transistor **$Q_2$ (PNP)** tem seu emissor mantido em $-2{,}5\text{ V}$. Ele só conduz saturação quando $V_\Sigma < -2{,}5\text{ V} - \vert{}V_{be}\vert{} \approx -3{,}1\text{ V}$ (casos $S_{int} = -2$ e $-3$). Ao conduzir, aciona a base de $Q_4$, puxando a saída $C_{out}$ para $-5\text{ V}$.
* Quando $-2{,}5\text{ V} < V_\Sigma < +2{,}5\text{ V}$, ambos os transistores de disparo ($Q_1$ e $Q_2$) cortam. O resistor $R_{center}$ força a saída $C_{out}$ a permanecer estável em $0\text{ V}$ (terra).


3. **Subtração e Restauração da Soma ($S$):**
* Pela relação $S = S_{int} - 3 \cdot C_{out}$, quando $C_{out} = +1$ ($+5\text{ V}$), injeta-se uma corrente de amortecimento na malha de soma através de $R_{sum2} = 30\text{ k}\Omega$ com fase invertida (via um inversor BJT simples $\pm 5\text{ V}$), reduzindo o patamar de tensão de volta para a faixa de $-5\text{ V}$ a $+5\text{ V}$.
* O estágio de saída da Soma utiliza a mesma topologia push-pull centralizada em $0\text{ V}$, garantindo casamento de impedância e níveis padronizados para alimentar os trits subsequentes do registrador acumulador.





## 5. Fundamentação Teórica Complementar: Euler, Pi e Gauss

A arquitetura do processador ternário e o sistema duodecimal conectam-se diretamente a três pilares da matemática clássica: a constante de Euler ($e$), a geometria de $\pi$ e a teoria dos números de Gauss.

### 5.1. Euler ($e$): Fasores e a Função Totiente na Base 12

Além da eficiência de radix ideal ($R = e$) que minimiza o custo físico de hardware $E(R) = R \cdot \frac{\ln N}{\ln R}$, a constante de Euler estabelece a ponte analítica entre a lógica ternária e as grandezas harmônicas por meio da **Identidade de Euler**:

$$e^{i\pi} + 1 = 0 \iff e^{i\pi} = -1$$

Ao estender essa identidade para a decomposição em três fases discretas no plano complexo, obtém-se o análogo contínuo do trit balanceado via raízes cúbicas da unidade:

$$\omega_k = e^{i \frac{2\pi k}{3}} \quad \text{para } k \in \{-1, 0, +1\}$$

$$\omega_0 = 1 \quad (0\text{ V / Referência}), \quad \omega_{+1} = -\frac{1}{2} + i\frac{\sqrt{3}}{2} \quad (+5\text{ V}), \quad \omega_{-1} = -\frac{1}{2} - i\frac{\sqrt{3}}{2} \quad (-5\text{ V})$$

A soma fasorial dessas três componentes é nula ($\sum \omega_k = 0$), espelhando a invariância de conservação de carga no nó de soma KCL da ALU ternária.

No nível duodecimal, a **Função Totiente de Euler** ($\phi$) evidencia a densidade estrutural da base:

$$\phi(12) = 12 \left(1 - \frac{1}{2}\right)\left(1 - \frac{1}{3}\right) = 4$$

Apenas quatro inteiros inferiores a 12 são coprimos com ele ($\{1, 5, 7, 11\}$). Os outros oito compartilham fatores com a base, permitindo fatorações limpas e simplificações de ponto fixo na ALU.

---

### 5.2. Pi ($\pi$): Geometria Angular e Aritmética Trigonométrica

A base 12 particiona o círculo trigonométrico euclidiano de comprimento $2\pi$ em arcos exatos sem truncamento irracional nas posições fundamentais de engenharia:

$$\theta_n = n \cdot \frac{2\pi}{12} = n \cdot \frac{\pi}{6} \quad (n \in \{0, 1, 2, \dots, 11\})$$

* Cada unidade do primeiro dígito fracionário em base 12 ($0{,}1_{12}$) equivale exatamente a $30^\circ$ ($\frac{\pi}{6}\text{ rad}$).
* O ciclo completo de uma rotação ($360^\circ$) corresponde a $2\pi\text{ rad}$, mapeando os senos e cossenos notáveis ($\frac{1}{2}, \frac{\sqrt{3}}{2}$) em subdivisões inteiras de clock em algoritmos de rotação vetorial (como CORDIC duodecimal).

A expansão de $\pi$ na base duodecimal apresenta convergência rápida:

$$\pi_{10} \approx 3{,}1415926535... \implies \pi_{12} = 3{,}184809493B91..._{12}$$

Na notação duodecimal, a fração $\frac{\pi}{4}$ (área do círculo unitário e limite de transformadas de Fourier discretas) mapeia-se como:

$$\left(\frac{\pi}{4}\right)_{12} \approx 0{,}9412..._{12}$$

---

### 5.3. Gauss: Resíduos Mínimos Absolutos e Imunidade ao Ruído

#### Aritmética de Resíduos Simétricos

Em *Disquisitiones Arithmeticae* (1801, Seção I), Carl Friedrich Gauss formalizou que qualquer congruência modular $x \equiv a \pmod m$ admite um **sistema de resíduos de menor valor absoluto**, onde os restos são escolhidos no intervalo simétrico:

$$-\frac{m}{2} < r \le \frac{m}{2}$$

Para $m = 3$, o sistema de resíduos mínimos de Gauss é exatamente o conjunto $\{-1, 0, +1\}$. A lógica ternária balanceada do 6502-T é a materialização em silício da aritmética de resíduos de menor valor absoluto de Gauss, eliminando a assimetria do bit de sinal binário (onde o intervalo de complemento de dois de $n$ bits cobre $[-2^{n-1}, +2^{n-1}-1]$ sem elemento inverso para o menor negativo).

#### Distribuição Gaussiana de Ruído Térmico

Nos transistores BJT da ALU, as flutuações térmicas de tensão (ruído de Johnson-Nyquist) nos resistores de $10\text{ k}\Omega$ seguem uma distribuição normal gaussiana centrada em zero:

$$p(V_n) = \frac{1}{\sigma \sqrt{2\pi}} \exp\left(-\frac{V_n^2}{2\sigma^2}\right), \quad \text{onde } \sigma^2 = 4 k_B T R \Delta f$$

Como o trilho neutro do trit está rigidamente referenciado a $0\text{ V}$ (GND) e os limiares de transição são perfeitamente equidistantes ($\Delta V = 2{,}5\text{ V}$ tanto para $+5\text{ V}$ quanto para $-5\text{ V}$), a integral das caudas gaussianas de falso disparo é rigorosamente idêntica para transições positivas e negativas:

$$P(\text{erro } 0 \to +1) = P(\text{erro } 0 \to -1) = \frac{1}{2} \operatorname{erfc}\left(\frac{V_{th}}{\sigma \sqrt{2}}\right)$$

Essa simetria gaussiana confere ao circuito BJT push-pull uma margem de imunidade estática a ruídos uniformemente balanceada nos três estados de chaveamento.

---

## 6. Referências Bibliográficas e Materiais Acadêmicos

1. **BRUSENTSOV, N. P. et al.** *Small Digital Computing Machine "Setun"*. Moscow: Moscow State University Press, 1962. (Documentação técnica original da arquitetura ternária balanceada do computador Setun).
2. **BRUSENTSOV, N. P.** "Ternary Principle: In Search of Natural Simplicity". *Informatika i Eyo Primeneniya* (Informatics and Its Applications), v. 2, n. 2, p. 104–112, 2008. (Revisão histórica sobre a eficiência da computação ternária em relação à binária).
3. **CAUCHY, A.-L.** "Sur les signes d'inégalité et sur la notation abrégée des nombres". *Comptes Rendus de l'Académie des Sciences*, v. 11, p. 789–798, Paris, 1840. (Fundamentação da notação com dígitos negativos na aritmética posicional).
4. **EULER, L.** *Introductio in analysin infinitorum*. Tomus Primus. Lausanne: Marc-Michel Bousquet & Co., 1748. (Dedução das propriedades analíticas da base dos logaritmos naturais $e$ e análise de funções transcendentes).
5. **GAUSS, C. F.** *Disquisitiones Arithmeticae*. Leipzig: Gerh. Fleischer, Jun., 1801. (Definição dos sistemas de congruências e resíduos de menor magnitude absoluta na Teoria dos Números).
6. **HURST, S. L.** "Multiple-Valued Logic: Its Status and Its Future". *IEEE Transactions on Computers*, v. C-33, n. 12, p. 1160–1179, dez. 1984. DOI: 10.1109/TC.1984.1676392. (Estudo sobre contagem de interconexões, custos de área de silício e eficiência de radix multi-nível).
7. **KNUTH, D. E.** *The Art of Computer Programming, Volume 2: Seminumerical Algorithms*. 3. ed. Boston: Addison-Wesley, 1997. Cap. 4.1: "Positional Number Systems", p. 192–212. (Análise formal da elegância matemática, ausência de sinal explícito e arredondamento natural da base ternária balanceada).
8. **SHANNON, C. E.** "A Mathematical Theory of Communication". *Bell System Technical Journal*, v. 27, p. 379–423, 623–656, 1948. (Fundamentos de entropia da informação e capacidade de canais aplicados à densidade de radix).
9. **VASILYEV, A. S.; SARKISOV, V. G.** "Design of Multi-Valued Current-Mode and Voltage-Mode Circuits Using BJT Technology". *IEEE International Symposium on Multiple-Valued Logic (ISMVL)*, 1999. (Topologias push-pull e limites de saturação em circuitos discretos de lógica ternária).
10. **VISUAL6502 TEAM.** *Visual 6502: Transistor-level simulation and reverse engineering of the MOS 6502*. 2010. Disponível em: `[http://visual6502.org/](http://visual6502.org/)`. (Mapeamento topológico dos nós de silício e das netlists utilizadas para a derivação do fork 6502-T).




















