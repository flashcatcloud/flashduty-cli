# fduty route — command card

Prereq: `SKILL.md` read. `upsert` is a **full replacement** of the rule — it overwrites all cases; always read first and pass `--version` for optimistic concurrency.

## Route here when

"路由规则 / 告警路由 / 集成路由 / 分派到频道 / route rule / alert routing / integration routing / which channel gets alerts" → **route**. Key IDs needed:
- **`integration-id`** (int) — the integration the rule belongs to; get it from `fduty channel list` (channel detail carries its integrations) or from the Flashduty console.
- **`channel-id`** (int) — the target channel to route matched alerts to; same source.

Do NOT use `route` for scheduling (→ `schedule`), templates (→ `template`), or channel management (→ `channel`).

## Intent → verb

| want | verb |
|---|---|
| read the rule for one integration | `info` |
| read rules for multiple integrations at once | `list` |
| create a rule / update an existing rule | `upsert` |

## Hot flow — read then upsert a routing rule

```bash
# 1. Read the current rule; note the returned `version` field for concurrency control.
fduty route info <integration-id> --output-format toon

# 2. Upsert: route critical alerts to channel 101, all others to channel 102 (default).
#    Pass the `version` from step 1 to prevent races.
fduty route upsert <integration-id> --version <version> \
  --data '{
    "cases": [
      {
        "if": [{"key": "alert_severity", "oper": "IN", "vals": ["Critical"]}],
        "channel_ids": [101],
        "fallthrough": false
      }
    ],
    "default": {"channel_ids": [102]}
  }'

# 3. Verify
fduty route info <integration-id> --output-format toon
```

```bash
# Bulk read — check rules for several integrations at once (positional ids).
fduty route list <integration-id-1> <integration-id-2> --output-format toon
```

<!-- GENERATED:route START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### info <integration-id>
Get routing rule detail
- `<integration-id>` (positional, required) int64 — Integration ID. Must be greater than 0.

### list <integration-id> [<id2>...]
List routing rules
- `<integration-ids>` (positional, required) intSlice — Integration IDs to fetch routing rules for.

### upsert <integration-id>
Upsert routing rule
- `<integration-id>` (positional, required) int64 — Integration the rule belongs to.
- `--version` int64 — Expected current version for optimistic concurrency control. Pass the value returned by the latest read.
- body-only (`--data`): cases (array<object>); default (object); sections (array<object>)

<!-- GENERATED:route END -->

## Key concepts

- **Cases are evaluated top-to-bottom.** The first matching case wins unless `fallthrough: true`, which lets evaluation continue to the next case even after a match.
- **`routing_mode`** per case:
  - `standard` (default / empty) — routes to the fixed `channel_ids` list.
  - `name_mapping` — reads `name_mapping_label` from the alert event label map and resolves the channel by name dynamically.
- **`default` branch** — fires when no case matches (or matched cases yield no valid channels). At least one of `cases` or `default` must be provided on upsert.
- **`vals` match patterns**: literal string, wildcard (`*`, `?`), regex (`/pattern/`), CIDR (`cidr:10.0.0.0/8`), numeric comparison (`num:lt:100`).
- **Condition operator** (`oper`): `IN` (field value is in `vals`) or `NOTIN` (field value is not in `vals`).

## Gotchas

- **`upsert` is a full replacement.** It overwrites all `cases`, `default`, and `sections` for the integration. Always `info` first, reconstruct the full body, then `upsert` — never send a partial update.
- **`--version` is strongly recommended on upsert.** Omitting it skips optimistic concurrency and silently overwrites concurrent changes. Pass the `version` value from the latest `info` response.
- **`list` positional form.** `use` is `list <integration-id> [<id2>...]` — pass all integration IDs as positional arguments; the `--integration-ids` flag is also accepted but the positional form is simpler for a handful of IDs.
- **`info` returns `null` when no rule is configured** — not an error. An empty result from `list` means none of the requested integrations have a routing rule.
- **All case sub-fields via `--data`** — `cases`, `default`, `sections` are nested arrays/objects and cannot be expressed as flat flags; use `--data '{...}'` or `--data -` to pipe JSON. `--integration-id` and `--version` remain flat flags and override matching `--data` keys.
- **Sections are display-only.** `sections[].position` is an index into `cases[]` — off-by-one errors here cause a 400. They have no effect on matching logic.

## Worked example

```bash
# Read current rule for integration 5000, then add a name-mapping case for team-based routing.
fduty route info <integration-id> --output-format toon
# → note current version, e.g. 3

fduty route upsert <integration-id> --version 3 \
  --data '{
    "cases": [
      {
        "if": [{"key": "labels.team", "oper": "IN", "vals": ["*"]}],
        "routing_mode": "name_mapping",
        "name_mapping_label": "team",
        "channel_ids": [],
        "fallthrough": false
      }
    ],
    "default": {"channel_ids": [<fallback-channel-id>]}
  }'
```
