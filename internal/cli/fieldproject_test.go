package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
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

// TestStructuredDefaultIsFullRecord: without --fields, the curated list verbs
// emit the full SDK record in json/toon mode — nested keys included — and say
// nothing on stderr about a projection.
func TestStructuredDefaultIsFullRecord(t *testing.T) {
	cases := []struct {
		name     string
		cmd      []string
		row      map[string]any
		mustHave []string
	}{
		{"incident list", []string{"incident", "list"}, incidentRow(), []string{"responders", "labels", "description"}},
		{"incident similar", []string{"incident", "similar", "inc-1"}, incidentRow(), []string{"responders", "labels", "description"}},
		{"alert list", []string{"alert", "list"}, alertRow(), []string{"events", "incident", "labels", "description"}},
		{"alert-event list", []string{"alert-event", "list"}, alertEventRow(), []string{"labels", "description"}},
		{"channel escalate-rule-list", []string{"channel", "escalate-rule-list", "4201"}, escalateRuleRow(), []string{"layers", "template_id", "description"}},
	}
	for _, tc := range cases {
		for _, format := range []string{"json", "toon"} {
			t.Run(tc.name+" "+format, func(t *testing.T) {
				saveAndResetGlobals(t)
				stub := newGFStub(t)
				stub.data = map[string]any{"items": []any{tc.row}, "total": 1}

				args := append(append([]string(nil), tc.cmd...), "--output-format", format)
				out, stderrText, err := execCommandSplit(args...)
				if err != nil {
					t.Fatalf("execCommandSplit: %v", err)
				}
				for _, key := range tc.mustHave {
					if !strings.Contains(out, key) {
						t.Errorf("default %s output should contain full-record key %q, got:\n%s", format, key, out)
					}
				}
				if strings.Contains(stderrText, "projected") {
					t.Errorf("default output must not announce a projection, got:\n%s", stderrText)
				}
			})
		}
	}
}

// TestStructuredListLargerThan16KiBIsComplete: a structured page far larger
// than 16 KiB comes back whole — every row, every value byte-identical, no
// "..." marker, no stderr note about a reduction.
func TestStructuredListLargerThan16KiBIsComplete(t *testing.T) {
	const total = 100
	longTitle := strings.Repeat("数据库主库磁盘空间不足告警 / db-01 / 磁盘使用率超过95% ", 40)
	cases := []struct {
		name string
		cmd  []string
		row  func(i int) map[string]any
		id   string
	}{
		{"incident list", []string{"incident", "list", "--limit", "100"}, func(i int) map[string]any {
			row := incidentRow()
			row["incident_id"] = fmt.Sprintf("%024x", i)
			row["title"] = fmt.Sprintf("%s(row %d)", longTitle, i)
			return row
		}, "incident_id"},
		{"alert-event list", []string{"alert-event", "list", "--limit", "100"}, func(i int) map[string]any {
			row := alertEventRow()
			row["event_id"] = fmt.Sprintf("%024x", i)
			row["title"] = fmt.Sprintf("%s(row %d)", longTitle, i)
			return row
		}, "event_id"},
	}
	for _, tc := range cases {
		for _, format := range []string{"json", "toon"} {
			t.Run(tc.name+" "+format, func(t *testing.T) {
				saveAndResetGlobals(t)
				stub := newGFStub(t)
				items := make([]any, total)
				for i := range items {
					items[i] = tc.row(i)
				}
				stub.data = map[string]any{"items": items, "total": total}

				args := append(append([]string(nil), tc.cmd...), "--output-format", format)
				out, stderrText, err := execCommandSplit(args...)
				if err != nil {
					t.Fatalf("execCommandSplit: %v", err)
				}
				if len(out) < 16*1024*10 {
					t.Fatalf("fixture too small to exercise a large page: %d bytes", len(out))
				}
				var decoded []map[string]any
				switch format {
				case "json":
					err = json.Unmarshal([]byte(strings.TrimSpace(out)), &decoded)
				case "toon":
					err = toon.Unmarshal([]byte(out), &decoded)
				}
				if err != nil {
					t.Fatalf("parse %s output as an array: %v", format, err)
				}
				if len(decoded) != total {
					t.Fatalf("got %d rows, want all %d", len(decoded), total)
				}
				for i, row := range decoded {
					want := tc.row(i)
					if row[tc.id] != want[tc.id] || row["title"] != want["title"] {
						t.Fatalf("row %d altered: %s=%v title=%.40q", i, tc.id, row[tc.id], row["title"])
					}
				}
				if strings.Contains(out, "...") {
					t.Errorf("output must not carry a truncation marker")
				}
				assertNoReductionNote(t, stderrText)
			})
		}
	}
}

// assertNoReductionNote fails when stderr says rows or values were withheld or
// shortened. Other notes (the query window, page completeness) may appear.
func assertNoReductionNote(t *testing.T, stderrText string) {
	t.Helper()
	for _, word := range []string{"emitted", "shortened", "truncat", "byte"} {
		if strings.Contains(stderrText, word) {
			t.Errorf("stderr reports a reduction (%q), got:\n%s", word, stderrText)
		}
	}
}

func alertEventRow() map[string]any {
	return map[string]any{
		"event_id":       "ev-1",
		"alert_id":       "al-1",
		"event_severity": "Warning",
		"event_status":   "Triggered",
		"event_time":     1712000000,
		"title":          "High CPU on web-02",
		"description":    "cpu > 90% for 5m",
		"labels":         map[string]any{"host": "web-02"},
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
			out, _, err := execCommandSplit(args...)
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

// alertEventOutlierFixture builds n alert-event rows with long (Mongo
// ObjectID-shaped) ids; the first `outliers` rows carry a long multi-word/CJK
// title. It returns the rows alongside the ids and titles they carry.
func alertEventOutlierFixture(n, outliers int) (items []any, ids, titles []string) {
	longTitle := strings.Repeat("K8S pod tcp 接收队列大于2000 / cluster-prod-a / node-17 / namespace kube-system / pod coredns-7db6d8ff4d-abcde ", 40)
	normalTitles := []string{
		"ERROR Detected / VMLogs-Prod",
		"CPU利用率较高 / fc-n9e-plus-18001",
		"服务器 dev-flasheye-01 连续飘红",
		"Disk usage high / db-02",
	}

	items = make([]any, n)
	ids = make([]string, 0, n*2)
	titles = make([]string, 0, n)
	for i := range items {
		eventID := fmt.Sprintf("%024x", i)
		alertID := fmt.Sprintf("%024x", i+1_000_000)
		ids = append(ids, eventID, alertID)

		title := normalTitles[i%len(normalTitles)]
		if i < outliers {
			title = fmt.Sprintf("%s (row %d)", longTitle, i)
		}
		titles = append(titles, title)

		items[i] = map[string]any{
			"event_id":       eventID,
			"alert_id":       alertID,
			"event_severity": "Warning",
			"event_status":   "Triggered",
			"event_time":     1712000000 + i,
			"title":          title,
		}
	}
	return items, ids, titles
}

// TestAlertEventListFieldsProjectionUnchanged: alert-event list's --fields
// selects exactly the named fields on every row.
func TestAlertEventListFieldsProjectionUnchanged(t *testing.T) {
	for _, format := range []string{"json", "toon"} {
		t.Run(format, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			items, ids, _ := alertEventOutlierFixture(30, 3)
			stub.data = map[string]any{"items": items, "total": len(items)}

			out, _, err := execCommandSplit("alert-event", "list", "--fields", "event_id,alert_id", "--output-format", format)
			if err != nil {
				t.Fatalf("execCommandSplit: %v", err)
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

// TestChannelEscalateRuleListStructuredProjection: an explicit --fields
// narrows channel escalate-rule-list rows, an unknown field fails fast, and
// table mode keeps its explicit columns. The full-record default is covered by
// TestStructuredDefaultIsFullRecord.
func TestChannelEscalateRuleListStructuredProjection(t *testing.T) {
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

	t.Run("table mode headers", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{escalateRuleRow()}}

		out, err := execCommand("channel", "escalate-rule-list", "4201")
		if err != nil {
			t.Fatalf("execCommandSplit: %v", err)
		}
		for _, h := range []string{"ID", "NAME", "STATUS", "PRIORITY", "UPDATED"} {
			if !strings.Contains(out, h) {
				t.Errorf("table output missing header %q, got:\n%s", h, out)
			}
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
