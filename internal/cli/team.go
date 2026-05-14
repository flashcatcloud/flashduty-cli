package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

func newTeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Manage teams",
		Long:  "Create, list, view, update, and delete teams in your FlashDuty account.",
	}
	cmd.AddCommand(newTeamListCmd())
	cmd.AddCommand(newTeamGetCmd())
	cmd.AddCommand(newTeamCreateCmd())
	cmd.AddCommand(newTeamUpdateCmd())
	cmd.AddCommand(newTeamDeleteCmd())
	return cmd
}

func newTeamListCmd() *cobra.Command {
	var (
		name     string
		page     int
		limit    int
		orderBy  string
		asc      bool
		personID int64
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List teams",
		Long: `List teams in your account.

Use --name to search by team name substring.
Use --person-id to filter teams containing a specific member.
Results are paginated; use --page and --limit to navigate.

Examples:
  flashduty team list
  flashduty team list --name "SRE"
  flashduty team list --person-id 12345 --limit 50
  flashduty team list --orderby team_name --asc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.ListTeams(cmdContext(ctx.Cmd), &flashduty.ListTeamsInput{
					Name:     name,
					Page:     page,
					Limit:    limit,
					OrderBy:  orderBy,
					Asc:      asc,
					PersonID: personID,
				})
				if err != nil {
					return err
				}

				cols := teamListColumns()
				return ctx.PrintTotal(result.Teams, cols, result.Total)
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Search by team name substring")
	cmd.Flags().IntVar(&page, "page", 1, "Page number (default 1)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size, max 100 (default 20)")
	cmd.Flags().StringVar(&orderBy, "orderby", "", "Sort field: created_at, updated_at, team_name")
	cmd.Flags().BoolVar(&asc, "asc", false, "Sort in ascending order")
	cmd.Flags().Int64Var(&personID, "person-id", 0, "Filter teams by member ID")

	return cmd
}

func newTeamGetCmd() *cobra.Command {
	var (
		teamID   int64
		teamName string
		refID    string
	)

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get team detail",
		Long: `Get detailed information about a specific team.

Specify the team by exactly one of: --id, --name, or --ref-id.
The output includes team metadata, member list, and audit information.

Examples:
  flashduty team get --id 123
  flashduty team get --name "SRE Team"
  flashduty team get --ref-id "hr-dept-42"
  flashduty team get --id 123 --json`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return requireExactlyOneFlag(cmd, "id", "name", "ref-id")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				team, err := ctx.Client.GetTeamInfo(cmdContext(ctx.Cmd), &flashduty.TeamGetInput{
					TeamID:   teamID,
					TeamName: teamName,
					RefID:    refID,
				})
				if err != nil {
					return err
				}

				if ctx.JSON {
					return ctx.Printer.Print(team, nil)
				}

				printTeamDetail(ctx.Writer, team)
				return nil
			})
		},
	}

	cmd.Flags().Int64Var(&teamID, "id", 0, "Team ID")
	cmd.Flags().StringVar(&teamName, "name", "", "Team name")
	cmd.Flags().StringVar(&refID, "ref-id", "", "External reference ID")

	return cmd
}

func newTeamCreateCmd() *cobra.Command {
	var (
		name        string
		description string
		personIDs   string
		emails      string
		refID       string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new team",
		Long: `Create a new team.

The --name flag is required and must be unique within the account (1-39 characters).
Use --person-ids to add existing members by their person IDs (comma-separated).
Use --emails to invite members by email address (comma-separated).

Examples:
  flashduty team create --name "SRE Team"
  flashduty team create --name "SRE Team" --description "Site Reliability" --person-ids 1,2,3
  flashduty team create --name "SRE Team" --emails alice@example.com,bob@example.com
  flashduty team create --name "SRE Team" --ref-id "hr-dept-42" --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				ids, err := parseIntSlice(personIDs)
				if err != nil {
					return fmt.Errorf("invalid --person-ids: %w", err)
				}

				result, err := ctx.Client.UpsertTeam(cmdContext(ctx.Cmd), &flashduty.TeamUpsertInput{
					TeamName:    name,
					Description: description,
					PersonIDs:   ids,
					Emails:      parseStringSlice(emails),
					RefID:       refID,
				})
				if err != nil {
					return err
				}

				return ctx.WriteResultJSON(result,
					fmt.Sprintf("Team created: %s (ID: %d)", result.TeamName, result.TeamID))
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Team name (required, 1-39 characters)")
	cmd.Flags().StringVar(&description, "description", "", "Team description (max 500 characters)")
	cmd.Flags().StringVar(&personIDs, "person-ids", "", "Comma-separated member person IDs")
	cmd.Flags().StringVar(&emails, "emails", "", "Comma-separated email addresses to invite")
	cmd.Flags().StringVar(&refID, "ref-id", "", "External reference ID for HR system integration")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func newTeamUpdateCmd() *cobra.Command {
	var (
		teamID      int64
		name        string
		description string
		personIDs   string
		emails      string
		refID       string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an existing team",
		Long: `Update an existing team.

The --id flag is required to identify which team to update.
WARNING: --person-ids REPLACES the entire member list. To keep existing members,
include all current member IDs along with the new ones. Use "team get --id <id>"
to see the current member list before updating.

Examples:
  flashduty team update --id 123 --name "New Name"
  flashduty team update --id 123 --description "Updated description"
  flashduty team update --id 123 --person-ids 1,2,3,4,5
  flashduty team update --id 123 --name "Renamed" --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if teamID == 0 {
				return fmt.Errorf("--id is required")
			}

			return runCommand(cmd, args, func(ctx *RunContext) error {
				ids, err := parseIntSlice(personIDs)
				if err != nil {
					return fmt.Errorf("invalid --person-ids: %w", err)
				}

				// The API requires team_name on every upsert. If the user didn't
				// provide --name, fetch the current name so we don't clear it.
				teamName := name
				if !cmd.Flags().Changed("name") {
					existing, err := ctx.Client.GetTeamInfo(cmdContext(ctx.Cmd), &flashduty.TeamGetInput{
						TeamID: teamID,
					})
					if err != nil {
						return fmt.Errorf("failed to fetch current team: %w", err)
					}
					teamName = existing.TeamName
				}

				input := &flashduty.TeamUpsertInput{
					TeamID:   teamID,
					TeamName: teamName,
				}
				if cmd.Flags().Changed("description") {
					input.Description = description
				}
				if cmd.Flags().Changed("person-ids") {
					input.PersonIDs = ids
				}
				if cmd.Flags().Changed("emails") {
					input.Emails = parseStringSlice(emails)
				}
				if cmd.Flags().Changed("ref-id") {
					input.RefID = refID
				}

				result, err := ctx.Client.UpsertTeam(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}

				return ctx.WriteResultJSON(result,
					fmt.Sprintf("Team updated: %s (ID: %d)", result.TeamName, result.TeamID))
			})
		},
	}

	cmd.Flags().Int64Var(&teamID, "id", 0, "Team ID (required)")
	cmd.Flags().StringVar(&name, "name", "", "New team name (1-39 characters)")
	cmd.Flags().StringVar(&description, "description", "", "New description (max 500 characters)")
	cmd.Flags().StringVar(&personIDs, "person-ids", "", "Comma-separated member person IDs (replaces entire member list)")
	cmd.Flags().StringVar(&emails, "emails", "", "Comma-separated email addresses to invite")
	cmd.Flags().StringVar(&refID, "ref-id", "", "External reference ID")
	_ = cmd.MarkFlagRequired("id")

	return cmd
}

func newTeamDeleteCmd() *cobra.Command {
	var (
		teamID   int64
		teamName string
		refID    string
		force    bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a team",
		Long: `Permanently delete a team.

Specify the team by exactly one of: --id, --name, or --ref-id.
This action is irreversible. You will be prompted for confirmation
unless --force is set or output is in JSON mode.

Examples:
  flashduty team delete --id 123
  flashduty team delete --name "Old Team" --force
  flashduty team delete --ref-id "hr-dept-99" --json`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return requireExactlyOneFlag(cmd, "id", "name", "ref-id")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				identifier := identifierDescription(teamID, teamName, refID)
				if !confirmAction(ctx.Cmd, fmt.Sprintf("Are you sure you want to delete team %s?", identifier)) {
					_, _ = fmt.Fprintln(ctx.Writer, "Aborted.")
					return nil
				}

				err := ctx.Client.DeleteTeam(cmdContext(ctx.Cmd), &flashduty.TeamDeleteInput{
					TeamID:   teamID,
					TeamName: teamName,
					RefID:    refID,
				})
				if err != nil {
					return err
				}

				ctx.WriteResult("Team deleted successfully.")
				return nil
			})
		},
	}

	cmd.Flags().Int64Var(&teamID, "id", 0, "Team ID")
	cmd.Flags().StringVar(&teamName, "name", "", "Team name")
	cmd.Flags().StringVar(&refID, "ref-id", "", "External reference ID")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func teamListColumns() []output.Column {
	return []output.Column{
		{Header: "ID", Field: func(v any) string { return strconv.FormatInt(v.(flashduty.TeamInfo).TeamID, 10) }},
		{Header: "NAME", Field: func(v any) string { return v.(flashduty.TeamInfo).TeamName }},
		{Header: "MEMBERS", MaxWidth: 50, Field: func(v any) string {
			members := v.(flashduty.TeamInfo).Members
			names := make([]string, 0, len(members))
			for _, m := range members {
				names = append(names, m.PersonName)
			}
			return strings.Join(names, ", ")
		}},
	}
}

func printTeamDetail(w io.Writer, team *flashduty.TeamItem) {
	members := make([]string, 0, len(team.Members))
	for _, m := range team.Members {
		if m.Email != "" {
			members = append(members, fmt.Sprintf("%s <%s>", m.PersonName, m.Email))
		} else {
			members = append(members, m.PersonName)
		}
	}
	if len(members) == 0 {
		for _, id := range team.PersonIDs {
			members = append(members, strconv.FormatInt(id, 10))
		}
	}

	_, _ = fmt.Fprintf(w, "ID:            %d\n", team.TeamID)
	_, _ = fmt.Fprintf(w, "Name:          %s\n", team.TeamName)
	_, _ = fmt.Fprintf(w, "Description:   %s\n", orDash(team.Description))
	_, _ = fmt.Fprintf(w, "Status:        %s\n", orDash(team.Status))
	_, _ = fmt.Fprintf(w, "Ref ID:        %s\n", orDash(team.RefID))
	_, _ = fmt.Fprintf(w, "Members:       %s\n", orDash(strings.Join(members, ", ")))
	_, _ = fmt.Fprintf(w, "Created:       %s\n", output.FormatTime(team.CreatedAt))
	_, _ = fmt.Fprintf(w, "Updated:       %s\n", output.FormatTime(team.UpdatedAt))
	_, _ = fmt.Fprintf(w, "Created By:    %s\n", orDash(team.CreatorName))
	_, _ = fmt.Fprintf(w, "Updated By:    %s\n", orDash(team.UpdatedByName))
}

func identifierDescription(id int64, name, refID string) string {
	if id != 0 {
		return fmt.Sprintf("ID=%d", id)
	}
	if name != "" {
		return fmt.Sprintf("%q", name)
	}
	return fmt.Sprintf("ref-id=%q", refID)
}
