# fduty monit — store rulesets

Prereq: `SKILL.md` + `reference/monit.md` read. Store rulesets are reusable rule bundles, managed independently of the rule folders in `reference/monit-rule.md`.

## Route here when

"规则集 / ruleset / 规则模板库" or "store ruleset" → this card.

**Mutating:** `store-ruleset-create`, `store-ruleset-update`, `store-ruleset-delete` — confirm before running.

## Intent → verb

| want | verb |
|---|---|
| store ruleset CRUD | `store-ruleset-create/list/info/update/delete` |

<!-- GENERATED:monit[store-ruleset] START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### store-ruleset-create
Create ruleset
- `--note` string (required) — Description or title of the ruleset.
- `--open-flag` int64 — Sharing scope. '0' = private (creator only), '1' = account-shared, '2' = public. Defaults to '0' if omitted.
- `--payload` string (required) — JSON string containing the alert rule definitions.
- `--type-ident` string (required) — Store type identifier this ruleset applies to, e.g. 'redis'.
- response: single object (`data` unwrapped to the top level) — fields: created_at (string); creator_account_id (integer); creator_id (integer); creator_name (string); id (integer); note (string); open_flag (integer); payload (string); type_ident (string); updated_at (string)

### store-ruleset-delete
Delete ruleset
- `--id` int64 (required) — Numeric ID of the target resource; the exact meaning depends on the API being called (e.g. datasource ID, ruleset ID).

### store-ruleset-info
Get ruleset detail
- `--id` int64 (required) — Numeric ID of the target resource; the exact meaning depends on the API being called (e.g. datasource ID, ruleset ID).
- response: same shape as `store-ruleset-create` above

### store-ruleset-list
List rulesets
- `--type-ident` string (required) — Store type identifier to filter by, e.g. 'redis'.
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: created_at (string); creator_account_id (integer); creator_id (integer); creator_name (string); id (integer); note (string); open_flag (integer); payload (string); type_ident (string); updated_at (string)

### store-ruleset-update
Update ruleset
- `--id` int64 (required) — Ruleset ID to update.
- `--note` string (required) — New description.
- `--open-flag` int64 — New sharing scope. '0' = private, '1' = account-shared, '2' = public.
- `--payload` string (required) — New JSON string of alert rule definitions.
- response: same shape as `store-ruleset-create` above

<!-- GENERATED:monit[store-ruleset] END -->
