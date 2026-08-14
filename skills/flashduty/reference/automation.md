# fduty automation - command card

Prereq: `SKILL.md` read. Automations create AI SRE sessions on a schedule or through an HTTP POST trigger. `create`, `update`, `delete`, and `fire` mutate or start work. If the user directly asks for that action and provides enough detail, treat it as confirmation and do not ask again.

## Route here when

"Automation / 自动化 / 定时任务 / 每天让 AI SRE 做 / weekly report / daily brief / webhook trigger / POST trigger / create an automation in chat" -> **automation**. This is for AI SRE Automations, not alert rules or notification templates.

## Intent -> verb

| want | verb |
|---|---|
| create a scheduled or HTTP POST Automation | `create` |
| list visible Automations | `list` |
| inspect one Automation | `get <rule-id>` |
| update mutable fields | `update <rule-id>` |
| delete an Automation | `delete <rule-id> --force` |
| list run history | `runs <rule-id>` |
| list preset templates | `templates` |
| test/fire an HTTP POST trigger | `fire <trigger-id>` |
| run a rule immediately, outside its schedule | `safari automation-rule-run <rule-id>` (the friendly group has no run verb; `fire` is not a substitute — it needs a configured HTTP POST trigger + token) |

## Scope and visibility

- `--team-id 0` or omitted means personal scope. `--team-id <id>` means the Automation runs as that team and creates sessions scoped to that team.
- Creation can target personal scope or any team in the account. Do not block on local team membership guesses; let the API enforce account boundaries.
- Scope is immutable after creation. The friendly `update` command intentionally has no `--team-id`; create a new Automation if the target scope must change.
- List visibility follows the backend: the caller sees Automations they created and Automations belonging to teams they can see.

## Scheduling

- Default create behavior: enabled immediately. Use `--disabled` only if the user asks for a disabled Automation.
- `create`/`update` expose no `--timezone` flag. The cron expression runs in the rule's timezone, which the server resolves from the caller's member timezone, then the account timezone (a server-side default applies when neither is set).
- Pass the user's local wall-clock time directly to `--at` or `--cron-expr` — do not convert it to UTC first. The rule already runs in the caller's own resolved timezone, so a manual UTC conversion shifts the schedule by the account's UTC offset.
- Helper schedules (times are in the rule's resolved timezone, not UTC):
  - `--schedule hourly --at 00:15` -> minute 15 of every hour.
  - `--schedule daily --at 01:30` -> every day at 01:30.
  - `--schedule weekly --weekday mon --at 02:00` -> every Monday at 02:00.
- For exact minute-level control, use `--cron-expr '<minute> <hour> <day> <month> <weekday>'` in that same local time.
- To pin a rule to a specific timezone (e.g. UTC) regardless of the caller's default, use `safari automation-rule-create --timezone <IANA tz>` instead — the curated `create`/`update` commands cannot set it, and `update` cannot change it after creation.
- HTTP POST-only rule: pass `--http-post-trigger` without schedule flags. The CLI sends a placeholder cron and disables the schedule trigger.

## Hot flow - create from chat

```bash
fduty automation create \
  --name "Daily SRE brief" \
  --team-id <team-id> \
  --schedule daily \
  --at 01:30 \
  --prompt "Summarize yesterday's incidents, noisy alerts, and follow-up risks." \
  --output-format toon
```

If the user did not specify a team, omit `--team-id` for personal scope. If the user gives a long task prompt, put it in a temp file and pass `--prompt-file <path>` to avoid shell quoting issues.

## Hot flow - create an HTTP POST trigger

```bash
fduty automation create \
  --name "Webhook triage" \
  --http-post-trigger \
  --prompt-file ./automation-prompt.md \
  --output-format toon
```

The response can include `http_post_trigger_id`, `http_post_trigger_url`, and one-time `http_post_token`. Tell the user to store the token; it cannot be retrieved later. Rotate it with:

```bash
fduty automation update <rule-id> --rotate-http-post-token --output-format toon
```

## Hot flow - exact cron

```bash
fduty automation create \
  --name "Weekday 08:05 review" \
  --cron-expr "5 8 * * 1-5" \
  --prompt "Review open incidents and alert noise before the workday." \
  --output-format toon
```

## Manage and inspect

```bash
fduty automation list --scope all --limit 20 --output-format toon
fduty automation get <rule-id> --output-format toon
fduty automation runs <rule-id> --since 7d --output-format toon

fduty automation update <rule-id> --disable --output-format toon
fduty automation update <rule-id> --enable --cron-expr "30 1 * * *" --output-format toon
fduty automation delete <rule-id> --force
```

## Fire an HTTP POST trigger

```bash
fduty automation fire <trigger-id> \
  --token "$FLASHDUTY_AUTOMATION_TRIGGER_TOKEN" \
  --text "manual validation run" \
  --output-format toon
```

The trigger API has no idempotency key: retry only when the failed call is known not to have reached the server. Do not invent a token. If it is missing, rotate the trigger token through `update` or ask the user to provide it through their secure shell/environment, not in chat.

<!-- GENERATED:automation START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### create
Create an Automation
- `--at` string
- `--cron-expr` string
- `--disabled` bool
- `--environment-id` string
- `--environment-kind` string
- `--http-post-trigger` bool
- `--name` string
- `--prompt` string
- `--prompt-file` string
- `--schedule` string
- `--schedule-enabled` bool
- `--team-id` int64
- `--weekday` string
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); can_edit (boolean); created_at (string); cron_expr (string); enabled (boolean); environment_id (string); environment_kind (string); http_post_token (string); http_post_trigger_enabled (boolean); http_post_trigger_id (string); http_post_trigger_url (string); name (string); oncall_incident_channel_ids (array<integer>); oncall_incident_severities (array<string>); oncall_incident_trigger_enabled (boolean); oncall_incident_trigger_id (string); owner_id (integer); prompt (string); rule_id (string); run_scope (string); schedule_next_fire_at_ms (string); schedule_trigger_enabled (boolean); schedule_trigger_id (string); team_id (integer); timezone (string); updated_at (string)

### delete <rule_id>
Delete an Automation
- `--force` bool

### fire <trigger_id>
Fire an Automation HTTP POST trigger
- `--text` string
- `--token` string

### get <rule_id>
Get an Automation
- response: same shape as `create` above

### list
List visible Automations
- `--enabled` bool
- `--keyword` string
- `--limit` int
- `--page` int
- `--scope` string
- `--team-ids` int64Slice
- response: single object (`data` unwrapped to the top level) — fields: rules (array<object>); total (integer)

### runs <rule_id>
List Automation runs
- `--limit` int
- `--page` int
- `--since` string
- `--status` string
- `--trigger-kind` string
- `--until` string
- response: single object (`data` unwrapped to the top level) — fields: runs (array<object>); total (integer)

### templates
List Automation templates
- `--locale` string
- response: single object (`data` unwrapped to the top level) — fields: templates (array<object>)

### update <rule_id>
Update an Automation
- `--at` string
- `--cron-expr` string
- `--disable` bool
- `--disable-http-post-trigger` bool
- `--disable-schedule` bool
- `--enable` bool
- `--enable-http-post-trigger` bool
- `--enable-schedule` bool
- `--environment-id` string
- `--environment-kind` string
- `--name` string
- `--prompt` string
- `--prompt-file` string
- `--rotate-http-post-token` bool
- `--schedule` string
- `--weekday` string
- response: same shape as `create` above

<!-- GENERATED:automation END -->

## Gotchas

- **Do not ask form-like follow-up questions** when the request is clear enough. Choose practical defaults: personal scope when no team is named, enabled on create, daily 09:00 for a vague daily schedule, Monday 09:00 for a vague weekly schedule.
- **Ask only when required data is missing**: task prompt, trigger token for `fire`, or an ambiguous target rule for update/delete.
- **`update` cannot move personal/team scope.** If the user asks to move scope, create a replacement Automation in the new scope and then delete or disable the old one after confirmation.
- **Use `--prompt-file` for long prompts.** Shell quoting is the most common failure when the prompt contains quotes, markdown, or JSON.
- **Delete is destructive.** In agent/non-interactive runs, pass `--force` only after the user has clearly asked to delete that rule.
