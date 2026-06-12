package cli

import (
	"fmt"
	"strconv"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
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
	var limit int
	var roleID int64
	var orderBy string
	var asc bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List members",
		Long:  curatedLong("List members in your account.", "Members", "MemberList"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				// go-flashduty's MemberListRequest exposes a single search
				// keyword (Query); the legacy SDK split name/email into separate
				// filters. Both --name and --email are keyword searches against
				// the same backend, so fold them into Query (name takes precedence).
				query := name
				if query == "" {
					query = email
				}
				req := &flashduty.MemberListRequest{
					Query:   query,
					Orderby: orderBy,
					Asc:     asc,
				}
				req.Page = page
				req.Limit = limit
				if roleID != 0 {
					req.RoleID = uint64(roleID)
				}

				result, _, err := ctx.Client.Members.MemberList(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}

				// MemberList returns member rows; an empty list renders the
				// "no members" path (structured: empty set; plain: a message).
				if len(result.Items) > 0 {
					cols := []output.Column{
						{Header: "ID", Field: func(v any) string { return strconv.FormatUint(v.(flashduty.MemberItem).MemberID, 10) }},
						{Header: "NAME", Field: func(v any) string { return v.(flashduty.MemberItem).MemberName }},
						{Header: "EMAIL", Field: func(v any) string { return v.(flashduty.MemberItem).Email }},
						{Header: "STATUS", Field: func(v any) string { return v.(flashduty.MemberItem).Status }},
						{Header: "TIMEZONE", Field: func(v any) string { return v.(flashduty.MemberItem).TimeZone }},
					}
					if err := ctx.Printer.Print(result.Items, cols); err != nil {
						return err
					}
				} else {
					if ctx.Structured() {
						return ctx.Printer.Print([]struct{}{}, nil)
					}
					_, _ = fmt.Fprintln(ctx.Writer, "No members found.")
					return nil
				}

				if !ctx.Structured() {
					_, _ = fmt.Fprintf(ctx.Writer, "Total: %d\n", result.Total)
				}
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Search by name")
	cmd.Flags().StringVar(&email, "email", "", "Search by email")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size, max 100 (default 20)")
	cmd.Flags().StringVar(&orderBy, "orderby", "", "Sort field: created_at, updated_at, member_name")
	cmd.Flags().BoolVar(&asc, "asc", false, "Sort in ascending order")
	cmd.Flags().Int64Var(&roleID, "role-id", 0, "Filter to members holding this role ID")

	return cmd
}
