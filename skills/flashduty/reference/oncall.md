# fduty oncall — command card

Prereq: `SKILL.md` read. All three verbs are **read-only** — no mutations, no irreversible actions. Safe to run without confirmation.

## Route here when

"谁在值班 / 当前值班 / 排班 / 值班轮换 / 下一个值班 / who is on call / schedule / shift / rotation / next responder / escalate to on-call" → **oncall**. You need a `schedule_id` (integer) only for `schedule get`; get it from `schedule list` first. `who` needs no IDs — run it directly.

Do NOT use oncall for MTTA/MTTR or noise analysis → use `insight`. For member phone/email resolution → join `fduty member list` client-side (oncall returns `person_ids` only, no names).

## Intent → verb

| want | verb |
|---|---|
| who is on call right now | `who` |
| who covers a future window | `who --since <start> --until <end>` |
| list all schedule IDs + names | `schedule list` |
| rotation layers + upcoming slots for one schedule | `schedule get <schedule_id>` |

## Hot flow — find the on-call person and escalate

```bash
# 1. See who is on call now across all schedules
fduty oncall who --output-format toon

# 2. Narrow to a specific schedule by name
fduty oncall who --query "Platform" --output-format toon

# 3. Get upcoming shifts for the next 7 days (schedule_id is POSITIONAL)
fduty oncall schedule list --output-format toon   # find schedule_id
fduty oncall schedule get <schedule_id> --since now --until +7d --output-format toon

# 4. Resolve names in ONE pass: capture member list, join person_ids → member_name.
#    who emits person_ids (numbers) under cur_oncall.group.members[].person_ids[];
#    member list rows live under .items[] keyed by member_id (+ member_name).
members=$(fduty member list --json)
fduty oncall who --json | jq --argjson m "$members" '
  map({schedule: .schedule_name,
       on_call: [ .cur_oncall.group.members[]?.person_ids[]? as $id
                  | $m.items[]? | select(.member_id == $id) | .member_name ]})'
# If the join is fiddly, just report schedule_name + person_ids — do NOT loop refining jq;
# the answer to "who is on call" is already in `who`.
```

<!-- GENERATED:oncall START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### get <schedule_id>
Get schedule detail
- `--since` string
- `--until` string

### list
List schedules
- `--limit` int
- `--page` int
- `--query` string
- `--since` string
- `--team` string
- `--until` string

### who
Show who is currently on call
- `--limit` int
- `--page` int
- `--query` string
- `--since` string
- `--team` string
- `--until` string

<!-- GENERATED:oncall END -->

## Key concepts

- **`who` vs `schedule list`**: `who` is optimized for "current person + shift window" — returns `cur_oncall` and `next_oncall` snapshots. `schedule list` returns schedule metadata (id, name, layer count) without per-slot detail.
- **`schedule get` window**: defaults to `now → +7d`. `final_schedule.schedules[]` contains the merged final rotation; `layer_schedules[]` shows each layer separately. For override vs normal layers, check `mode` (0 = common rotation, 1 = override).
- **`--since`/`--until`** accept relative durations (`now`, `+2h`, `+7d`), ISO dates, or Unix seconds — same for both `who` and `schedule` verbs.
- **`person_ids` are integers**, not names. The CLI has no by-id member lookup — dump `fduty member list --json` and join client-side.

## Gotchas

- **`schedule get` takes `<schedule_id>` as a POSITIONAL first argument** (`fduty oncall schedule get 123 --since now`), not `--schedule-id`. Passing it as a flag fails with a missing-positional error.
- **`who` and `schedule list` take all inputs as flags** (no positionals) — both `use` values have no `<…>`.
- **`--team` does not filter server-side** on either `who` or `schedule list` (confirmed against live API): any value — valid or bogus — returns the same full list. Use `--query <schedule_name>` to scope by schedule name, or filter `team_id` from the JSON response.
- **`schedule list` JSON is a top-level array**, not `.items[]` — pipe `jq '.[]'`, not `jq '.items[]'`.
- **Empty result is authoritative** — if `who` returns no rows or `cur_oncall` is null, no one is scheduled in that window. Do not widen the query or fabricate a responder.
- **No mutating verbs exist in this domain.** Schedule creation and editing are web-UI only.

## Worked example

```bash
# Find who is on call for the DB team this weekend, then get their next 3 days of shifts
fduty oncall who --query "DB" --since "2026-06-21T00:00:00" --until "2026-06-22T23:59:59" --output-format toon
fduty oncall schedule get <schedule_id> --since now --until +3d --output-format toon
```
