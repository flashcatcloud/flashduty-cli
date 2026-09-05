package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/flashcatcloud/go-flashduty"
)

// newInsightExportStub starts a stub server answering /insight/incident/export
// with the given raw CSV body and /insight/incident/list with an envelope
// carrying the given total. It records each request's decoded JSON body so a
// test can assert the filter reached both endpoints.
func newInsightExportStub(t *testing.T, csvBody string, total int) *insightExportStub {
	t.Helper()
	s := &insightExportStub{}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if raw, err := io.ReadAll(r.Body); err == nil && len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		switch r.URL.Path {
		case "/insight/incident/export":
			s.exportBody = body
			w.Header().Set("Content-Type", "text/csv")
			_, _ = io.WriteString(w, csvBody)
		case "/insight/incident/list":
			s.listBody = body
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"request_id": "test-request-id",
				"error":      map[string]any{"code": "OK", "message": ""},
				"data":       map[string]any{"total": total, "items": []any{}, "has_next_page": false},
			})
		default:
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
		}
	}))
	t.Cleanup(s.server.Close)

	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test-key", flashduty.WithBaseURL(s.server.URL))
	}
	return s
}

type insightExportStub struct {
	server     *httptest.Server
	exportBody map[string]any
	listBody   map[string]any
}

// insightIncidentRow builds one /insight/incident/list row for the stub,
// carrying both the compact-projection fields and the full-record fields a
// default projection must drop.
func insightIncidentRow() map[string]any {
	return map[string]any{
		"incident_id":      "inc-1",
		"title":            "Disk full on db-01",
		"severity":         "Critical",
		"progress":         "Triggered",
		"channel_id":       12345,
		"channel_name":     "db-alerts",
		"seconds_to_ack":   42,
		"seconds_to_close": 3600,
		"notifications":    3,
		"description":      "root volume at 98%",
		"labels":           map[string]any{"service": "db", "env": "prod"},
		"responders":       []map[string]any{{"person_id": 101, "person_name": "Alice"}},
	}
}

// TestInsightIncidentsStructuredDefaultUsesCompactProjection mirrors incident
// list: structured mode must not dump the full nested SDK row when --fields
// is omitted, and the default projection announces itself on stderr.
func TestInsightIncidentsStructuredDefaultUsesCompactProjection(t *testing.T) {
	for _, format := range []string{"json", "toon"} {
		t.Run(format, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			stub.data = map[string]any{"items": []any{insightIncidentRow()}, "total": 1}

			out, stderrText, err := execCommandSplit("insight", "incidents", "--output-format", format)
			if err != nil {
				t.Fatalf("execCommandSplit: %v", err)
			}
			for _, key := range []string{"incident_id", "title", "severity", "channel_name", "seconds_to_ack", "seconds_to_close", "notifications"} {
				if !strings.Contains(out, key) {
					t.Errorf("default %s output missing compact key %q, got:\n%s", format, key, out)
				}
			}
			// Full-record keys must not leak. (stdout only: the stderr note
			// embeds the compact field names, never these.)
			for _, key := range []string{"description", "labels", "responders"} {
				if strings.Contains(out, key) {
					t.Errorf("default %s output should not contain full-record key %q, got:\n%s", format, key, out)
				}
			}
			if !strings.Contains(stderrText, "note: rows projected to default compact fields") {
				t.Errorf("default projection should announce itself on stderr, got:\n%s", stderrText)
			}
		})
	}
}

// TestInsightIncidentsStructuredFieldsFlag: an explicit --fields wins over the
// default projection and stays exactly the named fields.
func TestInsightIncidentsStructuredFieldsFlag(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{insightIncidentRow()}, "total": 1}

	out, stderrText, err := execCommandSplit("insight", "incidents",
		"--fields", "incident_id,severity", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	assertProjectedJSONFields(t, out, []string{"incident_id", "severity"})
	if strings.Contains(stderrText, "note: rows projected to default compact fields") {
		t.Errorf("explicit --fields must not print the default-projection note, got:\n%s", stderrText)
	}
}

// TestInsightIncidentsStructuredBounded: an oversized projected page is
// bounded below the structured-output limit — reduced to the leading intact
// rows, or a single oversized row shortened with a marked, announced clip.
func TestInsightIncidentsStructuredBounded(t *testing.T) {
	t.Run("reduced page keeps every value intact", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		rows := make([]any, 10)
		for i := range rows {
			row := insightIncidentRow()
			row["incident_id"] = fmt.Sprintf("inc-%d", i)
			row["title"] = strings.Repeat(fmt.Sprintf("db-%d failover ", i), 200)
			rows[i] = row
		}
		stub.data = map[string]any{"items": rows, "total": 10}

		out, stderrText, err := execCommandSplit("insight", "incidents", "--output-format", "json")
		if err != nil {
			t.Fatalf("execCommandSplit: %v", err)
		}
		if len([]byte(out)) >= compactListOutputLimit {
			t.Fatalf("bounded page is %d bytes, want <%d", len([]byte(out)), compactListOutputLimit)
		}
		if !strings.Contains(stderrText, "note: emitted") {
			t.Errorf("reduced page should announce itself on stderr, got:\n%s", stderrText)
		}
		if strings.Contains(out, "...") {
			t.Errorf("page reduction must never shorten a value, got:\n%s", out)
		}
	})

	t.Run("single oversized row is shortened and announced", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		row := insightIncidentRow()
		row["title"] = strings.Repeat("数据库故障", 5000)
		stub.data = map[string]any{"items": []any{row}, "total": 1}

		out, stderrText, err := execCommandSplit("insight", "incidents", "--output-format", "json")
		if err != nil {
			t.Fatalf("execCommandSplit: %v", err)
		}
		if len([]byte(out)) >= compactListOutputLimit {
			t.Fatalf("shortened row is %d bytes, want <%d", len([]byte(out)), compactListOutputLimit)
		}
		if !utf8.ValidString(out) || !strings.Contains(out, "...") {
			t.Fatalf("shortened row must retain valid UTF-8 and show the truncation marker")
		}
		if !strings.Contains(stderrText, "were shortened to fit") || !strings.Contains(stderrText, "title") {
			t.Errorf("shortened row should announce the clipped field on stderr, got:\n%s", stderrText)
		}
	})
}

// TestInsightIncidentsTableUnchanged: the human table keeps its full column
// set — the structured projection must not leak into table mode.
func TestInsightIncidentsTableUnchanged(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{insightIncidentRow()}, "total": 1}

	out, _, err := execCommandSplit("insight", "incidents")
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	for _, want := range []string{"ID", "TITLE", "SEVERITY", "CHANNEL", "MTTA", "MTTR", "NOTIFICATIONS", "inc-1", "db-alerts"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q, got:\n%s", want, out)
		}
	}
}

// TestInsightIncidentExportComplete verifies the happy path: when the CSV
// data-row count matches the incident-list total, the command exits 0 and
// reports the actual written row count on stderr.
func TestInsightIncidentExportComplete(t *testing.T) {
	saveAndResetGlobals(t)
	csvBody := "incident_id,title\ninc-1,a\ninc-2,b\ninc-3,c\n"
	newInsightExportStub(t, csvBody, 3)

	stdout, stderr, err := execCommandSplit("insight", "incident-export", "--start-time", "1000", "--end-time", "2000")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if stdout != csvBody {
		t.Errorf("stdout = %q, want the CSV verbatim %q", stdout, csvBody)
	}
	if !strings.Contains(stderr, "rows=3") {
		t.Errorf("stderr missing rows=3, got %q", stderr)
	}
}

// TestInsightIncidentExportTruncatedFailsLoudly is the core guard: when the
// server truncates the export (CSV data rows < incident-list total), the
// partial CSV is still written but the command must exit non-zero stating
// written vs total — a silent partial export with exit 0 is the defect.
func TestInsightIncidentExportTruncatedFailsLoudly(t *testing.T) {
	saveAndResetGlobals(t)
	csvBody := "incident_id,title\ninc-1,a\ninc-2,b\n"
	newInsightExportStub(t, csvBody, 5)

	stdout, stderr, err := execCommandSplit("insight", "incident-export", "--start-time", "1000", "--end-time", "2000")
	if err == nil {
		t.Fatal("expected a non-nil error on truncated export, got nil")
	}
	if !strings.Contains(err.Error(), "2 of 5") {
		t.Errorf("error %q does not state written vs total (2 of 5)", err.Error())
	}
	if stdout != csvBody {
		t.Errorf("stdout = %q, want the partial CSV verbatim %q", stdout, csvBody)
	}
	if !strings.Contains(stderr, "rows=2") {
		t.Errorf("stderr missing rows=2, got %q", stderr)
	}
}

// TestInsightIncidentExportCountsQuotedRows verifies row counting goes through
// encoding/csv: a quoted field with an embedded newline is one record, not
// two, so such exports are not misjudged as truncated.
func TestInsightIncidentExportCountsQuotedRows(t *testing.T) {
	saveAndResetGlobals(t)
	csvBody := "incident_id,description\ninc-1,\"line one\nline two\"\ninc-2,\"a\nb\nc\"\n"
	newInsightExportStub(t, csvBody, 2)

	_, stderr, err := execCommandSplit("insight", "incident-export", "--start-time", "1000", "--end-time", "2000")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if !strings.Contains(stderr, "rows=2") {
		t.Errorf("stderr missing rows=2, got %q", stderr)
	}
}

// TestInsightIncidentExportForwardsFilterToBothEndpoints verifies the export
// request and the verification request carry the same filter window, and the
// verification asks for a single 1-item page.
func TestInsightIncidentExportForwardsFilterToBothEndpoints(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newInsightExportStub(t, "incident_id\ninc-1\n", 1)

	if _, _, err := execCommandSplit("insight", "incident-export",
		"--start-time", "1000", "--end-time", "2000", "--severities", "Critical"); err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	for name, body := range map[string]map[string]any{"export": stub.exportBody, "list": stub.listBody} {
		if body == nil {
			t.Fatalf("%s endpoint was never called", name)
		}
		if got, _ := body["start_time"].(float64); got != 1000 {
			t.Errorf("%s start_time = %#v, want 1000", name, body["start_time"])
		}
		if got, _ := body["end_time"].(float64); got != 2000 {
			t.Errorf("%s end_time = %#v, want 2000", name, body["end_time"])
		}
		if got := fmt.Sprint(body["severities"]); got != "[Critical]" {
			t.Errorf("%s severities = %q, want [Critical]", name, got)
		}
	}
	if got, _ := stub.listBody["limit"].(float64); got != 1 {
		t.Errorf("list limit = %#v, want 1", stub.listBody["limit"])
	}
}

// TestCountCSVDataRows covers the counter directly, including the header-only
// and empty-body edges.
func TestCountCSVDataRows(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{"empty body", "", 0, false},
		{"header only", "a,b\n", 0, false},
		{"header plus rows", "a,b\n1,2\n3,4\n", 2, false},
		{"quoted newline is one row", "a,b\n\"x\ny\",2\n", 1, false},
		{"no trailing newline", "a,b\n1,2", 1, false},
		{"unterminated quote errors", "a,b\n\"x,2\n", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := countCSVDataRows([]byte(tc.raw))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("countCSVDataRows = %d, want %d", got, tc.want)
			}
		})
	}
}
