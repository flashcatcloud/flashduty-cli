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
- `--application-name` string (required) — Application name. 1–40 characters. (1-40 chars)
- `--is-private` bool — Restrict access to team members only.
- `--no-geo` bool — Do not infer geographic location.
- `--no-ip` bool — Do not collect IP addresses.
- `<team-id>` (positional, required) int64 — Owning team ID. Get team IDs via 'POST /team/list'.
- `--type` string (required) — Platform identifier: | Value | Meaning | |---|---| | 'browser' | Web browser application (JavaScript SDK) | | 'ios' | Apple iOS application | | 'android' | Android application | | 'react-native' | React Native application | | 'flutter' | Flutter application | | 'kotlin-multiplatform' | Kotlin Multiplatform application | | 'roku' | Roku channel application | | 'unity' | Unity application | | 'miniprogram' | WeChat mini program | | 'harmony' | HarmonyOS application | | 'electron' | Electron desktop application | · enum: browser | ios | android | react-native | flutter | kotlin-multiplatform | roku | unity | miniprogram | harmony | electron
- body-only (`--data`): alerting (object); links (object); tracing (object)
- response: single object (`data` unwrapped to the top level) — fields: application_id (string); application_name (string); client_token (string)

### application-delete <application-id>
Delete application
- `<application-id>` (positional, required) string — RUM application ID. Get application IDs via 'POST /rum/application/list'.

### application-info <application-id>
Get application detail
- `<application-id>` (positional, required) string — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); alerting (object); application_id (string); application_name (string); client_token (string); created_at (string); created_by (integer); is_private (boolean); links (object); no_geo (boolean); no_ip (boolean); status (string); team_id (integer); tracing (object); type (string); updated_at (string); updated_by (integer)

### application-infos <application-id> [<id2>...]
Batch get applications
- `<application-ids>` (positional, required) stringSlice — Up to 200 application IDs. Get IDs via 'POST /rum/application/list'.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); alerting (object); application_id (string); application_name (string); client_token (string); created_at (string); created_by (integer); is_private (boolean); links (object); no_geo (boolean); no_ip (boolean); status (string); team_id (integer); tracing (object); type (string); updated_at (string); updated_by (integer)

### application-list
List applications
- `--asc` bool — Sort ascending if 'true'.
- `--is-my-team` bool — If 'true', return only applications belonging to the current user's teams.
- `--limit` int64 — Page size. Range: 1–100. Default: 20. (1-100)
- `--orderby` string — Sort field: 'created_at' (creation time) or 'updated_at' (last update time); defaults to 'updated_at' when omitted. · enum: created_at | updated_at
- `--page` int64 — Page number (1-based). Default: 1. (min 1)
- `--query` string — Substring match on the application name.
- `--search-after-ctx` string
- `--team-id` int64 — Filter by team ID. Get team IDs via 'POST /team/list'.
- response: `{items: [...], has_next_page, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); alerting (object); application_id (string); application_name (string); client_token (string); created_at (string); created_by (integer); is_private (boolean); links (object); no_geo (boolean); no_ip (boolean); status (string); team_id (integer); tracing (object); type (string); updated_at (string); updated_by (integer)

### application-update <application-id>
Update application
- `<application-id>` (positional, required) string — Application ID to update. Get application IDs via 'POST /rum/application/list'.
- `--application-name` string — New application name, 1–40 characters. Omit to leave unchanged. (1-40 chars)
- `--is-private` bool — Restrict access to members of the owning team; 'false' explicitly makes the application public. Omit to leave unchanged.
- `--no-geo` bool — When 'true', stop inferring geographic location from IP; when 'false', resume inferring it. Omit to leave unchanged.
- `--no-ip` bool — When 'true', stop collecting user IP addresses; when 'false', resume collecting them. Omit to leave unchanged.
- `--team-id` int64 — Owning team ID. Get team IDs via 'POST /team/list'. Omit to leave unchanged.
- `--type` string — Application type. Omit to leave unchanged. Platform identifier: | Value | Meaning | |---|---| | 'browser' | Web browser application (JavaScript SDK) | | 'ios' | Apple iOS application | | 'android' | Android application | | 'react-native' | React Native application | | 'flutter' | Flutter application | | 'kotlin-multiplatform' | Kotlin Multiplatform application | | 'roku' | Roku channel application | | 'unity' | Unity application | | 'miniprogram' | WeChat mini program | | 'harmony' | HarmonyOS application | | 'electron' | Electron desktop application | · enum: browser | ios | android | react-native | flutter | kotlin-multiplatform | roku | unity | miniprogram | harmony | electron
- body-only (`--data`): alerting (object); links (object); tracing (object)

### application-webhook-test <application-id>
Test application webhook
- `<application-id>` (positional, required) string — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--webhook-url` string (required) — Webhook URL to receive the sample alert event.
- response: single object (`data` unwrapped to the top level) — fields: message (string); ok (boolean); status_code (integer)

### data-query
Query RUM data
- `--end-time` int64 (required) — End of the query window, Unix epoch milliseconds. Maximum 31-day span.
- `--start-time` int64 (required) — Start of the query window, Unix epoch milliseconds.
- body-only (`--data`): queries (array<object>) (required)

### error-ingestion-rules-create <application-id>
Create an error ingestion rule
- `<application-id>` (positional, required) string — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--description` string — Rule description, up to 512 characters. (≤512 chars)
- `--rule-name` string (required) — Rule name, 1-128 characters. (1-128 chars)
- body-only (`--data`): filters (array<array<object>>) (required)
- response: single object (`data` unwrapped to the top level) — fields: rule_id (string); rule_name (string)

### error-ingestion-rules-delete
Delete an error ingestion rule
- `--application-id` string (required) — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--rule-id` string (required) — Rule ID. Get rule IDs via 'POST /rum/error-ingestion/rules/list'.

### error-ingestion-rules-disable
Disable an error ingestion rule
- `--application-id` string (required) — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--rule-id` string (required) — Rule ID. Get rule IDs via 'POST /rum/error-ingestion/rules/list'.

### error-ingestion-rules-enable
Enable an error ingestion rule
- `--application-id` string (required) — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--rule-id` string (required) — Rule ID. Get rule IDs via 'POST /rum/error-ingestion/rules/list'.

### error-ingestion-rules-history-list <application-id>
List error ingestion rule history
- `<application-id>` (positional, required) string — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--asc` bool — Sort ascending instead of the default descending order.
- `--limit` int64 — Page size. Default 20, capped at 100; values ≤ 0 fall back to the default. (max 100)
- `--orderby` string — Sort column: 'updated_at' or 'version'. Unrecognized values fall back to 'updated_at'.
- `--page` int64 — Zero-based page number. Default 0. (min 0)
- `--search-after-ctx` string
- response: `{items: [...], has_next_page, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: rules (array<object>); updated_at (string); updated_by (integer); updated_by_name (string); version (integer)

### error-ingestion-rules-history-revert <application-id>
Revert error ingestion rules to a history version
- `<application-id>` (positional, required) string — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--version` int64 (required) — History version number to revert to. Get versions via 'POST /rum/error-ingestion/rules/history/list'. (min 1)

### error-ingestion-rules-list <application-id>
List error ingestion rules
- `<application-id>` (positional, required) string — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: created_at (string); description (string); filters (array<array<object>>); rule_id (string); rule_name (string); status (string); updated_at (string)

### error-ingestion-rules-update
Update an error ingestion rule
- `--application-id` string (required) — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--description` string — New rule description, up to 512 characters. Omit to leave unchanged. (≤512 chars)
- `--rule-id` string (required) — Rule ID to update. Get rule IDs via 'POST /rum/error-ingestion/rules/list'.
- `--rule-name` string — New rule name, 1-128 characters. Omit to leave unchanged. (1-128 chars)
- body-only (`--data`): filters (array<array<object>>)

### facet-count
Count facet value distribution
- `--dql` string — RUM DQL filter expression applied before counting.
- `--end-time` int64 (required) — End of the time range, Unix epoch milliseconds. Maximum 31-day span.
- `--facet-key` string (required) — Field key whose value distribution to count; must be a registered field of the given 'scope'. List available fields via 'POST /rum/field/list'.
- `--kind` string — Symbol kind, used only when 'scope' is 'sourcemap' and only meaningful for 'android'/'harmony': 'mapping' (default) selects ProGuard/R8 mappings or ArkTS sourcemaps, 'native' selects native .so symbols. · enum: mapping | native
- `--limit` int64 — Maximum number of top values to return. Default 100, maximum 100. (max 100)
- `--scope` string (required) — RUM data scope to query. One of: | Value | Meaning | |---|---| | 'session' | User sessions | | 'view' | Page views | | 'action' | User actions | | 'error' | Error events | | 'resource' | Resource loads | | 'long_task' | Long tasks | | 'vital' | Performance vitals (Web Vitals, etc.) | | 'issue' | Aggregated error-tracking issues | | 'sourcemap' | Sourcemap / symbol files | · enum: session | view | action | error | resource | long_task | vital | issue | sourcemap
- `--sql` string — SQL WHERE clause (no SELECT) for additional filtering.
- `--start-time` int64 (required) — Start of the time range, Unix epoch milliseconds.
- `--type` string — Symbol-store platform, used only when 'scope' is 'sourcemap'. Defaults to 'browser' when omitted; 'web' and 'javascript' are accepted aliases of 'browser'. | Value | Store queried | |---|---| | 'browser' / 'web' / 'javascript' | JavaScript sourcemaps (excluding HarmonyOS ArkTS and React Native rows) | | 'android' | Android ProGuard/R8 mappings; with 'kind=native', Android NDK .so symbols | | 'ios' | iOS dSYM symbols | | 'miniprogram' | WeChat mini program sourcemaps | | 'harmony' | HarmonyOS ArkTS sourcemaps; with 'kind=native', HarmonyOS .so symbols | | 'flutter' | Flutter Dart AOT symbols | | 'electron' | Electron Breakpad symbols | | 'react-native' | React Native JS sourcemaps | · enum: browser | web | javascript | android | ios | miniprogram | harmony | flutter | electron | react-native
- body-only (`--data`): facet_value (any)
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: count (integer); facet_value (any)

### field-list
List RUM fields
- `--is-facet` bool — When omitted or 'null', return all fields. When 'true', return only facet-enabled fields. When 'false', return only fields that are not facet-enabled.
- `--scopes` stringSlice — Filter by RUM data scopes; unknown values are rejected with a parameter error. Omit to list fields of all scopes. | Value | Meaning | |---|---| | 'session' | User sessions | | 'view' | Page views | | 'action' | User actions | | 'error' | Error events | | 'resource' | Resource loads | | 'long_task' | Long tasks | | 'vital' | Performance vitals (Web Vitals, etc.) | | 'issue' | Aggregated error-tracking issues | | 'sourcemap' | Sourcemap / symbol files | · enum: session | view | action | error | resource | long_task | vital | issue | sourcemap
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); description (string); edit_able (boolean); enum_values (array<any>); field_key (string); field_name (string); group (string); is_facet (boolean); queryable (boolean); scopes (array<string>); show_type (string); status (string); unit_family (string); unit_name (string); value_type (string)

### issue-export
Export issues as CSV
- `--application-ids` stringSlice — Filter by application IDs. Get IDs via 'POST /rum/application/list'.
- `--asc` bool — Sort ascending when 'true'; descending by default.
- `--by-intersection` bool — When 'true', match by time-range overlap: export issues still active within the window ('last_seen_timestamp' >= 'start_time') even if created before it. Default 'false' exports only issues created inside the window.
- `--console-origin` string — Console origin used to build the 'issue_url' column, e.g. 'https://console.flashcat.cloud'. The service cannot infer it (SaaS, on-premises and dev releases answer on different origins).
- `--dql` string — DQL query for advanced filtering. Cannot be used with 'sql'.
- `--end-time` int64 (required) — End of the time range, Unix epoch milliseconds. Must be greater than 'start_time'; maximum range: 183 days.
- `--error-required` bool — If 'true', only export issues with at least one associated error event.
- `--export-fields` stringSlice — CSV columns to export, in the order they appear. Unknown keys are rejected with a parameter error; an empty array uses the default column set. | Value | Column content | |---|---| | 'issue_id' | Issue ID | | 'issue_url' | Console URL of the issue detail page (built from 'console_origin') | | 'application_name' | Owning application name | | 'service' | Service name | | 'error_type' | Error type | | 'error_message' | Error message | | 'status' | Triage status | | 'severity' | Severity | | 'is_crash' | Whether the error caused a crash | | 'error_count' | Error occurrence count | | 'session_count' | Affected session count | | 'first_seen_at' | First occurrence time (rendered in 'time_zone') | | 'first_seen_version' | Application version at first occurrence | | 'last_seen_at' | Most recent occurrence time (rendered in 'time_zone') | | 'last_seen_version' | Application version at the most recent occurrence | | 'versions' | All affected versions | | 'suspected_cause' | Suspected cause category | | 'resolved_at' | Resolution time (rendered in 'time_zone') | · enum: issue_id | issue_url | application_name | service | error_type | error_message | status | severity | is_crash | error_count | session_count | first_seen_at | first_seen_version | last_seen_at | last_seen_version | versions | suspected_cause | resolved_at
- `--limit` int64 — Page size (1–100). Ignored by the export — the row cap is fixed at 100. (1-100)
- `--orderby` string — Sort field; defaults to 'updated_at' when omitted. | Value | Meaning | |---|---| | 'created_at' | Issue creation time | | 'updated_at' | Last update time | | 'session_count' | Affected session count | | 'error_count' | Error occurrence count | | 'severity' | Severity rank ('Critical' > 'Warning' > 'Info') | · enum: created_at | updated_at | session_count | error_count | severity
- `--page` int64 — Page number (1-based). Ignored by the export — the first 100 matching rows are always read. (min 1)
- `--search-after-ctx` string
- `--sql` string — SQL-style query for advanced filtering. Cannot be used with 'dql'.
- `--start-time` int64 (required) — Start of the time range, Unix epoch milliseconds.
- `--statuses` stringSlice — Filter by triage status; any other value is rejected with a parameter error. | Value | Meaning | |---|---| | 'for_review' | Pending triage | | 'reviewed' | Reviewed | | 'ignored' | Ignored | | 'resolved' | Resolved | · enum: for_review | reviewed | ignored | resolved
- `--suspected-causes` stringSlice — Filter by suspected cause category. | Value | Meaning | |---|---| | 'api.failed_request' | API request failure (e.g. HTTP 4xx/5xx responses) | | 'network.error' | Network connectivity error (offline, aborted requests, etc.) | | 'code.exception' | Code exception (Syntax/Reference/Range and similar runtime errors) | | 'code.invalid_object_access' | Invalid object access (e.g. reading a property of 'undefined'/'null') | | 'code.invalid_argument' | Invalid argument passed to a function | | 'unknown' | Cause could not be determined | · enum: api.failed_request | network.error | code.exception | code.invalid_object_access | code.invalid_argument | unknown
- `--team-ids` intSlice — Filter by team IDs. Get team IDs via 'POST /team/list'.
- `--time-zone` string — IANA time zone used to render timestamps in the CSV, e.g. 'Asia/Shanghai' or 'UTC'. Default: 'Asia/Shanghai'.

### issue-info <issue-id>
Get issue detail
- `<issue-id>` (positional, required) string — Issue ID. Get issue IDs via 'POST /rum/issue/list'.
- response: single object (`data` unwrapped to the top level) — fields: age (integer); application_id (string); application_name (string); created_at (string); error (object); error_count (integer); first_seen (object); is_crash (boolean); issue_id (string); last_seen (object); regression (object); resolved_at (string); resolved_by (integer); service (string); session_count (integer); severity (string); status (string); suspected_cause (object); team_id (integer); updated_at (string); versions (array<string>)

### issue-list
List issues
- `--application-ids` stringSlice — Filter by application IDs. Get IDs via 'POST /rum/application/list'.
- `--asc` bool — Sort ascending when 'true'; descending by default.
- `--by-intersection` bool — When 'true', match by time-range overlap: return issues still active within the window ('last_seen_timestamp' >= 'start_time') even if created before it. Default 'false' returns only issues created inside the window.
- `--dql` string — DQL query for advanced filtering. Cannot be used with 'sql'.
- `--end-time` int64 (required) — End of the time range, Unix epoch milliseconds. Must be greater than 'start_time'; maximum range: 183 days.
- `--error-required` bool — If 'true', only return issues with at least one associated error event.
- `--limit` int64 — Page size. Range: 1–100. Default: 20. (1-100)
- `--orderby` string — Sort field; defaults to 'updated_at' when omitted. | Value | Meaning | |---|---| | 'created_at' | Issue creation time | | 'updated_at' | Last update time | | 'session_count' | Affected session count | | 'error_count' | Error occurrence count | | 'severity' | Severity rank ('Critical' > 'Warning' > 'Info') | · enum: created_at | updated_at | session_count | error_count | severity
- `--page` int64 — Page number (1-based). Default: 1. (min 1)
- `--search-after-ctx` string
- `--sql` string — SQL-style query for advanced filtering. Cannot be used with 'dql'.
- `--start-time` int64 (required) — Start of the time range, Unix epoch milliseconds.
- `--statuses` stringSlice — Filter by triage status; any other value is rejected with a parameter error. | Value | Meaning | |---|---| | 'for_review' | Pending triage | | 'reviewed' | Reviewed | | 'ignored' | Ignored | | 'resolved' | Resolved | · enum: for_review | reviewed | ignored | resolved
- `--suspected-causes` stringSlice — Filter by suspected cause category. | Value | Meaning | |---|---| | 'api.failed_request' | API request failure (e.g. HTTP 4xx/5xx responses) | | 'network.error' | Network connectivity error (offline, aborted requests, etc.) | | 'code.exception' | Code exception (Syntax/Reference/Range and similar runtime errors) | | 'code.invalid_object_access' | Invalid object access (e.g. reading a property of 'undefined'/'null') | | 'code.invalid_argument' | Invalid argument passed to a function | | 'unknown' | Cause could not be determined | · enum: api.failed_request | network.error | code.exception | code.invalid_object_access | code.invalid_argument | unknown
- `--team-ids` intSlice — Filter by team IDs. Get team IDs via 'POST /team/list'.
- response: `{items: [...], has_next_page, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: age (integer); application_id (string); application_name (string); created_at (string); error (object); error_count (integer); first_seen (object); is_crash (boolean); issue_id (string); last_seen (object); regression (object); resolved_at (string); resolved_by (integer); service (string); session_count (integer); severity (string); status (string); suspected_cause (object); team_id (integer); updated_at (string); versions (array<string>)

### issue-preset-severity-rules-create <application-id>
Create preset severity rule
- `<application-id>` (positional, required) string — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--description` string — Optional description, up to 512 characters. (≤512 chars)
- `--rule-name` string (required) — Rule display name, 1-128 characters. (1-128 chars)
- `--severity` string (required) — Severity to assign to errors matching this rule. · enum: Critical | Warning | Info
- body-only (`--data`): filters (array<array<object>>) (required)
- response: single object (`data` unwrapped to the top level) — fields: priority (integer); rule_id (string); rule_name (string)

### issue-preset-severity-rules-delete
Delete preset severity rule
- `--application-id` string (required) — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--rule-id` string (required) — Rule ID. Get rule IDs via 'POST /rum/issue/preset-severity/rules/list'.

### issue-preset-severity-rules-disable
Disable preset severity rule
- `--application-id` string (required) — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--rule-id` string (required) — Rule ID. Get rule IDs via 'POST /rum/issue/preset-severity/rules/list'.

### issue-preset-severity-rules-enable
Enable preset severity rule
- `--application-id` string (required) — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--rule-id` string (required) — Rule ID. Get rule IDs via 'POST /rum/issue/preset-severity/rules/list'.

### issue-preset-severity-rules-history-list <application-id>
List preset severity rule history
- `<application-id>` (positional, required) string — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--asc` bool — Sort ascending when true; results are descending by default.
- `--limit` int64 — Page size. Values <= 0 default to 20; values above 100 are capped at 100. (max 100)
- `--orderby` string — Sort column. Any other value (including omitted) falls back to 'updated_at'. · enum: updated_at | version
- `--page` int64 — Zero-based page number. (min 0)
- `--search-after-ctx` string
- response: same shape as `error-ingestion-rules-history-list <application-id>` above

### issue-preset-severity-rules-history-revert <application-id>
Revert preset severity rules to a history snapshot
- `<application-id>` (positional, required) string — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--version` int64 (required) — Snapshot version number to revert to. Get versions via 'POST /rum/issue/preset-severity/rules/history/list'. (min 1)

### issue-preset-severity-rules-list <application-id>
List preset severity rules
- `<application-id>` (positional, required) string — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: created_at (string); description (string); filters (array<array<object>>); priority (integer); rule_id (string); rule_name (string); severity (string); status (string); updated_at (string)

### issue-preset-severity-rules-reorder
Reorder preset severity rule
- `--application-id` string (required) — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--drag-rule-id` string (required) — ID of the rule being moved. Get rule IDs via 'POST /rum/issue/preset-severity/rules/list'.
- `--target-rule-id` string (required) — ID of the rule whose evaluation position 'drag_rule_id' moves to.

### issue-preset-severity-rules-update
Update preset severity rule
- `--application-id` string (required) — RUM application ID. Get application IDs via 'POST /rum/application/list'.
- `--description` string — New description, up to 512 characters. Omit to leave unchanged. (≤512 chars)
- `--rule-id` string (required) — Rule ID to update. Get rule IDs via 'POST /rum/issue/preset-severity/rules/list'.
- `--rule-name` string — New display name, 1-128 characters. Omit to leave unchanged. (1-128 chars)
- `--severity` string — New severity. Omit to leave unchanged. · enum: Critical | Warning | Info
- body-only (`--data`): filters (array<array<object>>)

### issue-update <issue-id>
Update issue
- `<issue-id>` (positional, required) string — Issue ID to update. Get issue IDs via 'POST /rum/issue/list'.
- `--status` string — New status. Setting 'resolved' records the resolution time and operator; switching away from 'resolved' clears them. | Value | Meaning | |---|---| | 'for_review' | Pending triage | | 'reviewed' | Reviewed | | 'ignored' | Ignored | | 'resolved' | Resolved | · enum: for_review | reviewed | ignored | resolved
- `--suspected-cause` string — New suspected cause; setting it marks the cause source as 'user', overriding the automatic classification. One of: | Value | Meaning | |---|---| | 'api.failed_request' | API request failure | | 'network.error' | Network connectivity error | | 'code.exception' | Code exception | | 'code.invalid_object_access' | Invalid object access | | 'code.invalid_argument' | Invalid argument | | 'unknown' | Unknown cause | · enum: api.failed_request | network.error | code.exception | code.invalid_object_access | code.invalid_argument | unknown

### resource-info
Get RUM resource info
- `--no-cache` bool — Bypass the short-lived cache of the resource record (plan version, quotas, status) and read it from source. Does not refresh the usage counts. Default 'false'.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); action.days (integer); created_at (string); error.days (integer); expired_at (string); long_task.days (integer); offering_id (integer); order_id (string); product (string); resource.days (integer); resource_id (string); resource_name (string); session.days (integer); session_investigate.free_cnt (integer); session_investigate.used_cnt (integer); session_limit_reached (boolean); session_measure.free_cnt (integer); session_measure.used_cnt (integer); session_replay.free_cnt (integer); session_replay.used_cnt (integer); status (string); updated_at (string); version (string); view.days (integer); window_end_time (string); window_start_time (string)

### session-replay-metadata <session-id>
Get session replay metadata
- `<session-id>` (positional, required) string — RUM session ID (the 'session.id' attribute on RUM events).
- `--ts` int64 — Unix timestamp in milliseconds of the session start time. Optional; disambiguates when a session ID has been reused across different time windows.
- response: single object (`data` unwrapped to the top level) — fields: application (object); device (object); foreground_periods (array<object>); session (object); views (array<object>)

### session-replay-segments <session-id>
List session replay segments
- `--limit` int64 — Maximum number of segments to return. 1-99, default 20. (1-99)
- `--search-after-ctx` string — Pagination cursor from a previous call. Take it from the 'search_after_ctx' field (URL mode) or the 'X-Search-After-Ctx' response header (streaming mode).
- `<session-id>` (positional, required) string — RUM session ID (the 'session.id' attribute on RUM events).
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
