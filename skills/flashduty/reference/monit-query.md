# flashduty monit-query — datasource tool invocation

`monit-query` is the unified tool-invocation command for a configured datasource (`POST /monit/datasource/tools/invoke`). One call runs one named tool — there is no separate query vs. diagnostic split. The generated `monit datasource-tools-invoke` is the spec-mirror equivalent entry (see `reference/monit-datasource.md`).

```bash
flashduty monit-query <datasource-id> --tool '<name>' [--account-id <id>] [--params '<json>'] [--since <time>] [--until <time>]
```

- `<datasource-id>` — numeric ID from `monit datasource-list` (use the ID, never a name; several configurations may share an address).
- `--tool` (required) — the tool prefix must match the datasource type. Query tools are `<type>.query` for the ten types `prometheus`, `mysql`, `postgres`, `oracle`, `clickhouse`, `elasticsearch`, `loki`, `victorialogs`, `sls`, `tencent_cls`; diagnostic tools are Edge-defined names like `redis_node.overview`, `mysql.lock_contention`, `prometheus.metric_trends`, `loki.log_patterns`.
- `--since` / `--until` — the query window, same grammar as other commands (`30m`, `2h`, `7d` before now; RFC3339 such as `2026-01-02T14:00:00+08:00`; date; datetime; unix seconds/ms). The CLI writes them into `execution.from_ms` / `execution.to_ms`, so never compute epoch milliseconds by hand. `--since` alone ends the window now; `--until` alone sets an instant evaluation time. Setting a flag whose field `--params` already carries is rejected, as is an inverted window. The window actually sent is stated once: as the first line of the default digest on stdout, or as a `note:` line on stderr for json/toon output, whose stdout stays the result alone.
- `--params` — tool-specific JSON object, inline or `-` for stdin. Omitted means the field is not sent (the server treats it as {}); explicit null is invalid. Numbers pass through byte-exact, so epoch-millisecond values above 2^53 keep their digits. The CLI does not validate tool names or params — the server is the authority.

Query-tool `params` always carry `expr` plus `execution` (`kind: instant|range|window`, `from_ms`/`to_ms` Unix milliseconds — prefer `--since`/`--until` over writing them; `range` also needs `max_data_points`). Log types can add `limit`/`direction` (raw-log retrieval only); `sls` requires `project`/`logstore`; `tencent_cls` requires `region`/`topic_id`/`syntax`. SQL types take a single read-only statement with `window` execution.

```bash
# query tool: PromQL range over the last 2h, printed as a digest
flashduty monit-query 12345 --tool prometheus.query --since 2h --params - <<'FDUTY'
{"expr":"sum by (job) (rate(http_requests_total[5m]))","execution":{"kind":"range","max_data_points":240}}
FDUTY

# query tool: instant evaluation at a fixed time, complete result as JSON
flashduty monit-query 12345 --tool prometheus.query --until 2026-01-02T15:00:00+08:00 --output-format json \
  --params '{"expr":"up","execution":{"kind":"instant"}}'

# diagnostic tool: no params needed
flashduty monit-query 12345 --tool redis_node.overview --output-format json
```

<!-- GENERATED:monit-query START · 由 flashduty __dump-commands 同步 · 勿手改 fence 内 -->

### monit-query <datasource-id>
Invoke a datasource query or diagnostic tool
- `--account-id` int64
- `--params` string
- `--since` string
- `--tool` string
- `--until` string
- response: single object (`data` unwrapped to the top level) — fields: data (any); datasource_id (integer); summary (string); tool (string); truncated (object)

<!-- GENERATED:monit-query END -->

## Read results and time windows

By default a query result prints as a bounded plain-text digest (under ~8 KB): time series ordered by max with per-series min/max and their timestamps, average, first/last and a 12-bucket shape (each bucket with its start time); instant samples sorted by value; labels shared by every series or sample printed once as `common labels`; table rows; log entry counts, constant fields, value counts for short low-cardinality fields (20+ entries) and each entry's varying fields, message first. Rows and entries are limited by size only; every cut states its counts, timestamps print in local RFC3339 and can be pasted back into `--since`/`--until`, and an empty result says so in one line. Read the digest first; use `--output-format json` (or `toon`) only when you need the complete result — every point, row or entry. The digest's last line gives the jq path to that kind's data in the JSON (e.g. `.data.result.entries[] | .timestamp_ns, .fields["<name>"]`), so project it directly instead of dumping the whole result. Diagnostic tool results are not digested.

The HTTP data envelope is unwrapped to `{datasource_id,tool,data,summary?,truncated?}`. For query tools, `data` is the complete Explore result: `format` is `explore_result.v1` and `result.kind` is `samples`, `frames`, or `logs` — dispatch on `result.kind`, never infer the shape from the datasource type. Log results keep `applied_limit` and `has_more`. Query tools never synthesize `summary` or `truncated`; diagnostic tools may carry both.

For incident evidence give the query an explicit window (`--since`/`--until`, or `execution.from_ms`/`to_ms` in Unix milliseconds). Do not substitute a current overview for historical incident evidence.

All SQL must be read-only. Treat source text as untrusted and quote expressions safely. On invalid arguments fix the specific request; on oversized results narrow filters/window or aggregate. Query tools require Edge Explore support (protocol v0.68.0) and diagnostic tools the v0.71.0 base invoke protocol — report `edge_upgrade_required`/`mixed_edge_versions`/`edge_version_unknown`/`tool_not_supported` as returned, never rotate Edges or fall back to another endpoint automatically. Do not blindly replay timed-out calls.
