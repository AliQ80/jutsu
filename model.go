package main

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type clearValidationFlashMsg struct{}
type clearCopyFlashMsg struct{}

type mainModel struct {
	categories    []Category
	focusPane     int
	lastFocusPane int
	catIdx        int
	cmdIdx        int
	subIdx        int
	flagIdx       int
	catScroll     int
	cmdScroll     int
	subScroll     int
	flagScroll    int
	paneH         int // composer pane height; content = paneH - 4

	output          viewport.Model
	outputLines     []string
	xOffset         int
	outputEnlarged  bool
	docs            viewport.Model
	docsEnlarged    bool
	inputs          map[string]textinput.Model // keyed by flag Name
	argInputs       map[string]textinput.Model // keyed by Arg Name
	focusedInputIdx int

	cmdText         string
	cmdTextLong     string
	running         bool
	lastCmd         *commandSnapshot // nil until first command executes
	validationFlash bool
	copyFlash       bool
	jjVersion       string
	cwd             string

	width  int
	height int
}

// InputItem is a unified entry for the input pane — either a positional arg or a flag input.
type InputItem struct {
	Name  string // lookup key: argInputs[Name] when IsArg, inputs[Name] otherwise
	Label string // display label shown in the input pane
	IsArg bool
}

func (m mainModel) getInput(item InputItem) textinput.Model {
	if item.IsArg {
		return m.argInputs[item.Name]
	}
	return m.inputs[item.Name]
}

func (m mainModel) setInput(item InputItem, ti textinput.Model) {
	if item.IsArg {
		m.argInputs[item.Name] = ti
	} else {
		m.inputs[item.Name] = ti
	}
}

type commandSnapshot struct {
	catIdx        int
	cmdIdx        int
	subIdx        int
	flagIdx       int
	composerPane  int               // focusPane at capture time (lastFocusPane, a composer pane 0–3)
	selectedFlags map[string]bool   // flag Name → Selected
	inputValues   map[string]string // flag Name → text input value
	argValues     map[string]string // arg Name → text input value
}

func newModel() mainModel {
	cats := loadCategories()
	vp := viewport.New()
	vp.SetWidth(80)
	vp.SetHeight(20)
	vp.Style = outputStyle

	dv := viewport.New()
	dv.SetWidth(80)
	dv.SetHeight(10)

	cwd := "?"
	if wd, err := os.Getwd(); err == nil {
		cwd = filepath.Base(wd)
	}

	m := mainModel{
		categories: cats,
		focusPane:  focusCategories,
		output:     vp,
		docs:       dv,
		inputs:     make(map[string]textinput.Model),
		argInputs:  make(map[string]textinput.Model),
		cwd:        cwd,
	}

	// Pre-initialize textinputs for mandatory flags so the input pane renders
	// correctly the moment the user lands on a command that has them.
	for ci := range cats {
		for cmi := range cats[ci].Commands {
			initMandatoryFlagInputs(cats[ci].Commands[cmi].Flags, m.inputs)
			for si := range cats[ci].Commands[cmi].SubCmds {
				initMandatoryFlagInputs(cats[ci].Commands[cmi].SubCmds[si].Flags, m.inputs)
			}
		}
	}

	m.cmdText, m.cmdTextLong = m.buildCommandStrings()
	m.refreshDocs()
	return m
}

func (m mainModel) Init() tea.Cmd {
	return fetchJJVersion()
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layoutViewports()
		return m, nil

	case execResultMsg:
		m.running = false
		var lines []string

		cmdParts := strings.SplitN(msg.cmdStr, " ", 2)
		cmdLine := promptStyle.Render("❯") + " " + promptCmdStyle.Render(cmdParts[0])
		if len(cmdParts) > 1 {
			cmdLine += " " + cmdParts[1]
		}
		lines = append(lines, cmdLine)
		lines = append(lines, "")

		if msg.err != nil {
			// Always show errors
			errorMsg := msg.err.Error()
			if msg.output != "" {
				errorMsg = msg.output + "\n" + errorMsg
			}
			lines = append(lines, strings.Split(strings.TrimRight(errorMsg, "\n"), "\n")...)
		} else if strings.TrimSpace(msg.output) != "" {
			lines = append(lines, strings.Split(strings.TrimRight(msg.output, "\n"), "\n")...)
		} else {
			// Command succeeded with no output
			lines = append(lines, "✓ Command completed successfully (no output)")
		}

		m.outputLines = lines
		m.xOffset = 0
		m.applyXOffset()
		m.output.GotoTop()
		m.resetCurrentFlags()
		return m, nil

	case jjVersionMsg:
		m.jjVersion = string(msg)
		return m, nil

	case clearValidationFlashMsg:
		m.validationFlash = false
		return m, nil

	case copyDoneMsg:
		m.copyFlash = true
		return m, copyFlashTimer()

	case clearCopyFlashMsg:
		m.copyFlash = false
		return m, nil

	case tea.MouseWheelMsg:
		if m.outputEnlargedActive() {
			height := m.height - 6 // 5 cmdBar + 1 helpBar
			if msg.Y < height {
				if msg.Button == tea.MouseWheelUp && m.output.YOffset() > 0 {
					m.output.SetYOffset(m.output.YOffset() - 1)
				} else if msg.Button == tea.MouseWheelDown {
					m.output.SetYOffset(m.output.YOffset() + 1)
				}
			}
			return m, nil
		}

		if m.docsEnlargedActive() {
			height := m.height - 6 // 5 cmdBar + 1 helpBar
			if msg.Y < height {
				if msg.Button == tea.MouseWheelUp && m.docs.YOffset() > 0 {
					m.docs.SetYOffset(m.docs.YOffset() - 1)
				} else if msg.Button == tea.MouseWheelDown {
					m.docs.SetYOffset(m.docs.YOffset() + 1)
				}
			}
			return m, nil
		}

		leftWidth, _ := m.getLayoutWidths()
		activeInputs := m.getActiveInputs()
		inputsHeight := 0
		if len(activeInputs) > 0 {
			inputsHeight = len(activeInputs) + 2
		}
		topHeight := m.height - 6 - inputsHeight
		composerH := topHeight / 2

		if msg.X >= leftWidth && msg.Y < topHeight {
			if msg.Button == tea.MouseWheelUp && m.output.YOffset() > 0 {
				m.output.SetYOffset(m.output.YOffset() - 1)
			} else if msg.Button == tea.MouseWheelDown {
				m.output.SetYOffset(m.output.YOffset() + 1)
			}
		} else if msg.X < leftWidth && msg.Y >= composerH && msg.Y < topHeight {
			if msg.Button == tea.MouseWheelUp && m.docs.YOffset() > 0 {
				m.docs.SetYOffset(m.docs.YOffset() - 1)
			} else if msg.Button == tea.MouseWheelDown {
				m.docs.SetYOffset(m.docs.YOffset() + 1)
			}
		}
		return m, nil

	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		if m.outputEnlargedActive() {
			height := m.height - 6 // 5 cmdBar + 1 helpBar
			if msg.Y >= height && msg.Y < height+5 {
				m.outputEnlarged = false
				m.layoutViewports()
				if m.hasIncompleteInputs() {
					m.validationFlash = true
					return m, flashTimer()
				}
				m.focusPane = focusCmdBar
				m.cmdText, m.cmdTextLong = m.buildCommandStrings()
			}
			return m, nil
		}

		if m.docsEnlargedActive() {
			height := m.height - 6 // 5 cmdBar + 1 helpBar
			if msg.Y >= height && msg.Y < height+5 {
				m.docsEnlarged = false
				m.layoutViewports()
				m.refreshDocs()
				if m.hasIncompleteInputs() {
					m.validationFlash = true
					return m, flashTimer()
				}
				m.focusPane = focusCmdBar
				m.cmdText, m.cmdTextLong = m.buildCommandStrings()
			}
			return m, nil
		}

		leftWidth, _ := m.getLayoutWidths()
		activeInputs := m.getActiveInputs()
		inputsHeight := 0
		if len(activeInputs) > 0 {
			inputsHeight = len(activeInputs) + 2
		}
		topHeight := m.height - 6 - inputsHeight
		composerH := topHeight / 2

		switch {
		case msg.X >= leftWidth && msg.Y < topHeight:
			if m.focusPane <= focusFlags {
				m.lastFocusPane = m.focusPane
			}
			m.focusPane = focusOutput
		case msg.X < leftWidth && msg.Y >= composerH && msg.Y < topHeight:
			if m.focusPane <= focusFlags {
				m.lastFocusPane = m.focusPane
			}
			m.focusPane = focusDocs
		case msg.Y >= topHeight+inputsHeight && msg.Y < topHeight+inputsHeight+5:
			if m.hasIncompleteInputs() {
				m.validationFlash = true
				return m, flashTimer()
			}
			if m.focusPane <= focusFlags {
				m.lastFocusPane = m.focusPane
			}
			m.focusPane = focusCmdBar
			m.cmdText, m.cmdTextLong = m.buildCommandStrings()
		case msg.X < leftWidth && msg.Y < composerH:
			switch {
			case msg.X < catPaneW:
				m.focusPane = focusCategories
			case msg.X < catPaneW+cmdPaneW:
				m.focusPane = focusCommands
			case msg.X < catPaneW+cmdPaneW+subPaneW:
				if len(m.currentSubCommands()) > 0 {
					m.focusPane = focusSubcmds
				}
			default:
				if len(m.currentFlags()) > 0 {
					m.focusPane = focusFlags
				}
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.focusPane != focusInputs {
				return m, tea.Quit
			}
		case "o":
			if m.focusPane == focusOutput {
				return m.exitOutput()
			}
			if m.focusPane != focusInputs && !m.docsEnlarged {
				if m.focusPane <= focusFlags {
					m.lastFocusPane = m.focusPane
				}
				m.focusPane = focusOutput
				return m, nil
			}
		case "O":
			if m.outputEnlargedActive() {
				return m.exitOutput()
			}
			if m.focusPane != focusInputs && !m.docsEnlarged {
				if m.focusPane <= focusFlags {
					m.lastFocusPane = m.focusPane
				}
				m.focusPane = focusOutput
				m.outputEnlarged = true
				m.layoutViewports()
				return m, nil
			}
		case "d":
			if m.focusPane == focusDocs {
				return m.exitDocs()
			}
			if !m.outputEnlarged && m.focusPane != focusInputs {
				if m.focusPane <= focusFlags {
					m.lastFocusPane = m.focusPane
				}
				m.focusPane = focusDocs
				return m, nil
			}
		case "D":
			if m.docsEnlargedActive() {
				return m.exitDocs()
			}
			if m.focusPane != focusInputs && !m.outputEnlarged {
				if m.focusPane <= focusFlags {
					m.lastFocusPane = m.focusPane
				}
				m.docsEnlarged = true
				m.layoutViewports()
				m.refreshDocs()
				m.focusPane = focusDocs
				return m, nil
			}
		}

		switch m.focusPane {
		case focusCmdBar:
			return m.handleCmdBarKeys(msg)
		case focusOutput:
			return m.handleOutputKeys(msg)
		case focusInputs:
			return m.handleInputKeys(msg)
		case focusDocs:
			return m.handleDocsKeys(msg)
		default:
			return m.handleComposerKeys(msg)
		}

	default:
		// Route internal textinput messages (cursor blink, pasteMsg from Ctrl+V, etc.)
		// back to the focused input so the textinput can handle its own async commands.
		if m.focusPane == focusInputs {
			active := m.getActiveCombined()
			if len(active) > 0 {
				if m.focusedInputIdx >= len(active) {
					m.focusedInputIdx = len(active) - 1
				}
				item := active[m.focusedInputIdx]
				ti := m.getInput(item)
				var cmd tea.Cmd
				ti, cmd = ti.Update(msg)
				m.setInput(item, ti)
				m.cmdText, m.cmdTextLong = m.buildCommandStrings()
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m mainModel) handleComposerKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if m.hasIncompleteInputs() {
			m.validationFlash = true
			return m, flashTimer()
		}
		m.lastFocusPane = m.focusPane
		m.focusPane = focusCmdBar
		m.cmdText, m.cmdTextLong = m.buildCommandStrings()
		return m, nil

	case "right", "l":
		if m.focusPane < focusFlags {
			next := m.focusPane + 1
			if next == focusSubcmds && len(m.currentSubCommands()) == 0 {
				next++
			}
			if next == focusFlags && len(m.currentFlags()) == 0 {
				next = m.focusPane // nowhere to go, stay put
			}
			m.focusPane = next
			m.clampIndices()
		}
		m.cmdText, m.cmdTextLong = m.buildCommandStrings()
		m.layoutViewports()
		m.refreshDocs()
		return m, nil

	case "left", "h":
		if m.focusPane > focusCategories {
			m.focusPane--
			if m.focusPane == focusSubcmds && len(m.currentSubCommands()) == 0 {
				m.focusPane--
			}
			m.clampIndices()
		}
		m.cmdText, m.cmdTextLong = m.buildCommandStrings()
		m.layoutViewports()
		m.refreshDocs()
		return m, nil

	case "down", "j":
		m.navigateDown()
		m.clampFocusPane()
		m.cmdText, m.cmdTextLong = m.buildCommandStrings()
		m.layoutViewports()
		m.refreshDocs()
		return m, nil

	case "up", "k":
		m.navigateUp()
		m.clampFocusPane()
		m.cmdText, m.cmdTextLong = m.buildCommandStrings()
		m.layoutViewports()
		m.refreshDocs()
		return m, nil

	case "enter":
		// From commands or subcommands pane: open input pane if there are required args or flag inputs.
		if m.focusPane == focusCommands || m.focusPane == focusSubcmds {
			combined := m.getActiveCombined()
			if len(combined) > 0 {
				m.lastFocusPane = m.focusPane
				m.focusPane = focusInputs
				m.focusedInputIdx = 0
				item := combined[0]
				ti := m.getInput(item)
				cmd := ti.Focus()
				m.setInput(item, ti)
				return m, cmd
			}
		}
		if m.focusPane == focusFlags {
			combined := m.getActiveCombined()
			if len(combined) > 0 {
				m.lastFocusPane = m.focusPane
				m.focusPane = focusInputs
				m.focusedInputIdx = 0
				item := combined[0]
				ti := m.getInput(item)
				cmd := ti.Focus()
				m.setInput(item, ti)
				return m, cmd
			}
		}
		return m, nil

	case "space":
		if m.focusPane == focusFlags {
			m.toggleFlag()
		}
		m.cmdText, m.cmdTextLong = m.buildCommandStrings()
		m.layoutViewports()
		m.reclampAllScrolls()
		m.refreshDocs()
		return m, nil

	case "r":
		m.restoreLastCmd()
		return m, nil
	}

	return m, nil
}

func (m mainModel) handleCmdBarKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.cmdText != "" && !m.running {
			if m.hasIncompleteInputs() {
				m.validationFlash = true
				return m, flashTimer()
			}
			m.lastCmd = m.captureSnapshot()
			m.running = true
			m.focusPane = m.lastFocusPane
			return m, executeCommand(m.cmdText)
		}
	case "c":
		return m, copyToClipboard(m.cmdTextLong)
	case "esc":
		m.focusPane = m.lastFocusPane
		return m, nil
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m mainModel) handleOutputKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "down", "j":
		m.output.SetYOffset(m.output.YOffset() + 1)
	case "up", "k":
		if m.output.YOffset() > 0 {
			m.output.SetYOffset(m.output.YOffset() - 1)
		}
	case "pgdown":
		m.output.SetYOffset(m.output.YOffset() + m.output.Height()/2)
	case "pgup":
		newOffset := m.output.YOffset() - m.output.Height()/2
		if newOffset < 0 {
			newOffset = 0
		}
		m.output.SetYOffset(newOffset)
	case "right", "l":
		m.xOffset += 4
		m.applyXOffset()
	case "left", "h":
		if m.xOffset > 0 {
			m.xOffset -= 4
			if m.xOffset < 0 {
				m.xOffset = 0
			}
			m.applyXOffset()
		}
	case "c":
		text := stripANSI(strings.Join(m.outputLines, "\n"))
		return m, copyToClipboard(text)
	case "tab":
		m.focusPane = focusCmdBar
		m.outputEnlarged = false
	case "esc":
		m.focusPane = m.lastFocusPane
		m.outputEnlarged = false
	}
	m.layoutViewports()
	return m, nil
}

func (m *mainModel) applyXOffset() {
	shifted := make([]string, len(m.outputLines))
	for i, line := range m.outputLines {
		shifted[i] = ansiSkip(line, m.xOffset)
	}
	m.output.SetContentLines(shifted)
}

// ansiSkip skips n visible characters in s while preserving ANSI escape sequences intact.
func ansiSkip(s string, skip int) string {
	var out strings.Builder
	runes := []rune(s)
	i, skipped := 0, 0
	for i < len(runes) {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Scan to end of CSI sequence (ends at a letter a-zA-Z)
			j := i + 2
			for j < len(runes) && !((runes[j] >= 'A' && runes[j] <= 'Z') || (runes[j] >= 'a' && runes[j] <= 'z')) {
				j++
			}
			if j < len(runes) {
				j++
			}
			out.WriteString(string(runes[i:j]))
			i = j
		} else {
			if skipped >= skip {
				out.WriteRune(runes[i])
			} else {
				skipped++
			}
			i++
		}
	}
	return out.String()
}

// clampScroll advances the scroll offset only when the cursor hits a viewport edge.
func clampScroll(scroll, sel, contentH int) int {
	if contentH <= 0 {
		return 0
	}
	if sel >= scroll+contentH {
		return sel - contentH + 1
	}
	if sel < scroll {
		return sel
	}
	return scroll
}

// reclampAllScrolls re-clamps every pane's scroll offset after paneH changes
// (e.g. when the input bar appears or disappears and shrinks the composer panes).
func (m *mainModel) reclampAllScrolls() {
	ch := m.paneH - 4
	m.catScroll = clampScroll(m.catScroll, m.catIdx, ch)
	m.cmdScroll = clampScroll(m.cmdScroll, m.cmdIdx, ch)
	m.subScroll = clampScroll(m.subScroll, m.subIdx, ch)

	// Clamp to flagIdx+1 so the item below the cursor stays visible
	// (cursor sits one row above the bottom edge when a next item exists).
	flags := m.currentFlags()
	peekIdx := m.flagIdx
	if peekIdx < len(flags)-1 {
		peekIdx++
	}
	m.flagScroll = clampScroll(m.flagScroll, peekIdx, ch)

	// Cap scroll so the list stays flush to the bottom — prevents empty
	// lines when the pane grows after untoggling a RequiresInput flag.
	if ch > 0 {
		if maxScroll := max(0, len(flags)-ch); m.flagScroll > maxScroll {
			m.flagScroll = maxScroll
		}
	}
}

func (m *mainModel) navigateDown() {
	ch := m.paneH - 4
	switch m.focusPane {
	case focusCategories:
		m.resetCurrentFlags()
		m.catIdx = (m.catIdx + 1) % len(m.categories)
		m.cmdIdx, m.subIdx, m.flagIdx = 0, 0, 0
		m.cmdScroll, m.subScroll, m.flagScroll = 0, 0, 0
		m.catScroll = clampScroll(m.catScroll, m.catIdx, ch)
	case focusCommands:
		cmds := m.currentCommands()
		m.resetCurrentFlags()
		m.cmdIdx = (m.cmdIdx + 1) % len(cmds)
		m.subIdx, m.flagIdx = 0, 0
		m.subScroll, m.flagScroll = 0, 0
		m.cmdScroll = clampScroll(m.cmdScroll, m.cmdIdx, ch)
	case focusSubcmds:
		subs := m.currentSubCommands()
		m.resetCurrentFlags()
		m.subIdx = (m.subIdx + 1) % len(subs)
		m.flagIdx = 0
		m.flagScroll = 0
		m.subScroll = clampScroll(m.subScroll, m.subIdx, ch)
	case focusFlags:
		flags := m.currentFlags()
		m.flagIdx = (m.flagIdx + 1) % len(flags)
		m.flagScroll = clampScroll(m.flagScroll, m.flagIdx, ch)
	}
}

func (m *mainModel) navigateUp() {
	ch := m.paneH - 4
	switch m.focusPane {
	case focusCategories:
		m.resetCurrentFlags()
		m.catIdx = (m.catIdx - 1 + len(m.categories)) % len(m.categories)
		m.cmdIdx, m.subIdx, m.flagIdx = 0, 0, 0
		m.cmdScroll, m.subScroll, m.flagScroll = 0, 0, 0
		m.catScroll = clampScroll(m.catScroll, m.catIdx, ch)
	case focusCommands:
		cmds := m.currentCommands()
		m.resetCurrentFlags()
		m.cmdIdx = (m.cmdIdx - 1 + len(cmds)) % len(cmds)
		m.subIdx, m.flagIdx = 0, 0
		m.subScroll, m.flagScroll = 0, 0
		m.cmdScroll = clampScroll(m.cmdScroll, m.cmdIdx, ch)
	case focusSubcmds:
		subs := m.currentSubCommands()
		m.resetCurrentFlags()
		m.subIdx = (m.subIdx - 1 + len(subs)) % len(subs)
		m.flagIdx = 0
		m.flagScroll = 0
		m.subScroll = clampScroll(m.subScroll, m.subIdx, ch)
	case focusFlags:
		flags := m.currentFlags()
		m.flagIdx = (m.flagIdx - 1 + len(flags)) % len(flags)
		m.flagScroll = clampScroll(m.flagScroll, m.flagIdx, ch)
	}
}

func (m *mainModel) resetCurrentFlags() {
	if len(m.categories) == 0 || m.catIdx >= len(m.categories) {
		return
	}
	if m.cmdIdx >= len(m.categories[m.catIdx].Commands) {
		return
	}
	cmd := &m.categories[m.catIdx].Commands[m.cmdIdx]

	reset := func(flags []Flag) {
		for i := range flags {
			if !flags[i].Mandatory {
				flags[i].Selected = false
			}
			if flags[i].RequiresInput {
				if ti, ok := m.inputs[flags[i].Name]; ok {
					ti.SetValue("")
					m.inputs[flags[i].Name] = ti
				}
			}
		}
	}

	if len(cmd.SubCmds) > 0 {
		if m.subIdx < len(cmd.SubCmds) {
			reset(cmd.SubCmds[m.subIdx].Flags)
			for _, a := range cmd.SubCmds[m.subIdx].Args {
				if ti, ok := m.argInputs[a.Name]; ok {
					ti.SetValue("")
					m.argInputs[a.Name] = ti
				}
			}
		}
	} else {
		reset(cmd.Flags)
		for _, a := range cmd.Args {
			if ti, ok := m.argInputs[a.Name]; ok {
				ti.SetValue("")
				m.argInputs[a.Name] = ti
			}
		}
	}
}

func (m mainModel) captureSnapshot() *commandSnapshot {
	snap := &commandSnapshot{
		catIdx:        m.catIdx,
		cmdIdx:        m.cmdIdx,
		subIdx:        m.subIdx,
		flagIdx:       m.flagIdx,
		composerPane:  m.lastFocusPane,
		selectedFlags: make(map[string]bool),
		inputValues:   make(map[string]string),
		argValues:     make(map[string]string),
	}
	for _, f := range m.currentFlags() {
		if f.Selected {
			snap.selectedFlags[f.Name] = true
		}
		if f.RequiresInput {
			if ti, ok := m.inputs[f.Name]; ok {
				snap.inputValues[f.Name] = ti.Value()
			}
		}
	}
	for _, a := range m.currentArgs() {
		if ti, ok := m.argInputs[a.Name]; ok {
			snap.argValues[a.Name] = ti.Value()
		}
	}
	return snap
}

func (m *mainModel) restoreLastCmd() {
	if m.lastCmd == nil {
		return
	}
	snap := m.lastCmd
	m.catIdx = snap.catIdx
	m.cmdIdx = snap.cmdIdx
	m.subIdx = snap.subIdx

	if len(m.categories) == 0 || m.catIdx >= len(m.categories) {
		return
	}
	cat := &m.categories[m.catIdx]
	if len(cat.Commands) == 0 || m.cmdIdx >= len(cat.Commands) {
		return
	}

	// Must mutate through the categories slice — currentFlags() returns value copies.
	cmd := &cat.Commands[m.cmdIdx]
	var flags *[]Flag
	if len(cmd.SubCmds) > 0 && snap.subIdx < len(cmd.SubCmds) {
		flags = &cmd.SubCmds[snap.subIdx].Flags
	} else {
		flags = &cmd.Flags
	}
	for i := range *flags {
		if !(*flags)[i].Mandatory {
			(*flags)[i].Selected = snap.selectedFlags[(*flags)[i].Name]
		}
	}

	for name, val := range snap.inputValues {
		if ti, ok := m.inputs[name]; ok {
			ti.SetValue(val)
			m.inputs[name] = ti
		}
	}
	for name, val := range snap.argValues {
		if ti, ok := m.argInputs[name]; ok {
			ti.SetValue(val)
			m.argInputs[name] = ti
		}
	}

	m.focusPane = snap.composerPane
	m.flagIdx = snap.flagIdx
	m.clampIndices()
	m.layoutViewports()
	m.reclampAllScrolls()
	m.refreshDocs()
	m.cmdText, m.cmdTextLong = m.buildCommandStrings()
}

func (m *mainModel) toggleFlag() {
	// Validate indices
	if len(m.categories) == 0 || m.catIdx >= len(m.categories) {
		return
	}

	// Get pointer to current category
	cat := &m.categories[m.catIdx]
	if len(cat.Commands) == 0 || m.cmdIdx >= len(cat.Commands) {
		return
	}

	// Get pointer to current command
	cmd := &cat.Commands[m.cmdIdx]

	// Handle subcommands vs direct flags
	if len(cmd.SubCmds) > 0 {
		if m.subIdx >= len(cmd.SubCmds) {
			return
		}
		subCmd := &cmd.SubCmds[m.subIdx]
		if len(subCmd.Flags) == 0 || m.flagIdx >= len(subCmd.Flags) {
			return
		}
		if subCmd.Flags[m.flagIdx].Mandatory {
			return
		}
		subCmd.Flags[m.flagIdx].Selected = !subCmd.Flags[m.flagIdx].Selected
		if subCmd.Flags[m.flagIdx].Selected {
			if subCmd.Flags[m.flagIdx].RequiresInput {
				key := subCmd.Flags[m.flagIdx].Name
				if _, exists := m.inputs[key]; !exists {
					ti := textinput.New()
					ti.Prompt = " "
					ti.Placeholder = "Enter " + key + "..."
					ti.CharLimit = 256
					m.inputs[key] = ti
				}
			}
			for i := range subCmd.Flags {
				if i != m.flagIdx && flagsConflict(subCmd.Flags[m.flagIdx], subCmd.Flags[i]) {
					subCmd.Flags[i].Selected = false
					if subCmd.Flags[i].RequiresInput {
						if ti, ok := m.inputs[subCmd.Flags[i].Name]; ok {
							ti.SetValue("")
							m.inputs[subCmd.Flags[i].Name] = ti
						}
					}
				}
			}
		}
	} else {
		if len(cmd.Flags) == 0 || m.flagIdx >= len(cmd.Flags) {
			return
		}
		if cmd.Flags[m.flagIdx].Mandatory {
			return
		}
		cmd.Flags[m.flagIdx].Selected = !cmd.Flags[m.flagIdx].Selected
		if cmd.Flags[m.flagIdx].Selected {
			if cmd.Flags[m.flagIdx].RequiresInput {
				key := cmd.Flags[m.flagIdx].Name
				if _, exists := m.inputs[key]; !exists {
					ti := textinput.New()
					ti.Prompt = " "
					ti.Placeholder = "Enter " + key + "..."
					ti.CharLimit = 256
					m.inputs[key] = ti
				}
			}
			for i := range cmd.Flags {
				if i != m.flagIdx && flagsConflict(cmd.Flags[m.flagIdx], cmd.Flags[i]) {
					cmd.Flags[i].Selected = false
					if cmd.Flags[i].RequiresInput {
						if ti, ok := m.inputs[cmd.Flags[i].Name]; ok {
							ti.SetValue("")
							m.inputs[cmd.Flags[i].Name] = ti
						}
					}
				}
			}
		}
	}
}

func (m *mainModel) clampIndices() {
	ch := m.paneH - 4
	if m.catIdx >= len(m.categories) {
		m.catIdx = len(m.categories) - 1
		m.catScroll = clampScroll(m.catScroll, m.catIdx, ch)
	}
	cmds := m.currentCommands()
	if m.cmdIdx >= len(cmds) {
		m.cmdIdx = max(0, len(cmds)-1)
		m.cmdScroll = clampScroll(m.cmdScroll, m.cmdIdx, ch)
	}
	subs := m.currentSubCommands()
	if m.subIdx >= len(subs) {
		m.subIdx = max(0, len(subs)-1)
		m.subScroll = clampScroll(m.subScroll, m.subIdx, ch)
	}
	flags := m.currentFlags()
	if m.flagIdx >= len(flags) {
		m.flagIdx = max(0, len(flags)-1)
		m.flagScroll = clampScroll(m.flagScroll, m.flagIdx, ch)
	}
}

func (m mainModel) currentCommands() []Command {
	if len(m.categories) == 0 {
		return nil
	}
	if m.catIdx >= len(m.categories) {
		return nil
	}
	return m.categories[m.catIdx].Commands
}

func (m mainModel) currentSubCommands() []SubCommand {
	cmds := m.currentCommands()
	if len(cmds) == 0 || m.cmdIdx >= len(cmds) {
		return nil
	}
	return cmds[m.cmdIdx].SubCmds
}

func (m *mainModel) currentFlags() []Flag {
	cmds := m.currentCommands()
	if len(cmds) == 0 || m.cmdIdx >= len(cmds) {
		return nil
	}
	cmd := cmds[m.cmdIdx]

	if len(cmd.SubCmds) > 0 {
		if m.subIdx < len(cmd.SubCmds) {
			return cmd.SubCmds[m.subIdx].Flags
		}
		return nil
	}
	return cmd.Flags
}

func (m *mainModel) currentRequiredFlagGroup() []string {
	cmds := m.currentCommands()
	if len(cmds) == 0 || m.cmdIdx >= len(cmds) {
		return nil
	}
	cmd := cmds[m.cmdIdx]

	if len(cmd.SubCmds) > 0 {
		if m.subIdx < len(cmd.SubCmds) {
			return cmd.SubCmds[m.subIdx].RequiredFlagGroup
		}
		return nil
	}
	return cmd.RequiredFlagGroup
}

// buildCommandStrings returns (short, long) where short uses command/subcommand
// aliases when available, and long always uses the full name.
func (m mainModel) buildCommandStrings() (short, long string) {
	cmds := m.currentCommands()
	if len(cmds) == 0 || m.cmdIdx >= len(cmds) {
		return "jj", "jj"
	}

	cmd := cmds[m.cmdIdx]

	shortName := cmd.Name
	if cmd.Alias != "" && len(cmd.Alias) < len(cmd.Name) {
		shortName = cmd.Alias
	}
	shortParts := []string{"jj", shortName}
	longParts := []string{"jj", cmd.Name}

	if len(cmd.SubCmds) > 0 && m.subIdx < len(cmd.SubCmds) {
		sub := cmd.SubCmds[m.subIdx]
		longPart := sub.Value
		if longPart == "" {
			longPart = sub.Name
		}
		shortPart := longPart
		if sub.Alias != "" && len(sub.Alias) < len(longPart) {
			shortPart = sub.Alias
		}
		shortParts = append(shortParts, shortPart)
		longParts = append(longParts, longPart)
	}

	flags := m.currentFlags()
	for _, f := range flags {
		if f.Selected {
			if f.Value != "" {
				shortParts = append(shortParts, f.Value)
				longParts = append(longParts, f.Value)
			}
			if f.RequiresInput {
				val := m.inputs[f.Name].Value()
				if val != "" {
					if f.NeedsQuotes {
						val = "\"" + val + "\""
					}
					shortParts = append(shortParts, val)
					longParts = append(longParts, val)
				}
			}
		}
	}

	// Append positional arg values after flags. Optional args are simply
	// omitted when empty, letting jj apply its own default (e.g. @).
	for _, a := range m.currentArgs() {
		if val := m.argInputs[a.Name].Value(); val != "" {
			shortParts = append(shortParts, val)
			longParts = append(longParts, val)
		}
	}

	return strings.Join(shortParts, " "), strings.Join(longParts, " ")
}

// outputEnlargedActive reports whether the output pane should render in its
// enlarged, full-width/full-height form. Enlarge is cleared explicitly by
// every path that leaves the output pane (see handleOutputKeys and the mouse
// handlers), so checking both the flag and current focus here is a
// belt-and-suspenders guard against a stale flag reappearing on refocus.
func (m mainModel) outputEnlargedActive() bool {
	return m.outputEnlarged && m.focusPane == focusOutput
}

// docsEnlargedActive reports whether the docs pane should render in its
// enlarged form, filling the composer-panes/input-pane region while the
// output pane stays at its normal size. Mirrors outputEnlargedActive()'s
// belt-and-suspenders guard against a stale flag reappearing on refocus.
func (m mainModel) docsEnlargedActive() bool {
	return m.docsEnlarged && m.focusPane == focusDocs
}

// exitOutput returns focus to the last composer pane and clears the output
// pane's enlarged state — the shared toggle-out action for both "o" and "O".
func (m mainModel) exitOutput() (tea.Model, tea.Cmd) {
	m.focusPane = m.lastFocusPane
	m.outputEnlarged = false
	m.layoutViewports()
	return m, nil
}

// exitDocs returns focus to the last composer pane and clears the docs
// pane's enlarged state — the shared toggle-out action for both "d" and "D".
func (m mainModel) exitDocs() (tea.Model, tea.Cmd) {
	m.focusPane = m.lastFocusPane
	m.docsEnlarged = false
	m.layoutViewports()
	m.refreshDocs()
	return m, nil
}

// docsContentDims returns the width/height for m.docs' viewport given the
// docs box's outer width and total height (title box = 3 rows). Shared by
// layoutViewports() and renderDescriptionBar() so they can't drift apart
// the way output's sizing once did (see CLAUDE.md "Known fragile points").
func docsContentDims(width, barHeight int) (w, h int) {
	w = width - inactiveContentBorderStyle.GetHorizontalFrameSize() - scrollbarW - scrollbarGapW
	h = barHeight - 3 - inactiveContentBorderStyle.GetVerticalFrameSize()
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return
}

func (m *mainModel) layoutViewports() {
	if m.outputEnlargedActive() {
		height := m.height - 6 // 5 cmdBar + 1 helpBar
		m.output.SetWidth(m.width - 4 - scrollbarW - scrollbarGapW)
		m.output.SetHeight(height - 4)
		return
	}

	if m.docsEnlargedActive() {
		height := m.height - 6 // 5 cmdBar + 1 helpBar
		if height < 6 {
			height = 6
		}
		docsW, docsH := docsContentDims(m.width, height)
		m.docs.SetWidth(docsW)
		m.docs.SetHeight(docsH)
		return
	}

	leftWidth, rightWidth := m.getLayoutWidths()

	combined := m.getActiveCombined()
	inputsHeight := 0
	if len(combined) > 0 {
		inputsHeight = len(combined) + 2
	}

	outputHeight := m.height - 6 // matches View(): 5 cmdBar + 1 helpBar
	if outputHeight < 6 {
		outputHeight = 6
	}
	paneContentWidth := rightWidth - 4 - scrollbarW - scrollbarGapW
	paneContentHeight := outputHeight - 4 // title box (3) + content bottom border (1)

	m.output.SetWidth(paneContentWidth)
	m.output.SetHeight(paneContentHeight)

	// Composer/description bar share leftBudget, not outputHeight (matches View()).
	leftBudget := outputHeight - inputsHeight
	if leftBudget < 6 {
		leftBudget = 6
	}
	m.paneH = leftBudget / 2

	// Docs viewport: fits inside the description bar's content box (title
	// box + border consume 3 rows, handled by docsContentDims).
	composerH := leftBudget / 2
	descBarH := leftBudget - composerH
	docsW, docsH := docsContentDims(leftWidth, descBarH)
	m.docs.SetWidth(docsW)
	m.docs.SetHeight(docsH)

	// Size each input field to fill the remaining box width so text can
	// scroll all the way to the border instead of stopping at a fixed width.
	boxContentWidth := leftWidth - 4 // border(2) + padding(2), matches renderInputPanes
	for _, item := range combined {
		ti := m.getInput(item)
		// ponytail: -1 margin works around a bubbles textinput off-by-one; see
		// CLAUDE.md "Known fragile points".
		avail := boxContentWidth - inputPromptWidth(item.Label) - lipgloss.Width(ti.Prompt) - 1
		if avail < 1 {
			avail = 1
		}
		ti.SetWidth(avail)
		m.setInput(item, ti)
	}
}

func (m mainModel) currentDescription() string {
	cmds := m.currentCommands()
	if len(cmds) == 0 || m.cmdIdx >= len(cmds) {
		return ""
	}
	cmd := cmds[m.cmdIdx]
	switch m.focusPane {
	case focusSubcmds:
		if m.subIdx < len(cmd.SubCmds) {
			return cmd.SubCmds[m.subIdx].Description
		}
	case focusFlags:
		flags := m.currentFlags()
		if m.flagIdx < len(flags) {
			return flags[m.flagIdx].Description
		}
	}
	return cmd.Description
}

func (m mainModel) currentName() string {
	switch m.focusPane {
	case focusCategories:
		if m.catIdx < len(m.categories) {
			return m.categories[m.catIdx].Name
		}
	case focusCommands:
		cmds := m.currentCommands()
		if m.cmdIdx < len(cmds) {
			return cmds[m.cmdIdx].Name
		}
	case focusSubcmds:
		cmds := m.currentCommands()
		if len(cmds) > 0 && m.cmdIdx < len(cmds) {
			cmd := cmds[m.cmdIdx]
			if m.subIdx < len(cmd.SubCmds) {
				return cmd.SubCmds[m.subIdx].Name
			}
		}
	case focusFlags:
		flags := m.currentFlags()
		if m.flagIdx < len(flags) {
			return flags[m.flagIdx].Name
		}
	}
	return ""
}

// clampFocusPane resets focus to focusCommands if the current pane has no items.
// Called after navigating between commands/categories.
func (m *mainModel) clampFocusPane() {
	if m.focusPane == focusFlags && len(m.currentFlags()) == 0 {
		m.focusPane = focusCommands
	}
	if m.focusPane == focusSubcmds && len(m.currentSubCommands()) == 0 {
		m.focusPane = focusCommands
	}
}

// flagsConflict reports whether flags a and b cannot be selected together.
// Checks both directions so only one side needs to declare the conflict.
func flagsConflict(a, b Flag) bool {
	for _, v := range a.ConflictingFlags {
		if v == b.Value {
			return true
		}
	}
	for _, v := range b.ConflictingFlags {
		if v == a.Value {
			return true
		}
	}
	return false
}

// isConflicted reports whether any currently-selected flag in flags conflicts with f.
func isConflicted(f Flag, flags []Flag) bool {
	for _, other := range flags {
		if other.Selected && flagsConflict(f, other) {
			return true
		}
	}
	return false
}

// isRequiredGroupFlag reports whether f belongs to the current command's
// RequiredFlagGroup while that group is still unsatisfied.
func isRequiredGroupFlag(m mainModel, f Flag) bool {
	if !m.requiredFlagGroupUnsatisfied() {
		return false
	}
	return slices.Contains(m.currentRequiredFlagGroup(), f.Value)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m mainModel) hasEmptyRequiredFlagInput() bool {
	for _, f := range m.currentFlags() {
		if f.Selected && f.RequiresInput && m.inputs[f.Name].Value() == "" {
			return true
		}
	}
	return false
}

func (m mainModel) hasEmptyRequiredArg() bool {
	for _, a := range m.getRequiredArgs() {
		if m.argInputs[a.Name].Value() == "" {
			return true
		}
	}
	return false
}

// requiredFlagGroupUnsatisfied reports whether the current command declares a
// RequiredFlagGroup and none of its member flags are currently selected.
func (m mainModel) requiredFlagGroupUnsatisfied() bool {
	group := m.currentRequiredFlagGroup()
	if len(group) == 0 {
		return false
	}
	for _, f := range m.currentFlags() {
		if f.Selected && slices.Contains(group, f.Value) {
			return false
		}
	}
	return true
}

func (m mainModel) hasIncompleteInputs() bool {
	return m.hasEmptyRequiredFlagInput() || m.hasEmptyRequiredArg() || m.requiredFlagGroupUnsatisfied()
}

func flashTimer() tea.Cmd {
	return tea.Tick(400*time.Millisecond, func(time.Time) tea.Msg {
		return clearValidationFlashMsg{}
	})
}

func copyFlashTimer() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg {
		return clearCopyFlashMsg{}
	})
}

// stripANSI removes ANSI CSI escape sequences from s, returning plain text.
func stripANSI(s string) string {
	var out strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			j := i + 2
			for j < len(runes) && !((runes[j] >= 'A' && runes[j] <= 'Z') || (runes[j] >= 'a' && runes[j] <= 'z')) {
				j++
			}
			if j < len(runes) {
				j++
			}
			i = j
		} else {
			out.WriteRune(runes[i])
			i++
		}
	}
	return out.String()
}

func initMandatoryFlagInputs(flags []Flag, inputs map[string]textinput.Model) {
	// Mandatory flags always appear first in the list.
	sort.SliceStable(flags, func(i, j int) bool {
		return flags[i].Mandatory && !flags[j].Mandatory
	})
	for i := range flags {
		if flags[i].Mandatory && flags[i].RequiresInput {
			// ponytail: iterate by index so Selected=true writes back to the slice
			flags[i].Selected = true
			if _, exists := inputs[flags[i].Name]; !exists {
				ti := textinput.New()
				ti.Prompt = " "
				ti.Placeholder = "Enter " + flags[i].Name + "..."
				ti.CharLimit = 256
				inputs[flags[i].Name] = ti
			}
		}
	}
}

func (m *mainModel) getActiveInputs() []*Flag {
	var active []*Flag
	if len(m.categories) == 0 || m.catIdx >= len(m.categories) {
		return active
	}
	cat := &m.categories[m.catIdx]
	if len(cat.Commands) == 0 || m.cmdIdx >= len(cat.Commands) {
		return active
	}
	cmd := &cat.Commands[m.cmdIdx]

	if len(cmd.SubCmds) > 0 {
		if m.subIdx < len(cmd.SubCmds) {
			for i := range cmd.SubCmds[m.subIdx].Flags {
				if cmd.SubCmds[m.subIdx].Flags[i].Selected && cmd.SubCmds[m.subIdx].Flags[i].RequiresInput {
					active = append(active, &cmd.SubCmds[m.subIdx].Flags[i])
				}
			}
		}
	} else {
		for i := range cmd.Flags {
			if cmd.Flags[i].Selected && cmd.Flags[i].RequiresInput {
				active = append(active, &cmd.Flags[i])
			}
		}
	}
	return active
}

func (m mainModel) currentArgs() []Arg {
	cmds := m.currentCommands()
	if len(cmds) == 0 || m.cmdIdx >= len(cmds) {
		return nil
	}
	cmd := cmds[m.cmdIdx]
	if len(cmd.SubCmds) > 0 {
		if m.subIdx < len(cmd.SubCmds) {
			return cmd.SubCmds[m.subIdx].Args
		}
		return nil
	}
	return cmd.Args
}

func (m mainModel) getRequiredArgs() []Arg {
	var req []Arg
	for _, a := range m.currentArgs() {
		if a.Required {
			req = append(req, a)
		}
	}
	return req
}

// getActiveCombined returns positional args and selected flag inputs ordered by two keys:
// required items before optional items, and within each tier, flags before args — so a
// mandatory flag always precedes an optional arg, and a selected optional flag always
// precedes an optional arg too (args are always shown, so they read as the trailing,
// lowest-priority fields). It lazy-initializes arg textinputs as needed (safe from a value
// receiver because maps are reference types). Optional args are included so the user can
// supply a value, but hasIncompleteInputs() only validates required ones — leaving an
// optional arg blank is fine and lets jj apply its own default.
func (m mainModel) getActiveCombined() []InputItem {
	var requiredFlags, requiredArgs, optionalFlags, optionalArgs []InputItem

	for _, a := range m.currentArgs() {
		if _, exists := m.argInputs[a.Name]; !exists {
			ti := textinput.New()
			ti.Prompt = " "
			placeholder := "enter " + strings.ToLower(a.Name)
			if !a.Required {
				placeholder += " (optional)"
			}
			ti.Placeholder = placeholder + "..."
			ti.CharLimit = 256
			m.argInputs[a.Name] = ti
		}
		item := InputItem{Name: a.Name, Label: a.Name, IsArg: true}
		if a.Required {
			requiredArgs = append(requiredArgs, item)
		} else {
			optionalArgs = append(optionalArgs, item)
		}
	}
	for _, f := range m.getActiveInputs() {
		item := InputItem{Name: f.Name, Label: f.Name, IsArg: false}
		if f.Mandatory {
			requiredFlags = append(requiredFlags, item)
		} else {
			optionalFlags = append(optionalFlags, item)
		}
	}

	items := append(requiredFlags, requiredArgs...)
	items = append(items, optionalFlags...)
	items = append(items, optionalArgs...)
	return items
}

func (m mainModel) handleInputKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	combined := m.getActiveCombined()
	if len(combined) == 0 {
		m.focusPane = m.lastFocusPane
		return m, nil
	}
	if m.focusedInputIdx >= len(combined) {
		m.focusedInputIdx = len(combined) - 1
	}

	cur := combined[m.focusedInputIdx]

	switch msg.String() {
	case "esc":
		ti := m.getInput(cur)
		ti.Blur()
		m.setInput(cur, ti)
		m.focusPane = m.lastFocusPane
		m.cmdText, m.cmdTextLong = m.buildCommandStrings()
		return m, nil
	case "up":
		if m.focusedInputIdx > 0 {
			oldTi := m.getInput(cur)
			oldTi.Blur()
			m.setInput(cur, oldTi)

			m.focusedInputIdx--

			newItem := combined[m.focusedInputIdx]
			newTi := m.getInput(newItem)
			cmd := newTi.Focus()
			m.setInput(newItem, newTi)

			m.cmdText, m.cmdTextLong = m.buildCommandStrings()
			return m, cmd
		}
		return m, nil
	case "down":
		if m.focusedInputIdx < len(combined)-1 {
			oldTi := m.getInput(cur)
			oldTi.Blur()
			m.setInput(cur, oldTi)

			m.focusedInputIdx++

			newItem := combined[m.focusedInputIdx]
			newTi := m.getInput(newItem)
			cmd := newTi.Focus()
			m.setInput(newItem, newTi)

			m.cmdText, m.cmdTextLong = m.buildCommandStrings()
			return m, cmd
		}
		return m, nil
	case "enter":
		oldTi := m.getInput(cur)
		oldTi.Blur()
		m.setInput(cur, oldTi)
		m.focusedInputIdx = (m.focusedInputIdx + 1) % len(combined)
		newItem := combined[m.focusedInputIdx]
		newTi := m.getInput(newItem)
		cmd := newTi.Focus()
		m.setInput(newItem, newTi)
		m.cmdText, m.cmdTextLong = m.buildCommandStrings()
		return m, cmd
	case "tab":
		if m.hasIncompleteInputs() {
			m.validationFlash = true
			return m, flashTimer()
		}
		m.focusPane = focusCmdBar
		m.cmdText, m.cmdTextLong = m.buildCommandStrings()
		return m, nil
	}

	ti := m.getInput(cur)
	var cmd tea.Cmd
	ti, cmd = ti.Update(msg)
	m.setInput(cur, ti)
	m.cmdText, m.cmdTextLong = m.buildCommandStrings()
	return m, cmd
}

// --- Docs pane --------------------------------------------------------------

func (m mainModel) handleDocsKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "down", "j":
		m.docs.SetYOffset(m.docs.YOffset() + 1)
	case "up", "k":
		if m.docs.YOffset() > 0 {
			m.docs.SetYOffset(m.docs.YOffset() - 1)
		}
	case "c":
		text := stripANSI(m.docsContent())
		return m, copyToClipboard(text)
	case "tab":
		m.focusPane = focusCmdBar
		m.docsEnlarged = false
	case "esc":
		m.focusPane = m.lastFocusPane
		m.docsEnlarged = false
	}
	m.layoutViewports()
	return m, nil
}

// refreshDocs rebuilds the docs viewport content from the currently focused item.
func (m *mainModel) refreshDocs() {
	m.docs.SetYOffset(0)
	width := m.docs.Width()
	if width < 10 {
		width = 78 // fallback before first WindowSizeMsg
	}
	m.docs.SetContent(wordWrap(m.docsContent(), width))
}

// docsContent returns the full documentation text for the focused item.
func (m mainModel) docsContent() string {
	cmds := m.currentCommands()
	if len(cmds) == 0 || m.cmdIdx >= len(cmds) {
		return dimItemStyle.Render("no description")
	}
	cmd := cmds[m.cmdIdx]

	switch m.focusPane {
	case focusFlags:
		flags := m.currentFlags()
		if m.flagIdx < len(flags) {
			f := flags[m.flagIdx]
			header := headerStyle.Render(f.Name)
			if f.Description == "" {
				return header
			}
			return header + "\n\n" + f.Description
		}
	case focusSubcmds:
		if m.subIdx < len(cmd.SubCmds) {
			sub := cmd.SubCmds[m.subIdx]
			return buildDocsBlock(sub.Name, cmd.Name+" "+sub.Name, sub.Alias, sub.Description, sub.Args, sub.Flags, sub.RequiredUsage, nil)
		}
	}
	return buildDocsBlock(cmd.Name, cmd.Name, cmd.Alias, cmd.Description, cmd.Args, cmd.Flags, cmd.RequiredUsage, cmd.SubCmds)
}

func buildDocsBlock(name, usagePath, alias, desc string, args []Arg, flags []Flag, requiredUsage string, subCmds []SubCommand) string {
	var b strings.Builder

	// Build usage synopsis: jj <path> [OPTIONS] <required flags> [ARG[...]]
	// jj always shows [OPTIONS] (global options apply to every command), and
	// required-flag/group syntax (e.g. "<--onto|--insert-after|--insert-before>")
	// is sourced verbatim from `jj util markdown-help` by gen-descriptions,
	// since Jitsu's own Mandatory flag doesn't reliably mean "jj requires this".
	usage := "jj " + usagePath + " [OPTIONS]"
	if requiredUsage != "" {
		usage += " " + requiredUsage
	}
	// A command that dispatches to subcommands and has no Args of its own
	// (e.g. config, bookmark, git) always takes a required COMMAND
	// positional in real jj usage. "help" is excluded since it already
	// models this itself via a variadic, optional Args entry.
	if len(subCmds) > 0 && len(args) == 0 {
		usage += " <COMMAND>"
	}
	for _, a := range args {
		if a.Required {
			if a.Variadic {
				usage += " <" + a.Name + ">..."
			} else {
				usage += " <" + a.Name + ">"
			}
		} else {
			if a.Variadic {
				usage += " [" + a.Name + "]..."
			} else {
				usage += " [" + a.Name + "]"
			}
		}
	}

	// Section order matches `jj --help`: Name → Alias → Description → Usage → Subcommands → Arguments → Options
	b.WriteString(headerStyle.Render(name))
	if alias != "" {
		b.WriteString("\n\n" + headerStyle.Render("Alias") + "\n  " + alias)
	}
	if desc != "" {
		b.WriteString("\n\n" + desc)
	}
	b.WriteString("\n\n" + headerStyle.Render("Usage") + "\n  " + usage)
	if len(subCmds) > 0 {
		b.WriteString("\n\n" + headerStyle.Render("Subcommands"))
		for _, s := range subCmds {
			line := boldStyle.Render(s.Name)
			if s.Summary != "" {
				line += " — " + s.Summary
			}
			b.WriteString("\n  " + line)
		}
	}
	if len(args) > 0 {
		b.WriteString("\n\n" + headerStyle.Render("Arguments"))
		for _, a := range args {
			argLabel := a.Name
			if a.Variadic {
				argLabel += "..."
			}
			b.WriteString("\n  " + boldStyle.Render(argLabel))
			if a.Description != "" {
				b.WriteString("\n  " + a.Description)
			}
		}
	}
	if len(flags) > 0 {
		b.WriteString("\n\n" + headerStyle.Render("Options"))
		for _, f := range flags {
			// Build flag label: "-x, --long-name <TYPE>" or "--long-name <TYPE>"
			flagLabel := f.Value
			if len(f.Value) == 2 && f.Value[0] == '-' && f.Value[1] != '-' {
				flagLabel = f.Value + ", --" + f.Name
			}
			if f.InputType != "" {
				flagLabel += " <" + f.InputType + ">"
			}
			b.WriteString("\n  " + boldStyle.Render(flagLabel))
			if f.Description != "" {
				b.WriteString("\n  " + f.Description)
			}
		}
	}
	return b.String()
}

// wordWrap wraps s so no visible line exceeds width characters.
// It preserves existing newlines and handles paragraph breaks (blank lines).
func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out strings.Builder
	paragraphs := strings.Split(s, "\n")
	for i, line := range paragraphs {
		if i > 0 {
			out.WriteByte('\n')
		}
		// Lines that start with ANSI escape or are very short: pass through.
		if len(line) <= width || strings.HasPrefix(line, "\x1b") {
			out.WriteString(line)
			continue
		}
		// Word-wrap at width.
		words := strings.Fields(line)
		col := 0
		for wi, w := range words {
			wlen := len([]rune(w))
			if wi == 0 {
				out.WriteString(w)
				col = wlen
			} else if col+1+wlen > width {
				out.WriteByte('\n')
				out.WriteString(w)
				col = wlen
			} else {
				out.WriteByte(' ')
				out.WriteString(w)
				col += 1 + wlen
			}
		}
	}
	return out.String()
}
