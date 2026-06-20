# fduty enrichment — command card

Prereq: `SKILL.md` read. Read verbs are free. **`upsert` fully replaces all rules for an integration** (atomic, irreversible in the sense that the previous ruleset is gone); `mapping-schema-delete`, `mapping-api-delete`, and `mapping-data-truncate` are irreversible — confirm IDs before running.

## Route here when

"告警丰富 / 字段提取 / 标签映射 / 标签组合 / 标签删除 / 映射表 / 映射 API / enrichment rules / label extraction / label composition / mapping schema / lookup table / alert enrichment" → **enrichment**, NOT `route` (routing = which channel an alert goes to) or `template` (notification rendering). You need two kinds of IDs:
- **`integration-id`** (int64) — from `fduty channel list` or the Flashduty console integration page.
- **`schema-id`** / **`api-id`** (MongoDB ObjectID hex string) — from `mapping-schema-list` / `mapping-api-list`.

## Intent → verb

| want | verb |
|---|---|
| view current enrichment rules for an integration | `info` |
| view rules for multiple integrations at once | `list` |
| create or fully replace enrichment rules | `upsert` |
| see all mapping schemas | `mapping-schema-list` |
| create a mapping schema (define lookup keys + output labels) | `mapping-schema-create` |
| get one schema's detail | `mapping-schema-info` |
| rename / redescribe a schema | `mapping-schema-update` |
| delete a schema | `mapping-schema-delete` |
| browse rows in a schema | `mapping-data-list` |
| add / update rows (up to 1000 at a time) | `mapping-data-upsert` |
| bulk-load rows from a CSV file | `mapping-data-upload` |
| export schema data to CSV | `mapping-data-download` |
| delete specific rows by key | `mapping-data-delete` |
| wipe all rows in a schema | `mapping-data-truncate` |
| see all external HTTP lookup APIs | `mapping-api-list` |
| register an HTTP lookup endpoint | `mapping-api-create` |
| get one API's detail | `mapping-api-info` |
| update an API's URL / name / timeout | `mapping-api-update` |
| remove an HTTP lookup API | `mapping-api-delete` |

## Hot flow — create a mapping schema and populate it

```bash
# 1. Create schema: define which alert labels are the lookup keys and what labels will be added
fduty enrichment mapping-schema-create \
  --schema-name "service-owner-map" \
  --source-labels service \
  --result-labels owner_team,oncall_email \
  --description "Maps service name to owning team and oncall email"
# → returns schema_id (hex string); save it

# 2. Populate rows (up to 1000 per call; docs array via --data)
fduty enrichment mapping-data-upsert <schema-id> \
  --data '{"docs":[{"service":"payments","owner_team":"platform","oncall_email":"platform@example.com"},{"service":"auth","owner_team":"identity","oncall_email":"identity@example.com"}]}'

# 3. Verify rows landed
fduty enrichment mapping-data-list <schema-id> --output-format toon
```

## Hot flow — attach enrichment rules to an integration

```bash
# 1. Find the integration ID from the channel list (or console)
fduty channel list --output-format toon

# 2. Check existing rules before replacing
fduty enrichment info <integration-id> --output-format toon

# 3. Upsert rules (full replacement; rules array via --data)
fduty enrichment upsert <integration-id> \
  --data '{"rules":[{"kind":"mapping","settings":{"schema_id":"<schema-id>","source_labels":["service"],"result_labels":["owner_team","oncall_email"]}},{"kind":"composition","settings":{"target":"summary","template":"[{{.owner_team}}] {{.title}}"}}]}'

# 4. Confirm the new ruleset
fduty enrichment info <integration-id> --output-format toon
```

<!-- GENERATED:enrichment START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### info <integration-id>
Get enrichment rules
- `<integration-id>` (positional, required) int64 — Integration ID to query enrichment rules for. Must be greater than 0. (min 1)

### list <integration-id> [<id2>...]
List enrichment rules
- `<integration-ids>` (positional, required) intSlice — List of integration IDs to query.

### mapping-api-create
Create mapping API
- `--api-name` string (required) — Unique API name (max 199 chars). (≤199 chars)
- `--description` string — Optional description.
- `--insecure-skip-verify` bool — Skip TLS certificate verification. Default 'false'.
- `--retry-count` int64 — Number of retries on failure (0–1). Default 0.
- `--team-id` int64 — Owning team ID.
- `--timeout` int64 — Request timeout in seconds (1–3). Default 2.
- `--url` string (required) — HTTP/HTTPS endpoint URL (max 500 chars). (≤500 chars)
- body-only (`--data`): headers (object)

### mapping-api-delete <api-id>
Delete mapping API
- `<api-id>` (positional, required) string — Mapping API ID (MongoDB ObjectID hex).

### mapping-api-info <api-id>
Get mapping API detail
- `<api-id>` (positional, required) string — Mapping API ID (MongoDB ObjectID hex).

### mapping-api-list
List mapping APIs

### mapping-api-update <api-id>
Update mapping API
- `<api-id>` (positional, required) string — Mapping API ID (MongoDB ObjectID hex).
- `--api-name` string — New API name (max 199 chars). (≤199 chars)
- `--description` string — New description.
- `--insecure-skip-verify` bool — New TLS skip-verify setting.
- `--retry-count` int64 — New retry count.
- `--team-id` int64 — New owning team ID.
- `--timeout` int64 — New timeout in seconds.
- `--url` string — New endpoint URL (max 500 chars). (≤500 chars)
- body-only (`--data`): headers (object)

### mapping-data-delete <schema-id>
Delete mapping data rows
- `--keys` stringSlice (required) — Keys of rows to delete.
- `<schema-id>` (positional, required) string — Mapping schema ID (MongoDB ObjectID hex).

### mapping-data-download <schema-id>
Download mapping data as CSV
- `<schema-id>` (positional, required) string — Mapping schema ID (MongoDB ObjectID hex).

### mapping-data-list <schema-id>
List mapping data
- `--asc` bool — Sort ascending when 'true'.
- `--limit` int64 — Page size (1–100, default 20).
- `--orderby` string — Sort field. · enum: created_at | updated_at
- `--page` int64 — Page number (1-based). Used for offset-based pagination.
- `<schema-id>` (positional, required) string — Mapping schema ID (MongoDB ObjectID hex).
- `--search-after-ctx` string — Opaque cursor token for cursor-based pagination.
- body-only (`--data`): query (object)

### mapping-data-truncate <schema-id>
Truncate mapping data
- `<schema-id>` (positional, required) string — Mapping schema ID (MongoDB ObjectID hex).

### mapping-data-upload
Upload mapping data via CSV
- `--file` string — CSV file to upload.
- `--schema-id` string — Mapping schema ID (query parameter).

### mapping-data-upsert <schema-id>
Upsert mapping data rows
- `<schema-id>` (positional, required) string — Mapping schema ID (MongoDB ObjectID hex).
- body-only (`--data`): docs (array<object>) (required)

### mapping-schema-create
Create mapping schema
- `--description` string — Optional description (max 500 chars). (≤500 chars)
- `--result-labels` stringSlice (required) — Output label names (1–10). Must not overlap with 'source_labels'.
- `--schema-name` string (required) — Unique schema name (max 39 chars). (≤39 chars)
- `--source-labels` stringSlice (required) — Lookup key label names (1–3). Must not overlap with 'result_labels'.
- `--team-id` int64 — Owning team ID. '0' means no team.

### mapping-schema-delete <schema-id>
Delete mapping schema
- `<schema-id>` (positional, required) string — Mapping schema ID (MongoDB ObjectID hex).

### mapping-schema-info <schema-id>
Get mapping schema detail
- `<schema-id>` (positional, required) string — Mapping schema ID (MongoDB ObjectID hex).

### mapping-schema-list
List mapping schemas

### mapping-schema-update <schema-id>
Update mapping schema
- `--description` string — New description (max 500 chars). (≤500 chars)
- `<schema-id>` (positional, required) string — Schema ID (MongoDB ObjectID hex).
- `--schema-name` string — New schema name (max 39 chars). (≤39 chars)
- `--team-id` int64 — New owning team ID. '0' removes the team association.

### upsert <integration-id>
Upsert enrichment rules
- `<integration-id>` (positional, required) int64 — Integration ID to configure enrichment rules for.
- body-only (`--data`): rules (array<object>) (required)

<!-- GENERATED:enrichment END -->

## Rule kinds (load-bearing — wrong `kind` or `settings` shape 400s)

| kind | what it does | key `settings` fields |
|---|---|---|
| `extraction` | extracts a new label via regex or GJson path | `source`, `target`, `method` (`regex`/`gjson`), `pattern` |
| `composition` | builds a label from a Go template over existing labels | `target`, `template` |
| `mapping` | looks up result labels from a schema or API by source label values | `schema_id` OR `api_id`, `source_labels`, `result_labels` |
| `drop` | removes labels matching a list | `labels` |

Each rule may have an optional `if` AND-filter: `[{"key":"env","oper":"IN","vals":["prod"]}]` — rule is skipped when the filter does not match. `oper` must be `IN` or `NOTIN`.

## Gotchas

- **`upsert` replaces the entire ruleset atomically.** There is no "add one rule" verb. Always read with `info` first, then reconstruct the full `rules` array before calling `upsert`. Omitting a rule deletes it silently.
- **`integration-id` is POSITIONAL on `info`, `list`, and `upsert`** — pass it as the first bare argument (e.g. `fduty enrichment upsert 12345 --data '...'`), not as `--integration-id`. Similarly, `schema-id` and `api-id` are POSITIONAL on all verbs where the `use` shows `<schema-id>` or `<api-id>`.
- **`list` accepts multiple integration IDs as positional args** (`use: list <integration-id> [<id2>...]`) — pass them space-separated: `fduty enrichment list 101 102 103`.
- **`mapping-data-upsert` requires `docs` via `--data`** — this array cannot be expressed as flat flags. Each doc must include all `source_labels` AND all `result_labels` fields for the schema, or the row is rejected.
- **`mapping-schema-create` requires Pro plan** — creating a schema on a free account returns a plan-gate error, not a 404.
- **`mapping-data-truncate` wipes all rows immediately** — there is no undo. Use `mapping-data-download` to export a backup CSV first if the data matters.
- **`source-labels` and `result-labels` must not overlap** on `mapping-schema-create`; max 3 source labels, max 10 result labels. Violating either constraint 400s.

## Worked example — inspect and extend enrichment rules

```bash
# Read current rules for integration 42 (positional), then extend them
fduty enrichment info 42 --output-format toon
# → copy the existing rules[] array, append the new rule, then upsert the full set:
fduty enrichment upsert 42 \
  --data '{"rules":[<existing_rules...>,{"kind":"drop","settings":{"labels":["raw_body","_meta"]}}]}'
```
