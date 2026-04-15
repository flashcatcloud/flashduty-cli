package cli

import (
	"fmt"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
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
	var channelID int64
	var since, until string
	var limit, page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List changes",
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

				result, err := ctx.Client.ListChanges(cmdContext(ctx.Cmd), &flashduty.ListChangesInput{
					ChannelID: channelID,
					StartTime: startTime,
					EndTime:   endTime,
					Limit:     limit,
					Page:      page,
				})
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return v.(flashduty.Change).ChangeID }},
					{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.Change).Title }},
					{Header: "STATUS", Field: func(v any) string { return v.(flashduty.Change).Status }},
					{Header: "CHANNEL", Field: func(v any) string { return v.(flashduty.Change).ChannelName }},
					{Header: "TIME", Field: func(v any) string { return output.FormatTime(v.(flashduty.Change).StartTime) }},
				}

				return ctx.PrintList(result.Changes, cols, len(result.Changes), page, result.Total)
			})
		},
	}

	cmd.Flags().Int64Var(&channelID, "channel", 0, "Filter by channel ID")
	cmd.Flags().StringVar(&since, "since", "24h", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}
