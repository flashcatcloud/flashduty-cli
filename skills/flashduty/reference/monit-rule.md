# fduty monit — alert rules

Prereq: `SKILL.md` + `reference/monit.md` read. This is the largest Flashmonit surface: rule CRUD, the folder tree, counters, change history, and export/import.

## Route here when

"监控规则 / 告警规则 / 规则文件夹 / 规则导出" or "alert rule / rule folder / rule export / rule audit" → this card.

**Mutating:** `rule-v2-create`, `rule-v2-update`, `rule-update-fields`, `rule-move`, `rule-delete`, `rule-delete-batch`, `rule-import` — confirm before running. **`rule-delete-batch` is irreversible**; confirm IDs with `rule-list-basic` first.

## Intent → verb

| want | verb |
|---|---|
| list rules in ONE folder (needs a real folder-id; `--include-descendants` adds subtree id/name rows) | `rule-list-basic` |
| full rule config | `rule-v2-info` |
| create / update a rule | `rule-v2-create` / `rule-v2-update` |
| delete one or many rules | `rule-delete` / `rule-delete-batch` |
| move rules to another folder | `rule-move` |
| toggle enabled/channels in bulk | `rule-update-fields` |
| rule change history | `rule-audits` → detail via `rule-audit-detail` |
| export / import rules (backup/migrate) | `rule-export` / `rule-import` |
| per-channel counts / total counter time series | `rule-counter-channel` / `rule-counter-total` |

## Hot flow — inspect configured rules

Use `rule-list-basic --folder-id <id>` only when a real folder ID is available from the user or trusted context; it lists direct rules, not descendants. Add `--include-descendants` (optionally `--query` / `--limit`, capped at 100) to enumerate subtree rules as id/folder_id/name rows. Omitting the ID or passing `0` returns "Folder not found".

```bash
fduty monit rule-list-basic --folder-id <known-folder-id> --output-format toon
fduty monit rule-list-basic --folder-id <known-folder-id> --include-descendants --limit 100 --output-format toon
```

The public CLI cannot discover the folder tree itself. If folder IDs are unavailable, report that limit rather than guessing IDs or treating a partial rule list as complete.

**CONFIGURED ≠ FIRED.** Never infer rule coverage from *fired* alerts (`insight top-alerts`, alert feeds): "not fired in 90d" does **not** mean "not configured", and reporting a rule as missing on that basis is confidently wrong. Fired-alert queries answer "what is noisy", not "what is monitored".

## Key concepts

**Check types in `rule_configs`** — three independent checks per rule; enable one or more:
- `check_threshold` — fires when a PromQL value crosses `critical` / `warning` / `info` thresholds (string expressions).
- `check_anydata` — fires when the query returns any rows (useful for log-pattern rules).
- `check_nodata` — fires when the query returns no data (detect silent failures).

**Severity enum** (inside `check_*`): `Critical` · `Warning` · `Info` (capital first letter; lowercase is rejected).

**Query name** — `rule_configs.queries[].name` is a single letter (e.g. `A`, `B`). `R` is reserved — do not use it.

## Gotchas

- **`rule_configs` and nested arrays require `--data`.** The queries, thresholds, enabled_times, and labels objects cannot be expressed as flat flags — pass them as inline JSON via `--data '{"rule_configs":{...}}'` on `rule-v2-create` / `rule-v2-update`. Typed scalar flags (`--name`, `--enabled`, `--cron-pattern`, `--ds-type`) override matching `--data` keys.
- **`folder-id 0` is not a universal "all rules" sentinel.** If the API says "Folder not found", believe it. Run `rule-list-basic` only against real folder IDs you actually know, adding `--include-descendants` when subtree id/name rows are enough.
- **"全量规则 / full rules" means exported monitor alert-rule definitions.** The concrete verb is `rule-export --ids ...`, usually after `rule-list-basic` selected the IDs. It does not mean dumping incidents or alerts.
- **`rule-delete-batch` and `datasource-delete` are irreversible.** Confirm IDs with `rule-list-basic` / `datasource-info` first.
- **`rule-audit-detail --id` takes the audit record ID**, not the rule ID. Get audit record IDs from `rule-audits --id <rule-id>` first; passing the rule ID returns HTTP 400.
- **`rule-list-basic` needs a REAL `--folder-id`; it does not accept `0`.** The command returns only that folder's *direct* rules unless `--include-descendants` is set; never substitute fired alerts as configured-rule inventory.

## Worked example — inspect a firing rule then batch-disable it

```bash
# 1. list direct rules in a folder whose ID is separately known
fduty monit rule-list-basic --folder-id <folder-id> --output-format toon
# 2. enumerate subtree rules (id/folder_id/name only, capped at 100)
fduty monit rule-list-basic --folder-id <folder-id> --include-descendants --limit 100 --output-format toon

# 3. get full config of one rule
fduty monit rule-v2-info --id <rule-id> --output-format toon

# 4. disable several rules at once without touching other fields
fduty monit rule-update-fields --ids <id1>,<id2> --fields enabled --enabled false
```

<!-- GENERATED:monit[rule] START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### rule-audit-detail
Get rule audit snapshot
- `--id` int64 (required) — Audit record ID — the 'id' of an audit row returned by 'POST /monit/rule/audits', NOT the rule ID. Passing a rule ID returns HTTP 400.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); action (string); alert_rule_id (integer); content (string); created_at (string); creator_id (integer); creator_name (string); id (integer)

### rule-audits
List rule change history
- `--id` int64 (required) — Alert rule ID. Obtainable per folder via 'POST /monit/rule/list/basic'.
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); action (string); alert_rule_id (integer); content (string); created_at (string); creator_id (integer); creator_name (string); id (integer)

### rule-counter-channel
Get rule counts by channel

### rule-counter-total
Get rule counter time series
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); clock (string); id (integer); num (integer)

### rule-delete
Delete alert rule
- `--id` int64 (required) — Alert rule ID. Obtainable per folder via 'POST /monit/rule/list/basic'.

### rule-delete-batch
Batch delete alert rules
- `--ids` intSlice (required) — Rule IDs.

### rule-export
Export alert rules
- `--ids` intSlice (required) — Rule IDs.
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: annotations (object); cron_pattern (string); debug_log_enabled (boolean); delay_seconds (integer); description (string); description_type (string); ds_ids (array<integer>); ds_list (array<string>); ds_type (string); enabled (boolean); enabled_times (array<object>); labels (object); name (string); repeat_interval (integer); repeat_total (integer); rule_configs (object); timezone (string)

### rule-import
Import alert rules
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: message (string); name (string)

### rule-list-basic
List alert rules
- `--folder-id` int64 — Folder ID. Must be an existing folder; '0' is rejected with a 'folder_not_found' error.
- `--include-descendants` bool — Also include rules from all descendant folders. When 'true', each returned item carries only 'id', 'folder_id' and 'name'; combine with 'query' / 'limit' for rule-picker scenarios.
- `--limit` int64 — Max number of rules returned; only effective when 'include_descendants' is 'true'. Defaults to 50, capped at 100. (max 100)
- `--query` string — Rule name fuzzy filter; only effective when 'include_descendants' is 'true'.
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); active_alert_count (integer); created_at (string); creator_id (integer); creator_name (string); cron_pattern (string); debug_log_enabled (boolean); delay_seconds (integer); ds_type (string); enabled (boolean); folder_id (integer); id (integer); labels (object); name (string); runtime_state (string); timezone (string); triggered (boolean); updated_at (string); updater_id (integer); updater_name (string)

### rule-move
Move alert rules to folder
- `--dest-folder-id` int64 (required) — Destination folder ID. Obtainable via 'POST /monit/folder/list'.
- `--ids` intSlice (required) — Rule IDs to move.
- response: same shape as `rule-import` above

### rule-update-fields
Batch update rule fields
- `--channel-ids` intSlice — IDs of the collaboration spaces alerts are sent to; may be empty. Effective only when 'fields' includes 'channel_ids'.
- `--cron-pattern` string — Schedule expression: a 6-field cron (with seconds) or an '@every 30s' interval descriptor; 'CRON_TZ='/'TZ=' prefixes are not allowed. Effective only when 'fields' includes 'cron_pattern'.
- `--debug-log-enabled` bool — Whether to enable debug logging; the edge emits detailed evaluation logs for troubleshooting. Effective only when 'fields' includes 'debug_log_enabled'.
- `--delay-seconds` int64 — Seconds to shift the evaluation query window backward, compensating for data ingestion latency. Effective only when 'fields' includes 'delay_seconds'.
- `--description` string — Rule description (Markdown). Effective only when 'fields' includes 'description'.
- `--ds-ids` intSlice — Datasource IDs, merged with 'ds_list' to decide which datasources the rule monitors; IDs survive datasource renames. Effective only when 'fields' includes 'ds_ids'.
- `--ds-list` stringSlice — Datasource name match patterns; wildcards supported. Effective only when 'fields' includes 'ds_list'.
- `--ds-type` string — Datasource type identifier. Effective only when 'fields' includes 'ds_type'.
- `--enabled` bool — Whether the rule is enabled. Setting it to 'false' makes the server clean up the rule's active alerts. Effective only when 'fields' includes 'enabled'.
- `--fields` stringSlice (required) — Field names to update. Only listed fields are updated, taking new values from the same-named request fields; values for unlisted fields are silently ignored. · enum: labels | ds_type | ds_list | ds_ids | enabled | debug_log_enabled | cron_pattern | timezone | delay_seconds | enabled_times | annotations | description | channel_ids | repeat_interval | repeat_total
- `--ids` intSlice (required) — Rule IDs to update.
- `--repeat-interval` int64 — Interval in seconds between repeated alert notifications. Effective only when 'fields' includes 'repeat_interval'.
- `--repeat-total` int64 — Maximum number of repeated notifications. Effective only when 'fields' includes 'repeat_total'.
- `--timezone` string — Timezone in which the rule executes. IANA timezone name; defaults to 'Asia/Shanghai'.
- body-only (`--data`): annotations (object); annotations_patch (object); enabled_times (array<object>); labels (object); labels_patch (object)
- response: same shape as `rule-import` above

### rule-v2-create <folder-id>
Create alert rule (V2)
- `--account-id` int64 — Account ID, filled by the server from the authentication context; any client-supplied value is ignored.
- `--channel-ids` intSlice — Collaboration space IDs alerts are sent to. May be empty; alerts then route through the global integration.
- `--created-at` string — Creation time as a Unix timestamp in seconds, generated by the server; any client-supplied value is ignored. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--creator-id` int64 — Creator member ID, filled by the server from the current user; any client-supplied value is ignored.
- `--creator-name` string — Creator name, filled by the server; any client-supplied value is ignored.
- `--cron-pattern` string (required) — Schedule expression: a 6-field cron (with seconds) or an '@every 30s' interval. Must not start with 'CRON_TZ=' or 'TZ='; set the timezone in the 'timezone' field instead.
- `--debug-log-enabled` bool — Enable debug logging; the edge then emits detailed evaluation logs for this rule, useful when the rule does not trigger as expected.
- `--delay-seconds` int64 — Seconds the evaluation query window is shifted back, compensating for data ingestion latency.
- `--description` string — Rule description, Markdown format.
- `--description-type` string — Format of the description content. Empty or omitted defaults to 'text'. 'text' = plain text; 'markdown' = Markdown, rendered as such in alert details. · enum: text | markdown
- `--ds-ids` intSlice — Datasource ID list, merged with 'ds_list' to decide the monitored datasources; IDs survive datasource renames. At least one of 'ds_list' / 'ds_ids' must be provided.
- `--ds-list` stringSlice — Datasource name match patterns (wildcards supported). At least one of 'ds_list' / 'ds_ids' must be non-empty; both are merged to decide which datasources the rule monitors.
- `--ds-type` string (required) — Datasource type identifier (e.g. 'prometheus', 'elasticsearch').
- `--enabled` bool (required) — Whether the rule is enabled. Required — the server enforces an explicit value (including 'false') while decoding. Setting it to 'false' on update clears the rule's active alerts.
- `<folder-id>` (positional, required) int64 — ID of the folder the rule belongs to; list folders via 'POST /monit/folder/list'. Cannot be changed through the update API — use '/monit/rule/move' instead.
- `--id` int64 — Rule ID. Required on update; omit on create (assigned by the server).
- `--name` string (required) — Rule name. Must be unique within the folder and at most 128 characters.
- `--repeat-interval` int64 — Notification repeat interval in seconds. Values below 1 fall back to the default 3600.
- `--repeat-total` int64 — Maximum number of repeat notifications. Values below 1 fall back to the default 3.
- `--timezone` string — Timezone the rule runs in; it decides how the cron schedule and enabled time windows are interpreted. Only IANA names are accepted (e.g. 'Asia/Shanghai', 'UTC', 'Europe/London'); abbreviations or offsets like 'Local', 'UTC+8', 'CST' are rejected. Empty falls back to 'Asia/Shanghai'.
- `--updated-at` string — Last update time as a Unix timestamp in seconds, generated by the server; any client-supplied value is ignored. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--updater-id` int64 — ID of the member who last updated the rule, filled by the server; any client-supplied value is ignored.
- `--updater-name` string — Name of the member who last updated the rule, filled by the server; any client-supplied value is ignored.
- body-only (`--data`): annotations (object); enabled_times (array<object>); investigation_targets (array<object>); labels (object); rule_configs (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); annotations (object); channel_ids (array<integer>); created_at (integer); creator_id (integer); creator_name (string); cron_pattern (string); debug_log_enabled (boolean); delay_seconds (integer); description (string); description_type (string); ds_ids (array<integer>); ds_list (array<string>); ds_type (string); enabled (boolean); enabled_times (array<object>); folder_id (integer); id (integer); investigation_targets (array<object>); labels (object); name (string); repeat_interval (integer); repeat_total (integer); rule_configs (object); timezone (string); updated_at (integer); updater_id (integer); updater_name (string)

### rule-v2-info
Get alert rule detail (V2)
- `--id` int64 (required) — Alert rule ID. Obtainable per folder via 'POST /monit/rule/list/basic'.
- response: same shape as `rule-v2-create <folder-id>` above

### rule-v2-update <folder-id>
Update alert rule (V2)
- `--account-id` int64 — Account ID, filled by the server from the authentication context; any client-supplied value is ignored.
- `--channel-ids` intSlice — Collaboration space IDs alerts are sent to. May be empty; alerts then route through the global integration.
- `--created-at` string — Creation time as a Unix timestamp in seconds, generated by the server; any client-supplied value is ignored. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--creator-id` int64 — Creator member ID, filled by the server from the current user; any client-supplied value is ignored.
- `--creator-name` string — Creator name, filled by the server; any client-supplied value is ignored.
- `--cron-pattern` string (required) — Schedule expression: a 6-field cron (with seconds) or an '@every 30s' interval. Must not start with 'CRON_TZ=' or 'TZ='; set the timezone in the 'timezone' field instead.
- `--debug-log-enabled` bool — Enable debug logging; the edge then emits detailed evaluation logs for this rule, useful when the rule does not trigger as expected.
- `--delay-seconds` int64 — Seconds the evaluation query window is shifted back, compensating for data ingestion latency.
- `--description` string — Rule description, Markdown format.
- `--description-type` string — Format of the description content. Empty or omitted defaults to 'text'. 'text' = plain text; 'markdown' = Markdown, rendered as such in alert details. · enum: text | markdown
- `--ds-ids` intSlice — Datasource ID list, merged with 'ds_list' to decide the monitored datasources; IDs survive datasource renames. At least one of 'ds_list' / 'ds_ids' must be provided.
- `--ds-list` stringSlice — Datasource name match patterns (wildcards supported). At least one of 'ds_list' / 'ds_ids' must be non-empty; both are merged to decide which datasources the rule monitors.
- `--ds-type` string (required) — Datasource type identifier (e.g. 'prometheus', 'elasticsearch').
- `--enabled` bool (required) — Whether the rule is enabled. Required — the server enforces an explicit value (including 'false') while decoding. Setting it to 'false' on update clears the rule's active alerts.
- `<folder-id>` (positional, required) int64 — ID of the folder the rule belongs to; list folders via 'POST /monit/folder/list'. Cannot be changed through the update API — use '/monit/rule/move' instead.
- `--id` int64 — Rule ID. Required on update; omit on create (assigned by the server).
- `--name` string (required) — Rule name. Must be unique within the folder and at most 128 characters.
- `--repeat-interval` int64 — Notification repeat interval in seconds. Values below 1 fall back to the default 3600.
- `--repeat-total` int64 — Maximum number of repeat notifications. Values below 1 fall back to the default 3.
- `--timezone` string — Timezone the rule runs in; it decides how the cron schedule and enabled time windows are interpreted. Only IANA names are accepted (e.g. 'Asia/Shanghai', 'UTC', 'Europe/London'); abbreviations or offsets like 'Local', 'UTC+8', 'CST' are rejected. Empty falls back to 'Asia/Shanghai'.
- `--updated-at` string — Last update time as a Unix timestamp in seconds, generated by the server; any client-supplied value is ignored. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--updater-id` int64 — ID of the member who last updated the rule, filled by the server; any client-supplied value is ignored.
- `--updater-name` string — Name of the member who last updated the rule, filled by the server; any client-supplied value is ignored.
- body-only (`--data`): annotations (object); enabled_times (array<object>); investigation_targets (array<object>); labels (object); rule_configs (object) (required)
- response: same shape as `rule-v2-create <folder-id>` above

<!-- GENERATED:monit[rule] END -->
