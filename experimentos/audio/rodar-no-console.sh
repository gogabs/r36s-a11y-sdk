#!/bin/sh
# Copie a pasta experimentos/audio para o console e rode: sh rodar-no-console.sh
# Cada teste toca o bip à esquerda, à frente e à direita.
cd "$(dirname "$0")"
W="$PWD/bip.wav"
echo "== Go";     chmod +x bin/goaudio && bin/goaudio "$W"
echo "== Rust";   chmod +x bin/rsaudio && bin/rsaudio "$W"
echo "== C#";     chmod +x bin/csaudio/csaudio && bin/csaudio/csaudio "$W"
echo "== Python"; python3 python/teste_audio.py "$W"
echo "== LÖVE";   LD_LIBRARY_PATH=/opt/love2d/lib /opt/love2d/love love
