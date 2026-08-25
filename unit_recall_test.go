package main

import (
	"testing"
)

// recallModel parks a synthetic model on a two-flag command with a snapshot
// that selects the second flag and points focus at the FLAGS pane, so a
// successful recall is visible in focusPane, flagIdx and the selection.
func recallModel() mainModel {
	m := syntheticModel([]Flag{optionalFlag("o1"), optionalFlag("o2")})
	m.width, m.height = 120, 40
	m.lastCmd = &commandSnapshot{
		flagIdx:       1,
		composerPane:  focusFlags,
		selectedFlags: map[string]bool{"o2": true},
		inputValues:   map[string]string{},
		argValues:     map[string]string{},
	}
	return m
}

// "r" is a global recall key: it must work from the output, docs and command
// bar panes, not just the composer. It used to live in handleComposerKeys only.
func TestRecallKeyIsGlobal(t *testing.T) {
	for _, pane := range []int{focusCategories, focusFlags, focusOutput, focusDocs, focusCmdBar} {
		m := recallModel()
		m.focusPane = pane

		next, _ := m.Update(keyPress('r'))
		got := next.(mainModel)

		if got.focusPane != focusFlags {
			t.Errorf("r from pane %d left focus at %d, want focusFlags (%d)", pane, got.focusPane, focusFlags)
		}
		if got.flagIdx != 1 {
			t.Errorf("r from pane %d: flagIdx = %d, want 1 (snapshot)", pane, got.flagIdx)
		}
		if flags := got.currentFlags(); !flags[1].Selected {
			t.Errorf("r from pane %d did not restore the snapshot's flag selection", pane)
		}
	}
}

// Recalling from an enlarged view closes it — the restored composer pane can't
// render while an enlarged output/docs flag is still set.
func TestRecallFromEnlargedViewsClosesThem(t *testing.T) {
	m := recallModel()
	m.focusPane = focusOutput
	m.outputEnlarged = true

	next, _ := m.Update(keyPress('r'))
	got := next.(mainModel)
	if got.outputEnlarged {
		t.Error("recall from enlarged output left outputEnlarged set")
	}
	if got.focusPane != focusFlags {
		t.Errorf("recall from enlarged output: focusPane = %d, want focusFlags", got.focusPane)
	}

	m = recallModel()
	m.focusPane = focusDocs
	m.docsEnlarged = true

	next, _ = m.Update(keyPress('r'))
	got = next.(mainModel)
	if got.docsEnlarged {
		t.Error("recall from enlarged docs left docsEnlarged set")
	}
	if got.focusPane != focusFlags {
		t.Errorf("recall from enlarged docs: focusPane = %d, want focusFlags", got.focusPane)
	}
}

// With nothing recorded yet, "r" changes nothing — not even the enlarged view.
func TestRecallWithoutSnapshotIsNoOp(t *testing.T) {
	m := recallModel()
	m.lastCmd = nil
	m.focusPane = focusOutput
	m.outputEnlarged = true

	next, _ := m.Update(keyPress('r'))
	got := next.(mainModel)

	if got.focusPane != focusOutput || !got.outputEnlarged {
		t.Errorf("r with no snapshot changed state: focusPane = %d, outputEnlarged = %v", got.focusPane, got.outputEnlarged)
	}
}

// ...but inside the inputs box "r" is just a letter. The global case has to
// fall through to the pane dispatch rather than swallow the keypress.
func TestRecallKeyTypesInsideInputs(t *testing.T) {
	m := modelOnCommandWithArgs(t)
	m.lastCmd = &commandSnapshot{
		composerPane:  focusCategories,
		selectedFlags: map[string]bool{},
		inputValues:   map[string]string{},
		argValues:     map[string]string{},
	}
	m.focusPane = focusCategories

	next, _ := m.Update(keyPress('i'))
	m = next.(mainModel)
	next, _ = m.Update(keyPress('r'))
	m = next.(mainModel)

	if m.focusPane != focusInputs {
		t.Fatalf("focus left the inputs box, at %d", m.focusPane)
	}
	item := m.getActiveCombined()[m.focusedInputIdx]
	if got := m.getInput(item).Value(); got != "r" {
		t.Errorf("input value = %q, want %q — the global r case swallowed the keypress", got, "r")
	}
}
