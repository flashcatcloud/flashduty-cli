package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

func newAlertCmd() *cobra.Command {
	cmd := newGroupCmd("alert", "Manage alerts")
	cmd.AddCommand(newAlertListCmd())
	cmd.AddCommand(newAlertGetCmd())
	cmd.AddCommand(newAlertEventsCmd())
	cmd.AddCommand(newAlertTimelineCmd())
	cmd.AddCommand(newAlertMergeCmd())
	return cmd
}

func newAlertListCmd() *cobra.Command {
	var severity, channel, integration, since, until, fields string
	var active, recovered, muted bool
	var limit, page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List alerts",
		Long:  curatedLong("List alerts within a time window, optionally filtered by severity, channel, active/recovered/muted state. No server-side title/text filter — to search by title, pipe --json to jq: 'select(.title|test(\"pat\";\"i\"))'. In json/toon mode, --fields projects each row to just the named fields (e.g. --fields alert_id,title,alert_severity,created_at) so you get a compact record without piping to jq. --limit max 100; --since/--until window must be < 31 days.", "Alerts", "ReadList"),
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

				req := &flashduty.AlertListRequest{
					StartTime:     startTime,
					EndTime:       endTime,
					AlertSeverity: severity,
				}
				req.Limit = limit
				req.Page = page

				// Preserve legacy semantics: --active sends is_active=true,
				// --recovered sends is_active=false, neither omits the filter.
				if active {
					req.IsActive = flashduty.Bool(true)
				} else if recovered {
					req.IsActive = flashduty.Bool(false)
				}

				if muted {
					req.EverMuted = flashduty.Bool(true)
				}

				if channel != "" {
					channelIDs, err := parseIntSlice(channel)
					if err != nil {
						return fmt.Errorf("invalid --channel: %w", err)
					}
					req.ChannelIDs = channelIDs
				}

				if integration != "" {
					integrationIDs, err := parseIntSlice(integration)
					if err != nil {
						return fmt.Errorf("invalid --integration: %w", err)
					}
					req.IntegrationIDs = integrationIDs
				}

				result, _, err := ctx.Client.Alerts.ReadList(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}

				if fields != "" && ctx.Structured() {
					proj, err := projectFields(result.Items, parseStringSlice(fields))
					if err != nil {
						return err
					}
					return ctx.PrintList(proj, nil, len(result.Items), page, int(result.Total))
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return v.(flashduty.AlertItem).AlertID }},
					{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.AlertItem).Title }},
					{Header: "SEVERITY", Field: func(v any) string { return v.(flashduty.AlertItem).AlertSeverity }},
					{Header: "STATUS", Field: func(v any) string { return v.(flashduty.AlertItem).AlertStatus }},
					{Header: "EVENTS", Field: func(v any) string { return fmt.Sprintf("%d", v.(flashduty.AlertItem).EventCnt) }},
					{Header: "CHANNEL", Field: func(v any) string { return v.(flashduty.AlertItem).ChannelName }},
					{Header: "STARTED", Field: func(v any) string { return output.FormatTime(v.(flashduty.AlertItem).StartTime) }},
				}

				return ctx.PrintList(result.Items, cols, len(result.Items), page, int(result.Total))
			})
		},
	}

	cmd.Flags().StringVar(&severity, "severity", "", "Filter: Critical,Warning,Info")
	cmd.Flags().BoolVar(&active, "active", false, "Show active only")
	registerEnumFlag(cmd, "severity", severityEnum...)
	cmd.Flags().BoolVar(&recovered, "recovered", false, "Show recovered only")
	cmd.Flags().StringVar(&channel, "channel", "", "Comma-separated channel IDs")
	cmd.Flags().StringVar(&integration, "integration", "", "Comma-separated integration IDs")
	cmd.Flags().BoolVar(&muted, "muted", false, "Show ever-muted only")
	cmd.Flags().StringVar(&since, "since", "24h", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated fields to project in json/toon output (e.g. alert_id,title,alert_severity,created_at); ignored in table mode. Use to avoid dumping the full nested record.")

	return cmd
}

func newAlertGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <alert_id>",
		Short: "Get alert detail",
		Long:  curatedLong("Get the full detail of a single alert by ID.", "Alerts", "ReadInfo"),
		Args:  requireArgs("alert_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, _, err := ctx.Client.Alerts.ReadInfo(cmdContext(ctx.Cmd), &flashduty.AlertInfoRequest{
					AlertID: ctx.Args[0],
				})
				if err != nil {
					return err
				}

				if ctx.Structured() {
					return ctx.Printer.Print(result, nil)
				}

				printAlertDetail(ctx.Writer, result)
				return nil
			})
		},
	}
}

func printAlertDetail(w io.Writer, a *flashduty.AlertItem) {
	if a == nil {
		return
	}

	labels := make([]string, 0, len(a.Labels))
	for k, v := range a.Labels {
		labels = append(labels, k+"="+v)
	}

	incidentInfo := "-"
	if a.Incident.IncidentID != "" {
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
		Long:  curatedLong("List the individual events that compose a given alert.", "Alerts", "ReadEventList"),
		Args:  requireArgs("alert_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, _, err := ctx.Client.Alerts.ReadEventList(cmdContext(ctx.Cmd), &flashduty.AlertEventListRequest{
					AlertID: ctx.Args[0],
				})
				if err != nil {
					return err
				}

				if len(result.Items) == 0 {
					ctx.WriteResult("No alert events found.")
					return nil
				}

				cols := []output.Column{
					{Header: "EVENT_ID", Field: func(v any) string { return v.(flashduty.AlertEventItem).EventID }},
					{Header: "SEVERITY", Field: func(v any) string { return v.(flashduty.AlertEventItem).EventSeverity }},
					{Header: "STATUS", Field: func(v any) string { return v.(flashduty.AlertEventItem).EventStatus }},
					{Header: "TIME", Field: func(v any) string { return output.FormatTime(v.(flashduty.AlertEventItem).EventTime) }},
					{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.AlertEventItem).Title }},
				}

				return ctx.PrintTotal(result.Items, cols, len(result.Items))
			})
		},
	}
}

func newAlertTimelineCmd() *cobra.Command {
	var limit, page int

	cmd := &cobra.Command{
		Use:   "timeline <alert_id>",
		Short: "View alert timeline",
		Long:  curatedLong("View the chronological feed of timeline events (actions, state changes) for an alert.", "Alerts", "ReadFeed"),
		Args:  requireArgs("alert_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				req := &flashduty.AlertFeedRequest{AlertID: ctx.Args[0]}
				req.Limit = limit
				req.Page = page

				result, _, err := ctx.Client.Alerts.ReadFeed(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}

				if len(result.Items) == 0 {
					ctx.WriteResult("No timeline events.")
					return nil
				}

				// go-flashduty returns raw feed items, so replicate the legacy
				// SDK's operator-name enrichment by resolving each entry's actor
				// (creator) person ID via /person/infos. Best-effort: the
				// OPERATOR column falls back to the numeric ID when a name can't
				// be resolved.
				nameByID := resolveAlertFeedOperators(ctx, result.Items)

				cols := []output.Column{
					{Header: "TIME", Field: func(v any) string { return output.FormatTime(v.(flashduty.FeedItem).CreatedAt) }},
					{Header: "TYPE", Field: func(v any) string { return string(v.(flashduty.FeedItem).Type) }},
					{Header: "OPERATOR", Field: func(v any) string {
						it := v.(flashduty.FeedItem)
						if it.CreatorID == 0 {
							return "system"
						}
						if n, ok := nameByID[it.CreatorID]; ok && n != "" {
							return n
						}
						return strconv.FormatInt(it.CreatorID, 10)
					}},
					{Header: "DETAIL", MaxWidth: 80, Field: func(v any) string {
						d := v.(flashduty.FeedItem).Detail
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

// newAlertMergeCmd curates the generated alert-write-merge command solely to
// replace its inline --comment string with --comment-file, closing the same
// shell-interpolation exposure that motivated incident comment's
// --comment-file (see newIncidentCommentCmd): an LLM-authored merge comment
// containing backticks, $(...), or quotes must never be typed into a shell
// argument. Everything else (positional alert IDs, --alert-ids, --incident-id,
// --owner-id, --title, --data) mirrors the generated command byte-for-byte via
// the same genAssembleBody/genBindBody request-building helpers, so this is a
// pure flag-source swap, not a behavior change. Registering it here (curated
// commands are added before registerGenerated runs, see root.go) makes
// genAddLeaf skip attaching the generated "merge" leaf under this same name.
func newAlertMergeCmd() *cobra.Command {
	var dataJSON string
	var fAlertIDs []string
	var fCommentFile string
	var fIncidentID string
	var fOwnerID int64
	var fTitle string

	cmd := &cobra.Command{
		Use:   "merge <alert-id> [<id2>...]",
		Short: "Merge alerts into an incident",
		Long: `Merge alerts into an incident.

Associate one or more alerts with an existing incident. If a source alert previously belonged to a different incident and that incident becomes empty after the merge, it will be automatically closed.

An optional comment on the merge action is read verbatim from --comment-file
(or from stdin, when the value is "-"); it is never parsed by a shell, so
backticks, $(...), and quotes inside it reach the API exactly as written.

API: POST /alert/merge (alert-write-merge)

Request fields:
  --alert-ids []string (required) — Alert IDs to merge.
  --comment-file string — Path to a file containing an optional comment on the merge action (- reads stdin).
  --incident-id string (required) — Target incident ID.
  --owner-id int — Optional new owner for the target incident.
  --title string — Optional new title for the target incident.
`,
		Args:    requireBodyFieldOrArgs("alert_ids", "alert-ids"),
		Example: `  flashduty alert merge <alert-id1> <alert-id2> --incident-id <incident-id> --comment-file ./comment.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				body, err := genAssembleBody(dataJSON, func(body map[string]any) error {
					if err := genFoldPositional(args, body, "alert_ids", "slice"); err != nil {
						return err
					}
					if cmd.Flags().Changed("alert-ids") {
						body["alert_ids"] = fAlertIDs
					}
					if cmd.Flags().Changed("comment-file") {
						comment, err := resolveCommentFile(fCommentFile)
						if err != nil {
							return err
						}
						body["comment"] = comment
					}
					if cmd.Flags().Changed("incident-id") {
						body["incident_id"] = fIncidentID
					}
					if cmd.Flags().Changed("owner-id") {
						body["owner_id"] = fOwnerID
					}
					if cmd.Flags().Changed("title") {
						body["title"] = fTitle
					}
					return nil
				})
				if err != nil {
					return err
				}
				req := new(flashduty.AlertMergeRequest)
				if err := genBindBody(body, req); err != nil {
					return err
				}
				resp, err := ctx.Client.Alerts.WriteMerge(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}
				if resp != nil && len(resp.Raw) > 0 {
					return ctx.WriteRaw(resp.Raw)
				}
				ctx.WriteResult("OK: POST /alert/merge")
				return nil
			})
		},
	}

	cmd.Flags().StringSliceVar(&fAlertIDs, "alert-ids", nil, "Alert IDs to merge. (required)")
	cmd.Flags().StringVar(&fCommentFile, "comment-file", "", "Path to a file containing an optional comment on the merge action (- reads stdin)")
	cmd.Flags().StringVar(&fIncidentID, "incident-id", "", "Target incident ID. (required)")
	cmd.Flags().Int64Var(&fOwnerID, "owner-id", 0, "Optional new owner for the target incident.")
	cmd.Flags().StringVar(&fTitle, "title", "", "Optional new title for the target incident.")
	cmd.Flags().StringVar(&dataJSON, "data", "", "Full request body as JSON; positional arguments and typed flags override its fields. Accepts inline JSON, or - to read stdin.")

	return cmd
}

// resolveAlertFeedOperators resolves the actor (creator) person IDs of
// alert-feed items to display names via /person/infos, replicating the
// operator-name enrichment the legacy SDK did server-side. Best-effort: a
// lookup failure yields a nil map and callers fall back to the numeric ID.
func resolveAlertFeedOperators(rc *RunContext, items []flashduty.FeedItem) map[int64]string {
	seen := make(map[int64]struct{}, len(items))
	ids := make([]uint64, 0, len(items))
	for _, it := range items {
		if it.CreatorID == 0 {
			continue
		}
		if _, ok := seen[it.CreatorID]; ok {
			continue
		}
		seen[it.CreatorID] = struct{}{}
		ids = append(ids, uint64(it.CreatorID))
	}
	if len(ids) == 0 {
		return nil
	}
	resp, _, err := rc.Client.Members.PersonInfos(cmdContext(rc.Cmd), &flashduty.PersonInfosRequest{PersonIDs: ids})
	if err != nil || resp == nil {
		return nil
	}
	out := make(map[int64]string, len(resp.Items))
	for _, p := range resp.Items {
		out[int64(p.PersonID)] = p.PersonName
	}
	return out
}
