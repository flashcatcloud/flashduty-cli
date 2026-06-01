package cli

import (
	"fmt"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

func newChangeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change",
		Short: "Manage changes",
	}
	cmd.AddCommand(newChangeListCmd())
	return cmd
}

func newChangeListCmd() *cobra.Command {
	var channel string
	var since, until string
	var limit, page int
	var query, integration string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List changes",
		Long:  curatedLong("List changes recorded in the change feed. Time window must be < 31 days; --limit max is 100.", "Changes", "List"),
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

				return ctx.PrintList(result.Items, cols, len(result.Items), page, int(result.Total))
			})
		},
	}

	cmd.Flags().StringVar(&channel, "channel", "", "Comma-separated channel IDs")
	cmd.Flags().StringVar(&query, "query", "", "Free-text/regex search over change fields")
	cmd.Flags().StringVar(&integration, "integration", "", "Comma-separated reporting integration IDs")
	cmd.Flags().StringVar(&since, "since", "24h", "Start time (accepts 7d/24h/now, RFC3339, or Unix epoch; window must be < 31 days)")
	cmd.Flags().StringVar(&until, "until", "now", "End time (accepts 7d/24h/now, RFC3339, or Unix epoch)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results (max 100)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}
