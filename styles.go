package main

import "charm.land/lipgloss/v2"

var (
	// Catppuccin Mocha Palette
	colorRosewater = lipgloss.Color("#f5e0dc")
	colorFlamingo  = lipgloss.Color("#f2cdcd")
	colorPink      = lipgloss.Color("#f5c2e7")
	colorMauve     = lipgloss.Color("#cba6f7")
	colorRed       = lipgloss.Color("#f38ba8")
	colorMaroon    = lipgloss.Color("#eba0ac")
	colorPeach     = lipgloss.Color("#fab387")
	colorYellow    = lipgloss.Color("#f9e2af")
	colorGreen     = lipgloss.Color("#a6e3a1")
	colorTeal      = lipgloss.Color("#94e2d5")
	colorSky       = lipgloss.Color("#89dceb")
	colorSapphire  = lipgloss.Color("#74c7ec")
	colorBlue      = lipgloss.Color("#89b4fa")
	colorLavender  = lipgloss.Color("#b4befe")
	colorText      = lipgloss.Color("#cdd6f4")
	colorSubtext1  = lipgloss.Color("#bac2de")
	colorSubtext   = lipgloss.Color("#a6adc8") // Subtext0
	colorOverlay2  = lipgloss.Color("#9399b2")
	colorOverlay1  = lipgloss.Color("#7f849c")
	colorOverlay   = lipgloss.Color("#6c7086") // Overlay0
	colorSurface2  = lipgloss.Color("#585b70")
	colorSurface1  = lipgloss.Color("#45475a")
	colorSurface0  = lipgloss.Color("#313244")
	colorBase      = lipgloss.Color("#1e1e2e")
	colorMantle    = lipgloss.Color("#181825")
	colorCrust     = lipgloss.Color("#11111b")

	paneTopBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "├",
		BottomRight: "┤",
	}

	paneBottomBorder = lipgloss.Border{
		Top:         "",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "",
		TopRight:    "",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	paneTopBorderThick = lipgloss.Border{
		Top:         "━",
		Bottom:      "━",
		Left:        "┃",
		Right:       "┃",
		TopLeft:     "┏",
		TopRight:    "┓",
		BottomLeft:  "┣",
		BottomRight: "┫",
	}

	paneBottomBorderThick = lipgloss.Border{
		Top:         "",
		Bottom:      "━",
		Left:        "┃",
		Right:       "┃",
		TopLeft:     "",
		TopRight:    "",
		BottomLeft:  "┗",
		BottomRight: "┛",
	}

	cmdBarBorderThick = lipgloss.Border{
		Top:         "━",
		Bottom:      "━",
		Left:        "┃",
		Right:       "┃",
		TopLeft:     "┏",
		TopRight:    "┓",
		BottomLeft:  "┗",
		BottomRight: "┛",
	}

	activeTitleBorderStyle = lipgloss.NewStyle().
				Border(paneTopBorderThick).
				BorderForeground(colorPeach).
				PaddingLeft(1).
				PaddingRight(1)

	activeContentBorderStyle = lipgloss.NewStyle().
					Border(paneBottomBorderThick, false, true, true, true).
					BorderForeground(colorPeach).
					PaddingLeft(1).
					PaddingRight(1)

	inactiveTitleBorderStyle = lipgloss.NewStyle().
					Border(paneTopBorder).
					BorderForeground(colorLavender).
					PaddingLeft(1).
					PaddingRight(1)

	inactiveContentBorderStyle = lipgloss.NewStyle().
					Border(paneBottomBorder, false, true, true, true).
					BorderForeground(colorLavender).
					PaddingLeft(1).
					PaddingRight(1)

	commandBarActiveStyle = lipgloss.NewStyle().
				BorderStyle(cmdBarBorderThick).
				BorderForeground(colorGreen).
				PaddingLeft(1).
				PaddingRight(1)

	commandBarInactiveStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(colorLavender).
				PaddingLeft(1).
				PaddingRight(1)

	headerStyle = lipgloss.NewStyle().
			Foreground(colorMauve).
			Bold(true)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorCrust).
				Background(colorPeach).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(colorText)

	dimItemStyle = lipgloss.NewStyle().
			Foreground(colorOverlay)

	flagSelectedStyle = lipgloss.NewStyle().
				Foreground(colorText)

	// Mandatory flags use sapphire so they're visually distinct from toggleable selected flags.
	flagMandatoryStyle = lipgloss.NewStyle().
				Foreground(colorCrust).
				Background(colorSapphire).
				Bold(true)

	flagUnselectedStyle = lipgloss.NewStyle().
				Foreground(colorOverlay)

	// Flags blocked by a conflicting selection: pinkish-red, clearly distinct from grey unselected.
	flagConflictedStyle = lipgloss.NewStyle().
				Foreground(colorMaroon)

	// Cursor landed on a conflicted flag: maroon background, same pattern as selected/mandatory.
	flagConflictedFocusedStyle = lipgloss.NewStyle().
					Foreground(colorCrust).
					Background(colorMaroon).
					Bold(true)

	commandTextStyle = lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true)

	outputStyle = lipgloss.NewStyle().
			Foreground(colorText)

	promptStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	promptCmdStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorCrust).
			Background(colorSurface2).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorSubtext1)

	helpBarStyle = lipgloss.NewStyle().
			Background(colorMantle).
			PaddingLeft(1)

	descriptionBarStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorOverlay).
				PaddingLeft(1).
				PaddingRight(1).
				Foreground(colorSubtext1)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	runningStyle = lipgloss.NewStyle().
			Foreground(colorMauve).
			Italic(true)
)

const (
	focusCategories = iota
	focusCommands
	focusSubcmds
	focusFlags
	focusOutput
	focusInputs
	focusCmdBar
	focusDocs
)

var (
	activeDescriptionBarStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(colorPeach).
					PaddingLeft(1).
					PaddingRight(1).
					Foreground(colorSubtext1)

	boldStyle = lipgloss.NewStyle().Bold(true)

	versionGreenStyle = lipgloss.NewStyle().
				Foreground(colorGreen)

	versionYellowStyle = lipgloss.NewStyle().
				Foreground(colorYellow)

	versionRedStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	copyFlashStyle = lipgloss.NewStyle().
			Foreground(colorCrust).
			Background(colorGreen).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)
)
