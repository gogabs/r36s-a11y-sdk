import os
import sys
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from r36s_a11y.buttons import FN_CODE
from r36s_a11y.input import EV_ABS, EV_KEY, InputState
from r36s_a11y.pc import (XINPUT_GUIDE, key_event, pad_buttons, pad_events, sapi_settings,
                          sdl_key_events)

ENTER, ARROW_UP, CTRL, F5 = 0x0D, 0x26, 0x11, 0x74
PAD_A, PAD_B, PAD_UP = 0x1000, 0x2000, 0x0001
REST = (0, 0, 0, 0, 0)


def feed(state, raw, ms=0):
    out = []
    for ev_type, code, value in raw:
        out.extend(state.feed(ev_type, code, value, ms))
    return out


class KeyboardTest(unittest.TestCase):
    def test_enter_is_confirm_and_repeat_is_ignored(self):
        pressed, s = set(), InputState()
        self.assertEqual(feed(s, [key_event(ENTER, True, pressed)]), [("button", "b", "down", 0)])
        self.assertEqual(feed(s, [key_event(ENTER, True, pressed)]), [])   # repetição do teclado
        self.assertEqual(feed(s, [key_event(ENTER, False, pressed)]), [("button", "b", "up", 0)])

    def test_unmapped_key_and_stray_release(self):
        self.assertIsNone(key_event(F5, True, set()))
        self.assertIsNone(key_event(ENTER, False, set()))

    def test_ctrl_is_fn(self):
        pressed, s = set(), InputState()
        raw = [key_event(CTRL, True, pressed), key_event(ARROW_UP, True, pressed),
               key_event(ARROW_UP, False, pressed), key_event(CTRL, False, pressed)]
        self.assertEqual(raw[0], (EV_KEY, FN_CODE, 1))
        self.assertEqual(feed(s, raw), [])


class WindowKeyboardTest(unittest.TestCase):
    def keys(self, *scancodes):
        k = [0] * 256
        for sc in scancodes:
            k[sc] = 1
        return k

    def test_enter_press_and_release(self):
        pressed, s = set(), InputState()
        self.assertEqual(feed(s, sdl_key_events(self.keys(40), pressed)), [("button", "b", "down", 0)])
        self.assertEqual(feed(s, sdl_key_events(self.keys(40), pressed)), [])
        self.assertEqual(feed(s, sdl_key_events(self.keys(), pressed)), [("button", "b", "up", 0)])

    def test_ctrl_pressed_together_still_suppresses(self):
        pressed, s = set(), InputState()
        raw = sdl_key_events(self.keys(82, 224), pressed)   # seta e Ctrl no mesmo quadro
        self.assertEqual(raw[0], (EV_KEY, FN_CODE, 1))
        raw += sdl_key_events(self.keys(), pressed)
        self.assertEqual(raw[-1], (EV_KEY, FN_CODE, 0))
        self.assertEqual(feed(s, raw), [])


class GamepadTest(unittest.TestCase):
    def test_xbox_a_is_bottom_button(self):
        s = InputState()
        down = (PAD_A, 0, 0, 0, 0)
        self.assertEqual(feed(s, pad_events(REST, down)), [("button", "b", "down", 0)])
        self.assertEqual(feed(s, pad_events(down, REST)), [("button", "b", "up", 0)])
        self.assertEqual(feed(s, pad_events(REST, (PAD_B, 0, 0, 0, 0))), [("button", "a", "down", 0)])

    def test_guide_is_fn_and_comes_first(self):
        cur = (XINPUT_GUIDE | PAD_UP, 0, 0, 0, 0)
        raw = pad_events(REST, cur)
        self.assertEqual(raw[0], (EV_KEY, FN_CODE, 1))
        self.assertEqual(raw[-1][:2], (EV_KEY, 0x220))
        self.assertEqual(pad_events(cur, REST)[-1], (EV_KEY, FN_CODE, 0))
        self.assertEqual(feed(InputState(), raw + pad_events(cur, REST)), [])

    def test_triggers_become_l2_r2(self):
        s = InputState()
        self.assertEqual(feed(s, pad_events(REST, (pad_buttons(0, 255, 0), 0, 0, 0, 0))),
                         [("button", "l2", "down", 0)])
        self.assertEqual(pad_buttons(0, 10, 10), 0)

    def test_stick_up_is_negative_y(self):
        s = InputState()
        s.set_axis_range(0x01, -32768, 32767)
        raw = pad_events(REST, (0, 0, 32767, 0, 0))
        self.assertEqual(raw, [(EV_ABS, 0x01, -32767)])
        (_, axis, value, _), = feed(s, raw)
        self.assertEqual(axis, "ly")
        self.assertLess(value, -0.99)


class SapiTest(unittest.TestCase):
    def test_user_voice_to_sapi(self):
        self.assertEqual(sapi_settings({}), (0, 100))
        self.assertEqual(sapi_settings({"RATE": 100, "VOLUME": 0}), (10, 50))
        self.assertEqual(sapi_settings({"RATE": -100, "VOLUME": -100}), (-10, 0))


if __name__ == "__main__":
    unittest.main()
