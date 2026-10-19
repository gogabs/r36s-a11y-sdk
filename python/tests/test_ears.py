import os
import struct
import sys
import tempfile
import time
import unittest
import wave

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, ".."))

from r36s_a11y import ears as E  # noqa: E402


class FakeMix:
    """Imita a SDL2_mixer: guarda pan, volume e o que está tocando."""

    def __init__(self):
        self.pan, self.volume, self.playing, self.next = {}, {}, set(), 0

    def Mix_PlayChannelTimed(self, channel, chunk, loops, ticks):
        ch, self.next = self.next, self.next + 1
        self.playing.add(ch)
        return ch

    def Mix_SetPanning(self, ch, left, right):
        self.pan[ch] = (left, right)

    def Mix_Volume(self, ch, volume):
        self.volume[ch] = volume

    def Mix_Playing(self, ch):
        return ch in self.playing

    def Mix_HaltChannel(self, ch):
        self.playing.discard(ch)


def fake_ears():
    ears = E.Ears.__new__(E.Ears)
    ears.base_dir, ears.fallback, ears.available = ".", None, True
    ears.sounds, ears.voices, ears._fallback_voices, ears._swap = {}, {}, [], False
    ears.mix = FakeMix()
    return ears


class EarsTest(unittest.TestCase):
    def test_each_ear_gets_only_its_channel(self):
        ears = fake_ears()
        sound = E.EarSound(ears, "bip", 1)
        left, right, both = (sound.play(ear=e) for e in ("left", "right", "both"))
        self.assertEqual(ears.mix.pan[left.channel], (255, 0))
        self.assertEqual(ears.mix.pan[right.channel], (0, 255))
        self.assertEqual(ears.mix.pan[both.channel], (255, 255))

    def test_swap_moves_playing_sounds(self):
        ears = fake_ears()
        voice = E.EarSound(ears, "bip", 1).play(ear="left")
        ears.swap = True
        self.assertEqual(ears.mix.pan[voice.channel], (0, 255))
        self.assertEqual(voice.ear, "left")   # o papel não muda, só o lado
        later = E.EarSound(ears, "bip", 1).play(ear="left")
        self.assertEqual(ears.mix.pan[later.channel], (0, 255))

    def test_bad_ear_name(self):
        with self.assertRaises(ValueError):
            E.EarSound(fake_ears(), "bip", 1).play(ear="esquerda")

    def test_fade_out_stops_and_forgets(self):
        ears = fake_ears()
        voice = E.EarSound(ears, "bip", 1).play(volume=1.0)
        voice.stop(fade=0.5)
        ears.update(0.25)
        self.assertEqual(ears.mix.volume[voice.channel], 64)
        ears.update(0.3)
        self.assertFalse(voice.playing)
        self.assertNotIn(voice.channel, ears.voices)

    def test_finished_channel_does_not_control_the_next_sound(self):
        ears = fake_ears()
        old = E.EarSound(ears, "bip", 1).play()
        ears.mix.playing.discard(old.channel)
        ears.update(0.01)
        old.set_ear("right")   # não pode mexer em quem herdou o canal
        self.assertEqual(ears.mix.pan[old.channel], (255, 255))


@unittest.skipUnless(os.environ.get("EARS_DISK_TEST"),
                     "defina EARS_DISK_TEST=1 para tocar pela SDL2_mixer de verdade")
class DiskDriverTest(unittest.TestCase):
    """Toca pelo driver "disk" da SDL e confere que nada vaza para o outro canal."""

    def test_left_stays_left(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        out = os.path.join(tmp.name, "saida.raw")
        os.environ.update(SDL_AUDIODRIVER="disk", SDL_DISKAUDIOFILE=out)
        path = os.path.join(tmp.name, "bip.wav")
        with wave.open(path, "wb") as w:
            w.setnchannels(1), w.setsampwidth(2), w.setframerate(44100)
            w.writeframes(struct.pack("<h", 10000) * 4410)
        ears = E.Ears(tmp.name)
        self.assertTrue(ears.available)
        ears.load("bip", "bip.wav").play(ear="left")
        time.sleep(0.3)
        ears.close()
        data = open(out, "rb").read()
        frames = [struct.unpack_from("<hh", data, i) for i in range(0, len(data) - 3, 4)]
        self.assertGreater(max(abs(l) for l, r in frames), 5000)
        self.assertEqual(max(abs(r) for l, r in frames), 0)


if __name__ == "__main__":
    unittest.main()
