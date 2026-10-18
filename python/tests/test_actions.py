import os
import sys
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from r36s_a11y.actions import Actions, button_label


def actions(mapping):
    a = Actions()
    a.define(mapping)
    return a


class ActionsTest(unittest.TestCase):
    def test_explicit_bindings_per_device(self):
        a = actions({"pular": {"key": "space", "r36s": "b", "pad": "a"}})
        self.assertEqual(a.feed("key", "space", "down", 5), [("action", "pular", "down", 5)])
        self.assertEqual(a.feed("key", "space", "up", 9), [("action", "pular", "up", 9)])
        self.assertEqual(a.feed("key", "return", "down", 10), [])   # tecla explícita substitui a padrão
        self.assertEqual(a.feed("r36s", "b", "down", 11), [("action", "pular", "down", 11)])

    def test_defaults_follow_the_r36s_button(self):
        a = actions({"confirmar": {"r36s": "b"}, "pausa": "start"})
        self.assertEqual(a.bindings["confirmar"]["key"], ["return", "z", "kp enter"])
        self.assertEqual(a.bindings["confirmar"]["pad"], ["a"])    # o de baixo
        self.assertEqual(a.bindings["pausa"]["pad"], ["start"])
        self.assertEqual(a.feed("pad", "a", "down", 0), [("action", "confirmar", "down", 0)])

    def test_two_keys_one_action(self):
        a = actions({"atirar": {"key": ["z", "x"]}})
        self.assertEqual(len(a.feed("key", "z", "down", 0)), 1)
        self.assertEqual(a.feed("key", "x", "down", 1), [])
        self.assertEqual(a.feed("key", "z", "up", 2), [])
        self.assertTrue(a.is_held("atirar"))
        self.assertEqual(a.feed("key", "x", "up", 3), [("action", "atirar", "up", 3)])
        self.assertEqual(a.feed("key", "x", "up", 4), [])         # soltar de novo não sobe de novo

    def test_labels_by_device(self):
        a = actions({"pular": {"key": "space", "r36s": "b"},
                     "voltar": {"r36s": "a"},
                     "mirar": {"key": "shift", "mouse": "right", "r36s": "l1"}})
        self.assertEqual(a.label("pular", "keyboard"), "espaço")
        self.assertEqual(a.label("pular", "r36s"), "B")
        self.assertEqual(a.label("pular", "xbox"), "A")
        self.assertEqual(a.label("pular", "playstation"), "xis")
        self.assertEqual(a.label("pular", "nintendo"), "B")
        self.assertEqual(a.label("voltar", "xbox"), "B")
        self.assertEqual(a.label("voltar", "keyboard", every=True), "Backspace ou X")
        self.assertEqual(a.label("mirar", "keyboard", every=True),
                         "Shift esquerdo ou Shift direito ou botão direito do mouse")
        self.assertEqual(a.label("mirar", "playstation"), "L1")

    def test_aliases_and_errors(self):
        a = actions({"ok": {"key": "Enter", "pad": "RB"}})
        self.assertEqual(a.bindings["ok"]["key"], ["return"])
        self.assertEqual(a.bindings["ok"]["pad"], ["rightshoulder"])
        for bad in ({"r36s": "z"}, {"pad": "guide"}, {"key": "ctrl"}, {"teclado": "a"},
                    {"mouse": "lado"}):
            with self.assertRaises(ValueError):
                actions({"x": bad})

    def test_release_all(self):
        a = actions({"andar": {"key": "w"}})
        a.feed("key", "w", "down", 0)
        self.assertEqual(a.release_all(7), [("action", "andar", "up", 7)])
        self.assertEqual(a.release_all(8), [])

    def test_button_label(self):
        self.assertEqual(button_label("b", "r36s"), "B")
        self.assertEqual(button_label("b", "keyboard"), "Enter")
        self.assertEqual(button_label("b", "xbox"), "A")
        self.assertEqual(button_label("a", "playstation"), "bolinha")
        self.assertEqual(button_label("start", "nintendo"), "mais")


if __name__ == "__main__":
    unittest.main()
