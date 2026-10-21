import os
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import r36s_a11y as a11y


class KeepAwakeTests(unittest.TestCase):
    """Áudio longo (audiolivro): o a11yd não suspende enquanto o arquivo existir."""

    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.flag = os.path.join(tmp.name, "a11y-reproduzindo")
        patcher = mock.patch.dict(os.environ, {
            "A11Y_REPRODUZINDO": self.flag, "A11Y_FALA": "terminal", "A11Y_PLATAFORMA": "",
            "A11Y_SAVE_DIR": tmp.name, "A11Y_HISTORICO_FALAS": os.path.join(tmp.name, "h.txt")})
        patcher.start()
        self.addCleanup(patcher.stop)
        self.game = a11y.Game()

    def test_writes_pid_and_removes_it(self):
        self.game.keep_awake(True)
        with open(self.flag) as f:
            self.assertEqual(f.read(), str(os.getpid()))
        self.game.keep_awake(False)
        self.assertFalse(os.path.exists(self.flag))

    def test_off_without_on_leaves_other_flags_alone(self):
        with open(self.flag, "w") as f:
            f.write("1")
        self.game.keep_awake(False)
        self.assertTrue(os.path.exists(self.flag))

    def test_quitting_releases_it(self):
        @self.game.on_start
        def start():
            self.game.keep_awake()
            self.game.quit()
        with mock.patch("r36s_a11y.game.GamepadReader") as reader:
            reader.return_value.files = []
            reader.return_value.fileno_list.return_value = []
            self.game.run()
        self.assertFalse(os.path.exists(self.flag))


if __name__ == "__main__":
    unittest.main()
