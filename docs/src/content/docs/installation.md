---
title: Installation
description: Complete guide to installing, configuring, and troubleshooting Jutsu.
---

Get Jutsu up and running on your system. Before installing, ensure you meet the system prerequisites.

---

## Prerequisites

To run Jutsu, your system must meet the following requirements:

*   **Go 1.25+** — [Install Go](https://go.dev/doc/install) if you plan to build from source or install via Go tools.
*   **Jujutsu (`jj`) CLI** — [Install `jj`](https://jj-vcs.github.io/jj/latest/install-and-setup/) version control system, which Jutsu composes commands for.
*   **Terminal Emulator** — A modern terminal emulator with:
    *   256 color support
    *   Unicode / UTF-8 encoding
    *   Minimum width of 80 columns

:::tip[Tested Terminals]
Jutsu is tested on modern terminals, including **Alacritty, Ghostty, iTerm2, Kitty, WezTerm, GNOME Terminal, Konsole**, and within **tmux/screen** multiplexer sessions.
:::

---

## Installation Methods

Choose the installation method that best fits your workflow and operating system.

### 1. Quick Install (Linux/macOS)

For most Unix-like environments, you can download and install the latest compiled release with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/AliQ80/jutsu/main/install.sh | sh
```

:::note
This script automatically detects your operating system and architecture, downloads the correct release binary, verifies its checksum, and installs it to `~/.local/bin`. 
To customize the installation location, override the `INSTALL_DIR` environment variable:
```bash
curl -fsSL https://raw.githubusercontent.com/AliQ80/jutsu/main/install.sh | INSTALL_DIR=/usr/local/bin sh
```
:::

### 2. Download from GitHub Releases

If you are on Windows or prefer downloading compiled binaries manually:

1. Navigate to the [Jutsu Releases page](https://github.com/AliQ80/jutsu/releases) and download the archive matching your OS/architecture (`.tar.gz` for Linux/macOS, `.zip` for Windows).
2. Extract the archive and enter the extracted folder.
3. Choose one of the setup options below:

#### Option A: Automatic Symlinking with `binmy` (Recommended)
If you have [binmy](https://github.com/AliQ80/binmy) installed, run:
```bash
binmy jutsu
```
This symlinks the `jutsu` binary into `~/.local/bin`, marks it executable, and ensures that directory is added to your `$PATH`.

#### Option B: Manual Installation
Move the executable binary to a folder in your system `$PATH` and ensure it has execution permissions:
```bash
chmod +x jutsu
mv jutsu ~/.local/bin/
```

### 3. Installation via Go

If you have Go configured on your path, install the latest package using `go install`:

```bash
go install github.com/AliQ80/jutsu@latest
```

---

## Building from Source

For development or compiling directly from the latest codebase:

```bash
# 1. Clone the repository
git clone https://github.com/AliQ80/jutsu.git
cd jutsu

# 2. Fetch and tidy dependencies
go mod tidy

# 3. Compile the binary
go build -o jutsu .

# 4. Launch the interface
./jutsu
```

---

## Troubleshooting

### `jj` Command Not Found

Jutsu relies entirely on having the Jujutsu (`jj`) command-line tool accessible in your environment. If Jutsu fails to detect it, verify its availability:

```bash
which jj
jj --version
```

If it is not present, follow the [official Jujutsu installation guide](https://jj-vcs.github.io/jj/latest/install-and-setup/).

### Terminal Compatibility Issues

If the TUI characters appear corrupted, colors are off, or elements are misaligned, check the following:

*   **Unicode Support:** Ensure your terminal environment is set to UTF-8. Run `locale` in your shell and check that `LANG` contains `UTF-8` (e.g., `en_US.UTF-8`).
*   **Color Profile:** Ensure your terminal supports 256 colors. Try running `echo $TERM`—it should report `xterm-256color`, `screen-256color`, or similar.
*   **Window Constraints:** Jutsu layout is optimized for interactive navigation and requires a terminal width of at least 80 columns. Expand your terminal window if elements overlap.

### Go Build Errors

If compilation fails during Go build, check that your Go toolchain matches requirements:

```bash
go version
```

Ensure you are using **Go 1.25+**. You can also clean and refresh dependencies by running:

```bash
go mod tidy
go get -u
```
