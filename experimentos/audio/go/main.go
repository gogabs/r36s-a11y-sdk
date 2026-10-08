package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ebitengine/purego"
)

func main() {
	sdl, err := purego.Dlopen("libSDL2-2.0.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil { fmt.Println("sdl:", err); os.Exit(1) }
	mix, err := purego.Dlopen("libSDL2_mixer-2.0.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil { fmt.Println("mixer:", err); os.Exit(1) }
	var sdlInit func(uint32) int32
	var openAudio func(int32, uint16, int32, int32) int32
	var loadWAV func(string) uintptr
	var playChannel func(int32, uintptr, int32, int32) int32
	var setPosition func(int32, int16, uint8) int32
	var rwFromFile func(string, string) uintptr
	var loadWAVRW func(uintptr, int32) uintptr
	purego.RegisterLibFunc(&sdlInit, sdl, "SDL_Init")
	purego.RegisterLibFunc(&rwFromFile, sdl, "SDL_RWFromFile")
	purego.RegisterLibFunc(&openAudio, mix, "Mix_OpenAudio")
	purego.RegisterLibFunc(&loadWAVRW, mix, "Mix_LoadWAV_RW")
	purego.RegisterLibFunc(&playChannel, mix, "Mix_PlayChannelTimed")
	purego.RegisterLibFunc(&setPosition, mix, "Mix_SetPosition")
	_ = loadWAV
	sdlInit(0x10) // SDL_INIT_AUDIO
	if openAudio(44100, 0x8010, 2, 512) != 0 { fmt.Println("open audio falhou"); os.Exit(1) }
	chunk := loadWAVRW(rwFromFile(os.Args[1], "rb"), 1)
	if chunk == 0 { fmt.Println("load falhou"); os.Exit(1) }
	for _, ang := range []int16{270, 0, 90} { // esquerda, frente, direita
		ch := playChannel(-1, chunk, 0, -1)
		setPosition(ch, ang, 0)
		time.Sleep(700 * time.Millisecond)
	}
	fmt.Println("ok")
}
