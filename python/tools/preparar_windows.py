"""Baixa as DLLs que o r36s_a11y usa no Windows para a pasta r36s_a11y/dll.

    python tools\\preparar_windows.py                    # a biblioteca deste repositório
    python tools\\preparar_windows.py C:\\jogos\\meu-jogo  # a cópia dentro de um jogo

Tudo vem de lançamentos oficiais (GitHub da SDL e da OpenAL Soft, PyPI) e é
conferido pelo SHA-256 antes de ser extraído:

  SDL2.dll, SDL2_image.dll, SDL2_ttf.dll   janela, teclado, mouse, controles,
                                           imagens e texto (licença zlib)
  soft_oal.dll                             áudio 3D, OpenAL Soft (LGPL)
  libsndfile_x64.dll                       OGG, do pacote soundfile (LGPL)
  nvdaControllerClient64.dll               fala pelo NVDA, do pacote
                                           accessible_output2 (LGPL)

A lista e os downloads ficam em criador/baixar.py. Com o a11y.py, "run"
faz isto sozinho quando falta alguma DLL.
"""
import os
import sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))

from criador import baixar  # noqa: E402


def target_dir(arg):
    """A pasta dll da biblioteca: a deste repositório ou a de um jogo."""
    here = os.path.dirname(os.path.abspath(__file__))
    base = os.path.abspath(arg) if arg else os.path.join(here, "..")
    if os.path.basename(os.path.normpath(base)) != "r36s_a11y":
        base = os.path.join(base, "r36s_a11y")
    if not os.path.isfile(os.path.join(base, "__init__.py")):
        sys.exit("não achei a biblioteca r36s_a11y em %s" % base)
    return os.path.normpath(os.path.join(base, "dll"))


def main(argv):
    dest = target_dir(argv[1] if len(argv) > 1 else None)
    baixar.install_dlls(dest)
    print("Pronto: DLLs em %s" % dest)


if __name__ == "__main__":
    main(sys.argv)
