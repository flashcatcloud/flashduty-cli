# fduty monit-query — command card

Prereq: `SKILL.md` read. Datasource-side RCA: query a monitoring datasource directly. Both verbs are read-only. Pairs with **`monit`** (rule config) and **`monit-agent`** (on-box host/db diagnostics).

## Route here when

"指标查询 / 日志查询 / PromQL / LogsQL / SQL 验证 / 趋势 / 日志聚类 / 数据源 RCA" → **monit-query**. You need a **datasource name + type** — get them from `fduty monit datasource-list` first; **never guess a datasource name** (a wrong name 400s `can not find datasource`).

## Intent → verb

| want | verb |
|---|---|
| pre-clustered RCA findings (surging log patterns / notable metric trends) | `diagnose --operation log_patterns\|metric_trends` |
| run a raw query and get values/rows back as the datasource returns them | `rows --expr "<query>"` |

## Hot flow — diagnose a noisy datasource

```bash
# 1. discover the real datasource name + type (never guess)
fduty monit datasource-list --output-format toon
# 2a. validate / run a raw query — time goes INSIDE the query, there are NO time flags
fduty monit-query rows --ds-name <name> --ds-type <type> --expr "rate(http_requests_total[5m])"
# 2b. or get pre-clustered RCA over a window
fduty monit-query diagnose --ds-name <name> --ds-type <type> \
  --operation log_patterns --time-start -1h --time-end now
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

### rows
Raw datasource passthrough (returns values/rows as the datasource itself would)
- `--args` stringSlice
- `--ds-name` string
- `--ds-type` string
- `--expr` string

<!-- GENERATED:monit-query END -->

## Key concepts

- **`rows` = raw passthrough.** Response `data` is a **top-level array** of row objects — pipe `jq '.[]'`, NOT `.items[]`. Numeric fields under `values` (metric canonical key `__value__`); labels/columns under `fields`. **Time belongs in the query expression**, not in flags.
- **`diagnose` = pre-clustered findings.** `--operation log_patterns` returns surging/new/gone log templates (RCA-sorted); `metric_trends` returns notable series (current vs baseline). Takes `--time-start` / `--time-end` (relative like `-1h`, `now`, or unix seconds).

## Gotchas

- **Discover the datasource name first** (`monit datasource-list`). A wrong/guessed name 400s `can not find datasource` — re-list, don't retry variants.
- **A 5xx or HTML-body error is TRANSIENT** — retry the same call ≤3×. Do NOT fall back to SSH, `monit-agent`, or incident search on a transient datasource error.
- `rows` has **no time flags** — putting `--time-start` on `rows` is wrong; embed the range in `--expr`.
- Empty results = the query genuinely matched nothing in that window — report it, don't widen blindly.

## Worked example — surging log patterns in the last hour

```bash
fduty monit-query diagnose --ds-name prod-loki --ds-type loki \
  --operation log_patterns --time-start -1h --time-end now --output-format toon
```
