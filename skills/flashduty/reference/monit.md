# fduty monit — command card

Prereq: `SKILL.md` read. **SKILL.md + this card = full competence on monitors — no `--help` needed.** Read verbs are free. Mutating verbs (`datasource-create/update/delete`, `rule-create/update/delete/delete-batch/import`, `rule-update-fields`, `rule-move`, `store-ruleset-create/update/delete`, `tools-invoke`) change state — confirm before running. `datasource-delete` and `rule-delete-batch` are **irreversible**.

## Route here when

"监控规则 / 告警规则 / 数据源 / PromQL查询 / 日志查询 / 诊断 / 监控目标 / 主机工具" or "alert rule / datasource / metric query / log pattern / diagnose / monitored host / tools catalog" → **monit**. NOT `incident` (that domain = the alert graph after rules fire). Key IDs: **rule ID (int)** from `rule-list-basic`; **datasource name (string)** — never guess, always discover via `datasource-list`.

## Intent → verb

| want | verb |
|---|---|
| list all datasources (by type) | `datasource-list` |
| datasource detail | `datasource-info` |
| create / update a datasource | `datasource-create` / `datasource-update` |
| delete a datasource | `datasource-delete` |
| SLS project/logstore discovery | `datasource-sls-projects` / `datasource-sls-logstores` |
| list rules directly in ONE folder (needs a real folder-id) | `rule-list-basic` |
| count rules per top-level folder (subtree totals) | `rule-counter-status` |
| full rule config | `rule-info` |
| create / update a rule | `rule-create` / `rule-update` |
| preview a query before saving it into a rule | `preview-sync` |
| delete one or many rules | `rule-delete` / `rule-delete-batch` |
| move rules to another folder | `rule-move` |
| toggle enabled/channels in bulk | `rule-update-fields` |
| rule trigger status by folder | `rule-status` / `rule-counter-status` |
| rule change history | `rule-audits` → detail via `rule-audit-detail` |
| export / import rules (backup/migrate) | `rule-export` / `rule-import` |
| what datasource types support rules | `rule-dstypes` |
| per-channel / per-node / total counters | `rule-counter-channel` / `rule-counter-node` / `rule-counter-total` |
| run ad-hoc PromQL / SQL / LogQL | `query-rows` |
| log-pattern / metric-trend RCA evidence | `query-diagnose` |
| list monitored hosts/targets | `targets` |
| what tools a target exposes | `tools-catalog` |
| run host/db diagnostic tools | `tools-invoke` |
| store ruleset CRUD | `store-ruleset-create/list/info/update/delete` |

## Hot flow — ad-hoc query + diagnose

```bash
# 1. discover the real datasource name — NEVER guess
fduty monit datasource-list --output-format toon
fduty monit datasource-list --type prometheus --output-format toon

# 2a. point-in-time query (PromQL/SQL/LogQL); ALL time range goes INSIDE --expr
fduty monit query-rows --ds-type prometheus --ds-name <ds-name> \
  --expr 'rate(http_requests_total{job="api"}[5m])' --output-format toon

# 2b. log pattern RCA over last 15 min (time_range via --data; omit = last 15 min default)
fduty monit query-diagnose --ds-type loki --ds-name <ds-name> \
  --data '{"input":{"query":"{app=\"payment\"} |= \"error\""}}'

# 2c. metric trend analysis with explicit window
fduty monit query-diagnose --ds-type prometheus --ds-name <ds-name> \
  --data '{"input":{"query":"rate(http_errors_total[5m])"},"time_range":{"start":1718780000,"end":1718783600}}'
```

## Hot flow — host diagnostics

```bash
# 1. find the target locator (prefix search; --keyword is prefix-only)
fduty monit targets --keyword prod-web --output-format toon

# 2. discover what tools the target exposes
fduty monit tools-catalog --target-locator <hostname-or-ip> --output-format toon

# 3. invoke tools (up to 8 concurrently); use heredoc to avoid shell quoting hell
fduty monit tools-invoke --target-locator <hostname-or-ip> --output-format toon --data - <<'EOF'
{"tools":[{"tool":"os.overview"},{"tool":"os.top_processes","params":{"top_n":10}}]}
EOF
```

## Hot flow — enumerate configured rules (and its hard limit)

`rule-list-basic --folder-id <id>` lists only the rules **directly in that folder**, NOT its sub-folders; `--folder-id 0` or omitting it **400s "Folder not found"**. There is no "all rules" call, so enumeration means walking the folder tree:

```bash
# 1. top-level folders, each with its whole-subtree rule_total
fduty monit rule-counter-status --output-format toon
# 2. descend a folder to its DIRECT child folders (recurse until a folder has no children)
fduty monit rule-status --folder-id <folder-id> --output-format toon
# 3. list the rules sitting directly in each folder you reach
fduty monit rule-list-basic --folder-id <node-id> --output-format toon
```

**Hard limit — large accounts cannot be fully enumerated.** `rule-counter-status` / `rule-status` abort with 400 "too many rules" past a server cap (default 100 rules; "too many folders" past 500), and no account-wide rule list exists. When you hit that cap you **cannot** enumerate every configured rule from the CLI — say so plainly ("cannot fully enumerate configured rules on this account") instead of fabricating a completeness percentage.

**CONFIGURED ≠ FIRED.** Never infer rule coverage from *fired* alerts (`insight top-alerts`, alert feeds): "not fired in 90d" does **not** mean "not configured", and reporting a rule as missing on that basis is confidently wrong. Fired-alert queries answer "what is noisy", not "what is monitored".

<!-- GENERATED:monit START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### datasource-create
Create datasource
- `--address` string — Connection address. For Prometheus/Loki/VictoriaLogs: HTTP URL. For MySQL/Oracle/Postgres/ClickHouse: 'host:port'. For SLS: endpoint without http/https prefix. Not required for Elasticsearch cloud deployment.
- `--edge-cluster-name` string (required) — Monitors edge cluster name responsible for evaluating rules using this datasource.
- `--id` int64 — Datasource ID. Required for update; omit for create.
- `--name` string (required) — Datasource display name. This is the name referenced as 'ds_name' in query and diagnose APIs.
- `--note` string — Optional description.
- `--type-ident` string (required) — Datasource type identifier. Allowed: 'prometheus', 'loki', 'mysql', 'oracle', 'postgres', 'clickhouse', 'elasticsearch', 'sls', 'victorialogs'.
- body-only (`--data`): payload (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); address (string); edge_cluster_name (string); enabled (boolean); id (integer); name (string); note (string); payload (object); type_ident (string); updated_at (string)

### datasource-delete
Delete datasource
- `--id` int64 (required) — Numeric ID of the target resource; the exact meaning depends on the API being called (e.g. datasource ID, ruleset ID).

### datasource-info
Get datasource detail
- `--id` int64 (required) — Numeric ID of the target resource; the exact meaning depends on the API being called (e.g. datasource ID, ruleset ID).
- response: same shape as `datasource-create` above

### datasource-list
List datasources
- `--type` string — Filter by datasource type identifier. Omit to return all types. Allowed values: 'prometheus', 'loki', 'mysql', 'oracle', 'postgres', 'clickhouse', 'elasticsearch', 'sls', 'victorialogs'.
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); address (string); edge_cluster_name (string); enabled (boolean); id (integer); name (string); note (string); payload (object); type_ident (string); updated_at (string)

### datasource-sls-logstores
List SLS logstores
- `--id` int64 — ID of an SLS-type datasource. Obtainable via 'POST /monit/datasource/list'.
- `--offset` int64 — Pagination offset.
- `--project` string — SLS project name. Obtainable via 'POST /monit/datasource/sls/projects'.
- `--size` int64 — Page size. Defaults to 200 server-side when 0.

### datasource-sls-projects
List SLS projects
- `--id` int64 — ID of an SLS-type datasource. Obtainable via 'POST /monit/datasource/list'.
- `--offset` int64 — Pagination offset.
- `--query` string — Fuzzy filter on project description (maps to the 'description' parameter of Aliyun SLS ListProject). Leave empty to return all.
- `--size` int64 — Page size. Defaults to 200 server-side when 0.

### datasource-update
Update datasource
- `--address` string — Connection address. For Prometheus/Loki/VictoriaLogs: HTTP URL. For MySQL/Oracle/Postgres/ClickHouse: 'host:port'. For SLS: endpoint without http/https prefix. Not required for Elasticsearch cloud deployment.
- `--edge-cluster-name` string (required) — Monitors edge cluster name responsible for evaluating rules using this datasource.
- `--id` int64 — Datasource ID. Required for update; omit for create.
- `--name` string (required) — Datasource display name. This is the name referenced as 'ds_name' in query and diagnose APIs.
- `--note` string — Optional description.
- `--type-ident` string (required) — Datasource type identifier. Allowed: 'prometheus', 'loki', 'mysql', 'oracle', 'postgres', 'clickhouse', 'elasticsearch', 'sls', 'victorialogs'.
- body-only (`--data`): payload (object) (required)
- response: same shape as `datasource-create` above

### preview-sync
Preview datasource query
- `--delay-seconds` int64 — Shift the query window backward by this many seconds to compensate for data ingestion latency.
- `--ds-name` string (required) — Datasource display name as configured in the account.
- `--ds-type` string (required) — Datasource type, e.g. 'prometheus', 'loki', 'elasticsearch'.
- `--expr` string (required) — Query expression. Format depends on 'ds_type' (PromQL for Prometheus, LogQL for Loki, etc.).
- body-only (`--data`): args (object)

### query-diagnose
Diagnose data source
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied.
- `--ds-name` string (required) — Data source name configured under the tenant.
- `--ds-type` string (required) — Data source type. 'log_patterns' supports 'loki' and 'victorialogs'; 'metric_trends' supports 'prometheus'.
- `--operation` string — Diagnostic operation. When omitted, inferred from 'ds_type' (loki / victorialogs → 'log_patterns', prometheus → 'metric_trends'). Other sources must specify explicitly. · enum: log_patterns | metric_trends
- body-only (`--data`): input (object) (required); methods (array<object>); options (object); time_range (object)
- response: single object (`data` unwrapped to the top level) — fields: data_handling (object); ds_name (string); ds_type (string); operation (string); query (string); results (array<object>); schema_version (string); window (object)

### query-rows
Query data source rows
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied; mismatched values are rejected. Business execution always uses the authenticated account.
- `--delay-seconds` int64 — Look-back offset in seconds applied to point-in-time queries (Prometheus, Loki stats, VictoriaLogs stats). Ignored for raw / detail queries.
- `--ds-name` string (required) — Data source name; must match a configured data source under the tenant.
- `--ds-type` string (required) — Data source type; must match a configured data source under the tenant. Examples: 'prometheus', 'loki', 'victorialogs', 'sls', 'elasticsearch', 'mysql', 'postgres', 'oracle', 'clickhouse'.
- `--expr` string (required) — Query expression. Syntax depends on 'ds_type' and is interpreted by the corresponding monit-edge client (PromQL for Prometheus, LogQL for Loki, SQL for SQL sources, etc.).
- body-only (`--data`): args (object)
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: fields (object); values (object)

### rule-audit-detail
Get rule audit snapshot
- `--id` int64 (required) — Audit record ID — the 'id' of an audit row returned by 'POST /monit/rule/audits', NOT the rule ID. Passing a rule ID returns HTTP 400.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); action (string); alert_rule_id (integer); content (string); created_at (integer); creator_id (integer); creator_name (string); id (integer)

### rule-audits
List rule change history
- `--id` int64 (required) — Alert rule ID. Obtainable per folder via 'POST /monit/rule/list/basic'.
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); action (string); alert_rule_id (integer); content (string); created_at (integer); creator_id (integer); creator_name (string); id (integer)

### rule-counter-channel
Get rule counts by channel

### rule-counter-node
Get rule counts by folder node

### rule-counter-status
Get rule status counters for top-level folders
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: folder_id (integer); folder_name (string); rule_total (integer); triggered_rule_count (integer)

### rule-counter-total
Get rule counter time series
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); clock (string); id (integer); num (integer)

### rule-create
Create alert rule
- `--account-id` int64 — Account ID. Filled by the server from the authenticated identity; do not provide.
- `--channel-ids` intSlice — Channel IDs to send alerts to.
- `--created-at` string — Creation time as a Unix timestamp in seconds. Generated by the server; do not provide. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--creator-id` int64 — Creator user ID. Filled by the server from the current user; do not provide.
- `--creator-name` string — Creator name. Filled by the server; do not provide.
- `--cron-pattern` string — Schedule expression: a 6-field cron (with seconds) or an '@every 30s' interval descriptor. Must not start with 'CRON_TZ=' or 'TZ='; use the 'timezone' field instead.
- `--debug-log-enabled` bool — Whether to enable debug logging; the edge emits detailed evaluation logs, useful for troubleshooting rules that do not trigger as expected.
- `--delay-seconds` int64 — Seconds to shift the evaluation query window backward, compensating for data ingestion latency.
- `--description` string — Rule description, in Markdown.
- `--description-type` string — Format for the description. Defaults to 'text' when omitted or empty. · enum: text | markdown
- `--ds-ids` intSlice — Datasource IDs, merged with 'ds_list' to decide which datasources the rule monitors; IDs survive datasource renames. At least one of 'ds_list' and 'ds_ids' must be provided.
- `--ds-list` stringSlice — Data source name patterns (supports wildcards).
- `--ds-type` string — Datasource type identifier; allowed values are listed by 'POST /monit/rule/dstypes' (e.g. 'prometheus', 'elasticsearch').
- `--enabled` bool — Whether the rule is enabled. Updating to 'false' makes the server clean up the rule's active alerts.
- `--folder-id` int64 — ID of the folder the rule belongs to. Obtainable via 'POST /monit/folder/list'.
- `--id` int64 — Rule ID. Required for update; omit for create (assigned by the server).
- `--name` string — Rule name. Must be unique within the same folder.
- `--repeat-interval` int64 — Notification repeat interval in seconds.
- `--repeat-total` int64 — Max number of repeat notifications.
- `--timezone` string — Timezone in which the rule executes. Determines how the cron schedule and effective time windows are interpreted. Only IANA timezone names are accepted (e.g. 'Asia/Shanghai', 'UTC', 'Europe/London'); shortcuts and offsets such as 'Local', 'UTC+8', or 'CST' are rejected. Treated as 'Asia/Shanghai' if empty.
- `--updated-at` string — Last update time as a Unix timestamp in seconds. Generated by the server; do not provide. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--updater-id` int64 — Last updater user ID. Filled by the server; do not provide.
- `--updater-name` string — Last updater name. Filled by the server; do not provide.
- body-only (`--data`): annotations (object); enabled_times (array<object>); labels (object); rule_configs (object)
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); annotations (object); channel_ids (array<integer>); created_at (integer); creator_id (integer); creator_name (string); cron_pattern (string); debug_log_enabled (boolean); delay_seconds (integer); description (string); description_type (string); ds_ids (array<integer>); ds_list (array<string>); ds_type (string); enabled (boolean); enabled_times (array<object>); folder_id (integer); id (integer); labels (object); name (string); repeat_interval (integer); repeat_total (integer); rule_configs (object); timezone (string); updated_at (integer); updater_id (integer); updater_name (string)

### rule-delete
Delete alert rule
- `--id` int64 (required) — Alert rule ID. Obtainable per folder via 'POST /monit/rule/list/basic'.

### rule-delete-batch
Batch delete alert rules
- `--ids` intSlice (required) — Rule IDs.

### rule-dstypes
List available datasource types
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); id (integer); ident (string); name (string); weight (integer)

### rule-export
Export alert rules
- `--ids` intSlice (required) — Rule IDs.
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: annotations (object); cron_pattern (string); debug_log_enabled (boolean); delay_seconds (integer); description (string); description_type (string); ds_ids (array<integer>); ds_list (array<string>); ds_type (string); enabled (boolean); enabled_times (array<object>); labels (object); name (string); repeat_interval (integer); repeat_total (integer); rule_configs (object); timezone (string)

### rule-import
Import alert rules
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: message (string); name (string)

### rule-info
Get alert rule detail
- `--id` int64 (required) — Alert rule ID. Obtainable per folder via 'POST /monit/rule/list/basic'.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); annotations (object); channel_ids (array<integer>); created_at (string); creator_id (integer); creator_name (string); cron_pattern (string); debug_log_enabled (boolean); delay_seconds (integer); description (string); description_type (string); ds_ids (array<integer>); ds_list (array<string>); ds_type (string); enabled (boolean); enabled_times (array<object>); folder_id (integer); id (integer); labels (object); name (string); repeat_interval (integer); repeat_total (integer); rule_configs (object); timezone (string); updated_at (string); updater_id (integer); updater_name (string)

### rule-list-basic
List alert rules
- `--folder-id` int64 — Folder ID. 0 to list all accessible rules.
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); created_at (integer); creator_id (integer); creator_name (string); cron_pattern (string); debug_log_enabled (boolean); delay_seconds (integer); ds_type (string); enabled (boolean); folder_id (integer); id (integer); labels (object); name (string); timezone (string); triggered (boolean); updated_at (integer); updater_id (integer); updater_name (string)

### rule-move
Move alert rules to folder
- `--dest-folder-id` int64 (required) — Destination folder ID. Obtainable via 'POST /monit/folder/list'.
- `--ids` intSlice (required) — Rule IDs to move.
- response: same shape as `rule-import` above

### rule-status
Get rule trigger status under folder
- `--folder-id` int64 — Folder ID to summarize. Obtainable via 'POST /monit/folder/list'. Trigger statistics are returned grouped by direct child folder.
- response: same shape as `rule-counter-status` above

### rule-update
Update alert rule
- `--account-id` int64 — Account ID. Filled by the server from the authenticated identity; do not provide.
- `--channel-ids` intSlice — Channel IDs to send alerts to.
- `--created-at` string — Creation time as a Unix timestamp in seconds. Generated by the server; do not provide. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--creator-id` int64 — Creator user ID. Filled by the server from the current user; do not provide.
- `--creator-name` string — Creator name. Filled by the server; do not provide.
- `--cron-pattern` string — Schedule expression: a 6-field cron (with seconds) or an '@every 30s' interval descriptor. Must not start with 'CRON_TZ=' or 'TZ='; use the 'timezone' field instead.
- `--debug-log-enabled` bool — Whether to enable debug logging; the edge emits detailed evaluation logs, useful for troubleshooting rules that do not trigger as expected.
- `--delay-seconds` int64 — Seconds to shift the evaluation query window backward, compensating for data ingestion latency.
- `--description` string — Rule description, in Markdown.
- `--description-type` string — Format for the description. Defaults to 'text' when omitted or empty. · enum: text | markdown
- `--ds-ids` intSlice — Datasource IDs, merged with 'ds_list' to decide which datasources the rule monitors; IDs survive datasource renames. At least one of 'ds_list' and 'ds_ids' must be provided.
- `--ds-list` stringSlice — Data source name patterns (supports wildcards).
- `--ds-type` string — Datasource type identifier; allowed values are listed by 'POST /monit/rule/dstypes' (e.g. 'prometheus', 'elasticsearch').
- `--enabled` bool — Whether the rule is enabled. Updating to 'false' makes the server clean up the rule's active alerts.
- `--folder-id` int64 — ID of the folder the rule belongs to. Obtainable via 'POST /monit/folder/list'.
- `--id` int64 — Rule ID. Required for update; omit for create (assigned by the server).
- `--name` string — Rule name. Must be unique within the same folder.
- `--repeat-interval` int64 — Notification repeat interval in seconds.
- `--repeat-total` int64 — Max number of repeat notifications.
- `--timezone` string — Timezone in which the rule executes. Determines how the cron schedule and effective time windows are interpreted. Only IANA timezone names are accepted (e.g. 'Asia/Shanghai', 'UTC', 'Europe/London'); shortcuts and offsets such as 'Local', 'UTC+8', or 'CST' are rejected. Treated as 'Asia/Shanghai' if empty.
- `--updated-at` string — Last update time as a Unix timestamp in seconds. Generated by the server; do not provide. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--updater-id` int64 — Last updater user ID. Filled by the server; do not provide.
- `--updater-name` string — Last updater name. Filled by the server; do not provide.
- body-only (`--data`): annotations (object); enabled_times (array<object>); labels (object); rule_configs (object)
- response: same shape as `rule-create` above

### rule-update-fields
Batch update rule fields
- `--channel-ids` intSlice — IDs of the collaboration spaces alerts are sent to; may be empty. Effective only when 'fields' includes 'channel_ids'.
- `--cron-pattern` string — Schedule expression: a 6-field cron (with seconds) or an '@every 30s' interval descriptor; 'CRON_TZ='/'TZ=' prefixes are not allowed. Effective only when 'fields' includes 'cron_pattern'.
- `--debug-log-enabled` bool — Whether to enable debug logging; the edge emits detailed evaluation logs for troubleshooting. Effective only when 'fields' includes 'debug_log_enabled'.
- `--delay-seconds` int64 — Seconds to shift the evaluation query window backward, compensating for data ingestion latency. Effective only when 'fields' includes 'delay_seconds'.
- `--description` string — Rule description (Markdown). Effective only when 'fields' includes 'description'.
- `--ds-ids` intSlice — Datasource IDs, merged with 'ds_list' to decide which datasources the rule monitors; IDs survive datasource renames. Effective only when 'fields' includes 'ds_ids'.
- `--ds-list` stringSlice — Datasource name match patterns; wildcards supported. Effective only when 'fields' includes 'ds_list'.
- `--ds-type` string — Datasource type identifier; allowed values are listed by 'POST /monit/rule/dstypes'. Effective only when 'fields' includes 'ds_type'.
- `--enabled` bool — Whether the rule is enabled. Setting it to 'false' makes the server clean up the rule's active alerts. Effective only when 'fields' includes 'enabled'.
- `--fields` stringSlice (required) — Field names to update. Only listed fields are updated, taking new values from the same-named request fields; values for unlisted fields are silently ignored. · enum: labels | ds_type | ds_list | ds_ids | enabled | debug_log_enabled | cron_pattern | timezone | delay_seconds | enabled_times | annotations | description | channel_ids | repeat_interval | repeat_total
- `--ids` intSlice (required) — Rule IDs to update.
- `--repeat-interval` int64 — Interval in seconds between repeated alert notifications. Effective only when 'fields' includes 'repeat_interval'.
- `--repeat-total` int64 — Maximum number of repeated notifications. Effective only when 'fields' includes 'repeat_total'.
- `--timezone` string — Timezone in which the rule executes. IANA timezone name; defaults to 'Asia/Shanghai'.
- body-only (`--data`): annotations (object); enabled_times (array<object>); labels (object)
- response: same shape as `rule-import` above

### servicemap-fleet
Browse service map fleet hosts
- `--agent-versions` stringSlice — Filter to hosts on any of these exact agent versions. Up to 20 values.
- `--capture-modes` stringSlice — Filter to hosts using any of these capture modes. 'unknown' matches hosts that have not reported a capture mode yet. · enum: ebpf | polling | unknown
- `--cursor` string — Opaque pagination cursor. Pass back the exact value from a previous response's 'next_cursor'; omit for the first page.
- `--edge-clusters` stringSlice — Filter to hosts in any of these exact edge cluster names. Up to 20 values.
- `--limit` int64 — Maximum number of matching hosts to return in this page. Default 50, range 1-100. (1-100)
- `--scan-limit` int64 — Maximum number of candidate hosts to examine while filling this page. Default 1000, range 'limit'-2000. (max 2000)
- `--statuses` stringSlice — Filter to hosts currently in any of these statuses. Up to 20 values. · enum: active | degraded | stale | initializing | disabled | unsupported | no_data
- response: single object (`data` unwrapped to the top level) — fields: coverage (object); generated_at_ms (string); items (array<object>); next_cursor (string); partial (boolean); truncated (boolean); truncation_reasons (array<string>)

### servicemap-fleet-summary
Get service map fleet summary
- `--agent-versions` stringSlice — Filter to hosts on any of these exact agent versions. Up to 20 values.
- `--capture-modes` stringSlice — Filter to hosts using any of these capture modes. 'unknown' matches hosts that have not reported a capture mode yet. · enum: ebpf | polling | unknown
- `--edge-clusters` stringSlice — Filter to hosts in any of these exact edge cluster names. Up to 20 values.
- `--scan-limit` int64 — Maximum number of candidate hosts to scan. Default 2000, range 1-5000. (1-5000)
- response: single object (`data` unwrapped to the top level) — fields: coverage (object); generated_at_ms (string); partial (boolean); scan_limit (integer); truncated (boolean); truncation_reasons (array<string>)

### servicemap-status
Get service map status
- `--fleet` bool — When 'true', ignore 'host_id'/'host_ids' and instead sample up to 'limit' fleet candidate hosts for the account. Default 'false'.
- `--host-id` string — A single host ID to check. Combine with 'host_ids' to check several; mutually exclusive with 'fleet=true'. (≤128 chars)
- `--host-ids` stringSlice — Multiple host IDs to check in one call, up to 200 combined with 'host_id'. Mutually exclusive with 'fleet=true'.
- `--limit` int64 — In 'fleet' mode, the number of candidate hosts to sample. Ignored otherwise. Default 100, range 1-200. (1-200)
- response: single object (`data` unwrapped to the top level) — fields: coverage (object); fleet (boolean); generated_at_ms (string); items (array<object>); partial (boolean)

### servicemap-summary
Get service map summary
- `--network-scope-id` string — Optional integrity check: if set, must match the network scope already associated with 'anchor.host_id', or the request is rejected with 'InvalidParameter'.
- body-only (`--data`): anchor (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: anchor_entity_id (string); anchor_host_id (string); authoritative (boolean); context_ref_detail (string); coverage (object); freshness (object); graph_role (string); latest_collection_authoritative (boolean); latest_health_at_ms (string); neighbors (array<object>); network_scope_id (string); observed_at_ms (string); received_at_ms (string); resolution_counts (object); status (string); truncated (boolean); truncation_reasons (array<string>)

### servicemap-topology
Get service map topology
- `--at` string — Time selector for the query. Only 'now' is currently supported; omitting the field behaves the same. · enum: now
- `--depth` int64 — Maximum traversal depth from the anchor. Default 1, maximum 3. (max 3)
- `--direction` string — Traversal direction. Only 'outbound' is currently supported; omitting the field behaves the same. · enum: outbound
- `--include-metrics` bool — Whether to include the raw per-edge 'metrics' payload in the response. Default 'false'.
- `--max-edges` int64 — Maximum number of edges to examine before truncating. Default 200, maximum 1000. (max 1000)
- `--max-nodes` int64 — Maximum number of nodes to return before truncating. Default 100, maximum 500. (max 500)
- `--network-scope-id` string — Optional integrity check: if set, must match the network scope already associated with 'anchor.host_id', or the request is rejected with 'InvalidParameter'.
- `--unresolved-mode` string — How unresolved edges are projected. 'full' (default) includes them in 'edges' and 'unresolved_endpoints'; 'summary' omits them from 'edges' and returns only a bounded sample in 'unresolved_endpoints'. · enum: summary | full
- body-only (`--data`): anchor (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: anchor_entity_id (string); anchor_host_id (string); coverage (object); edges (array<object>); freshness (object); network_scope_id (string); nodes (array<object>); observed_at_ms (string); resolution_counts (object); truncated (boolean); truncation_reasons (array<string>); unresolved_endpoints (array<object>); unresolved_projection (object)

### store-ruleset-create
Create ruleset
- `--note` string (required) — Description or title of the ruleset.
- `--open-flag` int64 — Sharing scope. '0' = private (creator only), '1' = account-shared, '2' = public. Defaults to '0' if omitted.
- `--payload` string (required) — JSON string containing the alert rule definitions.
- `--type-ident` string (required) — Datasource type identifier this ruleset applies to, e.g. 'prometheus'.
- response: single object (`data` unwrapped to the top level) — fields: created_at (string); creator_account_id (integer); creator_id (integer); creator_name (string); id (integer); note (string); open_flag (integer); payload (string); type_ident (string); updated_at (string)

### store-ruleset-delete
Delete ruleset
- `--id` int64 (required) — Numeric ID of the target resource; the exact meaning depends on the API being called (e.g. datasource ID, ruleset ID).

### store-ruleset-info
Get ruleset detail
- `--id` int64 (required) — Numeric ID of the target resource; the exact meaning depends on the API being called (e.g. datasource ID, ruleset ID).
- response: same shape as `store-ruleset-create` above

### store-ruleset-list
List rulesets
- `--type-ident` string (required) — Datasource type identifier to filter by, e.g. 'prometheus'.
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: created_at (string); creator_account_id (integer); creator_id (integer); creator_name (string); id (integer); note (string); open_flag (integer); payload (string); type_ident (string); updated_at (string)

### store-ruleset-update
Update ruleset
- `--id` int64 (required) — Ruleset ID to update.
- `--note` string (required) — New description.
- `--open-flag` int64 — New sharing scope. '0' = private, '1' = account-shared, '2' = public.
- `--payload` string (required) — New JSON string of alert rule definitions.
- response: same shape as `store-ruleset-create` above

### targets
List monitored targets
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied.
- `--cursor` string — Opaque pagination cursor from the previous response's 'next_cursor'. Omit / pass empty string for the first page. Reset whenever 'keyword', 'limit', or tenant changes.
- `--keyword` string — Prefix match against 'target_locator'. ASCII only, no whitespace, no '|', max 256 bytes. Substring search is not supported.
- `--limit` int64 — Page size. Default 50, max 200. (max 200)
- response: `{items: [...], next_cursor, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: agent_version (string); cluster_name (string); edge_ipport (string); target_kind (string); target_locator (string); updated_at (string)

### tools-catalog
List target tool catalog
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied.
- `--target-kind` string — Optional target kind. When omitted, webapi infers it from current target routing. If the call returns 'ambiguous_target_kind', retry with a value from 'target_kinds'.
- `--target-locator` string (required) — Target identifier (host name, MySQL address, …). Max 256 bytes; no whitespace, control characters, or '|'.
- response: single object (`data` unwrapped to the top level) — fields: error (object); target (object); tools (array<object>)

### tools-invoke
Invoke target tools
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied.
- `--target-kind` string — Optional target kind; auto-inferred when omitted.
- `--target-locator` string (required) — Target identifier. Same validation rules as '/monit/tools/catalog'.
- body-only (`--data`): tools (array<object>) (required)
- response: single object (`data` unwrapped to the top level) — fields: error (object); results (array<object>); target (object)

<!-- GENERATED:monit END -->

## Key concepts

**Check types in `rule_configs`** — three independent checks per rule; enable one or more:
- `check_threshold` — fires when a PromQL value crosses `critical` / `warning` / `info` thresholds (string expressions).
- `check_anydata` — fires when the query returns any rows (useful for log-pattern rules).
- `check_nodata` — fires when the query returns no data (detect silent failures).

**Severity enum** (inside `check_*`): `Critical` · `Warning` · `Info` (capital first letter; lowercase is rejected).

**Query name** — `rule_configs.queries[].name` is a single letter (e.g. `A`, `B`). `R` is reserved — do not use it.

**`operation` on `query-diagnose`**: `log_patterns` (loki / victorialogs) or `metric_trends` (prometheus); inferred from `--ds-type` when omitted — only pass it explicitly for ambiguous source types.

**`query-diagnose` output**: results are versioned evidence, not the former summary-only pattern/series lists. Read `pattern_evidence` for logs or `series_evidence` for metrics; their optional comparison fields are absent when the edge has no evidence. Log output also includes `data_handling`, which declares redaction coverage and paths carrying untrusted observed data.

**`targets`**: `updated_at` means "last seen", not "online now".

## Gotchas

- **Datasource name is not guessable.** A `can not find datasource` 400 means the name is wrong — re-run `datasource-list` and copy the exact `Name`. Never invent variants.
- **`datasource-info` (and the `datasource-create`/`datasource-update` responses) return credentials exactly as configured — nothing is masked.** The `payload` object includes whatever passwords, API keys, tokens, and similar fields were set, in the clear. Treat the response as sensitive: don't dump it into logs or chat, don't echo it back beyond what the task needs, and don't pass it on to another tool.
- **`query-rows` has no time flags.** There is no `--time-start` / `--time-end` / `--operation`. Embed all time range and bucketing inside `--expr`. Passing those flags is a silent no-op or error.
- **`query-diagnose` time window via `--data`**, not flags. Pass `{"time_range":{"start":<unix>,"end":<unix>},...}`. Window wider than 6 hours is rejected server-side. Omitting `time_range` defaults to the last 15 minutes.
- **`rule_configs` and nested arrays require `--data`.** The queries, thresholds, enabled_times, and labels objects cannot be expressed as flat flags — pass them as inline JSON via `--data '{"rule_configs":{...}}'`. Typed scalar flags (`--name`, `--enabled`, `--cron-pattern`, `--ds-type`) override matching `--data` keys.
- **`folder-id 0` is not a universal "all rules" sentinel.** If the API says "Folder not found", believe it. For global inventory use `rule-counter-status` / `rule-counter-node` first, then run `rule-list-basic` against real folder IDs only.
- **"全量规则 / full rules" means exported monitor alert-rule definitions.** The concrete verb is `rule-export --ids ...`, usually after `rule-list-basic` selected the IDs. It does not mean dumping incidents or alerts.
- **For rule counts, prefer the counter verbs over list pagination.** `rule-counter-status`, `rule-counter-node`, and `rule-counter-total` are the authoritative aggregation surfaces; do not infer counts by walking `rule-list-basic` pages.
- **`tools-catalog` / `tools-invoke` `--target-locator` is required and not guessable.** If the user has not provided a host or IP, ask — do not invent one. Tool names in `invoke` must come from the `tools-catalog` response — never hallucinate them.
- **`rule-delete-batch` and `datasource-delete` are irreversible.** Confirm IDs with `rule-list-basic` / `datasource-info` first.
- **`rule-audit-detail --id` takes the audit record ID**, not the rule ID. Get audit record IDs from `rule-audits --id <rule-id>` first; passing the rule ID returns HTTP 400.
- **`rule-list-basic` and `rule-status` both need a REAL `--folder-id`; neither accepts `0`.** `--folder-id 0` / omitting it 400s "Folder not found" on either verb — the generated `--folder-id` help text below ("0 to list all accessible rules" on `rule-list-basic`, "0 for all" on `rule-status`) is a known SDK/OpenAPI bug on both; ignore it. `rule-list-basic` returns only that folder's *direct* rules; `rule-status` returns trigger counts for that folder and its descendants. Enumerate by walking the tree (`rule-counter-status` → `rule-status` → `rule-list-basic`); past the server cap the counters 400 "too many rules" and full enumeration isn't possible from the CLI — report that limit, never substitute fired alerts (see the enumerate hot flow).

## Worked example — inspect a firing rule then batch-disable it

```bash
# 1. find a folder with triggered rules (top-level folders + subtree counts)
fduty monit rule-counter-status --output-format toon
# 2. list the rules directly in a chosen folder (descend with rule-status if empty)
fduty monit rule-list-basic --folder-id <folder-id> --output-format toon
# look at triggered=true rows; note their ids

# 3. get full config of one rule
fduty monit rule-info --id <rule-id> --output-format toon

# 4. disable several rules at once without touching other fields
fduty monit rule-update-fields --ids <id1>,<id2> --fields enabled --enabled false
```
