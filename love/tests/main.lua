-- Testes da biblioteca. No console: cd love/tests && love .
package.path = "../?.lua;" .. package.path
local a11y = require("a11y")
local IS = a11y._InputState
local failures, total = 0, 0

local function check(name, cond)
  total = total + 1
  if not cond then
    failures = failures + 1
    print("FALHOU: " .. name)
  end
end

local function same(...) return table.concat({ ... }, ",") end

function love.load()
  local s = IS.new(0.2)
  check("aperto", same(s:feed(1, 0x130, 1, 10)) == "button,b,down,10")
  check("soltura", same(s:feed(1, 0x130, 0, 20)) == "button,b,up,20")

  s = IS.new(0.2)
  check("fn não chega", s:feed(1, 0x2C4, 1, 0) == nil)
  check("botão com fn suprimido", s:feed(1, 0x220, 1, 1) == nil)
  s:feed(1, 0x2C4, 0, 2)
  check("soltura suprimida", s:feed(1, 0x220, 0, 3) == nil)
  check("aperto seguinte", same(s:feed(1, 0x220, 1, 4)) == "button,up,down,4")

  s = IS.new(0.2)
  check("volume ignorado", s:feed(1, 115, 1, 0) == nil)
  s:feed(1, 0x130, 1, 0)
  check("auto-repeat ignorado", s:feed(1, 0x130, 2, 5) == nil)

  s = IS.new(0.2)
  s.range[0] = { -1800, 1800 }
  check("zona morta", s:feed(3, 0, 100, 0) == nil)
  check("eixo máximo", same(s:feed(3, 0, 1800, 1)) == "axis,lx,1,1")
  check("eixo centro", same(s:feed(3, 0, 0, 2)) == "axis,lx,0,2")

  check("escape", a11y.speech.escape("oi\n.ponto") == "oi\r\n..ponto\r\n.\r\n")

  print(("%d de %d testes passaram"):format(total - failures, total))
  love.event.quit(failures == 0 and 0 or 1)
end
