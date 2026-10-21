"""Recordes no formato que a Página no PC lê (docs/high-scores.md).

Cada jogo guarda os seus em <save_dir>/high_scores.json: um ou mais placares
("boards"), cada um com até 10 entradas, já em ordem. Nenhum jogo precisa
escrever o arquivo à mão:

    game.high_scores.add("klondike", 4120, "Gabs", board_name="Klondike")
    game.high_scores.add("volta", 61.4, unit="segundos", order="asc")

add() devolve a posição alcançada (1 é o melhor) ou None se não entrou.
A gravação troca o arquivo inteiro de uma vez, então acabar a bateria no meio
não estraga os recordes.
"""
import datetime
import json
import os

FILE = "high_scores.json"
VERSION = 1
MAX_ENTRIES = 10


class HighScores:
    def __init__(self, save_dir):
        self.path = os.path.join(save_dir, FILE)

    def _load(self):
        try:
            with open(self.path, encoding="utf-8") as f:
                data = json.load(f)
        except (OSError, ValueError):
            return {"version": VERSION, "boards": []}
        if not isinstance(data, dict) or not isinstance(data.get("boards"), list):
            return {"version": VERSION, "boards": []}
        data["boards"] = [b for b in data["boards"] if isinstance(b, dict) and b.get("id")]
        return data

    def _save(self, data):
        os.makedirs(os.path.dirname(self.path), exist_ok=True)
        tmp = self.path + ".novo"
        with open(tmp, "w", encoding="utf-8") as f:
            json.dump(data, f, ensure_ascii=False, indent=1)
            f.flush()
            os.fsync(f.fileno())
        os.replace(tmp, self.path)

    def _board(self, data, board):
        for b in data["boards"]:
            if b["id"] == board:
                return b
        return None

    def add(self, board, score, name=None, board_name=None, unit="pontos", order="desc", date=None):
        """Registra score no placar board. Devolve a posição (1 é o melhor) ou None."""
        if order not in ("asc", "desc"):
            raise ValueError("order é 'asc' (menor é melhor) ou 'desc' (maior é melhor)")
        if isinstance(score, bool) or not isinstance(score, (int, float)):
            raise TypeError("score precisa ser um número")
        data = self._load()
        b = self._board(data, board)
        if b is None:
            b = {"id": board, "entries": []}
            data["boards"].append(b)
        b["name"] = board_name or b.get("name") or board
        b["unit"] = unit
        b["order"] = order
        entry = {"score": score, "name": name or "",
                 "date": date or datetime.date.today().isoformat()}
        entries = [e for e in b.get("entries", []) if isinstance(e, dict) and isinstance(e.get("score"), (int, float))]
        entries.append(entry)
        # Empate: quem chegou antes fica na frente (sort estável).
        entries.sort(key=lambda e: e["score"], reverse=order == "desc")
        b["entries"] = entries[:MAX_ENTRIES]
        data["version"] = VERSION
        self._save(data)
        for n, e in enumerate(b["entries"], 1):
            if e is entry:
                return n
        return None

    def entries(self, board):
        """As entradas do placar, da melhor para a pior; [] se ele não existe."""
        b = self._board(self._load(), board)
        return list(b.get("entries", [])) if b else []

    def best(self, board):
        """A melhor entrada do placar, ou None."""
        found = self.entries(board)
        return found[0] if found else None

    def qualifies(self, board, score):
        """True se score entraria no placar (para pedir o nome só quando vale)."""
        b = self._board(self._load(), board)
        if not b or len(b.get("entries", [])) < MAX_ENTRIES:
            return True
        worst = b["entries"][-1]["score"]
        return score < worst if b.get("order") == "asc" else score > worst
