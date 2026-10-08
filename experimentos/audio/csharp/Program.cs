using System.Runtime.InteropServices;
static class Mix {
    [DllImport("libSDL2-2.0.so.0")] public static extern int SDL_Init(uint f);
    [DllImport("libSDL2-2.0.so.0")] public static extern IntPtr SDL_RWFromFile(string f, string m);
    [DllImport("libSDL2_mixer-2.0.so.0")] public static extern int Mix_OpenAudio(int r, ushort f, int c, int s);
    [DllImport("libSDL2_mixer-2.0.so.0")] public static extern IntPtr Mix_LoadWAV_RW(IntPtr rw, int free);
    [DllImport("libSDL2_mixer-2.0.so.0")] public static extern int Mix_PlayChannelTimed(int ch, IntPtr c, int l, int t);
    [DllImport("libSDL2_mixer-2.0.so.0")] public static extern int Mix_SetPosition(int ch, short a, byte d);
}
static class P { static void Main(string[] a) {
    Mix.SDL_Init(0x10); Mix.Mix_OpenAudio(44100, 0x8010, 2, 512);
    var c = Mix.Mix_LoadWAV_RW(Mix.SDL_RWFromFile(a[0], "rb"), 1);
    foreach (short ang in new short[]{270,0,90}) { var ch = Mix.Mix_PlayChannelTimed(-1, c, 0, -1); Mix.Mix_SetPosition(ch, ang, 0); Thread.Sleep(700); }
    Console.WriteLine("ok");
}}
