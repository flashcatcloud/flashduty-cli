package cli

import (
	"encoding/json"
	"strings"
	"testing"
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
	t.Run("json default", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{incidentRow()}, "total": 1}

		out, err := execCommand("incident", "list", "--output-format", "json")
		if err != nil {
			t.Fatalf("execCommand: %v", err)
		}

		assertProjectedJSONFields(t, out, []string{"incident_id", "title", "incident_severity", "progress", "start_time", "channel_id"})
	})

	t.Run("toon default", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{incidentRow()}, "total": 1}

		out, err := execCommand("incident", "list", "--output-format", "toon")
		if err != nil {
			t.Fatalf("execCommand: %v", err)
		}

		for _, key := range []string{"incident_id", "title", "incident_severity", "progress", "start_time", "channel_id"} {
			if !strings.Contains(out, key) {
				t.Errorf("default toon output missing compact key %q, got:\n%s", key, out)
			}
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
