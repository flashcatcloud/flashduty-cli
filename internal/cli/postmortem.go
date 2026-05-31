package cli

import (
	"fmt"

	gflashduty "github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

func newPostmortemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "postmortem",
		Short: "Manage post-mortems",
	}
	cmd.AddCommand(newPostmortemListCmd())
	return cmd
}

func newPostmortemListCmd() *cobra.Command {
	var status, channel, team, since, until string
	var limit, page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List post-mortem reports",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGFCommand(cmd, args, func(ctx *RunContext) error {
				req := &gflashduty.ListPostMortemsRequest{
					Status: status,
				}
				req.Page = page
				req.Limit = limit

				if channel != "" {
					channelIDs, err := parseIntSlice(channel)
					if err != nil {
						return fmt.Errorf("invalid --channel: %w", err)
					}
					req.ChannelIDs = channelIDs
				}

				if team != "" {
					teamIDs, err := parseIntSlice(team)
					if err != nil {
						return fmt.Errorf("invalid --team: %w", err)
					}
					req.TeamIDs = teamIDs
				}

				if since != "" {
					startTime, err := timeutil.Parse(since)
					if err != nil {
						return fmt.Errorf("invalid --since: %w", err)
					}
					req.CreatedAtStartSeconds = startTime
				}

				if until != "" {
					endTime, err := timeutil.Parse(until)
					if err != nil {
						return fmt.Errorf("invalid --until: %w", err)
					}
					req.CreatedAtEndSeconds = endTime
				}

				result, _, err := ctx.GFClient.Incidents.PostMortemList(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return v.(gflashduty.PostMortemMeta).PostMortemID }},
					{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(gflashduty.PostMortemMeta).Title }},
					{Header: "STATUS", Field: func(v any) string { return v.(gflashduty.PostMortemMeta).Status }},
					{Header: "CHANNEL", Field: func(v any) string { return v.(gflashduty.PostMortemMeta).ChannelName }},
					{Header: "CREATED", Field: func(v any) string {
						return output.FormatTime(v.(gflashduty.PostMortemMeta).CreatedAtSeconds)
					}},
				}

				return ctx.PrintList(result.Items, cols, len(result.Items), page, int(result.Total))
			})
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "Filter: drafting or published")
	cmd.Flags().StringVar(&channel, "channel", "", "Comma-separated channel IDs")
	cmd.Flags().StringVar(&team, "team", "", "Comma-separated team IDs")
	cmd.Flags().StringVar(&since, "since", "", "Created after (time filter)")
	cmd.Flags().StringVar(&until, "until", "", "Created before (time filter)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}
