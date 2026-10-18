# r36s_a11y (Python)

Biblioteca-ponte Python para jogos acessíveis no R36S. Python 3.7+, só
biblioteca padrão. No console usa:

- o controle direto em `/dev/input` (sem exclusividade, o sistema continua
  recebendo os botões);
- o Speech Dispatcher, com a voz que o usuário ajustou;
- a OpenAL Soft do sistema, com HRTF quando disponível;
- a `libvorbisfile` do sistema para arquivos OGG.

O mesmo jogo roda no PC com Windows, sem mudar nada: veja
[Rodando no PC](#rodando-no-pc).

## Instalação

Copie a pasta `r36s_a11y` para dentro do seu jogo, ao lado do `main.py`, ou
crie o jogo pelo [MAGE](https://github.com/gogabs/mage),
que faz isso por você.

## Estrutura de um jogo

```
meu-jogo/
  game.json
  main.py
  r36s_a11y/
  sounds/
```

`game.json` (veja o [contrato](../docs/contrato.md)):

```json
{
  "id": "meu-jogo",
  "name": "Meu Jogo",
  "version": "0.1.0",
  "contract": 1,
  "runtime": "python3",
  "exec": "main.py"
}
```

## API

### Game

```python
game = a11y.Game(fps=60, repeat=False, axes=False, deadzone=0.2)
```

| Membro | Descrição |
|---|---|
| `@game.on_button` | `fn(botão, estado, ms)`; estado `"down"`, `"up"` ou `"repeat"` (com `repeat=True`) |
| `@game.on_axis` | `fn(eixo, valor, ms)`; eixos `lx ly rx ry` de -1 a 1 (com `axes=True`) |
| `@game.on_update` | `fn(dt)` a cada quadro |
| `@game.on_start` / `@game.on_quit` | Início e saída (salve aqui) |
| `game.say(texto, queue=False, on_done=None)` | Fala; interrompe a anterior, a menos que `queue=True`. `on_done(terminou)` ao fim |
| `game.hush()` | Cala a fala |
| `game.after(segundos, fn)` | Chama `fn()` depois de um tempo |
| `game.sound` | O áudio (abaixo) |
| `game.save_dir`, `game.lang`, `game.dir` | Pastas e idioma |
| `game.quit()` | Sai do jogo |

Botões: `up down left right a b x y l1 r1 l2 r2 l3 r3 start select`.
Convenção: `b` confirma, `a` volta, `start` pausa, `select` repete. Use
`a11y.is_confirm(b)` e `a11y.is_back(b)`. Fn, volume e Power nunca chegam
ao jogo.

### Áudio

```python
passo = game.sound.load("passo", "sounds/passo.ogg")   # WAV ou OGG
voz = passo.play(loop=False, volume=1.0, pitch=1.0, pos=(x, y, z))
voz = passo.play(pan=-1.0)                             # atalho: -1 esquerda, 1 direita
voz.set_position(x, y, z); voz.set_volume(0.5, fade=0.3); voz.set_pitch(1.2)
voz.stop(fade=0.5); voz.playing
game.sound.listener((x, y, z), facing=90)              # graus; 0 = frente, 90 = direita
```

Ambiente (reverberação EFX da OpenAL), para sons tocados com `pos` ou
`reverb=True`:

```python
game.sound.set_reverb("igreja")   # quarto, corredor, estacionamento, cozinha, metro, igreja, floresta
game.sound.set_reverb(decay=2.0, gain=0.3)   # ou parâmetros soltos
game.sound.set_reverb(None)       # desliga
```

Coordenadas em metros: `x` positivo à direita, `y` para cima, `z` negativo
à frente. Só sons **mono** são posicionados. Sons de banda larga (ruído,
estalos) ficam mais fáceis de localizar na frente e atrás do que tons puros.

### Menu

```python
menu = a11y.Menu(game, "Menu principal", ["Jogar", "Instruções", "Sair"],
                 on_select=lambda i, item: ..., on_back=game.quit,
                 sounds={"move": game.sound.load("move", "sounds/move.wav")})
menu.open()
```

Cima/baixo navegam e falam "Jogar, 1 de 3"; B confirma; A volta; Select
repete. Enquanto aberto, o menu recebe os botões no lugar do jogo.

## Gráficos (opcionais)

Formas, sprites e texto pela SDL2 do console; veja [docs/graficos.md](../docs/graficos.md).

```python
heroi = game.gfx.image("sprites/heroi.png")

@game.on_draw
def desenhar(gfx):
    gfx.clear("#101830")
    gfx.text("Sala 1", 320, 20, align="center")
    gfx.draw(heroi, 300, 200, scale=2)
```

## Exemplo

[`examples/onde-esta-o-bip`](examples/onde-esta-o-bip): um som toca à
esquerda, à frente, à direita ou atrás; aperte o direcional do lado certo.

No console:

```bash
scp -r python r36s:/tmp/sdk
ssh r36s "cd /tmp/sdk/examples/onde-esta-o-bip && python3 main.py"
```

## Rodando no PC

No Windows, o mesmo jogo roda sem mudar nada, numa janela. Serve para
testar rápido e também para lançar o jogo para PC; o teste final de um jogo
do console continua no console, que é mais lento e usa outra voz.

### Preparar (uma vez)

Baixe as DLLs para dentro da biblioteca (SDL2, OpenAL Soft, libsndfile e o
cliente do NVDA, todas de lançamentos oficiais e conferidas pelo SHA-256):

```bat
python tools\preparar_windows.py
python tools\preparar_windows.py C:\caminho\do\meu-jogo   :: a cópia dentro de um jogo
```

Elas ficam em `r36s_a11y\dll\`. Sem elas o jogo ainda abre, no modo antigo:
teclado só pelo terminal, controle só XInput e nenhum som.

```bat
cd examples\onde-esta-o-bip
python main.py
```

### Janela

- Sempre abre uma janela, mesmo em jogo só de áudio. O título é o nome do
  jogo, que o leitor de tela fala ao trocar de janela.
- Jogo sem `@game.on_draw` mostra o nome e a última fala, como legenda.
- A área de desenho continua 640x480, ampliada para o tamanho da janela com
  tarjas. A janela pode ser redimensionada e maximizada, e respeita a escala
  do Windows (150%, 200%) sem ficar borrada.
- F11 ou Alt + Enter: tela cheia. Fechar a janela ou Ctrl + C: sair.

### Teclado, mouse e controles

| Teclado | R36S |
|---|---|
| Setas | Direcional |
| Enter ou Z | B (confirmar) |
| Backspace ou X | A (voltar) |
| S | X (de cima) |
| A | Y (da esquerda) |
| Q / W | L1 / R1 |
| 1 / 2 | L2 / R2 |
| 3 / 4 | L3 / R3 |
| Esc | Start (pausa) |
| Tab | Select (repetir) |
| Ctrl | Fn: o que for apertado com ele não chega ao jogo |

Qualquer controle que a SDL conheça funciona (Xbox, PlayStation, Nintendo,
8BitDo e genéricos), com conexão e desconexão a qualquer hora ("Controle
conectado" é falado), analógicos (`lx ly rx ry`) e vibração
(`game.rumble()`). Os botões valem pelo lugar: o de baixo é o B do R36S em
qualquer controle. O botão central (Xbox, PS, Home) é o Fn.

### Ações: um mapa de teclas por jogo

Em vez de pensar nos botões do R36S, o jogo pode dar nomes às ações e dizer
o que aperta cada uma em cada dispositivo:

```python
game.actions({
    "pular":  {"key": "space", "r36s": "b"},              # controle: o botão de baixo
    "atirar": {"key": ["z", "x"], "r36s": "y", "pad": "x", "mouse": "left"},
    "pausa":  "start",                                      # só o botão do R36S
})

@game.on_action
def acao(nome, estado, ms):
    if nome == "pular" and estado == a11y.DOWN:
        ...

game.say("Aperte %s para pular." % game.label("pular"))
```

`game.label("pular")` diz o nome no que o jogador está usando agora
(`game.device`): "espaço" no teclado, "B" no R36S, "A" no Xbox, "xis" no
PlayStation, "B" num controle da Nintendo. `game.label("atirar", every=True)`
junta todas as opções com "ou".

| Origem | Valores |
|---|---|
| `r36s` | `up down left right a b x y l1 r1 l2 r2 l3 r3 start select` |
| `key` | Nome da tecla: `space`, `return` (ou `enter`), `escape`, `tab`, `backspace`, letras e números (`a`, `ç`, `1`), `up` `down` `left` `right`, `f1`...`f12`, `lshift` `rshift` (ou `shift`), `lalt` `ralt` (ou `alt`), `kp1`, `kp enter`... Ctrl não pode (é o Fn) |
| `pad` | Nome do Xbox, pelo lugar: `a` (de baixo) `b` `x` `y`, `back` `start`, `leftshoulder` `rightshoulder` (`lb` `rb`), `lefttrigger` `righttrigger` (`lt` `rt`), `leftstick` `rightstick`, `dpup` `dpdown` `dpleft` `dpright` |
| `mouse` | `left`, `middle`, `right` |

Sem `key` ou sem `pad`, valem as teclas e o botão que ficam no lugar do botão
`r36s` da ação. Com `Game(repeat=True)`, ações seguradas repetem como os
botões. Um menu aberto (`a11y.Menu`) fica com a entrada: as ações só chegam
quando ele fecha. `on_button` continua funcionando, e o menu fala a ajuda
com os nomes do dispositivo ("Enter confirma" no teclado).

### Outros eventos do PC

| Membro | Descrição |
|---|---|
| `@game.on_key` | `fn(tecla, estado, ms)`: toda tecla, com os nomes acima; estado `"down"`, `"up"` ou `"repeat"` (repetição do Windows) |
| `@game.on_text` | `fn(texto)`: texto digitado, com acentos, para nomes e chat |
| `@game.on_mouse` | `fn(tipo, x, y, valor)`: `"move"`, `"down"` / `"up"` (valor `"left"`, `"middle"`, `"right"`) e `"wheel"` (valor = cliques); x e y na área de 640x480 |
| `@game.on_focus` | `fn(com_foco)`: a janela ganhou ou perdeu o foco (ao perder, tudo que estava apertado é solto) |
| `@game.on_device` | `fn(evento, família, nome)`: controle `"added"` ou `"removed"`; família `"xbox"`, `"playstation"` ou `"nintendo"`. Com esta função, o jogo não fala "Controle conectado" sozinho |
| `game.device` | `"r36s"`, `"keyboard"`, `"xbox"`, `"playstation"` ou `"nintendo"` |
| `game.rumble(força, segundos)` | Vibra o último controle usado |
| `game.fullscreen(on=None)` | Liga, desliga ou alterna a tela cheia |
| `game.button_label(botão)` | Nome de um botão do R36S no dispositivo em uso |

No console nada disso muda: as ações vêm dos botões do R36S, e os eventos
do PC nunca acontecem.

### Fala

Com o **NVDA** aberto, o jogo fala por ele, com a voz e a velocidade que o
jogador escolheu, e manda o texto também para a linha braille. Ctrl cala a
fala, como sempre no NVDA. O NVDA não avisa quando termina de falar, então o
`on_done` de `game.say` é chamado depois de um tempo estimado.

Sem o NVDA, usa a voz do Windows no idioma do jogo (pt-BR, por exemplo, a
Maria), pelo PowerShell que já vem no Windows; ela começa a falar um ou dois
segundos depois de abrir o jogo. Variáveis:

- `A11Y_FALA`: `nvda`, `sapi` (voz do Windows) ou `terminal` (só escreve no
  terminal).
- `A11Y_VOZ`: escolhe a voz do Windows pelo nome, por exemplo
  `set A11Y_VOZ=Microsoft Daniel Desktop`.

### Áudio

OpenAL Soft (`soft_oal.dll`), igual ao console, com HRTF. OGG é decodificado
pela libsndfile, já que não existe uma vorbisfile.dll oficial para Windows.

### Linux

O modo PC também roda no Linux com `A11Y_PLATAFORMA=pc` e a SDL2 instalada,
com a fala pelo Speech Dispatcher (o mesmo do Orca). Sem a variável, o
Linux é tratado como o console.

## Ações no game.json

As ações podem ficar no `game.json`, e o `Game()` as registra sozinho, no
mesmo formato de `game.actions` (veja [Ações](#ações-um-mapa-de-teclas-por-jogo)):

```json
"actions": {
  "bip": {"r36s": "b", "key": ["space", "return"]},
  "sair": {"r36s": "a", "key": "escape"}
}
```

Assim dá para trocar teclas sem mexer no código.

## Criar, rodar e exportar jogos

O [MAGE](https://github.com/gogabs/mage) cria um jogo
a partir de um modelo, roda, confere e exporta um zip para o cartão do R36S e
um `.exe` para Windows que abre sem instalar nada. Ela é opcional: um
`main.py` com a pasta `r36s_a11y` ao lado continua sendo um jogo completo.

## Testes

```bash
python -m unittest discover -s tests
```
