"""Biblioteca-ponte Python da plataforma de jogos acessíveis do R36S.

Requer Python 3.7+ e nada além da biblioteca padrão. No console usa o
controle em /dev/input, o Speech Dispatcher e a OpenAL Soft do sistema.
"""
from .audio import Audio, Sound, Voice
from .buttons import BUTTONS, DOWN, REPEAT, UP, is_back, is_confirm
from .game import Game
from .menu import Menu
from .speech import Speech

__version__ = "0.1.0"
CONTRACT = 1

__all__ = [
    "Audio", "BUTTONS", "CONTRACT", "DOWN", "Game", "Menu", "REPEAT", "Sound",
    "Speech", "UP", "Voice", "is_back", "is_confirm",
]
