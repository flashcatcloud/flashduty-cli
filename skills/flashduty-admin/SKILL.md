---
name: flashduty-admin
version: 1.0.0
description: "Flashduty administration: manage teams (list, get, create, update, delete), list members, and search audit logs for compliance and investigation. Commands: team list/get/create/update/delete, member list, audit search. Use when managing team structure, looking up person IDs or team IDs for other commands, finding contact information, searching who performed specific actions, or reviewing audit trails for compliance."
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
| `--name` | string | | Search by team name substring |
| `--page` | int | 1 | Page number |
| `--limit` | int | 20 | Page size, max 100 |
| `--orderby` | string | | Sort field: `created_at`, `updated_at`, `team_name` |
| `--asc` | bool | false | Sort in ascending order |
| `--person-id` | int | 0 | Filter teams by member ID |

Output columns: ID, NAME, MEMBERS.

### team get

Get detailed information about a specific team. Specify the team by exactly one of `--id`, `--name`, or `--ref-id`.

```bash
flashduty team get --id <team_id>
flashduty team get --name "SRE Team"
flashduty team get --ref-id "hr-dept-42"
```

| Flag | Type | Description |
|------|------|-------------|
| `--id` | int | Team ID |
| `--name` | string | Team name (exact match) |
| `--ref-id` | string | External reference ID |

Output fields: ID, Name, Description, Status, Ref ID, Members, Created, Updated, Created By, Updated By.

### team create

Create a new team. The `--name` flag is required and must be unique (1-39 characters).

```bash
flashduty team create --name "SRE Team" [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--name` | string | **Required.** Team name (1-39 characters) |
| `--description` | string | Team description (max 500 characters) |
| `--person-ids` | string | Comma-separated member person IDs |
| `--emails` | string | Comma-separated email addresses to invite |
| `--ref-id` | string | External reference ID for HR system integration |

### team update

Update an existing team. The `--id` flag is required. **WARNING:** `--person-ids` replaces the entire member list. Use `team get` to see current members before updating.

```bash
flashduty team update --id <team_id> [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--id` | int | **Required.** Team ID |
| `--name` | string | New team name (1-39 characters) |
| `--description` | string | New description (max 500 characters) |
| `--person-ids` | string | Comma-separated member person IDs (replaces entire member list) |
| `--emails` | string | Comma-separated email addresses to invite |
| `--ref-id` | string | External reference ID |

### team delete

Permanently delete a team. Specify the team by exactly one of `--id`, `--name`, or `--ref-id`. This action is **irreversible**. You will be prompted for confirmation unless `--force` is set.

```bash
flashduty team delete --id <team_id>
flashduty team delete --name "Old Team" --force
```

| Flag | Type | Description |
|------|------|-------------|
| `--id` | int | Team ID |
| `--name` | string | Team name |
| `--ref-id` | string | External reference ID |
| `--force` | bool | Skip confirmation prompt |

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

# Get full detail for a specific team
flashduty team get --id 123

# Look up a team by external reference ID
flashduty team get --ref-id "hr-dept-42"
```

### Team Lifecycle Management

```bash
# Create a new team with initial members
flashduty team create --name "SRE Team" --description "Site Reliability" --person-ids 1,2,3

# Create a team and invite members by email
flashduty team create --name "Backend Team" --emails alice@example.com,bob@example.com

# Rename a team
flashduty team update --id 123 --name "Platform SRE"

# Replace the entire member list (check current members first!)
flashduty team get --id 123
flashduty team update --id 123 --person-ids 1,2,3,4,5

# Delete a team (prompts for confirmation)
flashduty team delete --id 123

# Delete without confirmation (for scripting)
flashduty team delete --id 123 --force
```

## Key Concepts

- **Member IDs** (int64) are used across many commands: incident assign/reassign, audit filters, oncall schedules.
- **Team IDs** (int64) are used for filtering: oncall schedules, postmortem list, channels.
- **Team update replaces members** -- `--person-ids` is a full replacement, not an append. Always check current members with `team get` before updating.
- **Team delete is irreversible** -- requires confirmation in interactive mode; requires `--force` in non-interactive (CI/scripted) mode.
- **Audit logs** track all mutations in the system -- useful for compliance, incident investigation, and change tracking.

## Cross-References

- **Prerequisites:** flashduty-shared (authentication, output formatting)
- **Related:** flashduty-incident (reassign needs person IDs), flashduty-oncall (team-based schedule filtering), flashduty-insight (responder metrics)
