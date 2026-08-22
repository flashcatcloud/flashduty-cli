# fduty template — command card

Prereq: `SKILL.md` read. Read verbs are free. `create`, `update`, `delete` mutate account-wide notification templates — confirm before running. `delete <template-id>` is **irreversible**.

**`update` is a full-object replace.** Every channel field you do not pass is written
empty. A one-channel edit is still a whole-object write — never call `update` without the
`info --json` snapshot from the hot flow below.

## Route here when

"通知模板 / 消息模板 / 告警通知格式 / 飞书模板 / Slack 模板 / 邮件模板 / template CRUD / custom template / preview notification / validate template" → **template**. NOT `channel` (channel = escalation policy routing; template = the rendered text/card body). The key ID is **`template_id`** (string), returned by `list` or `create`.

## Intent → verb

| want | verb |
|---|---|
| list all custom templates | `list` |
| detail of one template | `info <template-id>` |
| create a new template | `create` |
| update an existing template | `update <template-id>` |
| delete a template | `delete <template-id>` |
| see the built-in preset for a channel | `get-preset` |
| validate + preview a template file | `validate` |
| render inline template content against incident data | `preview` |
| browse available Go template variables | `variables` |
| browse Sprig / custom template functions | `functions` |

## Hot flow — customize and deploy a channel template

```bash
# 1. Fetch the built-in preset as a starting point (channel enum below)
fduty template get-preset --channel feishu --output-format toon

# 2. Save the source, edit in an editor, then validate from file
fduty template validate --channel feishu --file ./feishu.tpl

# 3. Preview with a real incident for realistic rendering (no file — inline content)
fduty template preview \
  --type feishu \
  --content "$(cat ./feishu.tpl)" \
  --incident-id <incident-id>

# 4. Create the template (template-name unique per account)
fduty template create \
  --template-name "Critical-Feishu-v2" \
  --feishu "$(cat ./feishu.tpl)" \
  --team-id 0

# 5. Verify
fduty template info <template-id> --output-format toon
```

## Hot flow — change one channel on an existing template

`update` overwrites the whole object, so this is read → modify → preview → write →
verify. Skipping step 1 or step 5 is how a live channel gets silently blanked.

```bash
T=<template-id>          # POSITIONAL on update/info/delete; --template-name always required
CHLEN='["dingtalk","dingtalk_app","email","feishu","feishu_app","slack","slack_app","sms","teams_app","telegram","voice","wecom","wecom_app","zoom"] as $ch | . as $t | $ch[] | "\(.)\t\($t[.] // "" | length)"'

# 1. Snapshot the whole template — this is both your backup and your write payload
fduty template info "$T" --json > /tmp/tpl.json
jq -r "$CHLEN" /tmp/tpl.json          # every channel and its current byte length

# 2. Edit only the channel you care about, on disk
jq -r '.feishu_app' /tmp/tpl.json > /tmp/feishu_app.tpl
#    …edit /tmp/feishu_app.tpl…

# 3. Preview the edited source against a REAL incident before writing
jq -n --rawfile c /tmp/feishu_app.tpl \
  '{type:"feishu_app", content:$c, incident_id:"<incident-id>"}' \
  | fduty template preview --data -

# 4. Write — rebuild the body from the snapshot, moving template bodies with jq and
#    NEVER through "$(...)": command substitution strips every trailing newline, so a
#    body ending in a blank line would come back silently shortened.
jq -c --rawfile feishu_app /tmp/feishu_app.tpl \
  '{template_id, template_name, description,
    dingtalk, dingtalk_app, email, feishu, feishu_app, slack, slack_app, sms,
    teams_app, telegram, voice, wecom, wecom_app, zoom}
   | .feishu_app = $feishu_app' /tmp/tpl.json \
  | fduty template update --data -
#    Everything left out of that object is patch-semantics and survives untouched:
#    team_id, feishu_app_card_v2_table_enabled, incident_card_hidden_fields.

# 5. Verify LENGTHS, not just the field you edited
fduty template info "$T" --json > /tmp/tpl_after.json
diff <(jq -r "$CHLEN" /tmp/tpl.json) <(jq -r "$CHLEN" /tmp/tpl_after.json)
#    Only the channel you edited may differ. A channel that dropped to 0 was wiped; a
#    channel a few bytes shorter was truncated — restore it from /tmp/tpl.json.
```

Checking only the field you edited is **not** verification — the damage from a full-object
replace always lands on the fields you did not touch, and comparing only *which* fields are
non-empty misses a body that was shortened rather than cleared.

<!-- GENERATED:template START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### create
Create a template
- `--description` string — Free-form description. Up to 500 characters. (≤500 chars)
- `--dingtalk` string — DingTalk robot message template source.
- `--dingtalk-app` string — DingTalk app message template source.
- `--email` string — Email body template source (Go 'html/template' syntax).
- `--feishu` string — Feishu robot message template source.
- `--feishu-app` string — Feishu app message template source.
- `--feishu-app-card-v2-table-enabled` bool — Render alert labels as a table in Feishu app cards.
- `--slack` string — Slack robot message template source.
- `--slack-app` string — Slack app message template source.
- `--sms` string — SMS template source (Go 'text/template' syntax).
- `--team-id` int64 — Team scope. 0 for account-wide.
- `--teams-app` string — Microsoft Teams app message template source.
- `--telegram` string — Telegram bot message template source.
- `--template-name` string (required) — Template name, unique per account. 1–39 characters. (1-39 chars)
- `--voice` string — Voice call script template source.
- `--wecom` string — WeCom robot message template source.
- `--wecom-app` string — WeCom app message template source.
- `--zoom` string — Zoom bot message template source.
- body-only (`--data`): incident_card_hidden_fields (object)
- response: single object (`data` unwrapped to the top level) — fields: template_id (string); template_name (string)

### delete <template-id>
Delete a template
- `<template-id>` (positional, required) string — Target template ID. Pass '000000000000000000000001' to address the built-in preset.

### functions
List available template functions
- `--type` string

### get-preset
Get the preset template for a channel
- `--channel` string
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); created_at (string); creator_id (integer); deleted_at (string); description (string); dingtalk (string); dingtalk_app (string); email (string); feishu (string); feishu_app (string); feishu_app_card_v2_table_enabled (boolean); incident_card_hidden_fields (object); slack (string); slack_app (string); sms (string); status (string); team_id (integer); teams_app (string); telegram (string); template_id (string); template_name (string); updated_at (string); updated_by (integer); voice (string); wecom (string); wecom_app (string); zoom (string)

### info <template-id>
Get template detail
- `<template-id>` (positional, required) string — Target template ID. Pass '000000000000000000000001' to address the built-in preset.
- response: same shape as `get-preset` above

### list
List templates
- `--asc` bool — Ascending sort order.
- `--creator-id` int64 — Filter by creator member ID; obtain member IDs from 'POST /member/list'.
- `--is-my-team` bool — When true, only return templates scoped to teams the caller belongs to.
- `--limit` int64 — Page size. Capped at 100. (1-100)
- `--orderby` string — Sort field. · enum: created_at | updated_at
- `--page` int64 — Page number, starting at 1. (min 1)
- `--query` string — Regex or substring match on template_name.
- `--search-after-ctx` string
- `--team-ids` intSlice — Filter by specific team IDs.
- response: `{items: [...], has_next_page, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); created_at (string); creator_id (integer); deleted_at (string); description (string); dingtalk (string); dingtalk_app (string); email (string); feishu (string); feishu_app (string); feishu_app_card_v2_table_enabled (boolean); incident_card_hidden_fields (object); slack (string); slack_app (string); sms (string); status (string); team_id (integer); teams_app (string); telegram (string); template_id (string); template_name (string); updated_at (string); updated_by (integer); voice (string); wecom (string); wecom_app (string); zoom (string)

### preview
Preview template
- `--content` string (required) — Template content to render.
- `--incident-id` string — Incident ID whose data is used to render the template; mock data is used when omitted. A MongoDB ObjectID hex string.
- `--type` string (required) — Template channel type that selects the rendering engine.
- body-only (`--data`): incident_card_hidden_fields (object)
- response: single object (`data` unwrapped to the top level) — fields: content (string); fixed_fields (array<object>); message (string); success (boolean)

### update <template-id>
Update a template
- `--description` string — Free-form description. Up to 500 characters. (≤500 chars)
- `--dingtalk` string — DingTalk robot message template source.
- `--dingtalk-app` string — DingTalk app message template source.
- `--email` string — Email body template source (Go 'html/template' syntax).
- `--feishu` string — Feishu robot message template source.
- `--feishu-app` string — Feishu app message template source.
- `--feishu-app-card-v2-table-enabled` bool — When set, enable or disable table rendering for alert labels in Feishu app cards. Omit to keep the existing setting.
- `--slack` string — Slack robot message template source.
- `--slack-app` string — Slack app message template source.
- `--sms` string — SMS template source (Go 'text/template' syntax).
- `--team-id` int64 — Team scope. 0 for account-wide.
- `--teams-app` string — Microsoft Teams app message template source.
- `--telegram` string — Telegram bot message template source.
- `<template-id>` (positional, required) string — Target template ID; obtain it from 'POST /template/list'.
- `--template-name` string (required) — Template name. 1–39 characters. (1-39 chars)
- `--voice` string — Voice call script template source.
- `--wecom` string — WeCom robot message template source.
- `--wecom-app` string — WeCom app message template source.
- `--zoom` string — Zoom bot message template source.
- body-only (`--data`): incident_card_hidden_fields (object)

### validate
Validate and preview a template
- `--channel` string
- `--file` string
- `--incident` string
- response: same shape as `preview` above

### variables
List available template variables
- `--category` string

<!-- GENERATED:template END -->

## Channel identifiers (load-bearing — wrong value 400s)

`--channel` / `--type` values (both flags use the same enum):

`dingtalk` · `dingtalk_app` · `email` · `feishu` · `feishu_app` · `slack` · `slack_app` · `sms` · `teams_app` · `telegram` · `wecom` · `wecom_app` · `zoom`

Note: `create` / `update` flags use **hyphenated** names (`--dingtalk-app`, `--feishu-app`, `--slack-app`, `--wecom-app`, `--teams-app`). `get-preset` / `validate` / `preview` use **underscored enum values** (`dingtalk_app`, `feishu_app` …).

## Gotchas

- **`info`, `update`, `delete` take `<template-id>` as a positional first argument** — pass it bare, not as `--template-id`. `create`, `list`, `preview`, `validate`, `get-preset`, `functions`, `variables` take all inputs as flags.
- **`update` is a full-object replace — every channel field you omit is CLEARED.** The
  server writes all 14 channel-content fields plus `description` on every call, and a
  field absent from the request arrives as the empty string: omitting `--dingtalk-app`
  sets `dingtalk_app` to `""`, and that channel silently stops rendering for every
  escalation rule bound to the template. Only the pointer-typed inputs survive omission:
  `team_id`, `feishu_app_card_v2_table_enabled`, `incident_card_hidden_fields`. (`status`
  is not part of `update`'s request at all — it moves only through the separate
  enable/disable endpoints, which the CLI does not expose — so `update` can never change
  it.) Always snapshot with `info --json` first and rebuild the body from that snapshot —
  see the hot flow above. `--template-name` is required on every update even when
  unchanged.
- **Never move a template body through `"$(cat …)"` or `"$(jq -r …)"`.** Bash command
  substitution strips *all* trailing newlines, so a body that legitimately ends in a blank
  line is written back shortened — and a check that only asks which fields are non-empty
  cannot see it, because the field is still non-empty. Carry bodies with `jq --rawfile`
  and write with `--data -`, as the hot flow does.
- **`--feishu-app-card-v2-table-enabled` uses pointer semantics on `update`** — unlike the plain string channel-content flags, it patches the table-rendering setting only when the flag is explicitly passed; omit it to leave the existing setting untouched. It is a plain bool on `create` (no prior setting to preserve).
- **`list` returns every channel's full template source for every row** — a few dozen
  templates blow past a tool-output cap in one call. Never render it directly: go to a
  file and project. `fduty template list --limit 100 --json > /tmp/tpl_list.json && jq -r
  '.items[] | [.template_id, .template_name, .team_id] | @tsv' /tmp/tpl_list.json`.
- **`delete` is permanent.** The built-in preset (`template_id = 000000000000000000000001`) can be addressed by that sentinel ID in `info` and `delete` — don't delete it.
- **`validate` reads from a local `--file`; `preview` takes inline `--content`.** They are complementary: `validate` gives size-vs-limit diagnostics; `preview` renders against real or mock incident data.
- **`email` uses `html/template` syntax; `sms` and `voice` use `text/template`** — auto-escaping rules differ. Don't mix them.
- **`functions --type` values**: `custom`, `sprig`, or `all`. **`variables --category` values**: `core`, `time`, `people`, `alerts`, `labels`, `context`, `notification`, `post_incident`.

## Worked example

```bash
# Browse variables available in templates, then validate a draft
fduty template variables --category core --output-format toon
fduty template validate --channel slack --file ./slack-draft.tpl --incident <incident-id>
# On success, create it
fduty template create --template-name "Ops-Slack-Alert" --slack "$(cat ./slack-draft.tpl)"
# → returns template_id; assign it to a channel in the escalation policy UI.
```
