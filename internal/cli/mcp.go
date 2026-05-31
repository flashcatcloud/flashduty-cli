package cli

import (
	"fmt"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP server registrations",
	}
	cmd.AddCommand(newMCPCreateCmd())
	return cmd
}

func newMCPCreateCmd() *cobra.Command {
	var (
		serverName     string
		description    string
		transport      string
		command        string
		argsFlag       []string
		envEntries     []string
		url            string
		headerEntries  []string
		connectTimeout int
		callTimeout    int
		teamID         int64
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Register an MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if strings.TrimSpace(serverName) == "" {
					return fmt.Errorf("--server-name is required")
				}
				envMap, err := parseKVSlice(envEntries)
				if err != nil {
					return fmt.Errorf("invalid --env: %w", err)
				}
				headerMap, err := parseKVSlice(headerEntries)
				if err != nil {
					return fmt.Errorf("invalid --headers: %w", err)
				}
				input := &flashduty.McpServerCreateRequest{
					ServerName:     serverName,
					Description:    description,
					Transport:      transport,
					Command:        command,
					Args:           argsFlag,
					Env:            envMap,
					URL:            url,
					Headers:        headerMap,
					ConnectTimeout: int64(connectTimeout),
					CallTimeout:    int64(callTimeout),
					TeamID:         teamID,
				}
				result, _, err := ctx.Client.McpServers.WriteServerCreate(cmdContext(ctx.Cmd), input)
				if err != nil {
					return err
				}
				return ctx.WriteResultJSON(result,
					fmt.Sprintf("MCP server registered: %s (status: %s)", result.ServerID, result.Status))
			})
		},
	}

	cmd.Flags().StringVar(&serverName, "server-name", "", "MCP server display name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Server description")
	cmd.Flags().StringVar(&transport, "transport", "streamable-http", "Transport: stdio|sse|streamable-http")
	cmd.Flags().StringVar(&command, "command", "", "Executable (stdio transport)")
	cmd.Flags().StringSliceVar(&argsFlag, "args", nil, "Executable args (stdio transport, repeatable)")
	cmd.Flags().StringSliceVar(&envEntries, "env", nil, "Env entries KEY=VALUE (repeatable)")
	cmd.Flags().StringVar(&url, "url", "", "URL (sse / streamable-http)")
	cmd.Flags().StringSliceVar(&headerEntries, "headers", nil, "Header entries KEY=VALUE (repeatable)")
	cmd.Flags().IntVar(&connectTimeout, "connect-timeout", 10, "Connection timeout in seconds")
	cmd.Flags().IntVar(&callTimeout, "call-timeout", 60, "Tool-call timeout in seconds")
	cmd.Flags().Int64Var(&teamID, "team-id", 0, "Team scope (0 = account-scope)")

	return cmd
}
