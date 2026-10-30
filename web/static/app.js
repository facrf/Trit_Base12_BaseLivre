    (function() {
      const termOutput = document.getElementById('term-output');
      const cmdInput = document.getElementById('cmd-input');
      const termForm = document.getElementById('terminal-form');
      const connStatus = document.getElementById('conn-status');
      const linkAutoscroll = document.getElementById('link-autoscroll');

      // Telemetry elements
      const elCycle = document.getElementById('stat-cycle');
      const elState = document.getElementById('stat-state');
      const elPc = document.getElementById('stat-pc');
      const elAcc = document.getElementById('stat-acc');
      const elRegX = document.getElementById('stat-regx');
      const elRegY = document.getElementById('stat-regy');
      const elFlags = document.getElementById('stat-flags');

      let autoscroll = true;
      let cmdHistory = [];
      let historyIndex = -1;

      linkAutoscroll.addEventListener('click', () => {
        autoscroll = !autoscroll;
        linkAutoscroll.textContent = 'Auto-scroll: ' + (autoscroll ? 'ON' : 'OFF');
      });

      function parseTelemetry(text) {
        if (!text) return;

        // CICLO: 0 | FSM: FETCH_OPCODE
        const cycleMatch = text.match(/CICLO:\s*(\d+)/i);
        if (cycleMatch) elCycle.textContent = cycleMatch[1];

        const fsmMatch = text.match(/FSM:\s*(\w+)/i);
        if (fsmMatch) elState.textContent = fsmMatch[1];

        // Prog. Counter : [0000000000] $0000 0z000
        const pcMatch = text.match(/Prog\.\s*Counter\s*:\s*\[[^\]]+\]\s*(\S+)\s*(\S+)/i);
        if (pcMatch) elPc.textContent = pcMatch[1] + ' (' + pcMatch[2] + ')';

        // Acumulador (A) : [000000] +0 0z000
        const accMatch = text.match(/Acumulador\s*\(A\)\s*:\s*\[[^\]]+\]\s*([+\-]?\d+)\s*(\S+)/i);
        if (accMatch) elAcc.textContent = accMatch[1] + ' (' + accMatch[2] + ')';

        // Índice X : [000000] +0 0z000
        const xMatch = text.match(/Índice\s*X\s*:\s*\[[^\]]+\]\s*([+\-]?\d+)\s*(\S+)/i);
        if (xMatch) elRegX.textContent = xMatch[1] + ' (' + xMatch[2] + ')';

        // Índice Y : [000000] +0 0z000
        const yMatch = text.match(/Índice\s*Y\s*:\s*\[[^\]]+\]\s*([+\-]?\d+)\s*(\S+)/i);
        if (yMatch) elRegY.textContent = yMatch[1] + ' (' + yMatch[2] + ')';

        // FLAGS TERNÁRIAS: [Zero (Z): 0] [Sinal (S): 0] [Overflow (O): 0]
        const zMatch = text.match(/Zero\s*\(Z\):\s*([+\-]?\d+)/i);
        const sMatch = text.match(/Sinal\s*\(S\):\s*([+\-0])/i);
        const oMatch = text.match(/Overflow\s*\(O\):\s*([+\-]?\d+)/i);
        if (zMatch && sMatch && oMatch) {
          elFlags.textContent = '[' + zMatch[1] + ' | ' + sMatch[1] + ' | ' + oMatch[1] + ']';
        }
      }

      function updateTerminal(text, append) {
        if (append) {
          termOutput.textContent += text;
        } else {
          termOutput.textContent = text;
        }
        parseTelemetry(text);
        if (autoscroll) {
          termOutput.scrollTop = termOutput.scrollHeight;
        }
      }

      async function sendCommand(cmd) {
        try {
          connStatus.textContent = 'EXECUTANDO...';
          const res = await fetch('/api/cmd', {
            method: 'POST',
            headers: { 'Content-Type': 'text/plain' },
            body: cmd
          });
          if (!res.ok) {
            throw new Error('HTTP ' + res.status + ': ' + (await res.text()));
          }
          const text = await res.text();
          updateTerminal(text, false);
          connStatus.textContent = 'CPU ONLINE';
        } catch (err) {
          updateTerminal('\n[ERRO DE CONEXÃO]: ' + err.message + '\n', true);
          connStatus.textContent = 'ERRO';
        }
      }

      async function resetCPU() {
        try {
          connStatus.textContent = 'REINICIANDO...';
          const res = await fetch('/api/reset', { method: 'POST' });
          if (!res.ok) {
            throw new Error('HTTP ' + res.status + ': ' + (await res.text()));
          }
          const text = await res.text();
          updateTerminal(text, false);
          connStatus.textContent = 'CPU ONLINE';
        } catch (err) {
          updateTerminal('\n[ERRO AO REINICIAR]: ' + err.message + '\n', true);
          connStatus.textContent = 'ERRO';
        }
      }

      // Initial Fetch
      async function init() {
        try {
          const res = await fetch('/api/cmd');
          if (res.ok) {
            const text = await res.text();
            updateTerminal(text, false);
          }
        } catch (e) {
          console.warn('Erro ao carregar estado inicial:', e);
        }
      }
      init();

      // Form Submit
      termForm.addEventListener('submit', (e) => {
        e.preventDefault();
        const cmd = cmdInput.value.trim();
        cmdInput.value = '';
        if (cmd) {
          cmdHistory.push(cmd);
          historyIndex = cmdHistory.length;
        }
        sendCommand(cmd);
      });

      // History navigation (Up / Down)
      cmdInput.addEventListener('keydown', (e) => {
        if (e.key === 'ArrowUp') {
          if (cmdHistory.length > 0 && historyIndex > 0) {
            historyIndex--;
            cmdInput.value = cmdHistory[historyIndex];
          }
          e.preventDefault();
        } else if (e.key === 'ArrowDown') {
          if (historyIndex < cmdHistory.length - 1) {
            historyIndex++;
            cmdInput.value = cmdHistory[historyIndex];
          } else {
            historyIndex = cmdHistory.length;
            cmdInput.value = '';
          }
          e.preventDefault();
        }
      });

      // Quick Control Buttons
      document.getElementById('btn-step').addEventListener('click', () => sendCommand('s'));
      document.getElementById('btn-inst').addEventListener('click', () => sendCommand('i'));
      document.getElementById('btn-run').addEventListener('click', () => sendCommand('c'));
      document.getElementById('btn-dump').addEventListener('click', () => sendCommand('m 0 32'));
      document.getElementById('btn-reset').addEventListener('click', () => resetCPU());

      document.getElementById('btn-clear').addEventListener('click', () => {
        termOutput.textContent = '';
      });

      document.getElementById('btn-copy').addEventListener('click', () => {
        navigator.clipboard.writeText(termOutput.textContent).then(() => {
          const original = document.getElementById('btn-copy').textContent;
          document.getElementById('btn-copy').textContent = '✔ Copiado!';
          setTimeout(() => {
            document.getElementById('btn-copy').textContent = original;
          }, 1500);
        });
      });

      // Global hotkeys when input is not focused
      window.addEventListener('keydown', (e) => {
        if (document.activeElement === cmdInput || document.activeElement === document.getElementById('vamos-code')) return;
        if (e.ctrlKey || e.metaKey || e.altKey) return;

        const key = e.key.toLowerCase();
        if (key === ' ' || key === 's') {
          e.preventDefault();
          sendCommand('s');
        } else if (key === 'i') {
          e.preventDefault();
          sendCommand('i');
        } else if (key === 'c') {
          e.preventDefault();
          sendCommand('c');
        } else if (key === 'm') {
          e.preventDefault();
          sendCommand('m 0 32');
        } else if (key === 'r') {
          e.preventDefault();
          resetCPU();
        }
      });

      // Toast notification helper
      function showToast(message, type = 'info') {
        const container = document.getElementById('toast-container');
        if (!container) return;
        const toast = document.createElement('div');
        toast.style.pointerEvents = 'auto';
        toast.style.padding = '12px 18px';
        toast.style.borderRadius = '8px';
        toast.style.fontSize = '13px';
        toast.style.fontWeight = '500';
        toast.style.boxShadow = '0 8px 24px rgba(0,0,0,0.5)';
        toast.style.display = 'flex';
        toast.style.alignItems = 'center';
        toast.style.gap = '8px';
        toast.style.transition = 'all 0.3s ease';
        toast.style.transform = 'translateY(20px)';
        toast.style.opacity = '0';

        if (type === 'success') {
          toast.style.background = '#1a2b27';
          toast.style.border = '1px solid var(--accent-green)';
          toast.style.color = 'var(--accent-green)';
        } else if (type === 'error') {
          toast.style.background = '#2c1e28';
          toast.style.border = '1px solid var(--accent-red)';
          toast.style.color = 'var(--accent-red)';
        } else {
          toast.style.background = '#1f2335';
          toast.style.border = '1px solid var(--accent-cyan)';
          toast.style.color = 'var(--accent-cyan)';
        }

        toast.innerHTML = message;
        container.appendChild(toast);

        requestAnimationFrame(() => {
          toast.style.transform = 'translateY(0)';
          toast.style.opacity = '1';
        });

        setTimeout(() => {
          toast.style.transform = 'translateY(20px)';
          toast.style.opacity = '0';
          setTimeout(() => toast.remove(), 300);
        }, 4000);
      }

      // Tab Navigation
      const tabVamos = document.getElementById('tab-vamos');
      const tabManual = document.getElementById('tab-manual');
      const tabChallenges = document.getElementById('tab-challenges');
      const tabTerminal = document.getElementById('tab-terminal');
      const tabAbout = document.getElementById('tab-about');
      const viewVamos = document.getElementById('view-vamos');
      const viewManual = document.getElementById('view-manual');
      const viewChallenges = document.getElementById('view-challenges');
      const viewTerminal = document.getElementById('view-terminal');
      const viewAbout = document.getElementById('view-about');

      function switchTab(target) {
        tabVamos.classList.toggle('active', target === 'vamos');
        tabManual.classList.toggle('active', target === 'manual');
        if (tabChallenges) tabChallenges.classList.toggle('active', target === 'challenges');
        tabTerminal.classList.toggle('active', target === 'terminal');
        if (tabAbout) tabAbout.classList.toggle('active', target === 'about');

        viewVamos.style.display = (target === 'vamos') ? 'block' : 'none';
        viewManual.style.display = (target === 'manual') ? 'block' : 'none';
        if (viewChallenges) viewChallenges.style.display = (target === 'challenges') ? 'block' : 'none';
        viewTerminal.style.display = (target === 'terminal') ? 'block' : 'none';
        if (viewAbout) viewAbout.style.display = (target === 'about') ? 'block' : 'none';
      }

      tabVamos.addEventListener('click', () => switchTab('vamos'));
      tabManual.addEventListener('click', () => switchTab('manual'));
      if (tabChallenges) tabChallenges.addEventListener('click', () => switchTab('challenges'));
      tabTerminal.addEventListener('click', () => switchTab('terminal'));
      if (tabAbout) tabAbout.addEventListener('click', () => switchTab('about'));

      // VAMOS-LANG Playground & Visualizer
      const vamosCode = document.getElementById('vamos-code');
      const vamosPresets = document.getElementById('vamos-presets');
      const btnVamosRun = document.getElementById('btn-vamos-run');
      const btnVamosShare = document.getElementById('btn-vamos-share');
      const btnVamosVerifyActive = document.getElementById('btn-vamos-verify-active');
      const vamosChallengeBadge = document.getElementById('vamos-challenge-badge');
      const vamosCompileStatus = document.getElementById('vamos-compile-status');
      const vamosAsmOutput = document.getElementById('vamos-asm-output');
      const vamosAsmBadge = document.getElementById('vamos-asm-badge');
      const vStatCycles = document.getElementById('v-stat-cycles');
      const vStatPc = document.getElementById('v-stat-pc');
      const vStatAcc = document.getElementById('v-stat-acc');
      const vStatRegx = document.getElementById('v-stat-regx');
      const vStatRegy = document.getElementById('v-stat-regy');
      const vStatFlags = document.getElementById('v-stat-flags');
      const vamosVarsTbody = document.getElementById('vamos-vars-tbody');

      const vBtnPlay = document.getElementById('v-btn-play');
      const vPlayIcon = document.getElementById('v-play-icon');
      const vPlayText = document.getElementById('v-play-text');
      const vBtnPrev = document.getElementById('v-btn-prev');
      const vBtnNext = document.getElementById('v-btn-next');
      const vBtnStart = document.getElementById('v-btn-start');
      const vBtnEnd = document.getElementById('v-btn-end');
      const vStepLabel = document.getElementById('v-step-label');
      const vStepSlider = document.getElementById('v-step-slider');
      const vSpeedSelect = document.getElementById('v-speed-select');
      const vTtyText = document.getElementById('v-tty-text');
      const vTtyBadge = document.getElementById('v-tty-badge');
      const vTritValue = document.getElementById('v-trit-value');

      let activeChallengeId = null;

      const presets = {
        ola: "pacote principal\n\n// Exemplo VAMOS-T: Olá Mundo na CPU Ternária 6502-T\n// Emite cada caractere via MMIO (Porta 300 / 0z210) no Barramento de 6 Trits\nfuncao principal() {\n    escrever(\"Olá Mundo!\\n\")\n    parar()\n}",
        calc: "pacote principal\n\n// Exemplo VAMOS-T: Calculadora Duodecimal (Base 12) no 6502-T\n// Demonstra soma e subtração com literais duodecimais 0z...\nfuncao principal() {\n    // 0z010 = 12 em decimal, 0z005 = 5 em decimal\n    num1 := 0z010\n    num2 := 0z005\n\n    // Soma: 12 + 5 = 17 (em base 12: 0z015)\n    soma := num1 + num2\n\n    // Subtração: 17 - 8 = 9 (0z009)\n    resultado := soma - 0z008\n\n    A = resultado\n    escrever(\"Calc Base 12: OK!\\n\")\n    parar()\n}",
        trit: "pacote principal\n\n// Exemplo VAMOS-T: Lógica Ternária Balanceada {-1, 0, +1}\n// Demonstra negação unária simétrica nativa (NEG) e flags de sinal\nfuncao principal() {\n    // Carrega um valor positivo (+50 decimal / 0z042)\n    valor := 0z042\n\n    // Inversão simétrica nativa via instrução NEG em hardware!\n    invertido := -valor\n\n    A = invertido\n    escrever(\"Trit Balanceado Simétrico!\\n\")\n    parar()\n}",
        loop: "pacote principal\n\n// Exemplo VAMOS-T: Loop de Contagem e Acúmulo no 6502-T\nfuncao principal() {\n    limite := 5\n    total := 0\n    i := 0\n\n    para i < limite {\n        total = total + i\n        i = i + 1\n    }\n\n    A = total\n    parar()\n}",
        cond: "pacote principal\n\n// Exemplo VAMOS-T: Lógica Condicional e Flags Ternárias\nfuncao principal() {\n    alvo := -12\n    status := 0\n\n    se alvo < 0 {\n        status = -1\n    } senao {\n        status = 1\n    }\n\n    A = status\n    parar()\n}",
        mult: "pacote principal\n\n// Exemplo VAMOS-T: Multiplicação Aritmética por Acúmulo no 6502-T\n// Multiplica a (4) por b (3) = 12 (0z010 em base 12)\nfuncao principal() {\n    a := 4\n    b := 3\n    produto := 0\n\n    para b > 0 {\n        produto = produto + a\n        b = b - 1\n    }\n\n    A = produto\n    escrever(\"Mult OK!\\n\")\n    parar()\n}",
        fib: "pacote principal\n\n// Exemplo VAMOS-T: Sequência de Fibonacci na CPU 6502-T\n// Computa termos progressivos usando registradores e memória balanceada\nfuncao principal() {\n    const termos = 6\n    fn2 := 0\n    fn1 := 1\n    i := 0\n\n    para i < termos {\n        proximo := fn2 + fn1\n        fn2 = fn1\n        fn1 = proximo\n        i++\n    }\n\n    escrever(\"Fibonacci 6502-T OK!\\n\")\n    A = fn1 // Termo F(7) = +13 (+0z011)\n    parar()\n}",
        olamindo: "pacote principal\n\n// Exemplo VAMOS-T: olá mindo! (Solicitação literal)\nfuncao principal() {\n    escrever(\"olá mindo!\\n\")\n    parar()\n}"
      };

      const challenges = {
        1: {
          title: "Inversor Simétrico Sem Multiplicação",
          code: "pacote principal\n\n// DESAFIO 1: Inversor Simétrico Sem Multiplicação\n// Obtenha o inverso simétrico de 'entrada' na variável 'saida' e no Acumulador A.\nfuncao principal() {\n    entrada := 0z042 // +50 decimal\n    \n    // Seu código aqui (inversão unária simétrica direta: -entrada):\n    saida := -entrada\n    \n    A = saida\n    parar()\n}",
          validator: (exec) => {
            const v = exec.variables || {};
            if (v.saida && v.saida.decimal === -50 && exec.acc === -50) {
              return { ok: true, msg: "🎉 Parabéns! Simetria ternária perfeita: +50 invertido para -50 sem complemento de dois!" };
            }
            return { ok: false, msg: "Critério não atendido. Esperado 'saida' = -50 e A = -50. Atual: saida = " + (v.saida ? v.saida.decimal : "não definida") + ", A = " + exec.acc };
          }
        },
        2: {
          title: "Aritmética Duodecimal (Base 12)",
          code: "pacote principal\n\n// DESAFIO 2: Aritmética Duodecimal (Base 12)\n// Some num1 (0z008) com num2 (0z005) e armazene na variável 'total' e em A.\nfuncao principal() {\n    num1 := 0z008\n    num2 := 0z005\n    \n    // Seu código aqui:\n    total := num1 + num2\n    \n    A = total\n    parar()\n}",
          validator: (exec) => {
            const v = exec.variables || {};
            if (v.total && v.total.decimal === 13 && exec.acc === 13) {
              return { ok: true, msg: "🎉 Excelente! 8 + 5 = 13 (0z011 em base 12). Aritmética duodecimal validada!" };
            }
            return { ok: false, msg: "Critério não atendido. Esperado 'total' = 13 (+0z011). Atual: total = " + (v.total ? v.total.decimal : "não definida") + ", A = " + exec.acc };
          }
        },
        3: {
          title: "Contador Regressivo com Loop",
          code: "pacote principal\n\n// DESAFIO 3: Contador Regressivo com Loop\n// Decremente 'contador' de 3 até 0 usando 'para' e carregue o resultado em A.\nfuncao principal() {\n    contador := 3\n    \n    // Seu código aqui:\n    para contador != 0 {\n        contador--\n    }\n    \n    A = contador\n    parar()\n}",
          validator: (exec) => {
            const v = exec.variables || {};
            if (v.contador && v.contador.decimal === 0 && exec.acc === 0 && exec.flag_zero) {
              return { ok: true, msg: "🎉 Muito bem! Laço iterativo completado e Flag Zero (Z=1) ativada!" };
            }
            return { ok: false, msg: "Critério não atendido. Esperado 'contador' = 0 e Flag Z=1. Atual: contador = " + (v.contador ? v.contador.decimal : "não definida") };
          }
        },
        4: {
          title: "Classificador Ternário de Três Estados",
          code: "pacote principal\n\n// DESAFIO 4: Classificador Ternário de Três Estados\n// Classifique 'alvo' (-25): defina 'classe' como -1 se for negativo, ou +1 se não.\nfuncao principal() {\n    alvo := -25\n    classe := 0\n    \n    // Seu código aqui:\n    se alvo < 0 {\n        classe = -1\n    } senao {\n        classe = 1\n    }\n    \n    A = classe\n    parar()\n}",
          validator: (exec) => {
            const v = exec.variables || {};
            if (v.classe && v.classe.decimal === -1 && exec.acc === -1 && exec.flag_sign === -1) {
              return { ok: true, msg: "🎉 Espetacular! Classificador ternário executado e Flag Sign (S=-1) confirmada!" };
            }
            return { ok: false, msg: "Critério não atendido. Esperado 'classe' = -1 e Flag S=-1. Atual: classe = " + (v.classe ? v.classe.decimal : "não definida") + ", Flag S = " + exec.flag_sign };
          }
        }
      };

      function startChallenge(chId) {
        const ch = challenges[chId];
        if (!ch) return;
        activeChallengeId = chId;
        vamosCode.value = ch.code;
        if (vamosChallengeBadge) {
          vamosChallengeBadge.textContent = 'Desafio ' + chId + ' Ativo';
          vamosChallengeBadge.style.display = 'inline-block';
        }
        if (btnVamosVerifyActive) {
          btnVamosVerifyActive.style.display = 'inline-block';
        }
        switchTab('vamos');
        window.scrollTo({ top: 0, behavior: 'smooth' });
        showToast('🎯 Desafio ' + chId + ' carregado no editor! Escreva o código e clique em "Validar Desafio".', 'info');
        runVamos(true);
      }

      async function verifyChallenge(chId) {
        const ch = challenges[chId];
        if (!ch) return;

        showToast('Validando desafio ' + chId + '...', 'info');
        const code = vamosCode.value.trim();
        if (!code) return;

        try {
          const res = await fetch('/api/vamos', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ source: code })
          });
          const data = await res.json();
          if (!res.ok || !data.success) {
            showToast('Erro de compilação: ' + (data.error || 'Falha ao compilar'), 'error');
            const b = document.getElementById('badge-ch-' + chId);
            if (b) {
              b.textContent = 'Erro ❌';
              b.style.borderColor = 'var(--accent-red)';
              b.style.color = 'var(--accent-red)';
            }
            return;
          }

          const result = ch.validator(data.execution || {});
          const b = document.getElementById('badge-ch-' + chId);
          if (result.ok) {
            showToast(result.msg, 'success');
            if (b) {
              b.textContent = 'Aprovado ✅';
              b.style.borderColor = 'var(--accent-green)';
              b.style.color = 'var(--accent-green)';
            }
          } else {
            showToast(result.msg, 'error');
            if (b) {
              b.textContent = 'Falhou ❌';
              b.style.borderColor = 'var(--accent-yellow)';
              b.style.color = 'var(--accent-yellow)';
            }
          }
        } catch (e) {
          showToast('Erro de conexão ao validar desafio.', 'error');
        }
      }

      document.querySelectorAll('.btn-start-challenge').forEach(btn => {
        btn.addEventListener('click', () => {
          const chId = btn.getAttribute('data-ch');
          if (chId) startChallenge(chId);
        });
      });

      document.querySelectorAll('.btn-verify-challenge').forEach(btn => {
        btn.addEventListener('click', () => {
          const chId = btn.getAttribute('data-ch');
          if (chId) verifyChallenge(chId);
        });
      });

      if (btnVamosVerifyActive) {
        btnVamosVerifyActive.addEventListener('click', () => {
          if (activeChallengeId) verifyChallenge(activeChallengeId);
        });
      }

      if (btnVamosShare) {
        btnVamosShare.addEventListener('click', () => {
          const code = vamosCode.value;
          try {
            const b64 = btoa(encodeURIComponent(code).replace(/%([0-9A-F]{2})/g, (match, p1) => String.fromCharCode('0x' + p1)));
            const newUrl = window.location.origin + window.location.pathname + '#code=' + b64;
            history.replaceState(null, '', newUrl);
            navigator.clipboard.writeText(newUrl).then(() => {
              showToast('🔗 Link copiado para a área de transferência!', 'success');
            }).catch(() => {
              prompt('Copie o link abaixo:', newUrl);
            });
          } catch (e) {
            showToast('Erro ao gerar link de compartilhamento.', 'error');
          }
        });
      }

      function checkUrlParams() {
        const hash = window.location.hash;
        if (hash.startsWith('#code=')) {
          try {
            const b64 = hash.substring(6);
            const raw = decodeURIComponent(Array.prototype.map.call(atob(b64), c => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2)).join(''));
            if (raw && raw.trim()) {
              vamosCode.value = raw;
              switchTab('vamos');
              showToast('📥 Código carregado a partir do link compartilhado!', 'info');
              runVamos(true);
              return true;
            }
          } catch (e) {
            console.error('Erro ao decodificar hash da URL', e);
          }
        } else if (hash.startsWith('#ch=')) {
          const chId = hash.substring(4);
          if (challenges[chId]) {
            startChallenge(chId);
            return true;
          }
        }
        return false;
      }

      // Load code from manual cards
      document.querySelectorAll('.btn-load-code').forEach(btn => {
        btn.addEventListener('click', () => {
          const pName = btn.getAttribute('data-preset');
          if (pName && presets[pName]) {
            vamosPresets.value = pName;
            vamosCode.value = presets[pName];
            switchTab('vamos');
            window.scrollTo({ top: 0, behavior: 'smooth' });
            runVamos(true);
          }
        });
      });

      vamosPresets.addEventListener('change', () => {
        const val = vamosPresets.value;
        if (presets[val]) {
          vamosCode.value = presets[val];
          runVamos(false);
        }
      });
      // Set initial code to "ola"
      vamosCode.value = presets.ola;

      // Snippet helpers
      document.querySelectorAll('.btn-snippet').forEach(btn => {
        btn.addEventListener('click', () => {
          const snip = btn.getAttribute('data-snip');
          if (snip) {
            const code = snip.replace(/\\n/g, '\n');
            const start = vamosCode.selectionStart;
            const end = vamosCode.selectionEnd;
            vamosCode.value = vamosCode.value.substring(0, start) + code + vamosCode.value.substring(end);
            vamosCode.focus();
            vamosCode.selectionStart = vamosCode.selectionEnd = start + code.length;
          }
        });
      });
      const btnClear = document.getElementById('btn-vamos-clear');
      if (btnClear) {
        btnClear.addEventListener('click', () => {
          vamosCode.value = '';
          vamosCode.focus();
        });
      }

      // Step & Timeline Execution State
      let currentTrace = [];
      let currentStepIndex = 0;
      let playTimer = null;

      function renderStep(idx) {
        if (!currentTrace || currentTrace.length === 0) return;
        if (idx < 0) idx = 0;
        if (idx >= currentTrace.length) idx = currentTrace.length - 1;
        currentStepIndex = idx;

        const step = currentTrace[idx];
        vStepSlider.value = idx;
        vStepLabel.textContent = 'Passo ' + (idx + 1) + ' / ' + currentTrace.length + ' (Ciclo ' + step.cycle + ')';

        vStatCycles.textContent = step.cycle;
        vStatPc.textContent = '$' + (step.pc || 0).toString(16).padStart(4, '0');
        vStatAcc.textContent = (step.acc >= 0 ? '+' : '') + step.acc + ' (' + (step.acc_dozenal || '0z000') + ')';
        vStatRegx.textContent = (step.reg_x >= 0 ? '+' : '') + step.reg_x;
        vStatRegy.textContent = (step.reg_y >= 0 ? '+' : '') + step.reg_y;
        vStatFlags.textContent = '[' + (step.flag_zero ? 1 : 0) + ' | ' + (step.flag_sign || 0) + ' | ' + (step.flag_overflow ? 1 : 0) + ']';

        // TTY Console
        if (step.stdout && step.stdout.length > 0) {
          vTtyText.textContent = '> ' + step.stdout;
          vTtyBadge.textContent = step.stdout.length + ' chars';
        } else {
          vTtyText.textContent = '> [Aguardando saída TTY MMIO...]';
          vTtyBadge.textContent = '0 chars';
        }

        // 6-Trit LEDs
        const trits = step.acc_trits || [0, 0, 0, 0, 0, 0];
        vTritValue.textContent = (step.acc >= 0 ? '+' : '') + step.acc + ' (' + (step.acc_dozenal || '0z000') + ')';
        for (let i = 0; i < 6; i++) {
          const led = document.getElementById('trit-' + i);
          if (led) {
            const val = trits[i] || 0;
            led.className = 'trit-led ' + (val > 0 ? 'pos' : (val < 0 ? 'neg' : 'zer'));
            const valSpan = led.querySelector('.trit-val');
            if (valSpan) valSpan.textContent = (val > 0 ? '+1' : (val < 0 ? '-1' : '0'));
          }
        }

        // Assembly highlighting
        const lines = vamosAsmOutput.querySelectorAll('.asm-line');
        lines.forEach(lineEl => {
          const lpc = parseInt(lineEl.getAttribute('data-pc'), 10);
          if (lpc === step.pc) {
            lineEl.classList.add('active');
            lineEl.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
          } else {
            lineEl.classList.remove('active');
          }
        });
      }

      function stopPlay() {
        if (playTimer) {
          clearInterval(playTimer);
          playTimer = null;
          vBtnPlay.classList.remove('active');
          vPlayIcon.textContent = '▶';
          vPlayText.textContent = 'Animar';
        }
      }

      function startPlay() {
        stopPlay();
        if (!currentTrace || currentTrace.length === 0) return;
        if (currentStepIndex >= currentTrace.length - 1) {
          currentStepIndex = 0;
          renderStep(0);
        }
        vBtnPlay.classList.add('active');
        vPlayIcon.textContent = '⏸';
        vPlayText.textContent = 'Pausar';

        const speed = parseInt(vSpeedSelect.value, 10) || 150;
        playTimer = setInterval(() => {
          if (currentStepIndex < currentTrace.length - 1) {
            renderStep(currentStepIndex + 1);
          } else {
            stopPlay();
          }
        }, speed);
      }

      vBtnPlay.addEventListener('click', () => {
        if (playTimer) stopPlay();
        else startPlay();
      });
      vBtnPrev.addEventListener('click', () => {
        stopPlay();
        renderStep(currentStepIndex - 1);
      });
      vBtnNext.addEventListener('click', () => {
        stopPlay();
        renderStep(currentStepIndex + 1);
      });
      vBtnStart.addEventListener('click', () => {
        stopPlay();
        renderStep(0);
      });
      vBtnEnd.addEventListener('click', () => {
        stopPlay();
        renderStep(currentTrace.length - 1);
      });
      vStepSlider.addEventListener('input', () => {
        stopPlay();
        renderStep(parseInt(vStepSlider.value, 10));
      });

      function renderAssemblyWithPC(asmText) {
        let pc = 0;
        const lines = asmText.split('\n');
        let html = '';
        lines.forEach((line) => {
          const trimmed = line.trim();
          let linePC = pc;
          let isInst = false;
          if (trimmed && !trimmed.startsWith('//') && !trimmed.startsWith(';') && !trimmed.endsWith(':')) {
            isInst = true;
            const parts = trimmed.split(/\s+/);
            if (parts.length > 1 && !parts[1].startsWith('//') && !parts[1].startsWith(';')) {
              pc += 2;
            } else {
              pc += 1;
            }
          }
          const escaped = line.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
          html += '<span class="asm-line" data-pc="' + (isInst ? linePC : -1) + '">' + escaped + '</span>';
        });
        vamosAsmOutput.innerHTML = html;
      }

      async function runVamos(autoPlay) {
        const code = vamosCode.value.trim();
        if (!code) return;

        stopPlay();
        vamosCompileStatus.textContent = 'COMPILANDO...';
        vamosCompileStatus.style.borderColor = 'var(--accent-yellow)';
        vamosCompileStatus.style.color = 'var(--accent-yellow)';

        try {
          const res = await fetch('/api/vamos', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ source: code })
          });
          const data = await res.json();

          if (!res.ok || !data.success) {
            const err = data.error || 'Erro na compilação';
            vamosCompileStatus.textContent = 'ERRO';
            vamosCompileStatus.style.borderColor = 'var(--accent-red)';
            vamosCompileStatus.style.color = 'var(--accent-red)';
            vamosAsmOutput.textContent = '; [ERRO DE COMPILAÇÃO]\n; ' + err;
            vamosAsmBadge.textContent = 'Erro';
            return;
          }

          vamosCompileStatus.textContent = 'SUCESSO';
          vamosCompileStatus.style.borderColor = 'var(--accent-green)';
          vamosCompileStatus.style.color = 'var(--accent-green)';

          renderAssemblyWithPC(data.assembly || '; Sem código gerado');
          vamosAsmBadge.textContent = (data.words_count || 0) + ' palavras';

          const ex = data.execution;
          if (ex) {
            currentTrace = ex.trace || [];
            vStepSlider.min = 0;
            vStepSlider.max = Math.max(0, currentTrace.length - 1);
            vStepSlider.value = 0;

            // Variables table
            const vars = ex.variables || {};
            const keys = Object.keys(vars);
            if (keys.length === 0) {
              vamosVarsTbody.innerHTML = '<tr><td colspan="5" style="text-align: center; color: var(--text-muted); padding: 10px;">Nenhuma variável declarada</td></tr>';
            } else {
              let rowsHtml = '';
              for (const k of keys) {
                const item = vars[k];
                const decSign = item.decimal >= 0 ? '+' : '';
                rowsHtml += '<tr>' +
                  '<td><strong style="color:var(--accent-cyan);">' + k + '</strong></td>' +
                  '<td>$' + item.address.toString(16).padStart(4, '0') + ' (' + item.address + ')</td>' +
                  '<td>' + decSign + item.decimal + '</td>' +
                  '<td style="color:var(--accent-purple);">' + item.dozenal + '</td>' +
                  '<td style="color:var(--accent-cyan);">' + (item.trits || '') + '</td>' +
                '</tr>';
              }
              vamosVarsTbody.innerHTML = rowsHtml;
            }

            if (autoPlay) {
              startPlay();
            } else {
              renderStep(currentTrace.length - 1);
            }
          }
        } catch (err) {
          vamosCompileStatus.textContent = 'ERRO HTTP';
          vamosCompileStatus.style.borderColor = 'var(--accent-red)';
          vamosCompileStatus.style.color = 'var(--accent-red)';
          vamosAsmOutput.textContent = '; [ERRO HTTP]\n; ' + err.message;
        }
      }

      btnVamosRun.addEventListener('click', () => runVamos(true));

      vamosCode.addEventListener('keydown', (e) => {
        if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
          e.preventDefault();
          runVamos(true);
        }
      });

      // Auto-compile default preset on startup unless URL hash contains code/challenge
      if (!checkUrlParams()) {
        runVamos(false);
      }
    })();
