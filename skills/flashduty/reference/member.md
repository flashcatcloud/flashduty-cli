# fduty member — command card

Prereq: `SKILL.md` read. `invite` sends invitation emails immediately (up to 20 per call). `delete` is **irreversible** — it removes the member from the organization; default safety check rejects deletes when the member is referenced by escalation rules or schedules (pass `--is-force` to bypass). `role-update` **replaces** all role assignments atomically; `role-grant`/`role-revoke` are additive/subtractive.

## Route here when

"成员 / 邀请 / 用户 / 角色 / member / invite / user profile / role assignment / org roster" → **member**. Sibling domains: `team` (team membership lists, not org-level members); `role` (role definitions — get role IDs here first). Key IDs: **`member_id` (int)** from `member list`; **`role_id` (int)** from `fduty role list`.

## Intent → verb

| want | verb |
|---|---|
| find a member / look up their ID | `list` |
| who am I (current user) | `info` |
| update a member's profile fields | `info-reset` |
| invite new members to the org | `invite` |
| remove a member from the org | `delete` |
| add roles without touching others | `role-grant` |
| remove specific roles | `role-revoke` |
| set exactly these roles (replace all) | `role-update` |

## Hot flow — invite then assign role

```bash
# 1. find available role IDs
fduty role list --output-format toon

# 2. invite up to 20 members in one call; members array MUST go via --data
fduty member invite \
  --data '{"members":[{"email":"alice@example.com","member_name":"Alice","role_ids":[<role_id>]},{"email":"bob@example.com","member_name":"Bob","role_ids":[<role_id>]}]}'
# → returns items[].member_id for each new member

# 3. confirm they appear (status will be 'pending' until invite accepted)
fduty member list --query "alice" --output-format toon
```

## Hot flow — role change for an existing member

```bash
# 1. look up the member
fduty member list --query "alice" --output-format toon
# note member_id and current account_role_ids

# 2a. add a role without disturbing others (role-id is POSITIONAL)
fduty member role-grant <role_id> --member-id <member_id>

# 2b. OR: set the complete new role list (role-ids positional; replaces ALL roles)
fduty member role-update <role_id> <role_id2> --member-id <member_id>

# 3. verify
fduty member list --query "alice" --output-format toon
```

<!-- GENERATED:member START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### delete
Delete member
- `--country-code` string — Phone country code, used with phone
- `--email` string — Email address
- `--is-force` bool — Force delete. Defaults to false, which checks for references from escalation rules, schedules, etc. Set to true to skip the reference check and delete immediately
- `--member-id` int64 — Member ID
- `--member-name` string — Member name
- `--phone` string — Phone number
- `--ref-id` string — External reference ID

### info
Get current member info

### info-reset <member-id>
Reset member info
- `--avatar` string — Avatar URL
- `--country-code` string — Country code
- `--email` string — Email address
- `--locale` string — Locale · enum: zh-CN | en-US
- `<member-id>` (positional, required) int64 — Member ID of the member to update
- `--member-name` string — Display name (2-39 chars)
- `--phone` string — Phone number
- `--time-zone` string — Time zone

### invite
Invite members
- `--from` string — Invite source context
- body-only (`--data`): members (array<object>) (required)

### list
List members
- `--asc` bool — Ascending order
- `--limit` int64 — Page size (1-100)
- `--orderby` string — Sort field · enum: created_at | updated_at
- `--page` int64 — Page number (min 1)
- `--query` string — Search keyword
- `--role-id` int64 — Filter by role ID
- `--search-after-ctx` string

### role-grant <role-id> [<id2>...]
Grant role to member
- `--member-id` int64 (required) — Member ID
- `<role-ids>` (positional, required) intSlice — Role IDs to grant; appended to the member's current roles (duplicates are deduplicated).

### role-revoke <role-id> [<id2>...]
Revoke role from member
- `--member-id` int64 (required) — Member ID
- `<role-ids>` (positional, required) intSlice — Role IDs to remove from the member.

### role-update <role-id> [<id2>...]
Update member roles
- `--member-id` int64 (required) — Member ID
- `<role-ids>` (positional, required) intSlice — New set of role IDs

<!-- GENERATED:member END -->

## Status values

`member list` returns `status` on each row:

- `enabled` — active, can log in
- `pending` — invitation sent, not yet accepted
- `deleted` — removed from the org (only visible if the API returns them; typically filtered out)

## Gotchas

- **`invite` members array is body-only — use `--data`.** Individual members cannot be passed as flat flags; the `members` array (with nested `role_ids`, `email`, `phone`, etc.) lives only in the JSON body. Up to 20 members per call.
- **`info-reset <member-id>` is POSITIONAL.** Pass the member ID as the first bare argument, not `--member-id`: `fduty member info-reset <member_id> --member-name "New Name"`. The `--member-id` flag exists but the positional form is required per the `use` field.
- **`role-grant / role-revoke / role-update` — role IDs are POSITIONAL.** All three verbs take role IDs as positional args: `fduty member role-grant <role_id> [<role_id2>...] --member-id <member_id>`. The `--role-ids` flag also exists but the positional form is authoritative.
- **`role-update` is a full replacement.** List current roles with `member list` first; omitting a role removes it.
- **`delete` default is safe** (checks escalation rules / schedules). If it rejects with a reference error, review those references before using `--is-force`.
- **Empty `member list` result is authoritative** — if `--query` returns nothing the member does not exist; do not widen the query.

## Worked example

Look up a member then promote them to a new role:

```bash
# find member
fduty member list --query "carol" --output-format toon
# → member_id=4217, account_role_ids=[2]

# find the admin role ID
fduty role list --output-format toon
# → role_id=1 is "Admin"

# grant admin role (keeps existing role 2)
fduty member role-grant 1 --member-id 4217

# confirm
fduty member list --query "carol" --output-format toon
```
