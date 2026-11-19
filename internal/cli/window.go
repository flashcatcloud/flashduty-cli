package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

// openStateSince is the --since a list verb applies, when --since was not
// given, to a query restricted to still-open records (incidents in progress,
// active alerts). Open records can be weeks old, so the usual 24h default
// would hide most of them; 30d is the widest window the list endpoints
// accept (< 31 days).
const openStateSince = "30d"

// onlyOpenProgress reports whether an incident --progress filter names only
// open states (Triggered, Processing) — the query whose answer the 24h default
// would silently cut to the incidents opened today.
func onlyOpenProgress(progress string) bool {
	states := parseStringSlice(progress)
	if len(states) == 0 {
		return false
	}
	for _, s := range states {
		if !strings.EqualFold(s, "Triggered") && !strings.EqualFold(s, "Processing") {
			return false
		}
	}
	return true
}

// nowFn is the clock the window note measures "ago"/"in" against; tests pin it.
var nowFn = time.Now

// parseWindow parses a curated verb's --since/--until pair into unix seconds
// and announces the effective window on stderr (see noteWindow). why, when
// non-empty, says which default produced the window.
//
// A defaulted lookback --since ("24h", "30d") spans back from --until, not
// from now: with --until moved, a now-relative default would stretch the
// window past the server's 31-day cap (future --until) or invert it (past
// --until). With --until at its "now" default the two readings coincide.
func parseWindow(cmd *cobra.Command, since, until, why string) (start, end int64, err error) {
	end, err = timeutil.Parse(until)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid --until: %w", err)
	}
	if span, ok := timeutil.Lookback(since); ok && !cmd.Flags().Changed("since") {
		start = end - int64(span/time.Second)
	} else if start, err = timeutil.Parse(since); err != nil {
		return 0, 0, fmt.Errorf("invalid --since: %w", err)
	}
	noteWindow(cmd.ErrOrStderr(), time.Unix(start, 0), time.Unix(end, 0), why)
	return start, end, nil
}

// timeRange names the request fields a --since/--until pair writes: the start
// and end fields, the object holding them ("" = the request's top level),
// their unit, and Where — the other ways those fields are set — for the
// conflict error. cligen emits one per generated read verb whose request has a
// start/end timestamp pair.
type timeRange struct {
	Container  string
	Start, End string
	Millis     bool
	Where      string
}

// toolRange is the query window inside monit-query's --params.
var toolRange = timeRange{Container: "execution", Start: "from_ms", End: "to_ms", Millis: true, Where: "--params"}

func (r timeRange) path(wire string) string {
	if r.Container == "" {
		return wire
	}
	return r.Container + "." + wire
}

func (r timeRange) time(v int64) time.Time {
	if r.Millis {
		return time.UnixMilli(v)
	}
	return time.Unix(v, 0)
}

// applyRange writes --since/--until into obj's range fields, converted to the
// fields' unit; call it only when at least one flag was set. --since alone
// ends the window now unless the end is already set; --until alone sets only
// the end (an instant evaluation time, or an end the server pairs with its own
// start). A flag whose field is already set elsewhere (Where) is rejected
// rather than one value silently winning, as are an empty flag value and a
// window whose start is not before its end.
func applyRange(obj map[string]any, r timeRange, since, until string, setSince, setUntil bool) error {
	if setSince && strings.TrimSpace(since) == "" {
		return fmt.Errorf("invalid --since: empty value")
	}
	if setUntil && strings.TrimSpace(until) == "" {
		return fmt.Errorf("invalid --until: empty value")
	}
	fields := obj
	if r.Container != "" {
		switch v := obj[r.Container].(type) {
		case map[string]any:
			fields = v
		case nil:
			fields = map[string]any{}
			obj[r.Container] = fields
		default:
			return fmt.Errorf("%s %s must be an object to apply --since/--until", r.Where, r.Container)
		}
	}
	_, hasStart := fields[r.Start]
	_, hasEnd := fields[r.End]
	if setSince && hasStart {
		return fmt.Errorf("--since conflicts with %s in %s: set the window start in one place", r.path(r.Start), r.Where)
	}
	if setUntil && hasEnd {
		return fmt.Errorf("--until conflicts with %s in %s: set the window end in one place", r.path(r.End), r.Where)
	}
	scale := int64(1)
	if r.Millis {
		scale = 1000
	}
	if setUntil || !hasEnd {
		end, err := timeutil.Parse(until)
		if err != nil {
			return fmt.Errorf("invalid --until: %w", err)
		}
		fields[r.End] = end * scale
	}
	if setSince {
		start, err := timeutil.Parse(since)
		if err != nil {
			return fmt.Errorf("invalid --since: %w", err)
		}
		fields[r.Start] = start * scale
	}
	if from, to, ok := r.bounds(fields); ok && from >= to {
		return fmt.Errorf("inverted window: start %s is not before end %s (check --since/--until and %s)",
			formatLocal(r.time(from)), formatLocal(r.time(to)), r.Where)
	}
	return nil
}

// genApplyRange is a generated verb's --since/--until: applyRange on the
// assembled request body, then a note of the window when a flag resolved it
// (--since/--until, or a top-level field's own flag) and both ends are set. A
// window that came in only through --data is the caller's literal body, not a
// value the CLI resolved, so it is not announced.
func genApplyRange(cmd *cobra.Command, body map[string]any, r timeRange, since, until string) error {
	setSince, setUntil := cmd.Flags().Changed("since"), cmd.Flags().Changed("until")
	if setSince || setUntil {
		if err := applyRange(body, r, since, until, setSince, setUntil); err != nil {
			return err
		}
	}
	// A top-level field's flag is its wire name in kebab case; nested range
	// fields have no flag of their own.
	ownFlag := r.Container == "" &&
		(cmd.Flags().Changed(strings.ReplaceAll(r.Start, "_", "-")) || cmd.Flags().Changed(strings.ReplaceAll(r.End, "_", "-")))
	if !setSince && !setUntil && !ownFlag {
		return nil
	}
	fields, _ := body[r.Container].(map[string]any)
	if r.Container == "" {
		fields = body
	}
	if from, to, ok := r.bounds(fields); ok {
		noteWindow(cmd.ErrOrStderr(), r.time(from), r.time(to), "")
	}
	return nil
}

// bounds reads both range fields from the object holding them; ok is false
// unless both are numbers.
func (r timeRange) bounds(fields map[string]any) (from, to int64, ok bool) {
	from, okFrom := rangeValue(fields[r.Start])
	to, okTo := rangeValue(fields[r.End])
	return from, to, okFrom && okTo
}

// rangeValue reads a range field set by a flag (int64) or carried in --data /
// --params (json.Number).
func rangeValue(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	}
	return 0, false
}

// noteWindow writes the query window a command actually sent, in the
// process's local timezone, with its span and where its end sits relative to
// now. A list that comes back short or empty can then be told apart from one
// whose window was not the intended one — a default the caller didn't know
// about, or an epoch converted in the wrong timezone (which lands the window
// in the future: "ends in 7h"). stderr keeps stdout byte-identical for
// json/toon consumers.
func noteWindow(w io.Writer, start, end time.Time, why string) {
	detail := formatSpan(end.Sub(start)) + ", " + relativeToNow(end, "ended", "ends")
	if why != "" {
		detail += "; " + why
	}
	_, _ = fmt.Fprintf(w, "note: window %s..%s (%s)\n", formatLocal(start), formatLocal(end), detail)
}

// noteInstant is noteWindow for a single evaluation timestamp.
func noteInstant(w io.Writer, at time.Time) {
	_, _ = fmt.Fprintf(w, "note: instant %s (%s)\n", formatLocal(at), relativeToNow(at, "", ""))
}

// noteToolWindow announces the window of a datasource query tool from its
// params' execution block ({"kind","from_ms","to_ms"}). Params without an
// execution window (diagnostic tools) produce no note; the params themselves
// are never altered.
func noteToolWindow(w io.Writer, params json.RawMessage) {
	if len(params) == 0 {
		return
	}
	var p struct {
		Execution *struct {
			Kind   string       `json:"kind"`
			FromMS *json.Number `json:"from_ms"`
			ToMS   *json.Number `json:"to_ms"`
		} `json:"execution"`
	}
	dec := json.NewDecoder(bytes.NewReader(params))
	dec.UseNumber()
	if err := dec.Decode(&p); err != nil || p.Execution == nil || p.Execution.ToMS == nil {
		return
	}
	to, err := p.Execution.ToMS.Int64()
	if err != nil {
		return
	}
	if p.Execution.FromMS == nil {
		noteInstant(w, time.UnixMilli(to))
		return
	}
	from, err := p.Execution.FromMS.Int64()
	if err != nil {
		return
	}
	noteWindow(w, time.UnixMilli(from), time.UnixMilli(to), "")
}

func formatLocal(t time.Time) string {
	return t.In(time.Local).Format(time.RFC3339)
}

// relativeToNow renders t against nowFn: "3m ago" / "in 2h" (or, with
// verbs, "ended 3m ago" / "ends in 2h"); within a minute of now it is "now".
func relativeToNow(t time.Time, past, future string) string {
	d := t.Sub(nowFn())
	switch {
	case d.Abs() < time.Minute:
		return strings.TrimSpace(past + " now")
	case d < 0:
		return strings.TrimSpace(past + " " + formatSpan(-d) + " ago")
	default:
		return strings.TrimSpace(future + " in " + formatSpan(d))
	}
}

// formatSpan renders d compactly, to the minute: 30d, 1d2h, 1h, 1h30m, 45m.
func formatSpan(d time.Duration) string {
	d = d.Round(time.Minute)
	if d < time.Minute {
		return "0m"
	}
	days := int(d / (24 * time.Hour))
	d -= time.Duration(days) * 24 * time.Hour
	hours := int(d / time.Hour)
	mins := int((d - time.Duration(hours)*time.Hour) / time.Minute)
	var b strings.Builder
	if days > 0 {
		fmt.Fprintf(&b, "%dd", days)
	}
	if hours > 0 {
		fmt.Fprintf(&b, "%dh", hours)
	}
	if mins > 0 {
		fmt.Fprintf(&b, "%dm", mins)
	}
	return b.String()
}
