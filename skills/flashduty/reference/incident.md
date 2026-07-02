# fduty incident — command card

Prereq: `SKILL.md` read. Read verbs are free. **Mutating verbs notify responders or alter state** — confirm scope first. `merge` and `remove` are **irreversible**; `remove` permanently deletes.

## Route here when

"告警 / 故障 / 事件 / 响应 / 值班 / incident / page / outage / triage / acknowledge / resolve / snooze / escalate / post-mortem" → **incident**, NOT `alert` (alert = deduplicated signal; incident = actionable item responders work). NOT `insight` (metrics/MTTA/MTTR). You need **`incident_id` (24-char MongoDB ObjectID)** for most verbs — not the 6-char `num` shown in the UI. If you only have a num, use `incident info --num <num>` first.

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
| add responder by person ID | `add-responder <id>` |
| replace responder list | `reassign <id>` |
| merge duplicates (IRREVERSIBLE) | `merge <target_id>` |
| stop auto-merging alerts in | `disable-merge <incident-id> [<id2>...]` |
| permanently delete (IRREVERSIBLE) | `remove <id> [<id2>...]` |
| post-mortem reports | `post-mortem-list` / `post-mortem-info <post-mortem-id>` / `post-mortem-delete <post-mortem-id>` |
| war room (IM chat) | `war-room-list <incident-id>` → `war-room-create <incident-id>` |

## Hot flow — triage an active incident

```bash
# 1. Find unacknowledged critical incidents (last 4h).
#    toon/json list output is compact by default:
#    incident_id,title,incident_severity,progress,start_time,channel_id
fduty incident list --severity Critical --progress Triggered --since 4h --output-format toon

# 2. Get AI summary + full detail (use the 24-char incident_id from step 1)
fduty incident detail <incident-id> --output-format toon

# 3. See contributing alerts
fduty incident alerts <incident-id> --output-format toon

# 4. Check for prior similar incidents (channel-backed only; see Gotchas)
fduty incident similar <incident-id> --limit 5 --output-format toon

# 5. Acknowledge ownership
fduty incident ack <incident-id>

# 6. Post a status comment
fduty incident comment <incident-id> --comment "Root cause identified: DB failover. Fix deploying."

# 7. Resolve with root-cause note
fduty incident resolve <incident-id> --root-cause "DB primary failover delay" --resolution "Failover completed; latency normal."
```

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
fduty incident detail   "$ID" --output-format toon        # ① 详情 + AI summary + alert counts + channel_id
fduty incident alerts   "$ID" --output-format toon        # ② contributing alerts (detail's embedded alerts are empty here)
fduty incident timeline "$ID" --output-format toon        # ④ timeline  (or `incident feed "$ID"` for the paginated view)
fduty incident similar  "$ID" --limit 5 --output-format toon          # ⑤ similar past incidents (channel-backed; see Gotchas)
fduty incident post-mortem-list --channel-ids <channel-id> --output-format toon   # ⑥ post-mortems for this incident's channel
fduty change list --since 24h --output-format toon        # ③ correlated changes — by shared labels + time; see reference/change.md
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
fduty incident timeline <primary-incident-id> --output-format toon
```

<!-- GENERATED:incident START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### ack <incident-id> [<id2>...]
Acknowledge incident
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to acknowledge. At most 100 per call.

### add-responder <id>
Add responders to an incident
- `--follow-preference` bool
- `--notify-channel` string
- `--person` string
- `--template-id` string

### alert-list <incident-id>
List alerts of incident
- `<incident-id>` (positional, required) string — Incident ID (MongoDB ObjectID).
- `--include-events` bool — When true, include raw alert events in each alert item.
- `--is-active` bool — When true return only active alerts (Critical/Warning/Info); when false return only recovered alerts (Ok). Omit to include all.
- `--limit` int64 — Page size, at most 1000. (0-1000)
- `--page` int64 — Page number starting at 1. (min 0)
- `--search-after-ctx` string

### alerts <id>
View incident alerts
- `--limit` int

### assign
Assign incident
- `--incident-id` string — Single incident ID. Ignored when 'incident_ids' is also provided.
- `--incident-ids` stringSlice — Batch incident IDs.
- body-only (`--data`): assigned_to (object) (required)

### close <id> [<id2> ...]
Close incidents

### comment <id> [<id2> ...]
Add a comment to incident timelines
- `--comment` string
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

### detail <id>
View full incident detail with AI summary

### disable-merge <incident-id> [<id2>...]
Disable incident merge
- `<incident-ids>` (positional, required) stringSlice — Incident IDs whose automatic merge should be disabled.

### feed <id>
View incident feed (paginated timeline)
- `--limit` int
- `--page` int

### field-reset <incident-id>
Update incident custom field
- `--field-name` string (required) — Custom field name; must match a field defined on the account.
- `<incident-id>` (positional, required) string — Incident ID (MongoDB ObjectID).
- body-only (`--data`): field_value (any)

### get <id> [<id2> ...]
Get incident details

### info [<incident-id>]
Get incident detail
- `--incident-id` string — Incident ID (MongoDB ObjectID).
- `--num` string — Short incident ID (the 6-character uppercased id shown in the UI). Not unique — resolves to the most recent match. Supply either incident_id or num.

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

### list-by-ids <incident-id> [<id2>...]
List incidents by IDs
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to fetch.

### merge <target_id>
Merge incidents into a target incident
- `--source` string

### past-list <incident-id>
List past incidents
- `<incident-id>` (positional, required) string — Reference incident ID (MongoDB ObjectID).
- `--limit` int64 — Maximum number of similar incidents to return. (0-100)

### post-mortem-basics-reset <post-mortem-id>
Update post-mortem basics
- `--incidents-earliest-start-seconds` string (required) — Unix timestamp in seconds for the earliest linked incident start time. (min 1) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--incidents-highest-severity` string (required) — Highest severity among linked incidents.
- `--incidents-latest-close-seconds` string — Unix timestamp in seconds for the latest linked incident close time. 0 when still open. (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--incidents-total-duration-seconds` int64 — Total incident duration in seconds. (min 0)
- `<post-mortem-id>` (positional, required) string — Post-mortem ID.
- `--responder-ids` intSlice — Responder member IDs to store on the report.

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

### post-mortem-init <incident-id> [<id2>...]
Initialize post-mortem
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to link to the report. 1-10 incidents.
- `--template-id` string (required) — Template ID used to initialize the report.

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

### post-mortem-template-list
List post-mortem templates
- `--asc` bool — Ascending order when true.
- `--limit` int64 — Page size, at most 100. (0-100)
- `--order-by` string — Field used to order results. · enum: created_at_seconds
- `--page` int64 — Page number starting at 1. (min 0)
- `--search-after-ctx` string — Cursor from a previous response for forward pagination.

### post-mortem-template-upsert
Create or update post-mortem template
- `--content` string (required) — BlockNote JSON template content.
- `--content-markdown` string — Markdown version of the template content.
- `--description` string — Template description.
- `--name` string (required) — Template name.
- `--team-id` int64 — Managing team ID. Required when creating a custom template.
- `--template-id` string — Template ID. Omit to create a new template; provide it to update an existing template.

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
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to resolve. At most 100 per call.
- `--resolution` string — Optional resolution note applied to every resolved incident. (≤1024 chars)
- `--root-cause` string — Optional root cause note applied to every resolved incident. (≤1024 chars)

### responder-add <person-id> [<id2>...]
Add incident responder
- `--incident-id` string (required) — Incident ID (MongoDB ObjectID).
- `<person-ids>` (positional, required) intSlice — Member IDs to add as responders.
- body-only (`--data`): notify (object)

### similar <id>
Find similar incidents
- `--limit` int

### snooze <id> [<id2> ...]
Snooze incidents
- `--duration` string

### timeline <id>
View incident timeline

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

### delete <incident_id>
Delete an incident war room
- `--force` bool
- `--integration` int64

### get <chat_id>
Get incident war room details
- `--integration` int64

### list <incident_id>
List incident war rooms
- `--integration` int64

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

### war-room-default-observers <incident-id>
Get war-room default observers
- `<incident-id>` (positional, required) string — Incident ID, a MongoDB ObjectID hex string.

### war-room-delete
Delete war room
- `--incident-id` string (required) — Incident ID (MongoDB ObjectID).
- `--integration-id` int64 (required) — IM integration ID.

### war-room-detail <chat-id>
Get war room detail
- `<chat-id>` (positional, required) string — Chat/group ID on the IM side.
- `--integration-id` int64 (required) — IM integration ID that hosts the war room.

### war-room-list <incident-id>
List war rooms
- `<incident-id>` (positional, required) string — Incident ID (MongoDB ObjectID).
- `--integration-id` int64 — Optional filter: only return war rooms for this IM integration.

<!-- GENERATED:incident END -->

## Status / severity values

- **progress** (`--progress` filter): `Triggered` → `Processing` → `Closed`
- **severity** (`--severity` filter / `--severity` on create/update/reset): `Critical` · `Warning` · `Info`
- `ack` moves Triggered → Processing. `close`/`resolve` move any state → Closed. `reopen` moves Closed → Triggered.

## Gotchas

- **24-char `incident_id` vs 6-char `num`**: positional-id verbs (`ack`, `close`, `resolve`, `detail`, `alerts`, `timeline`, `merge`, `reassign`, `comment`, `reset`, …) require the full ObjectID. Passing a 6-char num 400s. Use `incident info --num <num>` to resolve, or `incident list --query <num>` and read `incident_id`.
- **`similar` only works on channel-backed incidents** (those with a real `channel_id`). Manually created incidents with no channel return HTTP 400 "Channel not found" — this is expected, not transient. Fall back to `incident list --query "<keywords>"` for text search.
- **`update` vs `reset`**: `update <id>` edits title/description/severity/custom fields. `reset <incident-id>` additionally supports `--impact`, `--root-cause`, `--resolution` (the AI narrative fields). Use `reset` for post-incident write-back.
- **`--list` window cap**: `--since`/`--until` window must be < 31 days; `--limit` max 100. Empty result is authoritative — do not widen filters or retry.
- **`merge` is irreversible**: source incidents are absorbed into target permanently. Always list and confirm both IDs before running.
- **`remove --force`** bypasses the interactive confirmation prompt — never pass `--force` unless the user has explicitly said so.
- **`assign` needs `--data` for the nested `assigned_to` object** (either `person_ids` or `escalate_rule_id`). Pass via `--data '{"incident_ids":["<id>"],"assigned_to":{"person_ids":[101]}}'`. `reassign <id> --person <ids>` is simpler for direct-person assignment.

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
