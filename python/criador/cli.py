"""Linha de comando: novo, rodar, conferir, atualizar e exportar."""
import argparse
import os
import subprocess
import sys

from . import baixar, exportar, projeto


def cmd_novo(args):
    projeto.create(args.pasta, name=args.nome, author=args.autor)
    print("Para jogar: python %s rodar %s" % (_self(), args.pasta))
    return 0


def cmd_rodar(args):
    path = os.path.abspath(args.pasta)
    manifest = projeto.load(path)
    if manifest["runtime"] != "python3":
        raise projeto.ProjectError("o rodar ainda é só para jogos em Python")
    if sys.platform == "win32":
        dll = os.path.join(path, "r36s_a11y", "dll")
        if not baixar.has_dlls(dll):
            print("Primeira vez no Windows: baixando as DLLs do SDK para o jogo.")
            baixar.install_dlls(dll)
    env = dict(os.environ, A11Y_GAME_DIR=path)
    if args.pc:
        env["A11Y_PLATAFORMA"] = "pc"
    return subprocess.call([sys.executable, manifest["exec"]], cwd=path, env=env)


def cmd_conferir(args):
    path = os.path.abspath(args.pasta)
    manifest, problems = projeto.check(path)
    if not problems:
        try:
            files = projeto.package_files(path, manifest)
        except projeto.ProjectError as e:
            problems = str(e).splitlines()
    if problems:
        for p in problems:
            print("- " + p)
        return 1
    print("Tudo certo: %s %s, %d arquivos para exportar." % (
        manifest["name"], manifest["version"], len(files)))
    return 0


def cmd_atualizar(args):
    projeto.update_sdk(os.path.abspath(args.pasta))
    return 0


def cmd_exportar(args):
    targets = list(exportar.TARGETS) if args.para == "todos" else [args.para]
    for target in targets:
        exportar.TARGETS[target](args.pasta)
    return 0


def _self():
    return os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "a11y.py")


def parser():
    p = argparse.ArgumentParser(
        prog="a11y.py", description="Cria, roda e exporta jogos feitos com o r36s_a11y.")
    sub = p.add_subparsers(dest="comando", metavar="comando")
    sub.required = True

    s = sub.add_parser("novo", help="cria um projeto a partir do modelo")
    s.add_argument("pasta", help="pasta do jogo novo (o nome dela vira o id)")
    s.add_argument("--nome", help="nome falado do jogo; padrão: o da pasta")
    s.add_argument("--autor", help="quem fez o jogo")
    s.set_defaults(fn=cmd_novo)

    s = sub.add_parser("rodar", help="abre o jogo")
    s.add_argument("pasta", nargs="?", default=".")
    s.add_argument("--pc", action="store_true",
                   help="no Linux, abre a janela do PC em vez de ler o controle do R36S")
    s.set_defaults(fn=cmd_rodar)

    s = sub.add_parser("conferir", help="confere o game.json e os nomes dos arquivos")
    s.add_argument("pasta", nargs="?", default=".")
    s.set_defaults(fn=cmd_conferir)

    s = sub.add_parser("atualizar", help="troca a cópia do r36s_a11y do jogo pela deste SDK")
    s.add_argument("pasta", nargs="?", default=".")
    s.set_defaults(fn=cmd_atualizar)

    s = sub.add_parser("exportar", help="gera o jogo pronto para o R36S e o Windows em dist/")
    s.add_argument("pasta", nargs="?", default=".")
    s.add_argument("--para", choices=["todos"] + list(exportar.TARGETS), default="todos")
    s.set_defaults(fn=cmd_exportar)
    return p


def main(argv):
    args = parser().parse_args(argv)
    try:
        return args.fn(args)
    except projeto.ProjectError as e:
        print("Não deu: %s" % e, file=sys.stderr)
        return 2
