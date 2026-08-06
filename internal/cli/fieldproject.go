package cli

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	compactListOutputLimit   = 16 * 1024
	compactDetailOutputLimit = 8 * 1024
)

// projectFields reduces each struct element of items to a map containing only
// the requested fields, matched against the struct's `json` tag (the leading
// component, with any `,omitempty` stripped). It exists so the curated `incident
// list` / `alert list` commands can emit a compact projection in structured
// (json/toon) mode instead of dumping the full nested SDK record — the root
// cause of the oversized list dumps the agent then re-queried with jq.
//
// Only top-level, exported, declared fields are selectable: there are no dotted
// nested paths. The original (typed) field value is preserved in the map so its
// custom MarshalJSON / toon tag behavior (e.g. flashduty.Timestamp) stays
// byte-consistent with the full-dump field. An unknown field name is a fail-fast
// error that lists the valid tag names for the row type.
func projectFields(items any, fields []string) ([]map[string]any, error) {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("internal error: projectFields expects a slice, got %T", items)
	}

	elemType := v.Type().Elem()
	for elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("internal error: projectFields expects a slice of structs, got element %s", elemType.Kind())
	}

	// Map each requested field name to its struct field index. Reject any
	// unknown name up front so a typo fails fast rather than silently emitting
	// an empty projection.
	tagToIndex := jsonTagIndex(elemType)
	indexes := make([]int, 0, len(fields))
	names := make([]string, 0, len(fields))
	for _, f := range fields {
		idx, ok := tagToIndex[f]
		if !ok {
			return nil, fmt.Errorf("unknown field %q; valid fields: %s", f, strings.Join(sortedKeys(tagToIndex), ", "))
		}
		indexes = append(indexes, idx)
		names = append(names, f)
	}

	out := make([]map[string]any, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		for elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		row := make(map[string]any, len(indexes))
		for j, idx := range indexes {
			row[names[j]] = elem.Field(idx).Interface()
		}
		out = append(out, row)
	}
	return out, nil
}

// noteDefaultProjection announces on stderr that structured rows were reduced
// to the command's compact default projection. Without it, a reader piping
// stdout to jq sees an unselected key (labels, description, …) as null on
// every row and can conclude the server never returns it, when it is one
// --fields away. stderr keeps stdout byte-identical for jq/toon pipelines.
func noteDefaultProjection(w io.Writer, fields []string) {
	_, _ = fmt.Fprintf(w, "note: rows projected to default compact fields (%s); other response fields are available via --fields\n",
		strings.Join(fields, ","))
}

// boundProjectedOutput keeps the new agent-oriented projections below their
// command budget without changing the selected keys. Short values remain byte
// identical; when retained strings alone would overflow the actual JSON/TOON
// encoding, they are shortened fairly and marked with "...". If keys and
// non-string values alone exceed the budget, the command fails with a small
// error instead of emitting an oversized payload.
func boundProjectedOutput(data any, maxBytes int) error {
	var rows []map[string]any
	switch value := data.(type) {
	case map[string]any:
		rows = []map[string]any{value}
	case []map[string]any:
		rows = value
	default:
		return fmt.Errorf("internal error: unsupported projected output %T", data)
	}

	encoded, err := marshalStructured(data)
	if err != nil {
		return err
	}
	if len(encoded)+1 < maxBytes {
		return nil
	}

	stringCount := 0
	for _, row := range rows {
		for _, value := range row {
			if _, ok := value.(string); ok {
				stringCount++
			}
		}
	}
	if stringCount == 0 {
		return fmt.Errorf("structured projection exceeds %d-byte limit; request fewer rows or fields", maxBytes)
	}

	fieldLimit := maxBytes / stringCount
	for {
		for _, row := range rows {
			for key, value := range row {
				if text, ok := value.(string); ok {
					row[key] = truncateUTF8Bytes(text, fieldLimit)
				}
			}
		}

		encoded, err = marshalStructured(data)
		if err != nil {
			return err
		}
		if len(encoded)+1 < maxBytes {
			return nil
		}
		if fieldLimit == 0 {
			return fmt.Errorf("structured projection exceeds %d-byte limit; request fewer rows or fields", maxBytes)
		}
		fieldLimit /= 2
	}
}

func truncateUTF8Bytes(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	if maxBytes <= 0 {
		return ""
	}
	if maxBytes <= 3 {
		end := maxBytes
		for end > 0 && !utf8.ValidString(value[:end]) {
			end--
		}
		return value[:end]
	}

	end := maxBytes - 3
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return value[:end] + "..."
}

// jsonTagIndex maps each exported field's json tag name (leading component, sans
// `,omitempty`) to its index in the struct. Fields tagged `json:"-"`, untagged
// fields, and embedded/anonymous fields are skipped — only declared, named,
// tagged top-level fields are selectable.
func jsonTagIndex(t reflect.Type) map[string]int {
	out := make(map[string]int, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous || field.PkgPath != "" { // skip embedded and unexported
			continue
		}
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := tag
		if comma := strings.IndexByte(name, ','); comma >= 0 {
			name = name[:comma]
		}
		if name == "" {
			continue
		}
		out[name] = i
	}
	return out
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
