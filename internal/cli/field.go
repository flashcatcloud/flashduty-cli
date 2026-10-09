package cli

import (
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

func newFieldCmd() *cobra.Command {
	cmd := newGroupCmd("field", "Manage custom fields")
	cmd.AddCommand(newFieldListCmd())
	return cmd
}

func newFieldListCmd() *cobra.Command {
	var name, query, orderby string
	var creatorID int64
	var asc bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List custom fields",
		Long:  curatedLong("List custom fields, optionally filtered by exact field name.", "AlertEnrichment", "FieldReadList"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				req := &flashduty.FieldListRequest{Query: query, Orderby: orderby, Asc: asc}
				if cmd.Flags().Changed("creator-id") {
					req.CreatorID = &creatorID
				}
				result, _, err := ctx.Client.AlertEnrichment.FieldReadList(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}

				// go-flashduty's /field/list has no exact field_name filter (its
				// Query field is a regex over field_name/display_name). Preserve
				// the legacy SDK's exact-name --name filter client-side so behavior
				// is unchanged.
				items := result.Items
				if name != "" {
					filtered := make([]flashduty.FieldItem, 0, len(items))
					for _, f := range items {
						if f.FieldName == name {
							filtered = append(filtered, f)
						}
					}
					items = filtered
				}

				cols := []output.Column{
					{Header: "ID", Field: func(v any) string { return v.(flashduty.FieldItem).FieldID }},
					{Header: "NAME", Field: func(v any) string { return v.(flashduty.FieldItem).FieldName }},
					{Header: "DISPLAY_NAME", Field: func(v any) string { return v.(flashduty.FieldItem).DisplayName }},
					{Header: "TYPE", Field: func(v any) string { return v.(flashduty.FieldItem).FieldType }},
					{Header: "OPTIONS", MaxWidth: 50, Field: func(v any) string {
						return strings.Join(v.(flashduty.FieldItem).Options, ", ")
					}},
				}

				return ctx.PrintTotal(items, cols, len(items))
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Filter by field name")
	cmd.Flags().StringVar(&query, "query", "", "Case-insensitive regex over the field name, server-side")
	cmd.Flags().Int64Var(&creatorID, "creator-id", 0, "Filter by creator member ID")
	cmd.Flags().StringVar(&orderby, "orderby", "", "Sort key (server default: created_at)")
	cmd.Flags().BoolVar(&asc, "asc", false, "Sort in ascending order")

	return cmd
}
