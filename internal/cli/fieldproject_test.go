package cli

import (
	"bytes"
	"encoding/json"
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

			if err := boundProjectedOutput(rows, 512); err != nil {
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

	err := boundProjectedOutput(rows, 512)
	if err == nil || !strings.Contains(err.Error(), "request fewer rows or fields") {
		t.Fatalf("irreducible output error = %v, want bounded guidance", err)
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

// TestFieldsProjectionDefaultUnchanged is the conductor constraint: with NO
// --fields, the structured (toon and json) output must still be the full nested
// record — the nested blobs the proposal deliberately preserves as the default.
func TestFieldsProjectionDefaultUnchanged(t *testing.T) {
	cases := []struct {
		name     string
		cmd      []string
		data     map[string]any
		format   string
		mustHave []string // nested keys that must survive in the full dump
	}{
		{"incident toon", []string{"incident", "list"}, incidentRow(), "toon", []string{"responders", "labels", "description"}},
		{"incident json", []string{"incident", "list"}, incidentRow(), "json", []string{"responders", "labels", "description"}},
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

			var rows []map[string]json.RawMessage
			if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &rows); err != nil {
				t.Fatalf("failed to parse projected json: %v\nraw:\n%s", err, out)
			}
			if len(rows) != 1 {
				t.Fatalf("expected 1 projected row, got %d:\n%s", len(rows), out)
			}
			row := rows[0]
			if len(row) != len(tc.fields) {
				t.Fatalf("expected exactly %d keys, got %d (%v)", len(tc.fields), len(row), row)
			}
			for _, f := range tc.fields {
				if _, ok := row[f]; !ok {
					t.Errorf("projected row missing key %q, got keys %v", f, row)
				}
			}
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

	out, err := execCommand("incident", "similar", "inc-1", "--limit", "20", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if len(out) >= 16*1024 {
		t.Fatalf("compact similar output is %d bytes, want <16 KiB", len(out))
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
	row["description"] = strings.Repeat("large description ", 500)
	row["images"] = []map[string]any{{"src": strings.Repeat("https://example.test/image/", 100)}}
	row["ai_summary"] = strings.Repeat("long AI summary ", 4000)
	row["root_cause"] = strings.Repeat("long root cause ", 4000)
	row["resolution"] = strings.Repeat("long resolution ", 4000)
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

	out, err := execCommand("alert-event", "list", "--limit", "30", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if len(out) >= 16*1024 {
		t.Fatalf("compact alert-event output is %d bytes, want <16 KiB", len(out))
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
