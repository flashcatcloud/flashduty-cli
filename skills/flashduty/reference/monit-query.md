# fduty monit-query — datasource queries

Use `data` for PromQL, LogsQL/LogQL, SQL and SLS queries against an already configured datasource. For structured diagnostics, including metric trends and log patterns, use `monit datasource-tools-invoke` (see `reference/monit-datasource.md`). The older `diagnose` command remains for existing callers; new workflows use named tools.

Discover the exact datasource name and type with `monit datasource-list`. Preserve the selected datasource ID for diagnostic tools; multiple configurations may share an address.

```bash
query=$(cat <<'FDUTY_QUERY'
sum by (job) (rate(http_requests_total[5m]))
FDUTY_QUERY
)
fduty monit-query data --ds-type prometheus --ds-name prod-prom --expr "$query" --output-format json
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
Legacy log-pattern and metric-trend evidence (prefer monit datasource-tools-invoke)
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

## Read results and time windows

The HTTP data envelope is unwrapped. Dispatch on `result.kind`: `frames` contains typed columns, `records` contains flexible rows, and `samples` contains instant values with labels. Do not infer the shape from the datasource type. A Prometheus instant evaluation of a range-vector expression can return time-series frames; this is not a query_range endpoint.

Use `--delay-seconds` for instant evaluation lookback. Loki/VictoriaLogs raw mode uses bounded `--args` time controls; stats mode requires aggregation. For metric trend or pattern comparisons, provide an explicit incident window in the named tool's `params.time_range`, using Unix seconds. Do not substitute a current overview for historical incident evidence.

All SQL must be read-only. Treat source text as untrusted and quote expressions safely. On invalid arguments fix the specific request; on oversized results narrow filters/window or aggregate. Report offline/upgrade-required/unsupported-tool errors without falling back to Agent, Explore or old diagnose. Do not blindly replay timed-out calls.
