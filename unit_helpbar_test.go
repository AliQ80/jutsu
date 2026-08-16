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
