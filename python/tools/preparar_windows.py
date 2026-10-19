"""Baixa as DLLs que o r36s_a11y usa no Windows para a pasta r36s_a11y/dll.

    python tools\\preparar_windows.py                    # a biblioteca deste repositório
    python tools\\preparar_windows.py C:\\jogos\\meu-jogo  # a cópia dentro de um jogo

Tudo vem de lançamentos oficiais (GitHub da SDL e da OpenAL Soft, PyPI) e é
conferido pelo SHA-256 antes de ser extraído:

  SDL2.dll, SDL2_image.dll, SDL2_ttf.dll   janela, teclado, mouse, controles,
                                           imagens e texto (licença zlib)
  soft_oal.dll                             áudio 3D, OpenAL Soft (LGPL)
  SDL2_mixer.dll                           som num ouvido só, para o modo
                                           dueto (licença zlib)
  libsndfile_x64.dll                       OGG, do pacote soundfile (LGPL)
  nvdaControllerClient64.dll               fala pelo NVDA, do pacote
                                           accessible_output2 (LGPL)

Só biblioteca padrão; roda com o mesmo Python do jogo.
"""
import hashlib
import io
import os
import sys
import urllib.request
import zipfile

GH = "https://github.com"
PYPI = "https://files.pythonhosted.org/packages"

# (url, sha256, {arquivo dentro do zip: nome na pasta dll})
DOWNLOADS = [
    (GH + "/libsdl-org/SDL/releases/download/release-2.32.10/SDL2-2.32.10-win32-x64.zip",
     "6cf9706eefd0a4a06dc764007934d428afaf029fabdd408a9e646048c91e18fb",
     {"SDL2.dll": "SDL2.dll", "README-SDL.txt": "LICENCA-SDL2.txt"}),
    (GH + "/libsdl-org/SDL_image/releases/download/release-2.8.8/SDL2_image-2.8.8-win32-x64.zip",
     "740d6c7fe40087db01e0d7bd636e3eaebe4bffbde58b595bf0c3b7c359a43033",
     {"SDL2_image.dll": "SDL2_image.dll"}),
    (GH + "/libsdl-org/SDL_ttf/releases/download/release-2.24.0/SDL2_ttf-2.24.0-win32-x64.zip",
     "8d72240aafc783d37d6cf960b1a2138e6ca4eb0c1b5b9a9e71ea299d1ef27ef9",
     {"SDL2_ttf.dll": "SDL2_ttf.dll"}),
    (GH + "/libsdl-org/SDL_mixer/releases/download/release-2.8.1/SDL2_mixer-2.8.1-win32-x64.zip",
     "ddd4454f9157a01259ac0fd6d9a99663305a358f19f828867064a2186b0f198f",
     {"SDL2_mixer.dll": "SDL2_mixer.dll", "README.txt": "LICENCA-SDL2_mixer.txt"}),
    (GH + "/kcat/openal-soft/releases/download/1.24.3/openal-soft-1.24.3-bin.zip",
     "03c8c0c0bcdba9d3fb54b0d3c2f9a565f81b3e2a19a538dd394f9b8cb6caaa22",
     {"openal-soft-1.24.3-bin/bin/Win64/soft_oal.dll": "soft_oal.dll",
      "openal-soft-1.24.3-bin/COPYING": "LICENCA-OpenAL-Soft.txt"}),
    (PYPI + "/ed/97/b39c18ac1df45e755ca22b8b00e872929da5d107998a207a5e4ac831bfda/"
     "soundfile-0.14.0-py2.py3-none-win_amd64.whl",
     "299491d3499460fb1b74bb4bd78b57ffc2d243a5fafa7b6ec1b264875c78453e",
     {"_soundfile_data/libsndfile_x64.dll": "libsndfile_x64.dll",
      "_soundfile_data/COPYING": "LICENCA-libsndfile.txt"}),
    (PYPI + "/02/8e/2bb81f845d8f3ecb07acd094544ecbf0c0e02f80364e35fdabaec96ee6dc/"
     "accessible_output2-0.17-py2.py3-none-any.whl",
     "bdcfc5f2c8ecb50b219fa7d547018dd0c99dfa4c56b506872c74f5dc941a3288",
     {"accessible_output2/lib/nvdaControllerClient64.dll": "nvdaControllerClient64.dll"}),
]


def target_dir(arg):
    """A pasta dll da biblioteca: a deste repositório ou a de um jogo."""
    here = os.path.dirname(os.path.abspath(__file__))
    base = os.path.abspath(arg) if arg else os.path.join(here, "..")
    if os.path.basename(os.path.normpath(base)) != "r36s_a11y":
        base = os.path.join(base, "r36s_a11y")
    if not os.path.isfile(os.path.join(base, "__init__.py")):
        sys.exit("não achei a biblioteca r36s_a11y em %s" % base)
    return os.path.normpath(os.path.join(base, "dll"))


def fetch(url, sha256):
    with urllib.request.urlopen(url, timeout=120) as response:
        data = response.read()
    digest = hashlib.sha256(data).hexdigest()
    if digest != sha256:
        raise RuntimeError("SHA-256 diferente do esperado para %s" % url)
    return data


def main(argv):
    dest = target_dir(argv[1] if len(argv) > 1 else None)
    os.makedirs(dest, exist_ok=True)
    for url, sha256, files in DOWNLOADS:
        name = url.rsplit("/", 1)[1]
        if all(os.path.exists(os.path.join(dest, out)) for out in files.values()):
            print("já tem: %s" % ", ".join(files.values()))
            continue
        print("baixando %s..." % name)
        with zipfile.ZipFile(io.BytesIO(fetch(url, sha256))) as archive:
            for inner, out in files.items():
                with open(os.path.join(dest, out), "wb") as f:
                    f.write(archive.read(inner))
    print("Pronto: DLLs em %s" % dest)


if __name__ == "__main__":
    main(sys.argv)
