# fduty safari — command card

Prereq: `SKILL.md` read. This is the **AI-SRE platform self-management** group: install/configure the account's own **MCP servers (connectors)**, **skills**, and **A2A agents**, plus inspect **sessions**. Mutating verbs (`create`, `update`, `delete`, `upload`) change account configuration — confirm before running. `delete` is **irreversible**.

> Registering an MCP server is THIS group (`fduty safari mcp-server-create`) — **not** a tool search. A tool search only discovers callable tools on servers already connected to you; it can neither register nor configure one.

## Route here when

"安装 / 添加 / 配置 MCP / connector / 连接器 / install mcp / add mcp server / 上传 skill / 自定义 skill / skill upload / A2A agent / customize / AI-SRE 平台配置 / session 导出" → **safari**. Key IDs: **`server_id`** (`mcp_…`) from `mcp-server-list`; **`skill_id`** from `skill-list`; **`agent_id`** from `a2a-agent-list`; **`session_id`** (`sess_…`).

## Intent → verb

| want | verb |
|---|---|
| list MCP servers / connectors | `mcp-server-list` |
| install / register an MCP server | `mcp-server-create` |
| change an MCP server's config | `mcp-server-update` |
| turn an MCP server on / off | `mcp-server-enable` / `mcp-server-disable` |
| inspect one MCP server (+ live tool probe) | `mcp-server-get` |
| remove an MCP server | `mcp-server-delete` |
| list / upload / update a skill | `skill-list` / `skill-upload` / `skill-update` |
| download / enable / disable / delete a skill | `skill-download` / `skill-enable` / `skill-disable` / `skill-delete` |
| list / create / update an A2A agent | `a2a-agent-list` / `a2a-agent-create` / `a2a-agent-update` |
| enable / disable / delete an A2A agent | `a2a-agent-enable` / `a2a-agent-disable` / `a2a-agent-delete` |
| list / get / export a session transcript | `session-list` / `session-get` / `session-export` |

## Hot flow — install an MCP server

Pass the nested `env` / `headers` objects through `--data` (they have no scalar flags); `args` has a repeatable `--args` flag but is shown via `--data` below for one-line copy-paste.

```bash
# stdio (local process): command + args + secrets via env
fduty safari mcp-server-create --data '{"server_name":"GitHub Tools","transport":"stdio","description":"Read issues and pull requests from GitHub.","command":"npx","args":["-y","@modelcontextprotocol/server-github"],"env":{"GITHUB_TOKEN":"ghp_xxx"},"team_id":0,"status":"enabled"}'

# remote (streamable-http) with per-user OAuth — oauth_metadata stays empty (auto-discovered + DCR at runtime)
fduty safari mcp-server-create --data '{"server_name":"Aliyun OpenAPI","transport":"streamable-http","description":"Alibaba Cloud OpenAPI MCP.","url":"https://openapi-mcp.example.com/mcp","auth_mode":"per_user_oauth","team_id":0,"status":"enabled"}'

# confirm it registered, then inspect its live tool catalogue
fduty safari mcp-server-list --output-format toon
fduty safari mcp-server-get --data '{"server_id":"mcp_xxx"}'
```

<!-- GENERATED:safari START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### a2a-agent-create
Create A2A agent
- `--agent-name` string (required) — Display name of the agent. (≤128 chars)
- `--auth-mode` string — Credential model; defaults to shared.
- `--auth-type` string — Authentication scheme used when calling the agent.
- `--card-url` string (required) — URL of the agent's published A2A agent card.
- `--description` string — What this agent does and when to delegate to it.
- `--oauth-metadata` string — OAuth metadata JSON; reserved for OAuth-based auth.
- `--secret-schema` string — JSON schema of the per-user secret; required when auth_mode is per_user_secret.
- `--streaming` bool — Whether the agent supports streaming responses.
- `--team-id` int64 — Owning team for the new agent; 0 for account scope.
- body-only (`--data`): auth_config (object)

### a2a-agent-delete <agent-id>
Delete A2A agent
- `<agent-id>` (positional, required) string — Identifier of the target agent.

### a2a-agent-disable <agent-id>
Disable A2A agent
- `<agent-id>` (positional, required) string — Identifier of the target agent.

### a2a-agent-enable <agent-id>
Enable A2A agent
- `<agent-id>` (positional, required) string — Identifier of the target agent.

### a2a-agent-get <agent-id>
Get A2A agent detail
- `<agent-id>` (positional, required) string — Identifier of the target agent.

### a2a-agent-list
List A2A agents
- `--include-account` bool — Include account-scoped rows alongside team-scoped ones; defaults to true.
- `--limit` int64 — Maximum number of rows to return; defaults to 20.
- `--offset` int64 — Number of rows to skip for pagination.
- `--team-ids` intSlice — Restrict results to resources owned by these teams; intersected with the caller's visible set.

### a2a-agent-update <agent-id>
Update A2A agent
- `<agent-id>` (positional, required) string — Identifier of the agent to update.
- `--agent-name` string — New display name. (≤128 chars)
- `--auth-mode` string — New credential model.
- `--auth-type` string — New authentication scheme.
- `--card-url` string — New agent card URL.
- `--description` string — New description.
- `--oauth-metadata` string — New OAuth metadata JSON.
- `--secret-schema` string — New per-user secret JSON schema.
- `--streaming` bool — Toggle streaming-response support.
- `--team-id` int64 — Reassign the agent to this team; omit to leave unchanged, 0 for account scope.
- body-only (`--data`): auth_config (object)

### mcp-server-create
Create MCP server
- `--args` stringSlice — Command-line arguments for the stdio executable.
- `--auth-mode` string — Credential model; defaults to shared.
- `--call-timeout` int64 — Per-call timeout in seconds.
- `--command` string — Executable to launch for stdio transport.
- `--connect-timeout` int64 — Connection timeout in seconds.
- `--description` string (required) — What this MCP server provides. (1-1024 chars)
- `--oauth-metadata` string — OAuth metadata JSON; reserved for OAuth-based auth.
- `--secret-schema` string — JSON schema of the per-user secret; required when auth_mode is per_user_secret.
- `--server-name` string (required) — Display name of the server. (1-255 chars)
- `--status` string — Initial lifecycle state of the server. · enum: enabled | disabled
- `--team-id` int64 — Owning team for the new server; 0 for account scope.
- `--transport` string (required) — Transport used to reach the server. · enum: stdio | sse | streamable-http
- `--url` string — Endpoint URL for sse or streamable-http transport.
- body-only (`--data`): env (object); headers (object)

### mcp-server-delete <server-id>
Delete MCP server
- `<server-id>` (positional, required) string — Identifier of the server to delete.

### mcp-server-disable <server-id>
Disable MCP server
- `<server-id>` (positional, required) string — Identifier of the target server.

### mcp-server-enable <server-id>
Enable MCP server
- `<server-id>` (positional, required) string — Identifier of the target server.

### mcp-server-get <server-id>
Get MCP server detail
- `<server-id>` (positional, required) string — Identifier of the server to fetch.

### mcp-server-list
List MCP servers
- `--include-account` bool — Include account-scoped rows alongside team-scoped ones; defaults to true.
- `--limit` int64 — Page size; defaults to 20.
- `--page` int64 — Page number, starting at 1.
- `--search-after-ctx` string
- `--team-ids` intSlice — Restrict results to resources owned by these teams; intersected with the caller's visible set.

### mcp-server-update <server-id>
Update MCP server
- `--args` stringSlice — New stdio arguments.
- `--auth-mode` string — New credential model.
- `--call-timeout` int64 — New per-call timeout in seconds.
- `--command` string — New stdio executable.
- `--connect-timeout` int64 — New connection timeout in seconds.
- `--description` string — New description. (1-1024 chars)
- `--oauth-metadata` string — New OAuth metadata JSON.
- `--secret-schema` string — New per-user secret JSON schema.
- `<server-id>` (positional, required) string — Identifier of the server to update.
- `--server-name` string — New display name. (1-255 chars)
- `--team-id` int64 — Reassign the server to this team; omit to leave unchanged, 0 for account scope.
- `--transport` string — New transport for the server. · enum: stdio | sse | streamable-http
- `--url` string — New endpoint URL for remote transports.
- body-only (`--data`): env (object); headers (object)

### session-export <session_id>
Stream a session's full event transcript as NDJSON
- `--include-subagents` bool

### session-get <session-id>
Get session detail
- `--limit` int64 — Alias for num_recent_events; takes precedence when both are set. (0-1000)
- `--num-recent-events` int64 — Number of most-recent events to return; 0 uses the server default. (0-1000)
- `--search-after-ctx` string — Opaque keyset cursor from a previous response's search_after_ctx, to page backward through older events. (≤4096 chars)
- `<session-id>` (positional, required) string — Session identifier. (≥1 chars)

### session-list
List sessions
- `--app-name` string (required) — Agent app whose sessions to list. · enum: ask-ai | support | support-website | support-flashcat | ai-sre | template-assistant
- `--asc` bool — Ascending sort when true; defaults to false (descending). Only honored when orderby is set.
- `--entry-kinds` stringSlice — Restrict to sessions produced by these entry surfaces. Empty returns every kind. · enum: web | im | api | scheduled
- `--include-subagent-sessions` bool — Include subagent (child) sessions in the result; defaults to false.
- `--keyword` string — Case-insensitive substring match against session name. (≤64 chars)
- `--limit` int64 — Page size, 1..100; defaults to 20. (1-100)
- `--orderby` string — Sort column. · enum: created_at | updated_at
- `--page` int64 — 1-based page number; defaults to 1.
- `--scope` string — Visibility scope: all (own + member-of-team rows, the default), personal (own only), or team (member teams only). · enum: all | personal | team
- `--search-after-ctx` string
- `--status` string — Archive bucket: active (default, not archived), archived, or all. · enum: active | archived | all
- `--team-ids` intSlice — Optional explicit team filter; intersected with the caller's visible set / scope.

### skill-delete <skill-id>
Delete skill
- `<skill-id>` (positional, required) string — Identifier of the skill to delete.

### skill-disable <skill-id>
Disable skill
- `<skill-id>` (positional, required) string — Identifier of the target skill.

### skill-download <skill-id>
Download skill
- `<skill-id>` (positional, required) string — Identifier of the skill to download.

### skill-enable <skill-id>
Enable skill
- `<skill-id>` (positional, required) string — Identifier of the target skill.

### skill-get <skill-id>
Get skill detail
- `<skill-id>` (positional, required) string — Identifier of the skill to fetch.

### skill-list
List skills
- `--include-account` bool — Include account-scoped rows alongside team-scoped ones; defaults to true.
- `--limit` int64 — Page size; defaults to 20.
- `--page` int64 — Page number, starting at 1.
- `--search-after-ctx` string
- `--team-ids` intSlice — Restrict results to resources owned by these teams; intersected with the caller's visible set.

### skill-update <skill-id>
Update skill
- `--description` string — New description for the skill. (≤1024 chars)
- `<skill-id>` (positional, required) string — Identifier of the skill to update.
- `--team-id` int64 — Reassign the skill to this team; omit to leave unchanged, 0 for account scope.

### skill-upload
Upload skill

<!-- GENERATED:safari END -->

## Key concepts

- **Transport ⇒ which fields matter.** `stdio` uses `command` + `args` + `env`; `sse` / `streamable-http` use `url` + `headers`. The nested `env` / `headers` objects have no scalar flags — pass them through `--data '{...}'`; typed scalar flags (`--server-name`, `--url`, `--args`, …) override matching `--data` keys.
- **Scope (`team_id`).** `0` = account-wide (every team sees it); `>0` = that team only. Same field on every safari verb.
- **Auth mode (`auth_mode`).** `shared` (default) = one credential for everyone, stored on the server. `per_user_secret` = each user supplies a secret matching `secret_schema` (which must carry a `header_name`). `per_user_oauth` = each user authorizes the server via OAuth.
- **`per_user_oauth` needs no OAuth config up front.** Create it with an **empty `oauth_metadata`** — that is the normal, complete state, not a missing prerequisite. The runtime **auto-discovers the OAuth server and dynamically registers a client (DCR)** the first time a user authorizes; you do **not** collect `authorization_url` / `client_id` / `client_secret` / `scopes`. Only pass `oauth_metadata` as a rare fallback, when the endpoint advertises no discovery document.

## Gotchas

- **`mcp-server-create` requires `server_name`, `description`, `transport`.** A `stdio` server also needs `command`; a remote (`sse` / `streamable-http`) server needs `url`.
- **`env` / `headers` go through `--data`** — there are no `--env` / `--header` scalar flags for them (`args` does have a repeatable `--args` flag).
- **Don't reach for a tool search to install or configure a server** — that only finds tools on already-connected servers. Registration and configuration are `mcp-server-create` / `mcp-server-update`.
- **`delete` is irreversible** — prefer `disable` to park a server / skill / agent without destroying it. `list` first to confirm the id.
