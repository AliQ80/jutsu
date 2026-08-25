package main

import (
	"testing"

	"charm.land/lipgloss/v2"
)

// paneRows() is the single source of truth for the left column's row
// boundaries: View() draws from it and the mouse handlers hit-test against it.
// When those two disagreed, clicks on the inputs box landed on the docs pane —
// so the invariant worth pinning is that the inputs band is exactly as tall as
// renderInputPanes draws it, and that it sits directly below the composer,
// above the docs bar.
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
	composerH, docsTop, bottom := m.paneRows()

	if got, want := docsTop-composerH, len(combined)+2; got != want {
		t.Errorf("inputs band = %d rows, renderInputPanes draws %d", got, want)
	}
	if docsTop >= bottom {
		t.Errorf("inputs box (ends %d) leaves no room for the docs bar (bottom %d)", docsTop, bottom)
	}
	if want := m.height - 6; bottom != want {
		t.Errorf("command bar starts at %d, want %d", bottom, want)
	}
}

// With nothing to fill in, the inputs band is empty and the docs bar starts
// right below the composer — a click there must not be mistaken for an input
// row.
func TestPaneRowsNoInputs(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40

	for ci := range m.categories {
		m.catIdx = ci
		for cmdi := range m.categories[ci].Commands {
			m.cmdIdx = cmdi
			m.subIdx = 0
			if len(m.getActiveCombined()) == 0 {
				composerH, docsTop, _ := m.paneRows()
				if docsTop != composerH {
					t.Fatalf("empty inputs band spans %d rows, want 0", docsTop-composerH)
				}
				return
			}
		}
	}
	t.Skip("every command in the database has inputs")
}

// The reordered left column must still add up to exactly the rows paneRows
// hands out: composer + inputs + docs == bottom, with docs absorbing the
// remainder. Pinning the rendered heights catches a fixed-height lipgloss box
// wrapping and pushing the help bar off-screen.
func TestRenderMainFrameHeights(t *testing.T) {
	for _, height := range []int{24, 40, 60} {
		m := newModel()
		m.width, m.height = 120, height
	outer:
		for ci := range m.categories {
			m.catIdx = ci
			for cmdi := range m.categories[ci].Commands {
				m.cmdIdx = cmdi
				m.subIdx = 0
				if len(m.currentArgs()) > 0 {
					break outer
				}
			}
		}
		m.layoutViewports()

		leftWidth, _ := m.getLayoutWidths()
		composerH, docsTop, bottom := m.paneRows()
		inputsH, docsH := docsTop-composerH, bottom-docsTop

		// Each section must render exactly the band it was handed, in the
		// order the left column joins them: composer, inputs, docs.
		got := []int{
			lipgloss.Height(m.renderLeftPane(composerH)),
			lipgloss.Height(m.renderInputPanes(leftWidth, inputsH, m.getActiveCombined())),
			lipgloss.Height(m.renderDescriptionBar(leftWidth, docsH)),
		}
		for i, want := range []int{composerH, inputsH, docsH} {
			if got[i] != want {
				t.Errorf("h=%d: section %d renders %d rows, want %d", height, i, got[i], want)
			}
		}
		if got := lipgloss.Height(m.renderMainFrame()); got != height {
			t.Errorf("h=%d: frame renders %d rows, want %d", height, got, height)
		}
	}
}
