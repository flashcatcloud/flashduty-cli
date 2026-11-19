package main

import "testing"

func TestIsUnixSecondsField(t *testing.T) {
	cases := []struct {
		name  string
		field string // wire name
		kind  string
		desc  string
		want  bool
	}{
		// Unix-seconds timestamps detected by description → relative-time applies.
		{"start_time seconds", "start_time", "int", "Start of the search window, Unix epoch seconds. (required)", true},
		{"end_time seconds", "end_time", "int", "End time, Unix seconds. Must be greater than 'start_time'. (required)", true},
		{"window seconds", "end", "int", "Window end (Unix seconds, 10 digits).", true},
		{"unix timestamp seconds", "before", "int", "Filter events started at or before this unix timestamp (seconds).", true},
		{"update timestamp unix seconds", "at_seconds", "int", "Update timestamp in unix seconds. Defaults to now when omitted.", true},

		// Unix-seconds timestamps the description under-documents, caught by name.
		{"bare start window boundary", "start", "int", "When set together with end, computed layer schedules are returned. Span must be less than 45 days.", true},
		{"close_at_seconds (no unix word)", "close_at_seconds", "int", "Scheduled close time for retrospective events. Must be greater than start_at_seconds.", true},
		{"created_at_start_seconds (no unix word)", "created_at_start_seconds", "int", "Filter by creation time: lower bound in seconds.", true},
		{"created_at_end_seconds (no unix word)", "created_at_end_seconds", "int", "Filter by creation time: upper bound in seconds.", true},

		// Millisecond timestamps → excluded (timeutil.Parse yields seconds).
		{"unix milliseconds", "end_time", "int", "End of upload time range, Unix epoch milliseconds.", false},
		{"millisecond timestamp", "end_time", "int", "End of time range, millisecond timestamp.", false},
		{"unix ms range", "start_time", "int", "Window start time in Unix milliseconds.", false},

		// Durations measured in seconds → excluded (a count, not a point in time).
		// delay_seconds ends in _seconds but has no `_at_` point marker.
		{"timeout in seconds", "seconds_to_ack", "int", "Auto-resolve timeout in seconds. 0 disables auto-resolve.", false},
		{"time-to-ack bound", "seconds_to_ack_from", "int", "Lower bound (inclusive) on time-to-acknowledge, in seconds.", false},
		{"delay_seconds duration", "delay_seconds", "int", "Look-back offset in seconds applied to point-in-time queries.", false},

		// Non-int, or no unit/name signal → excluded.
		{"string field", "start_time", "string", "Start of the search window, Unix epoch seconds.", false},
		{"no description, no name signal", "limit", "int", "", false},
		{"created_at empty desc (ambiguous, not a flag-set timestamp)", "created_at", "int", "", false},
	}
	for _, c := range cases {
		if got := isUnixSecondsField(c.field, c.kind, c.desc); got != c.want {
			t.Errorf("%s: isUnixSecondsField(%q, %q, %q) = %v, want %v", c.name, c.field, c.kind, c.desc, got, c.want)
		}
	}
}

func TestTimeVarNames(t *testing.T) {
	// The parsed/ok locals must align with flagVar so the emitted code compiles.
	if got := flagVar("start_time"); got != "fStartTime" {
		t.Errorf("flagVar = %q", got)
	}
	if got := parsedTimeVar("start_time"); got != "vStartTime" {
		t.Errorf("parsedTimeVar = %q, want vStartTime", got)
	}
	if got := okTimeVar("start_time"); got != "okStartTime" {
		t.Errorf("okTimeVar = %q, want okStartTime", got)
	}
}

func TestFindTimeRange(t *testing.T) {
	const (
		secs = "Window start, Unix seconds."
		ms   = "Window start, Unix epoch milliseconds."
	)
	op := func(path string, fields map[string]string, tree ...schemaField) (specOp, []scalarField, map[string]specField) {
		o := specOp{Path: path, ReqTree: tree}
		var scalars []scalarField
		byWire := map[string]specField{}
		for wire, desc := range fields {
			scalars = append(scalars, scalarField{Wire: wire, Kind: "int"})
			byWire[wire] = specField{Wire: wire, Desc: desc}
		}
		return o, scalars, byWire
	}
	cases := []struct {
		name   string
		path   string
		fields map[string]string
		tree   []schemaField
		want   timeRange
		wantOK bool
	}{
		{"seconds start/end_time", "/x/list", map[string]string{"start_time": secs, "end_time": secs, "limit": ""},
			nil, timeRange{Start: "start_time", End: "end_time", Where: "--data or --start-time/--end-time"}, true},
		{"millisecond start/end_time", "/x/list", map[string]string{"start_time": ms, "end_time": ms},
			nil, timeRange{Start: "start_time", End: "end_time", Millis: true, Where: "--data or --start-time/--end-time"}, true},
		{"bare start/end", "/x/info", map[string]string{"start": "", "end": ""},
			nil, timeRange{Start: "start", End: "end", Where: "--data or --start/--end"}, true},
		{"after/before ms", "/x/run/list", map[string]string{"started_after_ms": "Lower bound, Unix milliseconds.", "started_before_ms": "Upper bound, Unix milliseconds."},
			nil, timeRange{Start: "started_after_ms", End: "started_before_ms", Millis: true, Where: "--data or --started-after-ms/--started-before-ms"}, true},
		{"*_at_start/end_seconds", "/x/list", map[string]string{"created_at_start_seconds": "", "created_at_end_seconds": ""},
			nil, timeRange{Start: "created_at_start_seconds", End: "created_at_end_seconds", Where: "--data or --created-at-start-seconds/--created-at-end-seconds"}, true},
		{"nested from_ms/to_ms", "/x/run", nil,
			[]schemaField{{Wire: "time", Type: "object", Children: []schemaField{
				{Wire: "from_ms", Type: "integer", Desc: "Start as a Unix timestamp in milliseconds."},
				{Wire: "to_ms", Type: "integer", Desc: "End as a Unix timestamp in milliseconds."},
			}}},
			timeRange{Container: "time", Start: "from_ms", End: "to_ms", Millis: true, Where: "--data"}, true},
		{"durations are not a window", "/x/list", map[string]string{"seconds_to_ack_from": "Lower bound, in seconds.", "seconds_to_ack_to": "Upper bound, in seconds."},
			nil, timeRange{}, false},
		{"units must match", "/x/list", map[string]string{"start_time": secs, "end_time": ms},
			nil, timeRange{}, false},
		{"start without end", "/x/list", map[string]string{"start_time": secs},
			nil, timeRange{}, false},
		{"create stores a schedule", "/x/create", map[string]string{"start_time": secs, "end_time": secs},
			nil, timeRange{}, false},
		{"update stores a schedule", "/x/update", map[string]string{"start": "", "end": ""},
			nil, timeRange{}, false},
		{"two pairs are ambiguous", "/x/list", map[string]string{"start_time": secs, "end_time": secs},
			[]schemaField{{Wire: "time", Type: "object", Children: []schemaField{
				{Wire: "from_ms", Type: "integer", Desc: "Unix timestamp in milliseconds."},
				{Wire: "to_ms", Type: "integer", Desc: "Unix timestamp in milliseconds."},
			}}},
			timeRange{}, false},
		{"array items are not a window", "/x/list", nil,
			[]schemaField{{Wire: "rows", Type: "array<object>", Children: []schemaField{
				{Wire: "from_ms", Type: "integer", Desc: "Unix timestamp in milliseconds."},
				{Wire: "to_ms", Type: "integer", Desc: "Unix timestamp in milliseconds."},
			}}},
			timeRange{}, false},
	}
	for _, c := range cases {
		o, scalars, byWire := op(c.path, c.fields, c.tree...)
		got, ok := findTimeRange(o, scalars, byWire)
		if ok != c.wantOK || got != c.want {
			t.Errorf("%s: got (%+v, %v), want (%+v, %v)", c.name, got, ok, c.want, c.wantOK)
		}
	}
}

func TestIsUnixMillisField(t *testing.T) {
	cases := []struct {
		kind, desc string
		want       bool
	}{
		{"int", "End of the query window, Unix epoch milliseconds.", true},
		{"int", "Start of the time range, Unix timestamp in milliseconds.", true},
		{"int", "Window end time in Unix milliseconds.", true},
		{"int", "Minimum time-bucket width in milliseconds.", false},
		{"int", "Window start, Unix seconds.", false},
		{"string", "Unix epoch milliseconds.", false},
	}
	for _, c := range cases {
		if got := isUnixMillisField(c.kind, c.desc); got != c.want {
			t.Errorf("isUnixMillisField(%q, %q) = %v, want %v", c.kind, c.desc, got, c.want)
		}
	}
}

func TestHasSinceUntilField(t *testing.T) {
	if hasSinceUntilField([]scalarField{{Wire: "start_time"}, {Wire: "limit"}}) {
		t.Error("no since/until field reported as present")
	}
	for _, name := range []string{"since", "until"} {
		if !hasSinceUntilField([]scalarField{{Wire: "start_time"}, {Wire: name}}) {
			t.Errorf("spec field %q not detected", name)
		}
	}
}
