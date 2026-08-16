package main

import (
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
