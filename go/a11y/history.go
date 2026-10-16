package a11y

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Histórico de falas do sistema (/opt/a11y/log/historico-falas.txt).
//
// Cada fala do jogo vira uma linha no fim do arquivo; o a11yd deixa o jogador
// voltar e avançar nas últimas 50 com Fn + L1 e Fn + R1. Formato comum a todas
// as bibliotecas-ponte: uma fala por linha, em UTF-8, com as quebras de linha
// trocadas por espaço, só acrescentando no fim. Quem vê o arquivo passar de
// historyTrimBytes o reescreve com as últimas historyMaxEntries linhas. Sem a
// pasta do arquivo (no PC, por exemplo), nada é gravado.
const (
	historyMaxEntries = 50
	historyTrimBytes  = 16384
)

func historyPath() string {
	if p := os.Getenv("A11Y_HISTORICO_FALAS"); p != "" {
		return p
	}
	return "/opt/a11y/log/historico-falas.txt"
}

func recordSpeech(path, text string) {
	line := strings.Join(strings.Fields(text), " ")
	if line == "" {
		return
	}
	if st, err := os.Stat(filepath.Dir(path)); err != nil || !st.IsDir() {
		return
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return
	}
	_, err = f.WriteString(line + "\n")
	st, statErr := f.Stat()
	f.Close()
	if err != nil || statErr != nil || st.Size() <= historyTrimBytes {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var lines []string
	for _, l := range strings.Split(string(data), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) > historyMaxEntries {
		lines = lines[len(lines)-historyMaxEntries:]
	}
	tmp := path + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	if os.WriteFile(tmp, []byte(strings.Join(lines, "\n")+"\n"), 0o644) == nil {
		if os.Rename(tmp, path) != nil {
			os.Remove(tmp)
		}
	}
}
