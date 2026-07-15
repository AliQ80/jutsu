# Usage

## Panes & Layout

Jutsu has 8 focus states:

1. **Categories** — high-level command groups (see below)
2. **Commands** — jj commands within the selected category
3. **Sub-cmds** — nested subcommands (e.g. `git fetch`, `bookmark list`)
4. **Flags** — options that can be toggled on/off for the selected command
5. **Inputs** — a temporary pane that appears only when the current selection needs argument or flag values typed in
6. **Docs** — description of whatever's currently focused
7. **Output** — results of the last executed command
8. **Command Bar** — the composed command, ready to run

The Sub-cmds and Flags columns still render (empty) for commands that have
none — navigating past them with `h`/`l` is automatic, not a bug. The pane
you're in is bordered in peach; an inactive pane's border is lavender. A
selection that persists across panes — the chosen category/command/
subcommand, or a toggled-on flag — stays visible in sapphire even after you
move focus elsewhere.

## Command Categories

- **View** — read-only inspection (log, diff, status, browsing files, repo root); nothing here changes history
- **Change** — edits to the working-copy commit itself
- **Rewrite** — restructuring existing commits (splitting, squashing, rebasing, abandoning)
- **Sync** — ongoing exchange with remotes and other refs (git fetch/push, bookmarks, tags)
- **Journal** — the operation log; reverting or replaying prior operations
- **Advanced** — config, workspaces, repo maintenance, and shell completion
- **Setup** — creating a new repo (`git init`/`git clone`)

## Navigating the composer

| Key | Action |
|-----|--------|
| `j`/`k` or `↓`/`↑` | Move up/down within the current pane |
| `h`/`l` or `←`/`→` | Move focus between Categories → Commands → Sub-cmds → Flags |
| `Space` | Toggle the focused flag (Flags pane only) |
| `Enter` | Jump into Inputs, if the current selection needs values |
| `Tab` | Move to the Command Bar |
| `r` | Recall the last executed command's full selection state |
| `q` / `Ctrl+C` | Quit |

## Flag states

Three visual markers, easy to mix up:

- **`[*]` mandatory** (sapphire) — always selected, can't be toggled off (e.g. `--range` on `bisect run`)
- **`[-]` conflicted** (maroon) — blocked because a conflicting flag is currently selected; picking the new flag auto-clears the old one, no manual cleanup needed
- **`*` required group** (yellow) — at least one flag in this group must be selected before you can proceed; trying to move on without one flashes the Flags pane border red

## Filling in values

When a command needs an argument or a flag's value, `Enter` switches focus
to the Inputs pane:

- Typing goes straight into the focused field
- `↑`/`↓` moves between fields (not the cursor within one)
- `Enter` moves to the next field, wrapping around — it doesn't submit
- `Esc` backs out to the last composer pane
- `Tab` validates and moves to the Command Bar

Leaving a required field empty flashes the inputs box border red for 400ms
if you try to move on.

## Running a command

`Enter` in the Command Bar executes the composed command. What happens
next depends on the command:

- **Most commands** run in the background — the Output pane shows an
  italic mauve "Executing command..." status, then streams the result in
  without ever leaving the TUI.
- **Commands that open an editor or a merge/diff tool** (e.g. `describe`/
  `commit` without `-m`, `squash` without `-m`, `resolve` without `--list`,
  or anything with an explicit interactive flag like `-i`) hand the real
  terminal over to that program. The screen will visibly change — this is
  expected, not a freeze — and control returns to Jutsu automatically once
  the program exits.

## Docs, Output, and enlarging

- `o` / `d` jump to the Output / Docs pane; `Esc` or `Tab` returns to the composer
- `O` / `D` enlarge the Output / Docs pane to fill the screen — enlarging one isn't available while the other is already enlarged
- `j`/`k`, `pgup`/`pgdown` scroll; `h`/`l` pans horizontally (Output pane)
- `c` copies the pane's content to the clipboard (ANSI formatting stripped), flashing a green "✓ copied" in the help bar

## Help bar

The bottom bar shows key hints for whatever's focused, your current
directory (sky), and a badge comparing your installed jj version against
the one Jutsu was built against: green (compatible), yellow (partially
compatible), or red (incompatible).
