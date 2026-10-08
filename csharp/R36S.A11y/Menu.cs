namespace R36S.A11y;

/// <summary>
/// Menu vertical falado. Cima/Baixo navegam (dando a volta), B confirma, A
/// volta, Select repete o item atual. Enquanto aberto, recebe os botões no
/// lugar do jogo.
/// </summary>
public sealed class Menu : IFocus
{
    /// <summary>Instrução falada ao abrir um menu.</summary>
    public const string Help = "Cima e baixo escolhem. B confirma. A volta.";

    private readonly Game game;
    public string Title { get; set; }
    public IReadOnlyList<string> Items { get; set; }
    public Action<int, string> OnSelect { get; set; }

    /// <summary>Sem OnBack, Voltar fecha o menu.</summary>
    public Action? OnBack { get; set; }

    public Sound? MoveSound { get; set; }
    public Sound? SelectSound { get; set; }
    public int Index { get; private set; }
    public bool IsOpen { get; private set; }

    public Menu(Game game, string title, IReadOnlyList<string> items, Action<int, string> onSelect)
    {
        this.game = game;
        Title = title;
        Items = items;
        OnSelect = onSelect;
    }

    /// <summary>Abre no item index, falando o título, o item e (se help) a instrução.</summary>
    public void Show(int index = 0, bool help = true)
    {
        Index = index;
        if (!IsOpen)
        {
            IsOpen = true;
            game.PushFocus(this);
        }
        game.Say($"{Title}. {ItemText}" + (help ? ". " + Help : ""));
    }

    /// <summary>Fecha o menu e devolve os botões ao jogo.</summary>
    public void Close()
    {
        IsOpen = false;
        game.PopFocus(this);
    }

    private string ItemText => $"{Items[Index]}, {Index + 1} de {Items.Count}";

    private void Move(int delta)
    {
        Index = (Index + delta + Items.Count) % Items.Count;
        MoveSound?.Play();
        game.Say(ItemText);
    }

    public void HandleButton(ButtonEvent e)
    {
        if (e.State == ButtonState.Released) return;
        if (e.Button == Button.Down) Move(1);
        else if (e.Button == Button.Up) Move(-1);
        else if (e.State != ButtonState.Pressed) return;
        else if (Buttons.IsConfirm(e.Button))
        {
            SelectSound?.Play();
            OnSelect(Index, Items[Index]);
        }
        else if (Buttons.IsBack(e.Button))
        {
            if (OnBack != null) OnBack();
            else Close();
        }
        else if (e.Button == Button.Select) game.Say(ItemText);
    }
}
