package cli

import (
	"fmt"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/spf13/cobra"
)

const migrationSourceAtlassian = "atlassian"

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
	var urlName string

	cmd := &cobra.Command{
		Use:   "structure",
		Short: "Start structure and history migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateMigrationSource(source); err != nil {
				return err
			}
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.StartStatusPageMigration(cmdContext(ctx.Cmd), &flashduty.StartStatusPageMigrationInput{
					SourceAPIKey: sourceAPIKey,
					SourcePageID: sourcePageID,
					URLName:      urlName,
				})
				if err != nil {
					return err
				}

				return printMigrationStart(ctx, "structure", source, sourcePageID, 0, result)
			})
		},
	}

	cmd.Flags().StringVar(&source, "from", "", "Migration source provider (required)")
	cmd.Flags().StringVar(&sourcePageID, "source-page-id", "", "Source page ID in the provider (required)")
	cmd.Flags().StringVar(&sourceAPIKey, "api-key", "", "Source provider API key (required)")
	cmd.Flags().StringVar(&urlName, "url-name", "", "Optional URL name for a newly created Flashduty public status page; fails if the source page is already mapped to a different URL name")
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
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.StartStatusPageEmailSubscriberMigration(cmdContext(ctx.Cmd), &flashduty.StartStatusPageEmailSubscriberMigrationInput{
					SourceAPIKey: sourceAPIKey,
					SourcePageID: sourcePageID,
					TargetPageID: targetPageID,
				})
				if err != nil {
					return err
				}

				return printMigrationStart(ctx, "email-subscribers", source, sourcePageID, targetPageID, result)
			})
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
			return runCommand(cmd, args, func(ctx *RunContext) error {
				job, err := ctx.Client.GetStatusPageMigrationStatus(cmdContext(ctx.Cmd), jobID)
				if err != nil {
					return err
				}

				return printMigrationStatus(ctx, job)
			})
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
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if err := ctx.Client.CancelStatusPageMigration(cmdContext(ctx.Cmd), jobID); err != nil {
					return err
				}

				if ctx.JSON {
					statusCmd := "flashduty statuspage migrate status --job-id " + jobID
					return ctx.Printer.Print(map[string]any{
						"job_id":       jobID,
						"status":       "cancel_requested",
						"command":      statusCmd,
						"next_command": statusCmd,
					}, nil)
				}

				out := ctx.Writer
				if _, err := fmt.Fprintln(out, "Cancellation requested."); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(out, "Job ID: %s\n\n", jobID); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(out, "Check progress with:"); err != nil {
					return err
				}
				_, err := fmt.Fprintf(out, "  flashduty statuspage migrate status --job-id %s\n", jobID)
				return err
			})
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

func printMigrationStart(ctx *RunContext, migrationType, source, sourcePageID string, targetPageID int64, result *flashduty.StartStatusPageMigrationOutput) error {
	if ctx.JSON {
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
		return ctx.Printer.Print(payload, nil)
	}

	out := ctx.Writer
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

func printMigrationStatus(ctx *RunContext, job *flashduty.StatusPageMigrationJob) error {
	if ctx.JSON {
		return ctx.Printer.Print(job, nil)
	}

	out := ctx.Writer
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
