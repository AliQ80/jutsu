//go:build windows

package main

import (
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

// executeInteractive on Windows falls back to plain TTY inheritance:
// creack/pty has no real Windows pty support, so the tee/alt-screen
// capture in pty_exec.go is unavailable here; plainHandoff clears the
// screen via "cls && " instead.
func executeInteractive(cmdStr string, plainHandoff bool) tea.Cmd {
	shellCmd := cmdStr
	if plainHandoff {
		shellCmd = "cls && " + cmdStr
	}
	cmd := exec.Command("cmd", "/C", shellCmd)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execInteractiveResultMsg{cmdStr: cmdStr, err: err}
	})
}

// detachFromTTY is a no-op: Windows has no controlling-tty concept, and CreateProcess doesn't map to setsid.
func detachFromTTY(cmd *exec.Cmd) {}
