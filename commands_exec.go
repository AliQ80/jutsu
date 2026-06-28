package main

import (
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type execResultMsg struct {
	cmdStr string
	output string
	err    error
}

type jjVersionMsg string

func fetchJJVersion() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("jj", "--version").Output()
		if err != nil {
			return jjVersionMsg("")
		}
		// output is e.g. "jj 0.42.0\n" — take first line only
		line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
		return jjVersionMsg(line)
	}
}

type copyDoneMsg struct{}

func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("wl-copy")
		cmd.Stdin = strings.NewReader(text)
		_ = cmd.Run()
		return copyDoneMsg{}
	}
}

func executeCommand(cmdStr string) tea.Cmd {
	return func() tea.Msg {
		// Inject --color=always for jj commands since we're running in a pipe,
		// not a real TTY, and jj defaults to no color in that case.
		execStr := cmdStr
		if execStr == "jj" || strings.HasPrefix(execStr, "jj ") {
			execStr = "jj --color=always" + execStr[2:]
		}
		cmd := exec.Command("sh", "-c", execStr)
		out, err := cmd.CombinedOutput()
		return execResultMsg{
			cmdStr: cmdStr,
			output: string(out),
			err:    err,
		}
	}
}
