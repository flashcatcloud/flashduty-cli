package cli

import (
	"fmt"
	"io"
	"strings"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

func newAlertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alert",
		Short: "Manage alerts",
	}
	cmd.AddCommand(newAlertListCmd())
	cmd.AddCommand(newAlertGetCmd())
	cmd.AddCommand(newAlertEventsCmd())
	cmd.AddCommand(newAlertTimelineCmd())
	cmd.AddCommand(newAlertMergeCmd())
	return cmd
}

func newAlertListCmd() *cobra.Command {
	var severity, channel, title, since, until string
	var active, recovered, muted bool
	var limit, page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List alerts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if active && recovered {
					return fmt.Errorf("--active and --recovered are mutually exclusive")
				}

				startTime, err := timeutil.Parse(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				endTime, err := timeutil.Parse(until)
				if err != nil {
					return fmt.Errorf("invalid --until: %w", err)
				}

				input := &flashduty.ListAlertsInput{
					StartTime:     startTime,
					EndTime:       endTime,
					AlertSeverity: severity,
					Title:         title,
					Limit:         limit,
					Page:          page,
				}

				if active {
					input.IsActive = boolPtr(true)
				} else if recovered {
					input.IsActive = boolPtr(false)
				}

				if muted {
					input.EverMuted = boolPtr(true)
				}

				if channel != "" {
					channelIDs, err := parseIntSlice(channel)
					if err != nil {
						return fmt.Errorf("invalid --channel: %w", err)
					}
					input.ChannelIDs = channelIDs
				}

				result, err := ctx.Client.ListAlerts(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return v.(flashduty.Alert).AlertID }},
					{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.Alert).Title }},
					{Header: "SEVERITY", Field: func(v any) string { return v.(flashduty.Alert).AlertSeverity }},
					{Header: "STATUS", Field: func(v any) string { return v.(flashduty.Alert).AlertStatus }},
					{Header: "EVENTS", Field: func(v any) string { return fmt.Sprintf("%d", v.(flashduty.Alert).EventCnt) }},
					{Header: "CHANNEL", Field: func(v any) string { return v.(flashduty.Alert).ChannelName }},
					{Header: "STARTED", Field: func(v any) string { return output.FormatTime(v.(flashduty.Alert).StartTime) }},
				}

				return ctx.PrintList(result.Alerts, cols, len(result.Alerts), page, result.Total)
			})
		},
	}

	cmd.Flags().StringVar(&severity, "severity", "", "Filter: Critical,Warning,Info")
	cmd.Flags().BoolVar(&active, "active", false, "Show active only")
	cmd.Flags().BoolVar(&recovered, "recovered", false, "Show recovered only")
	cmd.Flags().StringVar(&channel, "channel", "", "Comma-separated channel IDs")
	cmd.Flags().BoolVar(&muted, "muted", false, "Show ever-muted only")
	cmd.Flags().StringVar(&title, "title", "", "Search by title keyword")
	cmd.Flags().StringVar(&since, "since", "24h", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}

func newAlertGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <alert_id>",
		Short: "Get alert detail",
		Args:  requireArgs("alert_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.GetAlertDetail(cmdContext(ctx.Cmd), &flashduty.GetAlertDetailInput{
					AlertID: ctx.Args[0],
				})
				if err != nil {
					return err
				}

				if ctx.JSON {
					return ctx.Printer.Print(result.Alert, nil)
				}

				printAlertDetail(ctx.Writer, result.Alert)
				return nil
			})
		},
	}
}

func printAlertDetail(w io.Writer, a flashduty.Alert) {
	labels := make([]string, 0, len(a.Labels))
	for k, v := range a.Labels {
		labels = append(labels, k+"="+v)
	}

	incidentInfo := "-"
	if a.Incident != nil {
		incidentInfo = fmt.Sprintf("%s (%s)", a.Incident.IncidentID, a.Incident.Progress)
	}

	mutedStr := "No"
	if a.EverMuted {
		mutedStr = "Yes"
	}

	_, _ = fmt.Fprintf(w, "ID:            %s\n", a.AlertID)
	_, _ = fmt.Fprintf(w, "Title:         %s\n", a.Title)
	_, _ = fmt.Fprintf(w, "Severity:      %s\n", a.AlertSeverity)
	_, _ = fmt.Fprintf(w, "Status:        %s\n", a.AlertStatus)
	_, _ = fmt.Fprintf(w, "Alert Key:     %s\n", orDash(a.AlertKey))
	_, _ = fmt.Fprintf(w, "Channel:       %s\n", a.ChannelName)
	_, _ = fmt.Fprintf(w, "Integration:   %s (%s)\n", a.IntegrationName, a.IntegrationType)
	_, _ = fmt.Fprintf(w, "Events:        %d\n", a.EventCnt)
	_, _ = fmt.Fprintf(w, "Started:       %s\n", output.FormatTime(a.StartTime))
	_, _ = fmt.Fprintf(w, "Last Event:    %s\n", output.FormatTime(a.LastTime))
	_, _ = fmt.Fprintf(w, "Recovered:     %s\n", output.FormatTime(a.EndTime))
	_, _ = fmt.Fprintf(w, "Muted:         %s\n", mutedStr)
	_, _ = fmt.Fprintf(w, "Incident:      %s\n", incidentInfo)
	_, _ = fmt.Fprintf(w, "Labels:        %s\n", orDash(strings.Join(labels, ", ")))
	_, _ = fmt.Fprintf(w, "Description:   %s\n", orDash(a.Description))
}

func newAlertEventsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "events <alert_id>",
		Short: "List alert events",
		Args:  requireArgs("alert_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.ListAlertEvents(cmdContext(ctx.Cmd), &flashduty.ListAlertEventsInput{
					AlertID: ctx.Args[0],
				})
				if err != nil {
					return err
				}

				if len(result.AlertEvents) == 0 {
					ctx.WriteResult("No alert events found.")
					return nil
				}

				cols := []output.Column{
					{Header: "EVENT_ID", Field: func(v any) string { return v.(flashduty.AlertEvent).EventID }},
					{Header: "SEVERITY", Field: func(v any) string { return v.(flashduty.AlertEvent).EventSeverity }},
					{Header: "STATUS", Field: func(v any) string { return v.(flashduty.AlertEvent).EventStatus }},
					{Header: "TIME", Field: func(v any) string { return output.FormatTime(v.(flashduty.AlertEvent).EventTime) }},
					{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.AlertEvent).Title }},
				}

				return ctx.PrintTotal(result.AlertEvents, cols, len(result.AlertEvents))
			})
		},
	}
}

func newAlertTimelineCmd() *cobra.Command {
	var limit, page int

	cmd := &cobra.Command{
		Use:   "timeline <alert_id>",
		Short: "View alert timeline",
		Args:  requireArgs("alert_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.GetAlertFeed(cmdContext(ctx.Cmd), &flashduty.GetAlertFeedInput{
					AlertID: ctx.Args[0],
					Limit:   limit,
					Page:    page,
				})
				if err != nil {
					return err
				}

				if len(result.Items) == 0 {
					ctx.WriteResult("No timeline events.")
					return nil
				}

				cols := []output.Column{
					{Header: "TIME", Field: func(v any) string { return output.FormatTime(v.(flashduty.TimelineEvent).Timestamp) }},
					{Header: "TYPE", Field: func(v any) string { return v.(flashduty.TimelineEvent).Type }},
					{Header: "OPERATOR", Field: func(v any) string { return v.(flashduty.TimelineEvent).OperatorName }},
					{Header: "DETAIL", MaxWidth: 80, Field: func(v any) string {
						d := v.(flashduty.TimelineEvent).Detail
						if d == nil {
							return "-"
						}
						return fmt.Sprintf("%v", d)
					}},
				}

				return ctx.Printer.Print(result.Items, cols)
			})
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Max events")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}

func newAlertMergeCmd() *cobra.Command {
	var incidentID, comment string

	cmd := &cobra.Command{
		Use:   "merge <alert_id> [<alert_id2> ...]",
		Short: "Merge alerts into an incident",
		Args:  requireArgs("alert_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if err := ctx.Client.MergeAlertsToIncident(cmdContext(ctx.Cmd), &flashduty.MergeAlertsInput{
					AlertIDs:   ctx.Args,
					IncidentID: incidentID,
					Comment:    comment,
				}); err != nil {
					return err
				}

				ctx.WriteResult(fmt.Sprintf("Merged %d alert(s) into incident %s.", len(ctx.Args), incidentID))
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&incidentID, "incident", "", "Target incident ID")
	cmd.Flags().StringVar(&comment, "comment", "", "Merge comment")
	_ = cmd.MarkFlagRequired("incident")

	return cmd
}
