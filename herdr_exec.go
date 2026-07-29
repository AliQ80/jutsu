package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Running interactive jj commands inside herdr
//
// Outside a multiplexer, an interactive command (an editor, a diff picker, a
// credential prompt) can only be given a terminal by suspending Jutsu and
// handing over the one terminal we have — that's executeInteractive in
// pty_exec.go, with all the raw-mode and pty-tee machinery it needs.
//
// Inside herdr we can just ask for another terminal. Jutsu keeps rendering,
// the subprocess gets a real tty of its own, and none of the pty path runs:
//
//	credential prompts → a split pane below Jutsu (small, transient)
//	editors/pickers    → a new full-size tab (a 30% strip is no place for nvim)
//
// The subprocess's stdout is deliberately never piped or redirected — it owns
// the new pane's tty, or full-screen editors misbehave ("Vim: Warning: Output
// is not to a terminal"). Its output is scraped back afterwards with
// `herdr pane read` instead. That reads better than the pty tee does: herdr's
// own emulator keeps the alternate screen out of the scrollback buffer, so an
// editor's redraw simply isn't in the transcript — only the plain-text tail
// jj prints after the editor closes, which is exactly what
// capturedTrailingOutput() has to go hunting for in the pty path.

const (
	// herdrSentinel is printed by the wrapper script after the command exits.
	// It cannot false-match the shell's echo of the line that starts the run,
	// because that line only ever contains the script's path.
	herdrSentinel = "__JUTSU_DONE__"

	// ponytail: a command printing more than this loses its head. Raise it if
	// a real `jj` invocation ever overflows.
	herdrReadLines = "1000"

	// ponytail: an editor session can legitimately last hours; on timeout we
	// leave the pane open and say so rather than kill something the user is
	// still typing into.
	herdrWaitTimeoutMs = "7200000"
)

// execFallbackMsg asks Update to run cmdStr through the ordinary pty handoff.
// Returned when herdr itself fails (socket gone, herdr restarted) so a broken
// multiplexer never means a command that can't run at all.
type execFallbackMsg struct {
	cmdStr string
	plain  bool
}

// inHerdr reports whether Jutsu is running inside a herdr-managed pane that we
// can actually drive. This is the seam another multiplexer (tmux) would slot
// into — nothing else in the codebase knows which one we're in.
func inHerdr() bool {
	if os.Getenv("HERDR_ENV") != "1" || os.Getenv("HERDR_PANE_ID") == "" {
		return false
	}
	_, err := exec.LookPath("herdr")
	return err == nil
}

// executeHandoff runs a command that needs a real terminal, preferring a
// sibling herdr pane/tab over suspending the TUI. plain marks a plain
// line-based command (a credential prompt) rather than an alt-screen tool;
// see executeInteractive for what it means on the fallback path.
func executeHandoff(cmdStr string, plain bool) tea.Cmd {
	if cmd := executeInHerdr(cmdStr, plain); cmd != nil {
		return cmd
	}
	return executeInteractive(cmdStr, plain)
}

// executeInHerdr runs cmdStr in a sibling herdr terminal and reports its
// output back, leaving Jutsu running throughout. Returns nil when we're not
// inside herdr, so callers fall back to the pty handoff.
func executeInHerdr(cmdStr string, plain bool) tea.Cmd {
	if !inHerdr() {
		return nil
	}
	return func() tea.Msg {
		fallback := execFallbackMsg{cmdStr: cmdStr, plain: plain}

		// A throwaway script keeps cmdStr from ever crossing a shell quoting
		// boundary — jj commit messages are full of quotes.
		dir, err := os.MkdirTemp("", "jutsu-herdr-")
		if err != nil {
			return fallback
		}
		defer os.RemoveAll(dir)
		script := filepath.Join(dir, "run.sh")
		exitFile := filepath.Join(dir, "exit")
		// The leading clear wipes the shell's echo of the line that started
		// this script, so the new pane opens on the command's own output and
		// nothing else. (trimPaneTranscript copes either way: whether the echo
		// survives in scrollback or not, it's already outside what we keep.)
		//
		// The subshell matters: a command that exits the shell it's given
		// (anything ending in `exit`) would otherwise take the wrapper with it,
		// leaving no exit code and no sentinel — i.e. a wait that never returns.
		body := fmt.Sprintf("printf '\\033[2J\\033[3J\\033[H'\n( %s\n)\necho $? > %s\nprintf '%s\\n'\n",
			cmdStr, exitFile, herdrSentinel)
		if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
			return fallback
		}

		term, err := openHerdrTerminal(plain)
		if err != nil {
			return fallback
		}

		if err := herdrExec("pane", "run", term.pane, "sh "+script); err != nil {
			_ = herdrExec(term.dispose...)
			return fallback
		}

		// Blocks until the wrapper script prints the sentinel. On timeout (or
		// if the user closed the pane themselves) the terminal is left alone.
		if _, err := herdrJSON("wait", "output", term.pane,
			"--match", herdrSentinel, "--source", "recent", "--timeout", herdrWaitTimeoutMs); err != nil {
			return execInteractiveResultMsg{cmdStr: cmdStr, err: fmt.Errorf("lost track of the herdr pane: %w", err)}
		}

		raw, _ := exec.Command("herdr", "pane", "read", term.pane,
			"--source", "recent-unwrapped", "--lines", herdrReadLines).Output()

		// Hand focus back explicitly before disposing of the terminal, rather
		// than relying on whatever herdr picks after a close.
		_ = herdrExec(term.refocus...)
		_ = herdrExec(term.dispose...)

		var runErr error
		if code, err := os.ReadFile(exitFile); err != nil {
			runErr = fmt.Errorf("command did not finish")
		} else if c := strings.TrimSpace(string(code)); c != "0" {
			runErr = fmt.Errorf("exit status %s", c)
		}

		output := cleanCapturedText([]byte(trimPaneTranscript(string(raw), script, herdrSentinel)))
		return execInteractiveResultMsg{cmdStr: cmdStr, output: output, err: runErr}
	}
}

// herdrTerminal is a freshly created herdr terminal to run one command in,
// plus the calls that give focus back to Jutsu and tear it down again.
type herdrTerminal struct {
	pane    string
	refocus []string
	dispose []string
}

// openHerdrTerminal creates the terminal a handoff should run in: a split pane
// under Jutsu for a short credential prompt, a full-size tab for anything that
// paints a whole screen.
func openHerdrTerminal(plain bool) (herdrTerminal, error) {
	wd, _ := os.Getwd()

	if plain {
		res, err := herdrJSON("pane", "split", os.Getenv("HERDR_PANE_ID"),
			"--direction", "down", "--ratio", "0.7", "--focus", "--cwd", wd)
		if err != nil {
			return herdrTerminal{}, err
		}
		pane := nestedString(res, "pane", "pane_id")
		if pane == "" {
			return herdrTerminal{}, fmt.Errorf("herdr pane split: no pane id in response")
		}
		return herdrTerminal{
			pane: pane,
			// We split downwards, so Jutsu is the pane above this one.
			refocus: []string{"pane", "focus", "--direction", "up", "--pane", pane},
			dispose: []string{"pane", "close", pane},
		}, nil
	}

	// `tab create` does not focus the new tab unless asked.
	res, err := herdrJSON("tab", "create", "--workspace", os.Getenv("HERDR_WORKSPACE_ID"),
		"--cwd", wd, "--label", "jutsu", "--focus")
	if err != nil {
		return herdrTerminal{}, err
	}
	pane, tab := nestedString(res, "root_pane", "pane_id"), nestedString(res, "tab", "tab_id")
	if pane == "" || tab == "" {
		return herdrTerminal{}, fmt.Errorf("herdr tab create: no pane/tab id in response")
	}
	return herdrTerminal{
		pane:    pane,
		refocus: []string{"tab", "focus", os.Getenv("HERDR_TAB_ID")},
		dispose: []string{"tab", "close", tab},
	}, nil
}

// herdrExec runs a herdr command whose output we don't care about.
func herdrExec(args ...string) error {
	return exec.Command("herdr", args...).Run()
}

// herdrJSON runs a herdr command that answers in JSON and returns its "result"
// object. herdr reports failures as a JSON "error" body (with a non-zero exit
// status), so the body is parsed even when the process exits non-zero.
func herdrJSON(args ...string) (map[string]any, error) {
	out, execErr := exec.Command("herdr", args...).Output()
	var resp struct {
		Result map[string]any `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		if execErr != nil {
			return nil, execErr
		}
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("herdr %s: %s", strings.Join(args, " "), resp.Error.Message)
	}
	return resp.Result, nil
}

// nestedString walks a decoded JSON object along keys and returns the string
// it lands on, or "" if any step is missing or the wrong type.
func nestedString(m map[string]any, keys ...string) string {
	var cur any = m
	for _, k := range keys {
		obj, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = obj[k]
	}
	s, _ := cur.(string)
	return s
}

// trimPaneTranscript reduces a herdr pane transcript to just the command's own
// output: everything after the shell's echo of the line that started the run
// (matched on the script path, taking the last occurrence so a redrawn prompt
// doesn't leave a stale copy in) and before the sentinel the script prints on
// the way out. Either marker missing is fine — that end is simply not trimmed.
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
