// Onde está o bip? Jogo de exemplo da biblioteca Go.
//
// Um bip toca à esquerda, à frente, à direita ou atrás de você. Aperte o
// direcional para o lado de onde ele veio. Com fone, a OpenAL usa HRTF e dá
// para distinguir frente e trás.
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/gogabs/r36s-a11y-sdk/go/a11y"
)

const rounds = 10

type direction struct {
	name    string
	x, y, z float64
}

var directions = map[a11y.Button]direction{
	a11y.Left:  {"à esquerda", -2, 0, 0},
	a11y.Up:    {"à frente", 0, 0, -2},
	a11y.Right: {"à direita", 2, 0, 0},
	a11y.Down:  {"atrás", 0, 0, 2},
}

var buttons = []a11y.Button{a11y.Left, a11y.Up, a11y.Right, a11y.Down}

type game struct {
	*a11y.Game
	bip, acerto, erro *a11y.Sound
	mainMenu, pause   *a11y.Menu

	number, score int
	answer        a11y.Button // "" fora de uma partida
	waiting       bool
}

func (g *game) recordFile() string { return filepath.Join(g.SaveDir, "recorde.json") }

func (g *game) loadRecord() int {
	var r struct{ Recorde int }
	if data, err := os.ReadFile(g.recordFile()); err == nil {
		json.Unmarshal(data, &r)
	}
	return r.Recorde
}

func (g *game) saveRecord(v int) {
	data, _ := json.Marshal(map[string]int{"recorde": v})
	os.WriteFile(g.recordFile(), data, 0o644)
}

func (g *game) startRound() {
	g.number, g.score = 0, 0
	g.SayThen("Valendo! Ouça o bip e aperte o direcional para o lado dele.", g.next)
}

func (g *game) next() {
	if g.number >= rounds {
		g.finish()
		return
	}
	g.number++
	g.answer = buttons[rand.Intn(len(buttons))]
	g.waiting = false
	g.After(600*time.Millisecond, g.playBip)
}

func (g *game) playBip() {
	d := directions[g.answer]
	g.bip.Play(a11y.Pos(d.x, d.y, d.z))
	g.waiting = true
}

func (g *game) respond(b a11y.Button) {
	if !g.waiting {
		return
	}
	if b == a11y.Select {
		g.playBip()
		return
	}
	if _, ok := directions[b]; !ok {
		return
	}
	g.waiting = false
	if b == g.answer {
		g.score++
		g.acerto.Play()
		g.After(500*time.Millisecond, g.next)
	} else {
		g.erro.Play()
		g.Speech.Say(fmt.Sprintf("Era %s.", directions[g.answer].name), true,
			func(bool) { g.next() })
	}
}

func (g *game) finish() {
	record := g.loadRecord()
	text := fmt.Sprintf("Fim. Você acertou %d de %d.", g.score, rounds)
	if g.score > record {
		g.saveRecord(g.score)
		text += " Novo recorde!"
	} else if record > 0 {
		text += fmt.Sprintf(" Recorde: %d.", record)
	}
	g.answer = ""
	g.SayThen(text, func() { g.mainMenu.Show(0, false) })
}

func main() {
	g := &game{Game: a11y.New(a11y.Options{})}
	load := func(name string) *a11y.Sound {
		s, err := g.Sound.Load(name, "sounds/"+name+".wav")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return s
	}
	g.bip, g.acerto, g.erro = load("bip"), load("acerto"), load("erro")
	move := load("move")

	g.mainMenu = a11y.NewMenu(g.Game, "Onde está o bip",
		[]string{"Jogar", "Instruções", "Sair"}, func(_ int, item string) {
			switch item {
			case "Jogar":
				g.mainMenu.Close()
				g.startRound()
			case "Instruções":
				g.Say("Um bip toca à esquerda, à frente, à direita ou atrás de você. " +
					"Aperte o direcional para o lado de onde ele veio. " +
					"Select repete o bip. Start pausa. Use fone de ouvido.")
			default:
				g.Quit()
			}
		})
	g.mainMenu.OnBack = g.Quit
	g.mainMenu.MoveSound = move

	g.pause = a11y.NewMenu(g.Game, "Pausa", []string{"Continuar", "Sair para o menu"},
		func(_ int, item string) {
			g.pause.Close()
			if item == "Continuar" {
				g.SayThen("Continuando.", g.playBip)
				return
			}
			g.answer = ""
			g.mainMenu.Show(0, false)
		})
	g.pause.OnBack = func() { g.pause.OnSelect(0, "Continuar") }
	g.pause.MoveSound = move

	g.OnButton(func(e a11y.ButtonEvent) {
		if e.State != a11y.Pressed || g.answer == "" {
			return
		}
		if e.Button == a11y.Start {
			g.waiting = false
			g.pause.Show(0, true)
			return
		}
		g.respond(e.Button)
	})
	g.OnStart(func() { g.mainMenu.Show(0, true) })
	g.Run()
}
