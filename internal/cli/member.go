package cli

import (
	"fmt"
	"strconv"

	gflashduty "github.com/flashcatcloud/go-flashduty"
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

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List members",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGFCommand(cmd, args, func(ctx *RunContext) error {
				// go-flashduty's MemberListRequest exposes a single search
				// keyword (Query); the legacy SDK split name/email into separate
				// filters. Both --name and --email are keyword searches against
				// the same backend, so fold them into Query (name takes precedence).
				query := name
				if query == "" {
					query = email
				}
				req := &gflashduty.MemberListRequest{
					Query: query,
				}
				req.Page = page

				result, _, err := ctx.GFClient.Members.MemberList(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}

				// MemberList returns member rows; an empty list renders the
				// "no members" path (structured: empty set; plain: a message).
				if len(result.Items) > 0 {
					cols := []output.Column{
						{Header: "ID", Field: func(v any) string { return strconv.FormatUint(v.(gflashduty.MemberItem).MemberID, 10) }},
						{Header: "NAME", Field: func(v any) string { return v.(gflashduty.MemberItem).MemberName }},
						{Header: "EMAIL", Field: func(v any) string { return v.(gflashduty.MemberItem).Email }},
						{Header: "STATUS", Field: func(v any) string { return v.(gflashduty.MemberItem).Status }},
						{Header: "TIMEZONE", Field: func(v any) string { return v.(gflashduty.MemberItem).TimeZone }},
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

	return cmd
}
