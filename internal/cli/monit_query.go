package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

func newMonitQueryCmd() *cobra.Command {
	cmd := newGroupCmd("monit-query", "Probe monit-backed datasources (9 types via data; diagnose/rows support prometheus|victorialogs|loki|mysql)")
	cmd.AddCommand(newMonitQueryDiagnoseCmd())
	cmd.AddCommand(newMonitQueryDataCmd())
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
	registerEnumFlag(cmd, "ds-type", "prometheus", "victorialogs", "loki", "mysql")
	cmd.Flags().StringVar(&timeStart, "time-start", "15m", "Window start: relative duration ('15m'/'1h'), 'now', a date/RFC3339 timestamp, or a unix epoch in seconds or milliseconds")
	cmd.Flags().StringVar(&timeEnd, "time-end", "now", "Window end: same formats as --time-start; span capped at 6h")
	cmd.Flags().StringVar(&inputQuery, "input-query", "", "Filter-only log query OR matrix PromQL (required)")
	cmd.Flags().StringVar(&operation, "operation", "", "log_patterns or metric_trends (default inferred from ds-type)")
	cmd.Flags().IntVar(&maxLogs, "max-logs", 0, "Max log lines scanned (default 10000, cap 50000)")
	cmd.Flags().IntVar(&maxPatterns, "max-patterns", 0, "Max patterns returned (default 20, cap 50)")
	cmd.Flags().IntVar(&timeoutSeconds, "timeout-seconds", 0, "Per-call timeout in seconds (default 25, cap 30)")

	return cmd
}

func newMonitQueryDataCmd() *cobra.Command {
	var (
		dsType, dsName, expr string
		delaySeconds         int64
		argsKV               []string
	)

	cmd := &cobra.Command{
		Use:   "data",
		Short: "Structured datasource query (returns a stable query_result.v1: frames/records/samples)",
		Long:  curatedLong("Structured datasource query returning the stable query_result.v1 result — frames, records, or samples — instead of the legacy flattened rows.", "Diagnostics", "QueryData"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dsType == "" || dsName == "" || expr == "" {
				return fmt.Errorf("--ds-type, --ds-name, --expr are required")
			}
			argsMap, err := parseKVSlice(argsKV)
			if err != nil {
				return fmt.Errorf("invalid --args: %w", err)
			}
			if err := normalizeRawTimeArgs(dsType, argsMap); err != nil {
				return err
			}

			return runCommand(cmd, args, func(ctx *RunContext) error {
				input := &flashduty.QueryDataRequest{
					DsType:       dsType,
					DsName:       dsName,
					Expr:         expr,
					DelaySeconds: delaySeconds,
					Args:         argsMap,
				}
				result, _, err := ctx.Client.Diagnostics.QueryData(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}
				return ctx.Printer.Print(result, nil)
			})
		},
	}

	cmd.Flags().StringVar(&dsType, "ds-type", "", "Datasource type (required)")
	cmd.Flags().StringVar(&dsName, "ds-name", "", "Datasource name as configured (required)")
	registerEnumFlag(cmd, "ds-type", "prometheus", "victorialogs", "loki", "mysql", "sls", "elasticsearch", "postgres", "oracle", "clickhouse")
	cmd.Flags().StringVar(&expr, "expr", "", "Query expression (required)")
	cmd.Flags().Int64Var(&delaySeconds, "delay-seconds", 0, "Look-back offset in seconds for point-in-time queries (default 0)")
	cmd.Flags().StringSliceVar(&argsKV, "args", nil, "Arg entries KEY=VALUE (repeatable; values must be strings per monit-query contract). "+
		"For loki/victorialogs raw mode, <ds-type>.start/<ds-type>.end accept a relative duration ('15m'), 'now', a date/RFC3339 timestamp, "+
		"or a unix epoch in seconds or milliseconds — normalized to the form the datasource requires before sending")

	return cmd
}

func newMonitQueryRowsCmd() *cobra.Command {
	var (
		dsType, dsName, expr string
		argsKV               []string
	)

	cmd := &cobra.Command{
		Use:        "rows",
		Short:      "Raw datasource passthrough (returns values/rows as the datasource itself would). Deprecated — prefer 'monit-query data'",
		Deprecated: "use 'monit-query data' instead",
		Long:       curatedLong("Deprecated. Raw datasource passthrough returning values/rows as the datasource itself would. Migrate to 'monit-query data', which preserves frames/records/samples without forcing results into legacy rows.", "Diagnostics", "QueryRows"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if dsType == "" || dsName == "" || expr == "" {
				return fmt.Errorf("--ds-type, --ds-name, --expr are required")
			}
			argsMap, err := parseKVSlice(argsKV)
			if err != nil {
				return fmt.Errorf("invalid --args: %w", err)
			}
			if err := normalizeRawTimeArgs(dsType, argsMap); err != nil {
				return err
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
	registerEnumFlag(cmd, "ds-type", "prometheus", "victorialogs", "loki", "mysql")
	cmd.Flags().StringVar(&expr, "expr", "", "Query expression (required)")
	cmd.Flags().StringSliceVar(&argsKV, "args", nil, "Arg entries KEY=VALUE (repeatable; values must be strings per monit-query contract). "+
		"For loki/victorialogs raw mode, <ds-type>.start/<ds-type>.end accept a relative duration ('15m'), 'now', a date/RFC3339 timestamp, "+
		"or a unix epoch in seconds or milliseconds — normalized to the form the datasource requires before sending")

	return cmd
}

// normalizeRawTimeArgs rewrites the raw-mode time-window args of a
// monit-query rows call (<ds-type>.start / <ds-type>.end) into the unix-
// seconds form the server requires, accepting any format timeutil.Parse
// understands (RFC3339, date/datetime, relative duration, unix seconds or
// milliseconds). Loki and VictoriaLogs are the only ds-types whose raw mode
// consumes these keys; other ds-types ignore args entirely, so nothing is
// touched for them.
func normalizeRawTimeArgs(dsType string, args map[string]string) error {
	if dsType != "loki" && dsType != "victorialogs" {
		return nil
	}
	for _, suffix := range []string{"start", "end"} {
		key := dsType + "." + suffix
		v, ok := args[key]
		if !ok || v == "" {
			continue
		}
		ts, err := timeutil.Parse(v)
		if err != nil {
			return fmt.Errorf("invalid --args %s=%s: %w", key, v, err)
		}
		args[key] = strconv.FormatInt(ts, 10)
	}
	return nil
}
