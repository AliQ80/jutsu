package main

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// richDocModel parks a synthetic model on a command with description, args,
// options and subcommands so the docs payload exercises every section.
func richDocModel() mainModel {
	m := newModel()
	m.categories = []Category{{
		Name: "test",
		Commands: []Command{{
			Name:        "testcmd",
			Alias:       "tc",
			Description: "Does a thing.",
			Args: []Arg{
				{Name: "PATH", Required: true, Description: "the path to act on"},
			},
			Flags: []Flag{
				{Name: "message", Value: "--message", InputType: "MESSAGE", Description: "commit message"},
			},
			SubCmds: []SubCommand{
				{Name: "sub", Summary: "a subcommand", Description: "sub desc"},
			},
		}},
	}}
	m.catIdx, m.cmdIdx, m.subIdx, m.flagIdx = 0, 0, 0, 0
	m.focusPane = focusCategories // command-level docs
	return m
}

// Docs copy must be plain text: same headings/usage/args/options the pane
// shows, but with no Lip Gloss ANSI escapes.
func TestDocsCopyPayloadIsPlainText(t *testing.T) {
	m := richDocModel()
	got := m.docsContentPlain()

	for _, want := range []string{"Usage", "Options", "--message", "Arguments", "PATH", "Subcommands", "sub"} {
		if !strings.Contains(got, want) {
			t.Errorf("docs plain text missing %q\n--- got ---\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b") {
		t.Errorf("docs plain text contained ANSI escape: %q", got)
	}
}

// A focused flag produces plain (unstyled) docs too, not the rendered header.
func TestDocsFlagCopyPayloadIsPlainText(t *testing.T) {
	m := richDocModel()
	m.focusPane = focusFlags
	m.flagIdx = 0
	got := m.docsContentPlain()

	if !strings.Contains(got, "--message") || !strings.Contains(got, "commit message") {
		t.Errorf("flag plain text missing expected content: %q", got)
	}
	if strings.Contains(got, "\x1b") {
		t.Errorf("flag plain text contained ANSI escape: %q", got)
	}
}

// buildDocsBlockStyled with the plain styler yields readable section text with
// no styling escapes — the shared payload builder for clipboard copy.
func TestBuildDocsBlockPlainHasSections(t *testing.T) {
	got := buildDocsBlockStyled("cmd", "cmd", "", "desc",
		[]Arg{{Name: "ARG", Required: false, Description: "an arg"}},
		[]Flag{{Name: "force", Value: "--force", Description: "force it"}},
		"", nil, plainStyler)

	for _, want := range []string{"Usage", "Options", "--force", "Arguments", "ARG"} {
		if !strings.Contains(got, want) {
			t.Errorf("plain docs block missing %q\n--- got ---\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b") {
		t.Errorf("plain docs block contained ANSI escape: %q", got)
	}
}

// A successful clipboard copy flips the green success flash, never the error one.
func TestCopySuccessSetsFlash(t *testing.T) {
	orig := clipboardCopy
	defer func() { clipboardCopy = orig }()
	clipboardCopy = func(string) error { return nil }

	m := syntheticModel([]Flag{optionalFlag("o1")})
	m.focusPane = focusDocs

	next, cmd := m.handleDocsKeys(keyPress('c'))
	if cmd == nil {
		t.Fatal("expected a copy command")
	}
	msg := cmd()
	updated, _ := next.(mainModel).Update(msg)
	got := updated.(mainModel)

	if !got.copyFlash {
		t.Error("copyFlash not set on success")
	}
	if got.copyErrorFlash {
		t.Error("copyErrorFlash set on success")
	}
}

// A failed clipboard copy must NOT show the green success indicator; it sets a
// distinct error flash instead.
func TestCopyFailureShowsErrorNotSuccess(t *testing.T) {
	orig := clipboardCopy
	defer func() { clipboardCopy = orig }()
	clipboardCopy = func(string) error { return errors.New("wl-copy unavailable") }

	for _, pane := range []int{focusDocs, focusOutput, focusCmdBar} {
		m := syntheticModel([]Flag{optionalFlag("o1")})
		m.focusPane = pane

		var next tea.Model
		var cmd tea.Cmd
		switch pane {
		case focusDocs:
			next, cmd = m.handleDocsKeys(keyPress('c'))
		case focusOutput:
			next, cmd = m.handleOutputKeys(keyPress('c'))
		case focusCmdBar:
			next, cmd = m.handleCmdBarKeys(keyPress('c'))
		}
		if cmd == nil {
			t.Fatalf("pane %d: expected a copy command", pane)
		}
		msg := cmd()
		upd, _ := next.(mainModel).Update(msg)
		got := upd.(mainModel)

		if got.copyFlash {
			t.Errorf("pane %d: green success flash set on failure", pane)
		}
		if !got.copyErrorFlash {
			t.Errorf("pane %d: copyErrorFlash not set on failure", pane)
		}
	}
}

// The clipboard invocation must explicitly publish text as
// `text/plain;charset=utf-8` so the desktop clipboard owns unambiguous plain
// text. Asserted via the command boundary without needing a live Wayland
// session (exec.Command only builds the struct; it never runs wl-copy here).
func TestClipboardPublishesPlainTextMIME(t *testing.T) {
	cmd := clipboardCmd("any content")
	if got := strings.Join(cmd.Args, " "); !strings.Contains(got, "-t") || !strings.Contains(got, "text/plain;charset=utf-8") {
		t.Errorf("clipboard invocation missing explicit plain-text MIME type: %q", got)
	}
}
