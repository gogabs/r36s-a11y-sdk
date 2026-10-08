# Plataforma de jogos acessíveis do R36S

Contrato entre jogos e a plataforma, e as bibliotecas-ponte por linguagem.

Status: versão 3 (2026-10-08). Decidido: galeria só lançadora, OpenAL como
base de áudio, primeira biblioteca em Python. Detalhes de áudio em
[audio-por-linguagem.md](audio-por-linguagem.md).

---

## 1. Modelo

Igual ao EmulationStation: a galeria só abre o jogo e espera ele terminar.
O jogo é um programa independente que lê o controle, fala e toca som
sozinho, sempre por meio da biblioteca-ponte da sua linguagem.

```
galeria        lista os jogos, fala o menu, abre o escolhido, espera ele sair
   │ inicia
jogo           lê /dev/input, fala no Speech Dispatcher, toca som pela OpenAL
   │ usa
biblioteca-ponte (Python, LÖVE, Go, Rust, C#)  ← quem garante as regras abaixo

a11yd          em paralelo, sempre: Fn, volume, brilho, energia, voz, saída de emergência
```

Consequências:

- Não existe protocolo entre jogo e galeria. O "contrato" é: o formato do
  pacote, o ambiente em que o jogo roda e as **regras de convivência** com o
  sistema (seção 4).
- Um jogo roda sozinho, pelo SSH ou pelo menu Ports do EmulationStation.
- Nada fica entre o botão e o jogo, então a latência é a menor possível.
- Quem garante as regras é a biblioteca-ponte. Um jogo que use a biblioteca
  oficial cumpre o contrato sem pensar nele.

---

## 2. Pacote de jogo

Cada jogo é uma pasta em `/opt/a11y/games/<id>/`:

```
corrida-de-sapos/
  game.json        obrigatório
  bin/jogo         executável, script com shebang, ou jogo.love
  sounds/*.ogg
```

### 2.1 game.json

```json
{
  "id": "corrida-de-sapos",
  "name": "Corrida de Sapos",
  "description": "Salte na hora certa para chegar primeiro.",
  "version": "1.0.0",
  "contract": 1,
  "runtime": "native",
  "exec": "bin/jogo",
  "languages": ["pt-BR"],
  "author": "Gabs"
}
```

| Campo | Obrigatório | Descrição |
|---|---|---|
| `id` | sim | Minúsculas, números e hífen; igual ao nome da pasta |
| `name` | sim | Falado na galeria |
| `description` | não | Falada quando se pede detalhes do jogo |
| `version` | sim | SemVer |
| `contract` | sim | Versão deste contrato (`1`) |
| `runtime` | sim | `native` (binário arm64), `python3`, `love` ou `dotnet` |
| `exec` | sim | Relativo à pasta. Para `love`, o `.love` ou a pasta; para `python3`, o `.py`; para `dotnet`, a `.dll` |
| `languages` | não | Idiomas que o jogo fala; padrão `["pt-BR"]` |
| `author` | não | Autor |

O `runtime` deixa a galeria montar o comando certo (`python3 x.py`,
`/opt/love2d/love x.love`, `dotnet x.dll`) sem o jogo precisar de script de
partida.

### 2.2 Ambiente

A galeria inicia o jogo com a pasta do jogo como diretório atual e:

| Variável | Exemplo | Uso |
|---|---|---|
| `A11Y_CONTRACT` | `1` | Versão do contrato da plataforma |
| `A11Y_GAME_DIR` | `/opt/a11y/games/corrida-de-sapos` | Pasta do jogo (somente leitura) |
| `A11Y_SAVE_DIR` | `/opt/a11y/saves/corrida-de-sapos` | Já criada; único lugar gravável garantido |
| `A11Y_LANG` | `pt-BR` | Idioma do sistema, escolhido no menu de configurações da galeria (`/opt/a11y/etc/sistema.conf`) |

Rodando fora da galeria (SSH, Ports, PC), a biblioteca-ponte preenche
valores padrão.

### 2.3 Saída

- stdout e stderr vão para `/opt/a11y/log/<id>.log`.
- Código `0` é saída normal. Outro código faz a galeria falar "O jogo
  fechou com erro" antes de voltar ao menu.

---

## 3. Responsabilidades

| Quem | Faz | Não faz |
|---|---|---|
| Galeria | Lista, fala o menu, abre o jogo, para de ler botões e de falar enquanto ele roda, volta quando ele sai | Não repassa botões nem falas |
| a11yd | Atalhos com Fn, volume, brilho, bateria, Power, ajustes de voz, saída de emergência | Não interfere no jogo fora disso |
| Biblioteca-ponte | Entrada, fala, áudio, pastas, convenções de botão, menu falado | — |
| Jogo | A lógica e o conteúdo | Não acessa hardware fora da biblioteca |

Mudanças necessárias na plataforma:

- **Galeria**: virar só lançadora (hoje ela repassa botões e falas), ler
  `game.json` e gravar o PID do jogo em `/run/a11y/jogo.pid`.
- **a11yd**: Fn + Start segurado por 3 s encerra o jogo do PID gravado
  (SIGTERM, depois SIGKILL). Ajustes de voz passam a valer para todas as
  conexões do Speech Dispatcher, inclusive a do jogo; hoje valem só para a
  do próprio a11yd até a próxima conexão.

---

## 4. Regras de convivência

São o contrato de verdade. A biblioteca-ponte implementa todas.

### 4.1 Entrada

1. Ler `/dev/input/event*` (ou SDL, no LÖVE) **sem exclusividade**: nada de
   `EVIOCGRAB`, porque o a11yd precisa ver os mesmos botões.
2. Descartar o botão Fn e **qualquer botão apertado enquanto Fn está
   pressionado**: são do sistema.
3. Descartar rocker de volume e Power.
4. Entregar ao jogo os botões com nomes padrão: `up` `down` `left` `right`
   `a` `b` `x` `y` `l1` `r1` `l2` `r2` `l3` `r3` `start` `select`, com os
   estados `down`, `up` e `repeat`, e o horário do evento em milissegundos.
5. Analógicos como eixos `lx` `ly` `rx` `ry` de -1 a 1, com zona morta.

Convenção de botões que todos os jogos seguem:

| Botão | Papel |
|---|---|
| `b` (de baixo) | Confirmar |
| `a` (da direita) | Voltar |
| `start` | Pausa |
| `select` | Repetir a última informação / ajuda |

### 4.2 Fala

1. Conectar ao Speech Dispatcher (socket em
   `$XDG_RUNTIME_DIR/speech-dispatcher/speechd.sock`) com nome de cliente
   `ark:a11y:<id>` e idioma `A11Y_LANG`.
2. Aplicar a velocidade, o tom e o volume que o usuário escolheu
   (`/opt/a11y/etc/voz.conf`, ajustados com Fn + direcionais), nunca valores
   próprios do jogo.
3. Fala interrompe a anterior por padrão; fila é opcional.

### 4.3 Áudio

1. Usar sempre o dispositivo `default` do ALSA (que é `dmix`), nunca
   `hw:0,0`. Assim o leitor de tela e o jogo tocam juntos.
2. Carregar as bibliotecas de áudio do sistema em tempo de execução
   (OpenAL Soft como base, SDL2_mixer como alternativa). Jogos não trazem
   `.so` próprias.
3. Formatos dos arquivos: OGG Vorbis e WAV.

### 4.4 Ciclo de vida

1. `SIGTERM`: salvar e sair em até 2 s.
2. Suspender e voltar (Power) é transparente: o sistema congela o processo.
3. Saída de emergência: Fn + Start segurado por 3 s, tratada pelo a11yd.
4. Gravar só em `A11Y_SAVE_DIR`.

### 4.5 Tela

Opcional. Quem quiser desenha pela SDL2 (KMSDRM) ou pelo LÖVE. Todo jogo
precisa ser jogável sem olhar a tela.

---

## 5. Bibliotecas-ponte

Uma por linguagem, com a mesma forma, para que o mesmo tutorial sirva para
todas.

| Linguagem | Entrada | Fala | Áudio |
|---|---|---|---|
| Python 3.7 | `python3-evdev` (já no console) | SSIP no socket (`a11y.speech`, já existe) | `ctypes` → OpenAL + libvorbisfile |
| LÖVE 11.4 | `love.joystick` (SDL) | FFI → `libspeechd.so.2` | `love.audio` (OpenAL, embutido) |
| Go | `/dev/input` (biblioteca padrão) | SSIP no socket | `purego` → OpenAL, sem cgo |
| Rust | `/dev/input` (biblioteca padrão) | SSIP no socket | `libloading` → OpenAL |
| C# | `/dev/input` (`FileStream`) | SSIP no socket | `DllImport` → OpenAL, NVorbis |

API comum (os nomes mudam conforme a convenção de cada linguagem):

- `Game`: laço com `on_button(botão, estado, ms)`, `on_axis`, `update(dt)`
  em taxa fixa, `on_quit`.
- `say(texto, fila=False)`, `hush()`, `say_and_wait(texto)`.
- `Sound`: `load(nome, arquivo)`, `play(nome, loop, volume, posição)`
  devolvendo uma fonte com `set_position`, `set_pitch`, `stop(fade)`;
  `listener(posição, direção)`.
- `save_dir`, `lang`, `is_confirm(b)`, `is_back(b)`.
- `Menu(["Jogar", "Instruções", "Sair"])` falado e pronto, porque é onde
  mais se erra acessibilidade.

No PC (Windows e Linux), cada biblioteca troca os bastidores: teclado no
lugar do controle, voz do sistema ou NVDA no lugar do Speech Dispatcher, e a
mesma OpenAL. O jogo não muda nada para rodar nos dois.

Exemplo em Python:

```python
import a11y

game = a11y.Game()
pulo = game.sound.load("pulo", "sounds/pulo.ogg")

@game.on_button
def botao(b, estado, ms):
    if estado == "down" and a11y.is_confirm(b):
        pulo.play()
        game.say("Pulou!")

game.run()
```

---

## 6. Ferramentas

- **Validador**: confere `game.json`, a pasta e a arquitetura do binário.
- **Instalador**: copia o jogo para o console pelo SSH.
- **Testes de conformidade** de cada biblioteca: botão com Fn não chega ao
  jogo, fala sem mudar a voz, áudio no `default`.

---

## 7. Regras de acessibilidade para jogos

1. Toda tela e menu é falado ao entrar, com como navegar.
2. Todo item é falado ao ser selecionado.
3. Start sempre pausa, e a pausa sempre tem "Sair".
4. Select repete a última informação importante.
5. Nenhuma informação só visual. Nenhuma ação com tempo curto sem aviso
   sonoro antes.
6. Português sempre; outros idiomas são bem-vindos.
7. Voltar no menu principal sai do jogo.
