package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Running interactive jj commands inside a terminal multiplexer
//
// Outside a multiplexer, an interactive command (an editor, a diff picker, a
// credential prompt) can only be given a terminal by suspending Jutsu and
// handing over the one terminal we have — that's executeInteractive in
// pty_exec.go, with all the raw-mode and pty-tee machinery it needs.
//
// Inside a multiplexer we can just ask for another terminal. Jutsu keeps
// rendering, the subprocess gets a real tty of its own, and none of the pty
// path runs:
//
//	credential prompts → a split pane below Jutsu (small, transient)
//	editors/pickers    → a new full-size tab/window (a 30% strip is no place for nvim)
//
// Both backends follow the same six steps — open a terminal, run a wrapper
// script in it, wait for the script to report completion, scrape the
// transcript, hand focus back, tear the terminal down — so all of that lives
// here and each backend fills in the parts that are its own CLI's business.
// See herdr_exec.go and tmux_exec.go.
//
// The subprocess's stdout is deliberately never piped or redirected — it owns
// the new terminal's tty, or full-screen editors misbehave ("Vim: Warning:
// Output is not to a terminal"). Its output is scraped back afterwards
// instead. That reads better than the pty tee does: a multiplexer's own
// emulator keeps the alternate screen out of the scrollback buffer, so an
// editor's redraw simply isn't in the transcript — only the plain-text tail jj
// prints after the editor closes, which is exactly what
// capturedTrailingOutput() has to go hunting for in the pty path.

// Both backends open the terminal empty so the user's own shell starts in it
// and sources their startup files — that, rather than any variable forwarding,
// is how the handoff gets $EDITOR, $PATH and the rest.
//
// ponytail: a variable set only in Jutsu's own process (`EDITOR=foo jutsu`) is
// therefore lost — the pane's shell sets its own from the user's rc, where the
// pty path would have inherited Jutsu's. Honouring it would mean re-exporting
// chosen variables after the rc runs, i.e. maintaining a list that only ever
// encodes one machine's setup. The shell's configuration is the right authority
// for a terminal the user is about to type into.

// muxSentinel is printed by the wrapper script once the command has exited. It
// marks the end of the command's own output in the scraped transcript, and
// herdr additionally blocks on it. It cannot false-match the shell's echo of
// the line that starts the run, because that line only ever contains the
// script's path.
const muxSentinel = "__JUTSU_DONE__"

// mux is one multiplexer's half of a handoff.
type mux struct {
	// notify is the tail of the wrapper script: how it reports that the
	// command has finished. Takes the script's temp dir so a backend can
	// derive a per-run identifier from it.
	notify func(dir string) string

	// open creates the terminal the script runs in and starts it. plain marks
	// a short line-based command (a credential prompt) rather than a tool that
	// paints the whole screen.
	open func(script string, plain bool) (*muxTerm, error)
}

// muxTerm is a live handoff terminal: a place a command is running, and the
// three things we do to it afterwards.
type muxTerm struct {
	wait    func() error  // block until the wrapper script reports it's done
	read    func() string // the command's output, scraped from the terminal
	dispose func()        // hand focus back to Jutsu, tear the terminal down
}

// currentMux returns the multiplexer Jutsu is running inside, or nil.
//
// tmux wins when both are set: with tmux running inside a herdr pane, tmux is
// what owns Jutsu's terminal, so its split lands next to Jutsu — herdr's would
// split the pane *containing* tmux.
func currentMux() *mux {
	switch {
	case inTmux():
		return &tmuxMux
	case inHerdr():
		return &herdrMux
	}
	return nil
}

// execFallbackMsg asks Update to run cmdStr through the ordinary pty handoff.
// Returned when the multiplexer itself fails (socket gone, server restarted)
// so a broken multiplexer never means a command that can't run at all.
type execFallbackMsg struct {
	cmdStr string
	plain  bool
}

// executeHandoff runs a command that needs a real terminal, preferring a
// sibling pane/tab in whatever multiplexer we're inside over suspending the
// TUI. plain marks a plain line-based command (a credential prompt) rather
// than an alt-screen tool; see executeInteractive for what it means on the
// fallback path.
func executeHandoff(cmdStr string, plain bool) tea.Cmd {
	if cmd := executeInMux(cmdStr, plain); cmd != nil {
		return cmd
	}
	return executeInteractive(cmdStr, plain)
}

// executeInMux runs cmdStr in a sibling multiplexer terminal and reports its
// output back, leaving Jutsu running throughout. Returns nil when we're not
// inside a multiplexer, so callers fall back to the pty handoff.
func executeInMux(cmdStr string, plain bool) tea.Cmd {
	m := currentMux()
	if m == nil {
		return nil
	}
	return func() tea.Msg {
		fallback := execFallbackMsg{cmdStr: cmdStr, plain: plain}

		dir, err := os.MkdirTemp("", "jutsu-mux-")
		if err != nil {
			return fallback
		}
		defer os.RemoveAll(dir)

		script := filepath.Join(dir, "run.sh")
		exitFile := filepath.Join(dir, "exit")
		if err := os.WriteFile(script, []byte(muxScript(cmdStr, exitFile, m.notify(dir))), 0o700); err != nil {
			return fallback
		}

		term, err := m.open(script, plain)
		if err != nil {
			return fallback
		}

		// Blocks for as long as the command runs — an editor session can
		// legitimately last hours. If it gives up (or the user closed the
		// terminal themselves) the terminal is left alone rather than killed.
		if err := term.wait(); err != nil {
			return execInteractiveResultMsg{cmdStr: cmdStr, err: fmt.Errorf("lost track of the handoff terminal: %w", err)}
		}

		output := term.read()
		term.dispose()

		var runErr error
		if code, err := os.ReadFile(exitFile); err != nil {
			runErr = fmt.Errorf("command did not finish")
		} else if c := strings.TrimSpace(string(code)); c != "0" {
			runErr = fmt.Errorf("exit status %s", c)
		}
		return execInteractiveResultMsg{cmdStr: cmdStr, output: output, err: runErr}
	}
}

// muxScript builds the wrapper script a handoff terminal runs. A script file
// (rather than a command line) keeps cmdStr from ever crossing a shell quoting
// boundary — jj commit messages are full of quotes.
func muxScript(cmdStr, exitFile, notify string) string {
	// The leading clear wipes anything the terminal opened with — for herdr,
	// the shell's echo of the line that started this script — so the new
	// terminal shows the command's own output and nothing else.
	//
	// The subshell matters: a command that exits the shell it's given
	// (anything ending in `exit`) would otherwise take the wrapper with it,
	// leaving no exit code and no completion report — i.e. a wait that never
	// returns.
	return fmt.Sprintf("printf '\\033[2J\\033[3J\\033[H'\n( %s\n)\necho $? > %s\nprintf '%s\\n'\n%s",
		cmdStr, exitFile, muxSentinel, notify)
}

// trimPaneTranscript reduces a scraped pane transcript to just the command's
// own output: everything after the shell's echo of the line that started the
// run (matched on the script path, taking the last occurrence so a redrawn
// prompt doesn't leave a stale copy in) and before the sentinel the script
// prints on the way out. Either marker missing is fine — that end is simply
// not trimmed.
func trimPaneTranscript(raw, scriptPath, sentinel string) string {
	if i := strings.LastIndex(raw, scriptPath); i >= 0 {
		if nl := strings.IndexByte(raw[i:], '\n'); nl >= 0 {
			raw = raw[i+nl+1:]
		} else {
			raw = ""
		}
	}
	if i := strings.Index(raw, sentinel); i >= 0 {
		raw = raw[:i]
	}
	return strings.TrimSpace(raw)
}
