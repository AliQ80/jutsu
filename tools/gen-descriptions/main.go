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
	description string
	args        map[string]string // Arg Name -> description
	options     map[string]string // flag token ("-r", "--revision", ...) -> description
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
		if e.newText == "" {
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

// edit is a byte-range replacement for one Description string literal.
type edit struct {
	start, end int // byte offsets of the literal (including quotes) in src
	newText    string
	label      string // for diagnostics
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
	}

	var walkCommand func(lit *ast.CompositeLit)
	walkCommand = func(lit *ast.CompositeLit) {
		name := stringField(lit, "Name")
		path := "jj " + name
		recordDescription(lit, path)
		walkArgs(lit, path)
		walkFlags(lit, path)
		for _, el := range compositeElts(fieldValue(lit, "SubCmds")) {
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
		fmt.Fprintln(os.Stderr, "gen-descriptions: no markdown-help text found for:")
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
	headerRe  = regexp.MustCompile("^## `(jj[^`]*)`$")
	sectionRe = regexp.MustCompile(`^###### \*\*(.+):\*\*$`)
	usageRe   = regexp.MustCompile(`^\*\*Usage:\*\*`)
	bulletRe  = regexp.MustCompile(`^\* (.+)$`)
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
	d := doc{args: map[string]string{}, options: map[string]string{}}

	// Description: everything before "**Usage:**", reflowed into paragraphs.
	i := 0
	var descLines []string
	for i < len(lines) && !usageRe.MatchString(strings.TrimSpace(lines[i])) {
		descLines = append(descLines, lines[i])
		i++
	}
	d.description = joinParagraphs(descLines)

	// Walk remaining "###### **Section:**" blocks (Arguments / Options /
	// Subcommands...); we only care about Arguments and Options.
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
			parseOptionItems(blockLines, d.options)
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

var optFormsRe = regexp.MustCompile(`^\* (.+?)(?: — (.*))?$`)
var optTokenRe = regexp.MustCompile("`(-{1,2}[A-Za-z0-9][A-Za-z0-9-]*)(?:\\s+<[^>]*>)?`")

func parseOptionItems(lines []string, out map[string]string) {
	for _, item := range splitItems(lines) {
		m := optFormsRe.FindStringSubmatch(item[0])
		if m == nil {
			continue
		}
		forms, first := m[1], m[2]
		desc := joinItemBody(first, item[1:])
		for _, tok := range optTokenRe.FindAllStringSubmatch(forms, -1) {
			out[tok[1]] = desc
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
// paragraph with a single space.
func splitParagraphs(lines []string) []string {
	var paras []string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			paras = append(paras, strings.Join(cur, " "))
			cur = nil
		}
	}
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			flush()
			continue
		}
		cur = append(cur, t)
	}
	flush()
	return paras
}

func joinParagraphs(lines []string) string {
	return strings.Join(splitParagraphs(lines), "\n\n")
}
