package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestCommandBarFullWidthCenter(t *testing.T) {
	for _, width := range []int{60, 87, 120, 121} {
		for _, command := range []string{"jj log", "jj describe --message 日本語", strings.Repeat("jj log ", 20)} {
			m := newModel()
			m.inv.long = command
			rendered := ansi.Strip(m.renderCommandBar(width))
			rows := strings.Split(rendered, "\n")
			if !strings.Contains(rendered, "COMMAND") {
				t.Fatal("missing command title")
			}
			if lipgloss.Height(rendered) < 5 {
				t.Fatal("command bar lost rows")
			}
			for _, row := range rows[1 : len(rows)-1] {
				if lipgloss.Width(row) != width {
					t.Fatalf("row width %d != %d", lipgloss.Width(row), width)
				}
				parts := strings.Split(row, "│")
				if len(parts) != 4 {
					t.Fatalf("invalid divider: %q", row)
				}
				body := parts[2]
				text := strings.TrimSpace(body)
				if text == "" {
					continue
				}
				left := lipgloss.Width(strings.TrimRight(body, " ")) - lipgloss.Width(text)
				start := 11 + left // border + 9-cell title + divider
				errorCells := 2*start + lipgloss.Width(text) - width
				if errorCells < -1 || errorCells > 1 {
					t.Fatalf("w=%d command=%q center error=%d row=%q", width, command, errorCells, row)
				}
			}
		}
	}
}

func TestTitledBarsBoundedAtNarrowWidths(t *testing.T) {
	m := newModel()
	for width := 1; width <= 40; width++ {
		for _, title := range []string{"INPUTS", "COMMAND"} {
			border, style := barFrameStyle(false, false, false)
			rendered := renderTitledBar(title, width, 3, []string{strings.Repeat("x", 100)}, border, style)
			if lipgloss.Height(rendered) != 3 {
				t.Fatalf("width=%d height=%d", width, lipgloss.Height(rendered))
			}
			for _, row := range strings.Split(rendered, "\n") {
				if lipgloss.Width(row) > width {
					t.Fatalf("width=%d overflow", width)
				}
			}
		}
		_ = m.renderCommandBar(width) // wrapping remains bounded, even at degenerate widths
	}
}

func barInputModel(t *testing.T) mainModel {
	t.Helper()
	m := newModel()
	m.width, m.height = 120, 40
	for ci := range m.categories {
		m.catIdx = ci
		for c := range m.categories[ci].Commands {
			m.cmdIdx = c
			m.subIdx = 0
			if len(m.getActiveCombined()) > 0 {
				m.layoutViewports()
				return m
			}
		}
	}
	t.Fatal("no input command")
	return m
}

func TestInputBarTitlesAndScrolling(t *testing.T) {
	m := barInputModel(t)
	m.focusPane = focusInputs
	combined := m.getActiveCombined()
	item := combined[0]
	ti := m.getInput(item)
	ti.Focus()
	ti.SetValue(strings.Repeat("abcdef", 50))
	ti.CursorEnd()
	for i := 0; i < 100; i++ {
		ti, _ = ti.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	}
	m.setInput(item, ti)
	width, _ := m.getLayoutWidths()
	rendered := m.renderInputPanes(width, len(combined)+2, combined)
	if !strings.Contains(ansi.Strip(rendered), "INPUTS") {
		t.Fatal("missing input title")
	}
	if lipgloss.Height(rendered) != len(combined)+2 {
		t.Fatal("input scrolling grew bar")
	}
	for _, row := range strings.Split(rendered, "\n") {
		if lipgloss.Width(row) != width {
			t.Fatal("input row width changed")
		}
	}
	if !strings.Contains(ansi.Strip(rendered), "┳") {
		t.Fatal("focused title missing thick junction")
	}
}

func TestCommandBarInEveryView(t *testing.T) {
	for _, mode := range []int{focusCategories, focusOutput, focusDocs} {
		m := barInputModel(t)
		m.focusPane = mode
		m.outputEnlarged = mode == focusOutput
		m.docsEnlarged = mode == focusDocs
		m.layoutViewports()
		frame := ansi.Strip(m.renderContentFrame())
		if strings.Count(frame, " COMMAND ") != 1 {
			t.Fatal("missing or duplicate command title")
		}
		if lipgloss.Height(frame) != m.height {
			t.Fatalf("mode=%d frame height=%d", mode, lipgloss.Height(frame))
		}
	}
}

func TestInputBarValidationFrame(t *testing.T) {
	m := barInputModel(t)
	found := false
outer:
	for ci := range m.categories {
		m.catIdx = ci
		for cmdi := range m.categories[ci].Commands {
			m.cmdIdx = cmdi
			m.subIdx = 0
			if m.hasEmptyRequiredArg() {
				found = true
				break outer
			}
		}
	}
	if !found {
		t.Fatal("no required argument found")
	}
	m.focusPane = focusCategories
	m.layoutViewports()
	width, _ := m.getLayoutWidths()
	combined := m.getActiveCombined()
	before := ansi.Strip(m.renderInputPanes(width, len(combined)+2, combined))
	m.validationFlash = true
	after := ansi.Strip(m.renderInputPanes(width, len(combined)+2, combined))
	if !strings.Contains(before, "┬") || !strings.Contains(after, "┳") || !strings.Contains(after, "┻") {
		t.Fatal("validation must update title and content frame together")
	}
	_, style := barFrameStyle(false, false, true)
	if style.GetForeground() != colorRed {
		t.Fatal("validation frame must be red")
	}
}
