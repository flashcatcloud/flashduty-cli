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
func parseWindow(cmd *cobra.Command, since, until, why string) (start, end int64, err error) {
	start, err = timeutil.Parse(since)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid --since: %w", err)
	}
	end, err = timeutil.Parse(until)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid --until: %w", err)
	}
	noteWindow(cmd.ErrOrStderr(), time.Unix(start, 0), time.Unix(end, 0), why)
	return start, end, nil
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
