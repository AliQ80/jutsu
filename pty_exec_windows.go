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
func executeInteractive(cmdStr string) tea.Cmd {
	cmd := exec.Command("cmd", "/C", cmdStr)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execInteractiveResultMsg{cmdStr: cmdStr, err: err}
	})
}
