# flashduty safari — command card

Prereq: `SKILL.md` read. This is the **AI-SRE platform self-management** group: install/configure the account's own **MCP servers (connectors)**, **skills**, and **A2A agents**, plus inspect **sessions**. Mutating verbs (`create`, `update`, `delete`, `upload`) change account configuration — confirm before running. `delete` is **irreversible**.

> Registering an MCP server is THIS group (`flashduty safari mcp-server-create`) — **not** a tool search. A tool search only discovers callable tools on servers already connected to you; it can neither register nor configure one.

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
| run an Automation rule immediately, outside its schedule | `automation-rule-run <rule-id>` |
| manage Automation rules directly (create/update/delete/list/runs/…) | `automation-rule-*` / `automation-run-list` / `automation-template-list` — see **`reference/automation.md`** for the primary router |

## Hot flow — install an MCP server

Pass the nested `env` / `headers` objects through `--data` (they have no scalar flags); `args` has a repeatable `--args` flag but is shown via `--data` below for one-line copy-paste.

```bash
# stdio (local process): command + args + secrets via env
flashduty safari mcp-server-create --data '{"server_name":"GitHub Tools","transport":"stdio","description":"Read issues and pull requests from GitHub.","command":"npx","args":["-y","@modelcontextprotocol/server-github"],"env":{"GITHUB_TOKEN":"ghp_xxx"},"team_id":0,"status":"enabled"}'

# remote (streamable-http) with per-user OAuth — oauth_metadata stays empty (auto-discovered + DCR at runtime)
flashduty safari mcp-server-create --data '{"server_name":"Aliyun OpenAPI","transport":"streamable-http","description":"Alibaba Cloud OpenAPI MCP.","url":"https://openapi-mcp.example.com/mcp","auth_mode":"per_user_oauth","team_id":0,"status":"enabled"}'

# confirm it registered, then inspect its live tool catalogue
flashduty safari mcp-server-list --output-format toon
flashduty safari mcp-server-get --data '{"server_id":"mcp_xxx"}'
```

<!-- GENERATED:safari START · 由 flashduty __dump-commands 同步 · 勿手改 fence 内 -->

### a2a-agent-create
Create A2A agent
- `--agent-name` string (required) — Agent display name, used to address the agent in sessions. Must not contain '@', must not be 'general' or 'explore', and must be unique within its scope (account-wide or one team). (≤128 chars)
- `--allow-insecure-oauth-http` bool — Allow non-loopback HTTP OAuth discovery/metadata endpoints for this agent instead of requiring HTTPS. Defaults to false.
- `--allow-insecure-tls-skip-verify` bool — Skip TLS certificate verification when connecting to this agent's endpoint (self-signed/private certs). Defaults to false.
- `--auth-mode` string — Who supplies the credential: - 'shared' (default; an empty value behaves the same): one credential stored on the agent ('auth_type' + 'auth_config') is used for every caller. - 'per_user_secret': each user stores their own secret, sent in the header named by 'secret_schema.header_name'. - 'per_user_oauth': each user completes their own OAuth authorization; discovery and client registration run on first use. 'per_user_secret' requires 'secret_schema' with a non-empty 'header_name'; any other value returns 'InvalidParameter'. · enum: shared | per_user_secret | per_user_oauth
- `--auth-type` string — Authentication used to reach the remote agent with a shared credential: - 'none': no credential header. - 'api_key': sends 'auth_config.api_key' in the header named by 'auth_config.header'. - 'bearer': sends 'Authorization: Bearer <auth_config.token>'. When omitted the value is stored as an empty string, which behaves like 'none'. · enum: none | api_key | bearer
- `--card-url` string (required) — URL of the remote agent card. Must be an absolute 'http' or 'https' URL with a non-empty host; a URL with no path is resolved to '/.well-known/agent-card.json'. If an execution environment is online, the card is fetched and must declare an A2A 1.x 'JSONRPC' or 'HTTP+JSON' interface.
- `--environments` stringSlice — Execution environments this agent is callable from: 'cloud' and/or BYOC runner environment IDs. Omitted or empty means all environments.
- `--instructions` string (required) — Natural-language instructions for the remote agent: a Markdown document with optional 'summary' frontmatter and a non-empty body, at most 50 KiB (51200 bytes). Required — a deprecated 'description' field is still accepted for legacy clients and, if both are sent, must exactly match 'instructions'. (≤51200 chars)
- `--oauth-metadata` string — JSON-encoded OAuth metadata; populated by the OAuth discovery flow for 'per_user_oauth' mode.
- `--secret-schema` string — JSON-encoded secret schema, e.g. '{"header_name":"X-Api-Key"}'; required when 'auth_mode=per_user_secret'.
- `--streaming` bool — Whether the remote agent supports streaming.
- `--team-id` int64 — Team scope: 0 = account-wide; >0 = team. Creating at account scope requires the owner/admin role; creating into a team requires actual membership in that team.
- body-only (`--data`): auth_config (object)
- response: single object (`data` unwrapped to the top level) — fields: agent_id (string)

### a2a-agent-delete <agent-id>
Delete A2A agent
- `<agent-id>` (positional, required) string — Target agent ID, from the list returned by 'POST /safari/a2a-agent/list'.

### a2a-agent-disable <agent-id>
Disable A2A agent
- `<agent-id>` (positional, required) string — Target agent ID, from the list returned by 'POST /safari/a2a-agent/list'.

### a2a-agent-enable <agent-id>
Enable A2A agent
- `<agent-id>` (positional, required) string — Target agent ID, from the list returned by 'POST /safari/a2a-agent/list'.

### a2a-agent-get <agent-id>
Get A2A agent detail
- `<agent-id>` (positional, required) string — Target agent ID, from the list returned by 'POST /safari/a2a-agent/list'.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); agent_card_name (string); agent_card_skills (array<string>); agent_id (string); agent_name (string); allow_insecure_oauth_http (boolean); allow_insecure_tls_skip_verify (boolean); auth_config (object); auth_mode (string); auth_type (string); can_edit (boolean); card_resolve_timeout (integer); card_url (string); created_at (string); created_by (integer); environments (array<string>); instructions (string); oauth_metadata (string); secret_schema (string); status (string); streaming (boolean); task_timeout (integer); team_id (integer); updated_at (string)

### a2a-agent-list
List A2A agents
- `--include-account` bool — Include account-scoped ('team_id=0') rows. Defaults to 'true'; ignored when 'scope' is 'account' (always included) or 'team' (never included).
- `--limit` int64 — Page size; 0 or less falls back to 20. No upper bound is enforced.
- `--offset` int64 — Pagination offset — number of rows to skip, starting from 0.
- `--query` string — Case-insensitive substring search across agent name, instructions, card URL, agent ID, and the resolved card name. (≤128 chars)
- `--scope` string — Visibility scope: 'all' (account-scope plus the caller's visible teams), 'account' (account-scope only), or 'team' (team-scoped rows across the caller's visible teams). · enum: all | account | team
- `--team-ids` intSlice — Restrict to these team IDs. IDs of teams the caller does not belong to are ignored; empty means all of the caller's teams.
- response: `{items: [...], total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); agent_card_name (string); agent_card_skills (array<string>); agent_id (string); agent_name (string); allow_insecure_oauth_http (boolean); allow_insecure_tls_skip_verify (boolean); auth_config (object); auth_mode (string); auth_type (string); can_edit (boolean); card_resolve_timeout (integer); card_url (string); created_at (string); created_by (integer); environments (array<string>); instructions (string); oauth_metadata (string); secret_schema (string); status (string); streaming (boolean); task_timeout (integer); team_id (integer); updated_at (string)

### a2a-agent-update <agent-id>
Update A2A agent
- `<agent-id>` (positional, required) string — Target agent ID, from the list returned by 'POST /safari/a2a-agent/list'.
- `--agent-name` string — New display name; omit to leave unchanged. Same rules as create: no '@', not 'general' / 'explore', unique within the scope. (≤128 chars)
- `--allow-insecure-oauth-http` bool — Toggle non-loopback HTTP OAuth discovery for this agent. Omit to leave unchanged.
- `--allow-insecure-tls-skip-verify` bool — Toggle TLS certificate verification skipping for this agent. Omit to leave unchanged.
- `--auth-mode` string — New auth mode; omit to leave unchanged. Values (who supplies the credential): - 'shared' (default; an empty value behaves the same): one credential stored on the agent ('auth_type' + 'auth_config') is used for every caller. - 'per_user_secret': each user stores their own secret, sent in the header named by 'secret_schema.header_name'. - 'per_user_oauth': each user completes their own OAuth authorization; discovery and client registration run on first use. When sent, 'secret_schema' is overwritten with the value in the same request. · enum: shared | per_user_secret | per_user_oauth
- `--auth-type` string — New auth type; omit to leave unchanged. - 'none': no credential header. - 'api_key': sends 'auth_config.api_key' in the header named by 'auth_config.header'. - 'bearer': sends 'Authorization: Bearer <auth_config.token>'. · enum: none | api_key | bearer
- `--card-url` string — New card URL; omit to leave unchanged. Validated and compatibility-checked the same way as on create; a real change also clears 'oauth_metadata' unless a new one is sent.
- `--environments` stringSlice — Execution environments this agent is callable from: 'cloud' and/or BYOC runner environment IDs. Omit (null) to leave unchanged; send a list to set it — an empty list clears the restriction back to all environments.
- `--instructions` string — New instructions document (same contract as create: optional 'summary' frontmatter, non-empty body, at most 50 KiB). Omit to leave unchanged. A deprecated 'description' field is also accepted; if both are sent they must match. (≤51200 chars)
- `--oauth-metadata` string — New JSON OAuth metadata. A non-empty value is stored as sent. When omitted or empty, the stored value is kept unless 'auth_mode' or 'card_url' changes, in which case it is cleared.
- `--secret-schema` string — New JSON secret schema, e.g. '{"header_name":"X-Api-Key"}'. Applied only together with 'auth_mode'; sent alone it is ignored.
- `--streaming` bool — Toggle streaming support. Omit to leave unchanged.
- `--team-id` int64 — Reassign team scope; omit to leave unchanged. Moving to a team requires membership in it; moving to account scope ('0') requires the account owner or an admin. If the team changes without 'environments' in the same request, the current runner entries must stay selectable by the caller or the update is rejected.
- body-only (`--data`): auth_config (object)

### artifact-download <artifact-id | file-id | link>
Download an artifact or session file
- `--output` string

### artifact-gallery-delete <artifact-id>
Remove artifact from gallery
- `<artifact-id>` (positional, required) string — Artifact ID ('art_' prefix). Also the key of the public-share link.

### artifact-gallery-file-state <file-id> [<id2>...]
Get file publish state
- `<file-ids>` (positional, required) stringSlice — Presented-file IDs ('pf_' prefix) to check. At most 50 entries per call, counted before duplicates and empty strings are removed; more fails with 'InvalidParameter'.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: artifact_id (string); file_id (string); gallery_path (string); title (string)

### artifact-gallery-get <artifact-id>
Get artifact detail
- `<artifact-id>` (positional, required) string — Artifact ID ('art_' prefix). Also the key of the public-share link.
- response: single object (`data` unwrapped to the top level) — fields: artifact_id (string); can_edit (boolean); content_type (string); created_at (string); creator_name (string); file_id (string); is_mine (boolean); name (string); person_id (integer); pinned_at (string); public_url (string); session_id (string); session_title (string); share_enabled (boolean); share_file_id (string); shared_at (string); shared_by (integer); size (integer); team_id (integer); team_name (string); title (string); updated_at (string)

### artifact-gallery-list
List artifacts
- `--asc` bool — Sort ascending when true, descending when false. Applies only when 'orderby' is set.
- `--limit` int64 — Number of artifacts per page. Values below 1 fall back to 20; values above 100 are capped at 100 (not rejected).
- `--orderby` string — Sort field: 'created_at' (publish time) or 'updated_at' (last change). Empty sorts by 'updated_at' descending. In every case the caller's pinned artifacts come first, and 'artifact_id' breaks ties. · enum: created_at | updated_at
- `--page` int64 — Page number, 1-based. Values below 1 are treated as 1.
- `--query` string — Case-insensitive substring match on the artifact title.
- `--scope` string — Visibility scope. 'all' (default) lists the caller's own personal artifacts plus artifacts of every team the caller belongs to; 'personal' lists only the caller's own; 'team' lists only team-owned artifacts of the caller's teams. · enum: all | personal | team
- `--team-ids` intSlice — Restrict to artifacts owned by these team IDs, intersected with the 'scope' filter. Non-positive IDs are ignored; teams the caller does not belong to return nothing, and personal artifacts are excluded whenever this is set.
- response: `{items: [...], total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: artifact_id (string); can_edit (boolean); content_type (string); created_at (string); creator_name (string); file_id (string); is_mine (boolean); name (string); person_id (integer); pinned_at (string); public_url (string); session_id (string); session_title (string); share_enabled (boolean); share_file_id (string); shared_at (string); shared_by (integer); size (integer); team_id (integer); team_name (string); title (string); updated_at (string)

### artifact-gallery-publish-from-file <file-id>
Publish file as artifact
- `<file-id>` (positional, required) string — Presented-file ID ('pf_' prefix) of a file produced in a session, as shown on the file card in chat.
- `--title` string (required) — Gallery display title. Trimmed; must be non-empty.
- response: single object (`data` unwrapped to the top level) — fields: artifact_id (string); gallery_path (string); title (string)

### artifact-gallery-share-enable <artifact-id>
Enable public sharing
- `<artifact-id>` (positional, required) string — Artifact ID ('art_' prefix). Also the key of the public-share link.
- response: single object (`data` unwrapped to the top level) — fields: artifact_id (string); public_url (string); share_enabled (boolean); shared_at (string); shared_by (integer)

### artifact-gallery-share-revoke <artifact-id>
Revoke public sharing
- `<artifact-id>` (positional, required) string — Artifact ID ('art_' prefix). Also the key of the public-share link.

### artifact-gallery-share-sync <artifact-id>
Update shared snapshot
- `<artifact-id>` (positional, required) string — Artifact ID ('art_' prefix). Also the key of the public-share link.
- response: same shape as `artifact-gallery-share-enable <artifact-id>` above

### artifact-gallery-update <artifact-id>
Update artifact
- `<artifact-id>` (positional, required) string — Artifact ID ('art_' prefix). Also the key of the public-share link.
- `--team-id` int64 — Transfer target. '0' moves the artifact to personal scope and only its creator may do that; a positive value moves it to that team, and the caller must be a member of the team; a negative value fails with 'InvalidParameter'. Omit or send 'null' to keep the current scope. The creator ('person_id') never changes.
- `--title` string — New title. Leading and trailing spaces are trimmed; a blank value fails with 'InvalidParameter'. Omit or send 'null' to keep the current title.
- response: same shape as `artifact-gallery-get <artifact-id>` above

### artifact-sign <file-id>
Create signed file URLs
- `<file-id>` (positional, required) string — Presented-file ID ('pf_' prefix) to sign.
- `--share-token` string — Optional session share-link token. Needed only when the caller reaches the file through a shared session link rather than account membership.
- response: single object (`data` unwrapped to the top level) — fields: content_type (string); download_url (string); expires_in (integer); frame_base (string); frame_expires_in (integer); frame_url (string); name (string); preview_url (string); size (integer)

### artifact-stream --t <token>
Download a file by signed token
- `--output` string
- `--t` string

### automation-rule-create
Create Automation rule
- `--cron-expr` string (required) — Schedule in local time of 'timezone'. Accepts 4 fields ('hour day month weekday', minute defaults to 0) or 5 fields ('minute hour day month weekday'). The minute must be one fixed integer; seconds (6 fields) are not supported; setting both day-of-month and day-of-week is rejected. Required even for rules that use only other triggers: send a valid expression with 'schedule_trigger_enabled=false'.
- `--enabled` bool — Whether the rule's automatic triggers (schedule, HTTP POST, On-call incident, war room) are armed after creation. Defaults to 'false'. Manual runs work either way.
- `--environment-id` string — BYOC runner ID. Required when 'environment_kind=byoc' and rejected for other kinds. A personal rule can only use account-scope runners; a team rule can use account-scope runners or runners of its own team.
- `--environment-kind` string — Runtime environment kind. Omit or send an empty value for automatic selection. One of: 'cloud' (platform-hosted cloud sandbox), 'byoc' (a self-hosted BYOC runner in the account, used with 'environment_id'); automatic selection prefers an online BYOC runner and falls back to the cloud sandbox. · enum: cloud | byoc
- `--http-post-trigger-enabled` bool — Whether to create an enabled HTTP POST trigger. When 'true', the response carries 'http_post_trigger_url' and the one-time 'http_post_token'.
- `--name` string (required) — Display name of the rule, 1–255 characters; leading and trailing spaces are trimmed. (1-255 chars)
- `--oncall-incident-channel-ids` intSlice — Channel IDs to watch; incidents in these channels can start a run. Each ID must be positive.
- `--oncall-incident-merge-similar` bool — Whether to merge similar incidents into one analysis. When 'true' (default), an incident that closely resembles one already being analyzed by this rule (a run in progress or finished within the last 10 minutes) joins that run's session instead of starting a new one; its own run is recorded as 'skipped' with 'joined_session_id' set. When 'false', every matching incident starts its own run.
- `--oncall-incident-severities` stringSlice — Incident severities to watch: 'Critical' (most severe), 'Warning', or 'Info' (least severe) — the On-call incident severity levels. · enum: Critical | Warning | Info
- `--oncall-incident-trigger-enabled` bool — Whether the On-call incident trigger is enabled. When the trigger is created (this is 'true', or channel IDs or severities are sent), both 'oncall_incident_channel_ids' and 'oncall_incident_severities' must be non-empty.
- `--prompt` string (required) — Task instruction sent to the AI SRE agent on every run. Must not be empty after trimming spaces. (≥1 chars)
- `--run-in` string — Where the rule's runs execute. 'web' (default): in a web console session. 'current-chat': in the IM chat the rule is created from; accepted only from a live IM conversation with the AI SRE agent, so API calls that send it get '400'. · enum: web | current-chat
- `--schedule-trigger-enabled` bool — Whether the schedule trigger is enabled. Defaults to 'true' when omitted; send 'false' for rules that should fire only from other triggers.
- `--team-id` int64 — Scope of the rule. '0': a personal rule owned by the caller. '>0': a team rule; the caller must be a member of that team. Omitted: a personal rule for an app key; a rule created by the AI SRE agent inherits its session's scope. The scope cannot be changed after creation. (min 0)
- `--timezone` string — IANA time zone name used to evaluate 'cron_expr', e.g. 'Asia/Shanghai'. Ambiguous names such as 'Local', 'CST', 'UTC+8', and 'GMT+8' are rejected. When omitted: the caller's member time zone, then the account time zone, then 'Asia/Shanghai'.
- `--war-room-channel-ids` intSlice — Channel IDs whose war rooms start a run. Each ID must be positive.
- `--war-room-severities` stringSlice — Incident severities that start a run when their war room is created: 'Critical' (most severe), 'Warning', or 'Info' (least severe) — the On-call incident severity levels. · enum: Critical | Warning | Info
- `--war-room-trigger-enabled` bool — Whether the war room trigger is enabled. It fires when a war room (IM group chat) is created for an incident in one of 'war_room_channel_ids' with a severity in 'war_room_severities'; the agent then runs inside that group chat. When the trigger is created (this is 'true', or 'war_room_channel_ids' is sent), both lists must be non-empty.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); can_edit (boolean); created_at (string); cron_expr (string); enabled (boolean); environment_id (string); environment_kind (string); execution_location (object); execution_place (string); http_post_token (string); http_post_trigger_enabled (boolean); http_post_trigger_id (string); http_post_trigger_url (string); last_run (object); name (string); oncall_incident_channel_ids (array<integer>); oncall_incident_merge_similar (boolean); oncall_incident_severities (array<string>); oncall_incident_trigger_enabled (boolean); oncall_incident_trigger_id (string); owner_id (integer); prompt (string); rule_id (string); run_scope (string); schedule_next_fire_at_ms (string); schedule_trigger_enabled (boolean); schedule_trigger_id (string); team_id (integer); timezone (string); updated_at (string); war_room_channel_ids (array<integer>); war_room_severities (array<string>); war_room_trigger_enabled (boolean); war_room_trigger_id (string)

### automation-rule-delete <rule-id>
Delete Automation rule
- `<rule-id>` (positional, required) string — Rule ID, from the list returned by 'POST /safari/automation/rule/list'.

### automation-rule-get <rule-id>
Get Automation rule
- `<rule-id>` (positional, required) string — Rule ID, from the list returned by 'POST /safari/automation/rule/list'.
- response: same shape as `automation-rule-create` above

### automation-rule-list
List Automation rules
- `--enabled` bool — Filter by enabled state: 'true' returns only enabled rules, 'false' only disabled; omit or pass null for no filter.
- `--include-person` bool — Legacy switch. Only read when 'scope' is empty: 'false' behaves like 'scope=team'; otherwise ignored.
- `--keyword` string — Returns rules whose name contains this text. At most 64 characters. (≤64 chars)
- `--limit` int64 — Page size. Values below 1 fall back to 20; values above 200 are capped at 200. (max 200)
- `--page` int64 — Page number, 1-based.
- `--scope` string — Scope filter: 'all' (default) your personal rules plus rules of your teams; 'personal' only your personal rules; 'team' only rules of your teams. · enum: all | personal | team
- `--search-after-ctx` string
- `--team-ids` intSlice — Filter to these team IDs; this narrows results and does not expand access.
- response: single object (`data` unwrapped to the top level) — fields: rules (array<object>); total (integer)

### automation-rule-run <rule-id>
Run Automation rule
- `<rule-id>` (positional, required) string — Rule ID, from the list returned by 'POST /safari/automation/rule/list'.
- response: single object (`data` unwrapped to the top level) — fields: preflight (object); rule_id (string); run (object); trigger_kind (string)

### automation-rule-update <rule-id>
Update Automation rule
- `--cron-expr` string — New schedule, same format as on create: 4 fields ('hour day month weekday') or 5 fields ('minute hour day month weekday'), fixed-integer minute, no seconds, not both day-of-month and day-of-week. Omit to keep the current expression.
- `--enabled` bool — Whether the rule's automatic triggers are armed. Omit to keep the current value.
- `--environment-id` string — BYOC runner ID; required when the resulting 'environment_kind' is 'byoc', rejected otherwise.
- `--environment-kind` string — New runtime environment: '""' automatic selection (prefers an online BYOC runner, falls back to the cloud sandbox), 'cloud' platform-hosted cloud sandbox, 'byoc' the runner in 'environment_id'. Sending this without 'environment_id' clears the stored 'environment_id'. · enum: cloud | byoc
- `--http-post-trigger-enabled` bool — Enable or disable the HTTP POST trigger. 'true' on a rule without one creates it and returns its token in this response.
- `--name` string — New display name, at most 255 characters. An empty string after trimming is rejected. (≤255 chars)
- `--oncall-incident-channel-ids` intSlice — Replaces the watched channel IDs when non-empty; an empty or omitted list keeps the stored IDs.
- `--oncall-incident-severities` stringSlice — Replaces the watched severities when non-empty ('Critical' (most severe), 'Warning', or 'Info' (least severe) — the On-call incident severity levels); an empty or omitted list keeps the stored values. · enum: Critical | Warning | Info
- `--oncall-incident-trigger-enabled` bool — Enable or disable the On-call incident trigger. Creating it on a rule that has none also needs 'oncall_incident_channel_ids' and 'oncall_incident_severities'.
- `--prompt` string — New task instruction for the agent. Omit to keep the current one.
- `--rotate-http-post-trigger-token` bool — When 'true', issues a new HTTP POST trigger token and invalidates the old one. The trigger must exist and be enabled. The new token is returned only in this response.
- `<rule-id>` (positional, required) string — Target rule ID, from the list returned by 'POST /safari/automation/rule/list'.
- `--schedule-trigger-enabled` bool — Whether the schedule trigger is enabled. Omit to keep the current value.
- `--timezone` string — New IANA time zone for 'cron_expr'. An empty string stores 'UTC'. Omit or send 'null' to keep the current time zone.
- `--war-room-channel-ids` intSlice — Replaces the channel IDs whose war rooms start a run when non-empty; an empty or omitted list keeps the stored IDs.
- `--war-room-severities` stringSlice — Replaces the war room severities when non-empty ('Critical' (most severe), 'Warning', or 'Info' (least severe) — the On-call incident severity levels); an empty or omitted list keeps the stored values. · enum: Critical | Warning | Info
- `--war-room-trigger-enabled` bool — Enable or disable the war room trigger. Creating it on a rule that has none also needs 'war_room_channel_ids' and 'war_room_severities'. Omit to keep the current value.
- response: same shape as `automation-rule-create` above

### automation-run-list <rule-id>
List Automation runs
- `--limit` int64 — Page size. Values below 1 fall back to 20.
- `--page` int64 — Page number, 1-based.
- `<rule-id>` (positional, required) string — Target rule ID, from the list returned by 'POST /safari/automation/rule/list'.
- `--search-after-ctx` string
- `--started-after-ms` int64 — Start-time lower bound, Unix milliseconds. Values below the 180-day run-history retention floor are clamped to it (that floor is also the default when omitted). (min 0)
- `--started-before-ms` int64 — Start-time upper bound, Unix milliseconds. Must be greater than or equal to the effective 'started_after_ms'; a value below the retention floor yields an empty result. (min 0)
- `--status` string — Run status filter: 'queued', 'running', 'retrying', 'succeeded', 'partial' (partially succeeded), 'failed', 'skipped' (e.g. rule or trigger no longer valid), 'abandoned' (stale run terminated by the system), 'blocked' (terminal; produced output but a connector is waiting on a human authorization); omit for no filter. · enum: queued | running | retrying | succeeded | partial | failed | skipped | abandoned | blocked
- `--trigger-kind` string — Trigger filter: 'schedule' schedule trigger, 'debug' debug run, 'manual' manual run, 'http_post' HTTP POST trigger, 'oncall_incident' On-call incident trigger, 'war_room_created' war room trigger; omit for no filter. · enum: schedule | debug | manual | http_post | oncall_incident | war_room_created
- response: single object (`data` unwrapped to the top level) — fields: runs (array<object>); total (integer)

### automation-template-list
List Automation templates
- `--locale` string — Template language. A value containing 'zh' returns Chinese, one containing 'en' returns English, any other non-empty value returns Chinese. When omitted, the 'locale'/'lang' query parameter or the 'Accept-Language' header decides, defaulting to Chinese. (≤16 chars)
- response: single object (`data` unwrapped to the top level) — fields: templates (array<object>)

### automation-triggers-{trigger_id}-fire <trigger_id>
Fire an Automation HTTP POST trigger
- `--text` string
- `--token` string

### knowledge-file-delete
Delete knowledge file
- `--expected-version` int64 — Optimistic-concurrency guard. '0' or omitted: delete unconditionally. A positive value: delete only if the knowledge's current 'version' equals it; otherwise nothing is deleted and the call returns HTTP 200 with 'status' = 'conflict' and the current 'version'. (min 0)
- `--force` bool — Delete even when other knowledge files reference this file; the referrers are then returned as warnings instead of blocking the delete.
- `--pack-id` string — Knowledge ID; defaults to the caller's account-scope knowledge.
- `--rel-path` string (required) — Path of the file relative to the knowledge root. Same path rules as 'POST /safari/knowledge/file/put'.
- response: single object (`data` unwrapped to the top level) — fields: message (string); status (string); version (integer); warnings (array<object>)

### knowledge-file-get
Get knowledge file
- `--pack-id` string — Knowledge ID; defaults to the caller's account-scope knowledge.
- `--rel-path` string (required) — Path of the file relative to the knowledge root. If no file exists at this path, the call still succeeds with 'file' set to 'null' and an empty 'content_b64'.
- response: single object (`data` unwrapped to the top level) — fields: content_b64 (string); file (any); version (integer)

### knowledge-file-list
List knowledge files
- `--limit` int64 — Page size. Accepted but currently ignored — the response always contains the full file list.
- `--pack-id` string — Knowledge ID; defaults to the caller's account-scope knowledge.
- `--page` int64 — Page number, 1-based. Accepted but currently ignored — the response always contains the full file list.
- `--search-after-ctx` string
- response: single object (`data` unwrapped to the top level) — fields: files (array<object>); total (integer); version (integer)

### knowledge-file-put
Upload knowledge file
- `--content-b64` string — Base64-encoded (standard alphabet) file content. The decoded bytes must be UTF-8 text with no NUL bytes and at most 1 MiB ('DUTY.md': at most 32 KiB). Omit or send an empty string to write an empty file.
- `--content-type` string — MIME type to store with the file. Optional: when omitted it is derived from the extension of 'rel_path' ('.md' → 'text/markdown', '.yaml'/'.yml' → 'text/yaml', '.json' → 'application/json', '.sh' → 'text/x-sh', otherwise 'text/plain'). Any value passed is stored as given.
- `--expected-version` int64 — Optimistic-concurrency guard. '0' or omitted: write unconditionally. A positive value: write only if the knowledge's current 'version' equals it; otherwise nothing is written and the call returns HTTP 200 with 'status' = 'conflict' and the current 'version'. (min 0)
- `--pack-id` string — Knowledge ID; defaults to the caller's account-scope knowledge.
- `--rel-path` string (required) — Destination path relative to the knowledge root; an existing file at this path is overwritten. Must use '/' separators, must not start with '/', must already be in normalized form, and no segment may be empty, '.', '..', or start with '.' (dotfiles are rejected).
- response: single object (`data` unwrapped to the top level) — fields: file (object); message (string); status (string); version (integer); warnings (array<object>)

### knowledge-get
Get account knowledge
- response: single object (`data` unwrapped to the top level) — fields: files (array<object>); pack (object)

### knowledge-pack-delete <pack-id>
Delete knowledge
- `<pack-id>` (positional, required) string — Knowledge ID to delete.
- response: single object (`data` unwrapped to the top level) — fields: ok (boolean)

### knowledge-pack-ensure
Ensure knowledge
- `--scope` string (required) — Scope of the knowledge to ensure. - 'account': the account knowledge. 'scope_id' is replaced by the caller's account ID; only the account owner and admins may call this, even when the knowledge already exists. If the knowledge has no 'DUTY.md', a default one is written. - 'team': the knowledge of the team in 'scope_id'. 'scope_id' is required and the caller must be a member of that team. · enum: account | team
- `--scope-id` int64 — Team ID. Required (non-zero) for 'team' scope, otherwise the request fails with 'InvalidParameter'; ignored for 'account' scope.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); can_edit (boolean); created_at_ms (string); created_by (integer); duty_version (integer); file_count (integer); pack_id (string); scope (string); scope_id (integer); team_name (string); total_bytes (integer); updated_at_ms (string); version (integer)

### knowledge-pack-list
List knowledge
- `--include-account` bool — Whether to include the account knowledge when 'scope' is 'all' or omitted. Defaults to 'true'.
- `--limit` int64 — Page size. Defaults to 20 when 'p' is set and 'limit' is omitted; no upper bound is enforced.
- `--page` int64 — Page number, 1-based. When both 'p' and 'limit' are omitted (or ≤ 0), all matching knowledge is returned without paging; when only 'limit' is set, 'p' defaults to 1.
- `--query` string — Case-insensitive substring filter over knowledge ID, scope, scope ID, account ID and team name. Account knowledge also matches '账户'/'账号' and team knowledge also matches '团队'. Leading and trailing spaces are ignored. (≤128 chars)
- `--scope` string — Which scopes to return: - 'all' (default): account knowledge (unless 'include_account' is 'false') plus the visible team knowledge. - 'account': only the account knowledge; 'include_account' and 'team_ids' are ignored. - 'team': only team knowledge, optionally narrowed by 'team_ids'; 'include_account' is ignored. · enum: all | account | team
- `--search-after-ctx` string
- `--team-ids` intSlice — Restrict team knowledge to these team IDs. IDs of teams the caller does not belong to are silently dropped — this applies to every caller, including owners and admins.
- response: single object (`data` unwrapped to the top level) — fields: packs (array<object>); total (integer)

### knowledge-pack-update <pack-id>
Update knowledge
- `<pack-id>` (positional, required) string — Knowledge ID to update.
- `--scope` string — Destination scope. Omit to leave the knowledge unchanged and get it back. - 'account': make it the account knowledge (only the account owner and admins). - 'team': move it to the team in 'scope_id' (the caller must be a member of that team). Any other value returns 'InvalidParameter'. If the destination already has knowledge, the request fails with 'ReferenceExist'. · enum: account | team
- `--scope-id` int64 — Destination team ID. Required (non-zero) when 'scope' is 'team'; ignored when 'scope' is 'account' (the account ID is used).
- response: same shape as `knowledge-pack-ensure` above

### mcp-server-create
Create MCP server
- `--allow-insecure-oauth-http` bool — Allow this server's OAuth token exchange over plaintext HTTP. Testing use only; defaults to false.
- `--allow-insecure-tls-skip-verify` bool — Skip TLS certificate verification when connecting to this server. Testing use only; defaults to false.
- `--args` stringSlice — Command arguments (stdio transport).
- `--auth-mode` string — Who supplies the credential: - 'shared' (default; an empty value behaves the same): one credential stored on the server ('env' / 'headers') is used for every caller. - 'per_user_secret': each user stores their own secret, sent in the header named by 'secret_schema.header_name'. - 'per_user_oauth': each user completes their own OAuth authorization; discovery and client registration run on first use. 'per_user_secret' requires 'secret_schema' with a non-empty 'header_name'; any other value returns 'InvalidParameter'. · enum: shared | per_user_secret | per_user_oauth
- `--call-timeout` int64 — Tool-call timeout in seconds. Omitted or '0' is stored as '0' and means the 60-second default.
- `--command` string — Executable command (stdio transport).
- `--connect-timeout` int64 — Connection timeout in seconds. Omitted or '0' is stored as '0' and means the 10-second default.
- `--description` string (required) — Server description. (1-1024 chars)
- `--environments` stringSlice — Execution environments this server is callable from: 'cloud' and/or BYOC runner environment IDs. Omitted or empty means all environments.
- `--oauth-metadata` string — JSON OAuth metadata; reserved for per_user_oauth.
- `--secret-schema` string — JSON secret schema; required when auth_mode=per_user_secret.
- `--server-name` string (required) — MCP server name: must start with a letter and contain only letters, digits, '-', or '_' ('@' is reserved); unique within its scope (account-wide or one team). (1-255 chars)
- `--source-template-name` string — Name of the marketplace template to install. When set, the template must exist, the server is created at account scope, and the connection fields must match the template except for its declared parameter values. Cannot be changed later.
- `--status` string — Initial status: 'enabled' (default) or 'disabled' (created but kept off). · enum: enabled | disabled
- `--team-id` int64 — Team scope: '0' = account-wide (account owner or admin only); '>0' = a team the caller belongs to. Ignored when 'source_template_name' is set — marketplace installs are always account-wide.
- `--transport` string (required) — Transport protocol: 'stdio' launches a local process via 'command'/'args'/'env', 'sse' / 'streamable-http' connects to a remote service via 'url'/'headers'. · enum: stdio | sse | streamable-http
- `--url` string — Server URL (sse / streamable-http transport).
- body-only (`--data`): env (object); headers (object)
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); ai_description (string); allow_insecure_oauth_http (boolean); allow_insecure_tls_skip_verify (boolean); args (array<string>); auth_mode (string); call_timeout (integer); can_edit (boolean); command (string); connect_timeout (integer); created_at (string); created_by (integer); description (string); env (object); environments (array<string>); headers (object); oauth_metadata (string); proxy_url (string); secret_schema (string); server_id (string); server_name (string); source_template_name (string); status (string); team_id (integer); test_error (string); tested_at (string); transport (string); updated_at (string); url (string)

### mcp-server-delete <server-id>
Delete MCP server
- `<server-id>` (positional, required) string — Target MCP server ID, from the list returned by 'POST /safari/mcp/server/list'.

### mcp-server-disable <server-id>
Disable MCP server
- `<server-id>` (positional, required) string — Target MCP server ID, from the list returned by 'POST /safari/mcp/server/list'.

### mcp-server-enable <server-id>
Enable MCP server
- `<server-id>` (positional, required) string — Target MCP server ID, from the list returned by 'POST /safari/mcp/server/list'.

### mcp-server-get <server-id>
Get MCP server detail
- `<server-id>` (positional, required) string — Target MCP server ID, from the list returned by 'POST /safari/mcp/server/list'.
- response: same shape as `mcp-server-create` above

### mcp-server-list
List MCP servers
- `--include-account` bool — Include account-scoped ('team_id=0') rows. Defaults to 'true'; ignored when 'scope' is 'account' (always included) or 'team' (never included).
- `--limit` int64 — Page size; values below 1 fall back to 20. No upper bound is enforced.
- `--page` int64 — Page number, 1-based; values below 1 are treated as 1.
- `--query` string — Case-insensitive substring search across name, description, AI-generated description, server ID, transport, URL, command, and source template name. (≤128 chars)
- `--scope` string — Restrict results to a scope: 'account' for account-wide rows only, 'team' for the caller's own visible team rows only, or omit (defaults to 'all') for both, subject to team_ids/include_account. · enum: all | account | team
- `--search-after-ctx` string
- `--team-ids` intSlice — Restrict to these team IDs. IDs of teams the caller does not belong to are ignored; empty means all of the caller's teams.
- response: single object (`data` unwrapped to the top level) — fields: servers (array<object>); total (integer)

### mcp-server-update <server-id>
Update MCP server
- `--allow-insecure-oauth-http` bool — Allow OAuth token exchange over plaintext HTTP. Omit to leave unchanged.
- `--allow-insecure-tls-skip-verify` bool — Skip TLS certificate verification. Omit to leave unchanged.
- `--args` stringSlice — Command arguments ('stdio' transport); replaces the whole list — pass '[]' to clear, omit to leave unchanged.
- `--auth-mode` string — New auth mode; omitted or empty leaves it unchanged. Who supplies the credential: - 'shared' (default; an empty value behaves the same): one credential stored on the server ('env' / 'headers') is used for every caller. - 'per_user_secret': each user stores their own secret, sent in the header named by 'secret_schema.header_name'. - 'per_user_oauth': each user completes their own OAuth authorization; discovery and client registration run on first use. 'per_user_secret' requires 'secret_schema' with a non-empty 'header_name'; any other value returns 'InvalidParameter'. · enum: shared | per_user_secret | per_user_oauth
- `--call-timeout` int64 — New tool-call timeout in seconds; omitted or '0' leaves it unchanged (a stored '0' means the 60-second default).
- `--command` string — New executable command ('stdio' transport); omitted or empty leaves it unchanged.
- `--connect-timeout` int64 — New connection timeout in seconds; omitted or '0' leaves it unchanged (a stored '0' means the 10-second default).
- `--description` string — New description; omitted or empty leaves it unchanged. (1-1024 chars)
- `--environments` stringSlice — Execution environments this server is callable from: 'cloud' and/or BYOC runner environment IDs. Omit (null) to leave unchanged; send a list to set it — an empty list clears the restriction back to all environments.
- `--oauth-metadata` string — New JSON OAuth metadata. A non-empty value is stored as sent. When omitted or empty, the stored value is kept unless 'auth_mode' or 'url' changes, in which case it is cleared.
- `--secret-schema` string — New JSON secret schema, e.g. '{"header_name":"X-Api-Key"}'. Applied only together with a non-empty 'auth_mode'; sent alone it is ignored.
- `<server-id>` (positional, required) string — Target MCP server ID, from the list returned by 'POST /safari/mcp/server/list'.
- `--server-name` string — New name; omitted or empty leaves it unchanged. (1-255 chars)
- `--team-id` int64 — Reassign team scope: '0' = account-wide (account owner or admin only); '>0' = a team the caller belongs to. Servers installed from the marketplace can only move to '0'. Omit to leave unchanged.
- `--transport` string — Transport protocol; when switching, also supply the matching fields ('command'/'args'/'env' for 'stdio', 'url'/'headers' for 'sse' / 'streamable-http'); omitted or empty leaves it unchanged. · enum: stdio | sse | streamable-http
- `--url` string — New server URL ('sse' / 'streamable-http' transport); omitted or empty leaves it unchanged. A real change also clears 'oauth_metadata' unless a new one is sent.
- body-only (`--data`): env (object); headers (object)
- response: same shape as `mcp-server-create` above

### send [message]
Send a message to an AI SRE session
- `--env-id` string
- `--env-kind` string
- `--file` stringArray
- `--message-file` string
- `--session-id` string
- `--team-id` int64

### session-delete <session-id>
Delete session
- `<session-id>` (positional, required) string — Target session ID, from the list returned by 'POST /safari/session/list'. (≥1 chars)

### session-export <session_id>
Stream a session's full event transcript as NDJSON
- `--include-subagents` bool

### session-get <session-id>
Get session detail
- `--limit` int64 — Page size for events; takes precedence over 'num_recent_events'. 0 uses the server default (100). (0-1000)
- `--num-recent-events` int64 — Legacy page size: number of most-recent events to return. Superseded by 'limit' when both are set; 0 uses the server default (100). (0-1000)
- `--search-after-ctx` string — Opaque keyset cursor from a previous response; pass it back to fetch the next older page. (≤4096 chars)
- `<session-id>` (positional, required) string — Target session ID, from the list returned by 'POST /safari/session/list'. (≥1 chars)
- `--share-token` string — Share token for accessing a session through its share link. Omit it for normal account-authorized access. (≤512 chars)
- response: single object (`data` unwrapped to the top level) — fields: events (array<object>); has_more_older (boolean); pending_messages (array<object>); search_after_ctx (string); session (object)

### session-list
List sessions
- `--app-name` string (required) — Agent app whose sessions to list. One of: | Value | Meaning | | --- | --- | | 'ask-ai' | Ask AI assistant | | 'support' | Customer-support agent | | 'support-website' | Website support agent (exposed over A2A, not built into the console) | | 'support-flashcat' | Flashcat-site support agent (exposed over A2A) | | 'ai-sre' | The AI SRE main app | | 'template-assistant' | Notification-template assistant (template editing/validation) | | 'query-assistant' | Monitors query-writing assistant | | 'swe' | Software-engineering agent app (not offered in the console) | · enum: ask-ai | support | support-website | support-flashcat | ai-sre | template-assistant | query-assistant | swe
- `--asc` bool — Ascending order when true, descending when false. Only honored together with 'orderby'; when 'orderby' is omitted the sort is always 'updated_at' descending.
- `--entry-kinds` stringSlice — Restrict to sessions created from these surfaces: 'web' (console), 'im' (IM bot or IM H5), 'api' (public API), 'automation' (automation rule). Empty returns every kind. · enum: web | im | api | automation
- `--include-subagent-sessions` bool — When true, also list child sessions spawned by subagent dispatch. Defaults to false (top-level sessions only).
- `--keyword` string — Return only sessions whose 'session_name' contains this text; leading and trailing spaces are ignored. (≤64 chars)
- `--limit` int64 — Page size, 1–100. (1-100)
- `--orderby` string — Sort field: 'created_at' by creation time, 'updated_at' by last update; defaults to 'updated_at' when omitted. · enum: created_at | updated_at
- `--page` int64 — Page number, 1-based. (min 1)
- `--person-ids` intSlice — Filter by who started the session: returns only sessions started by these members (a session is kept when 'person_id' matches any of them). Intersects with 'scope' and 'team_ids', so it never widens what the caller is allowed to see.
- `--scope` string — Visibility scope: - 'all' (default): sessions the caller created plus sessions bound to teams the caller belongs to - 'personal': the caller's own sessions with no team ('team_id' = 0) - 'team': sessions bound to teams the caller belongs to; empty when the caller is in no team · enum: all | personal | team
- `--search-after-ctx` string
- `--status` string — Archive filter: - 'active' (default): sessions that are not archived - 'archived': archived sessions only - 'all': both · enum: active | archived | all
- `--team-ids` intSlice — Optional explicit team filter; intersects with 'scope' and never expands access.
- response: single object (`data` unwrapped to the top level) — fields: sessions (array<object>); total (integer)

### skill-delete <skill-id>
Delete skill
- `<skill-id>` (positional, required) string — Target skill ID, from the list returned by 'POST /safari/skill/list'.

### skill-disable <skill-id>
Disable skill
- `<skill-id>` (positional, required) string — Target skill ID, from the list returned by 'POST /safari/skill/list'.

### skill-enable <skill-id>
Enable skill
- `<skill-id>` (positional, required) string — Target skill ID, from the list returned by 'POST /safari/skill/list'.

### skill-get <skill-id>
Get skill detail
- `<skill-id>` (positional, required) string — Target skill ID, from the list returned by 'POST /safari/skill/list'.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); author (string); can_edit (boolean); checksum (string); content (string); created (boolean); created_at (string); created_by (integer); description (string); description_en (string); examples (array<object>); is_modified (boolean); license (string); s3_key (string); skill_id (string); skill_name (string); source_template_name (string); source_template_version (string); status (string); tags (array<string>); team_id (integer); tools (array<string>); update_available (boolean); updated_at (string); venues (array<string>); version (string)

### skill-list
List skills
- `--include-account` bool — Include account-scoped (team_id=0) rows. Defaults to true. Ignored when 'scope' is 'account' or 'team'.
- `--limit` int64 — Page size; values below 1 fall back to 20.
- `--page` int64 — Page number, 1-based; values below 1 fall back to 1.
- `--query` string — Free-text search across skill name, description, English description, skill ID, marketplace source template name, and author. (≤128 chars)
- `--scope` string — Which skills to return; overrides 'include_account' when set: - 'all' (default): account-scope skills (unless 'include_account' is false) plus the caller's team skills - 'account': account-scope skills only ('team_id' = 0) - 'team': the caller's team skills only · enum: all | account | team
- `--search-after-ctx` string
- `--team-ids` intSlice — Limit team-scope results to these team IDs; IDs of teams the caller does not belong to are dropped. Empty = all of the caller's teams.
- response: single object (`data` unwrapped to the top level) — fields: skills (array<object>); total (integer)

### skill-update <skill-id>
Update skill
- `--description` string — New description. Cannot contain '<' or '>'. Sending an empty string leaves the current value unchanged — there is no way to clear it via this field. (≤1024 chars)
- `--description-en` string — New English description. Cannot contain '<' or '>'. Omit to leave unchanged; send an empty string to explicitly clear it. (≤1024 chars)
- `<skill-id>` (positional, required) string — Target skill ID, from the list returned by 'POST /safari/skill/list'.
- `--team-id` int64 — Reassign team scope: 0 = account-wide; >0 = team. Omit to leave unchanged.
- response: same shape as `skill-get <skill-id>` above

### skill-upload --file <archive>
Upload a skill archive
- `--file` string
- `--replace` bool
- `--skill-id` string
- `--team-id` int64

### watch --session-id <id> [--root-invocation-id <id>] [--since <unix-ms>]
Stream an AI SRE session's replies as one JSON line each
- `--root-invocation-id` string
- `--session-id` string
- `--since` int64
- `--since-event-id` string

<!-- GENERATED:safari END -->

## Key concepts

- **Transport ⇒ which fields matter.** `stdio` uses `command` + `args` + `env`; `sse` / `streamable-http` use `url` + `headers`. The nested `env` / `headers` objects have no scalar flags — pass them through `--data '{...}'`; typed scalar flags (`--server-name`, `--url`, `--args`, …) override matching `--data` keys.
- **Scope (`team_id`).** `0` = account-wide (every team sees it); `>0` = that team only. Same field on every safari verb.
- **Auth mode (`auth_mode`).** `shared` (default) = one credential for everyone, stored on the server. `per_user_secret` = each user supplies a secret matching `secret_schema` (which must carry a `header_name`). `per_user_oauth` = each user authorizes the server via OAuth.
- **`per_user_oauth` needs no OAuth config up front.** Create it with an **empty `oauth_metadata`** — that is the normal, complete state, not a missing prerequisite. The runtime **auto-discovers the OAuth server and dynamically registers a client (DCR)** the first time a user authorizes; you do **not** collect `authorization_url` / `client_id` / `client_secret` / `scopes`. Only pass `oauth_metadata` as a rare fallback, when the endpoint advertises no discovery document.
- **Chat with the agent: `send` then `watch`.** `send` queues one message and prints `{session_id, invocation_id, root_invocation_id, delivery_mode}`; `watch --session-id <id> --root-invocation-id <root>` prints one JSON line per agent reply (`reply` with `purpose` `progress` or `result`), `settled`, or `error`, and exits 0 when the request is fully done. Run `watch` under a tool that turns each stdout line into a notification (e.g. Claude Code's Monitor). Files go up with `send --file` as attachments, not inlined text.

## Gotchas

- **`mcp-server-create` requires `server_name`, `description`, `transport`.** A `stdio` server also needs `command`; a remote (`sse` / `streamable-http`) server needs `url`.
- **`env` / `headers` go through `--data`** — there are no `--env` / `--header` scalar flags for them (`args` does have a repeatable `--args` flag).
- **Don't reach for a tool search to install or configure a server** — that only finds tools on already-connected servers. Registration and configuration are `mcp-server-create` / `mcp-server-update`.
- **`delete` is irreversible** — prefer `disable` to park a server / skill / agent without destroying it. `list` first to confirm the id.
