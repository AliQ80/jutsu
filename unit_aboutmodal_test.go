package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// withJutsuVersion temporarily overrides the package-level jutsuVersion
// (restored after the test) so individual cases can simulate both dev
// builds and ldflag-stamped release builds without leaking across tests.
func withJutsuVersion(t *testing.T, v string) {
	t.Helper()
	saved := jutsuVersion
	jutsuVersion = v
	t.Cleanup(func() { jutsuVersion = saved })
}

// Pressing "?" from a composer pane opens the about modal; esc closes it.
// State must round-trip: open, then close, then no-op on a second close.
func TestAboutModalOpensAndClosesWithEsc(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.focusPane = focusCategories

	qMsg := tea.KeyPressMsg{Code: '?', Text: "?"}
	updated, _ := m.Update(qMsg)
	m = updated.(mainModel)
	if !m.aboutModalOpen {
		t.Fatal(`"?" from a composer pane did not open the about modal`)
	}

	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	updated, _ = m.Update(escMsg)
	m = updated.(mainModel)
	if m.aboutModalOpen {
		t.Fatal("esc did not close the about modal")
	}

	// Second esc is a no-op (modal already closed, focus still on a composer pane).
	updated, _ = m.Update(escMsg)
	m = updated.(mainModel)
	if m.focusPane != focusCategories {
		t.Errorf("a stray esc moved focus: want focusCategories, got %d", m.focusPane)
	}
}

// Pressing "?" while focused on the inputs pane must NOT open the modal —
// "?" is a literal character there and gets handed to the text input.
func TestAboutModalDoesNotOpenInInputsPane(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.focusPane = focusInputs

	qMsg := tea.KeyPressMsg{Code: '?', Text: "?"}
	updated, _ := m.Update(qMsg)
	m = updated.(mainModel)
	if m.aboutModalOpen {
		t.Fatal(`"?" opened the about modal while focusPane == focusInputs; should be a literal character`)
	}
}

// "?" must respect the modal mutex: if the global-options modal is already
// open, the about modal must NOT open and the existing modal stays in charge.
func TestAboutModalDoesNotOpenWhileGlobalOptionsModalOpen(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.focusPane = focusCategories
	m.freezeBackdrop()
	m.globalFlagsModalOpen = true

	qMsg := tea.KeyPressMsg{Code: '?', Text: "?"}
	updated, _ := m.Update(qMsg)
	m = updated.(mainModel)
	if m.aboutModalOpen {
		t.Error(`"?" opened the about modal while the global-options modal was open`)
	}
	if !m.globalFlagsModalOpen {
		t.Error(`global-options modal closed unexpectedly when "?" was pressed`)
	}
}

// "q" inside the about modal must quit the app, NOT close the modal — q's
// app-wide semantics are quit, never close. The returned cmd must be the quit
// signal Bubble Tea recognizes.
func TestQInsideAboutModalQuits(t *testing.T) {
	m := newModel()
	m.width, m.height = 120, 40
	m.aboutModalOpen = true

	qMsg := tea.KeyPressMsg{Code: 'q', Text: "q"}
	_, cmd := m.Update(qMsg)
	if cmd == nil {
		t.Fatal(`"q" inside the about modal returned nil cmd; expected tea.Quit`)
	}
	// Bubble Tea encodes Quit as a typed nil; the conventional check is that
	// the cmd exists (nil cmd == swallowed key). We don't compare the typed
	// nil directly because the underlying message type is unexported.
}

// renderAboutModal must surface both the version line and the docs link.
// This guards against the static content regressing (e.g. someone renaming
// jutsuVersion and forgetting the about popup). The modal renders the version
// verbatim (no "v" prefix) — goreleaser stamps the tag with its "v", so a
// release build reads "jutsu v1.2.3" while `go run .` reads "jutsu dev".
func TestRenderAboutModalContainsVersionAndLink(t *testing.T) {
	m := newModel()
	got := m.renderAboutModal()
	if !strings.Contains(got, "Jutsu") {
		t.Errorf("renderAboutModal() missing %q substring; got:\n%s", "Jutsu", got)
	}
	if !strings.Contains(got, jutsuVersion) {
		t.Errorf("renderAboutModal() missing %q substring; got:\n%s", jutsuVersion, got)
	}
	// The docs URL must appear as one unbroken substring — a wrap or clip
	// would split it across lines and fail this contains check.
	if !strings.Contains(got, "https://aliq80.github.io/jutsu/") {
		t.Errorf("renderAboutModal() missing docs URL; got:\n%s", got)
	}
	if strings.Contains(got, "github.com/AliQ80/jutsu") {
		t.Errorf("renderAboutModal() still shows the repository link; got:\n%s", got)
	}
	// The footer was removed — esc-only-close is documented in
	// handleAboutModalKeys; surfacing it inside the modal was redundant.
	if strings.Contains(got, "close") {
		t.Errorf("renderAboutModal() should not include an esc-close footer; got:\n%s", got)
	}
}

// The title line must be sized against the wider docs URL so the modal's
// content lines stay inside the box: the URL (31 chars) sets the content
// width, the title is centered within it, and no line wraps past it.
func TestRenderAboutModalTitleSizedToDocsURL(t *testing.T) {
	m := newModel()
	got := stripANSI(m.renderAboutModal())

	lines := strings.Split(got, "\n")
	var titleW, urlW int
	foundTitle, foundURL := false, false
	for _, line := range lines {
		if strings.Contains(line, "https://aliq80.github.io/jutsu/") {
			urlW = lipgloss.Width(line)
			foundURL = true
		}
		if strings.Contains(line, "Jutsu") && !strings.Contains(line, "https://") {
			titleW = lipgloss.Width(line)
			foundTitle = true
		}
	}
	if !foundTitle || !foundURL {
		t.Fatalf("renderAboutModal() missing title or docs URL; got:\n%s", got)
	}
	// Title centered across the same width as the URL line (the widest
	// content line) — the longer link sets the modal's content width.
	if titleW != urlW {
		t.Errorf("title line width = %d, docs URL line width = %d; want equal", titleW, urlW)
	}
	// The URL line must be the widest line in the box — nothing wraps past
	// it (and the unbroken-URL check above rules out clipping).
	maxW := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > maxW {
			maxW = w
		}
	}
	if maxW != urlW {
		t.Errorf("a line is wider than the docs URL line (%d > %d); the modal wraps or clips", maxW, urlW)
	}
}

// displayVersion returns the latestTag override when jutsuVersion is still
// the "dev" default and the runtime fetch succeeded; otherwise it returns
// the ldflag-stamped jutsuVersion (a release stamp always wins over the
// runtime fallback so the user never sees a tag less specific than what
// goreleaser baked in).
func TestRuntimeFallbackOverridesDev(t *testing.T) {
	t.Run("dev + tag resolves to the tag", func(t *testing.T) {
		withJutsuVersion(t, "dev")
		m := newModel()
		m.latestTag = "0.5.0"
		if got := m.displayVersion(); got != "0.5.0" {
			t.Errorf("displayVersion() = %q, want %q", got, "0.5.0")
		}
	})

	t.Run("dev + empty tag stays dev", func(t *testing.T) {
		withJutsuVersion(t, "dev")
		m := newModel()
		m.latestTag = ""
		if got := m.displayVersion(); got != "dev" {
			t.Errorf("displayVersion() = %q, want %q", got, "dev")
		}
	})

	t.Run("release stamp wins over runtime tag", func(t *testing.T) {
		withJutsuVersion(t, "v1.2.3")
		m := newModel()
		m.latestTag = "0.5.0"
		if got := m.displayVersion(); got != "v1.2.3" {
			t.Errorf("displayVersion() = %q, want %q (release stamp must win)", got, "v1.2.3")
		}
	})

	t.Run("release stamp with no runtime tag still resolves to the stamp", func(t *testing.T) {
		withJutsuVersion(t, "v1.2.3")
		m := newModel()
		m.latestTag = ""
		if got := m.displayVersion(); got != "v1.2.3" {
			t.Errorf("displayVersion() = %q, want %q", got, "v1.2.3")
		}
	})
}

// When the runtime fallback resolves a tag, the rendered about modal shows
// that tag instead of "dev" — proves the substitution actually reaches the
// user-visible output, not just the helper.
func TestRenderAboutModalUsesRuntimeFallback(t *testing.T) {
	withJutsuVersion(t, "dev")
	m := newModel()
	m.width, m.height = 120, 40
	m.latestTag = "0.5.0"

	got := m.renderAboutModal()
	if !strings.Contains(got, "jutsu     0.5.0") {
		t.Errorf("renderAboutModal() missing %q; runtime fallback did not reach render; got:\n%s",
			"jutsu     0.5.0", got)
	}
	if strings.Contains(got, "jutsu     dev") {
		t.Errorf("renderAboutModal() still shows %q after fallback; got:\n%s", "jutsu     dev", got)
	}
}
