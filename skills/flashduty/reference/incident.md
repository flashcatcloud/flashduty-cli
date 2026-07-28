# fduty incident — command card

Prereq: `SKILL.md` read. Read verbs are free. **Mutating verbs notify responders or alter state** — confirm scope first. `merge` and `remove` are **irreversible**; `remove` permanently deletes.

## Route here when

"告警 / 故障 / 事件 / 响应 / 值班 / incident / page / outage / triage / acknowledge / resolve / snooze / escalate / post-mortem" → **incident**, NOT `alert` (alert = deduplicated signal; incident = actionable item responders work). NOT `insight` (metrics/MTTA/MTTR). You need **`incident_id` (24-char MongoDB ObjectID)** for most verbs — not the 6-char `num` shown in the UI. **`detail` and `get` are the exception and accept either** (a num auto-resolves via a 30-day lookback). For any other verb, if you only have a num, use `incident info --num <num>` first.

## Intent → verb

| want | verb |
|---|---|
| list / search active incidents | `list` |
| CSV export of incidents | `fduty insight incident-export` |
| look up by 6-char UI num | `info --num <num>` |
| full detail + AI summary for a 24-char id | `detail <id>` (narrative) or `info --incident-id <id>` (same endpoint) |
| get structured data for one or more ids | `get <id> [<id2>...]` |
| contributing alerts | `alerts <id>` |
| full event history (short) | `timeline <id>` |
| paginated event history | `feed <id>` |
| past similar incidents | `similar <id>` |
| historical incidents related to this one | `past-list <incident-id>` |
| create a manual incident | `create` |
| edit title/description/severity | `update <id>` |
| edit title/description/severity/impact/root-cause/resolution | `reset <incident-id>` |
| set one custom field | `field-reset <incident-id>` |
| acknowledge (Triggered → Processing) | `ack <incident-id> [<id2>...]` |
| un-acknowledge | `unack <incident-id> [<id2>...]` |
| close | `close <id> [<id2>...]` |
| reopen | `reopen <incident-id> [<id2>...]` |
| resolve with optional note | `resolve <incident-id> [<id2>...]` |
| snooze / un-snooze | `snooze <id> [<id2>...]` / `wake <incident-id> [<id2>...]` |
| add comment | `comment <id> [<id2>...]` |
| add responder by member ID | `add-responder <id>` |
| add responders (alternate; positional person IDs) | `responder-add <person-id> [<id2>...] --incident-id <id>` |
| dispatch to an escalation level / responder | `assign --data '{"incident_id":"<id>","assigned_to":{...}}'` (body-only `assigned_to`) |
| replace responder list | `reassign <id>` |
| merge duplicates (IRREVERSIBLE) | `merge <target_id>` |
| stop auto-merging alerts in | `disable-merge <incident-id> [<id2>...]` |
| permanently delete (IRREVERSIBLE) | `remove <id> [<id2>...]` |
| post-mortem reports | `post-mortem-list` / `post-mortem-info <post-mortem-id>` / `post-mortem-delete <post-mortem-id>` |
| war room (IM chat) | `war-room-list <incident-id>` → `war-room-create <incident-id>` |
| war room (IM chat), nested subcommand form | `war-room list/create/get/add-member/default-observers/delete <id>` |

## Hot flow — triage an active incident

```bash
# 1. Find unacknowledged critical incidents (last 4h)
fduty incident list --severity Critical --progress Triggered --since 4h --fields incident_id,title,incident_severity,progress,start_time,channel_id --output-format toon

# 2. Get AI summary + full detail (use the 24-char incident_id from step 1)
fduty incident detail <incident-id> --fields incident_id,title,incident_severity,progress,ai_summary,root_cause,resolution,alert_cnt,start_time,channel_id --output-format toon

# 3. See contributing alerts
fduty incident alerts <incident-id>

# 4. Check for prior similar incidents (channel-backed only; see Gotchas)
fduty incident similar <incident-id> --limit 5 --output-format toon

# 5. Acknowledge ownership
fduty incident ack <incident-id>

# 6. Post a status comment — content goes into a file, never a shell argument
ID=<incident-id>
COMMENT_FILE=$(mktemp)
# Before running, choose a fresh delimiter that is absent as a full line in the intended comment.
cat > "$COMMENT_FILE" <<'FDUTY_COMMENT_7F3A9C2E_EOF'
Root cause identified: DB failover.
Fix deploying.
FDUTY_COMMENT_7F3A9C2E_EOF
fduty incident comment "$ID" --comment-file "$COMMENT_FILE"

# 7. Resolve with root-cause note
fduty incident resolve <incident-id> --root-cause "DB primary failover delay" --resolution "Failover completed; latency normal."
```

Projected `similar` lists stay below 16 KiB, and projected `detail --fields` output stays below 8 KiB. A trailing `...` means a long retained string was shortened; omit `--fields` only when the full unbounded detail is explicitly required.

`comment` never accepts the text as a command-line argument — only `--comment-file <path>` (or `--comment-file -` to read stdin), so backticks/`$()`/quotes inside the comment are inert. The command also reads back every target's timeline after writing and exits non-zero unless it finds an entry matching what it sent, so `Commented on ...` is proof of content fidelity, not just acceptance — no separate manual read-back is needed. Leading and trailing whitespace is stripped before sending (the server strips it too, so this is what gets stored); everything else, including interior blank lines, is preserved exactly.

> `incident list --output-format json|toon` defaults to the compact row projection `incident_id,title,incident_severity,progress,start_time,channel_id`. Pass `--fields incident_id,title,channel_id,start_time` when you need different list columns; use `incident detail <id>` / `incident get <id>` for full incident records.

## Hot flow — full fault analysis (read-only summary)

When asked to **summarize / analyze** an incident — 详情 + 关联告警 + 变更 + 时间线 + 相似故障 + 复盘 — `incident detail` does **not** contain the alerts / timeline / similar / post-mortem / change data; each is its own command. **Your first action must be the bundled script** — do not hand-pick one or two commands and write the rest from memory. One call fetches all six aspects:

```bash
bash <skill-dir>/scripts/incident-summary.sh <incident-id>
```

`<skill-dir>` is this skill's base directory — you were given it when the skill loaded (it is also the folder you read this card from). The script runs every command below and prints the results in one block, so each section of your summary is backed by real output and there is nothing to guess. (To tie post-mortems to *this* incident, re-run `incident post-mortem-list --channel-ids <channel-id>` with the `channel_id` from `detail`.)

If you fetch the pieces by hand instead, run **all six** — they are cheap reads:

```bash
ID=<incident-id>                                          # 24-char id from `incident list`
fduty incident detail   "$ID" --fields incident_id,title,incident_severity,progress,ai_summary,root_cause,resolution,alert_cnt,start_time,channel_id --output-format toon  # ① 详情 + AI summary + alert counts + channel
fduty incident alerts   "$ID"                             # ② contributing alerts (detail's embedded alerts are empty here)
fduty incident timeline "$ID"                             # ④ timeline  (or `incident feed "$ID"` for the paginated view)
fduty incident similar  "$ID" --limit 5 --output-format toon          # ⑤ similar past incidents (channel-backed; see Gotchas; compact by default)
fduty incident post-mortem-list --channel-ids <channel-id> # ⑥ post-mortems for this incident's channel
fduty change list --since 24h                              # ③ correlated changes — by shared labels + time; see reference/change.md
```

> **Never report a result you didn't fetch.** Do not write "返回空" / "无" / a count for any aspect whose command is **absent from your tool-call history this turn** — write `未查询 — 可运行 <command>` instead. "Empty" is a claim only a command you actually ran can make; inventing it is the worst failure mode of a fault summary.

## Hot flow — resolve, document, and merge duplicates

```bash
# Merge two duplicate incidents into a primary (IRREVERSIBLE — confirm first)
fduty incident merge <primary-incident-id> --source <dup1-id>,<dup2-id>

# Record post-incident narrative on the primary
fduty incident reset <primary-incident-id> \
  --root-cause "Redis OOM on shard-3" \
  --impact "Checkout latency P99 >5s for 12 min" \
  --resolution "Increased memory limit; deployed hot patch"

# Review the event timeline
fduty incident timeline <primary-incident-id>
```

<!-- GENERATED:incident START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### ack <incident-id> [<id2>...]
Acknowledge incident
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to acknowledge. At most 100 per call.
- `--summary` string — Form summary recorded as a timeline comment. Accepted only when the acknowledgement form contains a summary element.
- body-only (`--data`): custom_fields (object); images (array<object>)

### add-responder <id>
Add responders to an incident
- `--follow-preference` bool
- `--notify-channel` string
- `--person` string
- `--template-id` string

### alert-list <incident-id>
List alerts of incident
- `<incident-id>` (positional, required) string — Incident ID (MongoDB ObjectID).
- `--include-events` bool — When true, include at most the 20 newest raw events in each alert item as a preview.
- `--is-active` bool — When true return only active alerts (Critical/Warning/Info); when false return only recovered alerts (Ok). Omit to include all.
- `--limit` int64 — Page size, at most 1000. (0-1000)
- `--page` int64 — Page number starting at 1. (min 0)
- `--search-after-ctx` string
- response: `{items: [...], total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); alert_id (string); alert_key (string); alert_severity (string); alert_status (string); channel_id (integer); channel_name (string); channel_status (string); created_at (integer); data_source_id (integer); data_source_name (string); data_source_ref_id (string); data_source_type (string); deleted_at (integer); description (string); end_time (integer); event_cnt (integer); events (array<object>); ever_muted (boolean); images (array<object>); incident (object); integration_id (integer); integration_name (string); integration_ref_id (string); integration_type (string); labels (object); last_time (integer); responder_email (string); responder_name (string); start_time (integer); title (string); title_rule (string); updated_at (integer)

### alerts <id>
View incident alerts
- `--limit` int
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); alert_id (string); alert_key (string); alert_severity (string); alert_status (string); channel_id (integer); channel_name (string); channel_status (string); created_at (integer); data_source_id (integer); data_source_name (string); data_source_ref_id (string); data_source_type (string); deleted_at (integer); description (string); end_time (integer); event_cnt (integer); events (array<object>); ever_muted (boolean); images (array<object>); incident (object); integration_id (integer); integration_name (string); integration_ref_id (string); integration_type (string); labels (object); last_time (integer); responder_email (string); responder_name (string); start_time (integer); title (string); title_rule (string); updated_at (integer)

### assign
Assign incident
- `--incident-id` string — Single incident ID. Ignored when 'incident_ids' is also provided.
- `--incident-ids` stringSlice — Batch incident IDs.
- body-only (`--data`): assigned_to (object) (required)

### close <id> [<id2> ...]
Close incidents

### comment <id> [<id2> ...]
Add a comment to incident timelines
- `--comment-file` string
- `--mute-reply` bool

### create
Create a new incident
- `--assign` intSlice
- `--channel` int64
- `--description` string
- `--severity` string
- `--title` string

### custom-action-do
Execute custom action
- `--incident-id` string (required) — Incident ID (MongoDB ObjectID).
- `--integration-id` int64 (required) — Custom action integration ID. Must be enabled and associated with the incident's channel.
- response: single object (`data` unwrapped to the top level) — fields: message (string)

### detail <id>
View full incident detail with AI summary
- `--fields` string
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); account_locale (string); account_name (string); account_time_zone (string); ack_time (integer); active_alert_cnt (integer); ai_summary (string); alert_cnt (integer); alert_event_cnt (integer); alerts (array<object>); assigned_to (object); channel_id (integer); channel_name (string); channel_status (string); close_time (integer); closer (object); closer_id (integer); created_at (integer); creator (object); creator_id (integer); data_source_id (integer); data_source_ids (array<integer>); data_source_type (string); data_source_types (array<string>); dedup_key (string); deleted_at (integer); description (string); detail_url (string); end_time (integer); equals_md5 (string); ever_muted (boolean); fields (object); frequency (string); group_method (string); images (array<object>); impact (string); incident_id (string); incident_severity (string); incident_status (string); integration_id (integer); integration_ids (array<integer>); integration_type (string); integration_types (array<string>); labels (object); last_time (integer); links (array<object>); manual_overrides (array<string>); num (string); owner (object); owner_id (integer); post_mortem_id (string); progress (string); reporter_email (string); resolution (string); responders (array<object>); root_cause (string); silence_url (string); snoozed_before (integer); start_time (integer); title (string); updated_at (integer)

### disable-merge <incident-id> [<id2>...]
Disable incident merge
- `<incident-ids>` (positional, required) stringSlice — Incident IDs whose automatic merge should be disabled.

### feed <id>
View incident feed (paginated timeline)
- `--limit` int
- `--page` int
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); created_at (integer); creator_id (integer); deleted_at (integer); detail (object); ref_id (string); type (string); updated_at (integer)

### field-reset <incident-id>
Update incident custom field
- `--field-name` string (required) — Custom field name; must match a field defined on the account.
- `<incident-id>` (positional, required) string — Incident ID (MongoDB ObjectID).
- body-only (`--data`): field_value (any)

### get <id> [<id2> ...]
Get incident details
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); account_locale (string); account_name (string); account_time_zone (string); ack_time (integer); active_alert_cnt (integer); ai_summary (string); alert_cnt (integer); alert_event_cnt (integer); alerts (array<object>); assigned_to (object); channel_id (integer); channel_name (string); channel_status (string); close_time (integer); closer (object); closer_id (integer); created_at (integer); creator (object); creator_id (integer); data_source_id (integer); data_source_ids (array<integer>); data_source_type (string); data_source_types (array<string>); dedup_key (string); deleted_at (integer); description (string); detail_url (string); end_time (integer); equals_md5 (string); ever_muted (boolean); fields (object); frequency (string); group_method (string); images (array<object>); impact (string); incident_id (string); incident_severity (string); incident_status (string); integration_id (integer); integration_ids (array<integer>); integration_type (string); integration_types (array<string>); labels (object); last_time (integer); links (array<object>); manual_overrides (array<string>); num (string); owner (object); owner_id (integer); post_mortem_id (string); progress (string); reporter_email (string); resolution (string); responders (array<object>); root_cause (string); silence_url (string); snoozed_before (integer); start_time (integer); title (string); updated_at (integer)

### info [<incident-id>]
Get incident detail
- `--incident-id` string — Incident ID (MongoDB ObjectID).
- `--num` string — Short incident ID (the 6-character uppercased id shown in the UI). Not unique — resolves to the most recent match. Supply either incident_id or num.
- response: same shape as `detail <id>` above

### list
List incidents
- `--channel-id` int64
- `--fields` string
- `--limit` int
- `--nums` string
- `--page` int
- `--progress` string
- `--query` string
- `--severity` string
- `--since` string
- `--until` string
- response: same shape as `get <id> [<id2> ...]` above

### list-by-ids <incident-id> [<id2>...]
List incidents by IDs
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to fetch.
- response: `{items: [...], has_next_page, search_after_ctx, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); account_locale (string); account_name (string); account_time_zone (string); ack_time (integer); active_alert_cnt (integer); ai_summary (string); alert_cnt (integer); alert_event_cnt (integer); alerts (array<object>); assigned_to (object); channel_id (integer); channel_name (string); channel_status (string); close_time (integer); closer (object); closer_id (integer); created_at (integer); creator (object); creator_id (integer); data_source_id (integer); data_source_ids (array<integer>); data_source_type (string); data_source_types (array<string>); dedup_key (string); deleted_at (integer); description (string); detail_url (string); end_time (integer); equals_md5 (string); ever_muted (boolean); fields (object); frequency (string); group_method (string); images (array<object>); impact (string); incident_id (string); incident_severity (string); incident_status (string); integration_id (integer); integration_ids (array<integer>); integration_type (string); integration_types (array<string>); labels (object); last_time (integer); links (array<object>); manual_overrides (array<string>); num (string); owner (object); owner_id (integer); post_mortem_id (string); progress (string); reporter_email (string); resolution (string); responders (array<object>); root_cause (string); silence_url (string); snoozed_before (integer); start_time (integer); title (string); updated_at (integer)

### merge <target_id>
Merge incidents into a target incident
- `--source` string

### past-list <incident-id>
List past incidents
- `<incident-id>` (positional, required) string — Reference incident ID (MongoDB ObjectID).
- `--limit` int64 — Maximum number of similar incidents to return. (0-100)
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); account_locale (string); account_name (string); account_time_zone (string); ack_time (integer); active_alert_cnt (integer); ai_summary (string); alert_cnt (integer); alert_event_cnt (integer); alerts (array<object>); assigned_to (object); channel_id (integer); channel_name (string); channel_status (string); close_time (integer); closer (object); closer_id (integer); created_at (integer); creator (object); creator_id (integer); data_source_id (integer); data_source_ids (array<integer>); data_source_type (string); data_source_types (array<string>); dedup_key (string); deleted_at (integer); description (string); detail_url (string); end_time (integer); equals_md5 (string); ever_muted (boolean); fields (object); frequency (string); group_method (string); images (array<object>); impact (string); incident_id (string); incident_severity (string); incident_status (string); integration_id (integer); integration_ids (array<integer>); integration_type (string); integration_types (array<string>); labels (object); last_time (integer); links (array<object>); manual_overrides (array<string>); num (string); owner (object); owner_id (integer); post_mortem_id (string); progress (string); reporter_email (string); resolution (string); responders (array<object>); root_cause (string); score (number); silence_url (string); snoozed_before (integer); start_time (integer); title (string); updated_at (integer)

### post-mortem-basics-reset <post-mortem-id>
Update post-mortem basics
- `--incidents-earliest-start-seconds` string (required) — Unix timestamp in seconds for the earliest linked incident start time. (min 1) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--incidents-highest-severity` string (required) — Highest severity among linked incidents.
- `--incidents-latest-close-seconds` string — Unix timestamp in seconds for the latest linked incident close time. 0 when still open. (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--incidents-total-duration-seconds` int64 — Total incident duration in seconds. (min 0)
- `<post-mortem-id>` (positional, required) string — Post-mortem ID.
- `--responder-ids` intSlice — Responder member IDs to store on the report.

### post-mortem-content-reset <post-mortem-id>
Reset post-mortem Markdown content
- `--expected-revision` int64
- `--idempotency-key` string
- `--markdown-file` string
- response: single object (`data` unwrapped to the top level) — fields: generation (integer); markdown_bytes (integer); markdown_sha256 (string); post_mortem_id (string); previous_generation (integer); previous_revision (integer); revision (integer)

### post-mortem-delete <post-mortem-id>
Delete post-mortem
- `<post-mortem-id>` (positional, required) string — Post-mortem ID.

### post-mortem-follow-ups-reset <post-mortem-id>
Update post-mortem follow-ups
- `--follow-ups` string — Follow-up action items as free text.
- `<post-mortem-id>` (positional, required) string — Post-mortem ID.

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
- `--created-at-end-seconds` string — Filter by creation time: upper bound in seconds. (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--created-at-start-seconds` string — Filter by creation time: lower bound in seconds. (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--limit` int64 — Page size, at most 100. (0-100)
- `--order-by` string — Field used to order results. · enum: created_at_seconds | updated_at_seconds
- `--page` int64 — Page number starting at 1. (min 0)
- `--search-after-ctx` string — Cursor from a previous response for forward pagination.
- `--status` string — Report status. Defaults to 'published' on the server when omitted. · enum: drafting | published
- `--team-ids` intSlice — Team IDs to restrict the query to.
- response: `{items: [...], has_next_page, search_after_ctx, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); author_ids (array<integer>); channel_id (integer); channel_name (string); created_at_seconds (integer); generation (integer); incident_ids (array<string>); is_private (boolean); media_count (integer); post_mortem_id (string); revision (integer); status (string); team_id (integer); template_id (string); title (string); updated_at_seconds (integer)

### post-mortem-status-reset <post-mortem-id>
Update post-mortem status
- `<post-mortem-id>` (positional, required) string — Post-mortem ID.
- `--status` string (required) — Target report status. · enum: drafting | published

### post-mortem-template-delete <template-id>
Delete post-mortem template
- `<template-id>` (positional, required) string — Template ID.

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
- `<post-mortem-id>` (positional, required) string — Post-mortem ID.
- `--title` string (required) — New report title.

### reassign <id>
Reassign an incident to new responders
- `--person` string

### remove <id> [<id2> ...]
Permanently remove incidents
- `--force` bool

### reopen <incident-id> [<id2>...]
Reopen incident
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to reopen. At most 100 per call.
- `--reason` string — Optional reason recorded on the timeline. (≤1024 chars)

### reset <incident-id>
Update incident fields
- `--description` string — New description. (3-6144 chars)
- `--impact` string — New impact description. (3-6144 chars)
- `<incident-id>` (positional, required) string — Incident ID (MongoDB ObjectID).
- `--incident-severity` string — New severity. · enum: Info | Warning | Critical
- `--resolution` string — New resolution notes. (3-6144 chars)
- `--root-cause` string — New root cause analysis. (3-6144 chars)
- `--title` string — New incident title. (3-200 chars)

### resolve <incident-id> [<id2>...]
Resolve incident
- `--description` string — New incident description, up to 6,144 characters. When set, it replaces the current description before the incident closes. (≤6144 chars)
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to resolve. At most 100 per call.
- `--resolution` string — Optional resolution note applied to every resolved incident. (≤1024 chars)
- `--root-cause` string — Optional root cause note applied to every resolved incident. (≤1024 chars)
- `--summary` string — Form summary recorded as a timeline comment. Accepted only when the resolution form contains a summary element.
- body-only (`--data`): custom_fields (object); images (array<object>)

### responder-add <person-id> [<id2>...]
Add incident responder
- `--incident-id` string (required) — Incident ID (MongoDB ObjectID).
- `<person-ids>` (positional, required) intSlice — Member IDs to add as responders.
- body-only (`--data`): notify (object)

### sdp-request-list
Get ServiceDeskPlus linked incidents
- `--asc` bool — When 'true', sort by internal record ID ascending; otherwise descending.
- `--channel-ids` intSlice — Channel IDs to filter by.
- `--end-time` string — Window end, Unix seconds. Must be greater than or equal to 'start_time'. Optional when 'incident_id' is provided. (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--incident-id` string — Flashduty incident ID. When set, the time window can be omitted. (≤64 chars)
- `--integration-id` int64 — ServiceDeskPlus integration ID. (min 0)
- `--limit` int64 — Page size. Defaults to 20; maximum 100. (0-100)
- `--page` int64 — Page number starting at 1. Ignored when 'search_after_ctx' is set. (min 0)
- `--request-id` string — ServiceDeskPlus request ID. (≤64 chars)
- `--search-after-ctx` string — Cursor returned by the previous page.
- `--start-time` string — Window start, Unix seconds. Optional when 'incident_id' is provided. (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--status` string — Synchronization status filter. · enum: success | failed
- response: `{items: [...], has_next_page, search_after_ctx, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: channel_id (integer); channel_name (string); created_at (integer); error_message (string); incident_id (string); incident_title (string); integration_id (integer); request_id (string); request_link (string); status (string)

### similar <id>
Find similar incidents
- `--fields` string
- `--limit` int
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); account_locale (string); account_name (string); account_time_zone (string); ack_time (integer); active_alert_cnt (integer); ai_summary (string); alert_cnt (integer); alert_event_cnt (integer); alerts (array<object>); assigned_to (object); channel_id (integer); channel_name (string); channel_status (string); close_time (integer); closer (object); closer_id (integer); created_at (integer); creator (object); creator_id (integer); data_source_id (integer); data_source_ids (array<integer>); data_source_type (string); data_source_types (array<string>); dedup_key (string); deleted_at (integer); description (string); detail_url (string); end_time (integer); equals_md5 (string); ever_muted (boolean); fields (object); frequency (string); group_method (string); images (array<object>); impact (string); incident_id (string); incident_severity (string); incident_status (string); integration_id (integer); integration_ids (array<integer>); integration_type (string); integration_types (array<string>); labels (object); last_time (integer); links (array<object>); manual_overrides (array<string>); num (string); owner (object); owner_id (integer); post_mortem_id (string); progress (string); reporter_email (string); resolution (string); responders (array<object>); root_cause (string); score (number); silence_url (string); snoozed_before (integer); start_time (integer); title (string); updated_at (integer)

### snooze <id> [<id2> ...]
Snooze incidents
- `--duration` string

### timeline <id>
View incident timeline
- response: same shape as `feed <id>` above

### unack <incident-id> [<id2>...]
Unacknowledge incident
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to unacknowledge. At most 100 per call.

### update <id>
Update an incident
- `--description` string
- `--field` stringArray
- `--severity` string
- `--title` string

### wake <incident-id> [<id2>...]
Wake incident
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to wake. At most 100 per call.

### add-member <chat_id>
Add members to an incident war room
- `--integration` int64
- `--member` string

### create <incident_id>
Create an incident war room
- `--add-observers` bool
- `--integration` int64
- `--member` string

### default-observers <incident_id>
Preview historical responders for war-room observer invitation
- response: single object (`data` unwrapped to the top level) — fields: observers (array<object>)

### delete <incident_id>
Delete an incident war room
- `--force` bool
- `--integration` int64

### get <chat_id>
Get incident war room details
- `--integration` int64
- response: single object (`data` unwrapped to the top level) — fields: chat_id (string); chat_name (string); share_link (string)

### list <incident_id>
List incident war rooms
- `--integration` int64
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); chat_id (string); created_at (integer); created_by (integer); incident_id (string); integration_id (integer); plugin_type (string); status (string)

### war-room-add-member <chat-id>
Add war-room member
- `<chat-id>` (positional, required) string — Chat ID of the war room within the IM platform.
- `--integration-id` int64 (required) — IM integration that hosts the war room.
- `--member-ids` intSlice (required) — Person IDs to add to the war room.

### war-room-create
Create war room
- `--add-observers` bool — When true, also add historical responders of the incident as observers.
- `--incident-id` string (required) — Incident ID (MongoDB ObjectID).
- `--integration-id` int64 (required) — IM integration ID. Must have war room enabled; Feishu, DingTalk, WeCom (self-built), Slack and Teams are supported.
- `--member-ids` intSlice — Additional member IDs to add to the war room.
- response: same shape as `get <chat_id>` above

### war-room-default-observers <incident-id>
Get war-room default observers
- `<incident-id>` (positional, required) string — Incident ID, a MongoDB ObjectID hex string.
- response: same shape as `default-observers <incident_id>` above

### war-room-delete
Delete war room
- `--incident-id` string (required) — Incident ID (MongoDB ObjectID).
- `--integration-id` int64 (required) — IM integration ID.

### war-room-detail <chat-id>
Get war room detail
- `<chat-id>` (positional, required) string — Chat/group ID on the IM side.
- `--integration-id` int64 (required) — IM integration ID that hosts the war room.
- response: same shape as `get <chat_id>` above

### war-room-list <incident-id>
List war rooms
- `<incident-id>` (positional, required) string — Incident ID (MongoDB ObjectID).
- `--integration-id` int64 — Optional filter: only return war rooms for this IM integration.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); chat_id (string); created_at (integer); created_by (integer); incident_id (string); integration_id (integer); plugin_type (string); status (string)

<!-- GENERATED:incident END -->

## Status / severity values

- **progress** (`--progress` filter): `Triggered` → `Processing` → `Closed`
- **severity** (`--severity` filter / `--severity` on create/update/reset): `Critical` · `Warning` · `Info`
- `ack` moves Triggered → Processing. `close`/`resolve` move any state → Closed. `reopen` moves Closed → Triggered.

## Gotchas

- **24-char `incident_id` vs 6-char `num`**: most positional-id verbs (`ack`, `close`, `resolve`, `alerts`, `timeline`, `merge`, `reassign`, `comment`, `reset`, …) require the full ObjectID. Passing a 6-char num to any of them 400s. Use `incident info --num <num>` to resolve, or `incident list --query <num>` and read `incident_id`. **Exception: `detail` and `get` accept either form** — a 6-char num auto-resolves against the last 30 days via `/incident/list`; a miss errors `no incident with short id ... in the last 30 days`, and multiple matches list full-id candidates to disambiguate.
- **`similar` only works on channel-backed incidents** (those with a real `channel_id`). Manually created incidents with no channel return HTTP 400 "Channel not found" — this is expected, not transient. Fall back to `incident list --query "<keywords>"` for text search.
- **`update` vs `reset`**: `update <id>` edits title/description/severity/custom fields. `reset <incident-id>` additionally supports `--impact`, `--root-cause`, `--resolution` (the AI narrative fields). Use `reset` for post-incident write-back.
- **If `list` returns a `total`, use it instead of page-walking.** For "how many incidents are Triggered / Processing / Closed", run one filtered `incident list --progress <bucket> ...` per bucket and read the returned `total`. Do not fetch page 1/2/3 just to derive counts the server already computed.
- **Use `--fields` to keep list scans compact.** When the goal is to identify matching incidents or collect IDs/numbers/titles, project only the needed columns first, then fetch one target incident with `detail` / `alerts` / `timeline`.
- **`--list` window cap**: `--since`/`--until` window must be < 31 days; `--limit` max 100. Empty result is authoritative — do not widen filters or retry.
- **`merge` is irreversible**: source incidents are absorbed into target permanently. Always list and confirm both IDs before running.
- **`remove --force`** bypasses the interactive confirmation prompt — never pass `--force` unless the user has explicitly said so.
- **`assign` needs `--data` for the nested `assigned_to` object** (either `person_ids` or `escalate_rule_id`). Pass member IDs from `member list` in the API field: `--data '{"incident_ids":["<id>"],"assigned_to":{"person_ids":[101]}}'`. `reassign <id> --person <ids>` is simpler for direct member assignment.

## Worked example

```bash
# Start: a prod alert paged out; you have the 6-char num "A3F9B1" from Slack.
# Step 1: resolve the num to full id and get AI summary in one call.
fduty incident info --num A3F9B1 --output-format toon

# Step 2: acknowledge so teammates see it's being handled.
fduty incident ack <incident-id>

# Step 3: after fix, resolve with context.
fduty incident resolve <incident-id> \
  --root-cause "Misconfigured health-check threshold after deploy" \
  --resolution "Reverted threshold; all pods healthy."
```
