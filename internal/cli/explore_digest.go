package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/flashcatcloud/go-flashduty"
)

// Digest bounds. Items (samples, series, rows, entries, field summaries) are
// added while the whole digest stays within digestBudget. Only the first
// section that prints items is guaranteed one item (capped at digestLineCap);
// a later section that no longer fits is reduced to one line with its counts,
// so the digest stays near 8 KB however large the result is.
const (
	digestBudget     = 6400
	digestLineCap    = 1536
	digestMaxSeries  = 20
	digestMaxSamples = 20
	digestBuckets    = 12
	digestCellCap    = 200
	digestFieldCap   = 200
	digestLabelCap   = 300
	digestFooter     = "complete result: rerun with --output-format json\n"

	// Logs: entries own the budget; the constant and distribution sections
	// together get at most digestLogMetaBudget. A field distribution is shown
	// only for results of at least digestLowCardMinEntries entries, for at most
	// digestLowCardMaxFields fields with ≤ digestLowCard distinct values each
	// no longer than digestLowCardValueCap.
	digestLogMetaBudget     = digestBudget / 4
	digestLogLineCap        = 700
	digestLowCard           = 5
	digestLowCardMinEntries = 20
	digestLowCardMaxFields  = 8
	digestLowCardValueCap   = 40
)

// logMessageFields are conventional message field names; the first one an
// entry carries is printed right after its timestamp.
var logMessageFields = []string{"_msg", "message", "msg", "line", "log", "content"}

type exploreResult struct {
	Format string `json:"format"`
	Result struct {
		Kind    string `json:"kind"`
		Samples []struct {
			Labels map[string]string `json:"labels"`
			Value  any               `json:"value"`
		} `json:"samples"`
		Frames []struct {
			Kind   string         `json:"kind"`
			Fields []exploreField `json:"fields"`
		} `json:"frames"`
		Entries []struct {
			TimestampNS any            `json:"timestamp_ns"`
			Fields      map[string]any `json:"fields"`
		} `json:"entries"`
		AppliedLimit *json.Number `json:"applied_limit"`
		HasMore      *bool        `json:"has_more"`
	} `json:"result"`
	Execution struct {
		EffectiveStepSeconds *json.Number `json:"effective_step_seconds"`
	} `json:"execution"`
}

type exploreField struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Labels map[string]string `json:"labels"`
	Values []any             `json:"values"`
}

// exploreDigest renders a query tool's explore_result.v1 as a bounded
// plain-text digest. ok is false for any other result (diagnostic tools, an
// unknown result kind, a shape it cannot read): the caller then prints the
// result the usual way.
func exploreDigest(out *flashduty.DatasourceToolResult) (string, bool) {
	if out == nil {
		return "", false
	}
	var r exploreResult
	dec := json.NewDecoder(bytes.NewReader(out.Data))
	dec.UseNumber()
	if err := dec.Decode(&r); err != nil || r.Format != "explore_result.v1" {
		return "", false
	}
	d := &digest{}
	if out.Truncated != nil {
		d.line("note: result truncated by the server: %s", escapeText(capText(out.Truncated.Reason, digestFieldCap)))
	}
	var ok bool
	switch r.Result.Kind {
	case "samples":
		ok = d.samples(&r)
	case "frames":
		ok = d.frames(&r)
	case "logs":
		ok = d.logs(&r)
	}
	if !ok {
		return "", false
	}
	return d.b.String(), true
}

type digest struct {
	b     strings.Builder
	shown bool // a section has printed items; later sections get no minimum
}

func (d *digest) line(format string, a ...any) {
	_, _ = fmt.Fprintf(&d.b, format, a...)
	d.b.WriteByte('\n')
}

// fit returns how many of items (at most limit) fit in the remaining budget
// after reserve bytes still to be written ahead of them. The first section to
// print items gets at least one; later sections may get none.
func (d *digest) fit(items []string, limit, reserve int) int {
	used := d.b.Len() + reserve
	n := 0
	for _, it := range items {
		if n == limit || ((n > 0 || d.shown) && used+len(it)+1 > digestBudget) {
			break
		}
		used += len(it) + 1
		n++
	}
	if n > 0 {
		d.shown = true
	}
	return n
}

func (d *digest) items(items []string) {
	for _, it := range items {
		d.b.WriteString(it)
		d.b.WriteByte('\n')
	}
}

func (d *digest) nothing(what string) {
	d.line("query succeeded: no %s matched in the window", what)
}

func (d *digest) footer() {
	d.b.WriteString(digestFooter)
}

// --- samples ---

type digestSample struct {
	labels map[string]string
	id     string // the labels not common to every sample
	raw    string
	value  float64
	ok     bool // value is numeric (possibly non-finite)
}

func (d *digest) samples(r *exploreResult) bool {
	in := r.Result.Samples
	if len(in) == 0 {
		d.nothing("samples")
		return true
	}
	all := make([]digestSample, len(in))
	sets := make([]map[string]string, len(in))
	var finite []float64
	for i, s := range in {
		v, ok := digestFloat(s.Value)
		all[i] = digestSample{labels: s.Labels, raw: rawText(s.Value), value: v, ok: ok}
		sets[i] = s.Labels
		if ok && isFinite(v) {
			finite = append(finite, v)
		}
	}
	common := commonLabels(sets)
	for i := range all {
		all[i].id = labelString(withoutLabels(all[i].labels, common))
	}
	sort.SliceStable(all, func(i, j int) bool {
		ri, rj := sampleRank(all[i]), sampleRank(all[j])
		if ri != rj {
			return ri < rj
		}
		if ri == 1 && all[i].value != all[j].value {
			return all[i].value > all[j].value
		}
		return all[i].id < all[j].id
	})
	width := 0
	for _, s := range all[:min(len(all), digestMaxSamples)] {
		width = max(width, len(sampleValue(s)))
	}
	lines := make([]string, len(all))
	for i, s := range all {
		lines[i] = capLine(fmt.Sprintf("  %-*s  %s", width, sampleValue(s), capText(s.id, digestLabelCap)))
	}
	shown := d.fit(lines, digestMaxSamples, 200)
	header := fmt.Sprintf("samples: %d, sorted by value descending", len(all))
	if shown < len(all) {
		header = fmt.Sprintf("samples: showing %d of %d, sorted by value descending", shown, len(all))
	}
	d.line("%s", header)
	d.commonLabels(common)
	if shown < len(all) && len(finite) > 0 {
		sort.Float64s(finite)
		d.line("all %d finite values: min %s, median %s, max %s", len(finite),
			fmtNum(finite[0]), fmtNum(median(finite)), fmtNum(finite[len(finite)-1]))
	}
	if nonFinite := len(all) - len(finite); nonFinite > 0 {
		d.line("non-finite or non-numeric values: %d", nonFinite)
	}
	d.items(lines[:shown])
	d.footer()
	return true
}

// sampleRank orders +Inf first, finite values next, then -Inf, NaN and
// non-numeric values.
func sampleRank(s digestSample) int {
	switch {
	case !s.ok:
		return 4
	case math.IsInf(s.value, 1):
		return 0
	case math.IsInf(s.value, -1):
		return 2
	case math.IsNaN(s.value):
		return 3
	}
	return 1
}

func sampleValue(s digestSample) string {
	if !s.ok {
		return escapeText(capText(s.raw, digestCellCap))
	}
	return fmtNum(s.value)
}

// --- frames ---

type digestSeries struct {
	name      string // value field name, when not the default "value"
	labels    map[string]string
	id        string // name plus the labels not common to every series
	times     []time.Time
	values    []float64 // NaN marks a non-finite or non-numeric point
	excluded  int
	max       float64
	hasFinite bool
}

func (d *digest) frames(r *exploreResult) bool {
	var series []digestSeries
	var tables [][]exploreField
	tsFrames := 0
	for _, f := range r.Result.Frames {
		if f.Kind == "time_series" {
			s, ok := frameSeries(f.Fields)
			if !ok {
				return false
			}
			series = append(series, s...)
			tsFrames++
			continue
		}
		tables = append(tables, f.Fields)
	}
	rows := 0
	for _, t := range tables {
		rows += tableRows(t)
	}
	points := 0
	for _, s := range series {
		points += len(s.times)
	}
	if points == 0 && rows == 0 {
		d.nothing("series or rows")
		return true
	}
	if len(series) > 0 {
		d.timeSeries(r, series, tsFrames)
	}
	for i, t := range tables {
		if d.table(t, i, len(tables)) || i == len(tables)-1 {
			continue
		}
		rest := 0
		for _, more := range tables[i+1:] {
			rest += tableRows(more)
		}
		d.line("(%s, %s, not shown)", plural(len(tables)-i-1, "more table frame"), plural(rest, "row"))
		break
	}
	d.footer()
	return true
}

// frameSeries reads a time_series frame: its time field plus one series per
// float field.
func frameSeries(fields []exploreField) ([]digestSeries, bool) {
	var times []time.Time
	found := false
	for _, f := range fields {
		if f.Type != "time" {
			continue
		}
		found = true
		times = make([]time.Time, len(f.Values))
		for i, v := range f.Values {
			t, ok := digestTime(v)
			if !ok {
				return nil, false
			}
			times[i] = t
		}
		break
	}
	if !found {
		return nil, false
	}
	// Order the axis by time; stats, range and buckets read it positionally.
	order := make([]int, len(times))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return times[order[a]].Before(times[order[b]]) })
	sorted := make([]time.Time, len(times))
	for i, j := range order {
		sorted[i] = times[j]
	}
	times = sorted
	var out []digestSeries
	for _, f := range fields {
		if f.Type == "time" {
			continue
		}
		if len(f.Values) != len(times) {
			return nil, false
		}
		s := digestSeries{labels: f.Labels, times: times, values: make([]float64, len(times))}
		if f.Name != "value" {
			s.name = f.Name
		}
		for i, j := range order {
			v, ok := digestFloat(f.Values[j])
			if !ok || !isFinite(v) {
				s.values[i] = math.NaN()
				s.excluded++
				continue
			}
			s.values[i] = v
			if !s.hasFinite || v > s.max {
				s.max = v
			}
			s.hasFinite = true
		}
		out = append(out, s)
	}
	return out, true
}

func (d *digest) timeSeries(r *exploreResult, series []digestSeries, frames int) {
	sets := make([]map[string]string, len(series))
	for i := range series {
		sets[i] = series[i].labels
	}
	common := commonLabels(sets)
	for i := range series {
		series[i].id = escapeText(series[i].name) + labelString(withoutLabels(series[i].labels, common))
	}
	sort.SliceStable(series, func(i, j int) bool {
		a, b := series[i], series[j]
		if a.hasFinite != b.hasFinite {
			return a.hasFinite
		}
		if a.hasFinite && a.max != b.max {
			return a.max > b.max
		}
		return a.id < b.id
	})
	// When every series shares one time axis its point count and range are
	// stated once in the header instead of per series.
	shared := len(series[0].times) > 0
	for _, s := range series[1:] {
		shared = shared && sameAxis(s.times, series[0].times)
	}
	blocks := make([]string, len(series))
	for i, s := range series {
		blocks[i] = seriesBlock(i+1, s, shared)
	}
	shown := d.fit(blocks, digestMaxSeries, 400)
	head := fmt.Sprintf("time_series: %d series", len(series))
	if shown < len(series) {
		head = fmt.Sprintf("time_series: showing %d of %d series", shown, len(series))
	}
	head += " (" + plural(frames, "frame") + "), sorted by max descending"
	if step := r.Execution.EffectiveStepSeconds; step != nil {
		head += "; step " + step.String() + "s"
	}
	d.line("%s", head)
	d.commonLabels(common)
	if shared {
		s := series[0]
		d.line("all series: %s, %s .. %s", plural(len(s.times), "point"), formatLocal(s.times[0]), formatLocal(s.times[len(s.times)-1]))
	}
	d.items(blocks[:shown])
}

func sameAxis(a, b []time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}

// bucketWidth splits the series' span, counted to one step past its last
// point, into digestBuckets equal buckets; 0 when the points share one time.
// Float seconds keep span × points from overflowing int64 nanoseconds.
func bucketWidth(times []time.Time) time.Duration {
	n := float64(len(times))
	span := times[len(times)-1].Sub(times[0]).Seconds()
	return time.Duration(span * n / (n - 1) / digestBuckets * float64(time.Second))
}

func seriesBlock(n int, s digestSeries, shared bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%d] %s", n, capText(s.id, digestLabelCap))
	if !shared {
		fmt.Fprintf(&b, "  %s", plural(len(s.times), "point"))
		if len(s.times) > 0 {
			fmt.Fprintf(&b, ", %s .. %s", formatLocal(s.times[0]), formatLocal(s.times[len(s.times)-1]))
		}
	}
	if s.excluded > 0 {
		fmt.Fprintf(&b, "  (%d non-finite excluded from stats)", s.excluded)
	}
	if !s.hasFinite {
		b.WriteString("\n    no finite values")
		return b.String()
	}
	minI, maxI, sum, cnt := -1, -1, 0.0, 0
	for i, v := range s.values {
		if math.IsNaN(v) {
			continue
		}
		if minI < 0 || v < s.values[minI] {
			minI = i
		}
		if maxI < 0 || v > s.values[maxI] {
			maxI = i
		}
		sum += v
		cnt++
	}
	fmt.Fprintf(&b, "\n    max %s at %s, min %s at %s", fmtNum(s.values[maxI]), formatLocal(s.times[maxI]),
		fmtNum(s.values[minI]), formatLocal(s.times[minI]))
	fmt.Fprintf(&b, "\n    avg %s, first %s, last %s", fmtNum(sum/float64(cnt)),
		fmtPoint(s.values[0]), fmtPoint(s.values[len(s.values)-1]))
	if len(s.times) <= digestBuckets {
		stamps := compactTimes(s.times, s.times)
		parts := make([]string, len(s.times))
		for i := range s.times {
			parts[i] = stamps[i] + " " + fmtPoint(s.values[i])
		}
		b.WriteString("\n    points: " + strings.Join(parts, " | "))
		return b.String()
	}
	width := bucketWidth(s.times)
	if width <= 0 {
		return b.String()
	}
	buckets := make([]float64, digestBuckets)
	filled := make([]bool, digestBuckets)
	for i, t := range s.times {
		v := s.values[i]
		if math.IsNaN(v) {
			continue
		}
		k := min(max(int(float64(t.Sub(s.times[0]))/float64(width)), 0), digestBuckets-1)
		if !filled[k] || v > buckets[k] {
			buckets[k] = v
		}
		filled[k] = true
	}
	starts := make([]time.Time, digestBuckets)
	for k := range starts {
		starts[k] = s.times[0].Add(time.Duration(k) * width)
	}
	stamps := compactTimes(starts, s.times)
	parts := make([]string, digestBuckets)
	for k := range buckets {
		parts[k] = stamps[k] + " -"
		if filled[k] {
			parts[k] = stamps[k] + " " + fmtNum(buckets[k])
		}
	}
	fmt.Fprintf(&b, "\n    shape (bucket start → max, %s buckets): %s", fmtDuration(width), strings.Join(parts, " | "))
	return b.String()
}

// compactTimes formats ts in local time as 15:04 when the series' first and
// last points fall on one local date, else as 01-02T15:04; seconds are added
// when minutes alone would print two neighbouring times the same.
func compactTimes(ts, series []time.Time) []string {
	layout := "15:04"
	first, last := series[0].In(time.Local), series[len(series)-1].In(time.Local)
	if first.Format(time.DateOnly) != last.Format(time.DateOnly) {
		layout = "01-02T15:04"
	}
	for i := 1; i < len(ts); i++ {
		if ts[i].In(time.Local).Format(layout) == ts[i-1].In(time.Local).Format(layout) {
			layout += ":05"
			break
		}
	}
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.In(time.Local).Format(layout)
	}
	return out
}

func fmtPoint(v float64) string {
	if math.IsNaN(v) {
		return "non-finite"
	}
	return fmtNum(v)
}

// --- table frames ---

func tableRows(fields []exploreField) int {
	n := 0
	for _, f := range fields {
		n = max(n, len(f.Values))
	}
	return n
}

// table prints one table frame, or only its counts when it no longer fits
// the budget; it reports whether it printed the frame's content.
func (d *digest) table(fields []exploreField, i, total int) bool {
	rows := tableRows(fields)
	label := "table"
	if total > 1 {
		label = fmt.Sprintf("table %d/%d", i+1, total)
	}
	names := make([]string, len(fields))
	for c, f := range fields {
		names[c] = escapeText(capText(f.Name, digestCellCap))
	}
	lines := make([]string, rows)
	cells := make([]string, len(fields))
	for row := range rows {
		for c, f := range fields {
			cells[c] = ""
			if row < len(f.Values) {
				cells[c] = tableCell(f.Values[row], f.Type)
			}
		}
		lines[row] = capLine(strings.Join(cells, " | "))
	}
	colLine := capLine("  " + strings.Join(names, " | "))
	shown := d.fit(lines, rows, len(colLine)+100)
	if d.shown && shown == 0 && (rows > 0 || d.b.Len()+len(colLine)+100 > digestBudget) {
		d.line("%s: %d rows × %d columns not shown", label, rows, len(fields))
		return false
	}
	switch {
	case rows == 0:
		d.line("%s: 0 rows × %d columns", label, len(fields))
	case shown < rows:
		d.line("%s: %d rows × %d columns, showing %d of %d rows", label, rows, len(fields), shown, rows)
	default:
		d.line("%s: %d rows × %d columns", label, rows, len(fields))
	}
	d.line("%s", colLine)
	for _, l := range lines[:shown] {
		d.line("  %s", l)
	}
	return true
}

func tableCell(v any, typ string) string {
	if typ == "time" {
		if t, ok := digestTime(v); ok {
			return formatLocal(t)
		}
	}
	return escapeText(capText(rawText(v), digestCellCap))
}

// --- logs ---

type logField struct {
	name     string
	present  int
	counts   map[string]int
	echoesTS bool
}

func (d *digest) logs(r *exploreResult) bool {
	entries := r.Result.Entries
	if len(entries) == 0 {
		d.nothing("log entries")
		return true
	}
	limit := "unknown"
	if r.Result.AppliedLimit != nil {
		limit = r.Result.AppliedLimit.String()
	}
	hasMore := "unknown"
	if r.Result.HasMore != nil {
		hasMore = strconv.FormatBool(*r.Result.HasMore)
	}
	times := make([]time.Time, len(entries))
	for i, e := range entries {
		t, ok := nsTime(e.TimestampNS)
		if !ok {
			return false
		}
		times[i] = t
	}
	byName := map[string]*logField{}
	for i, e := range entries {
		for name, raw := range e.Fields {
			f := byName[name]
			if f == nil {
				f = &logField{name: name, counts: map[string]int{}, echoesTS: true}
				byName[name] = f
			}
			f.present++
			f.counts[rawText(raw)]++
			if f.echoesTS && !timestampEcho(raw, times[i]) {
				f.echoesTS = false
			}
		}
	}
	fields := make([]*logField, 0, len(byName))
	for _, f := range byName {
		fields = append(fields, f)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].name < fields[j].name })

	var constants, lowCard, varying []string
	for _, f := range fields {
		switch {
		case f.echoesTS:
		case f.present == len(entries) && len(f.counts) == 1:
			for v := range f.counts {
				constants = append(constants, capLine("  "+escapeText(f.name)+"="+logValue(v)))
			}
		default:
			varying = append(varying, f.name)
			if lowCardinality(f, len(entries)) {
				lowCard = append(lowCard, capLine("  "+escapeText(f.name)+": "+valueCounts(f, len(entries))))
			}
		}
	}
	varying = messageFirst(varying)

	d.line("logs: %s (applied_limit %s, has_more %s), first %s, last %s",
		plural(len(entries), "entry"), limit, hasMore, formatLocal(times[0]), formatLocal(times[len(times)-1]))
	meta := digestLogMetaBudget
	d.fieldGroup("constant fields (same in every entry)", constants, len(constants), &meta)
	d.fieldGroup("fields with few distinct values", lowCard[:min(len(lowCard), digestLowCardMaxFields)], len(lowCard), &meta)

	lines := make([]string, len(entries))
	dropped := false
	for i, e := range entries {
		line := "  " + formatLocal(times[i])
		var parts []string
		for _, name := range varying {
			if raw, present := e.Fields[name]; present {
				parts = append(parts, escapeText(name)+"="+logValue(rawText(raw)))
			}
		}
		for j, part := range parts {
			if j > 0 && len(line)+1+len(part) > digestLogLineCap {
				line += " …(+" + plural(len(parts)-j, "more field") + ")"
				dropped = true
				break
			}
			line += " " + part
		}
		lines[i] = line
	}
	shown := d.fit(lines, len(lines), 150)
	if shown < len(entries) {
		d.line("entries (varying fields), showing %d of %d:", shown, len(entries))
	} else {
		d.line("entries (varying fields):")
	}
	d.items(lines[:shown])
	if shown < len(entries) || dropped {
		d.b.WriteString(strings.TrimSuffix(digestFooter, "\n") + "; projecting fewer fields in the query narrows each entry\n")
	} else {
		d.footer()
	}
	return true
}

// lowCardinality reports whether a field's value distribution is worth a
// line: enough entries to show a pattern, few short distinct values, and at
// least one value repeated.
func lowCardinality(f *logField, entries int) bool {
	if entries < digestLowCardMinEntries || len(f.counts) > digestLowCard || len(f.counts) == f.present {
		return false
	}
	for v := range f.counts {
		if utf8.RuneCountInString(v) > digestLowCardValueCap {
			return false
		}
	}
	return true
}

// messageFirst moves the first conventional message field to the front;
// names is sorted and stays so otherwise.
func messageFirst(names []string) []string {
	for _, m := range logMessageFields {
		for i, n := range names {
			if n == m {
				return append([]string{m}, append(names[:i:i], names[i+1:]...)...)
			}
		}
	}
	return names
}

// fieldGroup prints a titled list of field lines while they fit the
// remaining section allowance, stating how many of total were left out.
func (d *digest) fieldGroup(title string, lines []string, total int, allowance *int) {
	if len(lines) == 0 {
		return
	}
	shown, used := 0, len(title)+32
	for _, l := range lines {
		if used+len(l)+1 > *allowance {
			break
		}
		used += len(l) + 1
		shown++
	}
	*allowance -= used
	if shown < total {
		d.line("%s, showing %d of %d:", title, shown, total)
	} else {
		d.line("%s:", title)
	}
	d.items(lines[:shown])
}

func valueCounts(f *logField, entries int) string {
	vals := make([]string, 0, len(f.counts))
	for v := range f.counts {
		vals = append(vals, v)
	}
	sort.Slice(vals, func(i, j int) bool {
		if f.counts[vals[i]] != f.counts[vals[j]] {
			return f.counts[vals[i]] > f.counts[vals[j]]
		}
		return vals[i] < vals[j]
	})
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%s (%d)", logValue(v), f.counts[v])
	}
	if missing := entries - f.present; missing > 0 {
		parts = append(parts, fmt.Sprintf("absent (%d)", missing))
	}
	return strings.Join(parts, ", ")
}

// logValue truncates a field value and quotes it when it would not read as
// one token.
func logValue(v string) string {
	v = capText(v, digestFieldCap)
	if v == "" || strings.ContainsAny(v, " \"=") || hasControl(v) {
		return strconv.Quote(v)
	}
	return v
}

// timestampEcho reports whether a log field value is the entry's own
// timestamp again (an RFC3339 string or a unix s/ms/us/ns number within a
// second of it).
func timestampEcho(raw any, ts time.Time) bool {
	t, ok := digestTime(raw)
	return ok && t.Sub(ts).Abs() < time.Second
}

func unixAnyPrecision(n int64) time.Time {
	switch {
	case n >= 1e17:
		return time.Unix(0, n)
	case n >= 1e14:
		return time.UnixMicro(n)
	case n >= 1e11:
		return time.UnixMilli(n)
	}
	return time.Unix(n, 0)
}

// --- value helpers ---

// digestFloat reads a numeric value: a JSON number, a numeric string, or one
// of the strings "NaN", "+Inf", "-Inf".
func digestFloat(v any) (float64, bool) {
	var s string
	switch x := v.(type) {
	case json.Number:
		s = x.String()
	case string:
		s = strings.TrimSpace(x)
	default:
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

func isFinite(f float64) bool { return !math.IsNaN(f) && !math.IsInf(f, 0) }

func digestTime(v any) (time.Time, bool) {
	switch x := v.(type) {
	case string:
		t, err := time.Parse(time.RFC3339Nano, x)
		return t, err == nil
	case json.Number:
		n, err := x.Int64()
		return unixAnyPrecision(n), err == nil
	}
	return time.Time{}, false
}

// nsTime reads a log entry's timestamp_ns: a decimal string or a JSON integer.
func nsTime(v any) (time.Time, bool) {
	var s string
	switch x := v.(type) {
	case string:
		s = x
	case json.Number:
		s = x.String()
	default:
		return time.Time{}, false
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || n > math.MaxInt64 {
		return time.Time{}, false
	}
	return time.Unix(0, int64(n)), true
}

// fmtNum prints whole numbers below 1e15 in full and other values with five
// significant digits, trailing zeros trimmed; exponent form only outside
// [1e-4, 1e15).
func fmtNum(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "+Inf"
	case math.IsInf(f, -1):
		return "-Inf"
	case f == 0:
		return "0"
	case math.Abs(f) < 1e15 && f == math.Trunc(f):
		return strconv.FormatInt(int64(f), 10)
	case math.Abs(f) >= 1e-4 && math.Abs(f) < 1e15:
		rounded, _ := strconv.ParseFloat(strconv.FormatFloat(f, 'g', 5, 64), 64)
		return strconv.FormatFloat(rounded, 'f', -1, 64)
	}
	return strconv.FormatFloat(f, 'g', 5, 64)
}

func fmtDuration(d time.Duration) string {
	switch {
	case d >= time.Minute && d%time.Minute == 0:
		return formatSpan(d)
	case d >= time.Second:
		return d.Round(time.Second).String()
	}
	return d.Round(time.Millisecond).String()
}

func median(sorted []float64) float64 {
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// commonLabels returns the label pairs every set carries with the same value.
func commonLabels(sets []map[string]string) map[string]string {
	if len(sets) == 0 {
		return nil
	}
	common := map[string]string{}
	for k, v := range sets[0] {
		common[k] = v
	}
	for _, set := range sets[1:] {
		for k, v := range common {
			if w, ok := set[k]; !ok || w != v {
				delete(common, k)
			}
		}
	}
	return common
}

func withoutLabels(labels, drop map[string]string) map[string]string {
	out := make(map[string]string, len(labels))
	for k, v := range labels {
		if _, common := drop[k]; !common {
			out[k] = v
		}
	}
	return out
}

func (d *digest) commonLabels(common map[string]string) {
	if len(common) > 0 {
		d.line("%s", capLine("common labels: "+labelString(common)))
	}
}

// labelString renders a label set as {k="v", ...} sorted by key.
func labelString(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = escapeText(k) + "=" + strconv.Quote(labels[k])
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// rawText renders a decoded JSON value as text: strings as-is, numbers by
// their original digits, everything else as compact JSON.
func rawText(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case nil:
		return "null"
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return strings.TrimRight(buf.String(), "\n")
}

// escapeText quotes s Go-style when it carries a control character or
// invalid UTF-8, so text from a result can never move the terminal cursor,
// clear the screen or break the line structure of the digest.
func escapeText(s string) string {
	if hasControl(s) {
		return strconv.Quote(s)
	}
	return s
}

func hasControl(s string) bool {
	if !utf8.ValidString(s) {
		return true
	}
	return strings.IndexFunc(s, unicode.IsControl) >= 0
}

// capText truncates s to limit runes, stating how many were cut.
func capText(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	runes := []rune(s)
	return fmt.Sprintf("%s…(+%d chars)", string(runes[:limit]), len(runes)-limit)
}

// capLine bounds one digest line by bytes, stating how many were cut.
func capLine(s string) string {
	if len(s) <= digestLineCap {
		return s
	}
	cut := digestLineCap
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return fmt.Sprintf("%s…(line cut, +%d bytes)", s[:cut], len(s)-cut)
}

func plural(n int, noun string) string {
	switch {
	case n == 1:
		return "1 " + noun
	case strings.HasSuffix(noun, "y"):
		return strconv.Itoa(n) + " " + strings.TrimSuffix(noun, "y") + "ies"
	}
	return strconv.Itoa(n) + " " + noun + "s"
}
