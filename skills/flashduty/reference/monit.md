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
| list alert rules (all or by folder) | `rule-list-basic` |
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
| run ad-hoc PromQL / SQL / LogQL | `query-rows` |
| log pattern clustering / trend RCA | `query-diagnose` |
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
{"tools":[{"tool":"host.cpu","params":{}},{"tool":"host.mem","params":{}}]}
EOF
```

## Hot flow — inventory all monitor alert rules without guessing folders

```bash
# 1. get the distribution first; this is the cheap way to answer "how many rules are where / in what status"
fduty monit rule-counter-status --output-format toon

# 2. list rules only for the concrete folder you care about
fduty monit rule-list-basic --folder-id <folder-id> --output-format toon

# 3. export the exact rules you selected
fduty monit rule-export --ids <id1>,<id2>,<id3> --output-format toon
```

<!-- GENERATED:monit START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### datasource-create
Create datasource
- `--address` string — Connection address. For Prometheus/Loki/VictoriaLogs: HTTP URL. For MySQL/Oracle/Postgres/ClickHouse: 'host:port'. For SLS: endpoint without http/https prefix. Not required for Elasticsearch cloud deployment.
- `--edge-cluster-name` string (required) — Monitors edge cluster name responsible for evaluating rules using this datasource.
- `--id` int64 — Datasource ID. Required for update; omit for create.
- `--name` string (required) — Datasource display name.
- `--note` string — Optional description.
- `--type-ident` string (required) — Datasource type identifier. Allowed: 'prometheus', 'loki', 'mysql', 'oracle', 'postgres', 'clickhouse', 'elasticsearch', 'sls', 'victorialogs'.
- body-only (`--data`): payload (object) (required)

### datasource-delete
Delete datasource
- `--id` int64 (required) — Resource ID.

### datasource-info
Get datasource detail
- `--id` int64 (required) — Resource ID.

### datasource-list
List datasources
- `--type` string — Filter by datasource type identifier. Omit to return all types. Allowed values: 'prometheus', 'loki', 'mysql', 'oracle', 'postgres', 'clickhouse', 'elasticsearch', 'sls', 'victorialogs'.

### datasource-sls-logstores
List SLS logstores
- `--id` int64 — SLS datasource ID.
- `--offset` int64 — Pagination offset.
- `--project` string — SLS project name.
- `--size` int64 — Page size.

### datasource-sls-projects
List SLS projects
- `--id` int64 — SLS datasource ID.
- `--offset` int64 — Pagination offset.
- `--query` string — Name prefix filter.
- `--size` int64 — Page size.

### datasource-update
Update datasource
- `--address` string — Connection address. For Prometheus/Loki/VictoriaLogs: HTTP URL. For MySQL/Oracle/Postgres/ClickHouse: 'host:port'. For SLS: endpoint without http/https prefix. Not required for Elasticsearch cloud deployment.
- `--edge-cluster-name` string (required) — Monitors edge cluster name responsible for evaluating rules using this datasource.
- `--id` int64 — Datasource ID. Required for update; omit for create.
- `--name` string (required) — Datasource display name.
- `--note` string — Optional description.
- `--type-ident` string (required) — Datasource type identifier. Allowed: 'prometheus', 'loki', 'mysql', 'oracle', 'postgres', 'clickhouse', 'elasticsearch', 'sls', 'victorialogs'.
- body-only (`--data`): payload (object) (required)

### query-diagnose
Diagnose data source
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied.
- `--ds-name` string (required) — Data source name configured under the tenant.
- `--ds-type` string (required) — Data source type. 'log_patterns' supports 'loki' and 'victorialogs'; 'metric_trends' supports 'prometheus'.
- `--operation` string — Diagnostic operation. When omitted, inferred from 'ds_type' (loki / victorialogs → 'log_patterns', prometheus → 'metric_trends'). Other sources must specify explicitly. · enum: log_patterns | metric_trends
- body-only (`--data`): input (object) (required); methods (array<object>); options (object); time_range (object)

### query-rows
Query data source rows
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied; mismatched values are rejected. Business execution always uses the authenticated account.
- `--delay-seconds` int64 — Look-back offset in seconds applied to point-in-time queries (Prometheus, Loki stats, VictoriaLogs stats). Ignored for raw / detail queries.
- `--ds-name` string (required) — Data source name; must match a configured data source under the tenant.
- `--ds-type` string (required) — Data source type; must match a configured data source under the tenant. Examples: 'prometheus', 'loki', 'victorialogs', 'sls', 'elasticsearch', 'mysql', 'postgres', 'oracle', 'clickhouse'.
- `--expr` string (required) — Query expression. Syntax depends on 'ds_type' and is interpreted by the corresponding monit-edge client (PromQL for Prometheus, LogQL for Loki, SQL for SQL sources, etc.).
- body-only (`--data`): args (object)

### rule-audit-detail
Get rule audit snapshot
- `--id` int64 (required) — Audit record ID — the 'id' of an audit row returned by 'POST /monit/rule/audits', NOT the rule ID. Passing a rule ID returns HTTP 400.

### rule-audits
List rule change history
- `--id` int64 (required) — Rule ID.

### rule-counter-channel
Get rule counts by channel

### rule-counter-node
Get rule counts by folder node

### rule-counter-status
Get rule status counters for top-level folders

### rule-counter-total
Get rule counter time series

### rule-create
Create alert rule
- `--account-id` int64
- `--channel-ids` intSlice — Channel IDs to send alerts to.
- `--created-at` int64
- `--creator-id` int64
- `--creator-name` string
- `--cron-pattern` string — 5-field cron schedule.
- `--debug-log-enabled` bool
- `--delay-seconds` int64
- `--description` string
- `--description-type` string — enum: text | markdown
- `--ds-ids` intSlice — Specific data source IDs.
- `--ds-list` stringSlice — Data source name patterns (supports wildcards).
- `--ds-type` string — Data source type.
- `--enabled` bool
- `--folder-id` int64 — Folder the rule belongs to.
- `--id` int64
- `--name` string — Rule name.
- `--repeat-interval` int64 — Notification repeat interval in seconds.
- `--repeat-total` int64 — Max number of repeat notifications.
- `--updated-at` int64
- `--updater-id` int64
- `--updater-name` string
- body-only (`--data`): annotations (object); enabled_times (array<object>); labels (object); rule_configs (object)

### rule-delete
Delete alert rule
- `--id` int64 (required) — Rule ID.

### rule-delete-batch
Batch delete alert rules
- `--ids` intSlice (required) — Rule IDs.

### rule-dstypes
List available datasource types

### rule-export
Export alert rules
- `--ids` intSlice (required) — Rule IDs.

### rule-import
Import alert rules

### rule-info
Get alert rule detail
- `--id` int64 (required) — Rule ID.

### rule-list-basic
List alert rules
- `--folder-id` int64 — Folder ID. 0 to list all accessible rules.

### rule-move
Move alert rules to folder
- `--dest-folder-id` int64 (required) — Destination folder ID.
- `--ids` intSlice (required) — Rule IDs to move.

### rule-status
Get rule trigger status under folder
- `--folder-id` int64 — Folder ID. 0 for all.

### rule-update
Update alert rule
- `--account-id` int64
- `--channel-ids` intSlice — Channel IDs to send alerts to.
- `--created-at` int64
- `--creator-id` int64
- `--creator-name` string
- `--cron-pattern` string — 5-field cron schedule.
- `--debug-log-enabled` bool
- `--delay-seconds` int64
- `--description` string
- `--description-type` string — enum: text | markdown
- `--ds-ids` intSlice — Specific data source IDs.
- `--ds-list` stringSlice — Data source name patterns (supports wildcards).
- `--ds-type` string — Data source type.
- `--enabled` bool
- `--folder-id` int64 — Folder the rule belongs to.
- `--id` int64
- `--name` string — Rule name.
- `--repeat-interval` int64 — Notification repeat interval in seconds.
- `--repeat-total` int64 — Max number of repeat notifications.
- `--updated-at` int64
- `--updater-id` int64
- `--updater-name` string
- body-only (`--data`): annotations (object); enabled_times (array<object>); labels (object); rule_configs (object)

### rule-update-fields
Batch update rule fields
- `--channel-ids` intSlice
- `--cron-pattern` string
- `--debug-log-enabled` bool
- `--delay-seconds` int64
- `--description` string
- `--ds-ids` intSlice
- `--ds-list` stringSlice
- `--ds-type` string
- `--enabled` bool
- `--fields` stringSlice (required) — Field names to update.
- `--ids` intSlice (required) — Rule IDs to update.
- `--repeat-interval` int64
- `--repeat-total` int64
- body-only (`--data`): annotations (object); enabled_times (array<object>); labels (object)

### store-ruleset-create
Create ruleset
- `--note` string (required) — Description or title of the ruleset.
- `--open-flag` int64 — Sharing scope. '0' = private (creator only), '1' = account-shared, '2' = public. Defaults to '0' if omitted.
- `--payload` string (required) — JSON string containing the alert rule definitions.
- `--type-ident` string (required) — Datasource type identifier this ruleset applies to, e.g. 'prometheus'.

### store-ruleset-delete
Delete ruleset
- `--id` int64 (required) — Resource ID.

### store-ruleset-info
Get ruleset detail
- `--id` int64 (required) — Resource ID.

### store-ruleset-list
List rulesets
- `--type-ident` string (required) — Datasource type identifier to filter by, e.g. 'prometheus'.

### store-ruleset-update
Update ruleset
- `--id` int64 (required) — Ruleset ID to update.
- `--note` string (required) — New description.
- `--open-flag` int64 — New sharing scope. '0' = private, '1' = account-shared, '2' = public.
- `--payload` string (required) — New JSON string of alert rule definitions.

### targets
List monitored targets
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied.
- `--cursor` string — Opaque pagination cursor from the previous response's 'next_cursor'. Omit / pass empty string for the first page. Reset whenever 'keyword', 'limit', or tenant changes.
- `--keyword` string — Prefix match against 'target_locator'. ASCII only, no whitespace, no '|', max 256 bytes. Substring search is not supported.
- `--limit` int64 — Page size. Default 50, max 200. (max 200)

### tools-catalog
List target tool catalog
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied.
- `--include-output-shape` bool — When true, each tool entry includes its 'output_shape' JSON Schema. Defaults to false to keep responses small for LLM consumption.
- `--target-kind` string — Optional target kind. When omitted webapi auto-infers across currently known kinds. Built-in kinds: 'host', 'mysql'. Required on retry when the previous call returned 'ambiguous_target_kind'.
- `--target-locator` string (required) — Target identifier (host name, MySQL address, …). Max 256 bytes; no whitespace, control characters, or '|'.

### tools-invoke
Invoke target tools
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied.
- `--target-kind` string — Optional target kind; auto-inferred when omitted.
- `--target-locator` string (required) — Target identifier. Same validation rules as '/monit/tools/catalog'.
- body-only (`--data`): tools (array<object>) (required)

<!-- GENERATED:monit END -->

## Key concepts

**Check types in `rule_configs`** — three independent checks per rule; enable one or more:
- `check_threshold` — fires when a PromQL value crosses `critical` / `warning` / `info` thresholds (string expressions).
- `check_anydata` — fires when the query returns any rows (useful for log-pattern rules).
- `check_nodata` — fires when the query returns no data (detect silent failures).

**Severity enum** (inside `check_*`): `Critical` · `Warning` · `Info` (capital first letter; lowercase is rejected).

**Query name** — `rule_configs.queries[].name` is a single letter (e.g. `A`, `B`). `R` is reserved — do not use it.

**`operation` on `query-diagnose`**: `log_patterns` (loki / victorialogs) or `metric_trends` (prometheus); inferred from `--ds-type` when omitted — only pass it explicitly for ambiguous source types.

**`targets` response shape** — rows are under `items[]` (not `data[]`); pipe `jq '.items[]'`, not `jq '.[]'`. `updated_at` means "last seen", not "online now".

## Gotchas

- **Datasource name is not guessable.** A `can not find datasource` 400 means the name is wrong — re-run `datasource-list` and copy the exact `Name`. Never invent variants.
- **`query-rows` has no time flags.** There is no `--time-start` / `--time-end` / `--operation`. Embed all time range and bucketing inside `--expr`. Passing those flags is a silent no-op or error.
- **`query-diagnose` time window via `--data`**, not flags. Pass `{"time_range":{"start":<unix>,"end":<unix>},...}`. Window wider than 6 hours is rejected server-side. Omitting `time_range` defaults to the last 15 minutes.
- **`rule_configs` and nested arrays require `--data`.** The queries, thresholds, enabled_times, and labels objects cannot be expressed as flat flags — pass them as inline JSON via `--data '{"rule_configs":{...}}'`. Typed scalar flags (`--name`, `--enabled`, `--cron-pattern`, `--ds-type`) override matching `--data` keys.
- **`folder-id 0` is not a universal "all rules" sentinel.** If the API says "Folder not found", believe it. For global inventory use `rule-counter-status` / `rule-counter-node` first, then run `rule-list-basic` against real folder IDs only.
- **"全量规则 / full rules" means exported monitor alert-rule definitions.** The concrete verb is `rule-export --ids ...`, usually after `rule-list-basic` selected the IDs. It does not mean dumping incidents or alerts.
- **For rule counts, prefer the counter verbs over list pagination.** `rule-counter-status`, `rule-counter-node`, and `rule-counter-total` are the authoritative aggregation surfaces; do not infer counts by walking `rule-list-basic` pages.
- **`tools-catalog` / `tools-invoke` `--target-locator` is required and not guessable.** If the user has not provided a host or IP, ask — do not invent one. Tool names in `invoke` must come from the `tools-catalog` response — never hallucinate them.
- **`rule-delete-batch` and `datasource-delete` are irreversible.** Confirm IDs with `rule-list-basic` / `datasource-info` first.
- **`rule-audit-detail --id` takes the audit record ID**, not the rule ID. Get audit record IDs from `rule-audits --id <rule-id>` first; passing the rule ID returns HTTP 400.

## Worked example — inspect a firing rule then batch-disable it

```bash
# 1. inspect top-level distribution, then choose a real folder id
fduty monit rule-counter-status --output-format toon
fduty monit rule-list-basic --folder-id <folder-id> --output-format toon
# look at the rows you actually care about; note their ids

# 2. get full config of one rule
fduty monit rule-info --id <rule-id> --output-format toon

# 3. disable several rules at once without touching other fields
fduty monit rule-update-fields --ids <id1>,<id2> --fields enabled --enabled false
```
