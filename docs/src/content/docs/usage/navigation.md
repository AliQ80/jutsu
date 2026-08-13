---
title: Navigation
description: Keyboard controls, horizontal focus movement, pane shortcuts, and quitting Jutsu.
---

Jutsu is designed for efficient, mouse-free navigation. The core navigation keys are:

| Key | Action |
|---|---|
| `j` / `k` or `↓` / `↑` | Navigate vertically within the current pane (list items or viewport scroll) |
| `h` / `l` or `←` / `→` | Shift focus horizontally: **Categories** ⇋ **Commands** ⇋ **Sub-cmds** ⇋ **Flags** |
| `Space` | Toggle the highlighted option on/off (Flags pane and Global Options modal) |
| `i` | Jump directly into the **Inputs Pane** (when a command or flag requires a text value); unavailable while an enlarged Output/Docs view is focused |
| `g` | Open the **Global Options Modal** (repository options, config overrides, etc.) from any pane except the Inputs Pane |
| `Tab` | Focus the **Command Bar** to run or copy the composed command |
| `Esc` | Back out of a current focused pane (Inputs, Output, Command Bar) to the composer |
| `o` / `d` | Jump focus directly to the **Output** / **Docs** pane; unavailable while focused in Inputs, or while the other pane is enlarged |
| `O` / `D` | Enlarge the **Output** / **Docs** pane to full-screen (press again or `Esc` to restore layout) |
| `c` | Copy focused pane's text content to system clipboard (strips ANSI color codes) |
| `r` | **Recall state:** Restore the exact selection, flags, and inputs from your last run command |
| `q` / `Ctrl+C` | Quit Jutsu |

## Horizontal Focus Movement

`h` / `l` (or `←` / `→`) move focus between the **Categories**, **Commands**, **Sub-cmds**, and **Flags** panes. If a command has no subcommands, moving right automatically skips the empty **Sub-cmds** pane and lands on **Flags**, avoiding unnecessary keystrokes.

## Jumping to Other Panes

Rather than tabbing through every composer pane, dedicated shortcuts jump focus directly:

- `i` jumps into the **Inputs Pane** whenever a value is required — from any composer pane, the Output/Docs panes, or the Command Bar, but not while an enlarged Output/Docs view is focused (press `Esc` first to restore the layout).
- `g` opens the **Global Options Modal** from the composer, Output, Docs, or Command Bar contexts (including enlarged views) — but not while focused in the Inputs Pane. Press `Esc` or `g` again to close it; `q` quits Jutsu instead of closing the modal.
- `Tab` focuses the **Command Bar**.
- `o` / `d` jump straight to the **Output** / **Docs** panes from composer contexts or the Command Bar, but not while focused in Inputs, and not while the *other* pane is enlarged (e.g. `o` is blocked while Docs is enlarged, and vice versa).
- `Esc` backs out of whichever pane is focused (Inputs, Output, Command Bar) and returns to the last composer pane.

## Related Topics

- **[Interface](/jutsu/usage/interface/)** — the panes these keys move between.
- **[Inputs](/jutsu/usage/inputs/)** — details on the Inputs Pane reached with `i`.
- **[Global Options](/jutsu/usage/global-options/)** — details on the modal opened with `g`.
- **[Output & Docs](/jutsu/usage/output-and-docs/)** — details on enlarging and scrolling panes with `o`/`d`/`O`/`D`.
