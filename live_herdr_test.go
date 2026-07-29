package main

import (
	"os"
	"testing"
)

// TestLiveHerdrRoundTrip drives the real split/run/wait/read/close cycle
// against the running herdr instance — the only way to check the parts that
// are herdr's behaviour rather than ours (id plumbing, what a scraped
// transcript actually looks like). It creates and destroys real panes and
// tabs and briefly steals focus, so it is opt-in:
//
//	JUTSU_LIVE_HERDR=1 go test -run TestLiveHerdrRoundTrip -v .
//
// Worth re-running after a herdr upgrade.
func TestLiveHerdrRoundTrip(t *testing.T) {
	if os.Getenv("JUTSU_LIVE_HERDR") != "1" {
		t.Skip("set JUTSU_LIVE_HERDR=1 to run against the live herdr instance")
	}
	if !inHerdr() {
		t.Fatal("inHerdr() = false")
	}

	cmd := executeInHerdr("echo hello-from-pane; echo second-line", true)
	if cmd == nil {
		t.Fatal("executeInHerdr returned nil")
	}
	msg := cmd()
	res, ok := msg.(execInteractiveResultMsg)
	if !ok {
		t.Fatalf("got %T (%+v), want execInteractiveResultMsg", msg, msg)
	}
	t.Logf("err=%v output=%q", res.err, res.output)
	if res.err != nil {
		t.Errorf("unexpected err: %v", res.err)
	}
	if res.output != "hello-from-pane\nsecond-line" {
		t.Errorf("output = %q", res.output)
	}

	// Failure path: exit status must surface, and `exit` must not take the
	// wrapper script down with it (it would hang the wait forever).
	msg = executeInHerdr("echo oops >&2; exit 3", true)()
	res = msg.(execInteractiveResultMsg)
	t.Logf("fail path: err=%v output=%q", res.err, res.output)
	if res.err == nil || res.err.Error() != "exit status 3" {
		t.Errorf("err = %v, want exit status 3", res.err)
	}
	if res.output != "oops" {
		t.Errorf("output = %q, want %q", res.output, "oops")
	}

	// Editor-shaped path: a new tab rather than a split pane.
	msg = executeInHerdr("echo from-a-tab", false)()
	res = msg.(execInteractiveResultMsg)
	t.Logf("tab path: err=%v output=%q", res.err, res.output)
	if res.err != nil || res.output != "from-a-tab" {
		t.Errorf("tab path: err=%v output=%q", res.err, res.output)
	}
}
