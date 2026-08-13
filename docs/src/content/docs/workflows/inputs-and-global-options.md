---
title: "Tutorial: Inputs & Global Options"
description: Master the inputs system, required arguments, and global option overrides in Jutsu.
---

Many Jujutsu operations require dynamic parameters—such as target commit revisions, custom messages, file paths, or global repository overrides. Jutsu handles these variables through two distinct mechanisms: the **Inputs Pane** (for arguments and flag values) and the **Global Options Modal** (for flags that modify `jj`'s global behavior).

For the general mechanics referenced below, see the canonical **[Inputs](/jutsu/usage/inputs/)** and **[Global Options](/jutsu/usage/global-options/)** pages. This tutorial focuses on a concrete scenario: targeting another repository with a global option while supplying a commit message.

---

## The Inputs Pane

The **Inputs Pane** is a dynamic input tray that automatically slides open at the bottom of the composer when your composed command requires additional text parameters (e.g., a message for `jj describe` or a target revision for `jj rebase`). See [Inputs](/jutsu/usage/inputs/) for field navigation and validation details.

```
┌────────────────────────────────────────────────────────┐
│ [Inputs]                                               │
│  Commit Message: [ My message here                   ] │
└────────────────────────────────────────────────────────┘
```

### Applying an Input Value

When a flag or command requiring a value is selected, press `i` from any non-Inputs, non-enlarged view to enter text-input mode, type your value, then press `Tab` to validate your inputs and focus the Command Bar. If a required field is left empty, the Inputs Pane border flashes red — see [Inputs](/jutsu/usage/inputs/) for the full validation flash behavior.

---

## Global Options Modal

Jujutsu's global options—such as `--repository <path>`, `--config <key=value>`, or `--at-operation <id>`—modify the environment of the command and must precede the subcommand (e.g., `jj --repository ../other log`). See [Global Options](/jutsu/usage/global-options/) for the full modal mechanics.

```
┌────────────────────────────────────────────────────────┐
│ Global Options Modal (g)                               │
│  [x] --repository          [ Docs for the highlighted ] │
│  [ ] --at-operation        [ option scroll here...    ] │
└────────────────────────────────────────────────────────┘
```

### Scenario: Targeting Another Repository

Suppose you want to describe the working-copy commit in a different repository checkout at `../other-repo`, rather than the one Jutsu was launched in:

1. Press `g` to open the Global Options Modal, highlight `--repository`, and press `Space` to select it.
2. Close the modal with `Esc` or `g` (pressing `q` here quits Jutsu entirely, not just the modal). Closing the modal returns you to your last composer pane — it does not automatically jump into the Inputs Pane.
3. Press `i` yourself to enter the Inputs Pane, where a new field for `--repository` appears alongside the command's own inputs (e.g., the commit message for `jj describe`). Note that `g` cannot reopen the modal while you're focused in the Inputs Pane.
4. Type `../other-repo` into the `--repository` field, and your message into the `Commit Message` field, then press `Tab` to move to the Command Bar.
5. The Command Bar splices the global option before the subcommand: `jj --repository ../other-repo describe -m "..."`.

### Visual Pinned Indicators

Once a global option is selected, Jutsu pins it to the top of the **Flags Pane** with a teal `◆` prefix so it stays visible even if you scroll down a long list of command flags — see [Global Options](/jutsu/usage/global-options/) for details.

---

## Next Steps

With command composing, inputs, and global options mastered, explore how Jutsu safely manages heavy background processes, interactive terminal handoffs, and remote operations:

- 👉 **[Interactive & Remote Operations](/jutsu/workflows/interactive-and-remote-operations/)**
- 👉 **[Running Commands](/jutsu/usage/execution/)**
