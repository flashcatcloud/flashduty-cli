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
- **`--ds-type` on `diagnose` only accepts `prometheus`, `victorialogs`, `loki`, `mysql`.** `monit datasource-list` can return other types (e.g. `oracle`, `postgres`, `clickhouse`, `elasticsearch`, `sls`) — those are not supported here.
- **Tunables and their caps**: `--max-logs` (default 10000, cap 50000), `--max-patterns` (default 20, cap 50), `--timeout-seconds` (default 25, cap 30).
- **LogsQL (`victorialogs`) puts `by (...)` BEFORE the aggregate.** Write `| stats by (level) count() n`. The SQL/PromQL/Loki habit of trailing `by` — `| stats count(*) by (level)`, `| stats count() by level`, `| stats count(*) as n by level` — is a **parse** error, not a semantic one: `cannot parse 'stats' pipe: unexpected token ... after [count(*)]`. Same shape for several keys (`| stats by (level, file) count() n`) and for a global aggregate (`| stats count() n`, no `by` at all). Downstream pipes are ordinary: `| sort (n desc) | limit 10`, `| filter n:>100`.
- **LogsQL `_time:` takes a duration or a bracketed range — never a slash range.** `_time:5m`, `_time:1h` and `_time:[2026-01-31T02:00:00Z, 2026-01-31T03:00:00Z]` parse. `_time:2026-01-31T02:00:00Z/2026-01-31T03:00:00Z`, `_time:1/31T02:00:00Z/…` and `_time:02:00Z-03:00Z` all fail with `cannot parse duration at _time filter` — timestamps must be full RFC 3339, and abbreviated dates never parse.

## Worked example — log-pattern evidence in the last hour

```bash
fduty monit-query diagnose --ds-name prod-loki --ds-type loki \
  --operation log_patterns --input-query '{app="payment"} |= "error"' --time-start -1h --time-end now --output-format toon
```

## Worked example — LogsQL top error sources in the last hour

```bash
fduty monit-query data --ds-name prod-vlogs --ds-type victorialogs \
  --expr '_time:1h _stream:{module="payment"} level:ERROR | stats by (file) count() n | sort (n desc) | limit 10' \
  --output-format toon
```
