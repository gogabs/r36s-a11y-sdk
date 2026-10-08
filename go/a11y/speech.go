package a11y

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Speech fala pelo Speech Dispatcher (protocolo SSIP no socket Unix).
//
// Regras do contrato: usa a voz do usuário (/opt/a11y/etc/voz.conf), nunca
// valores próprios; fala interrompe a anterior por padrão. Sem Speech
// Dispatcher (no PC, por exemplo), a fala vai para o stderr.
type Speech struct {
	client, language string

	conn      net.Conn
	available bool
	replies   chan []string
	done      chan func() // callbacks prontos para rodar no laço do jogo

	mu        sync.Mutex
	callbacks map[string]func(bool)
	early     map[string]bool // falas que terminaram antes de o callback ser registrado
}

func newSpeech(client, language string) *Speech {
	return &Speech{
		client: client, language: language, available: true,
		done:      make(chan func(), 64),
		callbacks: map[string]func(bool){},
		early:     map[string]bool{},
	}
}

func speechSocket() string {
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		runtime = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	return filepath.Join(runtime, "speech-dispatcher", "speechd.sock")
}

// userVoice lê RATE, PITCH e VOLUME (0 a 100, convertidos para -100..100) e
// VOICE_MODULE / VOICE (texto, pode vir entre aspas).
func userVoice() (map[string]int, map[string]string) {
	path := os.Getenv("A11Y_VOZ_CONF")
	if path == "" {
		path = "/opt/a11y/etc/voz.conf"
	}
	out, names := map[string]int{}, map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out, names
	}
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if ok && (k == "VOICE_MODULE" || k == "VOICE") {
			if v = strings.Trim(strings.TrimSpace(v), `'"`); v != "" {
				names[k] = v
			}
			continue
		}
		if !ok || (k != "RATE" && k != "PITCH" && k != "VOLUME") {
			continue
		}
		if n, err := strconv.Atoi(v); err == nil {
			n = max(0, min(100, n))
			out[k] = n*2 - 100
		}
	}
	return out, names
}

// escapeText formata o texto para o comando SPEAK do SSIP.
func escapeText(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r", ""), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, ".") {
			lines[i] = "." + l // linha começando com "." precisa ser dobrada
		}
	}
	return strings.Join(lines, "\r\n") + "\r\n.\r\n"
}

func (s *Speech) connect() error {
	conn, err := net.DialTimeout("unix", speechSocket(), 3*time.Second)
	if err != nil {
		return err
	}
	s.conn = conn
	s.replies = make(chan []string, 4)
	go s.read(conn, s.replies)
	user := os.Getenv("USER")
	if user == "" {
		user = "ark"
	}
	params, names := userVoice()
	cmds := []string{fmt.Sprintf("SET SELF CLIENT_NAME %s:a11y:%s", user, s.client)}
	// módulo antes do idioma, e voz depois: trocar o idioma pode trocar a voz
	if m, ok := names["VOICE_MODULE"]; ok {
		cmds = append(cmds, "SET SELF OUTPUT_MODULE "+m)
	}
	cmds = append(cmds, "SET SELF LANGUAGE "+s.language)
	if v, ok := names["VOICE"]; ok {
		cmds = append(cmds, "SET SELF SYNTHESIS_VOICE "+v)
	}
	for _, k := range []string{"RATE", "PITCH", "VOLUME"} {
		if v, ok := params[k]; ok {
			cmds = append(cmds, fmt.Sprintf("SET SELF %s %d", k, v))
		}
	}
	cmds = append(cmds, "SET SELF NOTIFICATION END on", "SET SELF NOTIFICATION CANCEL on")
	for _, c := range cmds {
		if _, err := s.cmd(c); err != nil {
			s.close()
			return err
		}
	}
	return nil
}

// read separa respostas de comandos (vão para replies) de eventos 7xx.
func (s *Speech) read(conn net.Conn, replies chan<- []string) {
	r := bufio.NewReader(conn)
	var block []string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			close(replies)
			return
		}
		line = strings.TrimRight(line, "\r\n")
		block = append(block, line)
		if len(line) < 4 || line[3] != ' ' {
			continue // "NNN-..." continua o bloco
		}
		if line[0] == '7' {
			s.event(block)
		} else {
			replies <- block
		}
		block = nil
	}
}

func (s *Speech) event(block []string) {
	if len(block) < 3 {
		return
	}
	id := block[0][4:]
	finished := strings.HasPrefix(block[len(block)-1], "702") // 702 END, 703 CANCEL
	s.mu.Lock()
	fn, ok := s.callbacks[id]
	if ok {
		delete(s.callbacks, id)
	} else {
		if len(s.early) > 256 { // falas sem callback: não deixa crescer para sempre
			s.early = map[string]bool{}
		}
		s.early[id] = finished
	}
	s.mu.Unlock()
	if ok {
		s.done <- func() { fn(finished) }
	}
}

func (s *Speech) cmd(command string) ([]string, error) {
	if _, err := s.conn.Write([]byte(command + "\r\n")); err != nil {
		return nil, err
	}
	return s.reply()
}

func (s *Speech) reply() ([]string, error) {
	select {
	case r, ok := <-s.replies:
		if !ok {
			return nil, fmt.Errorf("speech-dispatcher fechou a conexão")
		}
		return r, nil
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("speech-dispatcher não respondeu")
	}
}

func (s *Speech) ensure() bool {
	if s.conn == nil && s.available {
		if err := s.connect(); err != nil {
			s.available = false
		}
	}
	return s.conn != nil
}

func (s *Speech) close() {
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
}

// Say fala o texto. Com queue=false, interrompe a fala anterior. onDone
// (opcional) recebe true quando a fala termina e false se foi cortada.
func (s *Speech) Say(text string, queue bool, onDone func(bool)) {
	for attempt := 0; attempt < 2; attempt++ {
		if !s.ensure() {
			fmt.Fprintln(os.Stderr, "[fala] "+text)
			if onDone != nil {
				s.done <- func() { onDone(true) }
			}
			return
		}
		if err := s.speak(text, queue, onDone); err == nil {
			return
		}
		s.close()
	}
}

func (s *Speech) speak(text string, queue bool, onDone func(bool)) error {
	if !queue {
		if _, err := s.cmd("STOP SELF"); err != nil {
			return err
		}
	}
	if _, err := s.cmd("SPEAK"); err != nil {
		return err
	}
	if _, err := s.conn.Write([]byte(escapeText(text))); err != nil {
		return err
	}
	r, err := s.reply()
	if err != nil {
		return err
	}
	if onDone != nil && len(r) > 1 {
		id := r[0][4:] // 225-<id da mensagem> / 225 OK MESSAGE QUEUED
		s.mu.Lock()
		finished, ended := s.early[id]
		if ended {
			delete(s.early, id)
		} else {
			s.callbacks[id] = onDone
		}
		s.mu.Unlock()
		if ended {
			s.done <- func() { onDone(finished) }
		}
	}
	return nil
}

// Hush cala a fala atual.
func (s *Speech) Hush() {
	if s.ensure() {
		if _, err := s.cmd("STOP SELF"); err != nil {
			s.close()
		}
	}
}
