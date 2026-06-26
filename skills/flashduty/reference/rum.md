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
- body-only (`--data`): alerting (object); tracing (object)

### application-delete <application-id>
Delete application
- `<application-id>` (positional, required) string — RUM application ID.

### application-info <application-id>
Get application detail
- `<application-id>` (positional, required) string — RUM application ID.

### application-infos <application-id> [<id2>...]
Batch get applications
- `<application-ids>` (positional, required) stringSlice — Up to 200 application IDs.

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

### application-update <application-id>
Update application
- `<application-id>` (positional, required) string — Application ID to update.
- `--application-name` string — New application name.
- `--is-private` bool
- `--no-geo` bool
- `--no-ip` bool
- `--team-id` int64
- `--type` string — enum: browser | ios | android | react-native | flutter | kotlin-multiplatform | roku | unity
- body-only (`--data`): alerting (object); tracing (object)

### application-webhook-test <application-id>
Test application webhook
- `<application-id>` (positional, required) string — RUM application ID.
- `--webhook-url` string (required) — Webhook URL to receive the sample alert event.

### issue-info <issue-id>
Get issue detail
- `<issue-id>` (positional, required) string — Issue ID.

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

### issue-update <issue-id>
Update issue
- `<issue-id>` (positional, required) string — Issue ID to update.
- `--status` string — New status. · enum: for_review | reviewed | ignored | resolved
- `--suspected-cause` string — Suspected cause. · enum: api.failed_request | network.error | code.exception | code.invalid_object_access | code.invalid_argument | unknown

<!-- GENERATED:rum END -->

## Key enums & state machine

**`--type` (application-create / update) — closed enum:**
`browser` · `ios` · `android` · `react-native` · `flutter` · `kotlin-multiplatform` · `roku` · `unity`
No `miniprogram` / `wechat` — unsupported, do not guess a value.

**Issue `--status` (issue-update / issue-list `--statuses`):**
`for_review` → `reviewed` → `ignored` | `resolved`
Regression: a `resolved` issue that recurs gets a `regression{}` object on its record.

**Issue `--suspected-cause` / `--suspected-causes`:**
`api.failed_request` · `network.error` · `code.exception` · `code.invalid_object_access` · `code.invalid_argument` · `unknown`

**Application `status` (read-only on list/info):** `enabled` · `disabled` · `deleted`

## Gotchas

- **`issue-list` time flags are MILLISECOND epoch, both required.** Use `--start-time` / `--end-time` (NOT `--since`/`--until`, NOT seconds). Max range 183 days. Example: `$(date +%s)000` converts a seconds epoch to ms.
- **`application_id` ≠ `issue_id`.** `issue_id` comes from `issue-list` — never pass an `application_id` where `issue_id` is expected.
- **`application-create` positional:** `use` is `application-create <team-id>` — pass the team id as the first bare arg, NOT `--team-id`. Same pattern: `application-delete`, `application-info`, `application-infos`, `application-update`, `issue-info`, `issue-update` all take their primary id as positional. `application-list` and `issue-list` are all-flags.
- **`alerting` and `tracing` are nested objects** — configure them via `--data '{"alerting":{...},"tracing":{...}}'`; there are no flat flags for their sub-fields. Scalar flags (`--application-name`, `--type`, …) override matching `--data` keys.
- **Application records hold CONFIG only** — no traffic volume, error-rate, or session-count fields. For trend data, query `monit` RUM series.
- **Empty `issue-list` is authoritative** — a filter returning no items means no matching issues, not a missing feature. Do not widen the query or guess.
- **No `rum sourcemap` subcommand** — don't attempt it; it does not exist.

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
