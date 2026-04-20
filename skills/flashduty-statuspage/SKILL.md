---
name: flashduty-statuspage
version: 1.0.0
description: "Flashduty status page management and one-time migration from external providers. Commands: statuspage list, changes, create-incident, create-timeline; statuspage migrate structure, email-subscribers, status, cancel. Use when publishing an incident or maintenance to a public status page, posting a timeline update, importing an existing Atlassian Statuspage (structure, history, or email subscribers) into Flashduty, or polling a migration job. Does not cover incident response workflows inside Flashduty — see flashduty-incident for that."
metadata:
  requires:
    bins: ["flashduty"]
  cliHelp: "flashduty statuspage --help"
---

# flashduty-statuspage

**CRITICAL** -- Before using this skill, read [`../flashduty-shared/SKILL.md`](../flashduty-shared/SKILL.md) for authentication, the 3-layer noise reduction model, global flags, and safety rules.

## Overview

Status pages are the public-facing communication layer: they let customers and internal stakeholders see current service health, ongoing incidents, and scheduled maintenance. This skill covers two distinct use cases:

1. **Day-to-day operations** on existing Flashduty status pages: listing pages, posting incidents and maintenance windows, updating timelines.
2. **One-time migration** from an external provider (currently Atlassian Statuspage) into Flashduty: structure + history in one job, email subscribers in a separate job to control when verification emails go out.

Migration jobs are **asynchronous**. Start a job, poll its status, cancel if needed. The CLI never blocks waiting on a long-running migration.

## Quick Decision

| User wants to... | Command |
|---|---|
| See existing status pages | `statuspage list` |
| See open incidents or maintenance on a page | `statuspage changes --page-id <id> --type incident` |
| Publish a new incident | `statuspage create-incident --page-id <id> --title "..."` |
| Update an ongoing incident | `statuspage create-timeline --page-id <id> --change <id> --message "..."` |
| Migrate a page from Atlassian (contents first) | `statuspage migrate structure --from atlassian ...` |
| Migrate subscribers (after structure) | `statuspage migrate email-subscribers --from atlassian ...` |
| Check migration progress | `statuspage migrate status --job-id <id>` |
| Stop a runaway migration | `statuspage migrate cancel --job-id <id>` |

## Commands

### statuspage list

List all status pages visible to the account.

```bash
flashduty statuspage list [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--id` | string | Filter by page IDs (comma-separated) |

Output columns: ID, NAME, SLUG, STATUS, COMPONENTS.

Use this to look up page IDs for the other commands in this skill.

### statuspage changes

List active incidents or maintenance windows on a page.

```bash
flashduty statuspage changes --page-id <id> --type <incident|maintenance>
```

| Flag | Type | Description |
|------|------|-------------|
| `--page-id` | int | Page ID (**required**) |
| `--type` | string | `incident` or `maintenance` (**required**) |

Output columns: ID, TITLE, TYPE, STATUS, CREATED, UPDATED.

"Active" means not yet resolved / not yet completed. Returns both incident and maintenance changes depending on `--type`.

### statuspage create-incident

Publish a new incident on a status page. The incident appears to subscribers and on the public page immediately.

```bash
flashduty statuspage create-incident --page-id <id> --title <title> [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--page-id` | int | Page ID (**required**) |
| `--title` | string | Incident title, max 255 chars (**required**) |
| `--message` | string | Initial update message |
| `--components` | string | `id1:status,id2:status` — statuses: `operational`, `degraded`, `partial_outage`, `full_outage` |
| `--notify` | bool | Notify page subscribers (default: false) |

Use `--notify` deliberately — it sends email + push to every subscriber on the page.

### statuspage create-timeline

Add a timeline update to an existing incident or maintenance. Use this to move a change through its lifecycle (`investigating` → `identified` → `monitoring` → `resolved`).

```bash
flashduty statuspage create-timeline --page-id <id> --change <id> --message <msg> [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--page-id` | int | Page ID (**required**) |
| `--change` | int | Change ID (**required**) — get it from `statuspage changes` |
| `--message` | string | Timeline message (**required**) |
| `--status` | string | Incident: `investigating`, `identified`, `monitoring`, `resolved`; maintenance: `scheduled`, `ongoing`, `completed` |

The `--status` transition determines when a change is considered resolved / completed and stops appearing in `statuspage changes`.

### statuspage migrate structure

Start an asynchronous migration of status page **structure and history** from an external provider into a new Flashduty status page. Components, sections, past incidents, past maintenance windows, and notification templates are imported. **No emails are sent to subscribers.**

```bash
flashduty statuspage migrate structure --from atlassian --source-page-id <id> --api-key <key>
```

| Flag | Type | Description |
|------|------|-------------|
| `--from` | string | Source provider, currently only `atlassian` (**required**) |
| `--source-page-id` | string | Page ID in the source provider (**required**) |
| `--api-key` | string | Source provider API key (**required**) |

Returns a job ID plus the command to poll its status. Human output shows the new Flashduty `target_page_id` once the job reaches the `completed` phase — capture that for the subscriber migration.

### statuspage migrate email-subscribers

Start a **separate** migration of email subscribers from the external provider into an existing Flashduty status page. Split from `migrate structure` so that operators can verify imported content before waking up the subscriber list.

```bash
flashduty statuspage migrate email-subscribers --from atlassian --source-page-id <id> --target-page-id <id> --api-key <key>
```

| Flag | Type | Description |
|------|------|-------------|
| `--from` | string | Source provider, currently only `atlassian` (**required**) |
| `--source-page-id` | string | Page ID in the source provider (**required**) |
| `--target-page-id` | int | Flashduty page ID from `migrate structure` (**required**) |
| `--api-key` | string | Source provider API key (**required**) |

### statuspage migrate status

Poll the progress of an async migration job.

```bash
flashduty statuspage migrate status --job-id <id>
```

| Flag | Type | Description |
|------|------|-------------|
| `--job-id` | string | Migration job ID (**required**) |

Human output prints the current phase, status, progress counters (components, sections, incidents, maintenances, subscribers, templates), and any accumulated warnings or a fatal error. Poll until `Status: completed`, `failed`, or `cancelled`.

### statuspage migrate cancel

Request cancellation of an in-flight migration job. Best-effort: jobs in their final phase may still complete.

```bash
flashduty statuspage migrate cancel --job-id <id>
```

| Flag | Type | Description |
|------|------|-------------|
| `--job-id` | string | Migration job ID (**required**) |

Returns a confirmation and the command to poll the final state.

## Workflows

### Workflow 1: Publish and Manage an Incident

Post a new incident, move it through investigation, resolve it.

```bash
# 1. Find the page ID
flashduty statuspage list

# 2. Create the incident (no notifications yet)
flashduty statuspage create-incident \
  --page-id 42 \
  --title "Database latency elevated" \
  --message "We're investigating reports of slow database queries" \
  --components "comp_1:degraded"

# 3. See the change ID for follow-up updates
flashduty statuspage changes --page-id 42 --type incident

# 4. Post updates as the incident progresses
flashduty statuspage create-timeline \
  --page-id 42 --change 101 \
  --status identified \
  --message "Root cause: a runaway query. Rolling back now."

flashduty statuspage create-timeline \
  --page-id 42 --change 101 \
  --status monitoring \
  --message "Rollback complete. Monitoring for recurrence."

# 5. Resolve
flashduty statuspage create-timeline \
  --page-id 42 --change 101 \
  --status resolved \
  --message "Latency back to baseline. Closing."
```

### Workflow 2: Full Atlassian → Flashduty Migration

Import structure first, verify, then import subscribers.

```bash
# 1. Start the structure + history migration
flashduty statuspage migrate structure \
  --from atlassian \
  --source-page-id page_atl_123 \
  --api-key "$ATLASSIAN_STATUSPAGE_API_KEY"
# → captures Job ID: str_abc

# 2. Poll until completed
flashduty statuspage migrate status --job-id str_abc
# Repeat until Status: completed.
# Capture Target page ID from the completed job's output.

# 3. Sanity-check the imported page
flashduty statuspage list --id <new_page_id>
flashduty statuspage changes --page-id <new_page_id> --type incident

# 4. ONLY AFTER VERIFYING: start subscriber migration
flashduty statuspage migrate email-subscribers \
  --from atlassian \
  --source-page-id page_atl_123 \
  --target-page-id <new_page_id> \
  --api-key "$ATLASSIAN_STATUSPAGE_API_KEY"
# → captures Job ID: sub_xyz

# 5. Poll until completed
flashduty statuspage migrate status --job-id sub_xyz
```

### Workflow 3: Stop a Runaway Migration

```bash
# Saw unexpected warnings or want to retry with a different config
flashduty statuspage migrate cancel --job-id str_abc

# Confirm it reached a terminal state
flashduty statuspage migrate status --job-id str_abc
# Status should become cancelled.
```

## Key Concepts

- **Page ID** (int) is the Flashduty status page primary key. **Change ID** (int) is the ID of an incident/maintenance within a page. Don't confuse them.
- **Migration is async.** `migrate structure` and `migrate email-subscribers` return immediately with a job ID; the actual work happens on the backend.
- **Two migration jobs, not one.** Structure + history run separately from subscribers. This is deliberate — subscriber import triggers verification emails, so operators verify content first.
- **Migration phases** for the structure job progress in order: `components` → `sections` → `history` (incidents + maintenances) → `templates`. The subscribers job has a single `subscribers` phase.
- **Terminal statuses:** `completed`, `failed`, `cancelled`. Stop polling once any of these appears.
- **`--notify` is subscriber-visible.** In `create-incident`, omit or set `--notify=false` for silent incidents; set `--notify` when you want an announcement.
- **Component statuses vary by change type.** Incident statuses: `operational`, `degraded`, `partial_outage`, `full_outage`. Maintenance statuses: `operational`, `under_maintenance`.
- **Source provider support** is currently Atlassian Statuspage only. Other providers will require SDK + CLI updates.

## Cross-References

- **Prerequisites:** `flashduty-shared` — authentication, global flags (`--json`, `--no-trunc`), and safety rules.
- **Related skills:**
  - `flashduty-incident` — internal Flashduty incident response (distinct from public status page incidents).
  - `flashduty-channel` — channels are how alerts are routed internally; status pages publish to customers. They can be wired together but are independent concepts.
