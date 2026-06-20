package skilldoc

import (
	"regexp"
	"strings"
)

// Example is one harvested CLI invocation from a markdown document. Tokens are
// the whitespace-separated arguments AFTER the `fduty`/`flashduty` binary word
// (so Tokens[0] is the command group). Line is the 1-based line where the
// invocation began.
type Example struct {
	Line   int
	Tokens []string
}

// binaryWords are the recognized CLI invocation prefixes.
var binaryWords = map[string]bool{"fduty": true, "flashduty": true}

// placeholderRe matches an `ou_xxx`-style placeholder: a short lowercase prefix
// then one or more x's (e.g. ou_xxx, ch_xxx).
var placeholderRe = regexp.MustCompile(`^[a-z]{2,}_x+$`)

// HasPlaceholder reports whether tok is a documentation placeholder rather than
// a literal argument: angle-bracket tokens (<id>), shell vars ($VAR), the
// ellipsis (...), or `ou_xxx`-style stand-ins. The validator skips the value
// of any flag whose token is a placeholder.
func HasPlaceholder(tok string) bool {
	switch {
	case strings.ContainsAny(tok, "<>"):
		return true
	case strings.HasPrefix(tok, "$"):
		return true
	case tok == "...":
		return true
	case placeholderRe.MatchString(tok):
		return true
	default:
		return false
	}
}

// HarvestExamples pulls every `fduty`/`flashduty` invocation out of markdown:
// fenced code blocks (```…```) and inline backtick spans alike. A candidate is
// any line whose first shell word is the binary; trailing-backslash
// continuations are joined into one example. Prose lines (no binary word) are
// ignored.
func HarvestExamples(md string) []Example {
	var out []Example
	lines := strings.Split(md, "\n")
	inFence := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if isFenceMarker(line) {
			inFence = !inFence
			continue
		}

		if inFence {
			if cand, ok := fencedCandidate(line); ok {
				joined, next := joinContinuations(cand, lines, i)
				if ex, ok := parseInvocation(joined, i+1); ok {
					out = append(out, ex)
				}
				i = next
			}
			continue
		}

		// Outside fences, scan inline backtick spans on this line.
		for _, span := range inlineSpans(line) {
			if ex, ok := parseInvocation(span, i+1); ok {
				out = append(out, ex)
			}
		}
	}
	return out
}

// isFenceMarker reports whether a line opens or closes a ``` code fence.
func isFenceMarker(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "```")
}

// fencedCandidate returns the trimmed line if its first word is a binary word.
func fencedCandidate(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if firstWordIsBinary(trimmed) {
		return trimmed, true
	}
	return "", false
}

// joinContinuations folds trailing-backslash continuation lines (starting at
// index start) into a single logical command string. It returns the joined
// string and the index of the last consumed line.
func joinContinuations(first string, lines []string, start int) (string, int) {
	parts := []string{strings.TrimSuffix(strings.TrimSpace(first), "\\")}
	idx := start
	for strings.HasSuffix(strings.TrimSpace(lines[idx]), "\\") && idx+1 < len(lines) {
		idx++
		next := strings.TrimSpace(lines[idx])
		// A fence marker terminates the continuation run defensively.
		if isFenceMarker(next) {
			idx--
			break
		}
		parts = append(parts, strings.TrimSuffix(next, "\\"))
	}
	return strings.Join(parts, " "), idx
}

// inlineSpans returns the contents of each `…` inline code span on a line.
func inlineSpans(line string) []string {
	var spans []string
	for {
		open := strings.IndexByte(line, '`')
		if open < 0 {
			break
		}
		rest := line[open+1:]
		close := strings.IndexByte(rest, '`')
		if close < 0 {
			break
		}
		spans = append(spans, rest[:close])
		line = rest[close+1:]
	}
	return spans
}

// parseInvocation tokenizes a command string and, if it starts with a binary
// word, returns the post-binary tokens as an Example.
func parseInvocation(cmd string, line int) (Example, bool) {
	toks := strings.Fields(stripQuotes(cmd))
	if len(toks) == 0 || !binaryWords[toks[0]] {
		return Example{}, false
	}
	return Example{Line: line, Tokens: toks[1:]}, true
}

// firstWordIsBinary reports whether the first whitespace word of s is a binary.
func firstWordIsBinary(s string) bool {
	fields := strings.Fields(s)
	return len(fields) > 0 && binaryWords[fields[0]]
}

// stripQuotes removes ASCII double/single quote characters. For the PoC this is
// enough to keep `--title "x"` from splitting on the embedded space boundary
// incorrectly — flag *names* (the only thing the validator inspects) never
// contain spaces, so dropping quotes around values is safe.
func stripQuotes(s string) string {
	return strings.NewReplacer(`"`, "", `'`, "").Replace(s)
}
