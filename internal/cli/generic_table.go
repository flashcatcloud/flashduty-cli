package cli

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

// maxHeuristicColumns bounds the auto-derived column count for response row
// types that have no displayColumns entry, so a wide struct (incident rows carry
// ~50 fields) doesn't print an unreadably wide table.
const maxHeuristicColumns = 8

// genericStringMaxWidth caps free-text columns (titles, names) so one long value
// can't blow out the table width.
const genericStringMaxWidth = 40

// instantLike mirrors go-flashduty's Timestamp/TimestampMilli (and the output
// package's unexported instant) so the renderer can recognise timestamp fields
// by reflection.
type instantLike interface {
	Time() time.Time
	IsZero() bool
}

var instantLikeType = reflect.TypeOf((*instantLike)(nil)).Elem()

// genKV is one row of the vertical key/value table used for a single (non-list)
// object response.
type genKV struct {
	Field string
	Value string
}

// renderGenericTable renders a generated command's typed response for human
// (table) output. Generated commands carry no hand-written column set, so we
// derive one by reflection:
//   - a paginated list envelope ({Items:[...], Total, ...}) or a top-level row
//     array prints as an aligned table (columns from displayColumns, else a
//     reflective heuristic);
//   - a single object prints as a vertical key/value table;
//   - anything we can't model falls back to indented JSON, so output is never
//     empty.
func renderGenericTable(ctx *RunContext, data any) error {
	v := reflect.ValueOf(data)
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return jsonFallback(ctx, data)
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice:
		return renderRowTable(ctx, v, v.Len())
	case reflect.Struct:
		if rows, total, ok := listEnvelope(v); ok {
			return renderRowTable(ctx, rows, total)
		}
		return renderVertical(ctx, v)
	default:
		return jsonFallback(ctx, data)
	}
}

// listEnvelope reports whether struct v is a paginated list envelope: exactly
// one exported field that is a slice of structs (the rows), with the remaining
// fields being pagination metadata. It returns the rows value and the total
// (the int field named "Total" when present, else the row count).
func listEnvelope(v reflect.Value) (rows reflect.Value, total int, ok bool) {
	t := v.Type()
	total = -1
	found := false
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		fv := v.Field(i)
		if isRowSlice(fv.Type()) {
			if found {
				return reflect.Value{}, 0, false // ambiguous: more than one row slice
			}
			rows, found = fv, true
			continue
		}
		if total < 0 && f.Name == "Total" && fv.CanInt() {
			total = int(fv.Int())
		}
	}
	if !found {
		return reflect.Value{}, 0, false
	}
	if total < 0 {
		total = rows.Len()
	}
	return rows, total, true
}

// isRowSlice reports whether t is a slice whose element (after pointer deref) is
// a struct — i.e. a table-able row collection.
func isRowSlice(t reflect.Type) bool {
	if t.Kind() != reflect.Slice {
		return false
	}
	elem := t.Elem()
	for elem.Kind() == reflect.Pointer {
		elem = elem.Elem()
	}
	return elem.Kind() == reflect.Struct
}

func renderRowTable(ctx *RunContext, rows reflect.Value, total int) error {
	if rows.Len() == 0 {
		_, _ = fmt.Fprintln(ctx.Writer, "No results.")
		return nil
	}
	rowType := rows.Type().Elem()
	for rowType.Kind() == reflect.Pointer {
		rowType = rowType.Elem()
	}
	cols := columnsForType(rowType)
	if len(cols) == 0 {
		return jsonFallback(ctx, rows.Interface())
	}
	if err := ctx.Printer.Print(rows.Interface(), cols); err != nil {
		return err
	}
	// Reached only from the table (human) path — printGenericResult handles
	// structured output before calling the renderer — so the footer always prints.
	_, _ = fmt.Fprintf(ctx.Writer, "Total: %d\n", total)
	return nil
}

// columnsForType returns the display columns for a row type: the curated set
// from displayColumns when present, else a reflective heuristic.
func columnsForType(rowType reflect.Type) []output.Column {
	if specs, ok := displayColumns[rowType.Name()]; ok {
		cols := make([]output.Column, 0, len(specs))
		for _, s := range specs {
			cols = append(cols, output.Column{
				Header:   s.Header,
				MaxWidth: s.MaxWidth,
				Field:    func(item any) string { return fieldString(item, s.Field) },
			})
		}
		return cols
	}
	return heuristicColumns(rowType)
}

// heuristicColumns derives columns for a row type with no displayColumns entry:
// the first maxHeuristicColumns scalar (or timestamp) exported fields, in
// declaration order. Nested objects and arrays are skipped — they belong in
// json/toon output, not a human table.
func heuristicColumns(rowType reflect.Type) []output.Column {
	cols := make([]output.Column, 0, maxHeuristicColumns)
	for i := 0; i < rowType.NumField() && len(cols) < maxHeuristicColumns; i++ {
		f := rowType.Field(i)
		if f.PkgPath != "" || !isScalarType(f.Type) {
			continue
		}
		maxW := 0
		if f.Type.Kind() == reflect.String {
			maxW = genericStringMaxWidth
		}
		cols = append(cols, output.Column{
			Header:   headerFromField(f),
			MaxWidth: maxW,
			Field:    func(item any) string { return fieldString(item, f.Name) },
		})
	}
	return cols
}

// renderVertical prints a single object as a two-column FIELD/VALUE table,
// showing scalar fields with a non-empty value. Nested objects/arrays are
// omitted (json/toon carries the full shape for machines).
func renderVertical(ctx *RunContext, v reflect.Value) error {
	t := v.Type()
	rows := make([]genKV, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" || !isScalarType(f.Type) {
			continue
		}
		s := scalarString(v.Field(i))
		if s == "" || s == "-" {
			continue
		}
		rows = append(rows, genKV{Field: headerFromField(f), Value: s})
	}
	if len(rows) == 0 {
		return jsonFallback(ctx, v.Interface())
	}
	cols := []output.Column{
		{Header: "FIELD", Field: func(item any) string { return item.(genKV).Field }},
		{Header: "VALUE", MaxWidth: 80, Field: func(item any) string { return item.(genKV).Value }},
	}
	return ctx.Printer.Print(rows, cols)
}

// fieldString reads the named Go field from a row item (deref'ing a pointer row)
// and formats it. An absent field yields "" rather than panicking, so a stale
// displayColumns entry degrades a column instead of crashing the command.
func fieldString(item any, goField string) string {
	rv := reflect.ValueOf(item)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	fv := rv.FieldByName(goField)
	if !fv.IsValid() {
		return ""
	}
	return scalarString(fv)
}

// scalarString formats a scalar (or timestamp) reflect value. Non-scalars yield
// "" — the generic table never renders nested objects/arrays.
func scalarString(fv reflect.Value) string {
	for fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			return ""
		}
		fv = fv.Elem()
	}
	if fv.CanInterface() {
		if s, ok := output.FormatTimeValue(fv.Interface()); ok {
			return s
		}
	}
	switch fv.Kind() {
	case reflect.String:
		return fv.String()
	case reflect.Bool:
		return strconv.FormatBool(fv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(fv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(fv.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(fv.Float(), 'f', -1, 64)
	default:
		return ""
	}
}

// isScalarType reports whether t is renderable as a single table cell: a string,
// number, bool, or a timestamp (instant) type.
func isScalarType(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	// Timestamp/TimestampMilli satisfy instantLike with value receivers, so the
	// deref'd (non-pointer) type implements it directly.
	if t.Implements(instantLikeType) {
		return true
	}
	switch t.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// headerFromField derives a column header from a field's json tag (upper-cased),
// falling back to the Go field name.
func headerFromField(f reflect.StructField) string {
	if tag := f.Tag.Get("json"); tag != "" {
		if c := strings.IndexByte(tag, ','); c >= 0 {
			tag = tag[:c]
		}
		if tag != "" && tag != "-" {
			return strings.ToUpper(tag)
		}
	}
	return strings.ToUpper(f.Name)
}

// jsonFallback prints indented JSON — the last resort when a response is neither
// a list nor a renderable object. Preserves the pre-renderer behavior.
func jsonFallback(ctx *RunContext, data any) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}
	_, err = fmt.Fprintln(ctx.Writer, string(out))
	return err
}
