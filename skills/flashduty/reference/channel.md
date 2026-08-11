# fduty channel — command card

Prereq: `SKILL.md` read. Read verbs are free; `create`, `update`, `delete`, `disable`, `enable` mutate state — confirm before acting. `delete` is **irreversible**.

## Route here when

"协作空间 / 频道 / 渠道 / 告警分组 / channel / collaboration space / alert grouping / flapping" → **channel**, NOT `incident` (incidents live _inside_ a channel) or `alert` (alerts are routed _into_ a channel). **`协作空间` (collaboration space) IS the `channel` API noun** — a naive translation would be "频道", but Flashduty's product surfaces it as 协作空间. Key ID: **`channel-id` (int)** from `channel list`.

Rules INSIDE a channel have their own cards: escalation / 分派策略 → `reference/escalation.md`; silence / inhibit / drop (静默 / 抑制 / 丢弃 / 降噪) → `reference/noise.md`.

**Flashcat workspace exception.** When the user asks whether a "空间" is healthy, red/green, or specifically mentions **灭火图 / firemap**, do not assume they mean a Flashduty channel. In that context, "空间" may be a **Flashcat workspace**, and the answer must come from the Flashcat/firemap surface rather than channel incident stats. If you first resolved a name as a Flashduty `channel-id` and later resolve the same visible name as a Flashcat `workspace-id`, **do not silently switch** — tell the user these are different objects and state which ID/surface each conclusion uses.

## Intent → verb

| want | verb |
|---|---|
| list all channels (with team/name filter) | `list` |
| channel detail | `info <channel-id>` |
| batch fetch channels | `infos <channel-id> [id2 ...]` |
| create a channel | `create` |
| rename / reconfigure a channel | `update <channel-id>` |
| disable / re-enable a channel | `disable <channel-id>` / `enable <channel-id>` |
| delete a channel | `delete <channel-id>` |
| escalation rules (分派策略) | `reference/escalation.md` |
| silence / inhibit / drop rules (降噪) | `reference/noise.md` |

## Hot flow — create a channel

```bash
# 1. find owning team-id (from `fduty team list --output-format toon`)
fduty channel list --output-format toon
# 2. create the channel (no positional; --channel-name and --team-id are required)
fduty channel create --channel-name "production-api" --team-id <team-id> \
  --auto-resolve-timeout 3600 --auto-resolve-mode trigger
# → returns channel_id; next, add an escalation rule so incidents page someone:
#   see reference/escalation.md
```

<!-- GENERATED:channel START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### create
Create channel
- `--auto-resolve-mode` string — Auto-resolve timer reset mode. · enum: trigger | update
- `--auto-resolve-timeout` int64 — Auto-resolve timeout in seconds. 0 disables auto-resolve. Max 30 days. (0-2592000)
- `--channel-name` string (required) — Channel name. 1 to 59 characters. (1-59 chars)
- `--description` string — Free-form description. Up to 500 characters. (≤500 chars)
- `--disable-auto-close` bool — Disable automatic incident closing.
- `--disable-outlier-detection` bool — Disable outlier incident detection.
- `--is-external-report-enabled` bool — Allow external reporters to file incidents into this channel.
- `--is-private` bool — When true, the channel is visible only to its managing teams.
- `--managing-team-ids` intSlice — Additional teams that can manage the channel. Up to 3 entries.
- `--plugin-ids` intSlice — IDs of plugins (integrations) subscribed to this channel.
- `--team-id` int64 (required) — Owning team ID.
- body-only (`--data`): escalate_rule (object); flapping (object); group (object)
- response: single object (`data` unwrapped to the top level) — fields: channel_id (integer); channel_name (string); external_report_token (string)

### delete <channel-id>
Delete channel
- `<channel-id>` (positional, required) int64 — Channel ID.

### disable <channel-id>
Disable channel
- `<channel-id>` (positional, required) int64 — Channel ID.

### enable <channel-id>
Enable channel
- `<channel-id>` (positional, required) int64 — Channel ID.

### info <channel-id>
Get channel detail
- `<channel-id>` (positional, required) int64 — Channel ID to fetch.
- response: single object (`data` unwrapped to the top level) — fields: account_id (integer); active_incident_highest_severity (string); auto_resolve_mode (string); auto_resolve_timeout (integer); channel_id (integer); channel_name (string); created_at (integer); creator_id (integer); creator_name (string); deleted_at (integer); description (string); disable_auto_close (boolean); disable_outlier_detection (boolean); external_report_token (string); flapping (object); group (object); is_external_report_enabled (boolean); is_private (boolean); is_starred (boolean); last_incident_at (integer); managing_team_ids (array<integer>); progress_to_incident_cnts (object); status (string); team_id (integer); team_name (string); updated_at (integer)

### infos <channel-id> [<id2>...]
Batch get channels
- `<channel-ids>` (positional, required) intSlice — Channel IDs to look up. Up to 1000.
- response: `{items: [...]}` page wrapper — pipe `--json | jq '.items[]'` (NOT top-level `.[]`) — items fields: channel_id (integer); channel_name (string); status (string)

### list
List channels
- `--name` string
- `--team-ids` int64Slice
- response: TOP-LEVEL array — pipe `--json | jq '.[]'` (NOT `.items[]`) — fields: account_id (integer); active_incident_highest_severity (string); auto_resolve_mode (string); auto_resolve_timeout (integer); channel_id (integer); channel_name (string); created_at (integer); creator_id (integer); creator_name (string); deleted_at (integer); description (string); disable_auto_close (boolean); disable_outlier_detection (boolean); external_report_token (string); flapping (object); group (object); is_external_report_enabled (boolean); is_private (boolean); is_starred (boolean); last_incident_at (integer); managing_team_ids (array<integer>); progress_to_incident_cnts (object); status (string); team_id (integer); team_name (string); updated_at (integer)

### update <channel-id>
Update channel
- `--auto-resolve-mode` string — Auto-resolve timer reset mode. · enum: trigger | update
- `--auto-resolve-timeout` int64 — Auto-resolve timeout in seconds. 0 disables auto-resolve. Max 30 days. (0-2592000)
- `<channel-id>` (positional, required) int64 — Channel ID to update.
- `--channel-name` string — New channel name. 1 to 59 characters. (1-59 chars)
- `--description` string — New description. Up to 500 characters. (≤500 chars)
- `--disable-auto-close` bool — Disable automatic incident closing.
- `--disable-outlier-detection` bool — Disable outlier incident detection.
- `--is-external-report-enabled` bool — Allow external reporters to file incidents into this channel.
- `--is-private` bool — When true, the channel is visible only to its managing teams.
- `--managing-team-ids` intSlice — Additional teams that can manage the channel. Up to 3 entries.
- `--team-id` int64 — New owning team ID.
- body-only (`--data`): flapping (object); group (object)
- response: single object (`data` unwrapped to the top level) — fields: external_report_token (string)

<!-- GENERATED:channel END -->

## Key concepts

- **`--auto-resolve-mode`** enum: `trigger` (timer resets on each new alert trigger) | `update` (timer resets on any alert update).
- **Alert grouping `group.method`**: `i` = intelligent (embedding similarity), `p` = pattern (label equality), `n` = none. **`group.time_window` is in minutes** (default cap 1440 = 24h; extended accounts may allow up to 43200 = 30 days). Set via `--data '{"group":{"method":"p","equals":[["service","env"]],"time_window":30}}'` on `create`/`update`.

Escalation, silence, inhibit and unsubscribe rules live on their own cards: `reference/escalation.md` and `reference/noise.md`.

## Gotchas

- **`channel-id` can be passed positionally or via `--channel-id` — both work** on every verb of this card (`info`, `infos`, `update`, `delete`, `disable`, `enable`). The fence heading `### verb <channel-id>` shows the shorter positional form; the flag is an equally valid alternative, and if both are given the flag value wins.
- **`channel create` requires `--channel-name` and `--team-id`** even though they are not marked `required` in the flag list — the server rejects the request without them.
- **`delete` on a channel is irreversible** — all rules within it (escalation, silence, inhibit, drop) are also removed. Confirm the `channel-id` against `list` before proceeding.
