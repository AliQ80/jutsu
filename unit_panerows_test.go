package main

import "testing"

// paneRows() is the single source of truth for the left column's row
// boundaries: View() draws from it and the mouse handlers hit-test against it.
// When those two disagreed, clicks on the inputs box landed on the docs pane —
// so the invariant worth pinning is that the inputs band is exactly as tall as
// renderInputPanes draws it, and that it sits below the docs bar.
func TestPaneRowsMatchesInputsBox(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40

	// Park on the first command that has positional args — the case the mouse
	// handler used to get wrong, since it sized the box from flag inputs alone.
	found := false
outer:
	for ci := range m.categories {
		m.catIdx = ci
		for cmdi := range m.categories[ci].Commands {
			m.cmdIdx = cmdi
			m.subIdx = 0
			if len(m.currentArgs()) > 0 {
				found = true
				break outer
			}
		}
	}
	if !found {
		t.Fatal("no command with positional args in the command database")
	}

	combined := m.getActiveCombined()
	composerH, inputsTop, bottom := m.paneRows()

	if got, want := bottom-inputsTop, len(combined)+2; got != want {
		t.Errorf("inputs band = %d rows, renderInputPanes draws %d", got, want)
	}
	if inputsTop <= composerH {
		t.Errorf("inputs box (top %d) overlaps the composer (height %d)", inputsTop, composerH)
	}
	if want := m.height - 6; bottom != want {
		t.Errorf("command bar starts at %d, want %d", bottom, want)
	}
}

// With nothing to fill in, the inputs band is empty and the docs bar runs all
// the way down — a click there must not be mistaken for an input row.
func TestPaneRowsNoInputs(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40

	for ci := range m.categories {
		m.catIdx = ci
		for cmdi := range m.categories[ci].Commands {
			m.cmdIdx = cmdi
			m.subIdx = 0
			if len(m.getActiveCombined()) == 0 {
				_, inputsTop, bottom := m.paneRows()
				if inputsTop != bottom {
					t.Fatalf("empty inputs band spans %d rows, want 0", bottom-inputsTop)
				}
				return
			}
		}
	}
	t.Skip("every command in the database has inputs")
}
