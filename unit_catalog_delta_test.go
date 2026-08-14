package main

import (
	"strings"
	"testing"
)

// findCommand returns the command with the given Name from the category with
// the given name, or nil.
func findCommand(cats []Category, catName, cmdName string) *Command {
	for ci := range cats {
		if cats[ci].Name != catName {
			continue
		}
		for cmi := range cats[ci].Commands {
			if cats[ci].Commands[cmi].Name == cmdName {
				return &cats[ci].Commands[cmi]
			}
		}
	}
	return nil
}

// indexOfCommand returns the index of the command with the given Name in
// cmds, or -1.
func indexOfCommand(cmds []Command, name string) int {
	for i := range cmds {
		if cmds[i].Name == name {
			return i
		}
	}
	return -1
}

// indexOfSubCmd returns the index of the subcommand with the given Name in
// subs, or -1.
func indexOfSubCmd(subs []SubCommand, name string) int {
	for i := range subs {
		if subs[i].Name == name {
			return i
		}
	}
	return -1
}

// categoryIndex returns the index of the category with the given name in
// cats, failing the test if absent.
func categoryIndex(t *testing.T, cats []Category, name string) int {
	t.Helper()
	for i := range cats {
		if cats[i].Name == name {
			return i
		}
	}
	t.Fatalf("category %q not found", name)
	return -1
}

// TestCatalogDeltaCategoryPlacement checks that the newly added command trees
// land in their specified categories with exact positions: bench/debug in
// Tools, version in Tools immediately before help, git colocation under
// Setup's git command with root immediately after init, and git remote
// set-url under Sync's git command.
func TestCatalogDeltaCategoryPlacement(t *testing.T) {
	cats := loadCategories()

	if c := findCommand(cats, "View", "version"); c != nil {
		t.Error("version must not be in View (moved to Tools)")
	}
	if c := findCommand(cats, "Tools", "bench"); c == nil {
		t.Error("bench missing from Tools category")
	}
	if c := findCommand(cats, "Tools", "debug"); c == nil {
		t.Error("debug missing from Tools category")
	}
	if c := findCommand(cats, "View", "bench"); c != nil {
		t.Error("bench must not be in View")
	}

	tools := cats[categoryIndex(t, cats, "Tools")]
	versionIdx := indexOfCommand(tools.Commands, "version")
	helpIdx := indexOfCommand(tools.Commands, "help")
	if versionIdx < 0 {
		t.Error("version missing from Tools category")
	} else if helpIdx != versionIdx+1 {
		t.Errorf("version must be immediately before help in Tools (version=%d, help=%d)", versionIdx, helpIdx)
	}

	setupGit := findCommand(cats, "Setup", "git")
	if setupGit == nil {
		t.Fatal("Setup git command missing")
	}
	for _, sub := range []string{"colocation enable", "colocation disable", "colocation status", "root"} {
		if findSubCmd(setupGit.SubCmds, sub) == nil {
			t.Errorf("Setup git %s subcommand missing", sub)
		}
	}
	if initIdx, rootIdx := indexOfSubCmd(setupGit.SubCmds, "init"), indexOfSubCmd(setupGit.SubCmds, "root"); initIdx < 0 || rootIdx != initIdx+1 {
		t.Errorf("Setup git root must be immediately after init (init=%d, root=%d)", initIdx, rootIdx)
	}

	syncGit := findCommand(cats, "Sync", "git")
	if syncGit == nil {
		t.Fatal("Sync git command missing")
	}
	if findSubCmd(syncGit.SubCmds, "remote set-url") == nil {
		t.Error("Sync git remote set-url subcommand missing")
	}
}

// TestCatalogDeltaBenchTree checks the bench command tree and its flags.
func TestCatalogDeltaBenchTree(t *testing.T) {
	cats := loadCategories()
	bench := findCommand(cats, "Tools", "bench")
	if bench == nil {
		t.Fatal("bench missing")
	}
	for _, sub := range []string{"common-ancestors", "is-ancestor", "resolve-prefix", "revset"} {
		if findSubCmd(bench.SubCmds, sub) == nil {
			t.Errorf("bench %s subcommand missing", sub)
		}
	}
	if len(bench.SubCmds) != 4 {
		t.Errorf("bench has %d subcommands, want 4", len(bench.SubCmds))
	}

	revset := findSubCmd(bench.SubCmds, "revset")
	for _, v := range []string{"-f", "-s", "-b", "--sample-size"} {
		if findFlag(revset.Flags, v) == nil {
			t.Errorf("bench revset %s missing", v)
		}
	}
	if f := findFlag(revset.Flags, "-f"); f == nil || !f.RequiresInput {
		t.Errorf("bench revset -f (--file) missing or not RequiresInput")
	}
}

// TestCatalogDeltaDebugTree checks the debug tree: 23 flat entries covering
// the top-level tools plus the object/watchman subcommands, with the
// representative flags and args of each shape.
func TestCatalogDeltaDebugTree(t *testing.T) {
	cats := loadCategories()
	debug := findCommand(cats, "Tools", "debug")
	if debug == nil {
		t.Fatal("debug missing")
	}
	want := []string{
		"copy-detection", "fileset", "index", "index-changed-paths", "init-simple",
		"local-working-copy",
		"object commit", "object file", "object operation", "object symlink", "object tree", "object view",
		"reindex", "revset", "snapshot", "stacked-table", "template", "tree",
		"watchman status", "watchman query-clock", "watchman query-changed-files", "watchman reset-clock",
		"working-copy",
	}
	if len(debug.SubCmds) != len(want) {
		t.Errorf("debug has %d subcommands, want %d", len(debug.SubCmds), len(want))
	}
	for _, sub := range want {
		if findSubCmd(debug.SubCmds, sub) == nil {
			t.Errorf("debug %s subcommand missing", sub)
		}
	}

	indexChanged := findSubCmd(debug.SubCmds, "index-changed-paths")
	if f := findFlag(indexChanged.Flags, "-n"); f == nil || !f.RequiresInput {
		t.Error("debug index-changed-paths -n (--limit) missing or not RequiresInput")
	}
	tree := findSubCmd(debug.SubCmds, "tree")
	if f := findFlag(tree.Flags, "--id"); f == nil || !f.RequiresInput {
		t.Error("debug tree --id missing or not RequiresInput")
	}
	if f := findFlag(tree.Flags, "-r"); f == nil || !flagConflictsPair(*f, Flag{Value: "--id"}) {
		t.Error("debug tree -r must conflict with --id")
	}
	revset := findSubCmd(debug.SubCmds, "revset")
	if findFlag(revset.Flags, "--no-resolve") == nil || findFlag(revset.Flags, "--no-optimize") == nil {
		t.Error("debug revset --no-resolve/--no-optimize missing")
	}
	view := findSubCmd(debug.SubCmds, "object view")
	if f := findFlag(view.Flags, "--op"); f == nil || !f.RequiresInput {
		t.Error("debug object view --op missing or not RequiresInput")
	}
}

// flagConflictsPair reports whether either flag declares a conflict with the
// other (delegates to flagsConflict).
func flagConflictsPair(a, b Flag) bool {
	return flagsConflict(a, b)
}

// TestCatalogDeltaAlternateSpellings checks that alternates are retained in
// metadata, shown in the docs label, and never emitted by the command builder.
func TestCatalogDeltaAlternateSpellings(t *testing.T) {
	m := newModel()
	focusCommand(t, &m, "duplicate", "")
	onto := findFlag(m.currentFlags(), "-o")
	if onto == nil {
		t.Fatal("duplicate -o (--onto) missing")
	}
	if len(onto.Alternates) != 1 || onto.Alternates[0] != "-d" {
		t.Errorf("duplicate onto Alternates = %v, want [-d]", onto.Alternates)
	}
	if got := flagDocsLabel(*onto); got != "-o, --onto <REVSETS> [aliases: --destination, -d]" {
		t.Errorf("duplicate onto flagDocsLabel = %q", got)
	}

	focusCommand(t, &m, "run", "")
	root := findFlag(m.currentFlags(), "--root")
	if root == nil {
		t.Fatal("run --root missing")
	}
	if len(root.Alternates) != 0 {
		t.Errorf("run root Alternates = %v, want none (run -x is its own flag)", root.Alternates)
	}
	exec := findFlag(m.currentFlags(), "-x")
	if exec == nil {
		t.Fatal("run -x missing")
	}
	if exec.Name != "exec" || !exec.ShortOnly {
		t.Errorf("run -x = Name %q ShortOnly %v, want exec/true", exec.Name, exec.ShortOnly)
	}
	// Short-only flag: docs and bar show just -x — no invented --exec spelling.
	if got := flagDocsLabel(*exec); got != "-x" {
		t.Errorf("run -x flagDocsLabel = %q, want %q", got, "-x")
	}
	if got := flagLongForm(*exec); got != "-x" {
		t.Errorf("run -x flagLongForm = %q, want %q", got, "-x")
	}

	// The bar and exec forms always emit Value, never an alternate.
	focusCommand(t, &m, "squash", "")
	onto = findFlag(m.currentFlags(), "-o")
	if onto == nil {
		t.Fatal("squash -o missing")
	}
	if got := flagBarForm(*onto); got != "-o" {
		t.Errorf("squash onto flagBarForm = %q, want -o", got)
	}
	if got := flagExecForm(*onto); got != "-o" {
		t.Errorf("squash onto flagExecForm = %q, want -o", got)
	}
}

// TestCatalogDeltaLogRevisionConstruction pins the corrected `log --revision`
// option name and checks the composed command emits the primary spelling -r.
func TestCatalogDeltaLogRevisionConstruction(t *testing.T) {
	m := newModel()
	focusCommand(t, &m, "log", "")
	rev := findFlag(m.currentFlags(), "-r")
	if rev == nil {
		t.Fatal("log -r missing")
	}
	if rev.Name != "revision" {
		t.Errorf("log -r flag Name = %q, want %q (completion spells it --revision)", rev.Name, "revision")
	}
	if got := flagLongForm(*rev); got != "--revision" {
		t.Errorf("log -r flagLongForm = %q, want --revision", got)
	}

	selectFlag(t, &m, "-r", "main")
	short, _ := m.buildCommandStrings()
	if want := "jj log -r main"; short != want {
		t.Errorf("log command = %q, want %q", short, want)
	}
}

// TestCatalogDeltaShortAliasesEmitted checks the deterministic execution
// spelling for flags that gained (or already had) a short form.
func TestCatalogDeltaShortAliasesEmitted(t *testing.T) {
	cases := []struct {
		cmd, sub string
		value    string
	}{
		{"log", "", "-G"},           // --no-graph
		{"evolog", "", "-G"},        // --no-graph
		{"fix", "", "-s"},           // --source
		{"fix", "", "-a"},           // --all-lines
		{"squash", "", "-u"},        // --use-destination-message
		{"resolve", "", "-l"},       // --list
		{"interdiff", "", "-f"},     // --from
		{"interdiff", "", "-t"},     // --to
		{"restore", "", "-f"},       // --from
		{"restore", "", "-c"},       // --changes-in
		{"git", "fetch", "-b"},      // --branch
		{"git", "fetch", "-t"},      // --tag
		{"git", "push", "-b"},       // --bookmark
		{"git", "push", "-t"},       // --tag
		{"git", "push", "-o"},       // --option
		{"git", "clone", "-t"},      // --tag
		{"operation", "diff", "-f"}, // --from
		{"operation", "diff", "-t"}, // --to
		{"file", "search", "-p"},    // --pattern
		{"bookmark", "set", "-B"},   // --allow-backwards
		{"tag", "list", "-t"},       // --tracked
	}
	for _, tc := range cases {
		t.Run(tc.cmd+"/"+tc.sub+"/"+tc.value, func(t *testing.T) {
			m := newModel()
			focusCommand(t, &m, tc.cmd, tc.sub)
			if f := findFlag(m.currentFlags(), tc.value); f == nil {
				t.Errorf("%s %s flag %s missing", tc.cmd, tc.sub, tc.value)
			}
		})
	}
}

// TestCatalogDeltaNewFlags checks the flags added to existing commands.
func TestCatalogDeltaNewFlags(t *testing.T) {
	m := newModel()

	focusCommand(t, &m, "git", "clone")
	for _, v := range []string{"--remote", "-b", "--object-hash", "--no-colocate"} {
		if f := findFlag(m.currentFlags(), v); f == nil {
			t.Errorf("git clone %s missing", v)
		}
	}
	colocate := findFlag(m.currentFlags(), "--colocate")
	noColocate := findFlag(m.currentFlags(), "--no-colocate")
	if colocate == nil || noColocate == nil || !flagsConflict(*colocate, *noColocate) {
		t.Error("git clone --colocate and --no-colocate must conflict")
	}

	focusCommand(t, &m, "git", "init")
	if f := findFlag(m.currentFlags(), "--object-hash"); f == nil {
		t.Error("git init --object-hash missing")
	}
	gitRepo := findFlag(m.currentFlags(), "--git-repo")
	objectHash := findFlag(m.currentFlags(), "--object-hash")
	if gitRepo == nil || objectHash == nil || !flagsConflict(*gitRepo, *objectHash) {
		t.Error("git init --git-repo and --object-hash must conflict")
	}

	focusCommand(t, &m, "git", "remote add")
	if f := findFlag(m.currentFlags(), "--push-url"); f == nil || !f.RequiresInput {
		t.Error("git remote add --push-url missing or not RequiresInput")
	}

	focusCommand(t, &m, "git", "remote set-url")
	if f := findFlag(m.currentFlags(), "--push"); f == nil || !f.RequiresInput {
		t.Error("git remote set-url --push missing or not RequiresInput")
	}
	if f := findFlag(m.currentFlags(), "--fetch"); f == nil || !f.RequiresInput {
		t.Error("git remote set-url --fetch missing or not RequiresInput")
	}

	focusCommand(t, &m, "describe", "")
	if f := findFlag(m.currentFlags(), "--stdin"); f == nil {
		t.Error("describe --stdin missing")
	}
	if f := findFlag(m.currentFlags(), "--editor"); f == nil {
		t.Error("describe --editor missing")
	}
	msg := findFlag(m.currentFlags(), "-m")
	stdin := findFlag(m.currentFlags(), "--stdin")
	if msg == nil || stdin == nil || !flagsConflict(*msg, *stdin) {
		t.Error("describe -m and --stdin must conflict")
	}

	focusCommand(t, &m, "new", "")
	if f := findFlag(m.currentFlags(), "--edit"); f == nil {
		t.Error("new --edit missing")
	}
	edit := findFlag(m.currentFlags(), "--edit")
	noEdit := findFlag(m.currentFlags(), "--no-edit")
	if edit == nil || noEdit == nil || !flagsConflict(*edit, *noEdit) {
		t.Error("new --edit and --no-edit must conflict")
	}

	focusCommand(t, &m, "split", "")
	for _, v := range []string{"-o", "-A", "-B", "-p", "--editor"} {
		if f := findFlag(m.currentFlags(), v); f == nil {
			t.Errorf("split %s missing", v)
		}
	}
	parallel := findFlag(m.currentFlags(), "-p")
	onto := findFlag(m.currentFlags(), "-o")
	if parallel == nil || onto == nil || !flagsConflict(*parallel, *onto) {
		t.Error("split -p (--parallel) and -o (--onto) must conflict")
	}

	focusCommand(t, &m, "tag", "list")
	if f := findFlag(m.currentFlags(), "--remote"); f == nil || !f.RequiresInput {
		t.Error("tag list --remote missing or not RequiresInput")
	}
	allRemotes := findFlag(m.currentFlags(), "-a")
	tracked := findFlag(m.currentFlags(), "-t")
	if allRemotes == nil || tracked == nil || !flagsConflict(*allRemotes, *tracked) {
		t.Error("tag list -a and -t must conflict")
	}

	focusCommand(t, &m, "workspace", "root")
	if f := findFlag(m.currentFlags(), "--name"); f == nil || !f.RequiresInput {
		t.Error("workspace root --name missing or not RequiresInput")
	}

	focusCommand(t, &m, "config", "list")
	if f := findFlag(m.currentFlags(), "--include-overridden"); f == nil {
		t.Error("config list --include-overridden missing")
	}
	if f := findFlag(m.currentFlags(), "-T"); f == nil || !f.RequiresInput {
		t.Error("config list -T (--template) missing or not RequiresInput")
	}
}

// TestCatalogDeltaDeprecatedOptions checks that completion-defined deprecated
// or error-raising options are still offered, with input requirements and
// their original descriptions preserved.
func TestCatalogDeltaDeprecatedOptions(t *testing.T) {
	m := newModel()

	focusCommand(t, &m, "bisect", "run")
	command := findFlag(m.currentFlags(), "--command")
	if command == nil {
		t.Fatal("bisect run --command missing")
	}
	if !command.RequiresInput || command.InputType != "COMMAND" {
		t.Errorf("bisect run --command RequiresInput/InputType = %v/%q, want true/COMMAND", command.RequiresInput, command.InputType)
	}
	if !strings.Contains(command.Description, "Deprecated") {
		t.Errorf("bisect run --command description = %q, want deprecated note", command.Description)
	}

	focusCommand(t, &m, "restore", "")
	rev := findFlag(m.currentFlags(), "-r")
	if rev == nil {
		t.Fatal("restore -r missing")
	}
	if !rev.RequiresInput || rev.InputType != "REVSET" {
		t.Errorf("restore -r RequiresInput/InputType = %v/%q, want true/REVSET", rev.RequiresInput, rev.InputType)
	}
	if !strings.Contains(rev.Description, "DO NOT USE") {
		t.Errorf("restore -r description = %q, want DO NOT USE note", rev.Description)
	}
	if got := flagLongForm(*rev); got != "--revision" {
		t.Errorf("restore -r flagLongForm = %q, want --revision", got)
	}
}

// TestCatalogDeltaConflicts checks representative conflict declarations added
// from the completion data.
func TestCatalogDeltaConflicts(t *testing.T) {
	m := newModel()

	focusCommand(t, &m, "git", "push")
	deleted := findFlag(m.currentFlags(), "--deleted")
	bookmark := findFlag(m.currentFlags(), "-b")
	if deleted == nil || bookmark == nil || !flagsConflict(*deleted, *bookmark) {
		t.Error("git push --deleted and -b must conflict")
	}

	focusCommand(t, &m, "file", "search")
	nameOnly := findFlag(m.currentFlags(), "--name-only")
	lineNum := findFlag(m.currentFlags(), "-n")
	if nameOnly == nil || lineNum == nil || !flagsConflict(*nameOnly, *lineNum) {
		t.Error("file search --name-only and -n must conflict")
	}

	focusCommand(t, &m, "run", "")
	ignoreChanges := findFlag(m.currentFlags(), "--ignore-changes")
	restoreDesc := findFlag(m.currentFlags(), "--restore-descendants")
	if ignoreChanges == nil || restoreDesc == nil || !flagsConflict(*ignoreChanges, *restoreDesc) {
		t.Error("run --ignore-changes and --restore-descendants must conflict")
	}

	focusCommand(t, &m, "operation", "diff")
	from := findFlag(m.currentFlags(), "-f")
	operation := findFlag(m.currentFlags(), "--operation")
	if from == nil || operation == nil || !flagsConflict(*from, *operation) {
		t.Error("op diff -f and --operation must conflict")
	}

	focusCommand(t, &m, "squash", "")
	useDest := findFlag(m.currentFlags(), "-u")
	message := findFlag(m.currentFlags(), "-m")
	if useDest == nil || message == nil || !flagsConflict(*useDest, *message) {
		t.Error("squash -u and -m must conflict")
	}

	focusCommand(t, &m, "sparse", "set")
	remove := findFlag(m.currentFlags(), "--remove")
	clear := findFlag(m.currentFlags(), "--clear")
	if remove == nil || clear == nil || !flagsConflict(*remove, *clear) {
		t.Error("sparse set --remove and --clear must conflict")
	}
}

// TestCatalogDeltaInteractiveMarkers checks that new interactive options route
// through the existing terminal-handoff predicate and that --stdin suppresses
// describe's editor fallback.
func TestCatalogDeltaInteractiveMarkers(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, m *mainModel)
		want  bool
	}{
		{"commit --tool opens diff editor", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "commit", "")
			selectFlag(t, m, "--tool", "meld")
		}, true},
		{"commit --editor opens editor", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "commit", "")
			selectFlag(t, m, "--editor", "")
		}, true},
		{"describe --stdin is direct", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "describe", "")
			selectFlag(t, m, "--stdin", "")
		}, false},
		{"describe --editor opens editor", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "describe", "")
			selectFlag(t, m, "--editor", "")
		}, true},
		{"restore -i is interactive", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "restore", "")
			selectFlag(t, m, "-i", "")
		}, true},
		{"restore --tool is interactive", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "restore", "")
			selectFlag(t, m, "--tool", "meld")
		}, true},
		{"split --editor opens editor", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "split", "")
			selectFlag(t, m, "--editor", "")
		}, true},
		{"squash --editor opens editor", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "squash", "")
			selectFlag(t, m, "--editor", "")
		}, true},
		{"bench and debug are never interactive", func(t *testing.T, m *mainModel) {
			focusCommand(t, m, "bench", "revset")
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
