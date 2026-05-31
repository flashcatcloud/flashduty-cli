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
		Short: "Query insight metrics",
	}
	cmd.AddCommand(newInsightTeamCmd())
	cmd.AddCommand(newInsightChannelCmd())
	cmd.AddCommand(newInsightResponderCmd())
	cmd.AddCommand(newInsightTopAlertsCmd())
	cmd.AddCommand(newInsightIncidentsCmd())
	return cmd
}

func newInsightTeamCmd() *cobra.Command {
	var since, until string

	cmd := &cobra.Command{
		Use:   "team",
		Short: "Query insights by team",
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

				result, _, err := ctx.Client.Analytics.ByTeam(cmdContext(ctx.Cmd), &flashduty.InsightQueryRequest{
					StartTime: startTime,
					EndTime:   endTime,
				})
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "TEAM", MaxWidth: 30, Field: func(v any) string {
						return v.(flashduty.DimensionInsightItem).TeamName
					}},
					{Header: "INCIDENTS", Field: func(v any) string {
						return fmt.Sprintf("%d", v.(flashduty.DimensionInsightItem).TotalIncidentCnt)
					}},
					{Header: "ACK%", Field: func(v any) string {
						return fmt.Sprintf("%.0f%%", v.(flashduty.DimensionInsightItem).AcknowledgementPct*100)
					}},
					{Header: "MTTA", Field: func(v any) string {
						return output.FormatDurationFloat(v.(flashduty.DimensionInsightItem).MeanSecondsToAck)
					}},
					{Header: "MTTR", Field: func(v any) string {
						return output.FormatDurationFloat(v.(flashduty.DimensionInsightItem).MeanSecondsToClose)
					}},
					{Header: "NOISE_REDUCTION", Field: func(v any) string {
						return fmt.Sprintf("%.0f%%", v.(flashduty.DimensionInsightItem).NoiseReductionPct*100)
					}},
					{Header: "ALERTS", Field: func(v any) string {
						return fmt.Sprintf("%d", v.(flashduty.DimensionInsightItem).TotalAlertCnt)
					}},
					{Header: "EVENTS", Field: func(v any) string {
						return fmt.Sprintf("%d", v.(flashduty.DimensionInsightItem).TotalAlertEventCnt)
					}},
				}

				return ctx.PrintTotal(result.Items, cols, len(result.Items))
			})
		},
	}

	cmd.Flags().StringVar(&since, "since", "7d", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")

	return cmd
}

func newInsightChannelCmd() *cobra.Command {
	var since, until string

	cmd := &cobra.Command{
		Use:   "channel",
		Short: "Query insights by channel",
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

				result, _, err := ctx.Client.Analytics.ByChannel(cmdContext(ctx.Cmd), &flashduty.InsightQueryRequest{
					StartTime: startTime,
					EndTime:   endTime,
				})
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "CHANNEL", MaxWidth: 30, Field: func(v any) string {
						return v.(flashduty.DimensionInsightItem).ChannelName
					}},
					{Header: "INCIDENTS", Field: func(v any) string {
						return fmt.Sprintf("%d", v.(flashduty.DimensionInsightItem).TotalIncidentCnt)
					}},
					{Header: "ACK%", Field: func(v any) string {
						return fmt.Sprintf("%.0f%%", v.(flashduty.DimensionInsightItem).AcknowledgementPct*100)
					}},
					{Header: "MTTA", Field: func(v any) string {
						return output.FormatDurationFloat(v.(flashduty.DimensionInsightItem).MeanSecondsToAck)
					}},
					{Header: "MTTR", Field: func(v any) string {
						return output.FormatDurationFloat(v.(flashduty.DimensionInsightItem).MeanSecondsToClose)
					}},
					{Header: "NOISE_REDUCTION", Field: func(v any) string {
						return fmt.Sprintf("%.0f%%", v.(flashduty.DimensionInsightItem).NoiseReductionPct*100)
					}},
					{Header: "ALERTS", Field: func(v any) string {
						return fmt.Sprintf("%d", v.(flashduty.DimensionInsightItem).TotalAlertCnt)
					}},
					{Header: "EVENTS", Field: func(v any) string {
						return fmt.Sprintf("%d", v.(flashduty.DimensionInsightItem).TotalAlertEventCnt)
					}},
				}

				return ctx.PrintTotal(result.Items, cols, len(result.Items))
			})
		},
	}

	cmd.Flags().StringVar(&since, "since", "7d", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")

	return cmd
}

func newInsightResponderCmd() *cobra.Command {
	var since, until string

	cmd := &cobra.Command{
		Use:   "responder",
		Short: "Query insights by responder",
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

				result, _, err := ctx.Client.Analytics.ByResponder(cmdContext(ctx.Cmd), &flashduty.InsightQueryRequest{
					StartTime: startTime,
					EndTime:   endTime,
				})
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "RESPONDER", MaxWidth: 30, Field: func(v any) string {
						return v.(flashduty.ResponderInsightItem).ResponderName
					}},
					{Header: "INCIDENTS", Field: func(v any) string {
						return fmt.Sprintf("%d", v.(flashduty.ResponderInsightItem).TotalIncidentCnt)
					}},
					{Header: "ACK%", Field: func(v any) string {
						return fmt.Sprintf("%.0f%%", v.(flashduty.ResponderInsightItem).AcknowledgementPct*100)
					}},
					{Header: "MTTA", Field: func(v any) string {
						return output.FormatDurationFloat(v.(flashduty.ResponderInsightItem).MeanSecondsToAck)
					}},
					{Header: "INTERRUPTIONS", Field: func(v any) string {
						return fmt.Sprintf("%d", v.(flashduty.ResponderInsightItem).TotalInterruptions)
					}},
					{Header: "ENGAGED", Field: func(v any) string {
						return output.FormatDuration(int(v.(flashduty.ResponderInsightItem).TotalEngagedSeconds))
					}},
				}

				return ctx.PrintTotal(result.Items, cols, len(result.Items))
			})
		},
	}

	cmd.Flags().StringVar(&since, "since", "7d", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")

	return cmd
}

func newInsightTopAlertsCmd() *cobra.Command {
	var label, since, until string
	var limit int

	cmd := &cobra.Command{
		Use:   "top-alerts",
		Short: "Query top alert sources by label",
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

	cmd.Flags().StringVar(&label, "label", "", "Label key to group by (e.g., \"integration_name\")")
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
