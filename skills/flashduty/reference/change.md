# fduty change — command card

Prereq: `SKILL.md` read. One read-only verb. Change events are the "what changed" signal you correlate against an incident during fault analysis.

## Route here when

"变更 / 变更事件 / 变更关联 / 发布 / 部署 / change / change event / deployment / release / what changed / correlated change" → **change**. Change events carry `labels`; they correlate to incidents by **shared labels + time proximity** (there is no foreign key). For *status-page* maintenance/incident events use `status-page` instead — that is a different "change".

## Intent → verb

| want | verb |
|---|---|
| list recent change events (filter by window / channel / integration / keyword) | `list` |

## Hot flow — find changes around an incident

```bash
# Pull recent changes in the incident's window, then eyeball label/time overlap with the incident.
fduty change list --since 24h --output-format toon

# Narrow by the integration or channel that emitted them, or by a keyword:
fduty change list --since 48h --integration <integration-id> --query "deploy" --output-format toon
```

<!-- GENERATED:change START · 由 fduty __dump-commands 同步 · 勿手改 fence 内 -->

### list
List changes
- `--channel` string
- `--integration` string
- `--limit` int
- `--page` int
- `--query` string
- `--since` string
- `--until` string

<!-- GENERATED:change END -->

## Key concepts

- **Correlation is heuristic, not relational.** A change is "related" to an incident when their `labels` overlap and their timestamps are close — there is no `incident_id` on a change. Judge the overlap yourself; do not claim a causal link the data doesn't support.
- **`--integration` / `--channel`** scope to the source that emitted the change; **`--since` / `--until`** bound the window (relative like `24h`, `-1h`, `now`, or Unix seconds).

## Gotchas

- **List-only domain.** There is no `change get` / `change detail` verb — `list` (with filters) is the whole surface. Don't guess a detail verb.
- **Empty result is authoritative** — no changes in that window/scope. Report it; don't widen blindly or invent a change to explain the incident.

## Worked example

```bash
# Changes in the last 6h on a specific integration, newest first
fduty change list --since 6h --integration 5759613685214 --output-format toon
```
