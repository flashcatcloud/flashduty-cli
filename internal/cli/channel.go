package cli

import (
	"strconv"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

func newChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel",
		Short: "Manage channels",
	}
	cmd.AddCommand(newChannelListCmd())
	return cmd
}

// channelRow is the display projection for the channel list. go-flashduty's
// ChannelItem carries only TeamID/CreatorID, so we keep those IDs and resolve
// the team and creator names here (replicating the legacy SDK's enrichChannels)
// before rendering.
type channelRow struct {
	channelID   int64
	channelName string
	teamID      int64
	creatorID   int64
	teamName    string
	creatorName string
}

func newChannelListCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List channels",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				// Legacy parity: the hand-written SDK called /channel/list with an
				// empty body and applied the --name filter client-side as a
				// case-insensitive substring match. go-flashduty's ChannelName field
				// is an exact-match server filter, so we keep the client-side filter
				// to preserve behavior.
				result, _, err := ctx.Client.Channels.ChannelList(cmdContext(ctx.Cmd), &flashduty.ListChannelsRequest{})
				if err != nil {
					return err
				}

				rows := make([]channelRow, 0, len(result.Items))
				for _, ch := range result.Items {
					if name != "" && !strings.Contains(strings.ToLower(ch.ChannelName), strings.ToLower(name)) {
						continue
					}
					rows = append(rows, channelRow{
						channelID:   ch.ChannelID,
						channelName: ch.ChannelName,
						teamID:      ch.TeamID,
						creatorID:   ch.CreatorID,
					})
				}

				// Replicate the legacy enrichment: resolve TeamID -> TeamName and
				// CreatorID -> CreatorName. Best-effort, matching the legacy SDK
				// which swallowed lookup errors and left names blank.
				enrichChannelNames(ctx, rows)

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return strconv.FormatInt(v.(channelRow).channelID, 10) }},
					{Header: "NAME", Field: func(v any) string { return v.(channelRow).channelName }},
					{Header: "TEAM", Field: func(v any) string { return v.(channelRow).teamName }},
					{Header: "CREATOR", Field: func(v any) string { return v.(channelRow).creatorName }},
				}

				return ctx.PrintTotal(rows, cols, len(rows))
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Search by name")

	return cmd
}

// enrichChannelNames resolves each row's team and creator IDs to display names
// via /team/infos and /person/infos, filling teamName/creatorName in place.
// Best-effort: a lookup failure leaves the corresponding name blank, mirroring
// the legacy SDK's enrichChannels (which swallowed errors).
func enrichChannelNames(ctx *RunContext, rows []channelRow) {
	if len(rows) == 0 {
		return
	}

	teamSeen := make(map[int64]struct{}, len(rows))
	teamIDs := make([]uint64, 0, len(rows))
	personSeen := make(map[int64]struct{}, len(rows))
	personIDs := make([]uint64, 0, len(rows))
	for _, r := range rows {
		if r.teamID != 0 {
			if _, ok := teamSeen[r.teamID]; !ok {
				teamSeen[r.teamID] = struct{}{}
				teamIDs = append(teamIDs, uint64(r.teamID))
			}
		}
		if r.creatorID != 0 {
			if _, ok := personSeen[r.creatorID]; !ok {
				personSeen[r.creatorID] = struct{}{}
				personIDs = append(personIDs, uint64(r.creatorID))
			}
		}
	}

	teamNameByID := make(map[int64]string)
	if len(teamIDs) > 0 {
		if resp, _, err := ctx.Client.Teams.ReadInfos(cmdContext(ctx.Cmd), &flashduty.TeamInfosRequest{TeamIDs: teamIDs}); err == nil && resp != nil {
			for _, t := range resp.Items {
				teamNameByID[int64(t.TeamID)] = t.TeamName
			}
		}
	}

	personNameByID := make(map[int64]string)
	if len(personIDs) > 0 {
		if resp, _, err := ctx.Client.Members.PersonInfos(cmdContext(ctx.Cmd), &flashduty.PersonInfosRequest{PersonIDs: personIDs}); err == nil && resp != nil {
			for _, p := range resp.Items {
				personNameByID[int64(p.PersonID)] = p.PersonName
			}
		}
	}

	for i := range rows {
		rows[i].teamName = teamNameByID[rows[i].teamID]
		rows[i].creatorName = personNameByID[rows[i].creatorID]
	}
}
