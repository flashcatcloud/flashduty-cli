# flashduty integration — command card

Prereq: read `SKILL.md`. These commands manage **public/account-level integrations**, including supported alert, change, IM, and webhook types. They do not manage channel-owned dedicated integrations (`c_` references); use the console for those. For channel configuration and subscriptions, read `reference/channel.md`.

## Read before changing configuration

- Start with `type-list` and `list`. Create only a `plugin_type` whose type entry has `supports_api_create: true`.
- List/info return stored `settings` with each saved secret as `******`, and `integration_key` as is. Prefer `--fields` projections for discovery; do not paste the integration key into a report or unrelated tool. An integration key authenticates inbound pushes; it is not the account App Key used by this CLI.
- `editable` describes object permission; writes also require Integrations Manage. `no_editable: true` marks a read-only integration. Do not infer write authorization from read access.
- Before update/delete or key rotation, get per-target confirmation and save `info` to the protected workspace backup described in `SKILL.md`. Confirm references before deletion. Rotating the key invalidates the old key immediately; update the sender configuration as part of the approved change.
- Metadata-only updates preserve omitted fields. `--team-id 0` deliberately clears team ownership. Submitted `settings` replace type-specific configuration; known sensitive fields omitted or set to `******` preserve their stored value. A stored secret cannot be kept while its destination (such as `endpoint`, `url`, or a database host or port) changes; send the real secret with the new destination. An explicit `headers: {}` clears headers.
- `--is-test` on create, or on update with `settings`, returns plugin test results. Inspect `test.success` and `test.message`: HTTP success is not plugin-test success. An absent `test` means no test result was returned, not a successful test.

## Discovery

```bash
flashduty integration type-list --category event.alert --output-format toon
flashduty integration list --fields integration_id,name,plugin_type,team_id,editable,no_editable --output-format toon
flashduty integration info 42 --fields integration_id,name,plugin_type,team_id,editable,no_editable --output-format toon
```

## Approved configuration changes

Use the exact type-specific settings provided for the target integration; do not guess credential keys. `--integration-key` is optional on create; omitting it generates a key. If provided, it must satisfy the account suffix, length, and uniqueness checks.

```bash
# Only after confirmation; inspect the returned integration_id/key and test result.
flashduty integration create --plugin-type standard.alert --name "Production alerts" --is-test --data '{"settings":{}}' --json
# Only after backing up integration 42 and confirming this rename.
flashduty integration update 42 --name "Production alerts (primary)" --json
```

Do not add `--is-test` to a metadata-only update expecting a plugin test; settings must be submitted for an update test. Disable/enable preserve configuration. Delete may fail with `ReferenceExist` when other objects still reference the integration.

<!-- GENERATED:integration START · 由 flashduty __dump-commands 同步 · 勿手改 fence 内 -->

### create
Create integration
- `--description` string — Free-form description, at most 499 characters. (≤499 chars)
- `--integration-key` string — Optional pre-generated key: 35 characters, ending with the account ID modulo 1000 padded to three digits; it must not already be in use. Omit to generate a key. (35-35 chars)
- `--is-test` bool — Run the plugin test during creation; false when omitted. Inspect the returned 'test.success' for the test outcome.
- `--name` string — Integration name, 2–49 characters when nonempty; defaults to the integration type name when omitted or empty. (2-49 chars)
- `--plugin-type` string (required) — Integration type. Must be one listed by 'POST /integration/type/list' with 'supports_api_create: true'.
- `--team-id` int64 — Owning team ID; omit or use '0' for no team. A nonzero ID must belong to the current account. (min 0)
- body-only (`--data`): settings (object)
- response: single object (`data` unwrapped to the top level) — fields: integration_id (integer); integration_key (string); test (object)

### delete <integration-id>
Delete integration
- `<integration-id>` (positional, required) int64 — Integration ID. (min 1)

### disable <integration-id>
Disable integration
- `<integration-id>` (positional, required) int64 — Integration ID. (min 1)

### enable <integration-id>
Enable integration
- `<integration-id>` (positional, required) int64 — Integration ID. (min 1)

### info <integration-id>
Get integration detail
- `<integration-id>` (positional, required) int64 — Integration ID. (min 1)
- response: single object (`data` unwrapped to the top level) — fields: category (string); created_at (string); creator_id (integer); description (string); editable (boolean); integration_id (integer); integration_key (string); last_time (string); name (string); no_editable (boolean); plugin_id (integer); plugin_type (string); plugin_type_name (string); ref_id (string); settings (object); status (string); team_id (integer); test (object); updated_at (string); updated_by (integer)

### key-rotate <integration-id>
Rotate integration key
- `<integration-id>` (positional, required) int64 — Integration ID. (min 1)
- response: single object (`data` unwrapped to the top level) — fields: integration_key (string)

### list
List integrations
- `--asc` bool — Sort ascending when true, descending when false.
- `--category` string — Filter by category. Accepts a comma-separated list.
- `--is-my-team` bool — Limit the result to integrations owned by your teams.
- `--limit` int64 — Page size. Defaults to 100, maximum 100. (1-100)
- `--name` string — Filter by integration name.
- `--orderby` string — Sort field. Defaults to 'created_at'; 'plugin_type' is sorted by the underlying plugin. · enum: created_at | updated_at | name | plugin_type | status
- `--page` int64 — Page number, 1-based. (min 1)
- `--plugin-type` string — Filter by integration type. Accepts a comma-separated list.
- `--ref-ids` stringSlice — Filter by source reference IDs. Each value must start with 'c_' (channel), 'a_' (account) or 'w_'.
- `--search-after-ctx` string
- `--status` string — Filter by status. Accepts a comma-separated list.
- `--team-ids` intSlice — Filter by team IDs. With 'is_my_team', the values narrow that set further.
- `--type` string — Deprecated. Merged into 'plugin_type' when both are set.
- response: `{items: [...], limit, p, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); category (string); created_at (string); creator_id (integer); data_source_id (integer); description (string); editable (boolean); exclusive_data_source_id (integer); integration_id (integer); integration_key (string); last_time (string); name (string); no_editable (boolean); plugin_id (integer); plugin_type (string); plugin_type_name (string); ref_id (string); settings (object); status (string); team_id (integer); updated_at (string); updated_by (integer)

### type-list
List integration types
- `--asc` bool — Sort ascending when 'true' (the default); descending when 'false'.
- `--category` string — Filter by category. Accepts a comma-separated list, for example 'event.alert,event.change'.
- `--limit` int64 — Page size. Defaults to 20, maximum 100. (1-100)
- `--orderby` string — Sort field. When omitted, types are returned in console ranking order. · enum: id | created_at | updated_at | name | type
- `--page` int64 — Page number, 1-based. (min 1)
- `--search-after-ctx` string
- response: `{items: [...], limit, p, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: category (string); plugin_type (string); plugin_type_logo_url (string); plugin_type_name (string); status (string); supports_api_create (boolean)

### update <integration-id>
Update integration
- `--description` string — New description, at most 499 characters. (≤499 chars)
- `<integration-id>` (positional, required) int64 — Integration ID. (min 1)
- `--is-test` bool — Run the plugin test when submitting 'settings'; no test is performed for a metadata-only update.
- `--name` string — New name, 2–49 characters. (2-49 chars)
- `--team-id` int64 — New owning team ID; '0' clears the team assignment. (min 0)
- body-only (`--data`): settings (object)
- response: same shape as `info <integration-id>` above

<!-- GENERATED:integration END -->
