package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

// The global --fields flag projects structured (json/toon) output to the named
// top-level fields, on every command, at the one place structured records are
// printed (RunContext.Printer). A command that declares its own --fields
// (incident list, alert-event list, … — those carry default projections and
// size bounds wired to it, or give the flag another meaning entirely, like
// monit rule-update-fields) shadows the global flag, so flagFields stays empty
// for it and this projection never runs on top of the command's own.

// projection is the result of projectOutput: the projected value, the
// requested fields that matched no key, and every key the unprojected rows
// carried (for the note that names what does exist).
type projection struct {
	out     any
	missing []string
	keys    []string
}

// projectOutput reduces data to fields: each row of a top-level array, each
// row of a list envelope (rows under items/docs/list next to scalar paging
// keys, which are kept — see listEnvelopeKey), or the keys of a single record.
// A field some rows carry is emitted on every row (null where absent, so rows
// stay uniform); a field no row carries is left out and reported in missing —
// an omitempty key can legitimately be absent from a whole page, so that is
// not an error. Output of any other shape cannot be projected and is an error.
//
// It works on the decoded JSON form rather than on SDK structs (as
// projectFields does for the curated verbs that declare --fields), because it
// serves every command, generated ones included, whose results reach it as
// arbitrary SDK types.
func projectOutput(cmd *cobra.Command, data any, fields []string) (projection, error) {
	generic, err := genericStructured(data)
	if err != nil {
		return projection{}, err
	}
	switch v := generic.(type) {
	case []any:
		rows, ok := objectRows(v)
		if !ok {
			return projection{}, fieldsShapeError(cmd, "a list of non-object values")
		}
		return projectRows(rows, fields), nil
	case map[string]any:
		if key, ok := listEnvelopeKey(v); ok {
			if rows, ok := objectRows(v[key].([]any)); ok {
				p := projectRows(rows, fields)
				v[key] = p.out
				p.out = v
				return p, nil
			}
		}
		p := projectRows([]map[string]any{v}, fields)
		p.out = p.out.([]any)[0]
		return p, nil
	case nil:
		return projection{}, nil
	default:
		return projection{}, fieldsShapeError(cmd, fmt.Sprintf("a single %T value", v))
	}
}

func projectRows(rows []map[string]any, fields []string) projection {
	seen := map[string]bool{}
	for _, row := range rows {
		for k := range row {
			seen[k] = true
		}
	}
	var p projection
	for k := range seen {
		p.keys = append(p.keys, k)
	}
	sort.Strings(p.keys)
	var kept []string
	for _, f := range fields {
		switch {
		case seen[f]:
			kept = append(kept, f)
		case len(rows) > 0: // an empty page proves nothing about a name
			p.missing = append(p.missing, f)
		}
	}
	out := make([]any, len(rows))
	for i, row := range rows {
		r := make(map[string]any, len(kept))
		for _, f := range kept {
			r[f] = row[f]
		}
		out[i] = r
	}
	p.out = out
	return p
}

func fieldsShapeError(cmd *cobra.Command, shape string) error {
	return fmt.Errorf("--fields keeps named fields of an object or of each row in a list, but %q printed %s; drop --fields", cmd.CommandPath(), shape)
}

// noteFieldsMissing names the requested fields that matched no key, with the
// keys the output does carry, so a typo is told apart from an absent value.
func noteFieldsMissing(cmd *cobra.Command, p projection) {
	if len(p.missing) > 0 {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "note: --fields %s matched no key in the output (keys present: %s)\n",
			strings.Join(p.missing, ","), strings.Join(p.keys, ", "))
	}
}

// fieldsApplied records that the global --fields projected something during
// the current run; the root resets it before each command runs.
var fieldsApplied bool

// noteFieldsNotApplied runs after every command (root PersistentPostRun) and
// says so when --fields was given but projected nothing: table output, an
// acknowledgement, a raw (file/CSV) body, or a command that prints its own
// text (version, config show). One hook covers every output path, so no
// command can ignore the flag silently.
func noteFieldsNotApplied(cmd *cobra.Command) {
	if flagFields != "" && !fieldsApplied {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "note: --fields does not apply to the output of %q; printed unchanged\n", cmd.CommandPath())
	}
}

// requestedFields is the global --fields list when it applies: set, and the
// output is structured.
func requestedFields() []string {
	if !currentOutputFormat().Structured() {
		return nil
	}
	return parseStringSlice(flagFields)
}

// projectingPrinter applies the global --fields to every structured record a
// command prints.
type projectingPrinter struct {
	inner output.Printer
	cmd   *cobra.Command
}

func (p projectingPrinter) Print(data any, columns []output.Column) error {
	fields := requestedFields()
	if len(fields) == 0 {
		return p.inner.Print(data, columns)
	}
	proj, err := projectOutput(p.cmd, data, fields)
	if err != nil {
		return err
	}
	noteFieldsMissing(p.cmd, proj)
	fieldsApplied = true
	return p.inner.Print(proj.out, nil)
}
