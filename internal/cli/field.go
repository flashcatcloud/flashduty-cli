package cli

import (
	"strings"

	gflashduty "github.com/flashcatcloud/go-flashduty"
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
			return runGFCommand(cmd, args, func(ctx *RunContext) error {
				result, _, err := ctx.GFClient.AlertEnrichment.FieldReadList(cmdContext(ctx.Cmd), &gflashduty.FieldListRequest{})
				if err != nil {
					return err
				}

				// go-flashduty's /field/list has no exact field_name filter (its
				// Query field is a regex over field_name/display_name). Preserve
				// the legacy SDK's exact-name --name filter client-side so behavior
				// is unchanged.
				items := result.Items
				if name != "" {
					filtered := make([]gflashduty.FieldItem, 0, len(items))
					for _, f := range items {
						if f.FieldName == name {
							filtered = append(filtered, f)
						}
					}
					items = filtered
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return v.(gflashduty.FieldItem).FieldID }},
					{Header: "NAME", Field: func(v any) string { return v.(gflashduty.FieldItem).FieldName }},
					{Header: "DISPLAY_NAME", Field: func(v any) string { return v.(gflashduty.FieldItem).DisplayName }},
					{Header: "TYPE", Field: func(v any) string { return v.(gflashduty.FieldItem).FieldType }},
					{Header: "OPTIONS", MaxWidth: 50, Field: func(v any) string {
						return strings.Join(v.(gflashduty.FieldItem).Options, ", ")
					}},
				}

				return ctx.PrintTotal(items, cols, len(items))
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Filter by field name")

	return cmd
}
