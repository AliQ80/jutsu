package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

const (
	jutsuVersion    = "0.1.0"
	compatJJVersion = "0.43.0"
)

func main() {
	m := newModel()
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
