import os
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from r36s_a11y import speech


class SpeechOffTests(unittest.TestCase):
    """Fn + Y no console: o a11yd cria o arquivo e nenhum jogo fala."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.flag = os.path.join(self.tmp.name, "fala-desligada")
        patcher = mock.patch.dict(os.environ, {"A11Y_FALA_DESLIGADA": self.flag})
        patcher.start()
        self.addCleanup(patcher.stop)

    def test_off_only_while_the_file_exists(self):
        self.assertFalse(speech.speech_off())
        open(self.flag, "w").close()
        self.assertTrue(speech.speech_off())

    def test_say_does_not_connect_and_finishes_at_once(self):
        open(self.flag, "w").close()
        s = speech.Speech()
        done = []
        with mock.patch.object(s, "_ensure") as ensure:
            s.say("Oi", on_done=done.append)
        ensure.assert_not_called()
        for fn, finished in s.poll():
            fn(finished)
        self.assertEqual(done, [True])


if __name__ == "__main__":
    unittest.main()
