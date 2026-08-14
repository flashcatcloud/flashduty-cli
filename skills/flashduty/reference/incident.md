# fduty incident — command card

Prereq: `SKILL.md` read. Read verbs are free. **Mutating verbs notify responders or alter state** — confirm scope first. `merge` and `remove` are **irreversible**; `remove` permanently deletes.

## Route here when

"告警 / 故障 / 事件 / 响应 / 值班 / incident / page / outage / triage / acknowledge / resolve / snooze / escalate" → **incident**, NOT `alert` (alert = deduplicated signal; incident = actionable item responders work). NOT `insight` (metrics/MTTA/MTTR). Post-mortem reports (复盘) have their own card: `reference/postmortem.md`. You need **`incident_id` (24-char MongoDB ObjectID)** for most verbs — not the 6-char `num` shown in the UI. **`detail` and `get` are the exception and accept either** (a num auto-resolves via a 30-day lookback). For any other verb, if you only have a num, use `incident info --num <num>` first.

## Intent → verb

| want | verb |
|---|---|
| list / search active incidents | `list` |
| CSV export of incidents | `fduty insight incident-export` |
| look up by 6-char UI num | `info --num <num>` |
| full detail + AI summary for a 24-char id | `detail <id>` (narrative) or `info --incident-id <id>` (same endpoint) |
| get structured data for one or more ids | `get <id> [<id2>...]` |
| contributing alerts | `alerts <id>` |
| full event history (short) | `timeline <id>` |
| paginated event history | `feed <id>` |
| past similar incidents | `similar <id>` |
| historical incidents related to this one | `past-list <incident-id>` |
| create a manual incident | `create` |
| edit title/description/severity | `update <id>` |
| edit title/description/severity/impact/root-cause/resolution | `reset <incident-id>` |
| set one custom field | `field-reset <incident-id>` |
| acknowledge (Triggered → Processing) | `ack <incident-id> [<id2>...]` |
| un-acknowledge | `unack <incident-id> [<id2>...]` |
| close | `close <id> [<id2>...]` |
| reopen | `reopen <incident-id> [<id2>...]` |
| resolve with optional note | `resolve <incident-id> [<id2>...]` |
| snooze / un-snooze | `snooze <id> [<id2>...]` / `wake <incident-id> [<id2>...]` |
| add comment | `comment <id> [<id2>...]` |
| add responder by member ID | `add-responder <id>` |
| add responders (alternate; positional person IDs) | `responder-add <person-id> [<id2>...] --incident-id <id>` |
| dispatch to an escalation level / responder | `assign --data '{"incident_id":"<id>","assigned_to":{...}}'` (body-only `assigned_to`) |
| replace responder list | `reassign <id>` |
| merge duplicates (IRREVERSIBLE) | `merge <target_id>` |
| stop auto-merging alerts in | `disable-merge <incident-id> [<id2>...]` |
| permanently delete (IRREVERSIBLE) | `remove <id> [<id2>...]` |
| post-mortem reports (复盘) | `reference/postmortem.md` |
| war room (IM chat) | `war-room-list <incident-id>` → `war-room-create <incident-id>` |
| war room (IM chat), nested subcommand form | `war-room list/create/get/add-member/default-observers/delete <id>` |

## Hot flow — triage an active incident

```bash
# 1. Find unacknowledged critical incidents (last 4h)
fduty incident list --severity Critical --progress Triggered --since 4h --fields incident_id,title,incident_severity,progress,start_time,channel_id --output-format toon

# 2. Get AI summary + full detail (use the 24-char incident_id from step 1)
fduty incident detail <incident-id> --fields incident_id,title,incident_severity,progress,ai_summary,root_cause,resolution,alert_cnt,start_time,channel_id --output-format toon

# 3. See contributing alerts
fduty incident alerts <incident-id>

# 4. Check for prior similar incidents (channel-backed only; see Gotchas)
fduty incident similar <incident-id> --limit 5 --output-format toon

# 5. Acknowledge ownership
fduty incident ack <incident-id>

# 6. Post a status comment — content goes into a file, never a shell argument
ID=<incident-id>
COMMENT_FILE=$(mktemp)
# Before running, choose a fresh delimiter that is absent as a full line in the intended comment.
cat > "$COMMENT_FILE" <<'FDUTY_COMMENT_7F3A9C2E_EOF'
Root cause identified: DB failover.
Fix deploying.
FDUTY_COMMENT_7F3A9C2E_EOF
fduty incident comment "$ID" --comment-file "$COMMENT_FILE"

# 7. Resolve with root-cause note
fduty incident resolve <incident-id> --root-cause "DB primary failover delay" --resolution "Failover completed; latency normal."
```

Projected `similar` lists stay below 16 KiB; a trailing `...` in a list row means a long retained string was shortened. `detail --fields` is different: it never shortens values — the projection must fit within 8 KiB as requested or the command fails and names the largest fields, so drop some fields (or drop `--fields` for the full unbounded detail) and retry.

`comment` never accepts the text as a command-line argument — only `--comment-file <path>` (or `--comment-file -` to read stdin), so backticks/`$()`/quotes inside the comment are inert. The command also reads back every target's timeline after writing and exits non-zero unless it finds an entry matching what it sent, so `Commented on ...` is proof of content fidelity, not just acceptance — no separate manual read-back is needed. Leading and trailing whitespace is stripped before sending (the server strips it too, so this is what gets stored); everything else, including interior blank lines, is preserved exactly.

> `incident list --output-format json|toon` defaults to the compact row projection `incident_id,title,incident_severity,progress,start_time,channel_id`. Pass `--fields incident_id,title,channel_id,start_time` when you need different list columns; use `incident detail <id>` / `incident get <id>` for full incident records. Any list-response field — including `labels` — is selectable this way (a key missing from the output means it wasn't selected, NOT that the server omits it; the command prints a stderr note when the default projection applies). The one exception is `alerts`: neither list nor detail responses ever fill it — use `incident alerts <id>` for an incident's alerts. Wide fields over many rows can exceed the 16 KiB structured-output bound and the command errors with "request fewer rows or fields" — lower `--limit`/page through, or use `insight` aggregates for distributions instead of dumping labels row by row.

## Hot flow — full fault analysis (read-only summary)

When asked to **summarize / analyze** an incident — 详情 + 关联告警 + 变更 + 时间线 + 相似故障 + 复盘 — `incident detail` does **not** contain the alerts / timeline / similar / post-mortem / change data; each is its own command. **Your first action must be the bundled script** — do not hand-pick one or two commands and write the rest from memory. One call fetches all six aspects:

```bash
bash <skill-dir>/scripts/incident-summary.sh <incident-id>
```

`<skill-dir>` is this skill's base directory — you were given it when the skill loaded (it is also the folder you read this card from). The script runs every command below and prints the results in one block, so each section of your summary is backed by real output and there is nothing to guess. (To tie post-mortems to *this* incident, re-run `incident post-mortem-list --channel-ids <channel-id>` with the `channel_id` from `detail`.)

If you fetch the pieces by hand instead, run **all six** — they are cheap reads:

```bash
ID=<incident-id>                                          # 24-char id from `incident list`
fduty incident detail   "$ID" --fields incident_id,title,incident_severity,progress,ai_summary,root_cause,resolution,alert_cnt,start_time,channel_id --output-format toon  # ① 详情 + AI summary + alert counts + channel
fduty incident alerts   "$ID"                             # ② contributing alerts (detail's embedded alerts are empty here)
fduty incident timeline "$ID"                             # ④ timeline  (or `incident feed "$ID"` for the paginated view)
fduty incident similar  "$ID" --limit 5 --output-format toon          # ⑤ similar past incidents (channel-backed; see Gotchas; compact by default)
fduty incident post-mortem-list --channel-ids <channel-id> # ⑥ post-mortems for this incident's channel (verb card: reference/postmortem.md)
fduty change list --since 24h                              # ③ correlated changes — by shared labels + time; see reference/change.md
```

> **Never report a result you didn't fetch.** Do not write "返回空" / "无" / a count for any aspect whose command is **absent from your tool-call history this turn** — write `未查询 — 可运行 <command>` instead. "Empty" is a claim only a command you actually ran can make; inventing it is the worst failure mode of a fault summary.

## Hot flow — resolve, document, and merge duplicates

```bash
# Merge two duplicate incidents into a primary (IRREVERSIBLE — confirm first)
fduty incident merge <primary-incident-id> --source <dup1-id>,<dup2-id>

# Record post-incident narrative on the primary
fduty incident reset <primary-incident-id> \
  --root-cause "Redis OOM on shard-3" \
  --impact "Checkout latency P99 >5s for 12 min" \
  --resolution "Increased memory limit; deployed hot patch"

# Review the event timeline
fduty incident timeline <primary-incident-id>
```

<!-- GENERATED:incident START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### ack <incident-id> [<id2>...]
Acknowledge incident
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to acknowledge. At most 100 per call.
- `--summary` string — Form summary recorded as a timeline comment. Accepted only when the acknowledgement form contains a summary element.
- body-only (`--data`): custom_fields (object); images (array<object>)

### add-responder <id>
Add responders to an incident
- `--follow-preference` bool
- `--notify-channel` string
- `--person` string
- `--template-id` string

### alert-list <incident-id>
List alerts of incident
- `<incident-id>` (positional, required) string — Incident ID (MongoDB ObjectID).
- `--include-events` bool — When true, include at most the 20 newest raw events in each alert item as a preview.
- `--is-active` bool — When true return only active alerts (Critical/Warning/Info); when false return only recovered alerts (Ok). Omit to include all.
- `--limit` int64 — Page size, at most 1000. (0-1000)
- `--page` int64 — Page number starting at 1. (min 0)
- `--search-after-ctx` string
- response: `{items: [...], total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); alert_id (string); alert_key (string); alert_severity (string); alert_status (string); channel_id (integer); channel_name (string); channel_status (string); created_at (string); data_source_id (integer); data_source_name (string); data_source_ref_id (string); data_source_type (string); deleted_at (string); description (string); end_time (string); event_cnt (integer); events (array<object>); ever_muted (boolean); images (array<object>); incident (object); integration_id (integer); integration_name (string); integration_ref_id (string); integration_type (string); labels (object); last_time (string); responder_email (string); responder_name (string); start_time (string); title (string); title_rule (string); updated_at (string)

### alerts <id>
View incident alerts
- `--limit` int
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); alert_id (string); alert_key (string); alert_severity (string); alert_status (string); channel_id (integer); channel_name (string); channel_status (string); created_at (string); data_source_id (integer); data_source_name (string); data_source_ref_id (string); data_source_type (string); deleted_at (string); description (string); end_time (string); event_cnt (integer); events (array<object>); ever_muted (boolean); images (array<object>); incident (object); integration_id (integer); integration_name (string); integration_ref_id (string); integration_type (string); labels (object); last_time (string); responder_email (string); responder_name (string); start_time (string); title (string); title_rule (string); updated_at (string)

### assign
Assign incident
- `--incident-id` string — Single incident ID. Ignored when 'incident_ids' is also provided.
- `--incident-ids` stringSlice — Incident IDs to assign in bulk; obtain them from 'POST /incident/list'.
- body-only (`--data`): assigned_to (object) (required)

### close <id> [<id2> ...]
Close incidents

### comment <id> [<id2> ...]
Add a comment to incident timelines
- `--comment-file` string
- `--mute-reply` bool

### comment-type-create
Create a comment type
- `--color` string (required) — Label color as a hex value in #RRGGBB format. Normalized to uppercase.
- `--name` string (required) — Display name. Trimmed before storing; must be unique within the account (case-insensitive). At most 40 characters. (≤40 chars)
- response: single object (`data` unwrapped to the top level) — fields: comment_type_id (string); item (object)

### comment-type-delete <comment-type-id>
Delete a comment type
- `<comment-type-id>` (positional, required) string — ID of the comment type to delete (24-character hex ObjectID).

### comment-type-list
List comment types
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); color (string); comment_type_id (string); created_at (string); creator_id (integer); name (string); position (integer); updated_at (string); updated_by (integer)

### comment-type-reorder <comment-type-id> [<id2>...]
Reorder comment types
- `<comment-type-ids>` (positional, required) stringSlice — IDs of every comment type of the account in the desired order (24-character hex ObjectIDs).

### comment-type-update <comment-type-id>
Update a comment type
- `--color` string — New label color as a hex value in #RRGGBB format. Normalized to uppercase.
- `<comment-type-id>` (positional, required) string — ID of the comment type to update (24-character hex ObjectID).
- `--name` string — New display name. Trimmed before storing; must be unique within the account (case-insensitive). At most 40 characters. (≤40 chars)

### create
Create a new incident
- `--assign` intSlice
- `--channel` int64
- `--description` string
- `--severity` string
- `--title` string

### custom-action-do
Execute custom action
- `--incident-id` string (required) — Incident ID (MongoDB ObjectID).
- `--integration-id` int64 (required) — Custom action integration ID. Must be enabled and associated with the incident's channel.
- response: single object (`data` unwrapped to the top level) — fields: message (string)

### detail <id>
View full incident detail with AI summary
- `--fields` string
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); account_locale (string); account_name (string); account_time_zone (string); ack_time (string); active_alert_cnt (integer); ai_summary (string); alert_cnt (integer); alert_event_cnt (integer); alerts (array<object>); assigned_to (object); channel_id (integer); channel_name (string); channel_status (string); close_time (string); closer (object); closer_id (integer); created_at (string); creator (object); creator_id (integer); data_source_id (integer); data_source_ids (array<integer>); data_source_type (string); data_source_types (array<string>); dedup_key (string); deleted_at (string); description (string); detail_url (string); end_time (string); equals_md5 (string); ever_muted (boolean); fields (object); frequency (string); group_method (string); images (array<object>); impact (string); incident_id (string); incident_severity (string); incident_status (string); integration_id (integer); integration_ids (array<integer>); integration_type (string); integration_types (array<string>); labels (object); last_time (string); links (array<object>); manual_overrides (array<string>); num (string); owner (object); owner_id (integer); post_mortem_id (string); progress (string); reporter_email (string); resolution (string); responders (array<object>); root_cause (string); silence_url (string); snoozed_before (string); start_time (string); title (string); updated_at (string)

### disable-merge <incident-id> [<id2>...]
Disable incident merge
- `<incident-ids>` (positional, required) stringSlice — Incident IDs whose automatic merge should be disabled.

### feed <id>
View incident feed (paginated timeline)
- `--limit` int
- `--page` int
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); created_at (string); creator_id (integer); deleted_at (string); detail (object); ref_id (string); type (string); updated_at (string)

### field-reset <incident-id>
Update incident custom field
- `--field-name` string (required) — Custom field name; must match a field defined on the account.
- `<incident-id>` (positional, required) string — Incident ID (MongoDB ObjectID).
- body-only (`--data`): field_value (any)

### get <id> [<id2> ...]
Get incident details
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); account_locale (string); account_name (string); account_time_zone (string); ack_time (string); active_alert_cnt (integer); ai_summary (string); alert_cnt (integer); alert_event_cnt (integer); alerts (array<object>); assigned_to (object); channel_id (integer); channel_name (string); channel_status (string); close_time (string); closer (object); closer_id (integer); created_at (string); creator (object); creator_id (integer); data_source_id (integer); data_source_ids (array<integer>); data_source_type (string); data_source_types (array<string>); dedup_key (string); deleted_at (string); description (string); detail_url (string); end_time (string); equals_md5 (string); ever_muted (boolean); fields (object); frequency (string); group_method (string); images (array<object>); impact (string); incident_id (string); incident_severity (string); incident_status (string); integration_id (integer); integration_ids (array<integer>); integration_type (string); integration_types (array<string>); labels (object); last_time (string); links (array<object>); manual_overrides (array<string>); num (string); owner (object); owner_id (integer); post_mortem_id (string); progress (string); reporter_email (string); resolution (string); responders (array<object>); root_cause (string); silence_url (string); snoozed_before (string); start_time (string); title (string); updated_at (string)

### info [<incident-id>]
Get incident detail
- `--incident-id` string — Incident ID (MongoDB ObjectID).
- `--num` string — Short incident ID (the 6-character uppercased id shown in the UI). Not unique — resolves to the most recent match. Supply either incident_id or num.
- response: same shape as `detail <id>` above

### list
List incidents
- `--channel-id` int64
- `--fields` string
- `--limit` int
- `--nums` string
- `--page` int
- `--progress` string
- `--query` string
- `--severity` string
- `--since` string
- `--until` string
- response: same shape as `get <id> [<id2> ...]` above

### list-by-ids <incident-id> [<id2>...]
List incidents by IDs
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to query; obtain them from 'POST /incident/list'.
- response: `{items: [...], has_next_page, search_after_ctx, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); account_locale (string); account_name (string); account_time_zone (string); ack_time (string); active_alert_cnt (integer); ai_summary (string); alert_cnt (integer); alert_event_cnt (integer); alerts (array<object>); assigned_to (object); channel_id (integer); channel_name (string); channel_status (string); close_time (string); closer (object); closer_id (integer); created_at (string); creator (object); creator_id (integer); data_source_id (integer); data_source_ids (array<integer>); data_source_type (string); data_source_types (array<string>); dedup_key (string); deleted_at (string); description (string); detail_url (string); end_time (string); equals_md5 (string); ever_muted (boolean); fields (object); frequency (string); group_method (string); images (array<object>); impact (string); incident_id (string); incident_severity (string); incident_status (string); integration_id (integer); integration_ids (array<integer>); integration_type (string); integration_types (array<string>); labels (object); last_time (string); links (array<object>); manual_overrides (array<string>); num (string); owner (object); owner_id (integer); post_mortem_id (string); progress (string); reporter_email (string); resolution (string); responders (array<object>); root_cause (string); silence_url (string); snoozed_before (string); start_time (string); title (string); updated_at (string)

### merge <target_id>
Merge incidents into a target incident
- `--source` string

### past-list <incident-id>
List past incidents
- `<incident-id>` (positional, required) string — Reference incident ID (MongoDB ObjectID).
- `--limit` int64 — Maximum number of similar incidents to return. (0-100)
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); account_locale (string); account_name (string); account_time_zone (string); ack_time (string); active_alert_cnt (integer); ai_summary (string); alert_cnt (integer); alert_event_cnt (integer); alerts (array<object>); assigned_to (object); channel_id (integer); channel_name (string); channel_status (string); close_time (string); closer (object); closer_id (integer); created_at (string); creator (object); creator_id (integer); data_source_id (integer); data_source_ids (array<integer>); data_source_type (string); data_source_types (array<string>); dedup_key (string); deleted_at (string); description (string); detail_url (string); end_time (string); equals_md5 (string); ever_muted (boolean); fields (object); frequency (string); group_method (string); images (array<object>); impact (string); incident_id (string); incident_severity (string); incident_status (string); integration_id (integer); integration_ids (array<integer>); integration_type (string); integration_types (array<string>); labels (object); last_time (string); links (array<object>); manual_overrides (array<string>); num (string); owner (object); owner_id (integer); post_mortem_id (string); progress (string); reporter_email (string); resolution (string); responders (array<object>); root_cause (string); score (number); silence_url (string); snoozed_before (string); start_time (string); title (string); updated_at (string)

### reassign <id>
Reassign an incident to new responders
- `--person` string

### remove <id> [<id2> ...]
Permanently remove incidents
- `--force` bool

### reopen <incident-id> [<id2>...]
Reopen incident
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to reopen. At most 100 per call.
- `--reason` string — Optional reason recorded on the timeline. (≤1024 chars)

### reset <incident-id>
Update incident fields
- `--description` string — New description. (3-6144 chars)
- `--impact` string — New impact description. (3-6144 chars)
- `<incident-id>` (positional, required) string — Incident ID (MongoDB ObjectID).
- `--incident-severity` string — New severity: 'Info', 'Warning' or 'Critical' (most severe). · enum: Info | Warning | Critical
- `--resolution` string — New resolution notes. (3-6144 chars)
- `--root-cause` string — New root cause analysis. (3-6144 chars)
- `--title` string — New incident title. (3-200 chars)

### resolve <incident-id> [<id2>...]
Resolve incident
- `--description` string — New incident description, up to 6,144 characters. When set, it replaces the current description before the incident closes. (≤6144 chars)
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to resolve. At most 100 per call.
- `--resolution` string — Optional resolution note applied to every resolved incident. (≤1024 chars)
- `--root-cause` string — Optional root cause note applied to every resolved incident. (≤1024 chars)
- `--summary` string — Form summary recorded as a timeline comment. Accepted only when the resolution form contains a summary element.
- body-only (`--data`): custom_fields (object); images (array<object>)

### responder-add <person-id> [<id2>...]
Add incident responder
- `--incident-id` string (required) — Incident ID (MongoDB ObjectID).
- `<person-ids>` (positional, required) intSlice — Member IDs to add as responders.
- body-only (`--data`): notify (object)

### sdp-request-list
Get ServiceDeskPlus linked incidents
- `--asc` bool — When 'true', sort by internal record ID ascending; otherwise descending.
- `--channel-ids` intSlice — Channel IDs to filter by.
- `--end-time` string — Window end, Unix seconds. Must be greater than or equal to 'start_time'. Optional when 'incident_id' is provided. (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--incident-id` string — Flashduty incident ID. When set, the time window can be omitted. (≤64 chars)
- `--integration-id` int64 — ServiceDeskPlus integration ID. (min 0)
- `--limit` int64 — Page size. Defaults to 20; maximum 100. (0-100)
- `--page` int64 — Page number starting at 1. Ignored when 'search_after_ctx' is set. (min 0)
- `--request-id` string — ServiceDeskPlus request ID. (≤64 chars)
- `--search-after-ctx` string — Pagination cursor: leave empty for the first page, then pass the 'search_after_ctx' returned by the previous response.
- `--since` string
- `--start-time` string — Window start, Unix seconds. Optional when 'incident_id' is provided. (min 0) Accepts a duration (7d, 24h), '+7d' for the future, 'now', a date, or Unix seconds.
- `--status` string — Filter by sync status: 'success' or 'failed'. · enum: success | failed
- `--until` string
- response: `{items: [...], has_next_page, search_after_ctx, total}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: channel_id (integer); channel_name (string); created_at (string); error_message (string); incident_id (string); incident_title (string); integration_id (integer); request_id (string); request_link (string); status (string)

### similar <id>
Find similar incidents
- `--fields` string
- `--limit` int
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); account_locale (string); account_name (string); account_time_zone (string); ack_time (string); active_alert_cnt (integer); ai_summary (string); alert_cnt (integer); alert_event_cnt (integer); alerts (array<object>); assigned_to (object); channel_id (integer); channel_name (string); channel_status (string); close_time (string); closer (object); closer_id (integer); created_at (string); creator (object); creator_id (integer); data_source_id (integer); data_source_ids (array<integer>); data_source_type (string); data_source_types (array<string>); dedup_key (string); deleted_at (string); description (string); detail_url (string); end_time (string); equals_md5 (string); ever_muted (boolean); fields (object); frequency (string); group_method (string); images (array<object>); impact (string); incident_id (string); incident_severity (string); incident_status (string); integration_id (integer); integration_ids (array<integer>); integration_type (string); integration_types (array<string>); labels (object); last_time (string); links (array<object>); manual_overrides (array<string>); num (string); owner (object); owner_id (integer); post_mortem_id (string); progress (string); reporter_email (string); resolution (string); responders (array<object>); root_cause (string); score (number); silence_url (string); snoozed_before (string); start_time (string); title (string); updated_at (string)

### snooze <id> [<id2> ...]
Snooze incidents
- `--duration` string

### timeline <id>
View incident timeline
- response: same shape as `feed <id>` above

### unack <incident-id> [<id2>...]
Unacknowledge incident
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to unacknowledge. At most 100 per call.

### update <id>
Update an incident
- `--description` string
- `--field` stringArray
- `--severity` string
- `--title` string

### wake <incident-id> [<id2>...]
Wake incident
- `<incident-ids>` (positional, required) stringSlice — Incident IDs to wake. At most 100 per call.

### add-member <chat_id>
Add members to an incident war room
- `--integration` int64
- `--member` string

### create <incident_id>
Create an incident war room
- `--add-observers` bool
- `--integration` int64
- `--member` string

### default-observers <incident_id>
Preview historical responders for war-room observer invitation
- response: single object (`data` unwrapped to the top level) — fields: observers (array<object>)

### delete <incident_id>
Delete an incident war room
- `--force` bool
- `--integration` int64

### get <chat_id>
Get incident war room details
- `--integration` int64
- response: single object (`data` unwrapped to the top level) — fields: chat_id (string); chat_name (string); share_link (string)

### list <incident_id>
List incident war rooms
- `--integration` int64
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); chat_id (string); created_at (string); created_by (integer); incident_id (string); integration_id (integer); plugin_type (string); status (string)

### war-room-add-member <chat-id>
Add war-room member
- `<chat-id>` (positional, required) string — Chat ID of the war room within the IM platform.
- `--integration-id` int64 (required) — ID of the IM integration hosting the war room; obtain it from 'POST /datasource/im/war-room-enabled/list'.
- `--member-ids` intSlice (required) — Person IDs to add to the war room.

### war-room-create
Create war room
- `--add-observers` bool — When true, also add historical responders of the incident as observers.
- `--incident-id` string (required) — Incident ID (MongoDB ObjectID).
- `--integration-id` int64 (required) — IM integration ID. Must have war room enabled; Feishu, DingTalk, WeCom (self-built), Slack and Teams are supported.
- `--member-ids` intSlice — Additional member IDs to add to the war room.
- response: same shape as `get <chat_id>` above

### war-room-default-observers <incident-id>
Get war-room default observers
- `<incident-id>` (positional, required) string — Incident ID, a MongoDB ObjectID hex string.
- response: same shape as `default-observers <incident_id>` above

### war-room-delete
Delete war room
- `--incident-id` string (required) — Incident ID (MongoDB ObjectID).
- `--integration-id` int64 (required) — IM integration ID; obtain it from 'POST /datasource/im/war-room-enabled/list'.

### war-room-detail <chat-id>
Get war room detail
- `<chat-id>` (positional, required) string — Chat ID of the IM group hosting the war room; obtain it from 'POST /incident/war-room/list'.
- `--integration-id` int64 (required) — IM integration ID that hosts the war room.
- response: same shape as `get <chat_id>` above

### war-room-list <incident-id>
List war rooms
- `<incident-id>` (positional, required) string — Incident ID (MongoDB ObjectID).
- `--integration-id` int64 — Optional filter: only return war rooms for this IM integration.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: account_id (integer); chat_id (string); created_at (string); created_by (integer); incident_id (string); integration_id (integer); plugin_type (string); status (string)

### work-item-assignees-reset <work-item-id>
Reset work item assignees
- `--assignee-ids` intSlice — New assignee member IDs, replacing the current set. An empty array clears all assignees.
- `--version` int64 (required) — Current item version for optimistic locking. Must match the stored version.
- `<work-item-id>` (positional, required) string — Work item ID (opaque string, max 128 characters). (≤128 chars)
- response: single object (`data` unwrapped to the top level) — fields: added_assignee_ids (array<integer>); idempotent_replay (boolean); item (object); removed_assignee_ids (array<integer>)

### work-item-complete <work-item-id>
Complete a work item
- `--idempotency-key` string (required) — Client-generated idempotency key (max 128 characters; letters, digits, '_', '-', '.', ':' only). (≤128 chars)
- `--target-status` string (required) — Client-defined status to set (max 64 characters). There is no fixed state machine. (≤64 chars)
- `--version` int64 (required) — Current item version for optimistic locking. Must match the stored version.
- `<work-item-id>` (positional, required) string — Work item ID (opaque string, max 128 characters). (≤128 chars)
- response: same shape as `work-item-assignees-reset <work-item-id>` above

### work-item-convert <work-item-id>
Convert a work item to a follow-up
- `--idempotency-key` string (required) — Client-generated idempotency key (max 128 characters; letters, digits, '_', '-', '.', ':' only). (≤128 chars)
- `--target-status` string — Optional client-defined status to set on the converted follow-up (max 64 characters). (≤64 chars)
- `--version` int64 (required) — Current item version for optimistic locking. Must match the stored version.
- `<work-item-id>` (positional, required) string — Work item ID (opaque string, max 128 characters). (≤128 chars)
- response: same shape as `work-item-assignees-reset <work-item-id>` above

### work-item-create <incident-id>
Create a work item
- `--assignee-ids` intSlice — Initial assignee member IDs. Assignees must be active members who can already read the anchor; assignment never grants access.
- `--description` string — Optional longer description (max 65,535 characters). (≤65535 chars)
- `--idempotency-key` string (required) — Client-generated idempotency key (max 128 characters; letters, digits, '_', '-', '.', ':' only). (≤128 chars)
- `<incident-id>` (positional, required) string — Incident ID (MongoDB ObjectID) the item is anchored to.
- `--item-type` string (required) — 'action' anchors to an active incident and must not set 'post_mortem_id'; 'follow_up' requires 'post_mortem_id'. · enum: action | follow_up
- `--post-mortem-id` string — Post-mortem ID (32-character hex string). Required for 'follow_up', forbidden for 'action'. The post-mortem must be linked to 'incident_id'.
- `--priority` string — Optional client-defined priority (max 64 characters). (≤64 chars)
- `--status` string — Optional client-defined initial status (max 64 characters). (≤64 chars)
- `--title` string (required) — Item title (max 512 characters). (≤512 chars)
- response: single object (`data` unwrapped to the top level) — fields: added_assignee_ids (array<integer>); idempotent_replay (boolean); item (object)

### work-item-delete <work-item-id>
Delete a work item
- `--version` int64 (required) — Current item version for optimistic locking. Must match the stored version.
- `<work-item-id>` (positional, required) string — Work item ID (opaque string, max 128 characters). (≤128 chars)

### work-item-list
List work items
- `--assignee-id` int64 — Restrict results to items assigned to this member ID. Listing by assignee alone requires being that assignee or an account admin.
- `--cursor` string — Pagination cursor from a previous response's 'next_cursor'.
- `--incident-id` string — Incident ID (MongoDB ObjectID). Also returns follow-ups anchored on the incident's post-mortem.
- `--item-type` string — Filter by work item type: 'action' action item, 'follow_up' post-mortem follow-up. · enum: action | follow_up
- `--limit` int64 — Page size, at most 200. Defaults to 50. (1-200)
- `--post-mortem-id` string — Post-mortem ID (32-character hex string). Returns follow-ups bound to this post-mortem.
- response: `{items: [...], has_more, idempotent_replay, next_cursor}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: assignee_ids (array<integer>); converted_at_seconds (string); converted_by (integer); created_at_seconds (string); created_by (integer); description (string); incident_id (string); item_type (string); legacy_source_id (string); post_mortem_id (string); priority (string); source_kind (string); status (string); title (string); updated_at_seconds (string); updated_by (integer); version (integer); work_item_id (string)

### work-item-post-mortem-bind
Bind work items to a post-mortem
- `--idempotency-key` string (required) — Client-generated idempotency key (max 128 characters; letters, digits, '_', '-', '.', ':' only). (≤128 chars)
- `--incident-id` string (required) — Incident ID (MongoDB ObjectID) whose converted-but-unbound follow-ups are bound.
- `--post-mortem-id` string (required) — Post-mortem ID (32-character hex string) to bind the follow-ups to.
- response: same shape as `work-item-list` above

### work-item-update <work-item-id>
Update a work item
- `--description` string — New description (max 65,535 characters). (≤65535 chars)
- `--priority` string — New client-defined priority (max 64 characters). (≤64 chars)
- `--status` string — New client-defined status (max 64 characters). (≤64 chars)
- `--title` string — New title (max 512 characters). (≤512 chars)
- `--version` int64 (required) — Current item version for optimistic locking. Must match the stored version.
- `<work-item-id>` (positional, required) string — Work item ID (opaque string, max 128 characters). (≤128 chars)
- response: same shape as `work-item-assignees-reset <work-item-id>` above

<!-- GENERATED:incident END -->

## Status / severity values

- **progress** (`--progress` filter): `Triggered` → `Processing` → `Closed`
- **severity** (`--severity` filter / `--severity` on create/update/reset): `Critical` · `Warning` · `Info`
- `ack` moves Triggered → Processing. `close`/`resolve` move any state → Closed. `reopen` moves Closed → Triggered.

## Gotchas

- **24-char `incident_id` vs 6-char `num`**: most positional-id verbs (`ack`, `close`, `resolve`, `alerts`, `timeline`, `merge`, `reassign`, `comment`, `reset`, …) require the full ObjectID. Passing a 6-char num to any of them 400s. Use `incident info --num <num>` to resolve, or `incident list --query <num>` and read `incident_id`. **Exception: `detail` and `get` accept either form** — a 6-char num auto-resolves against the last 30 days via `/incident/list`; a miss errors `no incident with short id ... in the last 30 days`, and multiple matches list full-id candidates to disambiguate.
- **`similar` only works on channel-backed incidents** (those with a real `channel_id`). Manually created incidents with no channel return HTTP 400 "Channel not found" — this is expected, not transient. Fall back to `incident list --query "<keywords>"` for text search.
- **`update` vs `reset`**: `update <id>` edits title/description/severity/custom fields. `reset <incident-id>` additionally supports `--impact`, `--root-cause`, `--resolution` (the AI narrative fields). Use `reset` for post-incident write-back.
- **If `list` returns a `total`, use it instead of page-walking.** For "how many incidents are Triggered / Processing / Closed", run one filtered `incident list --progress <bucket> ...` per bucket and read the returned `total`. Do not fetch page 1/2/3 just to derive counts the server already computed.
- **Use `--fields` to keep list scans compact.** When the goal is to identify matching incidents or collect IDs/numbers/titles, project only the needed columns first, then fetch one target incident with `detail` / `alerts` / `timeline`.
- **`list` window cap**: `--since`/`--until` window must be < 31 days; `--limit` max 100. Empty result is authoritative — do not widen filters or retry.
- **`merge` is irreversible**: source incidents are absorbed into target permanently. Always list and confirm both IDs before running.
- **`remove --force`** bypasses the interactive confirmation prompt — never pass `--force` unless the user has explicitly said so.
- **`assign` needs `--data` for the nested `assigned_to` object** (either `person_ids` or `escalate_rule_id`). Pass member IDs from `member list` in the API field: `--data '{"incident_ids":["<id>"],"assigned_to":{"person_ids":[101]}}'`. `reassign <id> --person <ids>` is simpler for direct member assignment.

## Worked example

```bash
# Start: a prod alert paged out; you have the 6-char num "A3F9B1" from Slack.
# Step 1: resolve the num to full id and get AI summary in one call.
fduty incident info --num A3F9B1 --output-format toon

# Step 2: acknowledge so teammates see it's being handled.
fduty incident ack <incident-id>

# Step 3: after fix, resolve with context.
fduty incident resolve <incident-id> \
  --root-cause "Misconfigured health-check threshold after deploy" \
  --resolution "Reverted threshold; all pods healthy."
```
