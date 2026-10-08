# Áudio nos jogos, por linguagem

Investigação de 2026-10-08 num R36S com ArkOS 2.0 (Ubuntu 19.10, glibc
2.30). **Validado no console:** os cinco testes (Go, Rust, C#, Python, LÖVE)
tocaram um bip à esquerda, à frente e à direita, conferido de ouvido com
fone, com o leitor de tela falando junto. Fontes dos testes em
[`experimentos/audio/`](../experimentos/audio/).

## 1. O que o console já tem

| Peça | Versão / local | Para quê |
|---|---|---|
| ALSA (`libasound.so.2`) | sistema | Base de todo áudio |
| `~/.asoundrc` | `default` → `dmix` em `hw:0,0`, 44,1 kHz | Vários programas tocam ao mesmo tempo (jogo + voz) |
| SDL2 | `libSDL2-2.0.so.0` (várias versões, até 2.30) | Áudio, controle, tela |
| SDL2_mixer | `libSDL2_mixer-2.0.so.0` | Canais, OGG/WAV/MP3, panorâmica, posição 2D |
| OpenAL Soft 1.19.1 | `libopenal.so.1` | Áudio 3D, inclusive HRTF (som binaural no fone) |
| libvorbisfile, libogg, libsndfile, libmpg123 | sistema | Decodificar arquivos |
| PortAudio, PulseAudio (só a biblioteca) | sistema | Não recomendados aqui |
| libspeechd | `libspeechd.so.2` | Cliente do Speech Dispatcher em C |
| LÖVE 11.4 | `/opt/love2d/love` + LuaJIT | Motor de jogos Lua completo |
| glibc 2.30 | Ubuntu 19.10 | Limite para binários compilados fora |

Ou seja: não precisa instalar nada no console para nenhuma das cinco
linguagens. Isso importa porque a partição do sistema está quase cheia.

**Regra para toda biblioteca:** abrir o dispositivo `default` do ALSA, nunca
`hw:0,0` direto. Abrir o `hw` toma a placa só para o jogo e cala o leitor de
tela.

## 2. A ideia que serve para todas as linguagens

O jogo **não linka** com a biblioteca de áudio na compilação; ele **carrega
em tempo de execução** (`dlopen`) a `.so` que já está no console. Assim a
compilação cruzada continua sem sysroot nem toolchain do ARM, e o mesmo
código no PC carrega a `.dll` equivalente.

Testado no PC: um programa que inicia o SDL2_mixer e toca um bip à
esquerda, à frente e à direita.

| Linguagem | Como carrega | Compilação para o console | Tamanho do teste |
|---|---|---|---|
| Go | `purego` (sem cgo) | `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build` | 1,7 MB |
| Rust | `libloading` | alvo `aarch64-unknown-linux-gnu`, linker `zig cc` mirando glibc 2.30 | 0,35 MB |
| C# | `DllImport` | `dotnet publish -r linux-arm64` | 19 MB sozinho; 98 KB com runtime compartilhado |
| Python 3.7 | `ctypes` | não compila | 1 arquivo |
| LÖVE | já embutido (`love.audio`, OpenAL) | não compila | 1 arquivo |

Conferido: o binário Rust exige no máximo `GLIBC_2.30`, o que o console tem.

## 3. Duas bases possíveis para as bibliotecas-ponte

### SDL2_mixer (2D)

- Já decodifica OGG/WAV/MP3, mistura canais, faz loop, fade e volume.
- Posição com `Mix_SetPosition(canal, ângulo, distância)`: só esquerda e
  direita mais distância. Atrás soa igual à frente.
- Mesma API no Windows (`SDL2_mixer.dll`) para testar no PC.
- Menos código em cada biblioteca-ponte.

### OpenAL Soft (3D)

- Fontes com posição e velocidade em 3D, ouvinte com orientação, efeito
  Doppler, atenuação por distância.
- **HRTF**: com fone, dá para ouvir frente, trás, cima e baixo. Para jogo só
  de áudio isso é a diferença entre "o carro está à esquerda" e "o carro está
  atrás de você, à esquerda".
- É o que o LÖVE usa por dentro, então as bibliotecas e o LÖVE teriam o
  mesmo modelo (fontes 3D).
- Mais trabalho: OpenAL não lê arquivos. A biblioteca precisa decodificar
  OGG (pela `libvorbisfile` do console ou por decodificador da linguagem) e
  gerenciar buffers.
- *(inferido)* OpenAL Soft 1.19 já traz uma HRTF padrão embutida; precisa
  ser confirmado no console com fone.

## 4. Linguagem por linguagem

### Python 3.7

- Áudio: `ctypes` com SDL2_mixer ou OpenAL + libvorbisfile. Nada de pip.
- Fala: o módulo `a11y.speech` que já existe (SSIP direto no socket).
- Controle: `python3-evdev`, já instalado.
- Atenção: Python 3.7 é antigo (sem `match`, walrus só a partir do 3.8) e
  o interpretador é lento para mixagem própria. Mixar em Python não é
  opção; quem mixa é sempre a SDL ou a OpenAL.

### Go

- Áudio: `github.com/ebitengine/purego` para chamar SDL2_mixer/OpenAL sem
  cgo. Decodificação OGG pura em Go (`jfreymuth/oggvorbis`) é opcional se a
  base for OpenAL.
- Fala: SSIP direto no socket, biblioteca padrão.
- Controle: leitura de `/dev/input/event*` (struct de 24 bytes), biblioteca
  padrão.
- Compila do Windows sem nada além do Go.

### Rust

- Áudio: `libloading` + SDL2_mixer/OpenAL. Crates como `rodio`/`cpal`
  linkam a `libasound` na compilação e exigiriam sysroot do ARM; com
  `dlopen` isso some.
- Fala e controle: socket e `/dev/input` com a biblioteca padrão.
- Compilação: alvo `aarch64-unknown-linux-gnu` com `zig cc` como linker
  (`experimentos/audio/rust/zcc.cmd`), mirando glibc 2.30. O alvo `musl`
  estático não serve quando há `dlopen`.

### C#

- Áudio: `DllImport` para SDL2_mixer/OpenAL; decodificação OGG com NVorbis
  (gerenciado, sem nativo) se a base for OpenAL.
- Fala e controle: socket Unix e `FileStream` em `/dev/input`.
- Tamanho é o ponto fraco: 19 MB por jogo autocontido (com trimming) ou um
  runtime .NET de ~70 MB instalado uma vez em `/opt/a11y/runtime`.
- *(inferido)* Native AOT para Linux arm64 não compila a partir do Windows;
  precisaria de Linux/WSL para gerar binário nativo pequeno.

### LÖVE 11.4

- Áudio: `love.audio` (OpenAL), com fontes 3D. Nada a fazer.
- Fala: FFI do LuaJIT para `libspeechd.so.2` (exemplo em
  `experimentos/audio/love/main.lua`).
- Controle: `love.joystick`/gamepad (SDL). Fn chega como botão e a
  biblioteca-ponte precisa ignorar combinações com Fn.
- Distribuição: um `.love` (zip) por jogo, rodando com o LÖVE do sistema.
  O LÖVE é do ArkOS (`/opt/love2d`); se uma atualização do ArkOS o remover,
  levamos uma cópia para `/opt/a11y`.

## 5. Recomendação

1. **Base de áudio: OpenAL Soft**, pelo 3D com HRTF e porque iguala o modelo
   ao do LÖVE. SDL2_mixer fica como plano B se a OpenAL der problema de
   latência no console.
2. **Sempre `dlopen`** das bibliotecas do sistema; nenhum jogo traz `.so`.
3. **Ordem das bibliotecas-ponte:** Python (já temos fala e controle) e LÖVE
   (áudio pronto) primeiro; Go e Rust em seguida; C# por último, por causa
   do tamanho.

## 6. Próximo passo

Com o console no USB:

```
scp -r experimentos/audio r36s:/tmp/
ssh r36s sh /tmp/audio/rodar-no-console.sh
```

Cada teste toca o bip à esquerda, à frente e à direita, com o leitor de
tela funcionando ao mesmo tempo (validado). Falta medir a latência do botão
até o som: o `dmix` do `~/.asoundrc` usa `buffer_size 4096` a 44,1 kHz, o
que permite até ~93 ms de atraso. Reduzir afeta todo o áudio do sistema.
