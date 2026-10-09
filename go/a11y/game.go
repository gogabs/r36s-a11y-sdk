package a11y

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"time"
)

const (
	repeatDelay    = 400 * time.Millisecond
	repeatInterval = 100 * time.Millisecond
)

// Options configura o jogo. O valor zero serve: 60 quadros por segundo, sem
// repetição de botão segurado, sem eixos.
type Options struct {
	FPS      int     // quadros por segundo de OnUpdate (padrão 60)
	Repeat   bool    // gera eventos Repeated enquanto o botão está apertado
	Axes     bool    // entrega os analógicos em OnAxis
	Deadzone float64 // zona morta dos analógicos (padrão 0.2)
}

// Focus recebe os botões antes do jogo (por exemplo, um Menu aberto).
type Focus interface {
	HandleButton(e ButtonEvent)
}

// Game é um jogo acessível do R36S.
type Game struct {
	ID, Name, Dir, SaveDir, Lang string
	Sound                        *Audio
	Speech                       *Speech

	opts     Options
	input    *inputState
	start    time.Time
	running  bool
	timers   []timer
	repeatAt map[Button]time.Time
	focus    []Focus

	onButton []func(ButtonEvent)
	onAxis   []func(AxisEvent)
	onUpdate []func(dt float64)
	onDraw   []func(gfx *Graphics)
	gfx      *Graphics
	onStart  []func()
	onQuit   []func()
}

type timer struct {
	at time.Time
	fn func()
}

// New cria o jogo, lendo game.json e as variáveis A11Y_* do ambiente.
func New(opts Options) *Game {
	if opts.FPS <= 0 {
		opts.FPS = 60
	}
	if opts.Deadzone == 0 {
		opts.Deadzone = 0.2
	}
	g := &Game{opts: opts, start: time.Now(), repeatAt: map[Button]time.Time{}}
	g.Dir = os.Getenv("A11Y_GAME_DIR")
	if g.Dir == "" {
		if exe, err := os.Executable(); err == nil {
			g.Dir = filepath.Dir(exe)
		} else {
			g.Dir, _ = os.Getwd()
		}
	}
	var manifest struct{ ID, Name string }
	if data, err := os.ReadFile(filepath.Join(g.Dir, "game.json")); err == nil {
		json.Unmarshal(data, &manifest)
	}
	g.ID = manifest.ID
	if g.ID == "" {
		g.ID = filepath.Base(g.Dir)
	}
	g.Name = manifest.Name
	if g.Name == "" {
		g.Name = g.ID
	}
	g.Lang = os.Getenv("A11Y_LANG")
	if g.Lang == "" {
		g.Lang = "pt-BR"
	}
	g.SaveDir = os.Getenv("A11Y_SAVE_DIR")
	if g.SaveDir == "" {
		home, _ := os.UserHomeDir()
		g.SaveDir = filepath.Join(home, ".local", "share", "r36s-a11y", g.ID)
	}
	os.MkdirAll(g.SaveDir, 0o755)
	g.Speech = newSpeech(g.ID, g.Lang)
	g.Sound = newAudio(g.Dir)
	g.input = newInputState(opts.Deadzone)
	return g
}

// OnButton registra fn para cada botão apertado, solto ou repetido.
func (g *Game) OnButton(fn func(ButtonEvent)) { g.onButton = append(g.onButton, fn) }

// OnAxis registra fn para os analógicos (só com Options.Axes).
func (g *Game) OnAxis(fn func(AxisEvent)) { g.onAxis = append(g.onAxis, fn) }

// OnUpdate registra fn, chamada FPS vezes por segundo com o tempo do quadro.
func (g *Game) OnUpdate(fn func(dt float64)) { g.onUpdate = append(g.onUpdate, fn) }

// OnDraw registra fn, que desenha um quadro logo depois dos OnUpdate. Abre a tela.
func (g *Game) OnDraw(fn func(gfx *Graphics)) { g.onDraw = append(g.onDraw, fn) }

// Gfx devolve os gráficos, abrindo a tela no primeiro uso (para carregar
// imagens e fontes antes do Run).
func (g *Game) Gfx() *Graphics {
	if g.gfx == nil {
		g.gfx = newGraphics(g.Name, g.Dir)
	}
	return g.gfx
}

// OnStart registra fn, chamada quando o laço começa.
func (g *Game) OnStart(fn func()) { g.onStart = append(g.onStart, fn) }

// OnQuit registra fn, chamada ao sair: hora de salvar.
func (g *Game) OnQuit(fn func()) { g.onQuit = append(g.onQuit, fn) }

// Say fala o texto, interrompendo a fala anterior.
func (g *Game) Say(text string) { g.Speech.Say(text, false, nil) }

// SayThen fala o texto e chama then quando a fala termina.
func (g *Game) SayThen(text string, then func()) {
	g.Speech.Say(text, false, func(bool) { then() })
}

// SayQueued fala o texto depois das falas anteriores.
func (g *Game) SayQueued(text string) { g.Speech.Say(text, true, nil) }

// Hush cala a fala.
func (g *Game) Hush() { g.Speech.Hush() }

// After chama fn daqui a d.
func (g *Game) After(d time.Duration, fn func()) {
	g.timers = append(g.timers, timer{time.Now().Add(d), fn})
}

// Ms devolve os milissegundos desde o início, na escala dos eventos de botão.
func (g *Game) Ms() int64 { return time.Since(g.start).Milliseconds() }

// AxisValue devolve o último valor de um analógico.
func (g *Game) AxisValue(a Axis) float64 { return g.input.axisValue[a] }

// Quit encerra o laço.
func (g *Game) Quit() { g.running = false }

// PushFocus faz f receber os botões antes do jogo.
func (g *Game) PushFocus(f Focus) { g.focus = append(g.focus, f) }

// PopFocus devolve os botões ao jogo.
func (g *Game) PopFocus(f Focus) {
	for i, x := range g.focus {
		if x == f {
			g.focus = append(g.focus[:i], g.focus[i+1:]...)
			return
		}
	}
}

func (g *Game) dispatchButton(e ButtonEvent) {
	if n := len(g.focus); n > 0 {
		g.focus[n-1].HandleButton(e)
		return
	}
	for _, fn := range g.onButton {
		fn(e)
	}
}

func (g *Game) handle(ev interface{}) {
	switch e := ev.(type) {
	case *ButtonEvent:
		if g.opts.Repeat {
			if e.State == Pressed {
				g.repeatAt[e.Button] = time.Now().Add(repeatDelay)
			} else {
				delete(g.repeatAt, e.Button)
			}
		}
		g.dispatchButton(*e)
	case *AxisEvent:
		if g.opts.Axes {
			for _, fn := range g.onAxis {
				fn(*e)
			}
		}
	}
}

func (g *Game) tick(now time.Time) {
	for b, at := range g.repeatAt {
		if !now.Before(at) {
			g.repeatAt[b] = at.Add(repeatInterval)
			g.dispatchButton(ButtonEvent{b, Repeated, g.Ms()})
		}
	}
	if len(g.timers) > 0 {
		var due, keep []timer
		for _, t := range g.timers {
			if now.Before(t.at) {
				keep = append(keep, t)
			} else {
				due = append(due, t)
			}
		}
		g.timers = keep
		sort.Slice(due, func(i, j int) bool { return due[i].at.Before(due[j].at) })
		for _, t := range due {
			t.fn()
		}
	}
}

// Run roda o laço do jogo até Quit, SIGTERM ou SIGINT.
func (g *Game) Run() {
	raw := make(chan rawEvent, 256)
	if openGamepads(g.input, g.start, raw) == 0 {
		fmt.Fprintln(os.Stderr, "[a11y] nenhum controle encontrado em /dev/input")
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	if len(g.onDraw) > 0 {
		g.Gfx() // abre a tela antes do primeiro quadro
	}
	g.running = true
	for _, fn := range g.onStart {
		fn()
	}
	frame := time.NewTicker(time.Second / time.Duration(g.opts.FPS))
	defer frame.Stop()
	last := time.Now()
	defer g.shutdown()
	for g.running {
		select {
		case ev := <-raw:
			if e := g.input.feed(ev.evType, ev.code, ev.value, ev.ms); e != nil {
				g.handle(e)
			}
		case fn := <-g.Speech.done:
			fn()
		case <-sig:
			g.running = false
		case now := <-frame.C:
			g.tick(now)
			dt := now.Sub(last).Seconds()
			last = now
			g.Sound.update(dt)
			for _, fn := range g.onUpdate {
				fn(dt)
			}
			if len(g.onDraw) > 0 && g.gfx.Available {
				g.gfx.begin()
				for _, fn := range g.onDraw {
					fn(g.gfx)
				}
				g.gfx.end()
			}
		}
	}
}

func (g *Game) shutdown() {
	for _, fn := range g.onQuit {
		func() {
			defer func() {
				if r := recover(); r != nil { // salvar não pode impedir a saída
					fmt.Fprintln(os.Stderr, "[a11y] erro ao sair:", r)
				}
			}()
			fn()
		}()
	}
	g.Speech.close()
	g.Sound.close()
	if g.gfx != nil {
		g.gfx.close()
	}
}
