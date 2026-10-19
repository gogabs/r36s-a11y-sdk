package a11y

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const (
	codeB   = 0x130
	codeUp  = 0x220
	codeVol = 115
)

func btn(e interface{}) ButtonEvent {
	if b, ok := e.(*ButtonEvent); ok {
		return *b
	}
	return ButtonEvent{}
}

func TestPressAndRelease(t *testing.T) {
	s := newInputState(0.2)
	if got := btn(s.feed(evKey, codeB, 1, 10)); got != (ButtonEvent{B, Pressed, 10}) {
		t.Fatalf("aperto: %+v", got)
	}
	if got := btn(s.feed(evKey, codeB, 0, 20)); got != (ButtonEvent{B, Released, 20}) {
		t.Fatalf("soltura: %+v", got)
	}
}

func TestFnNeverDelivered(t *testing.T) {
	s := newInputState(0.2)
	if s.feed(evKey, fnCode, 1, 0) != nil || s.feed(evKey, fnCode, 0, 1) != nil {
		t.Fatal("Fn chegou ao jogo")
	}
}

func TestButtonWithFnSuppressedUntilReleased(t *testing.T) {
	s := newInputState(0.2)
	s.feed(evKey, fnCode, 1, 0)
	if s.feed(evKey, codeUp, 1, 1) != nil {
		t.Fatal("botão com Fn chegou ao jogo")
	}
	s.feed(evKey, fnCode, 0, 2) // solta Fn antes do botão
	if s.feed(evKey, codeUp, 0, 3) != nil {
		t.Fatal("soltura do botão suprimido chegou ao jogo")
	}
	if got := btn(s.feed(evKey, codeUp, 1, 4)); got.State != Pressed {
		t.Fatal("aperto seguinte não chegou")
	}
}

func TestSystemKeysAndKernelRepeatIgnored(t *testing.T) {
	s := newInputState(0.2)
	if s.feed(evKey, codeVol, 1, 0) != nil {
		t.Fatal("volume chegou ao jogo")
	}
	s.feed(evKey, codeB, 1, 0)
	if s.feed(evKey, codeB, 2, 5) != nil {
		t.Fatal("auto-repeat do kernel chegou ao jogo")
	}
}

func TestAxisDeadzoneAndScale(t *testing.T) {
	s := newInputState(0.2)
	s.setAxisRange(0, -1800, 1800)
	if s.feed(evAbs, 0, 100, 0) != nil {
		t.Fatal("zona morta não aplicada")
	}
	if e, _ := s.feed(evAbs, 0, 1800, 1).(*AxisEvent); e == nil || e.Value != 1 {
		t.Fatalf("eixo no máximo: %+v", e)
	}
	if e, _ := s.feed(evAbs, 0, 0, 2).(*AxisEvent); e == nil || e.Value != 0 {
		t.Fatalf("eixo no centro: %+v", e)
	}
}

func TestEscapeText(t *testing.T) {
	if got := escapeText("oi\n.ponto"); got != "oi\r\n..ponto\r\n.\r\n" {
		t.Fatalf("%q", got)
	}
}

func TestSpeechHistoryKeepsLastFifty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "historico-falas.txt")
	for i := 0; i < 2000; i++ {
		recordSpeech(path, "Fala\nnúmero "+strconv.Itoa(i))
	}
	recordSpeech(path, "   ")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if n := len(lines); n < historyMaxEntries || len(data) > historyTrimBytes+100 {
		t.Fatalf("%d linhas, %d bytes", n, len(data))
	}
	if last := lines[len(lines)-1]; last != "Fala número 1999" {
		t.Fatalf("última: %q", last)
	}
	recordSpeech(filepath.Join(t.TempDir(), "nao-existe", "h.txt"), "Oi") // sem pasta: nada
}

func TestSpeechOffSkipsSpeechAndFinishes(t *testing.T) {
	flag := filepath.Join(t.TempDir(), "fala-desligada")
	t.Setenv("A11Y_FALA_DESLIGADA", flag)
	t.Setenv("A11Y_HISTORICO_FALAS", filepath.Join(t.TempDir(), "nao-existe", "h.txt"))
	if speechOff() {
		t.Fatal("sem o arquivo a fala está ligada")
	}
	if err := os.WriteFile(flag, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	s := newSpeech("teste", "pt-BR")
	s.available = false // se tentasse conectar, cairia no stderr
	var got []bool
	s.Say("Oi", false, func(done bool) { got = append(got, done) })
	(<-s.done)()
	if s.conn != nil || len(got) != 1 || !got[0] {
		t.Fatalf("conn=%v got=%v", s.conn, got)
	}
}
