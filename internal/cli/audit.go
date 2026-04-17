package cli

import (
	"fmt"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
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
			return runCommand(cmd, args, func(ctx *RunContext) error {
				startTime, err := timeutil.Parse(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				endTime, err := timeutil.Parse(until)
				if err != nil {
					return fmt.Errorf("invalid --until: %w", err)
				}

				input := &flashduty.SearchAuditLogsInput{
					StartTime: startTime,
					EndTime:   endTime,
					Limit:     limit,
					PersonID:  person,
				}
				if operation != "" {
					input.Operations = parseStringSlice(operation)
				}

				var (
					result *flashduty.SearchAuditLogsOutput
					cursor string
				)
				for currentPage := 1; currentPage <= page; currentPage++ {
					input.SearchAfterCtx = cursor
					result, err = ctx.Client.SearchAuditLogs(cmdContext(ctx.Cmd), input)
					if err != nil {
						return err
					}
					if currentPage == page {
						break
					}
					if result.SearchAfterCtx == "" {
						result = &flashduty.SearchAuditLogsOutput{
							AuditLogs: []flashduty.AuditLogRecord{},
							Total:     result.Total,
						}
						break
					}
					cursor = result.SearchAfterCtx
				}

				cols := []output.Column{
					{Header: "TIME", Field: func(v any) string {
						return output.FormatTime(v.(flashduty.AuditLogRecord).CreatedAt)
					}},
					{Header: "PERSON", MaxWidth: 20, Field: func(v any) string {
						r := v.(flashduty.AuditLogRecord)
						if r.MemberName != "" {
							return r.MemberName
						}
						return fmt.Sprintf("%d", r.MemberID)
					}},
					{Header: "OPERATION", MaxWidth: 30, Field: func(v any) string {
						r := v.(flashduty.AuditLogRecord)
						if r.OperationName != "" {
							return r.OperationName
						}
						return r.Operation
					}},
					{Header: "DETAIL", MaxWidth: 50, Field: func(v any) string {
						r := v.(flashduty.AuditLogRecord)
						if r.Body != "" {
							return r.Body
						}
						return "-"
					}},
				}

				return ctx.PrintList(result.AuditLogs, cols, len(result.AuditLogs), page, int(result.Total))
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
