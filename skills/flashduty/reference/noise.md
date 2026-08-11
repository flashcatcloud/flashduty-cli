# fduty channel noise rules — silence / inhibit / drop

Prereq: `SKILL.md` read. Read verbs (`*-rule-list`) are free; every
`silence-rule-*`, `inhibit-rule-*`, `unsubscribe-rule-*` create / update /
enable / disable / delete mutates state — confirm before acting.

## Route here when

"静默 / 屏蔽 / 抑制 / 丢弃 / 降噪 / 维护窗口 / silence / mute / inhibit /
suppress / drop / discard / noise reduction / maintenance window" → this
card. These rules live INSIDE a channel (协作空间): **`channel-id` (int)**
from `fduty channel list` (channel management: `reference/channel.md`);
**`rule-id` (MongoDB ObjectID string)** from the matching `*-rule-list`.
Escalation / 分派策略 → `reference/escalation.md`.

Rule semantics: **silence** suppresses notifications for matching alerts in a
time window (alerts still arrive); **inhibit** suppresses target alerts while
a matching source alert is active; **drop (unsubscribe)** discards matching
alerts outright. Silence and inhibit can also discard instead of suppress via
`--is-directly-discard`.

## Intent → verb

| want | verb |
|---|---|
| list / create / update / toggle / delete silence rules | `silence-rule-list <channel-id>` / `silence-rule-create <channel-id>` / `silence-rule-update` / `silence-rule-enable` / `silence-rule-disable` / `silence-rule-delete` |
| list / create / update / toggle / delete inhibit rules | `inhibit-rule-list <channel-id>` / `inhibit-rule-create <channel-id>` / `inhibit-rule-update` / `inhibit-rule-enable` / `inhibit-rule-disable` / `inhibit-rule-delete` |
| list / create / update / toggle / delete drop (unsubscribe) rules | `unsubscribe-rule-list <channel-id>` / `unsubscribe-rule-create <channel-id>` / `unsubscribe-rule-update` / `unsubscribe-rule-enable` / `unsubscribe-rule-disable` / `unsubscribe-rule-delete` |

## Hot flow — add a silence rule during maintenance

A silence rule needs BOTH a time window (`time_filter` or `time_filters`) AND
`filters` naming which alerts the window applies to — a `time_filter`-only
rule matches nothing and the server rejects it. Build `filters` from the
target incident's own labels — read `reference/filters.md` first for the
construction rules and the valid key set.

```bash
# 1. inspect the incident to silence around — pulls incident_severity + labels
fduty incident detail <incident-id> --output-format toon

# 2. channel-id is POSITIONAL on silence-rule-create (see use: "silence-rule-create <channel-id>")
# filters is one AND group: a severity condition plus one labels.<key> condition
# per distinguishing label — id-shaped/long/date-shaped/noise-key label values are
# dropped, not passed through (construction rules: reference/filters.md).
fduty channel silence-rule-create <channel-id> \
  --rule-name "planned-maintenance-2026-07-01" \
  --is-auto-delete \
  --data '{"time_filter":{"start_time":1751328000,"end_time":1751371200},"filters":[[{"key":"severity","oper":"IN","vals":["Critical"]},{"key":"labels.service","oper":"IN","vals":["payments-api"]},{"key":"labels.env","oper":"IN","vals":["prod"]}]]}'

# 3. verify — read back `filters` to confirm the conditions round-tripped
fduty channel silence-rule-list <channel-id> --output-format toon
```

<!-- GENERATED:channel[silence-rule,inhibit-rule,unsubscribe-rule] START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### inhibit-rule-create <channel-id>
Create inhibit rule
- `<channel-id>` (positional, required) int64 — Owning channel ID; obtain it from 'POST /channel/list'.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--equals` stringSlice (required) — Label keys used to pair source and target alerts.
- `--is-directly-discard` bool — When true, suppressed target alerts are dropped instead of merged.
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- body-only (`--data`): source_filters (array<array<object>>); target_filters (array<array<object>>)
- response: single object (`data` unwrapped to the top level) — fields: rule_id (string); rule_name (string)

### inhibit-rule-delete
Delete inhibit rule
- `--channel-id` int64 (required) — Owning channel ID; obtain it from 'POST /channel/list'.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### inhibit-rule-disable
Disable inhibit rule
- `--channel-id` int64 (required) — Owning channel ID; obtain it from 'POST /channel/list'.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### inhibit-rule-enable
Enable inhibit rule
- `--channel-id` int64 (required) — Owning channel ID; obtain it from 'POST /channel/list'.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### inhibit-rule-list <channel-id>
List inhibit rules
- `<channel-id>` (positional, required) int64 — Channel to list rules for.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); channel_id (integer); created_at (integer); deleted_at (integer); description (string); equals (array<string>); is_directly_discard (boolean); priority (integer); rule_id (string); rule_name (string); source_filters (object); status (string); target_filters (object); updated_at (integer); updated_by (integer)

### inhibit-rule-update
Update inhibit rule
- `--channel-id` int64 (required) — Owning channel ID; obtain it from 'POST /channel/list'.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--equals` stringSlice (required) — Label keys used to pair source and target alerts.
- `--is-directly-discard` bool — When true, suppressed target alerts are dropped instead of merged.
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-id` string (required) — Inhibit rule ID (MongoDB ObjectID).
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- body-only (`--data`): source_filters (object); target_filters (object)

### silence-rule-create <channel-id>
Create silence rule
- `<channel-id>` (positional, required) int64 — Owning channel ID; obtain it from 'POST /channel/list'.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--from-incident-id` string — Source incident ID when the silence was created from an incident.
- `--is-auto-delete` bool — When true, the silence rule is automatically deleted after its time window expires. Defaults to false.
- `--is-directly-discard` bool — When true, silenced alerts are dropped instead of suppressed into incidents.
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- body-only (`--data`): filters (array<array<object>>); time_filter (object); time_filters (array<object>)
- response: same shape as `inhibit-rule-create <channel-id>` above

### silence-rule-delete
Delete silence rule
- `--channel-id` int64 (required) — Owning channel ID; obtain it from 'POST /channel/list'.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### silence-rule-disable
Disable silence rule
- `--channel-id` int64 (required) — Owning channel ID; obtain it from 'POST /channel/list'.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### silence-rule-enable
Enable silence rule
- `--channel-id` int64 (required) — Owning channel ID; obtain it from 'POST /channel/list'.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### silence-rule-list <channel-id>
List silence rules
- `<channel-id>` (positional, required) int64 — Channel to list rules for.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); channel_id (integer); created_at (integer); deleted_at (integer); description (string); filters (object); from_incident_id (string); is_auto_delete (boolean); is_directly_discard (boolean); is_effective (boolean); priority (integer); rule_id (string); rule_name (string); status (string); time_filter (object); time_filters (array<object>); updated_at (integer); updated_by (integer)

### silence-rule-update
Update silence rule
- `--channel-id` int64 (required) — Owning channel ID; obtain it from 'POST /channel/list'.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--is-auto-delete` bool — When true, the silence rule is automatically deleted after its time window expires. Defaults to false.
- `--is-directly-discard` bool — When true, silenced alerts are dropped instead of suppressed into incidents.
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-id` string (required) — Silence rule ID (MongoDB ObjectID).
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- body-only (`--data`): filters (object); time_filter (object); time_filters (array<object>)

### unsubscribe-rule-create <channel-id>
Create drop rule
- `<channel-id>` (positional, required) int64 — Owning channel ID; obtain it from 'POST /channel/list'.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- body-only (`--data`): filters (array<array<object>>)
- response: same shape as `inhibit-rule-create <channel-id>` above

### unsubscribe-rule-delete
Delete drop rule
- `--channel-id` int64 (required) — Owning channel ID; obtain it from 'POST /channel/list'.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### unsubscribe-rule-disable
Disable drop rule
- `--channel-id` int64 (required) — Owning channel ID; obtain it from 'POST /channel/list'.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### unsubscribe-rule-enable
Enable drop rule
- `--channel-id` int64 (required) — Owning channel ID; obtain it from 'POST /channel/list'.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### unsubscribe-rule-list <channel-id>
List drop rules
- `<channel-id>` (positional, required) int64 — Channel to list rules for.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); channel_id (integer); created_at (integer); deleted_at (integer); description (string); filters (object); priority (integer); rule_id (string); rule_name (string); status (string); updated_at (integer); updated_by (integer)

### unsubscribe-rule-update
Update drop rule
- `--channel-id` int64 (required) — Owning channel ID; obtain it from 'POST /channel/list'.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-id` string (required) — Drop rule ID (MongoDB ObjectID).
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- body-only (`--data`): filters (object)

<!-- GENERATED:channel[silence-rule,inhibit-rule,unsubscribe-rule] END -->

## Key concepts

- **`filters` / `source_filters` / `target_filters` construction** — read
  `reference/filters.md` BEFORE composing any of them (shape, operators, and
  the valid key set; a wrong key silently never matches).
- **Silence time windows**: `time_filter` (one-off, unix seconds) vs
  `time_filters` (recurring weekly HH:MM windows) — mutually exclusive. Pass
  via `--data`.
- **Inhibit `--equals`**: label keys that must be **equal** between the
  source (high-priority) and target (suppressed) alert to form a pair (e.g.
  `--equals service,env`).
- **Rule status**: `enabled` | `disabled` — applies to all three rule kinds.

## Gotchas

- **Positional trap**: `channel-id` is **positional** on every `*-rule-create`
  and `*-rule-list` verb here; it is a **flag** (`--channel-id`) on every
  `*-rule-update/delete/enable/disable`. The fence heading
  `### verb <channel-id>` = positional; heading without `<…>` = flag.
- **`rule-id` is a MongoDB ObjectID string**, not an integer. Retrieve it
  from the matching `*-rule-list` before any update/delete/enable/disable.
- **Empty rule list is authoritative** — if `silence-rule-list` /
  `inhibit-rule-list` / `unsubscribe-rule-list` returns no rows, no rules
  exist; do not widen the query.
