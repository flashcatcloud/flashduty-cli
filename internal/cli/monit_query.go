package cli

import (
	"fmt"
	"strconv"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

func newMonitQueryCmd() *cobra.Command {
	cmd := newGroupCmd("monit-query", "Query configured datasources; structured diagnostics use monit datasource-tools-invoke")
	cmd.AddCommand(newMonitQueryDataCmd())
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

// normalizeRawTimeArgs rewrites the raw-mode time-window args of a
// monit-query data call (<ds-type>.start / <ds-type>.end) into the unix-
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
