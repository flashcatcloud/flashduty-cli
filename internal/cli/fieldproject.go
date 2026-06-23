package cli

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
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
