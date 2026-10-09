using System.Globalization;
using System.Runtime.InteropServices;

namespace R36S.A11y;

/// <summary>Cor RGBA de 0 a 255.</summary>
public readonly record struct Color(byte R, byte G, byte B, byte A = 255)
{
    /// <summary>Lê "#rrggbb" ou "#rrggbbaa".</summary>
    public static Color Hex(string hex)
    {
        hex = hex.TrimStart('#');
        uint v = uint.Parse(hex, NumberStyles.HexNumber);
        return hex.Length == 8
            ? new Color((byte)(v >> 24), (byte)(v >> 16), (byte)(v >> 8), (byte)v)
            : new Color((byte)(v >> 16), (byte)(v >> 8), (byte)v);
    }

    public static readonly Color White = new(255, 255, 255), Black = new(0, 0, 0);

    internal uint Packed => R | (uint)G << 8 | (uint)B << 16 | (uint)A << 24; // SDL_Color por valor
}

/// <summary>Uma imagem (textura). Desenhe com Graphics.Draw.</summary>
public sealed class Image
{
    internal IntPtr Texture;
    public int Width { get; internal set; }
    public int Height { get; internal set; }
}

/// <summary>Uma fonte TTF num tamanho.</summary>
public sealed class Font
{
    internal IntPtr Handle;
    public int Size { get; internal init; }
}

public enum Align { Left, Center, Right }

/// <summary>
/// Gráficos 2D opcionais pela SDL2 do sistema: formas, sprites e texto. A área
/// de desenho é sempre 640x480 (escalada para a tela real); (0, 0) é o canto
/// superior esquerdo. Os gráficos são um extra: o jogo continua tendo de ser
/// jogável só pelo som.
/// </summary>
public sealed class Graphics
{
    public const int Width = 640, Height = 480;

    private static readonly string[] DefaultFonts =
    {
        "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
        "C:/Windows/Fonts/segoeui.ttf",
        "C:/Windows/Fonts/arial.ttf",
    };

    private readonly string baseDir;
    private readonly IntPtr window, renderer;
    private readonly Dictionary<(IntPtr, string, Color), (Image Image, int Unused)> textCache = new();
    private Font? defaultFont;

    /// <summary>A tela abriu.</summary>
    public bool Available { get; private set; }

    internal Graphics(string title, string baseDir)
    {
        this.baseDir = baseDir;
        try
        {
            if (Sdl.SDL_Init(0x20) != 0) { Warn(); return; } // SDL_INIT_VIDEO
            uint flags = Directory.Exists("/opt/a11y") ? 0x1001u : 0x4u; // tela cheia no console
            window = Sdl.SDL_CreateWindow(title, 0x2FFF0000, 0x2FFF0000, Width, Height, flags);
            if (window == IntPtr.Zero) { Warn(); return; }
            renderer = Sdl.SDL_CreateRenderer(window, -1, 0x2); // acelerado
            if (renderer == IntPtr.Zero) renderer = Sdl.SDL_CreateRenderer(window, -1, 0); // sem GPU (testes)
            if (renderer == IntPtr.Zero) { Warn(); return; }
            Sdl.SDL_RenderSetLogicalSize(renderer, Width, Height);
            Sdl.SDL_SetRenderDrawBlendMode(renderer, 1);
            Sdl.SDL_ShowCursor(0);
            Sdl.IMG_Init(0x2);
            Sdl.TTF_Init();
            Available = true;
        }
        catch (Exception ex) when (ex is DllNotFoundException or EntryPointNotFoundException)
        {
            Console.Error.WriteLine("[a11y] sem gráficos: " + ex.Message);
        }
    }

    private static void Warn() => Console.Error.WriteLine("[a11y] sem gráficos: " + Sdl.SDL_GetError());

    private string FullPath(string path) => Path.IsPathRooted(path) ? path : Path.Combine(baseDir, path);

    internal void Begin()
    {
        Sdl.SDL_PumpEvents(); // a SDL precisa disso para manter a tela viva
        Clear(Color.Black);
    }

    internal void End()
    {
        Sdl.SDL_RenderPresent(renderer);
        // textos que não foram desenhados por 60 quadros saem do cache
        foreach (var (key, entry) in textCache.ToArray())
        {
            if (entry.Unused + 1 > 60)
            {
                Sdl.SDL_DestroyTexture(entry.Image.Texture);
                textCache.Remove(key);
            }
            else textCache[key] = (entry.Image, entry.Unused + 1);
        }
    }

    /// <summary>Pinta a tela inteira.</summary>
    public void Clear(Color c)
    {
        if (!Available) return;
        Sdl.SDL_SetRenderDrawColor(renderer, c.R, c.G, c.B, c.A);
        Sdl.SDL_RenderClear(renderer);
    }

    /// <summary>Retângulo preenchido (fill: true) ou só o contorno.</summary>
    public void Rect(double x, double y, double w, double h, Color c, bool fill = true)
    {
        if (!Available) return;
        Sdl.SDL_SetRenderDrawColor(renderer, c.R, c.G, c.B, c.A);
        var r = new Sdl.Rect((int)x, (int)y, (int)w, (int)h);
        if (fill) Sdl.SDL_RenderFillRect(renderer, ref r);
        else Sdl.SDL_RenderDrawRect(renderer, ref r);
    }

    /// <summary>Linha; width 1 é suavizada.</summary>
    public void Line(double x1, double y1, double x2, double y2, Color c, double width = 1)
    {
        if (!Available) return;
        if (width <= 1) Sdl.aalineRGBA(renderer, (short)x1, (short)y1, (short)x2, (short)y2, c.R, c.G, c.B, c.A);
        else Sdl.thickLineRGBA(renderer, (short)x1, (short)y1, (short)x2, (short)y2, (byte)width, c.R, c.G, c.B, c.A);
    }

    /// <summary>Círculo preenchido (fill: true) ou só o contorno.</summary>
    public void Circle(double x, double y, double radius, Color c, bool fill = true)
    {
        if (!Available) return;
        if (fill) Sdl.filledCircleRGBA(renderer, (short)x, (short)y, (short)radius, c.R, c.G, c.B, c.A);
        else Sdl.aacircleRGBA(renderer, (short)x, (short)y, (short)radius, c.R, c.G, c.B, c.A);
    }

    /// <summary>Polígono (3 ou mais pontos) preenchido ou só o contorno.</summary>
    public void Polygon(IReadOnlyList<(double X, double Y)> points, Color c, bool fill = true)
    {
        if (!Available || points.Count < 3) return;
        var vx = points.Select(p => (short)p.X).ToArray();
        var vy = points.Select(p => (short)p.Y).ToArray();
        if (fill) Sdl.filledPolygonRGBA(renderer, vx, vy, vx.Length, c.R, c.G, c.B, c.A);
        else Sdl.aapolygonRGBA(renderer, vx, vy, vx.Length, c.R, c.G, c.B, c.A);
    }

    /// <summary>Um ponto.</summary>
    public void Point(double x, double y, Color c)
    {
        if (Available) Sdl.pixelRGBA(renderer, (short)x, (short)y, c.R, c.G, c.B, c.A);
    }

    /// <summary>Carrega um PNG (ou outro formato da SDL2_image), relativo à pasta do jogo.</summary>
    public Image LoadImage(string path)
    {
        if (!Available) return new Image();
        IntPtr surface = Sdl.IMG_Load(FullPath(path));
        if (surface == IntPtr.Zero) throw new IOException($"{path}: {Sdl.SDL_GetError()}");
        IntPtr texture = Sdl.SDL_CreateTextureFromSurface(renderer, surface);
        Sdl.SDL_FreeSurface(surface);
        return Wrap(texture);
    }

    private static Image Wrap(IntPtr texture)
    {
        Sdl.SDL_QueryTexture(texture, IntPtr.Zero, IntPtr.Zero, out int w, out int h);
        Sdl.SDL_SetTextureBlendMode(texture, 1);
        return new Image { Texture = texture, Width = w, Height = h };
    }

    /// <summary>
    /// Desenha a imagem com o canto superior esquerdo em (x, y). src recorta um
    /// quadro de uma folha de sprites; rotation em graus, em torno do centro;
    /// tint multiplica as cores.
    /// </summary>
    public void Draw(Image image, double x, double y, (int X, int Y, int W, int H)? src = null,
                     double scale = 1, double rotation = 0, bool flipX = false, bool flipY = false,
                     byte alpha = 255, Color? tint = null)
    {
        if (!Available || image.Texture == IntPtr.Zero) return;
        var s = src ?? (0, 0, image.Width, image.Height);
        var srect = new Sdl.Rect(s.X, s.Y, s.W, s.H);
        var drect = new Sdl.Rect((int)x, (int)y, (int)(s.W * scale), (int)(s.H * scale));
        var t = tint ?? Color.White;
        Sdl.SDL_SetTextureColorMod(image.Texture, t.R, t.G, t.B);
        Sdl.SDL_SetTextureAlphaMod(image.Texture, alpha);
        Sdl.SDL_RenderCopyEx(renderer, image.Texture, ref srect, ref drect, rotation, IntPtr.Zero,
                             (flipX ? 1 : 0) | (flipY ? 2 : 0));
    }

    /// <summary>Carrega uma fonte TTF; sem path, usa a fonte padrão do sistema.</summary>
    public Font LoadFont(string? path = null, int size = 24)
    {
        if (!Available) return new Font { Size = size };
        path ??= DefaultFonts.FirstOrDefault(File.Exists) ?? DefaultFonts[0];
        IntPtr handle = Sdl.TTF_OpenFont(FullPath(path), size);
        if (handle == IntPtr.Zero) throw new IOException($"{path}: {Sdl.SDL_GetError()}");
        return new Font { Handle = handle, Size = size };
    }

    private Font FontOrDefault(Font? font) => font ?? (defaultFont ??= LoadFont());

    /// <summary>Largura e altura do texto em pixels.</summary>
    public (int Width, int Height) Measure(string text, Font? font = null)
    {
        if (!Available || text.Length == 0) return (0, 0);
        Sdl.TTF_SizeUTF8(FontOrDefault(font).Handle, text, out int w, out int h);
        return (w, h);
    }

    /// <summary>Escreve o texto com o topo em y (sem font, usa a padrão de 24 px).</summary>
    public void Text(string text, double x, double y, Font? font = null, Color? color = null,
                     Align align = Align.Left)
    {
        if (!Available || text.Length == 0) return;
        var f = FontOrDefault(font);
        var c = color ?? Color.White;
        var key = (f.Handle, text, c);
        if (!textCache.TryGetValue(key, out var entry))
        {
            IntPtr surface = Sdl.TTF_RenderUTF8_Blended(f.Handle, text, c.Packed);
            if (surface == IntPtr.Zero) return;
            IntPtr texture = Sdl.SDL_CreateTextureFromSurface(renderer, surface);
            Sdl.SDL_FreeSurface(surface);
            entry = (Wrap(texture), 0);
        }
        textCache[key] = (entry.Image, 0);
        if (align == Align.Center) x -= entry.Image.Width / 2.0;
        else if (align == Align.Right) x -= entry.Image.Width;
        Draw(entry.Image, x, y);
    }

    /// <summary>Salva o quadro atual em PNG (chame no fim do Draw). Útil para testar.</summary>
    public void Screenshot(string path)
    {
        if (!Available) return;
        const uint argb8888 = 0x16762004;
        Sdl.SDL_GetRendererOutputSize(renderer, out int w, out int h);
        IntPtr surface = Sdl.SDL_CreateRGBSurfaceWithFormat(0, w, h, 32, argb8888);
        try
        {
            int pitch = Marshal.ReadInt32(surface, 24);    // SDL_Surface.pitch
            IntPtr pixels = Marshal.ReadIntPtr(surface, 32); // SDL_Surface.pixels
            Sdl.SDL_RenderReadPixels(renderer, IntPtr.Zero, argb8888, pixels, pitch);
            if (Sdl.IMG_SavePNG(surface, FullPath(path)) != 0) throw new IOException(Sdl.SDL_GetError());
        }
        finally
        {
            Sdl.SDL_FreeSurface(surface);
        }
    }

    internal void Close()
    {
        if (!Available) return;
        foreach (var (image, _) in textCache.Values) Sdl.SDL_DestroyTexture(image.Texture);
        textCache.Clear();
        Sdl.SDL_DestroyRenderer(renderer);
        Sdl.SDL_DestroyWindow(window);
        Sdl.SDL_QuitSubSystem(0x20);
        Available = false;
    }
}

internal static class Sdl
{
    private const string Lib = "libSDL2-2.0.so.0", Img = "libSDL2_image-2.0.so.0",
        Ttf = "libSDL2_ttf-2.0.so.0", Gfx = "libSDL2_gfx-1.0.so.0";

    [StructLayout(LayoutKind.Sequential)]
    internal record struct Rect(int X, int Y, int W, int H);

    [DllImport(Lib)] internal static extern int SDL_Init(uint flags);
    [DllImport(Lib)] internal static extern IntPtr SDL_CreateWindow([MarshalAs(UnmanagedType.LPUTF8Str)] string title, int x, int y, int w, int h, uint flags);
    [DllImport(Lib)] internal static extern IntPtr SDL_CreateRenderer(IntPtr window, int index, uint flags);
    [DllImport(Lib)] internal static extern int SDL_RenderSetLogicalSize(IntPtr r, int w, int h);
    [DllImport(Lib)] internal static extern int SDL_SetRenderDrawBlendMode(IntPtr r, int mode);
    [DllImport(Lib)] internal static extern int SDL_SetRenderDrawColor(IntPtr r, byte red, byte green, byte blue, byte alpha);
    [DllImport(Lib)] internal static extern int SDL_RenderClear(IntPtr r);
    [DllImport(Lib)] internal static extern int SDL_RenderFillRect(IntPtr r, ref Rect rect);
    [DllImport(Lib)] internal static extern int SDL_RenderDrawRect(IntPtr r, ref Rect rect);
    [DllImport(Lib)] internal static extern void SDL_RenderPresent(IntPtr r);
    [DllImport(Lib)] internal static extern void SDL_PumpEvents();
    [DllImport(Lib)] internal static extern int SDL_ShowCursor(int toggle);
    [DllImport(Lib)] internal static extern IntPtr SDL_CreateTextureFromSurface(IntPtr r, IntPtr surface);
    [DllImport(Lib)] internal static extern void SDL_FreeSurface(IntPtr surface);
    [DllImport(Lib)] internal static extern int SDL_QueryTexture(IntPtr t, IntPtr format, IntPtr access, out int w, out int h);
    [DllImport(Lib)] internal static extern int SDL_RenderCopyEx(IntPtr r, IntPtr t, ref Rect src, ref Rect dst, double angle, IntPtr center, int flip);
    [DllImport(Lib)] internal static extern int SDL_SetTextureAlphaMod(IntPtr t, byte a);
    [DllImport(Lib)] internal static extern int SDL_SetTextureColorMod(IntPtr t, byte r, byte g, byte b);
    [DllImport(Lib)] internal static extern int SDL_SetTextureBlendMode(IntPtr t, int mode);
    [DllImport(Lib)] internal static extern void SDL_DestroyTexture(IntPtr t);
    [DllImport(Lib)] internal static extern void SDL_DestroyRenderer(IntPtr r);
    [DllImport(Lib)] internal static extern void SDL_DestroyWindow(IntPtr w);
    [DllImport(Lib)] internal static extern void SDL_QuitSubSystem(uint flags);
    [DllImport(Lib)] [return: MarshalAs(UnmanagedType.LPUTF8Str)] internal static extern string SDL_GetError();
    [DllImport(Lib)] internal static extern IntPtr SDL_CreateRGBSurfaceWithFormat(uint flags, int w, int h, int depth, uint format);
    [DllImport(Lib)] internal static extern int SDL_RenderReadPixels(IntPtr r, IntPtr rect, uint format, IntPtr pixels, int pitch);
    [DllImport(Lib)] internal static extern int SDL_GetRendererOutputSize(IntPtr r, out int w, out int h);
    [DllImport(Img)] internal static extern int IMG_Init(int flags);
    [DllImport(Img)] internal static extern IntPtr IMG_Load([MarshalAs(UnmanagedType.LPUTF8Str)] string path);
    [DllImport(Img)] internal static extern int IMG_SavePNG(IntPtr surface, [MarshalAs(UnmanagedType.LPUTF8Str)] string path);
    [DllImport(Ttf)] internal static extern int TTF_Init();
    [DllImport(Ttf)] internal static extern IntPtr TTF_OpenFont([MarshalAs(UnmanagedType.LPUTF8Str)] string path, int size);
    [DllImport(Ttf)] internal static extern IntPtr TTF_RenderUTF8_Blended(IntPtr font, [MarshalAs(UnmanagedType.LPUTF8Str)] string text, uint color);
    [DllImport(Ttf)] internal static extern int TTF_SizeUTF8(IntPtr font, [MarshalAs(UnmanagedType.LPUTF8Str)] string text, out int w, out int h);
    [DllImport(Gfx)] internal static extern int filledCircleRGBA(IntPtr r, short x, short y, short rad, byte red, byte green, byte blue, byte alpha);
    [DllImport(Gfx)] internal static extern int aacircleRGBA(IntPtr r, short x, short y, short rad, byte red, byte green, byte blue, byte alpha);
    [DllImport(Gfx)] internal static extern int thickLineRGBA(IntPtr r, short x1, short y1, short x2, short y2, byte width, byte red, byte green, byte blue, byte alpha);
    [DllImport(Gfx)] internal static extern int aalineRGBA(IntPtr r, short x1, short y1, short x2, short y2, byte red, byte green, byte blue, byte alpha);
    [DllImport(Gfx)] internal static extern int filledPolygonRGBA(IntPtr r, short[] vx, short[] vy, int n, byte red, byte green, byte blue, byte alpha);
    [DllImport(Gfx)] internal static extern int aapolygonRGBA(IntPtr r, short[] vx, short[] vy, int n, byte red, byte green, byte blue, byte alpha);
    [DllImport(Gfx)] internal static extern int pixelRGBA(IntPtr r, short x, short y, byte red, byte green, byte blue, byte alpha);
}
