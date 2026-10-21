import json
import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from r36s_a11y.high_scores import MAX_ENTRIES, HighScores  # noqa: E402


class HighScoresTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.scores = HighScores(self.tmp.name)

    def tearDown(self):
        self.tmp.cleanup()

    def read(self):
        with open(os.path.join(self.tmp.name, "high_scores.json"), encoding="utf-8") as f:
            return json.load(f)

    def test_writes_the_format_the_page_reads(self):
        self.assertEqual(self.scores.add("klondike", 900, "Ana", board_name="Klondike", date="2026-10-20"), 1)
        self.assertEqual(self.scores.add("klondike", 4120, "Gabs", date="2026-10-21"), 1)
        self.assertEqual(self.read(), {"version": 1, "boards": [{
            "id": "klondike", "entries": [{"score": 4120, "name": "Gabs", "date": "2026-10-21"},
                                          {"score": 900, "name": "Ana", "date": "2026-10-20"}],
            "name": "Klondike", "unit": "pontos", "order": "desc"}]})
        self.assertEqual(self.scores.best("klondike")["name"], "Gabs")

    def test_lower_is_better_keeps_ten_and_ties_keep_the_first(self):
        for n in range(12):
            self.scores.add("volta", 60 + n, "p%d" % n, unit="segundos", order="asc")
        found = self.scores.entries("volta")
        self.assertEqual(len(found), MAX_ENTRIES)
        self.assertEqual(found[0]["score"], 60)
        self.assertIsNone(self.scores.add("volta", 99, "lento", unit="segundos", order="asc"))
        self.assertFalse(self.scores.qualifies("volta", 99))
        self.assertTrue(self.scores.qualifies("volta", 61))
        self.assertEqual(self.scores.add("volta", 60, "empate", unit="segundos", order="asc"), 2)

    def test_broken_file_starts_over_and_bad_input_is_refused(self):
        with open(os.path.join(self.tmp.name, "high_scores.json"), "w") as f:
            f.write("{quebrado")
        self.assertEqual(self.scores.entries("x"), [])
        self.assertEqual(self.scores.add("x", 1), 1)
        with self.assertRaises(TypeError):
            self.scores.add("x", "10")
        with self.assertRaises(ValueError):
            self.scores.add("x", 10, order="maior")
        self.assertFalse(os.path.exists(os.path.join(self.tmp.name, "high_scores.json.novo")))


if __name__ == "__main__":
    unittest.main()
