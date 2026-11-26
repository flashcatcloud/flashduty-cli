# flashduty change — command card

Prereq: `SKILL.md` read. Two read verbs (`list`, `info`) and two write verbs (`create`, `update`). Change events are the "what changed" signal you correlate against an incident during fault analysis; `create`/`update` record and progress a change yourself.

## Route here when

"变更 / 变更事件 / 变更关联 / 发布 / 部署 / 创建变更 / 更新变更状态 / change / change event / deployment / release / what changed / correlated change / record a change" → **change**. Change events carry `labels`; they correlate to incidents by **shared labels + time proximity** (there is no foreign key). For *status-page* maintenance/incident events use `status-page` instead — that is a different "change".

## Intent → verb

| want | verb |
|---|---|
| list recent change events (filter by window / channel / integration / keyword) | `list` |
| one change with all its events, its allowed next statuses and its notification group | `info` |
| record a new change in a collaboration space | `create` |
| move a change's status, edit its fields, tick its checklist | `update` |

## Hot flow — find changes around an incident

```bash
# Pull recent changes in the incident's window, then eyeball label/time overlap with the incident.
flashduty change list --since 24h --output-format toon

# Narrow by the integration or channel that emitted them, or by a keyword:
flashduty change list --since 48h --integration <integration-id> --query "deploy" --output-format toon
```

<!-- GENERATED:change START · 由 flashduty __dump-commands 同步 · 勿手改 fence 内 -->

### create
Create change
- `--backout-plan` string — Rollback plan of the change, as Markdown.
- `--change-status` string — Initial status. Defaults to 'Planned'. | Value | Meaning | |---|---| | 'Planned' | Planned, not started. | | 'Ready' | Ready for execution. | | 'Processing' | Being executed. | | 'Canceled' | Canceled. | | 'Done' | Completed. | | 'Failed' | Failed. | · enum: Planned | Ready | Processing | Canceled | Done | Failed
- `--channel-id` int64 (required) — Collaboration space the change belongs to. Can't be changed later. (min 1)
- `--description` string — Change description.
- `--impact` string — Impact of the change, as Markdown.
- `--link` string — External link to the change record: an absolute 'http' or 'https' URL, or empty.
- `--operator` string — Person performing the change, as a member email or a free-form name.
- `--planned-end-time` int64 — Planned end, Unix timestamp in seconds or milliseconds; not earlier than 'planned_start_time'. (min 0)
- `--planned-start-time` int64 — Planned start, Unix timestamp in seconds or milliseconds. (min 0)
- `--title` string (required) — Change title, non-empty. (≥1 chars)
- body-only (`--data`): checklist (array<object>); fields (object); labels (object)
- response: single object (`data` unwrapped to the top level) — fields: change_id (string); change_key (string)

### info
Get change detail
- `--change-id` string — Change ID, a 24-character hex string.
- `--change-key` string — Change key within that integration; use together with 'integration_id'.
- `--integration-id` int64 — Integration that reported the change; use together with 'change_key'. (min 1)
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); actual_start_time (string); backout_plan (string); can_edit (boolean); change_id (string); change_key (string); change_status (string); channel_id (integer); channel_name (string); channel_status (string); checklist (array<object>); creator (any); description (string); end_time (string); events (array<object>); fields (object); impact (string); integration_id (integer); integration_name (string); is_console (boolean); labels (object); last_time (string); link (string); next_statuses (array<string>); notify_target (any); operator (string); planned_end_time (string); planned_start_time (string); start_time (string); title (string)

### list
List changes
- `--asc` bool
- `--channel-ids` int64Slice
- `--filters` string
- `--include-events` bool
- `--integration` string
- `--limit` int
- `--orderby` string
- `--page` int
- `--query` string
- `--since` string
- `--until` string
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); actual_start_time (string); backout_plan (string); can_edit (boolean); change_id (string); change_key (string); change_status (string); channel_id (integer); channel_name (string); channel_status (string); checklist (array<object>); creator (any); description (string); end_time (string); events (array<object>); fields (object); impact (string); integration_id (integer); integration_name (string); is_console (boolean); labels (object); last_time (string); link (string); next_statuses (array<string>); operator (string); planned_end_time (string); planned_start_time (string); start_time (string); title (string)

### update <change-id>
Update change
- `--backout-plan` string — Rollback plan of the change, as Markdown. Omit to keep the current value; send an empty string to clear it.
- `<change-id>` (positional, required) string — ID of the change to update, a 24-character hex string.
- `--change-status` string — New status; must be the current status or one it may move to (see 'next_statuses' in 'POST /change/info'). | Value | Meaning | |---|---| | 'Planned' | Planned, not started. | | 'Ready' | Ready for execution. | | 'Processing' | Being executed. | | 'Canceled' | Canceled. | | 'Done' | Completed. | | 'Failed' | Failed. | · enum: Planned | Ready | Processing | Canceled | Done | Failed
- `--description` string — Change description. Omit to keep the current value; send an empty string to clear it.
- `--impact` string — Impact of the change, as Markdown. Omit to keep the current value; send an empty string to clear it.
- `--link` string — External link to the change record: an absolute 'http' or 'https' URL, or empty. Omit to keep the current value; send an empty string to clear it.
- `--operator` string — Person performing the change, as a member email or a free-form name. Omit to keep the current value; send an empty string to clear it.
- `--planned-end-time` int64 — Planned end, Unix timestamp in seconds or milliseconds; not earlier than 'planned_start_time'. Omit to keep the current value; send '0' to clear it. (min 0)
- `--planned-start-time` int64 — Planned start, Unix timestamp in seconds or milliseconds. Omit to keep the current value; send '0' to clear it. (min 0)
- `--title` string — Change title, non-empty. (≥1 chars)
- body-only (`--data`): checklist (array<object>); fields (object); labels (object)
- response: single object (`data` unwrapped to the top level) — fields: change_id (string); change_key (string); change_status (string)

<!-- GENERATED:change END -->

## Key concepts

- **Correlation is heuristic, not relational.** A change is "related" to an incident when their `labels` overlap and their timestamps are close — there is no `incident_id` on a change. Judge the overlap yourself; do not claim a causal link the data doesn't support.
- **`--integration` / `--channel-ids`** scope to the source that emitted the change; **`--since` / `--until`** bound the window (a non-negative duration like `24h`, a future offset like `+24h`, `now`, a date/datetime, or Unix seconds — negative durations like `-1h` are rejected).

## Gotchas

- **`info` has no time window**: look a change up by `--change-id`, or by `--integration-id` + `--change-key`; `list` only sees changes overlapping its window.
- **`create` / `update` are writes** — confirm with the user first (see `SKILL.md`). `create` needs `--title` and `--channel-id`; the platform generates the change key. On a console-created change, `create` and status moves send a notification to the group bound in the change form.
- **Status moves are fixed**: `Planned` → `Ready`/`Processing`/`Canceled`; `Ready` → `Processing`/`Canceled`; `Processing` → `Done`/`Failed`; `Done`/`Failed` → `Processing`. Read `next_statuses` from `info` instead of guessing.
- **`update` is partial**: an omitted flag keeps the stored value; `--description ""` (and the other text flags) clears it. `labels` / `fields` / `checklist` go through `--data`: an empty label value or a `null` field value removes that key, and `checklist` replaces the whole list.
- **`list` window cap**: `--since`/`--until` window must be < 31 days; `--limit` max 100.
- **Empty result is authoritative** — no changes in that window/scope. Report it; don't widen blindly or invent a change to explain the incident.

## Worked example

```bash
# Changes in the last 6h on a specific integration, newest first
flashduty change list --since 6h --integration 5759613685214 --output-format toon
```
