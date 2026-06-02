package cli

import (
	"fmt"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

func newAlertEventCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alert-event",
		Short: "Manage alert events",
	}
	cmd.AddCommand(newAlertEventListCmd())
	return cmd
}

func newAlertEventListCmd() *cobra.Command {
	var severity, channel, integrationType, since, until string
	var limit, page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List alert events globally",
		Long:  curatedLong("List alert events across all alerts within a time window, optionally filtered by severity, channel, or integration type.", "Alerts", "EventReadList"),
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

				input := &flashduty.AlertEventGlobalListRequest{
					StartTime: startTime,
					EndTime:   endTime,
				}
				input.Limit = limit
				input.Page = page

				if severity != "" {
					// go-flashduty takes severities as a comma-separated string.
					input.Severities = strings.Join(parseStringSlice(severity), ",")
				}

				if channel != "" {
					channelIDs, err := parseIntSlice(channel)
					if err != nil {
						return fmt.Errorf("invalid --channel: %w", err)
					}
					input.ChannelIDs = channelIDs
				}

				if integrationType != "" {
					input.IntegrationTypes = parseStringSlice(integrationType)
				}

				result, _, err := ctx.Client.Alerts.EventReadList(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "EVENT_ID", Field: func(v any) string { return v.(flashduty.AlertEventItem).EventID }},
					{Header: "ALERT_ID", Field: func(v any) string { return v.(flashduty.AlertEventItem).AlertID }},
					{Header: "SEVERITY", Field: func(v any) string { return v.(flashduty.AlertEventItem).EventSeverity }},
					{Header: "STATUS", Field: func(v any) string { return v.(flashduty.AlertEventItem).EventStatus }},
					{Header: "TIME", Field: func(v any) string { return output.FormatTime(v.(flashduty.AlertEventItem).EventTime) }},
					{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.AlertEventItem).Title }},
				}

				return ctx.PrintList(result.Items, cols, len(result.Items), page, int(result.Total))
			})
		},
	}

	cmd.Flags().StringVar(&severity, "severity", "", "Filter: Critical,Warning,Info (comma-separated)")
	cmd.Flags().StringVar(&channel, "channel", "", "Comma-separated channel IDs")
	registerEnumFlag(cmd, "severity", severityEnum...)
	cmd.Flags().StringVar(&integrationType, "integration-type", "", "Comma-separated integration types")
	cmd.Flags().StringVar(&since, "since", "1h", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}
