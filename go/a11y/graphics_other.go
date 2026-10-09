//go:build !linux

package a11y

import "fmt"

// Fora do Linux ainda não há gráficos: os desenhos são ignorados.
func loadSDL() error { return fmt.Errorf("gráficos só no Linux por enquanto") }
