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
```

## Hot flow — merge noisy alerts into an existing incident

```bash
# 1. find active critical alerts in the last 4 hours
fduty alert list --severity Critical --active --since 4h --output-format toon
# 2. merge (IRREVERSIBLE) — alert IDs are POSITIONAL; --incident-id is a flag
fduty alert merge <alert-id1> <alert-id2> --incident-id <incident-id> --comment "Related disk alerts"
```

<!-- GENERATED:alert START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### event-list <alert-id>
List events for an alert
- `<alert-id>` (positional, required) string — Alert ID (ObjectID hex string).

### events <alert_id>
List alert events

### feed <alert-id>
List alert activity feed
- `<alert-id>` (positional, required) string — Alert ID.
- `--asc` bool — Sort ascending.
- `--limit` int64 — Page size, max 100, default 20.
- `--page` int64 — Page number, starting at 1.
- `--search-after-ctx` string
- `--types` stringSlice — Filter by feed types.

### get <alert_id>
Get alert detail

### info <alert-id>
Get alert detail
- `<alert-id>` (positional, required) string — Alert ID (ObjectID hex string).

### list
List alerts
- `--active` bool
- `--channel` string
- `--fields` string
- `--limit` int
- `--muted` bool
- `--page` int
- `--recovered` bool
- `--severity` string
- `--since` string
- `--until` string

### list-by-ids <alert-id> [<id2>...]
List alerts by IDs
- `<alert-ids>` (positional, required) stringSlice — List of alert IDs (ObjectID hex strings).

### merge <alert-id> [<id2>...]
Merge alerts into an incident
- `<alert-ids>` (positional, required) stringSlice — Alert IDs to merge.
- `--comment` string — Optional comment on the merge action.
- `--incident-id` string (required) — Target incident ID.
- `--owner-id` int64 — Optional new owner for the target incident.
- `--title` string — Optional new title for the target incident.

### pipeline-info <integration-id>
Get alert pipeline
- `<integration-id>` (positional, required) int64 — Integration ID.

### pipeline-list <integration-id> [<id2>...]
List alert pipelines
- `<integration-ids>` (positional, required) intSlice — Integration IDs.

### pipeline-upsert <integration-id>
Create or update alert pipeline
- `<integration-id>` (positional, required) int64 — Integration ID to configure.
- body-only (`--data`): rules (array<object>) (required)

### timeline <alert_id>
View alert timeline
- `--limit` int
- `--page` int

<!-- GENERATED:alert END -->

## Alert status values

- **`alert_severity`** / **`alert_status`**: `Critical` · `Warning` · `Info` · `Ok`
- An alert is **active** if no recovery signal has been received; **recovered** once a recovery fires or it is manually resolved.
- `--active` and `--recovered` on `list` are mutually exclusive — passing both errors.

## Pipeline rule kinds

`pipeline-upsert` replaces the whole pipeline; `rules[].kind` values: `title_reset` · `description_reset` · `severity_reset` · `alert_drop` · `alert_inhibit`. The `rules` array has no typed flag — pass it via `--data '{"rules":[...]}'`. The call is idempotent (upsert), so re-running with the same body is safe.

## Gotchas

- **All alert verbs are positional except `list` and the two-ID `merge` flag.** Every verb with `<alert-id>` in its `use` form takes that ID as the first bare argument — do NOT pass `--alert-id`. The single exception: `merge` takes the first alert ID positionally AND requires `--incident-id` as a flag (two different IDs, different roles).
- **`alert get` vs `alert info`, `alert events` vs `alert-event list`:** both pairs exist; prefer `get`/`events` (shorter, no extra flag); `info`/`event-list` accept `--alert-id` as a flag override for scripting.
- **No server-side title filter on `list`.** To search by title, use `--json` and pipe to `jq`: `fduty alert list --json | jq '.[] | select(.title | test("disk";"i"))'`
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
