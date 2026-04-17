---
name: flashduty-insight
version: 1.0.0
description: "Flashduty analytics and SRE reporting: query MTTA/MTTR metrics by team, channel, or responder; identify noisy alert sources; review incident performance with response times; analyze notification volume and cost trends. Commands: insight team, channel, responder, top-alerts, incidents, notifications. Use when reviewing SRE performance, generating weekly or monthly incident reports, identifying noise reduction opportunities, analyzing alert fatigue, or tracking notification costs."
metadata:
  requires:
    bins: ["flashduty"]
  cliHelp: "flashduty insight --help"
---

# flashduty-insight

**CRITICAL** -- Before using this skill, read [`../flashduty-shared/SKILL.md`](../flashduty-shared/SKILL.md) for authentication, the 3-layer noise reduction model, global flags, and safety rules.

## Overview

Analytics, metrics, and reporting for Flashduty. All commands are read-only and query historical data. Use this skill when the user needs SRE performance metrics (MTTA, MTTR), noise reduction effectiveness, responder workload analysis, or notification cost trends.

## Quick Decision

| User wants to... | Command |
|---|---|
| Team-level performance overview | `insight team --since 7d` |
| Channel-level metrics | `insight channel --since 7d` |
| Individual responder stats | `insight responder --since 7d` |
| Find noisiest alert sources | `insight top-alerts --label integration_name` |
| List incidents with response metrics | `insight incidents --since 7d` |
| Notification volume/cost analysis | `insight notifications --step day` |

## Commands

### insight team

Query incident response metrics aggregated by team.

```bash
flashduty insight team [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--since` | `7d` | Start time (relative like `7d`, `24h`, or absolute) |
| `--until` | `now` | End time |

Output columns: TEAM, INCIDENTS, ACK%, MTTA, MTTR, NOISE_REDUCTION, ALERTS, EVENTS.

### insight channel

Query incident response metrics aggregated by channel.

```bash
flashduty insight channel [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--since` | `7d` | Start time |
| `--until` | `now` | End time |

Output columns: CHANNEL, INCIDENTS, ACK%, MTTA, MTTR, NOISE_REDUCTION, ALERTS, EVENTS.

### insight responder

Query incident response metrics aggregated by individual responder.

```bash
flashduty insight responder [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--since` | `7d` | Start time |
| `--until` | `now` | End time |

Output columns: RESPONDER, EMAIL, INCIDENTS, ACK%, MTTA, INTERRUPTIONS, ENGAGED.

### insight top-alerts

Show the noisiest alert sources grouped by a label. Essential for noise reduction work.

```bash
flashduty insight top-alerts [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--label` | *(required)* | Label key to group by (e.g., `integration_name`, `alert_key`) |
| `--since` | `7d` | Start time |
| `--until` | `now` | End time |
| `--limit` | `10` | Top K results to return |

Output columns: LABEL, ALERTS, EVENTS.

### insight incidents

List incidents with individual performance metrics.

```bash
flashduty insight incidents [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--since` | `7d` | Start time |
| `--until` | `now` | End time |
| `--limit` | `20` | Max results per page |
| `--page` | `1` | Page number |

Output columns: ID, TITLE, SEVERITY, CHANNEL, MTTA, MTTR, NOTIFICATIONS.

### insight notifications

Show notification volume over time. Useful for cost analysis.

```bash
flashduty insight notifications [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--step` | `day` | Aggregation period: `day`, `week`, `month` |
| `--since` | `30d` | Start time |
| `--until` | `now` | End time |

Output columns: DATE, SMS, VOICE, EMAIL.

## Workflows

### Weekly SRE Performance Review

```bash
# Overall team performance for the past week
flashduty insight team --since 7d

# Drill into channel-level metrics
flashduty insight channel --since 7d

# Individual responder metrics
flashduty insight responder --since 7d
```

### Identify and Reduce Alert Noise

```bash
# Find noisiest integration sources
flashduty insight top-alerts --label integration_name --since 7d --limit 5

# Find noisiest alert keys
flashduty insight top-alerts --label alert_key --since 7d --limit 10

# Check noise reduction effectiveness by channel
flashduty insight channel --since 7d
```

### Notification Cost Analysis

```bash
# Weekly notification trends over the past month
flashduty insight notifications --step week --since 30d

# Identify which incidents generated the most notifications
flashduty insight incidents --since 7d
```

### Monthly Incident Report

```bash
# Team-level summary for the month
flashduty insight team --since 30d

# Full incident list with metrics
flashduty insight incidents --since 30d --limit 50

# Top noisy sources over the month
flashduty insight top-alerts --label integration_name --since 30d
```

## Key Concepts

- **MTTA** -- Mean Time To Acknowledge: duration from incident trigger to first acknowledgement.
- **MTTR** -- Mean Time To Resolve: duration from incident trigger to close.
- **Noise reduction%** -- percentage of alert events that were suppressed (deduplicated/aggregated) before becoming incidents.
- **ACK%** -- percentage of incidents that were acknowledged by a responder.
- **Interruptions** -- number of times a responder was paged (notifications received).
- **Engaged time** -- total time a responder spent working on incidents.
- Times are displayed as human-readable durations (e.g., `2m 30s`, `1h 15m`).
- All insight commands are read-only -- they query historical metrics and never modify data.

## Cross-References

- **Prerequisites**: `flashduty-shared` (authentication, configuration, output formats)
- **Related skills**:
  - `flashduty-incident` -- drill into specific incidents surfaced by `insight incidents`
  - `flashduty-alert` -- investigate noisy alert sources identified by `insight top-alerts`
  - `flashduty-change` -- correlate deployment frequency via `change trend` with incident metrics
