function love.conf(t)
  t.identity = "onde-esta-o-bip-love"
  t.window = false          -- jogo só de áudio: não ocupa a tela
  t.modules.joystick = false -- o controle é lido pela biblioteca, direto do /dev/input
end
