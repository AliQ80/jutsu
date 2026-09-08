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

// real jj rejects global options placed after the subcommand.
func TestBuildCommandStringsGlobalFlagPrecedesSubcommand(t *testing.T) {
	m := newModel()
	focusCommand(t, &m, "log", "")
	selectGlobalFlag(t, &m, "-R", "../other-repo")

	short, long := m.buildInvocation().display()

	if want := "jj -R ../other-repo log"; short != want {
		t.Errorf("short = %q, want %q", short, want)
	}
	// The bar form emits the primary execution spelling (Value) too —
	// alternate spellings are docs-only, so -R stays -R in the long form.
	if want := "jj -R ../other-repo log"; long != want {
		t.Errorf("long = %q, want %q", long, want)
	}
}

// hasIncompleteInputs (via hasEmptyRequiredGlobalInput) blocks finalizing to the command bar.
func TestBuildCommandStringsGlobalFlagWithoutValueOmitsOnlyValue(t *testing.T) {
	m := newModel()
	focusCommand(t, &m, "log", "")
	selectGlobalFlag(t, &m, "-R", "")

	short, _ := m.buildInvocation().display()

	if want := "jj -R log"; short != want {
		t.Errorf("short = %q, want %q", short, want)
	}
}

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

// resetCurrentFlags must clear selections and blank input values when navigating away or finishing execution.

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
