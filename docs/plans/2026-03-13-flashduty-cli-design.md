# FlashDuty CLI Design

## Overview

A command-line interface for the FlashDuty platform, providing all functionality currently available through the flashduty-mcp-server but as a traditional CLI tool. Built in Go, consuming a shared `flashduty-go-sdk` module that is also used by the MCP server.

## Decisions

| Aspect | Decision |
|--------|----------|
| Language | Go |
| CLI framework | Cobra + Viper |
| API client | New `flashduty-go-sdk` repo, shared with MCP server |
| Command style | Noun-verb (`flashduty incident list`) |
| Auth | Config file (`~/.flashduty/config.yaml`) + env var + flag |
| Output | Human-readable tables by default, `--json` for machine output |
| Enrichment | On by default, `--raw` to skip |
| Distribution | GoReleaser + GitHub Releases + Homebrew tap |

## Dependency Graph

```
flashduty-go-sdk
    ^            ^
    |            |
flashduty-CLI    flashduty-mcp-server
```

## SDK Design (`flashduty-go-sdk`)

A new standalone repo extracted from the MCP server's `pkg/flashduty/` package (~3,600 lines).

### Structure

```
flashduty-go-sdk/
├── client.go              # HTTP client
├── client_options.go      # Functional options: WithBaseURL, WithTimeout, etc.
├── incidents.go           # Incident API methods + types
├── incidents_enrich.go    # Incident enrichment logic
├── changes.go             # Change API methods + types
├── statuspage.go          # Status page API methods + types
├── members.go             # Member/person API methods + types
├── teams.go               # Team API methods + types
├── channels.go            # Channel API methods + types
├── channels_enrich.go     # Channel enrichment logic
├── escalation_rules.go    # Escalation rule API methods + types
├── fields.go              # Custom field API methods + types
├── errors.go              # Error types and handling
├── go.mod
└── go.sum
```

### What moves to the SDK

- `Client` struct and HTTP plumbing (auth, headers, timeouts, response parsing)
- All API methods (`ListIncidents`, `CreateIncident`, `AckIncident`, etc.)
- All domain types (`Incident`, `Channel`, `Member`, `Team`, etc.)
- Enrichment logic (`EnrichIncidents`, `EnrichChannels`)
- Error types

### What stays in each consumer

- MCP server: tool registration, MCP parameter parsing, TOON formatting, toolset management
- CLI: Cobra commands, flag parsing, table formatting, config file handling

### API Style

```go
client := flashduty.NewClient("fd_xxx",
    flashduty.WithBaseURL("https://api.flashcat.cloud"),
    flashduty.WithTimeout(30 * time.Second),
)

incidents, err := client.ListIncidents(ctx, &flashduty.ListIncidentsInput{
    Status:    []string{"triggered", "acknowledged"},
    Severity:  []string{"critical"},
    ChannelID: "ch_abc",
})

enriched, err := client.EnrichIncidents(ctx, incidents)
```

## CLI Design

### Project Structure

```
flashduty-CLI/
├── cmd/
│   └── flashduty/
│       └── main.go                  # Entry point
├── internal/
│   ├── cli/
│   │   ├── root.go                  # Root command, global flags (--json, --raw)
│   │   ├── login.go                 # flashduty login / flashduty config
│   │   ├── incident.go              # incident subcommands
│   │   ├── change.go                # change subcommands
│   │   ├── statuspage.go            # status-page subcommands
│   │   ├── member.go                # member subcommands
│   │   ├── team.go                  # team subcommands
│   │   ├── channel.go               # channel subcommands
│   │   ├── escalationrule.go        # escalation-rule subcommands
│   │   └── field.go                 # field subcommands
│   ├── config/
│   │   └── config.go                # Config file + env var loading
│   └── output/
│       ├── table.go                 # Human-readable table formatter
│       └── json.go                  # JSON output formatter
├── go.mod
├── go.sum
├── Makefile
├── .goreleaser.yml
└── .github/
    └── workflows/
        ├── ci.yml                   # Lint + test + build
        └── release.yml              # GoReleaser on tag push
```

### Commands

```
flashduty login                          # auth setup (interactive)
flashduty config show                    # print resolved config (key masked)
flashduty config set <key> <value>       # override specific config values

flashduty incident list                  # query incidents
flashduty incident get <id>              # get specific incident
flashduty incident create                # create incident
flashduty incident update <id>           # modify incident
flashduty incident ack <id>              # acknowledge
flashduty incident close <id>            # close/resolve
flashduty incident timeline <id>         # view timeline
flashduty incident alerts <id>           # view alerts
flashduty incident similar <id>          # find similar

flashduty change list                    # query changes

flashduty status-page list               # list status pages
flashduty status-page changes            # list status change events
flashduty status-page create-incident    # create status page incident
flashduty status-page create-timeline    # add timeline to status change

flashduty member list                    # query members
flashduty team list                      # query teams
flashduty channel list                   # query channels
flashduty escalation-rule list           # query escalation rules
flashduty field list                     # query fields
```

### Authentication

**Config file** at `~/.flashduty/config.yaml`:

```yaml
app_key: fd_xxxxxxxxxxxxxxxx
base_url: https://api.flashcat.cloud
```

**`flashduty login` flow:**

1. Prompt for app key (masked input)
2. Validate by calling `POST /member/list` with limit=1
3. On success, write `~/.flashduty/config.yaml` (0600 permissions) and print authenticated user name
4. On failure, print error and exit non-zero

**Resolution order** (highest priority first):

1. `--app-key` flag (hidden from help output to discourage shell history exposure)
2. `FLASHDUTY_APP_KEY` environment variable
3. `~/.flashduty/config.yaml`

### Output

**Default: human-readable tables**

```
$ flashduty incident list

ID           TITLE                    SEVERITY   STATUS        CHANNEL       CREATED
inc_abc123   DB connection timeout    Critical   Triggered     Production    2026-03-13 10:23
inc_def456   High memory usage        Warning    Acknowledged  Staging       2026-03-13 09:15
inc_ghi789   SSL cert expiring        Info       Resolved      Production    2026-03-12 18:00

3 incidents found.
```

**`--json`: machine-parseable JSON**

```
$ flashduty incident list --json
[{"id":"inc_abc123","title":"DB connection timeout",...},...]
```

- Table formatting via `text/tabwriter` (stdlib)
- `--json` global flag inherited by all subcommands
- `--raw` skips enrichment, works with both table and JSON

### Distribution

- **GoReleaser** (`.goreleaser.yml`): builds for linux/darwin/windows across amd64/arm64
- **GitHub Releases**: triggered on tag push (`v0.1.0`, etc.)
- **Homebrew tap**: via `homebrew-tap` repo (`brew install flashcatcloud/tap/flashduty`)

## Implementation Order

1. Extract `flashduty-go-sdk` from MCP server's `pkg/flashduty/`
2. Refactor MCP server to consume the SDK (verify no regressions)
3. Build CLI skeleton (root command, config, login)
4. Implement commands resource by resource (incidents first -- largest surface area)
5. Add GoReleaser + Homebrew tap
6. Polish (shell completions, `--help` examples)
