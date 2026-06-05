package cli

import (
	"fmt"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"
)

func newMonitAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monit-agent",
		Short: "On-box diagnostics via flashmonit agents (host/mysql/redis/…)",
	}
	cmd.AddCommand(newMonitAgentCatalogCmd())
	cmd.AddCommand(newMonitAgentInvokeCmd())
	return cmd
}

func newMonitAgentCatalogCmd() *cobra.Command {
	var targetKind, targetLocator string

	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "List the diagnostic tools the agent exposes for a target",
		Long:  curatedLong("List the diagnostic tools a monit-agent exposes for a target.", "Diagnostics", "ToolsCatalog"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetLocator == "" {
				return fmt.Errorf("--target-locator is required")
			}
			return runCommand(cmd, args, func(ctx *RunContext) error {
				input := &flashduty.ToolCatalogRequest{
					TargetKind:    targetKind,
					TargetLocator: targetLocator,
				}
				result, _, err := ctx.Client.Diagnostics.ToolsCatalog(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}
				return ctx.Printer.Print(result, nil)
			})
		},
	}

	cmd.Flags().StringVar(&targetKind, "target-kind", "", "Target kind (host|mysql|redis|…); omit to let the agent infer")
	cmd.Flags().StringVar(&targetLocator, "target-locator", "", "Target locator: internal IP, hostname, or data-source name (required)")

	return cmd
}

func newMonitAgentInvokeCmd() *cobra.Command {
	var (
		targetKind, targetLocator string
		dataJSON                  string
	)

	cmd := &cobra.Command{
		Use:   "invoke",
		Short: "Run up to 8 monit-agent tools concurrently on a target",
		Long: curatedLong(`Run up to 8 monit-agent diagnostic tools concurrently on a target and return their output.

The tools to run are carried in the --data request body:
  --data '{"tools":[{"tool":"<name>","params":{<obj>}}, ... up to 8]}'
params is optional and defaults to {}. --data also accepts - to read stdin,
which avoids shell-quoting hell for params JSON that contains commas or quotes
(e.g. SQL). --target-locator (required) and --target-kind override any matching
keys in --data.

  # heredoc form for quoted/comma SQL:
  fduty monit-agent invoke --target-locator 'X' --data - <<'FDUTY'
  {"tools":[{"tool":"mysql.query","params":{"sql":"SELECT a, b FROM t WHERE s='RUNNING'","max_rows":50}}]}
  FDUTY`, "Diagnostics", "ToolsInvoke"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetLocator == "" {
				return fmt.Errorf("--target-locator is required")
			}

			// Assemble the body the standard way: --data (inline JSON or -
			// stdin) overlaid with the typed --target-* flags, mirroring
			// genAssembleBody's "typed flags override --data keys".
			body, err := genAssembleBody(dataJSON, func(body map[string]any) {
				body["target_locator"] = targetLocator
				if cmd.Flags().Changed("target-kind") {
					body["target_kind"] = targetKind
				}
			})
			if err != nil {
				return err
			}

			tools, err := parseInvokeTools(body["tools"])
			if err != nil {
				return err
			}
			if len(tools) == 0 {
				return fmt.Errorf(`--data must carry a non-empty "tools" array, e.g. --data '{"tools":[{"tool":"os.overview"}]}'`)
			}
			if len(tools) > 8 {
				return fmt.Errorf("at most 8 tools may be invoked at once (got %d)", len(tools))
			}

			return runCommand(cmd, args, func(ctx *RunContext) error {
				kind, _ := body["target_kind"].(string)
				input := &flashduty.ToolInvokeRequest{
					TargetKind:    kind,
					TargetLocator: targetLocator,
					Tools:         tools,
				}
				result, _, err := ctx.Client.Diagnostics.ToolsInvoke(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}
				return ctx.Printer.Print(result, nil)
			})
		},
	}

	cmd.Flags().StringVar(&targetKind, "target-kind", "", "Target kind (host|mysql|redis|…); omit to let the agent infer")
	cmd.Flags().StringVar(&targetLocator, "target-locator", "", "Target locator: internal IP, hostname, or data-source name (required)")
	cmd.Flags().StringVar(&dataJSON, "data", "", `Request body as JSON carrying the tools to run: {"tools":[{"tool":"<name>","params":{<obj>}}, ... max 8]}. Accepts inline JSON, or - to read stdin.`)

	return cmd
}

// parseInvokeTools converts the decoded "tools" value from the --data body into
// SDK tool items. Each entry must be an object with a non-empty "tool" string;
// "params" is optional and defaults to an empty object so no-arg tools serialize
// as `{}`.
func parseInvokeTools(raw any) ([]flashduty.ToolInvokeRequestToolsItem, error) {
	if raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf(`"tools" must be a JSON array of {"tool":...,"params":...} objects`)
	}
	out := make([]flashduty.ToolInvokeRequestToolsItem, 0, len(arr))
	for i, e := range arr {
		obj, ok := e.(map[string]any)
		if !ok {
			return nil, fmt.Errorf(`tools[%d] must be an object with a "tool" key`, i)
		}
		name, _ := obj["tool"].(string)
		if name == "" {
			return nil, fmt.Errorf(`tools[%d] is missing a non-empty "tool" name`, i)
		}
		params := map[string]any{}
		if p, ok := obj["params"]; ok && p != nil {
			m, ok := p.(map[string]any)
			if !ok {
				return nil, fmt.Errorf(`tools[%d].params must be a JSON object`, i)
			}
			params = m
		}
		out = append(out, flashduty.ToolInvokeRequestToolsItem{Tool: name, Params: params})
	}
	return out, nil
}
