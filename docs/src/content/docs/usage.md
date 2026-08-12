---
title: Usage Guide
description: Learn how to navigate Jutsu, compose commands, toggle flags, and interact with your Jujutsu repositories.
---

Jutsu provides an interactive, keyboard-driven interface to compose and run `jj` (Jujutsu) commands safely and efficiently. By mapping `jj`'s command taxonomy into an organized, multi-pane layout, you can quickly explore available subcommands and flags with immediate documentation feedback.

---

## Interface Overview

Jutsu's interface is organized into six primary panes and eight focus states. The active pane has a **peach** border, while inactive panes have a **lavender** border. Persistent choices (such as the selected command, subcommand, or active flags) are highlighted in **sapphire** or **peach** to keep your context visible as you move focus.

![Jutsu TUI Interface](/jutsu/images/hero.png)

### The 8 Focus States & Layout Panes

While the terminal is physically divided into a few main panels, Jutsu's navigation logic moves through **8 distinct focus states**:

1.  **Categories Pane** — High-level groupings of Jujutsu operations.
2.  **Commands Pane** — Specific `jj` commands available under the selected category.
3.  **Sub-cmds Pane** — Nested subcommands (e.g., `git fetch`, `bookmark list`) for the active command.
4.  **Flags Pane** — Toggleable options and flags for the current command.
5.  **Inputs Pane** — A dynamic field row that opens at the bottom when arguments or values are required.
6.  **Docs Pane** — Shows detailed description and help text for whichever command, subcommand, or flag is currently highlighted.
7.  **Output Pane** — Viewports showing live streaming execution output, logs, or results.
8.  **Command Bar** — A live-updated preview of the full `jj` shell command being composed, ready for execution.

---

## Command Categories

Jutsu groups Jujutsu commands into eight logical categories, structured roughly by their "blast radius"—starting from read-only inspection tools, moving to local work-copy mutations, history rewrites, remote synchronization, and administrative operations:

*   **View 🔍** — Read-only inspection (`log`, `diff`, `status`, file browsing, repository root, `bisect`). Nothing here mutates your codebase or history.
*   **Change ✏️** — Actions targeting the current working-copy commit `@` (e.g., descriptions, editing, untracking).
*   **Rewrite ⚙️** — Restructuring existing commit history (splitting, squashing, rebasing, abandoning commits).
*   **Sync 🔄** — Synchronizing and exchanging commits with Git remotes (git push/fetch, tracking bookmarks, tagging).
*   **Journal 📖** — Operations log and recovery (viewing undo logs, replaying or reverting transaction states).
*   **Navigate 🧭** — Changing the checkout target (`@`) without rewriting commits (`edit`, `next`, `prev`).
*   **Tools 🛠️** — Miscellaneous utilities (signing commits, managing sparse checkouts, batch scripting, auto-completion, help).
*   **Setup 📦** — Initialization and configuration (`git init`/`git clone` setups, workspace managers, config keys).

---

## Navigation & Controls

Jutsu is designed for efficient, mouse-free navigation. The core navigation keys are:

| Key | Action |
|---|---|
| `j` / `k` or `↓` / `↑` | Navigate vertically within the current pane (list items or viewport scroll) |
| `h` / `l` or `←` / `→` | Shift focus horizontally: **Categories** ⇋ **Commands** ⇋ **Sub-cmds** ⇋ **Flags** |
| `Space` | Toggle the highlighted option on/off (Flags pane and Global Options modal) |
| `i` | Jump directly into the **Inputs Pane** (when a command or flag requires a text value) |
| `g` | Open/Close the **Global Options Modal** (repository options, config overrides, etc.) |
| `Tab` | Focus the **Command Bar** to run or copy the composed command |
| `Esc` | Back out of a current focused pane (Inputs, Output, Command Bar) to the composer |
| `o` / `d` | Jump focus directly to the **Output** / **Docs** pane |
| `O` / `D` | Enlarge the **Output** / **Docs** pane to full-screen (press again or `Esc` to restore layout) |
| `c` | Copy focused pane's text content to system clipboard (strips ANSI color codes) |
| `r` | **Recall state:** Restore the exact selection, flags, and inputs from your last run command |
| `q` / `Ctrl+C` | Quit Jutsu |

---

## Flag States and Indicators

When navigating the **Flags Pane**, look for these distinct markers next to each option:

*   **`[*]` Mandatory** (Sapphire) — Always active, required by the command, and cannot be deselected (e.g., `--range` on `bisect run`).
*   **`[-]` Conflicted** (Maroon) — Locked out because another selected flag conflicts with this one. Selecting a conflicted flag will automatically deselect its conflicting counterpart—no manual cleanup required!
*   **`*` Required Group** (Yellow) — Highlights a group of flags where at least one option must be chosen. Trying to proceed or run the command without picking one triggers a red border flash.

---

## Global Options Modal

Jujutsu allows global options that precede the command (e.g., `jj --repository ../other log`). In Jutsu, these live in an isolated modal.

1.  Press `g` from any pane to toggle the Global Options Modal.
2.  The modal splits into a **Checklist** on the left and **Option Docs** on the right.
3.  Use `h` / `l` to move between checklist selection and scrolling the option's detailed help.
4.  Press `Space` to select global options.
5.  If a global option requires a value (e.g., `--config`), you do not type it inside the modal. Instead, simply toggle it, close the modal (`g` or `Esc`), and type the value directly in the unified **Inputs Pane** along with other parameters.
6.  Active global options are pinned to the top of the **Flags Pane** as a teal `◆` so you never forget they are active, and they are automatically placed in the correct location in the Command Bar.

---

## Command Execution & TTY Handoffs

When you focus the Command Bar (`Tab`) and press `Enter`, Jutsu runs your command in one of three ways based on the target command and execution environment:

### 1. Background Execution
Most read-only or minor mutation commands run completely in the background. The **Output Pane** displays an italicized mauve `Executing command...` indicator, and streams stdout/stderr directly into your viewport without interrupting your TUI layout.

### 2. Standalone Terminal Handoff
Commands requiring full-screen interactive editors or diff pickers (like `jj describe` or `jj commit` without `-m`) require direct terminal control.
When running standalone (outside a terminal multiplexer), Jutsu suspends itself and hands complete TTY control over to the editor. Once you save and close the editor (e.g., Vim, Nano), Jutsu resumes automatically, scrapes any plain-text summaries printed by Jujutsu, and displays them inside the Output Pane.

### 3. Multiplexer Handoff (Tmux & Herdr)
If you are running Jutsu inside a terminal multiplexer, Jutsu leverages its environment to keep your workspace clean:
*   **Interactive Editors:** Instead of suspending Jutsu, a new Tmux window or Herdr tab is spawned automatically containing the editor. Jutsu continues running intact; when you finish editing, the child window/tab closes, and focus returns seamlessly back to Jutsu.
*   **Secure Prompts (SSH / Credentials):** If a sync operation requires a passphrase, SSH fingerprint confirmation, or HTTPS credentials, Jutsu prompts you. Inside a multiplexer, this opens as a bottom split pane (~30% height), prompts you for inputs, and closes upon completion.

---

## Deep Dive Tutorials

To see Jutsu in action and learn specific workflows, explore our dedicated tutorials:

*   👉 **[Composing a Command](/jutsu/workflows/composing-a-command/)** — A step-by-step walk-through of selecting, configuring, and building command strings.
*   👉 **[Inputs and Global Options](/jutsu/workflows/inputs-and-global-options/)** — Mastering the dynamic inputs pane and utilizing global options.
*   👉 **[Interactive and Remote Operations](/jutsu/workflows/interactive-and-remote-operations/)** — Working with full-terminal handoffs, SSH keys, credential helpers, and multiplexer splits.
