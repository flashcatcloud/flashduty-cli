---
name: flashduty-oncall
version: 1.0.0
description: "Flashduty on-call schedule management: find who is currently on call, list schedules, view schedule details with rotation layers and shift slots. Commands: oncall who, oncall schedule list, oncall schedule get. Use when finding the current on-call responder, checking who covers a future time window, reviewing upcoming shifts, or looking up schedule IDs and rotation configurations."
metadata:
  requires:
    bins: ["flashduty"]
  cliHelp: "flashduty oncall --help"
---

# flashduty-oncall

**CRITICAL** — Before using this skill, read [`../flashduty-shared/SKILL.md`](../flashduty-shared/SKILL.md) for authentication, the 3-layer noise reduction model, global flags, and safety rules.

## Overview

On-call schedule queries for Flashduty. Find who is currently on call, list available schedules, and inspect schedule details including rotation layers and computed shift slots. All commands are read-only; schedule creation and editing are done in the Flashduty web UI.

## Commands

### oncall who

Show who is currently on call across all schedules.

```bash
flashduty oncall who [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--query` | string | | Search by schedule name |
| `--team` | string | | Comma-separated team IDs |
| `--since` | string | `now` | Start of time range |
| `--until` | string | `+24h` | End of time range |
| `--limit` | int | `20` | Max results per page |
| `--page` | int | `1` | Page number |

Output columns: SCHEDULE, ON_CALL, UNTIL, NEXT

### oncall schedule list

List all schedules with their ID, status, and layer count.

```bash
flashduty oncall schedule list [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--query` | string | | Search by schedule name |
| `--team` | string | | Comma-separated team IDs |
| `--since` | string | `now` | Start of time range |
| `--until` | string | `+24h` | End of time range |
| `--limit` | int | `20` | Max results per page |
| `--page` | int | `1` | Page number |

Output columns: ID, NAME, STATUS, LAYERS

Use this command to discover schedule IDs needed by `oncall schedule get`.

### oncall schedule get

Get detailed view of a single schedule including rotation layers, current/next on-call, and computed shift slots.

```bash
flashduty oncall schedule get <schedule_id> [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--since` | string | `now` | Start of time range |
| `--until` | string | `+7d` | End of time range |

Output: Key-value header (ID, Name, Status, Layers, Current, Next) followed by a slots table (START, END, GROUP) showing the computed final schedule within the requested time range.

## Workflows

### Find Who Is On Call Right Now

```bash
flashduty oncall who
```

Shows all schedules with the current on-call person and when their shift ends.

### Find On-Call for a Specific Team

```bash
# 1. Look up the team ID
flashduty team list --name "Platform"

# 2. Filter on-call by team
flashduty oncall who --team <team_id>
```

### Check Next Week's Schedule

```bash
flashduty oncall schedule get <schedule_id> --since now --until +7d
```

This is the default time range for `schedule get`, so `--since` and `--until` can be omitted.

### Find Who Will Be On Call at a Specific Time

```bash
flashduty oncall who --since "2024-01-20T00:00:00Z" --until "2024-01-21T00:00:00Z"
```

Provide an explicit time range to see who covers a future window.

## Key Concepts

- **Future duration syntax**: The `+24h` and `+7d` values in `--since`/`--until` are relative durations added to the current time. This is unique to on-call commands.
- **`oncall who` vs `oncall schedule list`**: `oncall who` is optimized for "who is on call NOW" (shows person and shift times). `oncall schedule list` is for browsing schedules (shows ID, name, status, layer count).
- **`oncall schedule get`**: Shows full rotation detail for a single schedule, including the computed final schedule as a slots table. Requires a `schedule_id` positional argument.
- **Read-only**: Schedules are read-only in the CLI. Creation and editing are done in the Flashduty web UI.

## Cross-References

- **Prerequisites:** flashduty-shared (authentication, global flags, output formatting)
- **Related:** flashduty-incident (reassign incidents to the current on-call person), flashduty-admin (look up team IDs and member details)
