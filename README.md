# 6502-T: Processador Ternário Balanceado em Base 12

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![Python 3.10+](https://img.shields.io/badge/Python-3.10%2B-green.svg)](https://www.python.org/)
[![Vivado 2023+](https://img.shields.io/badge/Xilinx%20Vivado-2023%2B-orange.svg)](https://www.xilinx.com/products/design-tools/vivado.html)
[![Icarus Verilog](https://img.shields.io/badge/Icarus%20Verilog-11.0%2B-yellow.svg)](http://iverilog.icarus.com/)
[![Architecture: Ternary](https://img.shields.io/badge/Logic-Balanced%20Ternary%20%7B--1%2C0%2C%2B1%7D-purple.svg)]()
[![Base: Dozenal](https://img.shields.io/badge/Radix-Duodecimal%20(Base%2012)-red.svg)]()

> **6502-T** é uma implementação de hardware aberto que reinventa o clássico microprocessador **MOS 6502**, substituindo a lógica booleana binária tradicional por **lógica ternária balanceada** $\{-1, 0, +1\}$ com comutação física via pares complementares de **transistores de junção bipolar (BJT)** operando em $\pm 5\text{ V} / 0\text{ V}$, articulada com abstração matemática e interfacing em **sistema duodecimal (base 12)**.

---

## Sumário

- [1. Introdução Executiva](#1-introdução-executiva)
  - [1.1. O Computador Setun (1958) e o Legado de Brusentsov](#11-o-computador-setun-1958-e-o-legado-de-brusentsov)
  - [1.2. A Constante de Euler ($e$) e a Eficiência de Radix](#12-a-constante-de-euler-e-e-a-eficiência-de-radix)
  - [1.3. A Abstração Duodecimal (Base 12)](#13-a-abstração-duodecimal-base-12)
- [2. Arquitetura Física e Circuitos](#2-arquitetura-física-e-circuitos)
  - [2.1. Célula BJT Push-Pull de Três Níveis ($\pm 5\text{ V}$, $0\text{ V}$)](#21-célula-bjt-push-pull-de-três-níveis-pm-5text-v-0text-v)
  - [2.2. Diagrama de Blocos da CPU 6502-T](#22-diagrama-de-blocos-da-cpu-6502-t)
- [3. Especificações Técnicas](#3-especificações-técnicas)
  - [3.1. Tabela Resumo](#31-tabela-resumo)
  - [3.2. Codificação da ISA e Modos de Endereçamento](#32-codificação-da-isa-e-modos-de-endereçamento)
- [4. Instalação e Requisitos](#4-instalação-e-requisitos)
- [5. Guia de Uso Rápido (Quickstart)](#5-guia-de-uso-rápido-quickstart)
  - [5.1. Montagem de Programas Assembly](#51-montagem-de-programas-assembly)
  - [5.2. Desassemblagem e Engenharia Reversa](#52-desassemblagem-e-engenharia-reversa)
  - [5.3. Execução Interativa no Emulador Ciclo a Ciclo](#53-execução-interativa-no-emulador-ciclo-a-ciclo)
  - [5.4. Interface Web Interativa (Tokyo Night) e Docker](#54-interface-web-interativa-tokyo-night-e-docker)
  - [5.5. Compilador e Laboratório de Programação VAMOS-T](#55-compilador-e-laboratório-de-programação-vamos-t)
- [6. Execução da Suite de Testes Automatizada](#6-execução-da-suite-de-testes-automatizada)
- [7. Estrutura do Repositório](#7-estrutura-do-repositório)
- [8. Referências Acadêmicas](#8-referências-acadêmicas)
- [9. Licença e Direitos](#9-licença-e-direitos)

---

## 1. Introdução Executiva

O processador **6502-T** une a elegância minimalista da arquitetura RISC pioneira do MOS 6502 (originalmente desenvolvido por Chuck Peddle em 1975) à física e matemática da computação ternária simétrica. Ao operar em três estados discretos balanceados $\{-1, 0, +1\}$ — denotados frequentemente como $\{-, 0, +\}$ —, o sistema elimina a assimetria do complemento de dois e bit de sinal, permitindo operações aritméticas simétricas e negação a custo computacional zero.

### 1.1. O Computador Setun (1958) e o Legado de Brusentsov

O marco histórico fundamental da computação ternária é o computador soviético **Setun**, projetado em 1958 na Universidade Estatal de Moscou sob liderança científica de **Nikolai Petrovich Brusentsov** e apoio do matemático Sergei Sobolev. 

O Setun utilizava núcleos magnéticos de ferrite saturável de três estados para implementar aritmética ternária balanceada. Os relatórios operacionais do Setun demonstraram:
1. Confiabilidade e imunidade a ruídos superiores às máquinas binárias a vácuo contemporâneas.
2. Menor contagem de componentes ativos para uma dada capacidade de cálculo.
3. Tratamento natural e elegante de números positivos e negativos sob os mesmos circuitos somadores.

O projeto **6502-T** resgata a visão de Brusentsov, transpondo-a para transistores bipolares discretos (BJT) e síntese moderna em FPGA.

### 1.2. A Constante de Euler ($e$) e a Eficiência de Radix

A escolha da base 3 (ternária) repousa sobre um princípio matemático rigoroso da teoria da informação e da física de dispositivos: a **Economia de Radix** (ou *Radix Economy*).

O custo de hardware $E(b, N)$ para representar uma faixa de $N$ inteiros usando uma base numérica inteira $b$ é proporcional ao produto da base pela quantidade de dígitos necessários $d$:

$$N = b^d \implies d = \frac{\ln N}{\ln b}$$

$$E(b, N) = b \cdot d = (\ln N) \cdot \left( \frac{b}{\ln b} \right)$$

Para encontrar a base ótima contínua que minimiza o custo $f(b) = \frac{b}{\ln b}$:

$$\frac{d}{db}\left(\frac{b}{\ln b}\right) = \frac{\ln b - 1}{(\ln b)^2} = 0 \implies \ln b = 1 \implies b = e \approx 2{,}71828\dots$$

A constante fundamental de Euler ($e$) é o número real matematicamente ideal. Ao restringir o projeto aos números inteiros realizáveis fisicamente:

| Radix ($b$) | Eficiência Relativa $f(b) = b/\ln(b)$ | Custo Comparativo |
|:-----------:|:-------------------------------------:|:-----------------:|
| $b = 2$ (Binário) | $2 / \ln(2) \approx 2{,}88539$ | $+5{,}66\%$ sobre o ideal |
| **$b = 3$ (Ternário)** | **$3 / \ln(3) \approx 2{,}73072$** | **Apenas $+0{,}46\%$ sobre $e$** |
| $b = 4$ (Quaternário) | $4 / \ln(4) \approx 2{,}88539$ | $+5{,}66\%$ sobre o ideal |
| $b = 10$ (Decimal) | $10 / \ln(10) \approx 4{,}34294$ | $+59{,}04\%$ de redundância |

A base 3 é a **mais próxima de $e$**, proporcionando a maior densidade teórica de informação por linha de barramento e menor número de interconexões no silício.

### 1.3. A Abstração Duodecimal (Base 12)

Embora o processador processe internamente trits balanceados, a camada de comunicação humana e representação de alto nível emprega a **base 12 (duodecimal ou dozenal)**:
* **Divisores Limpos:** O número 12 possui quatro divisores não-triviais $\{2, 3, 4, 6\}$ (contra apenas $\{2, 5\}$ da base 10), gerando frações exatas e facilitando particionamento de barramentos.
* **Empacotamento Trit-Dozenal:** Três trits balanceados codificam $3^3 = 27$ estados, cobrindo com folga subcampos duodecimais. No 6502-T, literais duodecimais são identificados pelo prefixo `0z` (com dígitos `0` a `9`, `A`=10, `B`=11).

---

## 2. Arquitetura Física e Circuitos

### 2.1. Célula BJT Push-Pull de Três Níveis ($\pm 5\text{ V}$, $0\text{ V}$)

Cada trit lógico é sintetizado fisicamente por uma célula simétrica push-pull constituída por um par de transistores bipolares NPN e PNP operando entre três trilhos de alimentação:

```
                     +5V (+1 Lógico / Trit POS)
                        │
                     ┌──┴──┐
                     │ NPN │ (Conduz quando V_in > +V_th)
                     └──┬──┘
                        │
         V_in ──────────┼────────────── V_out (Nível Analógico do Trit)
                        │                     │
                     ┌──┴──┐               [R_pull]
                     │ PNP │                  │
                     └──┬──┘                 GND (0V Central)
                        │
                     -5V (-1 Lógico / Trit NEG)

   (Quando V_in = 0V, ambos os BJTs entram em CORTE; o resistor R_pull
    centraliza V_out em 0.0V estável, definindo o estado neutro/0).
```

* **Estado $+1$ ($+5\text{ V}$):** Transistor NPN satura, forçando a linha de saída para $+5\text{ V}$.
* **Estado $0$ ($0\text{ V}$):** Ambos os transistores permanecem em corte. O divisor/resistor de centralização ancora a tensão em $0\text{ V}$ estático.
* **Estado $-1$ ($-5\text{ V}$):** Transistor PNP satura, ancorando a saída no trilho inferior de $-5\text{ V}$.

### 2.2. Diagrama de Blocos da CPU 6502-T

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

## 3. Especificações Técnicas

### 3.1. Tabela Resumo

| Parâmetro Arquitetural | Implementação 6502-T | Detalhes e Mapeamento Físico |
|:---|:---|:---|
| **Lógica Aritmética** | Ternária Balanceada | Estados $\{-1, 0, +1\}$ representados por $\{-, 0, +\}$ |
| **Níveis de Tensão (BJT)** | $\pm 5\text{ V}$ e $0\text{ V}$ | $+5\text{ V} \to +1$, $0\text{ V} \to 0$, $-5\text{ V} \to -1$ |
| **Mapeamento em Verilog** | 2 bits por trit | `2'b00` = $-1$, `2'b01` = $0$, `2'b10` = $+1$ |
| **Barramento de Dados** | **6 trits** ($12\text{ bits}$ Verilog) | Capacidade: $3^6 = 729$ estados, intervalo $[-364, +364]$ |
| **Barramento de Endereços** | **10 trits** ($20\text{ bits}$ Verilog) | Capacidade: $3^{10} = \mathbf{59.049}$ posições lineares |
| **Registradores Principais** | Acumulador (A), X, Y | Largura de 6 trits balanceados ($[-364, +364]$) |
| **Contador de Programa (PC)** | PC (10 trits) | Endereçamento direto de $0$ a $59.048$ ($+0\text{z}000$ a $+0\text{z}2A26$) |
| **Pilha (Hardware Stack)** | Descendente em RAM | Base padrão em $63$ ($+0\text{z}053$), expansão via macros |
| **Flags de Status Balanceadas** | **Zero (Z)**, **Sign (S)**, **Overflow (O)** | Z: Booleano; S: $\{-1, 0, +1\}$; O: Detecção de estouro de $\pm 364$ |
| **Ciclos por Instrução** | 3 a 5 ciclos de clock | FSM síncrona: `FETCH_OPCODE`, `DECODE`, `FETCH_OPERAND`, `EXECUTE` |

### 3.2. Codificação da ISA e Modos de Endereçamento

Cada instrução ocupa 1 palavra de 6 trits balanceados ($T_5, T_4, T_3, T_2, T_1, T_0$), estruturada em 3 campos ortogonais de 2 trits:

$$\underbrace{T_5 \; T_4}_{\text{Grupo}} \quad \underbrace{T_3 \; T_2}_{\text{Modo de Endereçamento}} \quad \underbrace{T_1 \; T_0}_{\text{Operação da ALU / Fluxo}}$$

* **Grupo ($T_5, T_4$):**
  * `(+1, +1)`: Operações Aritméticas e Lógicas da ALU (`LDA`, `ADD`, `SUB`, `NEG`, `INC`, `DEC`).
  * `(-1,  0)`: Transferências entre Registradores (`TAX`, `TXA`, `TAY`, `TYA`).
  * `(-1, -1)`: Saltos e Controle de Fluxo (`JMP`, `BEQ`, `BMI`, `BPL`).
  * `( 0, +1)`: Acesso à Memória RAM (`STA`).

* **Modo de Endereçamento ($T_3, T_2$):**
  * `(-1, -1)`: Implícito / Registrador interno (1 palavra).
  * `( 0, +1)`: Imediato `#valor` (2 palavras: opcode + constante).
  * `( 1, +1)`: Absoluto `$endereço` (2 palavras: opcode + endereço).

---

## 4. Instalação e Requisitos

### Dependências de Software

* **Python 3.10 ou superior:** Necessário para o montador, desassemblador, emulador e suite de testes.
* **Make:** Para automação de compilação e testes unificados.
* **Icarus Verilog (`iverilog`) e `vvp`:** (Opcional) Para simulação comportamental das RTLs Verilog.
* **Xilinx Vivado ML Standard (2023.1 ou superior):** (Opcional) Para síntese lógica, place & route e geração de bitstream para FPGAs Xilinx Artix-7 (ex.: Digilent Basys 3).

### Instalação no Linux (Ubuntu/Debian)

```bash
sudo apt update
sudo apt install -y python3 python3-pip make iverilog
```

Clone este repositório:
```bash
git clone https://github.com/facrf/Trit_Base12_BaseLivre.git
cd Trit_Base12_BaseLivre
```

---

## 5. Guia de Uso Rápido (Quickstart)

### 5.1. Montagem de Programas Assembly

Crie um arquivo assembly (ex.: `src_asm/program.s`) utilizando notações duodecimais e decimais:

```assembly
// Exemplo: Cálculo duodecimal
BUFFER_RES = 32

START:
    CLR                     // Pseudo-instrução: Zera o Acumulador A
    LDA #0z011              // Carrega literal duodecimal +13 decimal
    TAX                     // Salva no Registrador X
    ADD #0z0BB              // Soma +143 decimal -> A vira +156 (+0z110)
    STA BUFFER_RES          // Grava resultado na memória RAM [32]
    SUB #0z110              // Subtrai 156 -> Ativa flag Zero
    BEQ SUCCESS             // Desvia se Zero = 1
    JMP HLT

SUCCESS:
    LDA #0z042              // Carrega +50 decimal
    STA (BUFFER_RES + 1)    // Grava no endereço 33

HLT:
    JMP HLT                 // Loop de término
```

Para montar o binário de memória `.mem` via linha de comando ou Makefile:

```bash
# Via Makefile:
make asm

# Ou diretamente via montador Python:
python3 tools/asm6502t.py src_asm/program.s -o build/program.mem
```

### 5.2. Desassemblagem e Engenharia Reversa

Você pode inspecionar e reconstruir o código mnemônico a partir de qualquer binário de 12 bits:

```bash
# Via Makefile:
make disasm

# Ou diretamente:
python3 tools/disasm6502t.py build/program.mem
```

Saída gerada pelo desassemblador:
```assembly
[00]   12'b101010011010   LDA  #0            ; Dec:    0 | B12: 0z0 ; (Possível CLR)
[01]   12'b010101010101   ; (Operando Imediato)
[02]   12'b101010011010   LDA  #13           ; Dec:   13 | B12: +0z11
[03]   12'b010101101010   ; (Operando Imediato)
[04]   12'b010000000000   TAX               
[05]   12'b101010010000   ADD  #143          ; Dec:  143 | B12: +0zBB
[06]   12'b100000100100   ; (Operando Imediato)
[07]   12'b100110101010   STA  DATA_32       ; Alvo: $32 | B12: +0z28
...
```

### 5.3. Execução Interativa no Emulador Ciclo a Ciclo

O emulador `emu6502t.py` permite acompanhar em tempo real o chaveamento da FSM, barramentos analógicos, registradores e flags em representação de Trits, Decimal e Duodecimal:

```bash
# Modo Interativo com Debugger:
make emu
# Ou:
python3 tools/emu6502t.py build/program.mem

# Modo Autorun (executa até N ciclos e exibe dump de memória):
python3 tools/emu6502t.py build/program.mem --autorun --cycles 60 --dump
```

Comandos no console interativo do emulador:
* `[Enter]` ou `s`: Avança 1 ciclo de clock na FSM.
* `i`: Executa todos os ciclos até completar a instrução corrente.
* `c`: Executa continuamente até encontrar um breakpoint.
* `m <início> <tamanho>`: Exibe dump do mapa de memória em trits e base 12.
* `b <endereço>`: Insere ou remove um breakpoint no Program Counter.
* `r`: Reseta a CPU aos valores iniciais.
* `q`: Encerra o emulador.

### 5.4. Interface Web Interativa (Tokyo Night) e Docker

Para facilitar a depuração visual e o controle da CPU sem a necessidade de um terminal local dedicado, o projeto conta com um **frontend web em Go puro** (porta `:27005`) estilizado no tema **Tokyo Night** (`#1a1b26`, `#7dcfff`, `#bb9af7`), empacotado em um container Docker multi-stage leve:

* **Telemetria em Tempo Real:** Painel com cartões dinâmicos exibindo Ciclos, Estado da FSM (`FETCH_OPCODE`, `DECODE`, `EXECUTE`), Program Counter (PC), Acumulador (A), Registradores X/Y e Flags Ternárias $[Z \mid S \mid O]$ em Decimal, Trits e Base 12.
* **Barra de Ações Rápidas:**
  * `Step (Ciclo)` (`s` ou tecla `Space`): Avança 1 ciclo de clock.
  * `Instrução` (`i` ou tecla `I`): Avança até a conclusão da instrução atual.
  * `Run (Cont)` (`c` ou tecla `C`): Executa continuamente até 2.000 ciclos ou atingir um breakpoint.
  * `Dump RAM` (`m 0 32` ou tecla `M`): Inspeciona posições de memória.
  * `Reset CPU` (tecla `R`): Reinicia o processo da CPU e recarrega `build/program.mem`.
* **Terminal Interativo:** Console com histórico de comandos (navegação via setas `↑` e `↓`), atalhos de teclado globais, auto-scroll e cópia para área de transferência.
* **Endpoints da API HTTP:**
  * `GET /`: Servidor do SPA HTML5/CSS3/JavaScript embarcado.
  * `POST /api/cmd`: Envia comando ao sub-processo do emulador e retorna a resposta de texto (ou JSON via cabeçalho `Accept: application/json`).
  * `GET /api/cmd`: Retorna o buffer atual da CPU sem avançar o clock.
  * `POST /api/reset`: Reinicializa o processo da CPU 6502-T.

#### Compilação e Execução via Docker / Makefile

```bash
# 1. Compila a imagem Docker multi-stage (golang:alpine -> python:3.12-alpine)
make web-build

# 2. Inicia o container com a porta 27005 mapeada
make web-run
```

Acesse no navegador: **`http://localhost:27005`**

### 5.5. Compilador e Laboratório de Programação VAMOS-T

O 6502-T integra um compilador e transpiler para a linguagem **VAMOS-LANG** ([facrf/VAMOSLANG](https://github.com/facrf/VAMOSLANG)), uma linguagem com sintaxe no estilo Go em português brasileiro adaptada para o conjunto de instruções ternário balanceado e barramentos de 6 trits.

#### Exemplo em VAMOS-T: "Olá Mundo!" com Saída MMIO
```go
pacote principal

// Exemplo VAMOS-T: Olá Mundo na CPU Ternária 6502-T
// Emite cada caractere via Memory-Mapped I/O (Porta 300 / 0z210) no Barramento de 6 Trits
funcao principal() {
    escrever("Olá Mundo!\n")
    parar()
}
```

#### Uso via Linha de Comando (CLI)
```bash
# Executa diretamente o programa VAMOS-T com emulação na CPU e inspeção de variáveis
python3 tools/vamos6502t.py --run

# Compila para arquivo Assembly 6502-T
python3 tools/vamos6502t.py meu_codigo.vamos -o build/meu_codigo.asm

# Compila diretamente para binário .mem
python3 tools/vamos6502t.py meu_codigo.vamos -o build/meu_codigo.mem

# Retorna saída estruturada em JSON (Assembly + Trace de Execução + Variáveis)
python3 tools/vamos6502t.py meu_codigo.vamos --json
```

#### Laboratório Visual & Guia Interativo (Porta 27005)
A interface web conta com três ambientes dedicados acessíveis pelo menu superior:

1. **🧪 Laboratório VAMOS-T:**
   * **Player Passo a Passo:** Botões de reprodução animada (`▶`/`⏸`), avanço/recuo passo a passo (`⏮`/`⏭`) e controle de velocidade (50ms, 150ms, 350ms).
   * **Console CRT Virtual TTY:** Tela de fósforo verde com cursor pulsante exibindo a impressão em tempo real de caracteres via MMIO na porta 300 (`0z210`).
   * **Medidor LED de 6 Trits:** 6 lâmpadas LED com iluminação física para cada estado do Acumulador (`+1` ciano neon, `0` indigo escuro, `-1` magenta neon).
   * **Destaque no Assembly:** A instrução Assembly correspondente ao Program Counter atual é iluminada sincronizadamente durante a animação.
   * **Barra de Linha do Tempo (Slider):** Arraste para inspecionar qualquer ciclo ou instrução no tempo de execução.

2. **📖 Manual & Exemplos:**
   * **Fundamentos do Trit Balanceado:** Explicação didática sobre polaridade física push-pull ($-5\text{V}, 0\text{V}, +5\text{V}$) e negação simétrica sem complemento de dois.
   * **Tabela de Conversão Duodecimal:** Relação completa de pesos duodecimais (dígitos `0-9, A, B`) e prefixo `0z`.
   * **Cheatsheet da Sintaxe VAMOS-LANG:** Referência de declarações, operações, laços `para`, condicionais `se/senao` e registradores `A/X/Y`.
   * **Cards Interativos de Demonstração:** 5 exemplos prontos com botão **`▶ Carregar no Laboratório`** para teste instantâneo:
     - 🧮 *Calculadora Duodecimal:* Operações aritméticas em base 12 (`0z...`).
     - 💠 *O Poder do Trit:* Negação unária simétrica instantânea.
     - 🔄 *Contador & Somatório de Gauss:* Laço `para` acumulando valores na RAM.
     - ⚖️ *Decisão & Flags Ternárias:* Ramificação condicional e flags de Status Zero/Sign.
     - 🌟 *Olá Mundo:* Emissão sequencial de caracteres no terminal CRT.

3. **💻 Terminal & Telemetria:**
   * Sessão interativa direta com a CPU e o monitor de memória.

---

## 6. Execução da Suite de Testes Automatizada

O projeto conta com uma rigorosa suite de validação funcional ponta a ponta que compila programas, inicializa a CPU simulada, executa os ciclos e confere asserções de acumulador, índices, flags e escrita na memória:

```bash
# Execução da suite de regressão obrigatória (7/7):
make test

# Ou diretamente:
python3 tests/test_6502t_suite.py

# Execução da suite estendida e reversibilidade de toolchain (6/6 + Round-trip):
make test-all
```

### Saída Esperada

```text
================================================================================
 TEST SUITE INTEGRADO - CPU TERNÁRIA 6502-T (TRIT + BASE 12)
================================================================================
 [01/07] PASS: Aritmética Básica (LDA + ADD em Base 12)      (  8 ciclos)
 [02/07] PASS: Negação Unária e Flag Sign Negativa (NEG + SUB) ( 11 ciclos)
 [03/07] PASS: Transferência de Registradores e Memória (TAX, STA, LDA) ( 19 ciclos)
 [04/07] PASS: Desvio Condicional BEQ com Ativação de Flag Zero ( 16 ciclos)
 [05/07] PASS: Desvio Condicional BMI com Número Negativo    ( 16 ciclos)
 [06/07] PASS: Pseudo-Instruções (CLR, CMP, PUSH, POP)       ( 47 ciclos)
 [07/07] PASS: Detecção de Overflow Ternário (> +364)        (  8 ciclos)
--------------------------------------------------------------------------------
 RESULTADO FINAL: 7/7 testes passaram com sucesso.
 Tempo de Execução: 1.16 ms
================================================================================
```

---

## 7. Estrutura do Repositório

```text
Trit_Base12_BaseLivre/
├── constr/                      # Restrições físicas para FPGA Digilent Basys 3
│   └── basys3_6502t.xdc         # Mapeamento de pinos e clocks para Artix-7
├── rtl/                         # 10 módulos Verilog sintetizáveis (silício)
│   ├── alu_6trit.v              # Unidade Lógica e Aritmética ternária balanceada
│   ├── button_debounce.v        # Debouncer de botões para FPGA
│   ├── cpu_6502t_debugger.v     # Controlador de debug e interface do usuário
│   ├── cpu_6502t_top.v          # Top-level da CPU MOS 6502-T
│   ├── display_controller_4digit.v # Varredura multiplexada de displays de 7 segmentos
│   ├── dozenal_7seg_decoder.v   # Decodificador de base 12 para 7 segmentos
│   ├── fpga_test_top.v          # Top-level para sintetização em FPGA Basys 3
│   ├── instruction_decoder_6502t.v # Decodificador de instruções e FSM síncrona
│   ├── tfa_1trit.v              # Ternary Full Adder de 1 trit balanceado
│   └── trits_to_dozenal.v       # Conversor físico de 6 trits para 2 dígitos duodecimais
├── sim/                         # 7 testbenches para simulação Icarus Verilog
│   ├── tb_alu_6trit.v           # Verificação da ALU e detecção de overflow
│   ├── tb_cpu_6502t_system.v    # Simulação integrada da CPU executando programa
│   ├── tb_display_controller.v  # Testbench dos controladores de display
│   ├── tb_dozenal_display.v     # Verificação dos decodificadores de display
│   ├── tb_instruction_decoder_6502t.v # Verificação do decodificador de instruções
│   ├── tb_system_dozenal_monitor.v # Monitor de barramento e telemetria duodecimal
│   └── tb_tfa_1trit.v           # Verificação da tabela de verdade do somador
├── docs/                        # Documentação técnica e teses conceituais
│   ├── base_conceitual.md       # Fundamentação física e matemática
│   ├── especificacao_completa.md# Especificação detalhada da CPU e RTLs Verilog
│   ├── manual_emulador.md       # Manual de operação do emulador ciclo a ciclo
│   ├── simulacao.md             # Guia de simulação e formas de onda
│   ├── tese_antitese.md         # Análise comparativa Ternário vs Binário
│   └── assets/                  # Diagramas esquemáticos e imagens
├── tools/                       # Ferramentas CLI de software e emulação
│   ├── asm6502t.py              # Montador com suporte a NOP, BNE, .ORG e .WORD
│   ├── disasm6502t.py           # Desassemblador com modo limpo (-c) e suporte a @addr
│   ├── emu6502t.py              # Emulador ciclo a ciclo interativo (TYA, BNE, dump 64p)
│   └── vamos6502t.py            # Compilador VAMOS-LANG para 6502-T (CLI, JSON e trace)
├── tests/                       # Suites de validação funcional
│   ├── test_6502t_suite.py      # Testes ponta a ponta obrigatórios (7/7 PASS)
│   └── test_6502t_extended.py   # Testes estendidos e reversibilidade round-trip
├── scripts/                     # Scripts de automação EDA
│   └── build_bitstream.tcl      # Script em lote para síntese no Vivado
├── src_asm/                     # Programas Assembly de demonstração
│   └── program.s                # Código fonte exemplo em base 12
├── web/                         # Frontend Web HTTP em Go (Tema Tokyo Night)
│   ├── go.mod                   # Definição do módulo Go
│   └── main.go                  # Servidor web, Terminal, VAMOS-T Playground e TTY CRT
├── Dockerfile.web               # Multi-stage build (golang:alpine -> python:3.12-alpine)
├── Makefile                     # Orquestrador unificado de build e testes
├── AGENTS.md                    # Diretrizes e convenções para agentes de IA
├── CITATION.cff                 # Metadados acadêmicos de citação
├── LICENSE                      # GNU General Public License v3.0 (GPLv3)
├── LICENSE_PTBR                 # Tradução em Português Brasileiro da GPLv3
└── README.md                    # Documentação principal do repositório
```

---

## 8. Referências Acadêmicas

1. **Brusentsov, Nikolai P. et al. (1960).** *The Setun Small Digital Computer*. Moscow State University, Vestnik Moskovskogo Universiteta, No. 4, pp. 3–14.
2. **Euler, Leonhard (1748).** *Introductio in analysin infinitorum*. Capítulo sobre bases transcendentais e eficiência natural da base $e$.
3. **Gauss, Carl Friedrich (1801).** *Disquisitiones Arithmeticae*. Estudos fundamentais sobre congruências e sistemas de numeração simétricos.
4. **Cauchy, Augustin-Louis (1840).** *Sur les chiffres réduits*. Comptes Rendus des Séances de l'Académie des Sciences, Paris, Vol. 11, pp. 789–798. (Formalização da notação aritmética balanceada sem necessidade de sinal separado).
5. **Knuth, Donald E. (1997).** *The Art of Computer Programming, Volume 2: Seminumerical Algorithms*. 3rd Edition, Addison-Wesley. Seção 4.1: Positional Number Systems. ("*Perhaps the prettiest number system of all is the balanced ternary notation...*").
6. **Hayes, Brian (2001).** *Third Base: Is balanced ternary the most natural way to represent numbers?* American Scientist, Vol. 89, No. 6, pp. 490–494.

---

## 9. Licença e Direitos

Este projeto é software e hardware aberto distribuído sob os termos da **GNU General Public License versão 3.0 (GPLv3)**. Consulte o arquivo [LICENSE](LICENSE) para obter o texto integral em inglês ou [LICENSE_PTBR](LICENSE_PTBR) para a versão em português brasileiro.

```text
6502-T: Processador Ternário Balanceado em Base 12
Copyright (C) 2026 Fabiano Augusto Cesar da Rocha Filho e Colaboradores.
Este programa é um software livre: você pode redistribuí-lo e/ou modificá-lo
sob os termos da Licença Pública Geral GNU conforme publicada pela Free
Software Foundation, tanto a versão 3 da Licença, como (a seu critério)
qualquer versão posterior.
```
