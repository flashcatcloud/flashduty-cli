package cli

import (
	"strings"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
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
			return runCommand(cmd, args, func(ctx *RunContext) error {
				result, err := ctx.Client.ListFields(cmdContext(ctx.Cmd), &flashduty.ListFieldsInput{
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

				return ctx.PrintTotal(result.Fields, cols, result.Total)
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Filter by field name")

	return cmd
}
