package main

import (
	"testing"

	"charm.land/bubbles/v2/textinput"
)

// selectGlobalFlag marks the global flag with the given Value as selected,
// optionally giving its text input a value.
func selectGlobalFlag(t *testing.T, m *mainModel, value, input string) {
	t.Helper()
	for i := range m.globalFlags {
		if m.globalFlags[i].Value == value {
			m.globalFlags[i].Selected = true
			if input != "" {
				ti := textinput.New()
				ti.SetValue(input)
				m.inputs[m.globalFlags[i].Name] = ti
			}
			return
		}
	}
	t.Fatalf("global flag %q not found", value)
}

// A selected global flag must be spliced in right after "jj" and before the
// subcommand — real jj rejects global options placed after the subcommand.
func TestBuildCommandStringsGlobalFlagPrecedesSubcommand(t *testing.T) {
	m := newModel()
	focusCommand(t, &m, "log", "")
	selectGlobalFlag(t, &m, "-R", "../other-repo")

	short, long := m.buildCommandStrings()

	if want := "jj -R ../other-repo log"; short != want {
		t.Errorf("short = %q, want %q", short, want)
	}
	if want := "jj --repository ../other-repo log"; long != want {
		t.Errorf("long = %q, want %q", long, want)
	}
}

// A selected global flag's token appears even with no value entered — same
// as a command-scoped RequiresInput flag, buildCommandStrings only omits the
// value text itself; hasIncompleteInputs (via hasEmptyRequiredGlobalInput)
// is what blocks finalizing to the command bar in this state.
func TestBuildCommandStringsGlobalFlagWithoutValueOmitsOnlyValue(t *testing.T) {
	m := newModel()
	focusCommand(t, &m, "log", "")
	selectGlobalFlag(t, &m, "-R", "")

	short, _ := m.buildCommandStrings()

	if want := "jj -R log"; short != want {
		t.Errorf("short = %q, want %q", short, want)
	}
}

// stickyGlobalCount is the number of pinned global rows the FLAGS pane shows:
// the selected count, capped so at least one command-flag row always remains.
func TestStickyGlobalCount(t *testing.T) {
	m := newModel()

	// paneH such that contentHeight (paneH-4) is roomy.
	m.paneH = 20

	if got := m.stickyGlobalCount(); got != 0 {
		t.Errorf("no globals selected: stickyGlobalCount = %d, want 0", got)
	}

	selectGlobalFlag(t, &m, "-R", "../other-repo")
	selectGlobalFlag(t, &m, "--debug", "")
	if got := m.stickyGlobalCount(); got != 2 {
		t.Errorf("two globals selected in a roomy pane: stickyGlobalCount = %d, want 2", got)
	}

	// A pane with contentHeight == 2 (paneH-4) can pin at most 1 row so a
	// flag row survives; the rest fold into the "+N more" summary.
	m.paneH = 6
	if got := m.stickyGlobalCount(); got != 1 {
		t.Errorf("short pane: stickyGlobalCount = %d, want 1 (capped)", got)
	}
}

// Global flags reset on the same triggers as command flags — navigating away
// from the current command and finishing an execution both call
// resetCurrentFlags, which must clear selections and blank input values.
func TestResetCurrentFlagsClearsGlobalFlags(t *testing.T) {
	m := newModel()
	focusCommand(t, &m, "log", "")
	selectGlobalFlag(t, &m, "-R", "../other-repo")

	m.resetCurrentFlags()

	for _, f := range m.globalFlags {
		if f.Selected {
			t.Errorf("global flag %s still selected after reset", f.Value)
		}
	}
	if got := m.inputs["repository"].Value(); got != "" {
		t.Errorf("repository input = %q after reset, want empty", got)
	}
}
