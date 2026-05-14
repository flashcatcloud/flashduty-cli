---
name: flashduty-shared
version: 1.0.0
description: "Flashduty CLI foundation: authentication (login, app_key, config), the 3-layer noise reduction model (Alert Event to Alert to Incident), global flags (--json, --no-trunc), output modes (table, JSON, vertical detail), pagination (--limit, --page), time parsing (relative, absolute, unix, future durations), reference data lookups (member, team, channel, field, escalation-rule), and safety rules. Prerequisite for all other flashduty-* skills. Use when setting up flashduty-cli, encountering auth errors, looking up IDs, or needing to understand the Flashduty data model."
metadata:
  requires:
    bins: ["flashduty"]
  cliHelp: "flashduty --help"
---

# flashduty-shared

## Overview

Flashduty is an incident management platform. The CLI (`flashduty`) wraps the Flashduty Open API and is built with Go + Cobra. This skill is the foundation that all other `flashduty-*` skills build on. It covers the core data model, authentication, global flags, output formatting, pagination, time parsing, reference data lookups, and safety rules.

---

## The 3-Layer Noise Reduction Model

This is the **core mental model** for all Flashduty operations. Every command maps to one of these layers.

```
Monitoring / CI / External Systems
         |
         | push via integration_key
         v
┌──────────────────────────────────────┐
│  Layer 0: Alert Events (raw signals) │
│  enrichment -> pipeline -> routing   │
└──────────────┬───────────────────────┘
               | dedup by alert_key
┌──────────────v───────────────────────┐
│  Layer 1: Alerts (deduplicated)      │
│  drop -> grouping -> flapping        │
└──────────────┬───────────────────────┘
               | aggregate by grouping rules
┌──────────────v───────────────────────┐
│  Layer 2: Incidents (actionable)     │
│  silence -> inhibit -> escalate      │
│  -> notify                           │
└──────────────────────────────────────┘

Parallel: Change Events --(label correlation)--> Incidents
```

### Key Relationships

- One **Alert Event** creates or merges into one **Alert** (via `alert_key`).
- One **Alert** belongs to one **Incident** (via aggregation rules).
- One **Incident** contains 1 to 5000 **Alerts**.
- **Change Events** correlate with Incidents via shared label values and time proximity (no foreign key).

### CLI Command Mapping

| Layer | Entity | CLI command group |
|-------|--------|-------------------|
| 0 | Alert Event | `flashduty alert-event` |
| 1 | Alert | `flashduty alert` |
| 2 | Incident | `flashduty incident` |
| -- | Change Event | `flashduty change` |

---

## Authentication

### First-Time Setup

```bash
# Interactive login -- prompts for App Key (input is hidden)
flashduty login
```

The login command validates the key by making a test API call, then stores it in `~/.flashduty/config.yaml` with `0600` permissions.

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `FLASHDUTY_APP_KEY` | Override app key (takes precedence over config file) |
| `FLASHDUTY_BASE_URL` | Override base URL (default: `https://api.flashcat.cloud`) |

### Config Management

```bash
# View current config (key is masked)
flashduty config show

# Set a config value
flashduty config set app_key <KEY>
flashduty config set base_url https://api.flashcat.cloud
```

### Resolution Order

1. `--app-key` flag (hidden, for scripting)
2. `FLASHDUTY_APP_KEY` environment variable
3. `~/.flashduty/config.yaml` file

If no key is found, the CLI returns: `no app key configured. Run 'flashduty login' or set FLASHDUTY_APP_KEY`.

---

## Global Flags

These flags are available on **every** command via Cobra `PersistentFlags`:

| Flag | Type | Default | Effect |
|------|------|---------|--------|
| `--json` | bool | `false` | Output as JSON instead of table |
| `--no-trunc` | bool | `false` | Do not truncate long values in table output |
| `--app-key` | string | `""` | Override app key (hidden flag) |
| `--base-url` | string | `""` | Override base URL |

---

## Output Modes

### Table (default)

Human-readable aligned columns. Long values are truncated with `...` unless `--no-trunc` is set. List commands append a pagination footer:

```
Showing 20 results (page 1, total 142).
```

### JSON (`--json`)

Machine-readable full output. No truncation. Suitable for piping to `jq`. Success messages are wrapped as `{"message": "..."}`.

### Vertical Detail

Used automatically for single-item lookups (e.g., `flashduty incident get <id>` with one ID). Displays key-value pairs vertically instead of a table row.

---

## Pagination

Most list commands support offset-based pagination:

| Flag | Default | Description |
|------|---------|-------------|
| `--limit` | `20` | Max results per page (max 100) |
| `--page` | `1` | Page number (1-based) |

**Constraint**: `page * limit <= 10000`. For data beyond this window, narrow the query using time filters or other criteria.

---

## Time Parsing

Time flags (`--since`, `--until`) accept multiple formats. The parser lives in `internal/timeutil/parse.go`.

| Format | Example | Meaning |
|--------|---------|---------|
| `now` | `now` | Current time |
| Relative duration | `24h`, `30m`, `7d` | Subtract from now |
| Future duration | `+24h`, `+7d` | Add to now (used with `--until` in oncall) |
| Date | `2024-01-15` | Midnight in local timezone |
| Datetime | `2024-01-15 10:00:00` | Exact time in local timezone |
| Unix timestamp | `1705312200` | Exact time (must be > 1000000000) |

**Notes**:
- The `d` suffix is shorthand for days (e.g., `7d` becomes `168h` internally).
- Negative durations are rejected.
- Default for `--since` is typically `24h`; default for `--until` is typically `now`.

---

## Reference Data Commands

These lookup commands find IDs required by other commands:

```bash
# Find a person's ID
flashduty member list --name "John"
flashduty member list --email "john@example.com"

# Find a team ID
flashduty team list --name "SRE"

# Get full team detail by ID, name, or ref-id
flashduty team get --id 123

# Find a channel (collaboration space) ID
flashduty channel list --name "Production"

# Find custom field definitions
flashduty field list --name "priority"

# View escalation rules for a channel
flashduty escalation-rule list --channel <channel_id>

# List status pages
flashduty status-page list
```

---

## Safety Rules

**Hard constraints for AI agents operating the CLI:**

1. **NEVER** create or close incidents without explicit user confirmation.
2. **NEVER** merge incidents or alerts without user confirmation -- merges are **irreversible**.
3. **NEVER** snooze incidents unless the user specifies a duration.
4. **NEVER** reassign or reopen incidents without user confirmation.
5. **NEVER** delete a team without explicit user confirmation -- deletion is **irreversible**.
6. **NEVER** update a team's member list without showing the current members first -- `--person-ids` replaces the entire list.
7. **Always** show what will be affected before executing destructive or mutating operations.
8. When in doubt about severity or scope, **list first, then act**.
9. Prefer **read-only** operations (`list`, `get`, `detail`) unless the user explicitly requests a mutation.
10. For bulk operations (multiple IDs), enumerate the targets and confirm before proceeding.

---

## Related Skills

All `flashduty-*` skills depend on this foundation:

| Skill | Purpose |
|-------|---------|
| `flashduty-shared` | Foundation (this skill) |
| `flashduty-incident` | Incident lifecycle -- list, create, ack, close, merge, snooze, reassign (Layer 2) |
| `flashduty-alert` | Alert events and alerts -- list, detail, merge to incident (Layer 0 + 1) |
| `flashduty-change` | Change event tracking and trend analysis |
| `flashduty-oncall` | On-call schedules and coverage queries |
| `flashduty-channel` | Collaboration spaces and escalation rules |
| `flashduty-insight` | Analytics -- MTTA, MTTR, top-K alerts, notification trends |
| `flashduty-admin` | Teams, members, custom fields, audit logs |
| `flashduty-template` | Notification template preview and validation |
