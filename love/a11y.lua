--[[
Biblioteca-ponte LÖVE (11.x) da plataforma de jogos acessíveis do R36S.

    local a11y = require("a11y")

    function love.load()
      a11y.init()
      pulo = a11y.sound.load("pulo", "sounds/pulo.ogg")
    end

    function a11y.button(b, state, ms)
      if state == "down" and a11y.is_confirm(b) then
        pulo:play({pan = -0.5})
        a11y.say("Pulou!")
      end
    end

    function love.update(dt)
      a11y.update(dt)
    end

No console, lê o controle direto em /dev/input (sem exclusividade) e fala
pelo Speech Dispatcher com a voz do usuário, pelo FFI do LuaJIT. O som usa o
love.audio (OpenAL), com posição 3D e eco por ambiente.
]]

local ffi = require("ffi")

local a11y = {
  VERSION = "0.1.0",
  CONTRACT = 1,
  DOWN = "down", UP = "up", REPEAT = "repeat",
}

-- Callbacks que o jogo pode definir (como os do LÖVE).
function a11y.button(button, state, ms) end
function a11y.axis(axis, value, ms) end

ffi.cdef [[
  typedef struct { long sec; long usec; uint16_t type; uint16_t code; int32_t value; } a11y_input_event;
  typedef struct { uint16_t sun_family; char sun_path[108]; } a11y_sockaddr_un;
  typedef struct { int fd; short events; short revents; } a11y_pollfd;
  int open(const char *path, int flags);
  int close(int fd);
  long read(int fd, void *buf, unsigned long count);
  int ioctl(int fd, unsigned long request, void *arg);
  int socket(int domain, int type, int protocol);
  int connect(int fd, const void *addr, unsigned int len);
  long send(int fd, const void *buf, unsigned long len, int flags);
  long recv(int fd, void *buf, unsigned long len, int flags);
  int poll(a11y_pollfd *fds, unsigned long n, int timeout);
  unsigned int getuid(void);
]]
local C = ffi.C
local O_NONBLOCK = 0x800
local MSG_DONTWAIT, MSG_NOSIGNAL = 0x40, 0x4000

-- --- botões --------------------------------------------------------------

local BUTTON_CODES = {
  [0x130] = "b", [0x131] = "a", [0x133] = "x", [0x134] = "y",
  [0x136] = "l1", [0x137] = "r1", [0x138] = "l2", [0x139] = "r2",
  [0x220] = "up", [0x221] = "down", [0x222] = "left", [0x223] = "right",
  [0x2C0] = "select", [0x2C1] = "start", [0x2C2] = "l3", [0x2C3] = "r3",
}
local FN_CODE = 0x2C4 -- do sistema (a11yd): nunca chega ao jogo
local AXIS_CODES = { [0x00] = "lx", [0x01] = "ly", [0x03] = "rx", [0x04] = "ry" }
local EV_KEY, EV_ABS = 0x01, 0x03

--- B (de baixo) confirma.
function a11y.is_confirm(b) return b == "b" end
--- A (da direita) volta.
function a11y.is_back(b) return b == "a" end

--- Estado da entrada: descarta Fn e o que for apertado com ele, volume,
--- Power e o auto-repeat do kernel; aplica zona morta aos eixos.
local InputState = {}
InputState.__index = InputState
a11y._InputState = InputState -- para os testes

function InputState.new(deadzone)
  return setmetatable({ deadzone = deadzone or 0.2, fn = false, suppressed = {},
                        held = {}, range = {}, value = {} }, InputState)
end

function InputState:feed(ev_type, code, value, ms)
  if ev_type == EV_KEY then
    if code == FN_CODE then
      self.fn = value ~= 0
      return nil
    end
    local b = BUTTON_CODES[code]
    if not b or value == 2 then return nil end
    if value == 1 then
      if self.fn then
        self.suppressed[b] = true
        return nil
      end
      self.held[b] = true
      return "button", b, "down", ms
    end
    if self.suppressed[b] then
      self.suppressed[b] = nil
      return nil
    end
    if not self.held[b] then return nil end
    self.held[b] = nil
    return "button", b, "up", ms
  elseif ev_type == EV_ABS then
    local axis = AXIS_CODES[code]
    if not axis then return nil end
    local r = self.range[code] or { -1800, 1800 }
    local center, half = (r[1] + r[2]) / 2, (r[2] - r[1]) / 2
    if half == 0 then half = 1 end
    local v = math.max(-1, math.min(1, (value - center) / half))
    if math.abs(v) < self.deadzone then
      v = 0
    else
      local s = v > 0 and 1 or -1
      v = s * (math.abs(v) - self.deadzone) / (1 - self.deadzone)
    end
    local last = self.value[axis] or 0
    if v == last or (v ~= 0 and math.abs(v - last) < 0.02) then return nil end
    self.value[axis] = v
    return "axis", axis, v, ms
  end
  return nil
end

-- --- leitura do controle --------------------------------------------------

-- _IOR('E', 0x40 + axis, struct input_absinfo); conta comum porque bit.* é de 32 bits com sinal
local function eviocgabs(axis)
  return 2 * 2 ^ 30 + 24 * 2 ^ 16 + string.byte("E") * 2 ^ 8 + 0x40 + axis
end

local devices = {}

local function open_devices(state)
  for i = 0, 31 do
    local fd = C.open("/dev/input/event" .. i, O_NONBLOCK) -- O_RDONLY = 0
    if fd >= 0 then
      devices[#devices + 1] = fd
      for code in pairs(AXIS_CODES) do
        local info = ffi.new("int32_t[6]")
        if C.ioctl(fd, eviocgabs(code), info) == 0 and info[2] > info[1] then
          state.range[code] = { info[1], info[2] }
        end
      end
    end
  end
  return #devices
end

local ev_buf = ffi.new("a11y_input_event[64]")
local ev_size = ffi.sizeof("a11y_input_event")

-- --- fala (SSIP) ----------------------------------------------------------

local speech = { fd = -1, available = true, buf = "", callbacks = {}, early = {}, ready = {} }
a11y.speech = speech

local function voz_conf()
  local path = os.getenv("A11Y_VOZ_CONF") or "/opt/a11y/etc/voz.conf"
  local numbers, names = {}, {}
  local f = io.open(path)
  if not f then return numbers, names end
  for line in f:lines() do
    local k, v = line:match("^%s*([%w_]+)%s*=%s*(.-)%s*$")
    if k == "VOICE_MODULE" or k == "VOICE" then
      v = v:gsub("^['\"]", ""):gsub("['\"]$", "")
      if v ~= "" then names[k] = v end
    elseif (k == "RATE" or k == "PITCH" or k == "VOLUME") and tonumber(v) then
      numbers[k] = math.max(0, math.min(100, tonumber(v))) * 2 - 100
    end
  end
  f:close()
  return numbers, names
end

--- Formata o texto para o comando SPEAK do SSIP.
function speech.escape(text)
  local out = {}
  for line in (text:gsub("\r", "") .. "\n"):gmatch("(.-)\n") do
    out[#out + 1] = line:sub(1, 1) == "." and ("." .. line) or line
  end
  return table.concat(out, "\r\n") .. "\r\n.\r\n"
end

function speech.close()
  if speech.fd >= 0 then C.close(speech.fd) end
  speech.fd, speech.buf = -1, ""
end

local function recv_some(timeout_ms)
  local pfd = ffi.new("a11y_pollfd[1]")
  pfd[0].fd, pfd[0].events = speech.fd, 1
  if C.poll(pfd, 1, timeout_ms) <= 0 then return false end
  local tmp = ffi.new("char[4096]")
  local n = tonumber(C.recv(speech.fd, tmp, 4096, MSG_DONTWAIT))
  if n <= 0 then
    speech.close()
    return false
  end
  speech.buf = speech.buf .. ffi.string(tmp, n)
  return true
end

-- Tira do buffer um bloco "NNN-..." terminado por "NNN ...", se completo.
local function take_block()
  local lines, pos = {}, 1
  while true do
    local s, e = speech.buf:find("\r\n", pos, true)
    if not s then return nil end
    local line = speech.buf:sub(pos, s - 1)
    lines[#lines + 1] = line
    pos = e + 1
    if line:sub(4, 4) == " " then
      speech.buf = speech.buf:sub(pos)
      return lines
    end
  end
end

local function handle_event(lines)
  if #lines < 3 then return end
  local id = lines[1]:sub(5)
  local finished = lines[#lines]:sub(1, 3) == "702" -- 702 END, 703 CANCEL
  local fn = speech.callbacks[id]
  if fn then
    speech.callbacks[id] = nil
    speech.ready[#speech.ready + 1] = function() fn(finished) end
  else
    speech.early[id] = finished
  end
end

local function reply()
  local deadline = love.timer.getTime() + 5
  while speech.fd >= 0 do
    local block = take_block()
    if block then
      if block[#block]:sub(1, 1) == "7" then
        handle_event(block)
      else
        return block
      end
    elseif love.timer.getTime() > deadline or not recv_some(200) and speech.fd < 0 then
      break
    end
  end
  error("speech-dispatcher não respondeu")
end

local function send(data)
  if C.send(speech.fd, data, #data, MSG_NOSIGNAL) < 0 then
    speech.close()
    error("speech-dispatcher fechou a conexão")
  end
end

local function cmd(command)
  send(command .. "\r\n")
  return reply()
end

local function connect()
  local runtime = os.getenv("XDG_RUNTIME_DIR") or ("/run/user/" .. tonumber(C.getuid()))
  local path = runtime .. "/speech-dispatcher/speechd.sock"
  local fd = C.socket(1, 1, 0) -- AF_UNIX, SOCK_STREAM
  if fd < 0 then error("socket") end
  local addr = ffi.new("a11y_sockaddr_un")
  addr.sun_family = 1
  ffi.copy(addr.sun_path, path)
  if C.connect(fd, addr, ffi.sizeof(addr)) ~= 0 then
    C.close(fd)
    error("sem speech-dispatcher")
  end
  speech.fd, speech.buf = fd, ""
  local numbers, names = voz_conf()
  cmd(("SET SELF CLIENT_NAME %s:a11y:%s"):format(os.getenv("USER") or "ark", a11y.id or "jogo"))
  -- módulo antes do idioma, e voz depois: trocar o idioma pode trocar a voz
  if names.VOICE_MODULE then cmd("SET SELF OUTPUT_MODULE " .. names.VOICE_MODULE) end
  cmd("SET SELF LANGUAGE " .. (a11y.lang or "pt-BR"))
  if names.VOICE then cmd("SET SELF SYNTHESIS_VOICE " .. names.VOICE) end
  for _, k in ipairs({ "RATE", "PITCH", "VOLUME" }) do
    if numbers[k] then cmd(("SET SELF %s %d"):format(k, numbers[k])) end
  end
  cmd("SET SELF NOTIFICATION END on")
  cmd("SET SELF NOTIFICATION CANCEL on")
end

local function ensure()
  if speech.fd < 0 and speech.available then
    local ok = pcall(connect)
    if not ok then
      speech.close()
      speech.available = false
    end
  end
  return speech.fd >= 0
end

--- Fala o texto. opts.queue = true espera a fala anterior; opts.on_done(terminou)
--- é chamado ao fim (terminou = false se a fala foi cortada).
function a11y.say(text, opts)
  opts = opts or {}
  for _ = 1, 2 do
    if not ensure() then
      io.stderr:write("[fala] " .. text .. "\n")
      if opts.on_done then speech.ready[#speech.ready + 1] = function() opts.on_done(true) end end
      return
    end
    local ok, err = pcall(function()
      if not opts.queue then cmd("STOP SELF") end
      cmd("SPEAK")
      send(speech.escape(text))
      local r = reply()
      if opts.on_done and #r > 1 then
        local id = r[1]:sub(5) -- 225-<id da mensagem> / 225 OK MESSAGE QUEUED
        if speech.early[id] ~= nil then
          local finished = speech.early[id]
          speech.early[id] = nil
          speech.ready[#speech.ready + 1] = function() opts.on_done(finished) end
        else
          speech.callbacks[id] = opts.on_done
        end
      end
    end)
    if ok then return end
    speech.close()
  end
end

--- Fala e chama then quando terminar.
function a11y.say_then(text, then_fn)
  a11y.say(text, { on_done = function() then_fn() end })
end

--- Cala a fala.
function a11y.hush()
  if ensure() and not pcall(cmd, "STOP SELF") then speech.close() end
end

local function poll_speech()
  if speech.fd >= 0 then
    while recv_some(0) do end
    while true do
      local block = take_block()
      if not block then break end
      if block[#block]:sub(1, 1) == "7" then handle_event(block) end
    end
  end
  local ready = speech.ready
  speech.ready = {}
  for _, fn in ipairs(ready) do fn() end
  local n = 0
  for _ in pairs(speech.early) do n = n + 1 end
  if n > 256 then speech.early = {} end
end

-- --- som -----------------------------------------------------------------

local sound = { voices = {}, loaded = {}, reverb_on = false }
a11y.sound = sound

--- Ambientes prontos (presets EFX da Creative, nos nomes do LÖVE).
sound.REVERB_PRESETS = {
  quarto = { density = 0.43, diffusion = 1, gain = 0.32, highgain = 0.6, decaytime = 0.4, decayhighratio = 0.83, earlygain = 0.15, earlydelay = 0.002, lategain = 1.06, latedelay = 0.003 },
  corredor = { density = 1, diffusion = 1, gain = 0.32, highgain = 0.71, decaytime = 1.49, decayhighratio = 0.59, earlygain = 0.25, earlydelay = 0.007, lategain = 1.66, latedelay = 0.011 },
  estacionamento = { density = 1, diffusion = 1, gain = 0.32, highgain = 1, decaytime = 1.65, decayhighratio = 1.5, earlygain = 0.21, earlydelay = 0.008, lategain = 0.27, latedelay = 0.012 },
  cozinha = { density = 1, diffusion = 0.9, gain = 0.32, highgain = 0.9, decaytime = 1, decayhighratio = 1.2, earlygain = 0.5, earlydelay = 0.004, lategain = 1.2, latedelay = 0.006 },
  metro = { density = 1, diffusion = 1, gain = 0.32, highgain = 0.71, decaytime = 3.1, decayhighratio = 1, earlygain = 0.15, earlydelay = 0.03, lategain = 1.4, latedelay = 0.03 },
  igreja = { density = 1, diffusion = 1, gain = 0.32, highgain = 0.6, decaytime = 5.5, decayhighratio = 0.6, earlygain = 0.2, earlydelay = 0.04, lategain = 1.3, latedelay = 0.05 },
  floresta = { density = 1, diffusion = 0.3, gain = 0.32, highgain = 0.02, decaytime = 1.49, decayhighratio = 0.54, earlygain = 0.05, earlydelay = 0.16, lategain = 0.2, latedelay = 0.09 },
}

local Sound, Voice = {}, {}
Sound.__index, Voice.__index = Sound, Voice

--- Carrega um WAV ou OGG (relativo à pasta do jogo) com um nome.
function sound.load(name, path)
  local s = setmetatable({ name = name, source = love.audio.newSource(path, "static") }, Sound)
  s.channels = s.source:getChannelCount()
  sound.loaded[name] = s
  return s
end

--- Som já carregado, pelo nome.
function sound.get(name) return sound.loaded[name] end

local function place(src, pos, pan)
  if src:getChannelCount() ~= 1 then return end -- estéreo não é posicionado
  if pos then
    src:setRelative(false)
    src:setPosition(pos[1], pos[2], pos[3])
  elseif pan then
    pan = math.max(-1, math.min(1, pan))
    src:setRelative(true)
    src:setPosition(pan, 0, -math.sqrt(1 - pan * pan))
  else
    src:setRelative(true)
    src:setPosition(0, 0, 0)
  end
end

--- Toca o som. opts: loop, volume, pitch, pos = {x, y, z}, pan (-1 a 1),
--- reverb (padrão: só com pos). Coordenadas em metros: x à direita, y para
--- cima, z negativo à frente. Só sons mono são posicionados.
function Sound:play(opts)
  opts = opts or {}
  local src = self.source:clone()
  src:setLooping(opts.loop or false)
  src:setVolume(opts.volume or 1)
  src:setPitch(opts.pitch or 1)
  if src:getChannelCount() == 1 then
    src:setAttenuationDistances(1, 100)
    src:setRolloff(1)
  end
  place(src, opts.pos, opts.pan)
  local reverb = opts.reverb
  if reverb == nil then reverb = opts.pos ~= nil end
  if reverb and sound.reverb_on then pcall(src.setEffect, src, "a11y_ambiente", true) end
  src:play()
  local v = setmetatable({ source = src, volume = opts.volume or 1 }, Voice)
  sound.voices[#sound.voices + 1] = v
  return v
end

function Voice:playing() return self.source:isPlaying() or self.paused == true end
function Voice:set_position(x, y, z) place(self.source, { x, y, z }) end
function Voice:set_pan(p) place(self.source, nil, p) end
function Voice:set_pitch(p) self.source:setPitch(p) end
function Voice:pause()
  if self.source:isPlaying() then
    self.source:pause()
    self.paused = true
  end
end
function Voice:resume()
  if self.paused then
    self.paused = false
    self.source:play()
  end
end

--- Muda o volume; com fade (segundos) > 0, em transição.
function Voice:set_volume(v, fade)
  if fade and fade > 0 then
    self.fade = { from = self.volume, to = v, total = fade, elapsed = 0 }
  else
    self.fade = nil
    self.volume = v
    self.source:setVolume(v)
  end
end

--- Para o som; com fade (segundos) > 0, abaixando aos poucos.
function Voice:stop(fade)
  if fade and fade > 0 then
    self.fade = { from = self.volume, to = 0, total = fade, elapsed = 0, stop = true }
  else
    self.source:stop()
    self.paused = false
  end
end

--- Posição do ouvinte e para onde ele olha (graus; 0 = frente, 90 = direita).
function sound.listener(pos, facing)
  love.audio.setPosition(pos[1], pos[2], pos[3])
  local r = math.rad(facing or 0)
  love.audio.setOrientation(math.sin(r), 0, -math.cos(r), 0, 1, 0)
end

--- Muda o ambiente: nome de REVERB_PRESETS, tabela de parâmetros do LÖVE, ou nil.
function sound.set_reverb(preset)
  if not love.audio.isEffectsSupported() then return end
  if preset == nil then
    love.audio.setEffect("a11y_ambiente", false)
    sound.reverb_on = false
    return
  end
  local params = type(preset) == "table" and preset or sound.REVERB_PRESETS[preset]
  local effect = { type = "reverb" }
  for k, v in pairs(params) do effect[k] = v end
  sound.reverb_on = love.audio.setEffect("a11y_ambiente", effect)
end

--- Pausa todos os sons (por exemplo, no menu de pausa).
function sound.pause_all() for _, v in ipairs(sound.voices) do v:pause() end end
--- Retoma os sons pausados.
function sound.resume_all() for _, v in ipairs(sound.voices) do v:resume() end end

local function update_sound(dt)
  local keep = {}
  for _, v in ipairs(sound.voices) do
    local f = v.fade
    if f then
      f.elapsed = f.elapsed + dt
      local t = math.min(1, f.elapsed / f.total)
      v.volume = f.from + (f.to - f.from) * t
      v.source:setVolume(v.volume)
      if t >= 1 then
        v.fade = nil
        if f.stop then v.source:stop() end
      end
    end
    if v.fade or v:playing() then keep[#keep + 1] = v end
  end
  sound.voices = keep
end

-- --- menu ----------------------------------------------------------------

local focus = {}

local Menu = {}
Menu.__index = Menu
a11y.Menu = Menu
Menu.HELP = "Cima e baixo escolhem. B confirma. A volta."

--- Menu vertical falado: Cima/Baixo navegam (dando a volta), B confirma,
--- A volta (on_back, ou fecha), Select repete. Aberto, recebe os botões.
function Menu.new(title, items, on_select, opts)
  opts = opts or {}
  return setmetatable({ title = title, items = items, on_select = on_select,
                        on_back = opts.on_back, move_sound = opts.move_sound,
                        select_sound = opts.select_sound, index = 1, open = false }, Menu)
end

function Menu:item_text()
  return ("%s, %d de %d"):format(self.items[self.index], self.index, #self.items)
end

--- Abre no item index (padrão 1), falando o título, o item e a instrução (help).
function Menu:show(index, help)
  self.index = index or 1
  if not self.open then
    self.open = true
    focus[#focus + 1] = self
  end
  local text = self.title .. ". " .. self:item_text()
  if help ~= false then text = text .. ". " .. Menu.HELP end
  a11y.say(text)
end

function Menu:close()
  self.open = false
  for i = #focus, 1, -1 do
    if focus[i] == self then table.remove(focus, i) end
  end
end

function Menu:handle(b, state)
  if state == "up" then return end
  if b == "down" or b == "up" then
    local n = #self.items
    self.index = ((self.index - 1 + (b == "down" and 1 or -1)) % n) + 1
    if self.move_sound then self.move_sound:play() end
    a11y.say(self:item_text())
  elseif state ~= "down" then
    return
  elseif a11y.is_confirm(b) then
    if self.select_sound then self.select_sound:play() end
    self.on_select(self.index, self.items[self.index])
  elseif a11y.is_back(b) then
    if self.on_back then self.on_back() else self:close() end
  elseif b == "select" then
    a11y.say(self:item_text())
  end
end

-- --- laço ----------------------------------------------------------------

local state, options, timers, repeat_at = nil, {}, {}, {}
local start_time = 0
local REPEAT_DELAY, REPEAT_INTERVAL = 0.4, 0.1

--- Milissegundos desde o início do jogo.
function a11y.ms() return math.floor((love.timer.getTime() - start_time) * 1000) end

--- Chama fn daqui a alguns segundos.
function a11y.after(seconds, fn)
  timers[#timers + 1] = { at = love.timer.getTime() + seconds, fn = fn }
end

--- Último valor de um analógico ("lx", "ly", "rx", "ry").
function a11y.axis_value(name) return state and state.value[name] or 0 end

local function dispatch_button(b, st, ms)
  local top = focus[#focus]
  if top then top:handle(b, st, ms) else a11y.button(b, st, ms) end
end

--- Prepara a biblioteca. opts: repeat_buttons, axes, deadzone.
function a11y.init(opts)
  options = opts or {}
  a11y.lang = os.getenv("A11Y_LANG") or "pt-BR"
  a11y.game_dir = os.getenv("A11Y_GAME_DIR") or love.filesystem.getSource()
  a11y.id = a11y.game_dir:match("([^/]+)/*%.?$") or "jogo"
  local ok, manifest = pcall(love.filesystem.read, "game.json")
  if ok and manifest then a11y.id = manifest:match('"id"%s*:%s*"([^"]+)"') or a11y.id end
  a11y.save_dir = os.getenv("A11Y_SAVE_DIR") or
      ((os.getenv("HOME") or ".") .. "/.local/share/r36s-a11y/" .. a11y.id)
  os.execute("mkdir -p '" .. a11y.save_dir .. "'")
  start_time = love.timer.getTime()
  state = InputState.new(options.deadzone)
  love.audio.setDistanceModel("inverseclamped")
  sound.listener({ 0, 0, 0 }, 0)
  if open_devices(state) == 0 then
    io.stderr:write("[a11y] nenhum controle encontrado em /dev/input\n")
  end
end

--- Chame em love.update(dt): lê o controle, entrega falas terminadas,
--- repetições, temporizadores e transições de volume.
function a11y.update(dt)
  local now = a11y.ms()
  local wall = love.timer.getTime()
  for _, fd in ipairs(devices) do
    while true do
      local n = tonumber(C.read(fd, ev_buf, ev_size * 64))
      if not n or n <= 0 then break end
      for i = 0, n / ev_size - 1 do
        local ev = ev_buf[i]
        local kind, name, value = state:feed(ev.type, ev.code, ev.value, now)
        if kind == "button" then
          if options.repeat_buttons then
            repeat_at[name] = value == "down" and (wall + REPEAT_DELAY) or nil
          end
          dispatch_button(name, value, now)
        elseif kind == "axis" and options.axes then
          a11y.axis(name, value, now)
        end
      end
    end
  end
  for b, at in pairs(repeat_at) do
    if wall >= at then
      repeat_at[b] = at + REPEAT_INTERVAL
      dispatch_button(b, "repeat", now)
    end
  end
  poll_speech()
  local due, keep = {}, {}
  for _, t in ipairs(timers) do
    if wall >= t.at then due[#due + 1] = t else keep[#keep + 1] = t end
  end
  timers = keep
  table.sort(due, function(x, y) return x.at < y.at end)
  for _, t in ipairs(due) do t.fn() end
  update_sound(dt)
end

return a11y
