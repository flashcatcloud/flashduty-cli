package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStatusPageMigrationAPIStartStructure(t *testing.T) {
	t.Parallel()

	var gotMethod string
	var gotPath string
	var gotAppKey string
	var gotBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAppKey = r.URL.Query().Get("app_key")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"job_id": "job-1"},
		})
	}))
	defer ts.Close()

	api := &statusPageMigrationAPI{
		httpClient: ts.Client(),
		baseURL:    ts.URL,
		appKey:     "fd-app-key",
		userAgent:  "flashduty-cli/test",
	}

	out, err := api.StartStructure(context.Background(), "atlassian-key", "page_123")
	if err != nil {
		t.Fatalf("StartStructure() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/status-page/migrate-structure" {
		t.Fatalf("path = %s", gotPath)
	}
	if gotAppKey != "fd-app-key" {
		t.Fatalf("app_key = %s", gotAppKey)
	}
	if gotBody["api_key"] != "atlassian-key" || gotBody["source_page_id"] != "page_123" {
		t.Fatalf("unexpected body: %#v", gotBody)
	}
	if out.JobID != "job-1" {
		t.Fatalf("job_id = %s", out.JobID)
	}
}

func TestStatusPageMigrationAPIGetStatus(t *testing.T) {
	t.Parallel()

	var gotMethod string
	var gotPath string
	var gotJobID string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotJobID = r.URL.Query().Get("job_id")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"job_id":         "job-2",
				"source_page_id": "src-1",
				"target_page_id": 1024,
				"phase":          "history",
				"status":         "running",
				"progress": map[string]any{
					"total_steps":     5,
					"completed_steps": 3,
				},
			},
		})
	}))
	defer ts.Close()

	api := &statusPageMigrationAPI{
		httpClient: ts.Client(),
		baseURL:    ts.URL,
		appKey:     "fd-app-key",
		userAgent:  "flashduty-cli/test",
	}

	out, err := api.GetStatus(context.Background(), "job-2")
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/status-page/migration/status" {
		t.Fatalf("path = %s", gotPath)
	}
	if gotJobID != "job-2" {
		t.Fatalf("job_id query = %s", gotJobID)
	}
	if out.JobID != "job-2" || out.TargetPageID != 1024 {
		t.Fatalf("unexpected job: %#v", out)
	}
}

func TestStatusPageMigrationAPICancel(t *testing.T) {
	t.Parallel()

	var gotMethod string
	var gotPath string
	var gotBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	defer ts.Close()

	api := &statusPageMigrationAPI{
		httpClient: ts.Client(),
		baseURL:    ts.URL,
		appKey:     "fd-app-key",
		userAgent:  "flashduty-cli/test",
	}

	if err := api.Cancel(context.Background(), "job-3"); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/status-page/migration/cancel" {
		t.Fatalf("path = %s", gotPath)
	}
	if gotBody["job_id"] != "job-3" {
		t.Fatalf("unexpected body: %#v", gotBody)
	}
}

type stubMigrationService struct {
	startStructure        func(ctx context.Context, sourceAPIKey, sourcePageID string) (*migrationStartResult, error)
	startEmailSubscribers func(ctx context.Context, sourceAPIKey, sourcePageID string, targetPageID int64) (*migrationStartResult, error)
	getStatus             func(ctx context.Context, jobID string) (*migrationJob, error)
	cancel                func(ctx context.Context, jobID string) error
}

func (s stubMigrationService) StartStructure(ctx context.Context, sourceAPIKey, sourcePageID string) (*migrationStartResult, error) {
	return s.startStructure(ctx, sourceAPIKey, sourcePageID)
}

func (s stubMigrationService) StartEmailSubscribers(ctx context.Context, sourceAPIKey, sourcePageID string, targetPageID int64) (*migrationStartResult, error) {
	return s.startEmailSubscribers(ctx, sourceAPIKey, sourcePageID, targetPageID)
}

func (s stubMigrationService) GetStatus(ctx context.Context, jobID string) (*migrationJob, error) {
	return s.getStatus(ctx, jobID)
}

func (s stubMigrationService) Cancel(ctx context.Context, jobID string) error {
	return s.cancel(ctx, jobID)
}

func TestStatusPageMigrateStructureCommandPrintsStatusHint(t *testing.T) {
	original := newStatusPageMigrationService
	t.Cleanup(func() { newStatusPageMigrationService = original })
	newStatusPageMigrationService = func() (statusPageMigrationService, error) {
		return stubMigrationService{
			startStructure: func(ctx context.Context, apiKey, pageID string) (*migrationStartResult, error) {
				return &migrationStartResult{JobID: "job-123"}, nil
			},
		}, nil
	}

	cmd := newStatusPageMigrateStructureCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--from", "atlassian", "--source-page-id", "src-1", "--api-key", "key-1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Job ID: job-123") {
		t.Fatalf("missing job id in output: %s", out)
	}
	if !strings.Contains(out, "flashduty statuspage migrate status --job-id job-123") {
		t.Fatalf("missing status hint in output: %s", out)
	}
}

func TestStatusPageMigrateEmailSubscribersCommandPrintsStatusHint(t *testing.T) {
	original := newStatusPageMigrationService
	t.Cleanup(func() { newStatusPageMigrationService = original })
	newStatusPageMigrationService = func() (statusPageMigrationService, error) {
		return stubMigrationService{
			startEmailSubscribers: func(ctx context.Context, apiKey, pageID string, targetPageID int64) (*migrationStartResult, error) {
				if targetPageID != 2048 {
					t.Fatalf("target_page_id = %d, want 2048", targetPageID)
				}
				return &migrationStartResult{JobID: "job-456"}, nil
			},
		}, nil
	}

	cmd := newStatusPageMigrateEmailSubscribersCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--from", "atlassian", "--source-page-id", "src-1", "--target-page-id", "2048", "--api-key", "key-1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Target page ID: 2048") {
		t.Fatalf("missing target page id in output: %s", out)
	}
	if !strings.Contains(out, "flashduty statuspage migrate status --job-id job-456") {
		t.Fatalf("missing status hint in output: %s", out)
	}
}

func TestStatusPageMigrateStatusCommandPrintsJobDetails(t *testing.T) {
	original := newStatusPageMigrationService
	t.Cleanup(func() { newStatusPageMigrationService = original })
	newStatusPageMigrationService = func() (statusPageMigrationService, error) {
		return stubMigrationService{
			getStatus: func(ctx context.Context, jobID string) (*migrationJob, error) {
				return &migrationJob{
					JobID:        jobID,
					SourcePageID: "src-1",
					TargetPageID: 1024,
					Phase:        "history",
					Status:       "completed",
					Progress: migrationProgress{
						TotalSteps:           5,
						CompletedSteps:       5,
						SectionsImported:     2,
						ComponentsImported:   4,
						IncidentsImported:    3,
						MaintenancesImported: 1,
						TemplatesImported:    2,
						Warnings:             []string{"incident skipped"},
					},
				}, nil
			},
		}, nil
	}

	cmd := newStatusPageMigrateStatusCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--job-id", "job-123"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Target page ID: 1024") || !strings.Contains(out, "Warnings:") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestStatusPageMigrateCancelCommandPrintsStatusHint(t *testing.T) {
	original := newStatusPageMigrationService
	t.Cleanup(func() { newStatusPageMigrationService = original })
	newStatusPageMigrationService = func() (statusPageMigrationService, error) {
		return stubMigrationService{
			cancel: func(ctx context.Context, jobID string) error {
				if jobID != "job-789" {
					t.Fatalf("jobID = %s, want job-789", jobID)
				}
				return nil
			},
		}, nil
	}

	cmd := newStatusPageMigrateCancelCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--job-id", "job-789"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Cancellation requested.") {
		t.Fatalf("missing cancel message in output: %s", out)
	}
	if !strings.Contains(out, "flashduty statuspage migrate status --job-id job-789") {
		t.Fatalf("missing status hint in output: %s", out)
	}
}
