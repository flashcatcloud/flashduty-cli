package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the current authenticated identity",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			id, err := resolveIdentity(cmdContext(cmd), client)
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			if flagJSON {
				out, _ := json.MarshalIndent(id, "", "  ")
				_, _ = fmt.Fprintln(w, string(out))
				return nil
			}
			printIdentity(w, id)
			return nil
		},
	}
}
