package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flashcatcloud/go-flashduty"
)

// TestCommandStatusPageMigrateStructureSendsSDKInput asserts the structure
// command POSTs to /status-page/migrate-structure with the api_key and
// source_page_id wire fields and renders the returned job id.
func TestCommandStatusPageMigrateStructureSendsSDKInput(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"job_id": "job-1"}

	out, err := execCommand("statuspage", "migrate", "structure",
		"--from", "atlassian",
		"--source-page-id", "src-1",
		"--api-key", "atlassian-secret",
	)
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}

	if stub.lastPath != "/status-page/migrate-structure" {
		t.Fatalf("expected /status-page/migrate-structure, got %q", stub.lastPath)
	}
	if stub.lastBody["api_key"] != "atlassian-secret" {
		t.Errorf("api_key = %v, want atlassian-secret", stub.lastBody["api_key"])
	}
	if stub.lastBody["source_page_id"] != "src-1" {
		t.Errorf("source_page_id = %v, want src-1", stub.lastBody["source_page_id"])
	}
	// url_name is an optional *string; when --url-name is not passed it stays
	// nil and omitempty keeps it off the wire.
	if _, ok := stub.lastBody["url_name"]; ok {
		t.Errorf("url_name should not be sent when --url-name is omitted, got %#v", stub.lastBody["url_name"])
	}
	if !strings.Contains(out, "Job ID: job-1") {
		t.Errorf("missing job id in output:\n%s", out)
	}
	if !strings.Contains(out, "flashduty statuspage migrate status --job-id job-1") {
		t.Errorf("missing status hint in output:\n%s", out)
	}
}

// TestCommandStatusPageMigrateStructureForwardsURLName: MigrateStatusPageStructureRequest
// now carries url_name (*string), so --url-name is forwarded to the SDK as the
// url_name wire field — matching legacy behavior.
func TestCommandStatusPageMigrateStructureForwardsURLName(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"job_id": "job-2"}

	_, err := execCommand("statuspage", "migrate", "structure",
		"--from", "atlassian",
		"--source-page-id", "src-1",
		"--api-key", "atlassian-secret",
		"--url-name", "customer-facing-status",
	)
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if stub.requests != 1 {
		t.Errorf("expected exactly 1 request, got %d", stub.requests)
	}
	if stub.lastBody["url_name"] != "customer-facing-status" {
		t.Errorf("url_name = %#v, want customer-facing-status", stub.lastBody["url_name"])
	}
}

func TestCommandStatusPageMigrateStructureHelpDescribesURLNameBehavior(t *testing.T) {
	cmd := newStatusPageMigrateStructureCmd()
	flag := cmd.Flags().Lookup("url-name")
	if flag == nil {
		t.Fatal("expected --url-name flag to be registered")
	}

	for _, want := range []string{
		"newly created Flashduty public status page",
		"already mapped to a different URL name",
	} {
		if !strings.Contains(flag.Usage, want) {
			t.Errorf("--url-name usage missing %q: %s", want, flag.Usage)
		}
	}
}

func TestCommandStatusPageMigrateStructureRejectsUnsupportedSource(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand("statuspage", "migrate", "structure",
		"--from", "pagerduty",
		"--source-page-id", "src-1",
		"--api-key", "x",
	)
	if err == nil {
		t.Fatal("expected error for unsupported source")
	}
	if !strings.Contains(err.Error(), "unsupported migration source") {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "atlassian") {
		t.Errorf("error should mention supported source 'atlassian': %v", err)
	}
	if stub.requests != 0 {
		t.Errorf("client should not have been called for unsupported source, got %d request(s)", stub.requests)
	}
}

// TestCommandStatusPageMigrateStructureValidatesBeforeClient locks ordering:
// an invalid --from must surface its validation error before any
// client-build / auth work — matching PR #1 behavior.
func TestCommandStatusPageMigrateStructureValidatesBeforeClient(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand("statuspage", "migrate", "structure",
		"--from", "pagerduty",
		"--source-page-id", "src-1",
		"--api-key", "x",
	)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "unsupported migration source") {
		t.Errorf("got %v; want validation error about source", err)
	}
	if stub.requests != 0 {
		t.Errorf("client must not run when --from is invalid, got %d request(s)", stub.requests)
	}
}

// TestCommandStatusPageMigrateEmailSubscribersValidatesBeforeClient: same
// ordering guarantee for the subscribers variant.
func TestCommandStatusPageMigrateEmailSubscribersValidatesBeforeClient(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand("statuspage", "migrate", "email-subscribers",
		"--from", "pagerduty",
		"--source-page-id", "src-1",
		"--target-page-id", "1",
		"--api-key", "x",
	)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "unsupported migration source") {
		t.Errorf("got %v; want validation error about source", err)
	}
	if stub.requests != 0 {
		t.Errorf("client must not run when --from is invalid, got %d request(s)", stub.requests)
	}
}

func TestCommandStatusPageMigrateStructureJSON(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"job_id": "job-1"}

	out, err := execCommand("--json", "statuspage", "migrate", "structure",
		"--from", "atlassian",
		"--source-page-id", "src-1",
		"--api-key", "x",
	)
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if payload["type"] != "structure" {
		t.Errorf("type = %v, want structure", payload["type"])
	}
	if payload["source"] != "atlassian" {
		t.Errorf("source = %v, want atlassian", payload["source"])
	}
	if payload["source_page_id"] != "src-1" {
		t.Errorf("source_page_id = %v, want src-1", payload["source_page_id"])
	}
	if payload["job_id"] != "job-1" {
		t.Errorf("job_id = %v, want job-1", payload["job_id"])
	}
	if next, _ := payload["next_command"].(string); !strings.Contains(next, "job-1") {
		t.Errorf("next_command missing job id: %v", payload["next_command"])
	}
}

// TestCommandStatusPageMigrateEmailSubscribersSendsSDKInput asserts the
// email-subscribers command POSTs to /status-page/migrate-email-subscribers
// with the target_page_id wire field and renders the returned job id.
func TestCommandStatusPageMigrateEmailSubscribersSendsSDKInput(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"job_id": "sub-1"}

	out, err := execCommand("statuspage", "migrate", "email-subscribers",
		"--from", "atlassian",
		"--source-page-id", "src-1",
		"--target-page-id", "2048",
		"--api-key", "atlassian-secret",
	)
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}

	if stub.lastPath != "/status-page/migrate-email-subscribers" {
		t.Fatalf("expected /status-page/migrate-email-subscribers, got %q", stub.lastPath)
	}
	if stub.lastBody["api_key"] != "atlassian-secret" {
		t.Errorf("api_key = %v, want atlassian-secret", stub.lastBody["api_key"])
	}
	if stub.lastBody["source_page_id"] != "src-1" {
		t.Errorf("source_page_id = %v, want src-1", stub.lastBody["source_page_id"])
	}
	// JSON numbers decode to float64 through the stub.
	if got, _ := stub.lastBody["target_page_id"].(float64); got != 2048 {
		t.Errorf("target_page_id = %v, want 2048", stub.lastBody["target_page_id"])
	}
	if !strings.Contains(out, "Target page ID: 2048") {
		t.Errorf("missing target page id line in output:\n%s", out)
	}
	if !strings.Contains(out, "Job ID: sub-1") {
		t.Errorf("missing job id in output:\n%s", out)
	}
}

func TestCommandStatusPageMigrateStatusRendersJobFields(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"job_id":         "job-9",
		"source_page_id": "src-9",
		"target_page_id": 1024,
		"phase":          "history",
		"status":         "running",
		"progress": map[string]any{
			"total_steps":           5,
			"completed_steps":       3,
			"components_imported":   2,
			"sections_imported":     1,
			"incidents_imported":    4,
			"maintenances_imported": 1,
			"subscribers_imported":  0,
			"subscribers_skipped":   0,
			"templates_imported":    2,
			"warnings":              []string{"missing field X"},
		},
	}

	out, err := execCommand("statuspage", "migrate", "status", "--job-id", "job-9")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}

	// migration-status is a GET: job_id rides in the query string, so the
	// decoded body is empty. Assert the endpoint path instead.
	if stub.lastPath != "/status-page/migration/status" {
		t.Errorf("expected /status-page/migration/status, got %q", stub.lastPath)
	}
	for _, want := range []string{
		"Job ID: job-9",
		"Source page: src-9",
		"Target page ID: 1024",
		"Phase: history",
		"Status: running",
		"Progress: 3/5",
		"Incidents imported: 4",
		"Templates imported: 2",
		"Warnings:",
		"- missing field X",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestCommandStatusPageMigrateStatusJSON(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"job_id": "job-j",
		"phase":  "completed",
		"status": "completed",
	}

	out, err := execCommand("--json", "statuspage", "migrate", "status", "--job-id", "job-j")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if payload["job_id"] != "job-j" {
		t.Errorf("job_id = %v, want job-j", payload["job_id"])
	}
	if payload["status"] != "completed" {
		t.Errorf("status = %v, want completed", payload["status"])
	}
}

func TestCommandStatusPageMigrateCancelIssuesCancelAndHint(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	out, err := execCommand("statuspage", "migrate", "cancel", "--job-id", "job-c")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}

	if stub.lastPath != "/status-page/migration/cancel" {
		t.Fatalf("expected /status-page/migration/cancel, got %q", stub.lastPath)
	}
	if stub.lastBody["job_id"] != "job-c" {
		t.Errorf("job_id = %v, want job-c", stub.lastBody["job_id"])
	}
	if !strings.Contains(out, "Cancellation requested.") {
		t.Errorf("missing confirmation in output:\n%s", out)
	}
	if !strings.Contains(out, "flashduty statuspage migrate status --job-id job-c") {
		t.Errorf("missing status hint in output:\n%s", out)
	}
}

func TestCommandStatusPageMigrateCancelJSON(t *testing.T) {
	saveAndResetGlobals(t)
	newGFStub(t)

	out, err := execCommand("--json", "statuspage", "migrate", "cancel", "--job-id", "job-c")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if payload["job_id"] != "job-c" {
		t.Errorf("job_id = %v, want job-c", payload["job_id"])
	}
	if payload["status"] != "cancel_requested" {
		t.Errorf("status = %v, want cancel_requested", payload["status"])
	}
	if command, _ := payload["command"].(string); !strings.Contains(command, "job-c") {
		t.Errorf("command missing job id: %v", payload["command"])
	}
	if next, _ := payload["next_command"].(string); !strings.Contains(next, "job-c") {
		t.Errorf("next_command missing job id: %v", payload["next_command"])
	}
}

func TestCommandStatusPageMigrateStatusPropagatesSDKError(t *testing.T) {
	saveAndResetGlobals(t)

	// gfStub always replies with a success ("OK") envelope, so to exercise the
	// error path we stand up a tiny server that returns a failure envelope and
	// wire newClientFn at it directly. The client surfaces the envelope's
	// error.code/message in the returned error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id": "test-request-id",
			"error":      map[string]any{"code": "not_found", "message": "job missing"},
		})
	}))
	t.Cleanup(srv.Close)
	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test-key", flashduty.WithBaseURL(srv.URL))
	}

	_, err := execCommand("statuspage", "migrate", "status", "--job-id", "nope")
	if err == nil {
		t.Fatal("expected SDK error to propagate")
	}
	if !strings.Contains(err.Error(), "not_found") || !strings.Contains(err.Error(), "job missing") {
		t.Errorf("unexpected error: %v", err)
	}
}
