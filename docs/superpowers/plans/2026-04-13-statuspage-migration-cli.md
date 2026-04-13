# Statuspage Migration CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `flashduty statuspage migrate` subcommands for Atlassian-to-Flashduty status page migration, covering structure import, email-subscriber import, status lookup, and cancellation with simple async-job UX.

**Architecture:** Keep the new migration transport inside `flashduty-cli` so the CLI remains buildable without waiting for a separate `flashduty-sdk` release. Add one focused migration API helper plus Cobra subcommands that call it and print concise operator guidance, especially the follow-up `migrate status` command after `structure` and `email-subscribers`.

**Tech Stack:** Go, Cobra, `net/http`, existing CLI config/output helpers, Go `testing` + `httptest`

---

### Task 1: Add a Focused Migration API Helper

**Files:**
- Create: `internal/cli/status_page_migrate.go`
- Modify: `internal/cli/root.go`
- Test: `internal/cli/status_page_migrate_test.go`

- [ ] **Step 1: Write failing API helper tests for structure/status/cancel paths**

```go
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
```

- [ ] **Step 2: Run the focused API helper tests and confirm they fail**

Run: `go test ./internal/cli -run 'TestStatusPageMigrationAPI(StartStructure|GetStatus|Cancel)'`
Expected: FAIL with undefined `statusPageMigrationAPI` and related migration methods/types.

- [ ] **Step 3: Implement the migration API helper and config resolution**

```go
type statusPageMigrationAPI struct {
	httpClient *http.Client
	baseURL    string
	appKey     string
	userAgent  string
}

type migrationStartResult struct {
	JobID string `json:"job_id"`
}

type migrationProgress struct {
	TotalSteps           int      `json:"total_steps"`
	CompletedSteps       int      `json:"completed_steps"`
	ComponentsImported   int      `json:"components_imported"`
	SectionsImported     int      `json:"sections_imported"`
	IncidentsImported    int      `json:"incidents_imported"`
	MaintenancesImported int      `json:"maintenances_imported"`
	SubscribersImported  int      `json:"subscribers_imported"`
	SubscribersSkipped   int      `json:"subscribers_skipped"`
	TemplatesImported    int      `json:"templates_imported"`
	Warnings             []string `json:"warnings,omitempty"`
}

type migrationJob struct {
	JobID        string            `json:"job_id"`
	SourcePageID string            `json:"source_page_id"`
	TargetPageID int64             `json:"target_page_id"`
	Phase        string            `json:"phase"`
	Status       string            `json:"status"`
	Progress     migrationProgress `json:"progress"`
	Error        string            `json:"error,omitempty"`
	CreatedAt    int64             `json:"created_at"`
	UpdatedAt    int64             `json:"updated_at"`
}

func newStatusPageMigrationAPI() (*statusPageMigrationAPI, error) {
	cfg, err := loadResolvedConfig()
	if err != nil {
		return nil, err
	}
	if cfg.AppKey == "" {
		return nil, fmt.Errorf("no app key configured. Run 'flashduty login' or set FLASHDUTY_APP_KEY")
	}

	return &statusPageMigrationAPI{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		appKey:     cfg.AppKey,
		userAgent:  "flashduty-cli/" + versionStr,
	}, nil
}
```

- [ ] **Step 4: Re-run the focused API helper tests and confirm they pass**

Run: `go test ./internal/cli -run 'TestStatusPageMigrationAPI(StartStructure|GetStatus|Cancel)'`
Expected: PASS

- [ ] **Step 5: Commit the helper foundation**

```bash
git add internal/cli/root.go internal/cli/status_page_migrate.go internal/cli/status_page_migrate_test.go
git commit -m "feat: add statuspage migration api helper"
```

### Task 2: Add `statuspage migrate` Cobra Commands

**Files:**
- Modify: `internal/cli/status_page.go`
- Modify: `internal/cli/status_page_migrate.go`
- Test: `internal/cli/status_page_migrate_test.go`

- [ ] **Step 1: Write failing command tests for structure and email-subscribers output**

```go
func TestStatusPageMigrateStructureCommandPrintsStatusHint(t *testing.T) {
	t.Parallel()

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
```

- [ ] **Step 2: Run the new command tests and confirm they fail**

Run: `go test ./internal/cli -run 'TestStatusPageMigrate(StructureCommandPrintsStatusHint|EmailSubscribersCommandPrintsStatusHint)'`
Expected: FAIL with undefined migrate command constructors/service injection points.

- [ ] **Step 3: Implement the command tree and simple async-job UX**

```go
func newStatusPageMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage status page migration jobs",
	}
	cmd.AddCommand(newStatusPageMigrateStructureCmd())
	cmd.AddCommand(newStatusPageMigrateEmailSubscribersCmd())
	cmd.AddCommand(newStatusPageMigrateStatusCmd())
	cmd.AddCommand(newStatusPageMigrateCancelCmd())
	return cmd
}

func printMigrationStart(cmd *cobra.Command, migrationType, source, sourcePageID string, targetPageID int64, result *migrationStartResult) error {
	if flagJSON {
		payload := map[string]any{
			"type":           migrationType,
			"source":         source,
			"source_page_id": sourcePageID,
			"job_id":         result.JobID,
		}
		if targetPageID > 0 {
			payload["target_page_id"] = targetPageID
		}
		return newPrinter(cmd.OutOrStdout()).Print(payload, nil)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Migration started.")
	fmt.Fprintf(out, "Type: %s\n", migrationType)
	fmt.Fprintf(out, "Source: %s\n", source)
	fmt.Fprintf(out, "Source page: %s\n", sourcePageID)
	if targetPageID > 0 {
		fmt.Fprintf(out, "Target page ID: %d\n", targetPageID)
	}
	fmt.Fprintf(out, "Job ID: %s\n\n", result.JobID)
	fmt.Fprintln(out, "Check progress with:")
	fmt.Fprintf(out, "  flashduty statuspage migrate status --job-id %s\n", result.JobID)
	return nil
}
```

- [ ] **Step 4: Re-run the command tests and confirm they pass**

Run: `go test ./internal/cli -run 'TestStatusPageMigrate(StructureCommandPrintsStatusHint|EmailSubscribersCommandPrintsStatusHint)'`
Expected: PASS

- [ ] **Step 5: Commit the command layer**

```bash
git add internal/cli/status_page.go internal/cli/status_page_migrate.go internal/cli/status_page_migrate_test.go
git commit -m "feat: add statuspage migrate commands"
```

### Task 3: Add Status/Cancel UX and Update Documentation

**Files:**
- Modify: `internal/cli/status_page_migrate.go`
- Modify: `README.md`
- Test: `internal/cli/status_page_migrate_test.go`

- [ ] **Step 1: Write failing tests for status output and cancel guidance**

```go
func TestStatusPageMigrateStatusCommandPrintsJobDetails(t *testing.T) {
	t.Parallel()

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
```

- [ ] **Step 2: Run the new status/cancel/documentation-related tests and confirm they fail**

Run: `go test ./internal/cli -run 'TestStatusPageMigrate(StatusCommandPrintsJobDetails|CancelCommandPrintsStatusHint)'`
Expected: FAIL until status/cancel formatting is implemented.

- [ ] **Step 3: Implement status/cancel formatting and README command docs**

```go
func printMigrationStatus(cmd *cobra.Command, job *migrationJob) error {
	if flagJSON {
		return newPrinter(cmd.OutOrStdout()).Print(job, nil)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Job ID: %s\n", job.JobID)
	fmt.Fprintf(out, "Source page: %s\n", job.SourcePageID)
	if job.TargetPageID > 0 {
		fmt.Fprintf(out, "Target page ID: %d\n", job.TargetPageID)
	}
	fmt.Fprintf(out, "Phase: %s\n", job.Phase)
	fmt.Fprintf(out, "Status: %s\n", job.Status)
	fmt.Fprintf(out, "Progress: %d/%d\n", job.Progress.CompletedSteps, job.Progress.TotalSteps)
	fmt.Fprintf(out, "Sections imported: %d\n", job.Progress.SectionsImported)
	fmt.Fprintf(out, "Components imported: %d\n", job.Progress.ComponentsImported)
	fmt.Fprintf(out, "Incidents imported: %d\n", job.Progress.IncidentsImported)
	fmt.Fprintf(out, "Maintenances imported: %d\n", job.Progress.MaintenancesImported)
	fmt.Fprintf(out, "Subscribers imported: %d\n", job.Progress.SubscribersImported)
	fmt.Fprintf(out, "Subscribers skipped: %d\n", job.Progress.SubscribersSkipped)
	fmt.Fprintf(out, "Templates imported: %d\n", job.Progress.TemplatesImported)
	if job.Error != "" {
		fmt.Fprintf(out, "Error: %s\n", job.Error)
	}
	if len(job.Progress.Warnings) > 0 {
		fmt.Fprintln(out, "Warnings:")
		for _, warning := range job.Progress.Warnings {
			fmt.Fprintf(out, "- %s\n", warning)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run the package tests and targeted CLI build checks**

Run: `go test ./internal/cli`
Expected: PASS

Run: `go test ./...`
Expected: PASS

Run: `go build ./cmd/flashduty`
Expected: PASS

- [ ] **Step 5: Commit the status/cancel/docs work**

```bash
git add README.md internal/cli/status_page_migrate.go internal/cli/status_page_migrate_test.go
git commit -m "feat: document statuspage migration workflow"
```
