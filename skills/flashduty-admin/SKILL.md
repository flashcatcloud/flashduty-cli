---
name: flashduty-admin
version: 1.0.0
description: "Flashduty administration: list teams and members, search audit logs for compliance and investigation. Commands: team list, member list, audit search. Use when looking up person IDs or team IDs for other commands, finding contact information, searching who performed specific actions, or reviewing audit trails for compliance."
metadata:
  requires:
    bins: ["flashduty"]
  cliHelp: "flashduty member --help"
---

# flashduty-admin

**CRITICAL** -- Before using this skill, read [`../flashduty-shared/SKILL.md`](../flashduty-shared/SKILL.md) for authentication, the 3-layer noise reduction model, global flags, and safety rules.

## Overview

Manage teams, look up members, and search audit logs for compliance and investigation.

## Commands

### team list

List teams with their members.

```bash
flashduty team list [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--name` | string | | Search by team name |
| `--page` | int | 1 | Page number |

Output columns: ID, NAME, MEMBERS.

### member list

List organization members with contact details and status.

```bash
flashduty member list [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--name` | string | | Search by name |
| `--email` | string | | Search by email |
| `--page` | int | 1 | Page number |

Output columns: ID, NAME, EMAIL, STATUS, TIMEZONE.

### audit search

Search the audit log for system mutations. Uses cursor-based pagination internally.

```bash
flashduty audit search [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--since` | string | `7d` | Start time (relative like `24h`, `30d` or absolute) |
| `--until` | string | `now` | End time |
| `--person` | int | 0 | Filter by person ID |
| `--operation` | string | | Filter by operation type (comma-separated) |
| `--limit` | int | 20 | Max results |
| `--page` | int | 1 | Page number |

Output columns: TIMESTAMP, PERSON, OPERATION, TARGET, DETAILS.

## Workflows

### Find a Person's ID for Other Commands

Member IDs are required as inputs across many CLI commands (incident reassign, oncall who, insight responder, audit filters). Use member list to look them up.

```bash
# Search by name
flashduty member list --name "John"

# Search by email
flashduty member list --email "john@company.com"

# Use the resulting ID in other commands
flashduty incident reassign <incident_id> --assignee <person_id>
flashduty audit search --person <person_id> --since 7d
```

### Audit Investigation

```bash
# What did a specific person do in the past week?
flashduty audit search --person <person_id> --since 7d

# What operations of a specific type occurred in the past day?
flashduty audit search --operation "incident.close" --since 24h

# Full audit trail for compliance review
flashduty audit search --since 30d --limit 100
```

### Team Structure Review

```bash
# List all teams and their members
flashduty team list

# Search for a specific team by name
flashduty team list --name "Platform"
```

## Key Concepts

- **Member IDs** (int64) are used across many commands: incident assign/reassign, audit filters, oncall schedules.
- **Team IDs** (int64) are used for filtering: oncall schedules, postmortem list, channels.
- **Audit logs** track all mutations in the system -- useful for compliance, incident investigation, and change tracking.
- All admin commands in the CLI are **read-only**.

## Cross-References

- **Prerequisites:** flashduty-shared (authentication, output formatting)
- **Related:** flashduty-incident (reassign needs person IDs), flashduty-oncall (team-based schedule filtering), flashduty-insight (responder metrics)
