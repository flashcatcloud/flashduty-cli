package cli

import (
	"fmt"
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

// boundProjectedOutput keeps the new agent-oriented projections below their
// command budget without changing the selected keys. List rows (many small
// records) are shortened fairly when they overflow the budget, with
// shortened values marked with "...". A single-object detail projection is
// never modified: a truncated id or status string is indistinguishable from
// a genuinely short value, so silently shortening it would hand the caller
// wrong data instead of a compact one. If a detail projection doesn't fit,
// the command fails with an error instead.
func boundProjectedOutput(data any, maxBytes int) error {
	switch value := data.(type) {
	case map[string]any:
		return boundProjectedDetail(value, maxBytes)
	case []map[string]any:
		return boundProjectedList(value, maxBytes)
	default:
		return fmt.Errorf("internal error: unsupported projected output %T", data)
	}
}

// boundProjectedDetail rejects an oversized single-object projection instead
// of truncating it, naming the largest fields so the caller can fix the
// request in one pass: drop some of them from --fields, or drop --fields
// entirely for the full, unbounded detail.
func boundProjectedDetail(row map[string]any, maxBytes int) error {
	encoded, err := marshalStructured(row)
	if err != nil {
		return err
	}
	if len(encoded)+1 < maxBytes {
		return nil
	}

	type fieldSize struct {
		name string
		size int
	}
	sizes := make([]fieldSize, 0, len(row))
	for key, value := range row {
		fieldEncoded, err := marshalStructured(map[string]any{key: value})
		if err != nil {
			return err
		}
		sizes = append(sizes, fieldSize{key, len(fieldEncoded)})
	}
	// Ties break on name so the same oversized request always names the same
	// fields, despite Go's randomized map iteration order.
	sort.Slice(sizes, func(i, j int) bool {
		if sizes[i].size != sizes[j].size {
			return sizes[i].size > sizes[j].size
		}
		return sizes[i].name < sizes[j].name
	})
	if len(sizes) > 3 {
		sizes = sizes[:3]
	}
	largest := make([]string, len(sizes))
	for i, f := range sizes {
		largest[i] = fmt.Sprintf("%s (%d bytes)", f.name, f.size)
	}
	return fmt.Errorf("projected detail is %d bytes, exceeds the %d-byte limit; largest fields: %s; request fewer --fields, or omit --fields for the full, unbounded detail",
		len(encoded), maxBytes, strings.Join(largest, ", "))
}

// boundProjectedList shortens a list projection's string values fairly
// (across all rows) when the compact rows themselves overflow the budget,
// marking shortened values with "...". If keys and non-string values alone
// exceed the budget, the command fails with a small error instead of
// emitting an oversized payload.
func boundProjectedList(rows []map[string]any, maxBytes int) error {
	encoded, err := marshalStructured(rows)
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

		encoded, err = marshalStructured(rows)
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
