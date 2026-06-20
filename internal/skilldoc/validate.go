package skilldoc

import (
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
	Kind   string // "unknown-command" | "unknown-flag" | "positional-as-flag" | "stale-fence"
	Detail string
}

// globalFlags are always-valid persistent flags that any command accepts; the
// validator never flags them as unknown. Kept in one place to stay DRY.
var globalFlags = map[string]bool{
	"output-format": true,
	"json":          true,
	"no-trunc":      true,
	"app-key":       true,
	"base-url":      true,
	"data":          true,
}

// Validate checks every harvested `fduty …` example in docs against the dump:
// an example whose leading words resolve to no command path yields an
// unknown-command issue; an example flag absent from its command's flag set
// (and not a global flag) yields an unknown-flag issue. Placeholder tokens are
// skipped so documentation stand-ins (<id>, $VAR) never trip the validator.
func Validate(d Dump, docs []Doc) []Issue {
	idx := indexDump(d)
	var issues []Issue
	for _, doc := range docs {
		for _, ex := range HarvestExamples(doc.Body) {
			issues = append(issues, validateExample(idx, doc.Path, ex)...)
		}
	}
	return issues
}

// CheckFences asserts every GENERATED:<group> fence embedded in docs matches a
// fresh render from the dump. A fence whose inner content has drifted, or a
// start marker with no matching end marker, yields a stale-fence issue. Docs
// with no generated fence for a group are silently fine.
func CheckFences(d Dump, docs []Doc) []Issue {
	var issues []Issue
	for _, group := range groups(d) {
		fresh := GenerateFence(d, group)
		start, end := FenceStart(group), FenceEnd(group)
		for _, doc := range docs {
			si := strings.Index(doc.Body, start)
			if si < 0 {
				continue // no fence for this group in this doc
			}
			ei := strings.Index(doc.Body[si:], end)
			if ei < 0 {
				issues = append(issues, Issue{
					Doc:    doc.Path,
					Line:   lineOf(doc.Body, si),
					Kind:   "stale-fence",
					Detail: "unterminated GENERATED:" + group + " fence",
				})
				continue
			}
			block := doc.Body[si : si+ei+len(end)]
			if block != fresh {
				issues = append(issues, Issue{
					Doc:    doc.Path,
					Line:   lineOf(doc.Body, si),
					Kind:   "stale-fence",
					Detail: "GENERATED:" + group + " fence is out of date — run `make gen-cards`",
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

// commandIndex maps a command path to its set of declared flag names and to the
// set of flags cligen folded into required positionals, and carries the sorted
// list of paths for longest-prefix resolution.
type commandIndex struct {
	flags  map[string]map[string]bool
	folded map[string]map[string]bool
	paths  []string
}

func indexDump(d Dump) commandIndex {
	idx := commandIndex{
		flags:  make(map[string]map[string]bool),
		folded: make(map[string]map[string]bool),
	}
	for _, c := range d.Commands {
		set := make(map[string]bool, len(c.Flags))
		for _, f := range c.Flags {
			set[f.Name] = true
		}
		idx.flags[c.Path] = set
		idx.folded[c.Path] = foldedFlagNames(positionalsOf(c.Use))
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
		if len(words) == 0 || anyPlaceholder(words) {
			return nil
		}
		return []Issue{{
			Doc:    docPath,
			Line:   ex.Line,
			Kind:   "unknown-command",
			Detail: strings.Join(words, " "),
		}}
	}

	folded := idx.folded[path]
	var issues []Issue
	for _, tok := range ex.Tokens {
		name, isFlag := flagName(tok)
		if !isFlag || HasPlaceholder(name) {
			continue
		}
		// cligen folded this field into a required positional: the flag is still
		// registered (so it is in flagSet) but passing it as a flag fails the
		// binary's Args check. Catch it before the flagSet pass would wave it
		// through — this is the exact misuse only a live run surfaced before.
		if folded[name] {
			issues = append(issues, Issue{
				Doc:    docPath,
				Line:   ex.Line,
				Kind:   "positional-as-flag",
				Detail: "--" + name + " is folded into a required positional of `" + path + "` — pass it as a bare argument, not a flag",
			})
			continue
		}
		if globalFlags[name] || flagSet[name] {
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
