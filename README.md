# Jutsu — JJ TUI Command Composer

A sleek, modern Terminal User Interface for composing and executing Jujutsu (`jj`) commands. Built with the Charmbracelet ecosystem, featuring a superfile-inspired design with Catppuccin Mocha theming.

![Jutsu TUI](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-GPLv3-blue?style=flat-square)

## Features

- **Multi-pane command composer** with categories, commands, subcommands, and flags
- **Real-time command building** with visual feedback
- **Async command execution** with streaming output
- **Vim-style navigation** (hjkl) plus arrow keys
- **Catppuccin Mocha theme** with active/inactive panel highlighting
- **Responsive layout** that adapts to terminal size
- **Comprehensive jj command database** covering all major operations

## Prerequisites

- **Go 1.25+** — [Install Go](https://go.dev/doc/install)
- **Jujutsu (`jj`) CLI** — [Install jj](https://jj-vcs.github.io/jj/latest/install-and-setup/)
- A terminal that supports 256 colors and Unicode

## Installation

### Build from Source

```bash
# Clone the repository
git clone <your-repo-url>
cd jutsu

# Install dependencies
go mod tidy

# Build the binary
go build -o jutsu .

# Run the TUI
./jutsu
```

### Quick Start

```bash
# After building, simply run:
./jutsu

# Or install to your Go bin directory:
go install .
```

## Usage

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down in current pane |
| `k` / `↑` | Move up in current pane |
| `l` / `→` | Move focus to right pane |
| `h` / `←` | Move focus to left pane |
| `Space` | Toggle flag selection (in Flags pane) |
| `Tab` | Focus the Command Bar |
| `Esc` | Exit current pane (Command Bar / Output / Input) back to composer |
| `o` | Focus the Output pane |
| `Enter` | Execute command (in Command Bar only) |
| `q` / `Ctrl+C` | Quit (disabled in Input pane) |

### Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│ CATEGORIES │ COMMANDS │ SUB-CMDS │ FLAGS   │ OUTPUT CONSOLE        │
│            │          │          │         │                       │
│ > History  │ log      │ N/A      │ [x] -p  │                       │
│   Working  │ show     │          │ [ ] -T  │ $ jj log -p           │
│   Rewrite  │ diff     │          │ [ ] -s  │ * commit abc123       │
│            │          │          │         │ | Author: ...         │
├─────────────────────────────────────────────────────────────────────┤
│                     jj log -p                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Panes

1. **Categories** — High-level command groups (Working Copy, History, Rewrite, Collaboration, etc.)
2. **Commands** — Primary jj commands within the selected category
3. **Subcommands** — Nested subcommands (e.g., `git fetch`, `bookmark list`)
4. **Flags** — Command-line options that can be toggled on/off
5. **Output Console** — Displays command execution results with scrollable viewport
6. **Command Bar** — Shows the composed command in real-time; press Enter to execute

### Command Execution

1. Navigate to your desired category, command, and subcommand
2. Toggle any flags you want to include
3. Press `Tab` to focus the Command Bar
4. Press `Enter` to execute the command
5. Output appears in the Output Console on the right

## Command Categories

### View
- `log`, `show`, `diff`, `evolog`, `interdiff`, `status`
- `file` (list, show, annotate, search)

### Change
- `commit`, `describe`, `new`, `edit`, `next`, `prev`
- `file` (track, untrack, chmod)
- `fix`, `restore`, `resolve`

### Rewrite
- `squash`, `split`, `rebase`, `abandon`, `absorb`
- `duplicate`, `parallelize`, `revert`, `simplify-parents`

### Sync
- `git` (fetch, push, clone, init, import, export)
- `git remote` (add, list, remove, rename)
- `bookmark` (create, delete, list, move, rename, set, track, untrack, forget, advance)
- `tag` (create, set, list, delete)
- `workspace` (add, forget, list, rename, root, update-stale)

### Undo
- `undo`, `redo`
- `operation` (log, show, diff, undo, abandon, revert)

### Advanced
- `config` (edit, get, list, path, set, unset)
- `root`, `sign`, `unsign`
- `sparse` (list, reset, set)
- `util` (gc, completion bash/zsh/fish)

> **Note**: Flags marked `[*]` are mandatory and cannot be deselected (e.g., `-m` on `describe` and `commit`). Flags marked `[-]` are blocked by a conflicting selection — pressing Space on one auto-deselects the other, so no manual cleanup is needed.

## Troubleshooting

### `jj` command not found

Ensure `jj` is installed and in your `$PATH`:

```bash
which jj
jj --version
```

If not installed, follow the [official installation guide](https://jj-vcs.github.io/jj/latest/install-and-setup/).

### Terminal compatibility issues

Jutsu requires:
- 256 color support
- Unicode/UTF-8 encoding
- Terminal width of at least 80 columns

Tested terminals:
- iTerm2, Alacritty, Kitty, WezTerm, Ghostty
- GNOME Terminal, Konsole
- tmux/screen sessions

### Build errors

Ensure you're using Go 1.25+:

```bash
go version
```

Update dependencies:

```bash
go mod tidy
go get -u
```

## Architecture

Jutsu is built with:
- **Bubble Tea v2** — TUI framework (Model/Update/View pattern)
- **Lip Gloss v2** — Styling and layout
- **Bubbles v2** — Viewport component for output scrolling

### File Structure

```
jutsu/
├── main.go              # Entry point, tea.Program setup
├── model.go             # Core model, Init(), Update() with key routing
├── view.go              # View() — layout composition with lipgloss
├── styles.go            # Lipgloss style definitions (Catppuccin theme)
├── jj_commands.go       # Command database (categories, commands, flags)
├── commands_exec.go     # Async command execution via tea.Cmd
├── README.md            # This file
├── go.mod
└── go.sum
```

## Development

### Running in Development Mode

```bash
go run .
```

### Adding New Commands

Edit `jj_commands.go` and add entries to the appropriate category:

```go
{
    Name:        "new-command",
    Description: "Does something useful",
    Flags: []Flag{
        {Name: "message", Description: "Commit message", Value: "-m",
         RequiresInput: true, NeedsQuotes: true, Mandatory: true},
        {Name: "revision", Description: "Target revision", Value: "-r",
         RequiresInput: true},
        {Name: "from", Description: "Source revision", Value: "--from",
         RequiresInput: true, ConflictingFlags: []string{"-r"}},
    },
}
```

Set `Mandatory: true` on flags that must always be present (e.g. `-m` on commands that would otherwise open `$EDITOR`). Mandatory flags show with `[*]` and sapphire styling and cannot be deselected.

Set `ConflictingFlags` to a slice of `Value` strings for flags that cannot be used together. The TUI enforces this automatically — selecting one flag deselects any conflicting ones. By convention both sides declare the conflict, but only one side is required.

### Customizing Theme

Edit `styles.go` to modify the Catppuccin color palette or adjust border styles.

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Test your changes
4. Submit a pull request

## License

GPLv3 License — see LICENSE file for details

## Acknowledgments

- [Jujutsu VCS](https://jj-vcs.github.io/jj/) — The amazing version control system
- [Charmbracelet](https://charmbracelet.github.io/) — For the incredible TUI ecosystem
- [Catppuccin](https://catppuccin.com/) — For the beautiful color palette
- [Superfile](https://github.com/yorukot/superfile) — Design inspiration

---

**Built with ♥ using Go and Charmbracelet**
