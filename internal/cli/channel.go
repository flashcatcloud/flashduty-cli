package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

func newChannelCmd() *cobra.Command {
	cmd := newGroupCmd("channel", "Manage channels")
	cmd.AddCommand(newChannelListCmd())
	cmd.AddCommand(newChannelEscalateRuleListCmd())
	return cmd
}

// channelRow is the display projection for the channel list. go-flashduty's
// ChannelItem carries only TeamID/CreatorID, so we keep those IDs and resolve
// the team and creator names here (replicating the legacy SDK's enrichChannels)
// before rendering.
// Fields are exported with json tags so the json/toon printers (which marshal
// via reflection and skip unexported fields) emit the full row, not {}. The
// table printer uses the accessor funcs below. json keys mirror the legacy
// ChannelInfo contract (channel_id/channel_name/team_id/creator_id/...); TOON
// renders the Go field names, consistent with every other command's output.
type channelRow struct {
	ChannelID   int64  `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	TeamID      int64  `json:"team_id"`
	CreatorID   int64  `json:"creator_id"`
	TeamName    string `json:"team_name"`
	CreatorName string `json:"creator_name"`
}

func newChannelListCmd() *cobra.Command {
	var name string
	var teamIDs []int64

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List channels",
		Long:  curatedLong("List channels in the account, optionally filtered by name or owning team.", "Channels", "ChannelList"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				// Legacy parity: the hand-written SDK called /channel/list with an
				// empty body and applied the --name filter client-side as a
				// case-insensitive substring match. go-flashduty's ChannelName field
				// is an exact-match server filter, so we keep the client-side filter
				// to preserve behavior. --team-ids, by contrast, is a server-side
				// filter on the channel's owning team (empty = all teams, unchanged).
				result, _, err := ctx.Client.Channels.ChannelList(cmdContext(ctx.Cmd), &flashduty.ListChannelsRequest{TeamIDs: teamIDs})
				if err != nil {
					return err
				}

				rows := make([]channelRow, 0, len(result.Items))
				for _, ch := range result.Items {
					if name != "" && !strings.Contains(strings.ToLower(ch.ChannelName), strings.ToLower(name)) {
						continue
					}
					rows = append(rows, channelRow{
						ChannelID:   ch.ChannelID,
						ChannelName: ch.ChannelName,
						TeamID:      ch.TeamID,
						CreatorID:   ch.CreatorID,
					})
				}

				// Replicate the legacy enrichment: resolve TeamID -> TeamName and
				// CreatorID -> CreatorName. Best-effort, matching the legacy SDK
				// which swallowed lookup errors and left names blank.
				enrichChannelNames(ctx, rows)

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return strconv.FormatInt(v.(channelRow).ChannelID, 10) }},
					{Header: "NAME", Field: func(v any) string { return v.(channelRow).ChannelName }},
					{Header: "TEAM", Field: func(v any) string { return v.(channelRow).TeamName }},
					{Header: "CREATOR", Field: func(v any) string { return v.(channelRow).CreatorName }},
				}

				return ctx.PrintTotal(rows, cols, len(rows))
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Search by name")
	cmd.Flags().Int64SliceVar(&teamIDs, "team-ids", nil, "Filter by owning team ID(s), server-side (repeatable or comma-separated)")

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
		if r.TeamID != 0 {
			if _, ok := teamSeen[r.TeamID]; !ok {
				teamSeen[r.TeamID] = struct{}{}
				teamIDs = append(teamIDs, uint64(r.TeamID))
			}
		}
		if r.CreatorID != 0 {
			if _, ok := personSeen[r.CreatorID]; !ok {
				personSeen[r.CreatorID] = struct{}{}
				personIDs = append(personIDs, uint64(r.CreatorID))
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
		rows[i].TeamName = teamNameByID[rows[i].TeamID]
		rows[i].CreatorName = personNameByID[rows[i].CreatorID]
	}
}

func newChannelEscalateRuleListCmd() *cobra.Command {
	var dataJSON, fields string
	var fChannelID int64

	defaultStructuredFields := []string{"rule_id", "rule_name", "status", "priority", "filters"}

	cmd := &cobra.Command{
		Use:   "escalate-rule-list <channel-id>",
		Short: "List escalation rules",
		Long: curatedLong("List all escalation rules for a channel. In json/toon mode, rows default to the compact fields rule_id,rule_name,status,priority,filters; pass --fields to choose a different projection.",
			"Channels", "ChannelEscalateRuleList"),
		Args:    requireBodyFieldOrExactArg("channel_id", "channel-id"),
		Example: `  flashduty channel escalate-rule-list --data '{"channel_id":1001}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				body, err := genAssembleBody(dataJSON, func(body map[string]any) error {
					if err := genFoldPositional(args, body, "channel_id", "int"); err != nil {
						return err
					}
					if cmd.Flags().Changed("channel-id") {
						body["channel_id"] = fChannelID
					}
					return nil
				})
				if err != nil {
					return err
				}
				req := new(flashduty.ChannelScopedListRequest)
				if err := genBindBody(body, req); err != nil {
					return err
				}
				out, _, err := ctx.Client.Channels.ChannelEscalateRuleList(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}

				if ctx.Structured() {
					selectedFields := defaultStructuredFields
					if cmd.Flags().Changed("fields") {
						selectedFields = parseStringSlice(fields)
						if len(selectedFields) == 0 {
							return fmt.Errorf("--fields must name at least one field")
						}
					} else {
						noteDefaultProjection(cmd.ErrOrStderr(), selectedFields)
					}
					proj, err := projectFields(out.Items, selectedFields)
					if err != nil {
						return err
					}
					bounded, note, err := boundProjectedOutput(proj, compactListOutputLimit)
					if err != nil {
						return err
					}
					proj = bounded.([]map[string]any)
					noteProjectionBound(cmd.ErrOrStderr(), note)
					return ctx.PrintTotal(proj, nil, len(proj))
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return v.(flashduty.EscalateRuleItem).RuleID }},
					{Header: "NAME", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.EscalateRuleItem).RuleName }},
					{Header: "STATUS", Field: func(v any) string { return v.(flashduty.EscalateRuleItem).Status }},
					{Header: "PRIORITY", Field: func(v any) string { return strconv.FormatInt(v.(flashduty.EscalateRuleItem).Priority, 10) }},
					{Header: "UPDATED", Field: func(v any) string { return output.FormatTime(v.(flashduty.EscalateRuleItem).UpdatedAt) }},
				}
				return ctx.PrintTotal(out.Items, cols, len(out.Items))
			})
		},
	}
	cmd.Flags().Int64Var(&fChannelID, "channel-id", 0, "Channel to list rules for. (required)")
	cmd.Flags().StringVar(&dataJSON, "data", "", "Full request body as JSON; positional arguments and typed flags override its fields. Accepts inline JSON, or - to read stdin.")
	cmd.Flags().StringVar(&fields, "fields", "", "Comma-separated fields to project in json/toon output (e.g. rule_id,rule_name,status,priority); ignored in table mode. Use to avoid dumping the full nested record.")
	return cmd
}
