# fduty monit — querying datasources and inspecting hosts

Read only the card for the selected task. Use a configured datasource for metrics, logs and database/middleware diagnostics. Use a registered host for on-box checks.

| Need | Command / reference |
|---|---|
| PromQL, SQL, LogQL, LogsQL or SLS query | `monit-query data`; `reference/monit-query.md` |
| Metric trends, log patterns, database locks, Redis/Kafka/ES diagnostics | `monit datasource-tools-invoke`; `reference/monit-datasource.md` |
| Find a registered host | `monit targets --keyword <prefix>` |
| Discover/invoke host tools | `monit-agent catalog` / `monit-agent invoke`; `reference/monit-agent.md` |

Datasource tools use `datasource_id`, one tool per call and static tool guidance. Host tools use `target_locator`, a live host catalog and up to eight tools per call. These request/response formats are different. Database endpoints are no longer Agent targets.

The legacy `query-diagnose` command is retained for existing callers; new investigations use named tools. Trend/pattern tools take explicit `params.time_range` Unix seconds (up to six hours), while database overview tools observe the current server. Source evidence is not a confirmed root cause. Read warning/truncation fields before interpreting results.

`targets.updated_at` is last-seen time, not proof that an Agent is currently reachable. Host tool execution follows the selected tool's approval policy.

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
- `--target-kind` string — Optional target kind; only host is supported. Inferred when omitted. · enum: host
- `--target-locator` string (required) — Host name. Max 256 bytes; no whitespace, control characters or |.
- response: single object (`data` unwrapped to the top level) — fields: error (object); target (object); tools (array<object>)

### tools-invoke
Invoke target tools
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied.
- `--target-kind` string — Optional target kind; only host is supported. Inferred when omitted. · enum: host
- `--target-locator` string (required) — Host name. Max 256 bytes; no whitespace, control characters or |.
- body-only (`--data`): tools (array<object>) (required)
- response: single object (`data` unwrapped to the top level) — fields: error (object); results (array<object>); target (object)

<!-- GENERATED:monit[query,targets,tools] END -->
