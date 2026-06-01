package cli

import (
	"encoding/json"
	"fmt"

	"github.com/flashcatcloud/go-flashduty"
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
		Long:  curatedLong("Run pre-clustered RCA over a datasource window, returning log_patterns or metric_trends findings.", "Diagnostics", "QueryDiagnose"),
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
				input := &flashduty.DiagnoseRequest{
					DsType:    dsType,
					DsName:    dsName,
					Operation: operation,
					Input:     flashduty.DiagnoseRequestInput{Query: inputQuery},
					TimeRange: flashduty.DiagnoseRequestTimeRange{Start: startTime, End: endTime},
				}
				if maxLogs > 0 {
					input.Options.MaxLogsScanned = int64(maxLogs)
				}
				if maxPatterns > 0 {
					input.Options.MaxPatterns = int64(maxPatterns)
				}
				if timeoutSeconds > 0 {
					input.Options.TimeoutSeconds = int64(timeoutSeconds)
				}

				result, _, err := ctx.Client.Diagnostics.QueryDiagnose(cmdContext(ctx.Cmd), input)
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
		Long:  curatedLong("Raw datasource passthrough returning values/rows as the datasource itself would.", "Diagnostics", "QueryRows"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dsType == "" || dsName == "" || expr == "" {
				return fmt.Errorf("--ds-type, --ds-name, --expr are required")
			}
			argsMap, err := parseKVSlice(argsKV)
			if err != nil {
				return fmt.Errorf("invalid --args: %w", err)
			}

			return runCommand(cmd, args, func(ctx *RunContext) error {
				input := &flashduty.QueryRowsRequest{
					DsType: dsType,
					DsName: dsName,
					Expr:   expr,
					Args:   argsMap,
				}
				result, _, err := ctx.Client.Diagnostics.QueryRows(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}
				// This command is a raw datasource passthrough. The legacy SDK
				// captured the response body (a JSON array of {fields,values}
				// objects) as a RawMessage and wrote it through verbatim,
				// independent of the --json/--toon output format. go-flashduty
				// decodes that same array into []QueryRow, so re-marshal it to
				// the equivalent JSON array and write it through unchanged to
				// preserve the legacy single-blob output shape.
				if result == nil {
					_, err = fmt.Fprintln(ctx.Writer, "{}")
					return err
				}
				body, err := json.Marshal(*result)
				if err != nil {
					return fmt.Errorf("failed to marshal query rows: %w", err)
				}
				_, err = fmt.Fprintln(ctx.Writer, string(body))
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
