package cli

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
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

// projectionBound reports how boundProjectedList reduced an over-budget
// projection — the facts, not the prose. The byte-bounding helper knows bytes
// and nothing about the command it runs under, so it cannot name the flags a
// caller should narrow; the caller that owns the command turns this into the
// stderr note via noteProjectionBound. A prefix reduction and a value
// shortening are mutually exclusive, and the zero value means nothing was
// reduced.
type projectionBound struct {
	// A prefix reduction kept rowsEmitted of rowsTotal rows, every value intact.
	rowsEmitted int
	rowsTotal   int
	// A value shortening clipped shortened of valuesTotal string values, in fields.
	shortened   int
	valuesTotal int
	fields      []string
	// maxBytes is the published limit the payload had to fit under — not the
	// rows' share of it, which is smaller when they ride inside an envelope.
	maxBytes int
}

// reduced reports whether the payload was actually reduced — rows withheld or
// values clipped. The zero value means it was emitted intact, which is what the
// in-payload truncation marker keys off.
func (b projectionBound) reduced() bool {
	return b.rowsTotal > 0 || b.shortened > 0
}

// Commands declare here which of their flags narrow structured output, so the
// projection note and overflow error can name them. Nothing infers this from a
// flag's name: a verb may spell a request-body write selector --fields, or
// accept a --limit the server ignores. Only a command whose own definition
// states what its flags do sets these, via declareOutputNarrowing.
const (
	narrowsByProjection = "narrows-output-projection"
	narrowsByRows       = "narrows-output-rows"
)

// declareOutputNarrowing records, on cmd, the flags its own definition declares
// as narrowing its structured output. projectionFlag reduces how much each row
// carries (e.g. --fields); rowsFlag reduces how many rows are requested (e.g.
// --limit). Pass "" for a control the command does not expose.
func declareOutputNarrowing(cmd *cobra.Command, projectionFlag, rowsFlag string) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	if projectionFlag != "" {
		cmd.Annotations[narrowsByProjection] = projectionFlag
	}
	if rowsFlag != "" {
		cmd.Annotations[narrowsByRows] = rowsFlag
	}
}

// declaredNarrowing returns the flag cmd declared under annotation, or "" when
// it declared none. A declaration whose flag the command no longer carries (a
// renamed flag) is dropped, so the note can never name a flag that is not there.
func declaredNarrowing(cmd *cobra.Command, annotation string) string {
	name := cmd.Annotations[annotation]
	if name == "" || cmd.Flags().Lookup(name) == nil {
		return ""
	}
	return name
}

// projectionRemedy names the flags cmd declared as able to shrink an over-budget
// projection, suffixed with what they buy (purpose). rowsHelp is false for a
// single-row shortening, where a smaller page cannot shrink the one row that
// overflows, so a rows flag is not offered there. A command that declared no
// usable flag is told so rather than sent to one it may reject or misuse.
func projectionRemedy(cmd *cobra.Command, rowsHelp bool, purpose string) string {
	projection := declaredNarrowing(cmd, narrowsByProjection)
	rows := declaredNarrowing(cmd, narrowsByRows)
	if !rowsHelp {
		rows = ""
	}
	var offered []string
	if projection != "" {
		offered = append(offered, "narrow --"+projection)
	}
	if rows != "" {
		offered = append(offered, "lower --"+rows)
	}
	if len(offered) == 0 {
		return "this command declares no flag that narrows the output"
	}
	remedy := strings.Join(offered, " or ")
	if purpose != "" {
		remedy += " " + purpose
	}
	return remedy
}

// noteProjectionBound announces on stderr how an over-budget projection was
// reduced. Without it a reduced page or a shortened value is only visible to a
// reader, not to the jq filter or exact match a --json consumer runs over it,
// so a query that silently matches nothing looks like an empty result rather
// than a bounded one.
//
// The remedy comes from cmd's own narrowing declaration, not from the
// byte-bounding helper: the helper knows bytes and nothing about the command,
// so a sentence composed there could name a flag the command rejects
// (unknown flag), ignores, or means for something else entirely.
func noteProjectionBound(cmd *cobra.Command, bound projectionBound) {
	w := cmd.ErrOrStderr()
	switch {
	case bound.rowsTotal > 0:
		_, _ = fmt.Fprintf(w, "note: emitted %d of %d projected rows (every value intact) to stay below the %d-byte structured-output limit; %s — the rows past the first %d were not emitted\n",
			bound.rowsEmitted, bound.rowsTotal, bound.maxBytes, projectionRemedy(cmd, true, "to fit more rows per page"), bound.rowsEmitted)
	case bound.shortened > 0:
		_, _ = fmt.Fprintf(w, "note: %d of %d string values were shortened to fit the %d-byte limit and now end with \"...\" (fields: %s); matching or filtering on those fields will miss — %s\n",
			bound.shortened, bound.valuesTotal, bound.maxBytes, strings.Join(bound.fields, ", "), projectionRemedy(cmd, false, "for untruncated values"))
	}
}

// projectionOverflow is the byte-bounding failure for a projection that no
// reduction can bring under the limit: no leading prefix fits and no row can be
// shortened (detail marks the single-object projection, which is refused rather
// than truncated). It carries the facts; its own message is flag-neutral, and
// callers turn it into the command-appropriate failure with
// explainProjectionOverflow.
type projectionOverflow struct {
	detail bool
	bytes  int
	rows   int
	// maxBytes is the published limit, so the error names that cap rather than
	// the rows' internal share of it.
	maxBytes int
	largest  string
}

func (o *projectionOverflow) Error() string {
	if o.detail {
		return fmt.Sprintf("projected detail is %d bytes and does not fit under the %d-byte structured-output limit; largest fields: %s",
			o.bytes, o.maxBytes, o.largest)
	}
	// The count is the rows' own encoding, the limit the payload's: inside an
	// envelope the two live in different spaces, so the sentence states the
	// size and the refusal separately rather than comparing them.
	rowNoun := "rows"
	if o.rows == 1 {
		rowNoun = "row"
	}
	return fmt.Sprintf("projected list is %d bytes across %d %s and cannot be reduced to fit under the %d-byte structured-output limit; largest fields: %s",
		o.bytes, o.rows, rowNoun, o.maxBytes, o.largest)
}

// explainProjectionOverflow returns err with the remedy for cmd appended when
// err is a projectionOverflow, so the failure names only flags cmd declared as
// output-narrowing; any other error passes through unchanged. Callers wrap each
// error returned from the bounding path with it.
func explainProjectionOverflow(cmd *cobra.Command, err error) error {
	var overflow *projectionOverflow
	if !errors.As(err, &overflow) {
		return err
	}
	if overflow.detail {
		projection := declaredNarrowing(cmd, narrowsByProjection)
		if projection == "" {
			return fmt.Errorf("%w; this command declares no flag that narrows the output", overflow)
		}
		return fmt.Errorf("%w; request fewer --%s, or omit --%s for the full, unbounded detail", overflow, projection, projection)
	}
	return fmt.Errorf("%w; %s", overflow, projectionRemedy(cmd, true, ""))
}

// boundProjectedOutput keeps the new agent-oriented projections below their
// command budget without changing the selected keys. A list projection that
// overflows the budget is first reduced to the leading rows that fit with
// every value intact; only a single row that overflows the budget on its own
// is shortened fairly, with shortened values marked with "...". A
// single-object detail projection is never modified: a truncated id or
// status string is indistinguishable from a genuinely short value, so
// silently shortening it would hand the caller wrong data instead of a
// compact one. If a detail projection doesn't fit, the command fails with an
// error instead.
//
// It returns the bounded data with the same type it was given, plus a
// projectionBound describing what was reduced (its zero value when nothing
// was), so the caller can announce the loss on stderr via noteProjectionBound
// — the "..." marker is only visible to something that reads the value, never
// to the filter a --json consumer runs over it.
func boundProjectedOutput(data any, maxBytes int) (any, projectionBound, error) {
	switch value := data.(type) {
	case map[string]any:
		return value, projectionBound{}, boundProjectedDetail(value, maxBytes)
	case []map[string]any:
		return boundProjectedList(value, maxBytes, 0)
	default:
		return nil, projectionBound{}, fmt.Errorf("internal error: unsupported projected output %T", data)
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

// boundProjectedDetail rejects an oversized single-object projection instead of
// truncating it, naming the largest fields so the request can be narrowed in one
// pass. Its message is flag-neutral; the caller appends the command-appropriate
// remedy via explainProjectionOverflow.
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
	// +1 for the trailing newline the printer appends, so the reported size is
	// the one the fit test just measured.
	return &projectionOverflow{detail: true, bytes: len(encoded) + 1, maxBytes: maxBytes, largest: largest}
}

// isIdentifierField reports whether a projected field is an identifier:
// keys ending in _id or _key (incident_id, alert_key, …). Identifier values
// are matched, filtered, and passed back verbatim by the consumer — a jq
// exact-match over --json output, or a follow-up detail call — so clipping
// one silently defeats that consumer: an identifier either survives a
// projection intact or the projection errors out.
func isIdentifierField(key string) bool {
	return strings.HasSuffix(key, "_id") || strings.HasSuffix(key, "_key")
}

// boundProjectedList keeps a list projection below the budget. A page that
// overflows is first reduced to the largest leading prefix of rows that
// fits, with every value intact: a --json consumer filters and matches on
// the values, so a partial page of intact rows serves it, while a full page
// of "..."-clipped rows silently defeats the filter. Only when one row alone
// overflows the budget does it shorten that row's string values fairly: it
// finds the largest per-field byte cap that still makes the row fit, then
// applies that one cap to every shortenable string value. A field already
// shorter than the cap is left completely untouched — only the field(s)
// actually responsible for the overflow (typically a long title) get
// shortened, each marked with "...". Identifier fields (keys ending in _id
// or _key) are exempt at every step — sizing, fitting, and applying — so
// they survive byte-intact. The cap never drops low enough to make the "..."
// marker itself disappear, so a shortened value is always distinguishable
// from a genuinely short one; if no cap at or above that floor fits, the
// command fails with a small error instead of emitting values that look
// real but aren't. Whatever it reduces or clips, it reports back in the
// returned projectionBound.
//
// The rows are sized against limit minus framing, where framing is the bytes
// the payload costs around them besides the row array (0 when the rows are the
// payload), while the returned facts name limit: the caller's note quotes the
// published cap, not an internal remainder. The input rows are never modified,
// so a caller that re-bounds them against a smaller budget (the envelope
// re-fit) measures every pass from the original values and reports facts that
// match the payload it prints.
func boundProjectedList(rows []map[string]any, limit, framing int) ([]map[string]any, projectionBound, error) {
	budget := limit - framing
	encoded, err := marshalStructured(rows)
	if err != nil {
		return nil, projectionBound{}, err
	}
	if len(encoded)+1 < budget {
		return rows, projectionBound{}, nil
	}

	// The overflow error names the fields responsible, exactly as the detail
	// path does, so the request can be narrowed in one pass.
	tooBig := func() ([]map[string]any, projectionBound, error) {
		largest, err := largestProjectedFields(rows)
		if err != nil {
			return nil, projectionBound{}, err
		}
		// +1 for the trailing newline the printer appends, so the reported size
		// is the one the fit test just measured.
		return nil, projectionBound{}, &projectionOverflow{bytes: len(encoded) + 1, rows: len(rows), maxBytes: limit, largest: largest}
	}

	kept, err := largestFittingPrefix(rows, budget)
	if err != nil {
		return nil, projectionBound{}, err
	}
	if kept > 0 {
		return rows[:kept], projectionBound{rowsEmitted: kept, rowsTotal: len(rows), maxBytes: limit}, nil
	}

	maxLen := 0
	for _, row := range rows {
		for key, value := range row {
			if isIdentifierField(key) {
				continue
			}
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
				if text, ok := value.(string); ok && !isIdentifierField(key) {
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
		return len(trialEncoded)+1 < budget, nil
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
		return nil, projectionBound{}, err
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
			return nil, projectionBound{}, err
		}
		if ok {
			lo = mid
		} else {
			hi = mid - 1
		}
	}

	shortened, total := 0, 0
	fields := map[string]bool{}
	bounded := make([]map[string]any, len(rows))
	for i, row := range rows {
		boundedRow := make(map[string]any, len(row))
		for key, value := range row {
			text, ok := value.(string)
			if !ok || isIdentifierField(key) {
				boundedRow[key] = value
				continue
			}
			total++
			clipped := truncateUTF8Bytes(text, lo)
			if clipped != text {
				shortened++
				fields[key] = true
			}
			boundedRow[key] = clipped
		}
		bounded[i] = boundedRow
	}
	if shortened == 0 {
		return rows, projectionBound{}, nil
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return bounded, projectionBound{shortened: shortened, valuesTotal: total, fields: names, maxBytes: limit}, nil
}

// largestFittingPrefix returns the largest n < len(rows) whose encoded prefix
// rows[:n] fits the budget, or 0 when even one row overflows it. Prefix size
// is monotone — appending a row never shrinks the encoding — so the boundary
// is found by binary search between lo=1 (known fitting, checked first) and
// hi=len(rows) (known not to fit: the caller only reaches here on overflow).
func largestFittingPrefix(rows []map[string]any, maxBytes int) (int, error) {
	fits := func(n int) (bool, error) {
		encoded, err := marshalStructured(rows[:n])
		if err != nil {
			return false, err
		}
		return len(encoded)+1 < maxBytes, nil
	}
	ok, err := fits(1)
	if err != nil || !ok {
		return 0, err
	}
	lo, hi := 1, len(rows) // fits(lo) holds, fits(hi) does not
	for hi-lo > 1 {
		mid := lo + (hi-lo)/2
		ok, err := fits(mid)
		if err != nil {
			return 0, err
		}
		if ok {
			lo = mid
		} else {
			hi = mid
		}
	}
	return lo, nil
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
