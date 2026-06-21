# fduty template — command card

Prereq: `SKILL.md` read. Read verbs are free. `create`, `update`, `delete` mutate account-wide notification templates — confirm before running. `delete <template-id>` is **irreversible**.

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

## Hot flow — update one channel on an existing template

```bash
# template-id is POSITIONAL; --template-name is required even on update
fduty template update <template-id> \
  --template-name "Critical-Feishu-v2" \
  --feishu "$(cat ./feishu-v3.tpl)"
```

<!-- GENERATED:template START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### create
Create a template
- `--description` string — Free-form description. Up to 500 characters. (≤500 chars)
- `--dingtalk` string — DingTalk robot message template source.
- `--dingtalk-app` string — DingTalk app message template source.
- `--email` string — Email body template source (Go 'html/template' syntax).
- `--feishu` string — Feishu robot message template source.
- `--feishu-app` string — Feishu app message template source.
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

### delete <template-id>
Delete a template
- `<template-id>` (positional, required) string — Target template ID. Pass '000000000000000000000001' to address the built-in preset.

### functions
List available template functions
- `--type` string

### get-preset
Get the preset template for a channel
- `--channel` string

### info <template-id>
Get template detail
- `<template-id>` (positional, required) string — Target template ID. Pass '000000000000000000000001' to address the built-in preset.

### list
List templates
- `--asc` bool — Ascending sort order.
- `--creator-id` int64 — Filter by creator member ID.
- `--is-my-team` bool — When true, only return templates scoped to teams the caller belongs to.
- `--limit` int64 — Page size. Capped at 100. (1-100)
- `--orderby` string — Sort field. · enum: created_at | updated_at
- `--page` int64 — Page number, starting at 1. (min 1)
- `--query` string — Regex or substring match on template_name.
- `--search-after-ctx` string
- `--team-ids` intSlice — Filter by specific team IDs.

### preview
Preview template
- `--content` string (required) — Template content to render.
- `--incident-id` string — Incident ID whose data is used to render the template; mock data is used when omitted. A MongoDB ObjectID hex string.
- `--type` string (required) — Template channel type that selects the rendering engine.

### update <template-id>
Update a template
- `--description` string — Free-form description. Up to 500 characters. (≤500 chars)
- `--dingtalk` string — DingTalk robot message template source.
- `--dingtalk-app` string — DingTalk app message template source.
- `--email` string — Email body template source (Go 'html/template' syntax).
- `--feishu` string — Feishu robot message template source.
- `--feishu-app` string — Feishu app message template source.
- `--slack` string — Slack robot message template source.
- `--slack-app` string — Slack app message template source.
- `--sms` string — SMS template source (Go 'text/template' syntax).
- `--team-id` int64 — Team scope. 0 for account-wide.
- `--teams-app` string — Microsoft Teams app message template source.
- `--telegram` string — Telegram bot message template source.
- `<template-id>` (positional, required) string — Target template ID.
- `--template-name` string (required) — Template name. 1–39 characters. (1-39 chars)
- `--voice` string — Voice call script template source.
- `--wecom` string — WeCom robot message template source.
- `--wecom-app` string — WeCom app message template source.
- `--zoom` string — Zoom bot message template source.

### validate
Validate and preview a template
- `--channel` string
- `--file` string
- `--incident` string

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
- **`update` replaces every channel field you pass — omitted channel flags are left unchanged** (server behavior: only supplied fields overwrite). Always pass `--template-name` even if the name is unchanged — it is required on update.
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
