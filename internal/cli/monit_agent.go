package cli

import (
	"fmt"

	gflashduty "github.com/flashcatcloud/go-flashduty"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetLocator == "" {
				return fmt.Errorf("--target-locator is required")
			}
			return runGFCommand(cmd, args, func(ctx *RunContext) error {
				input := &gflashduty.ToolCatalogRequest{
					TargetKind:    targetKind,
					TargetLocator: targetLocator,
				}
				result, _, err := ctx.GFClient.Diagnostics.ToolsCatalog(cmdContext(ctx.Cmd), input)
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
		toolSpecs                 []string
	)

	cmd := &cobra.Command{
		Use:   "invoke",
		Short: "Run up to 8 monit-agent tools concurrently on a target",
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetLocator == "" {
				return fmt.Errorf("--target-locator is required")
			}
			if len(toolSpecs) == 0 {
				return fmt.Errorf("--tool-spec is required (repeatable; up to 8)")
			}
			if len(toolSpecs) > 8 {
				return fmt.Errorf("--tool-spec accepts up to 8 entries (got %d)", len(toolSpecs))
			}
			parsed, err := parseToolSpecs(toolSpecs)
			if err != nil {
				return fmt.Errorf("invalid --tool-spec: %w", err)
			}

			return runGFCommand(cmd, args, func(ctx *RunContext) error {
				input := &gflashduty.ToolInvokeRequest{
					TargetKind:    targetKind,
					TargetLocator: targetLocator,
					Tools:         parsed,
				}
				result, _, err := ctx.GFClient.Diagnostics.ToolsInvoke(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}
				return ctx.Printer.Print(result, nil)
			})
		},
	}

	cmd.Flags().StringVar(&targetKind, "target-kind", "", "Target kind (host|mysql|redis|…); omit to let the agent infer")
	cmd.Flags().StringVar(&targetLocator, "target-locator", "", "Target locator: internal IP, hostname, or data-source name (required)")
	// Use StringArray (not StringSlice) so commas inside params=<json> aren't
	// mis-parsed as CSV separators — each --tool-spec entry is taken verbatim.
	cmd.Flags().StringArrayVar(&toolSpecs, "tool-spec", nil, "Tool spec 'name=<tool>[,params=<json>]' (repeatable, max 8)")

	return cmd
}
