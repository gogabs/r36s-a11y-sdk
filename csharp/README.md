# R36S.A11y (C#)

Biblioteca-ponte C# para jogos acessíveis no R36S. No console usa:

- o controle direto em `/dev/input` (sem exclusividade, o sistema continua
  recebendo os botões);
- o Speech Dispatcher, com a voz que o usuário escolheu;
- a OpenAL Soft do sistema (P/Invoke), com HRTF e reverberação por ambiente;
- a `libvorbisfile` do sistema para arquivos OGG.

Fora do Linux, o jogo compila e roda, mas sem controle e sem som; a fala vai
para o terminal.

## Runtime compartilhado

Um jogo .NET autocontido leva o runtime inteiro junto (uns 70 MB). Aqui o
runtime fica instalado **uma vez** no console, em `/opt/a11y/runtime/dotnet`
(.NET 10, suporte até novembro de 2028), e cada jogo publicado tem só o
próprio código: o exemplo inteiro, com sons, ocupa 270 KB.

Para quem desenvolve, a diferença são três linhas no `.csproj`:

```xml
<RuntimeIdentifier>linux-arm64</RuntimeIdentifier>  <!-- o console -->
<SelfContained>false</SelfContained>                <!-- não embute o runtime -->
<RollForward>LatestMajor</RollForward>              <!-- roda no runtime mais novo instalado -->
```

A biblioteca mira o .NET 8, então qualquer SDK do 8 em diante compila. Com
`RollForward=LatestMajor`, o jogo roda no .NET 10 do console.

A galeria abre jogos com `"runtime": "dotnet"` usando o runtime
compartilhado e define `DOTNET_ROOT`, então o executável gerado pelo
`dotnet publish` também funciona com `"runtime": "native"`.

### Instalar o runtime no console (uma vez)

No PC, com o console no USB:

```bash
curl -LO https://builds.dotnet.microsoft.com/dotnet/Runtime/10.0.12/dotnet-runtime-10.0.12-linux-arm64.tar.gz
scp dotnet-runtime-10.0.12-linux-arm64.tar.gz r36s:/tmp/
ssh r36s "mkdir -p /opt/a11y/runtime/dotnet && tar xzf /tmp/dotnet-runtime-10.0.12-linux-arm64.tar.gz -C /opt/a11y/runtime/dotnet && rm /tmp/dotnet-runtime-*.tar.gz"
```

Ocupa 86 MB em `/opt/a11y`.

## Começando

```xml
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net8.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <RuntimeIdentifier>linux-arm64</RuntimeIdentifier>
    <SelfContained>false</SelfContained>
    <RollForward>LatestMajor</RollForward>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="caminho/para/R36S.A11y/R36S.A11y.csproj" />
    <None Include="game.json;sounds/**" CopyToOutputDirectory="PreserveNewest" CopyToPublishDirectory="PreserveNewest" />
  </ItemGroup>
</Project>
```

```csharp
using R36S.A11y;

var game = new Game();
var pulo = game.Sound.Load("pulo", "sounds/pulo.ogg");

game.Button += e =>
{
    if (e.State == ButtonState.Pressed && Buttons.IsConfirm(e.Button))
    {
        pulo.Play(pan: -0.5);
        game.Say("Pulou!");
    }
};
game.Run();
```

```bash
dotnet publish -c Release -o publish
scp -r publish r36s:/opt/a11y/games/meu-jogo
```

`game.json` com `"runtime": "dotnet"` e `"exec": "MeuJogo.dll"` (veja o
[contrato](../docs/contrato.md)). Se preferir `"runtime": "native"` com o
executável `MeuJogo`, lembre do `chmod +x` depois de copiar do Windows.

## API

### Game

| Membro | Descrição |
|---|---|
| `new Game(new GameOptions { Fps, Repeat, Axes, Deadzone })` | Cria o jogo; os padrões servem |
| `game.Button += e => ...` | `e.Button`, `e.State` (`Pressed`, `Released`, `Repeated`), `e.Ms` |
| `game.Axis += e => ...` | `LX LY RX RY` de -1 a 1 (com `Axes = true`) |
| `game.Update += dt => ...`, `game.Start`, `game.Quit` | Laço, início e saída (salve aqui) |
| `game.Say(texto, queue: false, onDone: terminou => ...)` | Fala; interrompe a anterior a menos que `queue: true` |
| `game.SayThen(texto, depois)` | Fala e chama `depois` ao fim |
| `game.Hush()` | Cala a fala |
| `game.After(segundos, acao)` | Chama `acao` depois de um tempo |
| `game.Post(acao)` | Roda `acao` na thread do jogo (seguro de qualquer thread) |
| `game.SaveDir`, `game.Lang`, `game.Dir`, `game.Id` | Pastas, idioma e identificador |
| `game.Exit()` | Sai do jogo |

Tudo roda na thread do `Run`: os eventos nunca rodam em paralelo.

Botões: `Up Down Left Right A B X Y L1 R1 L2 R2 L3 R3 Start Select`.
Convenção: `B` confirma, `A` volta, `Start` pausa, `Select` repete. Fn,
volume e Power nunca chegam ao jogo.

### Áudio

```csharp
var passo = game.Sound.Load("passo", "sounds/passo.ogg");   // WAV ou OGG
var voz = passo.Play(pos: (x, y, z), volume: 0.8, pitch: 1.1);
voz = passo.Play(pan: -1, loop: true);
voz.SetPosition(x, y, z); voz.SetVolume(0.5, fadeSeconds: 0.3); voz.SetPitch(1.2);
voz.Stop(fadeSeconds: 0.5); _ = voz.Playing;
game.Sound.Listener((0, 0, 0), facing: 90);   // graus; 0 = frente, 90 = direita
game.Sound.SetReverb("igreja");               // quarto, corredor, estacionamento, cozinha, metro, igreja, floresta; null desliga
```

Coordenadas em metros: `x` positivo à direita, `y` para cima, `z` negativo
à frente. Só sons **mono** são posicionados. `reverb: true/false` força o
ambiente num som (padrão: só sons com `pos`).

### Menu

```csharp
var menu = new Menu(game, "Menu principal", new[] { "Jogar", "Sair" }, (i, item) => { ... })
{
    OnBack = game.Exit,
    MoveSound = clique,
};
menu.Show();            // item 0, falando as instruções
menu.Show(2, help: false);
```

## Exemplo

[`examples/OndeEstaOBip`](examples/OndeEstaOBip), o mesmo jogo dos exemplos
em Python e Go.

## Testes

```bash
dotnet test R36S.A11y.Tests
```
