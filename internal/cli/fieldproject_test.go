package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

// incidentRow / alertRow are multi-field stub payloads with the nested blobs
// (responders/labels/alerts, events/incident/labels) that bloat the full dump.
// The SDK structs carry no `omitempty`, so the full toon/json marshal always
// emits every key — which is exactly what the regression tests assert stays put.
func incidentRow() map[string]any {
	return map[string]any{
		"incident_id":       "inc-1",
		"title":             "Disk full on db-01",
		"incident_severity": "Critical",
		"progress":          "Triggered",
		"start_time":        1712000000,
		"channel_id":        12345,
		"description":       "root volume at 98%",
		"labels":            map[string]any{"service": "db", "env": "prod"},
		"responders": []map[string]any{
			{"person_id": 101, "person_name": "Alice"},
		},
	}
}

func TestBoundProjectedOutputCapsStructuredFormats(t *testing.T) {
	for _, format := range []string{"json", "toon"} {
		t.Run(format, func(t *testing.T) {
			saveAndResetGlobals(t)
			flagOutputFormat = format
			rows := []map[string]any{{
				"incident_id": "inc-1",
				"title":       strings.Repeat("数据库故障", 2000),
			}}

			if _, err := boundProjectedOutput(rows, 512); err != nil {
				t.Fatalf("bound projected output: %v", err)
			}
			encoded, err := marshalStructured(rows)
			if err != nil {
				t.Fatalf("marshal bounded output: %v", err)
			}
			if len(encoded)+1 >= 512 {
				t.Fatalf("bounded %s output is %d bytes, want <512", format, len(encoded)+1)
			}
			title := rows[0]["title"].(string)
			if !utf8.ValidString(title) || !strings.HasSuffix(title, "...") {
				t.Fatalf("truncated title = %q, want valid UTF-8 with marker", title)
			}
		})
	}
}

func TestBoundProjectedOutputRejectsIrreducibleMetadata(t *testing.T) {
	saveAndResetGlobals(t)
	flagOutputFormat = "json"
	rows := make([]map[string]any, 200)
	for i := range rows {
		rows[i] = map[string]any{"count": i}
	}

	_, err := boundProjectedOutput(rows, 512)
	if err == nil || !strings.Contains(err.Error(), "request fewer rows") {
		t.Fatalf("irreducible output error = %v, want bounded guidance", err)
	}
}

// TestBoundProjectedOutputDetailWithinBudgetLeavesValuesUnchanged covers the
// single-object (map[string]any) shape used by `incident detail --fields`:
// when the projection already fits, every value must come back byte-for-byte
// identical to what went in.
func TestBoundProjectedOutputDetailWithinBudgetLeavesValuesUnchanged(t *testing.T) {
	saveAndResetGlobals(t)
	flagOutputFormat = "json"
	row := map[string]any{
		"incident_id": "inc-1",
		"title":       "Disk full on db-01",
		"progress":    "Triggered",
	}
	want := map[string]any{
		"incident_id": "inc-1",
		"title":       "Disk full on db-01",
		"progress":    "Triggered",
	}

	if _, err := boundProjectedOutput(row, compactDetailOutputLimit); err != nil {
		t.Fatalf("bound projected output: %v", err)
	}
	if !reflect.DeepEqual(row, want) {
		t.Fatalf("in-budget detail projection was modified: got %v, want %v", row, want)
	}
}

// TestBoundProjectedOutputDetailOversizedErrorsWithoutMutating is the
// regression guard for the truncation bug: a single-object projection that
// doesn't fit the budget must fail loudly instead of silently shipping
// truncated (and indistinguishable-from-real) values. The input map must
// come back completely untouched.
func TestBoundProjectedOutputDetailOversizedErrorsWithoutMutating(t *testing.T) {
	saveAndResetGlobals(t)
	flagOutputFormat = "json"
	row := map[string]any{
		"incident_id": "inc-1",
		"title":       strings.Repeat("数据库故障", 5000),
		"root_cause":  strings.Repeat("disk exhaustion details ", 3000),
	}
	want := map[string]any{
		"incident_id": "inc-1",
		"title":       strings.Repeat("数据库故障", 5000),
		"root_cause":  strings.Repeat("disk exhaustion details ", 3000),
	}

	_, err := boundProjectedOutput(row, 512)
	if err == nil {
		t.Fatal("expected an error for an oversized detail projection, got nil")
	}
	if !strings.Contains(err.Error(), "512") {
		t.Errorf("error should name the byte budget (512), got: %v", err)
	}
	if !strings.Contains(err.Error(), "title") && !strings.Contains(err.Error(), "root_cause") {
		t.Errorf("error should name the largest field, got: %v", err)
	}
	if !reflect.DeepEqual(row, want) {
		t.Fatalf("oversized detail projection mutated the input map: got %v, want %v", row, want)
	}
}

// TestBoundProjectedOutputDetailErrorIsDeterministic pins the tie-break in the
// largest-field ranking: with several fields at exactly the same encoded size,
// Go's randomized map iteration order must not leak into the error text, or the
// same failing command would name different fields on each run.
func TestBoundProjectedOutputDetailErrorIsDeterministic(t *testing.T) {
	saveAndResetGlobals(t)
	flagOutputFormat = "json"

	first := ""
	for i := range 20 {
		row := map[string]any{
			"alpha":   strings.Repeat("a", 400),
			"bravo":   strings.Repeat("b", 400),
			"charlie": strings.Repeat("c", 400),
			"delta":   strings.Repeat("d", 400),
			"echo":    strings.Repeat("e", 400),
		}
		_, err := boundProjectedOutput(row, 512)
		if err == nil {
			t.Fatal("expected an error for an oversized detail projection, got nil")
		}
		if i == 0 {
			first = err.Error()
			continue
		}
		if err.Error() != first {
			t.Fatalf("error text varies between runs on equal-sized fields:\n run 0: %s\n run %d: %s", first, i, err.Error())
		}
	}
	if !strings.Contains(first, "alpha") {
		t.Errorf("equal-sized fields should be ranked by name, expected alpha first, got: %s", first)
	}
}

func alertRow() map[string]any {
	return map[string]any{
		"alert_id":       "al-1",
		"title":          "High CPU on web-02",
		"alert_severity": "Warning",
		"alert_status":   "Triggered",
		"created_at":     1712000000,
		"description":    "cpu > 90% for 5m",
		"labels":         map[string]any{"host": "web-02"},
		"events": []map[string]any{
			{"event_id": "ev-1", "event_severity": "Warning"},
		},
		"incident": map[string]any{"incident_id": "inc-9", "progress": "Processing"},
	}
}

// TestIncidentListStructuredDefaultUsesCompactProjection is the default agent
// path: incident list in json/toon mode must not dump the full nested SDK row
// when --fields is omitted, while an explicit --fields still wins.
func TestIncidentListStructuredDefaultUsesCompactProjection(t *testing.T) {
	for _, format := range []string{"json", "toon"} {
		t.Run(format+" long title stays bounded", func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			row := incidentRow()
			row["title"] = strings.Repeat("数据库故障", 5000)
			stub.data = map[string]any{"items": []any{row}, "total": 1}

			out, stderrText, err := execCommandSplit("incident", "list", "--output-format", format)
			if err != nil {
				t.Fatalf("execCommandSplit: %v", err)
			}
			if len([]byte(out)) >= compactListOutputLimit {
				t.Fatalf("bounded %s incident list is %d bytes, want <%d", format, len([]byte(out)), compactListOutputLimit)
			}
			if !utf8.ValidString(out) || !strings.Contains(out, "...") {
				t.Fatalf("bounded %s incident list must retain valid UTF-8 and show truncation", format)
			}
			// The clipped value must be announced, not just marked: a --json
			// consumer filters on the value and never sees the "..." itself.
			if !strings.Contains(stderrText, "were shortened to fit") || !strings.Contains(stderrText, "title") {
				t.Errorf("shortened %s incident list should announce the clipped field on stderr, got:\n%s", format, stderrText)
			}
		})
	}

	t.Run("json default", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{incidentRow()}, "total": 1}

		out, stderrText, err := execCommandSplit("incident", "list", "--output-format", "json")
		if err != nil {
			t.Fatalf("execCommandSplit: %v", err)
		}

		assertProjectedJSONFields(t, out, []string{"incident_id", "title", "incident_severity", "progress", "start_time", "channel_id"})
		if !strings.Contains(stderrText, "note: rows projected to default compact fields") {
			t.Errorf("default projection should announce itself on stderr, got:\n%s", stderrText)
		}
	})

	t.Run("toon default", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{incidentRow()}, "total": 1}

		out, stderrText, err := execCommandSplit("incident", "list", "--output-format", "toon")
		if err != nil {
			t.Fatalf("execCommandSplit: %v", err)
		}

		// Positive keys must come from stdout alone: the stderr note embeds the
		// same field names, so a merged capture would satisfy this vacuously.
		for _, key := range []string{"incident_id", "title", "incident_severity", "progress", "start_time", "channel_id"} {
			if !strings.Contains(out, key) {
				t.Errorf("default toon output missing compact key %q, got:\n%s", key, out)
			}
		}
		if !strings.Contains(stderrText, "note: rows projected to default compact fields") {
			t.Errorf("default projection should announce itself on stderr, got:\n%s", stderrText)
		}
		for _, key := range []string{"responders", "labels", "description"} {
			if strings.Contains(out, key) {
				t.Errorf("default toon output should not contain full-record key %q, got:\n%s", key, out)
			}
		}
	})

	t.Run("explicit fields win", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{incidentRow()}, "total": 1}

		out, err := execCommand("incident", "list", "--fields", "incident_id,title", "--output-format", "json")
		if err != nil {
			t.Fatalf("execCommand: %v", err)
		}

		assertProjectedJSONFields(t, out, []string{"incident_id", "title"})
	})

	t.Run("explicit empty fields errors", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{incidentRow()}, "total": 1}

		_, err := execCommand("incident", "list", "--fields", "", "--output-format", "json")
		if err == nil {
			t.Fatal("expected an error for empty --fields, got nil")
		}
		if !strings.Contains(err.Error(), "--fields") {
			t.Errorf("error should name --fields, got: %v", err)
		}
	})
}

// TestAlertFieldsProjectionDefaultUnchanged is the conductor constraint for the
// sibling command: with NO --fields, alert list structured output still emits
// the full nested record. The compact default is incident-list-only.
func TestAlertFieldsProjectionDefaultUnchanged(t *testing.T) {
	cases := []struct {
		name     string
		cmd      []string
		data     map[string]any
		format   string
		mustHave []string // nested keys that must survive in the full dump
	}{
		{"alert toon", []string{"alert", "list"}, alertRow(), "toon", []string{"events", "incident", "labels", "description"}},
		{"alert json", []string{"alert", "list"}, alertRow(), "json", []string{"events", "incident", "labels", "description"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			stub.data = map[string]any{"items": []any{tc.data}, "total": 1}

			args := append(append([]string(nil), tc.cmd...), "--output-format", tc.format)
			out, err := execCommand(args...)
			if err != nil {
				t.Fatalf("execCommand: %v", err)
			}
			for _, key := range tc.mustHave {
				if !strings.Contains(out, key) {
					t.Errorf("default %s output should contain full-record key %q (shape must be unchanged), got:\n%s", tc.format, key, out)
				}
			}
		})
	}
}

func assertProjectedJSONFields(t *testing.T, out string, fields []string) {
	t.Helper()

	var rows []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &rows); err != nil {
		t.Fatalf("failed to parse projected json: %v\nraw:\n%s", err, out)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 projected row, got %d:\n%s", len(rows), out)
	}
	row := rows[0]
	if len(row) != len(fields) {
		t.Fatalf("expected exactly %d keys, got %d (%v)", len(fields), len(row), row)
	}
	for _, f := range fields {
		if _, ok := row[f]; !ok {
			t.Errorf("projected row missing key %q, got keys %v", f, row)
		}
	}
}

// TestFieldsProjectionTOON: --fields in toon mode emits exactly the requested
// keys and drops everything else.
func TestFieldsProjectionTOON(t *testing.T) {
	cases := []struct {
		name    string
		cmd     []string
		data    map[string]any
		fields  string
		want    []string
		dropped []string
	}{
		{
			name:    "alert",
			cmd:     []string{"alert", "list"},
			data:    alertRow(),
			fields:  "alert_id,title,alert_severity,created_at",
			want:    []string{"alert_id", "title", "alert_severity", "created_at"},
			dropped: []string{"labels", "events", "description", "incident"},
		},
		{
			name:    "incident",
			cmd:     []string{"incident", "list"},
			data:    incidentRow(),
			fields:  "incident_id,title,incident_severity,progress,start_time",
			want:    []string{"incident_id", "title", "incident_severity", "progress", "start_time"},
			dropped: []string{"responders", "labels", "description"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			stub.data = map[string]any{"items": []any{tc.data}, "total": 1}

			args := append(append([]string(nil), tc.cmd...), "--fields", tc.fields, "--output-format", "toon")
			out, err := execCommand(args...)
			if err != nil {
				t.Fatalf("execCommand: %v", err)
			}
			for _, key := range tc.want {
				if !strings.Contains(out, key) {
					t.Errorf("projected toon output missing requested key %q, got:\n%s", key, out)
				}
			}
			for _, key := range tc.dropped {
				if strings.Contains(out, key) {
					t.Errorf("projected toon output should not contain dropped key %q, got:\n%s", key, out)
				}
			}
		})
	}
}

// TestFieldsProjectionJSON: --fields in json mode yields rows with EXACTLY the
// requested keys (asserted structurally via json.Unmarshal).
func TestFieldsProjectionJSON(t *testing.T) {
	cases := []struct {
		name   string
		cmd    []string
		data   map[string]any
		fields []string
	}{
		{"alert", []string{"alert", "list"}, alertRow(), []string{"alert_id", "title", "alert_severity", "created_at"}},
		{"incident", []string{"incident", "list"}, incidentRow(), []string{"incident_id", "title", "incident_severity", "progress", "start_time"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			stub.data = map[string]any{"items": []any{tc.data}, "total": 1}

			args := append(append([]string(nil), tc.cmd...), "--fields", strings.Join(tc.fields, ","), "--output-format", "json")
			out, err := execCommand(args...)
			if err != nil {
				t.Fatalf("execCommand: %v", err)
			}

			assertProjectedJSONFields(t, out, tc.fields)
		})
	}
}

// TestFieldsIgnoredInTableMode: --fields is a no-op in the default table view —
// the normal column header is still printed.
func TestFieldsIgnoredInTableMode(t *testing.T) {
	cases := []struct {
		name    string
		cmd     []string
		data    map[string]any
		fields  string
		headers []string
	}{
		{"alert", []string{"alert", "list"}, alertRow(), "alert_id", []string{"ID", "TITLE", "SEVERITY", "STATUS"}},
		{"incident", []string{"incident", "list"}, incidentRow(), "incident_id", []string{"ID", "TITLE", "SEVERITY", "PROGRESS"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			stub.data = map[string]any{"items": []any{tc.data}, "total": 1}

			args := append(append([]string(nil), tc.cmd...), "--fields", tc.fields)
			out, err := execCommand(args...)
			if err != nil {
				t.Fatalf("execCommand: %v", err)
			}
			for _, h := range tc.headers {
				if !strings.Contains(out, h) {
					t.Errorf("table output should contain header %q (--fields is a no-op in table mode), got:\n%s", h, out)
				}
			}
		})
	}
}

// TestFieldsUnknownFieldErrors: a bad field name fails fast with the offending
// name in the message.
func TestFieldsUnknownFieldErrors(t *testing.T) {
	cases := []struct {
		name string
		cmd  []string
		data map[string]any
	}{
		{"alert", []string{"alert", "list"}, alertRow()},
		{"incident", []string{"incident", "list"}, incidentRow()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			stub.data = map[string]any{"items": []any{tc.data}, "total": 1}

			args := append(append([]string(nil), tc.cmd...), "--fields", "not_a_field", "--output-format", "json")
			_, err := execCommand(args...)
			if err == nil {
				t.Fatal("expected an error for an unknown field, got nil")
			}
			if !strings.Contains(err.Error(), "not_a_field") {
				t.Errorf("error should name the bad field %q, got: %v", "not_a_field", err)
			}
		})
	}
}

func TestIncidentSimilarStructuredProjection(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	items := make([]any, 20)
	for i := range items {
		row := incidentRow()
		row["incident_id"] = "similar-" + string(rune('a'+i))
		row["close_time"] = 1712000060
		row["ack_time"] = 1712000030
		row["alert_cnt"] = 3
		row["title"] = strings.Repeat("very long incident title ", 2000)
		row["root_cause"] = strings.Repeat("disk exhaustion details ", 2000)
		row["score"] = 0.9
		row["description"] = strings.Repeat("large description ", 100)
		row["images"] = []map[string]any{{"src": strings.Repeat("https://example.test/image/", 100)}}
		items[i] = row
	}
	stub.data = map[string]any{"items": items, "total": len(items)}

	out, stderrText, err := execCommandSplit("incident", "similar", "inc-1", "--limit", "20", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	if len(out) >= 16*1024 {
		t.Fatalf("compact similar output is %d bytes, want <16 KiB", len(out))
	}
	if !strings.Contains(stderrText, "note: rows projected to default compact fields") {
		t.Errorf("default projection should announce itself on stderr, got:\n%s", stderrText)
	}

	var rows []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &rows); err != nil {
		t.Fatalf("parse compact similar json: %v\n%s", err, out)
	}
	want := []string{"incident_id", "title", "incident_severity", "progress", "start_time", "close_time", "ack_time", "alert_cnt", "root_cause", "score"}
	if len(rows) != len(items) {
		t.Fatalf("got %d rows, want %d", len(rows), len(items))
	}
	for _, row := range rows {
		if len(row) != len(want) {
			t.Fatalf("got keys %v, want exactly %v", row, want)
		}
		for _, field := range want {
			if _, ok := row[field]; !ok {
				t.Errorf("projected row missing %q", field)
			}
		}
		if _, ok := row["description"]; ok {
			t.Errorf("projected row includes description: %v", row)
		}
	}

}

func TestIncidentDetailFieldsProjection(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	row := incidentRow()
	row["description"] = "root volume at 98%, mount point /var/lib/db"
	row["images"] = []map[string]any{{"src": "https://example.test/image/1.png"}}
	row["ai_summary"] = "Disk usage crossed 98% on db-01 at 03:14 UTC after log rotation stopped."
	row["root_cause"] = "Log rotation was disabled by the previous deploy's config change."
	row["resolution"] = "Re-enabled log rotation and cleared the stale archive files."
	stub.data = row

	fields := []string{"incident_id", "title", "incident_severity", "progress", "ai_summary", "root_cause", "resolution", "alert_cnt", "start_time", "channel_id"}
	out, err := execCommand("incident", "detail", "inc-1", "--fields", strings.Join(fields, ","), "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if len(out) >= 8*1024 {
		t.Fatalf("projected detail output is %d bytes, want <8 KiB", len(out))
	}
	var detail map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &detail); err != nil {
		t.Fatalf("parse projected detail json: %v\n%s", err, out)
	}
	if len(detail) != len(fields) {
		t.Fatalf("projected detail keys = %v, want exactly %v", detail, fields)
	}
	for _, field := range fields {
		if detail[field] == nil {
			t.Errorf("projected detail missing %q: %v", field, detail)
		}
	}
	if _, ok := detail["description"]; ok {
		t.Errorf("projected detail includes description: %v", detail)
	}

	// The projected values must come back byte-for-byte identical to the
	// source row — this command must never truncate a detail value.
	var gotSummary string
	if err := json.Unmarshal(detail["ai_summary"], &gotSummary); err != nil {
		t.Fatalf("unmarshal ai_summary: %v", err)
	}
	if gotSummary != row["ai_summary"] {
		t.Fatalf("ai_summary was altered: got %q, want %q", gotSummary, row["ai_summary"])
	}
}

// TestIncidentDetailFieldsProjectionOversizedErrors: when the requested
// --fields don't fit the 8 KiB detail budget, the command must fail with an
// actionable error instead of silently shipping truncated values.
func TestIncidentDetailFieldsProjectionOversizedErrors(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	row := incidentRow()
	row["ai_summary"] = strings.Repeat("long AI summary ", 4000)
	row["root_cause"] = strings.Repeat("long root cause ", 4000)
	row["resolution"] = strings.Repeat("long resolution ", 4000)
	stub.data = row

	fields := []string{"incident_id", "title", "ai_summary", "root_cause", "resolution"}
	_, err := execCommand("incident", "detail", "inc-1", "--fields", strings.Join(fields, ","), "--output-format", "json")
	if err == nil {
		t.Fatal("expected an error for an oversized detail projection, got nil")
	}
	if !strings.Contains(err.Error(), "8192") {
		t.Errorf("error should name the byte budget (8192), got: %v", err)
	}
	if !strings.Contains(err.Error(), "ai_summary") && !strings.Contains(err.Error(), "root_cause") && !strings.Contains(err.Error(), "resolution") {
		t.Errorf("error should name one of the largest fields, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--fields") {
		t.Errorf("error should point at --fields as the remedy, got: %v", err)
	}
}

func TestAlertEventListStructuredProjection(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	items := make([]any, 30)
	for i := range items {
		items[i] = map[string]any{
			"event_id":       "event-" + string(rune('a'+i)),
			"alert_id":       "alert-1",
			"event_severity": "Warning",
			"event_status":   "Triggered",
			"event_time":     1712000000,
			"title":          strings.Repeat("very long alert event title ", 2000),
			"description":    strings.Repeat("large description ", 100),
			"labels":         map[string]any{"host": "web-01"},
			"images":         []map[string]any{{"src": strings.Repeat("https://example.test/image/", 100)}},
		}
	}
	stub.data = map[string]any{"items": items, "total": len(items)}

	out, stderrText, err := execCommandSplit("alert-event", "list", "--limit", "30", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	if len(out) >= 16*1024 {
		t.Fatalf("compact alert-event output is %d bytes, want <16 KiB", len(out))
	}
	if !strings.Contains(stderrText, "note: rows projected to default compact fields") {
		t.Errorf("default projection should announce itself on stderr, got:\n%s", stderrText)
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &rows); err != nil {
		t.Fatalf("parse compact alert-event json: %v\n%s", err, out)
	}
	want := []string{"event_id", "alert_id", "event_severity", "event_status", "event_time", "title"}
	if len(rows) != len(items) {
		t.Fatalf("got %d rows, want %d", len(rows), len(items))
	}
	for _, row := range rows {
		if len(row) != len(want) {
			t.Fatalf("got keys %v, want exactly %v", row, want)
		}
		for _, field := range want {
			if _, ok := row[field]; !ok {
				t.Errorf("projected row missing %q", field)
			}
		}
	}

}

// alertEventOutlierFixture builds n alert-event rows with long (Mongo
// ObjectID-shaped) ids: the first `outliers` rows carry a pathologically long
// multi-word/CJK title (the field actually responsible for a page overflowing
// the compact-list budget), the rest carry short, realistic titles. It
// returns the row list alongside the exact ids and titles a correct
// projection must preserve or shorten.
func alertEventOutlierFixture(n, outliers int) (items []any, ids, shortTitles []string) {
	longTitle := strings.Repeat("K8S pod tcp 接收队列大于2000 / cluster-prod-a / node-17 / namespace kube-system / pod coredns-7db6d8ff4d-abcde ", 40)
	normalTitles := []string{
		"ERROR Detected / VMLogs-Prod",
		"CPU利用率较高 / fc-n9e-plus-18001",
		"服务器 dev-flasheye-01 连续飘红",
		"Disk usage high / db-02",
	}

	items = make([]any, n)
	ids = make([]string, 0, n*2)
	for i := range items {
		eventID := fmt.Sprintf("%024x", i)
		alertID := fmt.Sprintf("%024x", i+1_000_000)
		ids = append(ids, eventID, alertID)

		title := normalTitles[i%len(normalTitles)]
		if i >= outliers {
			shortTitles = append(shortTitles, title)
		} else {
			title = fmt.Sprintf("%s (row %d)", longTitle, i)
		}

		items[i] = map[string]any{
			"event_id":       eventID,
			"alert_id":       alertID,
			"event_severity": "Warning",
			"event_status":   "Triggered",
			"event_time":     1712000000 + i,
			"title":          title,
		}
	}
	return items, ids, shortTitles
}

// TestAlertEventListDefaultProjectionPreservesShortFields is the regression
// guard for the original defect: a minority of pathologically long titles
// pushing a page over the 16 KiB compact-list budget must never mangle the
// other rows' long (Mongo ObjectID-shaped) ids, or the short title rows on
// the same page — the shortening must land entirely on the field(s) actually
// responsible for the overflow.
func TestAlertEventListDefaultProjectionPreservesShortFields(t *testing.T) {
	for _, format := range []string{"json", "toon"} {
		t.Run(format, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			items, ids, shortTitles := alertEventOutlierFixture(30, 3)
			stub.data = map[string]any{"items": items, "total": len(items)}

			out, stderrText, err := execCommandSplit("alert-event", "list", "--output-format", format)
			if err != nil {
				t.Fatalf("execCommandSplit: %v", err)
			}
			if len(out) >= compactListOutputLimit {
				t.Fatalf("compact alert-event output is %d bytes, want <%d", len(out), compactListOutputLimit)
			}
			if !strings.Contains(stderrText, "note: rows projected to default compact fields") {
				t.Errorf("default projection should announce itself on stderr, got:\n%s", stderrText)
			}

			for _, id := range ids {
				if !strings.Contains(out, id) {
					t.Errorf("id %q was shortened; only the oversized outlier titles should shrink, got:\n%s", id, out)
				}
			}
			for _, title := range shortTitles {
				if !strings.Contains(out, title) {
					t.Errorf("short title %q was shortened even though it never exceeded the budget on its own, got:\n%s", title, out)
				}
			}
			if !strings.Contains(out, "...") {
				t.Errorf("expected the outlier titles to be visibly marked with \"...\", got:\n%s", out)
			}
		})
	}
}

// TestAlertEventListFieldsProjectionUnchanged is the conductor constraint for
// alert-event list's --fields path: it must keep selecting exactly the named
// fields, unaffected by the default-projection truncation logic.
func TestAlertEventListFieldsProjectionUnchanged(t *testing.T) {
	for _, format := range []string{"json", "toon"} {
		t.Run(format, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			items, ids, _ := alertEventOutlierFixture(30, 3)
			stub.data = map[string]any{"items": items, "total": len(items)}

			out, stderrText, err := execCommandSplit("alert-event", "list", "--fields", "event_id,alert_id", "--output-format", format)
			if err != nil {
				t.Fatalf("execCommandSplit: %v", err)
			}
			if strings.Contains(stderrText, "note: rows projected to default compact fields") {
				t.Errorf("explicit --fields must not print the default-projection note, got:\n%s", stderrText)
			}
			for _, id := range ids {
				if !strings.Contains(out, id) {
					t.Errorf("id %q missing from --fields output, got:\n%s", id, out)
				}
			}
			if strings.Contains(out, "event_severity") || strings.Contains(out, "title") {
				t.Errorf("--fields output should contain only the requested fields, got:\n%s", out)
			}
		})
	}
}

// TestBoundProjectedListNeverEmitsUnmarkedTruncation is the regression guard
// for the original defect's silent-corruption half: the old algorithm
// repeatedly halved a single shared per-field byte cap, and once that cap
// dropped to 3 bytes or below, truncateUTF8Bytes's no-room-for-a-marker
// fallback returned raw, unmarked bytes indistinguishable from a genuinely
// short value. Even under extreme row/field pressure that forces every
// shortenable string field to shrink, every shortened value must carry the
// "..." marker. (Row count is sized so the exempt _id fields and the JSON
// envelope still fit at the truncation floor — more rows would tip the page
// into the identifier-overflow error pinned by
// TestBoundProjectedListIdentifierOnlyOverflowErrors.)
func TestBoundProjectedListNeverEmitsUnmarkedTruncation(t *testing.T) {
	saveAndResetGlobals(t)
	flagOutputFormat = "json"

	rows := make([]map[string]any, 90)
	for i := range rows {
		rows[i] = map[string]any{
			"event_id":       fmt.Sprintf("%024x", i),
			"alert_id":       fmt.Sprintf("%024x", i+1_000_000),
			"event_severity": "Info",
			"event_status":   "Ok",
			"title":          strings.Repeat(fmt.Sprintf("row %d compound alert title with extra detail 详情 ", i), 3),
		}
	}
	originals := make([]map[string]any, len(rows))
	for i, row := range rows {
		clone := make(map[string]any, len(row))
		for k, v := range row {
			clone[k] = v
		}
		originals[i] = clone
	}

	if _, err := boundProjectedOutput(rows, compactListOutputLimit); err != nil {
		t.Fatalf("bound: %v", err)
	}

	for i, row := range rows {
		for key, value := range row {
			text, ok := value.(string)
			if !ok {
				continue
			}
			original := originals[i][key].(string)
			if text == original {
				continue
			}
			if !strings.HasSuffix(text, "...") {
				t.Fatalf("row %d field %q was shortened to %q without the \"...\" marker (original was %d bytes)", i, key, text, len(original))
			}
		}
	}
}

func TestStructuredFieldsEmptyErrors(t *testing.T) {
	cases := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
	}{
		{"incident similar", newIncidentSimilarCmd, []string{"inc-1", "--fields", ""}},
		{"incident detail", newIncidentDetailCmd, []string{"inc-1", "--fields", ""}},
		{"alert-event list", newAlertEventListCmd, []string{"--fields", ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			newGFStub(t)
			flagOutputFormat = "json"
			cmd := tc.cmd()
			cmd.SetOut(new(bytes.Buffer))
			cmd.SetErr(cmd.OutOrStderr())
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "--fields") {
				t.Fatalf("empty --fields error = %v, want --fields validation", err)
			}
		})
	}
}

// TestBoundProjectedListAnnouncesShortening pins that a list projection which
// had to clip values says so on the caller's side. The "..." marker alone is
// only visible to something that READS the value; a --json consumer runs a jq
// filter or an exact match over it, where a clipped string produces an empty
// result that is indistinguishable from "nothing matched" — the expensive
// failure this note exists to prevent.
func TestBoundProjectedListAnnouncesShortening(t *testing.T) {
	for _, format := range []string{"json", "toon"} {
		t.Run(format, func(t *testing.T) {
			saveAndResetGlobals(t)
			flagOutputFormat = format
			rows := []map[string]any{{
				"incident_id": "inc-1",
				"title":       strings.Repeat("payment-gateway timeout ", 200),
			}}

			note, err := boundProjectedOutput(rows, 512)
			if err != nil {
				t.Fatalf("bound projected output: %v", err)
			}
			if note == "" {
				t.Fatalf("shortened projection returned no note; caller cannot tell values were clipped")
			}
			if !strings.Contains(note, "title") {
				t.Fatalf("note = %q, want it to name the shortened field (title)", note)
			}
		})
	}
}

// TestBoundProjectedListNoNoteWhenNothingShortened keeps the note honest: a
// projection that fits must not claim anything was clipped.
func TestBoundProjectedListNoNoteWhenNothingShortened(t *testing.T) {
	saveAndResetGlobals(t)
	flagOutputFormat = "json"
	rows := []map[string]any{{"incident_id": "inc-1", "title": "disk full"}}

	note, err := boundProjectedOutput(rows, 512)
	if err != nil {
		t.Fatalf("bound projected output: %v", err)
	}
	if note != "" {
		t.Fatalf("fitting projection returned note %q, want none", note)
	}
}

// TestBoundProjectedListErrorNamesLargestFields pins that a list projection
// which cannot fit at all says WHICH fields are responsible, exactly as the
// detail path already does. Without it the only way to find the oversized
// field is to re-run the query once per field.
func TestBoundProjectedListErrorNamesLargestFields(t *testing.T) {
	saveAndResetGlobals(t)
	flagOutputFormat = "json"
	rows := make([]map[string]any, 200)
	for i := range rows {
		rows[i] = map[string]any{"count": i, "score": i * 2}
	}

	_, err := boundProjectedOutput(rows, 512)
	if err == nil {
		t.Fatalf("irreducible projection = nil error, want refusal")
	}
	if !strings.Contains(err.Error(), "largest fields:") {
		t.Fatalf("list overflow error = %q, want it to name the largest fields", err)
	}
}

// escalateRuleRow is a full EscalateRuleItem stub payload, including the
// nested layers/time_filters blobs that bloat the full dump — the same shape
// that made the raw generated command's toon output oversized per rule.
func escalateRuleRow() map[string]any {
	return map[string]any{
		"account_id":   1001,
		"aggr_window":  0,
		"channel_id":   4201,
		"channel_name": "payments",
		"created_at":   1712000000,
		"deleted_at":   0,
		"description":  "page the primary on-call, then the team",
		"filters": []any{
			[]any{map[string]any{"key": "incident_severity", "oper": "IN", "vals": []any{"Critical"}}},
		},
		"layers": []any{
			map[string]any{
				"escalate_window": 30,
				"max_times":       3,
				"notify_step":     5,
				"target":          map[string]any{"person_ids": []any{101}, "by": map[string]any{"critical": []any{"voice", "sms"}}},
			},
		},
		"priority":    10,
		"rule_id":     "6621b23f4a2c5e0012ab34d0",
		"rule_name":   "P1 on-call",
		"status":      "enabled",
		"template_id": "6630c34f5b3d6e0012cd45e1",
		"time_filters": []any{
			map[string]any{"start": "09:00", "end": "18:00", "repeat": []any{1, 2, 3}},
		},
		"updated_at": 1712100000,
		"updated_by": 101,
	}
}

// TestChannelEscalateRuleListStructuredProjection mirrors
// TestIncidentListStructuredDefaultUsesCompactProjection for the curated
// channel escalate-rule-list: json/toon mode must default to the compact
// projection (announced on stderr) instead of dumping the full nested rule
// record, an explicit --fields must override it, and table mode keeps its
// explicit columns without the note.
func TestChannelEscalateRuleListStructuredProjection(t *testing.T) {
	t.Run("json default", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{escalateRuleRow()}}

		out, stderrText, err := execCommandSplit("channel", "escalate-rule-list", "4201", "--output-format", "json")
		if err != nil {
			t.Fatalf("execCommandSplit: %v", err)
		}

		assertProjectedJSONFields(t, out, []string{"rule_id", "rule_name", "status", "priority", "filters"})
		if !strings.Contains(stderrText, "note: rows projected to default compact fields") {
			t.Errorf("default projection should announce itself on stderr, got:\n%s", stderrText)
		}
		for _, key := range []string{"layers", "template_id"} {
			if strings.Contains(out, key) {
				t.Errorf("default json output should not contain full-record key %q, got:\n%s", key, out)
			}
		}
	})

	t.Run("toon default", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{escalateRuleRow()}}

		out, stderrText, err := execCommandSplit("channel", "escalate-rule-list", "4201", "--output-format", "toon")
		if err != nil {
			t.Fatalf("execCommandSplit: %v", err)
		}

		// Positive keys must come from stdout alone: the stderr note embeds the
		// same field names, so a merged capture would satisfy this vacuously.
		for _, key := range []string{"rule_id", "rule_name", "status", "priority", "filters"} {
			if !strings.Contains(out, key) {
				t.Errorf("default toon output missing compact key %q, got:\n%s", key, out)
			}
		}
		if !strings.Contains(stderrText, "note: rows projected to default compact fields") {
			t.Errorf("default projection should announce itself on stderr, got:\n%s", stderrText)
		}
		for _, key := range []string{"layers", "template_id", "description"} {
			if strings.Contains(out, key) {
				t.Errorf("default toon output should not contain full-record key %q, got:\n%s", key, out)
			}
		}
	})

	t.Run("explicit fields win", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{escalateRuleRow()}}

		out, err := execCommand("channel", "escalate-rule-list", "4201", "--fields", "rule_id,layers", "--output-format", "json")
		if err != nil {
			t.Fatalf("execCommand: %v", err)
		}

		assertProjectedJSONFields(t, out, []string{"rule_id", "layers"})
	})

	t.Run("explicit empty fields errors", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{escalateRuleRow()}}

		_, err := execCommand("channel", "escalate-rule-list", "4201", "--fields", "", "--output-format", "json")
		if err == nil {
			t.Fatal("expected an error for empty --fields, got nil")
		}
		if !strings.Contains(err.Error(), "--fields") {
			t.Errorf("error should name --fields, got: %v", err)
		}
	})

	t.Run("unknown field errors", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{escalateRuleRow()}}

		_, err := execCommand("channel", "escalate-rule-list", "4201", "--fields", "not_a_field", "--output-format", "json")
		if err == nil {
			t.Fatal("expected an error for an unknown field, got nil")
		}
		if !strings.Contains(err.Error(), "not_a_field") {
			t.Errorf("error should name the bad field, got: %v", err)
		}
	})

	t.Run("table mode headers without note", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{escalateRuleRow()}}

		out, stderrText, err := execCommandSplit("channel", "escalate-rule-list", "4201")
		if err != nil {
			t.Fatalf("execCommandSplit: %v", err)
		}
		for _, h := range []string{"ID", "NAME", "STATUS", "PRIORITY", "UPDATED"} {
			if !strings.Contains(out, h) {
				t.Errorf("table output missing header %q, got:\n%s", h, out)
			}
		}
		if strings.Contains(stderrText, "note: rows projected") {
			t.Errorf("table mode must not print the projection note, got:\n%s", stderrText)
		}
	})

	t.Run("channel id folding", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			args []string
			want float64
		}{
			{"positional", []string{"channel", "escalate-rule-list", "4201"}, 4201},
			{"flag", []string{"channel", "escalate-rule-list", "--channel-id", "4202"}, 4202},
		} {
			t.Run(tc.name, func(t *testing.T) {
				saveAndResetGlobals(t)
				stub := newGFStub(t)
				stub.data = map[string]any{"items": []any{}}

				if _, err := execCommand(tc.args...); err != nil {
					t.Fatalf("execCommand: %v", err)
				}
				if stub.lastPath != "/channel/escalate/rule/list" {
					t.Fatalf("expected /channel/escalate/rule/list, got %q", stub.lastPath)
				}
				if stub.lastBody["channel_id"] != tc.want {
					t.Fatalf("channel_id = %#v, want %v", stub.lastBody["channel_id"], tc.want)
				}
			})
		}
	})
}

// TestBoundProjectedListNeverShortensIdentifierFields pins the identifier
// exemption: keys ending in _id/_key carry values a consumer matches,
// filters, or passes back verbatim (a jq exact-match over --json output, a
// follow-up detail call), so shortening one silently defeats that consumer.
// Even when a page overflows badly enough that the fair cap lands below an
// identifier's own length, identifiers must come back byte-identical and
// only free-text fields shorten; the note must name only the clipped fields.
func TestBoundProjectedListNeverShortensIdentifierFields(t *testing.T) {
	for _, format := range []string{"json", "toon"} {
		t.Run(format, func(t *testing.T) {
			saveAndResetGlobals(t)
			flagOutputFormat = format

			rows := make([]map[string]any, 10)
			wantIDs := make([]map[string]string, len(rows))
			for i := range rows {
				eventID := fmt.Sprintf("%024x", i)
				alertKey := fmt.Sprintf("%032x", i)
				rows[i] = map[string]any{
					"event_id":  eventID,
					"alert_key": alertKey,
					"title":     strings.Repeat("a", 500),
				}
				wantIDs[i] = map[string]string{"event_id": eventID, "alert_key": alertKey}
			}

			const budget = 1400
			note, err := boundProjectedOutput(rows, budget)
			if err != nil {
				t.Fatalf("bound projected output: %v", err)
			}

			for i, row := range rows {
				for _, field := range []string{"event_id", "alert_key"} {
					if got := row[field].(string); got != wantIDs[i][field] {
						t.Errorf("row %d %s was shortened: got %q, want byte-identical %q", i, field, got, wantIDs[i][field])
					}
				}
				if title := row["title"].(string); !strings.HasSuffix(title, "...") {
					t.Errorf("row %d title should be shortened with the \"...\" marker, got %q", i, title)
				}
			}

			if note == "" {
				t.Fatal("shortened projection returned no note; caller cannot tell values were clipped")
			}
			if !strings.Contains(note, "title") {
				t.Errorf("note = %q, want it to name the shortened field (title)", note)
			}
			if strings.Contains(note, "event_id") || strings.Contains(note, "alert_key") {
				t.Errorf("note = %q, want it to name only shortened fields, never exempt identifiers", note)
			}

			encoded, err := marshalStructured(rows)
			if err != nil {
				t.Fatalf("marshal bounded output: %v", err)
			}
			if len(encoded)+1 >= budget {
				t.Errorf("bounded %s output is %d bytes, want <%d", format, len(encoded)+1, budget)
			}
		})
	}
}

// TestBoundProjectedListIdentifierOnlyOverflowErrors pins the other half of
// the identifier exemption: when a page carries nothing shortenable and its
// identifier content alone overflows the budget, the command must fail with
// the narrowing error instead of clipping identifiers to fit — and the rows
// must come back untouched.
func TestBoundProjectedListIdentifierOnlyOverflowErrors(t *testing.T) {
	saveAndResetGlobals(t)
	flagOutputFormat = "json"

	// Sized so the full rows overflow 512 bytes while rows with ids clipped
	// to the truncation floor would still fit: the old fair cap "succeeded"
	// by shipping mangled ids, the exemption must instead refuse.
	rows := make([]map[string]any, 12)
	for i := range rows {
		rows[i] = map[string]any{"incident_id": fmt.Sprintf("%024x", i)}
	}
	originals := make([]string, len(rows))
	for i, row := range rows {
		originals[i] = row["incident_id"].(string)
	}

	_, err := boundProjectedOutput(rows, 512)
	if err == nil || !strings.Contains(err.Error(), "request fewer rows") {
		t.Fatalf("identifier-only overflow error = %v, want bounded guidance", err)
	}
	for i, row := range rows {
		if got := row["incident_id"].(string); got != originals[i] {
			t.Errorf("row %d incident_id was mutated despite the error: got %q, want %q", i, got, originals[i])
		}
	}
}
