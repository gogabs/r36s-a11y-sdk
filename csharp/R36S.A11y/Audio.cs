using System.Buffers.Binary;
using System.Runtime.InteropServices;

namespace R36S.A11y;

/// <summary>
/// Áudio 3D com a OpenAL Soft do sistema (P/Invoke, carregada em tempo de
/// execução). Coordenadas em metros: x positivo à direita, y para cima, z
/// negativo à frente. Só sons mono são posicionados.
/// </summary>
public sealed class Audio
{
    private const int MaxSources = 32;

    // Antes de ReverbPresets: campos estáticos inicializam na ordem em que aparecem.
    private static readonly string[] ReverbNames =
    {
        "density", "diffusion", "gain", "gain_hf", "decay", "decay_hf_ratio",
        "reflections_gain", "reflections_delay", "late_gain", "late_delay",
    };

    /// <summary>Ambientes prontos (baseados nos presets EFX da Creative).</summary>
    public static readonly IReadOnlyDictionary<string, IReadOnlyDictionary<string, double>> ReverbPresets =
        new Dictionary<string, IReadOnlyDictionary<string, double>>
        {
            ["quarto"] = P(0.43, 1, 0.32, 0.6, 0.4, 0.83, 0.15, 0.002, 1.06, 0.003),
            ["corredor"] = P(1, 1, 0.32, 0.71, 1.49, 0.59, 0.25, 0.007, 1.66, 0.011),
            ["estacionamento"] = P(1, 1, 0.32, 1, 1.65, 1.5, 0.21, 0.008, 0.27, 0.012),
            ["cozinha"] = P(1, 0.9, 0.32, 0.9, 1, 1.2, 0.5, 0.004, 1.2, 0.006),
            ["metro"] = P(1, 1, 0.32, 0.71, 3.1, 1, 0.15, 0.03, 1.4, 0.03),
            ["igreja"] = P(1, 1, 0.32, 0.6, 5.5, 0.6, 0.2, 0.04, 1.3, 0.05),
            ["floresta"] = P(1, 0.3, 0.32, 0.02, 1.49, 0.54, 0.05, 0.16, 0.2, 0.09),
        };

    private static IReadOnlyDictionary<string, double> P(params double[] v) =>
        ReverbNames.Select((n, i) => (n, v[i])).ToDictionary(x => x.n, x => x.Item2);

    private readonly string baseDir;
    private readonly IntPtr device, context;
    private uint slot, effect;
    private readonly Dictionary<string, Sound> sounds = new();
    private readonly List<Voice> voices = new(); // da mais antiga para a mais nova
    private readonly Stack<uint> free = new();

    /// <summary>A OpenAL abriu.</summary>
    public bool Available { get; private set; }

    /// <summary>HRTF ligado (som binaural no fone).</summary>
    public bool Hrtf { get; }

    internal Audio(string baseDir)
    {
        this.baseDir = baseDir;
        // sem OpenAL (DllNotFoundException), o catch abaixo deixa o áudio desligado
        try
        {
            device = Al.alcOpenDevice(IntPtr.Zero); // dispositivo padrão = dmix do ALSA
            if (device == IntPtr.Zero) return;
            int[]? attrs = Al.alcIsExtensionPresent(device, "ALC_SOFT_HRTF") != 0
                ? new[] { Al.ALC_HRTF_SOFT, 1, 0 } : null;
            context = Al.alcCreateContext(device, attrs);
            if (context == IntPtr.Zero) return;
            Al.alcMakeContextCurrent(context);
            if (attrs != null)
            {
                Al.alcGetIntegerv(device, Al.ALC_HRTF_SOFT, 1, out int status);
                Hrtf = status != 0;
            }
            Al.alDistanceModel(Al.AL_INVERSE_DISTANCE_CLAMPED);
            Available = true;
            Listener((0, 0, 0), 0);
        }
        catch (Exception)
        {
            Available = false;
            return;
        }
        try
        {
            if (Al.alcIsExtensionPresent(device, "ALC_EXT_EFX") != 0)
            {
                Al.alGenAuxiliaryEffectSlots(1, out slot);
                Al.alGenEffects(1, out effect);
                Al.alEffecti(effect, Al.AL_EFFECT_TYPE, Al.AL_EFFECT_REVERB);
            }
        }
        catch (EntryPointNotFoundException)
        {
            slot = 0; // sem EFX: tudo funciona, só sem reverberação
        }
    }

    /// <summary>Carrega um WAV (PCM 8/16 bits) ou OGG Vorbis, relativo à pasta do jogo.</summary>
    public Sound Load(string name, string path)
    {
        if (!Path.IsPathRooted(path)) path = Path.Combine(baseDir, path);
        var sound = new Sound(this, name);
        sounds[name] = sound;
        if (!Available) return sound;
        var (channels, width, rate, data) = path.EndsWith(".ogg", StringComparison.OrdinalIgnoreCase)
            ? DecodeOgg(path) : DecodeWav(path);
        int format = (channels, width) switch
        {
            (1, 1) => Al.AL_FORMAT_MONO8, (1, 2) => Al.AL_FORMAT_MONO16,
            (2, 1) => Al.AL_FORMAT_STEREO8, (2, 2) => Al.AL_FORMAT_STEREO16,
            _ => throw new InvalidDataException($"{path}: só mono ou estéreo, 8 ou 16 bits"),
        };
        Al.alGenBuffers(1, out uint buffer);
        Al.alBufferData(buffer, format, data, data.Length, rate);
        sound.Buffer = buffer;
        sound.Channels = channels;
        return sound;
    }

    /// <summary>Som já carregado, pelo nome.</summary>
    public Sound? this[string name] => sounds.GetValueOrDefault(name);

    internal Voice Play(Sound sound, bool loop, double volume, double pitch,
                        (double X, double Y, double Z)? pos, double? pan, bool? reverb)
    {
        if (!Available || sound.Buffer == 0) return new Voice(this, 0, false);
        uint src = Source();
        Al.alSourceStop(src);
        Al.alSourcei(src, Al.AL_BUFFER, (int)sound.Buffer);
        Al.alSourcei(src, Al.AL_LOOPING, loop ? 1 : 0);
        Al.alSourcef(src, Al.AL_PITCH, (float)pitch);
        Al.alSourcef(src, Al.AL_REFERENCE_DISTANCE, 1);
        Al.alSourcef(src, Al.AL_ROLLOFF_FACTOR, 1);
        Al.alSourcef(src, Al.AL_MAX_DISTANCE, 100);
        Place(src, pos, pan);
        if (slot != 0)
            Al.alSource3i(src, Al.AL_AUXILIARY_SEND_FILTER, (reverb ?? pos != null) ? (int)slot : 0, 0, 0);
        var voice = new Voice(this, src, true);
        voice.Gain(volume);
        voices.Add(voice);
        Al.alSourcePlay(src);
        return voice;
    }

    private uint Source()
    {
        if (free.Count > 0) return free.Pop();
        if (voices.Count < MaxSources)
        {
            Al.alGenSources(1, out uint src);
            return src;
        }
        // sem fonte livre: reaproveita a primeira que terminou, senão a mais antiga
        var victim = voices.FirstOrDefault(v => !v.Playing) ?? voices[0];
        Al.alSourceStop(victim.SourceId);
        voices.Remove(victim);
        victim.Active = false;
        return victim.SourceId;
    }

    internal static void Place(uint src, (double X, double Y, double Z)? pos, double? pan)
    {
        if (pos is { } p)
        {
            Al.alSourcei(src, Al.AL_SOURCE_RELATIVE, 0);
            Al.alSource3f(src, Al.AL_POSITION, (float)p.X, (float)p.Y, (float)p.Z);
        }
        else if (pan is { } pn)
        {
            pn = Math.Clamp(pn, -1, 1);
            Al.alSourcei(src, Al.AL_SOURCE_RELATIVE, 1);
            Al.alSource3f(src, Al.AL_POSITION, (float)pn, 0, (float)-Math.Sqrt(1 - pn * pn));
        }
        else
        {
            Al.alSourcei(src, Al.AL_SOURCE_RELATIVE, 1);
            Al.alSource3f(src, Al.AL_POSITION, 0, 0, 0);
        }
    }

    /// <summary>Posição do ouvinte e para onde ele olha (graus; 0 = frente, 90 = direita).</summary>
    public void Listener((double X, double Y, double Z) pos, double facing)
    {
        if (!Available) return;
        Al.alListener3f(Al.AL_POSITION, (float)pos.X, (float)pos.Y, (float)pos.Z);
        double r = facing * Math.PI / 180;
        Al.alListenerfv(Al.AL_ORIENTATION, new[] { (float)Math.Sin(r), 0f, (float)-Math.Cos(r), 0f, 1f, 0f });
    }

    /// <summary>Muda o ambiente para um preset de ReverbPresets, ou null para desligar.</summary>
    public void SetReverb(string? preset) =>
        SetReverb(preset == null ? null : ReverbPresets[preset]);

    /// <summary>Aplica parâmetros soltos de reverberação; null desliga.</summary>
    public void SetReverb(IReadOnlyDictionary<string, double>? parameters)
    {
        if (!Available || slot == 0) return;
        if (parameters == null)
        {
            Al.alAuxiliaryEffectSloti(slot, Al.AL_EFFECTSLOT_EFFECT, 0);
            return;
        }
        foreach (var (name, value) in parameters)
        {
            int index = Array.IndexOf(ReverbNames, name);
            if (index >= 0) Al.alEffectf(effect, index + 1, (float)value);
        }
        // recarregar o efeito no slot aplica os parâmetros novos
        Al.alAuxiliaryEffectSloti(slot, Al.AL_EFFECTSLOT_EFFECT, (int)effect);
    }

    /// <summary>Pausa todos os sons (por exemplo, no menu de pausa).</summary>
    public void PauseAll() => voices.ForEach(v => v.Pause());

    /// <summary>Retoma os sons pausados.</summary>
    public void ResumeAll() => voices.ForEach(v => v.Resume());

    internal void Update(double dt)
    {
        if (!Available) return;
        foreach (var v in voices.ToArray())
        {
            v.Tick(dt);
            if (!v.Fading && !v.Playing)
            {
                voices.Remove(v);
                v.Active = false;
                free.Push(v.SourceId);
            }
        }
    }

    internal void Close()
    {
        if (!Available) return;
        Al.alcMakeContextCurrent(IntPtr.Zero);
        Al.alcDestroyContext(context);
        Al.alcCloseDevice(device);
        Available = false;
    }

    /// <summary>Lê um WAV PCM 8 ou 16 bits.</summary>
    internal static (int Channels, int Width, int Rate, byte[] Data) DecodeWav(string path)
    {
        var raw = File.ReadAllBytes(path);
        if (raw.Length < 12 || raw[0] != 'R' || raw[8] != 'W') throw new InvalidDataException($"{path}: não é WAV");
        int channels = 0, width = 0, rate = 0;
        byte[]? data = null;
        for (int i = 12; i + 8 <= raw.Length;)
        {
            string id = System.Text.Encoding.ASCII.GetString(raw, i, 4);
            int size = BinaryPrimitives.ReadInt32LittleEndian(raw.AsSpan(i + 4));
            var body = raw.AsSpan(i + 8, Math.Min(size, raw.Length - i - 8));
            if (id == "fmt ")
            {
                if (BinaryPrimitives.ReadUInt16LittleEndian(body) != 1)
                    throw new InvalidDataException($"{path}: WAV precisa ser PCM");
                channels = BinaryPrimitives.ReadUInt16LittleEndian(body[2..]);
                rate = BinaryPrimitives.ReadInt32LittleEndian(body[4..]);
                width = BinaryPrimitives.ReadUInt16LittleEndian(body[14..]) / 8;
            }
            else if (id == "data")
            {
                data = body.ToArray();
            }
            i += 8 + size + size % 2;
        }
        if (data == null || channels == 0) throw new InvalidDataException($"{path}: WAV incompleto");
        return (channels, width, rate, data);
    }

    /// <summary>Decodifica um OGG Vorbis inteiro pela libvorbisfile do sistema.</summary>
    private static (int Channels, int Width, int Rate, byte[] Data) DecodeOgg(string path)
    {
        IntPtr vf = Marshal.AllocHGlobal(8192); // maior que OggVorbis_File em qualquer arquitetura
        try
        {
            if (Vorbis.ov_fopen(path, vf) != 0) throw new InvalidDataException($"{path}: não é um OGG Vorbis válido");
            try
            {
                IntPtr info = Vorbis.ov_info(vf, -1); // int version, int channels, long rate
                int channels = Marshal.ReadInt32(info, 4);
                int rate = (int)Marshal.ReadInt64(info, 8);
                using var output = new MemoryStream();
                var chunk = new byte[65536];
                while (true)
                {
                    long n = Vorbis.ov_read(vf, chunk, chunk.Length, 0, 2, 1, out _);
                    if (n <= 0) break;
                    output.Write(chunk, 0, (int)n);
                }
                return (channels, 2, rate, output.ToArray());
            }
            finally
            {
                Vorbis.ov_clear(vf);
            }
        }
        finally
        {
            Marshal.FreeHGlobal(vf);
        }
    }
}

/// <summary>Um som carregado. Toque com Play.</summary>
public sealed class Sound
{
    private readonly Audio audio;
    public string Name { get; }
    public int Channels { get; internal set; } = 1;
    internal uint Buffer;

    internal Sound(Audio audio, string name)
    {
        this.audio = audio;
        Name = name;
    }

    /// <summary>
    /// Toca o som. pos posiciona em 3D; pan de -1 (esquerda) a 1 (direita) é um
    /// atalho para posição a 1 m do ouvinte. reverb: padrão é só com pos.
    /// </summary>
    public Voice Play(bool loop = false, double volume = 1, double pitch = 1,
                      (double X, double Y, double Z)? pos = null, double? pan = null, bool? reverb = null) =>
        audio.Play(this, loop, volume, pitch, pos, pan, reverb);
}

/// <summary>Um som tocando.</summary>
public sealed class Voice
{
    private readonly Audio audio;
    internal readonly uint SourceId;
    internal bool Active;
    private double volume = 1;
    private (double From, double To, double Total, double Elapsed, bool Stop)? fade;

    internal Voice(Audio audio, uint source, bool active)
    {
        this.audio = audio;
        SourceId = source;
        Active = active;
    }

    internal bool Fading => fade != null;

    /// <summary>Ainda tocando (ou pausado).</summary>
    public bool Playing
    {
        get
        {
            if (!Active) return false;
            Al.alGetSourcei(SourceId, Al.AL_SOURCE_STATE, out int state);
            return state is Al.AL_PLAYING or Al.AL_PAUSED;
        }
    }

    internal void Gain(double v)
    {
        volume = v;
        Al.alSourcef(SourceId, Al.AL_GAIN, (float)v);
    }

    /// <summary>Muda o volume; com fade > 0 (segundos), em transição.</summary>
    public void SetVolume(double v, double fadeSeconds = 0)
    {
        if (!Active) return;
        if (fadeSeconds > 0) fade = (volume, v, fadeSeconds, 0, false);
        else
        {
            fade = null;
            Gain(v);
        }
    }

    public void SetPitch(double p)
    {
        if (Active) Al.alSourcef(SourceId, Al.AL_PITCH, (float)p);
    }

    public void SetPosition(double x, double y, double z)
    {
        if (Active) Audio.Place(SourceId, (x, y, z), null);
    }

    public void SetPan(double pan)
    {
        if (Active) Audio.Place(SourceId, null, pan);
    }

    public void Pause()
    {
        if (Active) Al.alSourcePause(SourceId);
    }

    /// <summary>Retoma um som pausado.</summary>
    public void Resume()
    {
        if (!Active) return;
        Al.alGetSourcei(SourceId, Al.AL_SOURCE_STATE, out int state);
        if (state == Al.AL_PAUSED) Al.alSourcePlay(SourceId);
    }

    /// <summary>Para o som; com fade > 0 (segundos), abaixando aos poucos.</summary>
    public void Stop(double fadeSeconds = 0)
    {
        if (!Active) return;
        if (fadeSeconds > 0) fade = (volume, 0, fadeSeconds, 0, true);
        else Al.alSourceStop(SourceId);
    }

    internal void Tick(double dt)
    {
        if (fade is not { } f) return;
        double elapsed = f.Elapsed + dt, t = Math.Min(1, elapsed / f.Total);
        Gain(f.From + (f.To - f.From) * t);
        if (t < 1)
        {
            fade = f with { Elapsed = elapsed };
            return;
        }
        fade = null;
        if (f.Stop) Al.alSourceStop(SourceId);
    }
}

internal static class Al
{
    internal const string Lib = "libopenal.so.1";
    internal const int AL_SOURCE_RELATIVE = 0x202, AL_PITCH = 0x1003, AL_POSITION = 0x1004,
        AL_LOOPING = 0x1007, AL_BUFFER = 0x1009, AL_GAIN = 0x100A, AL_ORIENTATION = 0x100F,
        AL_SOURCE_STATE = 0x1010, AL_PLAYING = 0x1012, AL_PAUSED = 0x1013,
        AL_REFERENCE_DISTANCE = 0x1020, AL_ROLLOFF_FACTOR = 0x1021, AL_MAX_DISTANCE = 0x1023,
        AL_FORMAT_MONO8 = 0x1100, AL_FORMAT_MONO16 = 0x1101, AL_FORMAT_STEREO8 = 0x1102,
        AL_FORMAT_STEREO16 = 0x1103, AL_INVERSE_DISTANCE_CLAMPED = 0xD002, ALC_HRTF_SOFT = 0x1992,
        AL_EFFECT_TYPE = 0x8001, AL_EFFECT_REVERB = 0x0001, AL_EFFECTSLOT_EFFECT = 0x0001,
        AL_AUXILIARY_SEND_FILTER = 0x20006;

    [DllImport(Lib)] internal static extern IntPtr alcOpenDevice(IntPtr name);
    [DllImport(Lib)] internal static extern IntPtr alcCreateContext(IntPtr device, int[]? attrs);
    [DllImport(Lib)] internal static extern byte alcMakeContextCurrent(IntPtr context);
    [DllImport(Lib)] internal static extern byte alcIsExtensionPresent(IntPtr device, string name);
    [DllImport(Lib)] internal static extern void alcGetIntegerv(IntPtr device, int param, int size, out int data);
    [DllImport(Lib)] internal static extern void alcDestroyContext(IntPtr context);
    [DllImport(Lib)] internal static extern byte alcCloseDevice(IntPtr device);
    [DllImport(Lib)] internal static extern void alDistanceModel(int model);
    [DllImport(Lib)] internal static extern void alGenBuffers(int n, out uint buffer);
    [DllImport(Lib)] internal static extern void alBufferData(uint buffer, int format, byte[] data, int size, int freq);
    [DllImport(Lib)] internal static extern void alGenSources(int n, out uint source);
    [DllImport(Lib)] internal static extern void alSourcei(uint source, int param, int value);
    [DllImport(Lib)] internal static extern void alSourcef(uint source, int param, float value);
    [DllImport(Lib)] internal static extern void alSource3f(uint source, int param, float x, float y, float z);
    [DllImport(Lib)] internal static extern void alSource3i(uint source, int param, int a, int b, int c);
    [DllImport(Lib)] internal static extern void alGetSourcei(uint source, int param, out int value);
    [DllImport(Lib)] internal static extern void alSourcePlay(uint source);
    [DllImport(Lib)] internal static extern void alSourceStop(uint source);
    [DllImport(Lib)] internal static extern void alSourcePause(uint source);
    [DllImport(Lib)] internal static extern void alListener3f(int param, float x, float y, float z);
    [DllImport(Lib)] internal static extern void alListenerfv(int param, float[] values);
    [DllImport(Lib)] internal static extern void alGenAuxiliaryEffectSlots(int n, out uint slot);
    [DllImport(Lib)] internal static extern void alGenEffects(int n, out uint effect);
    [DllImport(Lib)] internal static extern void alEffecti(uint effect, int param, int value);
    [DllImport(Lib)] internal static extern void alEffectf(uint effect, int param, float value);
    [DllImport(Lib)] internal static extern void alAuxiliaryEffectSloti(uint slot, int param, int value);
}

internal static class Vorbis
{
    private const string Lib = "libvorbisfile.so.3";

    [DllImport(Lib)] internal static extern int ov_fopen(string path, IntPtr vf);
    [DllImport(Lib)] internal static extern IntPtr ov_info(IntPtr vf, int link);
    [DllImport(Lib)] internal static extern long ov_read(IntPtr vf, byte[] buffer, int length, int bigEndian,
                                                          int word, int signed, out int bitstream);
    [DllImport(Lib)] internal static extern int ov_clear(IntPtr vf);
}
