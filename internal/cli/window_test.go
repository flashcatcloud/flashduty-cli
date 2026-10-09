package cli

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"
)

// pinClock fixes nowFn and the local timezone (+08:00) for the test.
func pinClock(t *testing.T, now time.Time) {
	t.Helper()
	origNow, origLocal := nowFn, time.Local
	nowFn = func() time.Time { return now }
	time.Local = time.FixedZone("CST", 8*3600)
	t.Cleanup(func() { nowFn, time.Local = origNow, origLocal })
}

func TestNoteWindowText(t *testing.T) {
	now := time.Date(2026, 10, 9, 6, 0, 0, 0, time.UTC) // 14:00 +08:00
	pinClock(t, now)
	cases := []struct {
		name       string
		start, end time.Time
		why        string
		want       string
	}{
		{
			name:  "ends now, with default reason",
			start: now.Add(-30 * 24 * time.Hour), end: now,
			why:  "default --since 30d because of --active",
			want: "note: window 2026-09-09T14:00:00+08:00..2026-10-09T14:00:00+08:00 (30d, ended now; default --since 30d because of --active)\n",
		},
		{
			name:  "past",
			start: now.Add(-time.Hour - 3*time.Minute), end: now.Add(-3 * time.Minute),
			want: "note: window 2026-10-09T12:57:00+08:00..2026-10-09T13:57:00+08:00 (1h, ended 3m ago)\n",
		},
		{
			name:  "future end exposes a timezone slip",
			start: now.Add(7 * time.Hour), end: now.Add(8*time.Hour + 30*time.Minute),
			want: "note: window 2026-10-09T21:00:00+08:00..2026-10-09T22:30:00+08:00 (1h30m, ends in 8h30m)\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			noteWindow(&buf, tc.start, tc.end, tc.why)
			if buf.String() != tc.want {
				t.Fatalf("got  %q\nwant %q", buf.String(), tc.want)
			}
		})
	}
}

func TestNoteToolWindow(t *testing.T) {
	now := time.Date(2026, 10, 9, 6, 0, 0, 0, time.UTC)
	pinClock(t, now)
	from, to := now.Add(-time.Hour).UnixMilli(), now.UnixMilli()
	cases := []struct {
		name, params, want string
	}{
		{"window", `{"expr":"x","execution":{"kind":"window","from_ms":` + itoa(from) + `,"to_ms":` + itoa(to) + `}}`,
			"note: window 2026-10-09T13:00:00+08:00..2026-10-09T14:00:00+08:00 (1h, ended now)\n"},
		{"instant", `{"expr":"up","execution":{"kind":"instant","to_ms":` + itoa(to) + `}}`,
			"note: instant 2026-10-09T14:00:00+08:00 (now)\n"},
		{"diagnostic tool, no execution", `{"table":"t"}`, ""},
		{"no params", ``, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			noteToolWindow(&buf, []byte(tc.params))
			if buf.String() != tc.want {
				t.Fatalf("got  %q\nwant %q", buf.String(), tc.want)
			}
		})
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// TestGeneratedWindowNote: a generated verb with a --start-time/--end-time
// pair announces the window when both flags are given. A window that came in
// through --data is the caller's literal body, not a value the CLI resolved,
// so it is not announced.
func TestGeneratedWindowNote(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"insight", "account", "--start-time", "2h", "--end-time", "1h"}, true},
		{[]string{"insight", "account", "--since", "2h", "--until", "now"}, true},
		{[]string{"insight", "account", "--data", `{"start_time":1791540000,"end_time":1791543600}`}, false},
	} {
		saveAndResetGlobals(t)
		newGFStub(t)
		_, stderr, err := execCommandSplit(append(tc.args, "--output-format", "json")...)
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		if got := strings.Contains(stderr, "note: window "); got != tc.want {
			t.Errorf("%v: window note = %v, want %v (stderr %q)", tc.args, got, tc.want, stderr)
		}
	}
}

// TestMonitQueryWindowNote: monit-query echoes the execution window of its
// params on stderr, leaving stdout untouched.
func TestMonitQueryWindowNote(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"datasource_id": 1, "tool": "prometheus.query", "data": map[string]any{}}
	stdout, stderr, err := execCommandSplit("monit-query", "1", "--tool", "prometheus.query",
		"--params", `{"expr":"up","execution":{"kind":"range","from_ms":1791540000000,"to_ms":1791543600000,"max_data_points":10}}`,
		"--output-format", "json")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if !strings.Contains(stderr, "note: window ") || !strings.Contains(stderr, "(1h, ") {
		t.Errorf("want a 1h window note, got %q", stderr)
	}
	if strings.Contains(stdout, "note:") {
		t.Errorf("stdout must stay pure, got %q", stdout)
	}
}

// TestOpenStateDefaultWindow: incident list with only open --progress states,
// and alert list --active, default to a 30d window when --since is not set;
// every other combination keeps the 24h default. The window is announced on
// stderr either way, and stdout stays pure JSON.
func TestOpenStateDefaultWindow(t *testing.T) {
	const day = int64(24 * 3600)
	cases := []struct {
		name     string
		args     []string
		wantSpan int64
		wantWhy  bool
	}{
		{"incident unfiltered", []string{"incident", "list"}, day, false},
		{"incident triggered", []string{"incident", "list", "--progress", "Triggered"}, 30 * day, true},
		{"incident open states", []string{"incident", "list", "--progress", "Triggered,Processing"}, 30 * day, true},
		{"incident closed", []string{"incident", "list", "--progress", "Closed"}, day, false},
		{"incident mixed", []string{"incident", "list", "--progress", "Triggered,Closed"}, day, false},
		{"incident explicit since wins", []string{"incident", "list", "--progress", "Triggered", "--since", "2h"}, 2 * 3600, false},
		{"alert active", []string{"alert", "list", "--active"}, 30 * day, true},
		{"alert unfiltered", []string{"alert", "list"}, day, false},
		{"alert active explicit since", []string{"alert", "list", "--active", "--since", "1h"}, 3600, false},
		// A defaulted --since spans back from --until, so a moved --until
		// neither stretches the window past 30d nor inverts it.
		{"incident triggered future until", []string{"incident", "list", "--progress", "Triggered", "--until", "+1d"}, 30 * day, true},
		{"alert active future until", []string{"alert", "list", "--active", "--until", "+1d"}, 30 * day, true},
		{"incident past until", []string{"incident", "list", "--until", "2d"}, day, false},
		{"incident explicit since and until", []string{"incident", "list", "--since", "3d", "--until", "1d"}, 2 * day, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			stub.data = map[string]any{"items": []any{}, "total": 0}
			stdout, stderr, err := execCommandSplit(append(tc.args, "--output-format", "json")...)
			if err != nil {
				t.Fatalf("execCommand: %v", err)
			}
			start, _ := stub.lastBody["start_time"].(float64)
			end, _ := stub.lastBody["end_time"].(float64)
			if span := int64(end - start); span < tc.wantSpan-5 || span > tc.wantSpan+5 {
				t.Errorf("window span = %ds, want %ds (body %v)", span, tc.wantSpan, stub.lastBody)
			}
			if !strings.Contains(stderr, "note: window ") {
				t.Errorf("stderr should announce the window, got %q", stderr)
			}
			if got := strings.Contains(stderr, "default --since 30d"); got != tc.wantWhy {
				t.Errorf("default reason in note = %v, want %v: %q", got, tc.wantWhy, stderr)
			}
			if strings.Contains(stdout, "note:") {
				t.Errorf("stdout must stay pure structured output, got %q", stdout)
			}
		})
	}
}
