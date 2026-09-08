package main

import (
	"os"
	"strings"
	"testing"
)

// Drives the real open/run/wait/read/dispose cycle to verify multiplexer behaviour.
func TestLiveMuxRoundTrip(t *testing.T) {
	if os.Getenv("JUTSU_LIVE_MUX") != "1" {
		t.Skip("set JUTSU_LIVE_MUX=1 to run against the live multiplexer")
	}
	if currentMux() == nil {
		t.Fatal("currentMux() = nil: not inside a multiplexer we can drive")
	}
	t.Logf("tmux=%v herdr=%v", inTmux(), inHerdr())

	// The handoff now carries literal argv, so a shell construct has to be
	// asked for explicitly (`sh -c`) — which is also what proves nothing else
	// is being re-split by a shell on the way through.
	run := func(t *testing.T, argv []string, plain bool) execInteractiveResultMsg {
		t.Helper()
		cmd := executeInMux(jjInvocation{argv: argv, short: strings.Join(argv, " ")}, plain)
		if cmd == nil {
			t.Fatal("executeInMux returned nil")
		}
		msg := cmd()
		res, ok := msg.(execInteractiveResultMsg)
		if !ok {
			t.Fatalf("got %T (%+v), want execInteractiveResultMsg", msg, msg)
		}
		return res
	}

	res := run(t, []string{"sh", "-c", "echo hello-from-pane; echo second-line"}, true)
	t.Logf("split pane: err=%v output=%q", res.err, res.output)
	if res.err != nil {
		t.Errorf("unexpected err: %v", res.err)
	}
	if res.output != "hello-from-pane\nsecond-line" {
		t.Errorf("output = %q", res.output)
	}

	// Failure path: `exit` must not take the wrapper script down with it (it
	// would hang the wait forever).
	res = run(t, []string{"sh", "-c", "echo oops >&2; exit 3"}, true)
	t.Logf("fail path: err=%v output=%q", res.err, res.output)
	if res.err == nil || res.err.Error() != "exit status 3" {
		t.Errorf("err = %v, want exit status 3", res.err)
	}
	if res.output != "oops" {
		t.Errorf("output = %q, want %q", res.output, "oops")
	}

	// Editor-shaped path: a whole tab/window rather than a split pane.
	res = run(t, []string{"echo", "from-a-tab"}, false)
	t.Logf("tab/window path: err=%v output=%q", res.err, res.output)
	if res.err != nil || res.output != "from-a-tab" {
		t.Errorf("tab/window path: err=%v output=%q", res.err, res.output)
	}

	// A value full of shell syntax must reach the command as one argument,
	// unexpanded — the whole point of quoting each argv element in the wrapper
	// script instead of pasting a command line together.
	const nasty = `a b $HOME "q" 'r' ; rm -rf x \ y`
	res = run(t, []string{"printf", "%s", nasty}, true)
	t.Logf("literal arg: err=%v output=%q", res.err, res.output)
	if res.err != nil || res.output != nasty {
		t.Errorf("literal arg: err=%v output=%q, want %q", res.err, res.output, nasty)
	}

	// The handoff terminal must come up with the user's own environment. It
	// only does if a real shell started and sourced their startup files —
	// running the script *as* the pane's command skips that, which is how
	// `jj describe` ended up with no $EDITOR and fell back to jj's built-in
	// default editor.
	if want := os.Getenv("EDITOR"); want != "" {
		res = run(t, []string{"printenv", "EDITOR"}, false)
		t.Logf("env path: err=%v output=%q want=%q", res.err, res.output, want)
		if res.output != want {
			t.Errorf("EDITOR in handoff terminal = %q, want %q", res.output, want)
		}
	}
}
