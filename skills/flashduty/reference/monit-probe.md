# fduty monit — probing a datasource or a target

Prereq: `SKILL.md` + `reference/monit.md` read. These are the runtime verbs: ask a datasource a question, or ask a monitored target about itself. Everything else under `monit` is configuration.

## Route here when

"指标查询 / 日志查询 / PromQL / 诊断 / 监控目标 / 主机工具" or "metric query / log query / diagnose / monitored host / tools catalog" → this card.

**Mutating:** `tools-invoke` runs code on the target — confirm before running. The query verbs are read-only.

## Intent → verb

| want | verb |
|---|---|
| run ad-hoc PromQL / SQL / LogQL | `query-data` here, or the curated `monit-query data` — see `reference/monit-query.md` |
| log-pattern / metric-trend RCA evidence | `query-diagnose` |
| list monitored hosts/targets | `targets` |
| what tools a target exposes | `tools-catalog` |
| run host/db diagnostic tools | `tools-invoke` |

## Hot flow — ad-hoc query + diagnose

```bash
# 1. discover the real datasource name — NEVER guess
fduty monit datasource-list --output-format toon
fduty monit datasource-list --type prometheus --output-format toon

# 2a. point-in-time query (PromQL/SQL/LogQL); ALL time range goes INSIDE --expr
#     (the curated 'monit-query data' — see the monit-query card)
fduty monit-query data --ds-type prometheus --ds-name <ds-name> \
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
{"tools":[{"tool":"os.overview"},{"tool":"os.top_processes","params":{"top_n":10}}]}
EOF
```

## Key concepts

**`operation` on `query-diagnose`**: `log_patterns` (loki / victorialogs) or `metric_trends` (prometheus); inferred from `--ds-type` when omitted — only pass it explicitly for ambiguous source types.

**`query-diagnose` output**: results are versioned evidence, not the former summary-only pattern/series lists. Read `pattern_evidence` for logs or `series_evidence` for metrics; their optional comparison fields are absent when the edge has no evidence. Log output also includes `data_handling`, which declares redaction coverage and paths carrying untrusted observed data.

**`targets`**: `updated_at` means "last seen", not "online now".

## Gotchas

- **`monit-query data` has no time flags.** There is no `--time-start` / `--time-end` / `--operation`. Embed all time range and bucketing inside `--expr`. Passing those flags is a silent no-op or error.
- **`query-diagnose` time window via `--data`**, not flags. Pass `{"time_range":{"start":<unix>,"end":<unix>},...}`. Window wider than 6 hours is rejected server-side. Omitting `time_range` defaults to the last 15 minutes.
- **`tools-catalog` / `tools-invoke` `--target-locator` is required and not guessable.** If the user has not provided a host or IP, ask — do not invent one. Tool names in `invoke` must come from the `tools-catalog` response — never hallucinate them.

<!-- GENERATED:monit[query,targets,tools] START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### query-data
Query structured data
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied; mismatched values are rejected. Business execution always uses the authenticated account.
- `--delay-seconds` int64 — Look-back offset in seconds applied to point-in-time queries (Prometheus, Loki stats, VictoriaLogs stats). Ignored for raw / detail queries.
- `--ds-name` string (required) — Data source name; must match a configured data source under the tenant.
- `--ds-type` string (required) — Data source type; must match a configured data source under the tenant. Examples: 'prometheus', 'loki', 'victorialogs', 'sls', 'elasticsearch', 'mysql', 'postgres', 'oracle', 'clickhouse'.
- `--expr` string (required) — Query expression. Syntax depends on 'ds_type' and is interpreted by the corresponding monit-edge client (PromQL for Prometheus, LogQL for Loki, SQL for SQL sources, etc.).
- body-only (`--data`): args (object)
- response: single object (`data` unwrapped to the top level) — fields: format (string); result (object)

### query-diagnose
Diagnose data source
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied.
- `--ds-name` string (required) — Data source name configured under the tenant.
- `--ds-type` string (required) — Data source type. 'log_patterns' supports 'loki' and 'victorialogs'; 'metric_trends' supports 'prometheus'.
- `--operation` string — Diagnostic operation. When omitted, inferred from 'ds_type' (loki / victorialogs → 'log_patterns', prometheus → 'metric_trends'). Other sources must specify explicitly. · enum: log_patterns | metric_trends
- body-only (`--data`): input (object) (required); methods (array<object>); options (object); time_range (object)
- response: single object (`data` unwrapped to the top level) — fields: data_handling (object); ds_name (string); ds_type (string); operation (string); query (string); results (array<object>); schema_version (string); window (object)

### targets
List monitored targets
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied.
- `--cursor` string — Opaque pagination cursor from the previous response's 'next_cursor'. Omit / pass empty string for the first page. Reset whenever 'keyword', 'limit', or tenant changes.
- `--keyword` string — Prefix match against 'target_locator'. ASCII only, no whitespace, no '|', max 256 bytes. Substring search is not supported.
- `--limit` int64 — Page size. Default 50, max 200. (max 200)
- response: single object (`data` unwrapped to the top level) — fields: items (array<object>); next_cursor (string); servicemap_coverage (object); total (integer)

### tools-catalog
List target tool catalog
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied.
- `--target-kind` string — Optional target kind. When omitted, webapi infers it from current target routing. If the call returns 'ambiguous_target_kind', retry with a value from 'target_kinds'.
- `--target-locator` string (required) — Target identifier (host name, MySQL address, …). Max 256 bytes; no whitespace, control characters, or '|'.
- response: single object (`data` unwrapped to the top level) — fields: error (object); target (object); tools (array<object>)

### tools-invoke
Invoke target tools
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied.
- `--target-kind` string — Optional target kind; auto-inferred when omitted.
- `--target-locator` string (required) — Target identifier. Same validation rules as '/monit/tools/catalog'.
- body-only (`--data`): tools (array<object>) (required)
- response: single object (`data` unwrapped to the top level) — fields: error (object); results (array<object>); target (object)

<!-- GENERATED:monit[query,targets,tools] END -->
