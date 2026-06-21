# fduty insight — command card

Prereq: `SKILL.md` read. All `insight` verbs are **read-only** — no mutations, no confirmations needed.

## Route here when

"噪声治理 / 高频告警 / MTTA / MTTR / 绩效复盘 / 月报 / SRE review / noise reduction / alert fatigue / who responds fastest / channel performance / team metrics / incident export / CSV export" → **insight**.

Do **not** hand-aggregate from `alert list` / `incident list` — `insight` does server-side aggregation and gives authoritative numbers. Key IDs you may need: `--team-ids` and `--channel-ids` from `fduty channel list` or `fduty team list`; `--responder-ids` from `fduty member list`.

## Intent → verb

| want | verb |
|---|---|
| top noisy alert sources by check/resource | `top-alerts` |
| finer noise drill-down (sort, severity, team/channel filter, time buckets) | `alert-topk-by-label` |
| account-wide MTTA/MTTR/ack-rate roll-up | `account` |
| per-channel breakdown | `channel` |
| per-team breakdown | `team` |
| per-responder MTTA/workload breakdown | `responder` |
| per-incident list with response time columns | `incidents` |
| per-incident list with rich filters (severity, responder, cursor) | `incident-list` |
| CSV export of incidents | `incident-export` |
| CSV export of responder metrics | `responder-export` |
| CSV export of channel metrics | `channel-export` |
| CSV export of team metrics | `team-export` |

## Hot flow — weekly SRE review

```bash
# account-level roll-up for past 30 days
fduty insight account --start-time 30d --end-time now --output-format toon

# per-team and per-channel breakdowns (same flags)
fduty insight team    --start-time 30d --end-time now --output-format toon
fduty insight channel --start-time 30d --end-time now --output-format toon

# who responded slowest (per-responder MTTA)
fduty insight responder --start-time 30d --end-time now --output-format toon

# top-10 noisiest check sources this week
fduty insight top-alerts --label check --since 7d --output-format toon

# per-incident list with MTTA/MTTR (uses --since not --start-time)
fduty insight incidents --since 30d --limit 50 --output-format toon
```

## Hot flow — 月报 CSV export

```bash
# incident-export takes epoch seconds ONLY (int64, not relative strings)
S=$(date -v-30d +%s); E=$(date +%s)
fduty insight incident-export --start-time $S --end-time $E > incidents.csv

# responder/channel/team-export accept relative strings
fduty insight responder-export --start-time 30d --end-time now > responders.csv
fduty insight channel-export   --start-time 30d --end-time now > channels.csv
fduty insight team-export      --start-time 30d --end-time now > teams.csv
```

<!-- GENERATED:insight START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### account
Get account-level insight
- `--aggregate-unit` string — Aggregate metrics into time buckets. When set, the time range must cover at least 24 hours; 'day' additionally caps the range at 31 days. · enum: day | week | month
- `--asc` bool — Sort ascending when 'true', descending otherwise.
- `--channel-ids` intSlice — Filter by channel IDs. At most 100 entries.
- `--description-html-to-text` bool — Strip HTML markup from the description column when exporting.
- `--end-time` string (required) — End time, Unix seconds. Must be greater than 'start_time'. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--export-fields` stringSlice — Subset of CSV column keys to include in the export. At most 50 entries. Only used by the export endpoints. · enum: incident_id | title | severity | progress | channel_id | channel_name | team_id | team_name | created_at | seconds_to_ack | seconds_to_close | closed_by | engaged_seconds | hours | notifications | interruptions | acknowledgements | assignments | reassignments | escalations | manual_escalations | timeout_escalations | assigned_to | responders | description | labels | fields | creator_id | creator_name
- `--incident-ids` stringSlice — Filter by incident IDs (MongoDB ObjectIDs). At most 100 entries.
- `--is-my-team` bool — Restrict results to teams the caller belongs to. When true and the caller has no teams, the result set is empty.
- `--orderby` string — Field to sort the underlying incident set by. · enum: created_at
- `--query` string — Full-text query applied to incident title and description.
- `--responder-ids` intSlice — Filter by responder person IDs. At most 100 entries.
- `--seconds-to-ack-from` int64 — Lower bound (inclusive) on time-to-acknowledge, in seconds.
- `--seconds-to-ack-to` int64 — Upper bound (exclusive) on time-to-acknowledge, in seconds. Must be greater than 'seconds_to_ack_from' when both are set.
- `--seconds-to-close-from` int64 — Lower bound (inclusive) on time-to-close, in seconds.
- `--seconds-to-close-to` int64 — Upper bound (exclusive) on time-to-close, in seconds. Must be greater than 'seconds_to_close_from' when both are set.
- `--severities` stringSlice — Filter by severity. At most 3 entries. · enum: Critical | Warning | Info | Ok
- `--split-hours` bool — When true, metrics are split into 'work'/'sleep'/'off' hour buckets.
- `--start-time` string (required) — Start time, Unix seconds. Must be greater than 0. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-ids` intSlice — Filter by team IDs. At most 100 entries.
- `--time-zone` string — IANA time zone name used to interpret the time range (e.g. 'Asia/Shanghai'). Defaults to the account time zone.
- body-only (`--data`): fields (object); labels (object)

### alert-topk-by-label
Get top-K alerts grouped by check or resource
- `--aggregate-unit` string — Aggregate metrics into time buckets. When set, the time range must cover at least 24 hours; 'day' additionally caps the range at 31 days. · enum: day | week | month
- `--asc` bool — Sort ascending when 'true', descending otherwise.
- `--channel-ids` intSlice — Filter by channel IDs. At most 100 entries.
- `--description-html-to-text` bool — Strip HTML markup from the description column when exporting.
- `--end-time` string (required) — End time, Unix seconds. Must be greater than 'start_time'. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--export-fields` stringSlice — Subset of CSV column keys to include in the export. At most 50 entries. Only used by the export endpoints. · enum: incident_id | title | severity | progress | channel_id | channel_name | team_id | team_name | created_at | seconds_to_ack | seconds_to_close | closed_by | engaged_seconds | hours | notifications | interruptions | acknowledgements | assignments | reassignments | escalations | manual_escalations | timeout_escalations | assigned_to | responders | description | labels | fields | creator_id | creator_name
- `--incident-ids` stringSlice — Filter by incident IDs (MongoDB ObjectIDs). At most 100 entries.
- `--is-my-team` bool — Restrict results to teams the caller belongs to. When true and the caller has no teams, the result set is empty.
- `--k` int64 — Number of top entries to return, between 1 and 100.
- `--label` string (required) — Dimension to aggregate by. · enum: check | resource
- `--orderby` string — Field to sort results by. · enum: total_alert_cnt | total_alert_event_cnt
- `--query` string — Full-text query applied to incident title and description.
- `--responder-ids` intSlice — Filter by responder person IDs. At most 100 entries.
- `--seconds-to-ack-from` int64 — Lower bound (inclusive) on time-to-acknowledge, in seconds.
- `--seconds-to-ack-to` int64 — Upper bound (exclusive) on time-to-acknowledge, in seconds. Must be greater than 'seconds_to_ack_from' when both are set.
- `--seconds-to-close-from` int64 — Lower bound (inclusive) on time-to-close, in seconds.
- `--seconds-to-close-to` int64 — Upper bound (exclusive) on time-to-close, in seconds. Must be greater than 'seconds_to_close_from' when both are set.
- `--severities` stringSlice — Filter by severity. At most 3 entries. · enum: Critical | Warning | Info | Ok
- `--split-hours` bool — When true, metrics are split into 'work'/'sleep'/'off' hour buckets.
- `--start-time` string (required) — Start time, Unix seconds. Must be greater than 0. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-ids` intSlice — Filter by team IDs. At most 100 entries.
- `--time-zone` string — IANA time zone name used to interpret the time range (e.g. 'Asia/Shanghai'). Defaults to the account time zone.
- body-only (`--data`): fields (object); labels (object)

### channel
Get channel insight
- `--aggregate-unit` string — Aggregate metrics into time buckets. When set, the time range must cover at least 24 hours; 'day' additionally caps the range at 31 days. · enum: day | week | month
- `--asc` bool — Sort ascending when 'true', descending otherwise.
- `--channel-ids` intSlice — Filter by channel IDs. At most 100 entries.
- `--description-html-to-text` bool — Strip HTML markup from the description column when exporting.
- `--end-time` string (required) — End time, Unix seconds. Must be greater than 'start_time'. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--export-fields` stringSlice — Subset of CSV column keys to include in the export. At most 50 entries. Only used by the export endpoints. · enum: incident_id | title | severity | progress | channel_id | channel_name | team_id | team_name | created_at | seconds_to_ack | seconds_to_close | closed_by | engaged_seconds | hours | notifications | interruptions | acknowledgements | assignments | reassignments | escalations | manual_escalations | timeout_escalations | assigned_to | responders | description | labels | fields | creator_id | creator_name
- `--incident-ids` stringSlice — Filter by incident IDs (MongoDB ObjectIDs). At most 100 entries.
- `--is-my-team` bool — Restrict results to teams the caller belongs to. When true and the caller has no teams, the result set is empty.
- `--orderby` string — Field to sort the underlying incident set by. · enum: created_at
- `--query` string — Full-text query applied to incident title and description.
- `--responder-ids` intSlice — Filter by responder person IDs. At most 100 entries.
- `--seconds-to-ack-from` int64 — Lower bound (inclusive) on time-to-acknowledge, in seconds.
- `--seconds-to-ack-to` int64 — Upper bound (exclusive) on time-to-acknowledge, in seconds. Must be greater than 'seconds_to_ack_from' when both are set.
- `--seconds-to-close-from` int64 — Lower bound (inclusive) on time-to-close, in seconds.
- `--seconds-to-close-to` int64 — Upper bound (exclusive) on time-to-close, in seconds. Must be greater than 'seconds_to_close_from' when both are set.
- `--severities` stringSlice — Filter by severity. At most 3 entries. · enum: Critical | Warning | Info | Ok
- `--split-hours` bool — When true, metrics are split into 'work'/'sleep'/'off' hour buckets.
- `--start-time` string (required) — Start time, Unix seconds. Must be greater than 0. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-ids` intSlice — Filter by team IDs. At most 100 entries.
- `--time-zone` string — IANA time zone name used to interpret the time range (e.g. 'Asia/Shanghai'). Defaults to the account time zone.
- body-only (`--data`): fields (object); labels (object)

### channel-export
Export channel insight
- `--aggregate-unit` string — Aggregate metrics into time buckets. When set, the time range must cover at least 24 hours; 'day' additionally caps the range at 31 days. · enum: day | week | month
- `--asc` bool — Sort ascending when 'true', descending otherwise.
- `--channel-ids` intSlice — Filter by channel IDs. At most 100 entries.
- `--description-html-to-text` bool — Strip HTML markup from the description column when exporting.
- `--end-time` string (required) — End time, Unix seconds. Must be greater than 'start_time'. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--export-fields` stringSlice — Subset of CSV column keys to include in the export. At most 50 entries. Only used by the export endpoints. · enum: incident_id | title | severity | progress | channel_id | channel_name | team_id | team_name | created_at | seconds_to_ack | seconds_to_close | closed_by | engaged_seconds | hours | notifications | interruptions | acknowledgements | assignments | reassignments | escalations | manual_escalations | timeout_escalations | assigned_to | responders | description | labels | fields | creator_id | creator_name
- `--incident-ids` stringSlice — Filter by incident IDs (MongoDB ObjectIDs). At most 100 entries.
- `--is-my-team` bool — Restrict results to teams the caller belongs to. When true and the caller has no teams, the result set is empty.
- `--orderby` string — Field to sort the underlying incident set by. · enum: created_at
- `--query` string — Full-text query applied to incident title and description.
- `--responder-ids` intSlice — Filter by responder person IDs. At most 100 entries.
- `--seconds-to-ack-from` int64 — Lower bound (inclusive) on time-to-acknowledge, in seconds.
- `--seconds-to-ack-to` int64 — Upper bound (exclusive) on time-to-acknowledge, in seconds. Must be greater than 'seconds_to_ack_from' when both are set.
- `--seconds-to-close-from` int64 — Lower bound (inclusive) on time-to-close, in seconds.
- `--seconds-to-close-to` int64 — Upper bound (exclusive) on time-to-close, in seconds. Must be greater than 'seconds_to_close_from' when both are set.
- `--severities` stringSlice — Filter by severity. At most 3 entries. · enum: Critical | Warning | Info | Ok
- `--split-hours` bool — When true, metrics are split into 'work'/'sleep'/'off' hour buckets.
- `--start-time` string (required) — Start time, Unix seconds. Must be greater than 0. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-ids` intSlice — Filter by team IDs. At most 100 entries.
- `--time-zone` string — IANA time zone name used to interpret the time range (e.g. 'Asia/Shanghai'). Defaults to the account time zone.
- body-only (`--data`): fields (object); labels (object)

### incident-export
Export insight incidents
- `--asc` bool
- `--channel-ids` intSlice
- `--description-html-to-text` bool
- `--end-time` int64
- `--export-fields` stringSlice
- `--incident-ids` stringSlice
- `--is-my-team` bool
- `--orderby` string
- `--query` string
- `--responder-ids` intSlice
- `--seconds-to-ack-from` int64
- `--seconds-to-ack-to` int64
- `--seconds-to-close-from` int64
- `--seconds-to-close-to` int64
- `--severities` stringSlice
- `--start-time` int64
- `--team-ids` intSlice
- `--time-zone` string
- body-only (`--data`): fields (JSON); labels (JSON)

### incident-list
List insight incidents
- `--asc` bool — Sort ascending when 'true', descending otherwise.
- `--channel-ids` intSlice — Filter by channel IDs. At most 100 entries.
- `--description-html-to-text` bool — Strip HTML markup from the description column when exporting.
- `--end-time` string (required) — End time, Unix seconds. Must be greater than 'start_time'. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--export-fields` stringSlice — Subset of CSV column keys to include in the export. At most 50 entries. Only used by the export endpoints. · enum: incident_id | title | severity | progress | channel_id | channel_name | team_id | team_name | created_at | seconds_to_ack | seconds_to_close | closed_by | engaged_seconds | hours | notifications | interruptions | acknowledgements | assignments | reassignments | escalations | manual_escalations | timeout_escalations | assigned_to | responders | description | labels | fields | creator_id | creator_name
- `--incident-ids` stringSlice — Filter by incident IDs (MongoDB ObjectIDs). At most 100 entries.
- `--is-my-team` bool — Restrict results to teams the caller belongs to. When true and the caller has no teams, the result set is empty.
- `--limit` int64 — Page size, between 1 and 100. Defaults to 20. (1-100)
- `--orderby` string — Field to sort the underlying incident set by. · enum: created_at
- `--page` int64 — Page number, starting at 1. Defaults to 1. (min 1)
- `--query` string — Full-text query applied to incident title and description.
- `--responder-ids` intSlice — Filter by responder person IDs. At most 100 entries.
- `--search-after-ctx` string — Cursor token returned by a previous page. Pass it back to fetch the next page.
- `--seconds-to-ack-from` int64 — Lower bound (inclusive) on time-to-acknowledge, in seconds.
- `--seconds-to-ack-to` int64 — Upper bound (exclusive) on time-to-acknowledge, in seconds. Must be greater than 'seconds_to_ack_from' when both are set.
- `--seconds-to-close-from` int64 — Lower bound (inclusive) on time-to-close, in seconds.
- `--seconds-to-close-to` int64 — Upper bound (exclusive) on time-to-close, in seconds. Must be greater than 'seconds_to_close_from' when both are set.
- `--severities` stringSlice — Filter by severity. At most 3 entries. · enum: Critical | Warning | Info | Ok
- `--start-time` string (required) — Start time, Unix seconds. Must be greater than 0. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-ids` intSlice — Filter by team IDs. At most 100 entries.
- `--time-zone` string — IANA time zone name used to interpret the time range (e.g. 'Asia/Shanghai'). Defaults to the account time zone.
- body-only (`--data`): fields (object); labels (object)

### incidents
Query incidents with performance metrics
- `--limit` int
- `--page` int
- `--since` string
- `--until` string

### responder
Get responder insight
- `--aggregate-unit` string — Aggregate metrics into time buckets. When set, the time range must cover at least 24 hours; 'day' additionally caps the range at 31 days. · enum: day | week | month
- `--asc` bool — Sort ascending when 'true', descending otherwise.
- `--channel-ids` intSlice — Filter by channel IDs. At most 100 entries.
- `--description-html-to-text` bool — Strip HTML markup from the description column when exporting.
- `--end-time` string (required) — End time, Unix seconds. Must be greater than 'start_time'. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--export-fields` stringSlice — Subset of CSV column keys to include in the export. At most 50 entries. Only used by the export endpoints. · enum: incident_id | title | severity | progress | channel_id | channel_name | team_id | team_name | created_at | seconds_to_ack | seconds_to_close | closed_by | engaged_seconds | hours | notifications | interruptions | acknowledgements | assignments | reassignments | escalations | manual_escalations | timeout_escalations | assigned_to | responders | description | labels | fields | creator_id | creator_name
- `--incident-ids` stringSlice — Filter by incident IDs (MongoDB ObjectIDs). At most 100 entries.
- `--is-my-team` bool — Restrict results to teams the caller belongs to. When true and the caller has no teams, the result set is empty.
- `--orderby` string — Field to sort the underlying incident set by. · enum: created_at
- `--query` string — Full-text query applied to incident title and description.
- `--responder-ids` intSlice — Filter by responder person IDs. At most 100 entries.
- `--seconds-to-ack-from` int64 — Lower bound (inclusive) on time-to-acknowledge, in seconds.
- `--seconds-to-ack-to` int64 — Upper bound (exclusive) on time-to-acknowledge, in seconds. Must be greater than 'seconds_to_ack_from' when both are set.
- `--seconds-to-close-from` int64 — Lower bound (inclusive) on time-to-close, in seconds.
- `--seconds-to-close-to` int64 — Upper bound (exclusive) on time-to-close, in seconds. Must be greater than 'seconds_to_close_from' when both are set.
- `--severities` stringSlice — Filter by severity. At most 3 entries. · enum: Critical | Warning | Info | Ok
- `--split-hours` bool — When true, metrics are split into 'work'/'sleep'/'off' hour buckets.
- `--start-time` string (required) — Start time, Unix seconds. Must be greater than 0. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-ids` intSlice — Filter by team IDs. At most 100 entries.
- `--time-zone` string — IANA time zone name used to interpret the time range (e.g. 'Asia/Shanghai'). Defaults to the account time zone.
- body-only (`--data`): fields (object); labels (object)

### responder-export
Export responder insight
- `--aggregate-unit` string — Aggregate metrics into time buckets. When set, the time range must cover at least 24 hours; 'day' additionally caps the range at 31 days. · enum: day | week | month
- `--asc` bool — Sort ascending when 'true', descending otherwise.
- `--channel-ids` intSlice — Filter by channel IDs. At most 100 entries.
- `--description-html-to-text` bool — Strip HTML markup from the description column when exporting.
- `--end-time` string (required) — End time, Unix seconds. Must be greater than 'start_time'. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--export-fields` stringSlice — Subset of CSV column keys to include in the export. At most 50 entries. Only used by the export endpoints. · enum: incident_id | title | severity | progress | channel_id | channel_name | team_id | team_name | created_at | seconds_to_ack | seconds_to_close | closed_by | engaged_seconds | hours | notifications | interruptions | acknowledgements | assignments | reassignments | escalations | manual_escalations | timeout_escalations | assigned_to | responders | description | labels | fields | creator_id | creator_name
- `--incident-ids` stringSlice — Filter by incident IDs (MongoDB ObjectIDs). At most 100 entries.
- `--is-my-team` bool — Restrict results to teams the caller belongs to. When true and the caller has no teams, the result set is empty.
- `--orderby` string — Field to sort the underlying incident set by. · enum: created_at
- `--query` string — Full-text query applied to incident title and description.
- `--responder-ids` intSlice — Filter by responder person IDs. At most 100 entries.
- `--seconds-to-ack-from` int64 — Lower bound (inclusive) on time-to-acknowledge, in seconds.
- `--seconds-to-ack-to` int64 — Upper bound (exclusive) on time-to-acknowledge, in seconds. Must be greater than 'seconds_to_ack_from' when both are set.
- `--seconds-to-close-from` int64 — Lower bound (inclusive) on time-to-close, in seconds.
- `--seconds-to-close-to` int64 — Upper bound (exclusive) on time-to-close, in seconds. Must be greater than 'seconds_to_close_from' when both are set.
- `--severities` stringSlice — Filter by severity. At most 3 entries. · enum: Critical | Warning | Info | Ok
- `--split-hours` bool — When true, metrics are split into 'work'/'sleep'/'off' hour buckets.
- `--start-time` string (required) — Start time, Unix seconds. Must be greater than 0. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-ids` intSlice — Filter by team IDs. At most 100 entries.
- `--time-zone` string — IANA time zone name used to interpret the time range (e.g. 'Asia/Shanghai'). Defaults to the account time zone.
- body-only (`--data`): fields (object); labels (object)

### team
Get team insight
- `--aggregate-unit` string — Aggregate metrics into time buckets. When set, the time range must cover at least 24 hours; 'day' additionally caps the range at 31 days. · enum: day | week | month
- `--asc` bool — Sort ascending when 'true', descending otherwise.
- `--channel-ids` intSlice — Filter by channel IDs. At most 100 entries.
- `--description-html-to-text` bool — Strip HTML markup from the description column when exporting.
- `--end-time` string (required) — End time, Unix seconds. Must be greater than 'start_time'. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--export-fields` stringSlice — Subset of CSV column keys to include in the export. At most 50 entries. Only used by the export endpoints. · enum: incident_id | title | severity | progress | channel_id | channel_name | team_id | team_name | created_at | seconds_to_ack | seconds_to_close | closed_by | engaged_seconds | hours | notifications | interruptions | acknowledgements | assignments | reassignments | escalations | manual_escalations | timeout_escalations | assigned_to | responders | description | labels | fields | creator_id | creator_name
- `--incident-ids` stringSlice — Filter by incident IDs (MongoDB ObjectIDs). At most 100 entries.
- `--is-my-team` bool — Restrict results to teams the caller belongs to. When true and the caller has no teams, the result set is empty.
- `--orderby` string — Field to sort the underlying incident set by. · enum: created_at
- `--query` string — Full-text query applied to incident title and description.
- `--responder-ids` intSlice — Filter by responder person IDs. At most 100 entries.
- `--seconds-to-ack-from` int64 — Lower bound (inclusive) on time-to-acknowledge, in seconds.
- `--seconds-to-ack-to` int64 — Upper bound (exclusive) on time-to-acknowledge, in seconds. Must be greater than 'seconds_to_ack_from' when both are set.
- `--seconds-to-close-from` int64 — Lower bound (inclusive) on time-to-close, in seconds.
- `--seconds-to-close-to` int64 — Upper bound (exclusive) on time-to-close, in seconds. Must be greater than 'seconds_to_close_from' when both are set.
- `--severities` stringSlice — Filter by severity. At most 3 entries. · enum: Critical | Warning | Info | Ok
- `--split-hours` bool — When true, metrics are split into 'work'/'sleep'/'off' hour buckets.
- `--start-time` string (required) — Start time, Unix seconds. Must be greater than 0. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-ids` intSlice — Filter by team IDs. At most 100 entries.
- `--time-zone` string — IANA time zone name used to interpret the time range (e.g. 'Asia/Shanghai'). Defaults to the account time zone.
- body-only (`--data`): fields (object); labels (object)

### team-export
Export team insight
- `--aggregate-unit` string — Aggregate metrics into time buckets. When set, the time range must cover at least 24 hours; 'day' additionally caps the range at 31 days. · enum: day | week | month
- `--asc` bool — Sort ascending when 'true', descending otherwise.
- `--channel-ids` intSlice — Filter by channel IDs. At most 100 entries.
- `--description-html-to-text` bool — Strip HTML markup from the description column when exporting.
- `--end-time` string (required) — End time, Unix seconds. Must be greater than 'start_time'. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--export-fields` stringSlice — Subset of CSV column keys to include in the export. At most 50 entries. Only used by the export endpoints. · enum: incident_id | title | severity | progress | channel_id | channel_name | team_id | team_name | created_at | seconds_to_ack | seconds_to_close | closed_by | engaged_seconds | hours | notifications | interruptions | acknowledgements | assignments | reassignments | escalations | manual_escalations | timeout_escalations | assigned_to | responders | description | labels | fields | creator_id | creator_name
- `--incident-ids` stringSlice — Filter by incident IDs (MongoDB ObjectIDs). At most 100 entries.
- `--is-my-team` bool — Restrict results to teams the caller belongs to. When true and the caller has no teams, the result set is empty.
- `--orderby` string — Field to sort the underlying incident set by. · enum: created_at
- `--query` string — Full-text query applied to incident title and description.
- `--responder-ids` intSlice — Filter by responder person IDs. At most 100 entries.
- `--seconds-to-ack-from` int64 — Lower bound (inclusive) on time-to-acknowledge, in seconds.
- `--seconds-to-ack-to` int64 — Upper bound (exclusive) on time-to-acknowledge, in seconds. Must be greater than 'seconds_to_ack_from' when both are set.
- `--seconds-to-close-from` int64 — Lower bound (inclusive) on time-to-close, in seconds.
- `--seconds-to-close-to` int64 — Upper bound (exclusive) on time-to-close, in seconds. Must be greater than 'seconds_to_close_from' when both are set.
- `--severities` stringSlice — Filter by severity. At most 3 entries. · enum: Critical | Warning | Info | Ok
- `--split-hours` bool — When true, metrics are split into 'work'/'sleep'/'off' hour buckets.
- `--start-time` string (required) — Start time, Unix seconds. Must be greater than 0. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-ids` intSlice — Filter by team IDs. At most 100 entries.
- `--time-zone` string — IANA time zone name used to interpret the time range (e.g. 'Asia/Shanghai'). Defaults to the account time zone.
- body-only (`--data`): fields (object); labels (object)

### top-alerts
Query top alert sources by label
- `--label` string
- `--limit` int
- `--since` string
- `--until` string

<!-- GENERATED:insight END -->

## Time-flag families (critical — wrong name = unknown flag error)

Two families with **identical value syntax** but different flag names:

| flag names | required? | commands |
|---|---|---|
| `--since` / `--until` | optional (defaults: `7d` / `now`) | `top-alerts`, `incidents` |
| `--start-time` / `--end-time` | **both required** | all others (`account`, `alert-topk-by-label`, `channel`, `channel-export`, `incident-export`, `incident-list`, `responder`, `responder-export`, `team`, `team-export`) |

Both families accept: relative duration (`30d`, `24h`), `now`, `+7d`, a date, or Unix seconds — **except** `incident-export`, which takes **epoch seconds only** for `--start-time`/`--end-time` (flag type is int64).

## Gotchas

- **Two time-flag families.** Passing `--since` to an `--start-time` command (or vice-versa) fails with `unknown flag`. See the table above.
- **`incident-export --start-time`/`--end-time` are epoch seconds only**, not relative strings — use `$(date -v-30d +%s)`. All other `--start-time` commands accept `30d`/`now`.
- **`top-alerts --label`** only accepts `check` or `resource`. Any other value (e.g. `integration_name`) returns HTTP 400.
- **Export commands output raw CSV, not JSON.** Redirect to a file; dumping CSV into context burns tokens and is unreadable. No `--limit`/`--page` — exports emit the full filtered set.
- **`insight incidents` and `incident-list` are siblings**, not the same. `incidents` uses `--since`/`--until`, paginates, and is token-light. `incident-list` uses `--start-time`/`--end-time`, adds `--severities`/`--responder-ids`/`--query`/cursor (`--search-after-ctx`), and is the filterable variant.
- **All `insight` commands hit the OLAP backend.** HTTP 500 means the backend is down — report it, do not retry.
- **Empty result is authoritative.** A zero-row response means no matching data for that scope/window — do not widen filters or re-query with shifted keywords.
- **`--aggregate-unit`** (on `account`, `alert-topk-by-label`, `channel`, `responder`, `team` and their exports) splits results into time buckets: `day` / `week` / `month`. When set, the window must span ≥24 h; `day` additionally caps the range at 31 days.

## Worked example — identify noisiest check sources

```bash
# Top-20 noisiest check sources in the past 7 days, sorted by raw event count
fduty insight alert-topk-by-label \
  --label check \
  --k 20 \
  --orderby total_alert_event_cnt \
  --start-time 7d --end-time now \
  --output-format toon
# → returns label, total_alert_cnt, total_alert_event_cnt per check
# Drill into a specific team: add --team-ids <id>
```
