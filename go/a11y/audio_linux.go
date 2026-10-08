//go:build linux

package a11y

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	loadOnce sync.Once
	loadErr  error

	ovFopen func(path string, vf unsafe.Pointer) int32
	ovInfo  func(vf unsafe.Pointer, link int32) *vorbisInfo
	ovRead  func(vf, buf unsafe.Pointer, length, bigEndian, word, signed int32, bitstream *int32) int64
	ovClear func(vf unsafe.Pointer) int32
)

// loadOpenAL carrega a OpenAL Soft do sistema em tempo de execução (sem cgo).
func loadOpenAL() error {
	loadOnce.Do(func() {
		lib, err := purego.Dlopen("libopenal.so.1", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = err
			return
		}
		reg := func(fn interface{}, name string) {
			purego.RegisterLibFunc(fn, lib, name)
		}
		reg(&alcOpenDevice, "alcOpenDevice")
		reg(&alcCreateContext, "alcCreateContext")
		reg(&alcMakeContextCurrent, "alcMakeContextCurrent")
		reg(&alcIsExtensionPresent, "alcIsExtensionPresent")
		reg(&alcGetIntegerv, "alcGetIntegerv")
		reg(&alcDestroyContext, "alcDestroyContext")
		reg(&alcCloseDevice, "alcCloseDevice")
		reg(&alDistanceModel, "alDistanceModel")
		reg(&alGenBuffers, "alGenBuffers")
		reg(&alBufferData, "alBufferData")
		reg(&alGenSources, "alGenSources")
		reg(&alSourcei, "alSourcei")
		reg(&alSourcef, "alSourcef")
		reg(&alSource3f, "alSource3f")
		reg(&alSource3i, "alSource3i")
		reg(&alGetSourcei, "alGetSourcei")
		reg(&alSourcePlay, "alSourcePlay")
		reg(&alSourceStop, "alSourceStop")
		reg(&alSourcePause, "alSourcePause")
		reg(&alListener3f, "alListener3f")
		reg(&alListenerfv, "alListenerfv")
		// EFX: a OpenAL Soft exporta direto; sem elas, tudo funciona sem reverberação.
		efxAvailable = true
		for _, name := range []string{"alGenAuxiliaryEffectSlots", "alGenEffects", "alEffecti",
			"alEffectf", "alAuxiliaryEffectSloti"} {
			if _, err := purego.Dlsym(lib, name); err != nil {
				efxAvailable = false
			}
		}
		if efxAvailable {
			reg(&alGenAuxEffectSlots, "alGenAuxiliaryEffectSlots")
			reg(&alGenEffects, "alGenEffects")
			reg(&alEffecti, "alEffecti")
			reg(&alEffectf, "alEffectf")
			reg(&alAuxiliaryEffectSloti, "alAuxiliaryEffectSloti")
		}

		if vorbis, err := purego.Dlopen("libvorbisfile.so.3", purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
			purego.RegisterLibFunc(&ovFopen, vorbis, "ov_fopen")
			purego.RegisterLibFunc(&ovInfo, vorbis, "ov_info")
			purego.RegisterLibFunc(&ovRead, vorbis, "ov_read")
			purego.RegisterLibFunc(&ovClear, vorbis, "ov_clear")
		}
	})
	return loadErr
}

type vorbisInfo struct {
	version, channels int32
	rate              int64
}

// decodeOgg decodifica um OGG Vorbis inteiro para PCM 16 bits.
func decodeOgg(path string) (channels, rate int, data []byte, err error) {
	if ovFopen == nil {
		err = fmt.Errorf("libvorbisfile não encontrada")
		return
	}
	vf := make([]byte, 8192) // maior que OggVorbis_File em qualquer arquitetura
	vfp := unsafe.Pointer(&vf[0])
	if ovFopen(path, vfp) != 0 {
		err = fmt.Errorf("%s: não é um OGG Vorbis válido", path)
		return
	}
	defer ovClear(vfp)
	info := ovInfo(vfp, -1)
	channels, rate = int(info.channels), int(info.rate)
	chunk := make([]byte, 65536)
	var section int32
	for {
		n := ovRead(vfp, unsafe.Pointer(&chunk[0]), int32(len(chunk)), 0, 2, 1, &section)
		if n <= 0 {
			break
		}
		data = append(data, chunk[:n]...)
	}
	return
}
