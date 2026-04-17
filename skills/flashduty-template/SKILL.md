---
name: flashduty-template
version: 1.0.0
description: "Flashduty notification template management: get preset templates, validate and preview custom templates, explore available variables and functions. Commands: template get-preset, validate, variables, functions. Use when customizing incident notification rendering, testing template changes, debugging template errors, or exploring available template variables and Sprig/custom functions."
metadata:
  requires:
    bins: ["flashduty"]
  cliHelp: "flashduty template --help"
---

# flashduty-template

**CRITICAL** -- Before using this skill, read [`../flashduty-shared/SKILL.md`](../flashduty-shared/SKILL.md) for authentication, the 3-layer noise reduction model, global flags, and safety rules.

## Overview

Templates control how incident notifications are rendered across different channels (Slack, email, SMS, etc.). This skill covers retrieving preset templates, validating custom templates, and exploring the available template variables and functions.

## Commands

### template get-preset

Get the default/preset template for a specific notification channel. Useful as a starting point for customization.

```bash
flashduty template get-preset --channel <notification_channel>
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--channel` | string | | Notification channel (**required**). Values include various Flashduty notification channels (e.g. Slack, email, SMS). |

Example:
```bash
# Get the preset Slack notification template
flashduty template get-preset --channel slack
```

### template validate

Validate a template file and preview the rendered output. Reports validation status, errors, warnings, rendered size vs channel limit, and a preview of the rendered notification.

```bash
flashduty template validate --channel <channel> --file <path> [--incident <id>]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--channel` | string | | Notification channel (**required**) |
| `--file` | string | | Path to template file on local filesystem (**required**) |
| `--incident` | string | | Real incident ID for preview (optional; uses mock data if omitted) |

Examples:
```bash
# Validate with mock data
flashduty template validate --channel slack --file ./my-template.tpl

# Validate with a real incident for realistic preview
flashduty template validate --channel slack --file ./my-template.tpl --incident abc123
```

### template variables

List all available template variables with name, type, description, and example values.

```bash
flashduty template variables [--category <category>]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--category` | string | | Filter by category: `core`, `time`, `people`, `alerts`, `labels`, `context`, `notification`, `post_incident` |

Examples:
```bash
# List all variables
flashduty template variables

# List only core incident variables
flashduty template variables --category core

# List alert-related variables
flashduty template variables --category alerts
```

### template functions

List available template functions with name, syntax, and description. Includes both Flashduty custom functions and Sprig template functions.

```bash
flashduty template functions [--type <type>]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type` | string | `all` | Filter by type: `custom`, `sprig`, or `all` |

Examples:
```bash
# List all functions
flashduty template functions

# List only Flashduty custom functions
flashduty template functions --type custom

# List only Sprig functions
flashduty template functions --type sprig
```

## Workflows

### Workflow 1: Customize a Notification Template

Start from a preset, modify it, and validate the result.

```bash
# 1. Get the preset template as a starting point
flashduty template get-preset --channel slack

# 2. Save the output to a file and edit it
#    (redirect output or copy-paste into ./my-template.tpl)

# 3. Validate your changes with mock data
flashduty template validate --channel slack --file ./my-template.tpl

# 4. Test with a real incident for a realistic preview
flashduty template validate --channel slack --file ./my-template.tpl --incident <incident_id>
```

### Workflow 2: Explore Available Template Data

Discover what variables and functions are available for use in templates.

```bash
# 1. See all variable categories
flashduty template variables

# 2. Focus on specific data areas
flashduty template variables --category core
flashduty template variables --category alerts
flashduty template variables --category time

# 3. See available functions
flashduty template functions --type custom
flashduty template functions --type sprig
```

### Workflow 3: Debug a Template Rendering Issue

Diagnose why a template is not rendering as expected.

```bash
# 1. Check for syntax errors and validation issues
flashduty template validate --channel email --file ./broken-template.tpl

# 2. Review errors and warnings in the output
#    - Syntax errors point to the exact location of the problem
#    - Warnings highlight deprecated variables or risky patterns

# 3. Check rendered size vs channel limit
#    - Each channel enforces a maximum rendered size
#    - The validate command reports the current size and the limit

# 4. Look up variables used in the template
flashduty template variables --category core
```

## Key Concepts

- Templates use **Go template syntax** (similar to Jinja2/Mustache but with `{{ }}` delimiters and pipeline operators).
- Each notification channel has its own template format and **size limits** -- validate always reports size vs limit.
- **Sprig functions** (string manipulation, date formatting, math, etc.) are available out of the box.
- Flashduty adds **custom functions** on top of Sprig for domain-specific operations.
- Validation checks **syntax correctness**, **size limits**, and **variable existence**.
- Template CRUD (create, update, delete) is planned for a future CLI release -- currently templates are managed via the Flashduty web UI.

## Cross-References

- **Prerequisites:** `flashduty-shared` -- authentication, global flags (`--json`, `--api-key`, `--api-host`).
- **Related skills:**
  - `flashduty-incident` -- templates render incident data; use to find real incident IDs for preview.
  - `flashduty-channel` -- templates are used within escalation notification channels.
