---
name: flashduty-change
version: 1.0.0
description: "Flashduty change event tracking: list recent deployments, config changes, and releases; query change volume trends over time. Commands: change list, change trend. Use when correlating incidents with recent deployments, investigating whether a change caused an outage, reviewing deployment frequency (DORA metrics), or auditing changes for a specific service or channel."
metadata:
  requires:
    bins: ["flashduty"]
  cliHelp: "flashduty change --help"
---

# flashduty-change

**CRITICAL** — Before using this skill, read [`../flashduty-shared/SKILL.md`](../flashduty-shared/SKILL.md) for authentication, the 3-layer noise reduction model, global flags, and safety rules.

## Overview

Change events run on a **parallel track** alongside the 3-layer noise reduction model (Alert Event -> Alert -> Incident). They are pushed by CI/CD systems via Flashduty integrations and are **read-only** in the CLI.

Correlation with incidents is by **label matching + time proximity**, not by foreign key. Change trend data maps directly to the DORA deployment frequency metric.

## Commands

### change list

List recent change events with title, channel, status, and timestamps.

```bash
flashduty change list [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--channel` | int | | Filter by channel ID |
| `--since` | string | `24h` | Start time (relative like `2h`, `7d` or absolute) |
| `--until` | string | `now` | End time |
| `--limit` | int | `20` | Max results per page |
| `--page` | int | `1` | Page number |

Output columns: ID, TITLE, STATUS, CHANNEL, TIME.

### change trend

Show change volume over time. Useful for DORA deployment frequency metrics.

```bash
flashduty change trend [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--step` | string | `day` | Aggregation level: `day`, `week`, `month` |
| `--since` | string | `30d` | Start time |
| `--until` | string | `now` | End time |

Output columns: DATE, CHANGES, EVENTS.

## Workflows

### Correlate an Incident with Recent Changes

```bash
# 1. Get incident details — note the channel and timestamp
flashduty incident detail <incident_id>

# 2. List changes in that channel around the incident time
flashduty change list --channel <channel_id> --since 2h

# 3. Compare timestamps and labels to identify potential cause
```

### Review Deployment Frequency

```bash
# Weekly deployment volume over the past month
flashduty change trend --step week --since 30d

# Daily granularity for the past week
flashduty change trend --step day --since 7d
```

### Audit Changes for a Specific Service

```bash
# 1. Find the channel ID
flashduty channel list --name "my-service"

# 2. List all changes in the past week
flashduty change list --channel <channel_id> --since 7d
```

## Key Concepts

- Change events are pushed by CI/CD systems via Flashduty integrations — they are **read-only** in the CLI.
- Correlation with incidents is by **label matching + time proximity**, not by foreign key.
- Change trend data maps directly to the DORA deployment frequency metric.

## Cross-References

- **Prerequisites:** flashduty-shared (authentication, output formatting)
- **Related:** flashduty-incident (incidents that changes may have caused), flashduty-insight (MTTA/MTTR context alongside deployment frequency)
