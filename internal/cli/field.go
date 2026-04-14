package cli

import (
	"fmt"
	"strings"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/spf13/cobra"
)

func newFieldCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "field",
		Short: "Manage custom fields",
	}
	cmd.AddCommand(newFieldListCmd())
	return cmd
}

func newFieldListCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List custom fields",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			result, err := client.ListFields(cmdContext(cmd), &flashduty.ListFieldsInput{
				FieldName: name,
			})
			if err != nil {
				return err
			}

			cols := []output.Column{
				{Header: "ID", Field: func(v any) string { return v.(flashduty.FieldInfo).FieldID }},
				{Header: "NAME", Field: func(v any) string { return v.(flashduty.FieldInfo).FieldName }},
				{Header: "DISPLAY_NAME", Field: func(v any) string { return v.(flashduty.FieldInfo).DisplayName }},
				{Header: "TYPE", Field: func(v any) string { return v.(flashduty.FieldInfo).FieldType }},
				{Header: "OPTIONS", MaxWidth: 50, Field: func(v any) string {
					return strings.Join(v.(flashduty.FieldInfo).Options, ", ")
				}},
			}

			p := newPrinter(cmd.OutOrStdout())
			if err := p.Print(result.Fields, cols); err != nil {
				return err
			}
			if !flagJSON {
				fmt.Fprintf(cmd.OutOrStdout(), "Total: %d\n", result.Total)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Filter by field name")

	return cmd
}
