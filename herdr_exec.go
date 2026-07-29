package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// The herdr backend of the multiplexer handoff. See mux_exec.go for the shared
// driver and what the six steps are.
//
// herdr opens an empty terminal — running the user's own shell — and then types
// the wrapper script's path into it, which is why the shell's echo of that line
// has to be trimmed back out of the transcript. The tmux backend works the same
// way, for the same reason: only a real shell brings the user's environment.

const (
	// ponytail: a command printing more than this loses its head. Raise it if
	// a real `jj` invocation ever overflows.
	herdrReadLines = "1000"

	// ponytail: an editor session can legitimately last hours; on timeout we
	// leave the pane open and say so rather than kill something the user is
	// still typing into.
	herdrWaitTimeoutMs = "7200000"
)

// inHerdr reports whether Jutsu is running inside a herdr-managed pane that we
// can actually drive.
func inHerdr() bool {
	if os.Getenv("HERDR_ENV") != "1" || os.Getenv("HERDR_PANE_ID") == "" {
		return false
	}
	_, err := exec.LookPath("herdr")
	return err == nil
}

var herdrMux = mux{
	// Nothing to add: `wait output` blocks on the sentinel the shared script
	// already prints.
	notify: func(string) string { return "" },
	open:   openHerdrTerminal,
}

// openHerdrTerminal creates the terminal a handoff should run in — a split
// pane under Jutsu for a short credential prompt, a full-size tab for anything
// that paints a whole screen — and starts the script in it.
func openHerdrTerminal(script string, plain bool) (*muxTerm, error) {
	wd, _ := os.Getwd()

	var pane string
	var refocus, dispose []string

	if plain {
		res, err := herdrJSON("pane", "split", os.Getenv("HERDR_PANE_ID"),
			"--direction", "down", "--ratio", "0.7", "--focus", "--cwd", wd)
		if err != nil {
			return nil, err
		}
		if pane = nestedString(res, "pane", "pane_id"); pane == "" {
			return nil, fmt.Errorf("herdr pane split: no pane id in response")
		}
		// We split downwards, so Jutsu is the pane above this one.
		refocus = []string{"pane", "focus", "--direction", "up", "--pane", pane}
		dispose = []string{"pane", "close", pane}
	} else {
		// `tab create` does not focus the new tab unless asked.
		res, err := herdrJSON("tab", "create", "--workspace", os.Getenv("HERDR_WORKSPACE_ID"),
			"--cwd", wd, "--label", "jutsu", "--focus")
		if err != nil {
			return nil, err
		}
		tab := nestedString(res, "tab", "tab_id")
		if pane = nestedString(res, "root_pane", "pane_id"); pane == "" || tab == "" {
			return nil, fmt.Errorf("herdr tab create: no pane/tab id in response")
		}
		refocus = []string{"tab", "focus", os.Getenv("HERDR_TAB_ID")}
		dispose = []string{"tab", "close", tab}
	}

	if err := herdrExec("pane", "run", pane, "sh "+script); err != nil {
		_ = herdrExec(dispose...)
		return nil, err
	}

	return &muxTerm{
		wait: func() error {
			_, err := herdrJSON("wait", "output", pane,
				"--match", muxSentinel, "--source", "recent", "--timeout", herdrWaitTimeoutMs)
			return err
		},
		read: func() string {
			raw, _ := exec.Command("herdr", "pane", "read", pane,
				"--source", "recent-unwrapped", "--lines", herdrReadLines).Output()
			return cleanCapturedText([]byte(trimPaneTranscript(string(raw), script, muxSentinel)))
		},
		dispose: func() {
			// Hand focus back explicitly before disposing of the terminal,
			// rather than relying on whatever herdr picks after a close.
			_ = herdrExec(refocus...)
			_ = herdrExec(dispose...)
		},
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
