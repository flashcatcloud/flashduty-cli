# Flashduty CLI

English | [中文](README_zh.md)

[![License](https://img.shields.io/github/license/flashcatcloud/flashduty-cli?style=flat-square&color=24bfa5&label=License)](LICENSE)
[![Release](https://img.shields.io/github/v/release/flashcatcloud/flashduty-cli?style=flat-square&color=24bfa5)](https://github.com/flashcatcloud/flashduty-cli/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/flashcatcloud/flashduty-cli/ci.yml?style=flat-square&branch=main&label=CI)](https://github.com/flashcatcloud/flashduty-cli/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/flashcatcloud/flashduty-cli?style=flat-square)](https://goreportcard.com/report/github.com/flashcatcloud/flashduty-cli)

A command-line interface for the [Flashduty](https://flashcat.cloud) platform. Manage incidents, on-call schedules, status pages, and more from your terminal.

## Installation

### macOS / Linux

```bash
curl -sSL https://static.flashcat.cloud/flashduty-cli/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://static.flashcat.cloud/flashduty-cli/install.ps1 | iex
```

### Manual Download

Download the latest release for your platform from [GitHub Releases](https://github.com/flashcatcloud/flashduty-cli/releases).

### Options

| Variable | Description | Default |
|----------|-------------|---------|
| `FLASHDUTY_VERSION` | Install a specific version (e.g. `v0.1.2`) | latest |
| `FLASHDUTY_INSTALL_DIR` | Custom install directory | `/usr/local/bin` (shell), `~\.flashduty\bin` (PowerShell) |
| `MIRROR_URL` | Override installer release asset mirror | `https://static.flashcat.cloud/flashduty-cli` |
| `FLASHDUTY_UPDATE_BASE_URL` | Override `flashduty update` and auto update-check base URL | `https://static.flashcat.cloud/flashduty-cli` |

## Quick Start

### 1. Authenticate

```bash
flashduty login
```

You will be prompted for your Flashduty APP key. To obtain one, log into the [Flashduty console](https://console.flashcat.cloud) and navigate to **Account Settings > APP Key**.

Alternatively, set the key via environment variable:

```bash
export FLASHDUTY_APP_KEY=your_app_key
```

### 2. Use

```bash
# List recent incidents
flashduty incident list

# Get incident details
flashduty incident get <incident_id>

# List team members
flashduty member list

# View channels
flashduty channel list
```

---

## Authentication

The CLI resolves credentials in this order (highest priority first):

1. `--app-key` flag (hidden, for scripting)
2. `FLASHDUTY_APP_KEY` environment variable
3. `~/.flashduty/config.yaml` (written by `flashduty login`)

### Configuration File

Stored at `~/.flashduty/config.yaml` with `0600` permissions:

```yaml
app_key: your_app_key
base_url: https://api.flashcat.cloud
```

### Configuration Commands

```bash
flashduty config show              # Print current config (key masked)
flashduty config set app_key KEY   # Set app key
flashduty config set base_url URL  # Override API endpoint
```

---

## Global Flags

| Flag | Description |
|------|-------------|
| `--output-format` | Output format: `table` (default), `json`, or `toon` (compact, fewer tokens) |
| `--json` | Output as JSON (alias for `--output-format json`) |
| `--no-trunc` | Do not truncate long fields in table output |
| `--base-url` | Override the API base URL |

---

## Available Commands

### `incident` - Incident Lifecycle Management (9 commands)

```bash
flashduty incident list [flags]        # List incidents (default: last 24h)
flashduty incident get <id> [<id2>]    # Get incident details (vertical view for single ID)
flashduty incident create [flags]      # Create a new incident (interactive if flags missing)
flashduty incident update <id> [flags] # Update incident fields
flashduty incident ack <id> [<id2>]    # Acknowledge incidents
flashduty incident close <id> [<id2>]  # Close (resolve) incidents
flashduty incident timeline <id>       # View incident timeline
flashduty incident alerts <id>         # View incident alerts
flashduty incident similar <id>        # Find similar historical incidents
```

**List flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--progress` | Filter: Triggered, Processing, Closed | all |
| `--severity` | Filter: Critical, Warning, Info | all |
| `--channel` | Filter by channel ID | - |
| `--title` | Search by title keyword | - |
| `--since` | Start time (duration, date, datetime, or unix) | `24h` |
| `--until` | End time | `now` |
| `--limit` | Max results | `20` |
| `--page` | Page number | `1` |

**Time format examples:** `5m`, `1h`, `24h`, `168h`, `2026-04-01`, `2026-04-01 10:00:00`, `1712000000`

### `change` - Change Record Query (1 command)

```bash
flashduty change list [flags]    # List changes (deployments, configs)
```

Supports `--channel`, `--since`, `--until`, `--type`, `--limit`, `--page`.

### `member` - Member Query (1 command)

```bash
flashduty member list [flags]    # List members
```

Supports `--name`, `--email`, `--page`.

### `team` - Team Query (1 command)

```bash
flashduty team list [flags]      # List teams with members
```

Supports `--name`, `--page`.

### `channel` - Channel Query (1 command)

```bash
flashduty channel list [flags]   # List collaboration spaces
```

Supports `--name`.

### `escalation-rule` - Escalation Rule Query (1 command)

```bash
flashduty escalation-rule list --channel <id>          # By channel ID
flashduty escalation-rule list --channel-name <name>   # By channel name (auto-resolved)
```

### `field` - Custom Field Query (1 command)

```bash
flashduty field list [flags]     # List custom field definitions
```

Supports `--name`.

### `status-page` - Status Page Management (28 commands)

The group is `status-page` (hyphenated), not `statuspage`. Nested object/array
fields carry no typed flag and must be supplied as JSON through `--data`;
`--data -` reads the entire request body from stdin. Positional arguments and
explicitly-set typed flags override the matching keys inside `--data`.

**Pages, components, sections**

```bash
flashduty status-page list                                     # List status pages (JSON: {"items":[...]})
flashduty status-page info <page-id>                           # Page detail, incl. component and section IDs
flashduty status-page create --name <name> --url-name <slug> --type <public|internal> \
    --date-view <calendar|list> --display-uptime-mode <chart_and_percentage|chart|none>
flashduty status-page update <page-id> [--name <name>] [--url-name <slug>] ...   # Update a page
flashduty status-page delete <page-id>                         # Delete a page
flashduty status-page component-upsert <page-id> --data '{"components":[{"name":"API","section_id":"<section-id>"}]}'
flashduty status-page component-delete <component-id> [<id2>...] --page-id <page-id>
flashduty status-page section-upsert <page-id> --data '{"sections":[{"name":"Core"}]}'
flashduty status-page section-delete <section-id> [<id2>...] --page-id <page-id>
```

**Events (incident / maintenance) and their timeline**

```bash
flashduty status-page change-active-list <page-id> --type <incident|maintenance>   # Only in-progress events
flashduty status-page change-list <page-id> --type <incident|maintenance> --status <status>
flashduty status-page change-info --page-id <page-id> --change-id <change-id>
flashduty status-page change-create <page-id> --type <incident|maintenance> --title <title> \
    --status <status> --description <text> --data '{"updates":[...]}'
flashduty status-page change-update --page-id <page-id> --change-id <change-id> [--title <title>]
flashduty status-page change-delete --page-id <page-id> --change-id <change-id>
flashduty status-page change-timeline-create --page-id <page-id> --change-id <change-id> \
    --status <status> --description <text> [--data '{"component_changes":[...]}']
flashduty status-page change-timeline-update --page-id <page-id> --change-id <change-id> --update-id <update-id> [--description <text>]
flashduty status-page change-timeline-delete --page-id <page-id> --change-id <change-id> --update-id <update-id>
```

`change-create` takes `<page-id>` as a **required positional argument**, and its
required `updates` array (with the nested `component_changes`) has no flag — so a
real `change-create` call always carries a `--data` payload:

```bash
flashduty status-page change-create 5750613685214 --type incident \
  --title "API latency elevated" --status investigating \
  --description "Investigating elevated latency." \
  --data '{"updates":[{"status":"investigating","description":"Team is investigating.","component_changes":[{"component_id":"01KC3GAZ6ZJE40H55GM31RPWZE","status":"degraded"}]}]}'
```

The whole body can also come from stdin with `--data -`:

```bash
cat change.json | flashduty status-page change-create 5750613685214 --data -
```

Resolving an incident goes through `change-timeline-create`; every component the
event touched must be moved back to `operational`:

```bash
flashduty status-page change-timeline-create --page-id 5750613685214 --change-id 5821693893131 \
  --status resolved --description "Recovered." \
  --data '{"component_changes":[{"component_id":"01KC3GAZ6ZJE40H55GM31RPWZE","status":"operational"}]}'
```

**Subscribers and templates**

```bash
flashduty status-page subscriber-list <page-id> [--component-ids <ids>] [--page <n>] [--limit <n>]
flashduty status-page subscriber-import <page-id> --method <email|im> --data '{"subscribers":[...]}'
flashduty status-page subscriber-export <page-id> [--component-ids <ids>]
flashduty status-page template-list <page-id> --type <pre_defined|message>
flashduty status-page template-upsert <page-id> --type <pre_defined|message> --data '{"template":{...}}'
flashduty status-page template-delete --page-id <page-id> --template-id <template-id> --type <pre_defined|message>
```

**Migration from Atlassian Statuspage**

```bash
flashduty status-page migrate-structure <source-page-id> --api-key <key> [--url-name <slug>]   # Structure + history
flashduty status-page migrate-email-subscribers --source-page-id <id> --target-page-id <id> --api-key <key>
flashduty status-page migration-status <job-id>                # Check migration job status
flashduty status-page migration-cancel <job-id>                # Cancel a running migration job
```

Migration jobs are asynchronous. After starting `migrate-structure` or
`migrate-email-subscribers`, poll the returned `job_id`:

```bash
flashduty status-page migration-status <job-id>
```

Typical flow:

```bash
flashduty status-page migrate-structure page_123 --api-key $ATLASSIAN_STATUSPAGE_API_KEY
flashduty status-page migration-status <structure_job_id>
flashduty status-page migrate-email-subscribers --source-page-id page_123 \
  --target-page-id <target_page_id> --api-key $ATLASSIAN_STATUSPAGE_API_KEY
flashduty status-page migration-status <subscriber_job_id>
```

### `template` - Notification Template Management (4 commands)

```bash
flashduty template get-preset --channel <channel>                    # Get preset template code
flashduty template validate --channel <channel> --file <path>        # Validate and preview template
flashduty template variables [--category <category>]                 # List template variables
flashduty template functions [--type custom|sprig|all]               # List template functions
```

Supported channels: `dingtalk`, `dingtalk_app`, `feishu`, `feishu_app`, `wecom`, `wecom_app`, `slack`, `slack_app`, `telegram`, `teams_app`, `email`, `sms`, `zoom`.

### Utility Commands

```bash
flashduty login          # Authenticate interactively
flashduty config show    # Show current configuration
flashduty config set     # Set a configuration value
flashduty version        # Print version information
flashduty completion     # Generate shell completions (bash/zsh/fish/powershell)
```

---

## Output Formats

**Table (default):** Human-readable, aligned columns, long fields truncated.

```
ID           TITLE                    SEVERITY   PROGRESS     CHANNEL       CREATED
inc_abc123   DB connection timeout    Critical   Triggered    Production    2026-04-10 10:23
inc_def456   High memory usage        Warning    Processing   Staging       2026-04-10 09:15
Showing 2 results (page 1, total 2).
```

**JSON (`--json` / `--output-format json`):** Machine-parseable, full data, no truncation.

```bash
flashduty incident list --json | jq '.[].title'
```

**TOON (`--output-format toon`):** Token-Oriented Object Notation — full data, no truncation, but drops the per-row repeated keys that JSON emits for uniform arrays, so list output costs materially fewer tokens. Preferred for LLM/agent consumption. Not directly `jq`-able; use `--json` when you need to pipe into `jq`.

```bash
flashduty incident list --output-format toon
```

**No truncation (`--no-trunc`):** Table with full field content.

---

## Development

### Prerequisites

- Go 1.24+
- golangci-lint (auto-installed by Makefile)

### Build

```bash
make build       # Build binary to bin/flashduty
make test        # Run tests with race detection
make lint        # Run linter
make check       # Run all checks (fmt, lint, test, build)
make help        # Show all available targets
```

### Dependencies

| Package | Purpose |
|---------|---------|
| [flashduty-sdk](https://github.com/flashcatcloud/flashduty-sdk) | Flashduty API client |
| [cobra](https://github.com/spf13/cobra) | CLI framework |
| [yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) | Config file parsing |
| [x/term](https://pkg.go.dev/golang.org/x/term) | Masked password input |

---

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request, and note our [Code of Conduct](CODE_OF_CONDUCT.md).

- [Report a bug or request a feature](https://github.com/flashcatcloud/flashduty-cli/issues/new/choose)
- [Get help and support](SUPPORT.md)
- [Report a security vulnerability](SECURITY.md)

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
