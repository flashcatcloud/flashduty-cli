package cli

import (
	"fmt"
	"strconv"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/spf13/cobra"
)

func newChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel",
		Short: "Manage channels",
	}
	cmd.AddCommand(newChannelListCmd())
	return cmd
}

func newChannelListCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List channels",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			result, err := client.ListChannels(cmdContext(cmd), &flashduty.ListChannelsInput{
				Name: name,
			})
			if err != nil {
				return err
			}

			cols := []output.Column{
				{Header: "ID", Field: func(v any) string { return strconv.FormatInt(v.(flashduty.ChannelInfo).ChannelID, 10) }},
				{Header: "NAME", Field: func(v any) string { return v.(flashduty.ChannelInfo).ChannelName }},
				{Header: "TEAM", Field: func(v any) string { return v.(flashduty.ChannelInfo).TeamName }},
				{Header: "CREATOR", Field: func(v any) string { return v.(flashduty.ChannelInfo).CreatorName }},
			}

			p := newPrinter(nil)
			if err := p.Print(result.Channels, cols); err != nil {
				return err
			}
			fmt.Printf("Total: %d\n", result.Total)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Search by name")

	return cmd
}
