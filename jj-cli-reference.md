# jj CLI — Argument Reference

Reference for the `jj` (Jujutsu) command-line tool, organized by whether each
command/subcommand takes a **required** positional argument (`<...>`), only
**optional** arguments (`[...]`), or **no** arguments at all.

Source: [docs.jj-vcs.dev/latest/cli-reference](https://docs.jj-vcs.dev/latest/cli-reference/),
cross-checked against the Arch Linux `jj` man pages (built against 0.40.0)
and the 0.41.0/0.42.0 GitHub release notes. Verified accurate for jj
versions 0.40.0–0.42.0.

## Required argument(s)

| Command | Required argument(s) |
|---|---|
| `jj edit` | `<REVSET\|-r <REVSET>>` |
| `jj bisect run` | `<COMMAND>` (plus required option `--range <REVSETS>`) |
| `jj bookmark create` | `<NAMES>...` |
| `jj bookmark delete` | `<NAMES>...` |
| `jj bookmark forget` | `<NAMES>...` |
| `jj bookmark move` | `<NAMES\|--from <REVSETS>>` |
| `jj bookmark rename` | `<OLD> <NEW>` |
| `jj bookmark set` | `<NAMES>...` |
| `jj bookmark track` | `<BOOKMARK>...` |
| `jj bookmark untrack` | `<BOOKMARK>...` |
| `jj config edit` | `<--user\|--repo\|--workspace>` |
| `jj config get` | `<NAME>` |
| `jj config path` | `<--user\|--repo\|--workspace>` |
| `jj config set` | `<--user\|--repo\|--workspace> <NAME> <VALUE>` |
| `jj config unset` | `<--user\|--repo\|--workspace> <NAME>` |
| `jj file annotate` | `<PATH>` |
| `jj file chmod` | `<MODE> <FILESETS>...` |
| `jj file show` | `<FILESETS>...` |
| `jj file track` | `<FILESETS>...` |
| `jj file untrack` | `<FILESETS>...` |
| `jj git clone` | `<SOURCE>` |
| `jj git remote add` | `<REMOTE> <URL>` |
| `jj git remote remove` | `<REMOTE>` |
| `jj git remote rename` | `<OLD> <NEW>` |
| `jj git remote set-url` | `<REMOTE> <URL>` |
| `jj operation abandon` | `<OPERATIONS>...` |
| `jj operation restore` | `<OPERATION>` |
| `jj operation revert` | `<OPERATION>` |
| `jj operation show` | `<OPERATION>` |
| `jj tag delete` | `<NAMES>...` |
| `jj tag set` | `<NAMES>...` |
| `jj util completion` | `<SHELL>` |
| `jj util exec` | `<COMMAND>` (plus optional `[ARGS]...`) |
| `jj util install-man-pages` | `<PATH>` |
| `jj workspace add` | `<DESTINATION>` |

## Optional argument(s) only

| Command |
|---|
| `jj abandon` |
| `jj absorb` |
| `jj arrange` |
| `jj bookmark advance` |
| `jj bookmark list` |
| `jj commit` |
| `jj config list` |
| `jj describe` |
| `jj diff` |
| `jj diffedit` |
| `jj duplicate` |
| `jj evolog` |
| `jj file list` |
| `jj file search` |
| `jj fix` |
| `jj gerrit upload` |
| `jj git fetch` |
| `jj git init` |
| `jj git push` |
| `jj restore` |
| `jj workspace forget` |
| `jj workspace rename` |

## No arguments

| Command |
|---|
| `jj git colocation disable` |
| `jj git colocation enable` |
| `jj git colocation status` |
| `jj git export` |
| `jj git import` |
| `jj git remote list` |
| `jj git root` |
| `jj redo` |
| `jj root` |
| `jj undo` |
| `jj util backend name` |
| `jj util config-schema` |
| `jj util gc` |
| `jj util markdown-help` |
| `jj util snapshot` |
| `jj version` |
| `jj workspace list` |
| `jj workspace root` |
| `jj workspace update-stale` |

## Notes

- `<...>` = required, `[...]` = optional, `...` = can repeat, `|` = choose one.
- `jj help <COMMAND>` is the authoritative source; this file may drift from
  newer jj releases.
- As of 0.42.0, `jj show` accepts multiple revisions (`[REVSETS]...`),
  showing each one in turn — still optional, just repeatable.
- `jj util backend name` (no arguments) was added in 0.42.0.
- A new global flag, `--no-integrate-operation`, was added in 0.41.0. It
  applies to every command but is a flag, not a positional argument, so it
  doesn't affect the tables above.
- Many commands not listed above (`jj new`, `jj next`, `jj prev`, `jj log`,
  `jj split`, `jj squash`, `jj status`, `jj rebase`, `jj resolve`,
  `jj revert`, `jj sign`, `jj unsign`, `jj simplify-parents`, `jj sparse *`,
  `jj parallelize`, `jj interdiff`, `jj metaedit`) were not individually
  re-verified against the man pages in this pass. (`jj show` is the
  exception — it's confirmed above as optional/repeatable.) Most follow the
  same pattern of defaulting `REVSETS`/`FILESETS` to `[optional]`, but
  should be spot-checked with `jj help <command>` before relying on this
  list for those specifically.
