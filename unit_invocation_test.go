package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// display returns the invocation's two display strings — the pair most of the
// display assertions in these tests were written against.
func (inv jjInvocation) display() (short, long string) { return inv.short, inv.long }

// nastyValues are the input values a shell would mangle: word splitting,
// variable expansion, quote removal, command separation, escape processing.
// (No newline case: the inputs pane is a single-line textinput, which strips
// them before a value ever reaches the invocation.)
var nastyValues = []struct {
	name  string
	value string
}{
	{"spaces", "Martin von Zweigbergk"},
	{"dollar sign", "cost is $HOME $(id) ${PATH}"},
	{"double quotes", `say "hello" now`},
	{"single quotes", `it's a 'quoted' word`},
	{"semicolon", "fix bug; rm -rf /"},
	{"backslash", `path\to\thing \n \\ \'`},
	{"mixed", `a b "c" 'd' ;$e\f&g|h<i>j*k?l`},
}

// TestBuildInvocationKeepsValuesLiteral is the core guarantee: whatever the
// user typed lands in argv as exactly one element, byte for byte.
func TestBuildInvocationKeepsValuesLiteral(t *testing.T) {
	for _, tc := range nastyValues {
		t.Run(tc.name, func(t *testing.T) {
			m := newModel()
			focusCommand(t, &m, "describe", "")
			selectFlag(t, &m, "-m", tc.value)

			inv := m.buildInvocation()
			want := []string{"jj", "describe", "-m", tc.value}
			if !slices.Equal(inv.argv, want) {
				t.Errorf("argv = %q, want %q", inv.argv, want)
			}
		})
	}
}

// TestBuildInvocationVariadicArgsExpandOnSpaces: a variadic positional accepts
// several values in one field, each a separate argv element — the original
// behavior, e.g. `jj bookmark create foo bar` or `jj new @ main`.
func TestBuildInvocationVariadicArgsExpandOnSpaces(t *testing.T) {
	cases := []struct {
		name    string
		cmd     []string // focusCommand(cmdName, subName)
		argName string
		value   string
		want    []string
		disp    string // expected short display
	}{
		{"bookmark single", []string{"bookmark", "create"}, "NAMES", "v0.4.0",
			[]string{"jj", "bookmark", "create", "v0.4.0"}, "jj b c v0.4.0"},
		{"bookmark two", []string{"bookmark", "create"}, "NAMES", "foo bar",
			[]string{"jj", "bookmark", "create", "foo", "bar"}, "jj b c foo bar"},
		{"bookmark three", []string{"bookmark", "create"}, "NAMES", "@ main feature",
			[]string{"jj", "bookmark", "create", "@", "main", "feature"}, "jj b c @ main feature"},
		{"bookmark extra spaces collapse", []string{"bookmark", "create"}, "NAMES", "a   b",
			[]string{"jj", "bookmark", "create", "a", "b"}, "jj b c a b"},
		{"revset variadic", []string{"new", ""}, "REVSETS", "@ main",
			[]string{"jj", "new", "@", "main"}, "jj new @ main"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := newModel()
			focusCommand(t, &m, c.cmd[0], c.cmd[1])
			setArg(&m, c.argName, c.value)

			inv := m.buildInvocation()
			if !slices.Equal(inv.argv, c.want) {
				t.Errorf("argv = %q, want %q", inv.argv, c.want)
			}
			// Display shows the user's field verbatim (joined), not re-split.
			if got := inv.short; got != c.disp {
				t.Errorf("short = %q, want %q", got, c.disp)
			}
		})
	}
}

// TestBuildInvocationNonVariadicArgKeepsSpaces: a non-variadic positional arg
// (PATH on `file show`) keeps its value as exactly one argv element — a space
// is part of the value, never a separator.
func TestBuildInvocationNonVariadicArgKeepsSpaces(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{"single token", "src/main.go"},
		{"path with space", "my project/notes.txt"},
		{"value with shell syntax", `a b $HOME "q" 'r' ; rm -rf x`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newModel()
			focusCommand(t, &m, "file", "annotate")
			setArg(&m, "PATH", tc.value)

			inv := m.buildInvocation()
			want := []string{"jj", "file", "annotate", tc.value}
			if !slices.Equal(inv.argv, want) {
				t.Errorf("argv = %q, want %q", inv.argv, want)
			}
		})
	}
}

// TestBuildInvocationGlobalFlagValueIsLiteral covers the global-options path,
// which splices its args in before the subcommand.
func TestBuildInvocationGlobalFlagValueIsLiteral(t *testing.T) {
	m := newModel()
	focusCommand(t, &m, "log", "")
	selectGlobalFlag(t, &m, "-R", "/tmp/my repo; touch pwned")

	inv := m.buildInvocation()
	want := []string{"jj", "-R", "/tmp/my repo; touch pwned", "log"}
	if !slices.Equal(inv.argv, want) {
		t.Errorf("argv = %q, want %q", inv.argv, want)
	}
}

// TestDisplayQuotesNeverReachArgv pins the display/execution split: a
// NeedsQuotes flag reads quoted in the bar, but jj gets the bare value.
func TestDisplayQuotesNeverReachArgv(t *testing.T) {
	m := newModel()
	focusCommand(t, &m, "describe", "")
	selectFlag(t, &m, "-m", "fix the thing")

	inv := m.buildInvocation()
	short, long := inv.display()
	// short uses the command alias, long the full name — both quote the value
	// for readability only.
	if want := `jj desc -m "fix the thing"`; short != want {
		t.Errorf("short = %q, want %q", short, want)
	}
	if want := `jj describe -m "fix the thing"`; long != want {
		t.Errorf("long = %q, want %q", long, want)
	}
	if got := inv.argv[len(inv.argv)-1]; got != "fix the thing" {
		t.Errorf("argv value = %q, want unquoted %q", got, "fix the thing")
	}
}

// TestBuildInvocationDisplayStaysReadable guards the readable short/long forms
// (aliases in short, full names in long) against argv changes.
func TestBuildInvocationDisplayStaysReadable(t *testing.T) {
	m := newModel()
	focusCommand(t, &m, "describe", "")

	inv := m.buildInvocation()
	short, long := inv.display()
	if want := "jj desc"; short != want {
		t.Errorf("short = %q, want %q (alias)", short, want)
	}
	if want := "jj describe"; long != want {
		t.Errorf("long = %q, want %q (full name)", long, want)
	}
	// The alias is display-only: argv always spells the full command name,
	// which is what jj is guaranteed to accept.
	if want := []string{"jj", "describe"}; !slices.Equal(inv.argv, want) {
		t.Errorf("argv = %q, want %q", inv.argv, want)
	}
}

func TestBuildInvocationPromotedCommandGroupsSplitArgv(t *testing.T) {
	cases := []struct {
		name        string
		cmd         string
		sub         string
		setup       func(*testing.T, *mainModel)
		wantArgv    []string
		wantDisplay string
	}{
		{
			name: "git remote add",
			cmd:  "git remote",
			sub:  "add",
			setup: func(t *testing.T, m *mainModel) {
				setArg(m, "REMOTE", "origin")
				setArg(m, "URL", "git@example.com:me/repo.git")
			},
			wantArgv:    []string{"jj", "git", "remote", "add", "origin", "git@example.com:me/repo.git"},
			wantDisplay: "jj git remote add origin git@example.com:me/repo.git",
		},
		{
			name:        "git colocation enable",
			cmd:         "git colocation",
			sub:         "enable",
			wantArgv:    []string{"jj", "git", "colocation", "enable"},
			wantDisplay: "jj git colocation enable",
		},
		{
			name: "debug object view",
			cmd:  "debug object",
			sub:  "view",
			setup: func(t *testing.T, m *mainModel) {
				selectFlag(t, m, "--op", "operation id")
			},
			wantArgv:    []string{"jj", "debug", "object", "view", "--op", "operation id"},
			wantDisplay: "jj debug object view --op operation id",
		},
		{
			name:        "debug watchman status",
			cmd:         "debug watchman",
			sub:         "status",
			wantArgv:    []string{"jj", "debug", "watchman", "status"},
			wantDisplay: "jj debug watchman status",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newModel()
			focusCommand(t, &m, tc.cmd, tc.sub)
			if tc.setup != nil {
				tc.setup(t, &m)
			}

			inv := m.buildInvocation()
			if !slices.Equal(inv.argv, tc.wantArgv) {
				t.Errorf("argv = %q, want %q", inv.argv, tc.wantArgv)
			}
			if inv.short != tc.wantDisplay || inv.long != tc.wantDisplay {
				t.Errorf("display = (%q, %q), want %q", inv.short, inv.long, tc.wantDisplay)
			}
		})
	}
}

// TestExecArgvForcesColor covers the captured-output path's colour injection:
// --color=always must be a global option (before the subcommand) and must not
// disturb the composed arguments.
func TestExecArgvForcesColor(t *testing.T) {
	inv := jjInvocation{argv: []string{"jj", "-R", "/tmp/r", "describe", "-m", "a b"}}
	want := []string{"jj", "--color=always", "-R", "/tmp/r", "describe", "-m", "a b"}
	if got := inv.execArgv(); !slices.Equal(got, want) {
		t.Errorf("execArgv() = %q, want %q", got, want)
	}
	// The original invocation (the one carried into a retry) is untouched.
	if want := []string{"jj", "-R", "/tmp/r", "describe", "-m", "a b"}; !slices.Equal(inv.argv, want) {
		t.Errorf("execArgv() mutated inv.argv: %q", inv.argv)
	}
	// A non-jj argv (the mux integration test's sh -c, a future helper) is
	// left exactly as-is.
	other := jjInvocation{argv: []string{"sh", "-c", "echo hi"}}
	if got := other.execArgv(); !slices.Equal(got, other.argv) {
		t.Errorf("execArgv() = %q, want it unchanged", got)
	}
}

// TestShellQuoteRoundTrip runs the generated shell snippet through a real
// POSIX shell and checks each argument arrives unchanged — the mux wrapper's
// only correctness question.
func TestShellQuoteRoundTrip(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no POSIX sh available")
	}
	for _, tc := range nastyValues {
		t.Run(tc.name, func(t *testing.T) {
			argv := []string{"printf", "%s\x1e", "before", tc.value, "after"}
			// The snippet is exactly what muxScript embeds.
			out, err := exec.Command(sh, "-c", shellQuoteArgs(argv)).Output()
			if err != nil {
				t.Fatalf("sh -c %q: %v", shellQuoteArgs(argv), err)
			}
			got := strings.Split(strings.TrimSuffix(string(out), "\x1e"), "\x1e")
			want := []string{"before", tc.value, "after"}
			if !slices.Equal(got, want) {
				t.Errorf("reconstructed %q, want %q (snippet %q)", got, want, shellQuoteArgs(argv))
			}
		})
	}
}

// TestShellQuoteEmptyAndSafe: an empty value must still be one argument, and
// ordinary tokens stay unquoted so the script remains readable.
func TestShellQuoteEmptyAndSafe(t *testing.T) {
	if got := shellQuote(""); got != "''" {
		t.Errorf("shellQuote(\"\") = %q, want %q", got, "''")
	}
	for _, safe := range []string{"jj", "describe", "-m", "--color=always", "/tmp/repo", "v0.4.0", "a@b,c+d"} {
		if got := shellQuote(safe); got != safe {
			t.Errorf("shellQuote(%q) = %q, want it unquoted", safe, got)
		}
	}
	if got := shellQuote("it's"); got != `'it'\''s'` {
		t.Errorf("shellQuote(%q) = %q", "it's", got)
	}
}

// TestMuxScriptQuotesArguments checks the wrapper script the multiplexer
// backends actually write: shell-quoted argv, subshell, exit code, sentinel.
func TestMuxScriptQuotesArguments(t *testing.T) {
	inv := jjInvocation{
		argv:  []string{"jj", "describe", "-m", `it's "done"; $now`},
		short: "jj desc -m ...",
	}
	script := muxScript(inv, "/tmp/x/exit", "notify-here\n")

	if !strings.Contains(script, `( jj describe -m 'it'\''s "done"; $now'`) {
		t.Errorf("script does not contain the quoted argv:\n%s", script)
	}
	for _, want := range []string{"echo $? > /tmp/x/exit", muxSentinel, "notify-here"} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q:\n%s", want, script)
		}
	}
}

// TestMuxScriptExecutesLiterally runs a generated wrapper script end to end in
// a real shell: the command's argument must survive, and the exit file must be
// written (the sentinel/exit-status contract the mux driver reads back).
func TestMuxScriptExecutesLiterally(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no POSIX sh available")
	}
	dir := t.TempDir()
	exitFile := filepath.Join(dir, "exit")
	const value = `a b $HOME "q" 'r' ; echo pwned \ z`

	script := muxScript(jjInvocation{argv: []string{"printf", "%s", value}}, exitFile, "")
	path := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command(sh, path).Output()
	if err != nil {
		t.Fatalf("running wrapper: %v", err)
	}
	// printf emits the one argument and nothing else, so the whole transcript
	// (minus the script's leading clear, stripped here, and its sentinel) must
	// be the value byte for byte: no expansion, no splitting, no extra command.
	text := stripANSI(string(out))
	if want := value + muxSentinel + "\n"; text != want {
		t.Errorf("wrapper output = %q, want %q", text, want)
	}
	code, err := os.ReadFile(exitFile)
	if err != nil {
		t.Fatalf("exit file: %v", err)
	}
	if strings.TrimSpace(string(code)) != "0" {
		t.Errorf("exit file = %q, want 0", code)
	}
}

// TestIsRemoteGitCommandUsesArgv: classification reads adjacent argv elements,
// so a commit message that happens to mention "git push" is not a remote op.
func TestIsRemoteGitCommandUsesArgv(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want bool
	}{
		{"push", []string{"jj", "git", "push"}, true},
		{"push with flags", []string{"jj", "git", "push", "--remote", "origin", "-b", "main"}, true},
		{"fetch", []string{"jj", "git", "fetch"}, true},
		{"clone", []string{"jj", "git", "clone", "https://example.com/repo"}, true},
		{"log", []string{"jj", "log"}, false},
		{"git init", []string{"jj", "git", "init"}, false},
		{"remote list", []string{"jj", "git", "remote", "list"}, false},
		{"message mentioning git push", []string{"jj", "describe", "-m", "docs: explain git push"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isRemoteGitCommand(jjInvocation{argv: c.argv}); got != c.want {
				t.Errorf("isRemoteGitCommand(%q) = %v, want %v", c.argv, got, c.want)
			}
		})
	}
}

// TestSelectedRemoteNameFromArgv: the retry path resolves the remote from the
// literal args, so a name with a space stays whole.
func TestSelectedRemoteNameFromArgv(t *testing.T) {
	cases := []struct {
		argv []string
		want string
	}{
		{[]string{"jj", "git", "push", "--remote", "upstream"}, "upstream"},
		{[]string{"jj", "git", "push", "--remote=upstream"}, "upstream"},
		{[]string{"jj", "git", "push", "--remote", "my remote"}, "my remote"},
		{[]string{"jj", "git", "push", "-b", "main"}, ""},
		{[]string{"jj", "git", "push"}, ""},
		{[]string{"jj", "git", "push", "--remote"}, ""}, // dangling flag, no value
	}
	for _, c := range cases {
		if got := selectedRemoteName(jjInvocation{argv: c.argv}); got != c.want {
			t.Errorf("selectedRemoteName(%q) = %q, want %q", c.argv, got, c.want)
		}
	}
}

// TestRecallPreservesLiteralValues: recalling the last command must reproduce
// the same argv and the same display text, not a reparsed approximation.
func TestRecallPreservesLiteralValues(t *testing.T) {
	const msg = `it's "done"; $HOME \ now`

	m := newModel()
	focusCommand(t, &m, "describe", "")
	selectFlag(t, &m, "-m", msg)
	m.inv = m.buildInvocation()
	want := m.inv

	m.lastCmd = m.captureSnapshot()
	m.resetCurrentFlags()
	m.inv = m.buildInvocation()
	if slices.Equal(m.inv.argv, want.argv) {
		t.Fatalf("reset did not clear the invocation: %q", m.inv.argv)
	}

	m.restoreLastCmd()
	if !slices.Equal(m.inv.argv, want.argv) {
		t.Errorf("recalled argv = %q, want %q", m.inv.argv, want.argv)
	}
	if m.inv.short != want.short || m.inv.long != want.long {
		t.Errorf("recalled display = (%q, %q), want (%q, %q)", m.inv.short, m.inv.long, want.short, want.long)
	}
}
