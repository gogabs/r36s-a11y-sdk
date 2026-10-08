package a11y

import "math"

const (
	evKey = 0x01
	evAbs = 0x03
)

// ButtonEvent é um botão apertado, solto ou repetido.
type ButtonEvent struct {
	Button Button
	State  State
	Ms     int64 // milissegundos desde o início do jogo
}

// AxisEvent é um analógico que mudou, de -1 a 1.
type AxisEvent struct {
	Axis  Axis
	Value float64
	Ms    int64
}

// inputState transforma eventos crus do kernel em eventos do contrato:
// descarta Fn e o que for apertado com ele, volume, Power e o auto-repeat do
// kernel, e aplica zona morta aos eixos. Separado da leitura para ser testável.
type inputState struct {
	deadzone   float64
	fnHeld     bool
	suppressed map[Button]bool
	held       map[Button]bool
	axisRange  map[uint16][2]int32
	axisValue  map[Axis]float64
}

func newInputState(deadzone float64) *inputState {
	return &inputState{
		deadzone:   deadzone,
		suppressed: map[Button]bool{},
		held:       map[Button]bool{},
		axisRange:  map[uint16][2]int32{},
		axisValue:  map[Axis]float64{},
	}
}

func (s *inputState) setAxisRange(code uint16, min, max int32) {
	s.axisRange[code] = [2]int32{min, max}
}

// feed devolve um *ButtonEvent, um *AxisEvent ou nil.
func (s *inputState) feed(evType, code uint16, value int32, ms int64) interface{} {
	switch evType {
	case evKey:
		return s.key(code, value, ms)
	case evAbs:
		return s.abs(code, value, ms)
	}
	return nil
}

func (s *inputState) key(code uint16, value int32, ms int64) interface{} {
	if code == fnCode {
		s.fnHeld = value != 0
		return nil
	}
	name, ok := buttonCodes[code]
	if !ok || value == 2 { // fora do contrato, ou auto-repeat do kernel
		return nil
	}
	if value == 1 {
		if s.fnHeld {
			s.suppressed[name] = true
			return nil
		}
		s.held[name] = true
		return &ButtonEvent{name, Pressed, ms}
	}
	if s.suppressed[name] {
		delete(s.suppressed, name)
		return nil
	}
	if !s.held[name] {
		return nil
	}
	delete(s.held, name)
	return &ButtonEvent{name, Released, ms}
}

func (s *inputState) abs(code uint16, raw int32, ms int64) interface{} {
	name, ok := axisCodes[code]
	if !ok {
		return nil
	}
	r, ok := s.axisRange[code]
	if !ok {
		r = [2]int32{-1800, 1800}
	}
	center := float64(r[0]+r[1]) / 2
	half := float64(r[1]-r[0]) / 2
	if half == 0 {
		half = 1
	}
	v := math.Max(-1, math.Min(1, (float64(raw)-center)/half))
	if math.Abs(v) < s.deadzone {
		v = 0
	} else {
		v = math.Copysign((math.Abs(v)-s.deadzone)/(1-s.deadzone), v)
	}
	last := s.axisValue[name]
	if v == last || (v != 0 && math.Abs(v-last) < 0.02) {
		return nil
	}
	s.axisValue[name] = v
	return &AxisEvent{name, v, ms}
}
