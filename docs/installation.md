# Installation

## Prerequisites

- **Go 1.25+** — [Install Go](https://go.dev/doc/install)
- **Jujutsu (`jj`) CLI** — [Install jj](https://jj-vcs.github.io/jj/latest/install-and-setup/)
- A terminal that supports 256 colors and Unicode

## Quick Install (Linux/macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/AliQ80/jutsu/main/install.sh | sh
```

Downloads the latest release for your OS/arch, verifies its checksum, and
installs it to `~/.local/bin` (override with `INSTALL_DIR=...`). Windows
users: see [Download from Releases](#download-from-releases) below.

## Download from Releases

1. Grab the archive matching your OS/arch from the
   [Releases page](https://github.com/AliQ80/jutsu/releases) (`.tar.gz` for
   Linux/macOS, `.zip` for Windows), extract it, and `cd` into the
   extracted folder.
2. **With [binmy](https://github.com/AliQ80/binmy)** (recommended if you have it):
   ```bash
   binmy jutsu
   ```
   This simply symlinks `jutsu` into `~/.local/bin`, marks it executable, and adds that
   directory to your `$PATH` if it isn't already there. or alternatively do it manually.
   
3. **Manual alternative** (no binmy):
   ```bash
   chmod +x jutsu
   mv jutsu ~/.local/bin/
   ```

## Go Install

```bash
go install github.com/AliQ80/jutsu@latest
```

## Build from Source

```bash
# Clone the repository
git clone https://github.com/AliQ80/jutsu.git
cd jutsu

# Install dependencies
go mod tidy

# Build the binary
go build -o jutsu .

# Run the TUI
./jutsu
```

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
