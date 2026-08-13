---
title: Global Options
description: The Global Options modal, options that precede the command, value entry, and pinned indicators.
---

Global Options answer the question **"Which repository/context?"** They configure `jj`'s own environment rather than the command being run, and must precede the command on the shell (e.g., `jj --repository ../other log`).

## The Global Options Modal

Because these options are distinct from command-specific flags, they live in an isolated modal:

1. Press `g` to open the Global Options Modal — available from the composer, Output, Docs, or Command Bar (including enlarged Output/Docs views), but not while focused in the Inputs Pane.
2. The modal splits into a **Checklist** on the left and **Option Docs** on the right.
3. Use `h` / `l` (or `←` / `→`) to move between checklist selection and scrolling the option's detailed help.
4. Press `Space` to select global options.
5. Press `Esc` or `g` again to close the modal and return to your previous composer state. **Note:** `q` quits Jutsu entirely from inside the modal, the same as it does elsewhere — it does not just close the modal.

## Global Options That Require Values

If a global option requires a value (e.g., `--config`), you do not type it inside the modal. Instead:

1. Toggle the option with `Space` and close the modal (`Esc` or `g`).
2. Closing the modal returns focus to your last composer pane — it does not automatically jump into the Inputs Pane, and `g` cannot be used to reopen the modal while inputs are focused.
3. Press `i` yourself to enter the [Inputs Pane](/jutsu/usage/inputs/) (available from any non-Inputs, non-enlarged view), where a new input field for the option has been dynamically created alongside the command's normal inputs. This keeps all text values visible at once.

## Pinned Indicators and Command Splicing

Once a global option is selected, Jutsu provides clear visual indicators to remind you that your environment is modified:

- **Teal Pinned Rows (`◆`):** Active global options are pinned directly to the top of the standard **Flags Pane** with a teal `◆` prefix, staying visible even if you scroll down a long list of command flags.
- **Spliced Command String:** The **Command Bar** automatically splices active global options *before* the subcommand, matching `jj`'s precise command-line interface — `jj --repository ../my-repo log` instead of `jj log --repository ../my-repo`.

## Related Topics

- **[Inputs](/jutsu/usage/inputs/)** — where global option values are typed.
- **[Flags](/jutsu/usage/flags/)** — command-specific flags, as distinct from global options.
- **[Inputs and Global Options](/jutsu/workflows/inputs-and-global-options/)** — a scenario demonstrating targeting another repository with a global option.
