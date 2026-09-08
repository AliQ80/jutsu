package main

import (
	"strings"
	"testing"
)

// TestBuildCommandStringsConfigSetFlagsBeforeArgs verifies that, because
// config set opts into FlagsBeforeArgs, its target flag (--user/--repo/
// --workspace) is emitted before the NAME/VALUE positional args, matching how
// jj parses `jj config set`.
func TestBuildCommandStringsConfigSetFlagsBeforeArgs(t *testing.T) {
	cases := []struct {
		flagValue string
	}{
		{"--user"},
		{"--repo"},
		{"--workspace"},
	}
	for _, tc := range cases {
		t.Run(tc.flagValue, func(t *testing.T) {
			m := newModel()
			focusCommand(t, &m, "config", "set")
			selectFlag(t, &m, tc.flagValue, "")
			setArg(&m, "NAME", "user.name")
			setArg(&m, "VALUE", "\"Martin\"")

			short, long := m.buildInvocation().display()

			// Long form uses full names and is exact.
			want := "jj config set " + tc.flagValue + " user.name \"Martin\""
			if long != want {
				t.Errorf("long = %q, want %q", long, want)
			}
			// Short form may use aliases (e.g. "s" for set); assert only the
			// relative order: the target flag precedes NAME and VALUE.
			if idx := strings.Index(short, tc.flagValue); idx < 0 {
				t.Fatalf("short %q missing flag %q", short, tc.flagValue)
			} else if nameIdx := strings.Index(short, "user.name"); nameIdx < 0 {
				t.Fatalf("short %q missing NAME arg", short)
			} else if idx > nameIdx {
				t.Errorf("short = %q, flag %q should precede NAME arg", short, tc.flagValue)
			}
		})
	}
}

// TestBuildCommandStringsConfigUnsetArgsBeforeFlags verifies that config unset
// does NOT opt into FlagsBeforeArgs, so NAME stays before the --user flag
// (the pre-existing args-before-flags order is preserved).
func TestBuildCommandStringsConfigUnsetArgsBeforeFlags(t *testing.T) {
	m := newModel()
	focusCommand(t, &m, "config", "unset")
	selectFlag(t, &m, "--user", "")
	setArg(&m, "NAME", "user.name")

	short, long := m.buildInvocation().display()

	if want := "jj config unset user.name --user"; long != want {
		t.Errorf("long = %q, want %q", long, want)
	}
	// NAME must precede the flag in short form too.
	if nameIdx := strings.Index(short, "user.name"); nameIdx < 0 {
		t.Fatalf("short %q missing NAME arg", short)
	} else if flagIdx := strings.Index(short, "--user"); flagIdx < 0 {
		t.Fatalf("short %q missing flag --user", short)
	} else if flagIdx < nameIdx {
		t.Errorf("short = %q, NAME should precede flag (old order)", short)
	}
}

// TestBuildCommandStringsUnrelatedSubcommandArgsBeforeFlags proves the default
// ordering is untouched for a subcommand that has both flags and args but does
// not set FlagsBeforeArgs: args precede flags.
func TestBuildCommandStringsUnrelatedSubcommandArgsBeforeFlags(t *testing.T) {
	m := newModel()
	focusCommand(t, &m, "bookmark", "set")
	selectFlag(t, &m, "-r", "@-")
	setArg(&m, "NAMES", "main")

	short, long := m.buildInvocation().display()

	if want := "jj bookmark set main -r @-"; long != want {
		t.Errorf("long = %q, want %q", long, want)
	}
	if nameIdx := strings.Index(short, "main"); nameIdx < 0 {
		t.Fatalf("short %q missing NAMES arg", short)
	} else if flagIdx := strings.Index(short, "-r"); flagIdx < 0 {
		t.Fatalf("short %q missing flag -r", short)
	} else if flagIdx < nameIdx {
		t.Errorf("short = %q, NAMES should precede flag (old order)", short)
	}
}
