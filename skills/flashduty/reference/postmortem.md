# flashduty incident post-mortems — command card

Prereq: `SKILL.md` read. `post-mortem-list` / `post-mortem-info` /
`post-mortem-template-list` / `post-mortem-template-info` are free reads;
init / reset / upsert / delete verbs mutate report state — confirm before
acting. `post-mortem-delete` and `post-mortem-template-delete` are
**irreversible**.

## Route here when

"复盘 / 复盘报告 / 复盘模板 / post-mortem / postmortem / post-incident
review / RCA report / retrospective" → this card. A post-mortem is a report
linked to 1–10 incidents; the incidents themselves (triage, resolve, merge)
are `reference/incident.md`. Key IDs: **`post-mortem-id` (string)** from
`post-mortem-list` (deterministic hash of the linked incident set);
**`template-id` (string)** from `post-mortem-template-list`;
**`incident-id` (24-char MongoDB ObjectID)** from `flashduty incident list`.

## Intent → verb

| want | verb |
|---|---|
| list post-mortems (by channel / team / time) | `post-mortem-list` |
| read one report | `post-mortem-info <post-mortem-id>` |
| start a report from incident(s) | `post-mortem-init <incident-id> [<id2>...]` |
| replace the report's Markdown body | `post-mortem-content-reset <post-mortem-id>` |
| set title | `post-mortem-title-reset <post-mortem-id>` |
| set timeline/severity/responders metadata | `post-mortem-basics-reset <post-mortem-id>` |
| set follow-up action items | `post-mortem-follow-ups-reset <post-mortem-id>` |
| publish or send back to draft | `post-mortem-status-reset <post-mortem-id>` |
| delete a report (IRREVERSIBLE) | `post-mortem-delete <post-mortem-id>` |
| list / read / upsert / delete templates | `post-mortem-template-list` / `post-mortem-template-info <template-id>` / `post-mortem-template-upsert` / `post-mortem-template-delete <template-id>` |

## Hot flow — write up a resolved incident

```bash
# 1. pick a template
flashduty incident post-mortem-template-list --output-format toon
# 2. initialize the report from the incident (returns post_mortem_id in meta)
flashduty incident post-mortem-init <incident-id> --template-id <template-id>
# 3. write the narrative — Markdown goes into a file, then content-reset
BODY_FILE=$(mktemp)
cat > "$BODY_FILE" <<'FDUTY_PM_7F3A9C2E_EOF'
## What happened
...
## Root cause
...
FDUTY_PM_7F3A9C2E_EOF
flashduty incident post-mortem-content-reset <post-mortem-id> --markdown-file "$BODY_FILE"
# 4. follow-ups + publish
flashduty incident post-mortem-follow-ups-reset <post-mortem-id> --follow-ups "Add alert on replica lag; tune failover timeout"
flashduty incident post-mortem-status-reset <post-mortem-id> --status published
```

<!-- GENERATED:incident[post-mortem] START · 由 flashduty __dump-commands 同步 · 勿手改 fence 内 -->

### post-mortem-basics-reset <post-mortem-id>
Update post-mortem basics
- `--incidents-earliest-start-seconds` string (required) — Unix timestamp in seconds for the earliest linked incident start time. (min 1) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--incidents-highest-severity` string (required) — Highest severity among the linked incidents. - 'Critical': critical. - 'Warning': warning. - 'Info': informational. - 'Ok': recovered. · enum: Critical | Warning | Info | Ok
- `--incidents-latest-close-seconds` string — Latest close time among the linked incidents, Unix timestamp in seconds. 0 (also when omitted) means still open. (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--incidents-total-duration-seconds` int64 — Total incident duration in seconds; 0 when omitted. (min 0)
- `<post-mortem-id>` (positional, required) string — Post-mortem ID; obtain it from 'POST /incident/post-mortem/list'.
- `--responder-ids` intSlice — Member IDs to store as the report's responders. Replaces the stored list; omit or send '[]' to clear it. IDs that do not resolve to a person are dropped.

### post-mortem-content-reset <post-mortem-id>
Reset post-mortem Markdown content
- `--expected-revision` int64
- `--idempotency-key` string
- `--markdown-file` string
- response: single object (`data` unwrapped to the top level) — fields: generation (integer); markdown_bytes (integer); markdown_sha256 (string); post_mortem_id (string); previous_generation (integer); previous_revision (integer); revision (integer)

### post-mortem-delete <post-mortem-id>
Delete post-mortem
- `<post-mortem-id>` (positional, required) string — Post-mortem report ID; obtain it from 'POST /incident/post-mortem/list'.

### post-mortem-follow-ups-reset <post-mortem-id>
Update post-mortem follow-ups
- `--follow-ups` string — Follow-ups as free text, replacing the stored string. A JSON array in the console's follow-up format (objects with 'title', 'status', 'priority', 'assignee_ids', …) is also synced into structured follow-up work items.
- `<post-mortem-id>` (positional, required) string — Post-mortem ID; obtain it from 'POST /incident/post-mortem/list'.

### post-mortem-info <post-mortem-id>
Get post-mortem
- `<post-mortem-id>` (positional, required) string — Post-mortem ID: a 32-character hex string derived from the account ID and the linked incident IDs.
- response: single object (`data` unwrapped to the top level) — fields: basics (object); content (object); follow_ups (string); has_active_incident (boolean); meta (object)

### post-mortem-init <incident-id> [<id2>...]
Initialize post-mortem
- `<incident-ids>` (positional, required) stringSlice — Incident IDs (MongoDB ObjectIDs) to link to the report, 1–10. All must exist in the account. The same set always maps to the same report.
- `--template-id` string (required) — Template ID from 'POST /incident/post-mortem/template/list', e.g. the built-in 'post_mortem_default_tmpl_en-us' or 'post_mortem_default_tmpl_zh-cn', or a custom template ID.
- response: same shape as `post-mortem-info <post-mortem-id>` above

### post-mortem-list
List post-mortems
- `--asc` bool — Ascending order when true.
- `--channel-ids` intSlice — Channel IDs to restrict the query to.
- `--created-at-end-seconds` string — Upper bound of post-mortem creation time (Unix timestamp in seconds). (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--created-at-start-seconds` string — Lower bound of post-mortem creation time (Unix timestamp in seconds). (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--limit` int64 — Page size, at most 100. With page-number paging, 'p × limit' may not exceed 10,000. (0-100)
- `--order-by` string — Sort field: 'created_at_seconds' creation time, 'updated_at_seconds' last update time. · enum: created_at_seconds | updated_at_seconds
- `--page` int64 — Page number starting at 1. (min 0)
- `--search-after-ctx` string — Cursor from a previous response for forward pagination.
- `--status` string — Optional status filter: 'drafting' returns only drafts, 'published' returns only published post-mortems. When omitted, post-mortems in all statuses are returned. · enum: drafting | published
- `--team-ids` intSlice — Team IDs to restrict the query to.
- response: `{items: [...], has_next_page, search_after_ctx, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); author_ids (array<integer>); channel_id (integer); channel_name (string); created_at_seconds (string); generation (integer); incident_ids (array<string>); is_private (boolean); media_count (integer); post_mortem_id (string); revision (integer); status (string); team_id (integer); template_id (string); title (string); updated_at_seconds (string)

### post-mortem-status-reset <post-mortem-id>
Update post-mortem status
- `<post-mortem-id>` (positional, required) string — Post-mortem ID; obtain it from 'POST /incident/post-mortem/list'.
- `--status` string (required) — Target report status. - 'drafting': being edited; the body can still be replaced with a content reset. - 'published': published; media files no longer referenced by the body are deleted. · enum: drafting | published

### post-mortem-template-delete <template-id>
Delete post-mortem template
- `<template-id>` (positional, required) string — ID of the custom template to delete; obtain it from 'POST /incident/post-mortem/template/list'.

### post-mortem-template-info <template-id>
Get post-mortem template detail
- `<template-id>` (positional, required) string — Template ID: a built-in ID such as 'post_mortem_default_tmpl_en-us', or a custom template ID from 'POST /incident/post-mortem/template/list'.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); content (string); content_markdown (string); created_at_seconds (string); description (string); name (string); team_id (integer); template_id (string); updated_at_seconds (string)

### post-mortem-template-list
List post-mortem templates
- `--asc` bool — Sort custom templates in ascending order when true; default false (newest first).
- `--limit` int64 — Page size, 1–100; 0 or omitted means 20. (0-100)
- `--order-by` string — Sort field for custom templates. Only 'created_at_seconds' is supported, which is also the default. · enum: created_at_seconds
- `--page` int64 — Page number starting at 1; 0 or omitted means 1. (min 0)
- `--search-after-ctx` string — Not supported by this operation; leave it unset and paginate with 'p'. If sent, it must be an integer string, and 'p' is then ignored (results start from the first page).
- response: `{items: [...], has_next_page, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); content (string); content_markdown (string); created_at_seconds (string); description (string); name (string); team_id (integer); template_id (string); updated_at_seconds (string)

### post-mortem-template-upsert
Create or update post-mortem template
- `--content` string (required) — Template body as BlockNote editor JSON (an array of blocks), serialized as a string. New reports created from this template start from this body.
- `--content-markdown` string — Markdown version of the body, used as the outline when AI generates a report. Empty when omitted.
- `--description` string — Template description; empty when omitted.
- `--name` string (required) — Template name shown in the console.
- `--team-id` int64 — Managing team ID. Required when creating; the team must exist. On update, a different team ID moves the template; omit it or send 0 to keep the current team.
- `--template-id` string — ID of the custom template to replace. Omit it to create a new template (the server generates the ID).
- response: same shape as `post-mortem-template-info <template-id>` above

### post-mortem-title-reset <post-mortem-id>
Update post-mortem title
- `<post-mortem-id>` (positional, required) string — Post-mortem ID; obtain it from 'POST /incident/post-mortem/list'.
- `--title` string (required) — New report title, at most 256 characters. (≤256 chars)

<!-- GENERATED:incident[post-mortem] END -->

## Gotchas

- **Post-mortem verbs live under the `incident` command group** —
  `flashduty incident post-mortem-list`; there is no standalone post-mortem
  group.
- **`post-mortem-id` ≠ `incident-id`**: init takes incident IDs and returns
  the report; every later verb takes the `post-mortem-id` from
  `post-mortem-list` / init's response `meta`.
- **`post-mortem-list` defaults to `--status published`** on the server —
  pass `--status drafting` to see unpublished reports.
- **Tie reports to one incident's channel** with
  `post-mortem-list --channel-ids <channel-id>` (channel_id from
  `incident detail`); there is no per-incident list verb.
- **`post-mortem-template-upsert` without `--template-id` creates** a new
  template; with it, updates in place. `--team-id` is required when creating.
