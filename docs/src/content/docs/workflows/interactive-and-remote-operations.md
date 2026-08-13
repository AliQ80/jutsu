---
title: "Tutorial: Interactive & Remote Operations"
description: Learn how Jutsu securely manages full-screen terminal editors, credential requests, and remote push/pull workflows.
---

Working with version control often requires stepping outside a pure terminal UI—for example, to write a long commit message in your `$EDITOR`, resolve a merge conflict in an interactive diff tool, or enter a password for a secure remote repository. 

Jutsu handles these scenarios seamlessly using **TTY Handoffs**, **Multiplexer Splits**, and **Dynamic Prompts**. This tutorial explains how these features protect your workflow from freezing or hanging.

For an overview of the three execution paths (background, standalone handoff, multiplexer handoff), see the canonical **[Running Commands](/jutsu/usage/execution/)** page. This tutorial walks through the standalone and multiplexer scenarios in more depth, plus remote credential prompts.

---

## Interactive Commands (TTY Handoffs)

Certain `jj` commands—such as `jj describe`, `jj commit` (without `-m`), or `jj resolve` (without `--list`)—must open external interactive programs like Vim, Nano, or VS Code.

Jutsu handles this differently depending on whether you are running in a standard standalone terminal or inside a terminal multiplexer (like Tmux or Herdr).

### Scenario A: Standalone Terminal Handoff

When running in a standard terminal (e.g., Alacritty, iTerm2, Ghostty) without a multiplexer:

1.  **Handoff:** When you run `jj describe`, Jutsu temporarily suspends itself and hands complete control of your active TTY over to the child editor process.
2.  **External Editing:** Your terminal screen will switch completely to your text editor. This is expected behavior. You can write and format your commit description normally.
3.  **Automatic Resume:** Once you save and quit your editor, the child process exits. Jutsu instantly resumes, restores its state, reads the plain-text execution summary from Jujutsu (e.g., `Rebased 1 descendant commits`), and displays it in your **Output Pane**.

:::note[How It Works Under the Hood]
On Linux and macOS, standalone handoffs utilize a real pseudo-terminal (`pty`) to capture, buffer, and stream the subprocess's standard streams safely. When the editor exits, the TUI's input reader is securely resumed without dropping post-handoff keystrokes.
:::

### Scenario B: Multiplexer Handoff (Tmux & Herdr)

If you run Jutsu inside **Tmux** or **Herdr**, the TUI doesn't need to suspend itself. Instead, it utilizes your multiplexer's windowing capabilities to keep Jutsu open and visible:

```
┌────────────────────────┐  ┌────────────────────────┐
│                        │  │  $EDITOR (Vim / Nano)  │
│      Jutsu TUI         │  │                        │
│   (Active & Rendering) │  │  (Opens in new window  │
│                        │  │   or herdr tab)        │
└────────────────────────┘  └────────────────────────┘
```

1.  **Spawn Window/Tab:** The editor or diff tool opens automatically in a **new Tmux window** or a **new Herdr tab**.
2.  **Keep Rendering:** Jutsu remains active, visible, and fully interactive in your original terminal pane.
3.  **Auto-Cleanup:** When you close your editor, the multiplexer window or tab is destroyed automatically. Focus is returned to your Jutsu session, and the results are loaded into the Output Pane.

---

## Remote Operations & Secure Prompts

Synchronizing with Git remotes via `jj git push` or `jj git fetch` usually runs in the background. If a remote operation fails because it needs credentials or host verification, Jutsu first shows an in-TUI modal explaining that the command needs terminal input — the standalone or multiplexer handoff described below only happens after you confirm it; it does not open automatically the moment credentials are needed.

### 1. SSH Host Verification

If you are pushing to or fetching from a remote server for the first time, SSH requires you to verify the server's fingerprint.

```
┌────────────────────────────────────────────────────────┐
│ Unknown SSH Host Fingerprint                           │
│ Trust fingerprint: SHA256:abc123xyz...                 │
│                                                        │
│ [y] Trust & Save        [Esc] Cancel                   │
└────────────────────────────────────────────────────────┘
```

*   **In-TUI Verification:** Jutsu captures the SSH fingerprint prompt and displays a modal inside the TUI.
*   **One-Tap Save:** Pressing `y` automatically appends the host signature to your `~/.ssh/known_hosts` file and retries the remote operation instantly.
*   **Security Guard:** If an SSH host key changes (indicating a potential security issue), Jutsu raises a strict warning and will never offer a one-tap trust override.

:::caution[Verify the fingerprint yourself]
Jutsu fetches the host's key with `ssh-keyscan`, which is not authenticated on its own — it simply reports whatever key the server currently presents, the same way it would to an attacker performing a man-in-the-middle attack. Before pressing `y`, verify the displayed `SHA256:` fingerprint against a trusted source (e.g., your hosting provider's published host-key fingerprints) rather than trusting the prompt by default.
:::

### 2. Password and Passphrase Credentials

When a remote HTTPS or SSH server requests a username, password, or key passphrase:

*   **Prompt Detection:** Jutsu detects the authentication request and displays a "continue in terminal" prompt modal — it waits for your confirmation rather than opening a terminal automatically.
*   **Standalone Flow:** Pressing `Enter` confirms the prompt and hands your active TTY over to the secure credential prompt. Once typed, control returns to Jutsu. Pressing `n`/`Esc`/`q` cancels instead.
*   **Multiplexer Flow:** Inside Tmux or Herdr, confirming with `Enter` opens a **split pane** (~30% height) at the bottom of the screen to collect your credentials, leaving your main TUI layout untouched. The split pane is disposed of automatically once authentication succeeds.

---

## Troubleshooting Handoffs

If an editor or multiplexer ever behaves unexpectedly, you can force Jutsu to bypass multiplexer integrations and fall back to the standalone terminal handoff mode:

```bash
# Bypass Tmux/Herdr integrations and force standalone suspend-handoffs
env -u TMUX -u HERDR_ENV jutsu
```

This ensures that regardless of your complex terminal shell configuration, you can always commit, rebase, and write logs reliably.

---

## Related Topics

- 👉 **[Running Commands](/jutsu/usage/execution/)** — the canonical explanation of background execution, standalone handoffs, and multiplexer handoffs.
- 👉 **[Output & Docs](/jutsu/usage/output-and-docs/)** — where execution summaries and streamed output are displayed.
