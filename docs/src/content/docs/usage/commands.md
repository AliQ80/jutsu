---
title: Commands
description: Categories, command selection, subcommands, and the command-preview mental model.
---

Commands answer the question **"What do I want to do?"** Jutsu groups every `jj` command into a category, and some commands expose further subcommands.

## Command Categories

Jutsu groups Jujutsu commands into eight logical categories, structured roughly by their "blast radius"—starting from read-only inspection tools, moving to local work-copy mutations, history rewrites, remote synchronization, and administrative operations:

- **View 🔍** — Read-only inspection (`log`, `diff`, `status`, file browsing, repository root, `bisect`). Most of these commands are non-mutating, though `bisect run` evaluates revisions by editing the working copy to each candidate revision in turn as part of the search, so it can change your working-copy selection.
- **Change ✏️** — Actions targeting the current working-copy commit `@` (e.g., descriptions, editing, untracking).
- **Rewrite ⚙️** — Restructuring existing commit history (splitting, squashing, rebasing, abandoning commits).
- **Sync 🔄** — Synchronizing and exchanging commits with Git remotes (git push/fetch, tracking bookmarks, tagging).
- **Journal 📖** — Operations log and recovery (viewing undo logs, replaying or reverting transaction states).
- **Navigate 🧭** — Changing the checkout target (`@`) without rewriting commits (`edit`, `next`, `prev`).
- **Tools 🛠️** — Miscellaneous utilities (signing commits, managing sparse checkouts, batch scripting, auto-completion, help).
- **Setup 📦** — Initialization and configuration (`git init`/`git clone` setups, workspace managers, config keys).

## Selecting a Command and Subcommand

1. Highlighting a **Category** populates the **Commands** pane with the specific `jj` commands available under it.
2. Highlighting a **Command** populates the **Sub-cmds** pane with any nested subcommands (e.g., `git fetch`, `bookmark list`) for that command.
3. If a command has no subcommands, moving focus right automatically skips the empty **Sub-cmds** pane and lands on **Flags** — see [Navigation](/jutsu/usage/navigation/) for the horizontal movement keys.
4. Whichever command, subcommand, or flag is currently highlighted has its official description and help text shown live in the **Docs Pane**.

## The Command-Preview Mental Model

As you select a category, command, and subcommand, the **Command Bar** at the bottom of the screen updates in real time to reflect the `jj` invocation being built — for example, selecting `log` under **View** immediately shows `jj log` in the preview, before any flags or inputs are added. This lets you see exactly what will run at every stage of composition, rather than guessing at syntax.

## Related Topics

- **[Flags](/jutsu/usage/flags/)** — how flags attach to the selected command.
- **[Inputs](/jutsu/usage/inputs/)** — how argument values attach to the selected command or its flags.
- **[Composing a Command](/jutsu/workflows/composing-a-command/)** — a full walkthrough from category selection to execution.
