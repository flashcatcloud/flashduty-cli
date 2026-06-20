package skilldoc

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Fence markers. The generator owns ONLY the text between these; intent→verb
// routing, worked examples, and gotchas are hand-written outside the fence.
const (
	fenceStartFmt = "<!-- GENERATED:%s START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->"
	fenceEndFmt   = "<!-- GENERATED:%s END -->"
)

// GenerateFence renders the factual fenced block for one command group: a
// section per leaf verb with its short description and a flag table (name,
// type, required, usage + enum), plus a body-only (--data) note when the
// command has nested JSON-only fields. Required-ness and enums are sourced from
// the authoritative "Request fields:" text in each command's Long; the flag
// list falls back to the dump's Flags when no such block exists (read-only
// verbs). Output is deterministic.
func GenerateFence(d Dump, group string) string {
	cmds := groupCommands(d, group)

	var b strings.Builder
	fmt.Fprintf(&b, fenceStartFmt+"\n\n", group)
	for i, c := range cmds {
		if i > 0 {
			b.WriteString("\n")
		}
		writeCommand(&b, c)
	}
	fmt.Fprintf(&b, "\n"+fenceEndFmt, group)
	return b.String()
}

// FenceStart / FenceEnd return the literal markers for a group, used by the
// freshness check to locate fences in docs.
func FenceStart(group string) string { return fmt.Sprintf(fenceStartFmt, group) }
func FenceEnd(group string) string   { return fmt.Sprintf(fenceEndFmt, group) }

func groupCommands(d Dump, group string) []Command {
	var cmds []Command
	for _, c := range d.Commands {
		if c.Group == group {
			cmds = append(cmds, c)
		}
	}
	sort.Slice(cmds, func(i, j int) bool { return cmds[i].Path < cmds[j].Path })
	return cmds
}

func writeCommand(b *strings.Builder, c Command) {
	verb := verbOf(c.Path)
	fmt.Fprintf(b, "### %s\n", verb)
	if c.Short != "" {
		fmt.Fprintf(b, "%s\n", c.Short)
	}

	// Flag rows as bullets (not a table) so enum pipes render literally without
	// markdown-cell escaping.
	fields := parseRequestFields(c.Long)
	for _, r := range flagRows(c, fields.flags) {
		fmt.Fprintf(b, "- `--%s` %s%s%s\n", r.name, r.typ, reqSuffix(r.required), notesSuffix(r.notes))
	}
	if len(fields.bodyOnly) > 0 {
		fmt.Fprintf(b, "- body-only (`--data`): %s\n", strings.Join(fields.bodyOnly, "; "))
	}
}

// verbOf returns the last space-separated segment of a command path (the leaf
// verb), e.g. "status-page change-create" -> "change-create".
func verbOf(path string) string {
	parts := strings.Fields(path)
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}

// flagRow is one rendered flag row.
type flagRow struct {
	name     string
	typ      string
	required bool
	notes    string
}

// flagRows merges the dump's flag list with the richer Request-fields parse:
// the dump provides the authoritative flag set + type; the parse provides
// required-ness, usage, and enum. Flags appear in the dump's declared order,
// minus globals (data is shown as a body channel, not a flag row).
func flagRows(c Command, parsed map[string]parsedFlag) []flagRow {
	var rows []flagRow
	for _, f := range c.Flags {
		if globalFlags[f.Name] {
			continue
		}
		row := flagRow{name: f.Name, typ: f.Type}
		if pf, ok := parsed[f.Name]; ok {
			row.required = pf.required
			row.notes = withEnum(pf.usage, pf.enum)
		}
		rows = append(rows, row)
	}
	return rows
}

// reqSuffix renders the required marker appended to a flag's type token.
func reqSuffix(required bool) string {
	if required {
		return " (required)"
	}
	return ""
}

// notesSuffix renders the usage/enum description after an em-dash, or empty.
func notesSuffix(notes string) string {
	notes = strings.ReplaceAll(notes, "\n", " ")
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return ""
	}
	return " — " + notes
}

// withEnum appends an enum hint to a usage string.
func withEnum(usage string, enum []string) string {
	if len(enum) == 0 {
		return usage
	}
	hint := "enum: " + strings.Join(enum, " | ")
	if usage == "" {
		return hint
	}
	return usage + " · " + hint
}

// --- Long "Request fields:" parser -----------------------------------------

type parsedFlag struct {
	required bool
	enum     []string
	usage    string
}

type requestFields struct {
	flags    map[string]parsedFlag
	bodyOnly []string // nested --data-only top-level field summaries
}

var (
	flagLineRe  = regexp.MustCompile(`^\s{2}--([a-z0-9-]+)\s+\S+\s*(.*)$`)
	bodyLineRe  = regexp.MustCompile(`^\s{2}([a-z0-9_]+)\s+\(([^,)]*)[^)]*\)\s*(.*)$`)
	enumRe      = regexp.MustCompile(`\[([^\]]+)\]`)
	requiredTag = "(required)"
)

// parseRequestFields extracts the per-flag required/enum/usage and the
// body-only (--data) field summaries from a command's Long "Request fields:"
// block. Returns empty maps when the block is absent (read-only verbs).
func parseRequestFields(long string) requestFields {
	rf := requestFields{flags: map[string]parsedFlag{}}
	lines := strings.Split(long, "\n")
	in := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "Request fields:" {
			in = true
			continue
		}
		if !in {
			continue
		}
		// The block ends at a blank line or the Response fields header.
		if trimmed == "" || strings.HasPrefix(trimmed, "Response fields") {
			break
		}
		if m := flagLineRe.FindStringSubmatch(line); m != nil {
			name, tail := m[1], m[2]
			rf.flags[name] = parsedFlag{
				required: strings.Contains(tail, requiredTag),
				enum:     parseEnum(tail),
				usage:    cleanUsage(tail),
			}
			continue
		}
		// A top-level body-only field (no -- prefix, 2-space indent). Sub-fields
		// are indented deeper and skipped here. The type capture stops at the
		// first comma so "(array<object>, via --data)" yields just "array<object>".
		if m := bodyLineRe.FindStringSubmatch(line); m != nil {
			name, typ, tail := m[1], strings.TrimSpace(m[2]), m[3]
			summary := name + " (" + typ + ")"
			if strings.Contains(tail, requiredTag) {
				summary += " (required)"
			}
			rf.bodyOnly = append(rf.bodyOnly, summary)
		}
	}
	return rf
}

// parseEnum pulls the enum members out of a trailing "[a, b, c]" marker.
func parseEnum(tail string) []string {
	m := enumRe.FindStringSubmatch(tail)
	if m == nil {
		return nil
	}
	parts := strings.Split(m[1], ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// cleanUsage strips the leading em-dash, the (required) tag, and the trailing
// enum bracket from a flag line's tail, leaving the human description.
func cleanUsage(tail string) string {
	s := tail
	s = enumRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, requiredTag, "")
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "—")
	return strings.TrimSpace(s)
}
