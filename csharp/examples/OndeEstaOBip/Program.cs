// Onde está o bip? Jogo de exemplo da biblioteca C#.
//
// Um bip toca à esquerda, à frente, à direita ou atrás de você. Aperte o
// direcional para o lado de onde ele veio. Com fone, a OpenAL usa HRTF e dá
// para distinguir frente e trás.
using System.Text.Json;
using R36S.A11y;

const int Rounds = 10;
var directions = new Dictionary<Button, (string Name, (double, double, double) Pos)>
{
    [Button.Left] = ("à esquerda", (-2, 0, 0)),
    [Button.Up] = ("à frente", (0, 0, -2)),
    [Button.Right] = ("à direita", (2, 0, 0)),
    [Button.Down] = ("atrás", (0, 0, 2)),
};
var buttons = directions.Keys.ToArray();

var game = new Game();
var bip = game.Sound.Load("bip", "sounds/bip.wav");
var acerto = game.Sound.Load("acerto", "sounds/acerto.wav");
var erro = game.Sound.Load("erro", "sounds/erro.wav");
var move = game.Sound.Load("move", "sounds/move.wav");
var recordFile = Path.Combine(game.SaveDir, "recorde.json");

int number = 0, score = 0;
Button? answer = null; // null fora de uma partida
bool waiting = false;

int LoadRecord()
{
    try { return JsonDocument.Parse(File.ReadAllText(recordFile)).RootElement.GetProperty("recorde").GetInt32(); }
    catch (Exception) { return 0; }
}

void PlayBip()
{
    bip.Play(pos: directions[answer!.Value].Pos);
    waiting = true;
}

void Next()
{
    if (number >= Rounds)
    {
        Finish();
        return;
    }
    number++;
    answer = buttons[Random.Shared.Next(buttons.Length)];
    waiting = false;
    game.After(0.6, PlayBip);
}

void Respond(Button b)
{
    if (!waiting) return;
    if (b == Button.Select)
    {
        PlayBip();
        return;
    }
    if (!directions.ContainsKey(b)) return;
    waiting = false;
    if (b == answer)
    {
        score++;
        acerto.Play();
        game.After(0.5, Next);
    }
    else
    {
        erro.Play();
        game.Say($"Era {directions[answer!.Value].Name}.", queue: true, onDone: _ => Next());
    }
}

Menu mainMenu = null!, pause = null!;

void Finish()
{
    int record = LoadRecord();
    var text = $"Fim. Você acertou {score} de {Rounds}.";
    if (score > record)
    {
        File.WriteAllText(recordFile, JsonSerializer.Serialize(new { recorde = score }));
        text += " Novo recorde!";
    }
    else if (record > 0) text += $" Recorde: {record}.";
    answer = null;
    game.SayThen(text, () => mainMenu.Show(help: false));
}

mainMenu = new Menu(game, "Onde está o bip", new[] { "Jogar", "Instruções", "Sair" }, (_, item) =>
{
    switch (item)
    {
        case "Jogar":
            mainMenu.Close();
            number = score = 0;
            game.SayThen("Valendo! Ouça o bip e aperte o direcional para o lado dele.", Next);
            break;
        case "Instruções":
            game.Say("Um bip toca à esquerda, à frente, à direita ou atrás de você. " +
                     "Aperte o direcional para o lado de onde ele veio. " +
                     "Select repete o bip. Start pausa. Use fone de ouvido.");
            break;
        default:
            game.Exit();
            break;
    }
}) { OnBack = game.Exit, MoveSound = move };

pause = new Menu(game, "Pausa", new[] { "Continuar", "Sair para o menu" }, (_, item) =>
{
    pause.Close();
    if (item == "Continuar")
    {
        game.SayThen("Continuando.", PlayBip);
        return;
    }
    answer = null;
    mainMenu.Show(help: false);
}) { MoveSound = move };
pause.OnBack = () => pause.OnSelect(0, "Continuar");

game.Button += e =>
{
    if (e.State != ButtonState.Pressed || answer == null) return;
    if (e.Button == Button.Start)
    {
        waiting = false;
        pause.Show();
        return;
    }
    Respond(e.Button);
};
game.Start += () => mainMenu.Show();
game.Run();
