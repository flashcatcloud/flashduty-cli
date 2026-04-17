# Flashduty CLI

English | [中文](README_zh.md)

A command-line interface for the [Flashduty](https://flashcat.cloud) platform. Manage incidents, on-call schedules, status pages, and more from your terminal.

## Installation

### macOS / Linux

```bash
curl -sSL https://raw.githubusercontent.com/flashcatcloud/flashduty-cli/main/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/flashcatcloud/flashduty-cli/main/install.ps1 | iex
```

### Go Install

```bash
go install github.com/flashcatcloud/flashduty-cli/cmd/flashduty@latest
```

> Make sure `$(go env GOPATH)/bin` is in your `PATH`. If `flashduty` is not found after install, run:
> ```bash
> export PATH="$(go env GOPATH)/bin:$PATH"
> ```

### Manual Download

Download the latest release for your platform from [GitHub Releases](https://github.com/flashcatcloud/flashduty-cli/releases).

### Options

| Variable | Description | Default |
|----------|-------------|---------|
| `FLASHDUTY_VERSION` | Install a specific version (e.g. `v0.1.2`) | latest |
| `FLASHDUTY_INSTALL_DIR` | Custom install directory | `/usr/local/bin` (shell), `~\.flashduty\bin` (PowerShell) |

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
| `--json` | Output as JSON instead of table |
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

### `statuspage` - Status Page Management (4 commands)

```bash
flashduty statuspage list [--id <ids>]                                  # List status pages
flashduty statuspage changes --page-id <id> --type <incident|maintenance>  # List active changes
flashduty statuspage create-incident --page-id <id> --title <title>     # Create status incident
flashduty statuspage create-timeline --page-id <id> --change <id> --message <msg>  # Add timeline update
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

**JSON (`--json`):** Machine-parseable, full data, no truncation.

```bash
flashduty incident list --json | jq '.[].title'
```

**No truncation (`--no-trunc`):** Table with full field content.

---

## Agent Skills

Flashduty CLI ships with 9 [Claude Code agent skills](https://docs.anthropic.com/en/docs/claude-code) that teach AI agents how to operate Flashduty through the CLI. Install them globally to enable AI-assisted incident management:

```bash
npx skills add flashcatcloud/flashduty-cli -y -g
```

After installation, Claude Code will automatically discover and invoke the appropriate skill based on your requests.

### Available Skills

| Skill | Scope |
|-------|-------|
| `flashduty-shared` | Foundation: authentication, 3-layer model, global flags, safety rules |
| `flashduty-incident` | Incident lifecycle: triage, investigate, resolve, merge, snooze, reassign |
| `flashduty-alert` | Alert and alert event investigation: drill down, trace, merge |
| `flashduty-change` | Change event tracking and deployment frequency trends |
| `flashduty-oncall` | On-call schedule queries: who is on call, shift details |
| `flashduty-channel` | Channel and escalation rule lookups |
| `flashduty-insight` | Analytics: MTTA/MTTR, noise reduction, notification trends |
| `flashduty-admin` | Team/member lookups and audit log search |
| `flashduty-template` | Notification template validation and preview |

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

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
