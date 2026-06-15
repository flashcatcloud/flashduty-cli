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
