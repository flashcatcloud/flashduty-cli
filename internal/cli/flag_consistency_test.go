package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestPageImpliesLimit walks the full command tree and asserts that any
// command exposing a --page flag also exposes --limit. Agents transfer flag
// knowledge across command groups; a paginated list without --limit forces
// trial-and-error round-trips (seen in prod: `member list --limit` failing
// with "unknown flag").
func TestPageImpliesLimit(t *testing.T) {
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd.Flags().Lookup("page") != nil && cmd.Flags().Lookup("limit") == nil {
			t.Errorf("%s registers --page but not --limit", cmd.CommandPath())
		}
		for _, sub := range cmd.Commands() {
			walk(sub)
		}
	}
	walk(rootCmd)
}
