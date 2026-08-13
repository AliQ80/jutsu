---
title: Flags
description: Optional flags, mandatory flags, conflicts, required groups, and flag indicators.
---

Flags answer the question **"How should it behave?"** They configure the currently selected command or subcommand.

## Toggling Flags

In the **Flags Pane**, use `j`/`k` (or `↓`/`↑`) to highlight a flag and press `Space` to toggle it on or off. Toggled flags are highlighted in **peach** and immediately appended to the **Command Bar** preview. As you move focus to other panes, toggled flags remain visible in **sapphire** so you never lose track of your current configuration.

## Flag States and Indicators

When navigating the **Flags Pane**, look for these distinct markers next to each option:

- **`[*]` Mandatory** (Sapphire) — Always active, required by the command, and cannot be deselected (e.g., the `range` flag, `-r`, on `bisect run`).
- **`[-]` Conflicted** (Maroon) — Locked out because another selected flag conflicts with this one. Selecting a conflicted flag will automatically deselect its conflicting counterpart—no manual cleanup required!
- **`*` Required Group** (Yellow) — Highlights a group of flags where at least one option must be chosen. Trying to proceed or run the command without picking one triggers a red border flash.

## Flags That Require Values

Some flags require a text value in addition to being toggled (e.g., `--branch <name>`). Selecting such a flag opens a corresponding field in the **Inputs Pane** — see [Inputs](/jutsu/usage/inputs/) for how values are entered and validated.

## Related Topics

- **[Commands](/jutsu/usage/commands/)** — the command each flag configures.
- **[Inputs](/jutsu/usage/inputs/)** — entering values for flags that require them, and the validation flash for required fields.
- **[Global Options](/jutsu/usage/global-options/)** — options that precede the command rather than configuring it, handled in a separate modal.
