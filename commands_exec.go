package main

import (
	"bytes"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// errNoSSHHost signals fetchHostKey couldn't resolve an SSH host for the remote
// (non-SSH URL or unreadable remote list) — caller falls back to the terminal.
var errNoSSHHost = errors.New("no ssh host for remote")

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

// execInteractiveResultMsg carries the result of executeInteractive
// (defined per-OS in pty_exec.go / pty_exec_windows.go). On platforms with
// pty support, output holds the plain-text tail recovered after the
// subprocess's alternate screen closed (e.g. jj's post-squash summary);
// it's "" if nothing usable was captured, or always on Windows.
type execInteractiveResultMsg struct {
	cmdStr string
	output string
	err    error
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
		// Detach from the controlling terminal so a command that unexpectedly
		// needs a prompt (e.g. jj git push hitting an SSH host-key or
		// credential prompt) fails fast instead of hanging the TUI. The
		// failure is then classified in model.go's execResultMsg handler,
		// which surfaces the host-key / auth modal.
		detachFromTTY(cmd)
		out, err := cmd.CombinedOutput()
		return execResultMsg{
			cmdStr: cmdStr,
			output: string(out),
			err:    err,
		}
	}
}

// isRemoteGitCommand reports whether cmdStr is a jj network operation that can
// legitimately need a live prompt (SSH host-key approval, HTTPS credentials).
// Only these are worth classifying + surfacing a modal for after a failure.
func isRemoteGitCommand(cmdStr string) bool {
	return strings.Contains(cmdStr, "git push") ||
		strings.Contains(cmdStr, "git fetch") ||
		strings.Contains(cmdStr, "git clone")
}

// --- Failure classification -------------------------------------------------
//
// A remote op that failed under the detached (no-tty) run couldn't answer a
// prompt it needed. These predicates read the combined output to decide which
// modal (if any) to surface. Signatures are OpenSSH/git English strings; a
// locale-translated message falls through to a plain error (accepted risk).

// isChangedHostKey is the DANGER case: the remote's key no longer matches the
// one pinned in known_hosts (possible MITM, or a legitimate server rotation).
// Jutsu never offers one-tap trust here — the user must resolve it deliberately.
func isChangedHostKey(out string) bool {
	return strings.Contains(out, "REMOTE HOST IDENTIFICATION HAS CHANGED")
}

// isUnknownHostKey reports the safe first-connect case: the host simply isn't in
// known_hosts yet. This is the one Jutsu handles in-TUI (fingerprint modal).
func isUnknownHostKey(out string) bool {
	if isChangedHostKey(out) {
		return false
	}
	return strings.Contains(out, "Host key verification failed") ||
		strings.Contains(out, "host key is known for") // "No ED25519 host key is known for ..."
}

// isAuthFailure reports a failure the detached (no-tty) run couldn't get past
// but a real terminal might: an HTTPS username/password prompt, or a public-key
// rejection that's really a passphrase-protected key ssh couldn't unlock
// without a tty. These route to the guided-terminal modal. (When the key is
// genuinely unauthorized the terminal retry just fails again — harmless.)
func isAuthFailure(out string) bool {
	for _, sig := range []string{
		"could not read Username",
		"could not read Password",
		"Authentication failed",
		"terminal prompts disabled",
		"fatal: could not read",
		"Permission denied (publickey",
	} {
		if strings.Contains(out, sig) {
			return true
		}
	}
	return false
}

// --- Host-key resolution ----------------------------------------------------

type hostKeyFetchedMsg struct {
	host         string // host[:port] display label
	fingerprint  string // human `ssh-keygen -lf` output (one line per key type)
	keyscanLines string // raw ssh-keyscan output to append to known_hosts
	pushCmd      string // the original command to silently retry after trust
	err          error  // non-nil → couldn't scan; caller falls back to terminal
}

type knownHostAddedMsg struct {
	pushCmd string
	err     error
}

// remoteHostFor resolves the SSH host[:port] the failed command was pushing to,
// by reading `jj git remote list` and parsing the selected remote's URL. Returns
// ("", "") for non-SSH (https/git) remotes or when resolution fails — the caller
// then falls back to the guided-terminal modal.
func remoteHostFor(cmdStr string) (host, port string) {
	out, err := exec.Command("jj", "git", "remote", "list").Output()
	if err != nil {
		return "", ""
	}
	remote := selectedRemoteName(cmdStr)
	return parseSSHHost(remoteURL(string(out), remote))
}

// selectedRemoteName extracts the value of a --remote flag from cmdStr, or ""
// when none is given (jj then defaults to origin / the sole remote).
func selectedRemoteName(cmdStr string) string {
	fields := strings.Fields(cmdStr)
	for i, f := range fields {
		if f == "--remote" && i+1 < len(fields) {
			return fields[i+1]
		}
		if v, ok := strings.CutPrefix(f, "--remote="); ok {
			return v
		}
	}
	return ""
}

// remoteURL picks a remote's URL from `jj git remote list` output (lines of
// "name url"). It prefers the named remote, then "origin", then the only remote.
func remoteURL(list, name string) string {
	remotes := map[string]string{}
	var order []string
	for _, line := range strings.Split(list, "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 {
			remotes[f[0]] = f[1]
			order = append(order, f[0])
		}
	}
	if name != "" {
		return remotes[name]
	}
	if u, ok := remotes["origin"]; ok {
		return u
	}
	if len(order) == 1 {
		return remotes[order[0]]
	}
	return ""
}

// parseSSHHost extracts host and port from a git remote URL, returning ("", "")
// for non-SSH schemes (https/git), which don't do host-key verification.
func parseSSHHost(remote string) (host, port string) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", ""
	}
	if strings.Contains(remote, "://") {
		u, err := url.Parse(remote)
		if err != nil || u.Scheme != "ssh" {
			return "", "" // https:// or git:// — no host key to approve
		}
		return u.Hostname(), u.Port()
	}
	// scp-like syntax: [user@]host:path (the colon precedes a path, not a port).
	if at := strings.LastIndex(remote, "@"); at >= 0 {
		remote = remote[at+1:]
	}
	if colon := strings.Index(remote, ":"); colon >= 0 {
		return remote[:colon], ""
	}
	return remote, ""
}

// fetchHostKey resolves the remote's SSH host, scans its public keys
// (ssh-keyscan) and computes their fingerprints (ssh-keygen -lf) so the modal
// can show what the user is trusting. Any failure (non-SSH remote, missing
// tools, unreachable host) returns a non-nil err so the caller falls back to
// the guided-terminal modal. Runs the blocking work off the Update loop.
func fetchHostKey(pushCmd string) tea.Cmd {
	return func() tea.Msg {
		fail := func(err error) tea.Msg {
			return hostKeyFetchedMsg{pushCmd: pushCmd, err: err}
		}
		host, port := remoteHostFor(pushCmd)
		if host == "" {
			return fail(errNoSSHHost)
		}
		args := []string{}
		if port != "" && port != "22" {
			args = append(args, "-p", port)
		}
		args = append(args, host)
		scan, err := exec.Command("ssh-keyscan", args...).Output()
		if err != nil {
			return fail(err)
		}
		if len(bytes.TrimSpace(scan)) == 0 {
			return fail(os.ErrNotExist) // host unreachable / no keys returned
		}
		fpCmd := exec.Command("ssh-keygen", "-lf", "-")
		fpCmd.Stdin = bytes.NewReader(scan)
		fp, err := fpCmd.Output()
		if err != nil {
			return fail(err)
		}
		return hostKeyFetchedMsg{
			host:         host,
			fingerprint:  strings.TrimRight(string(fp), "\n"),
			keyscanLines: string(scan),
			pushCmd:      pushCmd,
		}
	}
}

// addKnownHost appends the scanned key lines to ~/.ssh/known_hosts (creating it
// 0600 if absent), then reports back so the caller can silently retry pushCmd.
func addKnownHost(keyscanLines, pushCmd string) tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return knownHostAddedMsg{pushCmd: pushCmd, err: err}
		}
		path := filepath.Join(home, ".ssh", "known_hosts")
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return knownHostAddedMsg{pushCmd: pushCmd, err: err}
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return knownHostAddedMsg{pushCmd: pushCmd, err: err}
		}
		defer f.Close()
		data := keyscanLines
		if !strings.HasSuffix(data, "\n") {
			data += "\n"
		}
		_, err = f.WriteString(data)
		return knownHostAddedMsg{pushCmd: pushCmd, err: err}
	}
}
