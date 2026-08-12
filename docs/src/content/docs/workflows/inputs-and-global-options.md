---
title: "Tutorial: Inputs & Global Options"
description: Master the inputs system, required arguments, and global option overrides in Jutsu.
---

Many Jujutsu operations require dynamic parameters—such as target commit revisions, custom messages, file paths, or global repository overrides. Jutsu handles these variables through two distinct mechanisms: the **Inputs Pane** (for arguments and flag values) and the **Global Options Modal** (for flags that modify `jj`'s global behavior).

---

## The Inputs Pane

The **Inputs Pane** is a dynamic input tray that automatically slides open at the bottom of the composer when your composed command requires additional text parameters (e.g., a message for `jj describe` or a target revision for `jj rebase`).

```
┌────────────────────────────────────────────────────────┐
│ [Inputs]                                               │
│  Commit Message: [ My message here                   ] │
└────────────────────────────────────────────────────────┘
```

### Entering and Navigating Inputs

When a flag or command requiring a value is selected, press `i` from any pane to enter text-input mode:

1.  **Direct Typing:** Once active, your keyboard focus shifts entirely to typing. The cursor flashes within the first active input field.
2.  **Field Navigation:** 
    *   Use `↑` / `↓` to move focus between different input fields (e.g., moving from a flag's value field to a positional argument field).
    *   Press `Enter` to cycle down to the next field (wrapping back to the top field if you are on the last one). **Note:** Pressing `Enter` in the Inputs Pane does *not* execute the command; it is reserved for field switching.
3.  **Applying Inputs:** Once finished, press `Tab` to validate your inputs and focus the Command Bar.
4.  **Canceling / Backing Out:** Press `Esc` at any time to discard focus from the inputs and return to the main composer. Your text entries are preserved so you can jump back in easily.

### Input Validation & Safety Flashes

To prevent you from running incomplete commands that would error out in the shell, Jutsu enforces structural input validation:

*   If a selected flag has `RequiresInput` enabled but you leave the field empty, attempting to navigate away to the Command Bar will trigger a **validation flash**.
*   The entire Inputs Pane border will flash **bright red for 400 milliseconds**, and focus is forced to remain inside the Inputs Pane until you either fill in the required field or deselect the flag.

---

## Global Options Modal

Jujutsu's global options—such as `--repository <path>`, `--config <key=value>`, or `--at-operation <id>`—modify the environment of the command and must precede the subcommand (e.g., `jj --repository ../other log`).

Because these options are distinct from command-specific flags, they reside in a specialized, dedicated modal.

```
┌────────────────────────────────────────────────────────┐
│ Global Options Modal (g)                               │
│  [x] --repository          [ Docs for the highlighted ] │
│  [ ] --at-operation        [ option scroll here...    ] │
└────────────────────────────────────────────────────────┘
```

### Accessing and Navigating the Modal

*   **Open/Close:** Press `g` from any composer pane to toggle the modal. Pressing `g`, `Esc`, or `q` will close it and return you to your previous composer state.
*   **Dual-Column Layout:** The modal is split into a **Checklist** on the left and full **Help Documentation** on the right.
*   **Focus Switch:** Use `h` / `l` (or `←` / `→`) to toggle focus between the checklist and the help viewport.
*   **Scrolling Docs:** Focus the right column with `l` and use `j` / `k` (or `↓` / `↑`) to scroll through long option help pages.
*   **Toggling Options:** Highlight an option on the left column and press `Space` to activate/deactivate it.

### Typing Global Values (The Multi-Value Flow)

If a global option takes a value (e.g., `--repository` requires a path), **you do not type it inside the modal**.

Instead, follow this streamlined flow:
1.  Open the modal (`g`), toggle `--repository` with `Space`, and close the modal (`g`).
2.  Jutsu notices that an active option requires input and automatically redirects your cursor.
3.  Press `i` to enter the Inputs Pane, where a new input field for `--repository` has been dynamically created alongside your command's normal inputs. This keeps all text values visible at once!

### Visual Pinned Indicators and Command Splicing

Once a global option is selected, Jutsu provides clear visual indicators to remind you that your environment is modified:

1.  **Teal Pinned Rows (`◆`):** Active global options are pinned directly to the very top of the standard **Flags Pane** with a teal `◆` prefix. They stay visible even if you scroll down a long list of command flags.
2.  **Spliced Command String:** Look at the Command Bar. Jutsu automatically splices active global options *before* the subcommand, matching `jj`'s precise command-line interface:
    `jj --repository ../my-repo log` instead of `jj log --repository ../my-repo`.

---

## Next Steps

With command composing, inputs, and global options mastered, explore how Jutsu safely manages heavy background processes, interactive terminal handoffs, and remote operations:

*   👉 **[Interactive & Remote Operations](/jutsu/workflows/interactive-and-remote-operations/)**
