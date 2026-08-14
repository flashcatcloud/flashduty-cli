# fduty team — command card

Prereq: `SKILL.md` read. **SKILL.md + this card = full competence on teams — no `--help` needed.** Read verbs are free; `delete` is **irreversible** (always `--force` in scripted contexts); `update --person-ids` **replaces** the entire member list — dangerous without a prior `get`.

## Route here when

"团队 / 成员管理 / 创建团队 / 查找团队 / HR同步 / team ID / person ID归属" → **team**. Key IDs:
- **`team_id` (int64)** — from `fduty team list` or `team get --name`.
- **`--person-ids` inputs are member IDs** — look up via `fduty member list --query <name-or-email>` (member card, not here). The API field is named `person_ids`, but team membership expects member IDs.

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
fduty team list --name "SRE Platform" --output-format toon
# 2. Create with initial members (member IDs from member list)
fduty team create --name "SRE Platform" --description "Site Reliability" \
  --person-ids 1001,1002,1003
# 3. Verify — note the returned team_id
fduty team get --name "SRE Platform" --output-format toon
```

## Hot flow — update members safely

```bash
# ALWAYS read current members before --person-ids (it REPLACES, not appends)
fduty team get --id <team-id> --output-format toon
# Then pass the FULL desired set (existing + new)
fduty team update --id <team-id> --person-ids 1001,1002,1003,1004
```

<!-- GENERATED:team START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

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
- `<team-ids>` (positional, required) intSlice — List of team IDs to look up. Max 100.
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
- `--country-code` string — Default country code applied to any 'phones' entries that are not in E.164 format.
- `--description` string — Free-form description. (≤500 chars)
- `--emails` stringSlice — Add existing members to the team by email. Addresses that don't match an existing member are silently ignored — no invitation is sent.
- `--person-ids` intSlice — Member IDs to set as team members. Replaces the existing member list.
- `--phones` stringSlice — Add existing members to the team by phone number. Numbers that don't match an existing member are silently ignored; non-E.164 numbers are parsed with 'countryCode'.
- `--ref-id` string — External reference ID for HR system integration.
- `--reset-if-name-exist` bool — If true and a team with the same name already exists, reset its membership to the provided person_ids.
- `--team-id` int64 — Team ID. Omit or set to 0 to create a new team.
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
- **`infos` accepts IDs either way** — space-separated positional args or comma-separated `--team-ids`: `fduty team infos 101 102 103` or `fduty team infos --team-ids 101,102,103`. If both are given, the flag wins.
- **`upsert` requires `--team-name`** even when updating by `--team-id`; omitting it returns a validation error.

## Worked example

```bash
# Idempotent HR-sync upsert: create "Payments" or reset its membership if it already exists
fduty team upsert --team-name "Payments" \
  --description "Payments engineering" \
  --person-ids 2001,2002,2003 \
  --reset-if-name-exist \
  --output-format toon
# → returns team_id; store it for oncall schedule / channel filtering
```
