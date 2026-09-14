package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newMonitPrometheusLabelValuesCmd builds the curated leaf for
// GET /monit/prometheus/api/v1/label/{label_name}/values. The op carries a path
// parameter, so it is excluded from generation and served by a hand-written SDK
// method (flashduty.DataSourcesService.ReadPrometheusLabelValues); the command
// name is the mechanical path-derived name (group "monit", verb the remaining
// path segments hyphen-joined) so it stays reachable at its path-name like every
// other operation.
func newMonitPrometheusLabelValuesCmd() *cobra.Command {
	var dataSourceID int64

	cmd := &cobra.Command{
		Use:   "prometheus-api-v1-label-{label_name}-values <label_name>",
		Short: "List Prometheus label values",
		Long: `List the values of one label from a Prometheus-compatible data source, through the Monitors proxy.

The response is the data source's native Prometheus HTTP API payload, not the standard Flashduty envelope: on success it carries status "success" and the label values in data; a query the data source itself rejects still surfaces here as a command error, with the data source's own error text as the message.

API: GET /monit/prometheus/api/v1/label/{label_name}/values (monit-prometheus-read-label-values)`,
		Example: `  flashduty monit prometheus-api-v1-label-{label_name}-values job --data-source-id 12345`,
		Args:    requireExactArg("label_name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if dataSourceID < 1 {
					return fmt.Errorf("--data-source-id is required")
				}
				out, _, err := ctx.Client.DataSources.ReadPrometheusLabelValues(cmdContext(ctx.Cmd), uint64(dataSourceID), ctx.Args[0])
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out)
			})
		},
	}

	cmd.Flags().Int64Var(&dataSourceID, "data-source-id", 0, "Data source ID to query. Must reference a Prometheus-compatible data source owned by the authenticated account; obtainable via 'flashduty monit datasource-list'. (required)")
	_ = cmd.MarkFlagRequired("data-source-id")
	return cmd
}

func attachMonitPrometheusLabelValues(root *cobra.Command) {
	g := genGroup(root, "monit", "Monitors API")
	genAddLeaf(g, newMonitPrometheusLabelValuesCmd())
}
