# fduty rum — command card

Prereq: `SKILL.md` read. Read verbs are free. `application-create` / `application-update` / `application-delete` / `issue-update` mutate state — confirm before running. `application-delete` is **irreversible**.

## Route here when

"前端监控 / RUM / web应用 / iOS应用 / Android应用 / 前端报错 / JS报错 / 崩溃 / crash / error tracking / RUM application / real user monitoring" → **rum**, NOT `monit` (server-side rules), NOT `channel`, NOT `team`. You need two distinct IDs: **`application_id` (string)** from `application-list`, and **`issue_id` (string)** from `issue-list` — they are NOT interchangeable.

## Intent → verb

| want | verb |
|---|---|
| find a RUM app by name / list all | `application-list` |
| config detail for one app | `application-info` |
| config detail for several apps at once | `application-infos` |
| **create** a new RUM app | `application-create` |
| edit app name / privacy / tracing / alerting | `application-update` |
| delete an app | `application-delete` |
| list front-end error issues (with time window) | `issue-list` |
| full detail of one error issue | `issue-info` |
| mark issue resolved / label cause | `issue-update` |
| run raw SQL-style queries over RUM data | `data-query` |
| top values for one facet field, by occurrence count | `facet-count` |
| browse facet-enabled RUM fields | `facet-list` |
| browse all RUM field definitions | `field-list` |
| send a test alert to an app's webhook | `application-webhook-test` |
| session replay metadata (app/device/views for a session) | `session-replay-metadata` |
| page through a session's replay segments | `session-replay-segments` |

## Hot flow — triage front-end errors

```bash
# 1. find the app (application_id is a string)
fduty rum application-list --query "checkout" --output-format toon

# 2. list open errors in the last 7 days (both time flags required, MILLISECOND epoch)
NOW=$(date +%s000)
WEEK_AGO=$(( $(date +%s) - 604800 ))000
fduty rum issue-list \
  --application-ids <application_id> \
  --start-time $WEEK_AGO --end-time $NOW \
  --statuses for_review --orderby error_count \
  --output-format toon

# 3. get full detail of the top issue
fduty rum issue-info <issue_id> --output-format toon

# 4. mark resolved after fix is confirmed
fduty rum issue-update <issue_id> --status resolved --suspected-cause code.exception
```

## Hot flow — create a new RUM application

```bash
# team-id is POSITIONAL (use: "application-create <team-id>"); other fields are flags
fduty rum application-create <team_id> \
  --application-name "Checkout Web" \
  --type browser
# → returns application_id + client_token for SDK init
```

<!-- GENERATED:rum START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### application-create <team-id>
Create application
- `--application-name` string (required) — Application name. 1–40 characters.
- `--is-private` bool — Restrict access to team members only.
- `--no-geo` bool — Do not infer geographic location.
- `--no-ip` bool — Do not collect IP addresses.
- `<team-id>` (positional, required) int64 — Owning team ID.
- `--type` string (required) — Application type. · enum: browser | ios | android | react-native | flutter | kotlin-multiplatform | roku | unity
- body-only (`--data`): alerting (object); links (object); tracing (object)
- response: single object (`data` unwrapped to the top level) — fields: application_id (string); application_name (string); client_token (string)

### application-delete <application-id>
Delete application
- `<application-id>` (positional, required) string — RUM application ID.

### application-info <application-id>
Get application detail
- `<application-id>` (positional, required) string — RUM application ID.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); alerting (object); application_id (string); application_name (string); client_token (string); created_at (integer); created_by (integer); is_private (boolean); links (object); no_geo (boolean); no_ip (boolean); status (string); team_id (integer); tracing (object); type (string); updated_at (integer); updated_by (integer)

### application-infos <application-id> [<id2>...]
Batch get applications
- `<application-ids>` (positional, required) stringSlice — Up to 200 application IDs.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); alerting (object); application_id (string); application_name (string); client_token (string); created_at (integer); created_by (integer); is_private (boolean); links (object); no_geo (boolean); no_ip (boolean); status (string); team_id (integer); tracing (object); type (string); updated_at (integer); updated_by (integer)

### application-list
List applications
- `--asc` bool — Sort ascending if 'true'.
- `--is-my-team` bool — If 'true', return only applications belonging to the current user's teams.
- `--limit` int64 — Page size. Range: 1–100. Default: 20.
- `--orderby` string — Sort field. · enum: created_at | updated_at
- `--page` int64 — Page number (1-based). Default: 1.
- `--query` string — Search query to filter by application name.
- `--search-after-ctx` string
- `--team-id` int64 — Filter by team ID.
- response: `{items: [...], has_next_page, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); alerting (object); application_id (string); application_name (string); client_token (string); created_at (integer); created_by (integer); is_private (boolean); links (object); no_geo (boolean); no_ip (boolean); status (string); team_id (integer); tracing (object); type (string); updated_at (integer); updated_by (integer)

### application-update <application-id>
Update application
- `<application-id>` (positional, required) string — Application ID to update.
- `--application-name` string — New application name.
- `--is-private` bool
- `--no-geo` bool
- `--no-ip` bool
- `--team-id` int64
- `--type` string — enum: browser | ios | android | react-native | flutter | kotlin-multiplatform | roku | unity
- body-only (`--data`): alerting (object); links (object); tracing (object)

### application-webhook-test <application-id>
Test application webhook
- `<application-id>` (positional, required) string — RUM application ID.
- `--webhook-url` string (required) — Webhook URL to receive the sample alert event.
- response: single object (`data` unwrapped to the top level) — fields: message (string); ok (boolean); status_code (integer)

### data-query
Query RUM data
- `--end-time` int64 (required) — End of the query window, Unix epoch milliseconds. Maximum 31-day span.
- `--start-time` int64 (required) — Start of the query window, Unix epoch milliseconds.
- body-only (`--data`): queries (array<object>) (required)

### error-ingestion-rules-create <application-id>
Create an error ingestion rule
- `<application-id>` (positional, required) string — RUM application ID.
- `--description` string — Rule description, up to 512 characters. (≤512 chars)
- `--rule-name` string (required) — Rule name, 1-128 characters. (1-128 chars)
- body-only (`--data`): filters (array<array>) (required)
- response: single object (`data` unwrapped to the top level) — fields: rule_id (string); rule_name (string)

### error-ingestion-rules-delete
Delete an error ingestion rule
- `--application-id` string (required) — RUM application ID.
- `--rule-id` string (required) — Rule ID.

### error-ingestion-rules-disable
Disable an error ingestion rule
- `--application-id` string (required) — RUM application ID.
- `--rule-id` string (required) — Rule ID.

### error-ingestion-rules-enable
Enable an error ingestion rule
- `--application-id` string (required) — RUM application ID.
- `--rule-id` string (required) — Rule ID.

### error-ingestion-rules-history-list <application-id>
List error ingestion rule history
- `<application-id>` (positional, required) string — RUM application ID.
- `--asc` bool — Sort ascending instead of the default descending order.
- `--limit` int64 — Page size. Default 20, capped at 100; values ≤ 0 fall back to the default. (max 100)
- `--orderby` string — Sort column: 'updated_at' or 'version'. Unrecognized values fall back to 'updated_at'.
- `--page` int64 — Zero-based page number. Default 0. (min 0)
- `--search-after-ctx` string
- response: `{items: [...], has_next_page, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: rules (array<object>); updated_at (integer); updated_by (integer); updated_by_name (string); version (integer)

### error-ingestion-rules-history-revert <application-id>
Revert error ingestion rules to a history version
- `<application-id>` (positional, required) string — RUM application ID.
- `--version` int64 (required) — History version number to revert to. (min 1)

### error-ingestion-rules-list <application-id>
List error ingestion rules
- `<application-id>` (positional, required) string — RUM application ID.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: created_at (integer); description (string); filters (array<array>); rule_id (string); rule_name (string); status (string); updated_at (integer)

### error-ingestion-rules-update
Update an error ingestion rule
- `--application-id` string (required) — RUM application ID.
- `--description` string — New rule description, up to 512 characters. Omit to leave unchanged. (≤512 chars)
- `--rule-id` string (required) — Rule ID to update.
- `--rule-name` string — New rule name, 1-128 characters. Omit to leave unchanged. (1-128 chars)
- body-only (`--data`): filters (array<array>)

### facet-count
Count facet value distribution
- `--dql` string — RUM DQL filter expression applied before counting.
- `--end-time` int64 (required) — End of the time range, Unix epoch milliseconds. Maximum 31-day span.
- `--facet-key` string (required) — The field key to count value distribution for.
- `--limit` int64 — Maximum number of top values to return. Default 100, maximum 100. (max 100)
- `--scope` string (required) — RUM data scope to query. · enum: session | view | action | error | resource | long_task | vital | issue | sourcemap
- `--sql` string — SQL WHERE clause (no SELECT) for additional filtering.
- `--start-time` int64 (required) — Start of the time range, Unix epoch milliseconds.
- body-only (`--data`): facet_value (any)
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: count (integer); facet_value (any)

### field-list
List RUM fields
- `--is-facet` bool — When true, return only facet-enabled fields. When false or omitted, return all fields.
- `--scopes` stringSlice — Filter by RUM data scopes. Valid values: 'session', 'view', 'action', 'error', 'resource', 'long_task', 'vital', 'issue', 'sourcemap'.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); description (string); edit_able (boolean); enum_values (array<any>); field_key (string); field_name (string); group (string); is_facet (boolean); queryable (boolean); scopes (array<string>); show_type (string); status (string); unit_family (string); unit_name (string); value_type (string)

### issue-info <issue-id>
Get issue detail
- `<issue-id>` (positional, required) string — Issue ID.
- response: single object (`data` unwrapped to the top level) — fields: age (integer); application_id (string); application_name (string); created_at (integer); error (object); error_count (integer); first_seen (object); is_crash (boolean); issue_id (string); last_seen (object); regression (object); resolved_at (integer); resolved_by (integer); service (string); session_count (integer); severity (string); status (string); suspected_cause (object); team_id (integer); updated_at (integer); versions (array<string>)

### issue-list
List issues
- `--application-ids` stringSlice — Filter by application IDs.
- `--asc` bool
- `--by-intersection` bool
- `--dql` string — DQL query for advanced filtering. Cannot be used with 'sql'.
- `--end-time` int64 (required) — End of time range, millisecond timestamp. Maximum range: 183 days.
- `--error-required` bool — If 'true', only return issues with at least one associated error event.
- `--limit` int64 — Page size. Range: 1–100. Default: 20.
- `--orderby` string — enum: created_at | updated_at | session_count | error_count
- `--page` int64 — Page number. Default: 1.
- `--search-after-ctx` string
- `--sql` string — SQL-style query for advanced filtering. Cannot be used with 'dql'.
- `--start-time` int64 (required) — Start of time range, millisecond timestamp.
- `--statuses` stringSlice — Filter by statuses. · enum: for_review | reviewed | ignored | resolved
- `--suspected-causes` stringSlice — Filter by suspected causes.
- `--team-ids` intSlice — Filter by team IDs.
- response: `{items: [...], has_next_page, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: age (integer); application_id (string); application_name (string); created_at (integer); error (object); error_count (integer); first_seen (object); is_crash (boolean); issue_id (string); last_seen (object); regression (object); resolved_at (integer); resolved_by (integer); service (string); session_count (integer); severity (string); status (string); suspected_cause (object); team_id (integer); updated_at (integer); versions (array<string>)

### issue-preset-severity-rules-create <application-id>
Create preset severity rule
- `<application-id>` (positional, required) string — RUM application ID.
- `--description` string — Optional description, up to 512 characters. (≤512 chars)
- `--rule-name` string (required) — Rule display name, 1-128 characters. (1-128 chars)
- `--severity` string (required) — Severity to assign to errors matching this rule. · enum: Critical | Warning | Info
- body-only (`--data`): filters (array<array>) (required)
- response: single object (`data` unwrapped to the top level) — fields: priority (integer); rule_id (string); rule_name (string)

### issue-preset-severity-rules-delete
Delete preset severity rule
- `--application-id` string (required) — RUM application ID.
- `--rule-id` string (required) — Rule ID.

### issue-preset-severity-rules-disable
Disable preset severity rule
- `--application-id` string (required) — RUM application ID.
- `--rule-id` string (required) — Rule ID.

### issue-preset-severity-rules-enable
Enable preset severity rule
- `--application-id` string (required) — RUM application ID.
- `--rule-id` string (required) — Rule ID.

### issue-preset-severity-rules-history-list <application-id>
List preset severity rule history
- `<application-id>` (positional, required) string — RUM application ID.
- `--asc` bool — Sort ascending when true; results are descending by default.
- `--limit` int64 — Page size. Values <= 0 default to 20; values above 100 are capped at 100. (max 100)
- `--orderby` string — Sort column. Any other value (including omitted) falls back to 'updated_at'. · enum: updated_at | version
- `--page` int64 — Zero-based page number. (min 0)
- `--search-after-ctx` string
- response: same shape as `error-ingestion-rules-history-list <application-id>` above

### issue-preset-severity-rules-history-revert <application-id>
Revert preset severity rules to a history snapshot
- `<application-id>` (positional, required) string — RUM application ID.
- `--version` int64 (required) — Version number of the snapshot to revert to. (min 1)

### issue-preset-severity-rules-list <application-id>
List preset severity rules
- `<application-id>` (positional, required) string — RUM application ID.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: created_at (integer); description (string); filters (array<array>); priority (integer); rule_id (string); rule_name (string); severity (string); status (string); updated_at (integer)

### issue-preset-severity-rules-reorder
Reorder preset severity rule
- `--application-id` string (required) — RUM application ID.
- `--drag-rule-id` string (required) — ID of the rule being moved.
- `--target-rule-id` string (required) — ID of the rule whose evaluation position 'drag_rule_id' moves to.

### issue-preset-severity-rules-update
Update preset severity rule
- `--application-id` string (required) — RUM application ID.
- `--description` string — New description, up to 512 characters. Omit to leave unchanged. (≤512 chars)
- `--rule-id` string (required) — Rule ID to update.
- `--rule-name` string — New display name, 1-128 characters. Omit to leave unchanged. (1-128 chars)
- `--severity` string — New severity. Omit to leave unchanged. · enum: Critical | Warning | Info
- body-only (`--data`): filters (array<array>)

### issue-update <issue-id>
Update issue
- `<issue-id>` (positional, required) string — Issue ID to update.
- `--status` string — New status. · enum: for_review | reviewed | ignored | resolved
- `--suspected-cause` string — Suspected cause. · enum: api.failed_request | network.error | code.exception | code.invalid_object_access | code.invalid_argument | unknown

### resource-info
Get RUM resource info
- `--no-cache` bool — Bypass the short-lived cache of the resource record (plan version, quotas, status) and read it from source. Does not refresh the usage counts. Default 'false'.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); action.days (integer); created_at (integer); error.days (integer); expired_at (integer); long_task.days (integer); offering_id (integer); order_id (string); product (string); resource.days (integer); resource_id (string); resource_name (string); session.days (integer); session_investigate.free_cnt (integer); session_investigate.used_cnt (integer); session_limit_reached (boolean); session_measure.free_cnt (integer); session_measure.used_cnt (integer); session_replay.free_cnt (integer); session_replay.used_cnt (integer); status (string); updated_at (integer); version (string); view.days (integer); window_end_time (integer); window_start_time (integer)

### session-replay-metadata <session-id>
Get session replay metadata
- `<session-id>` (positional, required) string — RUM session ID.
- `--ts` int64 — Unix timestamp in milliseconds of the session start time. Optional; disambiguates when a session ID has been reused across different time windows.
- response: single object (`data` unwrapped to the top level) — fields: application (object); device (object); foreground_periods (array<object>); session (object); views (array<object>)

### session-replay-segments <session-id>
List session replay segments
- `--limit` int64 — Maximum number of segments to return. 1-99, default 20. (1-99)
- `--search-after-ctx` string — Pagination cursor from a previous call. Take it from the 'search_after_ctx' field (URL mode) or the 'X-Search-After-Ctx' response header (streaming mode).
- `<session-id>` (positional, required) string — RUM session ID.
- `--ts` int64 — Unix timestamp in milliseconds. When set (and 'search_after_ctx' is empty), seeks to the most recent full-snapshot segment at or before this time instead of starting from the beginning.
- `--url-mode` bool — When 'true', return presigned download URLs as a JSON envelope instead of streaming segment bytes. Defaults to 'false'.
- `--view-id` string — Restrict results to segments belonging to this view. Omit to page through the entire session.
- response: single object (`data` unwrapped to the top level) — fields: items (array<string>); search_after_ctx (string)

<!-- GENERATED:rum END -->

## Key enums & state machine

**`--type` (application-create / update) — closed enum:**
`browser` · `ios` · `android` · `react-native` · `flutter` · `kotlin-multiplatform` · `roku` · `unity`
No `miniprogram` / `wechat` — you cannot create an application with these; do not guess a value. (Session/view `source` on `session-replay-metadata` does include `miniprogram` — that enum describes what recorded the data, not what you can create.)

**Issue `--status` (issue-update / issue-list `--statuses`):**
`for_review` → `reviewed` → `ignored` | `resolved`
Regression: a `resolved` issue that recurs gets a `regression{}` object on its record.

**Issue `--suspected-cause` / `--suspected-causes`:**
`api.failed_request` · `network.error` · `code.exception` · `code.invalid_object_access` · `code.invalid_argument` · `unknown`

**Application `status` (read-only on list/info):** `enabled` · `disabled` · `deleted`

## Gotchas

- **`issue-list` time flags are MILLISECOND epoch, both required.** Use `--start-time` / `--end-time` (NOT `--since`/`--until`, NOT seconds). Max range 183 days. Example: `$(date +%s)000` converts a seconds epoch to ms.
- **`application_id` ≠ `issue_id`.** `issue_id` comes from `issue-list` — never pass an `application_id` where `issue_id` is expected.
- **`application-create <team-id>` can be passed positionally or via `--team-id`** — both work; positional is shorter. Same pattern on `application-delete`, `application-info`, `application-update`, `issue-info`, `issue-update`: each takes its primary id either as the bare positional shown in the fence heading, or via the matching `--application-id`/`--issue-id` flag. `application-infos` only has the plural `--application-ids` as its flag alternative (comma-separated, vs space-separated positionals). If both positional and flag are given, the flag wins. `application-list` and `issue-list` are all-flags.
- **`alerting` and `tracing` are nested objects** — configure them via `--data '{"alerting":{...},"tracing":{...}}'`; there are no flat flags for their sub-fields. Scalar flags (`--application-name`, `--type`, …) override matching `--data` keys.
- **Application records hold CONFIG only** — no traffic volume, error-rate, or session-count fields. For trend data, query `monit` RUM series.
- **Empty `issue-list` is authoritative** — a filter returning no items means no matching issues, not a missing feature. Do not widen the query or guess.
- **No `rum sourcemap` subcommand** — sourcemap lookup and stack enrichment are top-level: read `reference/sourcemap.md` and use `fduty sourcemap ...`.

## Worked example

```bash
# Find the worst unreviewed crash in the "payment" app this week, then mark it resolved
APP_ID=$(fduty rum application-list --query "payment" --output-format json | jq -r '.items[0].application_id')
NOW=$(date +%s000)
WEEK_AGO=$(( $(date +%s) - 604800 ))000
fduty rum issue-list \
  --application-ids "$APP_ID" \
  --start-time $WEEK_AGO --end-time $NOW \
  --statuses for_review --orderby session_count \
  --limit 1 --output-format json | jq -r '.items[0].issue_id'
# → paste the returned issue_id below
fduty rum issue-update <issue_id> --status resolved --suspected-cause code.exception
```
