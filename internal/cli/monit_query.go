package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"
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
		accountID  int64
	)

	cmd := &cobra.Command{
		Use:   "monit-query <datasource-id>",
		Short: "Invoke a datasource query or diagnostic tool",
		Long: curatedLong(`Invoke one query or diagnostic tool against a configured datasource.

Query tools are '<type>.query' where '<type>' is one of 'prometheus', 'mysql', 'postgres', 'oracle', 'clickhouse', 'elasticsearch', 'loki', 'victorialogs', 'sls', 'tencent_cls'; their params are the per-datasource query schemas (expr + execution, plus log/sls/tencent_cls extensions). Diagnostic tools are defined by the executing Edge (e.g. 'mysql.overview'). The tool prefix must match the datasource type; the server validates the tool name and params.`, "DataSources", "ToolsInvoke"),
		Example: `  flashduty monit-query 12345 --tool prometheus.query --params '{"expr":"up","execution":{"kind":"instant","to_ms":1757462400000}}'
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
			return runCommand(cmd, args, func(ctx *RunContext) error {
				req := &flashduty.DatasourceToolInvokeRequest{
					DatasourceID: uint64(datasourceID),
					Tool:         tool,
					Params:       params,
				}
				if cmd.Flags().Changed("account-id") {
					req.AccountID = uint64(accountID)
				}
				out, _, err := ctx.Client.DataSources.ToolsInvoke(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out)
			})
		},
	}

	cmd.Flags().StringVar(&tool, "tool", "", "Single tool name prefixed by the datasource type: a query tool '<type>.query' or an Edge-defined diagnostic tool (e.g. 'mysql.overview'). (required)")
	_ = cmd.MarkFlagRequired("tool")
	cmd.Flags().StringVar(&paramsFlag, "params", "", "Tool-specific JSON parameters as inline JSON, or - to read stdin. Omitted means the params field is not sent (the server treats it as {}); explicit null is invalid. Numbers are sent byte-exact, so epoch-millisecond values above 2^53 keep their digits.")
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
	var obj map[string]any
	if err := decoder.Decode(&obj); err != nil {
		return nil, fmt.Errorf("invalid --params JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("invalid --params JSON: expected one JSON object")
	}
	if obj == nil {
		return nil, fmt.Errorf("invalid --params JSON: expected an object")
	}
	return json.RawMessage(raw), nil
}
