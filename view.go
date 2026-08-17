package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

	if m.globalFlagsModalOpen {
		content := m.renderOnBackdrop(m.renderGlobalFlagsModal())
		v := tea.NewView(content)
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	if m.hostKeyModalOpen {
		content := m.renderOnBackdrop(m.renderHostKeyModal())
		v := tea.NewView(content)
		v.AltScreen = true
		return v
	}

	if m.authModalOpen {
		content := m.renderOnBackdrop(m.renderAuthModal())
		v := tea.NewView(content)
		v.AltScreen = true
		return v
	}

	if m.aboutModalOpen {
		content := m.renderOnBackdrop(m.renderAboutModal())
		v := tea.NewView(content)
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	content := m.renderContentFrame()
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// renderContentFrame selects between the output-enlarged, docs-enlarged, and
// ordinary composer+output layouts based on m's enlarged-pane state. It's the
// single source of truth for that selection, used both by the live View()
// and by renderOnBackdrop() when reflowing the frozen pre-modal frame — so a
// modal opened while a pane is enlarged shows that same enlarged layout
// behind it instead of silently falling back to the ordinary frame.
func (m mainModel) renderContentFrame() string {
	if m.outputEnlargedActive() {
		height := m.height - 6 // 5 cmdBar + 1 helpBar
		rightPane := m.renderRightPane(m.width, height)
		cmdBar := m.renderCommandBar(m.width)
		helpBar := m.renderHelpBar(m.width)
		return lipgloss.JoinVertical(lipgloss.Left, rightPane, cmdBar, helpBar)
	}

	if m.docsEnlargedActive() {
		height := m.height - 6 // 5 cmdBar + 1 helpBar
		if height < 6 {
			height = 6
		}
		docsPane := m.renderDescriptionBar(m.width, height)
		cmdBar := m.renderCommandBar(m.width)
		helpBar := m.renderHelpBar(m.width)
		return lipgloss.JoinVertical(lipgloss.Left, docsPane, cmdBar, helpBar)
	}

	return m.renderMainFrame()
}

// renderMainFrame renders the ordinary (non-enlarged, non-modal) composer +
// output + command bar + help bar frame. It's one of renderContentFrame's
// layout choices, used for both the live view and the modal backdrop.
func (m mainModel) renderMainFrame() string {
	leftWidth, rightWidth := m.getLayoutWidths()

	combined := m.getActiveCombined()
	composerH, inputsTop, outputHeight := m.paneRows()
	descBarH := inputsTop - composerH
	actualInputsHeight := outputHeight - inputsTop

	composerSection := m.renderLeftPane(composerH)
	leftColumn := composerSection
	if descBarH >= 3 {
		descBar := m.renderDescriptionBar(leftWidth, descBarH)
		leftColumn = lipgloss.JoinVertical(lipgloss.Left, leftColumn, descBar)
	}
	if actualInputsHeight > 0 {
		inputsPane := m.renderInputPanes(leftWidth, actualInputsHeight, combined)
		leftColumn = lipgloss.JoinVertical(lipgloss.Left, leftColumn, inputsPane)
	}

	rightPane := m.renderRightPane(rightWidth, outputHeight)
	topSection := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightPane)

	cmdBar := m.renderCommandBar(m.width)
	helpBar := m.renderHelpBar(m.width)

	topSection = strings.TrimRight(topSection, "\n")
	return lipgloss.JoinVertical(lipgloss.Left, topSection, cmdBar, helpBar)
}

// renderOnBackdrop centers modal on top of the frozen pre-modal app frame
// (m.backdrop), reflowed at the current terminal size. Falling back to a live
// render if no snapshot was captured keeps this safe to call unconditionally.
func (m mainModel) renderOnBackdrop(modal string) string {
	frame := m
	if m.backdrop != nil {
		frame = *m.backdrop
		frame.width, frame.height = m.width, m.height
		frame.layoutViewports()
	}
	bg := dimBackdrop(frame.renderContentFrame())

	modalW, modalH := lipgloss.Width(modal), lipgloss.Height(modal)
	x := (m.width - modalW) / 2
	y := (m.height - modalH) / 2
	return overlay(bg, modal, x, y)
}

// dimBackdrop preserves the app's existing ANSI styling while muting its
// truecolor foreground/background values toward the base background. This
// keeps the backdrop recognizable without allowing its colors to compete with
// the modal. Non-truecolor sequences are left intact for terminal fallback.
func dimBackdrop(s string) string {
	var out strings.Builder
	foregroundSet := false
	for i := 0; i < len(s); {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			end := i + 2
			for end < len(s) && (s[end] < '@' || s[end] > '~') {
				end++
			}
			if end < len(s) {
				seq := s[i : end+1]
				if s[end] == 'm' {
					seq = dimSGR(seq)
					foregroundSet = updateForegroundState(seq, foregroundSet)
				}
				out.WriteString(seq)
				i = end + 1
				continue
			}
		}

		start := i
		for i < len(s) && s[i] != '\x1b' {
			i++
		}
		if start == i {
			out.WriteByte(s[i])
			i++
			continue
		}
		text := s[start:i]
		if foregroundSet {
			out.WriteString(text)
		} else {
			out.WriteString("\x1b[38;2;108;112;134m")
			out.WriteString(text)
			out.WriteString("\x1b[39m")
		}
	}
	return out.String()
}

func updateForegroundState(seq string, foregroundSet bool) bool {
	parts := strings.Split(seq[2:len(seq)-1], ";")
	for _, part := range parts {
		if part == "" {
			foregroundSet = false
			continue
		}
		code, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		switch {
		case code == 0 || code == 39:
			foregroundSet = false
		case code == 38 || code >= 30 && code <= 37 || code >= 90 && code <= 97:
			foregroundSet = true
		}
	}
	return foregroundSet
}

func dimSGR(seq string) string {
	parts := strings.Split(seq[2:len(seq)-1], ";")
	for i := 0; i+4 < len(parts); i++ {
		if parts[i] != "38" && parts[i] != "48" || parts[i+1] != "2" {
			continue
		}
		r, errR := strconv.Atoi(parts[i+2])
		g, errG := strconv.Atoi(parts[i+3])
		b, errB := strconv.Atoi(parts[i+4])
		if errR != nil || errG != nil || errB != nil {
			continue
		}
		parts[i+2] = strconv.Itoa(dimColorChannel(r))
		parts[i+3] = strconv.Itoa(dimColorChannel(g))
		parts[i+4] = strconv.Itoa(dimColorChannel(b))
		i += 4
	}
	return "\x1b[" + strings.Join(parts, ";") + "m"
}

func dimColorChannel(channel int) int {
	const dimFactor = 0.45
	const base = 30
	return int(float64(base) + float64(channel-base)*dimFactor)
}

// overlay composites fg on top of bg at the given top-left cell coordinates.
// It's ANSI-aware: each fg line replaces only the columns of the matching bg
// line that it covers, leaving the rest of that line's content and styling
// untouched, so the frozen backdrop shows through around the modal box.
//
// x/y may be negative and fg may be larger than bg in either dimension (an
// oversized modal, or one nudged/measured past the terminal edge) — every
// case is clipped to bg's actual bounds rather than assumed in-range, so a
// caller's arithmetic mistake degrades to a clipped modal instead of a
// panic or corrupted line.
func overlay(bg, fg string, x, y int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	for i, fgLine := range fgLines {
		row := y + i
		if row < 0 || row >= len(bgLines) {
			continue // this fg row falls above or below the visible bg entirely
		}

		fgW := lipgloss.Width(fgLine)
		col := x

		// Clip a line that starts left of column 0 (negative x, or a modal
		// wider than the terminal centered with a negative offset).
		if col < 0 {
			if -col >= fgW {
				continue // fully off the left edge
			}
			fgLine = ansi.Cut(fgLine, -col, fgW)
			fgW = lipgloss.Width(fgLine)
			col = 0
		}

		bgLine := bgLines[row]
		bgW := lipgloss.Width(bgLine)
		if col >= bgW {
			continue // starts at/past the right edge, nothing to composite
		}

		// Clip the right edge for an oversized modal or one that runs past
		// bg's width. ansi.Cut is grapheme/wide-character aware, so this
		// never splits a double-width cell (e.g. wide Unicode) in half.
		if col+fgW > bgW {
			fgLine = ansi.Cut(fgLine, 0, bgW-col)
			fgW = bgW - col
		}

		left := ansi.Cut(bgLine, 0, col)
		right := ansi.Cut(bgLine, col+fgW, bgW)
		bgLines[row] = left + fgLine + right
	}
	return strings.Join(bgLines, "\n")
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

	// Rows the pinned global-flag header occupies in the FLAGS pane (0 elsewhere).
	stickyN := 0
	if paneIdx == focusFlags {
		stickyN = m.stickyGlobalCount()
	}

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
		// Selected global options render as pinned display-only rows above
		// the command's own flags; the cursor never lands on them. When more
		// are selected than stickyN allows, the last pinned row summarizes
		// the remainder so none silently vanish.
		var globalNames []string
		for _, gf := range m.globalFlags {
			if gf.Selected {
				globalNames = append(globalNames, gf.Name)
			}
		}
		if stickyN >= len(globalNames) {
			for _, n := range globalNames {
				items = append(items, "◆ "+n)
			}
		} else if stickyN > 0 {
			for _, n := range globalNames[:stickyN-1] {
				items = append(items, "◆ "+n)
			}
			items = append(items, fmt.Sprintf("◆ +%d more", len(globalNames)-(stickyN-1)))
		}
		flags := m.currentFlags()
		// No cursor row unless the command has flags of its own — otherwise
		// the default selectedIdx (0) collides with the first pinned row.
		selectedIdx = -1
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
			selectedIdx = stickyN + m.flagIdx
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
		// FLAGS pane: the first stickyN screen rows are the pinned global
		// header — always shown, never scrolled — so they map to items[0:stickyN]
		// regardless of flagScroll. Rows below scroll the command flags as usual.
		if i < stickyN {
			itemIdx = i
		}
		if itemIdx < len(items) {
			if itemIdx == selectedIdx {
				if paneIdx == focusFlags {
					flags := m.currentFlags()
					if fi := itemIdx - stickyN; fi >= 0 && fi < len(flags) {
						f := flags[fi]
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
				if paneIdx == focusFlags && itemIdx < stickyN {
					// Pinned display-only global row.
					content.WriteString(flagGlobalStyle.Render(truncateItem(items[itemIdx], width)))
				} else if fi := itemIdx - stickyN; paneIdx == focusFlags && fi < len(m.currentFlags()) {
					flags := m.currentFlags()
					if flags[fi].Mandatory {
						content.WriteString(flagMandatoryStyle.Render(truncateItem(items[itemIdx], width)))
					} else if flags[fi].Selected {
						// ponytail: was flagSelectedStyle (colorText) — now activeSelectionStyle (sapphire+bold) for visual consistency with unfocused-pane selections.
						content.WriteString(activeSelectionStyle.Render(truncateItem(items[itemIdx], width)))
					} else if isConflicted(flags[fi], flags) {
						content.WriteString(flagConflictedStyle.Render(truncateItem(items[itemIdx], width)))
					} else if isRequiredGroupFlag(m, flags[fi]) {
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
		outputContent = m.spinner.View() + runningStyle.Render(" Executing command...")
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
	scrollbar := renderScrollbar(contentHeight, total, visible, m.output.YOffset(), scrollbarThumbStyle)
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
	scrollbar := renderScrollbar(docsH, m.docs.TotalLineCount(), m.docs.VisibleLineCount(), m.docs.YOffset(), scrollbarThumbStyle)
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
			{"O", "enlarge"},
			{"c", "copy"},
			{"tab", "finalize"},
			{"esc", "back"},
			{"g", "global"},
			{"?", "about"},
			{"q", "quit"},
		}
	case focusDocs:
		entries = []entry{
			{"D", "enlarge"},
			{"c", "copy"},
			{"tab", "finalize"},
			{"esc", "back"},
			{"g", "global"},
			{"?", "about"},
			{"q", "quit"},
		}
	case focusCmdBar:
		entries = []entry{
			{"enter", "execute"},
			{"c", "copy"},
			{"esc", "back"},
			{"g", "global"},
			{"?", "about"},
			{"q", "quit"},
		}
	case focusInputs:
		entries = []entry{
			{"enter", "next"},
			{"tab", "finalize"},
			{"esc", "back"},
		}
	default: // composer panes 0-3
		entries = []entry{
			{"d/D", "docs"},
			{"o/O", "output"},
			{"tab", "finalize"},
			{"g", "global"},
			{"?", "about"},
			{"q", "quit"},
		}
	}

	// Contextual hints follow the base list after a separator, in the stable
	// order: i inputs, r recall, spc toggle.
	// "i" is a global key (see Update) — advertise it from every pane it works
	// from. getActiveCombined covers positional args as well as command-flag
	// and global-flag inputs, so a command whose only input is an arg still
	// gets the hint.
	if m.focusPane != focusInputs && !m.outputEnlarged && !m.docsEnlarged &&
		len(m.getActiveCombined()) > 0 {
		contextual = append(contextual, entry{"i", "inputs"})
	}
	if m.lastCmd != nil && m.focusPane <= focusFlags {
		contextual = append(contextual, entry{"r", "recall"})
	}
	if m.focusPane == focusFlags {
		contextual = append(contextual, entry{"spc", "toggle"})
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
	// width-2 because helpBarStyle has PaddingLeft(1) and PaddingRight(1).
	gap := (width - 2) - lipgloss.Width(leftBar) - lipgloss.Width(rightBar)
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
// thumb (in the given style — scrollbarThumbStyle normally, an accent color
// to signal focus) over a dim track, sized to visible/total and positioned
// to yOffset/(total-visible). Returns a blank column when everything already
// fits (total <= visible), so it disappears rather than clutter the pane.
func renderScrollbar(height, total, visible, yOffset int, thumb lipgloss.Style) string {
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
			lines[i] = thumb.Render("█")
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
	if m.validationFlash && (m.hasEmptyRequiredFlagInput() || m.hasEmptyRequiredGlobalInput() || m.hasEmptyRequiredArg()) {
		borderColor = colorRed
		borderStyle = cmdBarBorderThick
	}
	style := lipgloss.NewStyle().BorderStyle(borderStyle).BorderForeground(borderColor).PaddingLeft(1).PaddingRight(1).Width(width).Height(height)

	return style.Render(b.String())
}

// Fixed dimensions for the global-options modal: a checklist column sized to
// its longest entry ("[x] --no-integrate-operation" = 28) beside a docs
// viewport. Long descriptions scroll inside the viewport, so the modal's
// footprint never changes with the highlighted flag.
const (
	globalListW = 30
	globalDocsW = 56
	globalDocsH = 18
)

// renderAboutModal renders the read-only popup that shows the running Jutsu
// version (build-stamped via the jutsuVersion ldflag, or resolved at startup
// from the local jj repo when the ldflag is absent — see displayVersion) and
// a link to the published docs. Opened via `?`, closed via `esc` — see
// handleAboutModalKeys.
func (m mainModel) renderAboutModal() string {
	// Title is centered across the modal's inner content width. The docs
	// URL is the widest line (31 chars) — size the title against the actual
	// content lines so a longer URL or version tag can never wrap or clip.
	const docsURL = "https://aliq80.github.io/jutsu/"
	lines := []string{
		"",
		"jutsu     " + m.displayVersion(),
		docsURL,
	}
	innerW := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > innerW {
			innerW = w
		}
	}
	lines[0] = lipgloss.PlaceHorizontal(innerW, lipgloss.Center, headerStyle.Render("Jutsu"))
	return modalBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// displayVersion returns the version string to show in the about modal.
// Release builds have jutsuVersion set to the stamped tag (e.g. "v1.2.3") and
// the latestTag fallback is a no-op. Source builds (`go run .`, `go build`)
// leave jutsuVersion as "dev"; if a runtime fetch of the latest jj tag
// succeeded, that tag overrides "dev" here. Either way the package-level
// jutsuVersion var is never mutated — it's a build-time constant for release
// builds, and the override is a pure render-time substitution.
func (m mainModel) displayVersion() string {
	if jutsuVersion == "dev" && m.latestTag != "" {
		return m.latestTag
	}
	return jutsuVersion
}

// renderHostKeyModal renders the SSH host-key approval popup: the unknown host,
// the fingerprint(s) to verify, and the trust/cancel choices.
func (m mainModel) renderHostKeyModal() string {
	p := m.hostKeyPrompt
	lines := []string{
		headerStyle.Render("🔒  Unknown SSH host"),
		"",
		"Jutsu doesn't recognize " + hostAccentStyle.Render(p.host) + " yet.",
		"Verify this fingerprint, then trust it to continue:",
		"",
	}
	for _, fp := range strings.Split(strings.TrimRight(p.fingerprint, "\n"), "\n") {
		lines = append(lines, fingerprintStyle.Render("  "+fp))
	}
	hint := helpKeyStyle.Render("y") + " " + helpDescStyle.Render("trust") +
		"    " + helpKeyStyle.Render("n") + " " + helpDescStyle.Render("cancel")
	lines = append(lines, "", hint)
	return modalBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// renderAuthModal renders the "remote needs input" popup used for credential
// prompts (or when the host key couldn't be scanned): a guided handoff to the
// real terminal, or cancel.
func (m mainModel) renderAuthModal() string {
	// The hint says only "continue" — where it continues is already in the
	// question above it.
	question := "Jutsu can't collect here. Continue in the terminal?"
	if currentMux() != nil {
		question = "Jutsu can't collect here. Continue in a split pane?"
	}
	hint := helpKeyStyle.Render("enter") + " " + helpDescStyle.Render("continue") +
		"    " + helpKeyStyle.Render("esc") + " " + helpDescStyle.Render("cancel")
	lines := []string{
		headerStyle.Render("🔑  Remote needs input"),
		"",
		"This remote needs a sign-in or key passphrase that",
		question,
		"",
		hint,
	}
	return modalBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// renderGlobalFlagsModal renders the "g"-triggered popup listing jj's global
// options (m.globalFlags): checklist on the left, the highlighted flag's
// full docs in a scrollable viewport on the right. Values for flags that
// need one are edited in the regular input bar after closing, not here.
func (m mainModel) renderGlobalFlagsModal() string {
	var list strings.Builder
	for i, f := range m.globalFlags {
		if i > 0 {
			list.WriteString("\n")
		}
		check := "[ ]"
		if f.Selected {
			check = "[x]"
		} else if isConflicted(f, m.globalFlags) {
			check = "[-]"
		}
		label := fmt.Sprintf("%s %s", check, flagBarForm(f))
		switch {
		case i == m.globalFlagIdx:
			list.WriteString(selectedItemStyle.Render(label))
		case f.Selected:
			list.WriteString(activeSelectionStyle.Render(label))
		case isConflicted(f, m.globalFlags):
			list.WriteString(flagConflictedStyle.Render(label))
		default:
			list.WriteString(flagUnselectedStyle.Render(label))
		}
	}

	// Shrink the docs window on very short terminals (modal chrome around
	// the columns is 6 rows); width — and thus the wrapped content — never
	// changes, and the checklist's 11 rows are the floor.
	docsH := globalDocsH
	if avail := m.height - 6; avail < docsH {
		docsH = max(len(m.globalFlags), avail)
	}
	docs := m.globalDocs
	docs.SetHeight(docsH)
	// A peach scrollbar thumb signals the docs column has ↑↓ focus.
	thumb := scrollbarThumbStyle
	if m.globalModalDocsFocus {
		thumb = lipgloss.NewStyle().Foreground(colorPeach)
	}
	scrollbar := renderScrollbar(docsH, docs.TotalLineCount(), docs.VisibleLineCount(), docs.YOffset(), thumb)
	docsContent := lipgloss.JoinHorizontal(lipgloss.Top, docs.View(), blankColumn(docsH), scrollbar)

	// Each column is a title-boxed pane in the main composer's focus
	// language: the column ↑↓ currently drives gets the thick peach border.
	listT, listC := activeTitleBorderStyle, activeContentBorderStyle
	docsT, docsC := inactiveTitleBorderStyle, inactiveContentBorderStyle
	if m.globalModalDocsFocus {
		listT, listC = inactiveTitleBorderStyle, inactiveContentBorderStyle
		docsT, docsC = activeTitleBorderStyle, activeContentBorderStyle
	}

	// Pad the checklist to the docs height so both panes line up.
	listLines := strings.Split(list.String(), "\n")
	for len(listLines) < docsH {
		listLines = append(listLines, "")
	}

	const listPaneW = globalListW + 4 // content + border(2)/padding(2), like the composer panes
	const docsPaneW = globalDocsW + scrollbarW + scrollbarGapW + 4

	listPane := lipgloss.JoinVertical(lipgloss.Left,
		listT.Width(listPaneW).Align(lipgloss.Center).Render(headerStyle.Render("GLOBAL OPTIONS")),
		listC.Width(listPaneW).Height(docsH+1).Render(strings.Join(listLines, "\n")),
	)
	docsPane := lipgloss.JoinVertical(lipgloss.Left,
		docsT.Width(docsPaneW).Align(lipgloss.Center).Render(headerStyle.Render("DOCS")),
		docsC.Width(docsPaneW).Height(docsH+1).Render(docsContent),
	)

	hint := func(key, desc string) string {
		return helpKeyStyle.Render(key) + " " + helpDescStyle.Render(desc)
	}
	var footer string
	if m.globalModalDocsFocus {
		footer = hint("↑↓", "scroll") + "  " + hint("←", "options") + "  " + hint("esc", "close") + "  " + hint("q", "quit")
	} else {
		footer = hint("↑↓", "navigate") + "  " + hint("→", "docs") + "  " + hint("spc", "toggle") + "  " + hint("esc", "close") + "  " + hint("q", "quit")
	}
	footer = lipgloss.PlaceHorizontal(listPaneW+docsPaneW, lipgloss.Center, footer)

	content := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, listPane, docsPane),
		"",
		footer,
	)
	return modalFrameStyle.Render(content)
}
