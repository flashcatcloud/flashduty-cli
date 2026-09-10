# fduty monit — datasource queries and diagnostics

Read only the card for the selected task. Use a configured datasource for metrics, logs and database/middleware diagnostics.

| Need | Command / reference |
|---|---|
| PromQL, SQL, LogQL, LogsQL or SLS query | `monit-query --tool '<type>.query'`; `reference/monit-query.md` |
| Metric trends, log patterns, database locks, Redis/Kafka/ES diagnostics | `monit-query --tool '<name>'`; `reference/monit-datasource.md` |

Datasource tools use `datasource_id`, one tool per call and static tool guidance.

Investigations use named datasource tools. Trend/pattern tools take explicit `params.time_range` Unix seconds (up to six hours), while database overview tools observe the current server. Source evidence is not a confirmed root cause. Read warning/truncation fields before interpreting results.


<!-- GENERATED:monit[query] START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### query-data
Query structured data
- `--account-id` int64 — Optional consistency check. Must equal the authenticated account when supplied; mismatched values are rejected. Business execution always uses the authenticated account.
- `--delay-seconds` int64 — Look-back offset in seconds applied to point-in-time queries (Prometheus, Loki stats, VictoriaLogs stats). Ignored for raw / detail queries.
- `--ds-name` string (required) — Data source name; must match a configured data source under the tenant.
- `--ds-type` string (required) — Data source type; must match a configured data source under the tenant. Examples: 'prometheus', 'loki', 'victorialogs', 'sls', 'elasticsearch', 'mysql', 'postgres', 'oracle', 'clickhouse'.
- `--expr` string (required) — Query expression. Syntax depends on 'ds_type' and is interpreted by the corresponding monit-edge client (PromQL for Prometheus, LogQL for Loki, SQL for SQL sources, etc.).
- body-only (`--data`): args (object)
- response: single object (`data` unwrapped to the top level) — fields: format (string); result (object)

<!-- GENERATED:monit[query] END -->
