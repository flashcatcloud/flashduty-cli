package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
