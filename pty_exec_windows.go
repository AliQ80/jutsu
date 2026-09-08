//go:build windows

package main

import (
	"errors"
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

// executeInteractive on Windows falls back to plain TTY inheritance:
// creack/pty has no real Windows pty support, so the tee/alt-screen
// capture in pty_exec.go is unavailable here. The invocation's literal argv
// is passed straight to CreateProcess — no shell, so input values keep their
// exact contents; plainHandoff clears the screen with a separate `cmd /C cls`
// ahead of the run, since there is no shell line to prefix any more.
func executeInteractive(inv jjInvocation, plainHandoff bool) tea.Cmd {
	argv := inv.argv
	if len(argv) == 0 {
		return func() tea.Msg {
			return execInteractiveResultMsg{inv: inv, err: errors.New("no command to run")}
		}
	}
	run := tea.ExecProcess(exec.Command(argv[0], argv[1:]...), func(err error) tea.Msg {
		return execInteractiveResultMsg{inv: inv, err: err}
	})
	if plainHandoff {
		return tea.Sequence(tea.ExecProcess(exec.Command("cmd", "/C", "cls"), nil), run)
	}
	return run
}

// detachFromTTY is a no-op: Windows has no controlling-tty concept, and CreateProcess doesn't map to setsid.
func detachFromTTY(cmd *exec.Cmd) {}
