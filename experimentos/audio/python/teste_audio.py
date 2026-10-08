#!/usr/bin/env python3
"""Teste de áudio posicional com SDL2_mixer via ctypes (Python 3.7, sem pip)."""
import ctypes
import sys
import time

sdl = ctypes.CDLL("libSDL2-2.0.so.0", mode=ctypes.RTLD_GLOBAL)
mix = ctypes.CDLL("libSDL2_mixer-2.0.so.0")
sdl.SDL_RWFromFile.restype = ctypes.c_void_p
mix.Mix_LoadWAV_RW.restype = ctypes.c_void_p
mix.Mix_LoadWAV_RW.argtypes = [ctypes.c_void_p, ctypes.c_int]
mix.Mix_PlayChannelTimed.argtypes = [ctypes.c_int, ctypes.c_void_p, ctypes.c_int, ctypes.c_int]
mix.Mix_SetPosition.argtypes = [ctypes.c_int, ctypes.c_int16, ctypes.c_uint8]

sdl.SDL_Init(0x10)
assert mix.Mix_OpenAudio(44100, 0x8010, 2, 512) == 0
chunk = mix.Mix_LoadWAV_RW(sdl.SDL_RWFromFile(sys.argv[1].encode(), b"rb"), 1)
assert chunk
for angle in (270, 0, 90):  # esquerda, frente, direita
    ch = mix.Mix_PlayChannelTimed(-1, chunk, 0, -1)
    mix.Mix_SetPosition(ch, angle, 0)
    time.sleep(0.7)
print("ok")
