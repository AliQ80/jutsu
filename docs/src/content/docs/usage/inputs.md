---
title: Inputs
description: Command/flag values, field navigation, required-field validation, and the validation flash.
---

Inputs answer the question **"What values are needed?"** They supply the text arguments and flag values that commands and flags require.

## The Inputs Pane

The **Inputs Pane** is a dynamic field row that automatically opens at the bottom of the composer when your composed command requires additional text parameters (e.g., a message for `jj describe` or a target revision for `jj rebase`).

Press `i` from any non-Inputs, non-enlarged view (composer panes, non-enlarged Output/Docs, or the Command Bar) to jump directly into the Inputs Pane when a command or flag requires a value. `i` is unavailable while Inputs is already focused (it types the letter "i" instead) and while an enlarged Output or Docs view is showing.

## Entering and Navigating Inputs

1. **Direct Typing:** Once active, your keyboard focus shifts entirely to typing. The cursor flashes within the first active input field.
2. **Field Navigation:**
   - Use `↑` / `↓` to move focus between different input fields (e.g., moving from a flag's value field to a positional argument field).
   - Press `Enter` to cycle down to the next field (wrapping back to the top field if you are on the last one). **Note:** Pressing `Enter` in the Inputs Pane does *not* execute the command; it is reserved for field switching.
3. **Applying Inputs:** Once finished, press `Tab` to validate your inputs and focus the Command Bar.
4. **Canceling / Backing Out:** Press `Esc` at any time to discard focus from the inputs and return to the main composer. Your text entries are preserved so you can jump back in easily.

## Required-Field Validation & the Validation Flash

To prevent you from running incomplete commands that would error out in the shell, Jutsu enforces structural input validation:

- If a selected flag has `RequiresInput` enabled but you leave the field empty, attempting to move to the Command Bar with `Tab` triggers a **validation flash** instead of moving focus.
- The entire Inputs Pane border flashes **bright red for 400 milliseconds**, and the command cannot proceed to the Command Bar until you either fill in the required field or deselect the flag. `Esc` still backs you out of the Inputs Pane at any time — the flash blocks advancing via `Tab`, not leaving the pane.

## Global Option Values

If an active [Global Option](/jutsu/usage/global-options/) requires a value (e.g., `--config <key=value>`), its field appears in the same unified Inputs Pane alongside the command's own inputs — you do not type it inside the Global Options modal.

## Related Topics

- **[Flags](/jutsu/usage/flags/)** — which flags require input values.
- **[Global Options](/jutsu/usage/global-options/)** — options whose values also flow through the Inputs Pane.
- **[Inputs and Global Options](/jutsu/workflows/inputs-and-global-options/)** — a scenario-driven walkthrough of both mechanisms together.
