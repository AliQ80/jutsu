package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// The tmux backend of the multiplexer handoff. See mux_exec.go for the shared
// driver and what the six steps are.
//
// Like herdr, the pane is opened empty — running the user's own shell — and the
// script is then typed into it. A pane created *with* a command runs no shell
// at all, so no startup file is sourced and the user's $EDITOR, $PATH and the
// rest never arrive; that is how `jj describe` ended up trying to run jj's
// built-in default editor.
//
// Where tmux does differ: it has no "wait until this pane prints X" primitive,
// so the script and Jutsu hand-shake over a `wait-for` channel instead — which
// also has to keep the pane alive, since a tmux pane closes the moment its
// command exits, taking the transcript with it. The script blocks on a second
// channel until we've captured it.

// ponytail: same head-loss ceiling as herdr's --lines.
const tmuxCaptureLines = "-1000"

// ponytail: matches herdr's 2h wait. If the user closes the pane by hand the
// script never signals, and this is what eventually unsticks Jutsu. Watching
// for the pane's disappearance would react sooner.
const tmuxWaitTimeout = 2 * time.Hour

// tmuxPromptTimeout caps how long we wait for a new pane's shell to be ready
// before typing into it. Generous: a heavy .zshrc can take a while.
const tmuxPromptTimeout = 3 * time.Second

// inTmux reports whether Jutsu is running inside a tmux pane we can drive.
func inTmux() bool {
	if os.Getenv("TMUX") == "" || os.Getenv("TMUX_PANE") == "" {
		return false
	}
	_, err := exec.LookPath("tmux")
	return err == nil
}

var tmuxMux = mux{
	notify: func(dir string) string {
		ch := tmuxChannel(dir)
		// Signal that the command is done, then block until Jutsu has scraped
		// the pane. tmux latches a signal sent with nobody waiting, so Jutsu
		// starting its wait late is not a race. The release signal never
		// actually arrives — dispose kills the pane instead — but blocking on
		// a channel is a cleaner "stay alive until told otherwise" than a
		// sleep with a made-up duration.
		return fmt.Sprintf("tmux wait-for -S %s\ntmux wait-for %s_release\n", ch, ch)
	},
	open: openTmuxTerminal,
}

// openTmuxTerminal creates the terminal a handoff should run in — a split pane
// under Jutsu for a short credential prompt, a new window for anything that
// paints a whole screen — with the script as its command.
func openTmuxTerminal(script string, plain bool) (*muxTerm, error) {
	wd, _ := os.Getwd()
	self := os.Getenv("TMUX_PANE")

	// Where to send focus back to afterwards. Read now, while we still know
	// nothing has moved.
	window, err := tmuxOutput("display-message", "-p", "-t", self, "#{window_id}")
	if err != nil {
		return nil, err
	}

	// No command argument: tmux starts the user's default-shell, which sources
	// their startup files (see the note at the top of this file). Neither of
	// these takes -d, so both focus the new terminal.
	args := []string{"new-window", "-n", "jutsu", "-c", wd, "-P", "-F", "#{pane_id}"}
	if plain {
		args = []string{"split-window", "-v", "-l", "30%", "-t", self, "-c", wd,
			"-P", "-F", "#{pane_id}"}
	}
	pane, err := tmuxOutput(args...)
	if err != nil {
		return nil, err
	}
	if pane == "" {
		return nil, fmt.Errorf("tmux %s: no pane id in output", args[0])
	}

	waitForPrompt(pane)
	if err := tmuxRun("send-keys", "-t", pane, "sh "+script, "Enter"); err != nil {
		_ = tmuxRun("kill-pane", "-t", pane)
		return nil, err
	}

	ch := tmuxChannel(filepath.Dir(script))
	return &muxTerm{
		wait: func() error {
			ctx, cancel := context.WithTimeout(context.Background(), tmuxWaitTimeout)
			defer cancel()
			return exec.CommandContext(ctx, "tmux", "wait-for", ch).Run()
		},
		read: func() string {
			// -J joins wrapped lines, which is what we want, but it also
			// preserves every line's trailing spaces — hence the trim.
			raw, _ := exec.Command("tmux", "capture-pane", "-p", "-J",
				"-S", tmuxCaptureLines, "-t", pane).Output()
			text := trimPaneTranscript(trimTrailingSpaces(string(raw)), script, muxSentinel)
			return cleanCapturedText([]byte(text))
		},
		dispose: func() {
			// Hand focus back explicitly before killing the pane, rather than
			// relying on where tmux moves it afterwards — for a closing
			// window that isn't reliably Jutsu.
			_ = tmuxRun("select-window", "-t", window)
			_ = tmuxRun("select-pane", "-t", self)
			_ = tmuxRun("kill-pane", "-t", pane)
		},
	}, nil
}

// waitForPrompt blocks until the pane's shell has drawn something — meaning it
// has finished its startup files and is reading input. Keystrokes sent before
// that can be swallowed, and a swallowed keystroke is the worst failure this
// design has: the script never runs, nothing ever signals the wait-for channel,
// and the handoff sits there until tmuxWaitTimeout. Gives up quietly and lets
// the caller send anyway rather than failing a handoff over a slow rc file.
func waitForPrompt(pane string) {
	deadline := time.Now().Add(tmuxPromptTimeout)
	for time.Now().Before(deadline) {
		if out, err := tmuxOutput("capture-pane", "-p", "-t", pane); err == nil && out != "" {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// tmuxChannel derives a per-handoff wait-for channel name from the script's
// temp dir, whose name is already unique.
func tmuxChannel(dir string) string {
	return strings.ReplaceAll(filepath.Base(dir), "-", "_")
}

func tmuxRun(args ...string) error {
	return exec.Command("tmux", args...).Run()
}

// tmuxOutput runs a tmux command and returns its single-line output.
func tmuxOutput(args ...string) (string, error) {
	out, err := exec.Command("tmux", args...).Output()
	return strings.TrimSpace(string(out)), err
}

// trimTrailingSpaces strips the padding capture-pane -J leaves on every line.
func trimTrailingSpaces(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return strings.Join(lines, "\n")
}
