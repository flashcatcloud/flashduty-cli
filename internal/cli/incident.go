package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
			client, err := newClient()
			if err != nil {
				return err
			}

			startTime, err := timeutil.Parse(since)
			if err != nil {
				return fmt.Errorf("invalid --since: %w", err)
			}
			endTime, err := timeutil.Parse(until)
			if err != nil {
				return fmt.Errorf("invalid --until: %w", err)
			}

			result, err := client.ListIncidents(cmdContext(cmd), &flashduty.ListIncidentsInput{
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

			p := newPrinter(nil)
			if err := p.Print(result.Incidents, incidentColumns()); err != nil {
				return err
			}
			fmt.Printf("Showing %d results (page %d, total %d).\n", len(result.Incidents), page, result.Total)
			return nil
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
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			result, err := client.ListIncidents(cmdContext(cmd), &flashduty.ListIncidentsInput{
				IncidentIDs:   args,
				IncludeAlerts: true,
			})
			if err != nil {
				return err
			}

			if flagJSON {
				return newPrinter(nil).Print(result.Incidents, nil)
			}

			// Single incident: vertical detail view
			if len(args) == 1 && len(result.Incidents) == 1 {
				printIncidentDetail(result.Incidents[0])
				return nil
			}

			// Multiple: table
			return newPrinter(nil).Print(result.Incidents, incidentColumns())
		},
	}
}

func printIncidentDetail(inc flashduty.EnrichedIncident) {
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

	fmt.Printf("ID:            %s\n", inc.IncidentID)
	fmt.Printf("Title:         %s\n", inc.Title)
	fmt.Printf("Severity:      %s\n", inc.Severity)
	fmt.Printf("Progress:      %s\n", inc.Progress)
	fmt.Printf("Channel:       %s\n", inc.ChannelName)
	fmt.Printf("Created:       %s\n", output.FormatTime(inc.StartTime))
	fmt.Printf("Creator:       %s (%s)\n", inc.CreatorName, inc.CreatorEmail)
	fmt.Printf("Responders:    %s\n", orDash(strings.Join(responders, ", ")))
	fmt.Printf("Description:   %s\n", orDash(inc.Description))
	fmt.Printf("Labels:        %s\n", orDash(strings.Join(labels, ", ")))
	fmt.Printf("Custom Fields: %s\n", orDash(strings.Join(fields, ", ")))
	fmt.Printf("Alerts:        %d total\n", inc.AlertsTotal)
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

			client, err := newClient()
			if err != nil {
				return err
			}

			result, err := client.CreateIncident(cmdContext(cmd), &flashduty.CreateIncidentInput{
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
					fmt.Printf("Incident created: %v\n", id)
					return nil
				}
			}
			fmt.Println("Incident created successfully.")
			return nil
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			customFields := make(map[string]any)
			for _, f := range fieldFlags {
				parts := strings.SplitN(f, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --field format %q, expected key=value", f)
				}
				customFields[parts[0]] = parts[1]
			}

			input := &flashduty.UpdateIncidentInput{
				IncidentID:   args[0],
				Title:        title,
				Description:  description,
				Severity:     severity,
				CustomFields: customFields,
			}

			updated, err := client.UpdateIncident(cmdContext(cmd), input)
			if err != nil {
				return err
			}

			if len(updated) == 0 {
				fmt.Println("No fields were updated.")
				return nil
			}
			fmt.Printf("Updated incident %s: %s.\n", args[0], strings.Join(updated, ", "))
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&description, "description", "", "New description")
	cmd.Flags().StringVar(&severity, "severity", "", "New severity")
	cmd.Flags().StringArrayVar(&fieldFlags, "field", nil, "Custom field: key=value (repeatable)")

	return cmd
}

func newIncidentAckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ack <id> [<id2> ...]",
		Short: "Acknowledge incidents",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			if err := client.AckIncidents(cmdContext(cmd), args); err != nil {
				return err
			}
			fmt.Printf("Acknowledged %d incident(s).\n", len(args))
			return nil
		},
	}
}

func newIncidentCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close <id> [<id2> ...]",
		Short: "Close incidents",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			if err := client.CloseIncidents(cmdContext(cmd), args); err != nil {
				return err
			}
			fmt.Printf("Closed %d incident(s).\n", len(args))
			return nil
		},
	}
}

func newIncidentTimelineCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "timeline <id>",
		Short: "View incident timeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			results, err := client.GetIncidentTimelines(cmdContext(cmd), []string{args[0]})
			if err != nil {
				return err
			}

			if len(results) == 0 || len(results[0].Timeline) == 0 {
				fmt.Println("No timeline events.")
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

			return newPrinter(nil).Print(results[0].Timeline, cols)
		},
	}
}

func newIncidentAlertsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "alerts <id>",
		Short: "View incident alerts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			results, err := client.ListIncidentAlerts(cmdContext(cmd), []string{args[0]}, limit)
			if err != nil {
				return err
			}

			if len(results) == 0 || len(results[0].Alerts) == 0 {
				fmt.Println("No alerts.")
				return nil
			}

			cols := []output.Column{
				{Header: "ALERT_ID", Field: func(v any) string { return v.(flashduty.AlertPreview).AlertID }},
				{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.AlertPreview).Title }},
				{Header: "SEVERITY", Field: func(v any) string { return v.(flashduty.AlertPreview).Severity }},
				{Header: "STATUS", Field: func(v any) string { return v.(flashduty.AlertPreview).Status }},
				{Header: "STARTED", Field: func(v any) string { return output.FormatTime(v.(flashduty.AlertPreview).StartTime) }},
			}

			p := newPrinter(nil)
			if err := p.Print(results[0].Alerts, cols); err != nil {
				return err
			}
			fmt.Printf("Total: %d\n", results[0].Total)
			return nil
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			result, err := client.ListSimilarIncidents(cmdContext(cmd), args[0], limit)
			if err != nil {
				return err
			}

			if len(result.Incidents) == 0 {
				fmt.Println("No similar incidents found.")
				return nil
			}

			return newPrinter(nil).Print(result.Incidents, incidentColumns())
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
