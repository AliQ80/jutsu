package main

import (
	"fmt"
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
		leftWidth, rightWidth := m.getLayoutWidths()
		height := m.height - 6 // 5 cmdBar + 1 helpBar
		if height < 6 {
			height = 6
		}
		docsPane := m.renderDescriptionBar(leftWidth, height)
		rightPane := m.renderRightPane(rightWidth, height)
		topSection := strings.TrimRight(lipgloss.JoinHorizontal(lipgloss.Top, docsPane, rightPane), "\n")
		cmdBar := m.renderCommandBar(m.width)
		helpBar := m.renderHelpBar(m.width)
		content := lipgloss.JoinVertical(lipgloss.Left, topSection, cmdBar, helpBar)
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
			label := cmd.Name
			if hasRequiredArgs(cmd.Args) {
				label += " <…>"
			}
			items = append(items, label)
		}
		selectedIdx = m.cmdIdx
	case focusSubcmds:
		for _, sub := range m.currentSubCommands() {
			label := sub.Name
			if hasRequiredArgs(sub.Args) {
				label += " <…>"
			}
			items = append(items, label)
		}
		selectedIdx = m.subIdx
	case focusFlags:
		flags := m.currentFlags()
		if len(flags) > 0 {
			for _, f := range flags {
				check := "[ ]"
				if f.Mandatory {
					check = "[*]" // locked-on, cannot be deselected
				} else if f.Selected {
					check = "[x]"
				} else if isConflicted(f, flags) {
					check = "[-]" // blocked by a conflicting selection
				}
				items = append(items, fmt.Sprintf("%s %s", check, f.Name))
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
							} else {
								content.WriteString(flagUnselectedStyle.Render(truncateItem(items[itemIdx], width)))
							}
						case isConflicted(f, flags):
							content.WriteString(flagConflictedFocusedStyle.Render(truncateItem(items[itemIdx], width)))
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

	title := "OUTPUT"
	if m.outputEnlargedActive() {
		title = "OUTPUT (ENLARGED)"
	}
	titleBox := tStyle.Width(width).Align(lipgloss.Center).Render(headerStyle.Render(title))
	contentBox := cStyle.Width(width).Height(height - 3).Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, titleBox, contentBox)
}

func (m mainModel) renderDescriptionBar(width, height int) string {
	style := descriptionBarStyle
	if m.focusPane == focusDocs {
		style = activeDescriptionBarStyle
	}
	// Set dimensions on a local copy so View() renders at the right size.
	m.docs.SetWidth(width - style.GetHorizontalFrameSize())
	m.docs.SetHeight(height - style.GetVerticalFrameSize())
	return style.Width(width).Height(height).Render(m.docs.View())
}

func (m mainModel) renderCommandBar(width int) string {
	// Show the long form (full name) in the bar; short alias is what gets executed.
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
			{"tab", "command"},
			{"q", "quit"},
		}
	case focusDocs:
		entries = []entry{
			{"↑↓", "scroll"},
			{"c", "copy"},
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
			{"enter", "next / command"},
			{"esc", "back"},
			{"tab", "command"},
		}
	default: // composer panes 0-3
		entries = []entry{
			{"↑↓", "navigate"},
			{"←→", "pane"},
			{"d", "docs"},
			{"o/O", "output"},
			{"tab", "command"},
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

	badge := m.renderVersionBadge()
	// width-1 because helpBarStyle has PaddingLeft(1).
	gap := (width - 1) - lipgloss.Width(leftBar) - lipgloss.Width(badge)
	if gap < 1 {
		gap = 1
	}
	return helpBarStyle.Width(width).Render(leftBar + strings.Repeat(" ", gap) + badge)
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
	if m.validationFlash {
		borderColor = colorRed
		borderStyle = cmdBarBorderThick
	}
	style := lipgloss.NewStyle().BorderStyle(borderStyle).BorderForeground(borderColor).PaddingLeft(1).PaddingRight(1).Width(width).Height(height)

	return style.Render(b.String())
}
