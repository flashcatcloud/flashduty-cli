package cli

import (
	"fmt"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

func newInsightCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "insight",
		Short: "Query aggregated incident metrics by team/responder/channel (preferred over incident list for analytics)",
	}
	// insight team/channel/responder are now served by the generated commands
	// (richer flag set: severities, *_ids, fields, aggregate-unit, …; relative
	// time on --start-time/--end-time). Their human tables are preserved via the
	// DimensionInsightItem / ResponderInsightItem entries in display_columns.go.
	cmd.AddCommand(newInsightTopAlertsCmd())
	cmd.AddCommand(newInsightIncidentsCmd())
	return cmd
}

func newInsightTopAlertsCmd() *cobra.Command {
	var label, since, until string
	var limit int

	cmd := &cobra.Command{
		Use:   "top-alerts",
		Short: "Query top alert sources by label",
		Long:  curatedLong("Query the top-K noisiest alert sources grouped by a label dimension over a time window.", "Analytics", "TopkAlertsByLabel"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				startTime, err := timeutil.Parse(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				endTime, err := timeutil.Parse(until)
				if err != nil {
					return fmt.Errorf("invalid --until: %w", err)
				}

				result, _, err := ctx.Client.Analytics.TopkAlertsByLabel(cmdContext(ctx.Cmd), &flashduty.InsightTopkAlertByLabelRequest{
					StartTime: startTime,
					EndTime:   endTime,
					Label:     label,
					K:         int64(limit),
				})
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "LABEL", MaxWidth: 50, Field: func(v any) string {
						return v.(flashduty.InsightAlertByLabelItem).Label
					}},
					{Header: "ALERTS", Field: func(v any) string {
						return fmt.Sprintf("%d", v.(flashduty.InsightAlertByLabelItem).TotalAlertCnt)
					}},
					{Header: "EVENTS", Field: func(v any) string {
						return fmt.Sprintf("%d", v.(flashduty.InsightAlertByLabelItem).TotalAlertEventCnt)
					}},
				}

				return ctx.PrintTotal(result.Items, cols, len(result.Items))
			})
		},
	}

	cmd.Flags().StringVar(&label, "label", "", "Group-by label dimension: one of [check, resource] (required)")
	cmd.Flags().StringVar(&since, "since", "7d", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")
	cmd.Flags().IntVar(&limit, "limit", 10, "Top K results")
	_ = cmd.MarkFlagRequired("label")

	return cmd
}

func newInsightIncidentsCmd() *cobra.Command {
	var since, until string
	var limit, page int

	cmd := &cobra.Command{
		Use:   "incidents",
		Short: "Query incidents with performance metrics",
		Long:  curatedLong("List incidents with per-incident performance metrics (MTTA, MTTR, notifications) over a time window.", "Analytics", "IncidentList"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				startTime, err := timeutil.Parse(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				endTime, err := timeutil.Parse(until)
				if err != nil {
					return fmt.Errorf("invalid --until: %w", err)
				}

				req := &flashduty.InsightIncidentListRequest{
					StartTime: startTime,
					EndTime:   endTime,
				}
				req.Limit = limit
				req.Page = page

				result, _, err := ctx.Client.Analytics.IncidentList(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string {
						return v.(flashduty.IncidentRawItem).IncidentID
					}},
					{Header: "TITLE", MaxWidth: 40, Field: func(v any) string {
						return v.(flashduty.IncidentRawItem).Title
					}},
					{Header: "SEVERITY", Field: func(v any) string {
						return v.(flashduty.IncidentRawItem).Severity
					}},
					{Header: "CHANNEL", MaxWidth: 20, Field: func(v any) string {
						return v.(flashduty.IncidentRawItem).ChannelName
					}},
					{Header: "MTTA", Field: func(v any) string {
						return output.FormatDuration(int(v.(flashduty.IncidentRawItem).SecondsToAck))
					}},
					{Header: "MTTR", Field: func(v any) string {
						return output.FormatDuration(int(v.(flashduty.IncidentRawItem).SecondsToClose))
					}},
					{Header: "NOTIFICATIONS", Field: func(v any) string {
						return fmt.Sprintf("%d", v.(flashduty.IncidentRawItem).Notifications)
					}},
				}

				return ctx.PrintList(result.Items, cols, len(result.Items), page, int(result.Total))
			})
		},
	}

	cmd.Flags().StringVar(&since, "since", "7d", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}
