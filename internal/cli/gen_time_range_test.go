package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// Fixed window used across these tests: 2026-01-02 14:00..15:00 UTC.
const (
	rangeSince   = "2026-01-02T14:00:00Z"
	rangeUntil   = "2026-01-02T15:00:00Z"
	rangeStartS  = int64(1767362400)
	rangeEndS    = int64(1767366000)
	rangeStartMS = rangeStartS * 1000
	rangeEndMS   = rangeEndS * 1000
)

var (
	topSecondsRange = timeRange{Start: "start_time", End: "end_time", Where: "--data or --start-time/--end-time"}
	topMillisRange  = timeRange{Start: "start_time", End: "end_time", Millis: true, Where: "--data or --start-time/--end-time"}
	nestedRange     = timeRange{Container: "time", Start: "from_ms", End: "to_ms", Millis: true, Where: "--data"}
)

func decodeRangeObj(t *testing.T, raw string) map[string]any {
	t.Helper()
	obj := map[string]any{}
	if raw == "" {
		return obj
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return obj
}

func TestApplyRange(t *testing.T) {
	cases := []struct {
		name         string
		r            timeRange
		obj          string
		since, until string
		setS, setU   bool
		want         map[string]int64 // dotted path → value
		absent       []string
		wantErr      string
	}{
		{name: "seconds, RFC3339 both ends", r: topSecondsRange, since: rangeSince, until: rangeUntil, setS: true, setU: true,
			want: map[string]int64{"start_time": rangeStartS, "end_time": rangeEndS}},
		{name: "milliseconds, RFC3339 both ends", r: topMillisRange, since: rangeSince, until: rangeUntil, setS: true, setU: true,
			want: map[string]int64{"start_time": rangeStartMS, "end_time": rangeEndMS}},
		{name: "seconds field from Unix ms input", r: topSecondsRange, since: "1767362400000", until: "1767366000", setS: true, setU: true,
			want: map[string]int64{"start_time": rangeStartS, "end_time": rangeEndS}},
		{name: "milliseconds field from Unix seconds input", r: topMillisRange, since: "1767362400", until: "1767366000000", setS: true, setU: true,
			want: map[string]int64{"start_time": rangeStartMS, "end_time": rangeEndMS}},
		{name: "nested, container created", r: nestedRange, obj: `{"dashboard_id":"d1"}`, since: rangeSince, until: rangeUntil, setS: true, setU: true,
			want: map[string]int64{"time.from_ms": rangeStartMS, "time.to_ms": rangeEndMS}},
		{name: "nested, siblings kept", r: nestedRange, obj: `{"time":{"kind":"range"}}`, since: rangeSince, until: rangeUntil, setS: true, setU: true,
			want: map[string]int64{"time.from_ms": rangeStartMS, "time.to_ms": rangeEndMS}},
		{name: "until alone sets only the end", r: topSecondsRange, until: rangeUntil, setU: true,
			want: map[string]int64{"end_time": rangeEndS}, absent: []string{"start_time"}},
		{name: "since alone keeps an end already set", r: topSecondsRange, obj: `{"end_time":1767366000}`, since: rangeSince, setS: true,
			want: map[string]int64{"start_time": rangeStartS, "end_time": rangeEndS}},
		{name: "since conflicts with start set elsewhere", r: topSecondsRange, obj: `{"start_time":1767362400}`, since: "1h", setS: true,
			wantErr: "--since conflicts with start_time in --data or --start-time/--end-time"},
		{name: "until conflicts with nested end in --data", r: nestedRange, obj: `{"time":{"to_ms":1767366000000}}`, until: "now", setU: true,
			wantErr: "--until conflicts with time.to_ms in --data"},
		{name: "empty since", r: topSecondsRange, since: " ", setS: true, wantErr: "invalid --since: empty value"},
		{name: "empty until", r: topMillisRange, until: "", setU: true, wantErr: "invalid --until: empty value"},
		{name: "invalid until", r: topSecondsRange, until: "soon", setU: true, wantErr: "invalid --until"},
		{name: "inverted window", r: topMillisRange, since: rangeUntil, until: rangeSince, setS: true, setU: true, wantErr: "inverted window"},
		{name: "since equal to until", r: topSecondsRange, since: rangeSince, until: rangeSince, setS: true, setU: true, wantErr: "inverted window"},
		{name: "container not an object", r: nestedRange, obj: `{"time":"1h"}`, since: "1h", setS: true, wantErr: "--data time must be an object"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			obj := decodeRangeObj(t, c.obj)
			err := applyRange(obj, c.r, c.since, c.until, c.setS, c.setU)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("want error containing %q, got %v", c.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for path, want := range c.want {
				got, ok := rangeValue(lookupPath(obj, path))
				if !ok || got != want {
					t.Errorf("%s = %v, want %d (obj %v)", path, lookupPath(obj, path), want, obj)
				}
			}
			for _, path := range c.absent {
				if v := lookupPath(obj, path); v != nil {
					t.Errorf("%s set to %v, want absent", path, v)
				}
			}
		})
	}
}

// TestApplyRangeRelative proves the duration forms resolve against now, and
// that --since alone ends the window now, in the field's unit.
func TestApplyRangeRelative(t *testing.T) {
	for _, r := range []timeRange{topSecondsRange, topMillisRange, nestedRange} {
		obj := map[string]any{}
		if err := applyRange(obj, r, "2h", "", true, false); err != nil {
			t.Fatalf("%+v: %v", r, err)
		}
		fields := obj
		if r.Container != "" {
			fields = obj[r.Container].(map[string]any)
		}
		scale := int64(1)
		if r.Millis {
			scale = 1000
		}
		now := time.Now().Unix()
		start, _ := rangeValue(fields[r.Start])
		end, _ := rangeValue(fields[r.End])
		if d := now - 2*3600 - start/scale; d < -5 || d > 5 {
			t.Errorf("%+v: start %d not ~2h ago", r, start)
		}
		if d := now - end/scale; d < -5 || d > 5 {
			t.Errorf("%+v: end %d not ~now", r, end)
		}
	}
}

func lookupPath(obj map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var cur any = obj
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[p]
	}
	return cur
}

// TestCommandTimeRangeMappings drives one command per recognized field shape
// and asserts --since/--until reach the right wire fields in the right unit.
func TestCommandTimeRangeMappings(t *testing.T) {
	cases := []struct {
		name string
		args []string
		path string
		want map[string]float64
	}{
		{"seconds start_time/end_time", []string{"insight", "account"}, "/insight/account",
			map[string]float64{"start_time": float64(rangeStartS), "end_time": float64(rangeEndS)}},
		{"seconds start/end", []string{"schedule", "info", "42"}, "/schedule/info",
			map[string]float64{"start": float64(rangeStartS), "end": float64(rangeEndS)}},
		{"seconds created_at_*_seconds", []string{"incident", "post-mortem-list"}, "/incident/post-mortem/list",
			map[string]float64{"created_at_start_seconds": float64(rangeStartS), "created_at_end_seconds": float64(rangeEndS)}},
		{"milliseconds start_time/end_time", []string{"rum", "issue-list"}, "/rum/issue/list",
			map[string]float64{"start_time": float64(rangeStartMS), "end_time": float64(rangeEndMS)}},
		{"milliseconds started_after/before_ms", []string{"safari", "automation-run-list", "--rule-id", "r1"}, "/safari/automation/run/list",
			map[string]float64{"started_after_ms": float64(rangeStartMS), "started_before_ms": float64(rangeEndMS)}},
		{"nested time.from_ms/to_ms", []string{"monit", "dashboard-panel-run", "--data", `{"dashboard_id":"d1","panel_id":"p1"}`}, "/monit/dashboard/panel/run",
			map[string]float64{"time.from_ms": float64(rangeStartMS), "time.to_ms": float64(rangeEndMS)}},
		{"nested execution.from_ms/to_ms", []string{"monit", "query-explore", "7", "--data", `{"expr":"up","args":{},"execution":{"kind":"range","max_data_points":10}}`}, "/monit/query/explore",
			map[string]float64{"execution.from_ms": float64(rangeStartMS), "execution.to_ms": float64(rangeEndMS)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			args := append(append([]string{}, c.args...), "--since", rangeSince, "--until", rangeUntil)
			if _, err := execCommand(args...); err != nil {
				t.Fatalf("%v: %v", args, err)
			}
			if stub.lastPath != c.path {
				t.Fatalf("path = %q, want %q", stub.lastPath, c.path)
			}
			for path, want := range c.want {
				if got := lookupPath(stub.lastBody, path); got != want {
					t.Errorf("%s = %v, want %v (body %v)", path, got, want, stub.lastBody)
				}
			}
		})
	}
	// A sibling the window does not own survives the nested write.
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	if _, err := execCommand("monit", "query-explore", "7", "--data", `{"expr":"up","args":{},"execution":{"kind":"range","max_data_points":10}}`, "--since", "1h"); err != nil {
		t.Fatal(err)
	}
	if kind := lookupPath(stub.lastBody, "execution.kind"); kind != "range" {
		t.Errorf("execution.kind = %v, want range", kind)
	}
}

// TestCommandTimeRangeWithShortIncidentID: a generated incident command with
// both an incident id and a window resolves the short id and writes the
// window in the same request.
func TestCommandTimeRangeWithShortIncidentID(t *testing.T) {
	saveAndResetGlobals(t)
	paths, bodies := shortIDStub(t, incidentItem(testFullID, testShortID, "High error rate"))
	if _, err := execCommand("incident", "sdp-request-list", "--incident-id", testShortID, "--since", rangeSince, "--until", rangeUntil); err != nil {
		t.Fatal(err)
	}
	if len(*paths) != 2 || (*paths)[0] != "/incident/list" || (*paths)[1] != "/incident/sdp/request/list" {
		t.Fatalf("paths = %v, want /incident/list then /incident/sdp/request/list", *paths)
	}
	body := (*bodies)[1]
	if body["incident_id"] != testFullID || body["start_time"] != float64(rangeStartS) || body["end_time"] != float64(rangeEndS) {
		t.Fatalf("body = %v, want resolved incident_id and the window", body)
	}
}

// TestCommandInsightIncidentExportRange covers the curated export, which
// shares the generated verbs' window flags.
func TestCommandInsightIncidentExportRange(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newInsightExportStub(t, "incident_id,title\n", 0)
	if _, err := execCommand("insight", "incident-export", "--since", rangeSince, "--until", rangeUntil); err != nil {
		t.Fatal(err)
	}
	if stub.exportBody["start_time"] != float64(rangeStartS) || stub.exportBody["end_time"] != float64(rangeEndS) {
		t.Fatalf("export body = %v", stub.exportBody)
	}
	// The relative forms now reach --start-time/--end-time too.
	if _, err := execCommand("insight", "incident-export", "--start-time", "7d", "--end-time", "now"); err != nil {
		t.Fatal(err)
	}
	start, _ := stub.exportBody["start_time"].(float64)
	if d := float64(time.Now().Unix()-7*86400) - start; d < -10 || d > 10 {
		t.Errorf("start_time %v not ~7d ago", start)
	}
}

// TestCommandTimeRangeConflicts proves a window end set in two places fails
// before any request, whichever other place it is.
func TestCommandTimeRangeConflicts(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"since vs canonical flag", []string{"insight", "account", "--since", "7d", "--start-time", "30d", "--until", "now"},
			"--since conflicts with start_time in --data or --start-time/--end-time"},
		{"since vs canonical flag, same value", []string{"insight", "account", "--since", "7d", "--start-time", "7d"},
			"--since conflicts with start_time"},
		{"until vs --data (ms)", []string{"rum", "issue-list", "--data", `{"end_time":1767366000000}`, "--until", "now"},
			"--until conflicts with end_time in --data or --start-time/--end-time"},
		{"since vs nested --data", []string{"monit", "dashboard-panel-run", "--data", `{"dashboard_id":"d1","panel_id":"p1","time":{"from_ms":1767362400000}}`, "--since", "1h"},
			"--since conflicts with time.from_ms in --data"},
		{"empty since", []string{"schedule", "info", "42", "--since", ""}, "invalid --since: empty value"},
		{"inverted", []string{"rum", "issue-list", "--since", rangeUntil, "--until", rangeSince}, "inverted window"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			_, err := execCommand(c.args...)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want error containing %q, got %v", c.want, err)
			}
			if stub.requests != 0 {
				t.Fatalf("conflict must fail before any request, got %d", stub.requests)
			}
		})
	}
}

// TestCommandTimeRangeCanonicalFlagsUnchanged proves the existing ways of
// setting the fields still reach the wire as before.
func TestCommandTimeRangeCanonicalFlagsUnchanged(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	if _, err := execCommand("rum", "issue-list", "--start-time", "1767362400123", "--end-time", "1767366000456"); err != nil {
		t.Fatal(err)
	}
	if stub.lastBody["start_time"] != float64(1767362400123) || stub.lastBody["end_time"] != float64(1767366000456) {
		t.Fatalf("ms flags lost precision: %v", stub.lastBody)
	}
	if _, err := execCommand("insight", "account", "--data", `{"start_time":1767362400,"end_time":1767366000}`); err != nil {
		t.Fatal(err)
	}
	if stub.lastBody["start_time"] != float64(rangeStartS) || stub.lastBody["end_time"] != float64(rangeEndS) {
		t.Fatalf("--data window lost: %v", stub.lastBody)
	}
}

// TestCommandTimeRangeNote proves the resolved window is announced on stderr
// in local time, keeping stdout to the result.
func TestCommandTimeRangeNote(t *testing.T) {
	saveAndResetGlobals(t)
	newGFStub(t)
	stdout, stderr, err := execCommandSplit("rum", "issue-list", "--since", rangeSince, "--until", rangeUntil, "--output-format", "json")
	if err != nil {
		t.Fatal(err)
	}
	want := "note: window " + formatLocal(time.Unix(rangeStartS, 0)) + ".." + formatLocal(time.Unix(rangeEndS, 0)) + " (1h, "
	if !strings.Contains(stderr, want) {
		t.Errorf("stderr %q lacks %q", stderr, want)
	}
	if strings.Contains(stdout, "note: window") {
		t.Errorf("window note leaked to stdout: %q", stdout)
	}
}

// TestTimeRangeTreeConsistency asserts the invariant across the whole tree:
// a command with a --start-time/--end-time pair also takes --since/--until,
// and every generated --since names the field it writes.
func TestTimeRangeTreeConsistency(t *testing.T) {
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		fl := cmd.Flags()
		if fl.Lookup("start-time") != nil && fl.Lookup("end-time") != nil &&
			(fl.Lookup("since") == nil || fl.Lookup("until") == nil) {
			t.Errorf("%s has --start-time/--end-time but no --since/--until", cmd.CommandPath())
		}
		if f := fl.Lookup("since"); f != nil && strings.HasPrefix(f.Usage, "Window start, written to ") {
			if !strings.Contains(f.Usage, "(Unix seconds)") && !strings.Contains(f.Usage, "(Unix milliseconds)") {
				t.Errorf("%s --since usage names no unit: %q", cmd.CommandPath(), f.Usage)
			}
		}
		for _, sub := range cmd.Commands() {
			walk(sub)
		}
	}
	walk(rootCmd)
}
