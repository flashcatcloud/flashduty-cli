# flashduty monit — datasource queries and diagnostics

Read only the card for the selected task. Use a configured datasource for metrics, logs and database/middleware diagnostics.

| Need | Command / reference |
|---|---|
| PromQL, SQL, LogQL, LogsQL or SLS query | `monit-query --tool '<type>.query'`; `reference/monit-query.md` |
| Metric trends, log patterns, database locks, Redis/Kafka/ES diagnostics | `monit-query --tool '<name>'`; `reference/monit-datasource.md` |

Datasource tools use `datasource_id`, one tool per call and static tool guidance.

Investigations use named datasource tools. Trend/pattern tools take explicit `params.time_range` Unix seconds (up to six hours), while database overview tools observe the current server. Source evidence is not a confirmed root cause. Read warning/truncation fields before interpreting results.


<!-- GENERATED:monit[query] START · 由 flashduty __dump-commands 同步 · 勿手改 fence 内 -->

### query-data
Query structured data
- `--account-id` int64 — Optional consistency check. When supplied it must equal the authenticated account, otherwise the request fails with 'AccessDenied'. The query always runs as the authenticated account.
- `--delay-seconds` int64 — Look-back offset in seconds, from 0 to 86400. Applied to point-in-time queries (Prometheus, Loki stats, VictoriaLogs stats); ignored for raw and detail queries. When 'args' carries an absolute '<source>.time' and the Edge is v0.74.0 or later, the absolute time takes precedence and this offset is not applied. (0-86400)
- `--ds-name` string (required) — Data source name, as returned by '/monit/datasource/list'. The data source must be enabled; a name that matches no data source of the account returns HTTP 400.
- `--ds-type` string (required) — Data source type. Together with 'ds_name' it must match an enabled data source of the account. - 'prometheus': Prometheus-compatible data source; 'expr' is PromQL. - 'loki': Grafana Loki; 'expr' is LogQL. - 'victorialogs': VictoriaLogs. - 'sls': Alibaba Cloud Log Service (SLS). - 'tencent_cls': Tencent Cloud Log Service (CLS); requires monit-edge v0.66.0 or later. - 'elasticsearch': Elasticsearch. - 'mysql': MySQL; 'expr' is SQL. - 'postgres': PostgreSQL; 'expr' is SQL. - 'oracle': Oracle Database; 'expr' is SQL. - 'clickhouse': ClickHouse; 'expr' is SQL. Other data source types (for example MongoDB, Redis, or Kafka) are rejected. · enum: clickhouse | elasticsearch | loki | mysql | oracle | postgres | prometheus | sls | tencent_cls | victorialogs
- `--expr` string (required) — Query expression in the language of 'ds_type' (PromQL for 'prometheus', LogQL for 'loki', SQL for SQL sources, and so on). Required, at most 64 KiB (12 KiB for 'tencent_cls'). Loki's '$__auto' is not supported here; use a fixed duration such as '[5m]'. Prometheus expressions are syntax-checked before dispatch. (≥1 chars)
- body-only (`--data`): args (object)
- response: single object (`data` unwrapped to the top level) — fields: format (string); result (object)

### query-explore <datasource-id>
Run Explore query
- `<datasource-id>` (positional, required) int64 — Data source ID from '/monit/datasource/list'. Must be a positive JavaScript-safe integer for an enabled data source of the authenticated account that the caller may query. (1-9007199254740991)
- `--expr` string (required) — Query expression in the data source's native language (PromQL, LogsQL, SQL, and so on). Non-empty UTF-8 of at most 64 KiB; 'mysql' allows at most 16 KiB and 'tencent_cls' at most 12 KiB. (≥1 chars)
- body-only (`--data`): args (object) (required); execution (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: execution (object); format (string); result (object)

<!-- GENERATED:monit[query] END -->
