package main

import (
	"testing"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbletea/v2"
)

// syntheticModel returns a model whose single command exposes exactly the
// given flags, so tests can exercise mandatory/optional focus behavior without
// depending on the real catalog.
func syntheticModel(flags []Flag) mainModel {
	m := newModel()
	m.categories = []Category{{
		Name: "test",
		Commands: []Command{{
			Name:  "testcmd",
			Flags: flags,
		}},
	}}
	m.catIdx, m.cmdIdx, m.subIdx = 0, 0, 0
	return m
}

func mandatoryFlag(name string) Flag {
	return Flag{Name: name, Value: "--" + name, Mandatory: true, Selected: true}
}

func optionalFlag(name string) Flag {
	return Flag{Name: name, Value: "--" + name}
}

// Entering a mixed FLAGS pane lands on a non-mandatory flag, not the leading
// mandatory row.
func TestMixedEntrySkipsMandatory(t *testing.T) {
	m := syntheticModel([]Flag{mandatoryFlag("m"), optionalFlag("o1"), optionalFlag("o2")})
	m.focusPane = focusSubcmds
	m.flagIdx = 0

	next, _ := m.handleComposerKeys(tea.KeyPressMsg{Code: tea.KeyRight})
	got := next.(mainModel)

	if got.focusPane != focusFlags {
		t.Fatalf("focusPane = %d, want focusFlags", got.focusPane)
	}
	if got.flagIdx != 1 {
		t.Errorf("flagIdx = %d, want 1 (first non-mandatory)", got.flagIdx)
	}
}

// Up/down movement skips mandatory flags and wraps only across optional ones.
func TestNavigateSkipsMandatoryAndWraps(t *testing.T) {
	m := syntheticModel([]Flag{mandatoryFlag("m"), optionalFlag("o1"), optionalFlag("o2")})
	m.focusPane = focusFlags
	m.flagIdx = 1

	m.navigateDown()
	if m.flagIdx != 2 {
		t.Errorf("down from 1 = %d, want 2", m.flagIdx)
	}
	m.navigateDown()
	if m.flagIdx != 1 {
		t.Errorf("down wrap = %d, want 1", m.flagIdx)
	}
	m.navigateUp()
	if m.flagIdx != 2 {
		t.Errorf("up wrap = %d, want 2", m.flagIdx)
	}
	m.navigateUp()
	if m.flagIdx != 1 {
		t.Errorf("up from 2 = %d, want 1", m.flagIdx)
	}
}

// A stale mandatory index is snapped to the nearest navigable flag before
// moving, so navigation never lands on an inert row.
func TestNavigateSnapsStaleMandatoryIndex(t *testing.T) {
	m := syntheticModel([]Flag{mandatoryFlag("m"), optionalFlag("o1"), optionalFlag("o2")})
	m.focusPane = focusFlags
	m.flagIdx = 0 // stale: mandatory

	m.navigateDown()
	if m.flagIdx != 2 {
		t.Errorf("down from stale 0 = %d, want 2", m.flagIdx)
	}
}

// Horizontal entry into an all-mandatory FLAGS pane stays put.
func TestAllMandatoryKeyboardEntryStays(t *testing.T) {
	m := syntheticModel([]Flag{mandatoryFlag("m1"), mandatoryFlag("m2")})
	m.focusPane = focusSubcmds

	next, _ := m.handleComposerKeys(tea.KeyPressMsg{Code: tea.KeyRight})
	got := next.(mainModel)

	if got.focusPane != focusSubcmds {
		t.Errorf("focusPane = %d, want focusSubcmds (all-mandatory)", got.focusPane)
	}
}

// Mouse entry into an all-mandatory FLAGS pane stays on the previous pane.
func TestAllMandatoryMouseEntryStays(t *testing.T) {
	m := syntheticModel([]Flag{mandatoryFlag("m1"), mandatoryFlag("m2")})
	m.width, m.height = 120, 60
	m.focusPane = focusSubcmds

	// X=70 lands in the FLAGS pane column (63..85); Y=1 is inside the composer.
	next, _ := m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 70, Y: 1})
	got := next.(mainModel)

	if got.focusPane != focusSubcmds {
		t.Errorf("focusPane = %d, want focusSubcmds (all-mandatory mouse)", got.focusPane)
	}
}

// clampFocusPane pulls focus off FLAGS when every flag is mandatory.
func TestAllMandatoryClampFocusPane(t *testing.T) {
	m := syntheticModel([]Flag{mandatoryFlag("m1"), mandatoryFlag("m2")})
	m.focusPane = focusFlags

	m.clampFocusPane()

	if m.focusPane != focusCommands {
		t.Errorf("focusPane = %d, want focusCommands", m.focusPane)
	}
}

// Restoring a snapshot whose flagIdx points at a mandatory row normalizes it
// to a non-mandatory flag.
func TestRestoreNormalizesStaleMandatoryIndex(t *testing.T) {
	m := syntheticModel([]Flag{mandatoryFlag("m"), optionalFlag("o1"), optionalFlag("o2")})
	m.lastCmd = &commandSnapshot{
		catIdx:        0,
		cmdIdx:        0,
		subIdx:        0,
		flagIdx:       0, // stale: mandatory
		composerPane:  focusFlags,
		selectedFlags: map[string]bool{},
		inputValues:   map[string]string{},
		argValues:     map[string]string{},
	}

	m.restoreLastCmd()

	if m.focusPane != focusFlags {
		t.Errorf("focusPane = %d, want focusFlags", m.focusPane)
	}
	if m.flagIdx != 1 {
		t.Errorf("flagIdx = %d, want 1 (normalized)", m.flagIdx)
	}
}

// Restoring onto an all-mandatory command leaves FLAGS unfocused.
func TestRestoreAllMandatoryUnfocuses(t *testing.T) {
	m := syntheticModel([]Flag{mandatoryFlag("m1"), mandatoryFlag("m2")})
	m.lastCmd = &commandSnapshot{
		catIdx:        0,
		cmdIdx:        0,
		subIdx:        0,
		flagIdx:       0,
		composerPane:  focusFlags,
		selectedFlags: map[string]bool{},
		inputValues:   map[string]string{},
		argValues:     map[string]string{},
	}

	m.restoreLastCmd()

	if m.focusPane == focusFlags {
		t.Errorf("focusPane = %d, want unfocused (all-mandatory restore)", m.focusPane)
	}
}

// Mandatory rows stay in the list, stay selected, and cannot be toggled.
func TestMandatoryRowsVisibleSelectedNonToggleable(t *testing.T) {
	flags := []Flag{mandatoryFlag("m"), optionalFlag("o")}
	m := syntheticModel(flags)

	got := m.currentFlags()
	if len(got) != 2 {
		t.Fatalf("currentFlags len = %d, want 2 (mandatory stays visible)", len(got))
	}
	if !got[0].Mandatory || !got[0].Selected {
		t.Errorf("mandatory flag lost locked presentation: Mandatory=%v Selected=%v", got[0].Mandatory, got[0].Selected)
	}

	inputs := map[string]textinput.Model{}
	toggleFlagAt(flags, 0, inputs) // try to toggle the mandatory flag
	if !flags[0].Selected {
		t.Errorf("mandatory flag was toggled off")
	}

	toggleFlagAt(flags, 1, inputs) // optional flag toggles normally
	if !flags[1].Selected {
		t.Errorf("optional flag did not toggle on")
	}
}
