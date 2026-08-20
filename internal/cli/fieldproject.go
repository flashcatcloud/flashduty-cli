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

// noteProjectionShortening tells the caller, on stderr, that some values came
// back clipped. Without it a shortened value is only visible to a reader, not
// to the jq filter or exact match a --json consumer runs over it, so a query
// that silently matches nothing looks like an empty result rather than a
// truncated one.
func noteProjectionShortening(w io.Writer, note string) {
	if note == "" {
		return
	}
	_, _ = fmt.Fprintln(w, note)
}

// boundProjectedOutput keeps the new agent-oriented projections below their
// command budget without changing the selected keys. List rows (many small
// records) are shortened fairly when they overflow the budget, with
// shortened values marked with "...". A single-object detail projection is
// never modified: a truncated id or status string is indistinguishable from
// a genuinely short value, so silently shortening it would hand the caller
// wrong data instead of a compact one. If a detail projection doesn't fit,
// the command fails with an error instead.
//
// It returns a caller-printable note (empty when nothing was shortened) that
// names the clipped fields, so the caller can announce the loss on stderr —
// the "..." marker is only visible to something that reads the value, never
// to the filter a --json consumer runs over it.
func boundProjectedOutput(data any, maxBytes int) (string, error) {
	switch value := data.(type) {
	case map[string]any:
		return "", boundProjectedDetail(value, maxBytes)
	case []map[string]any:
		return boundProjectedList(value, maxBytes)
	default:
		return "", fmt.Errorf("internal error: unsupported projected output %T", data)
	}
}

// largestProjectedFields names the up to three fields carrying the most bytes
// in a projection, so an over-budget request can be narrowed in one pass
// instead of one re-run per field. Sizes are summed per field across every
// row, which is what makes it meaningful for a list: the field responsible
// for the overflow is the one that is big in aggregate, not in any one row.
// Ties break on name so the same oversized request always names the same
// fields, despite Go's randomized map iteration order.
func largestProjectedFields(rows []map[string]any) (string, error) {
	totals := map[string]int{}
	for _, row := range rows {
		for key, value := range row {
			encoded, err := marshalStructured(map[string]any{key: value})
			if err != nil {
				return "", err
			}
			totals[key] += len(encoded)
		}
	}

	type fieldSize struct {
		name string
		size int
	}
	sizes := make([]fieldSize, 0, len(totals))
	for name, size := range totals {
		sizes = append(sizes, fieldSize{name, size})
	}
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
	return strings.Join(largest, ", "), nil
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

	largest, err := largestProjectedFields([]map[string]any{row})
	if err != nil {
		return err
	}
	return fmt.Errorf("projected detail is %d bytes, exceeds the %d-byte limit; largest fields: %s; request fewer --fields, or omit --fields for the full, unbounded detail",
		len(encoded), maxBytes, largest)
}

// boundProjectedList shortens a list projection's string values fairly when
// the compact rows themselves overflow the budget: it finds the largest
// per-field byte cap that still makes everything fit, then applies that one
// cap to every string value across every row. A field already shorter than
// the cap is left completely untouched — only the field(s) actually
// responsible for the overflow (typically a long title) get shortened, each
// marked with "...". The cap never drops low enough to make the "..."
// marker itself disappear, so a shortened value is always distinguishable
// from a genuinely short one; if no cap at or above that floor fits, the
// command fails with a small error instead of emitting values that look
// real but aren't. Whatever it clips, it reports back in the returned note.
func boundProjectedList(rows []map[string]any, maxBytes int) (string, error) {
	encoded, err := marshalStructured(rows)
	if err != nil {
		return "", err
	}
	if len(encoded)+1 < maxBytes {
		return "", nil
	}

	// The overflow error names the fields responsible, exactly as the detail
	// path does, so the request can be narrowed in one pass.
	tooBig := func() (string, error) {
		largest, err := largestProjectedFields(rows)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("projected list is %d bytes across %d rows, exceeds the %d-byte limit; largest fields: %s; request fewer rows (--limit) or fewer --fields",
			len(encoded), len(rows), maxBytes, largest)
	}

	maxLen := 0
	for _, row := range rows {
		for _, value := range row {
			if text, ok := value.(string); ok && len(text) > maxLen {
				maxLen = len(text)
			}
		}
	}
	if maxLen == 0 {
		return tooBig()
	}

	fits := func(limit int) (bool, error) {
		trial := make([]map[string]any, len(rows))
		for i, row := range rows {
			trialRow := make(map[string]any, len(row))
			for key, value := range row {
				if text, ok := value.(string); ok {
					trialRow[key] = truncateUTF8Bytes(text, limit)
				} else {
					trialRow[key] = value
				}
			}
			trial[i] = trialRow
		}
		trialEncoded, err := marshalStructured(trial)
		if err != nil {
			return false, err
		}
		return len(trialEncoded)+1 < maxBytes, nil
	}

	// minMarkedTruncationCap is the smallest cap for which truncateUTF8Bytes
	// still appends "..." (it needs 3 bytes of headroom beyond the marker
	// itself); below it a truncated value would be indistinguishable from a
	// genuinely short one, which is the defect this function must not
	// reintroduce.
	const minMarkedTruncationCap = 4
	if maxLen <= minMarkedTruncationCap {
		return tooBig()
	}
	if ok, err := fits(minMarkedTruncationCap); err != nil {
		return "", err
	} else if !ok {
		return tooBig()
	}

	// Binary search for the largest cap that still fits: fits(limit) is true
	// for small limits (more shortened) and false for large ones (less
	// shortened, up to and including maxLen, which is the untouched size we
	// already know overflows), so the boundary is unique.
	lo, hi := minMarkedTruncationCap, maxLen-1
	for lo < hi {
		mid := lo + (hi-lo+1)/2
		ok, err := fits(mid)
		if err != nil {
			return "", err
		}
		if ok {
			lo = mid
		} else {
			hi = mid - 1
		}
	}

	shortened, total := 0, 0
	fields := map[string]bool{}
	for _, row := range rows {
		for key, value := range row {
			text, ok := value.(string)
			if !ok {
				continue
			}
			total++
			clipped := truncateUTF8Bytes(text, lo)
			if clipped != text {
				shortened++
				fields[key] = true
			}
			row[key] = clipped
		}
	}
	if shortened == 0 {
		return "", nil
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return fmt.Sprintf("note: %d of %d string values were shortened to fit the %d-byte limit and now end with \"...\" (fields: %s); matching or filtering on those fields will miss — narrow --fields or --limit for untruncated values",
		shortened, total, maxBytes, strings.Join(names, ", ")), nil
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
