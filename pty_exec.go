//go:build !windows

package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/creack/pty"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
)

// executeInteractive suspends the TUI and hands a real pty (not just stdio
// inheritance) to the subprocess (editor, diff picker, merge tool), so
// Jutsu can tee its output and recover any plain-text tail printed after
// the tool exits its alternate screen (e.g. jj's "asked for combined
// description" summary after squash's $EDITOR closes). Unlike
// executeCommand, no --color=always is injected: the subprocess owns a
// real TTY and detects colour support itself.
func executeInteractive(cmdStr string) tea.Cmd {
	c := newPtyExecCommand(cmdStr)
	return tea.Exec(c, func(err error) tea.Msg {
		output, _ := c.capturedTrailingOutput()
		return execInteractiveResultMsg{cmdStr: cmdStr, output: output, err: err}
	})
}

// maxCapturedOutput bounds the in-memory tee buffer so a chatty full-screen
// editor redrawing repeatedly cannot grow memory unboundedly. 64KB is far
// more than enough to hold the trailing plain-text tail we actually want
// (a few lines of jj status output after the editor's alt-screen closes).
const maxCapturedOutput = 64 * 1024

// altScreenExitSeq is the CSI sequence terminals/editors use to leave the
// alternate screen buffer (used by vim, less, and most full-screen tools).
// Only this one marker is checked; older/rarer variants (?47l, raw
// clear-screen idioms) aren't worth the added false-positive risk.
var altScreenExitSeq = []byte("\x1b[?1049l")

// ptyExecCommand implements tea.ExecCommand (Run/SetStdin/SetStdout/SetStderr)
// by running cmdStr's subprocess attached to a pty instead of the process's
// real stdio. This lets Jutsu tee the subprocess's pty output into an
// in-memory buffer, recovering trailing plain-text jj output printed after
// an editor/pager closes (e.g. jj's post-squash summary), while still
// giving the subprocess a real interactive terminal via bidirectional byte
// forwarding. tea.Exec still brackets Run() with the program's
// releaseTerminal/RestoreTerminal, so Bubble Tea's raw-mode input reader is
// suspended/resumed exactly as it is for tea.ExecProcess.
type ptyExecCommand struct {
	cmdStr string

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	captured bytes.Buffer // pty-master → stdout tee, capped at maxCapturedOutput
}

func newPtyExecCommand(cmdStr string) *ptyExecCommand {
	return &ptyExecCommand{cmdStr: cmdStr}
}

func (c *ptyExecCommand) SetStdin(r io.Reader)  { c.stdin = r }
func (c *ptyExecCommand) SetStdout(w io.Writer) { c.stdout = w }
func (c *ptyExecCommand) SetStderr(w io.Writer) { c.stderr = w }

// Run starts the subprocess attached to a pty, forwards bytes and resize
// events for its lifetime, and returns its exit error (if any). After Run
// returns, c.captured holds up to the last maxCapturedOutput bytes the
// subprocess wrote, for capturedTrailingOutput to scan.
func (c *ptyExecCommand) Run() error {
	cmd := exec.Command("sh", "-c", c.cmdStr)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return err
	}

	// Bubble Tea already restored the real terminal to its pre-raw-mode
	// (cooked) state before invoking Run (see releaseTerminal/restoreInput
	// in bubbletea's tea.go) — it expects a directly-inherited child to
	// raw-mode the terminal itself, as tea.ExecProcess's target normally
	// would. Here the child is attached to the pty slave instead, so it
	// only ever raw-modes *that* — the real terminal is left cooked, which
	// line-buffers and echoes every keystroke before our forward loop ever
	// sees it (breaks cursor movement, control keys, everything in a
	// full-screen editor). We raw-mode the real terminal ourselves for the
	// forward loop's duration, exactly as ssh -t / tmux attach / pty's own
	// docs do.
	if f, ok := c.stdin.(term.File); ok && term.IsTerminal(f.Fd()) {
		if oldState, err := term.MakeRaw(f.Fd()); err == nil {
			defer term.Restore(f.Fd(), oldState)
		}
	}

	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	go func() {
		for range winch {
			_ = pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	winch <- syscall.SIGWINCH // sync size immediately on start

	// pty → real stdout, teed into the bounded capture buffer. Only this
	// direction is waited on below: it reliably unblocks when ptmx is
	// closed. The stdin → pty direction (below) reads from the real
	// terminal's stdin, which we don't own and can't close, so it is not
	// waited on — see the comment above the goroutine for why that's safe.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		w := io.MultiWriter(c.stdout, &boundedWriter{buf: &c.captured, max: maxCapturedOutput})
		_, _ = io.Copy(w, ptmx)
	}()

	// terminal → pty (keystrokes to the subprocess). This goroutine is
	// intentionally not joined: it blocks in Read() on the real stdin,
	// which we can't close out from under Bubble Tea. Once the subprocess
	// exits and ptmx is closed below, this goroutine's next Write to the
	// closed pty master errors and it exits on its own next keystroke —
	// a narrow, bounded leak, not an indefinite one. In the brief window
	// before that happens, a keystroke could race with Bubble Tea's
	// restored input reader; accepted as out of scope given how narrow
	// the window is.
	go func() {
		_, _ = io.Copy(ptmx, c.stdin)
	}()

	err = cmd.Wait()
	_ = ptmx.Close()
	wg.Wait()

	return err
}

// capturedTrailingOutput returns the plain-text tail of c.captured that was
// written after the subprocess's last "exit alternate screen" escape
// sequence, or ("", false) if no such marker is found (e.g. a non-alt-screen
// tool, or a merge tool that leaves the screen in an unrecognized state).
func (c *ptyExecCommand) capturedTrailingOutput() (string, bool) {
	return capturedTrailingOutput(c.captured.Bytes())
}

// capturedTrailingOutput scans raw (possibly ANSI-laden) pty output captured
// from an interactive subprocess and returns the plain-text tail written
// after the last alt-screen-exit marker, ANSI-stripped and trimmed. Returns
// ("", false) if no marker is present, signaling callers should fall back
// to a canned success message rather than risk showing garbled bytes.
func capturedTrailingOutput(raw []byte) (string, bool) {
	idx := bytes.LastIndex(raw, altScreenExitSeq)
	if idx < 0 {
		return "", false
	}
	tail := raw[idx+len(altScreenExitSeq):]
	// Ptys run in cooked mode by default, so the child's "\n" writes arrive
	// here as "\r\n" (ONLCR). Normalize before splitting into lines
	// downstream, or every recovered line would carry a trailing '\r'.
	text := strings.ReplaceAll(stripANSI(string(tail)), "\r\n", "\n")
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r", "\n"))
	if text == "" {
		return "", false
	}
	return text, true
}

// boundedWriter appends to buf, keeping only the last max bytes.
type boundedWriter struct {
	buf *bytes.Buffer
	max int
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	if extra := w.buf.Len() - w.max; extra > 0 {
		b := w.buf.Bytes()
		w.buf.Reset()
		w.buf.Write(b[extra:])
	}
	return len(p), nil
}
