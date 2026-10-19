# Compatibilidade nativa e integração gráfica

Alterações sobre a main `76bb5a8`, preservando gráficos e controles Windows
recentes. Reconciliam extensões usadas no console existente, sem alterar o
contrato `game.json` ou APIs atuais de `Sound`.

- `GamepadReader` conserva fragmentos de `input_event` entre leituras.
- Analógicos acionados com FN ficam suprimidos até retornarem ao centro,
  e repetição de botões não é despachada enquanto FN estiver pressionado.
- Fala respeita SPEECHD_ADDRESS, reaplica mudanças de voz/velocidade ao falar
  e permite reconectar após falha. Idiomas estrangeiros usam eSpeak quando
  a voz RHVoice configurada é portuguesa.
- `Audio.stream(name, path)` adiciona decodificação/buffers limitados, mantendo
  load()/Sound existentes, fade/pausa, reciclagem de fontes e encerramento.

Cinco testes de compatibilidade passam no PC, sem acessar hardware, cobrindo
eventos fragmentados, FN, atualização da voz e disponibilidade das APIs.
A versão reconciliada também foi instalada no Dead vs. Alive: prova nativa
com sete mapas/armas, fala/OpenAL, pausa/retomada, liberação de vídeo e saída.
O teste nativo usa saves privados e o perfil de áudio real do Luma.
Não foi executada campanha prolongada nem validada a nova extensão em Windows.

O jogo e a plataforma têm PRs separados: gogabs/dead-vs-alive#1 e
gogabs/r36s-a11y#5. A plataforma conserva a curva RHVoice 3× e limita as
animações em economia; o jogo usa a API gráfica do Mage Core com texturas cacheadas.
