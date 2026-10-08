# a11y (Go)

Biblioteca-ponte Go para jogos acessíveis no R36S. Sem cgo: compila do
Windows, Linux ou macOS direto para o console. No console usa:

- o controle direto em `/dev/input` (sem exclusividade, o sistema continua
  recebendo os botões);
- o Speech Dispatcher, com a voz que o usuário ajustou;
- a OpenAL Soft do sistema (carregada com [purego](https://github.com/ebitengine/purego)),
  com HRTF e reverberação por ambiente;
- a `libvorbisfile` do sistema para arquivos OGG.

Fora do Linux, o jogo compila e roda, mas sem controle e sem som; a fala vai
para o terminal.

Requer Go 1.25+.

## Começando

```bash
go get github.com/gogabs/r36s-a11y-sdk/go
```

```go
package main

import "github.com/gogabs/r36s-a11y-sdk/go/a11y"

func main() {
	g := a11y.New(a11y.Options{})
	pulo, _ := g.Sound.Load("pulo", "sounds/pulo.ogg")

	g.OnButton(func(e a11y.ButtonEvent) {
		if e.State == a11y.Pressed && a11y.IsConfirm(e.Button) {
			pulo.Play(a11y.Pan(-0.5))
			g.Say("Pulou!")
		}
	})
	g.Run()
}
```

Compilar para o console:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o meu-jogo .
```

`game.json` com `"runtime": "native"` e `"exec": "meu-jogo"` (veja o
[contrato](../docs/contrato.md)).

## API

### Game

| Membro | Descrição |
|---|---|
| `a11y.New(a11y.Options{FPS, Repeat, Axes, Deadzone})` | Cria o jogo; o valor zero serve |
| `g.OnButton(func(a11y.ButtonEvent))` | `e.Button`, `e.State` (`Pressed`, `Released`, `Repeated`), `e.Ms` |
| `g.OnAxis(func(a11y.AxisEvent))` | `LX LY RX RY` de -1 a 1 (com `Axes: true`) |
| `g.OnUpdate(func(dt float64))`, `g.OnStart`, `g.OnQuit` | Laço, início e saída (salve aqui) |
| `g.Say(texto)`, `g.SayQueued(texto)`, `g.SayThen(texto, depois)` | Fala; `SayThen` chama `depois` ao fim |
| `g.Speech.Say(texto, fila, func(terminou bool))` | Forma completa |
| `g.Hush()` | Cala a fala |
| `g.After(duração, fn)` | Chama `fn` depois de um tempo |
| `g.SaveDir`, `g.Lang`, `g.Dir`, `g.ID` | Pastas, idioma e identificador |
| `g.Quit()` | Sai do jogo |

Tudo roda numa goroutine só, a do `Run`: os callbacks nunca rodam em
paralelo.

Botões: `Up Down Left Right A B X Y L1 R1 L2 R2 L3 R3 Start Select`.
Convenção: `B` confirma, `A` volta, `Start` pausa, `Select` repete. Fn,
volume e Power nunca chegam ao jogo.

### Áudio

```go
passo, err := g.Sound.Load("passo", "sounds/passo.ogg")   // WAV ou OGG
voz := passo.Play(a11y.Pos(x, y, z), a11y.Volume(0.8), a11y.Pitch(1.1))
voz = passo.Play(a11y.Pan(-1), a11y.Loop())
voz.SetPosition(x, y, z); voz.SetVolume(0.5, 0.3); voz.SetPitch(1.2)
voz.Stop(0.5); voz.Playing()
g.Sound.Listener([3]float64{0, 0, 0}, 90)   // graus; 0 = frente, 90 = direita
g.Sound.SetReverb("igreja")                 // quarto, corredor, estacionamento, cozinha, metro, igreja, floresta; "" desliga
```

Coordenadas em metros: `x` positivo à direita, `y` para cima, `z` negativo
à frente. Só sons **mono** são posicionados. `a11y.Reverb(true/false)` força
o ambiente num som (padrão: só sons com `Pos`).

### Menu

```go
menu := a11y.NewMenu(g, "Menu principal", []string{"Jogar", "Sair"},
	func(i int, item string) { ... })
menu.OnBack = g.Quit
menu.MoveSound = clique
menu.Show(0, true)   // item inicial; true fala as instruções
```

## Exemplo

[`examples/onde-esta-o-bip`](examples/onde-esta-o-bip), o mesmo jogo do
exemplo em Python.

## Testes

```bash
go test ./...
```
