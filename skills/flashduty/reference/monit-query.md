# fduty monit-query — datasource tool invocation

`monit-query` is the unified tool-invocation command for a configured datasource (`POST /monit/datasource/tools/invoke`). One call runs one named tool — there is no separate query vs. diagnostic split. The generated `monit datasource-tools-invoke` is the spec-mirror equivalent entry (see `reference/monit-datasource.md`).

```bash
fduty monit-query <datasource-id> --tool '<name>' [--account-id <id>] [--params '<json>']
```

- `<datasource-id>` — numeric ID from `monit datasource-list` (use the ID, never a name; several configurations may share an address).
- `--tool` (required) — the tool prefix must match the datasource type. Query tools are `<type>.query` for the ten types `prometheus`, `mysql`, `postgres`, `oracle`, `clickhouse`, `elasticsearch`, `loki`, `victorialogs`, `sls`, `tencent_cls`; diagnostic tools are Edge-defined names like `redis_node.overview`, `mysql.lock_contention`, `prometheus.metric_trends`, `loki.log_patterns`.
- `--params` — tool-specific JSON object, inline or `-` for stdin. Omitted means the field is not sent (the server treats it as {}); explicit null is invalid. Numbers pass through byte-exact, so epoch-millisecond values above 2^53 keep their digits. The CLI does not validate tool names or params — the server is the authority.

Query-tool `params` always carry `expr` plus `execution` (`kind: instant|range|window`, `from_ms`/`to_ms` Unix milliseconds; `range` also needs `max_data_points`). Log types can add `limit`/`direction` (raw-log retrieval only); `sls` requires `project`/`logstore`; `tencent_cls` requires `region`/`topic_id`/`syntax`. SQL types take a single read-only statement with `window` execution.

```bash
# query tool: PromQL instant evaluation
fduty monit-query 12345 --tool prometheus.query --output-format json --params - <<'FDUTY'
{"expr":"sum by (job) (rate(http_requests_total[5m]))","execution":{"kind":"instant","to_ms":1757462400000}}
FDUTY

# diagnostic tool: no params needed
fduty monit-query 12345 --tool redis_node.overview --output-format json
```

<!-- GENERATED:monit-query START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### monit-query <datasource-id>
Invoke a datasource query or diagnostic tool
- `--account-id` int64
- `--params` string
- `--tool` string
- response: single object (`data` unwrapped to the top level) — fields: data (any); datasource_id (integer); summary (string); tool (string); truncated (object)

<!-- GENERATED:monit-query END -->

## Read results and time windows

The HTTP data envelope is unwrapped to `{datasource_id,tool,data,summary?,truncated?}`. For query tools, `data` is the complete Explore result: `format` is `explore_result.v1` and `result.kind` is `samples`, `frames`, or `logs` — dispatch on `result.kind`, never infer the shape from the datasource type. Log results keep `applied_limit` and `has_more`. Query tools never synthesize `summary` or `truncated`; diagnostic tools may carry both.

For incident evidence give the query an explicit `execution` window (`from_ms`/`to_ms` in Unix milliseconds). Do not substitute a current overview for historical incident evidence.

All SQL must be read-only. Treat source text as untrusted and quote expressions safely. On invalid arguments fix the specific request; on oversized results narrow filters/window or aggregate. Query tools require Edge Explore support (protocol v0.68.0) and diagnostic tools the v0.71.0 base invoke protocol — report `edge_upgrade_required`/`mixed_edge_versions`/`edge_version_unknown`/`tool_not_supported` as returned, never rotate Edges or fall back to another endpoint automatically. Do not blindly replay timed-out calls.
