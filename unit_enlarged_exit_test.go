package main

import (
	"testing"
)

// Lowercase "o" from an enlarged output view returns to the normal output
// pane: focus stays on output and the enlarged flag is cleared.
func TestLowerOPaneFromEnlargedOutput(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.lastFocusPane = focusCategories
	m.focusPane = focusOutput
	m.outputEnlarged = true
	m.layoutViewports()

	next, _ := m.Update(keyPress('o'))
	got := next.(mainModel)

	if got.focusPane != focusOutput {
		t.Errorf("focusPane = %d, want focusOutput (%d)", got.focusPane, focusOutput)
	}
	if got.outputEnlarged {
		t.Error("outputEnlarged still set after lowercase o from enlarged output")
	}
}

// Uppercase "O" from an enlarged output view exits to the last composer pane.
func TestUpperOPaneFromEnlargedOutput(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.lastFocusPane = focusCategories
	m.focusPane = focusOutput
	m.outputEnlarged = true
	m.layoutViewports()

	next, _ := m.Update(keyPress('O'))
	got := next.(mainModel)

	if got.focusPane != focusCategories {
		t.Errorf("focusPane = %d, want lastFocusPane focusCategories (%d)", got.focusPane, focusCategories)
	}
	if got.outputEnlarged {
		t.Error("outputEnlarged still set after uppercase O from enlarged output")
	}
}

// Lowercase "d" from an enlarged docs view returns to the normal docs pane:
// focus stays on docs and the enlarged flag is cleared.
func TestLowerDPaneFromEnlargedDocs(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.lastFocusPane = focusCategories
	m.focusPane = focusDocs
	m.docsEnlarged = true
	m.layoutViewports()

	next, _ := m.Update(keyPress('d'))
	got := next.(mainModel)

	if got.focusPane != focusDocs {
		t.Errorf("focusPane = %d, want focusDocs (%d)", got.focusPane, focusDocs)
	}
	if got.docsEnlarged {
		t.Error("docsEnlarged still set after lowercase d from enlarged docs")
	}
}

// Uppercase "D" from an enlarged docs view exits to the last composer pane.
func TestUpperDPaneFromEnlargedDocs(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.lastFocusPane = focusCategories
	m.focusPane = focusDocs
	m.docsEnlarged = true
	m.layoutViewports()

	next, _ := m.Update(keyPress('D'))
	got := next.(mainModel)

	if got.focusPane != focusCategories {
		t.Errorf("focusPane = %d, want lastFocusPane focusCategories (%d)", got.focusPane, focusCategories)
	}
	if got.docsEnlarged {
		t.Error("docsEnlarged still set after uppercase D from enlarged docs")
	}
}

// Normal lowercase "o" from a composer pane still moves focus to output.
func TestLowerONormalFocusesOutput(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.focusPane = focusCategories
	m.lastFocusPane = focusCategories
	m.layoutViewports()

	next, _ := m.Update(keyPress('o'))
	got := next.(mainModel)

	if got.focusPane != focusOutput {
		t.Errorf("focusPane = %d, want focusOutput (%d)", got.focusPane, focusOutput)
	}
	if got.outputEnlarged {
		t.Error("outputEnlarged set by normal lowercase o")
	}
}

// Normal lowercase "d" from a composer pane still moves focus to docs.
func TestLowerDNormalFocusesDocs(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.focusPane = focusCategories
	m.lastFocusPane = focusCategories
	m.layoutViewports()

	next, _ := m.Update(keyPress('d'))
	got := next.(mainModel)

	if got.focusPane != focusDocs {
		t.Errorf("focusPane = %d, want focusDocs (%d)", got.focusPane, focusDocs)
	}
	if got.docsEnlarged {
		t.Error("docsEnlarged set by normal lowercase d")
	}
}
