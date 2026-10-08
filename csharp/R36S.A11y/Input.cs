using System.Buffers.Binary;
using System.Runtime.InteropServices;

namespace R36S.A11y;

/// <summary>Botões do contrato. B (de baixo) confirma, A (da direita) volta.</summary>
public enum Button { Up, Down, Left, Right, A, B, X, Y, L1, R1, L2, R2, L3, R3, Start, Select }

/// <summary>Estado de um evento de botão.</summary>
public enum ButtonState { Pressed, Released, Repeated }

/// <summary>Analógicos, de -1 a 1.</summary>
public enum Axis { LX, LY, RX, RY }

/// <summary>Botão apertado, solto ou repetido; Ms é o tempo desde o início do jogo.</summary>
public readonly record struct ButtonEvent(Button Button, ButtonState State, long Ms);

/// <summary>Analógico que mudou.</summary>
public readonly record struct AxisEvent(Axis Axis, double Value, long Ms);

public static class Buttons
{
    /// <summary>B (de baixo) confirma.</summary>
    public static bool IsConfirm(Button b) => b == Button.B;

    /// <summary>A (da direita) volta.</summary>
    public static bool IsBack(Button b) => b == Button.A;
}

/// <summary>
/// Transforma eventos crus do kernel em eventos do contrato: descarta Fn e o
/// que for apertado com ele, volume, Power e o auto-repeat do kernel, e aplica
/// zona morta aos eixos. Separado da leitura para ser testável.
/// </summary>
internal sealed class InputState
{
    internal const ushort EvKey = 0x01, EvAbs = 0x03;
    internal const ushort FnCode = 0x2C4; // BTN_TRIGGER_HAPPY5: do sistema (a11yd)

    internal static readonly Dictionary<ushort, Button> ButtonCodes = new()
    {
        [0x130] = Button.B, [0x131] = Button.A, [0x133] = Button.X, [0x134] = Button.Y,
        [0x136] = Button.L1, [0x137] = Button.R1, [0x138] = Button.L2, [0x139] = Button.R2,
        [0x220] = Button.Up, [0x221] = Button.Down, [0x222] = Button.Left, [0x223] = Button.Right,
        [0x2C0] = Button.Select, [0x2C1] = Button.Start, [0x2C2] = Button.L3, [0x2C3] = Button.R3,
    };

    internal static readonly Dictionary<ushort, Axis> AxisCodes = new()
    {
        [0x00] = Axis.LX, [0x01] = Axis.LY, [0x03] = Axis.RX, [0x04] = Axis.RY,
    };

    private readonly double deadzone;
    private bool fnHeld;
    private readonly HashSet<Button> suppressed = new(), held = new();
    private readonly Dictionary<ushort, (int Min, int Max)> axisRange = new();
    internal readonly Dictionary<Axis, double> AxisValue = new();

    internal InputState(double deadzone) => this.deadzone = deadzone;

    internal void SetAxisRange(ushort code, int min, int max) => axisRange[code] = (min, max);

    /// <summary>Devolve um ButtonEvent, um AxisEvent ou null.</summary>
    internal object? Feed(ushort type, ushort code, int value, long ms) => type switch
    {
        EvKey => Key(code, value, ms),
        EvAbs => Abs(code, value, ms),
        _ => null,
    };

    private object? Key(ushort code, int value, long ms)
    {
        if (code == FnCode)
        {
            fnHeld = value != 0;
            return null;
        }
        if (!ButtonCodes.TryGetValue(code, out var b) || value == 2) return null;
        if (value == 1)
        {
            if (fnHeld)
            {
                suppressed.Add(b);
                return null;
            }
            held.Add(b);
            return new ButtonEvent(b, ButtonState.Pressed, ms);
        }
        if (suppressed.Remove(b)) return null;
        if (!held.Remove(b)) return null;
        return new ButtonEvent(b, ButtonState.Released, ms);
    }

    private object? Abs(ushort code, int raw, long ms)
    {
        if (!AxisCodes.TryGetValue(code, out var axis)) return null;
        var (min, max) = axisRange.TryGetValue(code, out var r) ? r : (-1800, 1800);
        double center = (min + max) / 2.0, half = (max - min) / 2.0;
        if (half == 0) half = 1;
        double v = Math.Clamp((raw - center) / half, -1, 1);
        v = Math.Abs(v) < deadzone ? 0 : Math.CopySign((Math.Abs(v) - deadzone) / (1 - deadzone), v);
        double last = AxisValue.GetValueOrDefault(axis);
        if (v == last || (v != 0 && Math.Abs(v - last) < 0.02)) return null;
        AxisValue[axis] = v;
        return new AxisEvent(axis, v, ms);
    }
}

/// <summary>Lê /dev/input/event* sem exclusividade (sem EVIOCGRAB).</summary>
internal static class GamepadReader
{
    private const int EventSize = 24; // timeval (2 x long) + type + code + value

    [DllImport("libc", SetLastError = true)]
    private static extern int ioctl(int fd, nuint request, int[] data);

    private static nuint Eviocgabs(ushort axis) =>
        (nuint)(2u << 30 | 24u << 16 | (uint)'E' << 8 | (0x40u + axis));

    /// <summary>Abre os dispositivos e chama onEvent (de outras threads) a cada evento.</summary>
    internal static int Open(InputState state, DateTimeOffset start, Action<ushort, ushort, int, long> onEvent)
    {
        if (!OperatingSystem.IsLinux()) return 0;
        int opened = 0;
        foreach (var path in Directory.GetFiles("/dev/input", "event*").Order())
        {
            FileStream fs;
            try
            {
                fs = new FileStream(path, FileMode.Open, FileAccess.Read, FileShare.ReadWrite, 0);
            }
            catch (Exception)
            {
                continue;
            }
            opened++;
            int fd = (int)fs.SafeFileHandle.DangerousGetHandle();
            foreach (var code in InputState.AxisCodes.Keys)
            {
                var info = new int[6];
                try
                {
                    if (ioctl(fd, Eviocgabs(code), info) == 0 && info[2] > info[1])
                        state.SetAxisRange(code, info[1], info[2]);
                }
                catch (Exception) { }
            }
            var thread = new Thread(() => ReadLoop(fs, start, onEvent)) { IsBackground = true };
            thread.Start();
        }
        return opened;
    }

    private static void ReadLoop(FileStream fs, DateTimeOffset start, Action<ushort, ushort, int, long> onEvent)
    {
        var buf = new byte[EventSize * 64];
        long startMs = start.ToUnixTimeMilliseconds();
        try
        {
            while (true)
            {
                int n = fs.Read(buf, 0, buf.Length);
                if (n <= 0) return;
                for (int i = 0; i + EventSize <= n; i += EventSize)
                {
                    var b = buf.AsSpan(i, EventSize);
                    long sec = BinaryPrimitives.ReadInt64LittleEndian(b);
                    long usec = BinaryPrimitives.ReadInt64LittleEndian(b[8..]);
                    onEvent(BinaryPrimitives.ReadUInt16LittleEndian(b[16..]),
                            BinaryPrimitives.ReadUInt16LittleEndian(b[18..]),
                            BinaryPrimitives.ReadInt32LittleEndian(b[20..]),
                            sec * 1000 + usec / 1000 - startMs);
                }
            }
        }
        catch (Exception)
        {
            // dispositivo removido: só para de ler
        }
        finally
        {
            fs.Dispose();
        }
    }
}
