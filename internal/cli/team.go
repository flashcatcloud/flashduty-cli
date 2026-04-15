package cli

import (
	"strconv"
	"strings"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/spf13/cobra"
)

func newTeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Manage teams",
	}
	cmd.AddCommand(newTeamListCmd())
	return cmd
}

func newTeamListCmd() *cobra.Command {
	var name string
	var page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List teams",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.ListTeams(cmdContext(ctx.Cmd), &flashduty.ListTeamsInput{
					Name: name,
					Page: page,
				})
				if err != nil {
					return err
				}

				cols := []output.Column{
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

				return ctx.PrintTotal(result.Teams, cols, result.Total)
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Search by name")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}
