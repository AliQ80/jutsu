---
title: "Tutorial: Composing a Command"
description: A step-by-step guide on how to navigate, select options, and dynamically compose commands in Jutsu.
---

Jutsu's core strength is its ability to let you compose complex `jj` commands interactively, with safety and documentation guiding every choice. This tutorial walks you through the step-by-step process of building a command from scratch and highlights key shortcuts that speed up your daily version control workflows.

For the general mechanics referenced below, see the canonical **[Commands](/jutsu/usage/commands/)** and **[Flags](/jutsu/usage/flags/)** pages.

---

## The Workflow Loop

Building a command in Jutsu follows a simple, repeating vertical and horizontal flow:

```
[ Categories ] ➔ [ Commands ] ➔ [ Sub-cmds ] ➔ [ Flags ]
      │               │               │            │
      └────── (Navigate horizontally with h / l) ──┘
```

1.  **Select a Category** (e.g., `View` or `Rewrite`).
2.  **Highlight the Command** (e.g., `log` or `rebase`).
3.  **Choose a Subcommand** if one exists (e.g., `bookmark list` or `git fetch`).
4.  **Toggle Flags & Options** (e.g., `--git-repo`, `-n`).
5.  **Provide Arguments/Inputs** if needed (e.g., commit hashes, branch names).
6.  **Run or Copy** the command.

Let's look at each phase in detail.

---

## Step-by-Step Walkthrough: Viewing a Filtered Log

Let's say you want to view the log filtered down to a specific revset instead of the default view.

### Step 1: Select the Category
1. When Jutsu launches, your focus begins in the **Categories** pane (far-left column).
2. Use `j`/`k` (or `↓`/`↑`) to highlight the **View** category.
3. Observe that the list of commands in the second column updates instantly to show read-only inspection commands.

### Step 2: Move to the Commands Column
1. Press `l` (or `→`) to slide your active focus into the **Commands** pane. The border around the Commands pane turns **peach** (active), while Categories becomes **lavender** (inactive).
2. Scroll down to highlight `log`.
3. Look at the bottom **Docs Pane**: it instantly displays the official command description and help documentation for `jj log`.

### Step 3: Select Subcommands (If Any)
For `jj log`, there are no subcommands. 
1. If you press `l` (or `→`), Jutsu detects that the `Sub-cmds` column is empty and **automatically skips** to the next available column—the **Flags** column.
2. This automatic column skipping prevents unnecessary keystrokes and keeps your focus moving efficiently.

### Step 4: Toggle a Flag That Requires a Value
Now, in the **Flags Pane**, you'll see a list of flags specific to `jj log`. See [Flags](/jutsu/usage/flags/) for the full set of flag states and indicators.
1. Scroll down using `j`/`k` to find the `revisions` flag (`-r`).
2. Press `Space` to toggle it. The flag is highlighted in **peach** (active), and the dynamic **Command Bar** at the very bottom updates in real-time to append `-r`.
3. Because `-r` requires a value, press `i` to jump into the **Inputs Pane** — see [Inputs](/jutsu/usage/inputs/) for details — and type a revset, e.g. `main`. The Command Bar updates to `jj log -r main`.

:::tip[Active Memory]
As you navigate between panes, your selected category, command, and toggled flags remain highlighted in **sapphire** (selected but unfocused) so you never lose track of your current configuration.
:::

---

## Building the Command String in Real-Time

As you navigate and toggle options, look at the **Command Bar** at the bottom of the screen.

As you make selections, Jutsu dynamically compiles them into a valid terminal-ready shell command:
*   Selecting a command adds `jj <command>`.
*   Toggling a flag appends `--<flag>`.
*   Adding input values inserts them in their syntactically correct positions.

This eliminates syntax guessing. You can see *exactly* what `jj` will run before you execute it.

---

## The Recall State Feature (`r`)

One of the most powerful features in Jutsu is the **Recall State** (`r`):

*   **The Problem:** In command-line interfaces, if you want to run the same command multiple times with slight adjustments (e.g., changing a commit description or tweaking a rebase destination), you have to retype or press up-arrow and edit manually.
*   **The Solution:** Pressing `r` inside the Jutsu composer instantly restores the exact state of your last executed command—all selected subcommands, toggled flags, and text inputs are restored precisely as they were.
*   **Workflow:** Run `jj describe -m "First attempt"`. Press `r` to instantly populate the description input again, change the text, and run it as `jj describe -m "Updated description"`.

---

## Next Steps

Now that you know how to navigate the basic composer, dive into handling parameter values and global configurations:

*   👉 **[Inputs and Global Options](/jutsu/workflows/inputs-and-global-options/)**
