package skilldoc

import (
	"regexp"
	"slices"
	"sort"
	"strings"
)

// Doc is a documentation file fed to the validator: its display Path (for
// issue reporting) and raw markdown Body.
type Doc struct {
	Path string
	Body string
}

// Issue is one validation finding against the command oracle.
type Issue struct {
	Doc    string
	Line   int
	Kind   string // "unknown-command" | "unknown-flag" | "stale-fence" | "fence-topology"
	Detail string
}

// Validate checks every harvested `fduty …` example in docs against the dump:
// an example whose leading words resolve to no command path yields an
// unknown-command issue; an example flag absent from its command's flag set
// (and not a global flag) yields an unknown-flag issue. Placeholder tokens are
// skipped so documentation stand-ins (<id>, $VAR) never trip the validator.
// The verb cells of each card's intent table get the same two checks (see
// validateVerbTable).
func Validate(d Dump, docs []Doc) []Issue {
	idx := indexDump(d)
	var issues []Issue
	for _, doc := range docs {
		for _, ex := range HarvestExamples(doc.Body) {
			issues = append(issues, validateExample(idx, doc.Path, ex)...)
		}
		issues = append(issues, validateVerbTable(idx, doc)...)
	}
	return issues
}

// verbSpanRe matches a code span that opens with a command word. Everything a
// verb cell quotes besides verbs fails it: response fields (`change_id`), field
// paths (`.days`), card references (`reference/noise.md`) and wildcard families
// (`automation-rule-*`), which name no single command. A bare lowercase word
// still reads as a verb, so a card quotes a plain field name as a path.
var verbSpanRe = regexp.MustCompile(`^[a-z][a-z0-9-]*(\s|$)`)

// validateVerbTable checks every verb cell in doc's `| want | verb |` table. A
// cell may hold several code spans, each a verb relative to the card's group,
// optionally with positionals and flags; a span holding only a flag belongs to
// the verb before it. Spans carrying the binary word are skipped, because
// HarvestExamples already covers them.
func validateVerbTable(idx commandIndex, doc Doc) []Issue {
	cells := harvestVerbCells(doc.Body)
	if len(cells) == 0 {
		return nil
	}
	group, ok := cardGroup(doc.Body)
	if !ok {
		return []Issue{{
			Doc:    doc.Path,
			Line:   cells[0].Line,
			Kind:   "unknown-command",
			Detail: "verb table has no GENERATED fence to name its command group",
		}}
	}

	var issues []Issue
	for _, cell := range cells {
		verb := ""
		for _, span := range inlineSpans(cell.Text) {
			switch {
			case strings.HasPrefix(span, "--"):
				span = verb + " " + span
			case verbSpanRe.MatchString(span) && !firstWordIsBinary(span):
				verb = span
			default:
				continue
			}
			issues = append(issues, validateExample(idx, doc.Path, verbExample(idx, group, span, cell.Line))...)
		}
	}
	return issues
}

// cardGroup returns the command group a card documents: the group of its first
// GENERATED fence, which holds for a card split off a group under a subset fence
// whatever the card's file name.
func cardGroup(body string) (string, bool) {
	locs := FenceLocs(body)
	if len(locs) == 0 {
		return "", false
	}
	spec, err := ParseFenceID(locs[0].ID)
	return spec.Group, err == nil
}

// verbExample turns a verb span into an Example: relative to group, or, when
// only that resolves, as a full command path (`safari automation-rule-run`
// cited from the automation card). Positional placeholders are dropped, since
// validateExample treats a placeholder among the path words as a template and
// would wave a stale verb through.
func verbExample(idx commandIndex, group, span string, line int) Example {
	relative, _ := parseInvocation("fduty "+group+" "+span, line)
	relative.Tokens = dropPositionalPlaceholders(relative.Tokens)
	if _, _, ok := resolveCommand(idx, relative.Tokens); ok {
		return relative
	}
	full, _ := parseInvocation("fduty "+span, line)
	full.Tokens = dropPositionalPlaceholders(full.Tokens)
	if _, _, ok := resolveCommand(idx, full.Tokens); ok {
		return full
	}
	return relative
}

func dropPositionalPlaceholders(tokens []string) []string {
	return slices.DeleteFunc(tokens, func(t string) bool {
		return !strings.HasPrefix(t, "-") && HasPlaceholder(t)
	})
}

// CheckFences asserts every GENERATED fence embedded in docs matches a fresh
// render from the dump, and that each group's fences form a valid partition
// of the group's commands (see RenderGroupFences). A drifted fence or a start
// marker with no matching end marker yields a stale-fence issue; a malformed
// or unknown-group marker, and any partition violation, yields a
// fence-topology issue anchored at the group's first fence.
func CheckFences(d Dump, docs []Doc) []Issue {
	dumpGroups := map[string]bool{}
	for _, g := range groups(d) {
		dumpGroups[g] = true
	}

	type loc struct {
		doc  string
		body string
		off  int
		id   string
	}
	var issues []Issue
	byGroup := map[string][]loc{}
	for _, doc := range docs {
		for _, fl := range FenceLocs(doc.Body) {
			spec, err := ParseFenceID(fl.ID)
			if err != nil {
				issues = append(issues, Issue{
					Doc:    doc.Path,
					Line:   lineOf(doc.Body, fl.Offset),
					Kind:   "fence-topology",
					Detail: err.Error(),
				})
				continue
			}
			if !dumpGroups[spec.Group] {
				issues = append(issues, Issue{
					Doc:    doc.Path,
					Line:   lineOf(doc.Body, fl.Offset),
					Kind:   "fence-topology",
					Detail: "GENERATED:" + fl.ID + " names unknown command group " + spec.Group,
				})
				continue
			}
			byGroup[spec.Group] = append(byGroup[spec.Group], loc{doc: doc.Path, body: doc.Body, off: fl.Offset, id: fl.ID})
		}
	}

	// groups(d) is already sorted; every byGroup key is a member of it.
	for _, group := range groups(d) {
		locs, present := byGroup[group]
		if !present {
			continue
		}
		ids := make([]string, len(locs))
		for i, l := range locs {
			ids[i] = l.id
		}
		rendered, violations := RenderGroupFences(d, group, ids)
		for _, v := range violations {
			issues = append(issues, Issue{
				Doc:    locs[0].doc,
				Line:   lineOf(locs[0].body, locs[0].off),
				Kind:   "fence-topology",
				Detail: v,
			})
		}
		for _, l := range locs {
			start, end, ok := FindFence(l.body, l.id)
			if !ok {
				issues = append(issues, Issue{
					Doc:    l.doc,
					Line:   lineOf(l.body, l.off),
					Kind:   "stale-fence",
					Detail: "unterminated GENERATED:" + l.id + " fence",
				})
				continue
			}
			if fresh, rok := rendered[l.id]; rok && l.body[start:end] != fresh {
				issues = append(issues, Issue{
					Doc:    l.doc,
					Line:   lineOf(l.body, l.off),
					Kind:   "stale-fence",
					Detail: "GENERATED:" + l.id + " fence is out of date — run `make gen-cards`",
				})
			}
		}
	}
	return issues
}

// groups returns the sorted, de-duplicated set of command groups in the dump.
func groups(d Dump) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range d.Commands {
		if c.Group != "" && !seen[c.Group] {
			seen[c.Group] = true
			out = append(out, c.Group)
		}
	}
	sort.Strings(out)
	return out
}

// lineOf returns the 1-based line number of byte offset off within body.
func lineOf(body string, off int) int {
	return strings.Count(body[:off], "\n") + 1
}

// commandIndex maps a command path to its set of declared flag names, and
// carries the sorted list of paths for longest-prefix resolution.
type commandIndex struct {
	flags map[string]map[string]bool
	paths []string
	// global is the flags every command accepts (the root's persistent flags
	// and --help); root is every flag the bare root command accepts.
	global map[string]bool
	root   map[string]bool
}

func indexDump(d Dump) commandIndex {
	idx := commandIndex{
		flags:  make(map[string]map[string]bool),
		global: make(map[string]bool),
		root:   make(map[string]bool),
	}
	for _, f := range d.Root {
		idx.root[f.Name] = true
		if f.Persistent {
			idx.global[f.Name] = true
		}
	}
	for _, c := range d.Commands {
		set := make(map[string]bool, len(c.Flags))
		for _, f := range c.Flags {
			set[f.Name] = true
		}
		idx.flags[c.Path] = set
		idx.paths = append(idx.paths, c.Path)
	}
	// Longest paths first so resolveCommand prefers the most specific match.
	sort.Slice(idx.paths, func(i, j int) bool {
		return len(idx.paths[i]) > len(idx.paths[j])
	})
	return idx
}

func validateExample(idx commandIndex, docPath string, ex Example) []Issue {
	path, flagSet, ok := resolveCommand(idx, ex.Tokens)
	if !ok {
		// An unresolved command that is empty (a bare `fduty` prose mention) or
		// templated (a placeholder in the command-path position, e.g.
		// `fduty <group> <verb>`) is a documentation reference, not a runnable
		// example — skip it, mirroring the placeholder tolerance applied to flag
		// values below. A non-empty, non-placeholder path that still doesn't
		// resolve is a genuine wrong command name (e.g. `statuspage`) and is
		// reported.
		words := leadingWords(ex.Tokens)
		if len(words) == 0 {
			// A bare `fduty` mention is prose, unless it carries flags: then it
			// is a root invocation (`fduty --version`), and its flags must be
			// ones the root accepts.
			return unknownFlags(docPath, ex, "fduty", idx.root)
		}
		if anyPlaceholder(words) {
			return nil
		}
		return []Issue{{
			Doc:    docPath,
			Line:   ex.Line,
			Kind:   "unknown-command",
			Detail: strings.Join(words, " "),
		}}
	}

	valid := make(map[string]bool, len(idx.global)+len(flagSet))
	for name := range idx.global {
		valid[name] = true
	}
	for name := range flagSet {
		valid[name] = true
	}
	return unknownFlags(docPath, ex, path, valid)
}

// unknownFlags reports every flag of ex that is not in valid; path names the
// command in the issue.
func unknownFlags(docPath string, ex Example, path string, valid map[string]bool) []Issue {
	var issues []Issue
	for _, tok := range ex.Tokens {
		name, isFlag := flagName(tok)
		if !isFlag || HasPlaceholder(name) {
			continue
		}
		// A field cligen folds into a required positional keeps a same-named flag
		// registered as a genuine alternative source: every generated command
		// with such a fold uses requireBodyFieldOrExactArg/requireBodyFieldOrArgs
		// (internal/cli/args.go), which explicitly accepts the flag alone — cligen
		// never emits the bare requireExactArg/requireArgs form that would make
		// passing the flag fail. So a folded name is just a flag like any other
		// here: fall through to the flagSet check below.
		if valid[name] {
			continue
		}
		issues = append(issues, Issue{
			Doc:    docPath,
			Line:   ex.Line,
			Kind:   "unknown-flag",
			Detail: "--" + name + " not a flag of `" + path + "`",
		})
	}
	return issues
}

// resolveCommand finds the longest dump command path that is a prefix of the
// example's leading non-flag words. Returns the path, its flag set, and whether
// a match was found.
func resolveCommand(idx commandIndex, tokens []string) (string, map[string]bool, bool) {
	words := leadingWords(tokens)
	candidate := strings.Join(words, " ")
	for _, p := range idx.paths {
		if candidate == p || strings.HasPrefix(candidate+" ", p+" ") {
			return p, idx.flags[p], true
		}
	}
	return "", nil, false
}

// leadingWords returns the run of non-flag tokens at the start of an example
// (the command path words, before any --flag).
func leadingWords(tokens []string) []string {
	var words []string
	for _, t := range tokens {
		if strings.HasPrefix(t, "-") {
			break
		}
		words = append(words, t)
	}
	return words
}

// anyPlaceholder reports whether any of the command-path words is a
// documentation placeholder (e.g. <group>), meaning the example is a template
// rather than a concrete invocation.
func anyPlaceholder(words []string) bool {
	for _, w := range words {
		if HasPlaceholder(w) {
			return true
		}
	}
	return false
}

// flagName extracts the bare flag name from a token like "--type" or
// "--type=x", returning ("type", true). Non-flag tokens return ("", false).
func flagName(tok string) (string, bool) {
	if !strings.HasPrefix(tok, "--") {
		return "", false
	}
	name := strings.TrimPrefix(tok, "--")
	if i := strings.IndexByte(name, '='); i >= 0 {
		name = name[:i]
	}
	if name == "" {
		return "", false
	}
	return name, true
}
