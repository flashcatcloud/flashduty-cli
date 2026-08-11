# fduty channel escalation rules — 分派策略

Prereq: `SKILL.md` read. `escalate-rule-list` / `escalate-rule-info` are free
reads; `escalate-rule-create/update/delete/enable/disable` mutate who gets
paged — confirm before acting.

## Route here when

"分派策略 / 分派规则 / 升级规则 / 升级策略 / escalation rule / escalation
policy / notify layers / who gets paged / on-call notification chain" → this
card. Escalation rules live INSIDE a channel (协作空间) and pick the PEOPLE
notified once an incident lands there — NOT `reference/route.md` (alert
routing picks the *channel*), NOT `reference/schedule.md` (on-call schedules
are a notify *target* referenced from layers). Key IDs: **`channel-id`
(int)** from `fduty channel list`; **`rule-id` (MongoDB ObjectID string)**
from `escalate-rule-list`; **`template-id`** from `fduty template list`.

## Intent → verb

| want | verb |
|---|---|
| list escalation rules | `escalate-rule-list <channel-id>` |
| escalation rule detail | `escalate-rule-info` |
| add escalation rule | `escalate-rule-create` |
| edit escalation rule | `escalate-rule-update` |
| toggle escalation rule | `escalate-rule-enable` / `escalate-rule-disable` |
| remove escalation rule | `escalate-rule-delete` |

## Hot flow — add an escalation rule

```bash
# 1. find the channel and its existing rules
fduty channel escalate-rule-list <channel-id> --output-format toon
# 2. add the rule (layers is required via --data)
# API field `person_ids` expects member IDs from `fduty member list`.
fduty channel escalate-rule-create \
  --channel-id <channel-id> --rule-name "P1 on-call" --template-id <template-id> \
  --data '{"layers":[{"target":{"person_ids":[<member-id>],"by":{"critical":["voice","sms"],"warning":["feishu"]}},"notify_step":5,"max_times":3,"escalate_window":30}]}'
```

<!-- GENERATED:channel[escalate-rule] START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### escalate-rule-create
Create escalation rule
- `--aggr-window` int64 — Delay window in seconds. 0 disables delay. (0-3600)
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--priority` int64 — Evaluation priority. Lower runs first. (0-200)
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- `--template-id` string (required) — Notification template ID (MongoDB ObjectID).
- body-only (`--data`): filters (array<array<object>>); layers (array<object>) (required); time_filters (array<object>)
- response: single object (`data` unwrapped to the top level) — fields: rule_id (string); rule_name (string)

### escalate-rule-delete
Delete escalation rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### escalate-rule-disable
Disable escalation rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### escalate-rule-enable
Enable escalation rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### escalate-rule-info
Get escalation rule detail
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); aggr_window (integer); channel_id (integer); channel_name (string); created_at (integer); deleted_at (integer); description (string); filters (object); layers (array<object>); priority (integer); rule_id (string); rule_name (string); status (string); template_id (string); time_filters (array<object>); updated_at (integer); updated_by (integer)

### escalate-rule-list <channel-id>
List escalation rules
- `<channel-id>` (positional, required) int64 — Channel to list rules for.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); aggr_window (integer); channel_id (integer); channel_name (string); created_at (integer); deleted_at (integer); description (string); filters (object); layers (array<object>); priority (integer); rule_id (string); rule_name (string); status (string); template_id (string); time_filters (array<object>); updated_at (integer); updated_by (integer)

### escalate-rule-update
Update escalation rule
- `--aggr-window` int64 — Delay window in seconds. 0 disables delay.
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-id` string (required) — Escalation rule ID (MongoDB ObjectID).
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- `--template-id` string (required) — Notification template ID (MongoDB ObjectID).
- body-only (`--data`): filters (object); layers (array<object>) (required); time_filters (array<object>)

<!-- GENERATED:channel[escalate-rule] END -->

## Key concepts

- **`layers`** (required via `--data` on create/update): each layer needs
  `target` (with `person_ids` / `team_ids` / `schedule_to_role_ids` /
  `emails` + `by`, OR `webhooks`) and optionally `notify_step`, `max_times`,
  `escalate_window`, `force_escalate`.
- **`filters`** (optional) scope the rule to matching incidents — read
  `reference/filters.md` BEFORE composing them. Escalation filters match
  against the *incident* (its keys include `dedup_key`; alert-event keys like
  `alert_key` do not exist here).
- **Rule status**: `enabled` | `disabled`.

## Gotchas

- **`escalate-rule-create` needs `layers` via `--data`** — it is required and
  cannot be expressed as a flat flag. Omitting it returns a validation error.
- **`channel-id` is positional ONLY on `escalate-rule-list`**; every other
  `escalate-rule-*` verb takes it as the `--channel-id` flag.
- **`rule-id` is a MongoDB ObjectID string**, not an integer. Retrieve it
  from `escalate-rule-list` before any update/delete/enable/disable.

## Worked example — inspect a channel's escalation policy

```bash
fduty channel list --name "payments" --output-format toon
# → find channel_id (e.g. 4201)
fduty channel escalate-rule-list 4201 --output-format toon
# → find rule_id (MongoDB ObjectID string, e.g. "6643abc123def456789012aa")
fduty channel escalate-rule-info --channel-id 4201 --rule-id "6643abc123def456789012aa" --output-format toon
```
