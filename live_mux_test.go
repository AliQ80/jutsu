package main

import (
	"os"
	"testing"
)

// TestLiveMuxRoundTrip drives the real open/run/wait/read/dispose cycle against
// whichever multiplexer is running — the only way to check the parts that are
// herdr's or tmux's behaviour rather than ours (id plumbing, how a scraped
// transcript actually comes back, whether a signal races a wait). It creates
// and destroys real panes and tabs/windows and briefly steals focus, so it is
// opt-in:
//
//	JUTSU_LIVE_MUX=1 go test -run TestLiveMuxRoundTrip -v .
//
// Worth re-running from inside each multiplexer after upgrading either.
func TestLiveMuxRoundTrip(t *testing.T) {
	if os.Getenv("JUTSU_LIVE_MUX") != "1" {
		t.Skip("set JUTSU_LIVE_MUX=1 to run against the live multiplexer")
	}
	if currentMux() == nil {
		t.Fatal("currentMux() = nil: not inside a multiplexer we can drive")
	}
	t.Logf("tmux=%v herdr=%v", inTmux(), inHerdr())

	run := func(t *testing.T, cmdStr string, plain bool) execInteractiveResultMsg {
		t.Helper()
		cmd := executeInMux(cmdStr, plain)
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

	res := run(t, "echo hello-from-pane; echo second-line", true)
	t.Logf("split pane: err=%v output=%q", res.err, res.output)
	if res.err != nil {
		t.Errorf("unexpected err: %v", res.err)
	}
	if res.output != "hello-from-pane\nsecond-line" {
		t.Errorf("output = %q", res.output)
	}

	// Failure path: exit status must surface, and `exit` must not take the
	// wrapper script down with it (it would hang the wait forever).
	res = run(t, "echo oops >&2; exit 3", true)
	t.Logf("fail path: err=%v output=%q", res.err, res.output)
	if res.err == nil || res.err.Error() != "exit status 3" {
		t.Errorf("err = %v, want exit status 3", res.err)
	}
	if res.output != "oops" {
		t.Errorf("output = %q, want %q", res.output, "oops")
	}

	// Editor-shaped path: a whole tab/window rather than a split pane.
	res = run(t, "echo from-a-tab", false)
	t.Logf("tab/window path: err=%v output=%q", res.err, res.output)
	if res.err != nil || res.output != "from-a-tab" {
		t.Errorf("tab/window path: err=%v output=%q", res.err, res.output)
	}

	// The handoff terminal must come up with the user's own environment. It
	// only does if a real shell started and sourced their startup files —
	// running the script *as* the pane's command skips that, which is how
	// `jj describe` ended up with no $EDITOR and fell back to jj's built-in
	// default editor.
	if want := os.Getenv("EDITOR"); want != "" {
		res = run(t, "printenv EDITOR", false)
		t.Logf("env path: err=%v output=%q want=%q", res.err, res.output, want)
		if res.output != want {
			t.Errorf("EDITOR in handoff terminal = %q, want %q", res.output, want)
		}
	}
}
