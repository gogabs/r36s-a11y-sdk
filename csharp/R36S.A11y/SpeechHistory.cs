using System.Text;

namespace R36S.A11y;

/// <summary>
/// Histórico de falas do sistema (/opt/a11y/log/historico-falas.txt).
/// Cada fala do jogo vira uma linha no fim do arquivo; o a11yd deixa o jogador
/// voltar e avançar nas últimas 50 com Fn + L1 e Fn + R1. Formato comum a todas
/// as bibliotecas-ponte: uma fala por linha, em UTF-8, com as quebras de linha
/// trocadas por espaço, só acrescentando no fim. Quem vê o arquivo passar de
/// TrimBytes o reescreve com as últimas MaxEntries linhas. Sem a pasta do
/// arquivo (no PC, por exemplo), nada é gravado.
/// </summary>
internal static class SpeechHistory
{
    internal const int MaxEntries = 50;
    internal const long TrimBytes = 16384;
    private static readonly UTF8Encoding Utf8 = new(false);

    internal static string DefaultPath =>
        Environment.GetEnvironmentVariable("A11Y_HISTORICO_FALAS") is { Length: > 0 } p
            ? p : "/opt/a11y/log/historico-falas.txt";

    internal static void Record(string text, string? path = null)
    {
        path ??= DefaultPath;
        string line = string.Join(' ', text.Split((char[]?)null, StringSplitOptions.RemoveEmptyEntries));
        string? dir = Path.GetDirectoryName(path);
        if (line.Length == 0 || dir == null || !Directory.Exists(dir)) return;
        try
        {
            long size;
            using (var f = new FileStream(path, FileMode.Append, FileAccess.Write, FileShare.ReadWrite | FileShare.Delete))
            {
                f.Write(Utf8.GetBytes(line + "\n"));
                size = f.Length;
            }
            if (size <= TrimBytes) return;
            var lines = File.ReadAllLines(path, Utf8).Select(l => l.Trim()).Where(l => l.Length > 0).ToArray();
            string tmp = $"{path}.{Environment.ProcessId}.tmp";
            File.WriteAllText(tmp, string.Concat(lines.Skip(Math.Max(0, lines.Length - MaxEntries)).Select(l => l + "\n")), Utf8);
            File.Move(tmp, path, true);
        }
        catch (Exception) { }
    }
}
