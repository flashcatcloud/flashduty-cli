package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

func newIncidentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "incident",
		Short: "Manage incidents",
	}
	cmd.AddCommand(newIncidentListCmd())
	cmd.AddCommand(newIncidentGetCmd())
	cmd.AddCommand(newIncidentCreateCmd())
	cmd.AddCommand(newIncidentUpdateCmd())
	cmd.AddCommand(newIncidentAckCmd())
	cmd.AddCommand(newIncidentCloseCmd())
	cmd.AddCommand(newIncidentTimelineCmd())
	cmd.AddCommand(newIncidentAlertsCmd())
	cmd.AddCommand(newIncidentSimilarCmd())
	return cmd
}

func incidentColumns() []output.Column {
	return []output.Column{
		{Header: "ID", Field: func(v any) string { return v.(flashduty.EnrichedIncident).IncidentID }},
		{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.EnrichedIncident).Title }},
		{Header: "SEVERITY", Field: func(v any) string { return v.(flashduty.EnrichedIncident).Severity }},
		{Header: "PROGRESS", Field: func(v any) string { return v.(flashduty.EnrichedIncident).Progress }},
		{Header: "CHANNEL", Field: func(v any) string { return v.(flashduty.EnrichedIncident).ChannelName }},
		{Header: "CREATED", Field: func(v any) string { return output.FormatTime(v.(flashduty.EnrichedIncident).StartTime) }},
	}
}

func newIncidentListCmd() *cobra.Command {
	var progress, severity, title, since, until string
	var channelID int64
	var limit, page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List incidents",
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

				result, err := ctx.Client.ListIncidents(cmdContext(ctx.Cmd), &flashduty.ListIncidentsInput{
					Progress:      progress,
					Severity:      severity,
					ChannelID:     channelID,
					StartTime:     startTime,
					EndTime:       endTime,
					Title:         title,
					Limit:         limit,
					Page:          page,
					IncludeAlerts: false,
				})
				if err != nil {
					return err
				}

				return ctx.PrintList(result.Incidents, incidentColumns(), len(result.Incidents), page, result.Total)
			})
		},
	}

	cmd.Flags().StringVar(&progress, "progress", "", "Filter: Triggered,Processing,Closed")
	cmd.Flags().StringVar(&severity, "severity", "", "Filter: Critical,Warning,Info")
	cmd.Flags().Int64Var(&channelID, "channel", 0, "Filter by channel ID")
	cmd.Flags().StringVar(&title, "title", "", "Search by title keyword")
	cmd.Flags().StringVar(&since, "since", "24h", "Start time (duration, date, datetime, or unix timestamp)")
	cmd.Flags().StringVar(&until, "until", "now", "End time")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results (max 100)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}

func newIncidentGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id> [<id2> ...]",
		Short: "Get incident details",
		Args:  requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.ListIncidents(cmdContext(ctx.Cmd), &flashduty.ListIncidentsInput{
					IncidentIDs:   ctx.Args,
					IncludeAlerts: true,
				})
				if err != nil {
					return err
				}

				if ctx.JSON {
					return ctx.Printer.Print(result.Incidents, nil)
				}

				// Single incident: vertical detail view
				if len(ctx.Args) == 1 && len(result.Incidents) == 1 {
					printIncidentDetail(ctx.Writer, result.Incidents[0])
					return nil
				}

				// Multiple: table
				return ctx.Printer.Print(result.Incidents, incidentColumns())
			})
		},
	}
}

func printIncidentDetail(w io.Writer, inc flashduty.EnrichedIncident) {
	responders := make([]string, 0, len(inc.Responders))
	for _, r := range inc.Responders {
		responders = append(responders, r.PersonName)
	}

	labels := make([]string, 0, len(inc.Labels))
	for k, v := range inc.Labels {
		labels = append(labels, k+"="+v)
	}

	fields := make([]string, 0, len(inc.CustomFields))
	for k, v := range inc.CustomFields {
		fields = append(fields, fmt.Sprintf("%s=%v", k, v))
	}

	_, _ = fmt.Fprintf(w, "ID:            %s\n", inc.IncidentID)
	_, _ = fmt.Fprintf(w, "Title:         %s\n", inc.Title)
	_, _ = fmt.Fprintf(w, "Severity:      %s\n", inc.Severity)
	_, _ = fmt.Fprintf(w, "Progress:      %s\n", inc.Progress)
	_, _ = fmt.Fprintf(w, "Channel:       %s\n", inc.ChannelName)
	_, _ = fmt.Fprintf(w, "Created:       %s\n", output.FormatTime(inc.StartTime))
	_, _ = fmt.Fprintf(w, "Creator:       %s (%s)\n", inc.CreatorName, inc.CreatorEmail)
	_, _ = fmt.Fprintf(w, "Responders:    %s\n", orDash(strings.Join(responders, ", ")))
	_, _ = fmt.Fprintf(w, "Description:   %s\n", orDash(inc.Description))
	_, _ = fmt.Fprintf(w, "Labels:        %s\n", orDash(strings.Join(labels, ", ")))
	_, _ = fmt.Fprintf(w, "Custom Fields: %s\n", orDash(strings.Join(fields, ", ")))
	_, _ = fmt.Fprintf(w, "Alerts:        %d total\n", inc.AlertsTotal)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func newIncidentCreateCmd() *cobra.Command {
	var title, severity, description string
	var channelID int64
	var assign []int

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new incident",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Interactive prompts for required fields if stdin is a terminal
			if term.IsTerminal(int(os.Stdin.Fd())) {
				scanner := bufio.NewScanner(os.Stdin)
				if title == "" {
					fmt.Print("Title: ")
					if scanner.Scan() {
						title = scanner.Text()
					}
				}
				if severity == "" {
					fmt.Print("Severity (Critical/Warning/Info): ")
					if scanner.Scan() {
						severity = scanner.Text()
					}
				}
			}

			if title == "" {
				return fmt.Errorf("--title is required")
			}
			if severity == "" {
				return fmt.Errorf("--severity is required (Critical, Warning, Info)")
			}

			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.CreateIncident(cmdContext(ctx.Cmd), &flashduty.CreateIncidentInput{
					Title:       title,
					Severity:    severity,
					ChannelID:   channelID,
					Description: description,
					AssignedTo:  assign,
				})
				if err != nil {
					return err
				}

				if m, ok := result.(map[string]any); ok {
					if id, ok := m["incident_id"]; ok {
						ctx.WriteResult(fmt.Sprintf("Incident created: %v", id))
						return nil
					}
				}
				ctx.WriteResult("Incident created successfully.")
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Incident title (required, 3-200 chars)")
	cmd.Flags().StringVar(&severity, "severity", "", "Severity: Critical, Warning, Info (required)")
	cmd.Flags().Int64Var(&channelID, "channel", 0, "Channel ID")
	cmd.Flags().StringVar(&description, "description", "", "Description (max 6144 chars)")
	cmd.Flags().IntSliceVar(&assign, "assign", nil, "Person IDs to assign (use 'flashduty member list' to look up IDs)")

	return cmd
}

func newIncidentUpdateCmd() *cobra.Command {
	var title, description, severity string
	var fieldFlags []string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an incident",
		Args:  requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			customFields := make(map[string]any)
			for _, f := range fieldFlags {
				parts := strings.SplitN(f, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --field format %q, expected key=value", f)
				}
				customFields[parts[0]] = parts[1]
			}

			return runCommand(cmd, args, func(ctx *RunContext) error {
				input := &flashduty.UpdateIncidentInput{
					IncidentID:   ctx.Args[0],
					Title:        title,
					Description:  description,
					Severity:     severity,
					CustomFields: customFields,
				}

				updated, err := ctx.Client.UpdateIncident(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}

				if len(updated) == 0 {
					ctx.WriteResult("No fields were updated.")
					return nil
				}
				ctx.WriteResult(fmt.Sprintf("Updated incident %s: %s.", ctx.Args[0], strings.Join(updated, ", ")))
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&description, "description", "", "New description")
	cmd.Flags().StringVar(&severity, "severity", "", "New severity: Critical, Warning, Info")
	cmd.Flags().StringArrayVar(&fieldFlags, "field", nil, "Custom field: key=value (repeatable)")

	return cmd
}

func newIncidentAckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ack <id> [<id2> ...]",
		Short: "Acknowledge incidents",
		Args:  requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if err := ctx.Client.AckIncidents(cmdContext(ctx.Cmd), ctx.Args); err != nil {
					return err
				}
				ctx.WriteResult(fmt.Sprintf("Acknowledged %d incident(s).", len(ctx.Args)))
				return nil
			})
		},
	}
}

func newIncidentCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close <id> [<id2> ...]",
		Short: "Close incidents",
		Args:  requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if err := ctx.Client.CloseIncidents(cmdContext(ctx.Cmd), ctx.Args); err != nil {
					return err
				}
				ctx.WriteResult(fmt.Sprintf("Closed %d incident(s).", len(ctx.Args)))
				return nil
			})
		},
	}
}

func newIncidentTimelineCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "timeline <id>",
		Short: "View incident timeline",
		Args:  requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				results, err := ctx.Client.GetIncidentTimelines(cmdContext(ctx.Cmd), []string{ctx.Args[0]})
				if err != nil {
					return err
				}

				if len(results) == 0 || len(results[0].Timeline) == 0 {
					_, _ = fmt.Fprintln(ctx.Writer, "No timeline events.")
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

				return ctx.Printer.Print(results[0].Timeline, cols)
			})
		},
	}
}

func newIncidentAlertsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "alerts <id>",
		Short: "View incident alerts",
		Args:  requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				results, err := ctx.Client.ListIncidentAlerts(cmdContext(ctx.Cmd), []string{ctx.Args[0]}, limit)
				if err != nil {
					return err
				}

				if len(results) == 0 || len(results[0].Alerts) == 0 {
					_, _ = fmt.Fprintln(ctx.Writer, "No alerts.")
					return nil
				}

				cols := []output.Column{
					{Header: "ALERT_ID", Field: func(v any) string { return v.(flashduty.AlertPreview).AlertID }},
					{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.AlertPreview).Title }},
					{Header: "SEVERITY", Field: func(v any) string { return v.(flashduty.AlertPreview).Severity }},
					{Header: "STATUS", Field: func(v any) string { return v.(flashduty.AlertPreview).Status }},
					{Header: "STARTED", Field: func(v any) string { return output.FormatTime(v.(flashduty.AlertPreview).StartTime) }},
				}

				return ctx.PrintTotal(results[0].Alerts, cols, results[0].Total)
			})
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Max alerts to show")
	return cmd
}

func newIncidentSimilarCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "similar <id>",
		Short: "Find similar incidents",
		Args:  requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.ListSimilarIncidents(cmdContext(ctx.Cmd), ctx.Args[0], limit)
				if err != nil {
					return err
				}

				if len(result.Incidents) == 0 {
					_, _ = fmt.Fprintln(ctx.Writer, "No similar incidents found.")
					return nil
				}

				return ctx.Printer.Print(result.Incidents, incidentColumns())
			})
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 5, "Max results")
	return cmd
}

// parseIntSlice converts a comma-separated string to []int64.
func parseIntSlice(s string) ([]int64, error) {
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	result := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ID %q: %w", p, err)
		}
		result = append(result, v)
	}
	return result, nil
}
