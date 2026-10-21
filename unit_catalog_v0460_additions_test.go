package main

import (
	"slices"
	"testing"
)

func TestJJ0460NewPathsRequirePositionalsAndComposeArgv(t *testing.T) {
	cases := []struct {
		name      string
		cmd, sub  string
		argValues map[string]string
		want      []string
	}{
		{"util diff paths", "util", "diff", map[string]string{"PATH1": "a.txt", "PATH2": "b.txt"}, []string{"jj", "util", "diff", "a.txt", "b.txt"}},
		{"file edit", "file", "edit", map[string]string{"FILE": "src/main.go"}, []string{"jj", "file", "edit", "src/main.go"}},
		{"file delete", "file", "delete", map[string]string{"FILESETS": "old.txt"}, []string{"jj", "file", "delete", "old.txt"}},
		{"workspace remove", "workspace", "remove", map[string]string{"WORKSPACES": "feature"}, []string{"jj", "workspace", "remove", "feature"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newModel()
			focusCommand(t, &m, tc.cmd, tc.sub)
			args := m.currentArgs()
			if len(args) == 0 {
				t.Fatal("expected required positional input")
			}
			hasRequired := false
			for _, arg := range args {
				if arg.Required {
					hasRequired = true
				}
			}
			if !hasRequired || !m.hasIncompleteInputs() {
				t.Error("missing required positional input was not reported incomplete")
			}
			for name, value := range tc.argValues {
				setArg(&m, name, value)
			}
			if got := m.buildInvocation().argv; !slices.Equal(got, tc.want) {
				t.Errorf("argv = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestJJ0460ConfigFileTargetGroupAndOrder(t *testing.T) {
	for _, sub := range []string{"edit", "set", "unset"} {
		t.Run(sub+" target group", func(t *testing.T) {
			m := newModel()
			focusCommand(t, &m, "config", sub)
			if !m.requiredFlagGroupUnsatisfied() {
				t.Fatal("config command should require exactly one target selector")
			}
			selectFlag(t, &m, "--file", "")
			fileFlag := findFlag(m.currentFlags(), "--file")
			if fileFlag == nil || !fileFlag.RequiresInput {
				t.Fatalf("--file should require a path input: %+v", fileFlag)
			}
			if m.requiredFlagGroupUnsatisfied() {
				t.Error("selecting --file should satisfy the target group")
			}
			if !m.hasEmptyRequiredFlagInput() {
				t.Error("empty selected --file path should block command completion")
			}
			selectFlag(t, &m, "--file", "/tmp/config.toml")
			if m.hasEmptyRequiredFlagInput() {
				t.Error("non-empty --file path should satisfy its required input")
			}
			for _, value := range []string{"--user", "--repo", "--workspace"} {
				if f := findFlag(m.currentFlags(), value); f == nil || f.Selected {
					t.Errorf("target %s should be excluded when --file is selected: %+v", value, f)
				}
			}
		})
	}

	m := newModel()
	focusCommand(t, &m, "config", "set")
	selectFlag(t, &m, "--file", "/tmp/config.toml")
	setArg(&m, "NAME", "user.name")
	setArg(&m, "VALUE", "Example")
	if got, want := m.buildInvocation().argv, []string{"jj", "config", "set", "--file", "/tmp/config.toml", "user.name", "Example"}; !slices.Equal(got, want) {
		t.Errorf("argv = %q, want %q", got, want)
	}
}

func TestJJ0460NewOptionInputsConflictsAndInteractiveMarker(t *testing.T) {
	m := newModel()
	for _, tc := range []struct {
		cmd, sub, value string
	}{
		{"undo", "", "--allow-cross-workspace"},
		{"redo", "", "--allow-cross-workspace"},
		{"bisect", "run", "--trust-endpoints"},
		{"run", "", "--sparse-patterns"},
	} {
		focusCommand(t, &m, tc.cmd, tc.sub)
		if findFlag(m.currentFlags(), tc.value) == nil {
			t.Errorf("%s %s missing", tc.cmd, tc.value)
		}
	}

	focusCommand(t, &m, "workspace", "add")
	flags := m.currentFlags()
	for _, value := range []string{"--colocate", "--no-colocate"} {
		idx := slices.IndexFunc(flags, func(f Flag) bool { return f.Value == value })
		if idx < 0 {
			t.Fatalf("workspace add %s missing", value)
		}
		toggleFlagAt(flags, idx, m.inputs)
	}
	colocate := findFlag(m.currentFlags(), "--colocate")
	noColocate := findFlag(m.currentFlags(), "--no-colocate")
	if colocate == nil || noColocate == nil {
		t.Fatalf("workspace add colocation flags missing: --colocate=%+v --no-colocate=%+v", colocate, noColocate)
	}
	if colocate.Selected || !noColocate.Selected {
		t.Error("workspace add colocation options should be mutually exclusive")
	}

	focusCommand(t, &m, "converge", "")
	if findFlag(m.currentFlags(), "-i") != nil {
		t.Error("converge -i is completion-only and must not be offered")
	}

	focusCommand(t, &m, "file", "edit")
	if sub := findSubCmd(m.currentCommands()[m.cmdIdx].SubCmds, "edit"); sub == nil || !sub.Interactive {
		t.Errorf("file edit must be marked interactive: %+v", sub)
	}
}

func TestJJ0460GitPushSingleRemoteArgvIsPreserved(t *testing.T) {
	m := newModel()
	focusCommand(t, &m, "git", "push")
	selectFlag(t, &m, "--remote", "origin")
	if got, want := m.buildInvocation().argv, []string{"jj", "git", "push", "--remote", "origin"}; !slices.Equal(got, want) {
		t.Errorf("argv = %q, want %q", got, want)
	}
}

func TestJJ0460FileCommandsGroupedByEffect(t *testing.T) {
	for _, tc := range []struct{ category, sub string }{
		{"View", "search"}, {"Rewrite", "edit"}, {"Rewrite", "delete"}, {"Change", "track"},
	} {
		m := newModel()
		focusCommand(t, &m, "file", tc.sub)
		if got := m.categories[m.catIdx].Name; got != tc.category {
			t.Errorf("file %s is in %s, want %s", tc.sub, got, tc.category)
		}
	}
}
