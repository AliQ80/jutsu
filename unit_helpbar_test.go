package main

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestHelpBarInheritsTerminalBackground(t *testing.T) {
	m := newModel()
	m.focusPane = focusCategories

	got := m.renderHelpBar(120)
	if strings.Contains(got, "\x1b[48;2;24;24;37m") {
		t.Fatal("help bar still renders the fixed #181825 background")
	}
	if !strings.HasPrefix(stripANSI(got), " ") {
		t.Fatal("help bar lost its left padding")
	}
	if lipgloss.Width(got) != 120 {
		t.Fatalf("help bar width = %d, want 120", lipgloss.Width(got))
	}
	if !strings.Contains(got, helpKeyStyle.Render("d/D")) {
		t.Fatal("help key chip lost its existing styling")
	}
	if !strings.Contains(stripANSI(got), "about") {
		t.Fatal("help bar lost its content")
	}
}

func TestHelpBarRightPadding(t *testing.T) {
	m := newModel()
	m.focusPane = focusCategories

	got := m.renderHelpBar(120)
	plain := stripANSI(got)

	// One-column padding on both sides: the bar starts and ends with a
	// single space, so the left and right margins are equal.
	if !strings.HasPrefix(plain, " ") || !strings.HasSuffix(plain, " ") {
		t.Fatal("help bar lost a padding column")
	}
	if lipgloss.Width(plain) != 120 {
		t.Fatalf("help bar rendered %d columns, want 120", lipgloss.Width(plain))
	}

	// The right-side badges sit one column inward from the right edge:
	// the version badge's last rune is followed by exactly the one-column
	// right padding.
	badgeRunes := []rune(stripANSI(m.renderVersionBadge()))
	if !strings.HasSuffix(plain, string(badgeRunes[len(badgeRunes)-1])+" ") {
		t.Fatal("right badge not shifted one column inward")
	}
}

func TestHelpBarNarrowWidth(t *testing.T) {
	m := newModel()
	m.focusPane = focusCategories

	// Smallest width that renders on a single line: the internal gap is
	// clamped to its 1-column minimum there.
	narrow := 0
	for w := 10; w <= 200; w++ {
		if !strings.Contains(stripANSI(m.renderHelpBar(w)), "\n") {
			narrow = w
			break
		}
	}
	if narrow == 0 {
		t.Fatal("help bar never fits on a single line")
	}

	got := m.renderHelpBar(narrow)
	plain := stripANSI(got)

	if lipgloss.Width(got) != narrow {
		t.Fatalf("narrow help bar width = %d, want %d", lipgloss.Width(got), narrow)
	}
	if lipgloss.Width(plain) != narrow {
		t.Fatalf("narrow help bar rendered %d columns, want %d", lipgloss.Width(plain), narrow)
	}
	if !strings.HasPrefix(plain, " ") || !strings.HasSuffix(plain, " ") {
		t.Fatal("narrow help bar lost a padding column")
	}

	// Right-side badges still sit one column inward from the right edge.
	badgeRunes := []rune(stripANSI(m.renderVersionBadge()))
	if !strings.HasSuffix(plain, string(badgeRunes[len(badgeRunes)-1])+" ") {
		t.Fatal("narrow help bar right badge not shifted one column inward")
	}
	if !strings.Contains(got, helpKeyStyle.Render("d/D")) {
		t.Fatal("narrow help bar lost chip styling")
	}
	if !strings.Contains(plain, "about") {
		t.Fatal("narrow help bar lost its content")
	}
}

func TestHelpBarOutputPaneHints(t *testing.T) {
	m := newModel()
	m.focusPane = focusOutput

	got := stripANSI(m.renderHelpBar(120))

	for _, removed := range []string{"scroll", "pan"} {
		if strings.Contains(got, removed) {
			t.Fatalf("output help bar still shows %q hint", removed)
		}
	}
	for _, kept := range []string{"enlarge", "copy", "back", "finalize", "global", "about", "quit"} {
		if !strings.Contains(got, kept) {
			t.Fatalf("output help bar lost %q hint", kept)
		}
	}
}

func TestHelpBarDocsPaneHints(t *testing.T) {
	m := newModel()
	m.focusPane = focusDocs

	got := stripANSI(m.renderHelpBar(120))

	if strings.Contains(got, "scroll") {
		t.Fatal("docs help bar still shows the scroll hint")
	}
	for _, kept := range []string{"enlarge", "copy", "finalize", "back", "global", "about"} {
		if !strings.Contains(got, kept) {
			t.Fatalf("docs help bar lost %q hint", kept)
		}
	}
}

func TestHelpBarInputsPaneHints(t *testing.T) {
	m := newModel()
	m.focusPane = focusInputs

	got := stripANSI(m.renderHelpBar(120))

	if strings.Contains(got, "navigate") {
		t.Fatal("inputs help bar still shows the navigate hint")
	}
	for _, kept := range []string{"next", "finalize", "back"} {
		if !strings.Contains(got, kept) {
			t.Fatalf("inputs help bar lost %q hint", kept)
		}
	}
}

// helpBarHints renders the help bar and returns its ordered "key desc" hint
// labels, ignoring ANSI styling and spacing. The right-side badges are cut off
// at the cwd marker, and the contextual group after the " | " separator is
// appended after the base hints, so the returned slice is the exact visible
// hint sequence.
func helpBarHints(t *testing.T, m mainModel) []string {
	t.Helper()
	plain := stripANSI(m.renderHelpBar(120))
	if i := strings.Index(plain, "📂"); i >= 0 {
		plain = plain[:i]
	}
	re := regexp.MustCompile(`(\S+)\s+(\S+)`)
	var hints []string
	for _, part := range strings.Split(plain, " | ") {
		for _, m := range re.FindAllStringSubmatch(part, -1) {
			hints = append(hints, m[1]+" "+m[2])
		}
	}
	return hints
}

func TestHelpBarComposerHintOrder(t *testing.T) {
	m := newModel()
	m.focusPane = focusCategories

	got := helpBarHints(t, m)
	want := []string{"d/D docs", "o/O output", "tab finalize", "g global", "? about", "q quit", "i inputs"}
	if !slices.Equal(got, want) {
		t.Fatalf("composer help bar hints = %v, want %v", got, want)
	}
}

func TestHelpBarOutputHintOrder(t *testing.T) {
	m := newModel()
	m.focusPane = focusOutput

	got := helpBarHints(t, m)
	want := []string{"O enlarge", "c copy", "tab finalize", "esc back", "g global", "? about", "q quit", "i inputs"}
	if !slices.Equal(got, want) {
		t.Fatalf("output help bar hints = %v, want %v", got, want)
	}
}

func TestHelpBarDocsHintOrder(t *testing.T) {
	m := newModel()
	m.focusPane = focusDocs

	got := helpBarHints(t, m)
	want := []string{"D enlarge", "c copy", "tab finalize", "esc back", "g global", "? about", "q quit", "i inputs"}
	if !slices.Equal(got, want) {
		t.Fatalf("docs help bar hints = %v, want %v", got, want)
	}
}

func TestHelpBarCmdBarHintOrder(t *testing.T) {
	m := newModel()
	m.focusPane = focusCmdBar

	got := helpBarHints(t, m)
	want := []string{"enter execute", "c copy", "esc back", "g global", "? about", "q quit", "i inputs"}
	if !slices.Equal(got, want) {
		t.Fatalf("command-bar help bar hints = %v, want %v", got, want)
	}
}

func TestHelpBarInputsHintOrder(t *testing.T) {
	m := newModel()
	m.focusPane = focusInputs

	got := helpBarHints(t, m)
	want := []string{"enter next", "tab finalize", "esc back"}
	if !slices.Equal(got, want) {
		t.Fatalf("inputs help bar hints = %v, want %v", got, want)
	}
}

// The recall hint is advertised wherever "r" actually recalls: every pane but
// the inputs box, enlarged output/docs included, and only once a command has
// been executed.
func TestHelpBarRecallHintScope(t *testing.T) {
	for _, pane := range []int{focusCategories, focusFlags, focusOutput, focusDocs, focusCmdBar} {
		m := newModel()
		m.focusPane = pane
		m.lastCmd = &commandSnapshot{}

		if !slices.Contains(helpBarHints(t, m), "r recall") {
			t.Errorf("pane %d help bar missing the recall hint", pane)
		}
	}

	for _, enlarged := range []struct {
		pane int
		set  func(*mainModel)
	}{
		{focusOutput, func(m *mainModel) { m.outputEnlarged = true }},
		{focusDocs, func(m *mainModel) { m.docsEnlarged = true }},
	} {
		m := newModel()
		m.focusPane = enlarged.pane
		m.lastCmd = &commandSnapshot{}
		enlarged.set(&m)

		if !slices.Contains(helpBarHints(t, m), "r recall") {
			t.Errorf("enlarged pane %d help bar missing the recall hint", enlarged.pane)
		}
	}

	m := newModel()
	m.focusPane = focusInputs
	m.lastCmd = &commandSnapshot{}
	if slices.Contains(helpBarHints(t, m), "r recall") {
		t.Error("inputs help bar advertises recall, where r is typed text")
	}

	m = newModel()
	m.focusPane = focusOutput
	if slices.Contains(helpBarHints(t, m), "r recall") {
		t.Error("help bar advertises recall with no command recorded")
	}
}

func TestHelpBarContextualHintOrder(t *testing.T) {
	m := newModel()
	m.focusPane = focusFlags
	m.lastCmd = &commandSnapshot{}

	got := helpBarHints(t, m)
	want := []string{
		"d/D docs", "o/O output", "tab finalize", "g global", "? about", "q quit",
		"i inputs", "r recall", "spc toggle",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("help bar hints = %v, want %v", got, want)
	}

	// The contextual group is set off by a visible separator between the
	// last base hint and the first contextual hint.
	plain := stripANSI(m.renderHelpBar(120))
	sep := strings.Index(plain, " | ")
	if sep < 0 {
		t.Fatal("contextual hints not separated from base hints")
	}
	base := strings.Join(strings.Fields(plain[:sep]), " ")
	ctx := strings.Join(strings.Fields(plain[sep:]), " ")
	if !strings.HasSuffix(base, "q quit") || !strings.HasPrefix(ctx, "| i inputs") {
		t.Fatalf("separator not between base and contextual hints: %q", plain)
	}
}
