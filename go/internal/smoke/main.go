// Teste rápido da biblioteca no console, sem precisar de botões.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gogabs/r36s-a11y-sdk/go/a11y"
)

func main() {
	g := a11y.New(a11y.Options{})
	bip, err := g.Sound.Load("bip", os.Args[1])
	fmt.Println("hrtf:", g.Sound.HRTF, "erro:", err, "save:", g.SaveDir)
	if len(os.Args) > 2 {
		if _, err := g.Sound.Load("ogg", os.Args[2]); err != nil {
			fmt.Println("ogg:", err)
		} else {
			fmt.Println("ogg ok")
		}
	}
	start := time.Now()
	g.OnStart(func() {
		g.SayThen("Teste da biblioteca Go. Esquerda, frente, direita, atrás.", func() {
			fmt.Println("fala terminou em", time.Since(start).Round(time.Millisecond))
			pos := [][3]float64{{-2, 0, 0}, {0, 0, -2}, {2, 0, 0}, {0, 0, 2}}
			for i, p := range pos {
				p := p
				g.After(time.Duration(i)*900*time.Millisecond, func() { bip.Play(a11y.Pos(p[0], p[1], p[2])) })
			}
			g.After(4*time.Second, func() {
				g.Sound.SetReverb("igreja")
				bip.Play(a11y.Pos(2, 0, 0))
			})
			g.After(6500*time.Millisecond, g.Quit)
		})
	})
	g.Run()
	fmt.Println("ok")
}
