package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

const compatJJVersion = "0.46.0"

// jutsuVersion is stamped at build time via -ldflags -X main.jutsuVersion=<tag>.
var jutsuVersion = "dev"

func main() {
	if helper, err := runUpdateHelper(); helper {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	m := newModel()
	if executable, err := os.Executable(); err == nil {
		if notices := inspectUpdateRecovery(executable); len(notices) > 0 {
			m.setOutputLines(notices)
		}
	}
	p := tea.NewProgram(m)

	final, err := p.Run()
	if model, ok := final.(mainModel); ok && model.updateRestart == nil {
		model.stopUpdatePreparation()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if model, ok := final.(mainModel); ok && model.updateRestart != nil {
		if err := restartUpdated(model.updateRestart); err != nil {
			fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
			os.Exit(1)
		}
	}
}
