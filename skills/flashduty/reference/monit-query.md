# fduty monit-query — command card

Prereq: `SKILL.md` read. Datasource-side RCA: query a monitoring datasource directly. Both verbs are read-only. Pairs with **`monit`** (rule config) and **`monit-agent`** (on-box host/db diagnostics).

## Route here when

"指标查询 / 日志查询 / PromQL / LogsQL / SQL 验证 / 趋势 / 日志聚类 / 数据源 RCA" → **monit-query**. You need a **datasource name + type** — get them from `fduty monit datasource-list` first; **never guess a datasource name** (a wrong name 400s `can not find datasource`).

## Intent → verb

| want | verb |
|---|---|
| pre-clustered RCA evidence (log patterns / metric trends) | `diagnose --operation log_patterns\|metric_trends` |
| run a raw query and get values/rows back as the datasource returns them | `rows --expr "<query>"` |

## Hot flow — diagnose a noisy datasource

```bash
# 1. discover the real datasource name + type (never guess)
fduty monit datasource-list --output-format toon
# 2a. validate / run a raw query — time goes INSIDE the query, there are NO time flags
fduty monit-query rows --ds-name <name> --ds-type <type> --expr "rate(http_requests_total[5m])"
# 2b. or get pre-clustered RCA over a window
fduty monit-query diagnose --ds-name <name> --ds-type <type> \
  --operation log_patterns --input-query '{app="my-app"} |= "error"' --time-start -1h --time-end now
```

<!-- GENERATED:monit-query START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

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

### rows
Raw datasource passthrough (returns values/rows as the datasource itself would)
- `--args` stringSlice
- `--ds-name` string
- `--ds-type` string
- `--expr` string
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: fields (object); values (object)

<!-- GENERATED:monit-query END -->

## Key concepts

- **`rows` = raw passthrough.** Numeric fields under `values` (metric canonical key `__value__`); labels/columns under `fields`. **Time belongs in the query expression**, not in flags.
- **`diagnose` = pre-clustered evidence.** Its versioned response echoes the datasource, query, and RFC 3339 analysis window. Each result contains method-specific `pattern_evidence` (logs) or `series_evidence` (metrics), structured window statistics, and observations; log results also declare redaction and untrusted observed-data paths in `data_handling`. Takes `--time-start` / `--time-end` (relative like `-1h`, `now`, or unix seconds).

## Gotchas

- **Discover the datasource name first** (`monit datasource-list`). A wrong/guessed name 400s `can not find datasource` — re-list, don't retry variants.
- **A 5xx or HTML-body error is TRANSIENT** — retry the same call ≤3×. Do NOT fall back to SSH, `monit-agent`, or incident search on a transient datasource error.
- `rows` has **no time flags** — putting `--time-start` on `rows` is wrong; embed the range in `--expr`.
- Empty results = the query genuinely matched nothing in that window — report it, don't widen blindly.
- **`diagnose` rejects windows wider than 6 hours outright.** `--time-start`/`--time-end` span is capped at 6h server-side; the default window is the last 15 minutes (`--time-start 15m`, `--time-end now`). Widen within the cap, don't retry past it.
- **`--ds-type` on `diagnose` only accepts `prometheus`, `victorialogs`, `loki`, `mysql`.** `monit datasource-list` can return other types (e.g. `oracle`, `postgres`, `clickhouse`, `elasticsearch`, `sls`) — those are not supported here.
- **Tunables and their caps**: `--max-logs` (default 10000, cap 50000), `--max-patterns` (default 20, cap 50), `--timeout-seconds` (default 25, cap 30).

## Worked example — log-pattern evidence in the last hour

```bash
fduty monit-query diagnose --ds-name prod-loki --ds-type loki \
  --operation log_patterns --input-query '{app="payment"} |= "error"' --time-start -1h --time-end now --output-format toon
```
