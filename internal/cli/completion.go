package cli

import "github.com/spf13/cobra"

// severityEnum is the closed set of incident/alert severities, shared by every
// --severity flag.
var severityEnum = []string{"Critical", "Warning", "Info"}

// registerEnumFlag makes <flag>'s value tab-complete to a fixed set and
// suppresses the default filename completion. The error only fires on an
// unknown flag name (a programmer error), so it is ignored.
func registerEnumFlag(cmd *cobra.Command, flag string, values ...string) {
	_ = cmd.RegisterFlagCompletionFunc(flag, func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return values, cobra.ShellCompDirectiveNoFileComp
	})
}
