package a11y

import "fmt"

// MenuHelp é a instrução falada ao abrir um menu.
const MenuHelp = "Cima e baixo escolhem. B confirma. A volta."

// Menu é um menu vertical falado. Cima/Baixo navegam (dando a volta),
// B confirma, A volta, Select repete o item atual. Enquanto aberto, recebe
// os botões no lugar do jogo.
type Menu struct {
	Title    string
	Items    []string
	OnSelect func(index int, item string)
	OnBack   func() // sem OnBack, Voltar fecha o menu
	MoveSound, SelectSound *Sound

	game  *Game
	Index int
	Open  bool
}

// NewMenu cria um menu ligado ao jogo.
func NewMenu(g *Game, title string, items []string, onSelect func(int, string)) *Menu {
	return &Menu{game: g, Title: title, Items: items, OnSelect: onSelect}
}

// Show abre o menu no item index, falando o título, o item e (se help) a instrução.
func (m *Menu) Show(index int, help bool) {
	m.Index = index
	if !m.Open {
		m.Open = true
		m.game.PushFocus(m)
	}
	text := m.Title + ". " + m.itemText()
	if help {
		text += ". " + MenuHelp
	}
	m.game.Say(text)
}

// Close fecha o menu e devolve os botões ao jogo.
func (m *Menu) Close() {
	m.Open = false
	m.game.PopFocus(m)
}

func (m *Menu) itemText() string {
	return fmt.Sprintf("%s, %d de %d", m.Items[m.Index], m.Index+1, len(m.Items))
}

func (m *Menu) move(delta int) {
	m.Index = (m.Index + delta + len(m.Items)) % len(m.Items)
	m.MoveSound.Play()
	m.game.Say(m.itemText())
}

// HandleButton implementa Focus.
func (m *Menu) HandleButton(e ButtonEvent) {
	if e.State == Released {
		return
	}
	switch {
	case e.Button == Down:
		m.move(1)
	case e.Button == Up:
		m.move(-1)
	case e.State != Pressed:
	case IsConfirm(e.Button):
		m.SelectSound.Play()
		m.OnSelect(m.Index, m.Items[m.Index])
	case IsBack(e.Button):
		if m.OnBack != nil {
			m.OnBack()
		} else {
			m.Close()
		}
	case e.Button == Select:
		m.game.Say(m.itemText())
	}
}
