//go:build windows

package main

import (
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

// executeInteractive on Windows falls back to plain TTY inheritance:
// github.com/creack/pty has no real Windows pty support, so the
// tee/alt-screen-capture behavior in pty_exec.go isn't available here.
// Output still goes straight to the terminal, same as before this feature
// existed on other platforms.
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

// detachFromTTY is a no-op on Windows: there is no controlling-tty concept to
// detach from here, and CreateProcess doesn't line up with the setsid model.
func detachFromTTY(cmd *exec.Cmd) {}
