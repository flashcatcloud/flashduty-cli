# fduty field — command card

Prereq: `SKILL.md` read. Read verbs (`list`, `info`) are free. `delete` is **irreversible** — double-check the field-id before running it.

## Route here when

"自定义字段 / 事件字段 / 字段选项 / incident field / custom field / field schema" → **field**.
NOT `enrichment` (enrichment = rules that auto-populate field values; field = the schema that defines those fields).
You need a **`field_id`** (24-char hex ObjectID) — get it from `field list`.

## Intent → verb

| want | verb |
|---|---|
| see all custom fields | `list` |
| filter fields by name | `list --name <keyword>` |
| full detail for one field | `info <field-id>` |
| create a new custom field | `create` |
| rename, re-describe, or change options | `update <field-id>` |
| permanently remove a field | `delete <field-id>` |

## Hot flow — create a select field and update its options

```bash
# 1. Check what already exists (avoid duplicate display-name)
fduty field list --output-format toon

# 2. Create a single-select field (field-type + value-type + options are all required here)
fduty field create \
  --display-name "Root Cause" \
  --field-name "root_cause" \
  --field-type single_select \
  --value-type string \
  --options "hardware failure" --options "software bug" --options "human error"
# → returns field_id; save it.

# 3. Later: add an option (pass the full replacement list)
fduty field update <field-id> \
  --options "hardware failure" --options "software bug" --options "human error" --options "network issue"
```

<!-- GENERATED:field START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### create
Create field
- `--description` string — Optional free-text description. (≤499 chars)
- `--display-name` string (required) — Human-readable name. Must be unique within the account. (≤39 chars)
- `--field-name` string (required) — Machine name. Must start with a letter or underscore; 1–40 chars of ''. Immutable after creation. (≤39 chars) · enum: a-zA-Z0-9_
- `--field-type` string (required) — Field input type. Immutable after creation. · enum: checkbox | multi_select | single_select | text
- `--options` stringSlice — Required and non-empty for 'single_select'/'multi_select' (unique strings, each 1–200 chars). Must be omitted or empty for 'checkbox'/'text'.
- `--value-type` string (required) — Stored value type. 'checkbox' requires 'bool'; 'single_select'/'multi_select'/'text' require 'string'. Immutable after creation. · enum: string | bool | float
- body-only (`--data`): default_value (any)

### delete <field-id>
Delete field
- `<field-id>` (positional, required) string — Field ID — 24-character hex ObjectID.

### info <field-id>
Get field detail
- `<field-id>` (positional, required) string — Field ID — 24-character hex ObjectID.

### list
List custom fields
- `--name` string

### update <field-id>
Update field
- `--description` string — New description.
- `--display-name` string — New display name. Must remain unique within the account. (≤39 chars)
- `<field-id>` (positional, required) string — Field ID — 24-character hex ObjectID.
- `--options` stringSlice — Replacement options list. Must obey the same per-type rules as create.
- body-only (`--data`): default_value (any)

<!-- GENERATED:field END -->

## Type constraints (immutable triad — wrong values 400)

`--field-type`, `--field-name`, and `--value-type` are **permanently fixed at creation** and cannot be changed via `update`.

| `--field-type` | `--value-type` | `--options` |
|---|---|---|
| `single_select` | `string` | required, ≥1 unique string |
| `multi_select` | `string` | required, ≥1 unique string |
| `text` | `string` | must be omitted |
| `checkbox` | `bool` | must be omitted |

`default_value` (optional) can be set or changed; pass it via `--data '{"default_value": ...}'` because it has no typed flag.

## Gotchas

- **`delete`, `info`, `update` take `<field-id>` as a POSITIONAL first argument**, not `--field-id`. Example: `fduty field delete <field-id>`, not `--field-id <field-id>`.
- **`--options` replaces the whole list on `update`** — omitting it leaves options unchanged, but a partial list silently drops the missing values. Always pass the full desired set.
- **`--field-name` is the machine key** (`[a-zA-Z0-9_]`, starts with letter/underscore, ≤40 chars). It is the stable identifier for downstream enrichment rules — choose it carefully; it cannot be renamed.
- **`delete` is permanent and cascades** — any enrichment rules that reference the field by `field_name` will lose their target. Confirm the name against `field list` before deleting.
- **Empty `field list` is authoritative** — if the field isn't listed, it doesn't exist for this account. Do not retry with widened queries.

## Worked example

```bash
# Create a checkbox field (value-type must be bool; options must be omitted)
fduty field create \
  --display-name "Needs Postmortem" \
  --field-name "needs_postmortem" \
  --field-type checkbox \
  --value-type bool \
  --description "Flag incidents that require a postmortem write-up."
```
