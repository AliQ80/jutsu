---
title: Running Commands
description: Background execution, standalone terminal handoffs, and multiplexer handoffs.
---

When you focus the Command Bar (`Tab`) and press `Enter`, Jutsu runs your command in one of three ways based on the target command and execution environment.

## 1. Background Execution

Most read-only or minor mutation commands run completely in the background. While the command runs, the **Output Pane** displays an italicized mauve spinner with an `Executing command...` indicator. Jutsu waits for the command to finish and then displays its combined stdout/stderr in the Output Pane — output is not streamed line-by-line as the command runs.

## 2. Standalone Terminal Handoff

Commands requiring full-screen interactive editors or diff pickers (like `jj describe` or `jj commit` without `-m`) require direct terminal control.

When running standalone (outside a terminal multiplexer), Jutsu suspends itself and hands complete TTY control over to the editor. Once you save and close the editor (e.g., Vim, Nano), Jutsu resumes automatically. If the editor's alternate-screen session left a recoverable plain-text summary (e.g. `jj`'s post-squash summary), Jutsu captures it after the handoff returns and displays it inside the Output Pane.

## 3. Multiplexer Handoff (Tmux & Herdr)

If you are running Jutsu inside a terminal multiplexer, Jutsu leverages its environment to keep your workspace clean:

- **Interactive Editors:** Instead of suspending Jutsu, a new Tmux window or Herdr tab is spawned automatically containing the editor. Jutsu continues running intact; when you finish editing, the child window/tab closes, and focus returns seamlessly back to Jutsu.
- **Secure Prompts (SSH / Credentials):** If a background sync operation fails because it needed a passphrase, SSH fingerprint confirmation, or HTTPS credentials, Jutsu first shows an in-TUI modal explaining that the command needs terminal input. Only after you confirm (press `Enter`) does Jutsu hand off to a real terminal — a Tmux/Herdr split pane inside a multiplexer, or a standalone suspend outside one. The handoff does not open automatically; the modal always waits for your confirmation first.

## Related Topics

- **[Output & Docs](/jutsu/usage/output-and-docs/)** — where streamed output and execution summaries are displayed.
- **[Interactive & Remote Operations](/jutsu/workflows/interactive-and-remote-operations/)** — a deeper look at TTY handoffs, SSH verification, and credential prompts.
