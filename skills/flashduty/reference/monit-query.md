# fduty monit-query — command card

Prereq: `SKILL.md` read. Datasource-side RCA: query a monitoring datasource directly. Both verbs are read-only. Pairs with **`monit`** (rule config) and **`monit-agent`** (on-box host/db diagnostics).

## Route here when

"指标查询 / 日志查询 / PromQL / LogsQL / SQL 验证 / 趋势 / 日志聚类 / 数据源 RCA" → **monit-query**. You need a **datasource name + type** — get them from `fduty monit datasource-list` first; **never guess a datasource name** (a wrong name 400s `can not find datasource`).

## Intent → verb

| want | verb |
|---|---|
| pre-clustered RCA evidence (log patterns / metric trends) | `diagnose --operation log_patterns\|metric_trends` |
| run a query and get natural structured results (frames / records / samples) | `data --expr "<query>"` |

## Hot flow — diagnose a noisy datasource

```bash
# 1. discover the real datasource name + type (never guess)
fduty monit datasource-list --output-format toon
# 2a. validate / run a query — time goes INSIDE the query, there are NO time flags
fduty monit-query data --ds-name <name> --ds-type <type> --expr "rate(http_requests_total[5m])" --output-format toon
# 2b. or get pre-clustered RCA over a window
fduty monit-query diagnose --ds-name <name> --ds-type <type> \
  --operation log_patterns --input-query '{app="my-app"} |= "error"' --time-start -1h --time-end now
```

<!-- GENERATED:monit-query START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### data
Structured datasource query (returns a stable query_result.v1: frames/records/samples)
- `--args` stringSlice
- `--delay-seconds` int64
- `--ds-name` string
- `--ds-type` string
- `--expr` string
- response: single object (`data` unwrapped to the top level) — fields: format (string); result (object)

### diagnose
Pre-clustered RCA findings (log_patterns or metric_trends)
- `--ds-name` string
- `--ds-type` string
- `--input-query` string
- `--max-logs` int
- `--max-patterns` int
- `--operation` string
- `--time-end` string
- `--time-start` string
- `--timeout-seconds` int
- response: single object (`data` unwrapped to the top level) — fields: data_handling (object); ds_name (string); ds_type (string); operation (string); query (string); results (array<object>); schema_version (string); window (object)

<!-- GENERATED:monit-query END -->

## Key concepts

- **`data` = structured query.** Stable `query_result.v1` response: dispatch on `result.kind` — `frames` (typed tables / time series), `records` (schema-flexible rows, big ints as decimal strings), `samples` (instant samples with labels; non-finite floats as `"NaN"` / `"+Inf"` / `"-Inf"`).
- **`diagnose` = pre-clustered evidence.** Its versioned response echoes the datasource, query, and RFC 3339 analysis window. Each result contains method-specific `pattern_evidence` (logs) or `series_evidence` (metrics), structured window statistics, and observations; log results also declare redaction and untrusted observed-data paths in `data_handling`. Takes `--time-start` / `--time-end` (relative like `-1h`, `now`, or unix seconds).

## Gotchas

- **Discover the datasource name first** (`monit datasource-list`). A wrong/guessed name 400s `can not find datasource` — re-list, don't retry variants.
- **A 5xx or HTML-body error is TRANSIENT** — retry the same call ≤3×. Do NOT fall back to SSH, `monit-agent`, or incident search on a transient datasource error.
- **`data` has no time flags** — putting `--time-start` on it is wrong; embed the range in `--expr` (or use `--delay-seconds` for the point-in-time lookback).
- Empty results = the query genuinely matched nothing in that window — report it, don't widen blindly.
- **`diagnose` rejects windows wider than 6 hours outright.** `--time-start`/`--time-end` span is capped at 6h server-side; the default window is the last 15 minutes (`--time-start 15m`, `--time-end now`). Widen within the cap, don't retry past it.
- **`diagnose` pairs one operation with one set of datasource types, and rejects every other combination server-side.** `log_patterns` takes `loki` or `victorialogs`; `metric_trends` takes `prometheus`. There is no third operation, so no other `--ds-type` value can succeed — `mysql`, `oracle`, `postgres`, `clickhouse`, `elasticsearch`, and `sls` all come back as an invalid-parameter error however you pair them. `monit datasource-list` returns those types because `data` supports them; `diagnose` does not.
- **Tunables and their caps**: `--max-logs` (default 10000, cap 50000), `--max-patterns` (default 20, cap 50), `--timeout-seconds` (default 25, cap 30).

## Worked example — log-pattern evidence in the last hour

```bash
fduty monit-query diagnose --ds-name prod-loki --ds-type loki \
  --operation log_patterns --input-query '{app="payment"} |= "error"' --time-start -1h --time-end now --output-format toon
```
