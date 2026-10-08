# R36S A11y SDK

Ferramentas para criar jogos acessíveis para pessoas cegas e com baixa visão
no console portátil R36S (ArkOS), jogáveis só com som e fala.

*Tools for building audio games, accessible to blind and low-vision players,
for the R36S handheld (ArkOS). Docs are in Brazilian Portuguese.*

## O que tem aqui

| Pasta | Conteúdo |
|---|---|
| [`docs/contrato.md`](docs/contrato.md) | O contrato entre jogos e a plataforma: pacote, ambiente e regras de convivência |
| [`docs/audio-por-linguagem.md`](docs/audio-por-linguagem.md) | Como fazer áudio em Python, LÖVE, Go, Rust e C# no console |
| [`python/`](python/) | Biblioteca-ponte Python (`r36s_a11y`) e jogo de exemplo |
| [`go/`](go/) | Biblioteca-ponte Go (`a11y`), sem cgo, e jogo de exemplo |
| [`love/`](love/) | Biblioteca-ponte LÖVE (`a11y.lua`), um arquivo só, e jogo de exemplo |
| [`csharp/`](csharp/) | Biblioteca-ponte C# (`R36S.A11y`) com runtime .NET compartilhado, e jogo de exemplo |
| [`experimentos/audio/`](experimentos/audio/) | Testes de áudio posicional nas cinco linguagens |

## Como funciona

A galeria acessível do console só abre os jogos, como o EmulationStation.
Cada jogo lê o controle, fala pelo Speech Dispatcher e toca som 3D pela
OpenAL sozinho, sempre por meio da biblioteca-ponte da sua linguagem. A
biblioteca garante as regras do sistema: Fn e os botões apertados com ele
são do sistema, a voz segue os ajustes do usuário e o áudio passa pelo
mixer compartilhado, para o leitor de tela continuar falando.

## Começando em Python

```python
import r36s_a11y as a11y

game = a11y.Game()
pulo = game.sound.load("pulo", "sounds/pulo.ogg")

@game.on_button
def botao(b, estado, ms):
    if estado == a11y.DOWN and a11y.is_confirm(b):
        pulo.play(pan=-0.5)
        game.say("Pulou!")

game.run()
```

Veja [`python/README.md`](python/README.md).

## Estado

| Linguagem | Biblioteca |
|---|---|
| Python | 0.1, funcionando no console |
| Go | 0.1, funcionando no console |
| C# | 0.1, funcionando no console (runtime .NET compartilhado) |
| LÖVE | 0.1, funcionando no console (LÖVE 11.4 do ArkOS) |

## Licença

MIT. Veja [LICENSE](LICENSE).
