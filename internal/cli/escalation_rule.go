package cli

import (
	"fmt"
	"strconv"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/spf13/cobra"
)

func newEscalationRuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "escalation-rule",
		Short: "Manage escalation rules",
	}
	cmd.AddCommand(newEscalationRuleListCmd())
	return cmd
}

func newEscalationRuleListCmd() *cobra.Command {
	var channelID int64
	var channelName string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List escalation rules for a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			// Resolve channel name to ID if needed
			if channelID == 0 && channelName != "" {
				resolved, err := resolveChannelID(cmd, client, channelName)
				if err != nil {
					return err
				}
				channelID = resolved
			}

			if channelID == 0 {
				return fmt.Errorf("--channel or --channel-name is required")
			}

			result, err := client.ListEscalationRules(cmdContext(cmd), channelID)
			if err != nil {
				return err
			}

			cols := []output.Column{
				{Header: "ID", Field: func(v any) string { return v.(flashduty.EscalationRule).RuleID }},
				{Header: "NAME", Field: func(v any) string { return v.(flashduty.EscalationRule).RuleName }},
				{Header: "CHANNEL", Field: func(v any) string { return v.(flashduty.EscalationRule).ChannelName }},
				{Header: "STATUS", Field: func(v any) string { return v.(flashduty.EscalationRule).Status }},
				{Header: "PRIORITY", Field: func(v any) string { return strconv.Itoa(v.(flashduty.EscalationRule).Priority) }},
				{Header: "LAYERS", Field: func(v any) string { return strconv.Itoa(len(v.(flashduty.EscalationRule).Layers)) }},
			}

			return newPrinter(cmd.OutOrStdout()).Print(result.Rules, cols)
		},
	}

	cmd.Flags().Int64Var(&channelID, "channel", 0, "Channel ID")
	cmd.Flags().StringVar(&channelName, "channel-name", "", "Channel name (resolved to ID)")

	return cmd
}

// resolveChannelID resolves a channel name to its ID.
func resolveChannelID(cmd *cobra.Command, client flashdutyClient, name string) (int64, error) {
	result, err := client.ListChannels(cmdContext(cmd), &flashduty.ListChannelsInput{
		Name: name,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to resolve channel name: %w", err)
	}

	switch len(result.Channels) {
	case 0:
		return 0, fmt.Errorf("no channel found matching %q", name)
	case 1:
		return result.Channels[0].ChannelID, nil
	default:
		fmt.Fprintln(cmd.OutOrStdout(), "Multiple channels match:")
		for _, ch := range result.Channels {
			fmt.Fprintf(cmd.OutOrStdout(), "  %d  %s\n", ch.ChannelID, ch.ChannelName)
		}
		return 0, fmt.Errorf("multiple channels match %q, use --channel <id> to specify", name)
	}
}
