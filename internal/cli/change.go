package cli

import (
	"fmt"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
	"github.com/spf13/cobra"
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
	var since, until, changeType string
	var limit, page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List changes",
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

			result, err := client.ListChanges(cmdContext(cmd), &flashduty.ListChangesInput{
				ChannelID: channelID,
				StartTime: startTime,
				EndTime:   endTime,
				Type:      changeType,
				Limit:     limit,
				Page:      page,
			})
			if err != nil {
				return err
			}

			cols := []output.Column{
				{Header: "ID", Field: func(v any) string { return v.(flashduty.Change).ChangeID }},
				{Header: "TITLE", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.Change).Title }},
				{Header: "TYPE", Field: func(v any) string { return v.(flashduty.Change).Type }},
				{Header: "STATUS", Field: func(v any) string { return v.(flashduty.Change).Status }},
				{Header: "CHANNEL", Field: func(v any) string { return v.(flashduty.Change).ChannelName }},
				{Header: "TIME", Field: func(v any) string { return output.FormatTime(v.(flashduty.Change).StartTime) }},
			}

			p := newPrinter(nil)
			if err := p.Print(result.Changes, cols); err != nil {
				return err
			}
			fmt.Printf("Showing %d results (page %d, total %d).\n", len(result.Changes), page, result.Total)
			return nil
		},
	}

	cmd.Flags().Int64Var(&channelID, "channel", 0, "Filter by channel ID")
	cmd.Flags().StringVar(&since, "since", "24h", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")
	cmd.Flags().StringVar(&changeType, "type", "", "Filter by change type")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}
