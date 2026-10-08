# r36s_a11y (Python)

Biblioteca-ponte Python para jogos acessíveis no R36S. Python 3.7+, só
biblioteca padrão. No console usa:

- o controle direto em `/dev/input` (sem exclusividade, o sistema continua
  recebendo os botões);
- o Speech Dispatcher, com a voz que o usuário ajustou;
- a OpenAL Soft do sistema, com HRTF quando disponível;
- a `libvorbisfile` do sistema para arquivos OGG.

Fora do console (no PC), a fala aparece no terminal e o áudio usa a OpenAL
se ela estiver instalada. Ainda não há controle por teclado no PC.

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

## Exemplo

[`examples/onde-esta-o-bip`](examples/onde-esta-o-bip): um som toca à
esquerda, à frente, à direita ou atrás; aperte o direcional do lado certo.

No console:

```bash
scp -r python r36s:/tmp/sdk
ssh r36s "cd /tmp/sdk/examples/onde-esta-o-bip && python3 main.py"
```

## Testes

```bash
python -m unittest discover -s tests
```
