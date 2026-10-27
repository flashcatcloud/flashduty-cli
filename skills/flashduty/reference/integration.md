# flashduty integration — command card

Prereq: read `SKILL.md`. Choose the command family from the integration's ownership before reading or changing it. For channel configuration, subscriptions, and routing, read `reference/channel.md`.

## Choose the integration scope

| User intent or existing ownership | Command family | Permission boundary |
|---|---|---|
| Public integration managed in the integration center; shared ingestion connected to channels through subscriptions or routing | `integration *` | Integrations Read/Manage plus the integration's object permissions |
| Dedicated alert integration owned by one collaboration space (协作空间专属集成) | `channel integration-*` | Reads: Channels Read/Manage or Integrations Read plus channel visibility. Writes: Channels Manage plus channel management access |

For an existing integration, use its known ownership; an ID alone does not identify the scope. A target channel receiving alerts does not by itself mean the integration is dedicated: public integrations can also route there. For creation, use a dedicated integration when ingestion belongs to that one channel, or a public integration when the user wants shared ingestion managed in the integration center. If ownership or intended scope is unclear, ask before creating or changing configuration.

Never retry a failed call through the other integration family, including on access denied or not found. Public does not mean anonymous: both families use the management App Key and its owner's permissions. An integration key is a separate credential for inbound events.

`integration type-list` is a shared catalog for both families. It accepts Channels Read, Channels Manage, or Integrations Read. Scoped App Keys must allow `integration.api:read:typeList`; a `channel.integration:*:*` scope does not include this endpoint. Catalog access does not grant access to public integration instances.

## Public integrations

### Read before changing configuration

- Start with `type-list` and `list`. Create only a `plugin_type` whose type entry has `supports_api_create: true`.
- List/info return stored `settings` with each saved secret as `******`, and `integration_key` as is. Prefer `--fields` projections for discovery; do not paste the integration key into a report or unrelated tool. An integration key authenticates inbound pushes; it is not the account App Key used by this CLI.
- `editable` describes object permission; writes also require Integrations Manage. `no_editable: true` marks a read-only integration. Do not infer write authorization from read access.
- Before update/delete or key rotation, get per-target confirmation and save `info` to the protected workspace backup described in `SKILL.md`. Confirm references before deletion. Rotating the key invalidates the old key immediately; update the sender configuration as part of the approved change.
- Metadata-only updates preserve omitted fields. `--team-id 0` deliberately clears team ownership. Submitted `settings` replace type-specific configuration; known sensitive fields omitted or set to `******` preserve their stored value. A stored secret cannot be kept while its destination (such as `endpoint`, `url`, or a database host or port) changes; send the real secret with the new destination. An explicit `headers: {}` clears headers.
- `--is-test` on create, or on update with `settings`, returns plugin test results. Inspect `test.success` and `test.message`: HTTP success is not plugin-test success. An absent `test` means no test result was returned, not a successful test.

### Discovery

```bash
flashduty integration type-list --category event.alert --output-format toon
flashduty integration list --fields integration_id,name,plugin_type,team_id,editable,no_editable --output-format toon
flashduty integration info 42 --fields integration_id,name,plugin_type,team_id,editable,no_editable --output-format toon
```

### Approved configuration changes

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
Create public integration
- `--description` string — Free-form description, at most 499 characters. (≤499 chars)
- `--integration-key` string — Optional pre-generated key: 35 characters, ending with the account ID modulo 1000 padded to three digits; it must not already be in use. Omit to generate a key. (35-35 chars)
- `--is-test` bool — Run the plugin test during creation; false when omitted. Inspect the returned 'test.success' for the test outcome.
- `--name` string — Integration name, 2–49 characters when nonempty; defaults to the integration type name when omitted or empty. (2-49 chars)
- `--plugin-type` string (required) — Integration type. Must be one listed by 'POST /integration/type/list' with 'supports_api_create: true'.
- `--team-id` int64 — Owning team ID; omit or use '0' for no team. A nonzero ID must belong to the current account. (min 0)
- body-only (`--data`): settings (object)
- response: single object (`data` unwrapped to the top level) — fields: integration_id (integer); integration_key (string); test (object)

### delete <integration-id>
Delete public integration
- `<integration-id>` (positional, required) int64 — Integration ID. (min 1)

### disable <integration-id>
Disable public integration
- `<integration-id>` (positional, required) int64 — Integration ID. (min 1)

### enable <integration-id>
Enable public integration
- `<integration-id>` (positional, required) int64 — Integration ID. (min 1)

### info <integration-id>
Get public integration detail
- `<integration-id>` (positional, required) int64 — Integration ID. (min 1)
- response: single object (`data` unwrapped to the top level) — fields: category (string); created_at (string); creator_id (integer); description (string); editable (boolean); integration_id (integer); integration_key (string); last_time (string); name (string); no_editable (boolean); plugin_id (integer); plugin_type (string); plugin_type_name (string); ref_id (string); settings (object); status (string); team_id (integer); test (object); updated_at (string); updated_by (integer)

### key-rotate <integration-id>
Rotate public integration key
- `<integration-id>` (positional, required) int64 — Integration ID. (min 1)
- response: single object (`data` unwrapped to the top level) — fields: integration_key (string)

### list
List public integrations
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
Update public integration
- `--description` string — New description, at most 499 characters. (≤499 chars)
- `<integration-id>` (positional, required) int64 — Integration ID. (min 1)
- `--is-test` bool — Run the plugin test when submitting 'settings'; no test is performed for a metadata-only update.
- `--name` string — New name, 2–49 characters. (2-49 chars)
- `--team-id` int64 — New owning team ID; '0' clears the team assignment. (min 0)
- body-only (`--data`): settings (object)
- response: same shape as `info <integration-id>` above

<!-- GENERATED:integration END -->


## Dedicated integrations

Use the channel's `channel_id` to list or create dedicated integrations. Info, update, lifecycle, and key rotation use the returned `integration_id`; public integration IDs are not accepted. Creation requires an enabled channel and an enabled `event.alert` type allowed by the account's license. IM, change-event, and outbound webhook integrations use the public family. An alert-ingestion webhook type such as `webhook.alert` is still an `event.alert` type and can be dedicated.

- Reads require Channels Read or Manage and visibility of the channel. Private channels remain restricted.
- Writes require Channels Manage and management of the owning channel. Creating an integration does not grant its creator channel management rights.
- List/info return the complete `integration_key` and masked settings. Use field projections for discovery and keep keys out of reports. App Key scopes may further restrict calls.
- Before update/delete, enable/disable, or key rotation, follow `SKILL.md`'s confirmation and protected-backup rules for the specific target. Before create, confirm the target channel and intended scope. For key rotation, confirm sender reconfiguration because the old key stops working immediately. Remove references before deletion.
- Dedicated updates accept name, description, and settings. Omitted or null fields retain their value; `--description ""` clears the description. `settings: {}` is an explicit replacement, subject to type validation. Known secrets omitted or set to `******` are preserved; changing their destination requires the real secret. Dedicated commands have no `--team-id` or `--is-test` option and cannot move an integration to another channel.

```bash
# Discover only within the requested collaboration space.
flashduty channel integration-list --channel-id 20001 --fields integration_id,channel_id,name,plugin_type,status --output-format toon
flashduty channel integration-info 10001 --fields integration_id,channel_id,name,plugin_type,status --output-format toon
# Only after confirmation of this channel, scope, and type-specific settings.
flashduty channel integration-create --channel-id 20001 --plugin-type standard.alert --name "Primary alerts" --data '{"settings":{}}' --json
# Only after backing up integration 10001 and confirming this rename.
flashduty channel integration-update 10001 --name "Primary alerts (renamed)" --json
```

<!-- GENERATED:channel[integration] START · 由 flashduty __dump-commands 同步 · 勿手改 fence 内 -->

### integration-create <channel-id>
Create dedicated integration
- `<channel-id>` (positional, required) int64 — Target channel ID; the channel must be enabled. (min 1)
- `--description` string — Description, at most 499 characters. (≤499 chars)
- `--integration-key` string — Optional supplied integration key. A non-empty value must have 35 characters, match the account suffix, and be unique. (≤35 chars)
- `--name` string — Integration name, 2–49 characters when supplied; omitted or empty uses the integration type default. (≤49 chars)
- `--plugin-type` string (required) — Enabled alert integration type identifier, such as 'standard.alert'.
- body-only (`--data`): settings (object)
- response: single object (`data` unwrapped to the top level) — fields: integration_id (integer); integration_key (string)

### integration-delete <integration-id>
Delete dedicated integration
- `<integration-id>` (positional, required) int64 — Dedicated integration ID returned by create, list, or info. Public integration IDs are not accepted. (min 1)

### integration-disable <integration-id>
Disable dedicated integration
- `<integration-id>` (positional, required) int64 — Dedicated integration ID returned by create, list, or info. Public integration IDs are not accepted. (min 1)

### integration-enable <integration-id>
Enable dedicated integration
- `<integration-id>` (positional, required) int64 — Dedicated integration ID returned by create, list, or info. Public integration IDs are not accepted. (min 1)

### integration-info <integration-id>
Get dedicated integration detail
- `<integration-id>` (positional, required) int64 — Dedicated integration ID returned by create, list, or info. Public integration IDs are not accepted. (min 1)
- response: single object (`data` unwrapped to the top level) — fields: category (string); channel_id (integer); created_at (string); description (string); exclusive_data_source_id (integer); integration_id (integer); integration_key (string); last_source_time (string); last_time (string); name (string); plugin_id (integer); plugin_type (string); plugin_type_name (string); settings (object); status (string); updated_at (string)

### integration-key-rotate <integration-id>
Rotate dedicated integration key
- `<integration-id>` (positional, required) int64 — Dedicated integration ID returned by create, list, or info. Public integration IDs are not accepted. (min 1)
- response: single object (`data` unwrapped to the top level) — fields: integration_key (string)

### integration-list <channel-id>
List dedicated integrations
- `<channel-id>` (positional, required) int64 — Channel ID to list dedicated integrations for. (min 1)
- `--limit` int64 — Number of items per page, at most 100. (1-100)
- `--name` string — Filter by integration name. (≤49 chars)
- `--page` int64 — Page number, starting at 1. (min 1)
- `--plugin-type` string — Filter by one integration type.
- `--search-after-ctx` string
- `--status` string — Filter by 'enabled' integrations that accept alerts or 'disabled' integrations that pause ingestion. · enum: enabled | disabled
- response: `{items: [...], limit, p, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: category (string); channel_id (integer); created_at (string); description (string); exclusive_data_source_id (integer); integration_id (integer); integration_key (string); last_source_time (string); last_time (string); name (string); plugin_id (integer); plugin_type (string); plugin_type_name (string); settings (object); status (string); updated_at (string)

### integration-update <integration-id>
Update dedicated integration
- `--description` string — New description, at most 499 characters; empty clears it, omitted or null keeps it. (≤499 chars)
- `<integration-id>` (positional, required) int64 — Dedicated integration ID returned by create, list, or info. Public integration IDs are not accepted. (min 1)
- `--name` string — New name, 2–49 characters; omitted or null keeps the stored name. (2-49 chars)
- body-only (`--data`): settings (object)
- response: same shape as `integration-info <integration-id>` above

<!-- GENERATED:channel[integration] END -->
