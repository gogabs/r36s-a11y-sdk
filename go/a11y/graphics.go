package a11y

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Gráficos 2D opcionais pela SDL2 do sistema: formas, sprites e texto.
// A área de desenho é sempre 640x480 (escalada para a tela real); (0, 0) é o
// canto superior esquerdo. Os gráficos são um extra: o jogo continua tendo de
// ser jogável só pelo som.

const (
	ScreenWidth  = 640
	ScreenHeight = 480

	sdlInitVideo          = 0x20
	sdlWindowShown        = 0x4
	sdlWindowFullscreenDT = 0x1001
	sdlRendererAccel      = 0x2
	sdlBlendBlend         = 0x1
	sdlWindowPosCentered  = 0x2FFF0000
	imgInitPNG            = 0x2
	pixelFormatARGB8888   = 0x16762004
)

var defaultFonts = []string{
	"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	"C:/Windows/Fonts/segoeui.ttf",
	"C:/Windows/Fonts/arial.ttf",
}

type sdlRect struct{ X, Y, W, H int32 }

// sdlSurface é o começo de SDL_Surface (64 bits), até o ponteiro dos pixels.
type sdlSurface struct {
	Flags  uint32
	_      [4]byte
	Format uintptr
	W, H   int32
	Pitch  int32
	_      [4]byte
	Pixels uintptr
}

// Funções da SDL2, SDL2_image, SDL2_ttf e SDL2_gfx, preenchidas por loadSDL.
var (
	sdlInit                    func(flags uint32) int32
	sdlCreateWindow            func(title string, x, y, w, h int32, flags uint32) uintptr
	sdlCreateRenderer          func(win uintptr, index int32, flags uint32) uintptr
	sdlRenderSetLogicalSize    func(r uintptr, w, h int32) int32
	sdlSetRenderDrawBlendMode  func(r uintptr, mode int32) int32
	sdlSetRenderDrawColor      func(r uintptr, red, green, blue, alpha uint8) int32
	sdlRenderClear             func(r uintptr) int32
	sdlRenderFillRect          func(r uintptr, rect *sdlRect) int32
	sdlRenderDrawRect          func(r uintptr, rect *sdlRect) int32
	sdlRenderPresent           func(r uintptr)
	sdlPumpEvents              func()
	sdlShowCursor              func(toggle int32) int32
	sdlCreateTextureFromSurf   func(r, surface uintptr) uintptr
	sdlFreeSurface             func(surface uintptr)
	sdlQueryTexture            func(t uintptr, format *uint32, access, w, h *int32) int32
	sdlRenderCopyEx            func(r, t uintptr, src, dst *sdlRect, angle float64, center uintptr, flip int32) int32
	sdlSetTextureAlphaMod      func(t uintptr, a uint8) int32
	sdlSetTextureColorMod      func(t uintptr, r, g, b uint8) int32
	sdlSetTextureBlendMode     func(t uintptr, mode int32) int32
	sdlDestroyTexture          func(t uintptr)
	sdlDestroyRenderer         func(r uintptr)
	sdlDestroyWindow           func(w uintptr)
	sdlQuitSubSystem           func(flags uint32)
	sdlGetError                func() string
	sdlCreateRGBSurfaceWithFmt func(flags uint32, w, h, depth int32, format uint32) *sdlSurface
	sdlRenderReadPixels        func(r uintptr, rect *sdlRect, format uint32, pixels uintptr, pitch int32) int32
	sdlGetRendererOutputSize   func(r uintptr, w, h *int32) int32
	imgInit                    func(flags int32) int32
	imgLoad                    func(path string) uintptr
	imgSavePNG                 func(surface *sdlSurface, path string) int32
	sdlFreeSurfaceP            func(surface *sdlSurface)
	ttfInit                    func() int32
	ttfOpenFont                func(path string, size int32) uintptr
	ttfRenderUTF8Blended       func(font uintptr, text string, color uint32) uintptr // SDL_Color por valor = 4 bytes
	ttfSizeUTF8                func(font uintptr, text string, w, h *int32) int32
	gfxFilledCircle            func(r uintptr, x, y, rad int16, red, green, blue, alpha uint8) int32
	gfxAACircle                func(r uintptr, x, y, rad int16, red, green, blue, alpha uint8) int32
	gfxThickLine               func(r uintptr, x1, y1, x2, y2 int16, width, red, green, blue, alpha uint8) int32
	gfxAALine                  func(r uintptr, x1, y1, x2, y2 int16, red, green, blue, alpha uint8) int32
	gfxFilledPolygon           func(r uintptr, vx, vy *int16, n int32, red, green, blue, alpha uint8) int32
	gfxAAPolygon               func(r uintptr, vx, vy *int16, n int32, red, green, blue, alpha uint8) int32
	gfxPixel                   func(r uintptr, x, y int16, red, green, blue, alpha uint8) int32
)

// A SDL e o contexto OpenGL ES só funcionam na thread que os criou. Prender
// a goroutine principal à thread principal (no init, antes do main) garante
// que New, Gfx, Run e os callbacks rodem sempre na mesma thread.
func init() { runtime.LockOSThread() }

// Color é uma cor RGBA de 0 a 255.
type Color struct{ R, G, B, A uint8 }

// RGB cria uma cor opaca.
func RGB(r, g, b uint8) Color { return Color{r, g, b, 255} }

// RGBA cria uma cor com transparência.
func RGBA(r, g, b, a uint8) Color { return Color{r, g, b, a} }

// Hex lê "#rrggbb" ou "#rrggbbaa"; cor inválida vira branco.
func Hex(s string) Color {
	s = strings.TrimPrefix(s, "#")
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return RGB(255, 255, 255)
	}
	if len(s) == 8 {
		return Color{uint8(v >> 24), uint8(v >> 16), uint8(v >> 8), uint8(v)}
	}
	return Color{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}

func (c Color) packed() uint32 {
	return uint32(c.R) | uint32(c.G)<<8 | uint32(c.B)<<16 | uint32(c.A)<<24
}

// Point é um ponto na tela.
type Point struct{ X, Y float64 }

// Pt cria um ponto: a11y.Pt(320, 240).
func Pt(x, y float64) Point { return Point{X: x, Y: y} }

// Image é uma imagem (textura). Desenhe com Graphics.Draw.
type Image struct {
	texture       uintptr
	Width, Height int
}

// Font é uma fonte TTF num tamanho.
type Font struct {
	handle uintptr
	Size   int
}

// Graphics desenha na tela. Só funciona dentro de OnDraw.
type Graphics struct {
	Available bool
	title     string
	baseDir   string
	window    uintptr
	renderer  uintptr
	color     Color
	textCache map[textKey]*cachedText
	font      *Font
}

type textKey struct {
	font  uintptr
	text  string
	color Color
}

type cachedText struct {
	image  *Image
	unused int
}

func newGraphics(title, baseDir string) *Graphics {
	g := &Graphics{title: title, baseDir: baseDir, color: RGB(255, 255, 255),
		textCache: map[textKey]*cachedText{}}
	if err := loadSDL(); err != nil {
		fmt.Fprintln(os.Stderr, "[a11y] sem gráficos:", err)
		return g
	}
	if sdlInit(sdlInitVideo) != 0 {
		fmt.Fprintln(os.Stderr, "[a11y] sem gráficos:", sdlGetError())
		return g
	}
	flags := uint32(sdlWindowShown)
	if _, err := os.Stat("/opt/a11y"); err == nil {
		flags = sdlWindowFullscreenDT
	}
	g.window = sdlCreateWindow(title, sdlWindowPosCentered, sdlWindowPosCentered, ScreenWidth, ScreenHeight, flags)
	if g.window == 0 {
		fmt.Fprintln(os.Stderr, "[a11y] sem gráficos:", sdlGetError())
		return g
	}
	g.renderer = sdlCreateRenderer(g.window, -1, sdlRendererAccel)
	if g.renderer == 0 { // sem GPU (testes): renderizador por software
		g.renderer = sdlCreateRenderer(g.window, -1, 0)
	}
	if g.renderer == 0 {
		fmt.Fprintln(os.Stderr, "[a11y] sem gráficos:", sdlGetError())
		return g
	}
	sdlRenderSetLogicalSize(g.renderer, ScreenWidth, ScreenHeight)
	sdlSetRenderDrawBlendMode(g.renderer, sdlBlendBlend)
	sdlShowCursor(0)
	imgInit(imgInitPNG)
	ttfInit()
	g.Available = true
	return g
}

func (g *Graphics) path(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(g.baseDir, p)
}

func (g *Graphics) begin() {
	sdlPumpEvents() // a SDL precisa disso para manter a tela viva
	g.Clear(RGB(0, 0, 0))
	g.color = RGB(255, 255, 255)
}

func (g *Graphics) end() {
	sdlRenderPresent(g.renderer)
	// textos que não foram desenhados por 60 quadros saem do cache
	for k, c := range g.textCache {
		c.unused++
		if c.unused > 60 {
			sdlDestroyTexture(c.image.texture)
			delete(g.textCache, k)
		}
	}
}

// SetColor muda a cor padrão de Point.
func (g *Graphics) SetColor(c Color) { g.color = c }

// Clear pinta a tela inteira.
func (g *Graphics) Clear(c Color) {
	if !g.Available {
		return
	}
	sdlSetRenderDrawColor(g.renderer, c.R, c.G, c.B, c.A)
	sdlRenderClear(g.renderer)
}

// Rect desenha um retângulo preenchido.
func (g *Graphics) Rect(x, y, w, h float64, c Color) { g.rect(x, y, w, h, c, true) }

// RectLine desenha o contorno de um retângulo.
func (g *Graphics) RectLine(x, y, w, h float64, c Color) { g.rect(x, y, w, h, c, false) }

func (g *Graphics) rect(x, y, w, h float64, c Color, fill bool) {
	if !g.Available {
		return
	}
	sdlSetRenderDrawColor(g.renderer, c.R, c.G, c.B, c.A)
	r := sdlRect{int32(x), int32(y), int32(w), int32(h)}
	if fill {
		sdlRenderFillRect(g.renderer, &r)
	} else {
		sdlRenderDrawRect(g.renderer, &r)
	}
}

// Line desenha uma linha; width 1 é suavizada.
func (g *Graphics) Line(x1, y1, x2, y2, width float64, c Color) {
	if !g.Available {
		return
	}
	if width <= 1 {
		gfxAALine(g.renderer, int16(x1), int16(y1), int16(x2), int16(y2), c.R, c.G, c.B, c.A)
		return
	}
	gfxThickLine(g.renderer, int16(x1), int16(y1), int16(x2), int16(y2), uint8(width), c.R, c.G, c.B, c.A)
}

// Circle desenha um círculo preenchido.
func (g *Graphics) Circle(x, y, radius float64, c Color) {
	if g.Available {
		gfxFilledCircle(g.renderer, int16(x), int16(y), int16(radius), c.R, c.G, c.B, c.A)
	}
}

// CircleLine desenha o contorno de um círculo.
func (g *Graphics) CircleLine(x, y, radius float64, c Color) {
	if g.Available {
		gfxAACircle(g.renderer, int16(x), int16(y), int16(radius), c.R, c.G, c.B, c.A)
	}
}

// Polygon desenha um polígono preenchido (3 ou mais pontos).
func (g *Graphics) Polygon(points []Point, c Color) { g.polygon(points, c, true) }

// PolygonLine desenha o contorno de um polígono.
func (g *Graphics) PolygonLine(points []Point, c Color) { g.polygon(points, c, false) }

func (g *Graphics) polygon(points []Point, c Color, fill bool) {
	if !g.Available || len(points) < 3 {
		return
	}
	vx, vy := make([]int16, len(points)), make([]int16, len(points))
	for i, p := range points {
		vx[i], vy[i] = int16(p.X), int16(p.Y)
	}
	f := gfxAAPolygon
	if fill {
		f = gfxFilledPolygon
	}
	f(g.renderer, &vx[0], &vy[0], int32(len(points)), c.R, c.G, c.B, c.A)
}

// Point desenha um ponto na cor padrão (SetColor).
func (g *Graphics) Point(x, y float64) {
	if g.Available {
		c := g.color
		gfxPixel(g.renderer, int16(x), int16(y), c.R, c.G, c.B, c.A)
	}
}

// LoadImage carrega um PNG (ou outro formato da SDL2_image), relativo à pasta do jogo.
func (g *Graphics) LoadImage(path string) (*Image, error) {
	if !g.Available {
		return &Image{}, nil
	}
	surface := imgLoad(g.path(path))
	if surface == 0 {
		return &Image{}, fmt.Errorf("%s: %s", path, sdlGetError())
	}
	t := sdlCreateTextureFromSurf(g.renderer, surface)
	sdlFreeSurface(surface)
	return wrapTexture(t), nil
}

func wrapTexture(t uintptr) *Image {
	var w, h int32
	sdlQueryTexture(t, nil, nil, &w, &h)
	sdlSetTextureBlendMode(t, sdlBlendBlend)
	return &Image{texture: t, Width: int(w), Height: int(h)}
}

// DrawOption ajusta Draw.
type DrawOption func(*drawConfig)

type drawConfig struct {
	src             *sdlRect
	scale, rotation float64
	flip            int32
	alpha           uint8
	tint            Color
}

// Src recorta um quadro de uma folha de sprites.
func Src(x, y, w, h int) DrawOption {
	return func(c *drawConfig) { c.src = &sdlRect{int32(x), int32(y), int32(w), int32(h)} }
}

// Scale aumenta ou diminui a imagem.
func Scale(s float64) DrawOption { return func(c *drawConfig) { c.scale = s } }

// Rotate gira a imagem em graus, em torno do centro.
func Rotate(degrees float64) DrawOption { return func(c *drawConfig) { c.rotation = degrees } }

// FlipX espelha a imagem na horizontal.
func FlipX() DrawOption { return func(c *drawConfig) { c.flip |= 1 } }

// FlipY espelha a imagem na vertical.
func FlipY() DrawOption { return func(c *drawConfig) { c.flip |= 2 } }

// Alpha muda a opacidade (0 a 255).
func Alpha(a uint8) DrawOption { return func(c *drawConfig) { c.alpha = a } }

// Tint multiplica as cores da imagem.
func Tint(t Color) DrawOption { return func(c *drawConfig) { c.tint = t } }

// Draw desenha a imagem com o canto superior esquerdo em (x, y).
func (g *Graphics) Draw(img *Image, x, y float64, opts ...DrawOption) {
	if !g.Available || img == nil || img.texture == 0 {
		return
	}
	c := drawConfig{scale: 1, alpha: 255, tint: RGB(255, 255, 255)}
	for _, o := range opts {
		o(&c)
	}
	src := sdlRect{0, 0, int32(img.Width), int32(img.Height)}
	if c.src != nil {
		src = *c.src
	}
	dst := sdlRect{int32(x), int32(y), int32(float64(src.W) * c.scale), int32(float64(src.H) * c.scale)}
	sdlSetTextureColorMod(img.texture, c.tint.R, c.tint.G, c.tint.B)
	sdlSetTextureAlphaMod(img.texture, c.alpha)
	sdlRenderCopyEx(g.renderer, img.texture, &src, &dst, c.rotation, 0, c.flip)
}

// LoadFont carrega uma fonte TTF; path "" usa a fonte padrão do sistema.
func (g *Graphics) LoadFont(path string, size int) (*Font, error) {
	if !g.Available {
		return &Font{Size: size}, nil
	}
	if path == "" {
		path = defaultFonts[0]
		for _, f := range defaultFonts {
			if _, err := os.Stat(f); err == nil {
				path = f
				break
			}
		}
	}
	h := ttfOpenFont(g.path(path), int32(size))
	if h == 0 {
		return &Font{Size: size}, fmt.Errorf("%s: %s", path, sdlGetError())
	}
	return &Font{handle: h, Size: size}, nil
}

func (g *Graphics) defaultFont(f *Font) *Font {
	if f != nil {
		return f
	}
	if g.font == nil {
		g.font, _ = g.LoadFont("", 24)
	}
	return g.font
}

// Align é o alinhamento horizontal de Text.
type Align int

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

// Measure devolve largura e altura do texto em pixels (font nil = padrão).
func (g *Graphics) Measure(text string, font *Font) (int, int) {
	font = g.defaultFont(font)
	if !g.Available || font.handle == 0 || text == "" {
		return 0, 0
	}
	var w, h int32
	ttfSizeUTF8(font.handle, text, &w, &h)
	return int(w), int(h)
}

// Text escreve o texto com o topo em y (font nil = padrão, 24 px).
func (g *Graphics) Text(text string, x, y float64, font *Font, c Color, align Align) {
	font = g.defaultFont(font)
	if !g.Available || font.handle == 0 || text == "" {
		return
	}
	key := textKey{font.handle, text, c}
	entry, ok := g.textCache[key]
	if !ok {
		surface := ttfRenderUTF8Blended(font.handle, text, c.packed())
		if surface == 0 {
			return
		}
		t := sdlCreateTextureFromSurf(g.renderer, surface)
		sdlFreeSurface(surface)
		entry = &cachedText{image: wrapTexture(t)}
		g.textCache[key] = entry
	}
	entry.unused = 0
	switch align {
	case AlignCenter:
		x -= float64(entry.image.Width) / 2
	case AlignRight:
		x -= float64(entry.image.Width)
	}
	g.Draw(entry.image, x, y)
}

// Screenshot salva o quadro atual em PNG (chame no fim de OnDraw). Útil para testar.
func (g *Graphics) Screenshot(path string) error {
	if !g.Available {
		return fmt.Errorf("sem gráficos")
	}
	var w, h int32
	sdlGetRendererOutputSize(g.renderer, &w, &h)
	surface := sdlCreateRGBSurfaceWithFmt(0, w, h, 32, pixelFormatARGB8888)
	if surface == nil {
		return fmt.Errorf("%s", sdlGetError())
	}
	defer sdlFreeSurfaceP(surface)
	sdlRenderReadPixels(g.renderer, nil, pixelFormatARGB8888, surface.Pixels, surface.Pitch)
	if imgSavePNG(surface, g.path(path)) != 0 {
		return fmt.Errorf("%s: %s", path, sdlGetError())
	}
	return nil
}

func (g *Graphics) close() {
	if !g.Available {
		return
	}
	for _, c := range g.textCache {
		sdlDestroyTexture(c.image.texture)
	}
	sdlDestroyRenderer(g.renderer)
	sdlDestroyWindow(g.window)
	sdlQuitSubSystem(sdlInitVideo)
	g.Available = false
}
