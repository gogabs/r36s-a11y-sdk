-- Teste de áudio 3D (OpenAL, embutido no LÖVE) e fala via libspeechd (FFI).
local ffi = require("ffi")
ffi.cdef[[
typedef struct SPDConnection SPDConnection;
SPDConnection *spd_open(const char *client, const char *conn, const char *user, int mode);
int spd_say(SPDConnection *c, int priority, const char *text);
int spd_set_language(SPDConnection *c, const char *lang);
]]
local spd = ffi.load("libspeechd.so.2")

function love.load()
  local conn = spd.spd_open("love", "teste", "ark", 0)
  spd.spd_set_language(conn, "pt-BR")
  spd.spd_say(conn, 1, "Teste do LÖVE")  -- SPD_MESSAGE = 1
  local bip = love.audio.newSource("bip.wav", "static")  -- mono: posicionável
  love.audio.setPosition(0, 0, 0)
  local passos = {{-2, 0, 0}, {0, 0, -2}, {2, 0, 0}}  -- esquerda, frente, direita
  for _, p in ipairs(passos) do
    local s = bip:clone(); s:setPosition(unpack(p)); s:play()
    love.timer.sleep(0.7)
  end
  print("ok")
  love.event.quit()
end
