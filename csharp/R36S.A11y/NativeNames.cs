using System.Reflection;
using System.Runtime.CompilerServices;
using System.Runtime.InteropServices;

namespace R36S.A11y;

/// <summary>
/// As bibliotecas nativas têm o nome do console (libSDL2-2.0.so.0...). No
/// Windows e no macOS, tenta os nomes equivalentes, para o jogo rodar no PC
/// quando elas estiverem instaladas.
/// </summary>
internal static class NativeNames
{
    private static readonly Dictionary<string, string[]> Alternatives = new()
    {
        ["libSDL2-2.0.so.0"] = new[] { "SDL2", "libSDL2-2.0.0.dylib" },
        ["libSDL2_image-2.0.so.0"] = new[] { "SDL2_image", "libSDL2_image-2.0.0.dylib" },
        ["libSDL2_ttf-2.0.so.0"] = new[] { "SDL2_ttf", "libSDL2_ttf-2.0.0.dylib" },
        ["libSDL2_gfx-1.0.so.0"] = new[] { "SDL2_gfx", "libSDL2_gfx-1.0.0.dylib" },
        ["libopenal.so.1"] = new[] { "OpenAL32", "soft_oal", "libopenal.1.dylib" },
        ["libvorbisfile.so.3"] = new[] { "vorbisfile", "libvorbisfile-3", "libvorbisfile.3.dylib" },
    };

#pragma warning disable CA2255 // registrar o resolvedor ao carregar a biblioteca é justamente o caso de uso
    [ModuleInitializer]
#pragma warning restore CA2255
    internal static void Register() =>
        NativeLibrary.SetDllImportResolver(typeof(NativeNames).Assembly, Resolve);

    private static IntPtr Resolve(string name, Assembly assembly, DllImportSearchPath? path)
    {
        if (NativeLibrary.TryLoad(name, assembly, path, out var handle)) return handle;
        if (Alternatives.TryGetValue(name, out var alternatives))
            foreach (var alt in alternatives)
                if (NativeLibrary.TryLoad(alt, assembly, path, out handle) ||
                    NativeLibrary.TryLoad(alt, out handle)) // busca padrão do sistema (inclui o PATH)
                    return handle;
        return IntPtr.Zero;
    }
}
