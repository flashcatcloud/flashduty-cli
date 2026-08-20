# fduty monit — alert rules

Prereq: `SKILL.md` + `reference/monit.md` read. This is the largest Flashmonit surface: rule CRUD, the folder tree, counters, change history, and export/import.

## Route here when

"监控规则 / 告警规则 / 规则文件夹 / 规则导出" or "alert rule / rule folder / rule export / rule audit" → this card.

**Mutating:** `rule-create`, `rule-update`, `rule-update-fields`, `rule-move`, `rule-delete`, `rule-delete-batch`, `rule-import` — confirm before running. **`rule-delete-batch` is irreversible**; confirm IDs with `rule-list-basic` first.

## Intent → verb

| want | verb |
|---|---|
| list rules directly in ONE folder (needs a real folder-id) | `rule-list-basic` |
| count rules per top-level folder (subtree totals) | `rule-counter-status` |
| full rule config | `rule-info` |
| create / update a rule | `rule-create` / `rule-update` |
| delete one or many rules | `rule-delete` / `rule-delete-batch` |
| move rules to another folder | `rule-move` |
| toggle enabled/channels in bulk | `rule-update-fields` |
| rule trigger status by folder | `rule-status` / `rule-counter-status` |
| rule change history | `rule-audits` → detail via `rule-audit-detail` |
| export / import rules (backup/migrate) | `rule-export` / `rule-import` |
| what datasource types support rules | `rule-dstypes` |
| per-channel / per-node / total counters | `rule-counter-channel` / `rule-counter-node` / `rule-counter-total` |

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

## Key concepts

**Check types in `rule_configs`** — three independent checks per rule; enable one or more:
- `check_threshold` — fires when a PromQL value crosses `critical` / `warning` / `info` thresholds (string expressions).
- `check_anydata` — fires when the query returns any rows (useful for log-pattern rules).
- `check_nodata` — fires when the query returns no data (detect silent failures).

**Severity enum** (inside `check_*`): `Critical` · `Warning` · `Info` (capital first letter; lowercase is rejected).

**Query name** — `rule_configs.queries[].name` is a single letter (e.g. `A`, `B`). `R` is reserved — do not use it.

## Gotchas

- **`rule_configs` and nested arrays require `--data`.** The queries, thresholds, enabled_times, and labels objects cannot be expressed as flat flags — pass them as inline JSON via `--data '{"rule_configs":{...}}'`. Typed scalar flags (`--name`, `--enabled`, `--cron-pattern`, `--ds-type`) override matching `--data` keys.
- **`folder-id 0` is not a universal "all rules" sentinel.** If the API says "Folder not found", believe it. For global inventory use `rule-counter-status` / `rule-counter-node` first, then run `rule-list-basic` against real folder IDs only.
- **"全量规则 / full rules" means exported monitor alert-rule definitions.** The concrete verb is `rule-export --ids ...`, usually after `rule-list-basic` selected the IDs. It does not mean dumping incidents or alerts.
- **For rule counts, prefer the counter verbs over list pagination.** `rule-counter-status`, `rule-counter-node`, and `rule-counter-total` are the authoritative aggregation surfaces; do not infer counts by walking `rule-list-basic` pages.
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

<!-- GENERATED:monit[rule] START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

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

<!-- GENERATED:monit[rule] END -->
