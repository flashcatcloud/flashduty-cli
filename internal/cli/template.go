package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

// presetTemplateField returns the channel-specific source field from a
// go-flashduty TemplateItem. The field is selected by name out of the
// templateChannels map; TemplateItem exposes those same fields as named struct
// members, so this switch reproduces that selection with no behavior change. An
// unknown field name yields "".
func presetTemplateField(t *flashduty.TemplateItem, fieldName string) string {
	switch fieldName {
	case "dingtalk":
		return t.Dingtalk
	case "dingtalk_app":
		return t.DingtalkApp
	case "feishu":
		return t.Feishu
	case "feishu_app":
		return t.FeishuApp
	case "wecom":
		return t.Wecom
	case "wecom_app":
		return t.WecomApp
	case "slack":
		return t.Slack
	case "slack_app":
		return t.SlackApp
	case "telegram":
		return t.Telegram
	case "teams_app":
		return t.TeamsApp
	case "email":
		return t.Email
	case "sms":
		return t.SMS
	case "zoom":
		return t.Zoom
	default:
		return ""
	}
}

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
			return runCommand(cmd, args, func(ctx *RunContext) error {
				fieldName, ok := templateChannels[channel]
				if !ok {
					return fmt.Errorf("unknown channel: %s", channel)
				}

				item, _, err := ctx.Client.NotificationTemplates.ReadInfo(cmdContext(ctx.Cmd), &flashduty.TemplateIDRequest{
					TemplateID: presetTemplateID,
				})
				if err != nil {
					return err
				}

				templateCode := ""
				if item != nil {
					templateCode = presetTemplateField(item, fieldName)
				}
				if templateCode == "" {
					return fmt.Errorf("no preset template found for channel: %s", channel)
				}

				result := &presetTemplateResult{
					Channel:      channel,
					FieldName:    fieldName,
					TemplateCode: templateCode,
				}

				if ctx.Structured() {
					return ctx.Printer.Print(result, nil)
				}
				_, _ = fmt.Fprintln(ctx.Writer, result.TemplateCode)
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&channel, "channel", "", "Notification channel (required). Values: "+strings.Join(channelEnumValues(), ", "))
	_ = cmd.MarkFlagRequired("channel")

	return cmd
}

func newTemplateValidateCmd() *cobra.Command {
	var channel, file, incidentID string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate and preview a template",
		RunE: func(cmd *cobra.Command, args []string) error {
			templateCode, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("failed to read template file: %w", err)
			}

			return runCommand(cmd, args, func(ctx *RunContext) error {
				fieldName, ok := templateChannels[channel]
				if !ok {
					return fmt.Errorf("unknown channel: %s", channel)
				}

				preview, _, err := ctx.Client.NotificationTemplates.ReadPreview(cmdContext(ctx.Cmd), &flashduty.PreviewTemplateRequest{
					Content:    string(templateCode),
					Type:       channel,
					IncidentID: incidentID,
				})
				if err != nil {
					return err
				}

				// Reproduce the legacy SDK's client-side derivation of the
				// validation result: the /template/preview endpoint only returns
				// {success, content, message}; the size limit, errors, and
				// warnings were all computed here from channelSizeLimits.
				success := false
				renderedPreview := ""
				errorMessage := ""
				if preview != nil {
					success = preview.Success
					renderedPreview = preview.Content
					errorMessage = preview.Message
				}

				renderedSize := len(renderedPreview)
				sizeLimit := channelSizeLimits[channel]

				errs := []string{}
				warnings := []string{}

				if !success {
					errs = append(errs, errorMessage)
				}

				if sizeLimit > 0 {
					if renderedSize > sizeLimit {
						sizeWarning := fmt.Sprintf("Rendered output is %d bytes, exceeding the %d byte limit for %s.", renderedSize, sizeLimit, channel)
						switch channel {
						case "telegram":
							sizeWarning += " CRITICAL: Telegram will silently drop this message."
						case "teams_app":
							sizeWarning += " Teams will return an error for this message."
						}
						errs = append(errs, sizeWarning)
					} else if renderedSize > int(float64(sizeLimit)*0.8) {
						warnings = append(warnings, fmt.Sprintf("Rendered output is %d/%d bytes (%.0f%% of limit).", renderedSize, sizeLimit, float64(renderedSize)/float64(sizeLimit)*100))
					}
				}

				result := &validateTemplateResult{
					Channel:         channel,
					FieldName:       fieldName,
					TemplateCode:    string(templateCode),
					Success:         success && len(errs) == 0,
					RenderedPreview: renderedPreview,
					RenderedSize:    renderedSize,
					SizeLimit:       sizeLimit,
					Errors:          errs,
					Warnings:        warnings,
				}

				if ctx.Structured() {
					return ctx.Printer.Print(result, nil)
				}

				if result.Success {
					_, _ = fmt.Fprintln(ctx.Writer, "Status: VALID")
				} else {
					_, _ = fmt.Fprintln(ctx.Writer, "Status: INVALID")
				}
				for _, e := range result.Errors {
					_, _ = fmt.Fprintf(ctx.Writer, "Error: %s\n", e)
				}
				for _, w := range result.Warnings {
					_, _ = fmt.Fprintf(ctx.Writer, "Warning: %s\n", w)
				}
				_, _ = fmt.Fprintf(ctx.Writer, "Size: %d / %d bytes\n", result.RenderedSize, result.SizeLimit)
				if result.RenderedPreview != "" {
					_, _ = fmt.Fprintln(ctx.Writer, "\n--- Preview ---")
					_, _ = fmt.Fprintln(ctx.Writer, result.RenderedPreview)
				}
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&channel, "channel", "", "Notification channel (required). Values: "+strings.Join(channelEnumValues(), ", "))
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
			vars := templateVariables()

			if category != "" {
				filtered := make([]templateVariable, 0)
				for _, v := range vars {
					if v.Category == category {
						filtered = append(filtered, v)
					}
				}
				vars = filtered
			}

			cols := []output.Column{
				{Header: "NAME", Field: func(v any) string { return v.(templateVariable).Name }},
				{Header: "TYPE", Field: func(v any) string { return v.(templateVariable).Type }},
				{Header: "DESCRIPTION", MaxWidth: 60, Field: func(v any) string { return v.(templateVariable).Description }},
				{Header: "EXAMPLE", MaxWidth: 40, Field: func(v any) string { return v.(templateVariable).Example }},
			}

			return newPrinter(cmd.OutOrStdout()).Print(vars, cols)
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
			var funcs []templateFunction

			switch funcType {
			case "custom":
				funcs = templateCustomFunctions()
			case "sprig":
				funcs = templateSprigFunctions()
			default:
				funcs = append(templateCustomFunctions(), templateSprigFunctions()...)
			}

			cols := []output.Column{
				{Header: "NAME", Field: func(v any) string { return v.(templateFunction).Name }},
				{Header: "SYNTAX", MaxWidth: 50, Field: func(v any) string { return v.(templateFunction).Syntax }},
				{Header: "DESCRIPTION", MaxWidth: 60, Field: func(v any) string { return v.(templateFunction).Description }},
			}

			return newPrinter(cmd.OutOrStdout()).Print(funcs, cols)
		},
	}

	cmd.Flags().StringVar(&funcType, "type", "all", "Filter: custom, sprig, or all")
	registerEnumFlag(cmd, "type", "custom", "sprig", "all")

	return cmd
}
