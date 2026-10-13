import sys
from pathlib import Path
import types
import unittest
from unittest.mock import Mock

sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
from r36s_a11y.input import InputState, GamepadReader, EVENT, EV_KEY, EV_ABS
from r36s_a11y.buttons import FN_CODE
from r36s_a11y.game import Game
from r36s_a11y.speech import Speech
from r36s_a11y.audio import Audio


class NativeCompatibilityTests(unittest.TestCase):
    def test_fn_axes_require_center_after_release(self):
        state=InputState()
        state.set_axis_range(0,-100,100)
        state.feed(EV_KEY,FN_CODE,1,0)
        self.assertEqual(state.feed(EV_ABS,0,100,1),[])
        state.feed(EV_KEY,FN_CODE,0,2)
        self.assertEqual(state.feed(EV_ABS,0,100,3),[])
        state.feed(EV_ABS,0,0,4)
        self.assertTrue(state.feed(EV_ABS,0,100,5))

    def test_partial_evdev_event_is_preserved_between_reads(self):
        reader=GamepadReader.__new__(GamepadReader)
        reader.state=InputState()
        reader.pending={3:b''}
        reader.clock_offset=0
        event=EVENT.pack(1,0,EV_KEY,0x130,1)
        file=Mock();file.fileno.return_value=3
        reader.files=[file]
        file.read.side_effect=[event[:5],BlockingIOError()]
        self.assertEqual(reader.read(3),[])
        self.assertEqual(reader.pending[3],event[:5])
        file.read.side_effect=[event[5:],BlockingIOError()]
        self.assertEqual(reader.read(3)[0][1:3],('b','down'))
        self.assertEqual(reader.pending[3],b'')

    def test_fn_suppresses_repeat_dispatch(self):
        game=Game.__new__(Game)
        game.input=types.SimpleNamespace(fn_held=True)
        game._repeat_at={'b':0}
        game._dispatch=Mock()
        game._tick_repeat(10)
        game._dispatch.assert_not_called()

    def test_live_voice_language_and_transport(self):
        speech=Speech.__new__(Speech)
        speech.language='en-US'
        speech._cmd=Mock()
        values=dict(VOICE_MODULE='rhvoice',VOICE='Letícia',RATE=84,PITCH=50,VOLUME=85)
        speech._apply_voice(values)
        commands=[c.args[0] for c in speech._cmd.call_args_list]
        self.assertIn('SET SELF OUTPUT_MODULE espeak-ng',commands)
        self.assertIn('SET SELF SYNTHESIS_VOICE en-us',commands)
        self.assertIn('SET SELF RATE 84',commands)
        self.assertEqual(speech.voice_values,values)
        speech.set_language('pt-BR')
        self.assertIsNone(speech.voice_values)

    def test_streaming_is_additive_and_graphics_api_remains(self):
        self.assertTrue(callable(Audio.stream))
        self.assertTrue(callable(Audio.load))
        self.assertTrue(callable(Game.on_draw))


if __name__=='__main__': unittest.main()
