package main

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const barTitleWidth = 9 // COMMAND plus one space on either side

// barWidths reserves room for a symmetric command region even at narrow widths.
func barWidths(width int) (title, body int) {
	title = min(barTitleWidth, max(0, (width-7)/2))
	body = max(0, width-3-title) // two outer borders and the divider
	return
}

// renderTitledBar draws a single frame so the title and content share junctions.
// Content is bounded before styling: fixed-width Lip Gloss styles wrap, not clip.
func renderTitledBar(title string, width, height int, lines []string, border lipgloss.Border, style lipgloss.Style) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	titleW, bodyW := barWidths(width)
	rows := max(height-2, len(lines))
	result := make([]string, 0, rows+2)
	topJoin, bottomJoin := "┬", "┴"
	if border.Top == "━" {
		topJoin, bottomJoin = "┳", "┻"
	}
	result = append(result, style.Render(border.TopLeft+strings.Repeat(border.Top, titleW)+topJoin+strings.Repeat(border.Top, bodyW)+border.TopRight))
	for i := 0; i < rows; i++ {
		label, content := "", ""
		if i == (rows-1)/2 {
			label = barTitleStyle.Render(ansi.Truncate(title, titleW, ""))
		}
		if i < len(lines) {
			content = ansi.Truncate(lines[i], bodyW, "")
		}
		result = append(result, style.Render(border.Left)+lipgloss.PlaceHorizontal(titleW, lipgloss.Center, label)+style.Render(border.Left)+content+strings.Repeat(" ", max(0, bodyW-lipgloss.Width(content)))+style.Render(border.Right))
	}
	result = append(result, style.Render(border.BottomLeft+strings.Repeat(border.Bottom, titleW)+bottomJoin+strings.Repeat(border.Bottom, bodyW)+border.BottomRight))
	for i := range result {
		result[i] = ansi.Truncate(result[i], width, "")
	}
	if height < 2 {
		result = result[:height]
	}
	return strings.Join(result, "\n")
}
