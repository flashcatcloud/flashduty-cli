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
| run an Automation rule immediately, outside its schedule | `automation-rule-run <rule-id>` |
| manage Automation rules directly (create/update/delete/list/runs/…) | `automation-rule-*` / `automation-run-list` / `automation-template-list` — see **`reference/automation.md`** for the primary router |

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
- `--allow-insecure-oauth-http` bool — Allow non-loopback HTTP OAuth discovery/metadata endpoints for this agent instead of requiring HTTPS. Defaults to false.
- `--allow-insecure-tls-skip-verify` bool — Skip TLS certificate verification when connecting to this agent's endpoint (self-signed/private certs). Defaults to false.
- `--auth-mode` string — Authentication mode: 'shared' (default) shares one credential across all users; 'per_user_secret' requires 'secret_schema.header_name'; 'per_user_oauth' runs per-user OAuth.
- `--auth-type` string — Authentication type for reaching the remote agent: 'none' (default when omitted), 'api_key', or 'bearer'. · enum: none | api_key | bearer
- `--card-url` string (required) — URL of the remote agent card. Must be an absolute 'http' or 'https' URL with a non-empty host; reachability is enforced by the execution environment, not at creation time.
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
- `--include-account` bool — Include account-scoped (team_id=0) rows. Defaults to true.
- `--limit` int64 — Page size.
- `--offset` int64 — Pagination offset — number of rows to skip, starting from 0.
- `--query` string — Case-insensitive substring search across agent name, instructions, card URL, agent ID, and the resolved card name. (≤128 chars)
- `--scope` string — Visibility scope: 'all' (account-scope plus the caller's visible teams), 'account' (account-scope only), or 'team' (team-scoped rows across the caller's visible teams). · enum: all | account | team
- `--team-ids` intSlice — Filter to these team IDs; empty = the caller's visible set.
- response: `{items: [...], total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); agent_card_name (string); agent_card_skills (array<string>); agent_id (string); agent_name (string); allow_insecure_oauth_http (boolean); allow_insecure_tls_skip_verify (boolean); auth_config (object); auth_mode (string); auth_type (string); can_edit (boolean); card_resolve_timeout (integer); card_url (string); created_at (string); created_by (integer); environments (array<string>); instructions (string); oauth_metadata (string); secret_schema (string); status (string); streaming (boolean); task_timeout (integer); team_id (integer); updated_at (string)

### a2a-agent-update <agent-id>
Update A2A agent
- `<agent-id>` (positional, required) string — Target agent ID, from the list returned by 'POST /safari/a2a-agent/list'.
- `--agent-name` string — New display name. Omit to leave unchanged. (≤128 chars)
- `--allow-insecure-oauth-http` bool — Toggle non-loopback HTTP OAuth discovery for this agent. Omit to leave unchanged.
- `--allow-insecure-tls-skip-verify` bool — Toggle TLS certificate verification skipping for this agent. Omit to leave unchanged.
- `--auth-mode` string — New auth mode: shared, per_user_secret, or per_user_oauth. Changing it always rewrites secret_schema together with it.
- `--auth-type` string — New auth type: 'none', 'api_key', or 'bearer'. Omit to leave unchanged. · enum: none | api_key | bearer
- `--card-url` string — New card URL. Omit to leave unchanged.
- `--environments` stringSlice — Execution environments this agent is callable from: 'cloud' and/or BYOC runner environment IDs. Omit (null) to leave unchanged; send a list to set it — an empty list clears the restriction back to all environments.
- `--instructions` string — New instructions document (same contract as create: optional 'summary' frontmatter, non-empty body, at most 50 KiB). Omit to leave unchanged. A deprecated 'description' field is also accepted; if both are sent they must match. (≤51200 chars)
- `--oauth-metadata` string — New JSON OAuth metadata. If omitted while auth_mode changes, it is cleared to empty.
- `--secret-schema` string — New JSON secret schema.
- `--streaming` bool — Toggle streaming support. Omit to leave unchanged.
- `--team-id` int64 — Reassign team scope. Omit to leave unchanged. Reassigning requires rights on the destination team; if the team changes without also sending a new environment binding, the existing runner binding must remain selectable by the caller or the update is rejected.
- body-only (`--data`): auth_config (object)

### artifact-gallery-delete <artifact-id>
Remove artifact from gallery
- `<artifact-id>` (positional, required) string — Artifact ID ('art_' prefix). Also the key of the public-share link.

### artifact-gallery-file-state <file-id> [<id2>...]
Get file publish state
- `<file-ids>` (positional, required) stringSlice — Presented-file IDs ('pf_' prefix) to probe. At most 50 per call; duplicates and empty strings are ignored.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: artifact_id (string); file_id (string); gallery_path (string); title (string)

### artifact-gallery-get <artifact-id>
Get artifact detail
- `<artifact-id>` (positional, required) string — Artifact ID ('art_' prefix). Also the key of the public-share link.
- response: single object (`data` unwrapped to the top level) — fields: artifact_id (string); can_edit (boolean); content_type (string); created_at (string); creator_name (string); file_id (string); is_mine (boolean); name (string); person_id (integer); public_url (string); session_id (string); session_title (string); share_enabled (boolean); share_file_id (string); shared_at (string); shared_by (integer); size (integer); team_id (integer); team_name (string); title (string); updated_at (string)

### artifact-gallery-list
List artifacts
- `--asc` bool — Sort ascending when true, descending when false. Applies only when 'orderby' is set.
- `--limit` int64 — Page size. Defaults to 20; capped at 100. (max 100)
- `--orderby` string — Sort field: 'created_at' or 'updated_at'. Empty means 'updated_at' descending. · enum: created_at | updated_at
- `--page` int64 — Page number, 1-based. (min 1)
- `--query` string — Case-insensitive substring match on the artifact title.
- `--scope` string — Visibility scope. 'all' (default) lists the caller's own personal artifacts plus artifacts of every team the caller belongs to; 'personal' lists only the caller's own; 'team' lists only team-owned artifacts of the caller's teams. · enum: all | personal | team
- `--team-ids` intSlice — Restrict to artifacts owned by these team IDs, intersected with the caller's visibility — teams the caller does not belong to silently return nothing.
- response: `{items: [...], total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: artifact_id (string); can_edit (boolean); content_type (string); created_at (string); creator_name (string); file_id (string); is_mine (boolean); name (string); person_id (integer); public_url (string); session_id (string); session_title (string); share_enabled (boolean); share_file_id (string); shared_at (string); shared_by (integer); size (integer); team_id (integer); team_name (string); title (string); updated_at (string)

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
- `--team-id` int64 — Transfer target scope. '0' moves the artifact to personal scope (only the creator can manage it); a positive value moves it to a team the caller must belong to. Omit to leave unchanged.
- `--title` string — New title. Trimmed; must be non-empty when provided. Omit to leave unchanged.
- response: same shape as `artifact-gallery-get <artifact-id>` above

### artifact-sign <file-id>
Create signed file URLs
- `<file-id>` (positional, required) string — Presented-file ID ('pf_' prefix) to sign.
- `--share-token` string — Optional session share-link token. Needed only when the caller reaches the file through a shared session link rather than account membership.
- response: single object (`data` unwrapped to the top level) — fields: content_type (string); download_url (string); expires_in (integer); name (string); preview_url (string); size (integer)

### artifact-stream
Download or preview a file
- `--mode` string — 'download' (default) serves the file as an attachment; 'preview' serves it inline for browser display. Any other value falls back to 'download'. · enum: download | preview
- `--t` string (required) — Signed token issued by 'POST /safari/artifact/sign'. Bound to the calling account and person; valid for 5 minutes.

### automation-rule-create
Create Automation rule
- `--cron-expr` string (required) — Run cadence. Supports 4 fields ('hour day month weekday', minute defaults to 0) and 5 fields ('minute hour day month weekday'). The minute must be one fixed integer; 6-field seconds are not supported. A cron that sets both day-of-month and day-of-week is rejected. The create API currently requires this field even for HTTP-POST-only rules; send a valid cron and set 'schedule_trigger_enabled=false'.
- `--enabled` bool — Whether the rule is enabled after creation. Omitted API value is false; Chat/CLI create sends true by default unless the user asks for disabled.
- `--environment-id` string — BYOC Runner ID. Used only when 'environment_kind=byoc'.
- `--environment-kind` string — Runtime environment kind. Omit or send an empty value for automatic selection. One of: 'cloud' (platform-hosted cloud sandbox), 'byoc' (a self-hosted BYOC runner in the account, used with 'environment_id'); automatic selection prefers an online BYOC runner and falls back to the cloud sandbox. · enum: cloud | byoc
- `--http-post-trigger-enabled` bool — Whether to create and enable an HTTP POST trigger. When enabled, the response includes a one-time token.
- `--name` string (required) — Rule name. (1-255 chars)
- `--oncall-incident-channel-ids` intSlice — On-call integration IDs to watch. Creating or enabling this trigger requires at least one valid ID.
- `--oncall-incident-severities` stringSlice — Incident severities to watch. Supported values are Critical, Warning, and Info; creating or enabling this trigger requires at least one value. · enum: Critical | Warning | Info
- `--oncall-incident-trigger-enabled` bool — Whether the On-call incident trigger is enabled.
- `--prompt` string (required) — Task prompt sent to the AI SRE agent on each run. (≥1 chars)
- `--schedule-trigger-enabled` bool — Whether the schedule trigger is enabled. Defaults to true when omitted; HTTP-POST-only rules should send false.
- `--team-id` int64 — Scope team ID. 0 or omitted means a personal rule; >0 means a team in the account. Can be reassigned later via update (converting a team rule to personal is owner-only; moving into a team requires the caller to belong to it). (min 0)
- `--timezone` string — IANA timezone 'cron_expr' is evaluated in, e.g. 'Asia/Shanghai'. Must be a timezone name loadable by the server; an invalid value is rejected. Defaults to the caller's member timezone, then the account timezone, then the server default (Asia/Shanghai) when omitted.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); can_edit (boolean); created_at (string); cron_expr (string); enabled (boolean); environment_id (string); environment_kind (string); http_post_token (string); http_post_trigger_enabled (boolean); http_post_trigger_id (string); http_post_trigger_url (string); name (string); oncall_incident_channel_ids (array<integer>); oncall_incident_severities (array<string>); oncall_incident_trigger_enabled (boolean); oncall_incident_trigger_id (string); owner_id (integer); prompt (string); rule_id (string); run_scope (string); schedule_next_fire_at_ms (string); schedule_trigger_enabled (boolean); schedule_trigger_id (string); team_id (integer); timezone (string); updated_at (string)

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
- `--include-person` bool — Compatibility field; when scope is empty and this is false, behaves like team scope.
- `--keyword` string — Filter by name keyword. (≤64 chars)
- `--limit` int64 — Page size.
- `--page` int64 — Page number, 1-based.
- `--scope` string — Scope filter: 'all' (own personal + accessible team rules), 'personal', or 'team'; default 'all'. · enum: all | personal | team
- `--search-after-ctx` string
- `--team-ids` intSlice — Filter to these team IDs; this narrows results and does not expand access.
- response: single object (`data` unwrapped to the top level) — fields: rules (array<object>); total (integer)

### automation-rule-run <rule-id>
Run Automation rule
- `<rule-id>` (positional, required) string — Rule ID, from the list returned by 'POST /safari/automation/rule/list'.
- response: single object (`data` unwrapped to the top level) — fields: preflight (object); rule_id (string); run (object); trigger_kind (string)

### automation-rule-update <rule-id>
Update Automation rule
- `--cron-expr` string — Run cadence. Supports 4 fields ('hour day month weekday', minute defaults to 0) and 5 fields ('minute hour day month weekday'). The minute must be one fixed integer; 6-field seconds are not supported.
- `--enabled` bool — Whether the rule is enabled.
- `--environment-id` string — BYOC Runner ID.
- `--environment-kind` string — Runtime environment kind. Omit or send an empty value for automatic selection. · enum: cloud | byoc
- `--http-post-trigger-enabled` bool — Whether the HTTP POST trigger is enabled. Sending true creates one when missing.
- `--name` string — New rule name. (≤255 chars)
- `--oncall-incident-channel-ids` intSlice — On-call integration IDs to watch. Creating or enabling this trigger requires at least one valid ID.
- `--oncall-incident-severities` stringSlice — Incident severities to watch. Supported values are Critical, Warning, and Info; creating or enabling this trigger requires at least one value. · enum: Critical | Warning | Info
- `--oncall-incident-trigger-enabled` bool — Whether the On-call incident trigger is enabled.
- `--prompt` string — New task prompt.
- `--rotate-http-post-trigger-token` bool — Whether to rotate the HTTP POST trigger token. The new token is returned only in this response.
- `<rule-id>` (positional, required) string — Target rule ID, from the list returned by 'POST /safari/automation/rule/list'.
- `--schedule-trigger-enabled` bool — Whether the schedule trigger is enabled.
- `--team-id` int64 — Reassign the rule's scope: 0 converts to a personal rule (only the rule owner may convert a team rule); >0 moves it into a team the caller belongs to. Omit to leave unchanged. (min 0)
- response: same shape as `automation-rule-create` above

### automation-run-list <rule-id>
List Automation runs
- `--limit` int64 — Page size.
- `--page` int64 — Page number, 1-based.
- `<rule-id>` (positional, required) string — Target rule ID, from the list returned by 'POST /safari/automation/rule/list'.
- `--search-after-ctx` string
- `--started-after-ms` int64 — Start-time lower bound, Unix milliseconds. Values below the 180-day run-history retention floor are clamped to it (that floor is also the default when omitted). (min 0)
- `--started-before-ms` int64 — Start-time upper bound, Unix milliseconds. Must be greater than or equal to the effective 'started_after_ms'; a value below the retention floor yields an empty result. (min 0)
- `--status` string — Run status filter: 'queued', 'running', 'retrying', 'succeeded', 'partial' (partially succeeded), 'failed', 'skipped' (e.g. rule or trigger no longer valid), 'abandoned' (stale run terminated by the system), 'blocked' (terminal; produced output but a connector is waiting on a human authorization); omit for no filter. · enum: queued | running | retrying | succeeded | partial | failed | skipped | abandoned | blocked
- `--trigger-kind` string — Trigger source filter: 'schedule' cron trigger, 'debug' debug run, 'manual' manual run, 'http_post' HTTP POST trigger, 'oncall_incident' on-call incident trigger; omit for no filter. · enum: schedule | debug | manual | http_post | oncall_incident
- response: single object (`data` unwrapped to the top level) — fields: runs (array<object>); total (integer)

### automation-template-list
List Automation templates
- `--locale` string — Template locale such as zh-CN or en-US. Omit to detect from the request locale. (≤16 chars)
- response: single object (`data` unwrapped to the top level) — fields: templates (array<object>)

### automation-triggers-{trigger_id}-fire <trigger_id>
Fire an Automation HTTP POST trigger
- `--text` string
- `--token` string

### knowledge-file-delete
Delete knowledge file
- `--force` bool — Delete even when other pack files reference this file; the referrers are then returned as warnings instead of blocking the delete.
- `--pack-id` string — Knowledge pack ID; defaults to the caller's account-scope pack.
- `--rel-path` string (required) — Path of the file relative to the pack root.
- response: single object (`data` unwrapped to the top level) — fields: warnings (array<object>)

### knowledge-file-get
Get knowledge file
- `--pack-id` string — Knowledge pack ID; defaults to the caller's account-scope pack.
- `--rel-path` string (required) — Path of the file relative to the pack root.
- response: single object (`data` unwrapped to the top level) — fields: content_b64 (string); file (object)

### knowledge-file-list
List knowledge files
- `--limit` int64 — Page size. Accepted but currently ignored — the response always contains the full file list.
- `--pack-id` string — Knowledge pack ID; defaults to the caller's account-scope pack.
- `--page` int64 — Page number, 1-based. Accepted but currently ignored — the response always contains the full file list.
- `--search-after-ctx` string
- response: single object (`data` unwrapped to the top level) — fields: files (array<object>); total (integer)

### knowledge-file-put
Upload knowledge file
- `--content-b64` string — Base64-encoded file content; must decode to valid UTF-8 text (binary is rejected). Per-file limit 1 MiB.
- `--content-type` string — MIME type; inferred from the file extension when omitted.
- `--pack-id` string — Knowledge pack ID; defaults to the caller's account-scope pack.
- `--rel-path` string (required) — Destination path relative to the pack root; existing files are overwritten.
- response: single object (`data` unwrapped to the top level) — fields: file (object); warnings (array<object>)

### knowledge-get
Get account knowledge pack
- response: single object (`data` unwrapped to the top level) — fields: files (array<object>); pack (object)

### knowledge-pack-delete <pack-id>
Delete knowledge pack
- `<pack-id>` (positional, required) string — Knowledge pack ID to delete.
- response: single object (`data` unwrapped to the top level) — fields: ok (boolean)

### knowledge-pack-ensure
Ensure knowledge pack
- `--scope` string (required) — Scope of the pack to ensure. One of: 'account' (account-level pack; scope_id is forced to the caller's account ID and only account admins may create it; first creation seeds a default DUTY.md), 'team' (team-level pack; the 'scope_id' team ID is required and the caller must belong to that team). · enum: account | team
- `--scope-id` int64 — Team ID; required for 'team' scope, ignored for 'account' scope.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); can_edit (boolean); created_at_ms (string); created_by (integer); duty_version (integer); file_count (integer); pack_id (string); scope (string); scope_id (integer); team_name (string); total_bytes (integer); updated_at_ms (string); version (integer)

### knowledge-pack-list
List knowledge packs
- `--include-account` bool — Include the account-scope pack; defaults to true.
- `--limit` int64 — Page size.
- `--page` int64 — Page number, 1-based; returns all results when both 'p' and 'limit' are unset.
- `--query` string — Case-insensitive substring filter over pack ID, scope, scope ID/account ID, and team name. (≤128 chars)
- `--scope` string — Restrict to one scope; 'all' (default) overrides 'include_account'. One of: 'all' (account scope plus visible team scopes), 'account' (account-level packs only), 'team' (team-level packs only, can be combined with 'team_ids'). · enum: all | account | team
- `--search-after-ctx` string
- `--team-ids` intSlice — Restrict to these team IDs; for non-admins the list is intersected with their own teams.
- response: single object (`data` unwrapped to the top level) — fields: packs (array<object>); total (integer)

### knowledge-pack-update <pack-id>
Update knowledge pack
- `<pack-id>` (positional, required) string — Knowledge pack ID to update.
- `--scope` string — Destination scope; omit for a no-op that returns the current pack. · enum: account | team
- `--scope-id` int64 — Destination team ID; required when 'scope' is 'team', set automatically for 'account'.
- response: same shape as `knowledge-pack-ensure` above

### mcp-server-create
Create MCP server
- `--allow-insecure-oauth-http` bool — Allow this server's OAuth token exchange over plaintext HTTP. Testing use only; defaults to false.
- `--allow-insecure-tls-skip-verify` bool — Skip TLS certificate verification when connecting to this server. Testing use only; defaults to false.
- `--args` stringSlice — Command arguments (stdio transport).
- `--auth-mode` string — Authentication mode: shared (default), per_user_secret, or per_user_oauth.
- `--call-timeout` int64 — Tool-call timeout in seconds. 0 = default (60s).
- `--command` string — Executable command (stdio transport).
- `--connect-timeout` int64 — Connection timeout in seconds. 0 = default (10s).
- `--description` string (required) — Server description. (1-1024 chars)
- `--environments` stringSlice — Execution environments this server is callable from: 'cloud' and/or BYOC runner environment IDs. Omitted or empty means all environments.
- `--oauth-metadata` string — JSON OAuth metadata; reserved for per_user_oauth.
- `--secret-schema` string — JSON secret schema; required when auth_mode=per_user_secret.
- `--server-name` string (required) — MCP server name: must start with a letter and contain only letters, digits, '-', or '_' ('@' is reserved); unique within its scope (account-wide or one team), case-insensitive. (1-255 chars)
- `--source-template-name` string — Marketplace template name when created from a connector template.
- `--status` string — Initial status: 'enabled' (default) or 'disabled' (created but kept off). · enum: enabled | disabled
- `--team-id` int64 — Team scope: 0 = account-wide; >0 = team.
- `--transport` string (required) — Transport protocol: 'stdio' launches a local process via 'command'/'args'/'env', 'sse' / 'streamable-http' connects to a remote service via 'url'/'headers'. · enum: stdio | sse | streamable-http
- `--url` string — Server URL (sse / streamable-http transport).
- body-only (`--data`): env (object); headers (object)
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); ai_description (string); allow_insecure_oauth_http (boolean); allow_insecure_tls_skip_verify (boolean); args (array<string>); auth_mode (string); call_timeout (integer); can_edit (boolean); command (string); connect_timeout (integer); created_at (string); created_by (integer); description (string); env (object); environments (array<string>); headers (object); oauth_metadata (string); proxy_url (string); secret_schema (string); server_id (string); server_name (string); source_template_name (string); status (string); team_id (integer); transport (string); updated_at (string); url (string)

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
- `--include-account` bool — Include account-scoped (team_id=0) rows. Defaults to true.
- `--limit` int64 — Page size.
- `--page` int64 — Page number, 1-based.
- `--query` string — Case-insensitive substring search across name, description, AI-generated description, server ID, transport, URL, command, and source template name. (≤128 chars)
- `--scope` string — Restrict results to a scope: 'account' for account-wide rows only, 'team' for the caller's own visible team rows only, or omit (defaults to 'all') for both, subject to team_ids/include_account. · enum: all | account | team
- `--search-after-ctx` string
- `--team-ids` intSlice — Filter to these team IDs; empty = the caller's visible set.
- response: single object (`data` unwrapped to the top level) — fields: servers (array<object>); total (integer)

### mcp-server-update <server-id>
Update MCP server
- `--allow-insecure-oauth-http` bool — Allow OAuth token exchange over plaintext HTTP. Omit to leave unchanged.
- `--allow-insecure-tls-skip-verify` bool — Skip TLS certificate verification. Omit to leave unchanged.
- `--args` stringSlice — Command arguments ('stdio' transport); replaces the whole list — pass '[]' to clear, omit to leave unchanged.
- `--auth-mode` string — Authentication mode: shared (default), per_user_secret, or per_user_oauth.
- `--call-timeout` int64 — Tool-call timeout in seconds. 0 = default (60s).
- `--command` string — Executable command (stdio transport).
- `--connect-timeout` int64 — Connection timeout in seconds. 0 = default (10s).
- `--description` string — New description; omitted or empty leaves it unchanged. (1-1024 chars)
- `--environments` stringSlice — Execution environments this server is callable from: 'cloud' and/or BYOC runner environment IDs. Omit (null) to leave unchanged; send a list to set it — an empty list clears the restriction back to all environments.
- `--oauth-metadata` string — JSON OAuth metadata; reserved for per_user_oauth.
- `--secret-schema` string — JSON secret schema; required when auth_mode=per_user_secret.
- `<server-id>` (positional, required) string — Target MCP server ID, from the list returned by 'POST /safari/mcp/server/list'.
- `--server-name` string — New name; omitted or empty leaves it unchanged. (1-255 chars)
- `--team-id` int64 — Reassign team scope: 0 = account-wide; >0 = team. Omit to leave unchanged.
- `--transport` string — Transport protocol; when switching, also supply the matching fields ('command'/'args'/'env' for 'stdio', 'url'/'headers' for 'sse' / 'streamable-http'); omitted or empty leaves it unchanged. · enum: stdio | sse | streamable-http
- `--url` string — Server URL (sse / streamable-http transport).
- body-only (`--data`): env (object); headers (object)
- response: same shape as `mcp-server-create` above

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
- response: single object (`data` unwrapped to the top level) — fields: events (array<object>); has_more_older (boolean); search_after_ctx (string); session (object); suggest_init (boolean)

### session-list
List sessions
- `--app-name` string (required) — Agent app whose sessions to list. One of: | Value | Meaning | | --- | --- | | 'ask-ai' | Ask AI assistant | | 'support' | Customer-support agent | | 'support-website' | Website support agent (exposed over A2A, not built into the console) | | 'support-flashcat' | Flashcat-site support agent (exposed over A2A) | | 'ai-sre' | The AI SRE main app | | 'template-assistant' | Notification-template assistant (template editing/validation) | | 'swe' | Internal benchmarking app (not customer-facing) | · enum: ask-ai | support | support-website | support-flashcat | ai-sre | template-assistant | swe
- `--asc` bool — Ascending order when true, descending when false. Only honored together with 'orderby'; when 'orderby' is omitted the sort is always 'updated_at' descending.
- `--entry-kinds` stringSlice — Restrict to sessions produced by these surfaces; empty returns every kind. · enum: web | im | api | automation
- `--include-subagent-sessions` bool — Include subagent-dispatched sessions in the list.
- `--keyword` string — Filter by session-name keyword. (≤64 chars)
- `--limit` int64 — Page size, 1–100. (1-100)
- `--orderby` string — Sort field: 'created_at' by creation time, 'updated_at' by last update; defaults to 'updated_at' when omitted. · enum: created_at | updated_at
- `--page` int64 — Page number, 1-based. (min 1)
- `--scope` string — Visibility scope: 'all' (own personal + accessible team sessions), 'personal', or 'team'; default 'all'. · enum: all | personal | team
- `--search-after-ctx` string
- `--status` string — Archive bucket: active (default) returns un-archived, archived returns archived, all returns both. · enum: active | archived | all
- `--team-ids` intSlice — Optional explicit team filter; intersects with 'scope' and never expands access.
- response: single object (`data` unwrapped to the top level) — fields: sessions (array<object>); suggest_init (boolean); total (integer)

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
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); author (string); can_edit (boolean); checksum (string); content (string); created (boolean); created_at (string); created_by (integer); description (string); description_en (string); is_modified (boolean); license (string); s3_key (string); skill_id (string); skill_name (string); source_template_name (string); source_template_version (string); status (string); tags (array<string>); team_id (integer); tools (array<string>); update_available (boolean); updated_at (string); venues (array<string>); version (string)

### skill-list
List skills
- `--include-account` bool — Include account-scoped (team_id=0) rows. Defaults to true. Ignored when 'scope' is 'account' or 'team'.
- `--limit` int64 — Page size.
- `--page` int64 — Page number, 1-based.
- `--query` string — Free-text search across skill name, description, English description, skill ID, marketplace source template name, and author. (≤128 chars)
- `--scope` string — Restrict results to 'all' (default), 'account'-only (team_id=0), or 'team'-only (excludes account-scoped rows). Overrides 'include_account' when set. · enum: all | account | team
- `--search-after-ctx` string
- `--team-ids` intSlice — Filter to these team IDs; empty = the caller's visible set.
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
