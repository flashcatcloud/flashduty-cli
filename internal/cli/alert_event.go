package cli

import (
	"fmt"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

func newAlertEventCmd() *cobra.Command {
	cmd := newGroupCmd("alert-event", "Manage alert events")
	cmd.AddCommand(newAlertEventListCmd())
	return cmd
}

func newAlertEventListCmd() *cobra.Command {
	var severity, integration, integrationType, since, until, fields string
	var limit, page int
	var asc bool
	var channelIDs []int64

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List alert events globally",
		Long:  curatedLong("List alert events across all alerts within a time window, optionally filtered by severity, channel, or integration type. --since/--until window must be < 31 days. In json/toon mode, rows carry every field; pass --fields to project a subset.", "Alerts", "EventReadList"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if err := checkFieldsFlag(ctx, fields); err != nil {
					return err
				}

				startTime, endTime, err := parseWindow(cmd, since, until, "")
				if err != nil {
					return err
				}

				input := &flashduty.AlertEventGlobalListRequest{
					StartTime: startTime,
					EndTime:   endTime,
				}
				input.Limit = limit
				input.Page = page
				input.Asc = asc
				input.ChannelIDs = channelIDs

				if severity != "" {
					// go-flashduty takes severities as a comma-separated string.
					input.Severities = strings.Join(parseStringSlice(severity), ",")
				}

				if integration != "" {
					integrationIDs, err := parseIntSlice(integration)
					if err != nil {
						return fmt.Errorf("invalid --integration: %w", err)
					}
					input.IntegrationIDs = integrationIDs
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

				if fields != "" && ctx.Structured() {
					proj, err := projectFields(result.Items, parseStringSlice(fields))
					if err != nil {
						return err
					}
					return ctx.PrintCappedList(proj, nil, len(proj), page, limit, int(result.Total), result.HasNextPage)
				}

				return ctx.PrintCappedList(result.Items, cols, len(result.Items), page, limit, int(result.Total), result.HasNextPage)
			})
		},
	}

	cmd.Flags().StringVar(&severity, "severity", "", "Filter: Critical,Warning,Info (comma-separated)")
	channelIDsFlag(cmd, &channelIDs, "channel")
	registerEnumFlag(cmd, "severity", severityEnum...)
	cmd.Flags().StringVar(&integration, "integration", "", "Comma-separated integration IDs")
	cmd.Flags().StringVar(&integrationType, "integration-type", "", "Comma-separated integration types (plugin keys, e.g. AliCloud,Prometheus) — not integration IDs; use --integration for that")
	cmd.Flags().BoolVar(&asc, "asc", false, "Sort oldest first (default newest first)")
	cmd.Flags().StringVar(&since, "since", "1h", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results (max 100)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated fields to project in json/toon output (e.g. event_id,alert_id,event_severity,event_status,event_time,title); ignored in table mode. Without it, json/toon output carries every field.")

	return cmd
}
