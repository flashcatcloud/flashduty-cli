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
| SLS project/logstore discovery | `datasource-sls-projects` / `datasource-sls-logstores` |

## Gotchas

- **Datasource name is not guessable.** A `can not find datasource` 400 means the name is wrong — re-run `datasource-list` and copy the exact `Name`. Never invent variants.
- **`datasource-info` (and the `datasource-create`/`datasource-update` responses) return credentials exactly as configured — nothing is masked.** The `payload` object includes whatever passwords, API keys, tokens, and similar fields were set, in the clear. Treat the response as sensitive: don't dump it into logs or chat, don't echo it back beyond what the task needs, and don't pass it on to another tool.

<!-- GENERATED:monit[datasource] START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### datasource-create
Create datasource
- `--address` string — Connection address. Required for every type except 'elasticsearch' with 'deployment: cloud'. Prometheus/Loki/VictoriaLogs: HTTP URL; MySQL/Oracle/Postgres/ClickHouse: 'host:port'; SLS: endpoint without the 'http(s)://' prefix; 'tencent_cls': must be 'cls.tencentcloudapi.com' or 'cls.internal.tencentcloudapi.com' (requires Monitors edge >= v0.66.0).
- `--edge-cluster-name` string (required) — Monitors edge cluster name responsible for evaluating rules using this datasource.
- `--enabled` bool — Whether the datasource is enabled for rule evaluation. When omitted on create, the datasource is created disabled ('false').
- `--id` int64 — Datasource ID. Required for update; omit for create.
- `--name` string (required) — Datasource display name. This is the name referenced as 'ds_name' in query and diagnose APIs.
- `--note` string — Optional description.
- `--type-ident` string (required) — Datasource type identifier. Allowed: 'prometheus', 'loki', 'mysql', 'oracle', 'postgres', 'clickhouse', 'elasticsearch', 'sls', 'tencent_cls', 'victorialogs'.
- body-only (`--data`): payload (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); address (string); edge_cluster_name (string); enabled (boolean); id (integer); name (string); note (string); payload (any); type_ident (string); updated_at (string)

### datasource-delete
Delete datasource
- `--id` int64 (required) — Numeric ID of the target resource; the exact meaning depends on the API being called (e.g. datasource ID, ruleset ID).

### datasource-info
Get datasource detail
- `--id` int64 (required) — Numeric ID of the target resource; the exact meaning depends on the API being called (e.g. datasource ID, ruleset ID).
- response: same shape as `datasource-create` above

### datasource-list
List datasources
- `--type` string — Filter by datasource type identifier. Omit to return all types. Allowed values: 'prometheus', 'loki', 'mysql', 'oracle', 'postgres', 'clickhouse', 'elasticsearch', 'sls', 'tencent_cls', 'victorialogs'.
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); address (string); edge_cluster_name (string); enabled (boolean); id (integer); name (string); note (string); payload (any); type_ident (string); updated_at (string)

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

### datasource-update
Update datasource
- `--address` string — Connection address. Required for every type except 'elasticsearch' with 'deployment: cloud'. Prometheus/Loki/VictoriaLogs: HTTP URL; MySQL/Oracle/Postgres/ClickHouse: 'host:port'; SLS: endpoint without the 'http(s)://' prefix; 'tencent_cls': must be 'cls.tencentcloudapi.com' or 'cls.internal.tencentcloudapi.com' (requires Monitors edge >= v0.66.0).
- `--edge-cluster-name` string (required) — Monitors edge cluster name responsible for evaluating rules using this datasource.
- `--enabled` bool — Whether the datasource is enabled for rule evaluation. When omitted on create, the datasource is created disabled ('false').
- `--id` int64 — Datasource ID. Required for update; omit for create.
- `--name` string (required) — Datasource display name. This is the name referenced as 'ds_name' in query and diagnose APIs.
- `--note` string — Optional description.
- `--type-ident` string (required) — Datasource type identifier. Allowed: 'prometheus', 'loki', 'mysql', 'oracle', 'postgres', 'clickhouse', 'elasticsearch', 'sls', 'tencent_cls', 'victorialogs'.
- body-only (`--data`): payload (object) (required)
- response: same shape as `datasource-create` above

<!-- GENERATED:monit[datasource] END -->
