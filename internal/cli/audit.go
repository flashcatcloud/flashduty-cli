package cli

import (
	"fmt"

	gflashduty "github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

func newAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Manage audit logs",
	}
	cmd.AddCommand(newAuditSearchCmd())
	return cmd
}

func newAuditSearchCmd() *cobra.Command {
	var since, until, operation string
	var person int64
	var limit, page int

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search audit logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGFCommand(cmd, args, func(ctx *RunContext) error {
				startTime, err := timeutil.Parse(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				endTime, err := timeutil.Parse(until)
				if err != nil {
					return fmt.Errorf("invalid --until: %w", err)
				}

				input := &gflashduty.AuditSearchRequest{
					StartTime: startTime,
					EndTime:   endTime,
					Limit:     int64(limit),
					PersonID:  uint64(person),
				}
				if operation != "" {
					input.Operations = parseStringSlice(operation)
				}

				var (
					result *gflashduty.AuditSearchResponse
					cursor string
				)
				for currentPage := 1; currentPage <= page; currentPage++ {
					input.SearchAfterCtx = cursor
					result, _, err = ctx.GFClient.AuditLogs.Search(cmdContext(ctx.Cmd), input)
					if err != nil {
						return err
					}
					if currentPage == page {
						break
					}
					if result.SearchAfterCtx == "" {
						result = &gflashduty.AuditSearchResponse{
							Docs:  []gflashduty.AuditLog{},
							Total: result.Total,
						}
						break
					}
					cursor = result.SearchAfterCtx
				}

				cols := []output.Column{
					{Header: "TIME", Field: func(v any) string {
						return output.FormatTime(v.(gflashduty.AuditLog).CreatedAt)
					}},
					{Header: "PERSON", MaxWidth: 20, Field: func(v any) string {
						r := v.(gflashduty.AuditLog)
						if r.MemberName != "" {
							return r.MemberName
						}
						return fmt.Sprintf("%d", r.MemberID)
					}},
					{Header: "OPERATION", MaxWidth: 30, Field: func(v any) string {
						r := v.(gflashduty.AuditLog)
						if r.OperationName != "" {
							return r.OperationName
						}
						return r.Operation
					}},
					{Header: "DETAIL", MaxWidth: 50, Field: func(v any) string {
						r := v.(gflashduty.AuditLog)
						if r.Body != "" {
							return r.Body
						}
						return "-"
					}},
				}

				return ctx.PrintList(result.Docs, cols, len(result.Docs), page, int(result.Total))
			})
		},
	}

	cmd.Flags().StringVar(&since, "since", "7d", "Start time")
	cmd.Flags().StringVar(&until, "until", "now", "End time")
	cmd.Flags().Int64Var(&person, "person", 0, "Filter by person ID")
	cmd.Flags().StringVar(&operation, "operation", "", "Filter by operation type")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")

	return cmd
}
