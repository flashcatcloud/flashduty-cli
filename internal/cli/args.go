package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// requireArgs returns a positional argument validator that produces descriptive
// error messages naming the missing arguments, e.g.:
//
//	"missing incident_id. Usage: flashduty incident update <id>"
func requireArgs(argNames ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < len(argNames) {
			missing := argNames[len(args):]
			return fmt.Errorf("missing %s. Usage: %s", strings.Join(missing, ", "), cmd.UseLine())
		}
		return nil
	}
}
