using R36S.A11y;
using Xunit;

public class InputTests
{
    private const ushort B = 0x130, Up = 0x220, Vol = 115;

    [Fact]
    public void PressAndRelease()
    {
        var s = new InputState(0.2);
        Assert.Equal(new ButtonEvent(Button.B, ButtonState.Pressed, 10), s.Feed(InputState.EvKey, B, 1, 10));
        Assert.Equal(new ButtonEvent(Button.B, ButtonState.Released, 20), s.Feed(InputState.EvKey, B, 0, 20));
    }

    [Fact]
    public void FnNeverDelivered()
    {
        var s = new InputState(0.2);
        Assert.Null(s.Feed(InputState.EvKey, InputState.FnCode, 1, 0));
        Assert.Null(s.Feed(InputState.EvKey, InputState.FnCode, 0, 1));
    }

    [Fact]
    public void ButtonWithFnSuppressedUntilReleased()
    {
        var s = new InputState(0.2);
        s.Feed(InputState.EvKey, InputState.FnCode, 1, 0);
        Assert.Null(s.Feed(InputState.EvKey, Up, 1, 1));
        s.Feed(InputState.EvKey, InputState.FnCode, 0, 2); // solta Fn antes do botão
        Assert.Null(s.Feed(InputState.EvKey, Up, 0, 3));
        Assert.Equal(new ButtonEvent(Button.Up, ButtonState.Pressed, 4), s.Feed(InputState.EvKey, Up, 1, 4));
    }

    [Fact]
    public void SystemKeysAndKernelRepeatIgnored()
    {
        var s = new InputState(0.2);
        Assert.Null(s.Feed(InputState.EvKey, Vol, 1, 0));
        s.Feed(InputState.EvKey, B, 1, 0);
        Assert.Null(s.Feed(InputState.EvKey, B, 2, 5));
    }

    [Fact]
    public void AxisDeadzoneAndScale()
    {
        var s = new InputState(0.2);
        s.SetAxisRange(0, -1800, 1800);
        Assert.Null(s.Feed(InputState.EvAbs, 0, 100, 0));
        Assert.Equal(new AxisEvent(Axis.LX, 1, 1), s.Feed(InputState.EvAbs, 0, 1800, 1));
        Assert.Equal(new AxisEvent(Axis.LX, 0, 2), s.Feed(InputState.EvAbs, 0, 0, 2));
    }

    [Fact]
    public void EscapeText() =>
        Assert.Equal("oi\r\n..ponto\r\n.\r\n", Speech.EscapeText("oi\n.ponto"));

    [Fact]
    public void DecodeWav()
    {
        var path = Path.Combine(AppContext.BaseDirectory, "..", "..", "..", "..", "..", "python", "examples", "onde-esta-o-bip", "sounds", "bip.wav");
        var (channels, width, rate, data) = Audio.DecodeWav(path);
        Assert.Equal((1, 2, 44100), (channels, width, rate));
        Assert.NotEmpty(data);
    }

    [Fact]
    public void ReverbPresetsHaveAllParameters()
    {
        foreach (var preset in Audio.ReverbPresets.Values) Assert.Equal(10, preset.Count);
    }
}
