package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

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
	cmd.AddCommand(newIncidentUnackCmd())
	cmd.AddCommand(newIncidentCloseCmd())
	cmd.AddCommand(newIncidentWakeCmd())
	cmd.AddCommand(newIncidentTimelineCmd())
	cmd.AddCommand(newIncidentAlertsCmd())
	cmd.AddCommand(newIncidentSimilarCmd())
	cmd.AddCommand(newIncidentMergeCmd())
	cmd.AddCommand(newIncidentSnoozeCmd())
	cmd.AddCommand(newIncidentReopenCmd())
	cmd.AddCommand(newIncidentReassignCmd())
	cmd.AddCommand(newIncidentAddResponderCmd())
	cmd.AddCommand(newIncidentCommentCmd())
	cmd.AddCommand(newIncidentDisableMergeCmd())
	cmd.AddCommand(newIncidentRemoveCmd())
	cmd.AddCommand(newIncidentWarRoomCmd())
	cmd.AddCommand(newIncidentFeedCmd())
	cmd.AddCommand(newIncidentDetailCmd())
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
	var progress, severity, query, since, until string
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
					Query:         query,
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
	cmd.Flags().StringVar(&query, "query", "", "Free-text search across title/labels/content (also resolves a 24-char incident ID or 6-char incident num to a direct lookup)")
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

func newIncidentUnackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unack <id> [<id2> ...]",
		Short: "Cancel incident acknowledgement",
		Long: `Cancel acknowledgement for one or more incidents.

Use this when an incident was acknowledged by mistake and should return to the
unacknowledged state. The command accepts up to 100 incident IDs.`,
		Example: `  flashduty incident unack inc_123
  flashduty incident unack inc_123 inc_456`,
		Args: requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateIncidentIDBatch(args); err != nil {
				return err
			}
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if err := ctx.Client.UnackIncidents(cmdContext(ctx.Cmd), ctx.Args); err != nil {
					return err
				}
				ctx.WriteResult(fmt.Sprintf("Unacknowledged %d incident(s).", len(ctx.Args)))
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

func newIncidentWakeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "wake <id> [<id2> ...]",
		Short: "Restore notifications for snoozed incidents",
		Long: `Wake one or more snoozed incidents.

This cancels snooze and restores normal incident notifications. The command
accepts up to 100 incident IDs.`,
		Example: `  flashduty incident wake inc_123
  flashduty incident wake inc_123 inc_456`,
		Args: requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateIncidentIDBatch(args); err != nil {
				return err
			}
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if err := ctx.Client.WakeIncidents(cmdContext(ctx.Cmd), ctx.Args); err != nil {
					return err
				}
				ctx.WriteResult(fmt.Sprintf("Restored notifications for %d incident(s).", len(ctx.Args)))
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

// boolPtr returns a pointer to the given bool value.
func boolPtr(b bool) *bool { return &b }

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

// parseStringSlice splits a comma-separated string into trimmed, non-empty strings.
func parseStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func validateIncidentIDBatch(incidentIDs []string) error {
	if len(incidentIDs) > 100 {
		return fmt.Errorf("command accepts at most 100 incident IDs")
	}
	return nil
}

func newIncidentMergeCmd() *cobra.Command {
	var source string

	cmd := &cobra.Command{
		Use:   "merge <target_id>",
		Short: "Merge incidents into a target incident",
		Args:  requireArgs("target_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				sourceIDs := parseStringSlice(source)
				if len(sourceIDs) == 0 {
					return fmt.Errorf("--source is required")
				}
				if len(sourceIDs) > 100 {
					return fmt.Errorf("--source accepts at most 100 incident IDs")
				}

				if err := ctx.Client.MergeIncidents(cmdContext(ctx.Cmd), &flashduty.MergeIncidentsInput{
					SourceIncidentIDs: sourceIDs,
					TargetIncidentID:  ctx.Args[0],
				}); err != nil {
					return err
				}

				ctx.WriteResult(fmt.Sprintf("Merged %d incident(s) into %s.", len(sourceIDs), ctx.Args[0]))
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "Comma-separated source incident IDs (max 100)")
	_ = cmd.MarkFlagRequired("source")

	return cmd
}

func newIncidentSnoozeCmd() *cobra.Command {
	var duration string

	cmd := &cobra.Command{
		Use:   "snooze <id> [<id2> ...]",
		Short: "Snooze incidents",
		Args:  requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				d, err := time.ParseDuration(duration)
				if err != nil {
					return fmt.Errorf("invalid --duration: %w", err)
				}
				if d <= 0 || d > 24*time.Hour {
					return fmt.Errorf("--duration must be between 1m and 24h")
				}
				if d%time.Minute != 0 {
					return fmt.Errorf("--duration must be in whole minutes")
				}

				minutes := int64(d / time.Minute)

				if err := ctx.Client.SnoozeIncidents(cmdContext(ctx.Cmd), &flashduty.SnoozeIncidentsInput{
					IncidentIDs: ctx.Args,
					Minutes:     minutes,
				}); err != nil {
					return err
				}

				ctx.WriteResult(fmt.Sprintf("Snoozed %d incident(s) for %s.", len(ctx.Args), duration))
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&duration, "duration", "", "Snooze duration (e.g., \"2h\", \"30m\", max \"24h\")")
	_ = cmd.MarkFlagRequired("duration")

	return cmd
}

func newIncidentReopenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reopen <id> [<id2> ...]",
		Short: "Reopen closed incidents",
		Args:  requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if err := ctx.Client.ReopenIncidents(cmdContext(ctx.Cmd), ctx.Args); err != nil {
					return err
				}
				ctx.WriteResult(fmt.Sprintf("Reopened %d incident(s).", len(ctx.Args)))
				return nil
			})
		},
	}
}

func newIncidentReassignCmd() *cobra.Command {
	var person string

	cmd := &cobra.Command{
		Use:   "reassign <id>",
		Short: "Reassign an incident to new responders",
		Args:  requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				personIDs, err := parseIntSlice(person)
				if err != nil {
					return fmt.Errorf("invalid --person: %w", err)
				}
				if len(personIDs) == 0 {
					return fmt.Errorf("--person is required")
				}

				if err := ctx.Client.ReassignIncidents(cmdContext(ctx.Cmd), &flashduty.ReassignIncidentsInput{
					IncidentIDs: []string{ctx.Args[0]},
					PersonIDs:   personIDs,
				}); err != nil {
					return err
				}

				ctx.WriteResult(fmt.Sprintf("Reassigned incident %s to %d responder(s).", ctx.Args[0], len(personIDs)))
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&person, "person", "", "Comma-separated person IDs")
	_ = cmd.MarkFlagRequired("person")

	return cmd
}

func newIncidentAddResponderCmd() *cobra.Command {
	var person, notifyChannel, templateID string
	var followPreference bool

	cmd := &cobra.Command{
		Use:   "add-responder <id>",
		Short: "Add responders to an incident",
		Long: `Add one or more responders to an incident.

Responder IDs are person IDs. Use 'flashduty member list' to find the right
person ID before running this command. Optional notification flags let you ask
FlashDuty to notify added responders through their preferences, explicit
personal channels, or a template.`,
		Example: `  flashduty member list --name "Ada"
  flashduty incident add-responder inc_123 --person 101,202
  flashduty incident add-responder inc_123 --person 101 --follow-preference
  flashduty incident add-responder inc_123 --person 101 --notify-channel voice,sms,email`,
		Args: requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			personIDs, err := parseIntSlice(person)
			if err != nil {
				return fmt.Errorf("invalid --person: %w", err)
			}
			if len(personIDs) == 0 {
				return fmt.Errorf("--person is required")
			}

			var notify *flashduty.IncidentNotifyInput
			if followPreference || notifyChannel != "" || templateID != "" {
				notify = &flashduty.IncidentNotifyInput{
					FollowPreference: followPreference,
					PersonalChannels: parseStringSlice(notifyChannel),
					TemplateID:       templateID,
				}
			}

			return runCommand(cmd, args, func(ctx *RunContext) error {
				if err := ctx.Client.AddIncidentResponders(cmdContext(ctx.Cmd), &flashduty.IncidentAddResponderInput{
					IncidentID: ctx.Args[0],
					PersonIDs:  personIDs,
					Notify:     notify,
				}); err != nil {
					return err
				}

				ctx.WriteResult(fmt.Sprintf("Added %d responder(s) to incident %s.", len(personIDs), ctx.Args[0]))
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&person, "person", "", "Comma-separated person IDs to add")
	cmd.Flags().BoolVar(&followPreference, "follow-preference", false, "Follow each responder's notification preferences")
	cmd.Flags().StringVar(&notifyChannel, "notify-channel", "", "Comma-separated notification channels, e.g. voice,sms,email")
	cmd.Flags().StringVar(&templateID, "template-id", "", "Notification template ID")
	_ = cmd.MarkFlagRequired("person")

	return cmd
}

func newIncidentCommentCmd() *cobra.Command {
	var comment string
	var muteReply bool

	cmd := &cobra.Command{
		Use:   "comment <id> [<id2> ...]",
		Short: "Add a comment to incident timelines",
		Long: `Add a comment to one or more incident timelines.

The command accepts up to 100 incidents. Comment text is required and must be
at most 1024 characters. Use --mute-reply when the comment should not trigger
webhook reply behavior.`,
		Example: `  flashduty incident comment inc_123 --comment "Rollback started"
  flashduty incident comment inc_123 inc_456 --comment "Mitigation deployed" --mute-reply`,
		Args: requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateIncidentIDBatch(args); err != nil {
				return err
			}
			if strings.TrimSpace(comment) == "" {
				return fmt.Errorf("--comment is required")
			}
			if len([]rune(comment)) > 1024 {
				return fmt.Errorf("--comment must be at most 1024 characters")
			}

			return runCommand(cmd, args, func(ctx *RunContext) error {
				if err := ctx.Client.CommentIncidents(cmdContext(ctx.Cmd), &flashduty.IncidentCommentInput{
					IncidentIDs: ctx.Args,
					Comment:     comment,
					MuteReply:   muteReply,
				}); err != nil {
					return err
				}

				ctx.WriteResult(fmt.Sprintf("Commented on %d incident(s).", len(ctx.Args)))
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&comment, "comment", "", "Comment text")
	cmd.Flags().BoolVar(&muteReply, "mute-reply", false, "Do not trigger webhook reply behavior for this comment")
	_ = cmd.MarkFlagRequired("comment")

	return cmd
}

func newIncidentDisableMergeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable-merge <id> [<id2> ...]",
		Short: "Disable automatic merging for incidents",
		Long: `Disable automatic alert merging for one or more incidents.

Use this when an incident should stay isolated and must not absorb additional
matching alerts automatically. The command accepts up to 100 incident IDs.`,
		Example: `  flashduty incident disable-merge inc_123
  flashduty incident disable-merge inc_123 inc_456`,
		Args: requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if err := ctx.Client.DisableIncidentMerge(cmdContext(ctx.Cmd), ctx.Args); err != nil {
					return err
				}
				ctx.WriteResult(fmt.Sprintf("Disabled auto-merge for %d incident(s).", len(ctx.Args)))
				return nil
			})
		},
	}
}

func newIncidentRemoveCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove <id> [<id2> ...]",
		Short: "Permanently remove incidents",
		Long: `Permanently removes incidents from FlashDuty.

This is a destructive operation. Prompts for confirmation in an interactive
terminal unless --force is set. In non-interactive mode the command aborts
unless --force is provided. The command accepts up to 100 incident IDs.`,
		Example: `  flashduty incident remove inc_123
  flashduty incident remove inc_123 inc_456 --force`,
		Args: requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateIncidentIDBatch(args); err != nil {
				return err
			}
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if !confirmAction(ctx.Cmd, fmt.Sprintf("Are you sure you want to remove %d incident(s)?", len(ctx.Args))) {
					_, _ = fmt.Fprintln(ctx.Writer, "Aborted.")
					return nil
				}

				if err := ctx.Client.RemoveIncidents(cmdContext(ctx.Cmd), ctx.Args); err != nil {
					return err
				}
				ctx.WriteResult(fmt.Sprintf("Removed %d incident(s).", len(ctx.Args)))
				return nil
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newIncidentWarRoomCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "war-room",
		Short: "Manage incident war rooms",
		Long: `Manage incident war rooms.

War rooms are IM chats attached to incidents. Creating a war room can invite
explicit members and, when requested, historical responders as observers.
Commands that operate on an existing IM chat require the IM integration ID.`,
		Example: `  flashduty incident war-room create inc_123 --add-observers
  flashduty incident war-room list inc_123
  flashduty incident war-room get chat_123 --integration 42`,
	}
	cmd.AddCommand(newIncidentWarRoomCreateCmd())
	cmd.AddCommand(newIncidentWarRoomListCmd())
	cmd.AddCommand(newIncidentWarRoomGetCmd())
	cmd.AddCommand(newIncidentWarRoomDeleteCmd())
	cmd.AddCommand(newIncidentWarRoomAddMemberCmd())
	cmd.AddCommand(newIncidentWarRoomDefaultObserversCmd())
	return cmd
}

func newIncidentWarRoomCreateCmd() *cobra.Command {
	var integrationID int64
	var member string
	var addObservers bool

	cmd := &cobra.Command{
		Use:   "create <incident_id>",
		Short: "Create an incident war room",
		Long: `Create an incident war room in a configured IM integration.

If --integration is omitted, the CLI uses the first war-room-enabled IM
integration returned by FlashDuty. Use --member to invite person IDs directly.
Use 'flashduty member list' to find person IDs. Use --add-observers to also
invite historical responders selected by FlashDuty.`,
		Example: `  flashduty incident war-room create inc_123
  flashduty incident war-room create inc_123 --integration 42 --member 101,202
  flashduty incident war-room create inc_123 --add-observers`,
		Args: requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			memberIDs, err := parseIntSlice(member)
			if err != nil {
				return fmt.Errorf("invalid --member: %w", err)
			}
			return runCommand(cmd, args, func(ctx *RunContext) error {
				resolvedIntegrationID, err := resolveWarRoomIntegrationID(ctx)
				if err != nil {
					return err
				}
				warRoom, err := ctx.Client.CreateIncidentWarRoom(cmdContext(ctx.Cmd), &flashduty.IncidentWarRoomCreateInput{
					IncidentID:    ctx.Args[0],
					IntegrationID: resolvedIntegrationID,
					MemberIDs:     memberIDs,
					AddObservers:  addObservers,
				})
				if err != nil {
					return err
				}

				message := fmt.Sprintf("War room created: %s", warRoom.ChatID)
				if warRoom.ShareLink != "" {
					message += fmt.Sprintf("\nShare link: %s", warRoom.ShareLink)
				}
				return ctx.WriteResultJSON(warRoom, message)
			})
		},
	}

	cmd.Flags().Int64Var(&integrationID, "integration", 0, "IM integration ID; if omitted, first war-room-enabled IM integration is used")
	cmd.Flags().StringVar(&member, "member", "", "Comma-separated member person IDs to invite")
	cmd.Flags().BoolVar(&addObservers, "add-observers", false, "Invite historical responders as extra war-room members")
	return cmd
}

func resolveWarRoomIntegrationID(ctx *RunContext) (int64, error) {
	integrationID, err := ctx.Cmd.Flags().GetInt64("integration")
	if err != nil {
		return 0, err
	}
	if integrationID > 0 {
		return integrationID, nil
	}

	result, err := ctx.Client.ListWarRoomEnabledDataSources(cmdContext(ctx.Cmd))
	if err != nil {
		return 0, err
	}
	if result == nil || len(result.Items) == 0 {
		return 0, fmt.Errorf("no IM integration has war-room enabled; enable one in integration settings or pass --integration")
	}
	return result.Items[0].DataSourceID, nil
}

func newIncidentWarRoomListCmd() *cobra.Command {
	var integrationID int64

	cmd := &cobra.Command{
		Use:   "list <incident_id>",
		Short: "List incident war rooms",
		Long: `List war rooms attached to an incident.

Use this to discover chat IDs and integration IDs for follow-up commands such
as get, delete, and add-member.`,
		Example: `  flashduty incident war-room list inc_123
  flashduty incident war-room list inc_123 --integration 42`,
		Args: requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.ListIncidentWarRooms(cmdContext(ctx.Cmd), &flashduty.IncidentWarRoomListInput{
					IncidentID:    ctx.Args[0],
					IntegrationID: integrationID,
				})
				if err != nil {
					return err
				}
				return ctx.PrintTotal(result.Items, incidentWarRoomColumns(), len(result.Items))
			})
		},
	}

	cmd.Flags().Int64Var(&integrationID, "integration", 0, "Filter by IM integration ID")
	return cmd
}

func newIncidentWarRoomGetCmd() *cobra.Command {
	var integrationID int64

	cmd := &cobra.Command{
		Use:   "get <chat_id>",
		Short: "Get incident war room details",
		Long: `Get incident war room details by IM chat ID.

This command requires --integration because chat IDs are scoped to an IM
integration. Use 'flashduty incident war-room list' with an incident ID to find
the chat ID and integration ID for an incident.`,
		Example: `  flashduty incident war-room list inc_123
  flashduty incident war-room get chat_123 --integration 42`,
		Args: requireArgs("chat_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				warRoom, err := ctx.Client.GetIncidentWarRoom(cmdContext(ctx.Cmd), &flashduty.IncidentWarRoomDetailInput{
					IntegrationID: integrationID,
					ChatID:        ctx.Args[0],
				})
				if err != nil {
					return err
				}
				if ctx.JSON {
					return ctx.Printer.Print(warRoom, nil)
				}
				printWarRoomDetail(ctx.Writer, warRoom)
				return nil
			})
		},
	}

	cmd.Flags().Int64Var(&integrationID, "integration", 0, "IM integration ID (required)")
	_ = cmd.MarkFlagRequired("integration")
	return cmd
}

func newIncidentWarRoomDeleteCmd() *cobra.Command {
	var integrationID int64
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <incident_id>",
		Short: "Delete an incident war room",
		Long: `Delete the war room attached to an incident for an IM integration.

This is a destructive operation. Prompts for confirmation in an interactive
terminal unless --force is set. In non-interactive mode the command aborts
unless --force is provided. Use 'flashduty incident war-room list' to find the
integration ID.`,
		Example: `  flashduty incident war-room list inc_123
  flashduty incident war-room delete inc_123 --integration 42
  flashduty incident war-room delete inc_123 --integration 42 --force`,
		Args: requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if !confirmAction(ctx.Cmd, fmt.Sprintf("Are you sure you want to delete the war room for incident %s?", ctx.Args[0])) {
					_, _ = fmt.Fprintln(ctx.Writer, "Aborted.")
					return nil
				}
				if err := ctx.Client.DeleteIncidentWarRoom(cmdContext(ctx.Cmd), &flashduty.IncidentWarRoomDeleteInput{
					IncidentID:    ctx.Args[0],
					IntegrationID: integrationID,
				}); err != nil {
					return err
				}
				ctx.WriteResult(fmt.Sprintf("Deleted war room for incident %s.", ctx.Args[0]))
				return nil
			})
		},
	}

	cmd.Flags().Int64Var(&integrationID, "integration", 0, "IM integration ID (required)")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	_ = cmd.MarkFlagRequired("integration")
	return cmd
}

func newIncidentWarRoomAddMemberCmd() *cobra.Command {
	var integrationID int64
	var member string

	cmd := &cobra.Command{
		Use:   "add-member <chat_id>",
		Short: "Add members to an incident war room",
		Long: `Add members to an existing incident war room by IM chat ID.

This command requires --integration because chat IDs are scoped to an IM
integration. Member IDs are person IDs. Use 'flashduty member list' to find
person IDs, and 'flashduty incident war-room list' to find chat and integration
IDs.`,
		Example: `  flashduty member list --name "Ada"
  flashduty incident war-room list inc_123
  flashduty incident war-room add-member chat_123 --integration 42 --member 101,202`,
		Args: requireArgs("chat_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			memberIDs, err := parseIntSlice(member)
			if err != nil {
				return fmt.Errorf("invalid --member: %w", err)
			}
			if len(memberIDs) == 0 {
				return fmt.Errorf("--member is required")
			}
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if err := ctx.Client.AddIncidentWarRoomMembers(cmdContext(ctx.Cmd), &flashduty.IncidentWarRoomAddMemberInput{
					IntegrationID: integrationID,
					ChatID:        ctx.Args[0],
					MemberIDs:     memberIDs,
				}); err != nil {
					return err
				}
				ctx.WriteResult(fmt.Sprintf("Added %d member(s) to war room %s.", len(memberIDs), ctx.Args[0]))
				return nil
			})
		},
	}

	cmd.Flags().Int64Var(&integrationID, "integration", 0, "IM integration ID (required)")
	cmd.Flags().StringVar(&member, "member", "", "Comma-separated member person IDs (required)")
	_ = cmd.MarkFlagRequired("integration")
	_ = cmd.MarkFlagRequired("member")
	return cmd
}

func newIncidentWarRoomDefaultObserversCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "default-observers <incident_id>",
		Short: "Preview historical responders for war-room observer invitation",
		Long: `Preview historical responders eligible for war-room observer invitation.

This is a read-only preview of the users FlashDuty would add when
--add-observers is used during war-room creation.`,
		Example: `  flashduty incident war-room default-observers inc_123
  flashduty incident war-room create inc_123 --add-observers`,
		Args: requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				observers, err := ctx.Client.GetIncidentWarRoomDefaultObservers(cmdContext(ctx.Cmd), ctx.Args[0])
				if err != nil {
					return err
				}
				return ctx.PrintTotal(observers, incidentWarRoomObserverColumns(), len(observers))
			})
		},
	}
}

func incidentWarRoomColumns() []output.Column {
	return []output.Column{
		{Header: "INTEGRATION", Field: func(v any) string { return fmt.Sprint(v.(flashduty.IncidentWarRoomItem).IntegrationID) }},
		{Header: "CHAT_ID", Field: func(v any) string { return v.(flashduty.IncidentWarRoomItem).ChatID }},
		{Header: "INCIDENT_ID", Field: func(v any) string { return v.(flashduty.IncidentWarRoomItem).IncidentID }},
		{Header: "STATUS", Field: func(v any) string { return v.(flashduty.IncidentWarRoomItem).Status }},
		{Header: "PLUGIN", Field: func(v any) string { return v.(flashduty.IncidentWarRoomItem).PluginType }},
		{Header: "CREATED", Field: func(v any) string { return formatWarRoomCreatedAt(v.(flashduty.IncidentWarRoomItem).CreatedAt) }},
	}
}

func incidentWarRoomObserverColumns() []output.Column {
	return []output.Column{
		{Header: "PERSON_ID", Field: func(v any) string { return fmt.Sprint(v.(flashduty.IncidentWarRoomObserver).PersonID) }},
		{Header: "NAME", Field: func(v any) string { return v.(flashduty.IncidentWarRoomObserver).DisplayName() }},
		{Header: "EMAIL", Field: func(v any) string { return v.(flashduty.IncidentWarRoomObserver).Email }},
		{Header: "STATUS", Field: func(v any) string { return v.(flashduty.IncidentWarRoomObserver).Status }},
	}
}

func printWarRoomDetail(w io.Writer, warRoom *flashduty.IncidentWarRoom) {
	if warRoom == nil {
		return
	}
	_, _ = fmt.Fprintf(w, "Chat ID:    %s\n", warRoom.ChatID)
	_, _ = fmt.Fprintf(w, "Chat Name:  %s\n", orDash(warRoom.ChatName))
	_, _ = fmt.Fprintf(w, "Share Link: %s\n", orDash(warRoom.ShareLink))
}

func formatWarRoomCreatedAt(ts int64) string {
	if ts > 1_000_000_000_000 {
		ts /= 1000
	}
	return output.FormatTime(ts)
}

func newIncidentFeedCmd() *cobra.Command {
	var limit, page int

	cmd := &cobra.Command{
		Use:   "feed <id>",
		Short: "View incident feed (paginated timeline)",
		Args:  requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.GetIncidentFeed(cmdContext(ctx.Cmd), &flashduty.GetIncidentFeedInput{
					IncidentID: ctx.Args[0],
					Limit:      limit,
					Page:       page,
				})
				if err != nil {
					return err
				}

				if len(result.Items) == 0 {
					ctx.WriteResult("No feed events.")
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

func newIncidentDetailCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detail <id>",
		Short: "View full incident detail with AI summary",
		Args:  requireArgs("incident_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.GetIncidentDetail(cmdContext(ctx.Cmd), &flashduty.GetIncidentDetailInput{
					IncidentID: ctx.Args[0],
				})
				if err != nil {
					return err
				}

				if ctx.JSON {
					return ctx.Printer.Print(result.Incident, nil)
				}

				printIncidentFullDetail(ctx.Writer, result.Incident)
				return nil
			})
		},
	}
}

func printIncidentFullDetail(w io.Writer, inc flashduty.IncidentDetail) {
	responders := make([]string, 0, len(inc.Responders))
	for _, r := range inc.Responders {
		name := r.PersonName
		if name == "" {
			name = strconv.FormatInt(r.PersonID, 10)
		}
		responders = append(responders, name)
	}

	labels := make([]string, 0, len(inc.Labels))
	for k, v := range inc.Labels {
		labels = append(labels, k+"="+v)
	}

	fields := make([]string, 0, len(inc.Fields))
	for k, v := range inc.Fields {
		fields = append(fields, fmt.Sprintf("%s=%v", k, v))
	}

	_, _ = fmt.Fprintf(w, "ID:            %s\n", inc.IncidentID)
	_, _ = fmt.Fprintf(w, "Title:         %s\n", inc.Title)
	_, _ = fmt.Fprintf(w, "Severity:      %s\n", inc.Severity)
	_, _ = fmt.Fprintf(w, "Progress:      %s\n", inc.Progress)
	_, _ = fmt.Fprintf(w, "Channel:       %s\n", inc.ChannelName)
	_, _ = fmt.Fprintf(w, "Created:       %s\n", output.FormatTime(inc.StartTime))
	_, _ = fmt.Fprintf(w, "Acknowledged:  %s\n", output.FormatTime(inc.AckTime))
	_, _ = fmt.Fprintf(w, "Closed:        %s\n", output.FormatTime(inc.CloseTime))
	_, _ = fmt.Fprintf(w, "Alerts:        %d alerts, %d events\n", inc.AlertCnt, inc.AlertEventCnt)
	_, _ = fmt.Fprintf(w, "Frequency:     %s\n", orDash(inc.Frequency))
	_, _ = fmt.Fprintf(w, "AI Summary:    %s\n", orDash(inc.AISummary))
	_, _ = fmt.Fprintf(w, "Root Cause:    %s\n", orDash(inc.RootCause))
	_, _ = fmt.Fprintf(w, "Resolution:    %s\n", orDash(inc.Resolution))
	_, _ = fmt.Fprintf(w, "Impact:        %s\n", orDash(inc.Impact))
	_, _ = fmt.Fprintf(w, "Description:   %s\n", orDash(inc.Description))
	_, _ = fmt.Fprintf(w, "Labels:        %s\n", orDash(strings.Join(labels, ", ")))
	_, _ = fmt.Fprintf(w, "Custom Fields: %s\n", orDash(strings.Join(fields, ", ")))
	_, _ = fmt.Fprintf(w, "Responders:    %s\n", orDash(strings.Join(responders, ", ")))
}
