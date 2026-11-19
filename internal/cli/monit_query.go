package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

// newMonitQueryCmd builds the curated leaf for the unified datasource tool
// entry POST /monit/datasource/tools/invoke: query tools ('<type>.query') and
// Edge-defined diagnostic tools share this one invocation surface. The
// generated `monit datasource-tools-invoke` is the spec-mirror equivalent.
// The CLI only assembles the request and renders the result — tool-name and
// per-datasource params validation are the server's job.
func newMonitQueryCmd() *cobra.Command {
	var (
		tool       string
		paramsFlag string
		since      string
		until      string
		accountID  int64
	)

	cmd := &cobra.Command{
		Use:   "monit-query <datasource-id>",
		Short: "Invoke a datasource query or diagnostic tool",
		Long: curatedLong(`Invoke one query or diagnostic tool against a configured datasource.

Query tools are '<type>.query' where '<type>' is one of 'prometheus', 'mysql', 'postgres', 'oracle', 'clickhouse', 'elasticsearch', 'loki', 'victorialogs', 'sls', 'tencent_cls'; their params are the per-datasource query schemas (expr + execution, plus log/sls/tencent_cls extensions). Diagnostic tools are defined by the executing Edge (e.g. 'mysql.overview'). The tool prefix must match the datasource type; the server validates the tool name and params.

--since/--until write the query window into params.execution (from_ms/to_ms) so it need not be computed by hand: --since alone ends the window now, --until alone sets an instant evaluation time. A flag whose field the params already carry is rejected.

By default a query result (data.format 'explore_result.v1') prints as a bounded plain-text digest: per-series min/max with timestamps, average, first/last and a coarse shape for time series; sorted values for instant samples; rows for tables; constant and varying fields for logs. The first line states the resolved window, rows are limited by size only, every cut states its counts, and the last line gives the jq path to the data in the JSON form. Use --output-format json (or toon) for the complete result; those formats keep stdout to the result and note the window on stderr. Diagnostic tool results print as before.`, "DataSources", "ToolsInvoke"),
		Example: `  flashduty monit-query 12345 --tool prometheus.query --params '{"expr":"up","execution":{"kind":"instant"}}' --until 2026-01-02T15:04:05+08:00
  flashduty monit-query 12345 --tool prometheus.query --params '{"expr":"rate(http_requests_total[5m])","execution":{"kind":"range","max_data_points":240}}' --since 2h
  flashduty monit-query 12345 --tool prometheus.query --params '{"expr":"up","execution":{"kind":"range","max_data_points":240}}' --since 2026-01-02T14:00:00+08:00 --until 2026-01-02T15:00:00+08:00 --output-format json
  flashduty monit-query 12345 --tool redis_node.overview`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			datasourceID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil || datasourceID < 1 {
				return fmt.Errorf("invalid datasource-id %q: must be a positive integer", args[0])
			}
			params, err := resolveToolParams(paramsFlag)
			if err != nil {
				return err
			}
			if setSince, setUntil := cmd.Flags().Changed("since"), cmd.Flags().Changed("until"); setSince || setUntil {
				if params, err = applyToolWindow(params, since, until, setSince, setUntil); err != nil {
					return err
				}
			}
			return runCommand(cmd, args, func(ctx *RunContext) error {
				req := &flashduty.DatasourceToolInvokeRequest{
					DatasourceID: uint64(datasourceID),
					Tool:         tool,
					Params:       params,
				}
				if cmd.Flags().Changed("account-id") {
					req.AccountID = uint64(accountID)
				}
				if ctx.Format != output.FormatTable {
					noteToolWindow(cmd.ErrOrStderr(), params)
					out, _, err := ctx.Client.DataSources.ToolsInvoke(cmdContext(ctx.Cmd), req)
					if err != nil {
						return err
					}
					return printGenericResult(ctx, out, nil, listDefaults{})
				}
				// The default format prints the window once: as the first
				// line of a query digest on stdout, else as the usual stderr
				// note.
				var note bytes.Buffer
				noteToolWindow(&note, params)
				out, _, err := ctx.Client.DataSources.ToolsInvoke(cmdContext(ctx.Cmd), req)
				if err == nil {
					window := strings.TrimSuffix(strings.TrimPrefix(note.String(), "note: "), "\n")
					if digest, ok := exploreDigest(out, window); ok {
						_, err := io.WriteString(ctx.Writer, digest)
						return err
					}
				}
				_, _ = note.WriteTo(cmd.ErrOrStderr())
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out, nil, listDefaults{})
			})
		},
	}

	cmd.Flags().StringVar(&tool, "tool", "", "Single tool name prefixed by the datasource type: a query tool '<type>.query' or an Edge-defined diagnostic tool (e.g. 'mysql.overview'). (required)")
	_ = cmd.MarkFlagRequired("tool")
	cmd.Flags().StringVar(&paramsFlag, "params", "", "Tool-specific JSON parameters as inline JSON, or - to read stdin. Omitted means the params field is not sent (the server treats it as {}); explicit null is invalid. Numbers are sent byte-exact, so epoch-millisecond values above 2^53 keep their digits.")
	cmd.Flags().StringVar(&since, "since", "", "Window start written to params execution.from_ms: duration before now (30m, 2h, 7d), RFC3339, date, datetime, or unix seconds/ms; one-second precision. Without --until the window ends now.")
	cmd.Flags().StringVar(&until, "until", "", "Window end written to params execution.to_ms (same formats and one-second precision as --since). Alone it sets an instant evaluation time.")
	cmd.Flags().Int64Var(&accountID, "account-id", 0, "Optional consistency check; must equal the authenticated account.")
	return cmd
}

// resolveToolParams validates the --params value as a single JSON object and
// returns it byte-exact for the request's raw params field. An empty flag
// returns nil so the field is omitted (the server treats omitted as {};
// explicit null is invalid). Validation decodes with UseNumber only to reject
// malformed input — the wire payload is the original text, so large integers
// (epoch-millisecond execution windows) cannot be rounded through float64.
func resolveToolParams(flag string) (json.RawMessage, error) {
	raw := flag
	if flag == "-" {
		b, err := readStdin("--params")
		if err != nil {
			return nil, fmt.Errorf("failed to read --params from stdin: %w", err)
		}
		raw = string(b)
	}
	if raw == "" {
		return nil, nil
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("invalid --params JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("invalid --params JSON: expected one JSON object")
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, fmt.Errorf("--params must be a JSON object")
	}
	return json.RawMessage(raw), nil
}

// applyToolWindow writes --since/--until into params.execution as from_ms /
// to_ms (see applyRange). Re-encoding keeps integers exact (UseNumber) and
// leaves '<', '>' and '&' in query text unescaped; execution.kind is never
// invented.
func applyToolWindow(params json.RawMessage, since, until string, setSince, setUntil bool) (json.RawMessage, error) {
	obj := map[string]any{}
	if len(params) > 0 {
		dec := json.NewDecoder(bytes.NewReader(params))
		dec.UseNumber()
		if err := dec.Decode(&obj); err != nil {
			return nil, fmt.Errorf("--params must be a JSON object to apply --since/--until")
		}
	}
	if err := applyRange(obj, toolRange, since, until, setSince, setUntil); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return json.RawMessage(bytes.TrimRight(buf.Bytes(), "\n")), nil
}
