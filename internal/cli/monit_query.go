package cli

import (
	"fmt"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

func newMonitQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monit-query",
		Short: "Probe monit-backed datasources (prometheus|victorialogs|loki|mysql)",
	}
	cmd.AddCommand(newMonitQueryDiagnoseCmd())
	cmd.AddCommand(newMonitQueryRowsCmd())
	return cmd
}

func newMonitQueryDiagnoseCmd() *cobra.Command {
	var (
		dsType, dsName, timeStart, timeEnd, inputQuery, operation string
		maxLogs, maxPatterns, timeoutSeconds                      int
	)

	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "Pre-clustered RCA findings (log_patterns or metric_trends)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dsType == "" || dsName == "" || inputQuery == "" {
				return fmt.Errorf("--ds-type, --ds-name, --input-query are required")
			}
			startTime, err := timeutil.Parse(timeStart)
			if err != nil {
				return fmt.Errorf("invalid --time-start: %w", err)
			}
			endTime, err := timeutil.Parse(timeEnd)
			if err != nil {
				return fmt.Errorf("invalid --time-end: %w", err)
			}

			return runCommand(cmd, args, func(ctx *RunContext) error {
				input := &flashduty.MonitQueryDiagnoseInput{
					DsType:    dsType,
					DsName:    dsName,
					TimeStart: startTime,
					TimeEnd:   endTime,
					Operation: operation,
					Input:     flashduty.MonitQueryDiagnoseQuery{Query: inputQuery},
				}
				if maxLogs > 0 {
					input.MaxLogsScanned = maxLogs
				}
				if maxPatterns > 0 {
					input.MaxPatterns = maxPatterns
				}
				if timeoutSeconds > 0 {
					input.TimeoutSeconds = timeoutSeconds
				}

				result, err := ctx.Client.MonitQueryDiagnose(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}
				return ctx.Printer.Print(result, nil)
			})
		},
	}

	cmd.Flags().StringVar(&dsType, "ds-type", "", "Datasource type: prometheus|victorialogs|loki|mysql (required)")
	cmd.Flags().StringVar(&dsName, "ds-name", "", "Datasource name as configured (required)")
	cmd.Flags().StringVar(&timeStart, "time-start", "15m", "Window start (relative '15m'/'1h', unix seconds, or 'now')")
	cmd.Flags().StringVar(&timeEnd, "time-end", "now", "Window end (relative, unix seconds, or 'now'; span capped at 6h)")
	cmd.Flags().StringVar(&inputQuery, "input-query", "", "Filter-only log query OR matrix PromQL (required)")
	cmd.Flags().StringVar(&operation, "operation", "", "log_patterns or metric_trends (default inferred from ds-type)")
	cmd.Flags().IntVar(&maxLogs, "max-logs", 0, "Max log lines scanned (default 10000, cap 50000)")
	cmd.Flags().IntVar(&maxPatterns, "max-patterns", 0, "Max patterns returned (default 20, cap 50)")
	cmd.Flags().IntVar(&timeoutSeconds, "timeout-seconds", 0, "Per-call timeout in seconds (default 25, cap 30)")

	return cmd
}

func newMonitQueryRowsCmd() *cobra.Command {
	var (
		dsType, dsName, expr string
		argsKV               []string
	)

	cmd := &cobra.Command{
		Use:   "rows",
		Short: "Raw datasource passthrough (returns values/rows as the datasource itself would)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dsType == "" || dsName == "" || expr == "" {
				return fmt.Errorf("--ds-type, --ds-name, --expr are required")
			}
			argsMap, err := parseKVSlice(argsKV)
			if err != nil {
				return fmt.Errorf("invalid --args: %w", err)
			}

			return runCommand(cmd, args, func(ctx *RunContext) error {
				input := &flashduty.MonitQueryRowsInput{
					DsType: dsType,
					DsName: dsName,
					Expr:   expr,
					Args:   argsMap,
				}
				result, err := ctx.Client.MonitQueryRows(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}
				// MonitQueryRowsOutput intentionally captures the entire response
				// body as a RawMessage (data shape is datasource-specific). The
				// struct itself marshals to `{}`, so write the raw bytes through.
				if len(result.Data) == 0 {
					_, err = fmt.Fprintln(ctx.Writer, "{}")
				} else {
					_, err = fmt.Fprintln(ctx.Writer, string(result.Data))
				}
				return err
			})
		},
	}

	cmd.Flags().StringVar(&dsType, "ds-type", "", "Datasource type (required)")
	cmd.Flags().StringVar(&dsName, "ds-name", "", "Datasource name (required)")
	cmd.Flags().StringVar(&expr, "expr", "", "Query expression (required)")
	cmd.Flags().StringSliceVar(&argsKV, "args", nil, "Arg entries KEY=VALUE (repeatable; values must be strings per monit-query contract)")

	return cmd
}
