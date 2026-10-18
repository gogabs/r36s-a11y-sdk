#!/usr/bin/env python3
"""Cria, roda e exporta jogos feitos com o r36s_a11y.

    python a11y.py new C:\\jogos\\meu-jogo --name "Meu Jogo"
    python a11y.py run C:\\jogos\\meu-jogo
    python a11y.py export C:\\jogos\\meu-jogo              # R36S e Windows
    python a11y.py export C:\\jogos\\meu-jogo --to r36s

Veja "python a11y.py --help" e o README.
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from criador.cli import main  # noqa: E402

if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
