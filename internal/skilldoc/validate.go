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
	Kind   string // "unknown-command" | "unknown-flag" | "stale-fence"
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

// commandIndex maps a command path to its set of declared flag names, and
// carries the sorted list of paths for longest-prefix resolution.
type commandIndex struct {
	flags map[string]map[string]bool
	paths []string
}

func indexDump(d Dump) commandIndex {
	idx := commandIndex{flags: make(map[string]map[string]bool)}
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
		return []Issue{{
			Doc:    docPath,
			Line:   ex.Line,
			Kind:   "unknown-command",
			Detail: commandWords(ex.Tokens),
		}}
	}

	var issues []Issue
	for _, tok := range ex.Tokens {
		name, isFlag := flagName(tok)
		if !isFlag || HasPlaceholder(name) {
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

// commandWords joins the leading command words for issue detail text.
func commandWords(tokens []string) string {
	return strings.Join(leadingWords(tokens), " ")
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
