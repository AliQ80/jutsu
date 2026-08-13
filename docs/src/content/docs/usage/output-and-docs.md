---
title: Output & Docs
description: Pane navigation, scrolling, enlargement, horizontal panning, and clipboard copying.
---

The **Output** and **Docs** panes give you live feedback as you compose and run commands.

## Jumping to the Output and Docs Panes

Press `o` to jump directly to the **Output Pane**, or `d` to jump directly to the **Docs Pane**, from any composer pane or the Command Bar. Both shortcuts are unavailable while focused in the Inputs Pane, and each is blocked while the *other* pane is enlarged — `o` doesn't work while Docs is enlarged, and `d` doesn't work while Output is enlarged. Pressing `o` while already in the Output Pane (or `d` while already in Docs) exits back to the composer instead.

## Scrolling

While the **Output** or **Docs** pane is focused, use `j`/`k` (or `↓`/`↑`) to scroll vertically through its contents — long command help text or executed command output.

## Panning the Output Pane Horizontally

The **Output Pane** additionally supports horizontal panning: while it is focused, use `h`/`l` (or `←`/`→`) to shift long lines left and right, which is useful for wide command output that would otherwise wrap or run off-screen. The **Docs Pane** only scrolls vertically; it does not support horizontal panning.

## Enlarging a Pane

Press `O` to enlarge the **Output Pane**, or `D` to enlarge the **Docs Pane**, to full-screen. Press the same key again, or `Esc`, to restore the normal layout.

## Copying to the Clipboard

Press `c` while a pane is focused to copy its text content to the clipboard. ANSI color codes are stripped from the copied text first. Clipboard copying shells out to `wl-copy`, so it depends on having a Wayland session with `wl-copy` available — it is not a universal cross-platform clipboard integration.

## Related Topics

- **[Navigation](/jutsu/usage/navigation/)** — the full set of keyboard shortcuts, including `o`/`d`/`O`/`D`/`c`.
- **[Interface](/jutsu/usage/interface/)** — how the Docs and Output panes fit into Jutsu's overall layout.
- **[Running Commands](/jutsu/usage/execution/)** — how executed commands stream their results into the Output Pane.
