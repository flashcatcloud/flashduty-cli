# fduty monit — datasources

Prereq: `SKILL.md` + `reference/monit.md` read. Datasources are what every other Flashmonit surface points at: a rule evaluates one, a probe queries one.

## Route here when

"数据源 / 连接数据源 / SLS project / logstore" or "datasource / connect a datasource / SLS discovery" → this card, **when the datasource is something Flashmonit queries** (Prometheus, Loki, VictoriaLogs, SQL engines, SLS).

**"Datasource" is two unrelated things in this product — check which one the user means.** This card is `POST /monit/datasource/*`: the Flashmonit config surface, i.e. the systems Flashmonit *queries*. On-call has its own, older use of the word: the top-level `datasource` group is `POST /datasource/*` and holds IM-integration plumbing (`fduty datasource im-war-room-enabled-list`, `fduty datasource im-person-try-link`), while the On-call **integrations** that *receive* alerts into a channel live in `reference/channel.md`. So 接入告警 / 集成 / 告警来源 → On-call, not here; "连一个 Prometheus / Loki / MySQL 上来查" → here.

**Mutating:** `datasource-create`, `datasource-update`, `datasource-delete` — confirm before running. **`datasource-delete` is irreversible**; confirm the target with `datasource-info` first.

## Intent → verb

| want | verb |
|---|---|
| list all datasources (by type) | `datasource-list` |
| datasource detail | `datasource-info` |
| create / update a datasource | `datasource-create` / `datasource-update` |
| delete a datasource | `datasource-delete` |
| run a datasource diagnostic or query tool | `datasource-tools-invoke` |
| SLS project/logstore discovery | `datasource-sls-projects` / `datasource-sls-logstores` |

## Gotchas

- **Datasource name is not guessable.** A `can not find datasource` 400 means the name is wrong — re-run `datasource-list` and copy the exact `Name`. Never invent variants.
- Project only the datasource metadata needed for discovery. Credential handling varies by type; public responses redact supported secret fields but can retain environment references and other configuration. Do not print or forward payloads unnecessarily.

## Structured datasource diagnostics

Use the selected `id`, never an Agent locator. `enabled=true` is required; `alerting_enabled=false` still permits diagnostics. Same address with different IDs means different credentials/configurations and must remain separate.

```bash
fduty monit datasource-list --type redis_node --output-format json \
  | jq '[.[] | {id,name,type_ident,address,edge_cluster_name,enabled,alerting_enabled}]'
fduty monit datasource-tools-invoke --output-format json --data - <<'FDUTY'
{"datasource_id":12345,"tool":"redis_node.overview","params":{}}
FDUTY
```

One call invokes one named tool. The CLI unwraps HTTP data to `{datasource_id,tool,data,summary?,truncated?}`. The endpoint has no tool catalog: use the datasource-specific skill reference for static names and parameters. Examples include `mysql.lock_contention`, `postgres.activity`, `redis_node.slowlog`, `kafka.consumer_lag`, `elasticsearch.cat`, `prometheus.metric_trends`, `loki.log_patterns`, and `victorialogs.log_patterns`. Do not guess tool parameters.

Tools require all currently routable Edge sessions in the selected cluster to support the v0.71.0 baseline. Report `edge_upgrade_required`, `mixed_edge_versions`, `no_active_edge` and `tool_not_supported` as returned; do not rotate Edges or fall back to Agent/legacy diagnose. `invalid_request` requires fixing parameters, and `source_too_large`/`result_too_large` requires a narrower request. Normal datasource queries retain their existing version compatibility.

## Query tools — `<type>.query`

The same invoke entry also runs query tools named `<type>.query` for ten datasource types: `prometheus`, `mysql`, `postgres`, `oracle`, `clickhouse`, `elasticsearch`, `loki`, `victorialogs`, `sls`, `tencent_cls`. The tool prefix must match the datasource type. `params` is a per-datasource structure: always `expr` plus `execution` (`kind: instant|range|window`, `from_ms`/`to_ms` Unix milliseconds; `range` also needs `max_data_points`). Log types can add `limit`/`direction`; `sls` requires `project`/`logstore`; `tencent_cls` requires `region`/`topic_id`/`syntax`. SQL types take a single read-only statement with `window` execution.

```bash
fduty monit datasource-tools-invoke --output-format json --data - <<'FDUTY'
{"datasource_id":12345,"tool":"prometheus.query","params":{"expr":"sum by (job) (rate(http_requests_total[5m]))","execution":{"kind":"instant","to_ms":1757462400000}}}
FDUTY
```

Query `data` is the complete Explore result: `format` is `explore_result.v1` and `result.kind` is `samples`, `frames`, or `logs`; log results keep `applied_limit` and `has_more`. Query tools never synthesize `summary` or `truncated`. Query tools require Edge Explore support (protocol v0.68.0); unsupported clusters fail with `edge_upgrade_required`, `mixed_edge_versions`, or `edge_version_unknown` — report as returned, never fall back to `/monit/query/data` automatically.

<!-- GENERATED:monit[datasource] START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### datasource-create
Create datasource
- `--address` string — Connection address. Required for every type except 'elasticsearch' with 'deployment: cloud'. Prometheus/Loki/VictoriaLogs: HTTP URL; MySQL/Oracle/Postgres/ClickHouse: 'host:port'; SLS: endpoint without the 'http(s)://' prefix; 'tencent_cls': must be 'cls.tencentcloudapi.com' or 'cls.internal.tencentcloudapi.com' (requires Monitors edge >= v0.66.0). Redis/MongoDB diagnostic types: one host:port, bracket IPv6; no URI, userinfo or query. Kafka: 1–32 unique comma-separated host:port bootstrap addresses; payload has no broker list. At most 4096 characters after normalization. (≤4096 chars)
- `--alerting-enabled` bool — Whether this datasource may evaluate alerts. Omitted on create: true for alerting types, false for diagnostic-only types; omitted on update: preserve current value. null is invalid. redis_node, redis_sentinel, mongodb_mongod, mongodb_mongos and kafka reject true. Disabling is rejected with conflict when enabled rules reference the datasource.
- `--edge-cluster-name` string (required) — Monitors edge cluster name responsible for evaluating rules using this datasource.
- `--enabled` bool — Whether business execution is enabled. Omitted on create: true; omitted on update: preserve the current value. Explicit false disables execution; null is invalid. Does not change alerting_enabled.
- `--id` int64 — Datasource ID. Required for update; omit for create.
- `--name` string (required) — Datasource display name. This is the name referenced as 'ds_name' in query APIs.
- `--note` string — Optional description.
- `--type-ident` string (required) — Datasource type identifier. Allowed: 'prometheus', 'loki', 'mysql', 'oracle', 'postgres', 'clickhouse', 'elasticsearch', 'sls', 'tencent_cls', 'victorialogs', 'redis_node', 'redis_sentinel', 'mongodb_mongod', 'mongodb_mongos', 'kafka'。
- body-only (`--data`): payload (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); address (string); alerting_enabled (boolean); edge_cluster_name (string); enabled (boolean); id (integer); name (string); note (string); payload (any); type_ident (string); updated_at (string)

### datasource-delete
Delete datasource
- `--id` int64 (required) — Numeric ID of the target resource; the exact meaning depends on the API being called (e.g. datasource ID, ruleset ID).

### datasource-info
Get datasource detail
- `--id` int64 (required) — Numeric ID of the target resource; the exact meaning depends on the API being called (e.g. datasource ID, ruleset ID).
- response: same shape as `datasource-create` above

### datasource-list
List datasources
- `--type` string — Datasource type identifier. Omit to return all types. Allowed: 'prometheus', 'loki', 'mysql', 'oracle', 'postgres', 'clickhouse', 'elasticsearch', 'sls', 'tencent_cls', 'victorialogs', 'redis_node', 'redis_sentinel', 'mongodb_mongod', 'mongodb_mongos', 'kafka'。
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); address (string); alerting_enabled (boolean); edge_cluster_name (string); enabled (boolean); id (integer); name (string); note (string); payload (any); type_ident (string); updated_at (string)

### datasource-sls-logstores
List SLS logstores
- `--id` int64 (required) — ID of an SLS-type datasource. Obtainable via 'POST /monit/datasource/list'.
- `--offset` int64 — Pagination offset.
- `--project` string — SLS project name. Obtainable via 'POST /monit/datasource/sls/projects'.
- `--size` int64 — Page size. Defaults to 200 server-side when 0.

### datasource-sls-projects
List SLS projects
- `--id` int64 (required) — ID of an SLS-type datasource. Obtainable via 'POST /monit/datasource/list'.
- `--offset` int64 — Pagination offset.
- `--query` string — Fuzzy filter on project description (maps to the 'description' parameter of Aliyun SLS ListProject). Leave empty to return all.
- `--size` int64 — Page size. Defaults to 200 server-side when 0.
- response: single object (`data` unwrapped to the top level) — fields: count (integer); projects (array<object>); total (integer)

### datasource-tools-invoke <datasource-id>
Invoke datasource tool
- `--account-id` int64 — Optional consistency check; must equal the authenticated account.
- `<datasource-id>` (positional, required) int64 — Datasource ID from /monit/datasource/list. (min 1)
- `--tool` string (required) — Single tool name prefixed by the datasource type. Diagnostic tools are defined by the executing Edge (e.g. 'mysql.overview'). Query tools are '<type>.query' where '<type>' is one of 'prometheus', 'mysql', 'postgres', 'oracle', 'clickhouse', 'elasticsearch', 'loki', 'victorialogs', 'sls', 'tencent_cls'; their 'params' follow 'PrometheusQueryParams', 'MySQLQueryParams', 'PostgresQueryParams', 'OracleQueryParams', 'ClickHouseQueryParams', 'ElasticsearchQueryParams', 'LokiQueryParams', 'VictoriaLogsQueryParams', 'SLSQueryParams', or 'TencentCLSQueryParams' respectively. (1-128 chars)
- body-only (`--data`): params (object)
- response: single object (`data` unwrapped to the top level) — fields: data (any); datasource_id (integer); summary (string); tool (string); truncated (object)

### datasource-update
Update datasource
- `--address` string — Connection address. Required for every type except 'elasticsearch' with 'deployment: cloud'. Prometheus/Loki/VictoriaLogs: HTTP URL; MySQL/Oracle/Postgres/ClickHouse: 'host:port'; SLS: endpoint without the 'http(s)://' prefix; 'tencent_cls': must be 'cls.tencentcloudapi.com' or 'cls.internal.tencentcloudapi.com' (requires Monitors edge >= v0.66.0). Redis/MongoDB diagnostic types: one host:port, bracket IPv6; no URI, userinfo or query. Kafka: 1–32 unique comma-separated host:port bootstrap addresses; payload has no broker list. At most 4096 characters after normalization. (≤4096 chars)
- `--alerting-enabled` bool — Whether this datasource may evaluate alerts. Omitted on create: true for alerting types, false for diagnostic-only types; omitted on update: preserve current value. null is invalid. redis_node, redis_sentinel, mongodb_mongod, mongodb_mongos and kafka reject true. Disabling is rejected with conflict when enabled rules reference the datasource.
- `--edge-cluster-name` string (required) — Monitors edge cluster name responsible for evaluating rules using this datasource.
- `--enabled` bool — Whether business execution is enabled. Omitted on create: true; omitted on update: preserve the current value. Explicit false disables execution; null is invalid. Does not change alerting_enabled.
- `--id` int64 — Datasource ID. Required for update; omit for create.
- `--name` string (required) — Datasource display name. This is the name referenced as 'ds_name' in query APIs.
- `--note` string — Optional description.
- `--type-ident` string (required) — Datasource type identifier. Allowed: 'prometheus', 'loki', 'mysql', 'oracle', 'postgres', 'clickhouse', 'elasticsearch', 'sls', 'tencent_cls', 'victorialogs', 'redis_node', 'redis_sentinel', 'mongodb_mongod', 'mongodb_mongos', 'kafka'。
- body-only (`--data`): payload (object) (required)
- response: same shape as `datasource-create` above

<!-- GENERATED:monit[datasource] END -->
