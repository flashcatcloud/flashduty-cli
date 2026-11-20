package cli

import (
	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"
)

// Curated IM self-chat list commands attached to the generated `datasource`
// group. The four endpoints use gateway Auth "service" (app_key-callable,
// excluded from the public OpenAPI spec) and are served by hand-written SDK
// methods on flashduty.ImIntegrationsService.

func attachDatasourceSelfChats(root *cobra.Command) {
	g := genGroup(root, "datasource", "On-call API")
	genAddLeaf(g, newDatasourceDingtalkSelfChatListCmd())
	genAddLeaf(g, newDatasourceFeishuSelfChatListCmd())
	genAddLeaf(g, newDatasourceSlackSelfChatListCmd())
	genAddLeaf(g, newDatasourceTeamsSelfChatListCmd())
}

func newDatasourceDingtalkSelfChatListCmd() *cobra.Command {
	var dataJSON string
	var fDataSourceID int64

	cmd := &cobra.Command{
		Use:   "dingtalk-self-chat-list <data-source-id>",
		Short: "List DingTalk chats for an IM integration",
		Long: `List DingTalk group chats where the integration's bot is installed.

API: POST /datasource/dingtalk/self-chat/list

Request fields:
  --data-source-id int (required) — DingTalk IM integration ID (integration_id / data_source_id); obtain it from 'flashduty datasource im-war-room-enabled-list' or 'flashduty integration list'.

Response fields ('data' envelope is unwrapped — rows are nested under items[]; pipe 'jq '.items[]'', NOT '.data.items[]'):
  - items (array<object>) (required) — DingTalk group chats where the bot is installed.
    - chat_id (string) (required) — DingTalk chat ID (openConversationId) used in escalation rules.
    - name (string) (required) — Group chat name.
    - origin (string) — Chat origin tag ('user' or 'war_room').`,
		Args:    requireBodyFieldOrExactArg("data_source_id", "data-source-id"),
		Example: `  flashduty datasource dingtalk-self-chat-list 6100259006783 --output-format toon`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				body, err := genAssembleBody(dataJSON, func(body map[string]any) error {
					if err := genFoldPositional(args, body, "data_source_id", "int"); err != nil {
						return err
					}
					if cmd.Flags().Changed("data-source-id") {
						body["data_source_id"] = fDataSourceID
					}
					return nil
				})
				if err != nil {
					return err
				}
				req := new(flashduty.DingtalkSelfChatListRequest)
				if err := genBindBody(ctx.Cmd.ErrOrStderr(), body, req); err != nil {
					return err
				}
				out, _, err := ctx.Client.ImIntegrations.DingtalkSelfChatList(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out, body, listDefaults{})
			})
		},
	}

	cmd.Flags().Int64Var(&fDataSourceID, "data-source-id", 0, "DingTalk IM integration ID (integration_id / data_source_id). (required)")
	cmd.Flags().StringVar(&dataJSON, "data", "", "Full request body as JSON; positional arguments and typed flags override its fields. Accepts inline JSON, or - to read stdin.")
	return cmd
}

func newDatasourceFeishuSelfChatListCmd() *cobra.Command {
	var dataJSON string
	var fDataSourceID int64

	cmd := &cobra.Command{
		Use:   "feishu-self-chat-list <data-source-id>",
		Short: "List Feishu chats for an IM integration",
		Long: `List Feishu (Lark) group chats the integration's bot belongs to.

API: POST /datasource/feishu/self-chat/list

Request fields:
  --data-source-id int (required) — Feishu IM integration ID (integration_id / data_source_id); obtain it from 'flashduty datasource im-war-room-enabled-list' or 'flashduty integration list'.

Response fields ('data' envelope is unwrapped — rows are nested under items[]; pipe 'jq '.items[]'', NOT '.data.items[]'):
  - items (array<object>) (required) — Feishu group chats the bot belongs to.
    - chat_id (string) (required) — Feishu chat ID (oc_...) used in escalation rules.
    - name (string) (required) — Group chat name.
    - description (string) — Group chat description.
    - external (boolean) — Whether the chat includes external users.
    - origin (string) — Chat origin tag ('user' or 'war_room').`,
		Args:    requireBodyFieldOrExactArg("data_source_id", "data-source-id"),
		Example: `  flashduty datasource feishu-self-chat-list 6100259006783 --output-format toon`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				body, err := genAssembleBody(dataJSON, func(body map[string]any) error {
					if err := genFoldPositional(args, body, "data_source_id", "int"); err != nil {
						return err
					}
					if cmd.Flags().Changed("data-source-id") {
						body["data_source_id"] = fDataSourceID
					}
					return nil
				})
				if err != nil {
					return err
				}
				req := new(flashduty.FeishuSelfChatListRequest)
				if err := genBindBody(ctx.Cmd.ErrOrStderr(), body, req); err != nil {
					return err
				}
				out, _, err := ctx.Client.ImIntegrations.FeishuSelfChatList(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out, body, listDefaults{})
			})
		},
	}

	cmd.Flags().Int64Var(&fDataSourceID, "data-source-id", 0, "Feishu IM integration ID (integration_id / data_source_id). (required)")
	cmd.Flags().StringVar(&dataJSON, "data", "", "Full request body as JSON; positional arguments and typed flags override its fields. Accepts inline JSON, or - to read stdin.")
	return cmd
}

func newDatasourceSlackSelfChatListCmd() *cobra.Command {
	var dataJSON string
	var fDataSourceID int64
	var fForceRefresh bool

	cmd := &cobra.Command{
		Use:   "slack-self-chat-list <data-source-id>",
		Short: "List Slack channels for an IM integration",
		Long: `List Slack channels visible to the integration.

API: POST /datasource/slack/self-chat/list

Request fields:
  --data-source-id int (required) — Slack IM integration ID (integration_id / data_source_id).
  --force-refresh bool — Bypass server-side channel cache.

Response fields ('data' envelope is unwrapped — rows are nested under items[]; pipe 'jq '.items[]'', NOT '.data.items[]'):
  - items (array<object>) (required) — Slack channels visible to the integration.
    - id (string) (required) — Slack channel ID used in escalation rules.
    - name (string) (required) — Channel name.
    - is_private (boolean) — Whether the channel is private.
    - is_archived (boolean) — Whether the channel is archived.
    - origin (string) — Channel origin tag ('user' or 'war_room').`,
		Args:    requireBodyFieldOrExactArg("data_source_id", "data-source-id"),
		Example: `  flashduty datasource slack-self-chat-list 6100259006783 --output-format toon`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				body, err := genAssembleBody(dataJSON, func(body map[string]any) error {
					if err := genFoldPositional(args, body, "data_source_id", "int"); err != nil {
						return err
					}
					if cmd.Flags().Changed("data-source-id") {
						body["data_source_id"] = fDataSourceID
					}
					if cmd.Flags().Changed("force-refresh") {
						body["force_refresh"] = fForceRefresh
					}
					return nil
				})
				if err != nil {
					return err
				}
				req := new(flashduty.SlackSelfChatListRequest)
				if err := genBindBody(ctx.Cmd.ErrOrStderr(), body, req); err != nil {
					return err
				}
				out, _, err := ctx.Client.ImIntegrations.SlackSelfChatList(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out, body, listDefaults{})
			})
		},
	}

	cmd.Flags().Int64Var(&fDataSourceID, "data-source-id", 0, "Slack IM integration ID (integration_id / data_source_id). (required)")
	cmd.Flags().BoolVar(&fForceRefresh, "force-refresh", false, "Bypass server-side Slack channel cache.")
	cmd.Flags().StringVar(&dataJSON, "data", "", "Full request body as JSON; positional arguments and typed flags override its fields. Accepts inline JSON, or - to read stdin.")
	return cmd
}

func newDatasourceTeamsSelfChatListCmd() *cobra.Command {
	var dataJSON string
	var fDataSourceID int64
	var fTeamID string
	var fForceRefresh bool

	cmd := &cobra.Command{
		Use:   "teams-self-chat-list <data-source-id>",
		Short: "List Microsoft Teams channels for an IM integration",
		Long: `List Microsoft Teams channels linked to the integration.

API: POST /datasource/teams/self-chat/list

Request fields:
  --data-source-id int (required) — Microsoft Teams IM integration ID (integration_id / data_source_id).
  --team-id string — Optional Microsoft Teams team ID filter.
  --force-refresh bool — Bypass server-side channel cache.

Response fields ('data' envelope is unwrapped — rows are nested under items[]; pipe 'jq '.items[]'', NOT '.data.items[]'):
  - items (array<object>) (required) — Linked Microsoft Teams teams and their channels.
    - team_id (string) — Microsoft Teams team ID.
    - team_name (string) — Microsoft Teams team name.
    - channels (array<object>) — Channels under this team (channel_id, channel_name, origin).`,
		Args:    requireBodyFieldOrExactArg("data_source_id", "data-source-id"),
		Example: `  flashduty datasource teams-self-chat-list 6100259006783 --output-format toon`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				body, err := genAssembleBody(dataJSON, func(body map[string]any) error {
					if err := genFoldPositional(args, body, "data_source_id", "int"); err != nil {
						return err
					}
					if cmd.Flags().Changed("data-source-id") {
						body["data_source_id"] = fDataSourceID
					}
					if cmd.Flags().Changed("team-id") {
						body["team_id"] = fTeamID
					}
					if cmd.Flags().Changed("force-refresh") {
						body["force_refresh"] = fForceRefresh
					}
					return nil
				})
				if err != nil {
					return err
				}
				req := new(flashduty.TeamsSelfChatListRequest)
				if err := genBindBody(ctx.Cmd.ErrOrStderr(), body, req); err != nil {
					return err
				}
				out, _, err := ctx.Client.ImIntegrations.TeamsSelfChatList(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}
				return printGenericResult(ctx, out, body, listDefaults{})
			})
		},
	}

	cmd.Flags().Int64Var(&fDataSourceID, "data-source-id", 0, "Microsoft Teams IM integration ID (integration_id / data_source_id). (required)")
	cmd.Flags().StringVar(&fTeamID, "team-id", "", "Optional Microsoft Teams team ID filter.")
	cmd.Flags().BoolVar(&fForceRefresh, "force-refresh", false, "Bypass server-side channel cache.")
	cmd.Flags().StringVar(&dataJSON, "data", "", "Full request body as JSON; positional arguments and typed flags override its fields. Accepts inline JSON, or - to read stdin.")
	return cmd
}
