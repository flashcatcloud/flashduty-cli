package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	versionStr = "dev"
	commitStr  = "none"
	dateStr    = "unknown"
)

// SetVersionInfo sets build-time version info from ldflags.
func SetVersionInfo(version, commit, date string) {
	versionStr = version
	commitStr = commit
	dateStr = date
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			out := cmd.OutOrStdout()
			// A structured (--json / --output-format) request emits a
			// machine-readable object that includes broker_egress, the capability
			// the runner probes before advertising broker mode to safari. An older
			// fduty ignores these flags and prints the plain line below, so the
			// absence of the broker_egress field reads as "not capable". The plain
			// human output is unchanged.
			if currentOutputFormat().Structured() {
				b, err := marshalStructured(map[string]any{
					"version":       versionStr,
					"commit":        commitStr,
					"date":          dateStr,
					"broker_egress": brokerEgressCapable,
				})
				if err != nil {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), err)
					return
				}
				_, _ = fmt.Fprintln(out, string(b))
				return
			}
			_, _ = fmt.Fprintf(out, "flashduty version %s (%s) built %s\n", versionStr, commitStr, dateStr)
		},
	}
}
