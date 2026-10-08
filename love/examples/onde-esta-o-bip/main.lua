-- Onde está o bip? Jogo de exemplo da biblioteca LÖVE.
--
-- Um bip toca à esquerda, à frente, à direita ou atrás de você. Aperte o
-- direcional para o lado de onde ele veio. Com fone, a OpenAL usa HRTF e dá
-- para distinguir frente e trás.

-- Usa a cópia de a11y.lua ao lado do jogo; no repositório, a da pasta love/.
package.path = package.path .. ";../../?.lua"
local a11y = require("a11y")

local ROUNDS = 10
local DIRECTIONS = {
  left = { "à esquerda", { -2, 0, 0 } },
  up = { "à frente", { 0, 0, -2 } },
  right = { "à direita", { 2, 0, 0 } },
  down = { "atrás", { 0, 0, 2 } },
}
local BUTTONS = { "left", "up", "right", "down" }

local bip, acerto, erro
local main_menu, pause_menu
local number, score = 0, 0
local answer = nil -- nil fora de uma partida
local waiting = false

local function record_file() return a11y.save_dir .. "/recorde.txt" end

local function load_record()
  local f = io.open(record_file())
  if not f then return 0 end
  local v = tonumber(f:read("*a")) or 0
  f:close()
  return v
end

local function play_bip()
  bip:play({ pos = DIRECTIONS[answer][2] })
  waiting = true
end

local finish

local function next_round()
  if number >= ROUNDS then
    finish()
    return
  end
  number = number + 1
  answer = BUTTONS[love.math.random(#BUTTONS)]
  waiting = false
  a11y.after(0.6, play_bip)
end

function finish()
  local record = load_record()
  local text = ("Fim. Você acertou %d de %d."):format(score, ROUNDS)
  if score > record then
    local f = io.open(record_file(), "w")
    if f then
      f:write(tostring(score))
      f:close()
    end
    text = text .. " Novo recorde!"
  elseif record > 0 then
    text = text .. (" Recorde: %d."):format(record)
  end
  answer = nil
  a11y.say_then(text, function() main_menu:show(1, false) end)
end

local function respond(b)
  if not waiting then return end
  if b == "select" then
    play_bip()
    return
  end
  if not DIRECTIONS[b] then return end
  waiting = false
  if b == answer then
    score = score + 1
    acerto:play()
    a11y.after(0.5, next_round)
  else
    erro:play()
    a11y.say("Era " .. DIRECTIONS[answer][1] .. ".", { queue = true, on_done = next_round })
  end
end

function love.load()
  a11y.init()
  bip = a11y.sound.load("bip", "sounds/bip.wav")
  acerto = a11y.sound.load("acerto", "sounds/acerto.wav")
  erro = a11y.sound.load("erro", "sounds/erro.wav")
  local move = a11y.sound.load("move", "sounds/move.wav")

  main_menu = a11y.Menu.new("Onde está o bip", { "Jogar", "Instruções", "Sair" },
    function(_, item)
      if item == "Jogar" then
        main_menu:close()
        number, score = 0, 0
        a11y.say_then("Valendo! Ouça o bip e aperte o direcional para o lado dele.", next_round)
      elseif item == "Instruções" then
        a11y.say("Um bip toca à esquerda, à frente, à direita ou atrás de você. " ..
                 "Aperte o direcional para o lado de onde ele veio. " ..
                 "Select repete o bip. Start pausa. Use fone de ouvido.")
      else
        love.event.quit()
      end
    end, { on_back = love.event.quit, move_sound = move })

  pause_menu = a11y.Menu.new("Pausa", { "Continuar", "Sair para o menu" },
    function(_, item)
      pause_menu:close()
      if item == "Continuar" then
        a11y.say_then("Continuando.", play_bip)
        return
      end
      answer = nil
      main_menu:show(1, false)
    end, { move_sound = move })
  pause_menu.on_back = function() pause_menu.on_select(1, "Continuar") end

  main_menu:show()
end

function a11y.button(b, state)
  if state ~= "down" or answer == nil then return end
  if b == "start" then
    waiting = false
    pause_menu:show()
    return
  end
  respond(b)
end

function love.update(dt)
  a11y.update(dt)
end
