package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const migrationSourceAtlassian = "atlassian"
const migrationErrorBodyLimit = 4096

var migrationSensitiveBodyKeys = map[string]struct{}{
	"apikey":        {},
	"xapikey":       {},
	"accesskey":     {},
	"password":      {},
	"passwd":        {},
	"pwd":           {},
	"token":         {},
	"accesstoken":   {},
	"refreshtoken":  {},
	"idtoken":       {},
	"sessiontoken":  {},
	"authtoken":     {},
	"oauthtoken":    {},
	"bearertoken":   {},
	"authorization": {},
	"auth":          {},
	"secret":        {},
	"clientsecret":  {},
	"secretkey":     {},
	"privatekey":    {},
	"signingkey":    {},
	"credential":    {},
	"credentials":   {},
}

type statusPageMigrationService interface {
	StartStructure(ctx context.Context, sourceAPIKey, sourcePageID string) (*migrationStartResult, error)
	StartEmailSubscribers(ctx context.Context, sourceAPIKey, sourcePageID string, targetPageID int64) (*migrationStartResult, error)
	GetStatus(ctx context.Context, jobID string) (*migrationJob, error)
	Cancel(ctx context.Context, jobID string) error
}

var newStatusPageMigrationService = func() (statusPageMigrationService, error) {
	return newStatusPageMigrationAPI()
}

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

type migrationEnvelope[T any] struct {
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Data *T `json:"data,omitempty"`
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

func (a *statusPageMigrationAPI) StartStructure(ctx context.Context, sourceAPIKey, sourcePageID string) (*migrationStartResult, error) {
	return a.postStart(ctx, "/status-page/migrate-structure", map[string]any{
		"api_key":        sourceAPIKey,
		"source_page_id": sourcePageID,
	})
}

func (a *statusPageMigrationAPI) StartEmailSubscribers(ctx context.Context, sourceAPIKey, sourcePageID string, targetPageID int64) (*migrationStartResult, error) {
	return a.postStart(ctx, "/status-page/migrate-email-subscribers", map[string]any{
		"api_key":        sourceAPIKey,
		"source_page_id": sourcePageID,
		"target_page_id": targetPageID,
	})
}

func (a *statusPageMigrationAPI) GetStatus(ctx context.Context, jobID string) (*migrationJob, error) {
	query := url.Values{}
	query.Set("job_id", jobID)

	var result migrationEnvelope[migrationJob]
	if err := a.do(ctx, http.MethodGet, "/status-page/migration/status", query, nil, &result); err != nil {
		return nil, err
	}
	if result.Data == nil {
		return nil, fmt.Errorf("migration status response missing data")
	}
	return result.Data, nil
}

func (a *statusPageMigrationAPI) Cancel(ctx context.Context, jobID string) error {
	var result migrationEnvelope[map[string]any]
	return a.do(ctx, http.MethodPost, "/status-page/migration/cancel", nil, map[string]any{
		"job_id": jobID,
	}, &result)
}

func (a *statusPageMigrationAPI) postStart(ctx context.Context, path string, body map[string]any) (*migrationStartResult, error) {
	var result migrationEnvelope[migrationStartResult]
	if err := a.do(ctx, http.MethodPost, path, nil, body, &result); err != nil {
		return nil, err
	}
	if result.Data == nil {
		return nil, fmt.Errorf("migration start response missing data")
	}
	return result.Data, nil
}

func (a *statusPageMigrationAPI) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	fullURL, err := url.Parse(a.baseURL + path)
	if err != nil {
		return fmt.Errorf("parse request URL: %w", err)
	}

	values := fullURL.Query()
	values.Set("app_key", a.appKey)
	for key, items := range query {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	fullURL.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, method, fullURL.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if a.userAgent != "" {
		req.Header.Set("User-Agent", a.userAgent)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %s", redactAppKey(err.Error(), a.appKey))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, migrationErrorBodyLimit))
		if readErr != nil {
			return fmt.Errorf("API client error (HTTP %d)", resp.StatusCode)
		}
		return fmt.Errorf("API client error (HTTP %d): %s", resp.StatusCode, sanitizeMigrationBody(string(bodyBytes)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	switch envelope := out.(type) {
	case *migrationEnvelope[migrationStartResult]:
		if envelope.Error != nil {
			return fmt.Errorf("%s: %s", envelope.Error.Code, envelope.Error.Message)
		}
	case *migrationEnvelope[migrationJob]:
		if envelope.Error != nil {
			return fmt.Errorf("%s: %s", envelope.Error.Code, envelope.Error.Message)
		}
	case *migrationEnvelope[map[string]any]:
		if envelope.Error != nil {
			return fmt.Errorf("%s: %s", envelope.Error.Code, envelope.Error.Message)
		}
	}

	return nil
}

func redactAppKey(message, appKey string) string {
	if appKey == "" {
		return message
	}

	redacted := strings.ReplaceAll(message, appKey, "[redacted]")
	redacted = strings.ReplaceAll(redacted, url.QueryEscape(appKey), "[redacted]")
	return redacted
}

func sanitizeMigrationBody(body string) string {
	if body == "" {
		return body
	}

	var v any
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return body
	}

	sanitized, redacted := sanitizeMigrationJSONValue(v)
	if !redacted {
		return body
	}

	out, err := json.Marshal(sanitized)
	if err != nil {
		return body
	}
	return string(out)
}

func sanitizeMigrationJSONValue(v any) (any, bool) {
	switch value := v.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(value))
		redacted := false
		for key, item := range value {
			if isMigrationSensitiveBodyKey(key) {
				sanitized[key] = "[REDACTED]"
				redacted = true
				continue
			}

			sanitizedItem, itemRedacted := sanitizeMigrationJSONValue(item)
			sanitized[key] = sanitizedItem
			redacted = redacted || itemRedacted
		}
		return sanitized, redacted
	case []any:
		sanitized := make([]any, len(value))
		redacted := false
		for i, item := range value {
			sanitizedItem, itemRedacted := sanitizeMigrationJSONValue(item)
			sanitized[i] = sanitizedItem
			redacted = redacted || itemRedacted
		}
		return sanitized, redacted
	default:
		return v, false
	}
}

func isMigrationSensitiveBodyKey(key string) bool {
	_, ok := migrationSensitiveBodyKeys[normalizeMigrationSensitiveBodyKey(key)]
	return ok
}

func normalizeMigrationSensitiveBodyKey(key string) string {
	var b strings.Builder
	b.Grow(len(key))
	for _, r := range strings.ToLower(strings.TrimSpace(key)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

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

func newStatusPageMigrateStructureCmd() *cobra.Command {
	var source string
	var sourcePageID string
	var sourceAPIKey string

	cmd := &cobra.Command{
		Use:   "structure",
		Short: "Start structure and history migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateMigrationSource(source); err != nil {
				return err
			}

			service, err := newStatusPageMigrationService()
			if err != nil {
				return err
			}

			result, err := service.StartStructure(cmdContext(cmd), sourceAPIKey, sourcePageID)
			if err != nil {
				return err
			}

			return printMigrationStart(cmd, "structure", source, sourcePageID, 0, result)
		},
	}

	cmd.Flags().StringVar(&source, "from", "", "Migration source provider (required)")
	cmd.Flags().StringVar(&sourcePageID, "source-page-id", "", "Source page ID in the provider (required)")
	cmd.Flags().StringVar(&sourceAPIKey, "api-key", "", "Source provider API key (required)")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("source-page-id")
	_ = cmd.MarkFlagRequired("api-key")

	return cmd
}

func newStatusPageMigrateEmailSubscribersCmd() *cobra.Command {
	var source string
	var sourcePageID string
	var sourceAPIKey string
	var targetPageID int64

	cmd := &cobra.Command{
		Use:   "email-subscribers",
		Short: "Start email subscriber migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateMigrationSource(source); err != nil {
				return err
			}

			service, err := newStatusPageMigrationService()
			if err != nil {
				return err
			}

			result, err := service.StartEmailSubscribers(cmdContext(cmd), sourceAPIKey, sourcePageID, targetPageID)
			if err != nil {
				return err
			}

			return printMigrationStart(cmd, "email-subscribers", source, sourcePageID, targetPageID, result)
		},
	}

	cmd.Flags().StringVar(&source, "from", "", "Migration source provider (required)")
	cmd.Flags().StringVar(&sourcePageID, "source-page-id", "", "Source page ID in the provider (required)")
	cmd.Flags().StringVar(&sourceAPIKey, "api-key", "", "Source provider API key (required)")
	cmd.Flags().Int64Var(&targetPageID, "target-page-id", 0, "Target Flashduty status page ID (required)")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("source-page-id")
	_ = cmd.MarkFlagRequired("api-key")
	_ = cmd.MarkFlagRequired("target-page-id")

	return cmd
}

func newStatusPageMigrateStatusCmd() *cobra.Command {
	var jobID string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show migration job status",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := newStatusPageMigrationService()
			if err != nil {
				return err
			}

			job, err := service.GetStatus(cmdContext(cmd), jobID)
			if err != nil {
				return err
			}

			return printMigrationStatus(cmd, job)
		},
	}

	cmd.Flags().StringVar(&jobID, "job-id", "", "Migration job ID (required)")
	_ = cmd.MarkFlagRequired("job-id")

	return cmd
}

func newStatusPageMigrateCancelCmd() *cobra.Command {
	var jobID string

	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a running migration job",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := newStatusPageMigrationService()
			if err != nil {
				return err
			}

			if err := service.Cancel(cmdContext(cmd), jobID); err != nil {
				return err
			}

			if flagJSON {
				return newPrinter(cmd.OutOrStdout()).Print(map[string]any{
					"job_id":  jobID,
					"status":  "cancel_requested",
					"command": "flashduty statuspage migrate status --job-id " + jobID,
				}, nil)
			}

			out := cmd.OutOrStdout()
			if _, err := fmt.Fprintln(out, "Cancellation requested."); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "Job ID: %s\n\n", jobID); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out, "Check progress with:"); err != nil {
				return err
			}
			_, err = fmt.Fprintf(out, "  flashduty statuspage migrate status --job-id %s\n", jobID)
			return err
		},
	}

	cmd.Flags().StringVar(&jobID, "job-id", "", "Migration job ID (required)")
	_ = cmd.MarkFlagRequired("job-id")

	return cmd
}

func validateMigrationSource(source string) error {
	if source != migrationSourceAtlassian {
		return fmt.Errorf("unsupported migration source: %q (supported: %s)", source, migrationSourceAtlassian)
	}
	return nil
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
		payload["next_command"] = "flashduty statuspage migrate status --job-id " + result.JobID
		return newPrinter(cmd.OutOrStdout()).Print(payload, nil)
	}

	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintln(out, "Migration started."); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Type: %s\n", migrationType); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Source: %s\n", source); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Source page: %s\n", sourcePageID); err != nil {
		return err
	}
	if targetPageID > 0 {
		if _, err := fmt.Fprintf(out, "Target page ID: %d\n", targetPageID); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(out, "Job ID: %s\n\n", result.JobID); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "Check progress with:"); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "  flashduty statuspage migrate status --job-id %s\n", result.JobID)
	return err
}

func printMigrationStatus(cmd *cobra.Command, job *migrationJob) error {
	if flagJSON {
		return newPrinter(cmd.OutOrStdout()).Print(job, nil)
	}

	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(out, "Job ID: %s\n", job.JobID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Source page: %s\n", job.SourcePageID); err != nil {
		return err
	}
	if job.TargetPageID > 0 {
		if _, err := fmt.Fprintf(out, "Target page ID: %d\n", job.TargetPageID); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(out, "Phase: %s\n", job.Phase); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Status: %s\n", job.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Progress: %d/%d\n", job.Progress.CompletedSteps, job.Progress.TotalSteps); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Sections imported: %d\n", job.Progress.SectionsImported); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Components imported: %d\n", job.Progress.ComponentsImported); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Incidents imported: %d\n", job.Progress.IncidentsImported); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Maintenances imported: %d\n", job.Progress.MaintenancesImported); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Subscribers imported: %d\n", job.Progress.SubscribersImported); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Subscribers skipped: %d\n", job.Progress.SubscribersSkipped); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Templates imported: %d\n", job.Progress.TemplatesImported); err != nil {
		return err
	}
	if job.Error != "" {
		if _, err := fmt.Fprintf(out, "Error: %s\n", job.Error); err != nil {
			return err
		}
	}
	if len(job.Progress.Warnings) > 0 {
		if _, err := fmt.Fprintln(out, "Warnings:"); err != nil {
			return err
		}
		for _, warning := range job.Progress.Warnings {
			if _, err := fmt.Fprintf(out, "- %s\n", warning); err != nil {
				return err
			}
		}
	}
	return nil
}
