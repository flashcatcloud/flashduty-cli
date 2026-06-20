# fduty schedule — command card

Prereq: `SKILL.md` read. **Read verbs are free. `delete` is irreversible — confirm IDs before executing. `create` / `update` immediately change the live rotation — confirm scope first.**

## Route here when

"排班 / 轮班 / 值班表 / on-call schedule / rotation / 谁值班 / 班表 / 排班配置 / view shifts / edit schedule" → **schedule**. For *who is on duty right now* without editing, `schedule info` (with `--start`/`--end`) or `schedule list` with a window both compute live shifts. The key ID you need is **`schedule_id` (int)** — get it from `schedule list`.

Sibling domains: `oncall` (a separate skill for who-is-on-call queries via `oncall who`). `schedule` is for listing, inspecting, creating, and updating schedule definitions.

## Intent → verb

| want | verb |
|---|---|
| list all schedules (with name search / team filter) | `list` |
| schedules I am assigned to | `self` |
| detail + computed shifts for a schedule | `info <schedule-id>` |
| batch-fetch multiple schedules (no shifts) | `infos <schedule-id> [<id2>…]` |
| preview a schedule definition before saving | `preview` |
| create a new schedule | `create` |
| update an existing schedule | `update` |
| delete one or more schedules | `delete <schedule-id> [<id2>…]` |

## Hot flow — look up who is on call this week

```bash
# 1. Find the schedule ID
fduty schedule list --query "SRE" --output-format toon

# 2. Inspect it — computed shifts for the next 7 days (positional id; --start/--end required)
fduty schedule info <schedule-id> --start now --end +7d --output-format toon
```

## Hot flow — create a schedule via --data

```bash
# Layers are deeply nested; pass the full body via --data; scalar flags override matching keys.
fduty schedule create --schedule-name "SRE Weekly" --team-id <team-id> \
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
# → returns schedule_id; verify with: fduty schedule info <schedule-id> --start now --end +7d
```

<!-- GENERATED:schedule START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### create
Create schedule
- `--description` string — Schedule description. Max 500 characters. (≤500 chars)
- `--end` string — Preview window end (Unix seconds, 10 digits). Required for /schedule/preview. Max 45 days after start. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--name` string — Legacy schedule name field. Used when schedule_name is empty. (≤40 chars)
- `--schedule-id` int64 — Schedule ID. Required on update.
- `--schedule-name` string — Schedule display name. Max 40 characters. (≤40 chars)
- `--start` string — Preview window start (Unix seconds, 10 digits). Required for /schedule/preview. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-id` int64 — Owning team ID.
- body-only (`--data`): layers (array<object>); notify (object)

### delete <schedule-id> [<id2>...]
Delete schedules
- `<schedule-ids>` (positional, required) intSlice — Schedule IDs to operate on.

### info <schedule-id>
Get schedule info
- `--end` string (required) — Preview end timestamp (Unix seconds, 10 digits). Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `<schedule-id>` (positional, required) int64 — Schedule ID.
- `--start` string (required) — Preview start timestamp (Unix seconds, 10 digits). Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.

### infos <schedule-id> [<id2>...]
Batch get schedules
- `<schedule-ids>` (positional, required) intSlice — Schedule ID list.

### list
List schedules
- `--end` string — Window end timestamp (Unix seconds). Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--is-my-manage` bool — Only return schedules created by the current user within their teams.
- `--is-my-team` bool — Only return schedules whose owning team the current user belongs to.
- `--limit` int64 — Page size. Default 10, max 100. (max 100)
- `--page` int64 — Page number (1-indexed).
- `--query` string — Search keyword matched against schedule names.
- `--search-after-ctx` string
- `--start` string — When set together with end, computed layer schedules are returned. Span must be less than 45 days. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-ids` intSlice — Filter by team IDs.

### preview
Preview schedule
- `--description` string — Schedule description. Max 500 characters. (≤500 chars)
- `--end` string — Preview window end (Unix seconds, 10 digits). Required for /schedule/preview. Max 45 days after start. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--name` string — Legacy schedule name field. Used when schedule_name is empty. (≤40 chars)
- `--schedule-id` int64 — Schedule ID. Required on update.
- `--schedule-name` string — Schedule display name. Max 40 characters. (≤40 chars)
- `--start` string — Preview window start (Unix seconds, 10 digits). Required for /schedule/preview. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-id` int64 — Owning team ID.
- body-only (`--data`): layers (array<object>); notify (object)

### self
List my schedules
- `--end` string — Window end (Unix seconds, 10 digits). Must be within 30 days of start. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--start` string — Window start (Unix seconds, 10 digits). Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.

### update
Update schedule
- `--description` string — Schedule description. Max 500 characters. (≤500 chars)
- `--end` string — Preview window end (Unix seconds, 10 digits). Required for /schedule/preview. Max 45 days after start. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--name` string — Legacy schedule name field. Used when schedule_name is empty. (≤40 chars)
- `--schedule-id` int64 — Schedule ID. Required on update.
- `--schedule-name` string — Schedule display name. Max 40 characters. (≤40 chars)
- `--start` string — Preview window start (Unix seconds, 10 digits). Required for /schedule/preview. Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--team-id` int64 — Owning team ID.
- body-only (`--data`): layers (array<object>); notify (object)

<!-- GENERATED:schedule END -->

## Key concepts

- **Window for shifts:** `info` (and `list` when `--start`+`--end` are both set) computes actual rotation slots in the requested window. Max span = 45 days. `info` requires both `--start` and `--end`.
- **`self` window:** returns schedules the current user is assigned to in the given window. Max span = 30 days.
- **Layer `mode`:** `0` = common rotation, `1` = override layer (higher `weight` wins).
- **`rotation_unit`:** `hour | day | week | month`.
- **`restrict_mode`:** `0` = none, `1` = restrict by day, `2` = restrict by week.
- **`expire_time: 0`** means the layer never expires (open-ended).

## Gotchas

- **`info`, `infos`, `delete` take positional `<schedule-id>` — NOT `--schedule-id`.** Pass the ID bare: `fduty schedule info 123 --start now --end +7d`. Using `--schedule-id` on these verbs fails.
- **`create` / `update` / `preview` take all inputs as flags** (no positional). `update` requires `--schedule-id` as a flag to identify the target.
- **`layers` is body-only.** There is no per-layer typed flag — you must pass the entire `layers` array via `--data`. Scalar top-level flags (`--schedule-name`, `--team-id`) override matching `--data` keys.
- **`list` without `--start`/`--end` omits computed shifts** — only schedule metadata is returned. Pass both flags (≤45 day span) to get rotation slots in the list response.
- **`delete` is irreversible** — takes one or more `<schedule-id>` positionals; double-check IDs before executing.
- **`list` default page size is 10** — pass `--limit 100` when scanning all schedules.

## Worked example

```bash
# Find my own on-call windows for the next two weeks
fduty schedule self --start now --end +14d --output-format toon
```
