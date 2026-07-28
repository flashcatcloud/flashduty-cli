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
// command has nested JSON-only fields, plus a one-line response-shape summary
// (top-level object vs. bare array vs. `{items: [...]}` page wrapper, and the
// field names at that level) when the command documents one. Required-ness
// and enums are sourced from the authoritative "Request fields:" text in each
// command's Long; the response shape is likewise sourced from that same
// Long's "Response fields (...):" block (cligen's own ground truth — see
// responseShapeLine), not re-derived or hand-curated. The flag list falls
// back to the dump's Flags when no Request-fields block exists (read-only
// verbs). Output is deterministic.
func GenerateFence(d Dump, group string) string {
	cmds := groupCommands(d, group)

	var b strings.Builder
	fmt.Fprintf(&b, fenceStartFmt+"\n\n", group)
	// seenShapes maps a verbatim response-shape line to the name of the first
	// command in THIS group (only — never across cards) that rendered it in
	// full, so a later command with a byte-identical shape can point back at
	// it instead of repeating the field list. Scoped to one GenerateFence call,
	// so cards stay self-contained.
	seenShapes := map[string]string{}
	for i, c := range cmds {
		if i > 0 {
			b.WriteString("\n")
		}
		writeCommand(&b, c, seenShapes)
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

func writeCommand(b *strings.Builder, c Command, seenShapes map[string]string) {
	name := commandName(c)
	positionals := positionalsOf(c.Use)

	// Heading carries the positional signature verbatim from Use (authoritative),
	// e.g. "change-active-list <page-id>", so the reader sees the exact argument
	// order the binary requires.
	fmt.Fprintf(b, "### %s\n", name)
	if c.Short != "" {
		fmt.Fprintf(b, "%s\n", c.Short)
	}

	// Flag rows as bullets (not a table) so enum pipes render literally without
	// markdown-cell escaping. A field cligen folded into a required positional is
	// rendered as a positional row, NOT a --flag — passing it as a flag without
	// the positional fails the binary's Args check.
	fields := parseRequestFields(c.Long)
	folded := foldedFlagNames(positionals)
	for _, r := range flagRows(c, fields.flags) {
		if folded[r.name] {
			fmt.Fprintf(b, "- `<%s>` (positional, required) %s%s\n", r.name, r.typ, notesSuffix(r.notes))
			continue
		}
		fmt.Fprintf(b, "- `--%s` %s%s%s\n", r.name, r.typ, reqSuffix(r.required), notesSuffix(r.notes))
	}
	if len(fields.bodyOnly) > 0 {
		fmt.Fprintf(b, "- body-only (`--data`): %s\n", strings.Join(fields.bodyOnly, "; "))
	}
	if shape := responseShapeLine(c.Long); shape != "" {
		if first, ok := seenShapes[shape]; ok {
			// A later command in the same group whose response is byte-identical
			// to one already rendered in full — point back at it instead of
			// repeating the field list. Several commands in one group commonly
			// return the same resource (create/get/update/…), so this is the
			// common case, not the exception.
			fmt.Fprintf(b, "- response: same shape as `%s` above\n", first)
		} else {
			seenShapes[shape] = name
			b.WriteString(shape)
		}
	}
}

// commandName returns the heading text for a command: the leaf verb plus its
// positional signature verbatim from Use, e.g. "change-active-list <page-id>"
// or, with no positional, just the verb. Also used as the referent name in a
// deduplicated response-shape line ("same shape as `<commandName>` above"), so
// the reference always names exactly what the reader sees in that other
// section's own heading.
func commandName(c Command) string {
	verb := verbOf(c.Path)
	if positionals := positionalsOf(c.Use); len(positionals) > 0 {
		return verb + " " + strings.Join(positionals, " ")
	}
	return verb
}

// positionalsOf returns the placeholder tokens after the leaf verb in a Use
// string, e.g. "change-active-list <page-id>" -> ["<page-id>"] and
// "merge <incident-id> [<id2>...]" -> ["<incident-id>", "[<id2>...]"]. A Use with
// no positional ("list") returns nil.
func positionalsOf(use string) []string {
	f := strings.Fields(use)
	if len(f) <= 1 {
		return nil
	}
	return f[1:]
}

// foldedFlagNames returns the EXACT flag names that cligen has folded into a
// REQUIRED positional argument (a "<name>" placeholder). The binary still
// registers a same-named flag, but supplying it as a flag fails the positional
// Args check, so these names render as positionals (in writeCommand) and are
// rejected as flags (in the validator).
//
// A scalar positional "<page-id>" folds the exact flag "page-id". An array
// positional appears as "<incident-id> [<id2>...]" — cligen singularizes the
// "*-ids" wire name for the placeholder — so its folded flag is the plural wire,
// recovered as inner+"s": "<incident-id>" folds "incident-ids". Matching the
// exact name (not a trailing-"s"-stripped key) keeps an unrelated plural flag
// like "--types" from colliding with a scalar "<type>" positional.
func foldedFlagNames(positionals []string) map[string]bool {
	out := map[string]bool{}
	for i, p := range positionals {
		if !strings.HasPrefix(p, "<") {
			continue // optional [<...>] or variadic [<id2>...] — flag (if any) stays
		}
		inner := placeholderInner(p)
		if i+1 < len(positionals) && strings.HasPrefix(positionals[i+1], "[") {
			out[inner+"s"] = true // array positional: the plural "*-ids" wire flag
		} else {
			out[inner] = true // scalar positional: the exact flag name
		}
	}
	return out
}

// placeholderInner strips the surrounding <> (and a trailing "...") from a
// REQUIRED Use placeholder, e.g. "<page-id>" -> "page-id". Only "<...>" tokens
// reach this helper (foldedFlagNames guards on the "<" prefix), so optional
// "[<...>]" brackets never appear here.
func placeholderInner(p string) string {
	p = strings.TrimPrefix(p, "<")
	p = strings.TrimSuffix(p, "...")
	p = strings.TrimSuffix(p, ">")
	return p
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

// --- Long "Response fields:" parser -----------------------------------------
//
// cligen classifies every documented response into exactly one of three
// envelope shapes and says so verbatim in the "Response fields (...):" header
// it writes into Long — this parser only ever recognizes those three; it does
// not infer a shape of its own. That header (and the field list under it) is
// authoritative today but surfaces only via `--help`, which an agent that
// reads just the card fence never invokes. responseShapeLine folds a
// one-line summary of it into the fence so every generated verb — not only
// the handful some earlier hand-written card happened to cover — tells the
// agent up front whether `--json` is a bare array, a single object, or a
// `{items: [...]}` page wrapper, and exactly which field names exist at that
// level. Guessing a field name silently returns null instead of an error, so
// this is the difference between an agent noticing its own mistake and not.

// responseHeaderRe matches the header line, capturing the parenthetical shape
// description cligen wrote (verbatim, no leading indent — it starts a new
// paragraph in Long).
var responseHeaderRe = regexp.MustCompile(`^Response fields \((.*)\):$`)

// responseFieldRe matches one Response-fields bullet row at any indent depth,
// e.g. "  - account_id (integer) (required) — ..." or, one level deeper,
// "    - person_ids (array<integer>) ...". Capture groups: indent, name, type.
var responseFieldRe = regexp.MustCompile(`^( *)- ([a-zA-Z0-9_]+) \(([^)]*)\)`)

// wrapperWireNames are the exact wire names cligen's own listEnvelope
// (internal/cmd/cligen/main.go) treats as a paginated-list envelope field:
// a sole array-typed sibling named items, docs, or list. Mirrored here as a
// sanity check, not a duplicate classifier — see the guard in
// responseShapeLine below.
var wrapperWireNames = map[string]bool{"items": true, "docs": true, "list": true}

// respField is one parsed Response-fields bullet row.
type respField struct{ name, typ string }

// responseShapeLine renders the one-line response-shape summary for a
// command's Long, or "" when Long documents no Response fields block (mutation
// verbs with an empty body, and a few hand-written commands that predate
// cligen). The three shapes cligen's header can name:
//
//   - top-level object: the block's own fields are the response.
//   - top-level array: `--json` is a bare array of these row objects — pipe
//     `jq '.[]'`, never `.items[]`.
//   - `{items: [...]}` page wrapper: the block's top-level array field is
//     `items` (row fields nested one level deeper under it), alongside
//     scalar pagination siblings — `total`, `has_next_page`,
//     `search_after_ctx`, … (cligen's listEnvelope requires every
//     non-`items` top-level field to be a scalar, so these are always
//     pagination metadata, never more row data).
//
// Field names (with their documented type) are read from whichever indent
// depth holds the actual row/object fields for the detected shape, so the
// summary always names the fields an agent would pipe `jq` at — not the
// wrapper key. For the wrapper shape, the pagination siblings are named too
// (by name only — their type is implied by cligen's scalar-sibling
// invariant), since an agent told to paginate via `--search-after-ctx` still
// needs to know the returned cursor lives at `.search_after_ctx` next to
// `.items`, not buried in a field list it never sees.
func responseShapeLine(long string) string {
	lines := strings.Split(long, "\n")
	headerIdx, header := -1, ""
	for i, line := range lines {
		if m := responseHeaderRe.FindStringSubmatch(line); m != nil {
			headerIdx, header = i, m[1]
			break
		}
	}
	if headerIdx < 0 {
		return ""
	}

	wrapped := strings.Contains(header, "nested under items[]")
	fieldIndent := "  "
	if wrapped {
		fieldIndent = "    " // one level under the top-level "items" row
	}

	// fields holds the row-level fields the summary's "fields:" list names
	// (top-level for object/array shapes, one level under "items" for the
	// wrapper shape). siblings holds the wrapper shape's OTHER top-level
	// fields — pagination metadata alongside "items" (total, has_next_page,
	// search_after_ctx, …) — collected only when wrapped, since the
	// unwrapped shapes have no such siblings to speak of.
	var fields, siblings []respField
	for _, line := range lines[headerIdx+1:] {
		if strings.TrimSpace(line) == "" {
			break
		}
		m := responseFieldRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		indent, name, typ := m[1], m[2], m[3]
		switch {
		case indent == fieldIndent:
			fields = append(fields, respField{name, typ})
		case wrapped && indent == "  " && !wrapperWireNames[name]:
			siblings = append(siblings, respField{name, typ})
		}
	}
	if len(fields) == 0 {
		return ""
	}

	// Safety net: this parser only ever recognizes the wrapped shape by the
	// literal substring "nested under items[]" in the header (see the doc
	// comment above the const block). If cligen's wording for that header
	// ever drifts without this parser being updated to match, `wrapped` goes
	// false here even though the response really is a page wrapper — and the
	// sole top-level field is then exactly one of cligen's own wrapper wire
	// names (items/docs/list, from listEnvelope in cligen/main.go), holding
	// an array. Asserting "single object" in that case would be confidently
	// WRONG about the one thing this whole feature exists to get right, so
	// refuse to guess: say nothing rather than assert a shape we can no
	// longer be sure of. A missing line is recoverable via `--help`; a
	// wrong one isn't.
	if !wrapped && len(fields) == 1 && wrapperWireNames[fields[0].name] && strings.HasPrefix(fields[0].typ, "array") {
		return ""
	}

	var shape, fieldsLabel string
	switch {
	case wrapped:
		shape = fmt.Sprintf("`{items: [...]%s}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`)", siblingSuffix(siblings))
		fieldsLabel = "items fields" // disambiguates row fields from the wrapper's own pagination siblings named above
	case strings.Contains(header, "TOP-LEVEL array"):
		shape = "TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`)"
		fieldsLabel = "fields"
	default:
		shape = "single object (`data` unwrapped to the top level)"
		fieldsLabel = "fields"
	}
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.name + " (" + f.typ + ")"
	}
	return fmt.Sprintf("- response: %s — %s: %s\n", shape, fieldsLabel, strings.Join(names, "; "))
}

// siblingSuffix renders a wrapped response's top-level pagination-metadata
// siblings — e.g. ", total, has_next_page, search_after_ctx" — appended
// inside the `{items: [...]}` wrapper descriptor, in the order cligen
// documented them. Empty when the envelope carries no siblings beyond the
// row array itself.
func siblingSuffix(siblings []respField) string {
	if len(siblings) == 0 {
		return ""
	}
	names := make([]string, len(siblings))
	for i, s := range siblings {
		names[i] = s.name
	}
	return ", " + strings.Join(names, ", ")
}
