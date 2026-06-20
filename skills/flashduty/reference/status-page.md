# fduty status-page — command card

Prereq: `SKILL.md` read. **SKILL.md + this card = full competence on status pages — no `--help` needed.** Read verbs are free; any `change-*` create/update with `--notify-subscribers` pages subscribers immediately — confirm scope first.

## Route here when

"公开事件 / 公开时间线 / 状态页 / 维护窗口 / 订阅者 / 状态页迁移" → **status-page**, NOT `incident` (incident = the internal alert graph; status-page = the public-facing page). You need two IDs, both from `status-page list`: **`page_id` (int)** and **`component_id` (ULID string)**.

## Intent → verb

| want | verb |
|---|---|
| pages + their component IDs | `list` |
| what's live on a page now | `change-active-list` |
| every event incl. closed | `change-list` |
| one event's detail | `change-info` |
| **open** an incident/maintenance | `change-create` (save the returned `change_id`) |
| post a progress update | `change-timeline-create` |
| edit event title/responders | `change-update` |
| delete an event | `change-delete` |
| fix/remove a timeline entry | `change-timeline-update` / `change-timeline-delete` |
| subscribers | `subscriber-list` / `subscriber-import` / `subscriber-export` |
| migrate from Atlassian Statuspage | `migrate-structure` → (verify) → `migrate-email-subscribers`; poll `migration-status`; `migration-cancel` |

## Hot flow — publish & resolve an incident

```bash
# 1. find the page + impacted component IDs
fduty status-page list --output-format toon
# 2. confirm nothing already open (empty = nothing open; if one exists, reuse its change_id)
fduty status-page change-active-list --page-id <page_id> --type incident
# 3. open it (scalars as flags, the required `updates` array via --data); save change_id
fduty status-page change-create --page-id <page_id> --type incident \
  --title "API latency elevated" --status investigating --description "Investigating elevated latency." \
  --data '{"updates":[{"status":"investigating","description":"Team is investigating.","component_changes":[{"component_id":"<component_id>","status":"degraded"}]}]}'
# 4. post progress: investigating → identified → monitoring
fduty status-page change-timeline-create --page-id <page_id> --change-id <change_id> \
  --status identified --description "Root cause identified."
# 5. resolve — every referenced component MUST go back to operational
fduty status-page change-timeline-create --page-id <page_id> --change-id <change_id> \
  --status resolved --description "Recovered." \
  --data '{"component_changes":[{"component_id":"<component_id>","status":"operational"}]}'
# 6. confirm closed
fduty status-page change-active-list --page-id <page_id> --type incident
```

<!-- GENERATED:status-page START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### change-active-list
List active status page events
- `--page-id` int64 (required) — Status page ID.
- `--type` string (required) — Event type filter. Required. Returns only in-progress (non-terminal) events — 'investigating'/'identified'/'monitoring' for 'incident', 'scheduled'/'ongoing' for 'maintenance'. · enum: incident | maintenance

### change-create
Create status page event
- `--auto-update-by-schedule` bool — Maintenance only: automatically advance the status based on the scheduled window.
- `--close-at-seconds` string — Scheduled close time for retrospective events. Must be greater than 'start_at_seconds'. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--description` string — Event description (Markdown). Required by the validator.
- `--is-retrospective` bool — Mark this event as a retrospective (historical) one.
- `--linked-changes` stringSlice — Linked change IDs (related incidents, deployments, etc.).
- `--notify-subscribers` bool — Notify subscribers about this event and all its updates.
- `--page-id` int64 (required) — Status page ID.
- `--responders` intSlice — Member IDs responsible for this event.
- `--start-at-seconds` string — Event start time in unix seconds. Defaults to now when omitted. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--status` string (required) — Initial event status. 'investigating'/'identified'/'monitoring'/'resolved' apply to incidents; 'scheduled'/'ongoing'/'completed' apply to maintenances. · enum: investigating | identified | monitoring | resolved | scheduled | ongoing | completed
- `--title` string (required) — Event title, up to 255 characters. (≤255 chars)
- `--type` string (required) — Event type. · enum: incident | maintenance
- body-only (`--data`): updates (array<object>) (required)

### change-delete
Delete status page event
- `--change-id` int64 (required) — Target event ID.
- `--page-id` int64 (required) — Status page ID.

### change-info
Get status page event detail
- `--change-id` int64 (required) — Event (change) ID.
- `--page-id` int64 (required) — Status page ID.

### change-list
List status page events
- `--end-at-seconds` string — Filter events started at or before this unix timestamp (seconds). Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--page-id` int64 (required) — Status page ID.
- `--start-at-seconds` string — Filter events started at or after this unix timestamp (seconds). Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--status` string (required) — Event status filter. Required. Must be a status valid for the given 'type' (e.g. 'investigating'/'identified'/'monitoring'/'resolved' for incidents; 'scheduled'/'ongoing'/'completed' for maintenances). · enum: investigating | identified | monitoring | resolved | scheduled | ongoing | completed
- `--type` string (required) — Event type filter. Required. · enum: incident | maintenance

### change-timeline-create
Create event timeline entry
- `--at-seconds` string — Update timestamp in unix seconds. Defaults to now when omitted. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--change-id` int64 (required) — Target event ID.
- `--description` string — Update description (Markdown). Required.
- `--page-id` int64 (required) — Status page ID.
- `--status` string (required) — New event status. Must match the event type. When the status transitions to 'resolved' or 'completed', all referenced components must become 'operational'. · enum: investigating | identified | monitoring | resolved | scheduled | ongoing | completed
- body-only (`--data`): component_changes (array<object>)

### change-timeline-delete
Delete event timeline entry
- `--change-id` int64 (required) — Parent event ID.
- `--page-id` int64 (required) — Status page ID.
- `--update-id` string (required) — Timeline update ID to delete.

### change-timeline-update
Update event timeline entry
- `--at-seconds` string — New update timestamp in unix seconds. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--change-id` int64 (required) — Parent event ID.
- `--description` string — New update description (Markdown).
- `--page-id` int64 (required) — Status page ID.
- `--update-id` string (required) — Target timeline update ID.

### change-update
Update status page event
- `--change-id` int64 (required) — Target event ID.
- `--linked-changes` stringSlice — Linked event IDs. Pass the full replacement list.
- `--page-id` int64 (required) — Status page ID.
- `--responders` intSlice — Member IDs responsible for this event. Pass the full replacement list.
- `--title` string — New event title, up to 255 characters. Omit to keep the existing value. (≤255 chars)

### list
List status pages

### migrate-email-subscribers
Migrate email subscribers
- `--api-key` string (required) — Atlassian Statuspage API key with access to the source page.
- `--source-page-id` string (required) — Atlassian Statuspage source page ID.
- `--target-page-id` int64 (required) — Flashduty target status page ID that will receive the imported subscribers.

### migrate-structure
Migrate status page structure
- `--api-key` string (required) — Atlassian Statuspage API key with access to the source page.
- `--source-page-id` string (required) — Atlassian Statuspage source page ID.
- `--url-name` string — Target URL name for the migrated status page. When omitted, the source page's URL name is reused.

### migration-cancel
Cancel status page migration
- `--job-id` string (required) — Migration job ID.

### migration-status
Get migration status
- `--job-id` string (required) — Migration job ID returned by 'migrate-structure' or 'migrate-email-subscribers'.

### subscriber-export
Export subscribers
- `--component-ids` stringSlice — Optional component IDs to filter subscribers by.
- `--page-id` int64 (required) — Status page ID.

### subscriber-import
Import subscribers
- `--method` string (required) — Subscription method. 'email' is only valid for public pages; 'im' is only valid for internal pages. · enum: email | im
- `--page-id` int64 (required) — Target status page ID.
- body-only (`--data`): subscribers (array<object>)

### subscriber-list
List status page subscribers
- `--component-ids` string — Comma-separated component IDs to filter subscribers by.
- `--limit` int64 — Page size (1-100). (1-100)
- `--page` int64 — Page number (1-based). (min 1)
- `--page-id` int64 (required) — Status page ID.

<!-- GENERATED:status-page END -->

## Status values (load-bearing — a wrong value 400s)

- **Component status** (`component_changes[].status`), by event type:
  - incident → `operational` · `degraded` · `partial_outage` · `full_outage`
  - maintenance → `operational` · `under_maintenance`
- **Event status** (`--status` on create / timeline):
  - incident → `investigating` → `identified` → `monitoring` → `resolved`
  - maintenance → `scheduled` → `ongoing` → `completed`
- Transitioning to `resolved` / `completed` ⇒ **all** referenced components must be `operational` (the server rejects the update otherwise).

## Gotchas

- **`page_id` (int) ≠ `change_id` (int)** — page is the status page; change is one incident/maintenance within it. Don't cross them.
- **`updates` is required on `change-create`** and goes via `--data` (it nests `component_changes[]`, which can't be flat flags). `--description` is also required by the server even though it's not flagged required. Typed scalar flags (`--page-id`, `--title`, `--status`…) override matching `--data` keys.
- **`--notify-subscribers` emails + pushes every subscriber immediately** — set it only once scope is confirmed.
- **Migration is async and TWO separate jobs.** `migrate-structure` (structure + history, no emails) is deliberately separate from `migrate-email-subscribers` — verify the imported content before any subscriber verification emails go out. Poll `migration-status` until `completed` / `failed` / `cancelled`.
- Empty `change-active-list` is the authoritative "nothing open" — don't widen the query.

## Worked example — open an incident

```bash
fduty status-page change-create --page-id <page_id> --type incident \
  --title "Web Console Degraded" --status investigating \
  --description "Investigating degraded performance on the web console." \
  --data '{"updates":[{"status":"investigating","description":"Team is investigating.","component_changes":[{"component_id":"<component_id>","status":"degraded"}]}]}'
# → returns change_id; feed it to change-timeline-create for follow-up updates.
```
