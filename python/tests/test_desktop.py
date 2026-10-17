import ctypes
import os
import struct
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from r36s_a11y import desktop
from r36s_a11y.desktop import DesktopInput, Pad, key_name, parse_event
from r36s_a11y.graphics import _triangulate
from r36s_a11y.input import InputState
from r36s_a11y.game import wrap

SPACE, RETURN, LCTRL, F11 = 32, 13, (1 << 30) | 224, (1 << 30) | 68


def key_event(down, sym, scancode=0, repeat=0, mod=0):
    buf = bytearray(64)
    struct.pack_into("<IIIBB", buf, 0, desktop.SDL_KEYDOWN if down else desktop.SDL_KEYUP,
                     0, 1, 1 if down else 0, repeat)
    struct.pack_into("<iiH", buf, 16, scancode, sym, mod)
    return bytes(buf)


def pad_button(down, which, button):
    buf = bytearray(64)
    etype = desktop.SDL_CONTROLLERBUTTONDOWN if down else desktop.SDL_CONTROLLERBUTTONUP
    struct.pack_into("<IIiBB", buf, 0, etype, 0, which, button, 1 if down else 0)
    return bytes(buf)


class FakeGfx:
    def __init__(self):
        self.fullscreen_toggles = 0

    def toggle_fullscreen(self):
        self.fullscreen_toggles += 1


def make_input():
    d = DesktopInput.__new__(DesktopInput)
    d.gfx, d.state, d.clock = FakeGfx(), InputState(), lambda: 0
    for code in desktop.AXIS_STICKS.values():
        d.state.set_axis_range(code, -32768, 32767)
    d.pads, d.last_pad, d.keys_down, d.mouse_down, d.first_poll = {}, None, {}, set(), False
    return d


def feed(d, raw):
    out = []
    for buf in raw:
        out.extend(d._handle(parse_event(buf), 0))
    return out


def only(events, kind):
    return [e for e in events if e[0] == kind]


class ParseTest(unittest.TestCase):
    def test_key_names(self):
        self.assertEqual(key_name(SPACE), "space")
        self.assertEqual(key_name(RETURN), "return")
        self.assertEqual(key_name(ord("z")), "z")
        self.assertEqual(key_name(0xE7), "ç")
        self.assertEqual(key_name(F11), "f11")
        self.assertEqual(key_name((1 << 30) | 82), "up")
        self.assertEqual(key_name((1 << 30) | 89), "kp1")

    def test_parse_key_text_mouse_pad(self):
        self.assertEqual(parse_event(key_event(True, SPACE, 44)), ("key", True, False, 44, SPACE, 0))
        buf = bytearray(64)
        struct.pack_into("<III", buf, 0, desktop.SDL_TEXTINPUT, 0, 1)
        buf[12:15] = "ção".encode("utf-8")[:3]
        self.assertEqual(parse_event(bytes(buf))[0], "text")
        buf = bytearray(64)
        struct.pack_into("<IIIIBBBBii", buf, 0, desktop.SDL_MOUSEBUTTONDOWN, 0, 1, 0, 3, 1, 1, 0, 100, 200)
        self.assertEqual(parse_event(bytes(buf)), ("mousebutton", True, 3, 100, 200))
        buf = bytearray(64)
        struct.pack_into("<IIiB3xh", buf, 0, desktop.SDL_CONTROLLERAXISMOTION, 0, 7, 4, 30000)
        self.assertEqual(parse_event(bytes(buf)), ("padaxis", 7, 4, 30000))
        self.assertEqual(parse_event(pad_button(True, 7, 0)), ("padbutton", True, 7, 0))


class DesktopInputTest(unittest.TestCase):
    def test_space_is_a_control_and_enter_is_also_button_b(self):
        d = make_input()
        events = feed(d, [key_event(True, SPACE), key_event(False, SPACE)])
        self.assertEqual(only(events, "control"), [("control", "key", "space", "down", 0),
                                                   ("control", "key", "space", "up", 0)])
        self.assertEqual(only(events, "button"), [])
        events = feed(d, [key_event(True, RETURN)])
        self.assertEqual(only(events, "button"), [("button", "b", "down", 0)])
        self.assertEqual(only(events, "control"), [("control", "key", "return", "down", 0)])

    def test_ctrl_is_fn_and_ctrl_c_quits(self):
        d = make_input()
        events = feed(d, [key_event(True, LCTRL), key_event(True, SPACE), key_event(True, RETURN),
                          key_event(False, RETURN), key_event(False, SPACE), key_event(False, LCTRL)])
        self.assertEqual(only(events, "control") + only(events, "button"), [])
        self.assertEqual(len(only(events, "key")), 6)   # on_key vê tudo
        events = feed(d, [key_event(True, LCTRL), key_event(True, ord("c"))])
        self.assertIn(("quit",), events)

    def test_os_repeat_only_reaches_on_key(self):
        d = make_input()
        feed(d, [key_event(True, SPACE)])
        events = feed(d, [key_event(True, SPACE, repeat=1)])
        self.assertEqual(events, [("key", "space", "repeat", 0)])

    def test_f11_toggles_fullscreen(self):
        d = make_input()
        feed(d, [key_event(True, F11)])
        feed(d, [key_event(True, RETURN, mod=0x0100)])  # Alt + Enter
        self.assertEqual(d.gfx.fullscreen_toggles, 2)

    def test_pad_by_position_and_guide_is_fn(self):
        d = make_input()
        d.pads[7] = Pad(None, "Controle", "playstation")
        events = feed(d, [pad_button(True, 7, 0)])       # o de baixo (xis)
        self.assertEqual(only(events, "button"), [("button", "b", "down", 0)])
        self.assertEqual(only(events, "control"), [("control", "pad", "a", "down", 0)])
        self.assertEqual(only(events, "pad"), [("pad", "playstation")])
        events = feed(d, [pad_button(False, 7, 0), pad_button(True, 7, 5), pad_button(True, 7, 1)])
        self.assertEqual(only(events, "control"), [("control", "pad", "a", "up", 0)])
        self.assertEqual(only(events, "button"), [("button", "b", "up", 0)])

    def test_trigger_with_hysteresis(self):
        d = make_input()
        d.pads[1] = Pad(None, "Xbox", "xbox")
        out = d._pad_axis(1, 4, 20000, 0) + d._pad_axis(1, 4, 9000, 0)
        self.assertEqual(only(out, "control"), [("control", "pad", "lefttrigger", "down", 0)])
        self.assertEqual(only(out, "button"), [("button", "l2", "down", 0)])
        out = d._pad_axis(1, 4, 1000, 0)
        self.assertEqual(only(out, "control"), [("control", "pad", "lefttrigger", "up", 0)])

    def test_focus_lost_releases_everything(self):
        d = make_input()
        d.pads[1] = Pad(None, "Xbox", "xbox")
        feed(d, [key_event(True, SPACE), key_event(True, RETURN), pad_button(True, 1, 3)])
        events = d._handle(("focus", False), 0)
        self.assertEqual(sorted(e[2] for e in only(events, "control")), ["return", "space", "y"])
        self.assertEqual(sorted(e[1] for e in only(events, "button")), ["b", "x"])
        self.assertEqual(events[-1], ("focus", False))


class HelpersTest(unittest.TestCase):
    def test_triangulate_concave(self):
        l_shape = [(0, 0), (2, 0), (2, 1), (1, 1), (1, 2), (0, 2)]
        tris = _triangulate(l_shape)
        self.assertEqual(len(tris), 4)
        area = sum(abs((b[0] - a[0]) * (c[1] - a[1]) - (c[0] - a[0]) * (b[1] - a[1])) / 2
                   for a, b, c in tris)
        self.assertAlmostEqual(area, 3.0)

    def test_wrap(self):
        self.assertEqual(wrap("um dois três quatro", len, 9), ["um dois", "três", "quatro"])


def _sdl():
    try:
        return ctypes.CDLL("libSDL2-2.0.so.0")
    except OSError:
        return None


@unittest.skipIf(_sdl() is None, "sem SDL2 neste computador")
class RealWindowTest(unittest.TestCase):
    """Abre a janela de verdade (driver de vídeo "dummy") e roda o laço."""

    def test_game_runs_actions_from_sdl_events(self):
        env = {"A11Y_PLATAFORMA": "pc", "SDL_VIDEODRIVER": "dummy", "A11Y_FALA": "terminal",
               "ALSOFT_DRIVERS": "null"}
        old = {k: os.environ.get(k) for k in env}
        os.environ.update(env)
        try:
            from r36s_a11y.game import Game
            with tempfile.TemporaryDirectory() as tmp:
                os.environ["A11Y_SAVE_DIR"] = tmp
                game = Game()
                game.actions({"pular": {"key": "space", "r36s": "b"}})
                got, labels = [], []
                sdl = game.gfx.sdl
                sdl.SDL_PushEvent.argtypes = [ctypes.c_char_p]

                @game.on_start
                def start():
                    labels.append(game.label("pular"))
                    game.say("Olá, mundo do PC")
                    sdl.SDL_PushEvent(key_event(True, SPACE, 44))
                    sdl.SDL_PushEvent(key_event(False, SPACE, 44))
                    game.after(0.2, game.quit)

                @game.on_action
                def acao(nome, estado, ms):
                    got.append((nome, estado))

                game.run()
        finally:
            os.environ.pop("A11Y_SAVE_DIR", None)
            for k, v in old.items():
                if v is None:
                    os.environ.pop(k, None)
                else:
                    os.environ[k] = v
        self.assertEqual(labels, ["espaço"])
        self.assertEqual(got, [("pular", "down"), ("pular", "up")])


if __name__ == "__main__":
    unittest.main()
