# Gráficos

Os jogos da plataforma são feitos para serem jogados só pelo som. Gráficos
são um extra, para quem enxerga parcialmente, para jogar junto com alguém
que enxerga, ou só para ficar bonito. **Regra do contrato: nenhuma
informação pode existir só na tela.**

## Como funciona

As bibliotecas de Python, Go e C# desenham pela SDL2 que já vem no console,
carregada na hora de rodar (sem instalar nada, sem cgo):

| Biblioteca do sistema | Para quê |
|---|---|
| SDL2 | Janela em tela cheia (KMSDRM) e renderização acelerada (GLES2 na Mali) |
| SDL2_gfx | Círculos, linhas grossas, polígonos |
| SDL2_image | PNG (e JPG, BMP...) |
| SDL2_ttf | Texto com fontes TrueType |

No LÖVE, use o `love.graphics` normalmente; veja [o fim desta página](#löve).

### Regras

- A tela só é aberta se o jogo desenhar (registrar o evento de desenho ou
  usar os gráficos). Jogo só de áudio não ocupa a tela.
- A área de desenho é sempre **640x480**, com (0, 0) no canto superior
  esquerdo. Ela é escalada para a tela real, então o jogo não precisa saber
  a resolução.
- O desenho acontece uma vez por quadro, logo depois do update. A tela é
  limpa (preto) antes de cada quadro.
- Cores: vermelho, verde, azul e opacidade de 0 a 255, ou `"#rrggbb"`.
- Imagens e fontes são relativas à pasta do jogo. A fonte padrão é a DejaVu
  Sans do console, 24 px.
- Sem SDL (no PC sem as DLLs, por exemplo), os gráficos ficam desligados e o
  jogo segue funcionando.
- A captura de tela (`screenshot`) salva o quadro em PNG, para conferir o
  desenho sem olhar para o console.

## Operações

| Operação | Python | Go | C# |
|---|---|---|---|
| Desenhar a cada quadro | `@game.on_draw` com `def f(gfx)` | `g.OnDraw(func(gfx *a11y.Graphics))` | `game.Draw += gfx => ...` |
| Acessar antes do laço | `game.gfx` | `g.Gfx()` | `game.Gfx` |
| Limpar | `gfx.clear(cor)` | `gfx.Clear(cor)` | `gfx.Clear(cor)` |
| Retângulo | `gfx.rect(x, y, w, h, color=, fill=)` | `Rect` / `RectLine` | `Rect(x, y, w, h, cor, fill:)` |
| Linha | `gfx.line(x1, y1, x2, y2, color=, width=)` | `Line(x1, y1, x2, y2, largura, cor)` | `Line(x1, y1, x2, y2, cor, largura)` |
| Círculo | `gfx.circle(x, y, r, color=, fill=)` | `Circle` / `CircleLine` | `Circle(x, y, r, cor, fill:)` |
| Polígono | `gfx.polygon([(x, y), ...], color=, fill=)` | `Polygon` / `PolygonLine` | `Polygon(pontos, cor, fill:)` |
| Ponto | `gfx.point(x, y, color=)` | `Point(x, y)` (cor de `SetColor`) | `Point(x, y, cor)` |
| Carregar imagem | `gfx.image("sprites/heroi.png")` | `LoadImage(caminho)` | `LoadImage(caminho)` |
| Desenhar imagem | `gfx.draw(img, x, y, src=, scale=, rotation=, flip_x=, flip_y=, alpha=, tint=)` | `Draw(img, x, y, Src(...), Scale(...), Rotate(...), FlipX(), FlipY(), Alpha(...), Tint(...))` | `Draw(img, x, y, src:, scale:, rotation:, flipX:, flipY:, alpha:, tint:)` |
| Carregar fonte | `gfx.font("fontes/x.ttf", 32)` | `LoadFont(caminho, tamanho)` | `LoadFont(caminho, tamanho)` |
| Texto | `gfx.text(texto, x, y, font=, color=, align=)` | `Text(texto, x, y, fonte, cor, AlignCenter)` | `Text(texto, x, y, fonte, cor, Align.Center)` |
| Medir texto | `gfx.measure(texto, font)` | `Measure(texto, fonte)` | `Measure(texto, fonte)` |
| Capturar | `gfx.screenshot("quadro.png")` | `Screenshot(caminho)` | `Screenshot(caminho)` |

### Sprites

`src` recorta um quadro de uma folha de sprites: `(x, y, largura,
altura)` dentro da imagem. A rotação é em graus, em torno do centro.
`tint` multiplica as cores (branco não muda nada), útil para piscar de
vermelho quando o personagem leva dano.

```python
folha = game.gfx.image("sprites/zumbi.png")   # 4 quadros de 32x32 lado a lado
quadro = 0

@game.on_update
def animar(dt):
    global quadro
    quadro = (quadro + dt * 8) % 4

@game.on_draw
def desenhar(gfx):
    gfx.clear("#202020")
    gfx.draw(folha, 300, 200, src=(int(quadro) * 32, 0, 32, 32), scale=3)
    gfx.text("Sala 3", 320, 20, align="center")
```

O exemplo [`python/examples/onde-esta-o-bip`](../python/examples/onde-esta-o-bip)
desenha a rodada, as setas e um sprite, sem mudar nada do jogo por som.

## Desempenho

O console é um RK3326 (4 núcleos ARM, GPU Mali-G31). Imagens e textos
viram texturas na GPU; desenhar centenas de sprites por quadro é tranquilo.
O que custa:

- Criar texto novo: cada texto diferente vira uma textura. A biblioteca
  guarda os textos desenhados nos últimos 60 quadros, então texto que muda
  todo quadro (um cronômetro com milésimos, por exemplo) gera trabalho.
- Em Python, cada chamada de desenho atravessa o ctypes. Na casa de
  centenas de chamadas por quadro, prefira folhas de sprites a muitas formas
  pequenas.

## LÖVE

O LÖVE já tem gráficos completos (`love.graphics`), então a biblioteca não
acrescenta nada. Para usar, ligue a janela no `conf.lua`:

```lua
function love.conf(t)
  t.window.width, t.window.height = 640, 480
  t.window.fullscreen = true
  t.modules.joystick = false   -- o controle continua sendo lido pela a11y.lua
end
```

e desenhe em `love.draw`. Jogos LÖVE só de áudio continuam com
`t.window = false`.
