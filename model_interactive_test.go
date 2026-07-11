package main

import (
	"testing"

	"charm.land/bubbles/v2/textinput"
)

// focusCommand points the model at the named command (and subcommand, when
// given), mimicking composer navigation.
func focusCommand(t *testing.T, m *mainModel, cmdName, subName string) {
	t.Helper()
	for ci := range m.categories {
		for cmi := range m.categories[ci].Commands {
			cmd := &m.categories[ci].Commands[cmi]
			if cmd.Name != cmdName {
				continue
			}
			m.catIdx, m.cmdIdx, m.subIdx = ci, cmi, 0
			if subName == "" {
				return
			}
			for si := range cmd.SubCmds {
				if cmd.SubCmds[si].Name == subName {
					m.subIdx = si
					return
				}
			}
		}
	}
	t.Fatalf("command %q (sub %q) not found", cmdName, subName)
}

// selectFlag marks the flag with the given Value as selected, optionally
// giving its text input a value.
func selectFlag(t *testing.T, m *mainModel, value, input string) {
	t.Helper()
	flags := m.currentFlags()
	for i := range flags {
		if flags[i].Value == value {
			flags[i].Selected = true
			if input != "" {
				ti := textinput.New()
				ti.SetValue(input)
				m.inputs[flags[i].Name] = ti
			}
			return
		}
	}
	t.Fatalf("flag %q not found on current command", value)
}

func setArg(m *mainModel, name, value string) {
	ti := textinput.New()
	ti.SetValue(value)
	m.argInputs[name] = ti
}

func TestIsInteractiveInvocation(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, m *mainModel)
		want  bool
	}{
		{"log is never interactive", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "log", "")
		}, false},
		{"describe without -m opens editor", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "describe", "")
		}, true},
		{"describe with -m is direct", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "describe", "")
			selectFlag(t, m, "-m", "msg")
		}, false},
		{"commit -i forces picker even with -m", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "commit", "")
			selectFlag(t, m, "-m", "msg")
			selectFlag(t, m, "-i", "")
		}, true},
		{"bare resolve opens merge tool", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "resolve", "")
		}, true},
		{"resolve --list only prints", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "resolve", "")
			selectFlag(t, m, "--list", "")
		}, false},
		{"bare split opens diff editor", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "split", "")
		}, true},
		{"split with paths but no -m may still prompt", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "split", "")
			setArg(m, "FILESETS", "main.go")
		}, true},
		{"split with paths and -m is direct", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "split", "")
			setArg(m, "FILESETS", "main.go")
			selectFlag(t, m, "-m", "msg")
		}, false},
		{"config edit opens editor", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "config", "edit")
		}, true},
		{"config list is direct", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "config", "list")
		}, false},
		{"diffedit is always interactive", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "diffedit", "")
		}, true},
		{"sparse edit opens editor", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "sparse", "edit")
		}, true},
		{"bare metaedit opens editor", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "metaedit", "")
		}, true},
		{"metaedit with a metadata flag is direct", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "metaedit", "")
			selectFlag(t, m, "--update-author", "")
		}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newModel()
			tc.setup(t, &m)
			if got := m.isInteractiveInvocation(); got != tc.want {
				t.Errorf("isInteractiveInvocation() = %v, want %v", got, tc.want)
			}
		})
	}
}
