# fduty schedule — command card

Prereq: `SKILL.md` read. **Read verbs are free. `delete` is irreversible — confirm IDs before executing. `create` / `update` immediately change the live rotation — confirm scope first.**

## Route here when

"值班 / 排班 / 轮班 / 轮值 / 值班表 / 班表 / 谁在值班 / 当前值班 / 下一班 / 排班配置 / on-call / who is on call / schedule / rotation / shift / next on call / view or edit shifts" → **schedule**. This is the single home for everything 值班/on-call. The key ID you need is **`schedule_id` (int)** — get it from `schedule list`.

**Who is on call right now** is computed from a schedule, not stored: `schedule info <id> --start now --end +1h` returns the current shift (and its `person_ids`). The legacy **`oncall who`** aggregates the live on-call across *all* schedules in one call, but the **`oncall` command group is being deprecated and folded into `schedule`** — use `oncall who` only as a convenience for the global snapshot; prefer `schedule info` for the durable path, and do not build new flows on `oncall *`.

## Intent → verb

| want | verb |
|---|---|
| who is on call right now (one schedule) | `info <schedule-id> --start now --end +1h` |
| who is on call right now (all schedules, legacy) | `oncall who` — *deprecated group; prefer per-schedule `info`* |
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
# 1. Find the schedule ID
fduty schedule list --query "SRE" --output-format toon

# 2a. Current on-call for THIS schedule — a tiny now-window yields the live shift
fduty schedule info <schedule-id> --start now --end +1h --output-format toon

# 2b. Or the live on-call across ALL schedules in one call (legacy oncall group,
#     being deprecated into schedule — fine for a quick global snapshot)
fduty oncall who --output-format toon
```

Both return `person_ids` (integers), not names. Resolve names by joining `member list` client-side — its rows live under `.items[]` keyed by `member_id` (+ `member_name`):

```bash
members=$(fduty member list --json)
fduty schedule info <schedule-id> --start now --end +1h --json | jq --argjson m "$members" '
  [.. | .person_ids? // empty | .[]] | unique | map(. as $id | ($m.items[]? | select(.member_id==$id) | .member_name))'
# If the join is fiddly, just report person_ids — do NOT loop refining jq.
```

## Hot flow — inspect a schedule's upcoming shifts

```bash
fduty schedule list --query "SRE" --output-format toon          # find the schedule_id
fduty schedule info <schedule-id> --start now --end +7d --output-format toon   # next 7 days
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
- **Legacy `oncall who`:** `--team` does **not** filter server-side (any value returns the full list — scope by `--query <schedule_name>` instead), and an empty result is authoritative ("no one on call in that window") — report it, don't widen or fabricate a responder. The `oncall` group will be removed; don't depend on it.

## Worked example

```bash
# Find my own on-call windows for the next two weeks
fduty schedule self --start now --end +14d --output-format toon
```
