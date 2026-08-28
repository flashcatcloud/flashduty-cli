# fduty alert — command card

Prereq: `SKILL.md` read. Read verbs are free. `merge` is **irreversible** (alerts cannot be un-merged). `pipeline-upsert` **replaces** the full pipeline config. Confirm IDs before either.

## Route here when

"告警 / 告警事件 / 去重 / 合并到故障 / 告警流水线 / alert noise / dedup / severity filter / alert pipeline" → **alert**, NOT `incident` (incident = the actionable layer above alerts). Key ID: **`alert_id` (ObjectID hex string)** — get it from `alert list` output or from `incident alerts <incident-id>` (incident domain). Pipeline verbs need an **`integration_id` (int)** from the channel/integration domain.

## Intent → verb

| want | verb |
|---|---|
| active / recovered / muted alerts in a time window | `list` |
| full detail of one alert | `get` |
| full detail of one alert (alternate path) | `info` |
| raw events deduplicated into one alert | `events` |
| raw events for one alert (alternate) | `event-list` |
| alert state-transition history | `feed` |
| alert state-transition history (alternate) | `timeline` |
| fetch multiple alerts by ID list | `list-by-ids` |
| reassign alerts to a different incident | `merge` |
| get processing pipeline for an integration | `pipeline-info` |
| get pipelines for multiple integrations | `pipeline-list` |
| create or replace a processing pipeline | `pipeline-upsert` |

## Hot flow — investigate an incident's root alerts

```bash
# 1. list contributing alerts (from the incident domain)
fduty incident alerts <incident-id> --output-format toon
# 2. inspect the worst alert
fduty alert get <alert-id> --output-format toon
# 3. trace raw events deduplicated into that alert
fduty alert events <alert-id> --output-format toon
# 4. view state transitions (mute/severity changes/operator actions)
fduty alert feed <alert-id> --output-format toon
# 5. for a time-window view across alerts, alert-event list is compact by default
fduty alert-event list --channel <channel-id> --since 1h --limit 30 --output-format toon
```

Structured `alert-event list` output stays below 16 KiB: when the requested page would overflow, only the leading rows that fit are emitted — every value intact — and a stderr note says how many of the rows were emitted, so heed it before assuming the page is complete (narrow `--fields` or lower `--limit` to fit more rows per page). A trailing `...` on a value, with a stderr note naming the clipped fields, appears only when one row alone exceeds the budget — heed it before matching on that value, because the clipped text is what a `jq` filter sees. In json/toon mode rows default to the compact projection `event_id,alert_id,event_severity,event_status,event_time,title` (a stderr note says so when it applies); any other response field is one `--fields` away — a key missing from the output means it wasn't selected, not that the server omits it.

## Hot flow — merge noisy alerts into an existing incident

```bash
# 1. find active critical alerts in the last 4 hours
fduty alert list --severity Critical --active --since 4h --output-format toon
# 2. merge (IRREVERSIBLE) — alert IDs are POSITIONAL; --incident-id is a flag
# comment text comes from a file, never an inline shell argument (see incident.md's comment workflow)
printf '%s' 'Related disk alerts' > /tmp/merge-comment.txt
fduty alert merge <alert-id1> <alert-id2> --incident-id <incident-id> --comment-file /tmp/merge-comment.txt
```

<!-- GENERATED:alert START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### event-list <alert-id>
List events for an alert
- `<alert-id>` (positional, required) string — Alert ID (MongoDB ObjectID).
- `--asc` bool — When true, return events oldest-first. Defaults to newest-first.
- `--limit` int64 — Page size. Defaults to 20 and cannot exceed 100. (0-100)
- `--page` int64 — Page number starting at 1. Used when 'search_after_ctx' is omitted. (min 0)
- `--search-after-ctx` string — Cursor returned by the previous page. When supplied, cursor pagination is used instead of page-number pagination.
- response: `{items: [...], has_next_page, search_after_ctx, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); alert_id (string); alert_key (string); channel_id (integer); created_at (string); data_source_id (integer); deleted_at (string); description (string); event_id (string); event_severity (string); event_status (string); event_time (string); images (array<object>); integration_id (integer); integration_type (string); labels (object); title (string); title_rule (string); updated_at (string)

### events <alert_id>
List alert events
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); alert_id (string); alert_key (string); channel_id (integer); created_at (string); data_source_id (integer); deleted_at (string); description (string); event_id (string); event_severity (string); event_status (string); event_time (string); images (array<object>); integration_id (integer); integration_type (string); labels (object); title (string); title_rule (string); updated_at (string)

### feed <alert-id>
List alert activity feed
- `<alert-id>` (positional, required) string — Alert ID (ObjectID hex string); obtain it from 'POST /alert/list'.
- `--asc` bool — Sort ascending.
- `--limit` int64 — Page size, max 100, default 20. (1-100)
- `--page` int64 — Page number, starting at 1. (min 1)
- `--search-after-ctx` string
- `--types` stringSlice — Filter by feed type codes — see the 'type' field of the response items for the full list (e.g. 'a_new', 'a_comm', 'a_merge').
- response: `{items: [...], has_next_page}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); created_at (string); creator_id (integer); deleted_at (string); detail (object); ref_id (string); type (string); updated_at (string)

### get <alert_id>
Get alert detail
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); alert_id (string); alert_key (string); alert_severity (string); alert_status (string); channel_id (integer); channel_name (string); channel_status (string); created_at (string); data_source_id (integer); data_source_name (string); data_source_ref_id (string); data_source_type (string); deleted_at (string); description (string); end_time (string); event_cnt (integer); events (array<object>); ever_muted (boolean); images (array<object>); incident (object); integration_id (integer); integration_name (string); integration_ref_id (string); integration_type (string); labels (object); last_time (string); responder_email (string); responder_name (string); start_time (string); title (string); title_rule (string); updated_at (string)

### info <alert-id>
Get alert detail
- `<alert-id>` (positional, required) string — Alert ID (ObjectID hex string).
- response: same shape as `get <alert_id>` above

### list
List alerts
- `--active` bool
- `--channel` string
- `--fields` string
- `--integration` string
- `--limit` int
- `--muted` bool
- `--page` int
- `--recovered` bool
- `--severity` string
- `--since` string
- `--until` string
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); alert_id (string); alert_key (string); alert_severity (string); alert_status (string); channel_id (integer); channel_name (string); channel_status (string); created_at (string); data_source_id (integer); data_source_name (string); data_source_ref_id (string); data_source_type (string); deleted_at (string); description (string); end_time (string); event_cnt (integer); events (array<object>); ever_muted (boolean); images (array<object>); incident (object); integration_id (integer); integration_name (string); integration_ref_id (string); integration_type (string); labels (object); last_time (string); responder_email (string); responder_name (string); start_time (string); title (string); title_rule (string); updated_at (string)

### list-by-ids <alert-id> [<id2>...]
List alerts by IDs
- `<alert-ids>` (positional, required) stringSlice — Alert IDs (ObjectID hex strings) to fetch.
- response: `{items: [...], has_next_page, search_after_ctx, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); alert_id (string); alert_key (string); alert_severity (string); alert_status (string); channel_id (integer); channel_name (string); channel_status (string); created_at (string); data_source_id (integer); data_source_name (string); data_source_ref_id (string); data_source_type (string); deleted_at (string); description (string); end_time (string); event_cnt (integer); events (array<object>); ever_muted (boolean); images (array<object>); incident (object); integration_id (integer); integration_name (string); integration_ref_id (string); integration_type (string); labels (object); last_time (string); responder_email (string); responder_name (string); start_time (string); title (string); title_rule (string); updated_at (string)

### merge <alert-id> [<id2>...]
Merge alerts into an incident
- `<alert-ids>` (positional, required) stringSlice — Alert IDs to merge.
- `--comment-file` string — Path to a file containing an optional comment on the merge action (- reads stdin).
- `--incident-id` string (required) — Target incident ID.
- `--owner-id` int64 — Optional new owner for the target incident.
- `--title` string — Optional new title for the target incident.

### pipeline-info <integration-id>
Get alert pipeline
- `<integration-id>` (positional, required) int64 — Integration ID. Must be greater than 0.
- response: single object (`data` unwrapped to the top level) — fields: created_at (string); creator_id (integer); deleted_at (string); integration_id (integer); rules (array<object>); status (string); updated_at (string); updated_by (integer)

### pipeline-list <integration-id> [<id2>...]
List alert pipelines
- `<integration-ids>` (positional, required) intSlice — Integration IDs. At least one entry is required.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: created_at (string); creator_id (integer); deleted_at (string); integration_id (integer); rules (array<object>); status (string); updated_at (string); updated_by (integer)

### pipeline-upsert <integration-id>
Create or update alert pipeline
- `<integration-id>` (positional, required) int64 — Integration ID to configure.
- body-only (`--data`): rules (array<object>) (required)

### timeline <alert_id>
View alert timeline
- `--limit` int
- `--page` int
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); created_at (string); creator_id (integer); deleted_at (string); detail (object); ref_id (string); type (string); updated_at (string)

<!-- GENERATED:alert END -->

## Alert status values

- **`alert_severity`** / **`alert_status`**: `Critical` · `Warning` · `Info` · `Ok`
- An alert is **active** if no recovery signal has been received; **recovered** once a recovery fires or it is manually resolved.
- `--active` and `--recovered` on `list` are mutually exclusive — passing both errors.

## Pipeline rule kinds

`pipeline-upsert` replaces the whole pipeline (max 50 rules); `rules[].kind` values: `title_reset` · `description_reset` · `severity_reset` · `alert_drop` · `alert_inhibit`. The `rules` array has no typed flag — pass it via `--data '{"rules":[...]}'`. The call is idempotent (upsert), so re-running with the same body is safe.

`settings` shape depends on `kind`: `title_reset` → `{"title": "<template>"}`; `description_reset` → `{"description": "<template>"}`; `severity_reset` → `{"severity": "Critical"|"Warning"|"Info"}`; `alert_drop` → `{}` (empty object); `alert_inhibit` → `{"equals": ["<label_key>", ...], "source_filters": [<condition>, ...]}`.

**`rules[].if` and `alert_inhibit.source_filters` are FLAT AND-lists** — one array of `{key, oper, vals}` conditions, ALL of which must match:

```json
"if": [{"key": "labels.env", "oper": "IN", "vals": ["prod"]}]
```

This is the same shape as `enrichment`'s rule-level `if`, and it is **not** the
OR-of-AND tree used by silence / inhibit / drop / escalation rules. Wrapping the
conditions in a second array is rejected before the request leaves the CLI
(`cannot unmarshal array into ... of type FilterCondition`). Read
`reference/filters.md` for the operators, the missing-key trap, and the key
vocabulary — those all apply here; only the nesting differs.

## Writing `title_reset` / `description_reset` templates

The `settings.title` and `settings.description` values are NOT plain strings —
they are rendered, and the two kinds render differently:

| | with a leading `[TPL]` | without it |
|---|---|---|
| `title_reset` | rendered as a template | treated as a `::`-joined key list (below) |
| `description_reset` | rendered as a template | **silently does nothing** — the rule is a no-op |

**Always write `[TPL]` for `description_reset`.** Omitting it is not an error;
the rule simply never fires, which reads as "the pipeline didn't apply".

Inside a `[TPL]` value, two substitutions run in order:

1. `${label_name}` — replaced with that label's value; a missing label renders
   `<no value>` rather than failing.
2. Go `text/template` — the event is the dot, so labels are
   `{{.Labels.<name>}}` (`Labels` capitalised; it is a map).

```json
{"kind": "title_reset", "settings": {"title": "[TPL]{{.Labels.service}} / {{.Labels.check}}"}}
{"kind": "description_reset", "settings": {"description": "[TPL]${instance} is late by ${seconds}s"}}
```

**The bare `::` form (title only).** Without `[TPL]`, the title is split on
`::`; each segment is either a literal or `$name`, which resolves to
`labels.<name>`. Segments are joined with position-fixed separators — nothing,
then ` / `, then ` - `, then ` ⋅ ` for the rest — so `$service::$check` renders
as `payments-api / disk_used`. Prefer the `[TPL]` form: it is explicit about
where values come from and does not hard-code separators.

**Labels the template reads must already exist.** Enrichment runs BEFORE the
pipeline, so labels produced by `fduty enrichment upsert` (extraction,
composition, mapping) are available here — but a label produced by a LATER
pipeline rule is not, and a typo just renders `<no value>` into the title.

## Gotchas

- **All alert verbs are positional except `list` and the two-ID `merge` flag.** Every verb with `<alert-id>` in its `use` form takes that ID as the first bare argument — do NOT pass `--alert-id`. The single exception: `merge` takes the first alert ID positionally AND requires `--incident-id` as a flag (two different IDs, different roles).
- **`alert get` vs `alert info`, `alert events` vs `alert-event list`:** both pairs exist; prefer `get`/`events` (shorter, no extra flag); `info`/`event-list` accept `--alert-id` as a flag override for scripting.
- **No server-side title filter on `list`.** To search by title, use `--json` and pipe to `jq`: `fduty alert list --json | jq '.[] | select(.title | test("disk";"i"))'`
- **`list`'s structured output has no `total`/page metadata** — its `--json`/`toon` response is a bare TOP-LEVEL array (see the `list` fence entry above), not a `{items, total}` wrapper. To count matches, project the narrowest field with `--fields` and count elements, or use a wrapper-style verb whose fence shows `total` (e.g. `list-by-ids`). Don't paginate page 1/2/3... just to count alerts — narrow the query instead (`--active`, `--recovered`, `--severity`, `--channel`, `--since`).
- **Use `--fields` when hunting IDs, not full rows.** If the task is "find alert IDs / titles / channels / severities", project only those fields first, then drill into one alert with `get` / `events`. Dumping every field for 100 alerts wastes tokens and hides the one row you need.
- **`list` time window cap is 31 days**; `--limit` max is 100. For broader queries use `insight` domain.
- **`pipeline-upsert` fully replaces** the existing pipeline — always fetch current config with `pipeline-info` first and include unchanged rules in the new body.
- **Empty `list` result is authoritative** — report "no alerts match" and stop; do not widen filters or retry with alternate keywords.

## Worked example

```bash
# Find active Critical alerts in a specific channel and view the noisiest one
fduty alert list --severity Critical --active --channel 98765 --since 2h --output-format toon
fduty alert get <alert-id> --output-format toon
fduty alert events <alert-id> --output-format toon
```
