# Manual de Referência: Compilador VAMOS-T & Laboratório Educacional

> **Módulo:** `tools/vamos6502t.py` & `web/main.go`  
> **Linguagem:** VAMOS-LANG ([facrf/VAMOSLANG](https://github.com/facrf/VAMOSLANG)) adaptada para 6502-T  
> **Arquitetura Alvo:** Microprocessador Ternário Balanceado MOS 6502-T  
> **Sistemas Numéricos:** Lógica Ternária $\{-1, 0, +1\}$ e Abstração Duodecimal (Base 12)  

---

## 1. Visão Geral

O **VAMOS-T** é a camada de programação de alto nível do ecossistema **6502-T**. Ele implementa um compilador/transpiler nativo que converte código-fonte escrito em **VAMOS-LANG** diretamente para a linguagem de montagem da CPU ternária (`Assembly 6502-T`), podendo em seguida montar para binário de memória (`.mem`) ou executar passo a passo em ambiente de emulação física.

### 1.1. Propósito e Filosofia
A linguagem VAMOS-LANG adota uma sintaxe declarativa e imperativa inspirada em Go, utilizando palavras-chave em **Português Brasileiro** (`pacote`, `funcao`, `se/senao`, `para`, `escrever`, `parar`).

No contexto da arquitetura **6502-T**, o compilador resolve dois desafios fundamentais:
1. **Didático:** Permite que estudantes e desenvolvedores experimentem aritmética ternária balanceada e a elegância da base 12 sem precisar manipular mnemonicos e endereçamento de registradores manualmente.
2. **Pedagógico e Visual:** Integra-se perfeitamente à interface Web (porta `27005`), fornecendo reprodução animada da execução, terminal CRT em fósforo verde com Memory-Mapped I/O (MMIO), visualizador físico em LEDs tricolores dos 6 trits e sincronização do Program Counter (PC).

---

## 2. Mapa de Memória e Geometria da Arquitetura

O processador 6502-T opera com palavras de **6 trits balanceados** (faixa modular $[-364, +364]$) e barramento de endereçamento linear de **10 trits balanceados** ($3^{10} = 59.049$ posições).

O compilador VAMOS-T adota a seguinte convenção padronizada de layout de memória no espaço de endereçamento direto da ISA:

| Faixa de Endereços (Decimal) | Faixa (Base 12) | Finalidade Arquitetural | Descrição |
|:---|:---|:---|:---|
| `$000` a `$179` ($0$ a $179$) | `0z000` a `0z12B` | **Segmento de Código (Text)** | Instruções Assembly geradas pelo compilador. |
| `$180` a `$299` ($180$ a $299$) | `0z130` a `0z20B` | **Segmento de Dados (RAM/Vars)** | Alocação sequencial de variáveis estáticas e dinâmicas (`num1`, `soma`, etc.). |
| `$300` ($300$) | `0z210` | **Porta Virtual MMIO TTY CRT** | Escrita mapeada em memória para streaming de caracteres no console do usuário. |
| `$301` a `$364` | `0z211` a `0z264` | **Reserva de E/S & Stack** | Espaço livre dentro da faixa de 6 trits balanceados. |

> [!NOTE]
> Todos os endereços de operandos de instruções de 1 palavra no 6502-T (`LDA`, `STA`, `ADD`, `SUB`) são expressos em 6 trits, restringindo o endereçamento direto à faixa $[-364, +364]$. A alocação de variáveis a partir de `$180` e a porta MMIO em `$300` respeitam estritamente esse limite físico da ISA.

---

## 3. Fundamentos Matemáticos em VAMOS-T

### 3.1. Lógica Ternária Balanceada $\{-1, 0, +1\}$
Cada posição de memória armazena um valor composto por 6 trits balanceados, com pesos $3^0, 3^1, 3^2, 3^3, 3^4, 3^5$:

$$\text{Valor} = \sum_{k=0}^{5} t_k \cdot 3^k \quad \text{onde } t_k \in \{-1, 0, +1\}$$

* **Negação Simétrica Instantânea:** Ao contrário dos computadores binários em complemento de dois, onde negar requer inverter todos os bits e somar $1$ (gerando assimetria como $-128$ até $+127$), na lógica ternária balanceada o negativo de um número é simplesmente a inversão de polaridade de cada trit individual:
  * $+1 \to -1$
  * $0 \to 0$
  * $-1 \to +1$

Exemplo em VAMOS-T:
```go
valor := 0z042       // +50 decimal: trits [0, +, -, 0, -, -]
invertido := 0 - valor // -50 decimal: trits [0, -, +, 0, +, +]
```

### 3.2. Aritmética Duodecimal (Base 12)
Para leitura e notação amigável ao ser humano, o VAMOS-T suporta literais numéricos com o prefixo `0z`.

* Algarismos: `0, 1, 2, 3, 4, 5, 6, 7, 8, 9, A (=10), B (=11)`.
* Exemplos de Conversão:
  * `0z005` = $5_{10}$
  * `0z010` = $12_{10}$
  * `0z011` = $13_{10}$
  * `0z015` = $17_{10}$ ($1 \times 12 + 5$)
  * `0z042` = $50_{10}$ ($4 \times 12 + 2$)
  * `0z0BB` = $143_{10}$ ($11 \times 12 + 11$)
  * `0z110` = $156_{10}$ ($1 \times 144 + 1 \times 12 + 0$)

---

## 4. Guia da Linguagem e Sintaxe

### 4.1. Estrutura do Programa
Todo programa VAMOS-T inicia com a declaração do pacote principal e o ponto de entrada `funcao principal()`:

```go
pacote principal

funcao principal() {
    // Código executável
    parar()
}
```

### 4.2. Declaração e Atribuição de Variáveis e Constantes
Variáveis e constantes podem ser declaradas com inferência de tipo (`:=`) ou palavras-chave explícitas (`var`, `variavel`, `const`, `constante`):

```go
a := 10            // Literal decimal
b := 0z015         // Literal duodecimal (+17 decimal)
c := -5            // Valor negativo
var total = 0      // Declaração tradicional de variável
const PI = 3       // Declaração explícita de constante
constante TAXA := 12 // Constante com atribuição direta
```

As variáveis e constantes são alocadas sequencialmente na memória RAM (a partir do endereço `$180`), e o compilador gera automaticamente instruções `STA` e `LDA` correspondentes.

### 4.3. Operações Aritméticas e Negação Unária
Suporta expressões multi-termos associativas à esquerda com adição (`+`), subtração (`-`) e negação unária simétrica (`-`):

```go
soma := num1 + num2
diferenca := total - 0z00A
inverso := -positivo      // Inversão física direta via instrução NEG
expressao := taxa + 10 - 5 + inverso // Expressão encadeada multi-termos
contador := contador + 1
```

O compilador também suporta os operadores de incremento e decremento unário:
```go
contador++
contador--
```

### 4.4. Acesso aos Registradores da CPU
Para fins didáticos e otimização de baixo nível, o programador pode ler e escrever diretamente nos registradores internos da CPU 6502-T:

```go
A = resultado      // Carrega no Acumulador (LDA resultado)
X = A              // Transfere Acumulador para Índice X (TAX)
Y = A              // Transfere Acumulador para Índice Y (TAY)
A = X              // Transfere Índice X para Acumulador (TXA)
```

### 4.5. Estruturas de Decisão (`se` / `senao`)
Permite desvios baseados nas flags ternárias da ALU:

```go
se alvo < 0 {
    status = -1
} senao {
    status = 1
}
```

Operadores suportados: `==`, `!=`, `<`, `>`, `<=`, `>=`.

### 4.6. Laços de Repetição (`para`)
Implementa estruturas iterativas utilizando a instrução `para`:

```go
limite := 5
total := 0
i := 0

para i < limite {
    total = total + i
    i = i + 1
}
```

### 4.7. Entrada e Saída: Impressão no Console TTY CRT (`escrever`)
A função nativa `escrever(...)` emite cadeias de caracteres sequencialmente para a porta MMIO `$300` (`0z210`):

```go
escrever("Calculadora 6502-T: OK!\n")
```

Para cada caractere da cadeia, o compilador emite:
```assembly
    LDA #<código_ascii>
    STA 300 // TTY_OUT
```

---

## 5. Exemplos Canônicos Embutidos

### 5.1. Calculadora Duodecimal (Base 12)
```go
pacote principal

funcao principal() {
    num1 := 0z010      // 12 em decimal
    num2 := 0z005      // 5 em decimal

    soma := num1 + num2 // 17 decimal (0z015)
    resultado := soma - 0z008 // 9 decimal (0z009)

    A = resultado
    escrever("Calc Base 12: OK!\n")
    parar()
}
```

### 5.2. Negação Simétrica do Trit Balanceado
```go
pacote principal

funcao principal() {
    valor := 0z042     // +50 decimal
    invertido := 0 - valor // -50 decimal (Simetria pura!)

    A = invertido
    escrever("Trit Balanceado Simetrico!\n")
    parar()
}
```

### 5.3. Somatório Progressivo de Gauss (1..5)
```go
pacote principal

funcao principal() {
    limite := 5
    total := 0
    i := 0

    para i < limite {
        total = total + i
        i = i + 1
    }

    A = total
    parar()
}
```

### 5.4. Multiplicação por Acúmulo (Base 12)
```go
pacote principal

// Multiplica 4 por 3 = 12 (+0z010 em base 12)
funcao principal() {
    a := 4
    b := 3
    produto := 0

    para b > 0 {
        produto = produto + a
        b = b - 1
    }

    A = produto
    escrever("Mult OK!\n")
    parar()
}
```

---

## 6. Ferramentas CLI e Interface Web

### 6.1. Execução via Linha de Comando (`tools/vamos6502t.py`)

```bash
# Executa o código e exibe o mapa de variáveis final
python3 tools/vamos6502t.py programa.vamos --run

# Compila para assembly legível
python3 tools/vamos6502t.py programa.vamos -o build/programa.asm

# Compila para imagem binária da memória (.mem)
python3 tools/vamos6502t.py programa.vamos -o build/program.mem

# Emite JSON estruturado com trace passo a passo (usado pela Web)
python3 tools/vamos6502t.py programa.vamos --json
```

### 6.2. Interface Web Interativa (Porta 27005)

A interface Web provê um ambiente educacional completo com quatro abas:

1. **🧪 Laboratório VAMOS-T:**
   * Editor de código com realce e atalho `Ctrl+Enter` para compilação instantânea.
   * Player com controle de velocidade e avanço/recuo passo a passo.
   * Medidor de 6 lâmpadas LED tricolores refletindo os trits balanceados do Acumulador.
   * Tela CRT virtual com fósforo verde e cursor pulsante.
   * Destaque síncrono da instrução Assembly em execução.
   * **Compartilhamento por URL (`#code=...`):** Copia um link com o código codificado em base64 Unicode para envio a alunos e professores.
2. **📖 Manual & Exemplos:**
   * Guia resumido da linguagem, aritmética duodecimal e cards didáticos prontos para execução com um clique.
3. **🏆 Desafios Interativos (Trit Challenges):**
   * Quatro desafios com validação automática e feedback imediato:
     * *Desafio 1:* Inversor de sinal simétrico sem multiplicação.
     * *Desafio 2:* Soma e representação duodecimal (`0z011`).
     * *Desafio 3:* Contador iterativo e ativação de Flag Zero.
     * *Desafio 4:* Classificador de 3 estados e ativação de Flag Sign.
4. **💻 Terminal & Telemetria:**
   * Sessão interativa direta com a CPU 6502-T em tempo real.

### 6.3. Segurança para Exposição via Cloudflare
O servidor web implementa:
* Headers defensivos: `Content-Security-Policy`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: SAMEORIGIN`, `Referrer-Policy`.
* Rate limiter de 60 requisições/minuto por IP para `/api/*`, com extração de IP compatível com `CF-Connecting-IP` e `X-Forwarded-For`.
   * 5 cards práticos com o botão **`▶ Carregar no Laboratório`** para execução com um clique.
3. **💻 Terminal & Telemetria:**
   * Sessão interativa com a CPU 6502-T em tempo real.
