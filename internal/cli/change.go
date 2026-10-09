package cli

import (
	"encoding/json"
	"fmt"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

func newChangeCmd() *cobra.Command {
	cmd := newGroupCmd("change", "Manage changes")
	cmd.AddCommand(newChangeListCmd())
	return cmd
}

func newChangeListCmd() *cobra.Command {
	var channel string
	var since, until string
	var limit, page int
	var query, integration, filters string
	var orderby string
	var asc, includeEvents bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List changes",
		Long:  curatedLong("List changes recorded in the change feed. Time window must be < 31 days; --limit max is 100.", "Changes", "List"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				startTime, endTime, err := parseWindow(cmd, since, until, "")
				if err != nil {
					return err
				}

				// The legacy SDK clamped non-positive paging to sane defaults
				// before sending; go-flashduty forwards values verbatim and the
				// server rejects limit/page < 1. Clamp here to preserve the old
				// "negative values don't error" behavior. The footer still shows
				// the raw --page value, matching the legacy command.
				reqLimit, reqPage := limit, page
				if reqLimit <= 0 {
					reqLimit = 20
				}
				if reqPage <= 0 {
					reqPage = 1
				}

				input := &flashduty.ListChangeRequest{
					StartTime: startTime,
					EndTime:   endTime,
				}
				input.Limit = reqLimit
				input.Page = reqPage

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
				input.Query = query
				input.Orderby = orderby
				input.Asc = asc
				input.IncludeEvents = includeEvents
				if filters != "" {
					if err := json.Unmarshal([]byte(filters), &input.Filters); err != nil {
						return fmt.Errorf("invalid --filters: %w", err)
					}
				}

				result, _, err := ctx.Client.Changes.List(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return v.(flashduty.ChangeItem).ChangeID }},
					{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.ChangeItem).Title }},
					{Header: "STATUS", Field: func(v any) string { return v.(flashduty.ChangeItem).ChangeStatus }},
					{Header: "CHANNEL", Field: func(v any) string { return v.(flashduty.ChangeItem).ChannelName }},
					{Header: "TIME", Field: func(v any) string { return output.FormatTime(v.(flashduty.ChangeItem).StartTime) }},
				}

				return ctx.PrintList(result.Items, cols, len(result.Items), page, limit, int(result.Total))
			})
		},
	}

	cmd.Flags().StringVar(&channel, "channel", "", "Comma-separated channel IDs")
	cmd.Flags().StringVar(&query, "query", "", "Free-text/regex search over change fields")
	cmd.Flags().StringVar(&integration, "integration", "", "Comma-separated reporting integration IDs")
	cmd.Flags().StringVar(&filters, "filters", "", `Structured filters ANDed onto the query, as a JSON array of {"key","oper","vals"} (oper IN or NOTIN; key like labels.env). Keys starting with "incident" are ignored`)
	cmd.Flags().StringVar(&orderby, "orderby", "", "Sort field: start_time (default) or last_time")
	cmd.Flags().BoolVar(&asc, "asc", false, "Sort in ascending order")
	cmd.Flags().BoolVar(&includeEvents, "include-events", false, "Include the underlying change events for each change")
	cmd.Flags().StringVar(&since, "since", "24h", "Start time (accepts 7d/24h/now, RFC3339, or Unix epoch; window must be < 31 days)")
	cmd.Flags().StringVar(&until, "until", "now", "End time (accepts 7d/24h/now, RFC3339, or Unix epoch)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results (max 100)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}
