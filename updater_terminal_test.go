//go:build !windows

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/creack/pty"
)

type updaterTerminalModel struct{ mainModel }

func (m updaterTerminalModel) Init() tea.Cmd { return nil }
func (m updaterTerminalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.mainModel.Update(msg)
	m.mainModel = next.(mainModel)
	return m, cmd
}

// This subprocess executes the real Bubble Tea shutdown and Unix exec path.
// Both release images are test binaries, isolated from the development binary.
func TestUpdateTerminalChild(t *testing.T) {
	dir := os.Getenv("JUTSU_UPDATE_TERMINAL_FIXTURE")
	if dir == "" {
		t.Skip("subprocess fixture only")
	}
	executable, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	marker := filepath.Join(dir, "prepared")
	_, e = os.Stat(marker)
	updated := e == nil
	m := newModel()
	m.latestReleaseVersion = "v9.9.9"
	m.latestReleaseURL = "https://example.test/release"
	if updated {
		_ = inspectUpdateRecovery(executable)
		cwd, _ := os.Getwd()
		data, _ := json.Marshal(map[string]any{"cwd": cwd, "args": os.Args, "env": os.Getenv("JUTSU_UPDATE_LITERAL")})
		if e = os.WriteFile(filepath.Join(dir, "restarted"), data, 0600); e != nil {
			t.Fatal(e)
		}
	} else {
		prepareReleaseUpdate = func(ctx context.Context, _ updateSource, target, tag string, progress func(string)) (*updateTransaction, error) {
			tx, e := beginUpdate(target)
			if e != nil {
				return nil, e
			}
			binary, e := os.ReadFile(executable)
			if e != nil {
				tx.cleanup()
				return nil, e
			}
			progress("Verifying fixture…")
			if e = tx.stage(ctx, binary); e != nil {
				tx.cleanup()
				return nil, e
			}
			if e = os.WriteFile(marker, []byte("ready"), 0600); e != nil {
				tx.cleanup()
				return nil, e
			}
			return tx, nil
		}
	}
	final, e := tea.NewProgram(updaterTerminalModel{m}).Run()
	if e != nil {
		t.Fatal(e)
	}
	result := final.(updaterTerminalModel)
	if result.updateRestart != nil {
		if e = restartUpdated(result.updateRestart); e != nil {
			t.Fatal(e)
		}
	}
}

type updateTerminalOutput struct {
	sync.Mutex
	buffer bytes.Buffer
}

func (b *updateTerminalOutput) Write(p []byte) (int, error) {
	b.Lock()
	defer b.Unlock()
	return b.buffer.Write(p)
}
func (b *updateTerminalOutput) text() string { b.Lock(); defer b.Unlock(); return b.buffer.String() }
func waitUpdateFixture(t *testing.T, condition func() bool, output *updateTerminalOutput) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("terminal fixture timed out; output:\n%s", output.text())
}
func TestUpdateUnixTerminalRestart(t *testing.T) {
	dir := t.TempDir()
	original, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	binary, e := os.ReadFile(original)
	if e != nil {
		t.Fatal(e)
	}
	target := filepath.Join(dir, "jutsu fixture")
	if e = os.WriteFile(target, binary, 0755); e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command(target, "-test.run=^TestUpdateTerminalChild$", "-test.v")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "JUTSU_UPDATE_TERMINAL_FIXTURE="+dir, "JUTSU_UPDATE_LITERAL=literal $(value) with spaces")
	terminal, e := pty.StartWithSize(cmd, &pty.Winsize{Rows: 40, Cols: 120})
	if e != nil {
		t.Fatal(e)
	}
	defer terminal.Close()
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()
	output := &updateTerminalOutput{}
	readerDone := make(chan struct{})
	go func() { io.Copy(output, terminal); close(readerDone) }()
	// Answer terminal capability queries as a real emulator would. These bytes
	// are part of the fixture's terminal protocol, never shell commands.
	time.Sleep(100 * time.Millisecond)
	terminal.Write([]byte("\x1b[?1;2c\x1b[1;1R\x1b[?2026;2$y"))
	waitUpdateFixture(t, func() bool { return strings.Contains(output.text(), "about") }, output)
	terminal.Write([]byte("?"))
	waitUpdateFixture(t, func() bool { return strings.Contains(output.text(), "Update and restart") }, output)
	terminal.Write([]byte("u"))
	marker := filepath.Join(dir, "restarted")
	waitUpdateFixture(t, func() bool { _, e := os.Stat(marker); return e == nil }, output)
	terminal.Write([]byte("\x1b[?1;2c\x1b[1;1R\x1b[?2026;2$y"))
	// Wait for the fresh TUI's input loop to be ready, then send its first key.
	time.Sleep(200 * time.Millisecond)
	terminal.Write([]byte("q"))
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case e := <-done:
		if e != nil {
			t.Fatalf("updated child: %v\n%s", e, output.text())
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("first post-restart q did not quit\n%s", output.text())
	}
	terminal.Close()
	<-readerDone
	data, e := os.ReadFile(marker)
	if e != nil {
		t.Fatal(e)
	}
	var got struct {
		Cwd  string
		Args []string
		Env  string
	}
	if e = json.Unmarshal(data, &got); e != nil {
		t.Fatal(e)
	}
	if got.Cwd != dir || got.Env != "literal $(value) with spaces" || fmt.Sprint(got.Args) != fmt.Sprint(cmd.Args) {
		t.Fatalf("restart context changed: %s", data)
	}
	info, e := os.Stat(target)
	if e != nil || info.Mode().Perm() != 0755 {
		t.Fatalf("permissions: %v %v", info, e)
	}
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".jutsu-update-") {
			t.Fatal("completed restart left recovery staging")
		}
	}
}
