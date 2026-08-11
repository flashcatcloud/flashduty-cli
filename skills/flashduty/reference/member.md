# fduty member — command card

Prereq: `SKILL.md` read. `invite` sends invitation emails immediately (up to 20 per call). `delete` is **irreversible** — it removes the member from the organization. Default safety check rejects deletes when the member is referenced by escalation rules, schedules, team membership, etc. (pass `--is-force` to bypass). A member provisioned via SSO cannot be deleted at all, even with `--is-force` — disable SSO management for them first. `role-update` **replaces** all role assignments atomically; `role-grant`/`role-revoke` are additive/subtractive.

## Route here when

"成员 / 邀请 / 用户 / 角色 / member / invite / user profile / role assignment / org roster" → **member**. Sibling domains: `team` (team membership lists, not org-level members); `role` (role definitions — get role IDs here first); `person` (resolve a `person_id` → name with `fduty person infos <person_id> …`, e.g. ids returned by `schedule`/`oncall`/`incident`/`alert` output). Key IDs: **`member_id` (int)** from `member list`; **`role_id` (int)** from `fduty role list`.

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
- `--email` string — Email address. Only used when neither 'member_id' nor 'member_name' is provided
- `--is-force` bool — Force delete. Defaults to false, which checks for references from escalation rules, schedules, etc. Set to true to skip the reference check and delete immediately
- `--member-id` int64 — Member ID. When several lookup fields are provided, the first non-empty one wins in the order 'member_id' > 'member_name' > 'email' > 'phone' > 'ref_id'
- `--member-name` string — Member name. Only used when 'member_id' is not provided
- `--phone` string — Phone number. Only used when 'member_id', 'member_name', and 'email' are all absent
- `--ref-id` string — External reference ID. Only used when all other lookup fields are absent

### info
Get current member info
- response: single object (`data` unwrapped to the top level) — fields: account_avatar (string); account_email (string); account_id (integer); account_locale (string); account_name (string); account_role_ids (array<integer>); account_time_zone (string); avatar (string); country_code (string); domain (string); email (string); email_verified (boolean); is_external (boolean); locale (string); member_id (integer); member_name (string); phone (string); phone_verified (boolean); status (string); time_zone (string)

### info-reset
Reset member info
- `--country-code` string — Country or region code used to parse phone.
- `--email` string — Email address used to identify the member.
- `--from` string — Set to 'api' to mark an updated phone or email as verified. Only takes effect when the account has member invites disabled; any other value is ignored.
- `--member-id` int64 — Member ID used to identify the member.
- `--member-name` string — Member name used to identify the member.
- `--phone` string — Phone number used to identify the member. Include country_code when the number is not in E.164 format.
- `--ref-id` string — External reference ID used to identify the member.
- body-only (`--data`): updates (object) (required)

### invite
Invite members
- `--from` string — Invite source. Only takes effect when the account has member invites disabled and the value is 'api': members are created directly in the enabled state with email/phone marked verified and no invitation sent. Any other value follows the normal invite flow
- body-only (`--data`): members (array<object>) (required)
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: member_id (integer); member_name (string)

### list
List members
- `--asc` bool — Ascending order. Default: false (descending)
- `--limit` int64 — Page size. Defaults to 100 on the server when omitted or 0 (1-100)
- `--orderby` string — Sort field. Default: 'updated_at' · enum: created_at | updated_at
- `--page` int64 — Page number, 1-based (min 1)
- `--query` string — Substring match on member name or email; if the keyword parses as a phone number, an exact phone match is also applied
- `--role-id` int64 — Filter by role ID. Get role IDs from 'POST /role/list' (built-in roles: 2=Admin, 6=Responder, 8=Viewer)
- `--search-after-ctx` string
- response: `{items: [...], limit, p, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); account_role_ids (array<integer>); avatar (string); country_code (string); created_at (integer); email (string); email_verified (boolean); is_external (boolean); locale (string); member_id (integer); member_name (string); phone (string); phone_verified (boolean); ref_id (string); status (string); time_zone (string); updated_at (integer)

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
- `<role-ids>` (positional, required) intSlice — New role ID set. Replaces the member's existing roles entirely (not additive); get IDs from 'POST /role/list'. Leave empty to reset to the built-in Viewer role (ID 8)

<!-- GENERATED:member END -->

## Status values

`member list` returns `status` on each row:

- `enabled` — active, can log in
- `pending` — invitation sent, not yet accepted
- `deleted` — removed from the org (only visible if the API returns them; typically filtered out)

## Gotchas

- **Resolving a `person_id` → name: use `fduty person infos <person_id> …`, NOT `member list`.** `schedule`/`oncall`/`incident`/`alert` output returns `person_id`s, a **different namespace from `member_id`**. `fduty person infos` (the sibling `person` group) batch-resolves any number of `person_id`s to `person_name` in one call (rows under `.items[]`). Matching `member list` rows on `member_id == <person_id>` is wrong, and paginating the full roster to find them silently misses people on later pages.
- **`invite` members array is body-only — use `--data`.** Individual members cannot be passed as flat flags; the `members` array (with nested `role_ids`, `email`, `phone`, etc.) lives only in the JSON body. Up to 20 members per call.
- **`info-reset <member-id>` can be passed positionally or via `--member-id`** — both work: `fduty member info-reset <member_id> --member-name "New Name"` or `fduty member info-reset --member-id <member_id> --member-name "New Name"`. If both are given, the flag wins.
- **`role-grant` / `role-revoke` / `role-update` — role IDs can be passed positionally or via `--role-ids`.** Positional is shorter: `fduty member role-grant <role_id> [<role_id2>...] --member-id <member_id>`, or pass `--role-ids <role_id>,<role_id2>` instead. If both are given, the flag wins.
- **`role-update` is a full replacement.** List current roles with `member list` first; omitting a role removes it.
- **`delete` default is safe** (checks escalation rules / schedules / team membership). If it rejects with a reference error, review those references before using `--is-force`. An SSO-provisioned member rejects unconditionally — `--is-force` does not override that check.
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
