"""Gráficos 2D opcionais: formas, sprites e texto, pela SDL2 do sistema.

A tela é aberta só se o jogo registrar @game.on_draw. A área de desenho é
sempre 640x480 (escalada para a tela real); (0, 0) é o canto superior
esquerdo. Cores são tuplas (r, g, b) ou (r, g, b, a) de 0 a 255, ou "#rrggbb".

Os gráficos são um extra: o jogo continua tendo de ser jogável só pelo som.
"""
import ctypes
import ctypes.util
import os

WIDTH, HEIGHT = 640, 480
# Fonte padrão: a DejaVu Sans do console; no PC, a primeira que existir.
DEFAULT_FONTS = [
    "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
    "C:/Windows/Fonts/segoeui.ttf",
    "C:/Windows/Fonts/arial.ttf",
    "/System/Library/Fonts/Supplemental/Arial.ttf",
]
DEFAULT_FONT = next((f for f in DEFAULT_FONTS if os.path.exists(f)), DEFAULT_FONTS[0])

SDL_INIT_VIDEO = 0x20
SDL_WINDOW_SHOWN = 0x4
SDL_WINDOW_FULLSCREEN_DESKTOP = 0x1001
SDL_RENDERER_ACCELERATED = 0x2
SDL_BLENDMODE_BLEND = 0x1
SDL_FLIP_HORIZONTAL, SDL_FLIP_VERTICAL = 0x1, 0x2
IMG_INIT_PNG = 0x2


class _Rect(ctypes.Structure):
    _fields_ = [("x", ctypes.c_int), ("y", ctypes.c_int), ("w", ctypes.c_int), ("h", ctypes.c_int)]


class _Color(ctypes.Structure):
    _fields_ = [("r", ctypes.c_uint8), ("g", ctypes.c_uint8), ("b", ctypes.c_uint8), ("a", ctypes.c_uint8)]


def _load(names):
    for name in names:
        for candidate in (name, ctypes.util.find_library(name)):
            if candidate:
                try:
                    return ctypes.CDLL(candidate, mode=ctypes.RTLD_GLOBAL)
                except OSError:
                    pass
    raise OSError("biblioteca não encontrada: %s" % ", ".join(names))


def _color(c):
    if isinstance(c, str):
        c = c.lstrip("#")
        return (int(c[0:2], 16), int(c[2:4], 16), int(c[4:6], 16), int(c[6:8], 16) if len(c) >= 8 else 255)
    if len(c) == 3:
        return (int(c[0]), int(c[1]), int(c[2]), 255)
    return tuple(int(v) for v in c[:4])


def _signatures(sdl, img, ttf, gfx):
    vp, ci, cd, u8, i16 = ctypes.c_void_p, ctypes.c_int, ctypes.c_double, ctypes.c_uint8, ctypes.c_int16
    rp = ctypes.POINTER(_Rect)
    for name, res, args in [
        ("SDL_Init", ci, [ctypes.c_uint32]),
        ("SDL_CreateWindow", vp, [ctypes.c_char_p, ci, ci, ci, ci, ctypes.c_uint32]),
        ("SDL_CreateRenderer", vp, [vp, ci, ctypes.c_uint32]),
        ("SDL_RenderSetLogicalSize", ci, [vp, ci, ci]),
        ("SDL_SetRenderDrawBlendMode", ci, [vp, ci]),
        ("SDL_SetRenderDrawColor", ci, [vp, u8, u8, u8, u8]),
        ("SDL_RenderClear", ci, [vp]),
        ("SDL_RenderFillRect", ci, [vp, rp]),
        ("SDL_RenderDrawRect", ci, [vp, rp]),
        ("SDL_RenderPresent", None, [vp]),
        ("SDL_PumpEvents", None, []),
        ("SDL_ShowCursor", ci, [ci]),
        ("SDL_CreateTextureFromSurface", vp, [vp, vp]),
        ("SDL_FreeSurface", None, [vp]),
        ("SDL_QueryTexture", ci, [vp, ctypes.POINTER(ctypes.c_uint32), ctypes.POINTER(ci),
                                  ctypes.POINTER(ci), ctypes.POINTER(ci)]),
        ("SDL_RenderCopyEx", ci, [vp, vp, rp, rp, cd, vp, ci]),
        ("SDL_SetTextureAlphaMod", ci, [vp, u8]),
        ("SDL_SetTextureColorMod", ci, [vp, u8, u8, u8]),
        ("SDL_SetTextureBlendMode", ci, [vp, ci]),
        ("SDL_DestroyTexture", None, [vp]),
        ("SDL_DestroyRenderer", None, [vp]),
        ("SDL_DestroyWindow", None, [vp]),
        ("SDL_QuitSubSystem", None, [ctypes.c_uint32]),
        ("SDL_GetError", ctypes.c_char_p, []),
        ("SDL_CreateRGBSurfaceWithFormat", vp, [ctypes.c_uint32, ci, ci, ci, ctypes.c_uint32]),
        ("SDL_RenderReadPixels", ci, [vp, rp, ctypes.c_uint32, vp, ci]),
        ("SDL_GetRendererOutputSize", ci, [vp, ctypes.POINTER(ci), ctypes.POINTER(ci)]),
    ]:
        f = getattr(sdl, name)
        f.restype, f.argtypes = res, args
    img.IMG_Init.restype, img.IMG_Init.argtypes = ci, [ci]
    img.IMG_Load.restype, img.IMG_Load.argtypes = vp, [ctypes.c_char_p]
    img.IMG_SavePNG.restype, img.IMG_SavePNG.argtypes = ci, [vp, ctypes.c_char_p]
    ttf.TTF_Init.restype, ttf.TTF_Init.argtypes = ci, []
    ttf.TTF_OpenFont.restype, ttf.TTF_OpenFont.argtypes = vp, [ctypes.c_char_p, ci]
    ttf.TTF_CloseFont.restype, ttf.TTF_CloseFont.argtypes = None, [vp]
    ttf.TTF_RenderUTF8_Blended.restype = vp
    ttf.TTF_RenderUTF8_Blended.argtypes = [vp, ctypes.c_char_p, _Color]
    ttf.TTF_SizeUTF8.restype = ci
    ttf.TTF_SizeUTF8.argtypes = [vp, ctypes.c_char_p, ctypes.POINTER(ci), ctypes.POINTER(ci)]
    gfx.filledCircleRGBA.argtypes = [vp, i16, i16, i16, u8, u8, u8, u8]
    gfx.aacircleRGBA.argtypes = [vp, i16, i16, i16, u8, u8, u8, u8]
    gfx.thickLineRGBA.argtypes = [vp, i16, i16, i16, i16, u8, u8, u8, u8, u8]
    gfx.aalineRGBA.argtypes = [vp, i16, i16, i16, i16, u8, u8, u8, u8]
    gfx.filledPolygonRGBA.argtypes = [vp, ctypes.POINTER(i16), ctypes.POINTER(i16), ci, u8, u8, u8, u8]
    gfx.aapolygonRGBA.argtypes = [vp, ctypes.POINTER(i16), ctypes.POINTER(i16), ci, u8, u8, u8, u8]
    gfx.pixelRGBA.argtypes = [vp, i16, i16, u8, u8, u8, u8]


class Image:
    """Uma imagem (textura). Desenhe com Graphics.draw."""

    def __init__(self, texture, width, height):
        self.texture = texture
        self.width = width
        self.height = height


class Font:
    """Uma fonte TTF num tamanho."""

    def __init__(self, handle, size):
        self.handle = handle
        self.size = size


class Graphics:
    def __init__(self, title="Jogo", base_dir="."):
        self.title = title
        self.base_dir = base_dir
        self.available = False
        self.window = self.renderer = None
        self._color = (255, 255, 255, 255)
        self._text_cache = {}   # (fonte, texto, cor) -> [Image, quadros sem uso]
        self._default_font = None

    def open(self):
        """Abre a tela. Chamado pelo Game quando há @game.on_draw."""
        try:
            self.sdl = _load(["libSDL2-2.0.so.0", "SDL2", "SDL2.dll"])
            self.img = _load(["libSDL2_image-2.0.so.0", "SDL2_image", "SDL2_image.dll"])
            self.ttf = _load(["libSDL2_ttf-2.0.so.0", "SDL2_ttf", "SDL2_ttf.dll"])
            self.gfx = _load(["libSDL2_gfx-1.0.so.0", "SDL2_gfx", "SDL2_gfx.dll"])
        except OSError as exc:
            print("[r36s_a11y] sem gráficos: %s" % exc)
            return False
        _signatures(self.sdl, self.img, self.ttf, self.gfx)
        sdl = self.sdl
        if sdl.SDL_Init(SDL_INIT_VIDEO) != 0:
            print("[r36s_a11y] sem gráficos: %s" % sdl.SDL_GetError().decode())
            return False
        on_console = os.path.exists("/opt/a11y")
        flags = SDL_WINDOW_FULLSCREEN_DESKTOP if on_console else SDL_WINDOW_SHOWN
        self.window = sdl.SDL_CreateWindow(self.title.encode(), 0x2FFF0000, 0x2FFF0000,
                                           WIDTH, HEIGHT, flags)
        if not self.window:
            print("[r36s_a11y] sem gráficos: %s" % sdl.SDL_GetError().decode())
            return False
        self.renderer = sdl.SDL_CreateRenderer(self.window, -1, SDL_RENDERER_ACCELERATED)
        if not self.renderer:  # sem GPU (testes): renderizador por software
            self.renderer = sdl.SDL_CreateRenderer(self.window, -1, 0)
        if not self.renderer:
            print("[r36s_a11y] sem gráficos: %s" % sdl.SDL_GetError().decode())
            return False
        sdl.SDL_RenderSetLogicalSize(self.renderer, WIDTH, HEIGHT)
        sdl.SDL_SetRenderDrawBlendMode(self.renderer, SDL_BLENDMODE_BLEND)
        sdl.SDL_ShowCursor(0)
        self.img.IMG_Init(IMG_INIT_PNG)
        self.ttf.TTF_Init()
        self.available = True
        return True

    # --- quadro ----------------------------------------------------------
    def begin(self):
        self.sdl.SDL_PumpEvents()  # a SDL precisa disso para manter a tela viva
        self.clear((0, 0, 0))
        self._color = (255, 255, 255, 255)

    def end(self):
        self.sdl.SDL_RenderPresent(self.renderer)
        # textos que não foram desenhados por 60 quadros saem do cache
        for key, entry in list(self._text_cache.items()):
            entry[1] += 1
            if entry[1] > 60:
                self.sdl.SDL_DestroyTexture(entry[0].texture)
                del self._text_cache[key]

    # --- cor e formas --------------------------------------------------
    def set_color(self, color):
        """Cor padrão dos próximos desenhos."""
        self._color = _color(color)

    def _c(self, color):
        return self._color if color is None else _color(color)

    def clear(self, color=(0, 0, 0)):
        """Pinta a tela inteira."""
        r, g, b, a = _color(color)
        self.sdl.SDL_SetRenderDrawColor(self.renderer, r, g, b, a)
        self.sdl.SDL_RenderClear(self.renderer)

    def rect(self, x, y, w, h, color=None, fill=True):
        r, g, b, a = self._c(color)
        self.sdl.SDL_SetRenderDrawColor(self.renderer, r, g, b, a)
        rect = _Rect(int(x), int(y), int(w), int(h))
        if fill:
            self.sdl.SDL_RenderFillRect(self.renderer, ctypes.byref(rect))
        else:
            self.sdl.SDL_RenderDrawRect(self.renderer, ctypes.byref(rect))

    def line(self, x1, y1, x2, y2, color=None, width=1):
        r, g, b, a = self._c(color)
        if width <= 1:
            self.gfx.aalineRGBA(self.renderer, int(x1), int(y1), int(x2), int(y2), r, g, b, a)
        else:
            self.gfx.thickLineRGBA(self.renderer, int(x1), int(y1), int(x2), int(y2), int(width), r, g, b, a)

    def circle(self, x, y, radius, color=None, fill=True):
        r, g, b, a = self._c(color)
        f = self.gfx.filledCircleRGBA if fill else self.gfx.aacircleRGBA
        f(self.renderer, int(x), int(y), int(radius), r, g, b, a)

    def polygon(self, points, color=None, fill=True):
        """points: [(x, y), ...] com 3 ou mais pontos."""
        r, g, b, a = self._c(color)
        n = len(points)
        vx = (ctypes.c_int16 * n)(*[int(p[0]) for p in points])
        vy = (ctypes.c_int16 * n)(*[int(p[1]) for p in points])
        f = self.gfx.filledPolygonRGBA if fill else self.gfx.aapolygonRGBA
        f(self.renderer, vx, vy, n, r, g, b, a)

    def point(self, x, y, color=None):
        r, g, b, a = self._c(color)
        self.gfx.pixelRGBA(self.renderer, int(x), int(y), r, g, b, a)

    def _path(self, path):
        return path if os.path.isabs(path) else os.path.join(self.base_dir, path)

    def screenshot(self, path):
        """Salva o quadro atual em PNG (chame no fim de on_draw). Útil para testar."""
        if not self.available:
            return
        w, h = ctypes.c_int(0), ctypes.c_int(0)
        self.sdl.SDL_GetRendererOutputSize(self.renderer, ctypes.byref(w), ctypes.byref(h))
        fmt = 0x16762004  # SDL_PIXELFORMAT_ARGB8888
        surface = self.sdl.SDL_CreateRGBSurfaceWithFormat(0, w.value, h.value, 32, fmt)
        pixels = ctypes.cast(surface + 32, ctypes.POINTER(ctypes.c_void_p))[0]  # SDL_Surface.pixels
        pitch = ctypes.cast(surface + 24, ctypes.POINTER(ctypes.c_int))[0]       # SDL_Surface.pitch
        self.sdl.SDL_RenderReadPixels(self.renderer, None, fmt, pixels, pitch)
        self.img.IMG_SavePNG(surface, self._path(path).encode())
        self.sdl.SDL_FreeSurface(surface)

    # --- imagens --------------------------------------------------------
    def image(self, path):
        """Carrega um PNG (ou outro formato da SDL2_image), relativo à pasta do jogo."""
        if not self.available:
            return Image(None, 0, 0)
        surface = self.img.IMG_Load(self._path(path).encode())
        if not surface:
            raise IOError("%s: %s" % (path, self.sdl.SDL_GetError().decode()))
        texture = self.sdl.SDL_CreateTextureFromSurface(self.renderer, surface)
        self.sdl.SDL_FreeSurface(surface)
        return self._wrap(texture)

    def _wrap(self, texture):
        w, h = ctypes.c_int(0), ctypes.c_int(0)
        self.sdl.SDL_QueryTexture(texture, None, None, ctypes.byref(w), ctypes.byref(h))
        self.sdl.SDL_SetTextureBlendMode(texture, SDL_BLENDMODE_BLEND)
        return Image(texture, w.value, h.value)

    def draw(self, image, x, y, src=None, scale=1.0, rotation=0.0,
             flip_x=False, flip_y=False, alpha=255, tint=None):
        """Desenha uma imagem com o canto superior esquerdo em (x, y).

        src=(x, y, w, h) recorta um quadro de uma folha de sprites. rotation em
        graus, em torno do centro. tint=(r, g, b) multiplica as cores.
        """
        if not self.available or image.texture is None:
            return
        sx, sy, sw, sh = src if src else (0, 0, image.width, image.height)
        srect = _Rect(int(sx), int(sy), int(sw), int(sh))
        drect = _Rect(int(x), int(y), int(sw * scale), int(sh * scale))
        tr, tg, tb = _color(tint)[:3] if tint else (255, 255, 255)
        self.sdl.SDL_SetTextureColorMod(image.texture, tr, tg, tb)
        self.sdl.SDL_SetTextureAlphaMod(image.texture, int(alpha))
        flip = (SDL_FLIP_HORIZONTAL if flip_x else 0) | (SDL_FLIP_VERTICAL if flip_y else 0)
        self.sdl.SDL_RenderCopyEx(self.renderer, image.texture, ctypes.byref(srect),
                                  ctypes.byref(drect), float(rotation), None, flip)

    # --- texto ----------------------------------------------------------
    def font(self, path=None, size=24):
        """Carrega uma fonte TTF; sem path, usa a DejaVu Sans do sistema."""
        if not self.available:
            return Font(None, size)
        handle = self.ttf.TTF_OpenFont(self._path(path or DEFAULT_FONT).encode(), int(size))
        if not handle:
            raise IOError("%s: %s" % (path or DEFAULT_FONT, self.sdl.SDL_GetError().decode()))
        return Font(handle, size)

    def _font(self, font):
        if font is not None:
            return font
        if self._default_font is None:
            self._default_font = self.font()
        return self._default_font

    def measure(self, text, font=None):
        """Tamanho (largura, altura) do texto em pixels."""
        font = self._font(font)
        if font.handle is None or not text:
            return (0, 0)
        w, h = ctypes.c_int(0), ctypes.c_int(0)
        self.ttf.TTF_SizeUTF8(font.handle, text.encode("utf-8"), ctypes.byref(w), ctypes.byref(h))
        return (w.value, h.value)

    def text(self, text, x, y, font=None, color=None, align="left"):
        """Escreve o texto com o topo em y; align: "left", "center" ou "right"."""
        font = self._font(font)
        if not self.available or font.handle is None or not text:
            return
        c = self._c(color)
        key = (id(font), text, c)
        entry = self._text_cache.get(key)
        if entry is None:
            surface = self.ttf.TTF_RenderUTF8_Blended(font.handle, text.encode("utf-8"), _Color(*c))
            if not surface:
                return
            texture = self.sdl.SDL_CreateTextureFromSurface(self.renderer, surface)
            self.sdl.SDL_FreeSurface(surface)
            entry = self._text_cache[key] = [self._wrap(texture), 0]
        entry[1] = 0
        image = entry[0]
        if align == "center":
            x -= image.width / 2
        elif align == "right":
            x -= image.width
        self.draw(image, x, y)

    def close(self):
        if not self.available:
            return
        for image, _ in self._text_cache.values():
            self.sdl.SDL_DestroyTexture(image.texture)
        self._text_cache.clear()
        self.sdl.SDL_DestroyRenderer(self.renderer)
        self.sdl.SDL_DestroyWindow(self.window)
        self.sdl.SDL_QuitSubSystem(SDL_INIT_VIDEO)
        self.available = False
