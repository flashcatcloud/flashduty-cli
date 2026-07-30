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

func TestTimeFlagAliases(t *testing.T) {
	withTime := map[string]bool{"start_time": true, "end_time": true}

	// The canonical pair gets the curated dialect's spellings.
	aliases, collision := timeFlagAliases([]scalarField{
		{Wire: "start_time", Kind: "int"},
		{Wire: "end_time", Kind: "int"},
		{Wire: "limit", Kind: "int"},
	}, withTime)
	if collision != "" || aliases["start_time"] != "since" || aliases["end_time"] != "until" {
		t.Errorf("canonical pair: got aliases=%v collision=%q", aliases, collision)
	}

	// A millisecond start_time (isTime false) is NOT aliased: it has no
	// duration/date parsing for the alias to share.
	aliases, collision = timeFlagAliases([]scalarField{{Wire: "start_time", Kind: "int"}}, map[string]bool{})
	if collision != "" || len(aliases) != 0 {
		t.Errorf("non-relative-time start_time: got aliases=%v collision=%q", aliases, collision)
	}

	// Other time-flag names (start/end, *_at_seconds) are not start-time/end-time.
	aliases, collision = timeFlagAliases([]scalarField{
		{Wire: "start", Kind: "int"},
		{Wire: "close_at_seconds", Kind: "int"},
	}, map[string]bool{"start": true, "close_at_seconds": true})
	if collision != "" || len(aliases) != 0 {
		t.Errorf("other time flags: got aliases=%v collision=%q", aliases, collision)
	}

	// Collision guard: a spec-defined since/until param suppresses aliasing for
	// the whole command rather than shadowing the real flag.
	for _, name := range []string{"since", "until"} {
		aliases, collision = timeFlagAliases([]scalarField{
			{Wire: "start_time", Kind: "int"},
			{Wire: "end_time", Kind: "int"},
			{Wire: name, Kind: "string"},
		}, withTime)
		if collision != name || aliases != nil {
			t.Errorf("collision on %q: got aliases=%v collision=%q", name, aliases, collision)
		}
	}
}
