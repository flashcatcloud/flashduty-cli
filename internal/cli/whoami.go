package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the current authenticated identity",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newGFClient()
			if err != nil {
				return err
			}

			id, err := resolveIdentity(cmdContext(cmd), client)
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			if currentOutputFormat().Structured() {
				out, err := marshalStructured(id)
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(w, string(out))
				return nil
			}
			printIdentity(w, id)
			return nil
		},
	}
}
