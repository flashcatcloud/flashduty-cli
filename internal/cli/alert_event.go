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
	cmd := newGroupCmd("alert-event", "Manage alert events")
	cmd.AddCommand(newAlertEventListCmd())
	return cmd
}

func newAlertEventListCmd() *cobra.Command {
	var severity, channel, integration, integrationType, since, until, fields string
	var limit, page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List alert events globally",
		Long:  curatedLong("List alert events across all alerts within a time window, optionally filtered by severity, channel, or integration type. In json/toon mode, output defaults to compact event fields; use --fields to choose a different projection.", "Alerts", "EventReadList"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if ctx.Structured() && cmd.Flags().Changed("fields") && len(parseStringSlice(fields)) == 0 {
					return fmt.Errorf("--fields must name at least one field")
				}

				startTime, err := timeutil.Parse(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				endTime, err := timeutil.Parse(until)
				if err != nil {
					return fmt.Errorf("invalid --until: %w", err)
				}

				input := &flashduty.AlertEventGlobalListRequest{
					StartTime: flashduty.Int64(startTime),
					EndTime:   flashduty.Int64(endTime),
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

				if ctx.Structured() {
					fieldNames := []string{"event_id", "alert_id", "event_severity", "event_status", "event_time", "title"}
					if fields != "" {
						fieldNames = parseStringSlice(fields)
					} else {
						noteDefaultProjection(cmd.ErrOrStderr(), fieldNames)
					}
					proj, err := projectFields(result.Items, fieldNames)
					if err != nil {
						return err
					}
					bounded, note, err := boundProjectedOutput(proj, compactListOutputLimit)
					if err != nil {
						return err
					}
					proj = bounded.([]map[string]any)
					noteProjectionBound(cmd.ErrOrStderr(), note)
					effectiveLimit := limit
					if len(proj) < len(result.Items) {
						effectiveLimit = len(proj)
					}
					return ctx.PrintList(proj, nil, len(proj), page, effectiveLimit, int(result.Total))
				}

				return ctx.PrintList(result.Items, cols, len(result.Items), page, limit, int(result.Total))
			})
		},
	}

	cmd.Flags().StringVar(&severity, "severity", "", "Filter: Critical,Warning,Info (comma-separated)")
	cmd.Flags().StringVar(&channel, "channel", "", "Comma-separated channel IDs")
	registerEnumFlag(cmd, "severity", severityEnum...)
	cmd.Flags().StringVar(&integration, "integration", "", "Comma-separated integration IDs")
	cmd.Flags().StringVar(&integrationType, "integration-type", "", "Comma-separated integration types (plugin keys, e.g. AliCloud,Prometheus) — not integration IDs; use --integration for that")
	cmd.Flags().StringVar(&since, "since", "1h", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results (max 100)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated fields to project in json/toon output (e.g. event_id,alert_id,event_severity,event_status,event_time,title); ignored in table mode. Defaults to these compact event fields. If the page would exceed 16 KiB, only the leading rows that fit are emitted, with every value intact (announced on stderr).")

	return cmd
}
