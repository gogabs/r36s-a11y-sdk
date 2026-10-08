//go:build !linux

package a11y

import "time"

type rawEvent struct {
	evType, code uint16
	value        int32
	ms           int64
}

// Fora do Linux ainda não há leitura de controle.
func openGamepads(state *inputState, start time.Time, out chan<- rawEvent) int {
	return 0
}
