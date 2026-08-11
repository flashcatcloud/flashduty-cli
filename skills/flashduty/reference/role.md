# fduty role — command card

Prereq: `SKILL.md` read. Read verbs are free; `delete` is **irreversible** — confirm the role-id first. `upsert --permission-ids` **replaces** the entire permission set on an existing role.

## Route here when

"角色 / 权限 / RBAC / 授权 / 角色成员 / 自定义角色" → **role**, NOT `member` (member = person identity/contact) or `team` (team = ownership group). Key IDs: **`role-id` (int)** from `role list`; **`member-id` (int)** from `member list`; **`permission-id` (int)** from `role permission-list`.

## Intent → verb

| want | verb |
|---|---|
| all roles | `list` |
| one role's detail | `info` |
| create a custom role | `upsert` (omit `--role-id` or set to 0) |
| update role name / description / permissions | `upsert --role-id` (replaces permission set) |
| disable a role temporarily | `disable` |
| re-enable a disabled role | `enable` |
| permanently remove a role | `delete` |
| assign a role to members | `member-grant` |
| remove a role from members | `member-revoke` |
| browse all available permissions | `permission-list` |
| browse raw permission factors | `permission-factor-list` |

## Hot flow — create a role and assign it

```bash
# 1. Browse available permissions with role membership annotation
fduty role permission-list --with-all --output-format toon

# 2. Create the role with chosen permission IDs (note: ids from step 1)
fduty role upsert --role-name "Incident Responder" \
  --description "Read incidents and manage on-call." \
  --permission-ids 101,102,305

# 3. Find the new role ID
fduty role list --output-format toon

# 4. Find member IDs to assign (member-id is POSITIONAL, role-id is a flag)
fduty member list --output-format toon

# 5. Grant role to members (first member-id is positional; additional ids space-separated)
fduty role member-grant <member-id> --role-id <role-id>
# Grant to multiple: fduty role member-grant <id1> <id2> <id3> --role-id <role-id>
```

## Hot flow — audit and update an existing role

```bash
# 1. Find the role
fduty role list --output-format toon

# 2. Inspect current permissions (is_granted shows which are currently set)
fduty role permission-list --role-ids <role-id> --with-all --output-format toon

# 3. Update permissions (--permission-ids is the FULL replacement set)
fduty role upsert --role-id <role-id> --role-name "Incident Responder" \
  --permission-ids 101,102,305,410
```

<!-- GENERATED:role START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### delete <role-id>
Delete a role
- `<role-id>` (positional, required) int64 — Role ID to operate on. Get IDs from 'POST /role/list' (built-in roles: 2=Admin, 6=Responder, 8=Viewer).

### disable <role-id>
Disable a role
- `<role-id>` (positional, required) int64 — Role ID to operate on. Get IDs from 'POST /role/list' (built-in roles: 2=Admin, 6=Responder, 8=Viewer).

### enable <role-id>
Enable a role
- `<role-id>` (positional, required) int64 — Role ID to operate on. Get IDs from 'POST /role/list' (built-in roles: 2=Admin, 6=Responder, 8=Viewer).

### info <role-id>
Get role detail
- `<role-id>` (positional, required) int64 — Role ID to query. Get IDs from 'POST /role/list' (built-in roles: 2=Admin, 6=Responder, 8=Viewer).
- response: single object (`data` unwrapped to the top level) — fields: created_at (integer); description (string); editable (boolean); permission_ids (array<integer>); role_id (integer); role_name (string); status (string); updated_at (integer)

### list
List roles
- `--asc` bool — Ascending sort order. Default: false (descending).
- `--orderby` string — Sort field. Default: 'updated_at'. · enum: created_at | updated_at
- response: `{items: [...], total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: created_at (integer); description (string); editable (boolean); permission_ids (array<integer>); role_id (integer); role_name (string); status (string); updated_at (integer)

### member-grant <member-id> [<id2>...]
Grant role to members
- `<member-ids>` (positional, required) intSlice — Member IDs to grant/revoke the role. Max 100.
- `--role-id` int64 (required) — Role ID to grant or revoke. Get IDs from 'POST /role/list'.

### member-revoke <member-id> [<id2>...]
Revoke role from members
- `<member-ids>` (positional, required) intSlice — Member IDs to grant/revoke the role. Max 100.
- `--role-id` int64 (required) — Role ID to grant or revoke. Get IDs from 'POST /role/list'.

### permission-factor-list
List permission factors
- `--factor-types` stringSlice — Filter by factor type. · enum: api | button | visit | menu | url
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: factor_name (string); factor_type (string)

### permission-list
List permissions
- `--role-ids` intSlice — Filter to permissions granted to these roles.
- `--with-all` bool — If true, return all permissions with is_granted set to indicate which are granted.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: class (string); description (string); id (integer); is_granted (boolean); permission_name (string); permission_type (string); scope (string); status (string)

### upsert
Create or update a role
- `--description` string — Role description. (≤499 chars)
- `--permission-ids` intSlice — Permission IDs to grant. Replaces the existing set.
- `--role-id` int64 — Role ID. Omit or set to 0 to create.
- `--role-name` string (required) — Role display name. 1–39 characters. (1-39 chars)
- response: single object (`data` unwrapped to the top level) — fields: role_id (integer); role_name (string)

<!-- GENERATED:role END -->

## Key concepts

- **`permission-id` vs `permission-factor`**: `permission-list` returns coarse permission objects (id, name, class, scope, type=read|manage) — use these ids in `upsert --permission-ids`. `permission-factor-list` returns fine-grained factors (api/button/menu/url/visit strings like `template:read:info`) — useful for auditing what a permission covers, but not accepted by `upsert`.
- **`permission-list --with-all`**: returns every permission in the system with `is_granted=true/false` for the requested `--role-ids`. Omit `--role-ids` + `--with-all` to see the full catalog without annotation.

## Gotchas

- **`delete`, `disable`, `enable`, `info` take `<role-id>` positionally or via `--role-id`** — both work: `fduty role delete <role-id>` or `fduty role delete --role-id <role-id>`. If both are given, the flag wins.
- **`member-grant` / `member-revoke`: `<member-id>` is POSITIONAL (one or more space-separated); `--role-id` is a flag** — easy to flip. Example: `fduty role member-grant 123 456 --role-id 7`. Member IDs can also be passed via `--member-ids` instead of the positional (same fold-then-override rule).
- **`upsert --permission-ids` replaces the full set** on update — omitting it clears all permissions. Always read `permission-list --role-ids <id> --with-all` first to get the current set before modifying.
- **`upsert` with no `--role-id` (or `--role-id 0`) creates; with `--role-id N` updates** — the verb doubles as create and update; check for an existing role with `list` to avoid accidental duplicates.
- **`delete` is irreversible** — members who had this role lose its permissions immediately. Prefer `disable` to park a role without destroying it.
- **Max 100 members per grant/revoke call** — batch if the list is longer.

## Worked example

```bash
# Revoke a role from a single member
fduty role member-revoke <member-id> --role-id <role-id>
# Revoke from multiple members in one call
fduty role member-revoke <id1> <id2> <id3> --role-id <role-id>
```
