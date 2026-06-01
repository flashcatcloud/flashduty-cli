package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flashcatcloud/go-flashduty"
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

				result, _, err := ctx.Client.StatusPages.ReadPageList(cmdContext(ctx.Cmd))
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
					filtered := make([]flashduty.StatusPageItem, 0, len(pages))
					for _, p := range pages {
						if _, ok := want[p.PageID]; ok {
							filtered = append(filtered, p)
						}
					}
					pages = filtered
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return strconv.FormatInt(v.(flashduty.StatusPageItem).PageID, 10) }},
					{Header: "NAME", Field: func(v any) string { return v.(flashduty.StatusPageItem).Name }},
					{Header: "SLUG", Field: func(v any) string { return v.(flashduty.StatusPageItem).URLName }},
					// STATUS reads the account's overall_status, which the
					// /status-page/list endpoint does not return. The legacy SDK
					// likewise never populated it, so this column stays empty —
					// preserved here to keep the table shape identical.
					{Header: "STATUS", Field: func(v any) string { return "" }},
					{Header: "COMPONENTS", Field: func(v any) string {
						comps := v.(flashduty.StatusPageItem).Components
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, _, err := ctx.Client.StatusPages.ChangeActiveList(cmdContext(ctx.Cmd), &flashduty.StatusPagesChangeActiveListRequest{
					PageID: pageID,
					Type:   changeType,
				})
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return strconv.FormatInt(v.(flashduty.StatusPageChangeItem).ChangeID, 10) }},
					{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.StatusPageChangeItem).Title }},
					{Header: "TYPE", Field: func(v any) string { return v.(flashduty.StatusPageChangeItem).Type }},
					{Header: "STATUS", Field: func(v any) string { return v.(flashduty.StatusPageChangeItem).Status }},
					// The active-list endpoint returns the event's scheduled window
					// (start_at_seconds / close_at_seconds), not the row's created/
					// updated timestamps the legacy SDK reported. The CREATED/UPDATED
					// headers are preserved to keep the table shape identical; they now
					// reflect the event start and (scheduled) close times.
					{Header: "CREATED", Field: func(v any) string { return output.FormatTime(v.(flashduty.StatusPageChangeItem).StartAtSeconds) }},
					{Header: "UPDATED", Field: func(v any) string { return output.FormatTime(v.(flashduty.StatusPageChangeItem).CloseAtSeconds) }},
				}

				return ctx.Printer.Print(result.Items, cols)
			})
		},
	}

	cmd.Flags().Int64Var(&pageID, "page-id", 0, "Page ID (required)")
	cmd.Flags().StringVar(&changeType, "type", "", "Change type: incident or maintenance (required)")
	registerEnumFlag(cmd, "type", "incident", "maintenance")
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
				// Replicate the legacy SDK's request shaping exactly: default the
				// status to "investigating", build a single timeline update carrying
				// the message and any parsed component_changes, and fall back to the
				// title when no message was supplied. This keeps the wire payload
				// byte-for-byte equivalent so the migration introduces no drift.
				const status = "investigating"

				update := flashduty.CreateStatusPageChangeRequestUpdatesItem{
					AtSeconds: time.Now().Unix(),
					Status:    status,
				}
				if message != "" {
					update.Description = message
				}
				if components != "" {
					for _, part := range parseStringSlice(components) {
						kv := strings.SplitN(part, ":", 2)
						if len(kv) == 2 {
							update.ComponentChanges = append(update.ComponentChanges, flashduty.CreateStatusPageChangeRequestUpdatesItemComponentChangesItem{
								ComponentID: strings.TrimSpace(kv[0]),
								Status:      strings.TrimSpace(kv[1]),
							})
						} else if len(kv) == 1 && kv[0] != "" {
							update.ComponentChanges = append(update.ComponentChanges, flashduty.CreateStatusPageChangeRequestUpdatesItemComponentChangesItem{
								ComponentID: strings.TrimSpace(kv[0]),
								Status:      "partial_outage",
							})
						}
					}
				}

				description := message
				if description == "" {
					description = title
				}

				result, _, err := ctx.Client.StatusPages.ChangeCreate(cmdContext(ctx.Cmd), &flashduty.CreateStatusPageChangeRequest{
					PageID:            pageID,
					Title:             title,
					Type:              "incident",
					Status:            status,
					Description:       description,
					Updates:           []flashduty.CreateStatusPageChangeRequestUpdatesItem{update},
					NotifySubscribers: notify,
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
			return runCommand(cmd, args, func(ctx *RunContext) error {
				_, _, err := ctx.Client.StatusPages.ChangeTimelineCreate(cmdContext(ctx.Cmd), &flashduty.CreateStatusPageChangeTimelineRequest{
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
