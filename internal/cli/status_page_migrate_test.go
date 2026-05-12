package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
)

type mockStatusPageMigrate struct {
	mockClient

	startStructure        func(ctx context.Context, input *flashduty.StartStatusPageMigrationInput) (*flashduty.StartStatusPageMigrationOutput, error)
	startEmailSubscribers func(ctx context.Context, input *flashduty.StartStatusPageEmailSubscriberMigrationInput) (*flashduty.StartStatusPageMigrationOutput, error)
	getStatus             func(ctx context.Context, jobID string) (*flashduty.StatusPageMigrationJob, error)
	cancel                func(ctx context.Context, jobID string) error
}

func (m *mockStatusPageMigrate) StartStatusPageMigration(ctx context.Context, input *flashduty.StartStatusPageMigrationInput) (*flashduty.StartStatusPageMigrationOutput, error) {
	if m.startStructure == nil {
		return m.mockClient.StartStatusPageMigration(ctx, input)
	}
	return m.startStructure(ctx, input)
}

func (m *mockStatusPageMigrate) StartStatusPageEmailSubscriberMigration(ctx context.Context, input *flashduty.StartStatusPageEmailSubscriberMigrationInput) (*flashduty.StartStatusPageMigrationOutput, error) {
	if m.startEmailSubscribers == nil {
		return m.mockClient.StartStatusPageEmailSubscriberMigration(ctx, input)
	}
	return m.startEmailSubscribers(ctx, input)
}

func (m *mockStatusPageMigrate) GetStatusPageMigrationStatus(ctx context.Context, jobID string) (*flashduty.StatusPageMigrationJob, error) {
	if m.getStatus == nil {
		return m.mockClient.GetStatusPageMigrationStatus(ctx, jobID)
	}
	return m.getStatus(ctx, jobID)
}

func (m *mockStatusPageMigrate) CancelStatusPageMigration(ctx context.Context, jobID string) error {
	if m.cancel == nil {
		return m.mockClient.CancelStatusPageMigration(ctx, jobID)
	}
	return m.cancel(ctx, jobID)
}

func TestCommandStatusPageMigrateStructureSendsSDKInput(t *testing.T) {
	saveAndResetGlobals(t)

	var gotInput *flashduty.StartStatusPageMigrationInput
	mock := &mockStatusPageMigrate{
		startStructure: func(_ context.Context, input *flashduty.StartStatusPageMigrationInput) (*flashduty.StartStatusPageMigrationOutput, error) {
			gotInput = input
			return &flashduty.StartStatusPageMigrationOutput{JobID: "job-1"}, nil
		},
	}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("statuspage", "migrate", "structure",
		"--from", "atlassian",
		"--source-page-id", "src-1",
		"--api-key", "atlassian-secret",
	)
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}

	if gotInput == nil {
		t.Fatal("expected input to be captured")
	}
	if gotInput.SourceAPIKey != "atlassian-secret" {
		t.Errorf("SourceAPIKey = %q, want atlassian-secret", gotInput.SourceAPIKey)
	}
	if gotInput.SourcePageID != "src-1" {
		t.Errorf("SourcePageID = %q, want src-1", gotInput.SourcePageID)
	}
	if gotInput.URLName != "" {
		t.Errorf("URLName = %q, want empty", gotInput.URLName)
	}
	if !strings.Contains(out, "Job ID: job-1") {
		t.Errorf("missing job id in output:\n%s", out)
	}
	if !strings.Contains(out, "flashduty statuspage migrate status --job-id job-1") {
		t.Errorf("missing status hint in output:\n%s", out)
	}
}

func TestCommandStatusPageMigrateStructureSendsURLName(t *testing.T) {
	saveAndResetGlobals(t)

	var gotInput *flashduty.StartStatusPageMigrationInput
	mock := &mockStatusPageMigrate{
		startStructure: func(_ context.Context, input *flashduty.StartStatusPageMigrationInput) (*flashduty.StartStatusPageMigrationOutput, error) {
			gotInput = input
			return &flashduty.StartStatusPageMigrationOutput{JobID: "job-url"}, nil
		},
	}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	_, err := execCommand("statuspage", "migrate", "structure",
		"--from", "atlassian",
		"--source-page-id", "src-1",
		"--api-key", "atlassian-secret",
		"--url-name", "customer-facing-status",
	)
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}

	if gotInput == nil {
		t.Fatal("expected input to be captured")
	}
	if gotInput.URLName != "customer-facing-status" {
		t.Errorf("URLName = %q, want customer-facing-status", gotInput.URLName)
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

	called := false
	mock := &mockStatusPageMigrate{
		startStructure: func(context.Context, *flashduty.StartStatusPageMigrationInput) (*flashduty.StartStatusPageMigrationOutput, error) {
			called = true
			return nil, nil
		},
	}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

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
	if called {
		t.Error("SDK should not have been called for unsupported source")
	}
}

// TestCommandStatusPageMigrateStructureValidatesBeforeClient locks ordering:
// an invalid --from must surface its validation error before any
// client-build / auth work — matching PR #1 behavior.
func TestCommandStatusPageMigrateStructureValidatesBeforeClient(t *testing.T) {
	saveAndResetGlobals(t)

	clientBuilt := false
	newClientFn = func() (flashdutyClient, error) {
		clientBuilt = true
		return nil, fmt.Errorf("should not have been called")
	}

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
	if clientBuilt {
		t.Error("newClientFn must not run when --from is invalid")
	}
}

// TestCommandStatusPageMigrateEmailSubscribersValidatesBeforeClient: same
// ordering guarantee for the subscribers variant.
func TestCommandStatusPageMigrateEmailSubscribersValidatesBeforeClient(t *testing.T) {
	saveAndResetGlobals(t)

	clientBuilt := false
	newClientFn = func() (flashdutyClient, error) {
		clientBuilt = true
		return nil, fmt.Errorf("should not have been called")
	}

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
	if clientBuilt {
		t.Error("newClientFn must not run when --from is invalid")
	}
}

func TestCommandStatusPageMigrateStructureJSON(t *testing.T) {
	saveAndResetGlobals(t)

	mock := &mockStatusPageMigrate{
		startStructure: func(_ context.Context, _ *flashduty.StartStatusPageMigrationInput) (*flashduty.StartStatusPageMigrationOutput, error) {
			return &flashduty.StartStatusPageMigrationOutput{JobID: "job-1"}, nil
		},
	}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

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

func TestCommandStatusPageMigrateEmailSubscribersSendsSDKInput(t *testing.T) {
	saveAndResetGlobals(t)

	var gotInput *flashduty.StartStatusPageEmailSubscriberMigrationInput
	mock := &mockStatusPageMigrate{
		startEmailSubscribers: func(_ context.Context, input *flashduty.StartStatusPageEmailSubscriberMigrationInput) (*flashduty.StartStatusPageMigrationOutput, error) {
			gotInput = input
			return &flashduty.StartStatusPageMigrationOutput{JobID: "sub-1"}, nil
		},
	}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("statuspage", "migrate", "email-subscribers",
		"--from", "atlassian",
		"--source-page-id", "src-1",
		"--target-page-id", "2048",
		"--api-key", "atlassian-secret",
	)
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}

	if gotInput == nil {
		t.Fatal("expected input to be captured")
	}
	if gotInput.TargetPageID != 2048 {
		t.Errorf("TargetPageID = %d, want 2048", gotInput.TargetPageID)
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

	var gotJobID string
	mock := &mockStatusPageMigrate{
		getStatus: func(_ context.Context, jobID string) (*flashduty.StatusPageMigrationJob, error) {
			gotJobID = jobID
			return &flashduty.StatusPageMigrationJob{
				JobID:        "job-9",
				SourcePageID: "src-9",
				TargetPageID: 1024,
				Phase:        "history",
				Status:       "running",
				Progress: flashduty.StatusPageMigrationProgress{
					TotalSteps:           5,
					CompletedSteps:       3,
					ComponentsImported:   2,
					SectionsImported:     1,
					IncidentsImported:    4,
					MaintenancesImported: 1,
					SubscribersImported:  0,
					SubscribersSkipped:   0,
					TemplatesImported:    2,
					Warnings:             []string{"missing field X"},
				},
			}, nil
		},
	}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("statuspage", "migrate", "status", "--job-id", "job-9")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}

	if gotJobID != "job-9" {
		t.Errorf("jobID passed to SDK = %q, want job-9", gotJobID)
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

	mock := &mockStatusPageMigrate{
		getStatus: func(_ context.Context, _ string) (*flashduty.StatusPageMigrationJob, error) {
			return &flashduty.StatusPageMigrationJob{
				JobID:  "job-j",
				Phase:  "completed",
				Status: "completed",
			}, nil
		},
	}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

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

	var gotJobID string
	mock := &mockStatusPageMigrate{
		cancel: func(_ context.Context, jobID string) error {
			gotJobID = jobID
			return nil
		},
	}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("statuspage", "migrate", "cancel", "--job-id", "job-c")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}

	if gotJobID != "job-c" {
		t.Errorf("SDK received jobID %q, want job-c", gotJobID)
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

	mock := &mockStatusPageMigrate{
		cancel: func(context.Context, string) error { return nil },
	}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

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

	mock := &mockStatusPageMigrate{
		getStatus: func(context.Context, string) (*flashduty.StatusPageMigrationJob, error) {
			return nil, &flashduty.DutyError{Code: "not_found", Message: "job missing"}
		},
	}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	_, err := execCommand("statuspage", "migrate", "status", "--job-id", "nope")
	if err == nil {
		t.Fatal("expected SDK error to propagate")
	}
	if !strings.Contains(err.Error(), "not_found") || !strings.Contains(err.Error(), "job missing") {
		t.Errorf("unexpected error: %v", err)
	}
}
