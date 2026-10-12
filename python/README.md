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

Copie a pasta `r36s_a11y` para dentro do seu jogo, ao lado do `main.py`.

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

No Windows, a biblioteca troca os bastidores sozinha e o jogo não muda.
Serve para testar rápido; o teste final continua no console, que é mais
lento e usa outra voz.

```bat
cd examples\onde-esta-o-bip
python main.py
```

Rode no Prompt de Comando ou no Terminal do Windows, com a janela em foco:
o teclado é lido do console. Se o jogo abrir a tela (`@game.on_draw`), as
teclas passam a vir da janela do jogo, que fica com o foco.

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
| Ctrl + C | Sai do jogo |

**Controle de Xbox** (ou qualquer um compatível com XInput) também
funciona, junto com o teclado, inclusive os analógicos (`lx ly rx ry`). Os
botões valem pela posição: o de baixo (A no Xbox) é o B do R36S, o da
direita (B no Xbox) é o A. LT e RT são L2 e R2, View é Select, Menu é
Start e o botão Xbox é Fn (o Windows pode abrir a Game Bar com ele; Ctrl
no teclado faz o mesmo).

**Fala**: usa a voz do Windows no idioma do jogo (pt-BR, por exemplo, a
Maria), pelo PowerShell que já vem no Windows. A voz começa a falar um ou
dois segundos depois de abrir o jogo. Variáveis:

- `A11Y_VOZ`: escolhe a voz pelo nome, por exemplo
  `set A11Y_VOZ=Microsoft Daniel Desktop`.
- `A11Y_FALA=terminal`: em vez de falar, escreve no terminal, para o NVDA
  ler.

**Áudio**: precisa da OpenAL Soft. Baixe os binários em
[openal-soft.org](https://openal-soft.org/#download) e copie
`bin\Win64\soft_oal.dll` para dentro da pasta `r36s_a11y` (ou para
`C:\Windows\System32`). Para OGG, faça o mesmo com a `vorbisfile.dll` e
suas dependências; ou use WAV no PC.

## Testes

```bash
python -m unittest discover -s tests
```
