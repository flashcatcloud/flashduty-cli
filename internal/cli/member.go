package cli

import (
	"fmt"
	"strconv"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/spf13/cobra"
)

func newMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "Manage members",
	}
	cmd.AddCommand(newMemberListCmd())
	return cmd
}

func newMemberListCmd() *cobra.Command {
	var name, email string
	var page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List members",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.ListMembers(cmdContext(ctx.Cmd), &flashduty.ListMembersInput{
					Name:  name,
					Email: email,
					Page:  page,
				})
				if err != nil {
					return err
				}

				// SDK returns Members when listing, PersonInfos when querying by IDs
				if len(result.Members) > 0 {
					cols := []output.Column{
						{Header: "ID", Field: func(v any) string { return strconv.Itoa(v.(flashduty.MemberItem).MemberID) }},
						{Header: "NAME", Field: func(v any) string { return v.(flashduty.MemberItem).MemberName }},
						{Header: "EMAIL", Field: func(v any) string { return v.(flashduty.MemberItem).Email }},
						{Header: "STATUS", Field: func(v any) string { return v.(flashduty.MemberItem).Status }},
						{Header: "TIMEZONE", Field: func(v any) string { return v.(flashduty.MemberItem).TimeZone }},
					}
					if err := ctx.Printer.Print(result.Members, cols); err != nil {
						return err
					}
				} else if len(result.PersonInfos) > 0 {
					cols := []output.Column{
						{Header: "ID", Field: func(v any) string { return strconv.FormatInt(v.(flashduty.PersonInfo).PersonID, 10) }},
						{Header: "NAME", Field: func(v any) string { return v.(flashduty.PersonInfo).PersonName }},
						{Header: "EMAIL", Field: func(v any) string { return v.(flashduty.PersonInfo).Email }},
					}
					if err := ctx.Printer.Print(result.PersonInfos, cols); err != nil {
						return err
					}
				} else {
					if ctx.JSON {
						return ctx.Printer.Print([]struct{}{}, nil)
					}
					_, _ = fmt.Fprintln(ctx.Writer, "No members found.")
					return nil
				}

				if !ctx.JSON {
					_, _ = fmt.Fprintf(ctx.Writer, "Total: %d\n", result.Total)
				}
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Search by name")
	cmd.Flags().StringVar(&email, "email", "", "Search by email")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}
