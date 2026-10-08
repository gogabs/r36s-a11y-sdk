package a11y

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

// Coordenadas da OpenAL, em metros: x positivo à direita, y para cima,
// z negativo à frente. Só sons mono são posicionados.

const (
	alSourceRelative       = 0x202
	alPitch                = 0x1003
	alPosition             = 0x1004
	alLooping              = 0x1007
	alBuffer               = 0x1009
	alGain                 = 0x100A
	alOrientation          = 0x100F
	alSourceState          = 0x1010
	alPlaying              = 0x1012
	alPaused               = 0x1013
	alReferenceDistance    = 0x1020
	alRolloffFactor        = 0x1021
	alMaxDistance          = 0x1023
	alFormatMono8          = 0x1100
	alFormatMono16         = 0x1101
	alFormatStereo8        = 0x1102
	alFormatStereo16       = 0x1103
	alInverseDistClamped   = 0xD002
	alcHRTFSoft            = 0x1992
	alEffectType           = 0x8001
	alEffectReverb         = 0x0001
	alEffectSlotEffect     = 0x0001
	alAuxiliarySendFilter  = 0x20006
	maxSources             = 32
)

// Funções da OpenAL e da libvorbisfile, preenchidas por loadOpenAL (Linux).
var (
	alcOpenDevice          func(name uintptr) uintptr
	alcCreateContext       func(dev uintptr, attrs *int32) uintptr
	alcMakeContextCurrent  func(ctx uintptr) bool
	alcIsExtensionPresent  func(dev uintptr, name string) bool
	alcGetIntegerv         func(dev uintptr, param, size int32, data *int32)
	alcDestroyContext      func(ctx uintptr)
	alcCloseDevice         func(dev uintptr) bool
	alDistanceModel        func(model int32)
	alGenBuffers           func(n int32, out *uint32)
	alBufferData           func(buf uint32, format int32, data unsafe.Pointer, size, freq int32)
	alGenSources           func(n int32, out *uint32)
	alSourcei              func(src uint32, param, value int32)
	alSourcef              func(src uint32, param int32, value float32)
	alSource3f             func(src uint32, param int32, x, y, z float32)
	alSource3i             func(src uint32, param int32, a, b, c int32)
	alGetSourcei           func(src uint32, param int32, out *int32)
	alSourcePlay           func(src uint32)
	alSourceStop           func(src uint32)
	alSourcePause          func(src uint32)
	alListener3f           func(param int32, x, y, z float32)
	alListenerfv           func(param int32, values *float32)
	alGenAuxEffectSlots    func(n int32, out *uint32)
	alGenEffects           func(n int32, out *uint32)
	alEffecti              func(effect uint32, param, value int32)
	alEffectf              func(effect uint32, param int32, value float32)
	alAuxiliaryEffectSloti func(slot uint32, param, value int32)
	efxAvailable           bool
)

// ReverbPresets são os ambientes prontos (baseados nos presets EFX da Creative).
var ReverbPresets = map[string]map[string]float64{
	"quarto":         {"density": 0.43, "diffusion": 1, "gain": 0.32, "gain_hf": 0.6, "decay": 0.4, "decay_hf_ratio": 0.83, "reflections_gain": 0.15, "reflections_delay": 0.002, "late_gain": 1.06, "late_delay": 0.003},
	"corredor":       {"density": 1, "diffusion": 1, "gain": 0.32, "gain_hf": 0.71, "decay": 1.49, "decay_hf_ratio": 0.59, "reflections_gain": 0.25, "reflections_delay": 0.007, "late_gain": 1.66, "late_delay": 0.011},
	"estacionamento": {"density": 1, "diffusion": 1, "gain": 0.32, "gain_hf": 1, "decay": 1.65, "decay_hf_ratio": 1.5, "reflections_gain": 0.21, "reflections_delay": 0.008, "late_gain": 0.27, "late_delay": 0.012},
	"cozinha":        {"density": 1, "diffusion": 0.9, "gain": 0.32, "gain_hf": 0.9, "decay": 1, "decay_hf_ratio": 1.2, "reflections_gain": 0.5, "reflections_delay": 0.004, "late_gain": 1.2, "late_delay": 0.006},
	"metro":          {"density": 1, "diffusion": 1, "gain": 0.32, "gain_hf": 0.71, "decay": 3.1, "decay_hf_ratio": 1, "reflections_gain": 0.15, "reflections_delay": 0.03, "late_gain": 1.4, "late_delay": 0.03},
	"igreja":         {"density": 1, "diffusion": 1, "gain": 0.32, "gain_hf": 0.6, "decay": 5.5, "decay_hf_ratio": 0.6, "reflections_gain": 0.2, "reflections_delay": 0.04, "late_gain": 1.3, "late_delay": 0.05},
	"floresta":       {"density": 1, "diffusion": 0.3, "gain": 0.32, "gain_hf": 0.02, "decay": 1.49, "decay_hf_ratio": 0.54, "reflections_gain": 0.05, "reflections_delay": 0.16, "late_gain": 0.2, "late_delay": 0.09},
}

var reverbParams = map[string]int32{
	"density": 1, "diffusion": 2, "gain": 3, "gain_hf": 4, "decay": 5,
	"decay_hf_ratio": 6, "reflections_gain": 7, "reflections_delay": 8,
	"late_gain": 9, "late_delay": 10,
}

// Audio é o áudio 3D do jogo.
type Audio struct {
	baseDir   string
	available bool
	HRTF      bool // HRTF ligado (som binaural no fone)
	device    uintptr
	context   uintptr
	slot      uint32
	effect    uint32
	sounds    map[string]*Sound
	voices    map[uint32]*Voice
	order     []uint32 // fontes em uso, da mais antiga para a mais nova
	free      []uint32
}

func newAudio(baseDir string) *Audio {
	a := &Audio{baseDir: baseDir, sounds: map[string]*Sound{}, voices: map[uint32]*Voice{}}
	if err := loadOpenAL(); err != nil {
		return a
	}
	a.device = alcOpenDevice(0) // dispositivo padrão = dmix do ALSA
	if a.device == 0 {
		return a
	}
	var attrs *int32
	hrtf := []int32{alcHRTFSoft, 1, 0}
	if alcIsExtensionPresent(a.device, "ALC_SOFT_HRTF") {
		attrs = &hrtf[0]
	}
	a.context = alcCreateContext(a.device, attrs)
	if a.context == 0 {
		alcCloseDevice(a.device)
		return a
	}
	alcMakeContextCurrent(a.context)
	if attrs != nil {
		var status int32
		alcGetIntegerv(a.device, alcHRTFSoft, 1, &status)
		a.HRTF = status != 0
	}
	alDistanceModel(alInverseDistClamped)
	a.available = true
	a.Listener([3]float64{}, 0)
	if efxAvailable && alcIsExtensionPresent(a.device, "ALC_EXT_EFX") {
		alGenAuxEffectSlots(1, &a.slot)
		alGenEffects(1, &a.effect)
		alEffecti(a.effect, alEffectType, alEffectReverb)
	}
	return a
}

// Sound é um som carregado. Toque com Play.
type Sound struct {
	audio    *Audio
	Name     string
	buffer   uint32
	Channels int
}

// Voice é um som tocando.
type Voice struct {
	audio  *Audio
	source uint32
	active bool
	volume float64
	fade   *fade
}

type fade struct {
	from, to, total, elapsed float64
	stop                     bool
}

// Load carrega um WAV (PCM 8/16 bits) ou OGG Vorbis, relativo à pasta do jogo.
func (a *Audio) Load(name, path string) (*Sound, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(a.baseDir, path)
	}
	s := &Sound{audio: a, Name: name, Channels: 1}
	a.sounds[name] = s
	if !a.available {
		return s, nil
	}
	var channels, width, rate int
	var data []byte
	var err error
	if strings.HasSuffix(strings.ToLower(path), ".ogg") {
		channels, rate, data, err = decodeOgg(path)
		width = 2
	} else {
		channels, width, rate, data, err = decodeWav(path)
	}
	if err != nil {
		return s, err
	}
	format := map[[2]int]int32{{1, 1}: alFormatMono8, {1, 2}: alFormatMono16,
		{2, 1}: alFormatStereo8, {2, 2}: alFormatStereo16}[[2]int{channels, width}]
	if format == 0 || len(data) == 0 {
		return s, fmt.Errorf("%s: formato não suportado", path)
	}
	alGenBuffers(1, &s.buffer)
	alBufferData(s.buffer, format, unsafe.Pointer(&data[0]), int32(len(data)), int32(rate))
	s.Channels = channels
	return s, nil
}

// Sound devolve um som já carregado pelo nome (ou nil).
func (a *Audio) Sound(name string) *Sound { return a.sounds[name] }

// PlayOption ajusta como um som toca.
type PlayOption func(*playConfig)

type playConfig struct {
	loop          bool
	volume, pitch float64
	pos           *[3]float64
	pan           *float64
	reverb        *bool
}

// Loop repete o som até Stop.
func Loop() PlayOption { return func(c *playConfig) { c.loop = true } }

// Volume de 0 a 1.
func Volume(v float64) PlayOption { return func(c *playConfig) { c.volume = v } }

// Pitch: velocidade/tom; 1 é normal.
func Pitch(p float64) PlayOption { return func(c *playConfig) { c.pitch = p } }

// Pos posiciona o som em 3D (metros).
func Pos(x, y, z float64) PlayOption {
	return func(c *playConfig) { p := [3]float64{x, y, z}; c.pos = &p }
}

// Pan de -1 (esquerda) a 1 (direita), a 1 m do ouvinte.
func Pan(p float64) PlayOption { return func(c *playConfig) { c.pan = &p } }

// Reverb liga ou desliga o ambiente para este som (padrão: só com Pos).
func Reverb(on bool) PlayOption { return func(c *playConfig) { c.reverb = &on } }

// Play toca o som e devolve a Voice que está tocando.
func (s *Sound) Play(opts ...PlayOption) *Voice {
	c := playConfig{volume: 1, pitch: 1}
	for _, o := range opts {
		o(&c)
	}
	if s == nil || s.audio == nil || !s.audio.available || s.buffer == 0 {
		return &Voice{}
	}
	a := s.audio
	src := a.source()
	alSourceStop(src)
	alSourcei(src, alBuffer, int32(s.buffer))
	alSourcei(src, alLooping, b2i(c.loop))
	alSourcef(src, alPitch, float32(c.pitch))
	alSourcef(src, alReferenceDistance, 1)
	alSourcef(src, alRolloffFactor, 1)
	alSourcef(src, alMaxDistance, 100)
	a.place(src, c.pos, c.pan)
	reverb := c.pos != nil
	if c.reverb != nil {
		reverb = *c.reverb
	}
	if a.slot != 0 {
		slot := int32(0)
		if reverb {
			slot = int32(a.slot)
		}
		alSource3i(src, alAuxiliarySendFilter, slot, 0, 0)
	}
	v := &Voice{audio: a, source: src, active: true}
	v.gain(c.volume)
	a.voices[src] = v
	a.order = append(a.order, src)
	alSourcePlay(src)
	return v
}

func b2i(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

func (a *Audio) source() uint32 {
	if n := len(a.free); n > 0 {
		src := a.free[n-1]
		a.free = a.free[:n-1]
		return src
	}
	if len(a.voices) < maxSources {
		var src uint32
		alGenSources(1, &src)
		return src
	}
	// sem fonte livre: reaproveita a primeira que terminou, senão a mais antiga
	victim := a.order[0]
	for _, src := range a.order {
		if !a.voices[src].Playing() {
			victim = src
			break
		}
	}
	alSourceStop(victim)
	a.release(victim)
	return victim
}

func (a *Audio) release(src uint32) {
	if v, ok := a.voices[src]; ok {
		v.active = false
		delete(a.voices, src)
	}
	for i, s := range a.order {
		if s == src {
			a.order = append(a.order[:i], a.order[i+1:]...)
			break
		}
	}
}

func (a *Audio) place(src uint32, pos *[3]float64, pan *float64) {
	switch {
	case pos != nil:
		alSourcei(src, alSourceRelative, 0)
		alSource3f(src, alPosition, float32(pos[0]), float32(pos[1]), float32(pos[2]))
	case pan != nil:
		p := math.Max(-1, math.Min(1, *pan))
		alSourcei(src, alSourceRelative, 1)
		alSource3f(src, alPosition, float32(p), 0, float32(-math.Sqrt(1-p*p)))
	default:
		alSourcei(src, alSourceRelative, 1)
		alSource3f(src, alPosition, 0, 0, 0)
	}
}

// Playing diz se o som ainda está tocando (ou pausado).
func (v *Voice) Playing() bool {
	if !v.active {
		return false
	}
	var state int32
	alGetSourcei(v.source, alSourceState, &state)
	return state == alPlaying || state == alPaused
}

func (v *Voice) gain(volume float64) {
	v.volume = volume
	alSourcef(v.source, alGain, float32(volume))
}

// SetVolume muda o volume; com fadeSec > 0, em transição.
func (v *Voice) SetVolume(volume, fadeSec float64) {
	if !v.active {
		return
	}
	if fadeSec > 0 {
		v.fade = &fade{from: v.volume, to: volume, total: fadeSec}
		return
	}
	v.fade = nil
	v.gain(volume)
}

// SetPitch muda velocidade/tom.
func (v *Voice) SetPitch(p float64) {
	if v.active {
		alSourcef(v.source, alPitch, float32(p))
	}
}

// SetPosition move o som em 3D.
func (v *Voice) SetPosition(x, y, z float64) {
	if v.active {
		p := [3]float64{x, y, z}
		v.audio.place(v.source, &p, nil)
	}
}

// SetPan move o som entre esquerda (-1) e direita (1).
func (v *Voice) SetPan(p float64) {
	if v.active {
		v.audio.place(v.source, nil, &p)
	}
}

// Pause pausa o som.
func (v *Voice) Pause() {
	if v.active {
		alSourcePause(v.source)
	}
}

// Resume retoma um som pausado.
func (v *Voice) Resume() {
	if !v.active {
		return
	}
	var state int32
	alGetSourcei(v.source, alSourceState, &state)
	if state == alPaused {
		alSourcePlay(v.source)
	}
}

// Stop para o som; com fadeSec > 0, abaixando aos poucos.
func (v *Voice) Stop(fadeSec float64) {
	if !v.active {
		return
	}
	if fadeSec > 0 {
		v.fade = &fade{from: v.volume, to: 0, total: fadeSec, stop: true}
		return
	}
	alSourceStop(v.source)
}

// Listener posiciona o ouvinte; facing em graus (0 = frente, 90 = direita).
func (a *Audio) Listener(pos [3]float64, facing float64) {
	if !a.available {
		return
	}
	alListener3f(alPosition, float32(pos[0]), float32(pos[1]), float32(pos[2]))
	r := facing * math.Pi / 180
	ori := []float32{float32(math.Sin(r)), 0, float32(-math.Cos(r)), 0, 1, 0}
	alListenerfv(alOrientation, &ori[0])
}

// SetReverb muda o ambiente para um preset de ReverbPresets, ou "" para desligar.
func (a *Audio) SetReverb(preset string) {
	if preset == "" {
		a.SetReverbParams(nil)
		return
	}
	a.SetReverbParams(ReverbPresets[preset])
}

// SetReverbParams aplica parâmetros soltos de reverberação; nil desliga.
func (a *Audio) SetReverbParams(params map[string]float64) {
	if !a.available || a.slot == 0 {
		return
	}
	if params == nil {
		alAuxiliaryEffectSloti(a.slot, alEffectSlotEffect, 0)
		return
	}
	for k, v := range params {
		if p, ok := reverbParams[k]; ok {
			alEffectf(a.effect, p, float32(v))
		}
	}
	// recarregar o efeito no slot aplica os parâmetros novos
	alAuxiliaryEffectSloti(a.slot, alEffectSlotEffect, int32(a.effect))
}

func (a *Audio) update(dt float64) {
	if !a.available {
		return
	}
	for _, src := range append([]uint32(nil), a.order...) {
		v := a.voices[src]
		if f := v.fade; f != nil {
			f.elapsed += dt
			t := math.Min(1, f.elapsed/f.total)
			v.gain(f.from + (f.to-f.from)*t)
			if t >= 1 {
				v.fade = nil
				if f.stop {
					alSourceStop(src)
				}
			}
		}
		if v.fade == nil && !v.Playing() {
			a.release(src)
			a.free = append(a.free, src)
		}
	}
}

// PauseAll pausa todos os sons (por exemplo, no menu de pausa).
func (a *Audio) PauseAll() {
	for _, v := range a.voices {
		v.Pause()
	}
}

// ResumeAll retoma os sons pausados.
func (a *Audio) ResumeAll() {
	for _, v := range a.voices {
		v.Resume()
	}
}

func (a *Audio) close() {
	if !a.available {
		return
	}
	alcMakeContextCurrent(0)
	alcDestroyContext(a.context)
	alcCloseDevice(a.device)
	a.available = false
}

// decodeWav lê um WAV PCM 8 ou 16 bits.
func decodeWav(path string) (channels, width, rate int, data []byte, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if len(raw) < 12 || string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		err = fmt.Errorf("%s: não é WAV", path)
		return
	}
	for i := 12; i+8 <= len(raw); {
		id := string(raw[i : i+4])
		size := int(binary.LittleEndian.Uint32(raw[i+4:]))
		body := raw[i+8 : min(len(raw), i+8+size)]
		switch id {
		case "fmt ":
			if binary.LittleEndian.Uint16(body[0:]) != 1 {
				err = fmt.Errorf("%s: WAV precisa ser PCM", path)
				return
			}
			channels = int(binary.LittleEndian.Uint16(body[2:]))
			rate = int(binary.LittleEndian.Uint32(body[4:]))
			width = int(binary.LittleEndian.Uint16(body[14:])) / 8
		case "data":
			data = body
		}
		i += 8 + size + size%2
	}
	if data == nil || channels == 0 {
		err = fmt.Errorf("%s: WAV incompleto", path)
	}
	return
}
