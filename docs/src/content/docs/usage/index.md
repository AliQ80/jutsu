---
title: Usage Guide
description: An overview of Jutsu's interface, command composition, and execution model, with links to focused topic pages.
---

Jutsu provides an interactive, keyboard-driven interface to compose and run `jj` (Jujutsu) commands safely and efficiently. By mapping `jj`'s command taxonomy into an organized, multi-pane layout, you can quickly explore available subcommands and flags with immediate documentation feedback.

![Jutsu TUI Interface](/jutsu/images/hero.png)

Jutsu's interface is organized into six primary panes and eight focus states, moving from broad category selection through command, subcommand, flag, and input selection, out to a live command preview and executed output. The topics below cover each part of that model in depth:

1. **[Interface](/jutsu/usage/interface/)** — panes, focus states, layout, and persistent selection indicators.
2. **[Navigation](/jutsu/usage/navigation/)** — keyboard controls, horizontal focus movement, pane shortcuts, and quitting.
3. **[Commands](/jutsu/usage/commands/)** — categories, command selection, subcommands, and the command-preview mental model.
4. **[Flags](/jutsu/usage/flags/)** — optional flags, mandatory flags, conflicts, required groups, and flag indicators.
5. **[Inputs](/jutsu/usage/inputs/)** — command/flag values, field navigation, required-field validation, and the validation flash.
6. **[Global Options](/jutsu/usage/global-options/)** — the Global Options modal, options that precede the command, value entry, and pinned indicators.
7. **[Running Commands](/jutsu/usage/execution/)** — background execution, standalone terminal handoffs, and multiplexer handoffs.
8. **[Output & Docs](/jutsu/usage/output-and-docs/)** — pane navigation, scrolling, enlargement, horizontal panning, and clipboard copying.

## How the pieces fit together

Building a Jutsu command follows a simple mental model:

```text
What do I want to do?     → Commands
How should it behave?     → Flags
What values are needed?   → Inputs
Which repository/context? → Global Options
```

Each concept has one canonical page above; the [Deep Dive Tutorials](#deep-dive-tutorials) below show them combined into concrete, end-to-end scenarios.

## Deep Dive Tutorials

To see Jutsu in action and learn specific workflows, explore our dedicated tutorials:

- 👉 **[Composing a Command](/jutsu/workflows/composing-a-command/)** — A step-by-step walk-through of selecting, configuring, and building command strings.
- 👉 **[Inputs and Global Options](/jutsu/workflows/inputs-and-global-options/)** — Mastering the dynamic inputs pane and utilizing global options.
- 👉 **[Interactive and Remote Operations](/jutsu/workflows/interactive-and-remote-operations/)** — Working with full-terminal handoffs, SSH keys, credential helpers, and multiplexer splits.
