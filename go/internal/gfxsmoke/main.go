// Teste dos gráficos no console: desenha formas, sprite e texto e salva um PNG.
package main

import (
	"fmt"
	"os"

	"github.com/gogabs/r36s-a11y-sdk/go/a11y"
)

func main() {
	g := a11y.New(a11y.Options{})
	img, err := g.Gfx().LoadImage(os.Args[1])
	titulo, _ := g.Gfx().LoadFont("", 32)
	fmt.Println("gfx", g.Gfx().Available, "img", img.Width, img.Height, err)
	frames := 0
	g.OnDraw(func(gfx *a11y.Graphics) {
		gfx.Clear(a11y.Hex("#101830"))
		gfx.Text("Gráficos em Go ÁÇÉ", 320, 20, titulo, a11y.Hex("#ffd23c"), a11y.AlignCenter)
		gfx.Polygon([]a11y.Point{a11y.Pt(320, 120), a11y.Pt(290, 160), a11y.Pt(350, 160)}, a11y.RGB(60, 200, 90))
		gfx.Circle(320, 260, 60, a11y.RGB(40, 50, 80))
		gfx.CircleLine(320, 260, 60, a11y.RGB(120, 130, 170))
		gfx.Rect(20, 420, 200, 40, a11y.RGBA(200, 60, 60, 128))
		gfx.RectLine(20, 420, 200, 40, a11y.RGB(255, 255, 255))
		gfx.Line(250, 440, 620, 440, 4, a11y.RGB(255, 255, 255))
		gfx.Draw(img, 288, 228, a11y.Scale(2))
		gfx.Draw(img, 560, 300, a11y.Rotate(45), a11y.FlipX(), a11y.Alpha(160), a11y.Tint(a11y.RGB(255, 120, 120)))
		frames++
		if frames == 30 {
			fmt.Println("screenshot", gfx.Screenshot(os.Args[2]))
			g.Quit()
		}
	})
	g.Run()
	fmt.Println("ok")
}
