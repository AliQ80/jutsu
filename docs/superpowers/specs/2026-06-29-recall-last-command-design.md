# Recall Last Command — Design Spec

## Context

After running a command in Jitsu, all flag selections and input values reset. The two main use cases this creates friction for: re-running the same command again, or running a slight variation of it. Adding a single-key recall restores the full previous composer state so the user can review, tweak, and re-run without rebuilding from scratch.

## Approach

Single most-recent command snapshot. No history cycling, no persistence across sessions. Covers the dominant use case with minimal complexity; history can be layered on later using the same snapshot struct as its foundation.

## Data Model

Add `commandSnapshot` struct and one pointer field to `mainModel`:

```go
// model.go
type commandSnapshot struct {
    catIdx, cmdIdx, subIdx int
    selectedFlags          map[string]bool   // flag Name → Selected
    inputValues            map[string]string // flag Name → text input value
    argValues              map[string]string // arg Name → text input value
}
```

New field on `mainModel`:
```go
lastCmd *commandSnapshot // nil until first command executes
```

Scroll positions are not saved — they re-clamp naturally on restore via existing `clampIndices()` and `reclampAllScrolls()`.

## Capture Point

In `handleCmdBarKeys` (`model.go`), immediately before `return m, executeCommand(m.cmdText)`.

At this point flags are still selected and inputs still have values — `resetCurrentFlags()` runs later inside `execResultMsg` handling, so the snapshot sees the correct pre-reset state.

Snapshot logic: read `m.catIdx`, `m.cmdIdx`, `m.subIdx`, walk `currentFlags()` to populate `selectedFlags` and `inputValues`, walk `getRequiredArgs()` + any selected flag args for `argValues`.

## Restore Flow (`r` key in composer panes 0–3)

1. If `m.lastCmd == nil` → no-op (no command has been run yet in this session)
2. Set `m.catIdx`, `m.cmdIdx`, `m.subIdx` from snapshot
3. Set flag selections by index through `m.categories` (same pointer pattern as `toggleFlag()`) — `currentFlags()` returns value copies so mutations must go via `m.categories[catIdx].Commands[cmdIdx].Flags[i].Selected` (or `.SubCmds[subIdx].Flags[i].Selected` when subcommands exist)
4. Restore `m.inputs[name].SetValue(...)` and `m.argInputs[name].SetValue(...)` from snapshot maps
5. Call `m.clampIndices()`, `m.layoutViewports()`, `m.reclampAllScrolls()`, `m.refreshDocs()`, rebuild command strings
6. Stay in current pane — user navigates or presses Tab to run

## Key Binding

`r` in `handleComposerKeys` (panes 0–3). Free key, mnemonic (recall/repeat), consistent with existing vim-style bindings.

Does not apply in `focusCmdBar`, `focusOutput`, `focusInputs`, or `focusDocs` — those have their own key handlers and `r` would need explicit handling there to be available, which is out of scope.

## Help Bar

In `view.go`, add `r recall` hint to the composer pane help bar. Show conditionally only when `m.lastCmd != nil` to avoid showing a useless hint before any command has been run.

## Files to Modify

- `model.go` — add `commandSnapshot` struct, `lastCmd` field, snapshot capture in `handleCmdBarKeys`, restore logic in `handleComposerKeys`
- `view.go` — add `r recall` hint in composer help bar (conditional on `m.lastCmd != nil`)

## Verification

1. Run `go run .` in the jitsu directory
2. Navigate to any command, select some flags and fill inputs, run it via Tab → Enter
3. Flags and inputs reset (existing behaviour confirmed)
4. Press `r` — verify composer state is fully restored (correct category/command/subcommand selected, same flags ticked, same input values)
5. Press Tab → Enter — verify the same command re-runs
6. Change one flag after recall, Tab → Enter — verify modified command runs
7. Press `r` before running any command — verify no-op (no crash, no state change)
8. Verify `r recall` hint appears in help bar only after first command run
