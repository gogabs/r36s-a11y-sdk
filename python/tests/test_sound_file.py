import os
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, ".."))

from r36s_a11y.audio import sound_file  # noqa: E402


class SoundFileTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)

    def touch(self, name):
        path = os.path.join(self.tmp.name, name)
        open(path, "wb").close()
        return path

    def test_existing_file_wins(self):
        wav, ogg = self.touch("bip.wav"), self.touch("bip.ogg")
        self.assertEqual(sound_file(wav), wav)
        self.assertEqual(sound_file(ogg), ogg)

    def test_wav_falls_back_to_ogg(self):
        ogg = self.touch("bip.ogg")
        self.assertEqual(sound_file(os.path.join(self.tmp.name, "bip.wav")), ogg)

    def test_ogg_falls_back_to_wav(self):
        wav = self.touch("bip.wav")
        self.assertEqual(sound_file(os.path.join(self.tmp.name, "bip.ogg")), wav)

    def test_name_without_extension(self):
        ogg = self.touch("bip.ogg")
        self.assertEqual(sound_file(os.path.join(self.tmp.name, "bip")), ogg)

    def test_missing_keeps_the_path_for_the_error(self):
        path = os.path.join(self.tmp.name, "nada.wav")
        self.assertEqual(sound_file(path), path)


if __name__ == "__main__":
    unittest.main()
