# flashduty automation - command card

Prereq: `SKILL.md` read. Automations run AI SRE work on a schedule or through an HTTP POST trigger. A web Automation starts a fresh web-console execution session. An ordinary Automation bound to the current IM chat starts a fresh execution session on each run and, when the IM integration creates threads, starts a new IM thread. It remains bound to the chat and scope captured at creation, not to an old session or thread. Same-run retries reuse that run's execution identity. War-room Automations keep their existing behavior. `create`, `update`, `delete`, and `fire` mutate or start work. If the user directly asks for that action and provides enough detail, treat it as confirmation and do not ask again.

When this command runs from a live IM conversation, `create` without `--run-in`
lets the server bind the Automation to that current conversation and inherit its
scope. Use `--run-in web` only when the user asks to run in the web console.
Do not invent or pass a chat, platform, integration, or session ID: the server
is the only authority that resolves the current chat. Omit `--team-id` to
inherit the IM scope; pass `--team-id 0` only for an explicit personal scope.
With `--run-in current-chat`, an explicitly supplied `--team-id` must equal
the current conversation's scope; it cannot switch that conversation to a
different personal or team scope.
If that bound chat's scope no longer matches the rule when it runs, the run is
skipped; it never moves the rule or runs in another chat.
The execution destination and pinned scope cannot be changed with `update`.

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

- `--team-id 0` explicitly selects personal scope. Omit it to inherit the authenticated IM conversation scope when available; otherwise the server applies its normal default. For `--run-in current-chat`, an explicit team ID must equal that conversation's scope; it cannot switch the chat's scope. For web execution, `--team-id <id>` selects that team.
- Creation can target personal scope or any team in the account. Do not block on local team membership guesses; let the API enforce account boundaries.
- Scope is immutable after creation. The friendly `update` command intentionally has no `--team-id`; create a new Automation if the target scope must change.
- List visibility follows the backend: the caller sees Automations they created and Automations belonging to teams they can see.

## Scheduling

- Default create behavior: enabled immediately. Use `--disabled` only if the user asks for a disabled Automation.
- If the user names a timezone, pass `--timezone` with that IANA name on `create` or `update`. The cron expression is that timezone's wall clock. Do not convert it to UTC before calling.
- `create` without `--timezone` still uses the caller's member timezone, then the account timezone, then Asia/Shanghai. Do not invent a timezone in the CLI.
- `update` without `--timezone` keeps the stored value, including a legacy empty string.
- `update` recalculates the next fire immediately. Read `schedule_next_fire_at_ms` from the response and tell the user. Do not assume the previously scheduled occurrence will still run.
- An explicit empty string (`--timezone ""`) is UTC, not the account default. On `update` that empty string is sent and stored. On `create`, omit the flag for the default chain above; do not pass an empty string to mean UTC.
- Pass the user's local wall-clock time directly to `--at` or `--cron-expr`.
- Helper schedules (times are in the rule's timezone, not UTC):
  - `--schedule hourly --at 00:15` -> minute 15 of every hour.
  - `--schedule daily --at 01:30` -> every day at 01:30.
  - `--schedule weekly --weekday mon --at 02:00` -> every Monday at 02:00.
- Custom cron uses `--cron-expr '<minute> <hour> <day> <month> <weekday>'` in that same local time. The minute must be one fixed integer from `0` through `59`; minute wildcards, ranges, lists, and steps are unsupported. The fastest supported schedule is once per hour.
- HTTP POST-only rule: pass `--http-post-trigger` without schedule flags. The CLI sends a placeholder cron and disables the schedule trigger.

## Hot flow - create from chat

```bash
flashduty automation create \
  --name "Daily SRE brief" \
  --schedule daily \
  --at 01:30 \
  --prompt "Summarize yesterday's incidents, noisy alerts, and follow-up risks." \
  --output-format toon
```

To keep an IM-originated Automation in the web console, add `--run-in web`.

If the user did not specify a team, omit `--team-id` so the server can inherit the IM scope. If the user gives a long task prompt, put it in a temp file and pass `--prompt-file <path>` to avoid shell quoting issues.

Every ordinary current-chat run starts fresh, so the task prompt must state the
goal, relevant chat context (such as the service, team, and delivery target),
and required inputs directly. Do not say "continue the discussion above" or
rely on an old session or thread.

## Hot flow - create an HTTP POST trigger

```bash
flashduty automation create \
  --name "Webhook triage" \
  --http-post-trigger \
  --prompt-file ./automation-prompt.md \
  --output-format toon
```

The response can include `http_post_trigger_id`, `http_post_trigger_url`, and one-time `http_post_token`. Tell the user to store the token; it cannot be retrieved later. Rotate it with:

```bash
flashduty automation update <rule-id> --rotate-http-post-token --output-format toon
```

## Hot flow - exact cron

```bash
flashduty automation create \
  --name "Weekday 08:05 review" \
  --cron-expr "5 8 * * 1-5" \
  --prompt "Review open incidents and alert noise before the workday." \
  --output-format toon
```

## Manage and inspect

```bash
flashduty automation list --scope all --limit 20 --output-format toon
flashduty automation get <rule-id> --output-format toon
flashduty automation runs <rule-id> --since 7d --output-format toon

flashduty automation update <rule-id> --disable --output-format toon
flashduty automation update <rule-id> --enable --cron-expr "30 1 * * *" --output-format toon
flashduty automation delete <rule-id> --force
```

## Fire an HTTP POST trigger

```bash
flashduty automation fire <trigger-id> \
  --token "$FLASHDUTY_AUTOMATION_TRIGGER_TOKEN" \
  --text "manual validation run" \
  --output-format toon
```

The trigger API has no idempotency key: retry only when the failed call is known not to have reached the server. Do not invent a token. If it is missing, rotate the trigger token through `update` or ask the user to provide it through their secure shell/environment, not in chat.

<!-- GENERATED:automation START · 由 flashduty __dump-commands 同步 · 勿手改 fence 内 -->

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
- `--run-in` string
- `--schedule` string
- `--schedule-enabled` bool
- `--team-id` int64
- `--timezone` string
- `--weekday` string
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); can_edit (boolean); created_at (string); cron_expr (string); enabled (boolean); environment_id (string); environment_kind (string); execution_location (object); execution_place (string); http_post_token (string); http_post_trigger_enabled (boolean); http_post_trigger_id (string); http_post_trigger_url (string); name (string); oncall_incident_channel_ids (array<integer>); oncall_incident_severities (array<string>); oncall_incident_trigger_enabled (boolean); oncall_incident_trigger_id (string); owner_id (integer); prompt (string); rule_id (string); run_scope (string); schedule_next_fire_at_ms (string); schedule_trigger_enabled (boolean); schedule_trigger_id (string); team_id (integer); timezone (string); updated_at (string)

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
- `--timezone` string
- `--weekday` string
- response: same shape as `create` above

<!-- GENERATED:automation END -->

## Gotchas

- **Do not ask form-like follow-up questions** when the request is clear enough. Choose practical defaults: personal scope when no team is named, enabled on create, daily 09:00 for a vague daily schedule, Monday 09:00 for a vague weekly schedule.
- **An explicit empty `--timezone ""` is UTC, not the account default.** On `update` that value is sent and stored. Omitting the flag on `update` keeps the stored timezone, including a legacy empty string. Omitting it on `create` uses member, then account, then Asia/Shanghai. After `update`, read `schedule_next_fire_at_ms` and tell the user; the previous occurrence is not guaranteed to run.
- **Ask only when required data is missing**: task prompt, trigger token for `fire`, or an ambiguous target rule for update/delete.
- **`update` cannot move personal/team scope.** If the user asks to move scope, create a replacement Automation in the new scope and then delete or disable the old one after confirmation.
- **Use `--prompt-file` for long prompts.** Shell quoting is the most common failure when the prompt contains quotes, markdown, or JSON.
- **Delete is destructive.** In agent/non-interactive runs, pass `--force` only after the user has clearly asked to delete that rule.
