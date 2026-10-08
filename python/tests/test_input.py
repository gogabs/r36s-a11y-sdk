import os
import sys
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from r36s_a11y.buttons import FN_CODE
from r36s_a11y.input import EV_ABS, EV_KEY, InputState
from r36s_a11y.speech import escape_text

B, UP_DPAD, VOL_UP = 0x130, 0x220, 115


class InputStateTest(unittest.TestCase):
    def setUp(self):
        self.s = InputState()

    def test_press_and_release(self):
        self.assertEqual(self.s.feed(EV_KEY, B, 1, 10), [("button", "b", "down", 10)])
        self.assertEqual(self.s.feed(EV_KEY, B, 0, 20), [("button", "b", "up", 20)])

    def test_fn_is_never_delivered(self):
        self.assertEqual(self.s.feed(EV_KEY, FN_CODE, 1, 0), [])
        self.assertEqual(self.s.feed(EV_KEY, FN_CODE, 0, 1), [])

    def test_button_with_fn_is_suppressed_until_released(self):
        self.s.feed(EV_KEY, FN_CODE, 1, 0)
        self.assertEqual(self.s.feed(EV_KEY, UP_DPAD, 1, 1), [])
        self.s.feed(EV_KEY, FN_CODE, 0, 2)          # solta Fn antes do botão
        self.assertEqual(self.s.feed(EV_KEY, UP_DPAD, 0, 3), [])
        self.assertEqual(self.s.feed(EV_KEY, UP_DPAD, 1, 4), [("button", "up", "down", 4)])

    def test_system_keys_and_kernel_repeat_ignored(self):
        self.assertEqual(self.s.feed(EV_KEY, VOL_UP, 1, 0), [])
        self.s.feed(EV_KEY, B, 1, 0)
        self.assertEqual(self.s.feed(EV_KEY, B, 2, 5), [])

    def test_axis_deadzone_and_scale(self):
        self.s.set_axis_range(0, -1800, 1800)
        self.assertEqual(self.s.feed(EV_ABS, 0, 100, 0), [])        # dentro da zona morta
        self.assertEqual(self.s.feed(EV_ABS, 0, 1800, 1), [("axis", "lx", 1.0, 1)])
        self.assertEqual(self.s.feed(EV_ABS, 0, 0, 2), [("axis", "lx", 0.0, 2)])


class SpeechTextTest(unittest.TestCase):
    def test_escape_dot_lines(self):
        self.assertEqual(escape_text("oi\n.ponto"), "oi\r\n..ponto\r\n.\r\n")


if __name__ == "__main__":
    unittest.main()
