using System.Collections.Concurrent;
using System.Diagnostics;
using System.Runtime.InteropServices;
using System.Text.Json;

namespace R36S.A11y;

/// <summary>Opções do jogo; os padrões servem para a maioria dos jogos.</summary>
public sealed record GameOptions
{
    /// <summary>Quadros por segundo do evento Update.</summary>
    public int Fps { get; init; } = 60;

    /// <summary>Gera ButtonState.Repeated enquanto o botão está apertado.</summary>
    public bool Repeat { get; init; }

    /// <summary>Entrega os analógicos no evento Axis.</summary>
    public bool Axes { get; init; }

    /// <summary>Zona morta dos analógicos.</summary>
    public double Deadzone { get; init; } = 0.2;
}

/// <summary>Recebe os botões antes do jogo (por exemplo, um Menu aberto).</summary>
public interface IFocus
{
    void HandleButton(ButtonEvent e);
}

/// <summary>
/// Um jogo acessível do R36S. Tudo roda na thread de Run: os eventos nunca
/// rodam em paralelo.
/// </summary>
public sealed class Game
{
    private static readonly TimeSpan RepeatDelay = TimeSpan.FromMilliseconds(400);
    private static readonly TimeSpan RepeatInterval = TimeSpan.FromMilliseconds(100);

    public string Id { get; }
    public string Name { get; }
    public string Dir { get; }
    public string SaveDir { get; }
    public string Lang { get; }
    public Audio Sound { get; }
    public Speech Speech { get; }

    /// <summary>Botão apertado, solto ou repetido.</summary>
    public event Action<ButtonEvent>? Button;

    /// <summary>Analógico mudou (só com GameOptions.Axes).</summary>
    public event Action<AxisEvent>? Axis;

    /// <summary>A cada quadro, com o tempo do quadro em segundos.</summary>
    public event Action<double>? Update;

    /// <summary>Desenha um quadro, logo depois do Update. Abre a tela.</summary>
    public event Action<Graphics>? Draw;

    private Graphics? gfx;

    /// <summary>Os gráficos; abre a tela no primeiro uso (para carregar imagens e fontes antes do Run).</summary>
    public Graphics Gfx => gfx ??= new Graphics(Name, Dir);

    /// <summary>Quando o laço começa.</summary>
    public event Action? Start;

    /// <summary>Ao sair: hora de salvar.</summary>
    public event Action? Quit;

    private readonly GameOptions options;
    private readonly InputState input;
    private readonly DateTimeOffset start = DateTimeOffset.UtcNow;
    private readonly Stopwatch clock = Stopwatch.StartNew();
    private readonly ConcurrentQueue<Action> queue = new();
    private readonly AutoResetEvent wake = new(false);
    private readonly List<(TimeSpan At, Action Fn)> timers = new();
    private readonly Dictionary<Button, TimeSpan> repeatAt = new();
    private readonly List<IFocus> focus = new();
    private volatile bool running;

    public Game(GameOptions? options = null)
    {
        this.options = options ?? new GameOptions();
        Dir = Environment.GetEnvironmentVariable("A11Y_GAME_DIR") ?? AppContext.BaseDirectory;
        string? id = null, name = null;
        try
        {
            using var doc = JsonDocument.Parse(File.ReadAllText(Path.Combine(Dir, "game.json")));
            if (doc.RootElement.TryGetProperty("id", out var i)) id = i.GetString();
            if (doc.RootElement.TryGetProperty("name", out var n)) name = n.GetString();
        }
        catch (Exception) { }
        Id = id ?? new DirectoryInfo(Dir).Name;
        Name = name ?? Id;
        Lang = Environment.GetEnvironmentVariable("A11Y_LANG") ?? "pt-BR";
        SaveDir = Environment.GetEnvironmentVariable("A11Y_SAVE_DIR") ?? Path.Combine(
            Environment.GetFolderPath(Environment.SpecialFolder.UserProfile), ".local", "share", "r36s-a11y", Id);
        Directory.CreateDirectory(SaveDir);
        Speech = new Speech(Id, Lang, Post);
        Sound = new Audio(Dir);
        input = new InputState(this.options.Deadzone);
    }

    /// <summary>Roda uma ação na thread do jogo (seguro chamar de qualquer thread).</summary>
    public void Post(Action action)
    {
        queue.Enqueue(action);
        wake.Set();
    }

    /// <summary>Fala o texto. Com queue=false, interrompe a fala anterior.</summary>
    public void Say(string text, bool queue = false, Action<bool>? onDone = null) =>
        Speech.Say(text, queue, onDone);

    /// <summary>Fala o texto e chama then quando a fala termina.</summary>
    public void SayThen(string text, Action then) => Speech.Say(text, false, _ => then());

    /// <summary>Cala a fala.</summary>
    public void Hush() => Speech.Hush();

    /// <summary>Chama fn daqui a alguns segundos.</summary>
    public void After(double seconds, Action fn) =>
        timers.Add((clock.Elapsed + TimeSpan.FromSeconds(seconds), fn));

    /// <summary>Milissegundos desde o início, na escala dos eventos de botão.</summary>
    public long Ms => (long)(DateTimeOffset.UtcNow - start).TotalMilliseconds;

    /// <summary>Último valor de um analógico.</summary>
    public double AxisValue(Axis axis) => input.AxisValue.GetValueOrDefault(axis);

    /// <summary>Encerra o laço.</summary>
    public void Exit() => running = false;

    public void PushFocus(IFocus f) => focus.Add(f);

    public void PopFocus(IFocus f) => focus.Remove(f);

    private void DispatchButton(ButtonEvent e)
    {
        if (focus.Count > 0) focus[^1].HandleButton(e);
        else Button?.Invoke(e);
    }

    private void Handle(object ev)
    {
        switch (ev)
        {
            case ButtonEvent b:
                if (options.Repeat)
                {
                    if (b.State == ButtonState.Pressed) repeatAt[b.Button] = clock.Elapsed + RepeatDelay;
                    else repeatAt.Remove(b.Button);
                }
                DispatchButton(b);
                break;
            case AxisEvent a when options.Axes:
                Axis?.Invoke(a);
                break;
        }
    }

    private void Tick()
    {
        var now = clock.Elapsed;
        foreach (var (b, at) in repeatAt.ToArray())
        {
            if (now < at) continue;
            repeatAt[b] = at + RepeatInterval;
            DispatchButton(new ButtonEvent(b, ButtonState.Repeated, Ms));
        }
        if (timers.Count == 0) return;
        var due = timers.Where(t => t.At <= now).OrderBy(t => t.At).ToList();
        timers.RemoveAll(t => t.At <= now);
        foreach (var t in due) t.Fn();
    }

    /// <summary>Roda o laço do jogo até Exit, SIGTERM ou SIGINT.</summary>
    public void Run()
    {
        int devices = GamepadReader.Open(input, start, (type, code, value, ms) => Post(() =>
        {
            if (input.Feed(type, code, value, ms) is { } e) Handle(e);
        }));
        if (devices == 0) Console.Error.WriteLine("[a11y] nenhum controle encontrado em /dev/input");

        using var term = PosixSignalRegistration.Create(PosixSignal.SIGTERM, c => { c.Cancel = true; Stop(); });
        using var intr = PosixSignalRegistration.Create(PosixSignal.SIGINT, c => { c.Cancel = true; Stop(); });

        if (Draw != null) _ = Gfx; // abre a tela antes do primeiro quadro
        running = true;
        Start?.Invoke();
        var frame = TimeSpan.FromSeconds(1.0 / Math.Max(1, options.Fps));
        var last = clock.Elapsed;
        try
        {
            while (running)
            {
                var wait = last + frame - clock.Elapsed;
                if (wait > TimeSpan.Zero) wake.WaitOne(wait);
                while (running && queue.TryDequeue(out var action)) action();
                var now = clock.Elapsed;
                if (now - last < frame) continue;
                double dt = (now - last).TotalSeconds;
                last = now;
                Tick();
                Sound.Update(dt);
                Update?.Invoke(dt);
                if (Draw != null && gfx is { Available: true })
                {
                    gfx.Begin();
                    Draw(gfx);
                    gfx.End();
                }
            }
        }
        finally
        {
            try { Quit?.Invoke(); }
            catch (Exception ex) { Console.Error.WriteLine("[a11y] erro ao sair: " + ex.Message); }
            Speech.Close();
            Sound.Close();
            gfx?.Close();
        }
    }

    private void Stop()
    {
        running = false;
        wake.Set();
    }
}
