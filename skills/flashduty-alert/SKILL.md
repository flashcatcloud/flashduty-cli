---
name: flashduty-alert
version: 1.0.0
description: "Flashduty alert and alert event investigation: search, filter, and inspect alerts (Layer 1 deduplicated) and raw alert events (Layer 0 signals). Commands: alert list, get, events, timeline, merge; alert-event list. Use when drilling down from incidents to root cause alerts, tracing deduplication history, viewing alert state transitions, merging alerts into incidents, searching global alert events by severity or integration, or analyzing alert noise patterns."
metadata:
  requires:
    bins: ["flashduty"]
  cliHelp: "flashduty alert --help"
---

# flashduty-alert

**CRITICAL** — Before using this skill, read [`../flashduty-shared/SKILL.md`](../flashduty-shared/SKILL.md) for authentication, the 3-layer noise reduction model, global flags, and safety rules.

---

## Overview

This skill covers **Layer 0 (Alert Events)** and **Layer 1 (Alerts)** of the Flashduty 3-layer noise reduction model.

- **Layer 0 -- Alert Events**: Raw signals pushed by monitoring systems (Prometheus, Zabbix, Datadog, etc.) via an `integration_key`. These are immutable records of every firing/recovery signal received.
- **Layer 1 -- Alerts**: Deduplicated from Alert Events using `alert_key`. Multiple raw events with the same alert_key collapse into a single alert, incrementing its `EventCnt`.

Use this skill for **investigation** -- drilling down from incidents to their root alert signals.

---

## Quick Decision

| User wants to... | Command |
|---|---|
| Search alerts by severity/status | `alert list --severity Critical --active` |
| Inspect a specific alert | `alert get <alert_id>` |
| See raw events behind an alert | `alert events <alert_id>` |
| View alert state transitions | `alert timeline <alert_id>` |
| Correlate alerts to an incident | `alert merge <ids> --incident <id>` |
| Search all raw alert events globally | `alert-event list --since 1h` |

---

## Commands

### alert list

List alerts with filtering and pagination.

```bash
flashduty alert list [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--severity` | string | | Filter by severity: `Critical`, `Warning`, `Info` |
| `--active` | bool | false | Show active alerts only |
| `--recovered` | bool | false | Show recovered alerts only |
| `--channel` | string | | Comma-separated channel IDs |
| `--muted` | bool | false | Show ever-muted alerts only |
| `--title` | string | | Search by title keyword |
| `--since` | string | `24h` | Start time (duration or absolute) |
| `--until` | string | `now` | End time (duration or absolute) |
| `--limit` | int | 20 | Max results per page |
| `--page` | int | 1 | Page number |

**Constraint**: `--active` and `--recovered` are mutually exclusive. Specifying both produces an error.

Output columns: ID, TITLE, SEVERITY, STATUS, EVENTS, CHANNEL, STARTED.

Examples:
```bash
# Active critical alerts in the last 24 hours
flashduty alert list --severity Critical --active

# Warnings from a specific channel in the last 6 hours
flashduty alert list --severity Warning --channel 12345 --since 6h

# Search by title keyword
flashduty alert list --title "disk usage" --active
```

### alert get

Show full detail for a single alert.

```bash
flashduty alert get <alert_id>
```

Displays a vertical detail view including: ID, Title, Severity, Status, Alert Key, Channel, Integration (name and type), Event Count, Start/Last/End times, Muted status, linked Incident (ID and progress), Labels, and Description.

### alert events

List all raw alert events (Layer 0) that were deduplicated into a specific alert.

```bash
flashduty alert events <alert_id>
```

Output columns: EVENT_ID, SEVERITY, STATUS, TIME, TITLE.

This shows the **dedup history** for one alert -- how many raw signals were collapsed into it. Use this to understand event volume and timing for a single alert.

### alert timeline

View the timeline/feed for a specific alert, showing state transitions and operator actions.

```bash
flashduty alert timeline <alert_id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | 20 | Max timeline events |
| `--page` | int | 1 | Page number |

Output columns: TIME, TYPE, OPERATOR, DETAIL. Operator names are enriched (resolved to person names).

### alert merge

Merge one or more alerts into an existing incident. **This operation is IRREVERSIBLE.**

```bash
flashduty alert merge <alert_id> [<alert_id2> ...] --incident <incident_id> [--comment <text>]
```

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--incident` | string | Yes | Target incident ID |
| `--comment` | string | No | Merge comment |

Example:
```bash
flashduty alert merge abc123 def456 --incident inc789 --comment "Related disk alerts"
```

### alert-event list (global)

Search across ALL alert events globally (Layer 0). This is a separate top-level command, not a subcommand of `alert`.

```bash
flashduty alert-event list [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--severity` | string | | Filter by severity: `Critical`, `Warning`, `Info` (comma-separated) |
| `--channel` | string | | Comma-separated channel IDs |
| `--integration-type` | string | | Comma-separated integration types |
| `--since` | string | `1h` | Start time (duration or absolute) |
| `--until` | string | `now` | End time (duration or absolute) |
| `--limit` | int | 20 | Max results per page |
| `--page` | int | 1 | Page number |

Output columns: EVENT_ID, ALERT_ID, SEVERITY, STATUS, TIME, TITLE.

**Important**: The default time window is `1h`, which is shorter than `alert list`'s default of `24h`. This is intentional because raw event volume can be very high.

Example:
```bash
# All critical events in the last hour
flashduty alert-event list --severity Critical

# Events from a specific integration type in the last 30 minutes
flashduty alert-event list --integration-type Prometheus --since 30m

# Events from multiple severity levels
flashduty alert-event list --severity Critical,Warning --since 2h
```

---

## Workflows

### Workflow 1: Investigate an Incident's Root Cause

Drill down from an incident through its contributing alerts to the raw signals.

```bash
# 1. See all alerts contributing to this incident
flashduty incident alerts <incident_id>

# 2. Pick a suspicious alert and view its full detail
flashduty alert get <alert_id>

# 3. Trace the raw events that were deduplicated into this alert
flashduty alert events <alert_id>

# 4. View the alert's state transition history
flashduty alert timeline <alert_id>
```

### Workflow 2: Find Noisy Alert Sources

Identify which alerts or integrations are generating the most noise.

```bash
# 1. Find active warnings in the last 24 hours
flashduty alert list --since 24h --active --severity Warning

# 2. Check recent critical event volume (raw Layer 0 signals)
flashduty alert-event list --since 1h --severity Critical

# 3. For aggregate analysis, use the insight command (see flashduty-insight skill)
flashduty insight top-alerts --label integration_name
```

### Workflow 3: Manually Correlate Alerts to an Incident

Find related alerts and merge them into a single incident for unified response.

```bash
# 1. Find alerts matching a pattern
flashduty alert list --title "disk" --active

# 2. Merge selected alerts into an existing incident (IRREVERSIBLE)
flashduty alert merge <alert_id1> <alert_id2> --incident <incident_id> --comment "Related disk alerts"
```

---

## Key Concepts

### alert events vs alert-event list

These are different commands with different scopes:

| | `alert events <alert_id>` | `alert-event list` |
|---|---|---|
| Scope | Events for ONE specific alert | ALL events globally |
| Purpose | Dedup history of a single alert | Global raw signal search |
| Filters | None (alert_id is the filter) | Severity, channel, integration type, time |
| Default window | N/A (all events for the alert) | 1 hour |
| Use case | "How many raw events hit this alert?" | "What raw signals arrived recently?" |

### Alert States

- **Active**: The alert is currently firing. No recovery signal has been received.
- **Recovered**: A recovery signal was received (or the alert was manually resolved).
- The `--active` and `--recovered` flags on `alert list` are mutually exclusive boolean filters. Omitting both returns all alerts regardless of state.

### Muted Status

- `EverMuted` indicates whether the alert was muted at any point during its lifecycle (via noise reduction rules or manual muting).
- The `--muted` flag on `alert list` filters to alerts that have been muted at least once.

### Deduplication via alert_key

Multiple raw alert events with the same `alert_key` within a channel are deduplicated into a single alert. The alert's `EventCnt` reflects how many raw events were collapsed. Use `alert events <alert_id>` to see the individual raw events.

---

## Safety Notes

- **`alert merge` is IRREVERSIBLE.** Once alerts are merged into an incident, they cannot be separated. Always confirm the target incident ID before merging.
- **`alert-event list` defaults to a 1-hour window**, which is shorter than other commands' 24-hour default. This is by design due to potentially high raw event volume. Widen the window explicitly with `--since` if needed, but be aware of large result sets.

---

## Cross-References

| Relation | Skill | Purpose |
|----------|-------|---------|
| Prerequisites | `flashduty-shared` | Authentication, configuration, shared flags |
| Parent layer | `flashduty-incident` | Incidents contain alerts (Layer 2) |
| Analytics | `flashduty-insight` | Alert noise analytics, top-alerts aggregation |
| Rules | `flashduty-channel` | Noise reduction rules, aggregation configuration |
