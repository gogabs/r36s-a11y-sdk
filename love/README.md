# a11y.lua (LÖVE)

Biblioteca-ponte para jogos acessíveis em [LÖVE](https://love2d.org) 11.x
no R36S. É um arquivo só, `a11y.lua`: copie para a pasta do jogo.

O console já traz o LÖVE 11.4 (`/opt/love2d/love`), então não há nada para
instalar nem compilar. No console, a biblioteca:

- lê o controle direto em `/dev/input` pelo FFI do LuaJIT (sem
  exclusividade, o sistema continua recebendo os botões);
- fala pelo Speech Dispatcher, com a voz que o usuário escolheu;
- toca som 3D pelo `love.audio` (OpenAL), com eco por ambiente. A galeria
  liga o HRTF da OpenAL ao abrir o jogo.

Fora do console, o jogo roda, mas sem controle; a fala vai para o terminal.

## Começando

```
meu-jogo/
  game.json
  conf.lua
  main.lua
  a11y.lua
  sounds/
```

`conf.lua` (jogo só de áudio):

```lua
function love.conf(t)
  t.window = false            -- não ocupa a tela
  t.modules.joystick = false  -- o controle é lido pela biblioteca
end
```

`main.lua`:

```lua
local a11y = require("a11y")
local pulo

function love.load()
  a11y.init()
  pulo = a11y.sound.load("pulo", "sounds/pulo.ogg")
end

function a11y.button(b, state, ms)
  if state == "down" and a11y.is_confirm(b) then
    pulo:play({ pan = -0.5 })
    a11y.say("Pulou!")
  end
end

function love.update(dt)
  a11y.update(dt)   -- obrigatório: lê o controle, entrega falas e temporizadores
end
```

`game.json` com `"runtime": "love"` e `"exec": "."` (veja o
[contrato](../docs/contrato.md)). Para instalar, copie a pasta para
`/opt/a11y/games/meu-jogo`.

## API

| Membro | Descrição |
|---|---|
| `a11y.init({ repeat_buttons, axes, deadzone })` | Chame em `love.load`; as opções são opcionais |
| `a11y.update(dt)` | Chame em `love.update` |
| `function a11y.button(b, state, ms)` | Defina: `state` é `"down"`, `"up"` ou `"repeat"` (com `repeat_buttons`) |
| `function a11y.axis(name, value, ms)` | Defina: `lx ly rx ry` de -1 a 1 (com `axes`) |
| `a11y.say(texto, { queue = false, on_done = function(terminou) end })` | Fala; interrompe a anterior a menos que `queue` |
| `a11y.say_then(texto, depois)` | Fala e chama `depois` ao fim |
| `a11y.hush()` | Cala a fala |
| `a11y.after(segundos, fn)` | Chama `fn` depois de um tempo |
| `a11y.save_dir`, `a11y.lang`, `a11y.id` | Pasta de saves, idioma, identificador |
| `a11y.is_confirm(b)`, `a11y.is_back(b)` | `b` confirma, `a` volta |

Botões: `up down left right a b x y l1 r1 l2 r2 l3 r3 start select`.
Convenção: `b` confirma, `a` volta, `start` pausa, `select` repete. Fn,
volume e Power nunca chegam ao jogo. Para sair, `love.event.quit()`.

Salve em `a11y.save_dir` com `io.open`, a cada mudança importante: o
sistema pode fechar o jogo a qualquer momento.

### Som

```lua
local passo = a11y.sound.load("passo", "sounds/passo.ogg")  -- WAV ou OGG
local voz = passo:play({ pos = { x, y, z }, volume = 0.8, pitch = 1.1 })
voz = passo:play({ pan = -1, loop = true })
voz:set_position(x, y, z); voz:set_volume(0.5, 0.3); voz:set_pitch(1.2)
voz:stop(0.5); voz:playing()
a11y.sound.listener({ 0, 0, 0 }, 90)   -- graus; 0 = frente, 90 = direita
a11y.sound.set_reverb("igreja")        -- quarto, corredor, estacionamento, cozinha, metro, igreja, floresta; nil desliga
```

Coordenadas em metros: `x` positivo à direita, `y` para cima, `z` negativo
à frente. Só sons **mono** são posicionados. `reverb = true/false` força o
ambiente num som (padrão: só sons com `pos`).

### Menu

```lua
local menu = a11y.Menu.new("Menu principal", { "Jogar", "Sair" },
  function(i, item) ... end,
  { on_back = love.event.quit, move_sound = clique })
menu:show()          -- item 1, falando as instruções
menu:show(2, false)
menu:close()
```

## Gráficos

Use o `love.graphics` normalmente; veja [docs/graficos.md](../docs/graficos.md#löve)
para o `conf.lua` com a janela ligada.

## Exemplo

O jogo de exemplo, com som e gráficos, está em Python:
[`python/examples/onde-esta-o-bip`](../python/examples/onde-esta-o-bip).
A API desta biblioteca tem os mesmos nomes, adaptados à convenção da linguagem.

## Testes

No console:

```bash
cd love/tests && LD_LIBRARY_PATH=/opt/love2d/lib /opt/love2d/love .
```
