# flashduty monit — command card

Prereq: `SKILL.md` read. Flashmonit is five separate surfaces sharing one command group, so this card is an index: **read the card for the surface you need, not all of them.**

## Route here when

"监控规则 / 告警规则 / 数据源 / PromQL查询 / 日志查询 / 诊断" or "alert rule / datasource / metric query / log pattern / diagnose" → **monit**. NOT `incident` (that domain = the alert graph after rules fire), and **"数据源" here means a system Flashmonit queries** — On-call 集成 / 告警来源 is a different surface (`reference/channel.md`), and the top-level `datasource` group (`flashduty datasource im-war-room-enabled-list`) is On-call IM plumbing, not this one.

## Which card

| surface | intent | card |
|---|---|---|
| Datasources | connect / list / inspect a datasource, structured database/middleware diagnostics, SLS discovery | **`reference/monit-datasource.md`** |
| Alert rules | rule CRUD, folders, counters, audits | **`reference/monit-rule.md`** |
| Probing | ad-hoc query, log-pattern / metric-trend RCA | **`reference/monit-probe.md`** |

Key IDs are shared across all of them: **rule ID (int)** from `rule-list-basic`; **datasource ID (integer)** for tools and **datasource name (string)** for free queries — never guess, always discover via `datasource-list` (see `reference/monit-datasource.md`).

Read verbs are free. Mutating verbs change state — confirm before running; each card flags its own, and marks the irreversible ones.

<!-- GENERATED:monit START · 由 flashduty __dump-commands 同步 · 勿手改 fence 内 -->

### dashboard-create
Create dashboard
- `--dashboard-id` string (required) — Canonical UUIDv7 of the dashboard.
- `--folder-id` int64 (required) — Folder the dashboard is created in. Must be a folder the caller can write. (min 1)
- `--schema-version` string (required) — Wire schema version; only 'dashboard.v1' is accepted. · enum: dashboard.v1
- body-only (`--data`): definition (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: created_at (string); created_by (object); dashboard_id (string); definition (object); folder_breadcrumb (array<string>); folder_id (integer); revision (integer); schema_version (string); updated_at (string); updated_by (object)

### dashboard-delete <dashboard-id>
Delete dashboard
- `<dashboard-id>` (positional, required) string — Canonical UUIDv7 of the dashboard.
- `--expected-revision` int64 (required) — Revision the caller last read. The write fails with 'DashboardRevisionConflict' unless it still matches the stored revision. (min 1)
- response: single object (`data` unwrapped to the top level) — fields: dashboard_id (string); revision (integer)

### dashboard-get <dashboard-id>
Get dashboard detail
- `<dashboard-id>` (positional, required) string — Canonical UUIDv7 of the dashboard.
- response: same shape as `dashboard-create` above

### dashboard-list <folder-id>
List dashboards
- `<folder-id>` (positional, required) int64 — Folder whose dashboards are listed. (min 1)
- `--limit` int64 — Page size, 1–100. Defaults to 20. (1-100)
- `--page` int64 — Page number, 1-based. Defaults to 1. (min 1)
- `--query` string — Optional space-separated search words matched against title and description; at most 128 Unicode code points. (≤128 chars)
- `--search-after-ctx` string
- body-only (`--data`): sort (array<object>)
- response: `{items: [...], total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: dashboard_id (string); description (string); folder_breadcrumb (array<string>); folder_id (integer); revision (integer); title (string); updated_at (string); updated_by (object)

### dashboard-move
Move dashboard
- `--dashboard-id` string (required) — Canonical UUIDv7 of the dashboard.
- `--expected-revision` int64 (required) — Revision the caller last read. The write fails with 'DashboardRevisionConflict' unless it still matches the stored revision. (min 1)
- `--folder-id` int64 (required) — Destination folder ID. (min 1)
- response: single object (`data` unwrapped to the top level) — fields: changed (boolean); resource (object)

### dashboard-outline <dashboard-id>
Get dashboard outline
- `<dashboard-id>` (positional, required) string — Canonical UUIDv7 of the dashboard.
- `--target-id` string — Optional tab, section or panel ID. When set, the response keeps only the branch that contains it, and an unknown ID returns 'TargetNotFound'.
- response: single object (`data` unwrapped to the top level) — fields: dashboard (object); folder (object); tabs (array<object>); variables (array<object>)

### dashboard-panel-preview
Preview draft panel
- `--max-data-points` int64 — Downsampling target, 2–5000 points. (2-5000)
- body-only (`--data`): context (object) (required); panel (object) (required); selections (object) (required); time (object) (required); variables (array<object>) (required)
- response: single object (`data` unwrapped to the top level) — fields: budget (object); display (object); panel_id (string); refs (array<object>); run_state (string); time (object); variables (object)

### dashboard-panel-run
Run dashboard panel
- `--dashboard-id` string (required) — Canonical UUIDv7 of the dashboard.
- `--max-data-points` int64 — Downsampling target, 2–5000 points; omit or send null for the server default of 100. (2-5000)
- `--panel-id` string (required) — Panel to execute.
- `--revision` int64 — Revision guard; when set it must equal the current revision. (min 1)
- body-only (`--data`): time (object) (required); variables (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: budget (object); dashboard_id (string); display (object); panel_id (string); refs (array<object>); revision (integer); run_state (string); time (object); variables (object)

### dashboard-restore <dashboard-id>
Restore dashboard
- `<dashboard-id>` (positional, required) string — Canonical UUIDv7 of the dashboard.
- `--expected-revision` int64 (required) — Revision the caller last read. The write fails with 'DashboardRevisionConflict' unless it still matches the stored revision. (min 1)
- `--folder-id` int64 — Destination folder. Omit to restore to the original folder; when the original folder is no longer writable the call fails with 'RestoreFolderRequired' and you must pass one. (min 1)
- response: same shape as `dashboard-create` above

### dashboard-revisions-get <dashboard-id>
Get dashboard revision
- `<dashboard-id>` (positional, required) string — Canonical UUIDv7 of the dashboard.
- `--revision` int64 (required) — Revision number to fetch. (min 1)
- response: single object (`data` unwrapped to the top level) — fields: actor (object); created_at (string); dashboard_id (string); definition (object); folder_id (integer); message (string); revision (integer); schema_version (string)

### dashboard-revisions-list <dashboard-id>
List dashboard revisions
- `<dashboard-id>` (positional, required) string — Canonical UUIDv7 of the dashboard.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: actor (object); created_at (string); dashboard_id (string); folder_id (integer); message (string); revision (integer)

### dashboard-runtime-queries-resolve <panel-id> [<id2>...]
Resolve panel queries
- `--dashboard-id` string (required) — Canonical UUIDv7 of the dashboard.
- `<panel-ids>` (positional, required) stringSlice — Panels to resolve, 1–100 unique IDs. A panel that does not exist yields an 'error' arm rather than failing the call.
- body-only (`--data`): time (object) (required); variables (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: dashboard_id (string); panels (array<object>); revision (integer); time (object); variables (object)

### dashboard-runtime-variables-preview
Preview draft variables
- body-only (`--data`): context (object) (required); selections (object) (required); time (object) (required); variables (array<object>) (required)
- response: single object (`data` unwrapped to the top level) — fields: selections (object); variables (array<object>)

### dashboard-runtime-variables-resolve <dashboard-id>
Resolve dashboard variables
- `<dashboard-id>` (positional, required) string — Canonical UUIDv7 of the dashboard.
- body-only (`--data`): time (object) (required); variables (object) (required)
- response: single object (`data` unwrapped to the top level) — fields: dashboard_id (string); revision (integer); selections (object); variables (array<object>)

### dashboard-search
Search dashboards
- `--limit` int64 — Page size, 1–100. Defaults to 20. (1-100)
- `--page` int64 — Page number, 1-based. Defaults to 1. (min 1)
- `--query` string (required) — Space-separated search words; at least one word and at most 128 Unicode code points. (1-128 chars)
- `--search-after-ctx` string
- body-only (`--data`): sort (array<object>)
- response: same shape as `dashboard-list <folder-id>` above

### dashboard-trash-list
List trashed dashboards
- `--limit` int64 — Page size, 1–100. Defaults to 20. (1-100)
- `--page` int64 — Page number, 1-based. Defaults to 1. (min 1)
- `--search-after-ctx` string
- body-only (`--data`): sort (array<object>)
- response: `{items: [...], total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: dashboard_id (string); deleted_at (string); deleted_by (object); description (string); folder_breadcrumb (array<string>); folder_id (integer); revision (integer); title (string)

### dashboard-update <dashboard-id>
Update dashboard
- `<dashboard-id>` (positional, required) string — Canonical UUIDv7 of the dashboard.
- `--expected-revision` int64 (required) — Revision the caller last read. The write fails with 'DashboardRevisionConflict' unless it still matches the stored revision. (min 1)
- `--message` string — Optional revision message, at most 1024 Unicode code points. Stored with the revision and never returned by this endpoint. (≤1024 chars)
- `--schema-version` string (required) — Wire schema version; only 'dashboard.v1' is accepted. · enum: dashboard.v1
- body-only (`--data`): definition (object) (required)
- response: same shape as `dashboard-move` above

### folder-list
List monitor folders
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); created_at (string); creator_id (integer); creator_name (string); id (integer); name (string); note (string); parent_id (integer); parent_path (string); team_id (integer); updated_at (string); updater_id (integer); updater_name (string)

### prometheus-api-v1-label-{label_name}-values <label_name>
List Prometheus label values
- `--data-source-id` int64

<!-- GENERATED:monit END -->
