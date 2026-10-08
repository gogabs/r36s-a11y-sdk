// Package a11y é a biblioteca-ponte Go da plataforma de jogos acessíveis do
// R36S. No console, lê o controle em /dev/input, fala pelo Speech Dispatcher
// e toca áudio 3D pela OpenAL Soft do sistema, sem cgo.
package a11y

// Button é o nome padrão de um botão do contrato.
type Button string

const (
	Up     Button = "up"
	Down   Button = "down"
	Left   Button = "left"
	Right  Button = "right"
	A      Button = "a" // botão da direita: voltar
	B      Button = "b" // botão de baixo: confirmar
	X      Button = "x"
	Y      Button = "y"
	L1     Button = "l1"
	R1     Button = "r1"
	L2     Button = "l2"
	R2     Button = "r2"
	L3     Button = "l3"
	R3     Button = "r3"
	Start  Button = "start"
	Select Button = "select"
)

// State é o estado de um evento de botão.
type State string

const (
	Pressed  State = "down"
	Released State = "up"
	Repeated State = "repeat"
)

// Axis é o nome de um eixo analógico.
type Axis string

const (
	LX Axis = "lx"
	LY Axis = "ly"
	RX Axis = "rx"
	RY Axis = "ry"
)

// Códigos evdev -> nomes do contrato.
var buttonCodes = map[uint16]Button{
	0x130: B, // BTN_SOUTH
	0x131: A, // BTN_EAST
	0x133: X, // BTN_NORTH
	0x134: Y, // BTN_WEST
	0x136: L1,
	0x137: R1,
	0x138: L2,
	0x139: R2,
	0x220: Up,
	0x221: Down,
	0x222: Left,
	0x223: Right,
	0x2C0: Select, // BTN_TRIGGER_HAPPY1
	0x2C1: Start,  // BTN_TRIGGER_HAPPY2
	0x2C2: L3,
	0x2C3: R3,
}

// Fn é do sistema (a11yd): nunca chega ao jogo, nem o que for apertado com ele.
const fnCode = 0x2C4 // BTN_TRIGGER_HAPPY5

var axisCodes = map[uint16]Axis{
	0x00: LX, // ABS_X
	0x01: LY, // ABS_Y
	0x03: RX, // ABS_RX
	0x04: RY, // ABS_RY
}

// IsConfirm diz se o botão confirma (B, de baixo).
func IsConfirm(b Button) bool { return b == B }

// IsBack diz se o botão volta (A, da direita).
func IsBack(b Button) bool { return b == A }
