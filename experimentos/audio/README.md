# Testes de áudio por linguagem

Cada programa toca `bip.wav` à esquerda, à frente e à direita usando as
bibliotecas de áudio que já vêm no console (SDL2_mixer; o LÖVE usa OpenAL).

Compilar no PC (os binários vão para `bin/`):

```bash
# Go, sem cgo
cd go && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ../bin/goaudio .
# Rust (precisa de: rustup target add aarch64-unknown-linux-gnu, e do zig no PATH)
cd rust && cargo build --release --target aarch64-unknown-linux-gnu && cp target/aarch64-unknown-linux-gnu/release/rsaudio ../bin/
# C#
cd csharp && dotnet publish -c Release -r linux-arm64 --self-contained true -p:PublishTrimmed=true -o ../bin/csaudio
```

O `rust/zcc.cmd` é para Windows; no Linux/macOS troque o linker por um
script com `zig cc -target aarch64-linux-gnu.2.30 "$@"`.

Rodar no console:

```bash
scp -r experimentos/audio r36s:/tmp/
ssh r36s sh /tmp/audio/rodar-no-console.sh
```
