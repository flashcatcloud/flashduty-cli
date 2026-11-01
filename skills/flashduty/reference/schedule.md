# flashduty schedule — command card

Prereq: `SKILL.md` read. **Read verbs are free. `delete` is irreversible — confirm IDs before executing. `create` / `update` immediately change the live rotation — confirm scope first.**

## Route here when

"值班 / 排班 / 轮班 / 轮值 / 值班表 / 班表 / 谁在值班 / 当前值班 / 下一班 / 排班配置 / on-call / who is on call / schedule / rotation / shift / next on call / view or edit shifts" → **schedule**. This is the single home for everything 值班/on-call. The key ID you need is **`schedule_id` (int)** — get it from `schedule list`.

**Who is on call right now** is computed from a schedule, not stored: `schedule info <id> --start now --end +1h` returns the current shift (and its `person_ids`). The legacy **`oncall who`** returns the same data across *all* schedules in one call, but the response is 77-82KB (~2800 lines) — it always spills to a file, and still takes 3-10 follow-up reads to dig the 3-4 fields you wanted out of it. For several schedules at once, scope it instead: `schedule list --query <name>` or `--team-ids <id>` to get the schedule IDs, then `schedule info <id> --start now --end +1h` per schedule. The **`oncall` command group is being deprecated and folded into `schedule`** — do not build new flows on `oncall *`.

## Intent → verb

| want | verb |
|---|---|
| who is on call right now (one schedule) | `info <schedule-id> --start now --end +1h` |
| who is on call right now (all schedules, legacy) | `oncall who` — *deprecated, whole-account dump (~80KB); prefer `schedule list` + per-schedule `info`* |
| list all schedules (with name search / team filter) | `list` |
| schedules I am assigned to | `self` |
| detail + computed shifts for a schedule | `info <schedule-id>` |
| batch-fetch multiple schedules (no shifts) | `infos <schedule-id> [<id2>…]` |
| preview a schedule definition before saving | `preview` |
| create a new schedule | `create` |
| update an existing schedule | `update` |
| delete one or more schedules | `delete <schedule-id> [<id2>…]` |

## Hot flow — who is on call right now

```bash
# 1. Find the schedule ID(s) — scope by name or team; don't fetch every schedule
flashduty schedule list --query "SRE" --output-format toon
# or, for several: flashduty schedule list --team-ids <team-id> --output-format toon

# 2. Current on-call for each schedule — a tiny now-window yields the live shift
flashduty schedule info <schedule-id> --start now --end +1h --output-format toon
```

`--output-format toon` is for reading, not piping — it is **not** jq-parseable. If you need to filter/extract with `jq`, use `--json` instead.

`schedule info`'s on-call groups carry `person_ids` (integers), not names. Resolve every id in **one batch call** with `flashduty person infos` (the sibling `person` group — takes positional ids or `--person-ids`):

```bash
# person_ids come straight from the schedule/oncall output above
flashduty person infos <person-id> [<person-id2> …] --output-format toon
# → rows under .items[] with person_id + person_name; join on person_id client-side
```

**`person_id` ≠ `member_id` — do NOT resolve schedule/oncall people via `member list`.** They are different id namespaces, so matching `member list` rows on `member_id == <person_id>` is wrong, and paginating the full roster (often 20+ pages) silently drops people who land on later pages — a real prod miss. Always feed the `person_id`s to `flashduty person infos`. If a lookup genuinely fails, report the bare `person_id` rather than guessing.

## Hot flow — inspect a schedule's upcoming shifts

```bash
flashduty schedule list --query "SRE" --output-format toon          # find the schedule_id
flashduty schedule info <schedule-id> --start now --end +7d --output-format toon   # next 7 days
```

## Hot flow — create a schedule via --data

```bash
# Layers are deeply nested; pass the full body via --data; scalar flags override matching keys.
flashduty schedule create --schedule-name "SRE Weekly" --team-id <team-id> \
  --data '{
    "layers": [{
      "layer_name": "Week rotation",
      "mode": 0,
      "rotation_unit": "week",
      "rotation_value": 1,
      "rotation_duration": 604800,
      "handoff_time": 0,
      "enable_time": 1700000000,
      "expire_time": 0,
      "weight": 1,
      "hidden": 0,
      "fair_rotation": false,
      "restrict_mode": 0,
      "restrict_start": 0,
      "restrict_end": 0,
      "restrict_periods": [],
      "mask_continuous_enabled": false,
      "day_mask": {"repeat": [1,2,3,4,5]},
      "groups": [{
        "group_name": "Group A",
        "name": "group_a",
        "start": 1700000000,
        "end": 1700604800,
        "members": [{"person_ids": [<person-id>], "role_id": 0}]
      }],
      "name": "layer_1",
      "schedule_id": 0,
      "account_id": 0,
      "create_at": 0,
      "create_by": 0,
      "update_at": 0,
      "update_by": 0
    }]
  }'
# → returns schedule_id; verify with: flashduty schedule info <schedule-id> --start now --end +7d
```

<!-- GENERATED:schedule START · 由 flashduty __dump-commands 同步 · 勿手改 fence 内 -->

### by-person <person-id>
Get member on-call status
- `<person-id>` (positional, required) int64 — Member ID whose on-call status is returned.
- response: single object (`data` unwrapped to the top level) — fields: current (any); next (any); schedules (array<object>)

### create
Create schedule
- `--description` string — Schedule description, at most 500 characters. On update, omitting it keeps the current text and an empty string clears it. (≤500 chars)
- `--disabled` int64 — Create only: '0' = enabled (default), '1' = disabled. Ignored on update — use 'status' instead.
- `--end` string — Preview window end, 10-digit Unix seconds, after 'start' and at most 45 days later. Required for '/schedule/preview'; ignored by create and update. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--name` string — Legacy name field, used on create only when 'schedule_name' is empty. Ignored on update. (≤40 chars)
- `--schedule-id` int64 — Schedule ID, required on update; obtain it from 'POST /schedule/list'.
- `--schedule-name` string — Schedule name, at most 40 characters, unique within the account. On update, omitting it keeps the current name. (≤40 chars)
- `--start` string — Preview window start, 10-digit Unix seconds. Required for '/schedule/preview'; ignored by create and update. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--status` int64 — '0' = enabled, '1' = disabled. On update, omitting it keeps the current state. On create it takes precedence over 'disabled'.
- `--team-id` int64 — ID of the owning team; obtain it from 'POST /team/list'. On create, omitting it leaves the schedule without a team ('0'); on update, omitting it keeps the current team.
- `--time-zone` string — IANA time zone in which the schedule's wall-clock rules are evaluated, for example 'Asia/Shanghai' or 'UTC'; abbreviations such as 'EST' and 'Etc/GMT±N' are rejected. Set on create (default: evaluated in 'Asia/Shanghai'); immutable afterwards — on update send nothing or the current value. On preview it sets the evaluation time zone.
- body-only (`--data`): layers (array<object>); notify (any)
- response: single object (`data` unwrapped to the top level) — fields: schedule_id (integer)

### delete <schedule-id> [<id2>...]
Delete schedules
- `<schedule-ids>` (positional, required) intSlice — Schedule IDs to operate on; obtain them from 'POST /schedule/list'.

### info <schedule-id>
Get schedule info
- `--end` string (required) — Preview end timestamp (Unix seconds, 10 digits). Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `<schedule-id>` (positional, required) int64 — Schedule ID; obtain it from 'POST /schedule/list'.
- `--start` string (required) — Preview start timestamp (Unix seconds, 10 digits). Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); create_at (string); create_by (integer); cur_oncall (any); description (string); disabled (integer); end (string); final_schedule (object); group_id (integer); id (integer); layer_schedules (array<object>); layers (array<object>); name (string); next_oncall (any); notify (any); schedule_id (integer); schedule_layers (array<object>); schedule_name (string); start (string); status (integer); team_id (integer); time_zone (string); update_at (string); update_by (integer)

### infos <schedule-id> [<id2>...]
Batch get schedules
- `<schedule-ids>` (positional, required) intSlice — Schedule ID list; obtain IDs from 'POST /schedule/list'.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); create_at (string); create_by (integer); cur_oncall (any); description (string); disabled (integer); end (string); final_schedule (object); group_id (integer); id (integer); layer_schedules (array<object>); layers (array<object>); name (string); next_oncall (any); notify (any); schedule_id (integer); schedule_layers (array<object>); schedule_name (string); start (string); status (integer); team_id (integer); time_zone (string); update_at (string); update_by (integer)

### list
List schedules
- `--end` string — Window end, 10-digit Unix seconds, after 'start' and at most 45 days later. Sending 'end' without 'start' is rejected. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--is-my-manage` bool — Only schedules owned by the caller's teams or created by the caller. 'team_ids' narrows the team part; team IDs outside the caller's teams are ignored.
- `--is-my-team` bool — Only schedules owned by the caller's teams. When 'team_ids' shares no team with the caller, or the caller belongs to no team, the list is empty.
- `--limit` int64 — Page size. Default 10; values above 100 are treated as 100. (max 100)
- `--page` int64 — Page number (1-indexed).
- `--query` string — Substring matched against the schedule name or description.
- `--search-after-ctx` string
- `--start` string — Window start, 10-digit Unix seconds. Send together with 'end' to compute layers and shifts. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-ids` intSlice — Only schedules owned by these teams. With 'is_my_team' or 'is_my_manage', only the teams the caller belongs to are kept.
- response: `{items: [...], total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); create_at (string); create_by (integer); cur_oncall (any); description (string); disabled (integer); end (string); final_schedule (object); group_id (integer); id (integer); layer_schedules (array<object>); layers (array<object>); name (string); next_oncall (any); notify (any); schedule_id (integer); schedule_layers (array<object>); schedule_name (string); start (string); status (integer); team_id (integer); time_zone (string); update_at (string); update_by (integer)

### preview
Preview schedule
- `--description` string — Schedule description, at most 500 characters. On update, omitting it keeps the current text and an empty string clears it. (≤500 chars)
- `--disabled` int64 — Create only: '0' = enabled (default), '1' = disabled. Ignored on update — use 'status' instead.
- `--end` string — Preview window end, 10-digit Unix seconds, after 'start' and at most 45 days later. Required for '/schedule/preview'; ignored by create and update. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--name` string — Legacy name field, used on create only when 'schedule_name' is empty. Ignored on update. (≤40 chars)
- `--schedule-id` int64 — Schedule ID, required on update; obtain it from 'POST /schedule/list'.
- `--schedule-name` string — Schedule name, at most 40 characters, unique within the account. On update, omitting it keeps the current name. (≤40 chars)
- `--start` string — Preview window start, 10-digit Unix seconds. Required for '/schedule/preview'; ignored by create and update. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--status` int64 — '0' = enabled, '1' = disabled. On update, omitting it keeps the current state. On create it takes precedence over 'disabled'.
- `--team-id` int64 — ID of the owning team; obtain it from 'POST /team/list'. On create, omitting it leaves the schedule without a team ('0'); on update, omitting it keeps the current team.
- `--time-zone` string — IANA time zone in which the schedule's wall-clock rules are evaluated, for example 'Asia/Shanghai' or 'UTC'; abbreviations such as 'EST' and 'Etc/GMT±N' are rejected. Set on create (default: evaluated in 'Asia/Shanghai'); immutable afterwards — on update send nothing or the current value. On preview it sets the evaluation time zone.
- body-only (`--data`): layers (array<object>); notify (any)
- response: same shape as `info <schedule-id>` above

### self
List my schedules
- `--end` string (required) — Window end (Unix seconds, 10 digits). Required. Must be within 45 days of start. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--start` string (required) — Window start (Unix seconds, 10 digits). Required. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- response: same shape as `infos <schedule-id> [<id2>...]` above

### update
Update schedule
- `--description` string — Schedule description, at most 500 characters. On update, omitting it keeps the current text and an empty string clears it. (≤500 chars)
- `--disabled` int64 — Create only: '0' = enabled (default), '1' = disabled. Ignored on update — use 'status' instead.
- `--end` string — Preview window end, 10-digit Unix seconds, after 'start' and at most 45 days later. Required for '/schedule/preview'; ignored by create and update. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--name` string — Legacy name field, used on create only when 'schedule_name' is empty. Ignored on update. (≤40 chars)
- `--schedule-id` int64 — Schedule ID, required on update; obtain it from 'POST /schedule/list'.
- `--schedule-name` string — Schedule name, at most 40 characters, unique within the account. On update, omitting it keeps the current name. (≤40 chars)
- `--start` string — Preview window start, 10-digit Unix seconds. Required for '/schedule/preview'; ignored by create and update. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--status` int64 — '0' = enabled, '1' = disabled. On update, omitting it keeps the current state. On create it takes precedence over 'disabled'.
- `--team-id` int64 — ID of the owning team; obtain it from 'POST /team/list'. On create, omitting it leaves the schedule without a team ('0'); on update, omitting it keeps the current team.
- `--time-zone` string — IANA time zone in which the schedule's wall-clock rules are evaluated, for example 'Asia/Shanghai' or 'UTC'; abbreviations such as 'EST' and 'Etc/GMT±N' are rejected. Set on create (default: evaluated in 'Asia/Shanghai'); immutable afterwards — on update send nothing or the current value. On preview it sets the evaluation time zone.
- body-only (`--data`): layers (array<object>); notify (any)

<!-- GENERATED:schedule END -->

## Key concepts

- **Window for shifts:** `info` (and `list` when `--start`+`--end` are both set) computes actual rotation slots in the requested window. Max span = 45 days. `info` requires both `--start` and `--end`.
- **`self` window:** returns schedules the current user is assigned to in the given window. Max span = 30 days.
- **Layer `mode`:** `0` = common rotation, `1` = override layer (higher `weight` wins).
- **`rotation_unit`:** `hour | day | week | month`.
- **`restrict_mode`:** `0` = none, `1` = restrict by day, `2` = restrict by week.
- **`expire_time: 0`** means the layer never expires (open-ended).

## Gotchas

- **`info` takes `<schedule-id>` positionally or via `--schedule-id`** — both work: `flashduty schedule info 123 --start now --end +7d` or `flashduty schedule info --schedule-id 123 --start now --end +7d`. If both are given, the flag wins. **`infos` and `delete` have no singular `--schedule-id` flag** — only the plural `--schedule-ids` (comma-separated), which folds the same way as the positional list; passing `--schedule-id` there is an unknown-flag error, not a rejected alternative.
- **`create` / `update` / `preview` take all inputs as flags** (no positional). `update` requires `--schedule-id` as a flag to identify the target.
- **`layers` is body-only.** There is no per-layer typed flag — you must pass the entire `layers` array via `--data`. Scalar top-level flags (`--schedule-name`, `--team-id`) override matching `--data` keys.
- **`list` without `--start`/`--end` omits computed shifts** — only schedule metadata is returned. Pass both flags (≤45 day span) to get rotation slots in the list response.
- **`delete` is irreversible** — takes one or more `<schedule-id>` positionals; double-check IDs before executing.
- **`list` default page size is 10** — pass `--limit 100` when scanning all schedules.
- **Legacy `oncall who`:** returns ALL schedules in one call (77-82KB / ~2800 lines, always spills to a file) — prefer `schedule list --query|--team-ids` + per-schedule `info` instead. If you do use it: `--team` does **not** filter server-side (any value returns the full list — scope by `--query <schedule_name>`), and an empty result is authoritative ("no one on call in that window") — report it, don't widen or fabricate a responder. The `oncall` group will be removed; don't depend on it.

## Worked example

```bash
# Find my own on-call windows for the next two weeks
flashduty schedule self --start now --end +14d --output-format toon
```
