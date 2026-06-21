# fduty channel — command card

Prereq: `SKILL.md` read. **SKILL.md + this card = full competence on channels — no `--help` needed.** Read verbs are free; `create`, `update`, `delete`, `escalate-rule-create/update/delete`, `inhibit-rule-*`, `silence-rule-*`, `unsubscribe-rule-*` all mutate state — confirm before acting. `delete` is **irreversible**.

## Route here when

"协作空间 / 频道 / 渠道 / 告警分组 / 降噪 / 静默 / 抑制 / 丢弃 / 升级策略 / 告警收敛 / channel / collaboration space / escalation rule / silence / inhibit / drop rule" → **channel**, NOT `incident` (incidents live _inside_ a channel) or `alert` (alerts are routed _into_ a channel). **`协作空间` (collaboration space) IS the `channel` API noun** — a naive translation would be "频道", but Flashduty's product surfaces it as 协作空间. Key IDs: **`channel-id` (int)** from `channel list`; **`rule-id` (MongoDB ObjectID string)** from `escalate-rule-list`, `inhibit-rule-list`, `silence-rule-list`, `unsubscribe-rule-list`.

## Intent → verb

| want | verb |
|---|---|
| list all channels (with team/name filter) | `list` |
| channel detail | `info <channel-id>` |
| batch fetch channels | `infos <channel-id> [id2 ...]` |
| create a channel | `create` |
| rename / reconfigure a channel | `update <channel-id>` |
| disable / re-enable a channel | `disable <channel-id>` / `enable <channel-id>` |
| delete a channel | `delete <channel-id>` |
| list escalation rules | `escalate-rule-list <channel-id>` |
| escalation rule detail | `escalate-rule-info` |
| add escalation rule | `escalate-rule-create` |
| edit escalation rule | `escalate-rule-update` |
| toggle escalation rule | `escalate-rule-enable` / `escalate-rule-disable` |
| remove escalation rule | `escalate-rule-delete` |
| list / create / update / toggle / delete inhibit rules | `inhibit-rule-list <channel-id>` / `inhibit-rule-create <channel-id>` / `inhibit-rule-update` / `inhibit-rule-enable` / `inhibit-rule-disable` / `inhibit-rule-delete` |
| list / create / update / toggle / delete silence rules | `silence-rule-list <channel-id>` / `silence-rule-create <channel-id>` / `silence-rule-update` / `silence-rule-enable` / `silence-rule-disable` / `silence-rule-delete` |
| list / create / update / toggle / delete drop (unsubscribe) rules | `unsubscribe-rule-list <channel-id>` / `unsubscribe-rule-create <channel-id>` / `unsubscribe-rule-update` / `unsubscribe-rule-enable` / `unsubscribe-rule-disable` / `unsubscribe-rule-delete` |

## Hot flow — create channel + add escalation rule

```bash
# 1. find owning team-id (from `fduty team list --output-format toon`)
fduty channel list --output-format toon
# 2. create the channel (no positional; --channel-name and --team-id are required)
fduty channel create --channel-name "production-api" --team-id <team-id> \
  --auto-resolve-timeout 3600 --auto-resolve-mode trigger
# → returns channel_id; use it below

# 3. add an escalation rule (all flags; layers is required via --data)
fduty channel escalate-rule-create \
  --channel-id <channel-id> --rule-name "P1 on-call" --template-id <template-id> \
  --data '{"layers":[{"target":{"person_ids":[<member-id>],"by":{"critical":["voice","sms"],"warning":["feishu"]}},"notify_step":5,"max_times":3,"escalate_window":30}]}'
```

## Hot flow — add a silence rule during maintenance

```bash
# channel-id is POSITIONAL on silence-rule-create (see use: "silence-rule-create <channel-id>")
fduty channel silence-rule-create <channel-id> \
  --rule-name "planned-maintenance-2026-07-01" \
  --is-auto-delete \
  --data '{"time_filter":{"start_time":1751328000,"end_time":1751371200}}'
# verify
fduty channel silence-rule-list <channel-id> --output-format toon
```

<!-- GENERATED:channel START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### create
Create channel
- `--auto-resolve-mode` string — Auto-resolve timer reset mode. · enum: trigger | update
- `--auto-resolve-timeout` int64 — Auto-resolve timeout in seconds. 0 disables auto-resolve. Max 30 days. (0-2592000)
- `--channel-name` string (required) — Channel name. 1 to 59 characters. (1-59 chars)
- `--description` string — Free-form description. Up to 500 characters. (≤500 chars)
- `--disable-auto-close` bool — Disable automatic incident closing.
- `--disable-outlier-detection` bool — Disable outlier incident detection.
- `--is-external-report-enabled` bool — Allow external reporters to file incidents into this channel.
- `--is-private` bool — When true, the channel is visible only to its managing teams.
- `--managing-team-ids` intSlice — Additional teams that can manage the channel. Up to 3 entries.
- `--plugin-ids` intSlice — IDs of plugins (integrations) subscribed to this channel.
- `--team-id` int64 (required) — Owning team ID.
- body-only (`--data`): escalate_rule (object); flapping (object); group (object)

### delete <channel-id>
Delete channel
- `<channel-id>` (positional, required) int64 — Channel ID.

### disable <channel-id>
Disable channel
- `<channel-id>` (positional, required) int64 — Channel ID.

### enable <channel-id>
Enable channel
- `<channel-id>` (positional, required) int64 — Channel ID.

### escalate-rule-create
Create escalation rule
- `--aggr-window` int64 — Aggregation window in seconds. 0 disables aggregation. (0-3600)
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--priority` int64 — Evaluation priority. Lower runs first. (0-200)
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- `--template-id` string (required) — Notification template ID (MongoDB ObjectID).
- body-only (`--data`): filters (array<array>); layers (array<object>) (required); time_filters (array<object>)

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

### escalate-rule-list <channel-id>
List escalation rules
- `<channel-id>` (positional, required) int64 — Channel to list rules for.

### escalate-rule-update
Update escalation rule
- `--aggr-window` int64 — Aggregation window in seconds. 0 disables aggregation.
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-id` string (required) — Escalation rule ID (MongoDB ObjectID).
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- `--template-id` string (required) — Notification template ID (MongoDB ObjectID).
- body-only (`--data`): filters (object); layers (array<object>) (required); time_filters (array<object>)

### info <channel-id>
Get channel detail
- `<channel-id>` (positional, required) int64 — Channel ID to fetch.

### infos <channel-id> [<id2>...]
Batch get channels
- `<channel-ids>` (positional, required) intSlice — Channel IDs to look up. Up to 1000.

### inhibit-rule-create <channel-id>
Create inhibit rule
- `<channel-id>` (positional, required) int64 — Channel the rule belongs to.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--equals` stringSlice (required) — Label keys used to pair source and target alerts.
- `--is-directly-discard` bool — When true, suppressed target alerts are dropped instead of merged.
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- body-only (`--data`): source_filters (array<array>); target_filters (array<array>)

### inhibit-rule-delete
Delete inhibit rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### inhibit-rule-disable
Disable inhibit rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### inhibit-rule-enable
Enable inhibit rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### inhibit-rule-list <channel-id>
List inhibit rules
- `<channel-id>` (positional, required) int64 — Channel to list rules for.

### inhibit-rule-update
Update inhibit rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--equals` stringSlice (required) — Label keys used to pair source and target alerts.
- `--is-directly-discard` bool — When true, suppressed target alerts are dropped instead of merged.
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-id` string (required) — Inhibit rule ID (MongoDB ObjectID).
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- body-only (`--data`): source_filters (object); target_filters (object)

### list
List channels
- `--name` string
- `--team-ids` int64Slice

### silence-rule-create <channel-id>
Create silence rule
- `<channel-id>` (positional, required) int64 — Channel the rule belongs to.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--from-incident-id` string — Source incident ID when the silence was created from an incident.
- `--is-auto-delete` bool — When true, the silence rule is automatically deleted after its time window expires. Defaults to false.
- `--is-directly-discard` bool — When true, silenced alerts are dropped instead of suppressed into incidents.
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- body-only (`--data`): filters (array<array>); time_filter (object); time_filters (array<object>)

### silence-rule-delete
Delete silence rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### silence-rule-disable
Disable silence rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### silence-rule-enable
Enable silence rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### silence-rule-list <channel-id>
List silence rules
- `<channel-id>` (positional, required) int64 — Channel to list rules for.

### silence-rule-update
Update silence rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--is-auto-delete` bool — When true, the silence rule is automatically deleted after its time window expires. Defaults to false.
- `--is-directly-discard` bool — When true, silenced alerts are dropped instead of suppressed into incidents.
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-id` string (required) — Silence rule ID (MongoDB ObjectID).
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- body-only (`--data`): filters (object); time_filter (object); time_filters (array<object>)

### unsubscribe-rule-create <channel-id>
Create drop rule
- `<channel-id>` (positional, required) int64 — Channel the rule belongs to.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- body-only (`--data`): filters (array<array>)

### unsubscribe-rule-delete
Delete drop rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### unsubscribe-rule-disable
Disable drop rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### unsubscribe-rule-enable
Enable drop rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--rule-id` string (required) — Rule ID (MongoDB ObjectID).

### unsubscribe-rule-list <channel-id>
List drop rules
- `<channel-id>` (positional, required) int64 — Channel to list rules for.

### unsubscribe-rule-update
Update drop rule
- `--channel-id` int64 (required) — Channel the rule belongs to.
- `--description` string — Rule description, up to 500 characters. (≤500 chars)
- `--priority` int64 — Evaluation priority. Lower runs first.
- `--rule-id` string (required) — Drop rule ID (MongoDB ObjectID).
- `--rule-name` string (required) — Rule name, 1 to 39 characters. (1-39 chars)
- body-only (`--data`): filters (object)

### update <channel-id>
Update channel
- `--auto-resolve-mode` string — Auto-resolve timer reset mode. · enum: trigger | update
- `--auto-resolve-timeout` int64 — Auto-resolve timeout in seconds. 0 disables auto-resolve. Max 30 days. (0-2592000)
- `<channel-id>` (positional, required) int64 — Channel ID to update.
- `--channel-name` string — New channel name. 1 to 59 characters. (1-59 chars)
- `--description` string — New description. Up to 500 characters. (≤500 chars)
- `--disable-auto-close` bool — Disable automatic incident closing.
- `--disable-outlier-detection` bool — Disable outlier incident detection.
- `--is-external-report-enabled` bool — Allow external reporters to file incidents into this channel.
- `--is-private` bool — When true, the channel is visible only to its managing teams.
- `--managing-team-ids` intSlice — Additional teams that can manage the channel. Up to 3 entries.
- `--team-id` int64 — New owning team ID.
- body-only (`--data`): flapping (object); group (object)

<!-- GENERATED:channel END -->

## Key concepts

- **`--auto-resolve-mode`** enum: `trigger` (timer resets on each new alert trigger) | `update` (timer resets on any alert update).
- **Alert grouping `group.method`**: `i` = intelligent (embedding similarity), `p` = pattern (label equality), `n` = none. Set via `--data '{"group":{"method":"p","equals":[["service","env"]],"time_window":300}}'` on `create`/`update`.
- **Rule status**: `enabled` | `disabled` — apply to escalation, inhibit, silence, and drop rules alike.
- **Inhibit `--equals`**: label keys that must be **equal** between the source (high-priority) and target (suppressed) alert to form a pair (e.g. `--equals service,env`).
- **Silence time windows**: `time_filter` (one-off, unix seconds, mutually exclusive) vs `time_filters` (recurring weekly HH:MM windows). Pass via `--data`.
- **Escalation `layers`** (required via `--data` on create/update): each layer needs `target` (with `person_ids`/`team_ids`/`schedule_to_role_ids`/`emails` + `by` OR `webhooks`) and optionally `notify_step`, `max_times`, `escalate_window`, `force_escalate`.

## Gotchas

- **Positional trap**: `channel-id` is **positional** on `info`, `infos`, `update`, `delete`, `disable`, `enable`, `escalate-rule-list`, `inhibit-rule-create`, `inhibit-rule-list`, `silence-rule-create`, `silence-rule-list`, `unsubscribe-rule-create`, `unsubscribe-rule-list`. It is a **flag** (`--channel-id`) on all `escalate-rule-*`, `inhibit-rule-update/delete/enable/disable`, `silence-rule-update/delete/enable/disable`, `unsubscribe-rule-update/delete/enable/disable`. When in doubt, the fence heading `### verb <channel-id>` = positional; heading without `<…>` = flag.
- **`escalate-rule-create` needs `layers` via `--data`** — it is required and cannot be expressed as a flat flag. Omitting it returns a validation error.
- **`rule-id` is a MongoDB ObjectID string**, not an integer. Retrieve it from `escalate-rule-list`, `inhibit-rule-list`, `silence-rule-list`, or `unsubscribe-rule-list` before any update/delete/enable/disable.
- **`channel create` requires `--channel-name` and `--team-id`** even though they are not marked `required` in the flag list — the server rejects the request without them.
- **`delete` on a channel is irreversible** — all rules within it are also removed. Confirm the `channel-id` against `list` before proceeding.
- **Empty rule list is authoritative** — if `escalate-rule-list` / `silence-rule-list` / etc. returns no rows, no rules exist; do not widen the query.
- **`list` response is a top-level array** (pipe `jq '.[]'`); rule-list responses nest under `items[]` (pipe `jq '.items[]'`).

## Worked example — look up a channel and inspect its escalation policy

```bash
fduty channel list --name "payments" --output-format toon
# → find channel_id (e.g. 4201)
fduty channel escalate-rule-list 4201 --output-format toon
# → find rule_id (MongoDB ObjectID string, e.g. "6643abc123def456789012aa")
fduty channel escalate-rule-info --channel-id 4201 --rule-id "6643abc123def456789012aa" --output-format toon
```
