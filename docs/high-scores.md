# Recordes: `high_scores.json`

Cada jogo que tem recordes grava `A11Y_SAVE_DIR/high_scores.json`
(`/roms/a11y/saves/<id>/high_scores.json` no console). A
[Página no PC](https://github.com/gogabs/r36s-a11y/blob/main/docs/pagina-no-pc.md)
lê esse arquivo para mostrar os placares em tabelas e montar a imagem para
compartilhar. O jogo não precisa escrever o arquivo à mão: a biblioteca-ponte
faz isso.

```python
game.high_scores.add("klondike", 4120, "Gabs", board_name="Klondike")
game.high_scores.add("volta", 61.4, unit="segundos", order="asc")
```

`add()` devolve a posição alcançada (1 é o melhor) ou `None` se a marca não
entrou. `qualifies(board, score)` diz antes se ela entraria, para o jogo só
pedir o nome quando vale. `entries(board)` e `best(board)` leem de volta.

## Formato

```json
{
  "version": 1,
  "boards": [
    {
      "id": "klondike",
      "name": "Klondike",
      "unit": "pontos",
      "order": "desc",
      "entries": [
        {"score": 4120, "name": "Gabs", "date": "2026-10-21"},
        {"score": 900, "name": "Ana", "date": "2026-10-20"}
      ]
    }
  ]
}
```

| Campo | O que é |
| --- | --- |
| `version` | Sempre `1` por enquanto |
| `boards[].id` | Identificador do placar, fixo (o jogo usa para gravar) |
| `boards[].name` | Nome que a página e a fala usam ("Klondike", "Fácil"). Sem ele, vale o `id`. Num jogo de placar só, use o nome do jogo |
| `boards[].unit` | Palavra depois do número ("pontos", "moedas"). `segundos` faz a página mostrar como tempo (`4:05`) |
| `boards[].order` | `desc`: maior é melhor (pontos). `asc`: menor é melhor (tempo, jogadas) |
| `boards[].entries` | Até 10, da melhor para a pior. `score` é número; `name` pode ser vazio; `date` em `AAAA-MM-DD` |

Regras para quem grava sem a biblioteca-ponte:

1. Grave num arquivo temporário e troque com `rename` (a gravação inteira de
   uma vez), para o arquivo não ficar pela metade se a bateria acabar.
2. Empate: quem chegou antes fica na frente.
3. Corte em 10 entradas por placar.

A página ignora placares sem entradas e arquivos que não são JSON válido.
