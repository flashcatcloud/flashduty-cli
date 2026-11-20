package cli

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

func newInsightCmd() *cobra.Command {
	cmd := newGroupCmd("insight", "Query aggregated incident metrics by team, responder, or channel")
	// insight team/channel/responder are now served by the generated commands
	// (richer flag set: severities, *_ids, fields, aggregate-unit, …; relative
	// time on --start-time/--end-time). Their human tables are preserved via the
	// DimensionInsightItem / ResponderInsightItem entries in display_columns.go.
	// incident-export is curated (below) so it can verify the exported CSV
	// against the incident-list total; the generated twin is dropped by
	// genAddLeaf.
	cmd.AddCommand(newInsightTopAlertsCmd())
	cmd.AddCommand(newInsightIncidentsCmd())
	cmd.AddCommand(newInsightIncidentExportCmd())
	return cmd
}

func newInsightTopAlertsCmd() *cobra.Command {
	var label, since, until string
	var limit int
	var channelIDs []int64

	cmd := &cobra.Command{
		Use:   "top-alerts",
		Short: "Query top alert sources by label",
		Long:  curatedLong("Query the top-K noisiest alert sources grouped by a label dimension over a time window.", "Analytics", "TopkAlertsByLabel"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				startTime, endTime, err := parseWindow(cmd, since, until, "")
				if err != nil {
					return err
				}

				result, _, err := ctx.Client.Analytics.TopkAlertsByLabel(cmdContext(ctx.Cmd), &flashduty.InsightTopkAlertByLabelRequest{
					StartTime:  startTime,
					EndTime:    endTime,
					Label:      label,
					K:          int64(limit),
					ChannelIDs: channelIDs,
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

				return ctx.PrintPage(result.Items, cols, listPage{Count: len(result.Items), Limit: limit, Total: -1})
			})
		},
	}

	cmd.Flags().StringVar(&label, "label", "", "Group-by label dimension: one of [check, resource] (required)")
	cmd.Flags().StringVar(&since, "since", "7d", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")
	cmd.Flags().IntVar(&limit, "limit", 10, "Top K results")
	channelIDsFlag(cmd, &channelIDs)
	_ = cmd.MarkFlagRequired("label")

	return cmd
}

func newInsightIncidentsCmd() *cobra.Command {
	var since, until, fields string
	var limit, page int

	cmd := &cobra.Command{
		Use:   "incidents",
		Short: "Query incidents with performance metrics",
		Long:  curatedLong("List incidents with per-incident performance metrics (MTTA, MTTR, notifications) over a time window. In json/toon mode, rows carry every field; pass --fields to project a subset.", "Analytics", "IncidentList"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if err := checkFieldsFlag(ctx, fields); err != nil {
					return err
				}
				startTime, endTime, err := parseWindow(cmd, since, until, "")
				if err != nil {
					return err
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

				if fields != "" && ctx.Structured() {
					proj, err := projectFields(result.Items, parseStringSlice(fields))
					if err != nil {
						return err
					}
					return ctx.PrintList(proj, nil, len(proj), page, limit, int(result.Total))
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

				return ctx.PrintList(result.Items, cols, len(result.Items), page, limit, int(result.Total))
			})
		},
	}

	cmd.Flags().StringVar(&since, "since", "7d", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results (max 100)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated fields to project in json/toon output (e.g. incident_id,title,severity); ignored in table mode. Use to avoid dumping the full nested record.")

	return cmd
}

// insightExportRange is the window incident-export's --since/--until write,
// in the same dialect as the generated insight verbs (see timeRange).
var insightExportRange = timeRange{Start: "start_time", End: "end_time", Where: "--data or --start-time/--end-time"}

const (
	insightTimeForms   = " Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds."
	insightExportSince = "Window start, written to start_time (Unix seconds): a duration before now (30m, 2h, 7d), 'now', RFC3339, a date, a datetime, or Unix seconds/ms (one-second precision). Without --until the window ends now. Setting a field --data or --start-time/--end-time also sets is an error."
	insightExportUntil = "Window end, written to end_time (Unix seconds); same forms as --since, plus '+2h' for the future."
)

// newInsightIncidentExportCmd exports the filtered incident analytics list as
// CSV. It replaces the generated incident-export command (same flags) because
// the export endpoint answers a one-shot CSV with no pagination cursor and
// caps its row count server-side without saying so in the payload: a wide
// time window comes back silently truncated. Since the endpoint cannot be
// paged, the command verifies completeness instead — it counts the CSV data
// rows and compares against the incident-list total for the same filter (the
// authoritative count the analytics dashboard pages over). A short export is
// still written to stdout, but the command exits non-zero stating written vs
// total, so a truncated CSV can never pass for a complete one.
func newInsightIncidentExportCmd() *cobra.Command {
	var dataJSON string
	var fAsc bool
	var fChannelIDs []int
	var fDescriptionHTMLToText bool
	var fEndTime string
	var fExportFields []string
	var fIncidentIDs []string
	var fIncludeEverMuted bool
	var fIsMyTeam bool
	var fOrderby string
	var fQuery string
	var fResponderIDs []int
	var fSecondsToAckFrom int64
	var fSecondsToAckTo int64
	var fSecondsToCloseFrom int64
	var fSecondsToCloseTo int64
	var fSeverities []string
	var fStartTime string
	var fTeamIDs []int
	var fTimeZone string
	var fSince, fUntil string
	cmd := &cobra.Command{
		Use:   "incident-export",
		Short: "Export insight incidents",
		Long: `Export insight incidents.

Export the filtered incident analytics list as a CSV file (to stdout; redirect with '> file.csv'). CSV headers and formatted values use the request locale, falling back to the member locale and then the account locale. Set the window with --since/--until (or --start-time/--end-time): a duration (30d, 24h), 'now', RFC3339, a date, or Unix seconds.

The export endpoint returns a one-shot CSV and caps its row count server-side, so a wide window may come back truncated. After writing, this command compares the CSV data-row count against the incident-list total for the same filter: on a shortfall the partial CSV is still written, a 'rows=N' line (the actual written data-row count) goes to stderr, and the command exits non-zero stating written vs total — narrow the window and retry.

API: POST /insight/incident/export (insightIncidentExport)

Request fields:
  --asc bool
  --channel-ids []int
  --description-html-to-text bool
  --end-time string` + insightTimeForms + `
  --export-fields []string
  --incident-ids []string
  --include-ever-muted bool
  --is-my-team bool
  --orderby string
  --query string
  --responder-ids []int
  --seconds-to-ack-from int
  --seconds-to-ack-to int
  --seconds-to-close-from int
  --seconds-to-close-to int
  --severities []string
  --start-time string` + insightTimeForms + `
  --team-ids []int
  --time-zone string
  --since string — ` + insightExportSince + `
  --until string — ` + insightExportUntil + `
  fields (JSON, via --data)
  labels (JSON, via --data)`,
		Example: `  flashduty insight incident-export --since 7d > incidents.csv
  flashduty insight incident-export --since 2026-04-01 --until 2026-04-08 > incidents.csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				vEndTime, okEndTime, err := genParseTimeFlag(cmd, "end-time", fEndTime)
				if err != nil {
					return err
				}
				vStartTime, okStartTime, err := genParseTimeFlag(cmd, "start-time", fStartTime)
				if err != nil {
					return err
				}
				body, err := genAssembleBody(dataJSON, func(body map[string]any) error {
					if cmd.Flags().Changed("asc") {
						body["asc"] = fAsc
					}
					if cmd.Flags().Changed("channel-ids") {
						body["channel_ids"] = fChannelIDs
					}
					if cmd.Flags().Changed("description-html-to-text") {
						body["description_html_to_text"] = fDescriptionHTMLToText
					}
					if okEndTime {
						body["end_time"] = vEndTime
					}
					if cmd.Flags().Changed("export-fields") {
						body["export_fields"] = fExportFields
					}
					if cmd.Flags().Changed("incident-ids") {
						body["incident_ids"] = fIncidentIDs
					}
					if cmd.Flags().Changed("include-ever-muted") {
						body["include_ever_muted"] = fIncludeEverMuted
					}
					if cmd.Flags().Changed("is-my-team") {
						body["is_my_team"] = fIsMyTeam
					}
					if cmd.Flags().Changed("orderby") {
						body["orderby"] = fOrderby
					}
					if cmd.Flags().Changed("query") {
						body["query"] = fQuery
					}
					if cmd.Flags().Changed("responder-ids") {
						body["responder_ids"] = fResponderIDs
					}
					if cmd.Flags().Changed("seconds-to-ack-from") {
						body["seconds_to_ack_from"] = fSecondsToAckFrom
					}
					if cmd.Flags().Changed("seconds-to-ack-to") {
						body["seconds_to_ack_to"] = fSecondsToAckTo
					}
					if cmd.Flags().Changed("seconds-to-close-from") {
						body["seconds_to_close_from"] = fSecondsToCloseFrom
					}
					if cmd.Flags().Changed("seconds-to-close-to") {
						body["seconds_to_close_to"] = fSecondsToCloseTo
					}
					if cmd.Flags().Changed("severities") {
						body["severities"] = fSeverities
					}
					if okStartTime {
						body["start_time"] = vStartTime
					}
					if cmd.Flags().Changed("team-ids") {
						body["team_ids"] = fTeamIDs
					}
					if cmd.Flags().Changed("time-zone") {
						body["time_zone"] = fTimeZone
					}
					return nil
				})
				if err != nil {
					return err
				}
				if err := genApplyRange(cmd, body, insightExportRange, fSince, fUntil); err != nil {
					return err
				}
				req := new(flashduty.InsightFilter)
				if err := genBindBody(ctx.Cmd.ErrOrStderr(), body, req); err != nil {
					return err
				}
				resp, err := ctx.Client.Analytics.IncidentExport(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}
				if resp == nil || len(resp.Raw) == 0 {
					return printNoPayload(ctx, resp)
				}
				rows, err := countCSVDataRows(resp.Raw)
				if err != nil {
					return fmt.Errorf("insight incident-export: cannot parse the exported CSV: %w", err)
				}
				if err := ctx.WriteRaw(resp.Raw); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(ctx.Cmd.ErrOrStderr(), "rows=%d\n", rows)
				total, err := insightIncidentTotal(ctx, body)
				if err != nil {
					return fmt.Errorf("insight incident-export: wrote %d rows but cannot verify completeness against /insight/incident/list: %w", rows, err)
				}
				if int64(rows) < total {
					return fmt.Errorf("insight incident-export: incomplete export — wrote %d of %d incidents matching the filter; narrow the time window (--since/--until) and retry", rows, total)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&fAsc, "asc", false, "Request field ")
	cmd.Flags().IntSliceVar(&fChannelIDs, "channel-ids", nil, "Request field ")
	cmd.Flags().BoolVar(&fDescriptionHTMLToText, "description-html-to-text", false, "Request field ")
	cmd.Flags().StringVar(&fEndTime, "end-time", "", "Window end, Unix seconds."+insightTimeForms)
	cmd.Flags().StringSliceVar(&fExportFields, "export-fields", nil, "Request field ")
	cmd.Flags().StringSliceVar(&fIncidentIDs, "incident-ids", nil, "Request field ")
	cmd.Flags().BoolVar(&fIncludeEverMuted, "include-ever-muted", false, "Request field ")
	cmd.Flags().BoolVar(&fIsMyTeam, "is-my-team", false, "Request field ")
	cmd.Flags().StringVar(&fOrderby, "orderby", "", "Request field ")
	cmd.Flags().StringVar(&fQuery, "query", "", "Request field ")
	cmd.Flags().IntSliceVar(&fResponderIDs, "responder-ids", nil, "Request field ")
	cmd.Flags().Int64Var(&fSecondsToAckFrom, "seconds-to-ack-from", 0, "Request field ")
	cmd.Flags().Int64Var(&fSecondsToAckTo, "seconds-to-ack-to", 0, "Request field ")
	cmd.Flags().Int64Var(&fSecondsToCloseFrom, "seconds-to-close-from", 0, "Request field ")
	cmd.Flags().Int64Var(&fSecondsToCloseTo, "seconds-to-close-to", 0, "Request field ")
	cmd.Flags().StringSliceVar(&fSeverities, "severities", nil, "Request field ")
	cmd.Flags().StringVar(&fStartTime, "start-time", "", "Window start, Unix seconds."+insightTimeForms)
	cmd.Flags().IntSliceVar(&fTeamIDs, "team-ids", nil, "Request field ")
	cmd.Flags().StringVar(&fTimeZone, "time-zone", "", "Request field ")
	cmd.Flags().StringVar(&fSince, "since", "", insightExportSince)
	cmd.Flags().StringVar(&fUntil, "until", "", insightExportUntil)
	cmd.Flags().StringVar(&dataJSON, "data", "", "Full request body as JSON; positional arguments and typed flags override its fields. Accepts inline JSON, or - to read stdin.")
	return cmd
}

// countCSVDataRows counts the data records in an exported CSV body — every
// record except the header row. encoding/csv handles quoted fields with
// embedded newlines, which a naive line count would over-count.
func countCSVDataRows(raw []byte) (int, error) {
	r := csv.NewReader(bytes.NewReader(raw))
	r.FieldsPerRecord = -1
	n := 0
	for {
		if _, err := r.Read(); err == io.EOF {
			break
		} else if err != nil {
			return 0, err
		}
		n++
	}
	if n == 0 {
		return 0, nil
	}
	return n - 1, nil
}

// insightIncidentTotal returns the number of incidents matching the export
// filter, per the incident-list endpoint — the authoritative total an export
// is verified against. Only the total is needed, so a single 1-item page is
// fetched.
func insightIncidentTotal(ctx *RunContext, body map[string]any) (int64, error) {
	// The export call already warned about this body's unsendable fields.
	req := new(flashduty.InsightIncidentListRequest)
	if err := genBindBody(io.Discard, body, req); err != nil {
		return 0, err
	}
	req.Page = 1
	req.Limit = 1
	out, _, err := ctx.Client.Analytics.IncidentList(cmdContext(ctx.Cmd), req)
	if err != nil {
		return 0, err
	}
	return out.Total, nil
}
