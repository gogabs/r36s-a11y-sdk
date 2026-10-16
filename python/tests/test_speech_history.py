import os
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from r36s_a11y import speech_history


class SpeechHistoryTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.path = os.path.join(self.tmp.name, "historico-falas.txt")

    def lines(self):
        with open(self.path, encoding="utf-8") as f:
            return f.read().splitlines()

    def test_one_line_per_speech_and_last_fifty_kept(self):
        for i in range(2000):
            speech_history.record("Fala\nnúmero %d" % i, self.path)
        speech_history.record("   ", self.path)
        lines = self.lines()
        self.assertGreaterEqual(len(lines), speech_history.MAX_ENTRIES)
        self.assertEqual(lines[-1], "Fala número 1999")
        self.assertLessEqual(os.path.getsize(self.path), speech_history.TRIM_BYTES + 100)

    def test_without_folder_nothing_is_written(self):
        path = os.path.join(self.tmp.name, "nao-existe", "h.txt")
        speech_history.record("Oi", path)
        self.assertFalse(os.path.exists(path))

    def test_game_say_records(self):
        from r36s_a11y.game import Game
        game = Game.__new__(Game)
        game.speech = mock.Mock()
        with mock.patch.object(speech_history, "PATH", self.path):
            game.say("Pulou!")
        self.assertEqual(self.lines(), ["Pulou!"])
        game.speech.say.assert_called_once_with("Pulou!", queue=False, on_done=None)


if __name__ == "__main__":
    unittest.main()
