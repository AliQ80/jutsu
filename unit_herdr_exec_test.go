package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHerdrPaneWaitOutputInvocation pins the exact herdr CLI invocation used
// to block on a handoff terminal, without needing a live multiplexer. herdr
// renamed this subcommand from `wait output` to `pane wait-output`; a fake
// `herdr` executable on PATH records every argv it's called with so the
// assertion breaks loudly if the invocation regresses to the old form (or
// otherwise drifts) instead of silently hanging against a real herdr.
func TestHerdrPaneWaitOutputInvocation(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "herdr.log")
	fakeHerdr := writeFakeHerdr(t, logFile)

	t.Setenv("PATH", filepath.Dir(fakeHerdr)+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HERDR_PANE_ID", "pane-under-test")

	term, err := openHerdrTerminal("/tmp/does-not-need-to-exist/run.sh", true)
	if err != nil {
		t.Fatalf("openHerdrTerminal() error = %v", err)
	}
	if err := term.wait(); err != nil {
		t.Fatalf("term.wait() error = %v", err)
	}

	log, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("reading fake herdr log: %v", err)
	}

	var waitLine string
	for _, line := range strings.Split(strings.TrimSpace(string(log)), "\n") {
		if strings.HasPrefix(line, "pane wait-output ") {
			waitLine = line
			break
		}
	}
	if waitLine == "" {
		t.Fatalf("no `pane wait-output` invocation recorded; log:\n%s", log)
	}

	want := []string{
		"pane", "wait-output", "fake-pane",
		"--match", muxSentinel,
		"--source", "recent",
		"--timeout", herdrWaitTimeoutMs,
	}
	got := strings.Fields(waitLine)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("pane wait-output argv = %q, want %q", got, want)
	}

	// Guard against a regression back to the retired `wait output` form.
	if strings.Contains(string(log), "wait output ") {
		t.Errorf("found retired `wait output` invocation in log:\n%s", log)
	}
}

// writeFakeHerdr writes a stand-in `herdr` executable that logs its argv (one
// line per invocation) to logFile and answers just enough JSON to satisfy
// openHerdrTerminal's plain (split-pane) path.
func writeFakeHerdr(t *testing.T, logFile string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "herdr")
	script := `#!/bin/sh
echo "$*" >> "` + logFile + `"
case "$1 $2" in
  "pane split")
    echo '{"result":{"pane":{"pane_id":"fake-pane"}}}'
    ;;
  *)
    echo '{"result":{}}'
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("writing fake herdr: %v", err)
	}
	return path
}
