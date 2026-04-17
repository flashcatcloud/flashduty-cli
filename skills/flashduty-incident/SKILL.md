---
name: flashduty-incident
version: 1.0.0
description: "Flashduty incident lifecycle management: list, filter, triage, investigate, and resolve incidents. Commands: incident list, get, detail (AI summary), create, update, ack, close, merge, snooze, reopen, reassign, feed, timeline, alerts, similar; postmortem list. Use when responding to pages, triaging alerts, investigating outages, acknowledging or closing incidents, merging duplicates, snoozing during maintenance, reassigning responders, reviewing incident timelines, or listing post-mortem reports."
metadata:
  requires:
    bins: ["flashduty"]
  cliHelp: "flashduty incident --help"
---

# flashduty-incident

**CRITICAL** -- Before using this skill, read [`../flashduty-shared/SKILL.md`](../flashduty-shared/SKILL.md) for authentication, the 3-layer noise reduction model, global flags, and safety rules.

## Overview

Layer 2 of Flashduty's 3-layer noise reduction model (Alert -> **Incident** -> Event). Incidents are the actionable items that humans respond to. Alerts are aggregated into incidents via noise reduction rules; responders triage, investigate, and resolve incidents. This skill covers the full incident lifecycle: listing, creating, triaging, investigating, resolving, merging, snoozing, reassigning, and post-mortem review.

## Quick Decision

| User wants to... | Command |
|---|---|
| See active incidents | `incident list --progress Triggered` |
| Investigate an incident | `incident detail <id>` |
| Respond to a page | `incident ack <id>` |
| Combine related incidents | `incident merge <target> --source <ids>` |
| Delay an incident | `incident snooze <id> --duration 2h` |
| Re-route an incident | `incident reassign <id> --person <ids>` |
| Review past incidents | `postmortem list` |

## Commands

| Command | Description |
|---|---|
| [`incident list`](references/flashduty-incident-list.md) | List and filter incidents |
| [`incident get`](references/flashduty-incident-get.md) | Get incident(s) in table/detail view |
| [`incident detail`](references/flashduty-incident-detail.md) | Full detail with AI summary, root cause, impact |
| [`incident create`](references/flashduty-incident-create.md) | Create a new incident |
| [`incident update`](references/flashduty-incident-update.md) | Update incident fields |
| [`incident ack`](references/flashduty-incident-ack.md) | Acknowledge incidents (Triggered -> Processing) |
| [`incident close`](references/flashduty-incident-close.md) | Close incidents |
| [`incident merge`](references/flashduty-incident-merge.md) | Merge incidents (IRREVERSIBLE) |
| [`incident snooze`](references/flashduty-incident-snooze.md) | Snooze incidents for a duration |
| [`incident reopen`](references/flashduty-incident-reopen.md) | Reopen closed incidents |
| [`incident reassign`](references/flashduty-incident-reassign.md) | Reassign to new responders |
| [`incident feed`](references/flashduty-incident-feed.md) | Paginated event timeline |
| [`incident timeline`](references/flashduty-incident-timeline.md) | Full event history (non-paginated) |
| [`incident alerts`](references/flashduty-incident-alerts.md) | View contributing alerts |
| [`incident similar`](references/flashduty-incident-similar.md) | Find similar incidents |
| [`postmortem list`](references/flashduty-postmortem-list.md) | List post-mortem reports |

## Workflows

### Workflow 1: Triage an Active Incident

Investigate and respond to a newly triggered incident.

```bash
# 1. Find unacknowledged critical incidents
flashduty incident list --progress Triggered --severity Critical

# 2. Investigate with AI summary, root cause, and impact
flashduty incident detail <id>

# 3. Acknowledge ownership
flashduty incident ack <id>

# 4. See contributing alerts for root cause analysis
flashduty incident alerts <id>

# 5. Check for related past incidents
flashduty incident similar <id>

# 6. Resolve when fixed
flashduty incident close <id>
```

### Workflow 2: Merge Related Incidents

Consolidate multiple incidents caused by the same underlying issue.

```bash
# 1. Find related incidents by keyword
flashduty incident list --title "database" --progress Triggered

# 2. Review the results and identify the primary incident

# 3. Merge duplicates into the primary (IRREVERSIBLE)
flashduty incident merge <primary_id> --source <other_id1>,<other_id2>
```

### Workflow 3: Escalate and Reassign

Hand off an incident to the right responder.

```bash
# 1. Find the right person
flashduty member list --name "senior"

# 2. Reassign
flashduty incident reassign <id> --person <person_id>
```

### Workflow 4: Snooze for Maintenance Window

Temporarily silence an incident during planned maintenance.

```bash
# 1. Snooze for the maintenance duration
flashduty incident snooze <id> --duration 2h

# 2. After maintenance, check if it re-triggered
flashduty incident list --progress Triggered
```

### Workflow 5: Post-Incident Review

Review what happened after resolving an incident.

```bash
# 1. Get full incident detail with AI analysis
flashduty incident detail <id>

# 2. Review the timeline of events
flashduty incident timeline <id>

# 3. Check the feed for all actions taken
flashduty incident feed <id>

# 4. Look at related post-mortems
flashduty postmortem list --status published --since 7d
```

## Key Concepts

- **Incident states**: `Triggered` (new, unacknowledged) -> `Processing` (acknowledged, being worked) -> `Closed` (resolved).
- **Severity levels**: `Critical`, `Warning`, `Info`.
- **Noise reduction**: Multiple alerts are aggregated into a single incident via noise reduction rules, reducing responder fatigue.
- **AI analysis**: The `incident detail` command provides AI-generated summaries, root cause analysis, resolution suggestions, and impact assessments.

## Safety Notes

- `incident merge` is **IRREVERSIBLE** -- always double-check source and target IDs before running.
- `incident create` triggers notifications to responders -- confirm with the user before creating.
- Always `incident list` before bulk operations (`ack`, `close`, `reopen`) to verify the right incidents are targeted.
- `incident snooze` has a maximum duration of 24 hours and requires whole minutes.

## Cross-References

- **Prerequisites**: `flashduty-shared` -- authentication, 3-layer noise reduction model overview, global flags (`--json`, `--api-key`, `--api-host`).
- **Related skills**:
  - `flashduty-alert` -- drill into contributing alerts (Layer 1).
  - `flashduty-oncall` -- find who is on-call and should respond.
  - `flashduty-insight` -- MTTA/MTTR metrics and operational analytics.
