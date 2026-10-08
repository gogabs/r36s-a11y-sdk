//go:build !linux

package a11y

import "fmt"

// Fora do Linux ainda não há áudio: os sons viram silêncio.
func loadOpenAL() error { return fmt.Errorf("áudio só no Linux por enquanto") }

func decodeOgg(path string) (int, int, []byte, error) {
	return 0, 0, nil, fmt.Errorf("áudio só no Linux por enquanto")
}
