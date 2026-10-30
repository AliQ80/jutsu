---
title: Installation
description: Complete guide to installing, configuring, and troubleshooting Jutsu.
---

Get Jutsu up and running on your system. Before installing, ensure you meet the system prerequisites.

---

## Prerequisites

To run Jutsu, your system must meet the following requirements:

*   **Jujutsu (`jj`) CLI** — [Install `jj`](https://jj-vcs.github.io/jj/latest/install-and-setup/) version control system, which Jutsu composes commands for.
*   **Terminal Emulator** — A modern terminal emulator with:
    *   256 color support
    *   Unicode / UTF-8 encoding
    *   Minimum width of 80 columns

:::tip[Tested Terminals]
Jutsu is tested on modern terminals, including **Alacritty, Ghostty, iTerm2, Kitty, WezTerm, GNOME Terminal, Konsole**, and within **tmux/screen** multiplexer sessions.
:::

:::note
* **Go 1.25+** — [Install Go](https://go.dev/doc/install) if you plan to build from source or install via Go tools.
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

### 2. Quick Install (Windows)

Run this command in **Windows PowerShell 5.1 or PowerShell 7**:

```powershell
irm https://raw.githubusercontent.com/AliQ80/jutsu/main/install.ps1 | iex
```

The installer detects your native Windows architecture (**amd64 or arm64**), downloads the latest release ZIP, verifies its SHA-256 against the release's `checksums.txt`, and installs `jutsu.exe` into **`%LOCALAPPDATA%\Programs\jutsu`**. No administrator privileges or Go installation are needed. Install [jj](https://jj-vcs.github.io/jj/latest/install-and-setup/) separately if it is not already available.

The installation directory is added to your **user PATH** and your current PowerShell session. New terminal processes inherit the updated user PATH; an already-open terminal application may need to be fully restarted. Run `jutsu` to launch it.

**To upgrade:** close Jutsu, then rerun the same command. The installer replaces the executable after verifying the download and avoids adding duplicate PATH entries.

#### Download the script and choose a version or directory

The quick command downloads and immediately runs the script. To inspect it first or pass parameters, download it as a file:

```powershell
Invoke-WebRequest -UseBasicParsing -Uri https://raw.githubusercontent.com/AliQ80/jutsu/main/install.ps1 -OutFile .\install.ps1
Get-Content .\install.ps1
# After reviewing the script:
.\install.ps1
```

Optional parameters accept an **exact release tag** and a destination directory (including paths with spaces):

```powershell
# Replace v1.2.3 with a tag from the Releases page.
.\install.ps1 -Version v1.2.3 -InstallDir "$env:LOCALAPPDATA\My Tools\jutsu"
```

Checksums detect corrupted downloads; they do not independently authenticate the publisher. Download the script and assets only from the project's trusted repository.

### 3. Download from GitHub Releases

If you prefer downloading compiled binaries manually:

1. Navigate to the [Jutsu Releases page](https://github.com/AliQ80/jutsu/releases) and download the archive matching your OS/architecture (`.tar.gz` for Linux/macOS, `.zip` for Windows).
2. Extract the archive and enter the extracted folder.
3. Choose the setup instructions for your platform below.

#### Linux/macOS: Automatic Symlinking with `binmy`
If you have [binmy](https://github.com/AliQ80/binmy) installed, run:
```bash
binmy jutsu
```
This symlinks the `jutsu` binary into `~/.local/bin`, marks it executable, and ensures that directory is added to your `$PATH`.

#### Linux/macOS: Manual Installation
Move the executable binary to a folder in your system `$PATH` and ensure it has execution permissions:
```bash
chmod +x jutsu
mv jutsu ~/.local/bin/
```

#### Windows: Manual Installation

After extracting the Windows ZIP for your architecture, open PowerShell in the extracted folder:

```powershell
$installDir = Join-Path $env:LOCALAPPDATA 'Programs\jutsu'
New-Item -ItemType Directory -Path $installDir -Force | Out-Null
Copy-Item .\jutsu.exe -Destination (Join-Path $installDir 'jutsu.exe')
& (Join-Path $installDir 'jutsu.exe')
```

To launch with just `jutsu`, search Windows for **“Edit environment variables for your account”**, edit **Path** under **User variables**, and add the full installation directory (expand `%LOCALAPPDATA%` to your user's actual directory). Reopen your terminal application. Windows does not require `chmod` or `binmy`.

### 4. Installation via Go

If you have Go configured on your path, install the latest package using `go install`:

```bash
go install github.com/AliQ80/jutsu@latest
```

Go places the executable in `go env GOBIN`, or in the `bin` directory of `go env GOPATH` when GOBIN is unset. Ensure that directory is on PATH. On Windows, inspect these settings in PowerShell and add the resulting directory to your **user Path** using the environment-variable settings described above:

```powershell
go env GOBIN
go env GOPATH
# With the default GOPATH, the binary directory is usually:
# C:\Users\<you>\go\bin
```

---

## Updating from inside Jutsu

For direct binary installations on Linux, macOS, and Windows, open **About** with `?` when Jutsu announces a newer release, then press **`u` — Update and restart**. This downloads the release for your operating system and architecture, verifies its SHA-256 checksum, and replaces the installed executable. No Go installation is required.

About shows a download progress bar with percentage and downloaded/total size, followed by verification and preparation status. If the server does not provide a total size, the bar shows an indeterminate indicator and the number of bytes downloaded. Press `esc` to cancel and close About, or `q` to cancel and quit, before installation starts. Cancellation is unavailable during installation and restart. Wait for any running command or interactive editor to finish before updating.

Jutsu restarts in the same working directory with the same application arguments and environment. Your unexecuted composed command and viewport position are not restored. Custom installation locations and invocation symlinks are supported; the actual executable is updated in place.

If the installation directory is unwritable, Jutsu stays open and shows an error. It does not automatically request administrator access. Use the installer with suitable permissions or choose a writable user installation instead. Package-manager integrations are deferred; use the package manager's own upgrade process for managed installations.

### Failed or interrupted updates

Download and verification failures leave the installed binary unchanged. About displays the failure and lets you retry with `u`. Installation or restart-launch failures attempt to restore the previous binary. On Windows, replacement happens through a helper after Jutsu exits; failures at that stage are printed in the terminal with recovery instructions. Close other Jutsu instances if Windows reports that the executable is locked.

If automatic restoration fails, keep the recovery executable at the path printed in the error. Relaunch the restored executable using its full path, or reinstall a known working release. Pin the desired exact tag with the existing installers:

```bash
# Linux/macOS: substitute an actual published release tag.
curl -fsSL https://raw.githubusercontent.com/AliQ80/jutsu/main/install.sh | VERSION=v1.2.3 sh
# Include INSTALL_DIR=/your/current/directory before sh for a custom location.
```

```powershell
# Windows: download install.ps1 as described above, then choose an exact tag.
.\install.ps1 -Version v1.2.3
# Include -InstallDir 'C:\Your\Current Directory' for a custom location.
```

Older versions without the update action must first be upgraded using the installer. Checksums detect download corruption; the script, checksum manifest, and release assets share trust in the project's GitHub repository.

---

## Building from Source

For development or compiling directly from the latest codebase. **Linux/macOS:**

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

**Windows (PowerShell):**

```powershell
git clone https://github.com/AliQ80/jutsu.git
cd jutsu
go mod tidy
go build -o jutsu.exe .
.\jutsu.exe
```

To run the source build outside this folder, move `jutsu.exe` into a directory on your user PATH, as described in manual installation.

---

## Troubleshooting

### PowerShell script execution is disabled

If Windows blocks a downloaded `install.ps1`, review the file first. For a script you trust, you can run it in a new Windows PowerShell process with a policy override limited to that process:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\install.ps1
```

This does not change your persistent execution policy. Organization-managed policies can still block scripts; follow your organization's policy or use manual ZIP installation. A new process cannot update the parent shell's PATH, so reopen your terminal afterward.

### Windows executable is in use

Close running Jutsu instances and retry the installer. A failed replacement preserves the old executable. If it still fails, check that your account can write to the selected installation directory.

### Windows PATH or command resolution

If the installer reports that the executable was installed but PATH setup failed, add its reported directory to your **user Path** manually through Windows environment-variable settings, then fully reopen your terminal application.

Check which executable PowerShell resolves:

```powershell
Get-Command jutsu -All
```

If an older copy appears first, remove that copy or adjust your user PATH ordering. You can always launch the installed file directly:

```powershell
& "$env:LOCALAPPDATA\Programs\jutsu\jutsu.exe"
```

Use your chosen directory instead if you supplied `-InstallDir`.

### `jj` Command Not Found

Jutsu relies entirely on having the Jujutsu (`jj`) command-line tool accessible in your environment. If Jutsu fails to detect it, verify its availability:

```bash
which jj
jj --version
```

On Windows, use PowerShell instead:

```powershell
Get-Command jj
jj --version
```

If it is not present, follow the [official Jujutsu installation guide](https://jj-vcs.github.io/jj/latest/install-and-setup/).

### Terminal Compatibility Issues

If the TUI characters appear corrupted, colors are off, or elements are misaligned, check the following:

*   **Unicode Support:** Ensure your terminal environment is set to UTF-8. On Linux/macOS, run `locale` in your shell and check that `LANG` contains `UTF-8` (e.g., `en_US.UTF-8`).
*   **Color Profile:** Ensure your terminal supports 256 colors. On Linux/macOS, try running `echo $TERM`—it should report `xterm-256color`, `screen-256color`, or similar.
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
