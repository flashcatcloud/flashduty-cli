package cli

import (
	"fmt"
	"strconv"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
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
			return runCommand(cmd, args, func(ctx *RunContext) error {
				// Resolve channel name to ID if needed
				if channelID == 0 && channelName != "" {
					resolved, err := resolveChannelID(ctx, channelName)
					if err != nil {
						return err
					}
					channelID = resolved
				}

				if channelID == 0 {
					return fmt.Errorf("--channel or --channel-name is required")
				}

				result, _, err := ctx.Client.Channels.ChannelEscalateRuleList(cmdContext(ctx.Cmd), &flashduty.ChannelScopedListRequest{
					ChannelID: channelID,
				})
				if err != nil {
					return err
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return v.(flashduty.EscalateRuleItem).RuleID }},
					{Header: "NAME", Field: func(v any) string { return v.(flashduty.EscalateRuleItem).RuleName }},
					{Header: "CHANNEL", Field: func(v any) string { return v.(flashduty.EscalateRuleItem).ChannelName }},
					{Header: "STATUS", Field: func(v any) string { return v.(flashduty.EscalateRuleItem).Status }},
					{Header: "PRIORITY", Field: func(v any) string {
						return strconv.FormatInt(v.(flashduty.EscalateRuleItem).Priority, 10)
					}},
					{Header: "LAYERS", Field: func(v any) string {
						return strconv.Itoa(len(v.(flashduty.EscalateRuleItem).Layers))
					}},
				}

				return ctx.Printer.Print(result.Items, cols)
			})
		},
	}

	cmd.Flags().Int64Var(&channelID, "channel", 0, "Channel ID")
	cmd.Flags().StringVar(&channelName, "channel-name", "", "Channel name (resolved to ID)")

	return cmd
}

// resolveChannelID resolves a channel name to its ID.
func resolveChannelID(ctx *RunContext, name string) (int64, error) {
	result, _, err := ctx.Client.Channels.ChannelList(cmdContext(ctx.Cmd), &flashduty.ListChannelsRequest{
		ChannelName: name,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to resolve channel name: %w", err)
	}

	switch len(result.Items) {
	case 0:
		return 0, fmt.Errorf("no channel found matching %q", name)
	case 1:
		return result.Items[0].ChannelID, nil
	default:
		_, _ = fmt.Fprintln(ctx.Cmd.OutOrStdout(), "Multiple channels match:")
		for _, ch := range result.Items {
			_, _ = fmt.Fprintf(ctx.Cmd.OutOrStdout(), "  %d  %s\n", ch.ChannelID, ch.ChannelName)
		}
		return 0, fmt.Errorf("multiple channels match %q, use --channel <id> to specify", name)
	}
}
