package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

const compatJJVersion = "0.43.0"

// jutsuVersion is stamped at build time via -ldflags -X main.jutsuVersion=<tag>.
var jutsuVersion = "dev"

func main() {
	m := newModel()
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
