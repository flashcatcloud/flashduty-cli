package cli

import (
	"fmt"
	"strconv"
	"strings"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	gflashduty "github.com/flashcatcloud/go-flashduty"
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
			return runGFCommand(cmd, args, func(ctx *RunContext) error {
				pageIDs, err := parseIntSlice(ids)
				if err != nil {
					return fmt.Errorf("invalid --id: %w", err)
				}

				result, _, err := ctx.GFClient.StatusPages.ReadPageList(cmdContext(ctx.Cmd))
				if err != nil {
					return err
				}

				// ReadPageList lists every status page; the legacy SDK supported a
				// server-side page-id filter, so preserve --id by filtering here.
				pages := result.Items
				if len(pageIDs) > 0 {
					want := make(map[int64]struct{}, len(pageIDs))
					for _, id := range pageIDs {
						want[id] = struct{}{}
					}
					filtered := make([]gflashduty.StatusPageItem, 0, len(pages))
					for _, p := range pages {
						if _, ok := want[p.PageID]; ok {
							filtered = append(filtered, p)
						}
					}
					pages = filtered
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return strconv.FormatInt(v.(gflashduty.StatusPageItem).PageID, 10) }},
					{Header: "NAME", Field: func(v any) string { return v.(gflashduty.StatusPageItem).Name }},
					{Header: "SLUG", Field: func(v any) string { return v.(gflashduty.StatusPageItem).URLName }},
					// STATUS reads the account's overall_status, which the
					// /status-page/list endpoint does not return. The legacy SDK
					// likewise never populated it, so this column stays empty —
					// preserved here to keep the table shape identical.
					{Header: "STATUS", Field: func(v any) string { return "" }},
					{Header: "COMPONENTS", Field: func(v any) string {
						comps := v.(gflashduty.StatusPageItem).Components
						names := make([]string, 0, len(comps))
						for _, c := range comps {
							names = append(names, c.Name)
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
		// TODO(go-flashduty migration): not migrated. This lists *active* changes
		// via /status-page/change/active/list. go-flashduty v0.4.0 only covers the
		// general /status-page/change/list (StatusPages.ChangeList), which has
		// different semantics (no active filter) and requires a status argument.
		// Kept on the legacy SDK until the active-list endpoint is documented.
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

				if result != nil && result.ChangeID != 0 {
					ctx.WriteResult(fmt.Sprintf("Status incident created: %d", result.ChangeID))
					return nil
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
			return runGFCommand(cmd, args, func(ctx *RunContext) error {
				_, _, err := ctx.GFClient.StatusPages.ChangeTimelineCreate(cmdContext(ctx.Cmd), &gflashduty.CreateStatusPageChangeTimelineRequest{
					PageID:      pageID,
					ChangeID:    changeID,
					Description: message,
					Status:      status,
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
