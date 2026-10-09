//go:build linux

package a11y

import (
	"sync"

	"github.com/ebitengine/purego"
)

var (
	sdlOnce sync.Once
	sdlErr  error
)

// loadSDL carrega a SDL2 e as extensões do sistema em tempo de execução (sem cgo).
func loadSDL() error {
	sdlOnce.Do(func() {
		libs := map[string]uintptr{}
		for _, name := range []string{"libSDL2-2.0.so.0", "libSDL2_image-2.0.so.0",
			"libSDL2_ttf-2.0.so.0", "libSDL2_gfx-1.0.so.0"} {
			h, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
			if err != nil {
				sdlErr = err
				return
			}
			libs[name] = h
		}
		sdl, img := libs["libSDL2-2.0.so.0"], libs["libSDL2_image-2.0.so.0"]
		ttf, gfx := libs["libSDL2_ttf-2.0.so.0"], libs["libSDL2_gfx-1.0.so.0"]
		for _, f := range []struct {
			lib  uintptr
			fn   interface{}
			name string
		}{
			{sdl, &sdlInit, "SDL_Init"},
			{sdl, &sdlCreateWindow, "SDL_CreateWindow"},
			{sdl, &sdlCreateRenderer, "SDL_CreateRenderer"},
			{sdl, &sdlRenderSetLogicalSize, "SDL_RenderSetLogicalSize"},
			{sdl, &sdlSetRenderDrawBlendMode, "SDL_SetRenderDrawBlendMode"},
			{sdl, &sdlSetRenderDrawColor, "SDL_SetRenderDrawColor"},
			{sdl, &sdlRenderClear, "SDL_RenderClear"},
			{sdl, &sdlRenderFillRect, "SDL_RenderFillRect"},
			{sdl, &sdlRenderDrawRect, "SDL_RenderDrawRect"},
			{sdl, &sdlRenderPresent, "SDL_RenderPresent"},
			{sdl, &sdlPumpEvents, "SDL_PumpEvents"},
			{sdl, &sdlShowCursor, "SDL_ShowCursor"},
			{sdl, &sdlCreateTextureFromSurf, "SDL_CreateTextureFromSurface"},
			{sdl, &sdlFreeSurface, "SDL_FreeSurface"},
			{sdl, &sdlFreeSurfaceP, "SDL_FreeSurface"},
			{sdl, &sdlQueryTexture, "SDL_QueryTexture"},
			{sdl, &sdlRenderCopyEx, "SDL_RenderCopyEx"},
			{sdl, &sdlSetTextureAlphaMod, "SDL_SetTextureAlphaMod"},
			{sdl, &sdlSetTextureColorMod, "SDL_SetTextureColorMod"},
			{sdl, &sdlSetTextureBlendMode, "SDL_SetTextureBlendMode"},
			{sdl, &sdlDestroyTexture, "SDL_DestroyTexture"},
			{sdl, &sdlDestroyRenderer, "SDL_DestroyRenderer"},
			{sdl, &sdlDestroyWindow, "SDL_DestroyWindow"},
			{sdl, &sdlQuitSubSystem, "SDL_QuitSubSystem"},
			{sdl, &sdlGetError, "SDL_GetError"},
			{sdl, &sdlCreateRGBSurfaceWithFmt, "SDL_CreateRGBSurfaceWithFormat"},
			{sdl, &sdlRenderReadPixels, "SDL_RenderReadPixels"},
			{sdl, &sdlGetRendererOutputSize, "SDL_GetRendererOutputSize"},
			{img, &imgInit, "IMG_Init"},
			{img, &imgLoad, "IMG_Load"},
			{img, &imgSavePNG, "IMG_SavePNG"},
			{ttf, &ttfInit, "TTF_Init"},
			{ttf, &ttfOpenFont, "TTF_OpenFont"},
			{ttf, &ttfRenderUTF8Blended, "TTF_RenderUTF8_Blended"},
			{ttf, &ttfSizeUTF8, "TTF_SizeUTF8"},
			{gfx, &gfxFilledCircle, "filledCircleRGBA"},
			{gfx, &gfxAACircle, "aacircleRGBA"},
			{gfx, &gfxThickLine, "thickLineRGBA"},
			{gfx, &gfxAALine, "aalineRGBA"},
			{gfx, &gfxFilledPolygon, "filledPolygonRGBA"},
			{gfx, &gfxAAPolygon, "aapolygonRGBA"},
			{gfx, &gfxPixel, "pixelRGBA"},
		} {
			purego.RegisterLibFunc(f.fn, f.lib, f.name)
		}
	})
	return sdlErr
}
