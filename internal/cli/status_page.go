package cli

import (
	"fmt"
	"strconv"
	"strings"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

func newStatusPageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "statuspage",
		Short: "Manage status pages",
	}
	cmd.AddCommand(newStatusPageListCmd())
	cmd.AddCommand(newStatusPageChangesCmd())
	cmd.AddCommand(newStatusPageCreateIncidentCmd())
	cmd.AddCommand(newStatusPageCreateTimelineCmd())
	cmd.AddCommand(newStatusPageMigrateCmd())
	return cmd
}

func newStatusPageListCmd() *cobra.Command {
	var ids string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List status pages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				pageIDs, err := parseIntSlice(ids)
				if err != nil {
					return fmt.Errorf("invalid --id: %w", err)
				}

				pages, err := ctx.Client.ListStatusPages(cmdContext(ctx.Cmd), pageIDs)
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return strconv.FormatInt(v.(flashduty.StatusPage).PageID, 10) }},
					{Header: "NAME", Field: func(v any) string { return v.(flashduty.StatusPage).PageName }},
					{Header: "SLUG", Field: func(v any) string { return v.(flashduty.StatusPage).Slug }},
					{Header: "STATUS", Field: func(v any) string { return v.(flashduty.StatusPage).OverallStatus }},
					{Header: "COMPONENTS", Field: func(v any) string {
						comps := v.(flashduty.StatusPage).Components
						names := make([]string, 0, len(comps))
						for _, c := range comps {
							names = append(names, c.ComponentName)
						}
						return strings.Join(names, ", ")
					}},
				}

				return ctx.Printer.Print(pages, cols)
			})
		},
	}

	cmd.Flags().StringVar(&ids, "id", "", "Filter by page IDs (comma-separated)")

	return cmd
}

func newStatusPageChangesCmd() *cobra.Command {
	var pageID int64
	var changeType string

	cmd := &cobra.Command{
		Use:   "changes",
		Short: "List active status page changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.ListStatusChanges(cmdContext(ctx.Cmd), &flashduty.ListStatusChangesInput{
					PageID:     pageID,
					ChangeType: changeType,
				})
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return strconv.FormatInt(v.(flashduty.StatusChange).ChangeID, 10) }},
					{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.StatusChange).Title }},
					{Header: "TYPE", Field: func(v any) string { return v.(flashduty.StatusChange).Type }},
					{Header: "STATUS", Field: func(v any) string { return v.(flashduty.StatusChange).Status }},
					{Header: "CREATED", Field: func(v any) string { return output.FormatTime(v.(flashduty.StatusChange).CreatedAt) }},
					{Header: "UPDATED", Field: func(v any) string { return output.FormatTime(v.(flashduty.StatusChange).UpdatedAt) }},
				}

				return ctx.Printer.Print(result.Changes, cols)
			})
		},
	}

	cmd.Flags().Int64Var(&pageID, "page-id", 0, "Page ID (required)")
	cmd.Flags().StringVar(&changeType, "type", "", "Change type: incident or maintenance (required)")
	_ = cmd.MarkFlagRequired("page-id")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}

func newStatusPageCreateIncidentCmd() *cobra.Command {
	var pageID int64
	var title, message, components string
	var notify bool

	cmd := &cobra.Command{
		Use:   "create-incident",
		Short: "Create a status page incident",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.CreateStatusIncident(cmdContext(ctx.Cmd), &flashduty.CreateStatusIncidentInput{
					PageID:             pageID,
					Title:              title,
					Message:            message,
					AffectedComponents: components,
					NotifySubscribers:  notify,
				})
				if err != nil {
					return err
				}

				if m, ok := result.(map[string]any); ok {
					if id, ok := m["change_id"]; ok {
						ctx.WriteResult(fmt.Sprintf("Status incident created: %v", id))
						return nil
					}
				}
				ctx.WriteResult("Status incident created successfully.")
				return nil
			})
		},
	}

	cmd.Flags().Int64Var(&pageID, "page-id", 0, "Page ID (required)")
	cmd.Flags().StringVar(&title, "title", "", "Title (required, max 255 chars)")
	cmd.Flags().StringVar(&message, "message", "", "Initial update message")
	cmd.Flags().StringVar(&components, "components", "", "Affected components (format: id1:status,id2:status; incident statuses: operational, degraded, partial_outage, full_outage; maintenance statuses: operational, under_maintenance)")
	cmd.Flags().BoolVar(&notify, "notify", false, "Notify subscribers")
	_ = cmd.MarkFlagRequired("page-id")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func newStatusPageCreateTimelineCmd() *cobra.Command {
	var pageID, changeID int64
	var message, status string

	cmd := &cobra.Command{
		Use:   "create-timeline",
		Short: "Add a timeline update to a status page change",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				err := ctx.Client.CreateChangeTimeline(cmdContext(ctx.Cmd), &flashduty.CreateChangeTimelineInput{
					PageID:   pageID,
					ChangeID: changeID,
					Message:  message,
					Status:   status,
				})
				if err != nil {
					return err
				}

				ctx.WriteResult("Timeline update added.")
				return nil
			})
		},
	}

	cmd.Flags().Int64Var(&pageID, "page-id", 0, "Page ID (required)")
	cmd.Flags().Int64Var(&changeID, "change", 0, "Change ID (required)")
	cmd.Flags().StringVar(&message, "message", "", "Message (required)")
	cmd.Flags().StringVar(&status, "status", "", "Status (incident: investigating, identified, monitoring, resolved; maintenance: scheduled, ongoing, completed)")
	_ = cmd.MarkFlagRequired("page-id")
	_ = cmd.MarkFlagRequired("change")
	_ = cmd.MarkFlagRequired("message")

	return cmd
}
