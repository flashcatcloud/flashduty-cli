# fduty calendar — command card

Prereq: `SKILL.md` read. **`delete` is irreversible** (calendar + all its events gone). `event-delete` is irreversible per event. Reads are free; confirm IDs before any delete.

## Route here when

"服务日历 / 工作日 / 非工作日 / 节假日 / 补班 / 值班日历" → **calendar**, NOT `oncall` (oncall = who is on call in a schedule; calendar = custom working/non-working day definitions used by oncall rules).

You need a **`cal_id`** (string, format `cal.<uuid>`). Get it from `calendar list`. For events you also need **`event_id`** (returned by `event-upsert`; list with `event-list`).

Public holiday calendars (e.g. `zh-cn.china.official`) are read-only — list them with `--kind region.official.holiday`, then inherit them into a personal calendar via `--extra-cal-ids`.

## Intent → verb

| want | verb |
|---|---|
| list personal calendars | `list` |
| browse public-holiday calendars | `list --kind region.official.holiday` |
| calendar details (config) | `info <cal-id>` |
| create a new personal calendar | `create` |
| rename / change workdays / inherit holidays | `update <cal-id>` |
| delete a calendar (irreversible) | `delete <cal-id>` |
| list events in a calendar | `event-list <cal-id>` |
| add or edit a working/non-working day override | `event-upsert <cal-id>` |
| remove a calendar event (irreversible) | `event-delete` |

## Hot flow — create a calendar with holiday inheritance

```bash
# 1. Find available public-holiday cal IDs for your locale
fduty calendar list --kind region.official.holiday --output-format toon

# 2. Create a personal calendar that inherits CN public holidays, Mon–Fri workdays
fduty calendar create --cal-name "Ops Workdays" \
  --timezone Asia/Shanghai \
  --workdays 1,2,3,4,5 \
  --extra-cal-ids zh-cn.china.official
# → returns cal_id; save it

# 3. Mark a make-up workday (補班) on a Saturday
fduty calendar event-upsert <cal-id> --summary "補班 (New Year)" \
  --start-at 2026-01-17 --end-at 2026-01-18 --is-off false

# 4. Mark a custom holiday (non-working)
fduty calendar event-upsert <cal-id> --summary "Team offsite" \
  --start-at 2026-03-20 --end-at 2026-03-22 --is-off true
```

## Hot flow — audit & clean up events for a month

```bash
# List all events in January 2026
fduty calendar event-list <cal-id> --year 2026 --month 1 --output-format toon

# Delete a specific event (get event_id from the list above)
fduty calendar event-delete --cal-id <cal-id> --event-id <event-id>
```

<!-- GENERATED:calendar START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### create
Create calendar
- `--cal-name` string (required) — Calendar display name. (1-39 chars)
- `--description` string — Calendar description. (≤499 chars)
- `--extra-cal-ids` stringSlice — Additional public-holiday calendar IDs to inherit events from (for example zh-cn.china.official).
- `--team-id` int64 — Owning team ID. 0 means no team.
- `--timezone` string — IANA timezone. Defaults to Asia/Shanghai when empty.
- `--workdays` intSlice — Workday numbers (0 = Sunday, 6 = Saturday).

### delete <cal-id>
Delete calendar
- `<cal-id>` (positional, required) string — Calendar ID.

### event-delete
Delete calendar event
- `--cal-id` string (required) — Calendar ID.
- `--event-id` string (required) — Event ID.

### event-list <cal-id>
List calendar events
- `<cal-id>` (positional, required) string — Calendar ID.
- `--day` int64 — Day (1-31). 0 means no day filter. (0-31)
- `--month` int64 — Month (1-12). 0 means no month filter. (0-12)
- `--year` int64 — Year. Defaults to the current year when omitted. (min 2023)

### event-upsert <cal-id>
Upsert calendar event
- `<cal-id>` (positional, required) string — Calendar ID.
- `--description` string — Event description. (≤499 chars)
- `--end-at` string (required) — Event end date in YYYY-MM-DD (exclusive).
- `--event-id` string — Event ID. Omit when creating. (≤63 chars)
- `--is-off` bool (required) — Whether the event marks a non-working day. true = day off, false = working day override.
- `--start-at` string (required) — Event start date in YYYY-MM-DD.
- `--summary` string (required) — Event summary. (1-39 chars)

### info <cal-id>
Get calendar info
- `<cal-id>` (positional, required) string — Calendar ID.

### list
List calendars
- `--kind` string — Calendar kind filter. Defaults to personal when empty. · enum: region.official.holiday | personal
- `--no-locale` bool — Disable locale filtering when listing public-holiday calendars.

### update <cal-id>
Update calendar
- `<cal-id>` (positional, required) string — Calendar ID.
- `--cal-name` string — New calendar name. (1-39 chars)
- `--description` string — New description. (≤499 chars)
- `--extra-cal-ids` stringSlice — Additional public-holiday calendar IDs to inherit events from.
- `--team-id` int64 — New owning team ID.
- `--timezone` string — New IANA timezone.
- `--workdays` intSlice — Workday numbers (0 = Sunday, 6 = Saturday).

<!-- GENERATED:calendar END -->

## Key concepts

- **`is-off` (bool, required on event-upsert):** `true` = mark as non-working day (holiday/closure); `false` = override to working day (make-up workday / 補班). This is the only enum-like field — it must be explicit; the server rejects a missing value.
- **`end-at` is exclusive:** a single-day event on 2026-01-17 needs `--start-at 2026-01-17 --end-at 2026-01-18`.
- **`workdays` integers:** 0 = Sunday, 1 = Monday … 6 = Saturday. Standard Mon–Fri = `1,2,3,4,5`.
- **Calendar kinds:** `personal` (editable, default filter) vs `region.official.holiday` (read-only, browsable). The returned `kind` field can also be `religion.holiday`.
- **Account cap:** max 5 personal calendars per account by default.

## Gotchas

- **`cal-id` is POSITIONAL on `info`, `update`, `delete`, `event-list`, `event-upsert`** — pass it as the first bare argument: `fduty calendar info <cal-id>`. On `event-delete` both `--cal-id` and `--event-id` are flags (no positional — `use` is bare `event-delete`).
- **`event-upsert` creates OR updates** — omit `--event-id` to create; supply it to edit an existing event. The returned `event_id` is what to save for future edits or deletes.
- **`list` defaults to `--kind personal`** — you will NOT see public-holiday calendars unless you pass `--kind region.official.holiday`. Add `--no-locale` to see all locales, not just yours.
- **`delete` removes the calendar and ALL its events** — confirm `cal_id` with `list` first; irreversible.
- **`extra-cal-ids` on update is a full replacement list** — pass ALL desired public-holiday IDs, not just the new one; omitting an ID removes it.

## Worked example

Mark the Spring Festival week (2026) as non-working in a personal calendar:

```bash
fduty calendar event-upsert cal.abc123 \
  --summary "Spring Festival" \
  --start-at 2026-01-28 --end-at 2026-02-04 \
  --is-off true \
  --description "Golden Week — office closed."
# Returns event_id for future edits.
```
