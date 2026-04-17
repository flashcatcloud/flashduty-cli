---
name: flashduty-channel
version: 1.0.0
description: "Flashduty channel and escalation rule management: list channels (collaboration spaces where alerts are routed), view escalation rules with notification layers. Commands: channel list, escalation-rule list. Use when looking up channel IDs for filtering other commands, reviewing escalation policies, or understanding how alerts are routed and escalated in the noise reduction pipeline."
metadata:
  requires:
    bins: ["flashduty"]
  cliHelp: "flashduty channel --help"
---

# flashduty-channel

**CRITICAL** — Before using this skill, read [`../flashduty-shared/SKILL.md`](../flashduty-shared/SKILL.md) for authentication, the 3-layer noise reduction model, global flags, and safety rules.

## Overview

Channels are the collaboration spaces where alerts are routed, noise reduction rules are applied, and escalation policies are configured. In the noise reduction pipeline, channels sit between Layer 1 (Alerts) and Layer 2 (Incidents) -- they are the primary container for alert routing and escalation. This skill covers read-only operations for channels and escalation rules.

## Commands

### channel list

```bash
flashduty channel list [flags]
```

**Flags:**

| Flag     | Type   | Description        |
|----------|--------|--------------------|
| `--name` | string | Search by name     |

**Output columns:** ID, NAME, TEAM, CREATOR

Lists channels with their ID, name, owning team, and creator. Channels are the collaboration spaces where alerts land after routing.

### escalation-rule list

```bash
flashduty escalation-rule list [flags]
```

**Flags:**

| Flag             | Type   | Description                        |
|------------------|--------|------------------------------------|
| `--channel`      | int    | Channel ID                         |
| `--channel-name` | string | Channel name (resolved to ID internally) |

One of `--channel` or `--channel-name` is required. When `--channel-name` matches multiple channels, the CLI prints all matches and asks the user to specify a `--channel` ID.

**Output columns:** ID, NAME, CHANNEL, STATUS, PRIORITY, LAYERS

Shows escalation rules for a channel: rule ID, name, channel name, status, priority, and the number of notification layers.

## Workflows

### Find a Channel and Its Escalation Policy

```bash
# 1. Search for the channel by name
flashduty channel list --name "Production"

# 2. List escalation rules for that channel
flashduty escalation-rule list --channel <channel_id>
```

### Look Up Channel ID for Other Commands

Many commands (`incident list`, `alert list`, `change list`, etc.) accept a `--channel` flag. Use `channel list` to find the ID first:

```bash
flashduty channel list --name "my-service"

# Then use the ID in other commands:
flashduty incident list --channel <channel_id>
flashduty alert list --channel <channel_id>
flashduty change list --channel <channel_id>
```

### Review Escalation for Incident Response

When investigating slow response times or checking that escalation is properly configured:

```bash
flashduty channel list --name "critical-service"
flashduty escalation-rule list --channel <channel_id>
# Review the LAYERS column to confirm multi-layer escalation is in place
```

## Key Concepts

- **Channels** are where the noise reduction pipeline runs (Layer 1 -> Layer 2). They are the primary container for alert routing and escalation.
- Each channel belongs to a **team**.
- **Escalation rules** define notification layers: who gets notified, when, and how. The LAYERS column shows how many escalation tiers exist.
- **Channel IDs** are used as filters across many other CLI commands (incidents, alerts, changes, etc.).
- The CLI currently provides **read-only** access to channels and escalation rules.
- Full channel CRUD and noise reduction config (silence rules, inhibit rules, drop rules, alert pipelines, routing, label enrichment) are planned for a future release.

## Cross-References

- **Prerequisites:** `flashduty-shared` -- authentication and global flags
- **Related skills:**
  - `flashduty-incident` -- incidents are created in channels
  - `flashduty-alert` -- alerts are routed to channels
  - `flashduty-oncall` -- escalation may trigger on-call schedules
