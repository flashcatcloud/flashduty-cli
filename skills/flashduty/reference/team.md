# flashduty team — command card

Prereq: `SKILL.md` read. **SKILL.md + this card = full competence on teams — no `--help` needed.** Read verbs are free; `delete` is **irreversible** (always `--force` in scripted contexts); `update --person-ids` **replaces** the entire member list — dangerous without a prior `get`.

## Route here when

"团队 / 成员管理 / 创建团队 / 查找团队 / HR同步 / team ID / person ID归属" → **team**. Key IDs:
- **`team_id` (int64)** — from `flashduty team list` or `team get --name`.
- **`--person-ids` inputs are member IDs** — look up via `flashduty member list --query <name-or-email>` (member card, not here). The API field is named `person_ids`, but team membership expects member IDs.

NOT this card: on-call schedules (oncall), incidents (incident), channels (channel).

## Intent → verb

| want | verb |
|---|---|
| browse all teams + their member IDs | `list` |
| one team's full detail (members, ref-id, status) | `get` |
| same but via generated API path | `info` |
| batch resolve several team IDs at once | `infos` |
| create a brand-new team | `create` |
| rename / change description / swap members | `update` |
| create-or-update idempotently (HR sync) | `upsert` |
| permanently remove a team | `delete` |

## Hot flow — create a team and verify membership

```bash
# 1. Check name doesn't already exist
flashduty team list --name "SRE Platform" --output-format toon
# 2. Create with initial members (member IDs from member list)
flashduty team create --name "SRE Platform" --description "Site Reliability" \
  --person-ids 1001,1002,1003
# 3. Verify — note the returned team_id
flashduty team get --name "SRE Platform" --output-format toon
```

## Hot flow — update members safely

```bash
# ALWAYS read current members before --person-ids (it REPLACES, not appends)
flashduty team get --id <team-id> --output-format toon
# Then pass the FULL desired set (existing + new)
flashduty team update --id <team-id> --person-ids 1001,1002,1003,1004
```

<!-- GENERATED:team START · 由 flashduty __dump-commands 同步 · 勿手改 fence 内 -->

### create
Create a new team
- `--description` string
- `--emails` string
- `--name` string
- `--person-ids` string
- `--ref-id` string

### delete
Delete a team
- `--force` bool
- `--id` int64
- `--name` string
- `--ref-id` string

### get [<id>]
Get team detail
- `--id` int64
- `--name` string
- `--ref-id` string
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); created_at (string); creator_id (integer); creator_name (string); description (string); person_ids (array<integer>); ref_id (string); status (string); team_id (integer); team_name (string); updated_at (string); updated_by (integer); updated_by_name (string)

### info
Get team detail
- `--ref-id` string — External reference ID. When provided, takes precedence over 'team_name' and 'team_id'.
- `--team-id` int64 — Team ID. At least one of the three lookup fields is required; lowest priority — only used when neither 'ref_id' nor 'team_name' is provided.
- `--team-name` string — Team name. Only used when 'ref_id' is not provided; takes precedence over 'team_id'.
- response: same shape as `get [<id>]` above

### infos <team-id> [<id2>...]
Batch get teams
- `<team-ids>` (positional, required) intSlice — List of team IDs to look up.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: person_ids (array<integer>); team_id (integer); team_name (string)

### list
List teams
- `--asc` bool
- `--limit` int
- `--name` string
- `--orderby` string
- `--page` int
- `--person-id` int64
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); created_at (string); creator_id (integer); creator_name (string); description (string); person_ids (array<integer>); ref_id (string); status (string); team_id (integer); team_name (string); updated_at (string); updated_by (integer); updated_by_name (string)

### update
Update an existing team
- `--description` string
- `--emails` string
- `--id` int64
- `--name` string
- `--person-ids` string
- `--ref-id` string

### upsert
Create or update a team
- `--country-code` string — Region hint for 'phones' entries without a '+' prefix: an ISO 3166-1 alpha-2 code such as 'CN' (the default when omitted) or a digit calling code such as '86'.
- `--description` string — Free-form description, up to 500 characters. On update, omitting it clears the stored description. (≤500 chars)
- `--emails` stringSlice — Emails of existing members to place in the team; merged with 'person_ids' and 'phones'. Addresses that match no member are ignored and no invitation is sent.
- `--person-ids` intSlice — Person IDs to place in the team (member IDs, or the account ID for the account owner). Together with 'emails' and 'phones' they replace the entire member list; IDs that are not members of the account are ignored.
- `--phones` stringSlice — Phone numbers of existing members to place in the team; merged with 'person_ids' and 'emails'. Numbers without a '+' prefix are parsed with 'countryCode'; an unparseable number fails the whole request with '400', and numbers that match no member are ignored.
- `--ref-id` string — External reference ID (for example a department ID in an HR system), usable as a lookup key in 'POST /team/info' and 'POST /team/delete'. On update, omitting it clears the stored value.
- `--reset-if-name-exist` bool — When true and 'team_id' is 0, an existing team with the same 'team_name' is updated in place instead of returning a name-conflict error.
- `--team-id` int64 — ID of the team to update. Omit or set to 0 to create a new team; a non-zero ID must belong to an existing team of the account.
- `--team-name` string (required) — Team display name. 1–39 characters. (1-39 chars)
- response: single object (`data` unwrapped to the top level) — fields: team_id (integer); team_name (string)

<!-- GENERATED:team END -->

## Key concepts

- **`status`** on `team list` rows: `enabled` | `disabled`. A disabled team still exists but is excluded from most operational contexts.
- **`infos <team-id> [<id2>...]`** — takes team IDs as space-separated positional args, or comma-separated via `--team-ids`; both work.
- **`upsert` lookup key** — matched by `--team-id` (if non-zero) or by `--team-name` (name collision). Pass `--reset-if-name-exist` to overwrite membership on a name match; omit it to leave the existing members untouched.

## Gotchas

- **`--person-ids` on `update` / `create` / `upsert` is a full replacement**, not an append. Read the current list with `get --id` first, or you will silently remove members.
- **`get` vs `info`** — both fetch a single team; `get` accepts `--id`/`--name`/`--ref-id`; `get [<id>]` also allows the ID as a positional arg. `info` uses `--team-id`/`--team-name`/`--ref-id` flags only. Prefer `get` for interactive lookup.
- **`delete` is irreversible** and requires confirmation unless `--force` is set. Always confirm the correct `--id` (not `--name`) in scripts to avoid name-collision accidents.
- **`infos` accepts IDs either way** — space-separated positional args or comma-separated `--team-ids`: `flashduty team infos 101 102 103` or `flashduty team infos --team-ids 101,102,103`. If both are given, the flag wins.
- **`upsert` requires `--team-name`** even when updating by `--team-id`; omitting it returns a validation error.

## Worked example

```bash
# Idempotent HR-sync upsert: create "Payments" or reset its membership if it already exists
flashduty team upsert --team-name "Payments" \
  --description "Payments engineering" \
  --person-ids 2001,2002,2003 \
  --reset-if-name-exist \
  --output-format toon
# → returns team_id; store it for oncall schedule / channel filtering
```
