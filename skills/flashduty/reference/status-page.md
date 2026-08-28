# fduty status-page — command card

Prereq: `SKILL.md` read. **SKILL.md + this card = full competence on status pages — no `--help` needed.** Read verbs are free; any `change-*` create/update with `--notify-subscribers` pages subscribers immediately — confirm scope first.

Structure mutations (`component-upsert` / `component-delete` / `section-upsert` / `section-delete` / `update` / `delete`) on a **public** page change what visitors see immediately, though none of them notify subscribers (only `change-*` with `--notify-subscribers` does). If the task was advisory (「推荐 / 建议 / 看一下」), the user endorsing your proposed structure approves the design, not the write — confirm once before the first mutation. After mutating, report the public page URL, what you verified, and the exact undo (`component-delete` / `section-delete` with the returned IDs).

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
# page-id is POSITIONAL here (see fence headings: `### change-active-list <page-id>`); change-id stays a flag.
# 1. find the page + impacted component IDs
fduty status-page list --output-format toon
# 2. confirm nothing already open (empty = nothing open; if one exists, reuse its change_id)
fduty status-page change-active-list <page_id> --type incident
# 3. open it (page-id positional; scalars as flags; the required `updates` array via --data); save change_id
fduty status-page change-create <page_id> --type incident \
  --title "API latency elevated" --status investigating --description "Investigating elevated latency." \
  --data '{"updates":[{"status":"investigating","description":"Team is investigating.","component_changes":[{"component_id":"<component_id>","status":"degraded"}]}]}'
# 4. post progress: investigating → identified → monitoring (change-timeline-create takes BOTH ids as flags)
fduty status-page change-timeline-create --page-id <page_id> --change-id <change_id> \
  --status identified --description "Root cause identified."
# 5. resolve — every referenced component MUST go back to operational
fduty status-page change-timeline-create --page-id <page_id> --change-id <change_id> \
  --status resolved --description "Recovered." \
  --data '{"component_changes":[{"component_id":"<component_id>","status":"operational"}]}'
# 6. confirm closed
fduty status-page change-active-list <page_id> --type incident
```

<!-- GENERATED:status-page START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### change-active-list <page-id>
List active status page events
- `<page-id>` (positional, required) int64 — Status page ID.
- `--type` string (required) — Event type filter. Required. Returns only in-progress (non-terminal) events — 'investigating'/'identified'/'monitoring' for 'incident', 'scheduled'/'ongoing' for 'maintenance'. · enum: incident | maintenance
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: affected_components (array<object>); auto_update_by_schedule (boolean); change_id (integer); close_at_seconds (string); description (string); is_retrospective (boolean); linked_change_ids (array<string>); notify_subscribers (boolean); page_id (integer); responder_ids (array<integer>); start_at_seconds (string); status (string); title (string); type (string); updates (array<object>)

### change-create <page-id>
Create status page event
- `--auto-update-by-schedule` bool — Maintenance only: automatically advance the status based on the scheduled window.
- `--close-at-seconds` string — Event close time in Unix seconds. Must be greater than or equal to the first update's 'at_seconds'. For retrospective events this is the time the event ended; for maintenances with 'auto_update_by_schedule' it schedules the automatic transition to 'completed' and must be within 30 days from now. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--description` string (required) — Event description (Markdown). Must not be empty.
- `--is-retrospective` bool — Mark this event as a retrospective (historical) one.
- `--linked-changes` stringSlice — Linked change IDs (related incidents, deployments, etc.).
- `--notify-subscribers` bool — Notify subscribers about this event and all its updates.
- `<page-id>` (positional, required) int64 — Status page ID; obtain it from 'GET /status-page/list'.
- `--responders` intSlice — Member IDs responsible for the event.
- `--start-at-seconds` string — Event start time in Unix seconds. The stored start time is always derived from the first update's 'at_seconds' (which defaults to the current time when omitted); for maintenances with 'auto_update_by_schedule', this value schedules the automatic transition to 'ongoing'. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--status` string (required) — Initial event status. 'investigating'/'identified'/'monitoring'/'resolved' apply to incidents; 'scheduled'/'ongoing'/'completed' apply to maintenances. · enum: investigating | identified | monitoring | resolved | scheduled | ongoing | completed
- `--title` string (required) — Event title, up to 255 characters. (≤255 chars)
- `--type` string (required) — Change type: 'incident' unplanned incident, 'maintenance' planned maintenance. · enum: incident | maintenance
- body-only (`--data`): updates (array<object>) (required)
- response: single object (`data` unwrapped to the top level) — fields: change_id (integer); change_name (string)

### change-delete
Delete status page event
- `--change-id` int64 (required) — Target change ID; obtain it from 'GET /status-page/change/list'.
- `--page-id` int64 (required) — Status page ID; obtain it from 'GET /status-page/list'.

### change-info
Get status page event detail
- `--change-id` int64 (required) — Event (change) ID.
- `--page-id` int64 (required) — Status page ID.
- response: single object (`data` unwrapped to the top level) — fields: affected_components (array<object>); auto_update_by_schedule (boolean); change_id (integer); close_at_seconds (string); description (string); is_retrospective (boolean); linked_change_ids (array<string>); notify_subscribers (boolean); page_id (integer); responder_ids (array<integer>); start_at_seconds (string); status (string); title (string); type (string); updates (array<object>)

### change-list <page-id>
List status page events
- `--end-at-seconds` string — Upper bound of the event activity window: only events started at or before this Unix timestamp (seconds) are returned. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `<page-id>` (positional, required) int64 — Status page ID.
- `--start-at-seconds` string — Lower bound of the event activity window: only events still open at, or closed at or after, this Unix timestamp (seconds) are returned. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--status` string (required) — Event status filter. Required. Must be a status valid for the given 'type' ('investigating'/'identified'/'monitoring'/'resolved' for 'incident'; 'scheduled'/'ongoing'/'completed' for 'maintenance'). · enum: investigating | identified | monitoring | resolved | scheduled | ongoing | completed
- `--type` string (required) — Event type filter. Required. · enum: incident | maintenance
- response: same shape as `change-active-list <page-id>` above

### change-timeline-create
Create event timeline entry
- `--at-seconds` string — Update timestamp in Unix seconds. Defaults to the current time when omitted or 0. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--change-id` int64 (required) — Target change ID; obtain it from 'GET /status-page/change/list'.
- `--description` string (required) — Update description (Markdown). Must not be empty.
- `--page-id` int64 (required) — Status page ID; obtain it from 'GET /status-page/list'.
- `--status` string (required) — Change status after this update; must be valid for the change type. When transitioning to 'resolved' or 'completed', all affected components must be back to 'operational'. | Value | Meaning | |---|---| | 'investigating' | Investigating (incident). | | 'identified' | Root cause identified (incident). | | 'monitoring' | Fix deployed, monitoring (incident). | | 'resolved' | Resolved (incident). | | 'scheduled' | Scheduled (maintenance). | | 'ongoing' | In progress (maintenance). | | 'completed' | Completed (maintenance). | · enum: investigating | identified | monitoring | resolved | scheduled | ongoing | completed
- body-only (`--data`): component_changes (array<object>)
- response: single object (`data` unwrapped to the top level) — fields: update_id (string)

### change-timeline-delete
Delete event timeline entry
- `--change-id` int64 (required) — Owning change ID; obtain it from 'GET /status-page/change/list'.
- `--page-id` int64 (required) — Status page ID; obtain it from 'GET /status-page/list'.
- `--update-id` string (required) — Timeline update ID to delete; obtain it from 'GET /status-page/change/info'.

### change-timeline-update
Update event timeline entry
- `--at-seconds` string — New update timestamp in Unix seconds. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--change-id` int64 (required) — Owning change ID; obtain it from 'GET /status-page/change/list'.
- `--description` string — New update description (Markdown).
- `--page-id` int64 (required) — Status page ID; obtain it from 'GET /status-page/list'.
- `--update-id` string (required) — Target timeline update ID; obtain it from 'GET /status-page/change/info'.

### change-update
Update status page event
- `--change-id` int64 (required) — Target change ID; obtain it from 'GET /status-page/change/list'.
- `--linked-changes` stringSlice — Linked event IDs. Pass the full replacement list.
- `--page-id` int64 (required) — Status page ID; obtain it from 'GET /status-page/list'.
- `--responders` intSlice — Member IDs responsible for this event. Pass the full replacement list.
- `--title` string — New event title, up to 255 characters. Omit to keep the existing value. (≤255 chars)

### component-delete <component-id> [<id2>...]
Delete status page component
- `<component-ids>` (positional, required) stringSlice — Component IDs to delete; obtain them from 'GET /status-page/info'.
- `--page-id` int64 (required) — Status page ID; obtain it from 'GET /status-page/list'.

### component-upsert <page-id>
Upsert status page component
- `<page-id>` (positional, required) int64 — Status page ID; obtain it from 'GET /status-page/list'.
- body-only (`--data`): components (array<object>) (required)
- response: single object (`data` unwrapped to the top level) — fields: component_ids (array<string>)

### create
Create status page
- `--contact-info` string — Get-in-touch contact, such as a mailto or website URL.
- `--custom-domain` string — Custom domain for a public status page. (≤255 chars)
- `--date-view` string (required) — How change dates are displayed: 'calendar' calendar view, 'list' list view. · enum: calendar | list
- `--display-uptime-mode` string (required) — Uptime display mode: 'chart_and_percentage' chart plus percentage, 'chart' chart only, 'none' hidden. · enum: chart_and_percentage | chart | none
- `--name` string (required) — Display name of the status page. (≤255 chars)
- `--page-footer` string — Footer content shown on the status page.
- `--page-header` string — Header content shown on the status page.
- `--page-title` string — Browser title shown for the status page.
- `--type` string (required) — Visibility type: 'public' accessible to anyone, 'internal' restricted to logged-in members of this account. · enum: public | internal
- `--url-name` string (required) — URL-safe slug, unique per account and page type. (≤255 chars)
- body-only (`--data`): custom_links (array<object>); subscription (object)
- response: single object (`data` unwrapped to the top level) — fields: page_id (integer); page_name (string); page_url_name (string)

### delete <page-id>
Delete status page
- `<page-id>` (positional, required) int64 — Status page ID; obtain it from 'GET /status-page/list'.

### info <page-id>
Get status page detail
- `<page-id>` (positional, required) int64 — Status page ID.
- response: single object (`data` unwrapped to the top level) — fields: components (array<object>); contact_info (string); custom_domain (string); custom_links (array<object>); dark_logo (string); date_view (string); display_uptime_mode (string); favicon (string); logo (string); logo_url (string); managed_domain_feature_enabled (boolean); name (string); page_footer (string); page_header (string); page_id (integer); sections (array<object>); subscription (object); template_preference (string); type (string); url_name (string)

### list
List status pages
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: components (array<object>); contact_info (string); custom_domain (string); custom_links (array<object>); dark_logo (string); date_view (string); display_uptime_mode (string); favicon (string); logo (string); logo_url (string); name (string); page_footer (string); page_header (string); page_id (integer); sections (array<object>); subscription (object); template_preference (string); type (string); url_name (string)

### migrate-email-subscribers
Migrate email subscribers
- `--api-key` string (required) — Atlassian Statuspage API key with access to the source page.
- `--source-page-id` string (required) — Atlassian Statuspage source page ID.
- `--target-page-id` int64 (required) — Flashduty target status page ID that will receive the imported subscribers.
- response: single object (`data` unwrapped to the top level) — fields: job_id (string)

### migrate-structure <source-page-id>
Migrate status page structure
- `--api-key` string (required) — Atlassian Statuspage API key with access to the source page.
- `<source-page-id>` (positional, required) string — Atlassian Statuspage source page ID.
- `--url-name` string — Target URL name for the new status page, normalized to a URL-safe slug (max 255 characters). Omit or pass null to derive it from the source page name; an explicitly empty string is rejected. (≤255 chars)
- response: same shape as `migrate-email-subscribers` above

### migration-cancel <job-id>
Cancel status page migration
- `<job-id>` (positional, required) string — Migration job ID, returned when the migration job is created; check progress via 'GET /status-page/migration/status'.

### migration-status <job-id>
Get migration status
- `<job-id>` (positional, required) string — Migration job ID returned by 'migrate-structure' or 'migrate-email-subscribers'.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); created_at (string); error (string); job_id (string); phase (string); progress (object); source_page_id (string); status (string); target_page_id (integer); updated_at (string)

### section-delete <section-id> [<id2>...]
Delete status page section
- `--page-id` int64 (required) — Status page ID; obtain it from 'GET /status-page/list'.
- `<section-ids>` (positional, required) stringSlice — Section IDs to delete; obtain them from 'GET /status-page/info'.

### section-upsert <page-id>
Upsert status page section
- `<page-id>` (positional, required) int64 — Status page ID; obtain it from 'GET /status-page/list'.
- body-only (`--data`): sections (array<object>) (required)
- response: single object (`data` unwrapped to the top level) — fields: section_ids (array<string>)

### subscriber-export <page-id>
Export subscribers
- `--component-ids` stringSlice — Optional component IDs to filter subscribers by.
- `<page-id>` (positional, required) int64 — Status page ID; obtain it from 'GET /status-page/list'.

### subscriber-import <page-id>
Import subscribers
- `--method` string (required) — Subscription method. 'email' is only valid for public pages; 'im' is only valid for internal pages. · enum: email | im
- `<page-id>` (positional, required) int64 — Target status page ID; obtain it from 'GET /status-page/list'.
- body-only (`--data`): subscribers (array<object>)

### subscriber-list <page-id>
List status page subscribers
- `--component-ids` string — Comma-separated component IDs to filter subscribers by.
- `--limit` int64 — Page size (1-100). (1-100)
- `--page` int64 — Page number (1-based). (min 1)
- `<page-id>` (positional, required) int64 — Status page ID.
- response: `{items: [...], has_next_page, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: all (boolean); components (array<object>); locale (string); method (string); recipient (string)

### template-delete
Delete status page template
- `--page-id` int64 (required) — Status page ID; obtain it from 'GET /status-page/list'.
- `--template-id` string (required) — ID of the template to delete; obtain it from 'GET /status-page/template/list'.
- `--type` string (required) — Template kind: 'pre_defined' predefined template, 'message' message template. · enum: pre_defined | message

### template-list <page-id>
List status page templates
- `<page-id>` (positional, required) int64 — Status page ID.
- `--type` string (required) — Template category. 'pre_defined' returns predefined event templates; 'message' returns message notification templates. · enum: pre_defined | message
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: description (string); messages (object); status (string); template_id (string); title (string); type (string)

### template-upsert <page-id>
Upsert status page template
- `<page-id>` (positional, required) int64 — Status page ID; obtain it from 'GET /status-page/list'.
- `--type` string (required) — Template category. 'pre_defined' for predefined event templates; 'message' for notification message templates. · enum: pre_defined | message
- body-only (`--data`): template (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: template_id (string)

### update <page-id>
Update status page
- `--contact-info` string — Get-in-touch contact, such as a mailto or website URL. Omit or pass null to keep the existing value.
- `--custom-domain` string — Custom domain for a public status page. Omit or pass null to keep the existing value. (≤255 chars)
- `--dark-logo` string — Dark-mode logo image of the status page. Omit or pass null to keep the existing value.
- `--date-view` string — How change dates are displayed. Leave empty to keep the current value. 'calendar' uses a calendar view; 'list' uses a list view. · enum: calendar | list
- `--display-uptime-mode` string — How uptime is displayed. Leave empty to keep the current value. 'chart_and_percentage' shows both chart and percentage; 'chart' shows only the chart; 'none' hides uptime. · enum: chart_and_percentage | chart | none
- `--favicon` string — Favicon of the status page. Omit or pass null to keep the existing value.
- `--logo` string — Logo image of the status page. Omit or pass null to keep the existing value.
- `--logo-url` string — URL opened when the logo is clicked. Omit or pass null to keep the existing value. (≤255 chars)
- `--name` string — Display name of the status page. Omit or pass null to keep the existing value. (≤255 chars)
- `--page-footer` string — Footer content shown on the status page. Omit or pass null to keep the existing value.
- `--page-header` string — Header content shown on the status page. Omit or pass null to keep the existing value.
- `<page-id>` (positional, required) int64 — Status page ID; obtain it from 'GET /status-page/list'.
- `--page-title` string — Browser title shown for the status page. Omit or pass null to keep the existing value.
- `--template-preference` string — Preferred event template type: 'pre_defined' or 'message'. Omit or pass null to keep the existing value.
- `--url-name` string — URL-safe slug, unique per account and page type. Omit or pass null to keep the existing value. (≤255 chars)
- body-only (`--data`): custom_links (array<object>); subscription (object)

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

- **`page_id` can be passed positionally or via `--page-id` — both work.** Verbs whose fence heading reads `### <verb> <page-id>` (change-create, change-active-list, change-list, subscriber-export/import/list) accept it either way — positional is shorter: `change-create <page_id> …` or `change-create --page-id <page_id> …`; if both are given, the flag wins. Verbs that need *both* `page-id` and `change-id` (change-info, change-delete, change-timeline-*, change-update) take both as flags only — neither has a positional form. `migrate-structure`'s positional/flag is a different field, `source-page-id` (the Atlassian source page ID), not `page-id`.
- **`page_id` (int) ≠ `change_id` (int)** — page is the status page; change is one incident/maintenance within it. Don't cross them.
- **`updates` is required on `change-create`** and goes via `--data` (it nests `component_changes[]`, which can't be flat flags). `--description` is also required by the server even though it's not flagged required. Typed scalar flags (`--title`, `--status`…) override matching `--data` keys.
- **`--notify-subscribers` emails + pushes every subscriber immediately** — set it only once scope is confirmed.
- **Migration is async and TWO separate jobs.** `migrate-structure` (structure + history, no emails) is deliberately separate from `migrate-email-subscribers` — verify the imported content before any subscriber verification emails go out. Poll `migration-status` until `completed` / `failed` / `cancelled`.
- Empty `change-active-list` is the authoritative "nothing open" — don't widen the query.

## Worked example — open an incident

```bash
fduty status-page change-create <page_id> --type incident \
  --title "Web Console Degraded" --status investigating \
  --description "Investigating degraded performance on the web console." \
  --data '{"updates":[{"status":"investigating","description":"Team is investigating.","component_changes":[{"component_id":"<component_id>","status":"degraded"}]}]}'
# → returns change_id; feed it to change-timeline-create for follow-up updates.
```
