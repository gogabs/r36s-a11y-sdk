#!/usr/bin/env python3
"""Onde está o bip? Jogo de exemplo da biblioteca r36s_a11y.

Um bip toca à esquerda, à frente, à direita ou atrás de você. Aperte o
direcional para o lado de onde ele veio. Com fone, a OpenAL usa HRTF e dá
para distinguir frente e trás.
"""
import json
import os
import random
import sys
import time

# Permite rodar direto da pasta do repositório, sem instalar a biblioteca.
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", ".."))

import r36s_a11y as a11y

ROUNDS = 10
DIRECTIONS = {
    "left": ("à esquerda", (-2.0, 0.0, 0.0)),
    "up": ("à frente", (0.0, 0.0, -2.0)),
    "right": ("à direita", (2.0, 0.0, 0.0)),
    "down": ("atrás", (0.0, 0.0, 2.0)),
}

game = a11y.Game()
bip = game.sound.load("bip", "sounds/bip.wav")
acerto = game.sound.load("acerto", "sounds/acerto.wav")
erro = game.sound.load("erro", "sounds/erro.wav")
menu_sounds = {"move": game.sound.load("move", "sounds/move.wav")}

RECORD_FILE = os.path.join(game.save_dir, "recorde.json")


def load_record():
    try:
        with open(RECORD_FILE) as f:
            return json.load(f).get("recorde", 0)
    except (OSError, ValueError):
        return 0


def save_record(value):
    with open(RECORD_FILE, "w") as f:
        json.dump({"recorde": value}, f)


class Round:
    def __init__(self):
        self.number = 0
        self.score = 0
        self.answer = None      # direção certa da rodada atual
        self.waiting = False    # aceitando resposta
        self.feedback = None    # (direção certa, acertou, quando), para o desenho

    def start(self):
        self.number, self.score = 0, 0
        game.say("Valendo! Ouça o bip e aperte o direcional para o lado dele.",
                 on_done=lambda done: self.next())

    def next(self):
        if self.number >= ROUNDS:
            self.finish()
            return
        self.number += 1
        self.answer = random.choice(list(DIRECTIONS))
        self.waiting = False
        game.after(0.6, self.play_bip)

    def play_bip(self):
        bip.play(pos=DIRECTIONS[self.answer][1])
        self.waiting = True

    def respond(self, button):
        if not self.waiting:
            return
        if button == "select":
            self.play_bip()
            return
        if button not in DIRECTIONS:
            return
        self.waiting = False
        self.feedback = (self.answer, button == self.answer, time.monotonic())
        if button == self.answer:
            self.score += 1
            acerto.play()
            game.after(0.5, self.next)
        else:
            erro.play()
            game.say("Era %s." % DIRECTIONS[self.answer][0], queue=True,
                     on_done=lambda done: self.next())

    def finish(self):
        record = load_record()
        text = "Fim. Você acertou %d de %d." % (self.score, ROUNDS)
        if self.score > record:
            save_record(self.score)
            text += " Novo recorde!"
        elif record:
            text += " Recorde: %d." % record
        self.answer = None
        game.say(text, on_done=lambda done: main_menu.open(announce_help=False))


rodada = Round()


def on_main(index, item):
    if item == "Jogar":
        main_menu.close()
        rodada.start()
    elif item == "Instruções":
        game.say("Um bip toca à esquerda, à frente, à direita ou atrás de você. "
                 "Aperte o direcional para o lado de onde ele veio. "
                 "Select repete o bip. Start pausa. Use fone de ouvido.")
    else:
        game.quit()


main_menu = a11y.Menu(game, "Onde está o bip", ["Jogar", "Instruções", "Sair"],
                      on_select=on_main, on_back=game.quit, sounds=menu_sounds)


def on_pause(index, item):
    pause_menu.close()
    if item == "Continuar":
        game.say("Continuando.", on_done=lambda done: rodada.play_bip())
    else:
        rodada.answer = None
        main_menu.open(announce_help=False)


pause_menu = a11y.Menu(game, "Pausa", ["Continuar", "Sair para o menu"],
                       on_select=on_pause, on_back=lambda: on_pause(0, "Continuar"),
                       sounds=menu_sounds)


@game.on_button
def botao(button, state, ms):
    if state != a11y.DOWN or rodada.answer is None:
        return
    if button == "start":
        rodada.waiting = False
        pause_menu.open()
        return
    rodada.respond(button)


# --- gráficos (opcionais: o jogo é todo jogável pelo som) -----------------
ARROWS = {   # direção -> pontos do triângulo, ao redor do centro (320, 260)
    "up": [(320, 120), (290, 160), (350, 160)],
    "down": [(320, 400), (290, 360), (350, 360)],
    "left": [(180, 260), (220, 230), (220, 290)],
    "right": [(460, 260), (420, 230), (420, 290)],
}
ouvido = game.gfx.image("images/ouvido.png")
titulo = game.gfx.font(size=32)


@game.on_draw
def desenhar(gfx):
    gfx.clear("#101830")
    gfx.text("Onde está o bip?", 320, 20, font=titulo, color="#ffd23c", align="center")
    if rodada.answer is not None:
        gfx.text("Rodada %d de %d   Acertos: %d" % (rodada.number, ROUNDS, rodada.score),
                 320, 70, color=(200, 200, 220), align="center")
    fb = rodada.feedback
    recent = fb is not None and time.monotonic() - fb[2] < 0.8
    for direction, points in ARROWS.items():
        color = (70, 80, 110)
        if recent and direction == fb[0]:
            color = (60, 200, 90) if fb[1] else (220, 70, 60)
        gfx.polygon(points, color=color)
    gfx.circle(320, 260, 60, color=(40, 50, 80))
    gfx.circle(320, 260, 60, color=(120, 130, 170), fill=False)
    gfx.draw(ouvido, 320 - 32, 260 - 32, scale=2)


@game.on_start
def start():
    main_menu.open()


game.run()
