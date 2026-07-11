package main

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Fixed pane widths sized to their longest content (content + 4 for border/padding).
const (
	catPaneW  = 19 // "CATEGORIES" title (10) + scroll-arrow prefix/suffix (2+2) + 1 margin
	cmdPaneW  = 22 // "simplify-parents" = 16
	subPaneW  = 22 // "completion bash" = 15
	flagPaneW = 22 // full name visible in docs box on hover

	scrollbarW    = 1 // column reserved for the output/docs viewport scroll indicator
	scrollbarGapW = 1 // blank column separating content from the scroll indicator
)

func (m mainModel) getLayoutWidths() (int, int) {
	leftWidth := catPaneW + cmdPaneW + subPaneW + flagPaneW
	return leftWidth, m.width - leftWidth
}

func (m mainModel) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("Loading...")
	}

	if m.outputEnlargedActive() {
		height := m.height - 6 // 5 cmdBar + 1 helpBar
		rightPane := m.renderRightPane(m.width, height)
		cmdBar := m.renderCommandBar(m.width)
		helpBar := m.renderHelpBar(m.width)
		content := lipgloss.JoinVertical(lipgloss.Left, rightPane, cmdBar, helpBar)
		v := tea.NewView(content)
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	if m.docsEnlargedActive() {
		height := m.height - 6 // 5 cmdBar + 1 helpBar
		if height < 6 {
			height = 6
		}
		docsPane := m.renderDescriptionBar(m.width, height)
		cmdBar := m.renderCommandBar(m.width)
		helpBar := m.renderHelpBar(m.width)
		content := lipgloss.JoinVertical(lipgloss.Left, docsPane, cmdBar, helpBar)
		v := tea.NewView(content)
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	leftWidth, rightWidth := m.getLayoutWidths()

	combined := m.getActiveCombined()
	inputsHeight := 0
	if len(combined) > 0 {
		inputsHeight = len(combined) + 2
	}

	outputHeight := m.height - 6 // 5 cmdBar + 1 helpBar
	if outputHeight < 6 {
		outputHeight = 6
	}

	leftBudget := outputHeight - inputsHeight
	if leftBudget < 6 {
		leftBudget = 6
	}

	composerH := leftBudget / 2
	descBarH := leftBudget - composerH

	composerSection := m.renderLeftPane(composerH)
	leftColumn := composerSection
	if descBarH >= 3 {
		descBar := m.renderDescriptionBar(leftWidth, descBarH)
		leftColumn = lipgloss.JoinVertical(lipgloss.Left, leftColumn, descBar)
	}
	if inputsHeight > 0 {
		inputsPane := m.renderInputPanes(leftWidth, inputsHeight, combined)
		leftColumn = lipgloss.JoinVertical(lipgloss.Left, leftColumn, inputsPane)
	}

	rightPane := m.renderRightPane(rightWidth, outputHeight)
	topSection := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightPane)

	cmdBar := m.renderCommandBar(m.width)
	helpBar := m.renderHelpBar(m.width)

	topSection = strings.TrimRight(topSection, "\n")
	content := lipgloss.JoinVertical(lipgloss.Left, topSection, cmdBar, helpBar)
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func hasRequiredArgs(args []Arg) bool {
	for _, a := range args {
		if a.Required {
			return true
		}
	}
	return false
}

// argsMarker returns the list-pane suffix hinting at positional args: "<…>" when at
// least one is required, "[…]" when only optional ones exist, "" when there are none.
func argsMarker(args []Arg) string {
	if hasRequiredArgs(args) {
		return " <…>"
	}
	if len(args) > 0 {
		return " […]"
	}
	return ""
}

func (m mainModel) renderLeftPane(height int) string {
	return lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderPane(0, "CATEGORIES", catPaneW, height),
		m.renderPane(1, "COMMANDS", cmdPaneW, height),
		m.renderPane(2, "SUB-CMDS", subPaneW, height),
		m.renderPane(3, "FLAGS", flagPaneW, height),
	)
}

func (m mainModel) renderPane(paneIdx int, title string, width, height int) string {
	var items []string
	var selectedIdx int

	switch paneIdx {
	case focusCategories:
		for _, cat := range m.categories {
			items = append(items, cat.Name)
		}
		selectedIdx = m.catIdx
	case focusCommands:
		cmds := m.currentCommands()
		for _, cmd := range cmds {
			label := cmd.Name + argsMarker(cmd.Args)
			items = append(items, label)
		}
		selectedIdx = m.cmdIdx
	case focusSubcmds:
		for _, sub := range m.currentSubCommands() {
			label := sub.Name + argsMarker(sub.Args)
			items = append(items, label)
		}
		selectedIdx = m.subIdx
	case focusFlags:
		flags := m.currentFlags()
		if len(flags) > 0 {
			group := m.currentRequiredFlagGroup()
			groupUnsatisfied := m.requiredFlagGroupUnsatisfied()
			for _, f := range flags {
				check := "[ ]"
				if f.Mandatory {
					check = "[*]" // locked-on, cannot be deselected
				} else if f.Selected {
					check = "[x]"
				} else if isConflicted(f, flags) {
					check = "[-]" // blocked by a conflicting selection
				}
				label := fmt.Sprintf("%s %s", check, f.Name)
				if groupUnsatisfied && slices.Contains(group, f.Value) {
					label += " *" // one of these is required
				}
				items = append(items, label)
			}
			selectedIdx = m.flagIdx
		}
	}

	contentHeight := height - 4 // 3 for title box, 1 for bottom border
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Scroll offset is maintained in the model so the cursor moves freely
	// within the visible window and the list only scrolls at the edges.
	scrollOffset := 0
	switch paneIdx {
	case focusCategories:
		scrollOffset = m.catScroll
	case focusCommands:
		scrollOffset = m.cmdScroll
	case focusSubcmds:
		scrollOffset = m.subScroll
	case focusFlags:
		scrollOffset = m.flagScroll
	}

	isFocused := m.focusPane == paneIdx
	var content strings.Builder
	for i := 0; i < contentHeight; i++ {
		if i > 0 {
			content.WriteString("\n")
		}
		itemIdx := i + scrollOffset
		if itemIdx < len(items) {
			if itemIdx == selectedIdx {
				if paneIdx == focusFlags {
					flags := m.currentFlags()
					if itemIdx < len(flags) {
						f := flags[itemIdx]
						switch {
						case f.Mandatory:
							content.WriteString(flagMandatoryStyle.Render(truncateItem(items[itemIdx], width)))
						case !isFocused:
							// Unfocused flags pane: cursor position has no special highlight;
							// render the flag's own state so it blends with other flags.
							if f.Selected {
								content.WriteString(activeSelectionStyle.Render(truncateItem(items[itemIdx], width)))
							} else if isConflicted(f, flags) {
								content.WriteString(flagConflictedStyle.Render(truncateItem(items[itemIdx], width)))
							} else if isRequiredGroupFlag(m, f) {
								content.WriteString(flagRequiredStyle.Render(truncateItem(items[itemIdx], width)))
							} else {
								content.WriteString(flagUnselectedStyle.Render(truncateItem(items[itemIdx], width)))
							}
						case isConflicted(f, flags):
							content.WriteString(flagConflictedFocusedStyle.Render(truncateItem(items[itemIdx], width)))
						case isRequiredGroupFlag(m, f):
							content.WriteString(flagRequiredFocusedStyle.Render(truncateItem(items[itemIdx], width)))
						default:
							content.WriteString(selectedItemStyle.Render(truncateItem(items[itemIdx], width)))
						}
					}
				} else {
					// Categories, Commands, Subcommands
					if isFocused {
						content.WriteString(selectedItemStyle.Render(truncateItem(items[itemIdx], width)))
					} else {
						content.WriteString(activeSelectionStyle.Render(truncateItem(items[itemIdx], width)))
					}
				}
			} else {
				if paneIdx == focusFlags && itemIdx < len(m.currentFlags()) {
					flags := m.currentFlags()
					if flags[itemIdx].Mandatory {
						content.WriteString(flagMandatoryStyle.Render(truncateItem(items[itemIdx], width)))
					} else if flags[itemIdx].Selected {
						// ponytail: was flagSelectedStyle (colorText) — now activeSelectionStyle (sapphire+bold) for visual consistency with unfocused-pane selections.
						content.WriteString(activeSelectionStyle.Render(truncateItem(items[itemIdx], width)))
					} else if isConflicted(flags[itemIdx], flags) {
						content.WriteString(flagConflictedStyle.Render(truncateItem(items[itemIdx], width)))
					} else if isRequiredGroupFlag(m, flags[itemIdx]) {
						content.WriteString(flagRequiredStyle.Render(truncateItem(items[itemIdx], width)))
					} else {
						content.WriteString(flagUnselectedStyle.Render(truncateItem(items[itemIdx], width)))
					}
				} else {
					content.WriteString(normalItemStyle.Render(truncateItem(items[itemIdx], width)))
				}
			}
		}
	}

	// Ensure content fills the height by padding with empty lines
	contentStr := content.String()
	lines := strings.Split(contentStr, "\n")
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}
	contentStr = strings.Join(lines, "\n")

	var tStyle, cStyle lipgloss.Style
	if m.focusPane == paneIdx {
		tStyle = activeTitleBorderStyle.Copy()
		cStyle = activeContentBorderStyle.Copy()
	} else {
		tStyle = inactiveTitleBorderStyle.Copy()
		cStyle = inactiveContentBorderStyle.Copy()
	}
	if paneIdx == focusFlags && m.validationFlash && m.requiredFlagGroupUnsatisfied() {
		tStyle = tStyle.BorderForeground(colorRed)
		cStyle = cStyle.BorderForeground(colorRed)
	}

	hasAbove := scrollOffset > 0
	hasBelow := scrollOffset+contentHeight < len(items)
	prefix, suffix := "", ""
	if hasAbove {
		prefix = dimItemStyle.Render("▲") + " "
	} else if hasBelow {
		prefix = "  " // balance the ▼ suffix
	}
	if hasBelow {
		suffix = " " + dimItemStyle.Render("▼")
	} else if hasAbove {
		suffix = "  " // balance the ▲ prefix
	}
	titleBox := tStyle.Width(width).Align(lipgloss.Center).Render(prefix + headerStyle.Render(title) + suffix)
	contentBox := cStyle.Width(width).Height(height - 3).Render(contentStr)

	return lipgloss.JoinVertical(lipgloss.Left, titleBox, contentBox)
}

func (m mainModel) renderRightPane(width, height int) string {
	var tStyle, cStyle lipgloss.Style
	if m.focusPane == focusOutput {
		tStyle = activeTitleBorderStyle.Copy()
		cStyle = activeContentBorderStyle.Copy()
	} else {
		tStyle = inactiveTitleBorderStyle.Copy()
		cStyle = inactiveContentBorderStyle.Copy()
	}

	var outputContent string
	if m.running {
		outputContent = runningStyle.Render("Executing command...")
	} else {
		outputContent = m.output.View()
	}

	contentHeight := height - 4
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Ensure content fills the height
	lines := strings.Split(outputContent, "\n")
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}

	content := strings.Join(lines, "\n")

	total, visible := m.output.TotalLineCount(), m.output.VisibleLineCount()
	if m.running {
		total, visible = 0, 0 // metrics are stale mid-exec; show a blank column instead
	}
	scrollbar := renderScrollbar(contentHeight, total, visible, m.output.YOffset())
	content = lipgloss.JoinHorizontal(lipgloss.Top, content, blankColumn(contentHeight), scrollbar)

	title := "OUTPUT"
	if m.outputEnlargedActive() {
		title = "OUTPUT (ENLARGED)"
	}
	titleBox := tStyle.Width(width).Align(lipgloss.Center).Render(headerStyle.Render(title))
	contentBox := cStyle.Width(width).Height(height - 3).Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, titleBox, contentBox)
}

func (m mainModel) renderDescriptionBar(width, height int) string {
	var tStyle, cStyle lipgloss.Style
	if m.focusPane == focusDocs {
		tStyle = activeTitleBorderStyle.Copy()
		cStyle = activeContentBorderStyle.Copy()
	} else {
		tStyle = inactiveTitleBorderStyle.Copy()
		cStyle = inactiveContentBorderStyle.Copy()
	}

	title := "DOCS"
	if m.docsEnlargedActive() {
		title = "DOCS (ENLARGED)"
	}
	titleBox := tStyle.Width(width).Align(lipgloss.Center).Render(headerStyle.Render(title))

	// Set dimensions on a local copy so View() renders at the right size.
	docsW, docsH := docsContentDims(width, height)
	m.docs.SetWidth(docsW)
	m.docs.SetHeight(docsH)
	scrollbar := renderScrollbar(docsH, m.docs.TotalLineCount(), m.docs.VisibleLineCount(), m.docs.YOffset())
	docsContent := lipgloss.JoinHorizontal(lipgloss.Top, m.docs.View(), blankColumn(docsH), scrollbar)
	contentBox := cStyle.Width(width).Height(height - 3).Render(docsContent)

	return lipgloss.JoinVertical(lipgloss.Left, titleBox, contentBox)
}

func (m mainModel) renderCommandBar(width int) string {
	// Show the word form (long name, or a shorter word alias) in the bar;
	// the shortest form (short flags, command aliases) is what gets executed.
	cmdDisplay := m.cmdTextLong
	if cmdDisplay == "" {
		cmdDisplay = m.cmdText
	}
	if cmdDisplay == "" {
		cmdDisplay = "jj"
	}

	// Height(5) = 2 borders + 3 content rows
	contentWidth := width - 4 // 2 borders + 2 padding

	renderedText := commandTextStyle.Render(cmdDisplay)
	centeredText := lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, renderedText)

	// Pre-wrap to size blank padding by line count; see CLAUDE.md "Known
	// fragile points" for why this can't just rely on the outer Style's wrap.
	wrapped := lipgloss.Wrap(centeredText, contentWidth, "")
	lines := strings.Count(wrapped, "\n") + 1

	var content strings.Builder
	if lines <= 1 {
		content.WriteString("\n")
		content.WriteString(wrapped)
		content.WriteString("\n")
	} else {
		content.WriteString(wrapped)
		if pad := 3 - lines; pad > 0 {
			content.WriteString(strings.Repeat("\n", pad))
		}
	}

	style := commandBarInactiveStyle.Copy().Width(width).Height(5)
	if m.focusPane == focusCmdBar {
		style = commandBarActiveStyle.Copy().Width(width).Height(5)
	}

	return style.Render(content.String())
}

// parseVersion parses "jj 0.42.0" / "v0.42.0" / "0.42.0" into major, minor, patch.
func parseVersion(s string) (major, minor, patch int) {
	s = strings.TrimPrefix(s, "jj ")
	s = strings.TrimPrefix(s, "v")
	fmt.Sscanf(s, "%d.%d.%d", &major, &minor, &patch)
	return
}

func (m mainModel) renderVersionBadge() string {
	const dot = "●"
	compat := "v" + compatJJVersion
	label := dot + " jj " + compat

	if m.jjVersion == "" {
		return helpDescStyle.Render(label)
	}

	iMaj, iMin, iPatch := parseVersion(m.jjVersion)
	cMaj, cMin, cPatch := parseVersion(compatJJVersion)

	// Extract bare installed version number for the parenthetical.
	installed := "v" + strings.TrimPrefix(m.jjVersion, "jj ")

	switch {
	case iMaj == cMaj && iMin == cMin && iPatch == cPatch:
		return versionGreenStyle.Render(label)
	case iMaj == cMaj && iMin == cMin:
		return versionYellowStyle.Render(label + " (" + installed + ")")
	default:
		return versionRedStyle.Render(label + " (" + installed + ")")
	}
}

func (m mainModel) renderHelpBar(width int) string {
	type entry struct{ key, desc string }

	var entries, contextual []entry
	switch m.focusPane {
	case focusOutput:
		entries = []entry{
			{"↑↓", "scroll"},
			{"←→", "pan"},
			{"O", "enlarge"},
			{"c", "copy"},
			{"esc", "back"},
			{"tab", "finalize"},
			{"q", "quit"},
		}
	case focusDocs:
		entries = []entry{
			{"↑↓", "scroll"},
			{"D", "enlarge"},
			{"c", "copy"},
			{"tab", "finalize"},
			{"esc", "back"},
		}
	case focusCmdBar:
		entries = []entry{
			{"enter", "execute"},
			{"c", "copy"},
			{"esc", "back"},
			{"q", "quit"},
		}
	case focusInputs:
		entries = []entry{
			{"↑↓", "navigate"},
			{"enter", "next"},
			{"tab", "finalize"},
			{"esc", "back"},
		}
	default: // composer panes 0-3
		entries = []entry{
			{"↑↓", "navigate"},
			{"←→", "pane"},
			{"d/D", "docs"},
			{"o/O", "output"},
			{"tab", "finalize"},
			{"q", "quit"},
		}
		if m.lastCmd != nil {
			contextual = append(contextual, entry{"r", "recall"})
		}
		if m.focusPane == focusFlags {
			contextual = append(contextual, entry{"spc", "toggle"})
		}
		for _, f := range m.currentFlags() {
			if f.Selected && f.RequiresInput {
				contextual = append(contextual, entry{"enter", "inputs"})
				break
			}
		}
	}

	render := func(e entry) string {
		return helpKeyStyle.Render(e.key) + " " + helpDescStyle.Render(e.desc)
	}
	var parts []string
	for _, e := range entries {
		parts = append(parts, render(e))
	}
	leftBar := strings.Join(parts, " ")
	if len(contextual) > 0 {
		var ctx []string
		for _, e := range contextual {
			ctx = append(ctx, render(e))
		}
		leftBar += helpDescStyle.Render(" | ") + strings.Join(ctx, " ")
	}
	if m.copyFlash {
		leftBar += " " + copyFlashStyle.Render("✓ copied")
	}

	cwdBadge := cwdStyle.Render("📂 " + m.cwd)
	badge := m.renderVersionBadge()
	rightBar := cwdBadge + "   " + badge
	// width-1 because helpBarStyle has PaddingLeft(1).
	gap := (width - 1) - lipgloss.Width(leftBar) - lipgloss.Width(rightBar)
	if gap < 1 {
		gap = 1
	}
	return helpBarStyle.Width(width).Render(leftBar + strings.Repeat(" ", gap) + rightBar)
}

// blankColumn returns a height-tall, 1-column string of spaces, used to
// reserve width without rendering visible content (scrollbar gap, or the
// scrollbar itself when there's nothing to scroll).
func blankColumn(height int) string {
	if height < 1 {
		return ""
	}
	return strings.Repeat(" \n", height-1) + " "
}

// renderScrollbar draws a height-tall, 1-column vertical scroll indicator: a
// bright thumb over a dim track, sized to visible/total and positioned to
// yOffset/(total-visible). Returns a blank column when everything already
// fits (total <= visible), so it disappears rather than clutter the pane.
func renderScrollbar(height, total, visible, yOffset int) string {
	if height < 1 {
		return ""
	}
	if total <= visible {
		return blankColumn(height)
	}

	thumbSize := min(height, max(1, height*visible/total))
	thumbStart := 0
	if maxOffset := total - visible; maxOffset > 0 {
		thumbStart = yOffset * (height - thumbSize) / maxOffset
	}

	lines := make([]string, height)
	for i := range lines {
		if i >= thumbStart && i < thumbStart+thumbSize {
			lines[i] = scrollbarThumbStyle.Render("█")
		} else {
			lines[i] = scrollbarTrackStyle.Render("│")
		}
	}
	return strings.Join(lines, "\n")
}

// truncateItem truncates a string to fit within maxWidth, accounting for borders and padding
func truncateItem(s string, maxWidth int) string {
	// maxWidth accounts for border(2) + padding(2)
	available := maxWidth - 4
	if available < 3 {
		return s
	}
	runes := []rune(s)
	if len(runes) > available {
		return string(runes[:available-1]) + "…"
	}
	return s
}

// inputPromptWidth returns the width of the rendered "%15s : " label prefix
// used in renderInputPanes, so layoutViewports can size the textinput to
// exactly fill the remaining box width without duplicating the format string.
func inputPromptWidth(label string) int {
	return len(fmt.Sprintf("%15s : ", label))
}

func (m mainModel) renderInputPanes(width, height int, combined []InputItem) string {
	var b strings.Builder
	for i, item := range combined {
		if i > 0 {
			b.WriteString("\n")
		}

		prompt := fmt.Sprintf("%15s : ", item.Label)
		if m.focusPane == focusInputs && m.focusedInputIdx == i {
			prompt = selectedItemStyle.Render(prompt)
		} else {
			prompt = normalItemStyle.Render(prompt)
		}

		b.WriteString(prompt)
		b.WriteString(m.getInput(item).View())
	}

	borderColor := colorLavender
	borderStyle := lipgloss.RoundedBorder()
	if m.focusPane == focusInputs {
		borderColor = colorPeach
		borderStyle = cmdBarBorderThick
	}
	if m.validationFlash && (m.hasEmptyRequiredFlagInput() || m.hasEmptyRequiredArg()) {
		borderColor = colorRed
		borderStyle = cmdBarBorderThick
	}
	style := lipgloss.NewStyle().BorderStyle(borderStyle).BorderForeground(borderColor).PaddingLeft(1).PaddingRight(1).Width(width).Height(height)

	return style.Render(b.String())
}
