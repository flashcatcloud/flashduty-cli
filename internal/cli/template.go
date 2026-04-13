package cli

import (
	"fmt"
	"os"
	"strings"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/flashcatcloud/flashduty-cli/internal/output"
	"github.com/spf13/cobra"
)

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage notification templates",
	}
	cmd.AddCommand(newTemplateGetPresetCmd())
	cmd.AddCommand(newTemplateValidateCmd())
	cmd.AddCommand(newTemplateVariablesCmd())
	cmd.AddCommand(newTemplateFunctionsCmd())
	return cmd
}

func newTemplateGetPresetCmd() *cobra.Command {
	var channel string

	cmd := &cobra.Command{
		Use:   "get-preset",
		Short: "Get the preset template for a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			result, err := client.GetPresetTemplate(cmdContext(cmd), &flashduty.GetPresetTemplateInput{
				Channel: channel,
			})
			if err != nil {
				return err
			}

			fmt.Println(result.TemplateCode)
			return nil
		},
	}

	cmd.Flags().StringVar(&channel, "channel", "", "Notification channel (required). Values: "+strings.Join(flashduty.ChannelEnumValues(), ", "))
	_ = cmd.MarkFlagRequired("channel")

	return cmd
}

func newTemplateValidateCmd() *cobra.Command {
	var channel, file, incidentID string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate and preview a template",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			templateCode, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("failed to read template file: %w", err)
			}

			result, err := client.ValidateTemplate(cmdContext(cmd), &flashduty.ValidateTemplateInput{
				Channel:      channel,
				TemplateCode: string(templateCode),
				IncidentID:   incidentID,
			})
			if err != nil {
				return err
			}

			if flagJSON {
				return newPrinter(nil).Print(result, nil)
			}

			if result.Success {
				fmt.Println("Status: VALID")
			} else {
				fmt.Println("Status: INVALID")
			}
			for _, e := range result.Errors {
				fmt.Printf("Error: %s\n", e)
			}
			for _, w := range result.Warnings {
				fmt.Printf("Warning: %s\n", w)
			}
			fmt.Printf("Size: %d / %d bytes\n", result.RenderedSize, result.SizeLimit)
			if result.RenderedPreview != "" {
				fmt.Println("\n--- Preview ---")
				fmt.Println(result.RenderedPreview)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&channel, "channel", "", "Notification channel (required)")
	cmd.Flags().StringVar(&file, "file", "", "Path to template file (required)")
	cmd.Flags().StringVar(&incidentID, "incident", "", "Real incident ID for preview (uses mock data if empty)")
	_ = cmd.MarkFlagRequired("channel")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func newTemplateVariablesCmd() *cobra.Command {
	var category string

	cmd := &cobra.Command{
		Use:   "variables",
		Short: "List available template variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			vars := flashduty.TemplateVariables()

			if category != "" {
				filtered := make([]flashduty.TemplateVariable, 0)
				for _, v := range vars {
					if v.Category == category {
						filtered = append(filtered, v)
					}
				}
				vars = filtered
			}

			cols := []output.Column{
				{Header: "NAME", Field: func(v any) string { return v.(flashduty.TemplateVariable).Name }},
				{Header: "TYPE", Field: func(v any) string { return v.(flashduty.TemplateVariable).Type }},
				{Header: "DESCRIPTION", MaxWidth: 60, Field: func(v any) string { return v.(flashduty.TemplateVariable).Description }},
				{Header: "EXAMPLE", MaxWidth: 40, Field: func(v any) string { return v.(flashduty.TemplateVariable).Example }},
			}

			return newPrinter(nil).Print(vars, cols)
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "Filter by category: core, time, people, alerts, labels, context, notification, post_incident")

	return cmd
}

func newTemplateFunctionsCmd() *cobra.Command {
	var funcType string

	cmd := &cobra.Command{
		Use:   "functions",
		Short: "List available template functions",
		RunE: func(cmd *cobra.Command, args []string) error {
			var funcs []flashduty.TemplateFunction

			switch funcType {
			case "custom":
				funcs = flashduty.TemplateCustomFunctions()
			case "sprig":
				funcs = flashduty.TemplateSprigFunctions()
			default:
				funcs = append(flashduty.TemplateCustomFunctions(), flashduty.TemplateSprigFunctions()...)
			}

			cols := []output.Column{
				{Header: "NAME", Field: func(v any) string { return v.(flashduty.TemplateFunction).Name }},
				{Header: "SYNTAX", MaxWidth: 50, Field: func(v any) string { return v.(flashduty.TemplateFunction).Syntax }},
				{Header: "DESCRIPTION", MaxWidth: 60, Field: func(v any) string { return v.(flashduty.TemplateFunction).Description }},
			}

			return newPrinter(nil).Print(funcs, cols)
		},
	}

	cmd.Flags().StringVar(&funcType, "type", "all", "Filter: custom, sprig, or all")

	return cmd
}
