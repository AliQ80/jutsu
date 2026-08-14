package main

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
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
	maxInputsHt   int // fixed reservation for the inputs box; see maxInputsHeight()

	globalFlags          []Flag // jj's global options — selected independent of catIdx/cmdIdx/subIdx
	globalFlagIdx        int
	globalFlagsModalOpen bool
	globalModalDocsFocus bool           // true → ↑↓ scroll the docs column instead of the checklist
	globalDocs           viewport.Model // docs column inside the global-options modal

	output          viewport.Model
	outputLines     []string
	xOffset         int
	outputEnlarged  bool
	docs            viewport.Model
	docsEnlarged    bool
	inputs          map[string]textinput.Model // keyed by flag Name
	argInputs       map[string]textinput.Model // keyed by Arg Name
	focusedInputIdx int

	cmdText     string
	cmdTextLong string
	running     bool
	spinner     spinner.Model    // animates only while running (see spinner.TickMsg)
	lastCmd     *commandSnapshot // nil until first command executes

	// Remote-prompt modals (see execResultMsg classification). A failed
	// push/fetch/clone that needed a prompt surfaces one of these instead of
	// a raw error.
	hostKeyModalOpen bool              // SSH host-key approval popup
	hostKeyPrompt    hostKeyFetchedMsg // host / fingerprint / keyscan lines / pushCmd
	hostKeyTried     bool              // loop guard: only offer the host-key modal once per run
	authModalOpen    bool              // "continue in terminal" popup for credential prompts
	authModalCmd     string            // the command that popup would retry via the pty handoff
	aboutModalOpen   bool              // read-only popup with the running jutsu version + GitHub link (opened via ?)

	validationFlash bool
	copyFlash       bool
	jjVersion       string
	latestTag       string // latest jj tag resolved at startup; overrides "dev" in the about modal
	cwd             string

	width  int
	height int

	// backdrop is a snapshot of the model taken the instant before a modal
	// opens (global-options / host-key / auth). View() reflows it at the
	// current width/height and overlays the modal on top, so the full app
	// frame stays visible — frozen at its pre-modal state — behind the
	// popup. nil outside of a modal.
	backdrop *mainModel
}

// freezeBackdrop snapshots the current visual state for later use as the
// modal backdrop. Call this before flipping a *ModalOpen flag, not after —
// the snapshot must reflect the screen as it looked right before the modal
// took over.
//
// A plain `snap := *m` struct copy is not enough: mainModel's slices/maps
// (categories, globalFlags, inputs, argInputs, outputLines) are reference
// types, so a shallow copy shares their backing arrays/maps with the live
// model. Flag.Selected is mutated in place through those shared backing
// arrays (see toggleFlagAt, restoreLastCmd), and inputs/argInputs are
// mutated by reassigning map entries (see setInput) — both of which would
// silently rewrite the "frozen" backdrop out from under the modal. Each of
// those fields is deep-copied below so the snapshot is genuinely immutable
// from the live model's perspective.
func (m *mainModel) freezeBackdrop() {
	snap := *m
	snap.backdrop = nil // a frozen backdrop never carries its own backdrop

	snap.categories = cloneCategories(m.categories)
	snap.globalFlags = cloneFlags(m.globalFlags)
	snap.inputs = cloneTextInputs(m.inputs)
	snap.argInputs = cloneTextInputs(m.argInputs)

	// outputLines is always reassigned wholesale rather than mutated in
	// place, so sharing its backing array would be safe today — but a
	// fresh copy costs nothing and removes the assumption entirely.
	snap.outputLines = append([]string(nil), m.outputLines...)

	m.backdrop = &snap
}

// cloneFlags deep-copies a []Flag so mutations to Selected (or anything
// else) on the original slice can never be observed through the copy.
// ConflictingFlags is left sharing its backing array since it's populated
// once at load time and never mutated.
func cloneFlags(fs []Flag) []Flag {
	if fs == nil {
		return nil
	}
	out := make([]Flag, len(fs))
	copy(out, fs)
	return out
}

// cloneArgs deep-copies a []Arg. Arg has no mutable fields, but a fresh
// slice keeps the contract simple: nothing in a frozen snapshot shares a
// backing array with the live model.
func cloneArgs(as []Arg) []Arg {
	if as == nil {
		return nil
	}
	out := make([]Arg, len(as))
	copy(out, as)
	return out
}

// cloneSubCommands deep-copies a []SubCommand, including each entry's Args
// and Flags slices.
func cloneSubCommands(scs []SubCommand) []SubCommand {
	if scs == nil {
		return nil
	}
	out := make([]SubCommand, len(scs))
	for i, sc := range scs {
		sc.Args = cloneArgs(sc.Args)
		sc.Flags = cloneFlags(sc.Flags)
		out[i] = sc
	}
	return out
}

// cloneCommands deep-copies a []Command, including each entry's Args,
// SubCmds, and Flags slices.
func cloneCommands(cmds []Command) []Command {
	if cmds == nil {
		return nil
	}
	out := make([]Command, len(cmds))
	for i, cmd := range cmds {
		cmd.Args = cloneArgs(cmd.Args)
		cmd.SubCmds = cloneSubCommands(cmd.SubCmds)
		cmd.Flags = cloneFlags(cmd.Flags)
		out[i] = cmd
	}
	return out
}

// cloneCategories deep-copies the entire category tree. Selected is toggled
// in place on Flag entries nested arbitrarily deep inside it (Category ->
// Command -> Flags, or Category -> Command -> SubCommand -> Flags), so
// every level needs its own fresh slice.
func cloneCategories(cats []Category) []Category {
	if cats == nil {
		return nil
	}
	out := make([]Category, len(cats))
	for i, c := range cats {
		c.Commands = cloneCommands(c.Commands)
		out[i] = c
	}
	return out
}

// cloneTextInput returns an independent copy of a textinput.Model. Textinput
// keeps its text in an unexported []rune buffer that some of its own mutating
// methods (e.g. insertRunesFromUserInput's `head = append(head, r)`) can grow
// in place, which — if the buffer's backing array were shared with a frozen
// backdrop copy — could silently overwrite runes visible through the frozen
// copy. Rebuilding via the public API avoids sharing that buffer at all.
func cloneTextInput(t textinput.Model) textinput.Model {
	clone := textinput.New()
	clone.Prompt = t.Prompt
	clone.Placeholder = t.Placeholder
	clone.EchoMode = t.EchoMode
	clone.EchoCharacter = t.EchoCharacter
	clone.CharLimit = t.CharLimit
	clone.KeyMap = t.KeyMap
	clone.SetVirtualCursor(t.VirtualCursor())
	clone.SetStyles(t.Styles())
	clone.SetWidth(t.Width())
	clone.SetValue(t.Value())
	clone.SetCursor(t.Position())
	if t.Focused() {
		clone.Focus()
	} else {
		clone.Blur()
	}
	return clone
}

// cloneTextInputs deep-copies a map of textinput.Model, both so the map
// itself is a distinct object (setInput reassigns entries in place on the
// live map) and so no entry's rune buffer is shared with the live model.
func cloneTextInputs(in map[string]textinput.Model) map[string]textinput.Model {
	if in == nil {
		return nil
	}
	out := make(map[string]textinput.Model, len(in))
	for k, v := range in {
		out[k] = cloneTextInput(v)
	}
	return out
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

	// The global-options modal is fixed-size, so its docs viewport is sized
	// once here; wordWrap in refreshGlobalDocs relies on this width.
	gv := viewport.New()
	gv.SetWidth(globalDocsW)
	gv.SetHeight(globalDocsH)

	cwd := "?"
	if wd, err := os.Getwd(); err == nil {
		cwd = filepath.Base(wd)
	}

	m := mainModel{
		categories:  cats,
		focusPane:   focusCategories,
		output:      vp,
		docs:        dv,
		globalDocs:  gv,
		inputs:      make(map[string]textinput.Model),
		argInputs:   make(map[string]textinput.Model),
		cwd:         cwd,
		maxInputsHt: maxInputsHeight(cats),
		globalFlags: loadGlobalFlags(),
		spinner:     spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(runningStyle)),
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
	initMandatoryFlagInputs(m.globalFlags, m.inputs)

	m.cmdText, m.cmdTextLong = m.buildCommandStrings()
	m.refreshDocs()
	return m
}

func (m mainModel) Init() tea.Cmd {
	return tea.Batch(fetchJJVersion(), fetchLatestTag())
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

		// A failed remote op may have hit a prompt the detached (no-tty) run
		// couldn't answer. Classify and surface the right modal instead of a
		// raw error line. See the classifiers in commands_exec.go.
		if msg.err != nil && isRemoteGitCommand(msg.cmdStr) {
			switch {
			case isUnknownHostKey(msg.output) && !m.hostKeyTried:
				// First connect to an unknown host: scan its key off the
				// Update loop, then show the fingerprint-approval modal.
				m.hostKeyTried = true
				m.running = true
				return m, tea.Batch(m.spinner.Tick, fetchHostKey(msg.cmdStr))
			case isAuthFailure(msg.output):
				// Credentials needed: offer the guided terminal handoff.
				m.freezeBackdrop()
				m.authModalOpen = true
				m.authModalCmd = msg.cmdStr
				return m, nil
			}
		}

		lines := commandEchoLines(msg.cmdStr)
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

	case hostKeyFetchedMsg:
		m.running = false
		if msg.err != nil {
			// Couldn't scan/resolve the host key — fall back to the guided
			// terminal handoff so real ssh can prompt.
			m.freezeBackdrop()
			m.authModalOpen = true
			m.authModalCmd = msg.pushCmd
			return m, nil
		}
		m.hostKeyPrompt = msg
		m.freezeBackdrop()
		m.hostKeyModalOpen = true
		return m, nil

	case knownHostAddedMsg:
		if msg.err != nil {
			m.running = false
			lines := commandEchoLines(msg.pushCmd)
			lines = append(lines, errorStyle.Render("✗ could not update known_hosts: "+msg.err.Error()))
			m.outputLines = lines
			m.xOffset = 0
			m.applyXOffset()
			m.output.GotoTop()
			return m, nil
		}
		// Host now trusted — silently retry the original push.
		m.running = true
		return m, tea.Batch(m.spinner.Tick, executeCommand(msg.pushCmd))

	case execFallbackMsg:
		// herdr couldn't give us a terminal — suspend the TUI and use the
		// ordinary pty handoff instead of failing the command outright.
		return m, executeInteractive(msg.cmdStr, msg.plain)

	case execInteractiveResultMsg:
		// The subprocess ran on a pty; msg.output holds the plain-text tail
		// recovered after its alternate screen closed, if any was found.
		m.running = false
		lines := commandEchoLines(msg.cmdStr)

		if msg.err != nil {
			errorMsg := msg.err.Error()
			if msg.output != "" {
				errorMsg = msg.output + "\n" + errorMsg
			}
			lines = append(lines, strings.Split(errorMsg, "\n")...)
		} else if msg.output != "" {
			lines = append(lines, strings.Split(msg.output, "\n")...)
		} else {
			lines = append(lines, "✓ Command completed (output shown in the terminal)")
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

	case latestTagMsg:
		m.latestTag = string(msg)
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
		if m.globalFlagsModalOpen {
			return m.handleGlobalFlagsModalMouse(msg)
		}
		if m.hostKeyModalOpen || m.authModalOpen {
			return m, nil
		}
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
		composerH, inputsTop, bottom := m.paneRows()

		if msg.X >= leftWidth && msg.Y < bottom {
			if msg.Button == tea.MouseWheelUp && m.output.YOffset() > 0 {
				m.output.SetYOffset(m.output.YOffset() - 1)
			} else if msg.Button == tea.MouseWheelDown {
				m.output.SetYOffset(m.output.YOffset() + 1)
			}
		} else if msg.X < leftWidth && msg.Y >= composerH && msg.Y < inputsTop {
			if msg.Button == tea.MouseWheelUp && m.docs.YOffset() > 0 {
				m.docs.SetYOffset(m.docs.YOffset() - 1)
			} else if msg.Button == tea.MouseWheelDown {
				m.docs.SetYOffset(m.docs.YOffset() + 1)
			}
		}
		return m, nil

	case tea.MouseClickMsg:
		if m.globalFlagsModalOpen {
			return m.handleGlobalFlagsModalClick(msg)
		}
		if m.hostKeyModalOpen || m.authModalOpen {
			return m, nil
		}
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
		composerH, inputsTop, bottom := m.paneRows()

		switch {
		case msg.X >= leftWidth && msg.Y < bottom:
			if m.focusPane <= focusFlags {
				m.lastFocusPane = m.focusPane
			}
			m.focusPane = focusOutput
		case msg.X < leftWidth && msg.Y >= composerH && msg.Y < inputsTop:
			if m.focusPane <= focusFlags {
				m.lastFocusPane = m.focusPane
			}
			m.focusPane = focusDocs
		case msg.X < leftWidth && msg.Y >= inputsTop && msg.Y < bottom:
			// The box's top border owns the first row, so the Nth row
			// of the box is input N-1.
			return m, m.focusInput(msg.Y - inputsTop - 1)
		case msg.Y >= bottom && msg.Y < bottom+5:
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

	case spinner.TickMsg:
		// Ticks stop themselves once the command finishes — an idle Jutsu
		// shouldn't redraw every frame.
		if !m.running {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		if m.globalFlagsModalOpen {
			return m.handleGlobalFlagsModalKeys(msg)
		}
		if m.hostKeyModalOpen {
			return m.handleHostKeyModalKeys(msg)
		}
		if m.authModalOpen {
			return m.handleAuthModalKeys(msg)
		}
		if m.aboutModalOpen {
			return m.handleAboutModalKeys(msg)
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.focusPane != focusInputs {
				return m, tea.Quit
			}
		case "g":
			if m.focusPane != focusInputs {
				m.freezeBackdrop()
				m.globalFlagsModalOpen = true
				m.globalModalDocsFocus = false // always open on the checklist
				m.refreshGlobalDocs()
				return m, nil
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
				content := m.docsContent() // capture against the pre-switch focus pane
				m.docsEnlarged = true
				m.focusPane = focusDocs
				m.layoutViewports() // now sees docsEnlargedActive() == true, sizes for enlarged width
				m.setDocsContent(content)
				return m, nil
			}
		case "i":
			// Jump into the inputs box from any pane — including the output,
			// docs and command bar, and from any composer pane, since the
			// global-options modal can land focus on Categories with a pending
			// global input. ("i", not enter — enter means exactly one thing
			// app-wide: execute.) Blocked inside the inputs box itself, where
			// "i" is a character to type, and in the enlarged views, which
			// don't draw the box — same guard as d/o.
			if m.focusPane != focusInputs && !m.outputEnlarged && !m.docsEnlarged {
				if cmd := m.focusInput(0); cmd != nil {
					return m, cmd
				}
			}
		case "?":
			// Open the about modal (version + GitHub link). Blocked inside
			// the inputs pane, where "?" is a literal character being typed.
			// No enlarged-view guard needed: "? "is harmless in those views.
			if m.focusPane != focusInputs {
				m.aboutModalOpen = true
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
		m.reclampAllScrolls()
		m.refreshDocs()
		return m, nil

	case "up", "k":
		m.navigateUp()
		m.clampFocusPane()
		m.cmdText, m.cmdTextLong = m.buildCommandStrings()
		m.layoutViewports()
		m.reclampAllScrolls()
		m.refreshDocs()
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
			m.hostKeyTried = false // a fresh run may legitimately hit the host-key modal again
			m.focusPane = m.lastFocusPane
			if m.isInteractiveInvocation() {
				return m, executeHandoff(m.cmdText, false)
			}
			return m, tea.Batch(m.spinner.Tick, executeCommand(m.cmdText))
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

// handleAboutModalKeys drives the read-only about popup. esc closes; every
// other key — including q — is either a no-op or, in q's case, an explicit
// quit (matches `q`'s app-wide semantics; it is never a modal close).
func (m mainModel) handleAboutModalKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.aboutModalOpen = false
		return m, nil
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

// handleHostKeyModalKeys drives the SSH host-key approval popup. `y`/enter
// trusts the shown fingerprint (append to known_hosts, then silently retry the
// push); `n`/esc/q cancels, leaving the original failure in the output pane.
func (m mainModel) handleHostKeyModalKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		p := m.hostKeyPrompt
		m.hostKeyModalOpen = false
		m.backdrop = nil
		m.running = true
		return m, tea.Batch(m.spinner.Tick, addKnownHost(p.keyscanLines, p.pushCmd))
	case "n", "esc", "q", "ctrl+c":
		m.hostKeyModalOpen = false
		m.backdrop = nil
		// Surface the original failure so the user sees why nothing happened.
		lines := commandEchoLines(m.hostKeyPrompt.pushCmd)
		lines = append(lines, "Host key not trusted — push canceled.")
		m.outputLines = lines
		m.output.GotoTop()
		return m, nil
	}
	return m, nil
}

// handleAuthModalKeys drives the "remote needs input" popup shown for
// credential prompts (or when the host key couldn't be scanned). Enter hands
// the command to the real terminal via the pty handoff; esc/n/q cancels.
func (m mainModel) handleAuthModalKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		cmd := m.authModalCmd
		m.authModalOpen = false
		m.backdrop = nil
		m.running = true
		return m, executeHandoff(cmd, true) // plain handoff: a herdr split pane, else clear the screen and take over the terminal
	case "n", "esc", "q", "ctrl+c":
		m.authModalOpen = false
		m.backdrop = nil
		lines := commandEchoLines(m.authModalCmd)
		lines = append(lines, "Canceled.")
		m.outputLines = lines
		m.output.GotoTop()
		return m, nil
	}
	return m, nil
}

// handleGlobalFlagsModalKeys drives the global-options modal (opened with
// "g" from any composer/output/docs/cmdbar pane). Same spatial model as the
// main composer: ←→/h/l pick a column (checklist or docs), ↑↓/j/k move
// within it — navigating flags on the left, scrolling docs on the right.
// Space toggles the highlighted flag; esc (or g again) closes. Flags that
// take a value are edited in the regular input bar after closing — the
// modal itself has no editing mode.
func (m mainModel) handleGlobalFlagsModalKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "g":
		m.globalFlagsModalOpen = false
		m.backdrop = nil
		m.cmdText, m.cmdTextLong = m.buildCommandStrings()
		// Selections may have added/removed input-bar fields — resize now.
		m.layoutViewports()
		m.reclampAllScrolls()
		// A selected flag still needs its value typed: return to the last
		// composer pane (instead of the command bar / output / docs pane the
		// modal was opened from) so the input bar is one "i" away.
		if m.hasEmptyRequiredGlobalInput() && m.focusPane > focusFlags {
			m.focusPane = m.lastFocusPane
		}
		return m, nil
	case "right", "l":
		m.globalModalDocsFocus = true
	case "left", "h":
		m.globalModalDocsFocus = false
	case "down", "j":
		if m.globalModalDocsFocus {
			m.globalDocs.SetYOffset(m.globalDocs.YOffset() + 1)
		} else {
			m.globalFlagIdx = (m.globalFlagIdx + 1) % len(m.globalFlags)
			m.refreshGlobalDocs()
		}
	case "up", "k":
		if m.globalModalDocsFocus {
			if m.globalDocs.YOffset() > 0 {
				m.globalDocs.SetYOffset(m.globalDocs.YOffset() - 1)
			}
		} else {
			m.globalFlagIdx = (m.globalFlagIdx - 1 + len(m.globalFlags)) % len(m.globalFlags)
			m.refreshGlobalDocs()
		}
	case "space":
		if !m.globalModalDocsFocus {
			toggleFlagAt(m.globalFlags, m.globalFlagIdx, m.inputs)
			m.cmdText, m.cmdTextLong = m.buildCommandStrings()
		}
	case "pgdown":
		m.globalDocs.SetYOffset(m.globalDocs.YOffset() + 1)
	case "pgup":
		if m.globalDocs.YOffset() > 0 {
			m.globalDocs.SetYOffset(m.globalDocs.YOffset() - 1)
		}
	}
	return m, nil
}

// globalModalPoint converts terminal coordinates into the global-options
// modal's local coordinates. Events outside the modal are deliberately
// ignored so they cannot interact with the frozen backdrop.
func (m mainModel) globalModalPoint(x, y int) (int, int, bool) {
	modal := m.renderGlobalFlagsModal()
	modalW, modalH := lipgloss.Width(modal), lipgloss.Height(modal)
	originX := (m.width - modalW) / 2
	originY := (m.height - modalH) / 2
	localX, localY := x-originX, y-originY
	return localX, localY, localX >= 0 && localX < modalW && localY >= 0 && localY < modalH
}

func (m mainModel) handleGlobalFlagsModalMouse(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	localX, _, inside := m.globalModalPoint(msg.X, msg.Y)
	if !inside {
		return m, nil
	}

	if localX >= globalListW+4 {
		m.globalModalDocsFocus = true
		if msg.Button == tea.MouseWheelUp && m.globalDocs.YOffset() > 0 {
			m.globalDocs.SetYOffset(m.globalDocs.YOffset() - 1)
		} else if msg.Button == tea.MouseWheelDown {
			m.globalDocs.SetYOffset(m.globalDocs.YOffset() + 1)
		}
		return m, nil
	}

	m.globalModalDocsFocus = false
	if len(m.globalFlags) == 0 {
		return m, nil
	}
	switch msg.Button {
	case tea.MouseWheelUp:
		m.globalFlagIdx = (m.globalFlagIdx - 1 + len(m.globalFlags)) % len(m.globalFlags)
		m.refreshGlobalDocs()
	case tea.MouseWheelDown:
		m.globalFlagIdx = (m.globalFlagIdx + 1) % len(m.globalFlags)
		m.refreshGlobalDocs()
	}
	return m, nil
}

func (m mainModel) handleGlobalFlagsModalClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}
	localX, _, inside := m.globalModalPoint(msg.X, msg.Y)
	if !inside {
		return m, nil
	}

	listPaneW := globalListW + 4
	if localX >= listPaneW {
		m.globalModalDocsFocus = true
		return m, nil
	}

	// Any click in the checklist pane, including a flag row or empty space,
	// only moves focus to that pane. Flag selection remains keyboard-driven.
	m.globalModalDocsFocus = false
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

	// Selected global flags are pinned rows above the scrolling command
	// flags, so flagScroll and flagIdx both index the command flags only;
	// the scroll window is shrunk by however many rows the pins occupy.
	flags := m.currentFlags()
	flagCh := ch - m.stickyGlobalCount()

	// Clamp to flagIdx+1 so the item below the cursor stays visible
	// (cursor sits one row above the bottom edge when a next item exists).
	peekIdx := m.flagIdx
	if m.flagIdx < len(flags)-1 {
		peekIdx++
	}
	m.flagScroll = clampScroll(m.flagScroll, peekIdx, flagCh)

	// Cap scroll so the list stays flush to the bottom — prevents empty
	// lines when the pane grows after untoggling a RequiresInput flag.
	if flagCh > 0 {
		if maxScroll := max(0, len(flags)-flagCh); m.flagScroll > maxScroll {
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
		// flagScroll indexes command flags only; pinned globals shrink the window.
		m.flagScroll = clampScroll(m.flagScroll, m.flagIdx, ch-m.stickyGlobalCount())
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
		// flagScroll indexes command flags only; pinned globals shrink the window.
		m.flagScroll = clampScroll(m.flagScroll, m.flagIdx, ch-m.stickyGlobalCount())
	}
}

// resetFlags deselects every non-mandatory flag in flags and blanks the input
// of any RequiresInput flag. Shared by resetCurrentFlags for command/
// subcommand flags and for m.globalFlags, so global options reset on exactly
// the same triggers as command flags (navigation away, post-execution).
func resetFlags(flags []Flag, inputs map[string]textinput.Model) {
	for i := range flags {
		if !flags[i].Mandatory {
			flags[i].Selected = false
		}
		if flags[i].RequiresInput {
			if ti, ok := inputs[flags[i].Name]; ok {
				ti.SetValue("")
				inputs[flags[i].Name] = ti
			}
		}
	}
}

func (m *mainModel) resetCurrentFlags() {
	resetFlags(m.globalFlags, m.inputs)

	if len(m.categories) == 0 || m.catIdx >= len(m.categories) {
		return
	}
	if m.cmdIdx >= len(m.categories[m.catIdx].Commands) {
		return
	}
	cmd := &m.categories[m.catIdx].Commands[m.cmdIdx]

	if len(cmd.SubCmds) > 0 {
		if m.subIdx < len(cmd.SubCmds) {
			resetFlags(cmd.SubCmds[m.subIdx].Flags, m.inputs)
			for _, a := range cmd.SubCmds[m.subIdx].Args {
				if ti, ok := m.argInputs[a.Name]; ok {
					ti.SetValue("")
					m.argInputs[a.Name] = ti
				}
			}
		}
	} else {
		resetFlags(cmd.Flags, m.inputs)
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
	for _, f := range m.globalFlags {
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
	for i := range m.globalFlags {
		if !m.globalFlags[i].Mandatory {
			m.globalFlags[i].Selected = snap.selectedFlags[m.globalFlags[i].Name]
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
		toggleFlagAt(cmd.SubCmds[m.subIdx].Flags, m.flagIdx, m.inputs)
	} else {
		toggleFlagAt(cmd.Flags, m.flagIdx, m.inputs)
	}
}

// toggleFlagAt toggles flags[idx].Selected (no-op if idx is out of range or
// the flag is Mandatory), lazily creates a textinput.Model in inputs when a
// RequiresInput flag becomes selected, and clears any other flag in the same
// slice that now conflicts with it. Shared by toggleFlag (command/subcommand
// flags) and the global-options modal (m.globalFlags).
func toggleFlagAt(flags []Flag, idx int, inputs map[string]textinput.Model) {
	if len(flags) == 0 || idx >= len(flags) || flags[idx].Mandatory {
		return
	}
	flags[idx].Selected = !flags[idx].Selected
	if !flags[idx].Selected {
		return
	}
	if flags[idx].RequiresInput {
		key := flags[idx].Name
		if _, exists := inputs[key]; !exists {
			ti := textinput.New()
			ti.Prompt = " "
			ti.Placeholder = "Enter " + key + "..."
			ti.CharLimit = 256
			inputs[key] = ti
		}
	}
	for i := range flags {
		if i != idx && flagsConflict(flags[idx], flags[i]) {
			flags[i].Selected = false
			if flags[i].RequiresInput {
				if ti, ok := inputs[flags[i].Name]; ok {
					ti.SetValue("")
					inputs[flags[i].Name] = ti
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
	globalShort, globalLong := m.selectedGlobalFlagParts()

	cmds := m.currentCommands()
	if len(cmds) == 0 || m.cmdIdx >= len(cmds) {
		shortParts := append([]string{"jj"}, globalShort...)
		longParts := append([]string{"jj"}, globalLong...)
		return strings.Join(shortParts, " "), strings.Join(longParts, " ")
	}

	cmd := cmds[m.cmdIdx]

	shortName := cmd.Name
	if cmd.Alias != "" && len(cmd.Alias) < len(cmd.Name) {
		shortName = cmd.Alias
	}
	shortParts := append([]string{"jj"}, globalShort...)
	shortParts = append(shortParts, shortName)
	longParts := append([]string{"jj"}, globalLong...)
	longParts = append(longParts, cmd.Name)

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

	// Positional args come before flags, matching how jj commands are
	// naturally written (e.g. `jj tag set v0.2.0 --to @-`, not the reverse).
	// Optional args are simply omitted when empty, letting jj apply its own
	// default (e.g. @).
	for _, a := range m.currentArgs() {
		if val := m.argInputs[a.Name].Value(); val != "" {
			shortParts = append(shortParts, val)
			longParts = append(longParts, val)
		}
	}

	flags := m.currentFlags()
	for _, f := range flags {
		if f.Selected {
			if f.Value != "" {
				shortParts = append(shortParts, flagExecForm(f))
				longParts = append(longParts, flagBarForm(f))
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

	return strings.Join(shortParts, " "), strings.Join(longParts, " ")
}

// selectedGlobalFlagParts renders the currently-selected global flags (and
// their input values) in exec and bar form. buildCommandStrings splices these
// in right after "jj", before the subcommand, since jj requires global
// options to precede it.
func (m mainModel) selectedGlobalFlagParts() (short, long []string) {
	for _, f := range m.globalFlags {
		if !f.Selected {
			continue
		}
		if f.Value != "" {
			short = append(short, flagExecForm(f))
			long = append(long, flagBarForm(f))
		}
		if f.RequiresInput {
			val := m.inputs[f.Name].Value()
			if val != "" {
				if f.NeedsQuotes {
					val = "\"" + val + "\""
				}
				short = append(short, val)
				long = append(long, val)
			}
		}
	}
	return
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

// paneRows returns the left column's row boundaries: the composer occupies
// [0, composerH), the description/docs bar [composerH, inputsTop), the inputs
// box [inputsTop, bottom), and the command bar starts at bottom. View(),
// layoutViewports() and mouse hit-testing all read from here so the drawn
// layout and the clickable layout can't drift apart — they did, and clicks on
// the inputs box landed on the docs pane.
//
// composerH is a pure function of bottom/maxInputsHt — it never looks at the
// active inputs, so it never resizes during navigation or flag toggling. The
// docs bar absorbs all the slack instead of leaving a gap.
func (m mainModel) paneRows() (composerH, inputsTop, bottom int) {
	bottom = m.height - 6 // 5 cmdBar + 1 helpBar
	if bottom < 6 {
		bottom = 6
	}

	inputsTop = bottom
	if n := len(m.getActiveCombined()); n > 0 {
		inputsTop = bottom - (n + 2) // + border
	}

	composerBudget := bottom - m.maxInputsHt
	if composerBudget < 6 {
		composerBudget = 6
	}
	composerH = composerBudget / 2
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
	composerH, inputsTop, outputHeight := m.paneRows()

	paneContentWidth := rightWidth - 4 - scrollbarW - scrollbarGapW
	paneContentHeight := outputHeight - 4 // title box (3) + content bottom border (1)

	m.output.SetWidth(paneContentWidth)
	m.output.SetHeight(paneContentHeight)

	m.paneH = composerH

	// Docs viewport: fits inside the description bar's content box (title
	// box + border consume 3 rows, handled by docsContentDims).
	docsW, docsH := docsContentDims(leftWidth, inputsTop-composerH)
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

// selectedGlobalCount returns how many global flags are currently selected —
// the number of display-only rows prefixed to the FLAGS pane.
func (m mainModel) selectedGlobalCount() int {
	n := 0
	for _, f := range m.globalFlags {
		if f.Selected {
			n++
		}
	}
	return n
}

// stickyGlobalCount returns how many rows the pinned global-flag header
// actually occupies in the FLAGS pane: the selected count, capped so at least
// one command-flag row always survives (the last pinned row then becomes a
// "◆ +N more" summary — see renderPane). ch mirrors renderPane's own
// contentHeight (m.paneH - 4, both fed by composerBudget/2). Single source of
// truth shared by the view and the scroll math.
func (m mainModel) stickyGlobalCount() int {
	ch := m.paneH - 4
	if ch < 1 {
		ch = 1
	}
	return min(m.selectedGlobalCount(), max(0, ch-1))
}

// hasEmptyRequiredGlobalInput reports whether any selected global flag that
// RequiresInput has an empty value — part of hasIncompleteInputs, blocking
// finalize-to-command-bar the same way empty command-flag inputs do.
func (m mainModel) hasEmptyRequiredGlobalInput() bool {
	for _, f := range m.globalFlags {
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
	return m.hasEmptyRequiredFlagInput() || m.hasEmptyRequiredGlobalInput() || m.hasEmptyRequiredArg() || m.requiredFlagGroupUnsatisfied()
}

// selectedFlagValue returns the trimmed input value of the flag with the given
// Value string, or "" if the flag is not selected (or has no input).
func (m mainModel) selectedFlagValue(value string) string {
	for _, f := range m.currentFlags() {
		if f.Value == value && f.Selected {
			return strings.TrimSpace(m.inputs[f.Name].Value())
		}
	}
	return ""
}

// isFlagSelected reports whether the flag with the given Value string is
// currently selected on the composed command.
func (m mainModel) isFlagSelected(value string) bool {
	for _, f := range m.currentFlags() {
		if f.Value == value && f.Selected {
			return true
		}
	}
	return false
}

// isInteractiveInvocation reports whether the composed command will open an
// editor, diff picker, or merge tool, and must therefore run through the
// tea.ExecProcess TTY handoff instead of CombinedOutput (which would hang).
func (m mainModel) isInteractiveInvocation() bool {
	cmds := m.currentCommands()
	if len(cmds) == 0 || m.cmdIdx >= len(cmds) {
		return false
	}
	cmd := cmds[m.cmdIdx]

	// Statically interactive command or subcommand (diffedit, arrange,
	// config edit, sparse edit, ...).
	if cmd.Interactive {
		return true
	}
	if len(cmd.SubCmds) > 0 && m.subIdx < len(cmd.SubCmds) && cmd.SubCmds[m.subIdx].Interactive {
		return true
	}

	// A selected interactivity-triggering flag (-i on commit/split/squash).
	for _, f := range m.currentFlags() {
		if f.Selected && f.Interactive {
			return true
		}
	}

	// Interactive-unless commands: these fall back to an editor or picker
	// when their non-interactive inputs are absent.
	switch cmd.Name {
	case "describe", "commit":
		// No message → jj opens $EDITOR for one.
		return m.selectedFlagValue("-m") == ""
	case "squash":
		// No -m and no --use-destination-message → jj may open $EDITOR to
		// combine the source and destination descriptions. -k (--keep-emptied)
		// keeps the source revision alive, so jj never needs to combine
		// descriptions and never prompts, regardless of -m.
		return m.selectedFlagValue("-m") == "" && !m.isFlagSelected("--use-destination-message") && !m.isFlagSelected("-k")
	case "resolve":
		// Bare resolve runs the merge tool; --list only prints conflicts.
		return !m.isFlagSelected("--list")
	case "split":
		// Paths select the changes non-interactively, but without -m jj may
		// still open $EDITOR asking for a description.
		return strings.TrimSpace(m.argInputs["FILESETS"].Value()) == "" ||
			m.selectedFlagValue("-m") == ""
	}
	return false
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

// commandEchoLines returns the "❯ cmd args" header line plus a following
// blank line, as shown at the top of the output pane after any command run.
func commandEchoLines(cmdStr string) []string {
	cmdParts := strings.SplitN(cmdStr, " ", 2)
	cmdLine := promptStyle.Render("❯") + " " + promptCmdStyle.Render(cmdParts[0])
	if len(cmdParts) > 1 {
		cmdLine += " " + cmdParts[1]
	}
	return []string{cmdLine, ""}
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

// maxInputsHeight scans every command/subcommand once to find the most args +
// mandatory-input flags any single one has pre-selected at startup, so the
// composer layout can reserve a floor height that absorbs pure-navigation
// jitter without resizing. Deliberately toggling extra optional inputs with
// space can still grow the inputs box past this floor — same as before this
// function existed.
func maxInputsHeight(cats []Category) int {
	maxN := 0
	consider := func(args []Arg, flags []Flag) {
		n := len(args)
		for _, f := range flags {
			if f.Mandatory && f.RequiresInput {
				n++
			}
		}
		if n > maxN {
			maxN = n
		}
	}
	for _, cat := range cats {
		for _, cmd := range cat.Commands {
			if len(cmd.SubCmds) > 0 {
				for _, sub := range cmd.SubCmds {
					consider(sub.Args, sub.Flags)
				}
			} else {
				consider(cmd.Args, cmd.Flags)
			}
		}
	}
	if maxN == 0 {
		return 0
	}
	return maxN + 2 // border/padding, matches the existing len(combined)+2
}

func (m *mainModel) getActiveInputs() []*Flag {
	var active []*Flag
	if len(m.categories) > 0 && m.catIdx < len(m.categories) {
		cat := &m.categories[m.catIdx]
		if len(cat.Commands) > 0 && m.cmdIdx < len(cat.Commands) {
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
		}
	}
	// Selected global flags that take a value are edited in the same input
	// bar as command inputs — the global-options modal has no editing mode.
	for i := range m.globalFlags {
		if m.globalFlags[i].Selected && m.globalFlags[i].RequiresInput {
			active = append(active, &m.globalFlags[i])
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
		label := a.Name
		if i := strings.IndexByte(label, '['); i != -1 {
			label = label[:i]
		}
		item := InputItem{Name: a.Name, Label: label, IsArg: true}
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

// focusInput moves focus into the inputs box at row idx, blurring whichever
// field was focused before. idx is clamped, so callers can hand it a raw row
// number. Returns nil (and changes nothing) when there is nothing to fill in.
func (m *mainModel) focusInput(idx int) tea.Cmd {
	combined := m.getActiveCombined()
	if len(combined) == 0 {
		return nil
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(combined) {
		idx = len(combined) - 1
	}

	if m.focusPane == focusInputs {
		old := combined[min(m.focusedInputIdx, len(combined)-1)]
		ti := m.getInput(old)
		ti.Blur()
		m.setInput(old, ti)
	} else if m.focusPane <= focusFlags {
		m.lastFocusPane = m.focusPane
	}

	m.focusPane = focusInputs
	m.focusedInputIdx = idx
	item := combined[idx]
	ti := m.getInput(item)
	cmd := ti.Focus()
	m.setInput(item, ti)
	return cmd
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
	m.setDocsContent(m.docsContent())
}

// refreshGlobalDocs rebuilds the modal's docs column for the highlighted
// global flag — same header + full-description layout the main docs pane
// uses for flags in docsContent().
func (m *mainModel) refreshGlobalDocs() {
	f := m.globalFlags[m.globalFlagIdx]
	content := flagNameStyle.Render(flagDocsLabel(f))
	if f.Description != "" {
		content += "\n\n" + f.Description
	}
	m.globalDocs.SetContent(wordWrap(content, m.globalDocs.Width()))
	m.globalDocs.GotoTop()
}

// setDocsContent wraps and applies pre-computed docs text at the viewport's
// current width. Split out from refreshDocs so callers that must resize the
// viewport (e.g. entering enlarge) before the focus pane changes can capture
// docsContent() against the old focus, then apply it once the new width is set.
func (m *mainModel) setDocsContent(content string) {
	m.docs.SetYOffset(0)
	width := m.docs.Width()
	if width < 10 {
		width = 78 // fallback before first WindowSizeMsg
	}
	m.docs.SetContent(wordWrap(content, width))
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
			header := flagNameStyle.Render(flagDocsLabel(f))
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

// flagLongForm returns the canonical long spelling ("--insert-after").
func flagLongForm(f Flag) string {
	if strings.HasPrefix(f.Value, "--") {
		return f.Value
	}
	return "--" + f.Name
}

// flagBarForm returns the spelling shown in the command bar and copied to the
// clipboard: the shortest full-word form. The alias wins only when it beats
// the long name ("--after" over "--insert-after") — some aliases are longer
// ("--destination" for "--onto") and lose. NoBarAlias opts a flag out of this
// entirely, so its alias still appears in flagDocsLabel but never in the bar.
func flagBarForm(f Flag) string {
	long := flagLongForm(f)
	if !f.NoBarAlias && f.Alias != "" && len(f.Alias)+2 < len(long) {
		return "--" + f.Alias
	}
	return long
}

// flagExecForm returns the spelling used in the executed command: the
// shortest of all — the short flag when one exists ("-A"), else whatever
// flagBarForm picked.
func flagExecForm(f Flag) string {
	bar := flagBarForm(f)
	if f.Value != "" && len(f.Value) < len(bar) {
		return f.Value
	}
	return bar
}

// flagDocsLabel renders a flag as jj --help shows it:
// "-m, --message <MESSAGE>" or "-A, --insert-after <REVSETS> [aliases: --after]".
func flagDocsLabel(f Flag) string {
	label := f.Value
	if len(f.Value) == 2 && f.Value[0] == '-' && f.Value[1] != '-' {
		label = f.Value + ", --" + f.Name
	}
	if f.InputType != "" {
		label += " <" + f.InputType + ">"
	}
	if f.Alias != "" {
		label += " [aliases: --" + f.Alias + "]"
	}
	return label
}

func buildDocsBlock(name, usagePath, alias, desc string, args []Arg, flags []Flag, requiredUsage string, subCmds []SubCommand) string {
	var b strings.Builder

	// Build usage synopsis: jj <path> [OPTIONS] <required flags> [ARG[...]]
	// jj always shows [OPTIONS] (global options apply to every command), and
	// required-flag/group syntax (e.g. "<--onto|--insert-after|--insert-before>")
	// is sourced verbatim from `jj util markdown-help` by gen-descriptions,
	// since Jutsu's own Mandatory flag doesn't reliably mean "jj requires this".
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
				b.WriteString("\n" + indentBlock(a.Description))
			}
		}
	}
	if len(flags) > 0 {
		b.WriteString("\n\n" + headerStyle.Render("Options"))
		for i, f := range flags {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString("\n  " + flagNameStyle.Render(flagDocsLabel(f)))
			if f.Description != "" {
				b.WriteString("\n" + indentBlock(f.Description))
			}
		}
	}
	return b.String()
}

// indentMarker flags a prose line for wordWrap to wrap-and-indent under its
// heading. A literal leading space is already claimed by wordWrap's
// preformatted-diagram passthrough below, so a non-printable marker is used
// instead of indenting here directly.
const indentMarker = "\x01"

// indentBlock marks every prose line of a multi-paragraph description so
// each paragraph — not just the first — ends up indented under its flag or
// argument name once wordWrap runs. Blank separator lines and
// already-indented preformatted (diagram) lines pass through untouched.
func indentBlock(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		lines[i] = indentMarker + line
	}
	return strings.Join(lines, "\n")
}

// wordWrap wraps s so no visible line exceeds width characters. It preserves
// existing newlines and handles paragraph breaks (blank lines). Prose lines
// have their whitespace normalized to single spaces — this must apply
// regardless of whether the line needs to wrap, so short and long lines, and
// normal vs. enlarged panes, render identically. Lines starting with
// whitespace are preformatted (fenced diagram/code blocks, emitted indented
// by gen-descriptions): they pass through verbatim, hard-truncated at width
// so they can never wrap and grow a fixed-size box.
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
		// Already-styled lines (e.g. rendered headers) pass through untouched.
		if strings.HasPrefix(line, "\x1b") {
			out.WriteString(line)
			continue
		}
		// Preformatted lines: verbatim, truncated instead of wrapped.
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if r := []rune(line); len(r) > width {
				line = string(r[:width])
			}
			out.WriteString(line)
			continue
		}
		// Word-wrap at width, collapsing whitespace to single spaces. A
		// marked line (indentBlock) wraps at width-2 and indents every
		// resulting line, so continuation paragraphs line up under the
		// same heading as the first one instead of drifting to column 0.
		indent := ""
		avail := width
		if rest, ok := strings.CutPrefix(line, indentMarker); ok {
			line = rest
			indent = "  "
			if avail = width - 2; avail < 1 {
				avail = 1
			}
		}
		words := strings.Fields(line)
		col := 0
		for wi, w := range words {
			wlen := len([]rune(w))
			if wi == 0 {
				out.WriteString(indent + w)
				col = wlen
			} else if col+1+wlen > avail {
				out.WriteByte('\n')
				out.WriteString(indent + w)
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
