package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/skilldoc"
)

// RootForDump returns the fully-populated root command so dev tooling (the
// internal/cmd/skilldoc generator/validator) can build the command dump
// in-process, without shelling out to `flashduty __dump-commands`.
func RootForDump() *cobra.Command { return rootCmd }

// newDumpCommandsCmd builds the hidden `__dump-commands` command. It serializes
// the live cobra tree to indented JSON — the oracle the card generator and
// validator consume. Hidden because it is internal tooling, not a user verb.
func newDumpCommandsCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "__dump-commands",
		Short:  "Dump the command tree as JSON (internal tooling)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := json.MarshalIndent(skilldoc.Build(rootCmd), "", "  ")
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(append(out, '\n'))
			return err
		},
	}
}
