// gen-descriptions rewrites the Description fields in jj_commands.go using
// `jj util markdown-help` as the source of truth. That subcommand renders
// every paragraph as a single flowing line (blank line = paragraph break),
// unlike `jj <cmd> --help` which hard-wraps prose to the terminal width.
// Nothing else in jj_commands.go (Value, Mandatory, ConflictingFlags, Alias,
// ...) is touched: only Description string literals are replaced in place,
// by byte-splicing the original source, so unrelated formatting survives.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// doc holds the parsed text for one `## `jj foo bar“ section of
// markdown-help: its own description, plus per-argument and per-flag
// descriptions found in that section's Arguments/Options lists.
type doc struct {
	description   string
	args          map[string]string // Arg Name -> description
	options       map[string]string // flag token ("-r", "--revision", ...) -> description
	optionAliases map[string]string // flag token -> word alias ("after" for --insert-after)
	usage         string            // raw "jj <path> ..." text from the "**Usage:**" line
	subcommands   map[string]string // child subcommand Name -> one-line summary
}

func main() {
	out, err := exec.Command("jj", "util", "markdown-help").Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "running jj util markdown-help:", err)
		os.Exit(1)
	}
	docs := parseMarkdownHelp(string(out))

	const srcPath = "jj_commands.go"
	src, err := os.ReadFile(srcPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, srcPath, src, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parsing", srcPath, err)
		os.Exit(1)
	}

	edits := collectEdits(fset, file, docs)
	if len(edits) == 0 {
		fmt.Fprintln(os.Stderr, "no matching Description fields found; nothing to do")
		os.Exit(1)
	}

	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	updated := src
	applied, skipped := 0, 0
	for _, e := range edits {
		if e.newText == "" && !e.allowEmpty {
			skipped++
			continue
		}
		updated = append(updated[:e.start], append([]byte(strconv.Quote(e.newText)), updated[e.end:]...)...)
		applied++
	}

	if err := os.WriteFile(srcPath, updated, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("gen-descriptions: updated %d descriptions, %d had no match in markdown-help\n", applied, skipped)
}

// edit is a byte-range replacement for one Description (or other) string literal.
type edit struct {
	start, end int // byte offsets of the literal (including quotes) in src
	newText    string
	label      string // for diagnostics
	allowEmpty bool   // write newText even if "" (a genuinely empty result, not "no match found")
}

func collectEdits(fset *token.FileSet, file *ast.File, docs map[string]doc) []edit {
	var edits []edit
	var missing []string

	lookup := func(path string) (doc, bool) {
		d, ok := docs[path]
		return d, ok
	}

	recordDescription := func(lit *ast.CompositeLit, path string) {
		val := fieldValue(lit, "Description")
		if val == nil {
			return
		}
		bl, ok := val.(*ast.BasicLit)
		if !ok || bl.Kind != token.STRING {
			return
		}
		d, ok := lookup(path)
		if !ok || d.description == "" {
			missing = append(missing, path)
			return
		}
		edits = append(edits, edit{start: fset.Position(bl.Pos()).Offset, end: fset.Position(bl.End()).Offset, newText: d.description, label: path})
	}

	recordFlag := func(lit *ast.CompositeLit, cmdPath string) {
		valueLit, _ := fieldValue(lit, "Value").(*ast.BasicLit)
		descLit, _ := fieldValue(lit, "Description").(*ast.BasicLit)
		if valueLit == nil || descLit == nil || descLit.Kind != token.STRING {
			return
		}
		flagValue, _ := strconv.Unquote(valueLit.Value)
		d, ok := lookup(cmdPath)
		label := cmdPath + " " + flagValue
		if !ok {
			missing = append(missing, label)
			return
		}
		newText, ok := d.options[flagValue]
		if !ok {
			missing = append(missing, label)
			return
		}
		edits = append(edits, edit{start: fset.Position(descLit.Pos()).Offset, end: fset.Position(descLit.End()).Offset, newText: newText, label: label})

		// Alias drift check: the Alias field is hand-authored (no AST field
		// insertion here), so rewrite it when present and report when the DB
		// and jj help disagree about its existence.
		helpAlias := d.optionAliases[flagValue]
		aliasLit, _ := fieldValue(lit, "Alias").(*ast.BasicLit)
		switch {
		case aliasLit != nil && aliasLit.Kind == token.STRING:
			if helpAlias == "" {
				missing = append(missing, label+" (has Alias but jj help declares none — remove by hand)")
			} else if cur, _ := strconv.Unquote(aliasLit.Value); cur != helpAlias {
				edits = append(edits, edit{start: fset.Position(aliasLit.Pos()).Offset, end: fset.Position(aliasLit.End()).Offset, newText: helpAlias, label: label + " (alias)"})
			}
		case helpAlias != "":
			missing = append(missing, label+" (jj help declares alias \""+helpAlias+"\" — add Alias field by hand)")
		}
	}

	recordArg := func(lit *ast.CompositeLit, cmdPath string) {
		nameLit, _ := fieldValue(lit, "Name").(*ast.BasicLit)
		descLit, _ := fieldValue(lit, "Description").(*ast.BasicLit)
		if nameLit == nil || descLit == nil || descLit.Kind != token.STRING {
			return
		}
		name, _ := strconv.Unquote(nameLit.Value)
		d, ok := lookup(cmdPath)
		label := cmdPath + " <" + name + ">"
		if !ok {
			missing = append(missing, label)
			return
		}
		newText, ok := d.args[name]
		if !ok {
			missing = append(missing, label)
			return
		}
		edits = append(edits, edit{start: fset.Position(descLit.Pos()).Offset, end: fset.Position(descLit.End()).Offset, newText: newText, label: label})
	}

	recordRequiredUsage := func(lit *ast.CompositeLit, cmdPath string) {
		val := fieldValue(lit, "RequiredUsage")
		if val == nil {
			return
		}
		bl, ok := val.(*ast.BasicLit)
		if !ok || bl.Kind != token.STRING {
			return
		}
		label := cmdPath + " (usage)"
		d, ok := lookup(cmdPath)
		if !ok || d.usage == "" {
			missing = append(missing, label)
			return
		}
		remainder := strings.TrimPrefix(d.usage, cmdPath)
		if remainder == d.usage && remainder != cmdPath {
			// The usage line didn't start with "jj <path>" as expected.
			missing = append(missing, label+" (unexpected usage format)")
			return
		}
		remainder = strings.TrimPrefix(remainder, " ")

		tokens := tokenizeUsage(remainder)
		// markdown-help inconsistently includes a leading "[OPTIONS]" token
		// (present for e.g. rebase/log, absent for revert/status/config set),
		// unlike `jj <cmd> --help` which always shows it. Jutsu already adds
		// its own "[OPTIONS]" unconditionally, so drop this one to avoid a
		// duplicate.
		if len(tokens) > 0 && tokens[0] == "[OPTIONS]" {
			tokens = tokens[1:]
		}
		argCount := min(len(compositeElts(fieldValue(lit, "Args"))), len(tokens))
		fragment := strings.Join(tokens[:len(tokens)-argCount], " ")
		edits = append(edits, edit{
			start: fset.Position(bl.Pos()).Offset, end: fset.Position(bl.End()).Offset,
			newText: fragment, label: label, allowEmpty: true,
		})
	}

	recordSummary := func(lit *ast.CompositeLit, parentPath, name string) {
		val := fieldValue(lit, "Summary")
		if val == nil {
			return
		}
		bl, ok := val.(*ast.BasicLit)
		if !ok || bl.Kind != token.STRING {
			return
		}
		label := parentPath + " " + name + " (summary)"
		// Some entries flatten a nested jj command group into one Name, e.g.
		// "remote add" under "jj git" really means "jj git remote"'s own
		// Subcommands listing has the "add" bullet, not "jj git"'s.
		lookupPath, childName := parentPath, name
		if group, rest, ok := strings.Cut(name, " "); ok {
			lookupPath = parentPath + " " + group
			childName = rest
		}
		d, ok := lookup(lookupPath)
		if !ok {
			missing = append(missing, label)
			return
		}
		newText, ok := d.subcommands[childName]
		if !ok {
			missing = append(missing, label)
			return
		}
		edits = append(edits, edit{start: fset.Position(bl.Pos()).Offset, end: fset.Position(bl.End()).Offset, newText: newText, label: label})
	}

	var walkFlags = func(lit *ast.CompositeLit, cmdPath string) {
		for _, el := range compositeElts(fieldValue(lit, "Flags")) {
			recordFlag(el, cmdPath)
		}
	}
	var walkArgs = func(lit *ast.CompositeLit, cmdPath string) {
		for _, el := range compositeElts(fieldValue(lit, "Args")) {
			recordArg(el, cmdPath)
		}
	}

	var walkSubCommand func(lit *ast.CompositeLit, parentPath string)
	walkSubCommand = func(lit *ast.CompositeLit, parentPath string) {
		name := stringField(lit, "Name")
		path := parentPath + " " + name
		recordDescription(lit, path)
		walkArgs(lit, path)
		walkFlags(lit, path)
		recordRequiredUsage(lit, path)
	}

	var walkCommand func(lit *ast.CompositeLit)
	walkCommand = func(lit *ast.CompositeLit) {
		name := stringField(lit, "Name")
		path := "jj " + name
		recordDescription(lit, path)
		walkArgs(lit, path)
		walkFlags(lit, path)
		recordRequiredUsage(lit, path)
		for _, el := range compositeElts(fieldValue(lit, "SubCmds")) {
			recordSummary(el, path, stringField(el, "Name"))
			walkSubCommand(el, path)
		}
	}

	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "loadCategories" {
			return true
		}
		for _, stmt := range fn.Body.List {
			ret, ok := stmt.(*ast.ReturnStmt)
			if !ok || len(ret.Results) != 1 {
				continue
			}
			catsLit, ok := ret.Results[0].(*ast.CompositeLit)
			if !ok {
				continue
			}
			for _, catEl := range catsLit.Elts {
				cat, ok := catEl.(*ast.CompositeLit)
				if !ok {
					continue
				}
				for _, cmdEl := range compositeElts(fieldValue(cat, "Commands")) {
					walkCommand(cmdEl)
				}
			}
		}
		return false
	})

	if len(missing) > 0 {
		fmt.Fprintln(os.Stderr, "gen-descriptions: drift against jj markdown-help (fix by hand):")
		for _, m := range missing {
			fmt.Fprintln(os.Stderr, "  -", m)
		}
	}
	return edits
}

// fieldValue returns the Value expression of the KeyValueExpr named key in
// composite literal lit, or nil. lit may itself be nil.
func fieldValue(lit *ast.CompositeLit, key string) ast.Expr {
	if lit == nil {
		return nil
	}
	for _, el := range lit.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if id, ok := kv.Key.(*ast.Ident); ok && id.Name == key {
			return kv.Value
		}
	}
	return nil
}

func stringField(lit *ast.CompositeLit, key string) string {
	bl, ok := fieldValue(lit, key).(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return ""
	}
	s, _ := strconv.Unquote(bl.Value)
	return s
}

// compositeElts returns the element composite literals of a slice literal
// expression (e.g. the value of a "Flags:" or "SubCmds:" field), or nil.
func compositeElts(expr ast.Expr) []*ast.CompositeLit {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	var out []*ast.CompositeLit
	for _, el := range lit.Elts {
		if cl, ok := el.(*ast.CompositeLit); ok {
			out = append(out, cl)
		}
	}
	return out
}

var (
	headerRe    = regexp.MustCompile("^## `(jj[^`]*)`$")
	sectionRe   = regexp.MustCompile(`^###### \*\*(.+):\*\*$`)
	usageRe     = regexp.MustCompile(`^\*\*Usage:\*\*`)
	usageLineRe = regexp.MustCompile("^\\*\\*Usage:\\*\\* `(.+)`$")
	bulletRe    = regexp.MustCompile(`^\* (.+)$`)
)

// parseMarkdownHelp splits `jj util markdown-help` output into one doc per
// "## `jj ...`" section and extracts its description, argument descriptions
// and option descriptions.
func parseMarkdownHelp(text string) map[string]doc {
	lines := strings.Split(text, "\n")
	docs := make(map[string]doc)

	var curPath string
	var curLines []string
	flush := func() {
		if curPath != "" {
			docs[curPath] = parseSection(curLines)
		}
	}
	for _, line := range lines {
		if m := headerRe.FindStringSubmatch(line); m != nil {
			flush()
			curPath = m[1]
			curLines = nil
			continue
		}
		curLines = append(curLines, line)
	}
	flush()
	return docs
}

// parseSection parses the lines that follow one "## `jj ...`" header, up to
// (but not including) the next header.
func parseSection(lines []string) doc {
	d := doc{args: map[string]string{}, options: map[string]string{}, optionAliases: map[string]string{}, subcommands: map[string]string{}}

	// Description: everything before "**Usage:**", reflowed into paragraphs.
	i := 0
	var descLines []string
	for i < len(lines) && !usageRe.MatchString(strings.TrimSpace(lines[i])) {
		descLines = append(descLines, lines[i])
		i++
	}
	d.description = joinParagraphs(descLines)

	// Capture the raw "jj <path> ..." text of the "**Usage:**" line, if present.
	if i < len(lines) {
		if m := usageLineRe.FindStringSubmatch(strings.TrimSpace(lines[i])); m != nil {
			d.usage = m[1]
		}
	}

	// Walk remaining "###### **Section:**" blocks (Arguments / Options /
	// Subcommands...).
	for i < len(lines) {
		m := sectionRe.FindStringSubmatch(strings.TrimSpace(lines[i]))
		if m == nil {
			i++
			continue
		}
		heading := m[1]
		i++
		var blockLines []string
		for i < len(lines) && !sectionRe.MatchString(strings.TrimSpace(lines[i])) && !headerRe.MatchString(lines[i]) {
			blockLines = append(blockLines, lines[i])
			i++
		}
		switch heading {
		case "Arguments":
			parseArgItems(blockLines, d.args)
		case "Options":
			parseOptionItems(blockLines, d.options, d.optionAliases)
		case "Subcommands":
			parseSubcommandItems(blockLines, d.subcommands)
		}
	}
	return d
}

// splitItems groups block lines into bullet items: each item starts at a
// line matching bulletRe (column 0) and runs until the next such line.
func splitItems(lines []string) [][]string {
	var items [][]string
	var cur []string
	for _, line := range lines {
		if bulletRe.MatchString(line) {
			if len(cur) > 0 {
				items = append(items, cur)
			}
			cur = []string{line}
			continue
		}
		if len(cur) > 0 {
			cur = append(cur, line)
		}
	}
	if len(cur) > 0 {
		items = append(items, cur)
	}
	return items
}

var argNameRe = regexp.MustCompile("^\\* `<([A-Za-z0-9_]+)>`(?:.*? — (.*))?$")

func parseArgItems(lines []string, out map[string]string) {
	for _, item := range splitItems(lines) {
		m := argNameRe.FindStringSubmatch(item[0])
		if m == nil {
			continue
		}
		name, first := m[1], m[2]
		out[name] = joinItemBody(first, item[1:])
	}
}

var subcmdNameRe = regexp.MustCompile("^\\* `([a-z][a-z0-9-]*)`(?:.*? — (.*))?$")

func parseSubcommandItems(lines []string, out map[string]string) {
	for _, item := range splitItems(lines) {
		m := subcmdNameRe.FindStringSubmatch(item[0])
		if m == nil {
			continue
		}
		name, first := m[1], m[2]
		out[name] = joinItemBody(first, item[1:])
	}
}

var optFormsRe = regexp.MustCompile(`^\* (.+?)(?: — (.*))?$`)
var optTokenRe = regexp.MustCompile("`(-{1,2}[A-Za-z0-9][A-Za-z0-9-]*)(?:\\s+<[^>]*>)?`")
var optAliasRe = regexp.MustCompile("\\[alias: `([^`]+)`\\]")

func parseOptionItems(lines []string, out, aliases map[string]string) {
	for _, item := range splitItems(lines) {
		m := optFormsRe.FindStringSubmatch(item[0])
		if m == nil {
			continue
		}
		forms, first := m[1], m[2]
		var alias string
		if am := optAliasRe.FindStringSubmatch(forms); am != nil {
			alias = am[1]
		}
		desc := joinItemBody(first, item[1:])
		for _, tok := range optTokenRe.FindAllStringSubmatch(forms, -1) {
			out[tok[1]] = desc
			if alias != "" {
				aliases[tok[1]] = alias
			}
		}
	}
}

// joinItemBody combines a bullet's inline summary (may be empty) with any
// indented continuation lines into paragraphs separated by "\n\n".
func joinItemBody(first string, rest []string) string {
	var paras []string
	if strings.TrimSpace(first) != "" {
		paras = append(paras, strings.TrimSpace(first))
	}
	paras = append(paras, splitParagraphs(rest)...)
	return strings.Join(paras, "\n\n")
}

// splitParagraphs groups lines into blank-line-separated paragraphs,
// trimming indentation and joining any wrapped physical lines within a
// paragraph with a single space. Fenced code blocks (```...```) are the
// exception: their lines (ASCII diagrams, config examples) are kept verbatim
// as one paragraph — fence markers stripped, dedented, and re-indented two
// spaces, the marker wordWrap (model.go) uses to skip reflowing a line.
func splitParagraphs(lines []string) []string {
	var paras []string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			paras = append(paras, strings.Join(cur, " "))
			cur = nil
		}
	}
	var fence []string
	inFence := false
	flushFence := func() {
		if block := preformatBlock(fence); block != "" {
			paras = append(paras, block)
		}
		fence = nil
	}
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") {
			if inFence {
				flushFence()
			} else {
				flush()
			}
			inFence = !inFence
			continue
		}
		if inFence {
			fence = append(fence, line)
			continue
		}
		if t == "" {
			flush()
			continue
		}
		cur = append(cur, t)
	}
	if inFence {
		flushFence()
	}
	flush()
	return paras
}

// preformatBlock renders a fenced code block's lines for the docs pane:
// leading/trailing blank lines dropped, common leading whitespace dedented,
// every line re-indented by two spaces (interior blank lines preserved).
func preformatBlock(lines []string) string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return ""
	}
	indent := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " "))
		if indent < 0 || n < indent {
			indent = n
		}
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue // out[i] stays "" — preserved blank line
		}
		out[i] = "  " + strings.TrimRight(l[indent:], " ")
	}
	return strings.Join(out, "\n")
}

func joinParagraphs(lines []string) string {
	return strings.Join(splitParagraphs(lines), "\n\n")
}

// tokenizeUsage splits a Usage-line remainder (e.g. "--revision <REVSETS>
// <--onto <REVSETS>|--insert-after <REVSETS>>") on top-level spaces, treating
// anything inside <...> or [...] as part of the same token so multi-word
// alternation groups stay together.
func tokenizeUsage(s string) []string {
	var tokens []string
	var cur strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '<', '[':
			depth++
			cur.WriteRune(r)
		case '>', ']':
			depth--
			cur.WriteRune(r)
		case ' ':
			if depth == 0 {
				if cur.Len() > 0 {
					tokens = append(tokens, cur.String())
					cur.Reset()
				}
			} else {
				cur.WriteRune(r)
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}
