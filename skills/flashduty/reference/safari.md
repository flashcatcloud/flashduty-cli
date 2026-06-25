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
| enable / disable / delete a skill | `skill-enable` / `skill-disable` / `skill-delete` |
| list / create / update an A2A agent | `a2a-agent-list` / `a2a-agent-create` / `a2a-agent-update` |
| enable / disable / delete an A2A agent | `a2a-agent-enable` / `a2a-agent-disable` / `a2a-agent-delete` |
| list / get / export / delete a session | `session-list` / `session-get` / `session-export` / `session-delete` |

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
- `--agent-name` string (required) — Agent display name. (≤128 chars)
- `--auth-mode` string — Authentication mode: shared (default), per_user_secret, or per_user_oauth.
- `--auth-type` string — Authentication type for the remote agent.
- `--card-url` string (required) — URL of the remote agent card.
- `--description` string — Agent description.
- `--oauth-metadata` string — JSON OAuth metadata; reserved for per_user_oauth.
- `--secret-schema` string — JSON secret schema; required when auth_mode=per_user_secret.
- `--streaming` bool — Whether the remote agent supports streaming.
- `--team-id` int64 — Team scope: 0 = account-wide; >0 = team.
- body-only (`--data`): auth_config (object)

### a2a-agent-delete <agent-id>
Delete A2A agent
- `<agent-id>` (positional, required) string — Target agent ID.

### a2a-agent-disable <agent-id>
Disable A2A agent
- `<agent-id>` (positional, required) string — Target agent ID.

### a2a-agent-enable <agent-id>
Enable A2A agent
- `<agent-id>` (positional, required) string — Target agent ID.

### a2a-agent-get <agent-id>
Get A2A agent detail
- `<agent-id>` (positional, required) string — Target agent ID.

### a2a-agent-list
List A2A agents
- `--include-account` bool — Include account-scoped (team_id=0) rows. Defaults to true.
- `--limit` int64 — Page size.
- `--offset` int64 — Row offset for pagination.
- `--team-ids` intSlice — Filter to these team IDs; empty = the caller's visible set.

### a2a-agent-update <agent-id>
Update A2A agent
- `<agent-id>` (positional, required) string — Target agent ID.
- `--agent-name` string — New display name. Omit to leave unchanged. (≤128 chars)
- `--auth-mode` string — New auth mode: shared, per_user_secret, or per_user_oauth.
- `--auth-type` string — New auth type. Omit to leave unchanged.
- `--card-url` string — New card URL. Omit to leave unchanged.
- `--description` string — New description. Omit to leave unchanged.
- `--oauth-metadata` string — New JSON OAuth metadata.
- `--secret-schema` string — New JSON secret schema.
- `--streaming` bool — Toggle streaming support. Omit to leave unchanged.
- `--team-id` int64 — Reassign team scope. Omit to leave unchanged.
- body-only (`--data`): auth_config (object)

### mcp-server-create
Create MCP server
- `--args` stringSlice — Command arguments (stdio transport).
- `--auth-mode` string — Authentication mode: shared (default), per_user_secret, or per_user_oauth.
- `--call-timeout` int64 — Tool-call timeout in seconds. 0 = default (60s).
- `--command` string — Executable command (stdio transport).
- `--connect-timeout` int64 — Connection timeout in seconds. 0 = default (10s).
- `--description` string (required) — Server description. (1-1024 chars)
- `--oauth-metadata` string — JSON OAuth metadata; reserved for per_user_oauth.
- `--secret-schema` string — JSON secret schema; required when auth_mode=per_user_secret.
- `--server-name` string (required) — MCP server name, unique within the account. (1-255 chars)
- `--source-template-name` string — Marketplace template name when created from a connector template.
- `--status` string — Initial status. · enum: enabled | disabled
- `--team-id` int64 — Team scope: 0 = account-wide; >0 = team.
- `--transport` string (required) — Transport protocol. · enum: stdio | sse | streamable-http
- `--url` string — Server URL (sse / streamable-http transport).
- body-only (`--data`): env (object); headers (object)

### mcp-server-delete <server-id>
Delete MCP server
- `<server-id>` (positional, required) string — Target MCP server ID.

### mcp-server-disable <server-id>
Disable MCP server
- `<server-id>` (positional, required) string — Target MCP server ID.

### mcp-server-enable <server-id>
Enable MCP server
- `<server-id>` (positional, required) string — Target MCP server ID.

### mcp-server-get <server-id>
Get MCP server detail
- `<server-id>` (positional, required) string — Target MCP server ID.

### mcp-server-list
List MCP servers
- `--include-account` bool — Include account-scoped (team_id=0) rows. Defaults to true.
- `--limit` int64 — Page size.
- `--page` int64 — Page number, 1-based.
- `--search-after-ctx` string
- `--team-ids` intSlice — Filter to these team IDs; empty = the caller's visible set.

### mcp-server-update <server-id>
Update MCP server
- `--args` stringSlice — Command arguments (stdio transport).
- `--auth-mode` string — Authentication mode: shared (default), per_user_secret, or per_user_oauth.
- `--call-timeout` int64 — Tool-call timeout in seconds. 0 = default (60s).
- `--command` string — Executable command (stdio transport).
- `--connect-timeout` int64 — Connection timeout in seconds. 0 = default (10s).
- `--description` string — New description. (1-1024 chars)
- `--oauth-metadata` string — JSON OAuth metadata; reserved for per_user_oauth.
- `--secret-schema` string — JSON secret schema; required when auth_mode=per_user_secret.
- `<server-id>` (positional, required) string — Target MCP server ID.
- `--server-name` string — New name. (1-255 chars)
- `--team-id` int64 — Reassign team scope: 0 = account-wide; >0 = team. Omit to leave unchanged.
- `--transport` string — Transport protocol. · enum: stdio | sse | streamable-http
- `--url` string — Server URL (sse / streamable-http transport).
- body-only (`--data`): env (object); headers (object)

### session-delete <session-id>
Delete session
- `<session-id>` (positional, required) string — Target session ID. (≥1 chars)

### session-export <session_id>
Stream a session's full event transcript as NDJSON
- `--include-subagents` bool

### session-get <session-id>
Get session detail
- `--limit` int64 — Page size for events; takes precedence over 'num_recent_events'. 0 uses the server default (100). (0-1000)
- `--num-recent-events` int64 — Legacy page size: number of most-recent events to return. Superseded by 'limit' when both are set; 0 uses the server default (100). (0-1000)
- `--search-after-ctx` string — Opaque keyset cursor from a previous response; pass it back to fetch the next older page. (≤4096 chars)
- `<session-id>` (positional, required) string — Target session ID. (≥1 chars)

### session-list
List sessions
- `--app-name` string (required) — Agent app whose sessions to list. · enum: ask-ai | support | support-website | support-flashcat | ai-sre | template-assistant | swe
- `--asc` bool — Ascending order when true; applies only when 'orderby' is set.
- `--entry-kinds` stringSlice — Restrict to sessions produced by these surfaces; empty returns every kind. · enum: web | im | api | automation
- `--include-subagent-sessions` bool — Include subagent-dispatched sessions in the list.
- `--keyword` string — Filter by session-name keyword. (≤64 chars)
- `--limit` int64 — Page size, 1–100. (1-100)
- `--orderby` string — Sort field. · enum: created_at | updated_at
- `--page` int64 — Page number, 1-based. (min 1)
- `--scope` string — Visibility scope: all (own + member-of-team rows, default), personal, or team. · enum: all | personal | team
- `--search-after-ctx` string
- `--status` string — Archive bucket: active (default) returns un-archived, archived returns archived, all returns both. · enum: active | archived | all
- `--team-ids` intSlice — Optional explicit team filter; intersects with 'scope'.

### skill-delete <skill-id>
Delete skill
- `<skill-id>` (positional, required) string — Target skill ID.

### skill-disable <skill-id>
Disable skill
- `<skill-id>` (positional, required) string — Target skill ID.

### skill-enable <skill-id>
Enable skill
- `<skill-id>` (positional, required) string — Target skill ID.

### skill-get <skill-id>
Get skill detail
- `<skill-id>` (positional, required) string — Target skill ID.

### skill-list
List skills
- `--include-account` bool — Include account-scoped (team_id=0) rows. Defaults to true.
- `--limit` int64 — Page size.
- `--page` int64 — Page number, 1-based.
- `--search-after-ctx` string
- `--team-ids` intSlice — Filter to these team IDs; empty = the caller's visible set.

### skill-update <skill-id>
Update skill
- `--description` string — New description. (≤1024 chars)
- `<skill-id>` (positional, required) string — Target skill ID.
- `--team-id` int64 — Reassign team scope: 0 = account-wide; >0 = team. Omit to leave unchanged.

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
