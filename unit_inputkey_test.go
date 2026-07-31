package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// keyPress builds the message Bubble Tea delivers for a printable character.
func keyPress(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// modelOnCommandWithArgs parks a fresh model on the first command that has
// positional args, so the inputs box is non-empty.
func modelOnCommandWithArgs(t *testing.T) mainModel {
	t.Helper()
	m := newModel()
	m.width, m.height = 120, 40
	for ci := range m.categories {
		m.catIdx = ci
		for cmdi := range m.categories[ci].Commands {
			m.cmdIdx = cmdi
			m.subIdx = 0
			if len(m.getActiveCombined()) > 0 {
				return m
			}
		}
	}
	t.Fatal("no command with inputs in the command database")
	return m
}

// "i" is a global pane-jump key: it must work from the output, docs and command
// bar panes, not just the composer. It used to live in handleComposerKeys only.
func TestInputKeyIsGlobal(t *testing.T) {
	for _, pane := range []int{focusOutput, focusDocs, focusCmdBar, focusCategories} {
		m := modelOnCommandWithArgs(t)
		m.focusPane = pane

		next, _ := m.Update(keyPress('i'))
		got := next.(mainModel).focusPane
		if got != focusInputs {
			t.Errorf("i from pane %d left focus at %d, want focusInputs (%d)", pane, got, focusInputs)
		}
	}
}

// ...but inside the inputs box "i" is just a letter. The global case has to
// fall through to the pane dispatch rather than swallow the keypress.
func TestInputKeyTypesInsideInputs(t *testing.T) {
	m := modelOnCommandWithArgs(t)
	m.focusPane = focusCategories
	next, _ := m.Update(keyPress('i'))
	m = next.(mainModel)

	next, _ = m.Update(keyPress('i'))
	m = next.(mainModel)

	if m.focusPane != focusInputs {
		t.Fatalf("focus left the inputs box, at %d", m.focusPane)
	}
	item := m.getActiveCombined()[m.focusedInputIdx]
	if got := m.getInput(item).Value(); got != "i" {
		t.Errorf("input value = %q, want %q — the global i case swallowed the keypress", got, "i")
	}
}
