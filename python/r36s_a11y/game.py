"""Laço do jogo: entrada, fala, áudio, tempo e ciclo de vida."""
import json
import os
import select
import signal
import sys
import time

from .audio import Audio
from .buttons import DOWN, REPEAT
from .input import GamepadReader, InputState
from .speech import Speech

REPEAT_DELAY = 0.4
REPEAT_INTERVAL = 0.1


def _game_dir():
    env = os.environ.get("A11Y_GAME_DIR")
    if env:
        return env
    main = getattr(sys.modules.get("__main__"), "__file__", None)
    return os.path.dirname(os.path.abspath(main)) if main else os.getcwd()


def _manifest(game_dir):
    try:
        with open(os.path.join(game_dir, "game.json"), encoding="utf-8") as f:
            return json.load(f)
    except (OSError, ValueError):
        return {}


class Game:
    """Um jogo acessível do R36S.

    Uso:
        game = Game()

        @game.on_button
        def botao(b, estado, ms): ...

        game.run()
    """

    def __init__(self, fps=60, repeat=False, axes=False, deadzone=0.2):
        self.dir = _game_dir()
        self.manifest = _manifest(self.dir)
        self.id = self.manifest.get("id") or os.path.basename(self.dir)
        self.name = self.manifest.get("name") or self.id
        self.lang = os.environ.get("A11Y_LANG", "pt-BR")
        self.save_dir = os.environ.get("A11Y_SAVE_DIR") or os.path.join(
            os.path.expanduser("~"), ".local", "share", "r36s-a11y", self.id)
        os.makedirs(self.save_dir, exist_ok=True)

        self.fps = fps
        self.repeat = repeat
        self.axes = axes
        self.speech = Speech(client=self.id, language=self.lang)
        self.sound = Audio(self.dir)
        self.input = InputState(deadzone=deadzone)

        self._handlers = {"button": [], "axis": [], "update": [], "quit": [], "start": []}
        self._timers = []          # (quando, função)
        self._repeat_at = {}       # botão -> próximo repeat
        self._focus = []           # pilha de menus que recebem os botões antes do jogo
        self._running = False
        self.started_at = time.time()

    # --- registro de funções -------------------------------------------
    def on_button(self, fn):
        """fn(botão, estado, ms): estado é "down", "up" ou "repeat"."""
        self._handlers["button"].append(fn)
        return fn

    def on_axis(self, fn):
        """fn(eixo, valor, ms): só chamado com Game(axes=True)."""
        self._handlers["axis"].append(fn)
        return fn

    def on_update(self, fn):
        """fn(dt) chamado fps vezes por segundo."""
        self._handlers["update"].append(fn)
        return fn

    def on_start(self, fn):
        self._handlers["start"].append(fn)
        return fn

    def on_quit(self, fn):
        """fn() ao sair: hora de salvar."""
        self._handlers["quit"].append(fn)
        return fn

    # --- serviços ------------------------------------------------------
    def say(self, text, queue=False, on_done=None):
        self.speech.say(text, queue=queue, on_done=on_done)

    def hush(self):
        self.speech.hush()

    def after(self, seconds, fn):
        """Chama fn() daqui a alguns segundos."""
        self._timers.append((time.monotonic() + seconds, fn))

    def ms(self):
        """Milissegundos desde o início do jogo, na mesma escala dos botões."""
        return int((time.time() - self.started_at) * 1000)

    def quit(self):
        self._running = False

    def push_focus(self, handler):
        self._focus.append(handler)

    def pop_focus(self, handler):
        if handler in self._focus:
            self._focus.remove(handler)

    # --- laço ----------------------------------------------------------
    def _dispatch_button(self, button, state, ms):
        if self._focus:
            self._focus[-1].handle_button(button, state, ms)
            return
        for fn in list(self._handlers["button"]):
            fn(button, state, ms)

    def _dispatch(self, event):
        kind = event[0]
        if kind == "button":
            _, button, state, ms = event
            if self.repeat:
                if state == DOWN:
                    self._repeat_at[button] = time.monotonic() + REPEAT_DELAY
                else:
                    self._repeat_at.pop(button, None)
            self._dispatch_button(button, state, ms)
        elif kind == "axis" and self.axes:
            _, axis, value, ms = event
            for fn in list(self._handlers["axis"]):
                fn(axis, value, ms)

    def _tick_repeat(self, now):
        for button, when in list(self._repeat_at.items()):
            if now >= when:
                self._repeat_at[button] = when + REPEAT_INTERVAL
                self._dispatch_button(button, REPEAT, self.ms())

    def _tick_timers(self, now):
        due = [t for t in self._timers if t[0] <= now]
        if due:
            self._timers = [t for t in self._timers if t[0] > now]
            for _, fn in sorted(due, key=lambda t: t[0]):
                fn()

    def run(self):
        reader = GamepadReader(self.input, self.started_at) if sys.platform.startswith("linux") else None
        if reader is None or not reader.files:
            print("[r36s_a11y] nenhum controle encontrado em /dev/input", file=sys.stderr)

        def stop(signum, frame):
            self._running = False
        signal.signal(signal.SIGTERM, stop)
        signal.signal(signal.SIGINT, stop)

        self._running = True
        for fn in list(self._handlers["start"]):
            fn()

        step = 1.0 / self.fps
        last = time.monotonic()
        try:
            while self._running:
                fds = reader.fileno_list() if reader else []
                timeout = max(0.0, last + step - time.monotonic())
                try:
                    ready = select.select(fds, [], [], timeout)[0] if fds else []
                    if not fds:
                        time.sleep(timeout)
                except InterruptedError:
                    ready = []
                for fd in ready:
                    for event in reader.read(fd):
                        self._dispatch(event)
                for fn, done in self.speech.poll():
                    fn(done)
                now = time.monotonic()
                self._tick_repeat(now)
                self._tick_timers(now)
                if now - last >= step:
                    dt, last = now - last, now
                    self.sound.update(dt)
                    for fn in list(self._handlers["update"]):
                        fn(dt)
        finally:
            for fn in list(self._handlers["quit"]):
                try:
                    fn()
                except Exception as exc:  # salvar não pode impedir a saída
                    print("[r36s_a11y] erro ao sair: %s" % exc, file=sys.stderr)
            if reader:
                reader.close()
            self.speech.close()
            self.sound.close()
