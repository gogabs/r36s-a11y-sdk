import json
import os
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))


class ManifestActionsTest(unittest.TestCase):
    def test_game_registers_the_actions_of_game_json(self):
        with tempfile.TemporaryDirectory() as tmp:
            with open(os.path.join(tmp, "game.json"), "w", encoding="utf-8") as f:
                json.dump({"id": "teste", "name": "Teste", "actions": {
                    "pular": {"r36s": "b", "key": "space"},
                    "sair": {"r36s": "a", "key": "escape"}}}, f)
            env = {"A11Y_GAME_DIR": tmp, "A11Y_SAVE_DIR": os.path.join(tmp, "save"),
                   "A11Y_FALA": "terminal", "ALSOFT_DRIVERS": "null", "A11Y_PLATAFORMA": "pc"}
            with mock.patch.dict(os.environ, env):
                from r36s_a11y.game import Game
                game = Game()
        self.assertEqual(sorted(game.action_map.names()), ["pular", "sair"])
        self.assertEqual(game.label("pular"), "espaço")
        self.assertEqual(game.label("pular", "r36s"), "B")


if __name__ == "__main__":
    unittest.main()
