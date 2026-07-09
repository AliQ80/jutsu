package main

type Flag struct {
	Name             string
	Description      string
	Value            string
	InputType        string // e.g. "REVSET", "PATH", "TEMPLATE" — populated by gen-descriptions
	Selected         bool
	RequiresInput    bool
	NeedsQuotes      bool
	Mandatory        bool     // always selected, cannot be deselected
	ConflictingFlags []string // Values of flags this flag cannot be combined with
}

type Arg struct {
	Name        string
	Description string
	Variadic    bool // true when jj help shows [NAME]... — populated by gen-descriptions
	Required    bool // true for <ARG> (angle-bracket) — must be supplied
}

type SubCommand struct {
	Name              string
	Alias             string // e.g. "a" for bookmark advance — populated by gen-descriptions
	Description       string
	Summary           string // one-line summary shown in the parent command's Subcommands listing — populated by gen-descriptions
	Value             string // overrides Name in the built command string when set
	Args              []Arg
	Flags             []Flag
	RequiredFlagGroup []string // Values of flags where at least one must be selected
	RequiredUsage     string   // required-flag portion of the Usage line, e.g. "<--onto|--insert-after|--insert-before>" — populated by gen-descriptions
}

type Command struct {
	Name              string
	Alias             string // e.g. "st" for status — populated by gen-descriptions
	Description       string
	Args              []Arg
	SubCmds           []SubCommand
	Flags             []Flag
	RequiredFlagGroup []string // Values of flags where at least one must be selected
	RequiredUsage     string   // required-flag portion of the Usage line, e.g. "--revision <REVSETS> <--onto|...>" — populated by gen-descriptions
}

type Category struct {
	Name     string
	Commands []Command
}

func loadCategories() []Category {
	return []Category{
		{
			Name: "View",
			Commands: []Command{
				{
					Name:        "status",
					Alias:       "st",
					Description: "Show high-level repo status [default alias: st]\n\nThis includes:\n\n* The working copy commit and its parents, and a summary of the changes in the working copy (compared to the merged parents)\n\n* Conflicts in the working copy\n\n* [Conflicted bookmarks]\n\nNote: You can use `jj diff --summary -r <rev>` to see the changed files for a specific revision.\n\n[Conflicted bookmarks]: https://docs.jj-vcs.dev/latest/bookmarks/#conflicts",
					Args: []Arg{
						{Name: "FILESETS", Description: "Restrict the status display to these paths", Variadic: true},
					},
					Flags: []Flag{},
				},
				{
					Name:        "log",
					Description: "Show revision history\n\nRenders a graphical view of the project's history, ordered with children before parents. By default, the output only includes mutable revisions, along with some additional revisions for context. Use `jj log -r ::` to see all revisions. See [`jj help -k revsets`] for information about the syntax.\n\n[`jj help -k revsets`]: https://docs.jj-vcs.dev/latest/revsets/\n\nSpans of revisions that are not included in the graph per `--revisions` are rendered as a synthetic node labeled \"(elided revisions)\".\n\nThe working-copy commit is indicated by a `@` symbol in the graph. [Immutable revisions] have a `◆` symbol. Other commits have a `○` symbol. All of these symbols can be [customized].\n\n[Immutable revisions]: https://docs.jj-vcs.dev/latest/config/#set-of-immutable-commits\n\n[customized]: https://docs.jj-vcs.dev/latest/config/#node-style",
					Args: []Arg{
						{Name: "FILESETS", Description: "Show revisions modifying the given paths", Variadic: true},
					},
					Flags: []Flag{
						{Name: "revisions", Description: "Which revisions to show\n\nIf no paths nor revisions are specified, this defaults to the `revsets.log` setting.", Value: "-r", RequiresInput: true, InputType: "REVSETS"},
						{Name: "limit", Description: "Limit number of revisions to show\n\nApplied after revisions are filtered and reordered topologically, but before being reversed.", Value: "-n", RequiresInput: true, InputType: "LIMIT"},
						{Name: "reversed", Description: "Show revisions in the opposite order (older revisions first)", Value: "--reversed"},
						{Name: "no-graph", Description: "Don't show the graph, show a flat list of revisions", Value: "--no-graph"},
						{Name: "count", Description: "Print the number of commits instead of showing them", Value: "--count"},
						{Name: "template", Description: "Render each revision using the given template\n\nRun `jj log -T` to list the built-in templates.\n\nYou can also specify arbitrary template expressions using the [built-in keywords]. See [`jj help -k templates`] for more information.\n\nIf not specified, this defaults to the `templates.log` setting.\n\n[built-in keywords]: https://docs.jj-vcs.dev/latest/templates/#commit-keywords\n\n[`jj help -k templates`]: https://docs.jj-vcs.dev/latest/templates/", Value: "-T", RequiresInput: true, InputType: "TEMPLATE"},
						{Name: "patch", Description: "Show patch", Value: "-p"},
						{Name: "summary", Description: "For each path, show only whether it was modified, added, or deleted", Value: "-s", ConflictingFlags: []string{"--stat", "--types", "--name-only"}},
						{Name: "stat", Description: "Show a histogram of the changes", Value: "--stat", ConflictingFlags: []string{"-s", "--types", "--name-only"}},
						{Name: "types", Description: "For each path, show only its type before and after\n\nThe diff is shown as two letters. The first letter indicates the type before and the second letter indicates the type after. '-' indicates that the path was not present, 'F' represents a regular file, `L' represents a symlink, 'C' represents a conflict, and 'G' represents a Git submodule.", Value: "--types", ConflictingFlags: []string{"-s", "--stat", "--name-only"}},
						{Name: "name-only", Description: "For each path, show only its path\n\nTypically useful for shell commands like: `jj diff -r @- --name-only | xargs perl -pi -e's/OLD/NEW/g`", Value: "--name-only", ConflictingFlags: []string{"-s", "--stat", "--types"}},
						{Name: "git", Description: "Show a Git-format diff", Value: "--git", ConflictingFlags: []string{"--color-words"}},
						{Name: "color-words", Description: "Show a word-level diff with changes indicated only by color", Value: "--color-words", ConflictingFlags: []string{"--git"}},
						{Name: "ignore-all-space", Description: "Ignore whitespace when comparing lines", Value: "--ignore-all-space", ConflictingFlags: []string{"--ignore-space-change"}},
						{Name: "ignore-space-change", Description: "Ignore changes in amount of whitespace when comparing lines", Value: "--ignore-space-change", ConflictingFlags: []string{"--ignore-all-space"}},
						{Name: "tool", Description: "Generate diff by external command\n\nA builtin format can also be specified as `:<name>`. For example, `--tool=:git` is equivalent to `--git`.", Value: "--tool", RequiresInput: true, InputType: "TOOL"},
						{Name: "context", Description: "Number of lines of context to show", Value: "--context", RequiresInput: true, InputType: "CONTEXT"},
					},
				},
				{
					Name:        "show",
					Description: "Show revision metadata and diff",
					Args: []Arg{
						{Name: "REVSETS", Description: "Show changes in these revisions, compared to their parent(s) [default: @] [aliases: -r]", Variadic: true},
					},
					Flags: []Flag{
						{Name: "template", Description: "Render each revision using the given template\n\nYou can specify arbitrary template expressions using the [built-in keywords]. See [`jj help -k templates`] for more information.\n\n[built-in keywords]: https://docs.jj-vcs.dev/latest/templates/#commit-keywords\n\n[`jj help -k templates`]: https://docs.jj-vcs.dev/latest/templates/", Value: "-T", RequiresInput: true, InputType: "TEMPLATE"},
						{Name: "no-patch", Description: "Do not show the patch", Value: "--no-patch"},
						{Name: "summary", Description: "For each path, show only whether it was modified, added, or deleted", Value: "-s", ConflictingFlags: []string{"--stat", "--types", "--name-only"}},
						{Name: "stat", Description: "Show a histogram of the changes", Value: "--stat", ConflictingFlags: []string{"-s", "--types", "--name-only"}},
						{Name: "types", Description: "For each path, show only its type before and after\n\nThe diff is shown as two letters. The first letter indicates the type before and the second letter indicates the type after. '-' indicates that the path was not present, 'F' represents a regular file, `L' represents a symlink, 'C' represents a conflict, and 'G' represents a Git submodule.", Value: "--types", ConflictingFlags: []string{"-s", "--stat", "--name-only"}},
						{Name: "name-only", Description: "For each path, show only its path\n\nTypically useful for shell commands like: `jj diff -r @- --name-only | xargs perl -pi -e's/OLD/NEW/g`", Value: "--name-only", ConflictingFlags: []string{"-s", "--stat", "--types"}},
						{Name: "git", Description: "Show a Git-format diff", Value: "--git", ConflictingFlags: []string{"--color-words"}},
						{Name: "color-words", Description: "Show a word-level diff with changes indicated only by color", Value: "--color-words", ConflictingFlags: []string{"--git"}},
						{Name: "ignore-all-space", Description: "Ignore whitespace when comparing lines", Value: "-w", ConflictingFlags: []string{"-b"}},
						{Name: "ignore-space-change", Description: "Ignore changes in amount of whitespace when comparing lines", Value: "-b", ConflictingFlags: []string{"-w"}},
						{Name: "tool", Description: "Generate diff by external command\n\nA builtin format can also be specified as `:<name>`. For example, `--tool=:git` is equivalent to `--git`.", Value: "--tool", RequiresInput: true, InputType: "TOOL"},
						{Name: "context", Description: "Number of lines of context to show", Value: "--context", RequiresInput: true, InputType: "CONTEXT"},
					},
				},
				{
					Name:        "diff",
					Description: "Compare file contents between two revisions\n\nWith the `-r` option, shows the changes compared to the parent revision. If there are several parent revisions (i.e., the given revision is a merge), then they will be merged and the changes from the result to the given revision will be shown.\n\nWith the `--from` and/or `--to` options, shows the difference from/to the given revisions. If either is left out, it defaults to the working-copy commit. For example, `jj diff --from main` shows the changes from \"main\" (perhaps a bookmark name) to the working-copy commit.\n\nIf no option is specified, it defaults to `-r @`.",
					Args: []Arg{
						{Name: "FILESETS", Description: "Restrict the diff to these paths", Variadic: true},
					},
					Flags: []Flag{
						{Name: "revisions", Description: "Show changes in these revisions\n\nIf there are multiple revisions, then the total diff for all of them will be shown. For example, if you have a linear chain of revisions A..D, then `jj diff -r B::D` equals `jj diff --from A --to D`. Multiple heads and/or roots are supported, but gaps in the revset are not supported (e.g. `jj diff -r 'A|C'` in a linear chain A..C).\n\nIf a revision is a merge commit, this shows changes *from* the automatic merge of the contents of all of its parents *to* the contents of the revision itself.\n\nIf none of `-r`, `-f`, or `-t` is provided, then the default is `-r @`.", Value: "-r", RequiresInput: true, ConflictingFlags: []string{"--from", "--to"}, InputType: "REVSETS"},
						{Name: "from", Description: "Show changes from this revision\n\nIf none of `-r`, `-f`, or `-t` is provided, then the default is `-r @`.", Value: "--from", RequiresInput: true, ConflictingFlags: []string{"-r"}, InputType: "REVSET"},
						{Name: "to", Description: "Show changes to this revision\n\nIf none of `-r`, `-f`, or `-t` is provided, then the default is `-r @`.", Value: "--to", RequiresInput: true, ConflictingFlags: []string{"-r"}, InputType: "REVSET"},
						{Name: "summary", Description: "For each path, show only whether it was modified, added, or deleted", Value: "-s", ConflictingFlags: []string{"--stat", "--types", "--name-only"}},
						{Name: "stat", Description: "Show a histogram of the changes", Value: "--stat", ConflictingFlags: []string{"-s", "--types", "--name-only"}},
						{Name: "types", Description: "For each path, show only its type before and after\n\nThe diff is shown as two letters. The first letter indicates the type before and the second letter indicates the type after. '-' indicates that the path was not present, 'F' represents a regular file, `L' represents a symlink, 'C' represents a conflict, and 'G' represents a Git submodule.", Value: "--types", ConflictingFlags: []string{"-s", "--stat", "--name-only"}},
						{Name: "name-only", Description: "For each path, show only its path\n\nTypically useful for shell commands like: `jj diff -r @- --name-only | xargs perl -pi -e's/OLD/NEW/g`", Value: "--name-only", ConflictingFlags: []string{"-s", "--stat", "--types"}},
						{Name: "git", Description: "Show a Git-format diff", Value: "--git", ConflictingFlags: []string{"--color-words"}},
						{Name: "color-words", Description: "Show a word-level diff with changes indicated only by color", Value: "--color-words", ConflictingFlags: []string{"--git"}},
						{Name: "ignore-all-space", Description: "Ignore whitespace when comparing lines", Value: "-w", ConflictingFlags: []string{"-b"}},
						{Name: "ignore-space-change", Description: "Ignore changes in amount of whitespace when comparing lines", Value: "-b", ConflictingFlags: []string{"-w"}},
						{Name: "template", Description: "Render each file diff entry using the given template\n\nAll 0-argument methods of the [`TreeDiffEntry` type] are available as keywords in the template expression. See [`jj help -k templates`] for more information.\n\n[`TreeDiffEntry` type]: https://docs.jj-vcs.dev/latest/templates/#treediffentry-type\n\n[`jj help -k templates`]: https://docs.jj-vcs.dev/latest/templates/", Value: "-T", RequiresInput: true, InputType: "TEMPLATE"},
						{Name: "tool", Description: "Generate diff by external command\n\nA builtin format can also be specified as `:<name>`. For example, `--tool=:git` is equivalent to `--git`.", Value: "--tool", RequiresInput: true, InputType: "TOOL"},
						{Name: "context", Description: "Number of lines of context to show", Value: "--context", RequiresInput: true, InputType: "CONTEXT"},
					},
				},
				{
					Name:        "evolog",
					Alias:       "evolution-log",
					Description: "Show how a change has evolved over time\n\nLists the previous commits which a change has pointed to. The current commit of a change evolves when the change is updated, rebased, etc.",
					Flags: []Flag{
						{Name: "revisions", Description: "Follow changes from these revisions\n\nDefault value: `@`", Value: "-r", RequiresInput: true, InputType: "REVSETS"},
						{Name: "limit", Description: "Limit number of revisions to show\n\nApplied after revisions are reordered topologically, but before being reversed.", Value: "-n", RequiresInput: true, InputType: "LIMIT"},
						{Name: "reversed", Description: "Show revisions in the opposite order (older revisions first)", Value: "--reversed"},
						{Name: "no-graph", Description: "Don't show the graph, show a flat list of revisions", Value: "--no-graph"},
						{Name: "template", Description: "Render each revision using the given template\n\nAll 0-argument methods of the [`CommitEvolutionEntry` type] are available as keywords in the template expression. See [`jj help -k templates`] for more information.\n\nIf not specified, this defaults to the `templates.evolog` setting.\n\n[`CommitEvolutionEntry` type]: https://docs.jj-vcs.dev/latest/templates/#commitevolutionentry-type\n\n[`jj help -k templates`]: https://docs.jj-vcs.dev/latest/templates/", Value: "-T", RequiresInput: true, InputType: "TEMPLATE"},
						{Name: "patch", Description: "Show patch compared to the previous version of this change\n\nIf the previous version has different parents, it will be temporarily rebased to the parents of the new version, so the diff is not contaminated by unrelated changes.", Value: "-p"},
						{Name: "summary", Description: "For each path, show only whether it was modified, added, or deleted", Value: "-s", ConflictingFlags: []string{"--stat", "--types", "--name-only"}},
						{Name: "stat", Description: "Show a histogram of the changes", Value: "--stat", ConflictingFlags: []string{"-s", "--types", "--name-only"}},
						{Name: "types", Description: "For each path, show only its type before and after\n\nThe diff is shown as two letters. The first letter indicates the type before and the second letter indicates the type after. '-' indicates that the path was not present, 'F' represents a regular file, `L' represents a symlink, 'C' represents a conflict, and 'G' represents a Git submodule.", Value: "--types", ConflictingFlags: []string{"-s", "--stat", "--name-only"}},
						{Name: "name-only", Description: "For each path, show only its path\n\nTypically useful for shell commands like: `jj diff -r @- --name-only | xargs perl -pi -e's/OLD/NEW/g`", Value: "--name-only", ConflictingFlags: []string{"-s", "--stat", "--types"}},
						{Name: "git", Description: "Show a Git-format diff", Value: "--git", ConflictingFlags: []string{"--color-words"}},
						{Name: "color-words", Description: "Show a word-level diff with changes indicated only by color", Value: "--color-words", ConflictingFlags: []string{"--git"}},
						{Name: "ignore-all-space", Description: "Ignore whitespace when comparing lines", Value: "--ignore-all-space", ConflictingFlags: []string{"--ignore-space-change"}},
						{Name: "ignore-space-change", Description: "Ignore changes in amount of whitespace when comparing lines", Value: "--ignore-space-change", ConflictingFlags: []string{"--ignore-all-space"}},
						{Name: "tool", Description: "Generate diff by external command\n\nA builtin format can also be specified as `:<name>`. For example, `--tool=:git` is equivalent to `--git`.", Value: "--tool", RequiresInput: true, InputType: "TOOL"},
						{Name: "context", Description: "Number of lines of context to show", Value: "--context", RequiresInput: true, InputType: "CONTEXT"},
					},
				},
				{
					Name:        "interdiff",
					Description: "Show differences between the diffs of two revisions\n\nThis is like running `jj diff -r` on each change, then comparing those results. It answers: \"How do the modifications introduced by revision A differ from the modifications introduced by revision B?\"\n\nFor example, if two changes both add a feature but implement it differently, `jj interdiff --from @- --to other` shows what one implementation adds or removes that the other doesn't.\n\nA common use of this command is to compare how a change has changed since the last push to a remote:\n\n```sh $ jj interdiff --from push-xyz@origin --to push-xyz ```\n\nThis command is different from `jj diff --from A --to B`, which compares file contents directly. `interdiff` compares what the changes do in terms of their patches, rather than their file contents. This makes a difference when the two revisions have different parents: `jj diff --from A --to B` will include the changes between their parents while `jj interdiff --from A --to B` will not.\n\nTechnically, this works by rebasing `--from` onto `--to`'s parents and comparing the result to `--to`.\n\nTo see the changes throughout the whole evolution of a change instead of between just two revisions, use `jj evolog -p` instead.",
					Args: []Arg{
						{Name: "FILESETS", Description: "Restrict the diff to these paths", Variadic: true},
					},
					Flags: []Flag{
						{Name: "from", Description: "The first revision to compare (default: @)", Value: "--from", RequiresInput: true, Mandatory: true, Selected: true, InputType: "REVSET"},
						{Name: "to", Description: "The second revision to compare (default: @)", Value: "--to", RequiresInput: true, Mandatory: true, Selected: true, InputType: "REVSET"},
						{Name: "summary", Description: "For each path, show only whether it was modified, added, or deleted", Value: "-s", ConflictingFlags: []string{"--stat", "--types", "--name-only"}},
						{Name: "stat", Description: "Show a histogram of the changes", Value: "--stat", ConflictingFlags: []string{"-s", "--types", "--name-only"}},
						{Name: "types", Description: "For each path, show only its type before and after\n\nThe diff is shown as two letters. The first letter indicates the type before and the second letter indicates the type after. '-' indicates that the path was not present, 'F' represents a regular file, `L' represents a symlink, 'C' represents a conflict, and 'G' represents a Git submodule.", Value: "--types", ConflictingFlags: []string{"-s", "--stat", "--name-only"}},
						{Name: "name-only", Description: "For each path, show only its path\n\nTypically useful for shell commands like: `jj diff -r @- --name-only | xargs perl -pi -e's/OLD/NEW/g`", Value: "--name-only", ConflictingFlags: []string{"-s", "--stat", "--types"}},
						{Name: "git", Description: "Show a Git-format diff", Value: "--git", ConflictingFlags: []string{"--color-words"}},
						{Name: "color-words", Description: "Show a word-level diff with changes indicated only by color", Value: "--color-words", ConflictingFlags: []string{"--git"}},
						{Name: "ignore-all-space", Description: "Ignore whitespace when comparing lines", Value: "-w", ConflictingFlags: []string{"-b"}},
						{Name: "ignore-space-change", Description: "Ignore changes in amount of whitespace when comparing lines", Value: "-b", ConflictingFlags: []string{"-w"}},
						{Name: "tool", Description: "Generate diff by external command\n\nA builtin format can also be specified as `:<name>`. For example, `--tool=:git` is equivalent to `--git`.", Value: "--tool", RequiresInput: true, InputType: "TOOL"},
						{Name: "context", Description: "Number of lines of context to show", Value: "--context", RequiresInput: true, InputType: "CONTEXT"},
					},
				},
				{
					Name:        "file",
					Description: "File operations",
					SubCmds: []SubCommand{
						{
							Summary: "List files in a revision", Name: "list",
							Description: "List files in a revision",
							Args: []Arg{
								{Name: "FILESETS", Description: "Only list files matching these prefixes (instead of all files)", Variadic: true},
							},
							Flags: []Flag{
								{Name: "revision", Description: "The revision to list files in\n\nDefault value: `@`", Value: "-r", RequiresInput: true, InputType: "REVSET"},
								{Name: "template", Description: "Render each file entry using the given template\n\nAll 0-argument methods of the [`TreeEntry` type] are available as keywords in the template expression. See [`jj help -k templates`] for more information.\n\n[`TreeEntry` type]: https://docs.jj-vcs.dev/latest/templates/#treeentry-type\n\n[`jj help -k templates`]: https://docs.jj-vcs.dev/latest/templates/", Value: "-T", RequiresInput: true, InputType: "TEMPLATE"},
							},
						},
						{
							Summary: "Print contents of files in a revision", Name: "show",
							Description: "Print contents of files in a revision\n\nIf the given path is a directory, files in the directory will be visited recursively.",
							Args: []Arg{
								{Name: "FILESETS", Description: "Paths to print", Variadic: true, Required: true},
							},
							Flags: []Flag{
								{Name: "revision", Description: "The revision to get the file contents from\n\nDefault value: `@`", Value: "-r", RequiresInput: true, InputType: "REVSET"},
								{Name: "template", Description: "Render each file metadata using the given template\n\nAll 0-argument methods of the [`TreeEntry` type] are available as keywords in the template expression. See [`jj help -k templates`] for more information.\n\nIf not specified, this defaults to the `templates.file_show` setting.\n\n[`TreeEntry` type]: https://docs.jj-vcs.dev/latest/templates/#treeentry-type\n\n[`jj help -k templates`]: https://docs.jj-vcs.dev/latest/templates/", Value: "-T", RequiresInput: true, InputType: "TEMPLATE"},
							},
						},
						{
							Summary: "Show the source change for each line of the target file", Name: "annotate",
							Description: "Show the source change for each line of the target file.\n\nAnnotates a revision line by line. Each line includes the source change that introduced the associated line. A path to the desired file must be provided.",
							Args: []Arg{
								{Name: "PATH", Description: "the file to annotate", Required: true},
							},
							Flags: []Flag{
								{Name: "revision", Description: "an optional revision to start at", Value: "-r", RequiresInput: true, InputType: "REVSET"},
								{Name: "template", Description: "Render each line using the given template\n\nAll 0-argument methods of the [`AnnotationLine` type] are available as keywords in the template expression. See [`jj help -k templates`] for more information.\n\nIf not specified, this defaults to the `templates.file_annotate` setting.\n\n[`AnnotationLine` type]: https://docs.jj-vcs.dev/latest/templates/#annotationline-type\n\n[`jj help -k templates`]: https://docs.jj-vcs.dev/latest/templates/", Value: "-T", RequiresInput: true, InputType: "TEMPLATE"},
							},
						},
						{
							Summary: "Search for content in files", Name: "search",
							Description: "Search for content in files\n\nLists files containing the specified pattern.\n\nThis is an early version of the command. It only supports glob matching for now, it doesn't search files concurrently, and it doesn't indicate where in the file the match was found.",
							Args: []Arg{
								{Name: "FILESETS", Description: "Only search files matching these prefixes (instead of all files)", Variadic: true},
							},
							Flags: []Flag{
								{Name: "pattern", Description: "The pattern to search for in a single line\n\nIt is a [string pattern syntax] like `kind:pattern`.  The kind defaults to regex when omitted.\n\nIf it is a glob pattern, the whole line must match the pattern, so you may want to pass something like `--pattern 'glob:*foo*'`.\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Value: "--pattern", RequiresInput: true, Mandatory: true, Selected: true, InputType: "PATTERN"},
								{Name: "revision", Description: "The revision to search files in\n\nDefault value: `@`", Value: "-r", RequiresInput: true, InputType: "REVSET"},
							},
						},
					},
				},
				{
					Name:        "root",
					Description: "Show the current workspace root directory (shortcut for `jj workspace root`)",
					Flags:       []Flag{},
				},
			},
		},
		{
			Name: "Change",
			Commands: []Command{
				{
					Name:        "commit",
					Alias:       "ci",
					Description: "Update the description and create a new change on top [default alias: ci]\n\nWhen called without path arguments or `--interactive`, `jj commit` is equivalent to `jj describe` followed by `jj new`.\n\nWhen using `--interactive` or path arguments, the selected changes stay in the current commit while the remaining changes are moved to a new working-copy commit on top. This is very similar to `jj split`. Differences include:\n\n* `jj commit` is not interactive by default (it selects all changes).\n\n* `jj commit` doesn't have a `-r` option. It always acts on the working-copy commit (@).\n\n* `jj split` (without `-o`/`-A`/`-B`) will move bookmarks forward from the old change to the child change. `jj commit` doesn't move bookmarks forward.\n\n* `jj split` allows you to move the selected changes to a different destination with `-o`/`-A`/`-B`.",
					Args: []Arg{
						{Name: "FILESETS", Description: "Put these paths in the current commit", Variadic: true},
					},
					Flags: []Flag{
						{Name: "message", Description: "The change description to use (don't open editor)", Value: "-m", RequiresInput: true, NeedsQuotes: true, Mandatory: true, Selected: true, InputType: "MESSAGE"},
					},
				},
				{
					Name:        "new",
					Description: "Create a new, empty change and (by default) edit it in the working copy\n\nBy default, `jj` will edit the new change, making the [working copy] represent the new commit. This can be avoided with `--no-edit`.\n\nNote that you can create a merge commit by specifying multiple revisions as argument. For example, `jj new @ main` will create a new commit with the working copy and the `main` bookmark as parents.\n\n[working copy]: https://docs.jj-vcs.dev/latest/working-copy/",
					Args: []Arg{
						{Name: "REVSETS", Description: "Parent(s) of the new change [default: @] [aliases: -o, -r]", Variadic: true},
					},
					Flags: []Flag{
						{Name: "message", Description: "The change description to use", Value: "-m", RequiresInput: true, NeedsQuotes: true, InputType: "MESSAGE"},
						{Name: "insert-after", Description: "Insert the new change after the given commit(s)\n\nExample: `jj new --insert-after A` creates a new change between `A` and its children:\n\n```text B   C \\ / B   C   =>    @ \\ /          | A           A ```\n\nSpecifying `--insert-after` multiple times will relocate all children of the given commits.\n\nExample: `jj new --insert-after A --insert-after X` creates a change with `A` and `X` as parents, and rebases all children on top of the new change:\n\n```text B   Y \\ / B  Y    =>    @ |  |         / \\ A  X        A   X ```", Value: "--insert-after", RequiresInput: true, InputType: "REVSETS"},
						{Name: "insert-before", Description: "Insert the new change before the given commit(s)\n\nExample: `jj new --insert-before C` creates a new change between `C` and its parents:\n\n```text C | C     =>     @ / \\          / \\ A   B        A   B ```\n\n`--insert-after` and `--insert-before` can be combined.\n\nExample: `jj new --insert-after A --insert-before D`:\n\n```text\n\nD            D |           / \\ C          |   C |    =>    @   | B          |   B |           \\ / A            A ```\n\nSimilar to `--insert-after`, you can specify `--insert-before` multiple times.", Value: "--insert-before", RequiresInput: true, InputType: "REVSETS"},
						{Name: "no-edit", Description: "Do not edit the newly created change", Value: "--no-edit"},
					},
				},
				{
					Name:        "describe",
					Alias:       "desc",
					Description: "Update the change description or other metadata [default alias: desc]\n\nStarts an editor to let you edit the description of changes. The editor will be $EDITOR, or `nano` if that's not defined (`Notepad` on Windows).",
					Args: []Arg{
						{Name: "REVSETS", Description: "The revision(s) whose description to edit (default: @) [aliases: -r]", Variadic: true},
					},
					Flags: []Flag{
						{Name: "message", Description: "The change description to use (don't open editor)\n\nIf multiple revisions are specified, the same description will be used for all of them.", Value: "-m", RequiresInput: true, NeedsQuotes: true, Mandatory: true, Selected: true, InputType: "MESSAGE"},
						{Name: "revision", Description: "Describe this revision instead of @", Value: "-r", RequiresInput: true},
					},
				},
				{
					Name:        "edit",
					Description: "Sets the specified revision as the working-copy revision\n\nNote: it is [generally recommended] to instead use `jj new` and `jj squash`.\n\n[generally recommended]: https://docs.jj-vcs.dev/latest/FAQ#how-do-i-resume-working-on-an-existing-change",
					Args: []Arg{
						{Name: "REVSET", Description: "The commit to edit [aliases: -r]", Required: true},
					},
					Flags: []Flag{},
				},
				{
					Name:        "next",
					Description: "Move the working-copy commit to the child revision\n\nThe command creates a new empty working copy revision that is the child of a descendant `offset` revisions ahead of the parent of the current working copy.\n\nFor example, when the offset is 1:\n\n```text D        D @ |        |/ C @  =>  C |/       | B        B ```\n\nIf `--edit` is passed, the working copy revision is changed to the child of the current working copy revision.\n\n```text D        D |        | C        C |        | B   =>   @ |        | @        A ```",
					Args: []Arg{
						{Name: "OFFSET", Description: "How many revisions to move forward. Advances to the next child by default\n\nDefault value: `1`"},
					},
					Flags: []Flag{
						{Name: "OFFSET", Description: "How many revisions to move forward [default: 1]", Value: "", RequiresInput: true},
						{Name: "edit", Description: "Instead of creating a new working-copy commit on top of the target commit (like `jj new`), edit the target commit directly (like `jj edit`)\n\nTakes precedence over config in `ui.movement.edit`; i.e. will negate `ui.movement.edit = false`", Value: "--edit", ConflictingFlags: []string{"--no-edit"}},
						{Name: "no-edit", Description: "The inverse of `--edit`\n\nTakes precedence over config in `ui.movement.edit`; i.e. will negate `ui.movement.edit = true`", Value: "--no-edit", ConflictingFlags: []string{"--edit"}},
						{Name: "conflict", Description: "Jump to the next conflicted descendant", Value: "--conflict"},
					},
				},
				{
					Name:        "prev",
					Description: "Change the working copy revision relative to the parent revision\n\nThe command creates a new empty working copy revision that is the child of an ancestor `offset` revisions behind the parent of the current working copy.\n\nFor example, when the offset is 1:\n\n```text D @      D |/       | A   =>   A @ |        |/ B        B ```\n\nIf `--edit` is passed, the working copy revision is changed to the parent of the current working copy revision.\n\n```text D @      D |/       | C   =>   @ |        | B        B |        | A        A ```",
					Args: []Arg{
						{Name: "OFFSET", Description: "How many revisions to move backward. Moves to the parent by default\n\nDefault value: `1`"},
					},
					Flags: []Flag{
						{Name: "OFFSET", Description: "How many revisions to go back [default: 1]", Value: "", RequiresInput: true},
						{Name: "edit", Description: "Edit the parent directly, instead of moving the working-copy commit\n\nTakes precedence over config in `ui.movement.edit`; i.e. will negate `ui.movement.edit = false`", Value: "--edit", ConflictingFlags: []string{"--no-edit"}},
						{Name: "no-edit", Description: "The inverse of `--edit`\n\nTakes precedence over config in `ui.movement.edit`; i.e. will negate `ui.movement.edit = true`", Value: "--no-edit", ConflictingFlags: []string{"--edit"}},
						{Name: "conflict", Description: "Jump to the previous conflicted ancestor", Value: "--conflict"},
					},
				},
				{
					Name:        "resolve",
					Description: "Resolve conflicted files with an external merge tool\n\nOnly conflicts that can be resolved with a 3-way merge are supported. See docs for merge tool configuration instructions. External merge tools will be invoked for each conflicted file one-by-one until all conflicts are resolved. To stop resolving conflicts, exit the merge tool without making any changes.\n\nNote that conflicts can also be resolved without using this command. You may edit the conflict markers in the conflicted file directly with a text editor.",
					Args: []Arg{
						{Name: "FILESETS", Description: "Only resolve conflicts in these paths. You can use the `--list` argument to find paths to use here", Variadic: true},
					},
					Flags: []Flag{
						{Name: "list", Description: "Instead of resolving conflicts, list all the conflicts", Value: "--list"},
						{Name: "revision", Description: "Default value: `@`", Value: "-r", RequiresInput: true, InputType: "REVSET"},
					},
				},
				{
					Name:        "restore",
					Description: "Restore paths from another revision\n\nThat means that the paths get the same content in the destination (`--into`) as they had in the source (`--from`). This is typically used for undoing changes to some paths in the working copy (`jj restore <paths>`).\n\nIf only one of `--from` or `--into` is specified, the other one defaults to the working copy.\n\nWhen neither `--from` nor `--into` is specified, the command restores into the working copy from its parent(s). `jj restore` without arguments is similar to `jj abandon`, except that it leaves an empty revision with its description and other metadata preserved.\n\nSee `jj diffedit` if you'd like to restore portions of files rather than entire files.",
					Args: []Arg{
						{Name: "FILESETS", Description: "Restore only these paths (instead of all paths)", Variadic: true},
					},
					Flags: []Flag{
						{Name: "from", Description: "Revision to restore from (source)", Value: "--from", RequiresInput: true, InputType: "REVSET"},
						{Name: "into", Description: "Revision to restore into (destination)", Value: "--into", RequiresInput: true, InputType: "REVSET"},
						{Name: "changes-in", Description: "Undo the changes in a revision as compared to the merge of its parents.\n\nThis undoes the changes that can be seen with `jj diff -r REVSET`. If `REVSET` only has a single parent, this option is equivalent to `jj restore --into REVSET --from REVSET-`.\n\nThe default behavior of `jj restore` is equivalent to `jj restore --changes-in @`.", Value: "--changes-in", RequiresInput: true, InputType: "REVSET"},
						{Name: "restore-descendants", Description: "Preserve the content (not the diff) when rebasing descendants", Value: "--restore-descendants"},
					},
				},
				{
					Name:        "fix",
					Description: "Update files with formatting fixes or other changes\n\nThe primary use case for this command is to apply the results of automatic code formatting tools to revisions that may not be properly formatted yet. It can also be used to modify files with other tools like `sed` or `sort`.\n\nThe modification made by `jj fix` can be reviewed by `jj op show -p`.\n\n### How it works\n\nThe changed files in the given revisions will be updated with any fixes determined by passing their file content through any external tools the user has configured for those files. Descendants will also be updated by passing their versions of the same files through the same tools, which will ensure that the fixes are not lost. This will never result in new conflicts. Files with existing conflicts will be updated on all sides of the conflict, which can potentially increase or decrease the number of conflict markers.\n\n### Deduplication\n\nWhen fixing multiple commits, if the same file content appears at the same path in different commits, the tool is run only once and the result is reused. This means that tools used with `jj fix` must produce deterministic output.\n\n### Configuration\n\nSee `jj help -k config` chapter `Code formatting and other file content transformations` to understand how to configure your tools.\n\n### Execution Example\n\nLet's consider the following configuration is set. We have two code formatters (`clang-format` and `black`), which apply to three different file extensions (`.cc`, `.h`, and `.py`):\n\n```toml [fix.tools.clang-format] command = [\"/usr/bin/clang-format\", \"--assume-filename=$path\"] patterns = [\"glob:'**/*.cc'\", \"glob:'**/*.h'\"]\n\n[fix.tools.black] command = [\"/usr/bin/black\", \"-\", \"--stdin-filename=$path\"] patterns = [\"glob:'**/*.py'\"] ```\n\nNow, let's see what would happen to the following history, when executing `jj fix`.\n\n```text C (mutable) |  Modifies file: foo.py | B @ (working copy - mutable) |  Modifies file: README.md | A (mutable) |  Modifies files: src/bar.cc and src/bar.h | X (immutable) ```\n\nBy default, `jj fix` will modify revisions that matches the revset `reachable(@, mutable())` (see `jj help -k revsets`) which corresponds to the revisions `A`, `B` and `C` here.\n\nThe following operations will then happen:\n\n- For revision `A`, content from this revision for files `src/bar.cc` and `src/bar.h` will each be provided to `clang-format` and the result output will be used to recreate revision `A` which we will call `A'`. All other files are untouched. - For revision `B`, same thing happen for files `src/bar.cc` and `src/bar.h` Their content from revision `B` will go through `clang-format`. The file `README.md` as any other files, are untouched as no pattern matches it. We obtain revision `B'`. - For revision `C`, `src/bar.cc` and `src/bar.h` goes through `clang-format` and file `foo.py` is fixed using `black`. Any other file is untouched. We obtain revision `C'`.\n\n```text C (mutable)                    /-> C' |  src/bar.cc -> clang-format -|   | |  src/bar.h --> clang-format -|   | |  foo.py -----> black --------|   | |  * --------------------------/   | |                                  | B @ (working copy - mutable)   /-> B' @ |  src/bar.cc -> clang-format -|   | |  src/bar.h --> clang-format -|   | |  * --------------------------|   | |                                  | A (mutable)                    /-> A' |  src/bar.cc -> clang-format -|   | |  src/bar.h --> clang-format -|   | |  * --------------------------/   | |                                  | X (immutable)                      X ```\n\nThe revisions are now all correctly formatted according to the configuration.",
					Args: []Arg{
						{Name: "FILESETS", Description: "Fix only these paths", Variadic: true},
					},
					Flags: []Flag{
						{Name: "source", Description: "Fix files in the specified revision(s) and their descendants. If no revisions are specified, this defaults to the `revsets.fix` setting, or `reachable(@, mutable())` if it is not set", Value: "--source", RequiresInput: true, InputType: "REVSETS"},
						{Name: "include-unchanged-files", Description: "Fix unchanged files in addition to changed ones. If no paths are specified, all files in the repo will be fixed", Value: "--include-unchanged-files"},
						{Name: "all-lines", Description: "Format all lines instead of only modified lines.\n\nIf the formatter doesn't support formatting only modified lines, then this option has no effect since the formatter always formats all lines.", Value: "--all-lines"},
					},
				},
				{
					Name:        "file",
					Description: "File operations",
					SubCmds: []SubCommand{
						{
							Summary: "Start tracking specified paths in the working copy", Name: "track",
							Description: "Start tracking specified paths in the working copy\n\nWithout arguments, all paths that are not ignored will be tracked.\n\nBy default, new files in the working copy are automatically tracked, so this command has no effect. You can configure which paths to automatically track by setting `snapshot.auto-track` (e.g. to `\"none()\"` or `\"glob:**/*.rs\"`). Files that don't match the pattern can be manually tracked using this command. The default pattern is `all()`.",
							Args: []Arg{
								{Name: "FILESETS", Description: "Paths to track", Variadic: true, Required: true},
							},
							Flags: []Flag{
								{Name: "include-ignored", Description: "Track paths even if they're ignored or too large\n\nBy default, `jj file track` will not track files that are ignored by .gitignore or exceed the maximum file size. This flag overrides those restrictions, explicitly tracking the specified paths.", Value: "--include-ignored"},
							},
						},
						{
							Summary: "Stop tracking specified paths in the working copy", Name: "untrack",
							Description: "Stop tracking specified paths in the working copy",
							Args: []Arg{
								{Name: "FILESETS", Description: "Paths to untrack. They must already be ignored.\n\nThe paths could be ignored via a .gitignore or .git/info/exclude (in colocated workspaces).", Variadic: true, Required: true},
							},
							Flags: []Flag{},
						},
						{
							Summary: "Sets or removes the executable bit for paths in the repo", Name: "chmod",
							Description: "Sets or removes the executable bit for paths in the repo\n\nUnlike the POSIX `chmod`, `jj file chmod` also works on Windows, on conflicted files, and on arbitrary revisions.",
							Args: []Arg{
								{Name: "MODE", Description: "Possible values: - `n`: Make a path non-executable (alias: normal) - `x`: Make a path executable (alias: executable)", Required: true},
								{Name: "FILESETS", Description: "Paths to change the executable bit for", Variadic: true, Required: true},
							},
							Flags: []Flag{
								{Name: "revision", Description: "The revision to update\n\nDefault value: `@`", Value: "-r", RequiresInput: true, InputType: "REVSET"},
							},
						},
					},
				},
			},
		},
		{
			Name: "Sync",
			Commands: []Command{
				{
					Name:        "bookmark",
					Alias:       "b",
					Description: "Manage bookmarks [default alias: b]\n\nSee [`jj help -k bookmarks`] for more information.\n\n[`jj help -k bookmarks`]: https://docs.jj-vcs.dev/latest/bookmarks",
					SubCmds: []SubCommand{
						{
							Summary: "List bookmarks and their targets", Name: "list",
							Alias:       "l",
							Description: "List bookmarks and their targets\n\nBy default, a tracked remote bookmark will be included only if its target is different from the local target. An untracked remote bookmark won't be listed. For a conflicted bookmark (both local and remote), old target revisions are preceded by a \"-\" and new target revisions are preceded by a \"+\".\n\nSee [`jj help -k bookmarks`] for more information.\n\n[`jj help -k bookmarks`]: https://docs.jj-vcs.dev/latest/bookmarks",
							Args: []Arg{
								{Name: "NAMES", Description: "Show bookmarks whose local name matches\n\nBy default, the specified pattern matches bookmark names with glob syntax. You can also use other [string pattern syntax].\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Variadic: true},
							},
							Flags: []Flag{
								{Name: "all-remotes", Description: "Show all tracked and untracked remote bookmarks including the ones whose targets are synchronized with the local bookmarks", Value: "-a"},
								{Name: "remote", Description: "Show all tracked and untracked remote bookmarks belonging to this remote\n\nCan be combined with `--tracked` or `--conflicted` to filter the bookmarks shown (can be repeated.)\n\nBy default, the specified pattern matches remote names with glob syntax. You can also use other [string pattern syntax].\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Value: "--remote", RequiresInput: true, InputType: "REMOTE"},
								{Name: "tracked", Description: "Show tracked remote bookmarks only\n\nThis omits local Git-tracking bookmarks by default.", Value: "-t"},
								{Name: "conflicted", Description: "Show conflicted bookmarks only", Value: "-c"},
								{Name: "revision", Description: "Show bookmarks whose local targets are in the given revisions\n\nNote that `-r deleted_bookmark` will not work since `deleted_bookmark` wouldn't have a local target.", Value: "-r", RequiresInput: true, InputType: "REVSETS"},
							},
						},
						{
							Summary: "Advance the closest bookmarks to a target revision", Name: "advance",
							Alias:       "a",
							Description: "Advance the closest bookmarks to a target revision\n\nThe target `--to` defaults to `revsets.bookmark-advance-to` (which defaults to `@`).\n\nThe bookmarks to advance are determined by `revsets.bookmark-advance-from` (which defaults to `heads(::to & bookmarks())`).\n\nNote that the from revset has access to `to`.\n\nPositional bookmark name arguments can target specific bookmarks to advance to the target, in this case the default from revset is ignored.\n\nExample:\n\n`jj bookmark advance --to x` - Does the equivalent of `jj bookmark move --from 'heads(::x & bookmarks())' --to x`.",
							Args: []Arg{
								{Name: "NAMES", Description: "Move bookmarks matching the given name patterns\n\nBy default, the specified pattern matches bookmark names with glob syntax. You can also use other [string pattern syntax].\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Variadic: true},
							},
							Flags: []Flag{
								{Name: "to", Description: "Move bookmarks to this revision\n\nDefaults to `revsets.bookmark-advance-to`.", Value: "-t", RequiresInput: true, InputType: "REVSET"},
							},
						},
						{
							Summary: "Move existing bookmarks to target revision", Name: "move",
							Alias:       "m",
							Description: "Move existing bookmarks to target revision\n\nUnlike `jj bookmark set`, this command cannot create new bookmarks.\n\nIf bookmark names are given, the specified bookmarks will be updated to point to the target revision.\n\nIf `--from` options are given, bookmarks currently pointing to the specified revisions will be updated. The bookmarks can also be filtered by names.",
							Args: []Arg{
								{Name: "NAMES", Description: "Move bookmarks matching the given name patterns\n\nBy default, the specified pattern matches bookmark names with glob syntax. You can also use other [string pattern syntax].\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Variadic: true},
							},
							Flags: []Flag{
								{Name: "from", Description: "Move bookmarks from the given revisions", Value: "--from", RequiresInput: true, InputType: "REVSETS"},
								{Name: "to", Description: "Move bookmarks to this revision\n\nDefault value: `@`", Value: "--to", RequiresInput: true, InputType: "REVSET"},
								{Name: "allow-backwards", Description: "Allow moving bookmarks backwards or sideways", Value: "-B"},
							},
						},
						{
							Summary: "Create a new bookmark, or update an existing one by name", Name: "set",
							Alias:       "s",
							Description: "Create a new bookmark, or update an existing one by name\n\nIf you want to move bookmarks based on their current location rather than by name, use `jj bookmark move --from <REVSETS>`.",
							Args: []Arg{
								{Name: "NAMES", Description: "The bookmarks to update", Variadic: true, Required: true},
							},
							Flags: []Flag{
								{Name: "revision", Description: "The bookmark's target revision\n\nDefault value: `@`", Value: "-r", RequiresInput: true, InputType: "REVSET"},
								{Name: "allow-backwards", Description: "Allow moving the bookmark backwards or sideways", Value: "--allow-backwards"},
							},
						},
						{
							Summary: "Create a new bookmark", Name: "create",
							Alias:       "c",
							Description: "Create a new bookmark",
							Args: []Arg{
								{Name: "NAMES", Description: "The bookmarks to create", Variadic: true, Required: true},
							},
							Flags: []Flag{
								{Name: "revision", Description: "The bookmark's target revision\n\nDefault value: `@`", Value: "-r", RequiresInput: true, InputType: "REVSET"},
							},
						},
						{
							Summary: "Rename `old` bookmark name to `new` bookmark name", Name: "rename",
							Alias:       "r",
							Description: "Rename `old` bookmark name to `new` bookmark name\n\nThe new bookmark name points at the same commit as the old bookmark name.",
							Args: []Arg{
								{Name: "OLD", Description: "The old name of the bookmark", Required: true},
								{Name: "NEW", Description: "The new name of the bookmark", Required: true},
							},
							Flags: []Flag{
								{Name: "overwrite-existing", Description: "Allow renaming even if the new bookmark name already exists", Value: "--overwrite-existing"},
							},
						},
						{
							Summary: "Start tracking given remote bookmarks", Name: "track",
							Alias:       "t",
							Description: "Start tracking given remote bookmarks\n\nA tracked remote bookmark will be imported as a local bookmark of the same name. Changes to it will propagate to the existing local bookmark on future pulls.",
							Args: []Arg{
								{Name: "BOOKMARK", Description: "Bookmark names to track\n\nBy default, the specified pattern matches bookmark names with glob syntax. You can also use other [string pattern syntax].\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Variadic: true, Required: true},
							},
							Flags: []Flag{
								{Name: "remote", Description: "Remote names to track\n\nBy default, the specified pattern matches remote names with glob syntax. You can also use other [string pattern syntax].\n\nIf no remote names are given, all remote bookmarks matching the bookmark names will be tracked.\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Value: "--remote", RequiresInput: true, InputType: "REMOTE"},
							},
						},
						{
							Summary: "Stop tracking given remote bookmarks", Name: "untrack",
							Description: "Stop tracking given remote bookmarks\n\nAn untracked remote bookmark is just a pointer to the last-fetched remote bookmark. It won't be imported as a local bookmark on future pulls.\n\nIf you want to forget a local bookmark while also untracking the corresponding remote bookmarks, use `jj bookmark forget` instead.",
							Args: []Arg{
								{Name: "BOOKMARK", Description: "Bookmark names to untrack\n\nBy default, the specified pattern matches bookmark names with glob syntax. You can also use other [string pattern syntax].\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Variadic: true, Required: true},
							},
							Flags: []Flag{
								{Name: "remote", Description: "Remote names to untrack\n\nBy default, the specified pattern matches remote names with glob syntax. You can also use other [string pattern syntax].\n\nIf no remote names are given, all remote bookmarks matching the bookmark names will be untracked.\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Value: "--remote", RequiresInput: true, InputType: "REMOTE"},
							},
						},
						{
							Summary: "Forget a bookmark without marking it as a deletion to be pushed", Name: "forget",
							Alias:       "f",
							Description: "Forget a bookmark without marking it as a deletion to be pushed\n\nIf a local bookmark is forgotten, any corresponding remote bookmarks will become untracked to ensure that the forgotten bookmark will not impact remotes on future pushes.",
							Args: []Arg{
								{Name: "NAMES", Description: "The bookmarks to forget\n\nBy default, the specified pattern matches bookmark names with glob syntax. You can also use other [string pattern syntax].\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Variadic: true, Required: true},
							},
							Flags: []Flag{
								{Name: "include-remotes", Description: "When forgetting a local bookmark, also forget any corresponding remote bookmarks\n\nA forgotten remote bookmark will not impact remotes on future pushes. It will be recreated on future fetches if it still exists on the remote. If there is a corresponding Git-tracking remote bookmark, it will also be forgotten.", Value: "--include-remotes"},
							},
						},
						{
							Summary: "Delete an existing bookmark and propagate the deletion to remotes on the next push", Name: "delete",
							Description: "Delete an existing bookmark and propagate the deletion to remotes on the next push\n\nRevisions referred to by the deleted bookmarks are not abandoned. To delete revisions as well as bookmarks, use `jj abandon`. For example, `jj abandon main..<bookmark>` will abandon revisions belonging to the `<bookmark>` branch (relative to the `main` branch.)\n\nIf you don't want the deletion of the local bookmark to propagate to any tracked remote bookmarks, use `jj bookmark forget` instead.",
							Args: []Arg{
								{Name: "NAMES", Description: "The bookmarks to delete\n\nBy default, the specified pattern matches bookmark names with glob syntax. You can also use other [string pattern syntax].\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Variadic: true, Required: true},
							},
							Flags: []Flag{},
						},
					},
				},
				{
					Name:        "git",
					Description: "Commands for working with Git remotes and the underlying Git repo\n\nSee this [comparison], including a [table of commands].\n\n[comparison]: https://docs.jj-vcs.dev/latest/git-comparison/.\n\n[table of commands]: https://docs.jj-vcs.dev/latest/git-command-table",
					SubCmds: []SubCommand{
						{
							Summary: "Fetch from a Git remote", Name: "fetch",
							Description: "Fetch from a Git remote\n\nIf no remotes are specified, fetches the remotes specified by the `git.fetch` setting. If that is not configured and there are multiple remotes, the remote named \"origin\" will be used.\n\nIf no branches nor tags are specified, fetches bookmarks and tags specified by the `remotes.<name>.fetch-bookmarks`/`fetch-tags` settings. If `remotes.<name>.fetch-bookmarks` is not configured, the default fetch refspecs for the selected remotes are read from the Git configuration.\n\nCommits that are no longer reachable from any branch on the remote will be considered abandoned by the remote, and will be abandoned in the local repo to match the remote. Set `git.abandon-unreachable-commits` to `false` to disable this behavior.\n\nIf a working-copy commit gets abandoned, it will be given a new, empty commit. This is true in general; it is not specific to this command.",
							Flags: []Flag{
								{Name: "remote", Description: "The remote to fetch from (only named remotes are supported, can be repeated)\n\nBy default, the specified pattern matches remote names with glob syntax, e.g. `--remote '*'`. You can also use other [string pattern syntax].\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Value: "--remote", RequiresInput: true, ConflictingFlags: []string{"--all-remotes"}, InputType: "REMOTE"},
								{Name: "all-remotes", Description: "Fetch from all remotes", Value: "--all-remotes", ConflictingFlags: []string{"--remote"}},
								{Name: "branch", Description: "Name of the branch to fetch (can be repeated)\n\nBy default, the specified pattern matches branch names with glob syntax, but only `*` is expanded. Other wildcard characters such as `?` are *not* supported. Patterns can be repeated or combined with [logical operators] to specify multiple branches, but only union and negative intersection are supported.\n\nExamples: `push-*`, `(push-* | foo/*) ~ foo/unwanted`\n\n[logical operators]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Value: "--branch", RequiresInput: true, InputType: "BRANCH"},
								{Name: "tracked", Description: "Fetch only tracked bookmarks\n\nThis fetches only bookmarks that are already tracked from the specified remote(s).", Value: "--tracked"},
							},
						},
						{
							Summary: "Push to a Git remote", Name: "push",
							Description: "Push to a Git remote\n\nBy default, pushes tracking bookmarks pointing to `remote_bookmarks(remote=<remote>)..@`. Use `--bookmark` to push specific bookmarks. Use `--all` to push all bookmarks. Use `--change` to generate bookmark names based on the change IDs of specific commits.\n\nWhen pushing a bookmark, the command pushes all commits in the range from the remote's current position up to and including the bookmark's target commit. Any descendant commits beyond the bookmark are not pushed.\n\nIf the local bookmark has changed from the last fetch, push will update the remote bookmark to the new position after passing safety checks. This is similar to `git push --force-with-lease` - the remote is updated only if its current state matches what Jujutsu last fetched.\n\nUnlike in Git, the remote to push to is not derived from the tracked remote bookmarks. Use `--remote` to select the remote Git repository by name. There is no option to push to multiple remotes.\n\nBefore the command actually moves, creates, or deletes a remote bookmark, it makes several [safety checks]. If there is a problem, you may need to run `jj git fetch --remote <remote name>` and/or resolve some [bookmark conflicts].\n\n[safety checks]: https://docs.jj-vcs.dev/latest/bookmarks/#pushing-bookmarks-safety-checks\n\n[bookmark conflicts]: https://docs.jj-vcs.dev/latest/bookmarks/#conflicts",
							Flags: []Flag{
								{Name: "remote", Description: "The remote to push to (only named remotes are supported)\n\nThis defaults to the `git.push` setting. If that is not configured, and if there are multiple remotes, the remote named \"origin\" will be used.", Value: "--remote", RequiresInput: true, InputType: "REMOTE"},
								{Name: "bookmark", Description: "Push only this bookmark, or bookmarks matching a pattern (can be repeated)\n\nIf a bookmark isn't tracking anything yet, the remote bookmark will be tracked automatically.\n\nBy default, the specified pattern matches bookmark names with glob syntax. You can also use other [string pattern syntax].\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Value: "--bookmark", RequiresInput: true, InputType: "BOOKMARK"},
								{Name: "all", Description: "Push all bookmarks (including new bookmarks)", Value: "--all"},
								{Name: "tracked", Description: "Push all tracked bookmarks\n\nThis usually means that the bookmark was already pushed to or fetched from the [relevant remote].\n\n[relevant remote]: https://docs.jj-vcs.dev/latest/bookmarks#remotes-and-tracked-bookmarks", Value: "--tracked"},
								{Name: "deleted", Description: "Push all deleted bookmarks\n\nOnly tracked bookmarks can be successfully deleted on the remote. A warning will be printed if any untracked bookmarks on the remote correspond to missing local bookmarks.", Value: "--deleted"},
								{Name: "allow-empty-description", Description: "Allow pushing commits with empty descriptions", Value: "--allow-empty-description"},
								{Name: "allow-private", Description: "Allow pushing commits that are private\n\nThe set of private commits can be configured by the `git.private-commits` setting. The default is `none()`, meaning all commits are eligible to be pushed.", Value: "--allow-private"},
								{Name: "revision", Description: "Push bookmarks pointing to these commits (can be repeated)", Value: "-r", RequiresInput: true, InputType: "REVSETS"},
								{Name: "change", Description: "Push this commit by creating a bookmark (can be repeated)\n\nThe created bookmark will be tracked automatically. Use the `templates.git_push_bookmark` setting to customize the generated bookmark name. The default is `\"push-\" ++ change_id.short()`.", Value: "-c", RequiresInput: true, InputType: "REVSETS"},
								{Name: "named", Description: "Specify a new bookmark name and a revision to push under that name, e.g. '--named myfeature=@'\n\nAutomatically tracks the bookmark if it is new.", Value: "--named", RequiresInput: true},
								{Name: "dry-run", Description: "Only display what will change on the remote", Value: "--dry-run"},
							},
						},
						{
							Summary: "Update repo with changes made in the underlying Git repo", Name: "import",
							Description: "Update repo with changes made in the underlying Git repo\n\nCommits that are no longer reachable from any branch in the Git repo will be considered abandoned in the Git repo, and will be abandoned in the jj repo to match the Git repo. Set `git.abandon-unreachable-commits` to `false` to disable this behavior.\n\nIf a working-copy commit gets abandoned, it will be given a new, empty commit. This is true in general; it is not specific to this command.\n\nThere is no need to run this command if you're in colocated workspace because the import happens automatically there.",
							Flags:       []Flag{},
						},
						{
							Summary: "Update the underlying Git repo with changes made in the repo", Name: "export",
							Description: "Update the underlying Git repo with changes made in the repo\n\nThere is no need to run this command if you're in colocated workspace because the export happens automatically there.",
							Flags:       []Flag{},
						},
						{
							Summary: "Add a Git remote", Name: "remote add",
							Description: "Add a Git remote",
							Args: []Arg{
								{Name: "REMOTE", Description: "The remote's name", Required: true},
								{Name: "URL", Description: "The remote's URL or path\n\nLocal path will be resolved to absolute form.", Required: true},
							},
							Flags: []Flag{},
						},
						{
							Summary: "List Git remotes", Name: "remote list",
							Description: "List Git remotes",
							Flags:       []Flag{},
						},
						{
							Summary: "Remove a Git remote and forget its bookmarks", Name: "remote remove",
							Description: "Remove a Git remote and forget its bookmarks",
							Args: []Arg{
								{Name: "REMOTE", Description: "The remote's name", Required: true},
							},
							Flags: []Flag{},
						},
						{
							Summary: "Rename a Git remote", Name: "remote rename",
							Description: "Rename a Git remote",
							Args: []Arg{
								{Name: "OLD", Description: "The name of an existing remote", Required: true},
								{Name: "NEW", Description: "The desired name for `old`", Required: true},
							},
							Flags: []Flag{},
						},
					},
				},
				{
					Name:        "tag",
					Description: "Manage tags",
					SubCmds: []SubCommand{
						{
							Summary: "List tags and their targets", Name: "list",
							Alias:       "l",
							Description: "List tags and their targets\n\nBy default, a tracked remote tag will be included only if its target is different from the local tag. An untracked remote tag won't be listed. For a conflicted tag (both local and remote), old target revisions are preceded by a \"-\" and new target revisions are preceded by a \"+\".\n\nThe `-r` flag combined with revset expressions can be used for filtering. For example:\n\n* `jj tag list -r 'REV::'` shows tags whose targets are descendants of REV (similar to `git tag --contains REV`).\n\n* `jj tag list -r '::REV'` shows tags whose targets are ancestors of REV (similar to `git tag --merged REV`).",
							Args: []Arg{
								{Name: "NAMES", Description: "Show tags whose local name matches\n\nBy default, the specified pattern matches tag names with glob syntax. You can also use other [string pattern syntax].\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Variadic: true},
							},
							Flags: []Flag{
								{Name: "all-remotes", Description: "Show all tracked and untracked remote tags including the ones whose targets are synchronized with the local tags", Value: "-a"},
							},
						},
						{
							Summary: "Create or update tags", Name: "set",
							Alias:       "s",
							Description: "Create or update tags",
							Args: []Arg{
								{Name: "NAMES", Description: "Tag names to create or update", Variadic: true, Required: true},
							},
							Flags: []Flag{
								{Name: "revision", Description: "Target revision to point to\n\nDefault value: `@`", Value: "-r", RequiresInput: true, InputType: "REVSET"},
								{Name: "allow-move", Description: "Allow moving existing tags", Value: "--allow-move"},
							},
						},
						{
							Summary: "Delete existing tags", Name: "delete",
							Alias:       "d",
							Description: "Delete existing tags\n\nRevisions referred to by the deleted tags are not abandoned.",
							Args: []Arg{
								{Name: "NAMES", Description: "Tag names to delete\n\nBy default, the specified pattern matches tag names with glob syntax. You can also use other [string pattern syntax].\n\n[string pattern syntax]: https://docs.jj-vcs.dev/latest/revsets/#string-patterns", Variadic: true, Required: true},
							},
							Flags: []Flag{},
						},
					},
				},
			},
		},
		{
			Name: "Rewrite",
			Commands: []Command{
				{
					Name:        "squash",
					Description: "Move changes from a revision into another revision\n\nWithout any options, moves the changes from the working-copy revision to the parent revision.\n\nWith the `-r` option, moves the changes from the specified revision to the parent revision. Fails if there are several parent revisions (i.e., the given revision is a merge).\n\nWith the `--from` and/or `--into` options, moves changes from/to the given revisions. If either is left out, it defaults to the working-copy commit. For example, `jj squash --into @--` moves changes from the working-copy commit to the grandparent.\n\nIf, after moving changes out, the source revision is empty compared to its parent(s), and `--keep-emptied` is not set, it will be abandoned. Without `--interactive` or paths, the source revision will always be empty.\n\nIf the source was abandoned and both the source and destination had a non-empty description, you will be asked for the combined description. If either was empty, then the other one will be used.\n\nIf a working-copy commit gets abandoned, it will be given a new, empty commit. This is true in general; it is not specific to this command.\n\nThe name \"squash\" comes from the idea of combining (squashing) the changes from multiple revisions together.\n\nEXPERIMENTAL FEATURES\n\nAn alternative squashing UI is available via the `-o`, `-A`, and `-B` options. Using any of these options creates a new commit. They can be used together with one or more `--from` options (if no `--from` is specified, `--from @` is assumed).",
					Args: []Arg{
						{Name: "FILESETS", Description: "Move only changes to these paths (instead of all paths)", Variadic: true},
					},
					Flags: []Flag{
						{Name: "revision", Description: "Revision to squash into its parent (default: @). Incompatible with the experimental `-o`/`-A`/`-B` options", Value: "-r", RequiresInput: true, ConflictingFlags: []string{"-o", "-A", "-B"}, InputType: "REVSET"},
						{Name: "from", Description: "Revision(s) to squash from (default: @)", Value: "-f", RequiresInput: true, InputType: "REVSETS"},
						{Name: "into", Description: "Revision to squash into (default: @)", Value: "-t", RequiresInput: true, InputType: "REVSET"},
						{Name: "onto", Description: "(Experimental) The revision(s) to use as parent for the new commit (can be repeated to create a merge commit)", Value: "-o", RequiresInput: true, ConflictingFlags: []string{"-r", "-A", "-B"}, InputType: "REVSETS"},
						{Name: "insert-after", Description: "(Experimental) The revision(s) to insert the new commit after (can be repeated to create a merge commit)", Value: "-A", RequiresInput: true, ConflictingFlags: []string{"-r", "-o"}, InputType: "REVSETS"},
						{Name: "insert-before", Description: "(Experimental) The revision(s) to insert the new commit before (can be repeated to create a merge commit)", Value: "-B", RequiresInput: true, ConflictingFlags: []string{"-r", "-o"}, InputType: "REVSETS"},
						{Name: "message", Description: "The description to use for squashed revision (don't open editor)", Value: "-m", RequiresInput: true, NeedsQuotes: true, ConflictingFlags: []string{"--use-destination-message"}, InputType: "MESSAGE"},
						{Name: "use-destination-message", Description: "Use the description of the destination revision and discard the description(s) of the source revision(s)", Value: "--use-destination-message", ConflictingFlags: []string{"-m"}},
						{Name: "keep-emptied", Description: "The source revision will not be abandoned", Value: "-k"},
					},
				},
				{
					Name:        "split",
					Description: "Split a revision in two\n\nStarts a [diff editor] on the changes in the revision. Edit the right side of the diff until it has the content you want in the first commit. Once you close the editor, your revision will be split into two commits.\n\n[diff editor]: https://docs.jj-vcs.dev/latest/config/#editing-diffs\n\nBy default, the selected changes stay in the original commit, and the remaining changes go into a new child commit:\n\n```text L                 L' |                 | K (split)   =>    K\" (remaining) |                 | J                 K' (selected) | J ```\n\nWith `--parallel/-p`, the two parts become sibling commits instead of parent and child:\n\n```text L' L                / \\ |               K'  |  (selected) K (split)  =>   |   K\" (remaining) |                \\ / J                 J ```\n\nWith `-o`, `-A`, or `-B`, the selected changes are extracted into a new commit at the specified location, while the remaining changes stay in place:\n\n```text M                 M' |                 | L                 L' |                 | K (split)   =>    K' (remaining, stays here) |                 | J                 J' | K\" (selected, inserted before J with -B J) ```\n\nIf the change you split had a description, you will be asked to enter a change description for each commit. If the change did not have a description, the second commit will not get a description, and you will be asked for a description only for the first commit.\n\nSplitting an empty commit is not supported because the same effect can be achieved with `jj new`.",
					Args: []Arg{
						{Name: "FILESETS", Description: "Files matching any of these filesets are put in the selected changes", Variadic: true},
					},
					Flags: []Flag{
						{Name: "revision", Description: "The revision to split\n\nDefault value: `@`", Value: "-r", RequiresInput: true, InputType: "REVSET"},
						{Name: "message", Description: "The change description to use for the selected changes (don't open editor)\n\nSets the description for the revision containing the selected changes. The other revision will keep its original description, if any.", Value: "-m", RequiresInput: true, NeedsQuotes: true, InputType: "MESSAGE"},
					},
				},
				{
					Name:        "rebase",
					Description: "Move revisions to different parent(s)\n\nThis command moves revisions to different parent(s) while preserving the changes (diff) in the revisions.\n\nThere are three different ways of specifying which revisions to rebase:\n\n* `--source/-s` to rebase a revision and its descendants * `--branch/-b` to rebase a whole branch, relative to the destination * `--revisions/-r` to rebase the specified revisions without their descendants\n\nIf no option is specified, it defaults to `-b @`.\n\nThere are three different ways of specifying where the revisions should be rebased to:\n\n* `--onto/-o` to rebase the revisions onto the specified targets * `--insert-after/-A` to rebase the revisions onto the specified targets and to rebase the targets' descendants onto the rebased revisions * `--insert-before/-B` to rebase the revisions onto the specified targets' parents and to rebase the targets and their descendants onto the rebased revisions\n\nSee the sections below for details about the different ways of specifying which revisions to rebase where.\n\nIf a working-copy revision gets abandoned, it will be given a new, empty revision. This is true in general; it is not specific to this command.\n\n### Specifying which revisions to rebase\n\nWith `--source/-s`, the command rebases the specified revision and its descendants to the destination. For example, `jj rebase -s M -o O` would transform your history like this (letters followed by an apostrophe are post-rebase versions):\n\n```text O           N' |           | | N         M' | |         | | M         O | |    =>   | | | L       | L | |/        | | | K         | K |/          |/ J           J ```\n\nEach revision passed to `-s` will become a direct child of the destination, so if you instead run `jj rebase -s M -s N -o O` (or `jj rebase -s 'M|N' -o O`) in the example above, then N' would instead be a direct child of O.\n\nWith `--branch/-b`, the command rebases the whole \"branch\" containing the specified revision. A \"branch\" is the set of revisions that includes:\n\n* the specified revision and ancestors that are not also ancestors of the destination * all descendants of those revisions\n\nIn other words, `jj rebase -b X -o Y` rebases revisions in the revset `(Y..X)::` (which is equivalent to `jj rebase -s 'roots(Y..X)' -o Y` for a single root). For example, either `jj rebase -b L -o O` or `jj rebase -b M -o O` would transform your history like this (because `L` and `M` are on the same \"branch\", relative to the destination):\n\n```text O           N' |           | | N         M' | |         | | M         | L' | |    =>   |/ | | L       K' | |/        | | K         O |/          | J           J ```\n\nWith `--revisions/-r`, the command rebases only the specified revisions to the destination. Any \"hole\" left behind will be filled by rebasing descendants onto the specified revisions' parent(s). For example, `jj rebase -r K -o M` would transform your history like this:\n\n```text M          K' |          | | L        M | |   =>   | | K        | L' |/         |/ J          J ```\n\nMultiple revisions can be specified, and any dependencies (graph edges) within the set will be preserved. For example, `jj rebase -r 'K|N' -o O` would transform your history like this:\n\n```text O           N' |           | | N         K' | |         | | M         O | |    =>   | | | L       | M' | |/        |/ | K         | L' |/          |/ J           J ```\n\n`jj rebase -s X` is similar to `jj rebase -r X::` and will behave the same if X is a single revision. However, if X is a set of multiple revisions, or if you passed multiple `-s` arguments, then `jj rebase -s` will make each of the specified revisions an immediate child of the destination, while `jj rebase -r` will preserve dependencies within the set.\n\nNote that you can create a merge revision by repeating the `-o` argument. For example, if you realize that revision L actually depends on revision M in order to work (in addition to its current parent K), you can run `jj rebase -s L -o K -o M`:\n\n```text M          L' |          |\\ | L        M | | |   =>   | | | K        | K |/         |/ J          J ```\n\n### Specifying where to rebase the revisions\n\nWith `--onto/-o`, the command rebases the selected revisions onto the targets. Existing descendants of the targets will not be affected. See the section above for examples.\n\nWith `--insert-after/-A`, the selected revisions will be inserted after the targets. This is similar to `-o`, but if the targets have any existing descendants, then those will be rebased onto the rebased selected revisions.\n\nFor example, `jj rebase -r K -A L` will rewrite history like this: ```text N           N' |           | | M         | M' |/          |/ L      =>   K' |           | | K         L |/          | J           J ```\n\nThe `-A` (and `-B`) argument can also be used for reordering revisions. For example, `jj rebase -r M -A J` will rewrite history like this: ```text M          L' |          | L          K' |     =>   | K          M' |          | J          J ```\n\nWith `--insert-before/-B`, the selected revisions will be inserted before the targets. This is achieved by rebasing the selected revisions onto the target revisions' parents, and then rebasing the target revisions and their descendants onto the rebased revisions.\n\nFor example, `jj rebase -r K -B L` will rewrite history like this: ```text N           N' |           | | M         | M' |/          |/ L     =>    L' |           | | K         K' |/          | J           J ```\n\nThe `-A` and `-B` arguments can also be combined, which can be useful around merges. For example, you can use `jj rebase -r K -A J -B M` to create a new merge (but `jj rebase -r M -o L -o K` might be simpler in this particular case): ```text M           M' |           |\\ L           L | |     =>    | | | K         | K' |/          |/ J           J ```\n\nTo insert a commit inside an existing merge with `jj rebase -r O -A K -B M`: ```text O           N' |           |\\ N           | M' |\\          | |\\ | M         | O'| | |    =>   |/ / | L         | L | |         | | K |         K | |/          |/ J           J ```",
					Flags: []Flag{
						{Name: "revision", Description: "Rebase the given revisions, rebasing descendants onto this revision's parent(s)\n\nUnlike `-s` or `-b`, you may `jj rebase -r` a revision `A` onto a descendant of `A`.\n\nIf none of `-b`, `-s`, or `-r` is provided, then the default is `-b @`.", Value: "-r", RequiresInput: true, ConflictingFlags: []string{"-s", "-b"}, InputType: "REVSETS"},
						{Name: "source", Description: "Rebase specified revision(s) together with their trees of descendants (can be repeated)\n\nEach specified revision will become a direct child of the destination revision(s), even if some of the source revisions are descendants of others.\n\nIf none of `-b`, `-s`, or `-r` is provided, then the default is `-b @`.", Value: "-s", RequiresInput: true, ConflictingFlags: []string{"-r", "-b"}, InputType: "REVSETS"},
						{Name: "branch", Description: "Rebase the whole branch relative to destination's ancestors (can be repeated)\n\n`jj rebase -b=br -o=dst` is equivalent to `jj rebase '-s=roots(dst..br)' -o=dst`.\n\nIf none of `-b`, `-s`, or `-r` is provided, then the default is `-b @`.", Value: "-b", RequiresInput: true, ConflictingFlags: []string{"-r", "-s"}, InputType: "REVSETS"},
						{Name: "onto", Description: "The revision(s) to rebase onto (can be repeated to create a merge commit)", Value: "-o", RequiresInput: true, ConflictingFlags: []string{"--insert-after", "--insert-before"}, InputType: "REVSETS"},
						{Name: "insert-after", Description: "The revision(s) to insert after (can be repeated to create a merge commit)", Value: "--insert-after", RequiresInput: true, ConflictingFlags: []string{"-o"}, InputType: "REVSETS"},
						{Name: "insert-before", Description: "The revision(s) to insert before (can be repeated to create a merge commit)", Value: "--insert-before", RequiresInput: true, ConflictingFlags: []string{"-o"}, InputType: "REVSETS"},
						{Name: "skip-emptied", Description: "If true, when rebasing would produce an empty commit, the commit is abandoned. It will not be abandoned if it was already empty before the rebase. Will never skip merge commits with multiple non-empty parents", Value: "--skip-emptied"},
						{Name: "keep-divergent", Description: "Keep divergent commits while rebasing\n\nWithout this flag, divergent commits are abandoned while rebasing if another commit with the same change ID is already present in the destination with identical changes.", Value: "--keep-divergent"},
						{Name: "simplify-parents", Description: "Simplify parents of rebased commits, like `jj simplify-parents`, while rebasing them. Any parents that are ancestors of other parents will be removed", Value: "--simplify-parents"},
					},
					RequiredFlagGroup: []string{"-o", "--insert-after", "--insert-before"},
					RequiredUsage:     "<--onto <REVSETS>|--insert-after <REVSETS>|--insert-before <REVSETS>>",
				},
				{
					Name:        "absorb",
					Description: "Move changes from a revision into the stack of mutable revisions\n\nThis command splits changes in the source revision and moves each change to the closest mutable ancestor where the corresponding lines were modified last. If the destination revision cannot be determined unambiguously, the change will be left in the source revision.\n\nThe source revision will be abandoned if all changes are absorbed into the destination revisions, and if the source revision has no description.\n\nThe modification made by `jj absorb` can be reviewed by `jj op show -p`.",
					Args: []Arg{
						{Name: "FILESETS", Description: "Move only changes to these paths (instead of all paths)", Variadic: true},
					},
					Flags: []Flag{
						{Name: "from", Description: "Source revision to absorb from\n\nDefault value: `@`", Value: "-f", RequiresInput: true, InputType: "REVSET"},
						{Name: "into", Description: "Destination revisions to absorb into\n\nOnly ancestors of the source revision will be considered.\n\nDefault value: `mutable()`", Value: "-t", RequiresInput: true, InputType: "REVSETS"},
					},
				},
				{
					Name:        "abandon",
					Description: "Abandon a revision\n\nAbandon a revision, rebasing descendants onto its parent(s). The behavior is similar to `jj restore --changes-in`; the difference is that `jj abandon` gives you a new change, while `jj restore` updates the existing change.\n\nIf a working-copy commit gets abandoned, it will be given a new, empty commit. This is true in general; it is not specific to this command.",
					Args: []Arg{
						{Name: "REVSETS", Description: "The revision(s) to abandon (default: @) [aliases: -r]", Variadic: true},
					},
					Flags: []Flag{
						{Name: "retain-bookmarks", Description: "Do not delete bookmarks pointing to the revisions to abandon\n\nBookmarks will be moved to the parent revisions instead.", Value: "--retain-bookmarks"},
						{Name: "restore-descendants", Description: "Do not modify the content of the children of the abandoned commits", Value: "--restore-descendants"},
					},
				},
				{
					Name:        "duplicate",
					Description: "Create new changes with the same content as existing ones\n\nWhen none of the `--onto`, `--insert-after`, or `--insert-before` arguments are provided, commits will be duplicated onto their existing parents or onto other newly duplicated commits.\n\nWhen any of the `--onto`, `--insert-after`, or `--insert-before` arguments are provided, the roots of the specified commits will be duplicated onto the destination indicated by the arguments. Other specified commits will be duplicated onto these newly duplicated commits. If the `--insert-after` or `--insert-before` arguments are provided, the new children indicated by the arguments will be rebased onto the heads of the specified commits.\n\nBy default, the duplicated commits retain the descriptions of the originals. This can be customized with the `templates.duplicate_description` setting.",
					Args: []Arg{
						{Name: "REVSETS", Description: "The revision(s) to duplicate (default: @) [aliases: -r]", Variadic: true},
					},
					Flags: []Flag{
						{Name: "onto", Description: "The revision(s) to duplicate onto (can be repeated to create a merge commit)", Value: "-o", RequiresInput: true, ConflictingFlags: []string{"-A", "-B"}, InputType: "REVSETS"},
						{Name: "insert-after", Description: "The revision(s) to insert after (can be repeated to create a merge commit)", Value: "-A", RequiresInput: true, ConflictingFlags: []string{"-o"}, InputType: "REVSETS"},
						{Name: "insert-before", Description: "The revision(s) to insert before (can be repeated to create a merge commit)", Value: "-B", RequiresInput: true, ConflictingFlags: []string{"-o"}, InputType: "REVSETS"},
					},
				},
				{
					Name:        "parallelize",
					Description: "Parallelize revisions by making them siblings\n\nRunning `jj parallelize 1::2` will transform the history like this: ```text 3 |             3 2            / \\ |    ->     1   2 1            \\ / |             0 0 ```\n\nThe command effectively says \"these revisions are actually independent\", meaning that they should no longer be ancestors/descendants of each other. However, revisions outside the set that were previously ancestors of a revision in the set will remain ancestors of it. For example, revision 0 above remains an ancestor of both 1 and 2. Similarly, revisions outside the set that were previously descendants of a revision in the set will remain descendants of it. For example, revision 3 above remains a descendant of both 1 and 2.\n\nTherefore, `jj parallelize '1 | 3'` is a no-op. That's because 2, which is not in the target set, was a descendant of 1 before, so it remains a descendant, and it was an ancestor of 3 before, so it remains an ancestor.",
					Args: []Arg{
						{Name: "REVSETS", Description: "The revisions to parallelize [aliases: -r]", Variadic: true},
					},
					Flags: []Flag{},
				},
				{
					Name:        "simplify-parents",
					Description: "Simplify parent edges for the specified revision(s).\n\nRemoves all parents of each of the specified revisions that are also indirect ancestors of the same revisions through other parents. This has no effect on any revision's contents, including the working copy.\n\nIn other words, for all (A, B, C) where A has (B, C) as parents and C is an ancestor of B, A will be rewritten to have only B as a parent instead of B+C.",
					Flags: []Flag{
						{Name: "revision", Description: "Simplify specified revision(s) (can be repeated)\n\nIf both `--source` and `--revisions` are not provided, this defaults to the `revsets.simplify-parents` setting, or `reachable(@, mutable())` if it is not set.", Value: "-r", RequiresInput: true, InputType: "REVSETS"},
						{Name: "source", Description: "Simplify specified revision(s) together with their trees of descendants (can be repeated)", Value: "--source", RequiresInput: true, InputType: "REVSETS"},
					},
				},
				{
					Name:        "revert",
					Description: "Apply the reverse of the given revision(s)\n\nThe reverse of each of the given revisions is applied sequentially in reverse topological order at the given location.\n\nThe description of the new revisions can be customized with the `templates.revert_description` config variable.",
					Flags: []Flag{
						{Name: "revision", Description: "The revision(s) to apply the reverse of", Value: "-r", RequiresInput: true, Mandatory: true, InputType: "REVSETS"},
						{Name: "onto", Description: "The revision(s) to apply the reverse changes on top of", Value: "-o", RequiresInput: true, ConflictingFlags: []string{"-A", "-B"}, InputType: "REVSETS"},
						{Name: "insert-after", Description: "The revision(s) to insert the reverse changes after (can be repeated to create a merge commit)", Value: "-A", RequiresInput: true, ConflictingFlags: []string{"-o"}, InputType: "REVSETS"},
						{Name: "insert-before", Description: "The revision(s) to insert the reverse changes before (can be repeated to create a merge commit)", Value: "-B", RequiresInput: true, ConflictingFlags: []string{"-o"}, InputType: "REVSETS"},
					},
					RequiredFlagGroup: []string{"-o", "-A", "-B"},
					RequiredUsage:     "--revision <REVSETS> <--onto <REVSETS>|--insert-after <REVSETS>|--insert-before <REVSETS>>",
				},
			},
		},
		{
			Name: "Journal",
			Commands: []Command{
				{
					Name:        "operation",
					Alias:       "op",
					Description: "Commands for working with the operation log\n\nSee the [operation log documentation] for more information.\n\n[operation log documentation]: https://docs.jj-vcs.dev/latest/operation-log/",
					SubCmds: []SubCommand{
						{
							Summary: "Show the operation log", Name: "log",
							Description: "Show the operation log\n\nLike other commands, `jj op log` snapshots the current working-copy changes and reconciles divergent operations. Use `--at-op=@ --ignore-working-copy` to inspect the current state without mutation.",
							Flags: []Flag{
								{Name: "template", Description: "Render each operation using the given template\n\nYou can specify arbitrary template expressions using the [built-in keywords]. See [`jj help -k templates`] for more information.\n\n[built-in keywords]: https://docs.jj-vcs.dev/latest/templates/#operation-keywords\n\n[`jj help -k templates`]: https://docs.jj-vcs.dev/latest/templates/", Value: "-T", RequiresInput: true, InputType: "TEMPLATE"},
								{Name: "limit", Description: "Limit number of operations to show\n\nApplied after operations are reordered topologically, but before being reversed.", Value: "-n", RequiresInput: true, InputType: "LIMIT"},
								{Name: "reversed", Description: "Show operations in the opposite order (older operations first)", Value: "--reversed"},
								{Name: "no-graph", Description: "Don't show the graph, show a flat list of operations", Value: "-G"},
								{Name: "op-diff", Description: "Show changes to the repository at each operation", Value: "-d"},
								{Name: "patch", Description: "Show patch of modifications to changes (implies --op-diff)\n\nIf the previous version has different parents, it will be temporarily rebased to the parents of the new version, so the diff is not contaminated by unrelated changes.", Value: "-p"},
								{Name: "summary", Description: "For each path, show only whether it was modified, added, or deleted", Value: "-s", ConflictingFlags: []string{"--stat", "--types", "--name-only"}},
								{Name: "stat", Description: "Show a histogram of the changes", Value: "--stat", ConflictingFlags: []string{"-s", "--types", "--name-only"}},
								{Name: "types", Description: "For each path, show only its type before and after\n\nThe diff is shown as two letters. The first letter indicates the type before and the second letter indicates the type after. '-' indicates that the path was not present, 'F' represents a regular file, `L' represents a symlink, 'C' represents a conflict, and 'G' represents a Git submodule.", Value: "--types", ConflictingFlags: []string{"-s", "--stat", "--name-only"}},
								{Name: "name-only", Description: "For each path, show only its path\n\nTypically useful for shell commands like: `jj diff -r @- --name-only | xargs perl -pi -e's/OLD/NEW/g`", Value: "--name-only", ConflictingFlags: []string{"-s", "--stat", "--types"}},
								{Name: "git", Description: "Show a Git-format diff", Value: "--git", ConflictingFlags: []string{"--color-words"}},
								{Name: "color-words", Description: "Show a word-level diff with changes indicated only by color", Value: "--color-words", ConflictingFlags: []string{"--git"}},
								{Name: "tool", Description: "Generate diff by external command\n\nA builtin format can also be specified as `:<name>`. For example, `--tool=:git` is equivalent to `--git`.", Value: "--tool", RequiresInput: true, InputType: "TOOL"},
								{Name: "context", Description: "Number of lines of context to show", Value: "--context", RequiresInput: true, InputType: "CONTEXT"},
								{Name: "show-changes-in", Description: "Show only changed revisions matching the given revset expression\n\nIf no revisions are specified, this defaults to the `revsets.op-diff-changes-in` setting.", Value: "--show-changes-in", RequiresInput: true, InputType: "REVSETS"},
							},
						},
						{
							Summary: "Show changes to the repository in an operation", Name: "show",
							Description: "Show changes to the repository in an operation",
							Args: []Arg{
								{Name: "OPERATION", Description: "Show repository changes in this operation, compared to its parent(s)\n\nDefault value: `@`"},
							},
							Flags: []Flag{
								{Name: "no-graph", Description: "Don't show the graph, show a flat list of modified changes", Value: "-G"},
								{Name: "template", Description: "Render the operation using the given template\n\nYou can specify arbitrary template expressions using the [built-in keywords]. See [`jj help -k templates`] for more information.\n\n[built-in keywords]: https://docs.jj-vcs.dev/latest/templates/#operation-keywords\n\n[`jj help -k templates`]: https://docs.jj-vcs.dev/latest/templates/", Value: "-T", RequiresInput: true, InputType: "TEMPLATE"},
								{Name: "patch", Description: "Show patch of modifications to changes\n\nIf the previous version has different parents, it will be temporarily rebased to the parents of the new version, so the diff is not contaminated by unrelated changes.", Value: "-p"},
								{Name: "no-op-diff", Description: "Do not show operation diff", Value: "--no-op-diff"},
								{Name: "summary", Description: "For each path, show only whether it was modified, added, or deleted", Value: "-s", ConflictingFlags: []string{"--stat", "--types", "--name-only"}},
								{Name: "stat", Description: "Show a histogram of the changes", Value: "--stat", ConflictingFlags: []string{"-s", "--types", "--name-only"}},
								{Name: "types", Description: "For each path, show only its type before and after\n\nThe diff is shown as two letters. The first letter indicates the type before and the second letter indicates the type after. '-' indicates that the path was not present, 'F' represents a regular file, `L' represents a symlink, 'C' represents a conflict, and 'G' represents a Git submodule.", Value: "--types", ConflictingFlags: []string{"-s", "--stat", "--name-only"}},
								{Name: "name-only", Description: "For each path, show only its path\n\nTypically useful for shell commands like: `jj diff -r @- --name-only | xargs perl -pi -e's/OLD/NEW/g`", Value: "--name-only", ConflictingFlags: []string{"-s", "--stat", "--types"}},
								{Name: "git", Description: "Show a Git-format diff", Value: "--git", ConflictingFlags: []string{"--color-words"}},
								{Name: "color-words", Description: "Show a word-level diff with changes indicated only by color", Value: "--color-words", ConflictingFlags: []string{"--git"}},
								{Name: "tool", Description: "Generate diff by external command\n\nA builtin format can also be specified as `:<name>`. For example, `--tool=:git` is equivalent to `--git`.", Value: "--tool", RequiresInput: true, InputType: "TOOL"},
								{Name: "context", Description: "Number of lines of context to show", Value: "--context", RequiresInput: true, InputType: "CONTEXT"},
								{Name: "ignore-all-space", Description: "Ignore whitespace when comparing lines", Value: "--ignore-all-space", ConflictingFlags: []string{"--ignore-space-change"}},
								{Name: "ignore-space-change", Description: "Ignore changes in amount of whitespace when comparing lines", Value: "--ignore-space-change", ConflictingFlags: []string{"--ignore-all-space"}},
								{Name: "show-changes-in", Description: "Show only changed revisions matching the given revset expression\n\nIf no revisions are specified, this defaults to the `revsets.op-diff-changes-in` setting.", Value: "--show-changes-in", RequiresInput: true, InputType: "REVSETS"},
							},
						},
						{
							Summary: "Compare changes to the repository between two operations", Name: "diff",
							Description: "Compare changes to the repository between two operations",
							Flags: []Flag{
								{Name: "operation", Description: "Show repository changes in this operation, compared to its parent", Value: "--operation", RequiresInput: true, InputType: "OPERATION"},
								{Name: "from", Description: "Show repository changes from this operation", Value: "--from", RequiresInput: true, InputType: "FROM"},
								{Name: "to", Description: "Show repository changes to this operation", Value: "--to", RequiresInput: true, InputType: "TO"},
								{Name: "no-graph", Description: "Don't show the graph, show a flat list of modified changes", Value: "-G"},
								{Name: "patch", Description: "Show patch of modifications to changes\n\nIf the previous version has different parents, it will be temporarily rebased to the parents of the new version, so the diff is not contaminated by unrelated changes.", Value: "-p"},
								{Name: "summary", Description: "For each path, show only whether it was modified, added, or deleted", Value: "--summary", ConflictingFlags: []string{"--name-only", "--stat", "--types"}},
								{Name: "stat", Description: "Show a histogram of the changes", Value: "--stat", ConflictingFlags: []string{"--name-only", "--summary", "--types"}},
								{Name: "types", Description: "For each path, show only its type before and after\n\nThe diff is shown as two letters. The first letter indicates the type before and the second letter indicates the type after. '-' indicates that the path was not present, 'F' represents a regular file, `L' represents a symlink, 'C' represents a conflict, and 'G' represents a Git submodule.", Value: "--types", ConflictingFlags: []string{"--summary", "--stat", "--name-only"}},
								{Name: "name-only", Description: "For each path, show only its path\n\nTypically useful for shell commands like: `jj diff -r @- --name-only | xargs perl -pi -e's/OLD/NEW/g`", Value: "--name-only", ConflictingFlags: []string{"--summary", "--stat", "--types"}},
								{Name: "git", Description: "Show a Git-format diff", Value: "--git", ConflictingFlags: []string{"--color-words"}},
								{Name: "color-words", Description: "Show a word-level diff with changes indicated only by color", Value: "--color-words", ConflictingFlags: []string{"--git"}},
								{Name: "tool", Description: "Generate diff by external command\n\nA builtin format can also be specified as `:<name>`. For example, `--tool=:git` is equivalent to `--git`.", Value: "--tool", RequiresInput: true, InputType: "TOOL"},
								{Name: "context", Description: "Number of lines of context to show", Value: "--context", RequiresInput: true, InputType: "CONTEXT"},
								{Name: "ignore-all-space", Description: "Ignore whitespace when comparing lines", Value: "--ignore-all-space", ConflictingFlags: []string{"--ignore-space-change"}},
								{Name: "ignore-space-change", Description: "Ignore changes in amount of whitespace when comparing lines", Value: "--ignore-space-change", ConflictingFlags: []string{"--ignore-all-space"}},
								{Name: "show-changes-in", Description: "Show only changed revisions matching the given revset expression\n\nIf no revisions are specified, this defaults to the `revsets.op-diff-changes-in` setting.", Value: "--show-changes-in", RequiresInput: true, InputType: "REVSETS"},
							},
						},
						{
							Summary: "Create a new operation that restores the repo to an earlier state", Name: "restore",
							Description: "Create a new operation that restores the repo to an earlier state\n\nThis restores the repo to the state at the specified operation, effectively undoing all later operations. It does so by creating a new operation.",
							Args: []Arg{
								{Name: "OPERATION", Description: "The operation to restore to\n\nUse `jj op log` to find an operation to restore to. Use e.g. `jj --at-op=<operation ID> log` before restoring to an operation to see the state of the repo at that operation.", Required: true},
							},
							Flags: []Flag{
								{Name: "what", Description: "What portions of the local state to restore (can be repeated)\n\nThis option is EXPERIMENTAL.\n\nDefault values: `repo`, `remote-tracking`\n\nPossible values: - `repo`: The jj repo state and local bookmarks - `remote-tracking`: The remote-tracking bookmarks. Do not restore these if you'd like to push after the undo", Value: "--what", RequiresInput: true, InputType: "WHAT"},
							},
						},
						{
							Summary: "Create a new operation that reverts an earlier operation", Name: "revert",
							Description: "Create a new operation that reverts an earlier operation\n\nThis reverts an individual operation by applying the inverse of the operation.",
							Args: []Arg{
								{Name: "OPERATION", Description: "The operation to revert\n\nUse `jj op log` to find an operation to revert.\n\nDefault value: `@`"},
							},
							Flags: []Flag{},
						},
						{
							Summary: "Abandon operation history", Name: "abandon",
							Description: "Abandon operation history\n\nTo discard old operation history, use `jj op abandon ..<operation ID>`. It will abandon the specified operation and all its ancestors. The descendants will be reparented onto the root operation.\n\nTo discard recent operations, use `jj op restore <operation ID>` followed by `jj op abandon <operation ID>..@-`.\n\nPrevious versions of a change (or predecessors) are also discarded if they become unreachable from the operation history. The abandoned operations, commits, and other unreachable objects can later be garbage collected by using `jj util gc` command.",
							Args: []Arg{
								{Name: "OPERATION", Description: "The operation or operation range to abandon", Required: true},
							},
							Flags: []Flag{},
						},
						{
							Summary: "Make an operation part of the operation log", Name: "integrate",
							Description: "Make an operation part of the operation log\n\nBy default, operations are automatically integrated into the operation log, but `--no-integrate-operation` or internal errors may cause that to not happen. This command can then be used for making such operations part of the operation log.\n\nRunning this command on an operation that is already in the operation log (`jj op log`) has no effect.",
							Args: []Arg{
								{Name: "OPERATION", Description: "The operation to integrate"},
							},
							Flags: []Flag{},
						},
					},
				},
				{
					Name:        "undo",
					Description: "Undo the last operation\n\nIf used once after a normal (non-`undo`) operation, this will undo that last operation by restoring its parent. If `jj undo` is used repeatedly, it will restore increasingly older operations, going further back into the past.\n\nThere is also a complementary `jj redo` command that would instead move in the direction of the future after one or more `jj undo`s.\n\nUse `jj op log` to visualize the log of past operations, including a detailed description of any past undo/redo operations. See also `jj op restore` to explicitly restore an older operation by its id (available in the operation log).",
					Flags:       []Flag{},
				},
				{
					Name:        "redo",
					Description: "Redo the most recently undone operation\n\nThis is the natural counterpart of `jj undo`. Repeated invocations of `jj undo` and `jj redo` act similarly to Undo/Redo commands in a text editor.\n\nUse `jj op log` to visualize the log of past operations, including a detailed description of any past undo/redo operations. See also `jj op restore` to explicitly restore an older operation by its id (available in the operation log).",
					Flags:       []Flag{},
				},
			},
		},
		{
			Name: "Advanced",
			Commands: []Command{
				{
					Name:        "sign",
					Description: "Cryptographically sign a revision\n\nThis command requires configuring a [commit signing] backend.\n\n[commit signing]: https://docs.jj-vcs.dev/latest/config/#commit-signing",
					Flags: []Flag{
						{Name: "revision", Description: "What revision(s) to sign\n\nIf no revisions are specified, this defaults to the `revsets.sign` setting.\n\nNote that revisions are always re-signed.\n\nWhile that leads to discomfort for users, which sign with hardware devices, as of now we cannot reliably check if a commit is already signed by the user without creating a signature (see [#5786]).\n\n[#5786]: https://github.com/jj-vcs/jj/issues/5786", Value: "-r", RequiresInput: true, InputType: "REVSETS"},
						{Name: "key", Description: "The key used for signing", Value: "--key", RequiresInput: true, InputType: "KEY"},
					},
				},
				{
					Name:        "unsign",
					Description: "Drop a cryptographic signature\n\nSee also [commit signing] docs.\n\n[commit signing]: https://docs.jj-vcs.dev/latest/config/#commit-signing",
					Flags: []Flag{
						{Name: "revision", Description: "What revision(s) to unsign", Value: "-r", RequiresInput: true, InputType: "REVSETS"},
					},
				},
				{
					Name:        "sparse",
					Description: "Manage which paths from the working-copy commit are present in the working copy",
					SubCmds: []SubCommand{
						{
							Summary: "List the patterns that are currently present in the working copy", Name: "list",
							Description: "List the patterns that are currently present in the working copy\n\nBy default, a newly cloned or initialized repo will have have a pattern matching all files from the repo root. That pattern is rendered as `.` (a single period).",
							Flags:       []Flag{},
						},
						{
							Summary: "Update the patterns that are present in the working copy", Name: "set",
							Description: "Update the patterns that are present in the working copy\n\nFor example, if all you need is the `README.md` and the `lib/` directory, use `jj sparse set --clear --add README.md --add lib`. If you no longer need the `lib` directory, use `jj sparse set --remove lib`.",
							Flags: []Flag{
								{Name: "add", Description: "Patterns to add to the working copy", Value: "--add", RequiresInput: true, InputType: "ADD"},
								{Name: "remove", Description: "Patterns to remove from the working copy", Value: "--remove", RequiresInput: true, InputType: "REMOVE"},
								{Name: "clear", Description: "Include no files in the working copy (combine with --add)", Value: "--clear"},
							},
						},
						{
							Summary: "Reset the patterns to include all files in the working copy", Name: "reset",
							Description: "Reset the patterns to include all files in the working copy",
							Flags:       []Flag{},
						},
					},
				},
				{
					Name:        "config",
					Description: "Manage config options\n\nOperates on jj configuration, which comes from the config file and environment variables.\n\nSee [`jj help -k config`] to know more about file locations, supported config options, and other details about `jj config`.\n\n[`jj help -k config`]: https://docs.jj-vcs.dev/latest/config/",
					SubCmds: []SubCommand{
						{
							Summary: "Get the value of a given config option.", Name: "get",
							Alias:       "g",
							Description: "Get the value of a given config option.\n\nUnlike `jj config list`, the result of `jj config get` is printed without extra formatting and therefore is usable in scripting. For example:\n\n$ jj config list user.name user.name=\"Martin von Zweigbergk\" $ jj config get user.name Martin von Zweigbergk",
							Args: []Arg{
								{Name: "NAME", Description: "", Required: true},
							},
							Flags: []Flag{},
						},
						{
							Summary: "List variables set in config files, along with their values", Name: "list",
							Alias:       "l",
							Description: "List variables set in config files, along with their values",
							Args: []Arg{
								{Name: "NAME", Description: "An optional name of a specific config option to look up"},
							},
							Flags: []Flag{
								{Name: "include-defaults", Description: "Whether to explicitly include built-in default values in the list", Value: "--include-defaults"},
								{Name: "user", Description: "Target the user-level config", Value: "--user", ConflictingFlags: []string{"--repo", "--workspace"}},
								{Name: "repo", Description: "Target the repo-level config", Value: "--repo", ConflictingFlags: []string{"--user", "--workspace"}},
								{Name: "workspace", Description: "Target the workspace-level config", Value: "--workspace", ConflictingFlags: []string{"--user", "--repo"}},
							},
						},
						{
							Summary: "Print the paths to the config files", Name: "path",
							Alias:       "p",
							Description: "Print the paths to the config files\n\nA config file at that path may or may not exist.\n\nIf `--repo` or `--workspace` is specified and the config file does not exist, jj will generate a new config directory for this repo/workspace and print the path to the config file in that directory.\n\nSee `jj config edit` if you'd like to immediately edit a file.",
							Flags: []Flag{
								{Name: "user", Description: "Target the user-level config", Value: "--user", ConflictingFlags: []string{"--repo", "--workspace"}},
								{Name: "repo", Description: "Target the repo-level config", Value: "--repo", ConflictingFlags: []string{"--user", "--workspace"}},
								{Name: "workspace", Description: "Target the workspace-level config", Value: "--workspace", ConflictingFlags: []string{"--user", "--repo"}},
							},
							RequiredFlagGroup: []string{"--user", "--repo", "--workspace"},
							RequiredUsage:     "<--user|--repo|--workspace>",
						},
						{
							Summary: "Update a config file to set the given option to a given value", Name: "set",
							Alias:       "s",
							Description: "Update a config file to set the given option to a given value",
							Args: []Arg{
								{Name: "NAME", Description: "", Required: true},
								{Name: "VALUE", Description: "New value to set\n\nThe value should be specified as a TOML expression. If string value isn't enclosed by any TOML constructs (such as apostrophes or array notation), quotes can be omitted. Note that the value may also need shell quoting. TOML multi-line strings can be useful if the value contains apostrophes. For example, to set `foo.bar` to the string \"{don't}\" use `jj config set --user foo.bar \"'''{don't}'''\"`. This is valid in both Bash and Fish.\n\nAlternative, e.g. to avoid dealing with shell quoting, use `jj config edit` to edit the TOML file directly.", Required: true},
							},
							Flags: []Flag{
								{Name: "user", Description: "Target the user-level config", Value: "--user", ConflictingFlags: []string{"--repo", "--workspace"}},
								{Name: "repo", Description: "Target the repo-level config", Value: "--repo", ConflictingFlags: []string{"--user", "--workspace"}},
								{Name: "workspace", Description: "Target the workspace-level config", Value: "--workspace", ConflictingFlags: []string{"--user", "--repo"}},
							},
							RequiredFlagGroup: []string{"--user", "--repo", "--workspace"},
							RequiredUsage:     "<--user|--repo|--workspace>",
						},
						{
							Summary: "Update a config file to unset the given option", Name: "unset",
							Alias:       "u",
							Description: "Update a config file to unset the given option",
							Args: []Arg{
								{Name: "NAME", Description: "", Required: true},
							},
							Flags: []Flag{
								{Name: "user", Description: "Target the user-level config", Value: "--user", ConflictingFlags: []string{"--repo", "--workspace"}},
								{Name: "repo", Description: "Target the repo-level config", Value: "--repo", ConflictingFlags: []string{"--user", "--workspace"}},
								{Name: "workspace", Description: "Target the workspace-level config", Value: "--workspace", ConflictingFlags: []string{"--user", "--repo"}},
							},
							RequiredFlagGroup: []string{"--user", "--repo", "--workspace"},
							RequiredUsage:     "<--user|--repo|--workspace>",
						},
						{
							Summary: "Start an editor on a jj config file", Name: "edit",
							Alias:       "e",
							Description: "Start an editor on a jj config file.\n\nCreates the file if it doesn't already exist regardless of what the editor does.",
							Flags: []Flag{
								{Name: "user", Description: "Target the user-level config", Value: "--user", ConflictingFlags: []string{"--repo", "--workspace"}},
								{Name: "repo", Description: "Target the repo-level config", Value: "--repo", ConflictingFlags: []string{"--user", "--workspace"}},
								{Name: "workspace", Description: "Target the workspace-level config", Value: "--workspace", ConflictingFlags: []string{"--user", "--repo"}},
							},
							RequiredFlagGroup: []string{"--user", "--repo", "--workspace"},
							RequiredUsage:     "<--user|--repo|--workspace>",
						},
					},
				},
				{
					Name:        "help",
					Description: "Print this message or the help of the given subcommand(s)",
					Args: []Arg{
						{Name: "COMMAND", Description: "Print help for the subcommand(s)", Variadic: true},
					},
					SubCmds: []SubCommand{
						{Summary: "", Name: "bookmarks", Value: "-k bookmarks", Description: "Print this message or the help of the given subcommand(s)"},
						{Summary: "", Name: "config", Value: "-k config", Description: "Print this message or the help of the given subcommand(s)"},
						{Summary: "", Name: "filesets", Value: "-k filesets", Description: "Print this message or the help of the given subcommand(s)"},
						{Summary: "", Name: "glossary", Value: "-k glossary", Description: "Print this message or the help of the given subcommand(s)"},
						{Summary: "", Name: "revsets", Value: "-k revsets", Description: "Print this message or the help of the given subcommand(s)"},
						{Summary: "", Name: "templates", Value: "-k templates", Description: "Print this message or the help of the given subcommand(s)"},
						{Summary: "", Name: "tutorial", Value: "-k tutorial", Description: "Print this message or the help of the given subcommand(s)"},
					},
					Flags: []Flag{},
				},
				{
					Name:        "workspace",
					Description: "Commands for working with workspaces\n\nWorkspaces let you add additional working copies attached to the same repo. A common use case is so you can run a slow build or test in one workspace while you're continuing to write code in another workspace.\n\nEach workspace has its own working-copy commit. When you have more than one workspace attached to a repo, they are indicated by `<workspace name>@` in `jj log`.\n\nEach workspace also has own sparse patterns.",
					SubCmds: []SubCommand{
						{
							Summary: "List workspaces", Name: "list",
							Description: "List workspaces",
							Flags: []Flag{
								{Name: "template", Description: "Render each workspace using the given template\n\nAll 0-argument methods of the [`WorkspaceRef` type] are available as keywords in the template expression. See [`jj help -k templates`] for more information.\n\n[`WorkspaceRef` type]: https://docs.jj-vcs.dev/latest/templates/#workspaceref-type\n\n[`jj help -k templates`]: https://docs.jj-vcs.dev/latest/templates/", Value: "-T", RequiresInput: true, InputType: "TEMPLATE"},
							},
						},
						{
							Summary: "Show the workspace root directory", Name: "root",
							Description: "Show the workspace root directory",
							Flags:       []Flag{},
						},
						{
							Summary: "Add a workspace", Name: "add",
							Description: "Add a workspace\n\nBy default, the new workspace inherits the sparse patterns of the current workspace. You can override this with the `--sparse-patterns` option.",
							Args: []Arg{
								{Name: "DESTINATION", Description: "Where to create the new workspace", Required: true},
							},
							Flags: []Flag{
								{Name: "name", Description: "A name for the workspace\n\nTo override the default, which is the basename of the destination directory.", Value: "--name", RequiresInput: true, InputType: "NAME"},
								{Name: "revision", Description: "A list of parent revisions for the working-copy commit of the newly created workspace. You may specify nothing, or any number of parents.\n\nIf no revisions are specified, the new workspace will be created, and its working-copy commit will exist on top of the parent(s) of the working-copy commit in the current workspace, i.e. they will share the same parent(s).\n\nIf any revisions are specified, the new workspace will be created, and the new working-copy commit will be created with all these revisions as parents, i.e. the working-copy commit will exist as if you had run `jj new r1 r2 r3 ...`.", Value: "-r", RequiresInput: true, InputType: "REVSETS"},
								{Name: "message", Description: "The change description to use", Value: "-m", RequiresInput: true, NeedsQuotes: true, InputType: "MESSAGE"},
								{Name: "sparse-patterns", Description: "How to handle sparse patterns when creating a new workspace\n\nDefault value: `copy`\n\nPossible values: - `copy`: Copy all sparse patterns from the current workspace - `full`: Include all files in the new workspace - `empty`: Clear all files from the workspace (it will be empty)", Value: "--sparse-patterns", RequiresInput: true, InputType: "SPARSE_PATTERNS"},
							},
						},
						{
							Summary: "Renames the current workspace", Name: "rename",
							Description: "Renames the current workspace",
							Args: []Arg{
								{Name: "NEW_WORKSPACE_NAME", Description: "The name of the workspace to update to"},
							},
							Flags: []Flag{},
						},
						{
							Summary: "Stop tracking a workspace's working-copy commit in the repo", Name: "forget",
							Description: "Stop tracking a workspace's working-copy commit in the repo\n\nThe workspace will not be touched on disk. It can be deleted from disk before or after running this command.",
							Args: []Arg{
								{Name: "WORKSPACES", Description: "Names of the workspaces to forget. By default, forgets only the current workspace", Variadic: true},
							},
							Flags: []Flag{},
						},
						{
							Summary: "Update a workspace that has become stale", Name: "update-stale",
							Description: "Update a workspace that has become stale\n\nSee the [stale working copy documentation] for more information.\n\n[stale working copy documentation]: https://docs.jj-vcs.dev/latest/working-copy/#stale-working-copy",
							Flags:       []Flag{},
						},
					},
				},
				{
					Name:        "util",
					Description: "Infrequently used commands such as for generating shell completions",
					SubCmds: []SubCommand{
						{
							Summary: "Run backend-dependent garbage collection", Name: "gc",
							Description: "Run backend-dependent garbage collection.\n\nTo garbage-collect old operations and the commits/objects referenced by them, run `jj op abandon ..<some old operation>` before `jj util gc`.",
							Flags: []Flag{
								{Name: "expire", Description: "Time threshold\n\nBy default, only obsolete objects and operations older than 2 weeks are pruned.\n\nOnly the string \"now\" can be passed to this parameter. Support for arbitrary absolute and relative timestamps will come in a subsequent release.", Value: "--expire", RequiresInput: true, InputType: "EXPIRE"},
							},
						},
						{
							Summary: "", Name: "completion bash",
							Description: "Print a command-line-completion script\n\nApply it by running one of these:\n\n- Bash: `source <(jj util completion bash)`\n- Fish: `jj util completion fish | source`\n- Nushell:\n```nu\njj util completion nushell | save -f \"completions-jj.nu\"\nuse \"completions-jj.nu\" *  # Or `source \"completions-jj.nu\"`\n```\n- Zsh:\n```shell\nautoload -U compinit\ncompinit\nsource <(jj util completion zsh)\n```\n\nSee the docs on [command-line completion] for more details.\n\n[command-line completion]:\nhttps://docs.jj-vcs.dev/latest/install-and-setup/#command-line-completion",
							Args: []Arg{
								{Name: "SHELL", Description: "[possible values: bash, elvish, fish, nushell, power-shell, zsh]"},
							},
							Flags: []Flag{},
						},
						{
							Summary: "", Name: "completion zsh",
							Description: "Print a command-line-completion script\n\nApply it by running one of these:\n\n- Bash: `source <(jj util completion bash)`\n- Fish: `jj util completion fish | source`\n- Nushell:\n```nu\njj util completion nushell | save -f \"completions-jj.nu\"\nuse \"completions-jj.nu\" *  # Or `source \"completions-jj.nu\"`\n```\n- Zsh:\n```shell\nautoload -U compinit\ncompinit\nsource <(jj util completion zsh)\n```\n\nSee the docs on [command-line completion] for more details.\n\n[command-line completion]:\nhttps://docs.jj-vcs.dev/latest/install-and-setup/#command-line-completion",
							Args: []Arg{
								{Name: "SHELL", Description: "[possible values: bash, elvish, fish, nushell, power-shell, zsh]"},
							},
							Flags: []Flag{},
						},
						{
							Summary: "", Name: "completion fish",
							Description: "Print a command-line-completion script\n\nApply it by running one of these:\n\n- Bash: `source <(jj util completion bash)`\n- Fish: `jj util completion fish | source`\n- Nushell:\n```nu\njj util completion nushell | save -f \"completions-jj.nu\"\nuse \"completions-jj.nu\" *  # Or `source \"completions-jj.nu\"`\n```\n- Zsh:\n```shell\nautoload -U compinit\ncompinit\nsource <(jj util completion zsh)\n```\n\nSee the docs on [command-line completion] for more details.\n\n[command-line completion]:\nhttps://docs.jj-vcs.dev/latest/install-and-setup/#command-line-completion",
							Args: []Arg{
								{Name: "SHELL", Description: "[possible values: bash, elvish, fish, nushell, power-shell, zsh]"},
							},
							Flags: []Flag{},
						},
					},
				},
			},
		},
		{
			Name: "Setup",
			Commands: []Command{
				{
					Name:        "git",
					Description: "Commands for working with Git remotes and the underlying Git repo\n\nSee this [comparison], including a [table of commands].\n\n[comparison]: https://docs.jj-vcs.dev/latest/git-comparison/.\n\n[table of commands]: https://docs.jj-vcs.dev/latest/git-command-table",
					SubCmds: []SubCommand{
						{
							Summary: "Create a new repo backed by a clone of a Git repo", Name: "clone",
							Description: "Create a new repo backed by a clone of a Git repo",
							Args: []Arg{
								{Name: "SOURCE", Description: "URL or path of the Git repo to clone\n\nLocal path will be resolved to absolute form.", Required: true},
								{Name: "DESTINATION", Description: "Specifies the target directory for the Jujutsu repository clone. If not provided, defaults to a directory named after the last component of the source URL. The full directory path will be created if it doesn't exist"},
							},
							Flags: []Flag{
								{Name: "depth", Description: "Create a shallow clone of the given depth", Value: "--depth", RequiresInput: true, InputType: "DEPTH"},
								{Name: "colocate", Description: "Colocate the Jujutsu repo with the git repo\n\nSpecifies that the `jj` repo should also be a valid `git` repo, allowing the use of both `jj` and `git` commands in the same directory.\n\nThe repository will contain a `.git` dir in the top-level. Regular Git tools will be able to operate on the repo.\n\n**This is the default**, and this option has no effect, unless the [git.colocate config] is set to `false`.\n\n[git.colocate config]: https://docs.jj-vcs.dev/latest/config/#default-colocation", Value: "--colocate"},
							},
						},
						{
							Summary: "Create a new Git backed repo", Name: "init",
							Description: "Create a new Git backed repo",
							Args: []Arg{
								{Name: "DESTINATION", Description: "The destination directory where the `jj` repo will be created. If the directory does not exist, it will be created. If no directory is given, the current directory is used.\n\nBy default the `git` repo is under `$destination/.jj`\n\nDefault value: `.`"},
							},
							Flags: []Flag{
								{Name: "colocate", Description: "Colocate the Jujutsu repo with the git repo\n\nSpecifies that the `jj` repo should also be a valid `git` repo, allowing the use of both `jj` and `git` commands in the same directory.\n\nThe repository will contain a `.git` dir in the top-level. Regular Git tools will be able to operate on the repo.\n\n**This is the default**, and this option has no effect, unless the [git.colocate config] is set to `false`.\n\nThis option is mutually exclusive with `--git-repo`.\n\n[git.colocate config]: https://docs.jj-vcs.dev/latest/config/#default-colocation", Value: "--colocate", ConflictingFlags: []string{"--no-colocate", "--git-repo"}},
								{Name: "no-colocate", Description: "Disable colocation of the Jujutsu repo with the git repo\n\nPrevent Git tools that are unaware of `jj` and regular Git commands from operating on the repo. The Git repository that stores most of the repo data will be hidden inside a sub-directory of the `.jj` directory.\n\nSee [colocation docs] for some minor advantages of non-colocated workspaces.\n\n[colocation docs]: https://docs.jj-vcs.dev/latest/git-compatibility/#colocated-jujutsugit-repos", Value: "--no-colocate", ConflictingFlags: []string{"--colocate", "--git-repo"}},
								{Name: "git-repo", Description: "Specifies a path to an **existing** git repository to be used as the backing git repo for the newly created `jj` repo.\n\nIf the specified `--git-repo` path happens to be the same as the `jj` repo path (both .jj and .git directories are in the same working directory), then both `jj` and `git` commands will work on the same repo. This is called a colocated workspace.\n\nThis option is mutually exclusive with `--colocate`, and so if passed, turns colocation off.", Value: "--git-repo", RequiresInput: true, ConflictingFlags: []string{"--colocate", "--no-colocate"}, InputType: "GIT_REPO"},
							},
						},
					},
				},
			},
		},
	}
}
