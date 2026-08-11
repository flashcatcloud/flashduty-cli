# fduty incident post-mortems — command card

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
**`incident-id` (24-char MongoDB ObjectID)** from `fduty incident list`.

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
fduty incident post-mortem-template-list --output-format toon
# 2. initialize the report from the incident (returns post_mortem_id in meta)
fduty incident post-mortem-init <incident-id> --template-id <template-id>
# 3. write the narrative — Markdown goes into a file, then content-reset
BODY_FILE=$(mktemp)
cat > "$BODY_FILE" <<'FDUTY_PM_7F3A9C2E_EOF'
## What happened
...
## Root cause
...
FDUTY_PM_7F3A9C2E_EOF
fduty incident post-mortem-content-reset <post-mortem-id> --markdown-file "$BODY_FILE"
# 4. follow-ups + publish
fduty incident post-mortem-follow-ups-reset <post-mortem-id> --follow-ups "Add alert on replica lag; tune failover timeout"
fduty incident post-mortem-status-reset <post-mortem-id> --status published
```

<!-- GENERATED:incident[post-mortem] START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### post-mortem-basics-reset <post-mortem-id>
Update post-mortem basics
- `--incidents-earliest-start-seconds` string (required) — Unix timestamp in seconds for the earliest linked incident start time. (min 1) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--incidents-highest-severity` string (required) — Highest severity among linked incidents.
- `--incidents-latest-close-seconds` string — Unix timestamp in seconds for the latest linked incident close time. 0 when still open. (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--incidents-total-duration-seconds` int64 — Total incident duration in seconds. (min 0)
- `<post-mortem-id>` (positional, required) string — Post-mortem ID; obtain it from 'POST /incident/post-mortem/list'.
- `--responder-ids` intSlice — Responder member IDs to store on the report.

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
- `--follow-ups` string — Follow-up action items as free text.
- `<post-mortem-id>` (positional, required) string — Post-mortem ID; obtain it from 'POST /incident/post-mortem/list'.

### post-mortem-info <post-mortem-id>
Get post-mortem
- `<post-mortem-id>` (positional, required) string — Post-mortem ID. Deterministic hash derived from account ID and the set of linked incident IDs.
- response: single object (`data` unwrapped to the top level) — fields: basics (object); content (object); follow_ups (string); meta (object)

### post-mortem-init <incident-id> [<id2>...]
Initialize post-mortem
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to link to the report. 1-10 incidents.
- `--template-id` string (required) — Template ID used to initialize the report.
- response: same shape as `post-mortem-info <post-mortem-id>` above

### post-mortem-list
List post-mortems
- `--asc` bool — Ascending order when true.
- `--channel-ids` intSlice — Channel IDs to restrict the query to.
- `--created-at-end-seconds` string — Upper bound of post-mortem creation time (Unix timestamp in seconds). (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--created-at-start-seconds` string — Lower bound of post-mortem creation time (Unix timestamp in seconds). (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--limit` int64 — Page size, at most 100. (0-100)
- `--order-by` string — Field used to order results. · enum: created_at_seconds | updated_at_seconds
- `--page` int64 — Page number starting at 1. (min 0)
- `--search-after-ctx` string — Cursor from a previous response for forward pagination.
- `--status` string — Report status. Defaults to 'published' on the server when omitted. · enum: drafting | published
- `--team-ids` intSlice — Team IDs to restrict the query to.
- response: `{items: [...], has_next_page, search_after_ctx, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); author_ids (array<integer>); channel_id (integer); channel_name (string); created_at_seconds (integer); generation (integer); incident_ids (array<string>); is_private (boolean); media_count (integer); post_mortem_id (string); revision (integer); status (string); team_id (integer); template_id (string); title (string); updated_at_seconds (integer)

### post-mortem-status-reset <post-mortem-id>
Update post-mortem status
- `<post-mortem-id>` (positional, required) string — Post-mortem ID; obtain it from 'POST /incident/post-mortem/list'.
- `--status` string (required) — Target report status: 'drafting' draft, 'published' published. · enum: drafting | published

### post-mortem-template-delete <template-id>
Delete post-mortem template
- `<template-id>` (positional, required) string — Template ID; obtain it from 'POST /incident/post-mortem/template/list'.

### post-mortem-template-info <template-id>
Get post-mortem template detail
- `<template-id>` (positional, required) string — Template ID.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); content (string); content_markdown (string); created_at_seconds (integer); description (string); name (string); team_id (integer); template_id (string); updated_at_seconds (integer)

### post-mortem-template-list
List post-mortem templates
- `--asc` bool — Ascending order when true.
- `--limit` int64 — Page size, at most 100. (0-100)
- `--order-by` string — Field used to order results. · enum: created_at_seconds
- `--page` int64 — Page number starting at 1. (min 0)
- `--search-after-ctx` string — Cursor from a previous response for forward pagination.
- response: `{items: [...], has_next_page, search_after_ctx, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); content (string); content_markdown (string); created_at_seconds (integer); description (string); name (string); team_id (integer); template_id (string); updated_at_seconds (integer)

### post-mortem-template-upsert
Create or update post-mortem template
- `--content` string (required) — BlockNote JSON template content.
- `--content-markdown` string — Markdown version of the template content.
- `--description` string — Template description.
- `--name` string (required) — Template name.
- `--team-id` int64 — Managing team ID. Required when creating a custom template.
- `--template-id` string — Template ID. Omit to create a new template; provide it to update an existing template.
- response: same shape as `post-mortem-template-info <template-id>` above

### post-mortem-title-reset <post-mortem-id>
Update post-mortem title
- `<post-mortem-id>` (positional, required) string — Post-mortem ID; obtain it from 'POST /incident/post-mortem/list'.
- `--title` string (required) — New report title.

<!-- GENERATED:incident[post-mortem] END -->

## Gotchas

- **Post-mortem verbs live under the `incident` command group** —
  `fduty incident post-mortem-list`; there is no standalone post-mortem
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
