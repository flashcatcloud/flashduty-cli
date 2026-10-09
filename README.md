# Flashduty CLI

English | [中文](README_zh.md)

[![License](https://img.shields.io/github/license/flashcatcloud/flashduty-cli?style=flat-square&color=24bfa5&label=License)](LICENSE)
[![Release](https://img.shields.io/github/v/release/flashcatcloud/flashduty-cli?style=flat-square&color=24bfa5)](https://github.com/flashcatcloud/flashduty-cli/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/flashcatcloud/flashduty-cli/ci.yml?style=flat-square&branch=main&label=CI)](https://github.com/flashcatcloud/flashduty-cli/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/flashcatcloud/flashduty-cli?style=flat-square)](https://goreportcard.com/report/github.com/flashcatcloud/flashduty-cli)

**Flashduty CLI** (`flashduty`) is the official open-source command-line tool for [Flashduty](https://www.flashduty.com), the incident management and on-call platform. From a terminal, a shell script, or an AI coding agent you can triage incidents and alerts, query on-call schedules, publish status page updates, manage monitors and RUM, and drive the AI SRE.

[Website](https://www.flashduty.com) · [CLI documentation](https://docs.flashduty.com/en/developer/cli) · [API reference](https://docs.flashduty.com/en/openapi/introduction) · [Console](https://console.flashcat.cloud) · [Blog: a CLI for humans and agents](https://www.flashduty.com/en/now/blog/flashduty-cli) · [Releases](https://github.com/flashcatcloud/flashduty-cli/releases)

## Highlights

- **The whole public API.** Every public Flashduty API operation has a command, generated from the OpenAPI spec through the [go-flashduty](https://github.com/flashcatcloud/go-flashduty) SDK. Common workflows (incidents, alerts, on-call, status pages) also get hand-tuned commands with shorter flags and readable tables.
- **Predictable names.** An API path maps directly to a command: `POST /incident/merge` is `flashduty incident merge`, and `POST /status-page/change/create` is `flashduty status-page change-create`.
- **Built for scripts and agents.** Output as `table`, `json`, or `toon` (compact, fewer tokens). List pages are size-bounded and say so when reduced. `--fields` projects rows to the fields you need.
- **One binary.** macOS, Linux, and Windows on amd64 and arm64. `flashduty update` upgrades in place.

## Installation

### macOS / Linux

```bash
curl -sSL https://static.flashcat.cloud/flashduty-cli/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://static.flashcat.cloud/flashduty-cli/install.ps1 | iex
```

### Manual download

Download the archive for your platform from [GitHub Releases](https://github.com/flashcatcloud/flashduty-cli/releases), extract it, and put the binary on your `PATH`.

### Installer options

| Variable | Description | Default |
|----------|-------------|---------|
| `FLASHDUTY_VERSION` | Install a specific version (e.g. `v1.5.12`) | latest |
| `FLASHDUTY_INSTALL_DIR` | Install directory | `/usr/local/bin` (shell), `~\.flashduty\bin` (PowerShell) |
| `MIRROR_URL` | Release asset mirror (must be `https://`) | `https://static.flashcat.cloud/flashduty-cli` |

## Quick start

```bash
# 1. Authenticate with an APP key (console: My → APP Key)
flashduty login
flashduty whoami

# 2. Work with incidents
flashduty incident list --since 24h --severity Critical
flashduty incident info <incident_id>
flashduty incident ack <incident_id>
flashduty incident merge <target_id> --source <id1>,<id2>   # sources are closed and kept
flashduty incident close <incident_id>

# 3. Who is on call, and what changed?
flashduty oncall who
flashduty change list --since 2h

# 4. Explore any area
flashduty status-page --help
```

How to create an APP key is described in the [API reference](https://docs.flashduty.com/en/openapi/introduction).

## Command groups

Run `flashduty <group> --help` for the commands in a group, and `flashduty <group> <command> --help` for flags and examples. The [CLI documentation](https://docs.flashduty.com/en/developer/cli) walks through the common workflows.

| Area | Groups |
|------|--------|
| On-call | `incident`, `alert`, `alert-event`, `change`, `channel`, `route`, `oncall`, `schedule`, `calendar`, `integration`, `webhook`, `enrichment`, `field`, `template`, `insight`, `status-page` |
| Monitors | `monit`, `monit-query`, `datasource` |
| RUM | `rum`, `sourcemap` |
| AI SRE | `safari`, `session`, `automation` |
| Platform | `account`, `member`, `person`, `team`, `role`, `audit` |
| CLI | `login`, `whoami`, `config`, `update`, `version`, `completion` |

### Request bodies

Generated commands expose each top-level request field as a typed flag, and take the full JSON body through `--data` (`--data -` reads stdin). Positional arguments and typed flags override the matching keys in `--data`, so nested objects and arrays go in `--data` while scalars stay readable:

```bash
flashduty status-page change-create <page_id> --type incident \
  --title "API latency elevated" --status investigating \
  --data '{"updates":[{"status":"investigating","description":"Investigating.","component_changes":[{"component_id":"<component_id>","status":"degraded"}]}]}'
```

## Authentication and configuration

Credentials are resolved in this order:

1. `--app-key` flag (hidden, for scripting)
2. `FLASHDUTY_APP_KEY` environment variable
3. `~/.flashduty/config.yaml`, written by `flashduty login` with `0600` permissions

```yaml
app_key: your_app_key
base_url: https://api.flashcat.cloud
```

```bash
flashduty config show              # Print current config (key masked)
flashduty config set app_key KEY   # Set the APP key
flashduty config set base_url URL  # Override the API endpoint
```

| Environment variable | Purpose |
|----------------------|---------|
| `FLASHDUTY_APP_KEY` | APP key |
| `FLASHDUTY_BASE_URL` | API endpoint (default `https://api.flashcat.cloud`) |
| `FLASHDUTY_NO_UPDATE_CHECK=1` | Disable the daily background update check |
| `FLASHDUTY_UPDATE_BASE_URL` | Mirror used by `flashduty update` and the update check |

## Global flags

| Flag | Description |
|------|-------------|
| `--output-format` | `table` (default), `json`, or `toon` |
| `--json` | Alias for `--output-format json` |
| `--no-trunc` | Do not truncate long fields in table output |
| `--base-url` | Override the API base URL |

## Output formats

- **Table** (default): aligned columns for people; long fields are truncated unless `--no-trunc` is set.
- **JSON** (`--json`): for `jq` and scripts, e.g. `flashduty incident list --json | jq '.[].title'`.
- **TOON** (`--output-format toon`): [Token-Oriented Object Notation](https://github.com/toon-format/toon-go). It drops the keys JSON repeats on every row, so lists cost far fewer tokens. Use it when an LLM or agent reads the output.

Every structured list page is capped at 16 KiB. A page that had to be reduced is reported on stderr, and list envelopes also carry `"truncated": true` in the payload. With `"emitted_rows": N`, only the first N rows were returned: re-request with a smaller `--limit` until the rows you hold reach `total`. Without it, every row is present but long values were clipped: narrow `--fields`.

## Updating

```bash
flashduty update           # Install the latest release in place
flashduty update --check   # Only report whether a newer release exists
```

## Development

Requires Go 1.26+ (see `go.mod`). golangci-lint is installed by the Makefile.

```bash
make build        # Build bin/flashduty
make test         # Run tests with the race detector
make check        # fmt, lint, test, build
make gen-cards    # Regenerate the command fences in skills/flashduty/reference
make check-cards  # Check those fences against the real command tree
make help         # All targets
```

Generated commands live in `internal/cli/zz_generated_*.go` and are produced by `go run ./internal/cmd/cligen` from the OpenAPI spec bundled with go-flashduty. Hand-written commands live next to them. When a hand-written command takes a generated command's name, `TestCuratedCommandsCoverRequestFields` requires it to still expose every request field of that API.

| Dependency | Purpose |
|------------|---------|
| [go-flashduty](https://github.com/flashcatcloud/go-flashduty) | Flashduty API client, generated from the OpenAPI spec |
| [cobra](https://github.com/spf13/cobra) | Command framework |
| [toon-go](https://github.com/toon-format/toon-go) | TOON output |
| [yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) | Config file |
| [x/term](https://pkg.go.dev/golang.org/x/term) | Masked APP key input |

## Related projects

- [go-flashduty](https://github.com/flashcatcloud/go-flashduty): Go SDK for the Flashduty API
- [flashduty-mcp-server](https://github.com/flashcatcloud/flashduty-mcp-server): MCP server for Flashduty
- [terraform-provider-flashduty](https://github.com/flashcatcloud/terraform-provider-flashduty): Terraform provider for Flashduty resources

## Contributing

Contributions are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request, and note our [Code of Conduct](CODE_OF_CONDUCT.md).

- [Report a bug or request a feature](https://github.com/flashcatcloud/flashduty-cli/issues/new/choose)
- [Get help and support](SUPPORT.md)
- [Report a security vulnerability](SECURITY.md)

## License

MIT. See [LICENSE](LICENSE).
