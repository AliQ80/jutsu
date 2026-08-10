package main

import "testing"

// TestCompatJJVersionIsV0440 pins the compatibility badge target to the jj
// 0.44.0 release this catalog update targets.
func TestCompatJJVersionIsV0440(t *testing.T) {
	if compatJJVersion != "0.44.0" {
		t.Errorf("compatJJVersion = %q, want %q", compatJJVersion, "0.44.0")
	}
}

// findFlag returns the flag with the given Value from flags, or nil.
func findFlag(flags []Flag, value string) *Flag {
	for i := range flags {
		if flags[i].Value == value {
			return &flags[i]
		}
	}
	return nil
}

// findSubCmd returns the subcommand with the given Name from subs, or nil.
func findSubCmd(subs []SubCommand, name string) *SubCommand {
	for i := range subs {
		if subs[i].Name == name {
			return &subs[i]
		}
	}
	return nil
}

// TestCatalogHasJJv044CommandSurface checks that the jj 0.44.0 additions are
// present in the static catalog with the expected aliases, input
// requirements, and interactive markers, and that the removed `git clone
// --fetch-tags` option is not offered.
func TestCatalogHasJJv044CommandSurface(t *testing.T) {
	m := newModel()

	// jj absorb -i/--interactive and --tool must be marked interactive.
	focusCommand(t, &m, "absorb", "")
	absorbFlags := m.currentFlags()
	if f := findFlag(absorbFlags, "-i"); f == nil || !f.Interactive {
		t.Errorf("absorb -i missing or not marked Interactive: %+v", f)
	}
	if f := findFlag(absorbFlags, "--tool"); f == nil || !f.Interactive || !f.RequiresInput {
		t.Errorf("absorb --tool missing, not Interactive, or not RequiresInput: %+v", f)
	}

	// jj file search --name-only and -n/--line-number.
	focusCommand(t, &m, "file", "search")
	searchFlags := m.currentFlags()
	if f := findFlag(searchFlags, "--name-only"); f == nil {
		t.Error("file search --name-only missing")
	}
	if f := findFlag(searchFlags, "-n"); f == nil {
		t.Error("file search -n (--line-number) missing")
	}

	// jj run --passthrough, --ignore-changes, --ignore-errors.
	focusCommand(t, &m, "run", "")
	runFlags := m.currentFlags()
	for _, v := range []string{"--passthrough", "--ignore-changes", "--ignore-errors"} {
		if f := findFlag(runFlags, v); f == nil {
			t.Errorf("run %s missing", v)
		}
	}

	// jj git push --allow-conflicts, and --tag/--all still present.
	focusCommand(t, &m, "git", "push")
	pushFlags := m.currentFlags()
	if f := findFlag(pushFlags, "--allow-conflicts"); f == nil {
		t.Error("git push --allow-conflicts missing")
	}
	if f := findFlag(pushFlags, "--tag"); f == nil || !f.RequiresInput {
		t.Errorf("git push --tag missing or not RequiresInput: %+v", f)
	}
	if f := findFlag(pushFlags, "--all"); f == nil {
		t.Error("git push --all missing")
	}

	// jj git clone --tag, and --fetch-tags must not exist.
	focusCommand(t, &m, "git", "clone")
	cloneFlags := m.currentFlags()
	if f := findFlag(cloneFlags, "--tag"); f == nil || !f.RequiresInput {
		t.Errorf("git clone --tag missing or not RequiresInput: %+v", f)
	}
	if f := findFlag(cloneFlags, "--fetch-tags"); f != nil {
		t.Error("git clone --fetch-tags should not be offered (removed in jj 0.44.0)")
	}

	// jj tag track / untrack subcommands.
	focusCommand(t, &m, "tag", "")
	var tagCmd *Command
	for ci := range m.categories {
		for cmi := range m.categories[ci].Commands {
			if m.categories[ci].Commands[cmi].Name == "tag" {
				tagCmd = &m.categories[ci].Commands[cmi]
			}
		}
	}
	if tagCmd == nil {
		t.Fatal("tag command not found")
	}
	if findSubCmd(tagCmd.SubCmds, "track") == nil {
		t.Error("tag track subcommand missing")
	}
	if findSubCmd(tagCmd.SubCmds, "untrack") == nil {
		t.Error("tag untrack subcommand missing")
	}
}
