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

### info
Get team detail
- `--ref-id` string — External reference ID.
- `--team-id` int64 — Team ID.
- `--team-name` string — Team name.

### infos <team-id> [<id2>...]
Batch get teams
- `<team-ids>` (positional, required) intSlice — List of team IDs to look up. Max 100.

### list
List teams
- `--asc` bool
- `--limit` int
- `--name` string
- `--orderby` string
- `--page` int
- `--person-id` int64

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
- `--emails` stringSlice — Email addresses to invite as members.
- `--person-ids` intSlice — Member IDs to set as team members. Replaces the existing member list.
- `--phones` stringSlice — Phone numbers to invite as members.
- `--ref-id` string — External reference ID for HR system integration.
- `--reset-if-name-exist` bool — If true and a team with the same name already exists, reset its membership to the provided person_ids.
- `--team-id` int64 — Team ID. Omit or set to 0 to create a new team.
- `--team-name` string (required) — Team display name. 1–39 characters. (1-39 chars)

<!-- GENERATED:team END -->

## Key concepts

- **`status`** on `team list` rows: `enabled` | `disabled`. A disabled team still exists but is excluded from most operational contexts.
- **`infos <team-id> [<id2>...]`** — takes team IDs as **positional args** (space-separated), not `--team-ids`. The response wraps under `items[]` (pipe `jq '.items[]'` with `--json`), NOT `.data.items[]`.
- **`upsert` lookup key** — matched by `--team-id` (if non-zero) or by `--team-name` (name collision). Pass `--reset-if-name-exist` to overwrite membership on a name match; omit it to leave the existing members untouched.

## Gotchas

- **`--person-ids` on `update` / `create` / `upsert` is a full replacement**, not an append. Read the current list with `get --id` first, or you will silently remove members.
- **`get` vs `info`** — both fetch a single team; `get` accepts `--id`/`--name`/`--ref-id`; `get [<id>]` also allows the ID as a positional arg. `info` uses `--team-id`/`--team-name`/`--ref-id` flags only. Prefer `get` for interactive lookup.
- **`delete` is irreversible** and requires confirmation unless `--force` is set. Always confirm the correct `--id` (not `--name`) in scripts to avoid name-collision accidents.
- **`infos` positional trap** — the `use` is `infos <team-id> [<id2>...]`; IDs are space-separated positional args, not a flag. `fduty team infos 101 102 103`, not `--team-ids 101,102,103`.
- **`list` JSON shape** — `--json` returns a top-level array; pipe `jq '.[]'`, NOT `.items[]`.
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
