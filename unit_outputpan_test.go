package main

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestOutputWidthAndPanBounds(t *testing.T) {
	m := newModel()
	m.outputLines = []string{"\x1b[31mred\x1b[0m", "wide界"}
	m.output.SetContentLines(m.outputLines)
	m.output.SetWidth(5)

	if got := m.outputWidth(); got != 6 {
		t.Fatalf("output width = %d, want 6", got)
	}
	if got := m.maxXOffset(); got != 1 {
		t.Fatalf("max x offset = %d, want 1", got)
	}

	m.output.SetXOffset(99)
	if m.output.XOffset() != 1 {
		t.Fatalf("clamped x offset = %d, want 1", m.output.XOffset())
	}
}

func TestOutputRightPanStopsAtMaximum(t *testing.T) {
	m := newModel()
	m.width, m.height = 100, 20
	m.focusPane = focusOutput
	m.outputEnlarged = true
	m.outputLines = []string{strings.Repeat("x", 200)}
	m.output.SetContentLines(m.outputLines)
	m.layoutViewports()

	for range 100 {
		next, _ := m.handleOutputKeys(tea.KeyPressMsg{Code: tea.KeyRight})
		m = next.(mainModel)
	}
	if got, want := m.output.XOffset(), m.maxXOffset(); got != want {
		t.Fatalf("x offset = %d, want maximum %d", got, want)
	}
}

func TestOutputPanPreservesVerticalPosition(t *testing.T) {
	for _, scrolled := range []bool{false, true} {
		name := "top"
		if scrolled {
			name = "scrolled"
		}
		t.Run(name, func(t *testing.T) {
			m := newModel()
			m.width, m.height = 100, 20
			m.focusPane = focusOutput
			m.outputEnlarged = true
			m.outputLines = []string{strings.Repeat("x", 200), strings.Repeat("y", 200)}
			for i := 0; i < 30; i++ {
				m.outputLines = append(m.outputLines, fmt.Sprintf("line %02d %s", i, strings.Repeat("z", 200)))
			}
			m.output.SetContentLines(m.outputLines)
			m.layoutViewports()
			if scrolled {
				m.output.SetYOffset(10)
			}

			yBefore := m.output.YOffset()
			viewBefore := m.output.View()

			next, _ := m.handleOutputKeys(tea.KeyPressMsg{Code: tea.KeyRight})
			m = next.(mainModel)

			if m.output.YOffset() != yBefore {
				t.Fatalf("vertical offset changed: %d -> %d", yBefore, m.output.YOffset())
			}
			viewAfter := m.output.View()
			beforeLines := strings.Split(stripANSI(viewBefore), "\n")
			afterLines := strings.Split(stripANSI(viewAfter), "\n")
			if len(afterLines) != len(beforeLines) {
				t.Fatalf("visible row count changed: %d -> %d", len(beforeLines), len(afterLines))
			}
			// Panning right shifts content left by 4 cells: each visible row after
			// starts with the same row before, minus its first 4 cells
			// (trailing width padding ignored).
			for i := range afterLines {
				before := strings.TrimRight(beforeLines[i], " ")
				after := strings.TrimRight(afterLines[i], " ")
				if len(before) < 4 || !strings.HasPrefix(after, before[4:]) {
					t.Fatalf("row %d content changed unexpectedly: %q -> %q", i, before, after)
				}
			}
		})
	}
}

func TestOutputPanBoundaries(t *testing.T) {
	m := newModel()
	m.width, m.height = 100, 20
	m.focusPane = focusOutput
	m.outputEnlarged = true
	m.outputLines = []string{strings.Repeat("x", 200)}
	m.output.SetContentLines(m.outputLines)
	m.layoutViewports()

	// Left at the origin stays at the origin.
	next, _ := m.handleOutputKeys(tea.KeyPressMsg{Code: tea.KeyLeft})
	m = next.(mainModel)
	if m.output.XOffset() != 0 {
		t.Fatalf("left at origin moved to %d, want 0", m.output.XOffset())
	}

	// Right clamps at the maximum.
	for range 100 {
		next, _ = m.handleOutputKeys(tea.KeyPressMsg{Code: tea.KeyRight})
		m = next.(mainModel)
	}
	if got, want := m.output.XOffset(), m.maxXOffset(); got != want {
		t.Fatalf("x offset = %d, want maximum %d", got, want)
	}

	// Further right stays clamped.
	next, _ = m.handleOutputKeys(tea.KeyPressMsg{Code: tea.KeyRight})
	m = next.(mainModel)
	if got, want := m.output.XOffset(), m.maxXOffset(); got != want {
		t.Fatalf("x offset = %d, want maximum %d", got, want)
	}
}

func TestOutputPanRoundTrip(t *testing.T) {
	m := newModel()
	m.width, m.height = 100, 20
	m.focusPane = focusOutput
	m.outputEnlarged = true
	m.outputLines = []string{strings.Repeat("x", 200), strings.Repeat("y", 200)}
	m.output.SetContentLines(m.outputLines)
	m.layoutViewports()

	viewBefore := m.output.View()
	var next tea.Model
	for range 3 {
		next, _ = m.handleOutputKeys(tea.KeyPressMsg{Code: tea.KeyRight})
		m = next.(mainModel)
	}
	for range 3 {
		next, _ = m.handleOutputKeys(tea.KeyPressMsg{Code: tea.KeyLeft})
		m = next.(mainModel)
	}
	if m.output.XOffset() != 0 {
		t.Fatalf("x offset = %d after round trip, want 0", m.output.XOffset())
	}
	if got := m.output.View(); got != viewBefore {
		t.Fatalf("view not restored after round trip")
	}
}

func TestOutputScrollbarLayout(t *testing.T) {
	cases := []struct {
		name       string
		lines      []string
		wantBottom bool
		wantSide   bool
	}{
		{"fits", []string{"short"}, false, false},
		{"horizontal", []string{strings.Repeat("x", 200)}, true, false},
		{"ansi", []string{"\x1b[31m" + strings.Repeat("x", 200) + "\x1b[0m"}, true, false},
		{"wide-unicode", []string{strings.Repeat("界", 100)}, true, false},
		{"both", append([]string{strings.Repeat("x", 200)}, make([]string, 40)...), true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newModel()
			m.outputLines = tc.lines
			m.output.SetContentLines(tc.lines)
			got := stripANSI(m.renderRightPane(30, 10))
			lines := strings.Split(got, "\n")
			if len(lines) != 10 {
				t.Fatalf("pane height = %d, want 10", len(lines))
			}

			bottom := lines[len(lines)-2]
			if has := strings.Contains(bottom, "─"); has != tc.wantBottom {
				t.Fatalf("bottom scrollbar = %v, want %v: %q", has, tc.wantBottom, bottom)
			}
			hasSide := false
			for _, line := range lines[2 : len(lines)-1] {
				runes := []rune(line)
				if len(runes) >= 3 && runes[len(runes)-3] == '│' {
					hasSide = true
					break
				}
			}
			if hasSide != tc.wantSide {
				t.Fatalf("vertical scrollbar = %v, want %v", hasSide, tc.wantSide)
			}
			for _, line := range lines {
				if got := lipgloss.Width(line); got != 30 {
					t.Fatalf("rendered width = %d, want 30: %q", got, line)
				}
			}
		})
	}
}

func TestOutputScrollbarThumbMovement(t *testing.T) {
	at0 := stripANSI(renderHorizontalScrollbar(20, 100, 20, 0, scrollbarThumbStyle))
	atMid := stripANSI(renderHorizontalScrollbar(20, 100, 20, 40, scrollbarThumbStyle))
	atMax := stripANSI(renderHorizontalScrollbar(20, 100, 20, 80, scrollbarThumbStyle))

	if at0 == atMid || atMid == atMax || at0 == atMax {
		t.Fatalf("thumb should move with x offset")
	}
	if !strings.HasPrefix(at0, "▬") {
		t.Fatalf("thumb at origin should be left-aligned: %q", at0)
	}
	if !strings.HasSuffix(atMax, "▬") {
		t.Fatalf("thumb at maximum should be right-aligned: %q", atMax)
	}
	if got := lipgloss.Width(at0); got != 20 {
		t.Fatalf("horizontal scrollbar width = %d, want 20", got)
	}
}

func TestOutputScrollbarThumbGlyphs(t *testing.T) {
	horizontal := stripANSI(renderHorizontalScrollbar(12, 24, 12, 0, scrollbarThumbStyle))
	wantHorizontal := strings.Repeat("▬", 6) + strings.Repeat("─", 6)
	if horizontal != wantHorizontal {
		t.Fatalf("horizontal scrollbar = %q, want %q", horizontal, wantHorizontal)
	}
	if got := lipgloss.Width(horizontal); got != 12 {
		t.Fatalf("horizontal scrollbar width = %d, want 12", got)
	}

	vertical := stripANSI(renderScrollbar(12, 24, 12, 0, scrollbarThumbStyle))
	wantVertical := strings.Repeat("█\n", 5) + "█\n" + strings.Repeat("│\n", 5) + "│"
	if vertical != wantVertical {
		t.Fatalf("vertical scrollbar = %q, want %q", vertical, wantVertical)
	}
	if got := lipgloss.Width(vertical); got != 1 {
		t.Fatalf("vertical scrollbar width = %d, want 1", got)
	}
}

// tabDiffLines is a diff-style output whose changed lines are tab-indented
// and long enough to overflow the narrow test pane.
func tabDiffLines() []string {
	return []string{
		"diff --git a/data.json b/data.json",
		"--- a/data.json",
		"+++ b/data.json",
		"@@ -1,3 +1,3 @@",
		"\t\"name\": \"jutsu\", \"version\": \"1.0.0\", \"author\": \"ali\", \"license\": \"MIT\", \"repository\": \"github.com/AliQ80/jutsu\"",
	}
}

// tabLineRow returns the index of the pane row containing needle, or -1.
func tabLineRow(rows []string, needle string) int {
	for i, row := range rows {
		if strings.Contains(row, needle) {
			return i
		}
	}
	return -1
}

func TestOutputTabLinesDoNotWrap(t *testing.T) {
	m := newModel()
	m.width, m.height = 100, 20
	m.focusPane = focusOutput
	m.outputEnlarged = true
	m.setOutputLines(tabDiffLines())
	m.layoutViewports()

	// The tab-expanded line must overflow the pane or the test proves nothing.
	if m.outputWidth() <= m.output.Width() {
		t.Fatalf("test setup: output width %d should exceed viewport width %d", m.outputWidth(), m.output.Width())
	}

	render := func() []string {
		return strings.Split(stripANSI(m.renderRightPane(30, 10)), "\n")
	}

	// At the origin: one visual row per source line — the tab-indented line
	// must not wrap into a second row, and the pane height must stay stable.
	rows := render()
	if len(rows) != 10 {
		t.Fatalf("pane height = %d, want 10 (no wrapped rows)", len(rows))
	}
	tabRow := tabLineRow(rows, `"name": "jutsu"`)
	if tabRow == -1 {
		t.Fatalf("tab line not visible at origin")
	}
	// Row layout: border + padding prefix, then the tab expanded to 4 spaces.
	content := rows[tabRow]
	if len([]rune(content)) > 2 {
		content = string([]rune(content)[2:])
	}
	if !strings.HasPrefix(content, "    \"name\":") {
		t.Fatalf("tab line row = %q, want tab expanded to four spaces", rows[tabRow])
	}
	// The tab line occupies exactly one row — no wrapped continuation.
	if got := strings.Count(strings.Join(rows, "\n"), `"name": "jutsu"`); got != 1 {
		t.Fatalf("tab line appears %d times, want exactly 1 row", got)
	}
	if !strings.Contains(rows[len(rows)-2], "▬") {
		t.Fatalf("horizontal scrollbar missing for overflowing tab line: %q", rows[len(rows)-2])
	}

	// Pan right: same single-row-per-source-line property, content shifted.
	next, _ := m.handleOutputKeys(tea.KeyPressMsg{Code: tea.KeyRight})
	m = next.(mainModel)
	if m.output.XOffset() == 0 {
		t.Fatalf("pan right did not move the horizontal offset")
	}
	rows = render()
	if len(rows) != 10 {
		t.Fatalf("pane height after pan = %d, want 10", len(rows))
	}
	if tabLineRow(rows, `"name": "jutsu"`) == -1 {
		t.Fatalf("tab line not visible after panning")
	}
	if got := strings.Count(strings.Join(rows, "\n"), `"name": "jutsu"`); got != 1 {
		t.Fatalf("tab line appears %d times after pan, want exactly 1 row", got)
	}

	// Replacing the output resets the horizontal position and re-normalizes.
	m.setOutputLines(tabDiffLines())
	if m.output.XOffset() != 0 {
		t.Fatalf("x offset after new output = %d, want 0", m.output.XOffset())
	}
	rows = render()
	if len(rows) != 10 {
		t.Fatalf("pane height after replacement = %d, want 10", len(rows))
	}
	if tabLineRow(rows, `"name": "jutsu"`) == -1 {
		t.Fatalf("tab line not visible after replacement")
	}
}

func TestOutputAnsiTabLinesDoNotWrap(t *testing.T) {
	lines := []string{
		"\x1b[31m--- a/data.json\x1b[0m",
		"\x1b[32m+++ b/data.json\x1b[0m",
		"\x1b[36m@@ -1,3 +1,3 @@\x1b[0m",
		"\x1b[31m-\t\"old\": \"value\", \"key\": \"old-key\", \"nested\": {\"deep\": \"old-value\"}\x1b[0m",
		"\x1b[32m+\t\"new\": \"value\", \"key\": \"new-key\", \"nested\": {\"deep\": \"new-value\"}\x1b[0m",
	}
	m := newModel()
	m.setOutputLines(lines)
	m.layoutViewports()

	if m.outputWidth() <= m.output.Width() {
		t.Fatalf("test setup: output width %d should exceed viewport width %d", m.outputWidth(), m.output.Width())
	}

	render := func() []string {
		return strings.Split(m.renderRightPane(30, 10), "\n")
	}

	// At the origin: one visual row per styled source line, ANSI intact.
	rows := render()
	if len(rows) != 10 {
		t.Fatalf("pane height = %d, want 10 (no wrapped rows)", len(rows))
	}
	for _, needle := range []string{`"old": "value"`, `"new": "value"`} {
		if got := strings.Count(strings.Join(rows, "\n"), needle); got != 1 {
			t.Fatalf("styled tab line %q appears %d times, want exactly 1 row", needle, got)
		}
	}
	if !strings.Contains(strings.Join(rows, "\n"), "\x1b[31m") || !strings.Contains(strings.Join(rows, "\n"), "\x1b[32m") {
		t.Fatalf("ANSI styling lost in rendered pane")
	}
	if !strings.Contains(rows[len(rows)-2], "▬") {
		t.Fatalf("horizontal scrollbar missing for overflowing styled tab line: %q", rows[len(rows)-2])
	}

	// Pan right: row count and styling remain stable.
	next, _ := m.handleOutputKeys(tea.KeyPressMsg{Code: tea.KeyRight})
	m = next.(mainModel)
	if m.output.XOffset() == 0 {
		t.Fatalf("pan right did not move the horizontal offset")
	}
	rows = render()
	if len(rows) != 10 {
		t.Fatalf("pane height after pan = %d, want 10", len(rows))
	}
	for _, needle := range []string{`"old": "value"`, `"new": "value"`} {
		if got := strings.Count(strings.Join(rows, "\n"), needle); got != 1 {
			t.Fatalf("styled tab line %q appears %d times after pan, want exactly 1 row", needle, got)
		}
	}
	if !strings.Contains(strings.Join(rows, "\n"), "\x1b[31m") || !strings.Contains(strings.Join(rows, "\n"), "\x1b[32m") {
		t.Fatalf("ANSI styling lost after panning")
	}
}
