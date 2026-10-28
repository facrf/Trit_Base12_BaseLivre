# Manual de Operação e Referência: Emulador Ciclo a Ciclo 6502-T

> **Módulo:** `tools/emu6502t.py`  
> **Arquitetura:** Microprocessador MOS 6502 Ternário Balanceado $\{-1, 0, +1\}$  
> **Abstração Numérica:** Aritmética Duodecimal (Base 12) e Decimal  

---

## 1. Visão Geral

O emulador **6502-T** (`tools/emu6502t.py`) é uma ferramenta de depuração de nível de microarquitetura, projetada para reproduzir fielmente o comportamento elétrico e lógico do processador ternário balanceado. Diferente de emuladores que apenas interpretam opcodes em nível funcional, o 6502-T modela:

* A **FSM interna de 5 estados** (idêntica ao RTL Verilog `instruction_decoder_6502t.v`).
* Os **barramentos físicos** (dados de 6 trits / 12 bits; endereçamento linear de 10 trits / 20 bits).
* A **ALU balanceada** com saturação modular $[-364, +364]$ e flags nativas.
* A telemetria simultânea em **Trits Físicos** ($\{-, 0, +\}$), **Base 12** (`0z...`) e **Decimal**.

---

## 2. Instalação e Requisitos

* **Python:** 3.10 ou superior.
* **Dependências Externas:** Nenhuma (utiliza exclusivamente módulos nativos da biblioteca padrão: `sys`, `os`, `re`, `typing`, `argparse`).

---

## 3. Modos de Execução

### 3.1. Modo Interativo (Depuração Passo a Passo)

Inicia o emulador abrindo o console interativo de depuração:

```bash
# Executa com o programa demonstrativo embutido
python3 tools/emu6502t.py

# Carrega um arquivo binário compilado (.mem)
python3 tools/emu6502t.py build/program.mem

# Define uma memória RAM expandida (ex.: 128 palavras)
python3 tools/emu6502t.py build/program.mem -s 128

# Via orquestrador Make (compila e inicia)
make emu
```

### 3.2. Modo Automatizado (Batch / Headless)

Executa um número pré-determinado de ciclos e emite a telemetria final:

```bash
# Executa 60 ciclos e exibe status e dump completo da memória
python3 tools/emu6502t.py build/program.mem --autorun --cycles 60 --dump

# Executa 500 ciclos apenas para verificar estado final dos registradores
python3 tools/emu6502t.py build/program.mem -a -n 500
```

#### Parâmetros de Linha de Comando (CLI)

| Opção | Argumento | Padrão | Descrição |
| :--- | :--- | :--- | :--- |
| `file` | `[caminho]` | *Demo embutido* | Caminho do arquivo de memória (`.mem`, `.hex` ou bloco Verilog). |
| `-s`, `--size` | `INT` | `64` | Capacidade da memória RAM em palavras de 6 trits. |
| `-a`, `--autorun` | *flag* | `False` | Executa diretamente em lote sem abrir console interativo. |
| `-n`, `--cycles` | `INT` | `500` | Limite de ciclos de clock no modo autorun. |
| `-d`, `--dump` | *flag* | `False` | Exibe o dump completo da memória ao término da execução. |
| `-h`, `--help` | *flag* | - | Exibe a mensagem de ajuda e sintaxe. |

---

## 4. Console Interativo de Depuração

Ao iniciar em modo interativo, o emulador exibe o prompt `(6502-T) > ` e aguarda instruções do operador.

```text
==============================================================================
 EMULADOR 6502-T (TRIT + BASE 12) - DEPURAÇÃO INTERATIVA CICLO A CICLO
 Comandos: [Enter]=Passo Ciclo | 'i'=Passo Instrução | 'c'=Continuar
           'd'=Desassemblar    | 'm'=Ver Memória     | 'set'=Alterar Registrador
           'b <end>'=Breakpoint | 'r'=Reset          | 'h'=Ajuda | 'q'=Sair
==============================================================================
```

### 4.1. Tabela de Comandos

| Comando | Sintaxe | Descrição |
| :--- | :--- | :--- |
| **Passo de Ciclo** | `[Enter]` ou `s` | Avança exatamente **1 ciclo de clock** na FSM. Permite inspecionar barramentos nas fases de busca e decodificação. |
| **Passo de Instrução** | `i` | Executa quantos ciclos forem necessários até completar a instrução corrente e retornar a `FETCH_OPCODE`. |
| **Continuar** | `c` | Executa continuamente até atingir um breakpoint ativo ou o limite de 2000 ciclos. |
| **Desassemblar** | `d [inicio] [tamanho]` | Desassembla e lista instruções a partir do PC ou de um endereço especificado com indicador `=>`. Ex: `d 0 20`. |
| **Exibir Memória** | `m [inicio] [tamanho]` | Apresenta o dump formatado das palavras de memória. Ex: `m 30 10` exibe 10 palavras a partir do endereço 30. |
| **Alterar Registrador** | `set <reg> <val>` | Altera dinamicamente o valor de registradores (`a`, `x`, `y`, `pc`) ou memória (`set mem <end> <val>`). |
| **Breakpoint** | `b <endereco>` | Alterna um ponto de parada (*toggle*). Ex: `b 17` adiciona/remove breakpoint no endereço 17. |
| **Listar Breakpoints** | `b` | Lista todos os endereços com breakpoints ativos. |
| **Resetar CPU** | `r` | Reinicializa registradores, PC e FSM, mantendo o conteúdo gravado na memória RAM. |
| **Ajuda** | `h` ou `help` | Exibe a lista rápida com todos os comandos disponíveis no console. |
| **Sair** | `q` | Encerra a execução do emulador. |

---

## 5. Interpretação da Telemetria e Painel de Estado

A cada passo, o emulador emite o quadro de telemetria completo:

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ CICLO: 60     | FSM: DECODE         | BARRAMENTO: ADDR=$21 DATA_OUT=+050 │
├────────────────────────────────────────────────────────────────────────────┤
│ REGISTRADORES       TRITS (MST->LST)    DECIMAL     BASE 12                │
│  Acumulador (A) :   [0+-0--]            +50        +0z042                │
│  Índice X       :   [000+++]            +13        +0z011                │
│  Índice Y       :   [000000]             +0         0z000                │
│  Prog. Counter  :   [000000+-+0]       $0021        +0z019                │
├────────────────────────────────────────────────────────────────────────────┤
│ FLAGS TERNÁRIAS: [Zero (Z): 0] [Sinal (S): +] [Overflow (O): 0]               │
└────────────────────────────────────────────────────────────────────────────┘
```

### 5.1. Componentes do Painel

1. **Cabeçalho:**
   * `CICLO`: Contador absoluto de ciclos de clock transcorridos.
   * `FSM`: Estado atual da máquina de estados (`FETCH_OPCODE`, `DECODE`, `FETCH_OPERAND`, `FETCH_ADDR`, `EXECUTE`).
   * `BARRAMENTO`: Endereço ativo no barramento externo (`ADDR`) e dado apresentado na saída de escrita (`DATA_OUT`).

2. **Registradores:**
   * `Acumulador (A)`, `Índice X`, `Índice Y`: Registradores de dados de 6 trits balanceados (faixa $[-364, +364]$).
   * `Prog. Counter (PC)`: Registrador de ponteiro de instruções de 10 trits ($59.049$ posições lineares).
   * Para cada registrador são fornecidas três representações simultâneas:
     * **Trits Físicos:** Vetor na ordem mais significativo $\to$ menos significativo (`MST -> LST`), usando os caracteres `-` ($-1$), `0` ($0$) e `+` ($+1$).
     * **Decimal:** Inteiro com sinal correspondente ao valor aritmético real.
     * **Base 12 (Duodecimal):** Notação literal `0z...` utilizando os algarismos `0-9, A, B`.

3. **Flags Ternárias:**
   * `Zero (Z)`: `1` quando o resultado da ALU ou transferência for zero; `0` caso contrário.
   * `Sinal (S)`: `-` para valores negativos, `0` para neutro/zero, `+` para valores estritamente positivos.
   * `Overflow (O)`: `1` se a operação na ALU extrapolou $[-364, +364]$; `0` em operação regular.

---

## 6. Formatos de Arquivo Suportados

O emulador aceita carregar código-máquina e dados a partir de arquivos em múltiplos formatos:

### 6.1. Formato `.mem` (Padrão do Assembler)
Vetores de 12 bits linha a linha, com ou sem comentários:
```text
12'b101010011010 // [ 0] LDA #0
12'b010101010101 // [ 1] Dado (0)
12'b101010011010 // [ 2] LDA #13
12'b010101101010 // [ 3] Dado (13)
```

### 6.2. Diretivas de Endereçamento `@addr`
Compatível com a diretiva `$readmemb` do Verilog:
```text
@00 101010011010
@01 010101010101
@20 010110100000
```

### 6.3. Blocos de Atribuição Verilog
Códigos exportados diretamente para testbenches:
```verilog
initial begin
    ram_block[ 0] = 12'b101010011010; // LDA #0
    ram_block[ 1] = 12'b010101010101; // Dado (0)
end
```

---

## 7. Exemplo Passo a Passo de Sessão de Depuração

```bash
$ python3 tools/emu6502t.py build/program.mem
[✓] 23 palavras de 6 trits carregadas de 'build/program.mem'.
(6502-T) > b 17
[+] Breakpoint adicionado em $17

(6502-T) > c
[!] Breakpoint atingido no endereço 17!

(6502-T) > m 30 5
--- MAPA DE MEMÓRIA (Palavras 30 a 34) ---
  [30] :    0 |  0z000 | Trits: [000000]
  [31] :    0 |  0z000 | Trits: [000000]
  [32] : +156 | +0z110 | Trits: [+-0-+0]
  [33] :    0 |  0z000 | Trits: [000000]
  [34] :    0 |  0z000 | Trits: [000000]

(6502-T) > i
(6502-T) > i
(6502-T) > m 32 2
--- MAPA DE MEMÓRIA (Palavras 32 a 33) ---
  [32] : +156 | +0z110 | Trits: [+-0-+0]
  [33] :  +50 | +0z042 | Trits: [0+-0--]

(6502-T) > q
```
