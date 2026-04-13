# FlashDuty CLI Implementation Plan

> Revised 2026-04-10. Supersedes the original design from 2026-03-13 where noted.

## Challenges to Original Design

### Challenge 1: SDK naming mismatch

The original plan referred to `flashduty-go-sdk`. The actual repo is `flashduty-sdk` at `github.com/flashcatcloud/flashduty-sdk`. All references updated.

**Resolution:** Updated all references to `flashduty-sdk`.

### Challenge 2: Enrichment is always-on in the SDK

The original plan proposed "Enrich by default, `--raw` to skip." The SDK has no `SkipEnrich` field -- enrichment is automatic and always-on. There is no way to skip it without SDK changes.

**Resolution:** Dropped `--raw` flag. All output is enriched. The CLI aligns with SDK behavior.

### Challenge 3: Incident "status" vs "progress"

The original design used `--status triggered,acknowledged,resolved`. The SDK uses `Progress` field with values `Triggered, Processing, Closed`.

**Resolution:** Use SDK terminology. Flag is `--progress` with values `Triggered,Processing,Closed`.

### Challenge 4: `incident get <id>` kept as convenience

No separate "get one incident" SDK method. `ListIncidents(IncidentIDs: [id])` does the job. However, `incident get` provides differentiated UX: a detailed vertical layout for single-incident inspection, vs the summary table row from `list`.

**Resolution:** Keep `incident get`. Single ID shows vertical detail view (ID, title, severity, progress, channel, creator, responders, description, labels, custom fields). Multiple IDs show table.

### Challenge 5: Templates API added for MCP server parity

The SDK has a full Templates API that was missing from the original design. The flashduty-mcp-server exposes 4 template tools. CLI should match all existing MCP server functionality.

**Resolution:** Added `flashduty template` command group with 4 subcommands: get-preset, validate, variables, functions.

### Challenge 6: Config file format

YAML requires `gopkg.in/yaml.v3`. JSON needs nothing extra.

**Resolution:** Keep YAML. More human-friendly for hand-editing. Dependency is trivial and battle-tested.

### Challenge 7: Table output truncation

`text/tabwriter` handles alignment but not truncation. One long title breaks the entire table layout.

**Resolution:** Use `text/tabwriter` for v1 with a `truncate(s string, maxLen int)` helper for long fields (title at 50 chars, description at 80 chars). Add `--no-trunc` global flag to disable truncation in table mode. JSON output is always full (untruncated).

### Challenge 8: Interactive incident creation

Requiring all fields via flags is awkward for interactive use.

**Resolution:** In v1, if required flags (`--title`, `--severity`) are missing and stdin is a terminal, prompt interactively. Not a full TUI -- just simple `fmt.Print` + `bufio.Scanner` prompts for required fields.

### Challenge 9: Time range defaults

SDK requires `StartTime` and `EndTime` when querying by filters. Bare `flashduty incident list` would fail.

**Resolution:** Default `--since 24h`, `--until now`. Supported formats:
- Relative duration (parsed by Go `time.ParseDuration`): `5m`, `1h`, `24h`, `168h`
- Absolute date: `2026-04-01`
- Absolute datetime: `2026-04-01 10:00:00`
- Unix timestamp: `1712000000`

No magic shorthands like `today`/`yesterday` -- keep it to standard Go duration format plus absolute timestamps.

### Challenge 10: Escalation rules require channel ID

`ListEscalationRules` takes `channelID int64` as required. Users may not know numeric IDs.

**Resolution:** Accept both `--channel <id>` and `--channel-name <name>`. If `--channel-name` is given, resolve via `ListChannels` first:
- 0 matches: error "no channel found matching '<name>'"
- 1 match: use that channel ID, proceed
- Multiple matches: list all matches and error "multiple channels match '<name>', use --channel <id> to specify"

### Challenge 11: `CreateIncident` and `CreateStatusIncident` return `any`

The SDK's `CreateIncident` returns `(any, error)` where `any` is raw `response.Data`. No typed struct to extract the incident ID from. `CreateStatusIncident` has the same issue.

**Resolution:** Use type assertion in the CLI: assert `any` to `map[string]any` and extract the ID field. This is a pragmatic workaround for v1. The SDK should be fixed to return typed responses in a future version.

### Challenge 12: `IncludeAlerts` defaults to `false`

The SDK's `ListIncidentsInput.IncludeAlerts` is `false` by default (Go zero value).

**Resolution:** Set `IncludeAlerts: false` for `incident list` (performance -- alerts aren't shown in summary table anyway). Set `IncludeAlerts: true` for `incident get` (full detail view should include alerts).

### Challenge 13: `--assign` takes person IDs only

`CreateIncidentInput.AssignedTo` is `[]int`. Users won't know numeric person IDs by heart.

**Resolution:** IDs only for v1. Help text reminds users: "Use 'flashduty member list' to look up person IDs." Email-based resolution can be added in a future version.

### Challenge 14: Viper is unnecessary

Viper brings ~20 transitive dependencies. Our config needs are simple: one YAML file + a few env vars + flag overrides.

**Resolution:** Dropped Viper. Use `gopkg.in/yaml.v3` + `os.Getenv()` + Cobra flags directly.

### Challenge 15: ID argument patterns are consistent

Positional args for the primary resource being acted on. Flags for filters, parent references, and optional parameters. Examples:
- `incident get <id>` -- positional (primary resource)
- `incident ack <id>` -- positional (primary resource)
- `status-page list --id 123` -- flag (optional filter)
- `escalation-rule list --channel <id>` -- flag (parent resource)

**Resolution:** No change needed. The pattern is already consistent.

### Challenge 16: Module path has uppercase `CLI`

Go module paths are case-sensitive. Uppercase `CLI` is unconventional and can cause issues on case-insensitive filesystems (macOS default).

**Resolution:** Rename repo to `flashduty-cli` (lowercase). Module path: `github.com/flashcatcloud/flashduty-cli`.

### Challenge 17: No pagination in the SDK

The SDK hardcodes `p=1` for incidents, members, changes, and teams. The API backend supports pagination (`p` + `limit` params) but the SDK doesn't expose it.

**Resolution:** Fix the SDK first -- add `Page int` field to `ListIncidentsInput`, `ListMembersInput`, `ListChangesInput`, `ListTeamsInput`. Replace hardcoded `"p": 1` with `"p": input.Page` (default to 1 if 0). Then add `--page` flag to the corresponding CLI commands.

---

## Pre-requisite: SDK Changes

Before starting CLI implementation, the following SDK changes are needed:

### SDK Change 1: Add pagination support

Add `Page int` field to these input structs and wire it through to the API request body:
- `ListIncidentsInput` (incidents.go, `fetchIncidentsByFilters`, hardcoded `"p": 1`)
- `ListMembersInput` (members.go, hardcoded `"p": 1`)
- `ListChangesInput` (changes.go, hardcoded `"p": 1`)
- `ListTeamsInput` (teams.go, hardcoded `"p": 1`)

For each, replace `"p": 1` with:
```go
page := input.Page
if page <= 0 {
    page = 1
}
// use page in request body: "p": page
```

Also ensure the output structs return `Total int` so the CLI can display "Showing page X of Y" or "N total results".

### SDK Change 2: (Tracked, not blocking) Type `CreateIncident` and `CreateStatusIncident` returns

Currently return `(any, error)`. Should return typed response structs. Not blocking CLI v1 -- CLI uses type assertion as workaround.

---

## Revised Command Map

```
flashduty login                                    # interactive auth
flashduty config show                              # print resolved config
flashduty config set <key> <value>                 # set config value

flashduty incident list [--progress ...] [--severity ...] [--channel ...] [--since ...] [--until ...] [--title ...] [--limit ...] [--page ...]
flashduty incident get <id> [<id2> ...]            # get by ID(s), vertical detail for single ID
flashduty incident create --title ... --severity ... [--channel ...] [--description ...] [--assign ...]
flashduty incident update <id> [--title ...] [--description ...] [--severity ...] [--field key=value ...]
flashduty incident ack <id> [<id2> ...]            # acknowledge
flashduty incident close <id> [<id2> ...]          # close/resolve
flashduty incident timeline <id>                   # view timeline
flashduty incident alerts <id> [--limit ...]       # view alerts
flashduty incident similar <id> [--limit ...]      # find similar

flashduty change list [--channel ...] [--since ...] [--until ...] [--type ...] [--limit ...] [--page ...]

flashduty member list [--name ...] [--email ...] [--page ...]
flashduty team list [--name ...] [--page ...]

flashduty channel list [--name ...]
flashduty escalation-rule list --channel <id> | --channel-name <name>

flashduty field list [--name ...]

flashduty status-page list [--id ...]
flashduty status-page changes --page-id <id> --type <incident|maintenance>
flashduty status-page create-incident --page-id <id> --title ... [--message ...] [--impact ...] [--components ...] [--notify]
flashduty status-page create-timeline --page-id <id> --change <id> --message ... [--status ...]

flashduty template get-preset --channel <channel>
flashduty template validate --channel <channel> --file <path> [--incident <id>]
flashduty template variables [--category ...]
flashduty template functions [--type custom|sprig|all]

flashduty version                                  # print version info
```

---

## Project Structure

```
flashduty-cli/
├── cmd/
│   └── flashduty/
│       └── main.go                     # Entry point, version info via ldflags
├── internal/
│   ├── cli/
│   │   ├── root.go                     # Root command, global flags (--json, --no-trunc, --app-key, --base-url), SDK client init
│   │   ├── version.go                  # flashduty version
│   │   ├── login.go                    # flashduty login
│   │   ├── config.go                   # flashduty config {show,set}
│   │   ├── incident.go                 # flashduty incident {list,get,create,update,ack,close,timeline,alerts,similar}
│   │   ├── change.go                   # flashduty change {list}
│   │   ├── member.go                   # flashduty member {list}
│   │   ├── team.go                     # flashduty team {list}
│   │   ├── channel.go                  # flashduty channel {list}
│   │   ├── escalation_rule.go          # flashduty escalation-rule {list}
│   │   ├── field.go                    # flashduty field {list}
│   │   ├── status_page.go             # flashduty status-page {list,changes,create-incident,create-timeline}
│   │   └── template.go                # flashduty template {get-preset,validate,variables,functions}
│   ├── config/
│   │   └── config.go                   # Load/save ~/.flashduty/config.yaml, env vars, flag override
│   ├── output/
│   │   ├── printer.go                  # Printer interface + factory (table vs JSON)
│   │   ├── table.go                    # Table formatter (tabwriter, truncation, --no-trunc support)
│   │   └── json.go                     # JSON formatter (encoding/json, always full/untruncated)
│   └── timeutil/
│       └── parse.go                    # Parse durations ("24h"), dates, datetimes, unix timestamps
├── go.mod
├── go.sum
├── Makefile
├── .goreleaser.yml
└── .github/
    └── workflows/
        ├── ci.yml
        └── release.yml
```

---

## Implementation Steps

### Phase 1: Project Skeleton

#### Step 1.1: Initialize Go module

```bash
cd ~/go/src/github.com/flashcatcloud/flashduty-cli
go mod init github.com/flashcatcloud/flashduty-cli
```

Add dependencies:
```bash
go get github.com/flashcatcloud/flashduty-sdk@v0.3.1
go get github.com/spf13/cobra@latest
go get gopkg.in/yaml.v3@latest
go get golang.org/x/term@latest
```

Note: Viper is dropped (Challenge 14, pending). Use `gopkg.in/yaml.v3` + `os.Getenv()` + Cobra flags directly.

#### Step 1.2: `cmd/flashduty/main.go`

Minimal entry point:
```go
package main

import (
    "os"
    "github.com/flashcatcloud/flashduty-cli/internal/cli"
)

// Set via ldflags at build time
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

func main() {
    cli.SetVersionInfo(version, commit, date)
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}
```

#### Step 1.3: `internal/config/config.go`

Config loading logic:

```go
type Config struct {
    AppKey  string `yaml:"app_key"`
    BaseURL string `yaml:"base_url"`
}
```

Key behaviors:
- `Load() (*Config, error)` -- reads from (in order): flag > env > file
- `Save(cfg *Config) error` -- writes `~/.flashduty/config.yaml` with 0600 perms
- `ConfigDir() string` -- returns `~/.flashduty`, creates if not exists
- `ConfigPath() string` -- returns `~/.flashduty/config.yaml`
- Default `BaseURL`: `https://api.flashcat.cloud`
- Env vars: `FLASHDUTY_APP_KEY`, `FLASHDUTY_BASE_URL`
- No Viper -- use `gopkg.in/yaml.v3` for file read/write, `os.Getenv()` for env vars

#### Step 1.4: `internal/output/printer.go`

Output abstraction:

```go
type Printer interface {
    Print(data any, columns []Column) error
}

type Column struct {
    Header    string                 // Display header: "ID", "TITLE", etc.
    Field     func(item any) string  // Extract field value from item
    MaxWidth  int                    // Max width for truncation (0 = no limit)
}

func NewPrinter(jsonMode bool, noTrunc bool, w io.Writer) Printer
```

`internal/output/table.go`:
- Uses `text/tabwriter`
- Prints header row, data rows, and footer count
- Truncates fields to `Column.MaxWidth` unless `--no-trunc` is set
- `truncate(s string, maxLen int) string` -- appends "..." when truncated
- Formats unix timestamps as `2006-01-02 15:04` in local timezone

`internal/output/json.go`:
- Uses `encoding/json` with `MarshalIndent` for readability
- Always prints full data (ignores truncation settings)

#### Step 1.5: `internal/timeutil/parse.go`

Time parsing utility:

```go
// Parse converts time strings to unix timestamps (seconds).
// Supported formats:
//   - Go duration (relative to now): "5m", "1h", "24h", "168h"
//     Interpreted as "now minus duration"
//   - Date: "2026-04-01" (parsed as local midnight)
//   - Datetime: "2026-04-01 10:00:00" (parsed as local time)
//   - Unix timestamp: "1712000000" (passed through)
func Parse(s string) (int64, error)
```

Uses `time.ParseDuration` for relative formats. No custom shorthands.

#### Step 1.6: `internal/cli/root.go`

Root command setup:

```go
var rootCmd = &cobra.Command{
    Use:   "flashduty",
    Short: "FlashDuty CLI - incident management from your terminal",
}
```

Global persistent flags:
- `--json` (bool) -- output as JSON instead of table
- `--no-trunc` (bool) -- do not truncate table output
- `--app-key` (string, hidden) -- override app key for this command
- `--base-url` (string) -- override base URL

Helper function shared by all subcommands:
```go
// newClient creates a flashduty SDK client from resolved config.
func newClient() (*flashduty.Client, error) {
    cfg, err := config.Load()
    // apply --app-key and --base-url flag overrides if set
    // return flashduty.NewClient(cfg.AppKey, flashduty.WithBaseURL(cfg.BaseURL))
}
```

Register all subcommands:
```go
func init() {
    rootCmd.AddCommand(newLoginCmd())
    rootCmd.AddCommand(newConfigCmd())
    rootCmd.AddCommand(newIncidentCmd())
    rootCmd.AddCommand(newChangeCmd())
    rootCmd.AddCommand(newMemberCmd())
    rootCmd.AddCommand(newTeamCmd())
    rootCmd.AddCommand(newChannelCmd())
    rootCmd.AddCommand(newEscalationRuleCmd())
    rootCmd.AddCommand(newFieldCmd())
    rootCmd.AddCommand(newStatusPageCmd())
    rootCmd.AddCommand(newTemplateCmd())
    rootCmd.AddCommand(newVersionCmd())
}
```

#### Step 1.7: `internal/cli/version.go`

```
$ flashduty version
flashduty version 0.1.0 (abc1234) built 2026-04-10
```

#### Step 1.8: `internal/cli/login.go`

`flashduty login` implementation:
1. Prompt: "Enter your FlashDuty App Key:" (masked with `golang.org/x/term` ReadPassword)
2. Create SDK client with the key
3. Call `client.ListMembers(ctx, &ListMembersInput{})` with a short timeout
4. On success: save to config, print "Logged in successfully. Account has N members."
5. On failure: print error, exit 1

#### Step 1.9: `internal/cli/config.go`

`flashduty config show`:
- Print current config with key partially masked: `app_key: fd_xxxx...xxxx`
- Show resolution source: "(from env)" or "(from ~/.flashduty/config.yaml)" or "(from --app-key flag)"

`flashduty config set <key> <value>`:
- Supported keys: `app_key`, `base_url`
- Validates key name, writes to config file

#### Step 1.10: Verify Phase 1

```bash
go build ./...
go vet ./...
./bin/flashduty version
./bin/flashduty --help
```

At this point: login, config, version work. No resource commands yet.

---

### Phase 2: Resource Commands

Each command file follows the same pattern:

1. Parent command (noun): `newIncidentCmd() *cobra.Command`
2. Subcommands (verbs): registered in parent's `init()`
3. Each subcommand's `RunE`:
   a. Parse flags
   b. Call `newClient()`
   c. Call SDK method
   d. Call `output.Print()`

#### Step 2.1: `internal/cli/incident.go`

The largest file. 9 subcommands:

**`incident list`**
```
Flags:
  --progress string    Filter by progress: Triggered,Processing,Closed (comma-sep)
  --severity string    Filter by severity: Critical,Warning,Info (comma-sep)
  --channel  int64     Filter by channel ID
  --title    string    Search by title keyword
  --since    string    Start time (default "24h", Go duration or absolute)
  --until    string    End time (default "now")
  --limit    int       Max results (default 20, max 100)
  --page     int       Page number (default 1)
```

Table columns: `ID | TITLE | SEVERITY | PROGRESS | CHANNEL | CREATED`

Title column: truncated at 50 chars by default, full with `--no-trunc`.

Footer: "Showing N results (page X, total Y)."

SDK call: `client.ListIncidents(ctx, &ListIncidentsInput{IncludeAlerts: false, ...})`

Note: `IncludeAlerts` set to `false` for list (performance). Use `incident get` for full details with alerts.

Time handling: parse `--since` and `--until` via `timeutil.Parse()`. Default since=24h ago, until=now.

**`incident get <id> [<id2> ...]`**
```
Args: one or more incident IDs (required)
```

SDK call: `client.ListIncidents(ctx, &ListIncidentsInput{IncidentIDs: args, IncludeAlerts: true})`

For single ID, display detailed vertical layout:
```
ID:            inc_abc123
Title:         DB connection timeout
Severity:      Critical
Progress:      Triggered
Channel:       Production (123)
Created:       2026-04-10 10:23:00
Creator:       John Smith (john@example.com)
Responders:    Jane Doe, Bob Lee
Description:   The database connection pool is exhausted...
Labels:        env=prod, region=us-east-1
Custom Fields: priority=P1
Alerts:        3 total
```

For multiple IDs, display table (same as `list`).

**`incident create`**
```
Flags:
  --title       string    Incident title (required, 3-200 chars)
  --severity    string    Severity: Critical,Warning,Info (required)
  --channel     int64     Channel ID (optional)
  --description string    Description (optional, max 6144 chars)
  --assign      []int     Person IDs to assign (optional, comma-sep). Use 'flashduty member list' to look up IDs.
```

Interactive mode: if `--title` or `--severity` is missing and stdin is a terminal, prompt interactively:
```
Title: _
Severity (Critical/Warning/Info): _
```

SDK call: `client.CreateIncident(ctx, &CreateIncidentInput{...})`

Response handling: SDK returns `(any, error)`. Use type assertion to extract incident ID:
```go
if result != nil {
    if m, ok := result.(map[string]any); ok {
        if id, ok := m["incident_id"]; ok {
            fmt.Printf("Incident created: %v\n", id)
        }
    }
}
```

On success: print "Incident created: <id>." or "Incident created successfully." if ID extraction fails.

**`incident update <id>`**
```
Args: exactly one incident ID (required)
Flags:
  --title       string    New title (optional)
  --description string    New description (optional)
  --severity    string    New severity (optional)
  --field       []string  Custom field: key=value (repeatable)
```

SDK call: `client.UpdateIncident(ctx, &UpdateIncidentInput{...})`

Parse `--field` flags into `map[string]any`.

On success: print "Updated incident <id>: <list of updated fields>."

**`incident ack <id> [<id2> ...]`**
```
Args: one or more incident IDs (required)
```

SDK call: `client.AckIncidents(ctx, ids)`

Output: "Acknowledged N incident(s)."

**`incident close <id> [<id2> ...]`**
```
Args: one or more incident IDs (required)
```

SDK call: `client.CloseIncidents(ctx, ids)`

Output: "Closed N incident(s)."

**`incident timeline <id>`**
```
Args: exactly one incident ID (required)
```

SDK call: `client.GetIncidentTimelines(ctx, []string{id})`

Table columns: `TIME | TYPE | OPERATOR | DETAIL`

**`incident alerts <id>`**
```
Args: exactly one incident ID
Flags:
  --limit int    Max alerts (default 10)
```

SDK call: `client.ListIncidentAlerts(ctx, []string{id}, limit)`

Table columns: `ALERT_ID | TITLE | SEVERITY | STATUS | STARTED`

**`incident similar <id>`**
```
Args: exactly one incident ID
Flags:
  --limit int    Max results (default 5)
```

SDK call: `client.ListSimilarIncidents(ctx, id, limit)`

Table columns: same as `incident list`

#### Step 2.2: `internal/cli/change.go`

**`change list`**
```
Flags:
  --channel  int64     Filter by channel ID
  --since    string    Start time (default "24h")
  --until    string    End time (default "now")
  --type     string    Filter by change type
  --limit    int       Max results (default 20, max 100)
  --page     int       Page number (default 1)
```

Table columns: `ID | TITLE | TYPE | STATUS | CHANNEL | TIME`

SDK call: `client.ListChanges(ctx, &ListChangesInput{...})`

#### Step 2.3: `internal/cli/member.go`

**`member list`**
```
Flags:
  --name   string    Search by name
  --email  string    Search by email
  --page   int       Page number (default 1)
```

Table columns: `ID | NAME | EMAIL | STATUS | TIMEZONE`

SDK call: `client.ListMembers(ctx, &ListMembersInput{...})`

Note: SDK returns either `PersonInfos` (when queried by IDs) or `Members` (when queried by name/email). The CLI should handle both cases and normalize to a single table format.

#### Step 2.4: `internal/cli/team.go`

**`team list`**
```
Flags:
  --name string    Search by name
  --page int       Page number (default 1)
```

Table columns: `ID | NAME | MEMBERS`

Members column: comma-separated member names, truncated at 50 chars (respects `--no-trunc`).

SDK call: `client.ListTeams(ctx, &ListTeamsInput{...})`

#### Step 2.5: `internal/cli/channel.go`

**`channel list`**
```
Flags:
  --name string    Search by name
```

Table columns: `ID | NAME | TEAM | CREATOR`

SDK call: `client.ListChannels(ctx, &ListChannelsInput{...})`

#### Step 2.6: `internal/cli/escalation_rule.go`

**`escalation-rule list`**
```
Flags:
  --channel      int64     Channel ID (required unless --channel-name given)
  --channel-name string    Channel name (resolved to ID via ListChannels)
```

Channel name resolution:
1. Call `client.ListChannels(ctx, &ListChannelsInput{Name: name})`
2. If 0 matches: error "no channel found matching '<name>'"
3. If 1 match: use that channel ID
4. If multiple matches: list all matches, error "multiple channels match '<name>', use --channel <id> to specify"

Table columns: `ID | NAME | CHANNEL | STATUS | PRIORITY | LAYERS`

Layers column: count of layers.

SDK call: `client.ListEscalationRules(ctx, channelID)`

#### Step 2.7: `internal/cli/field.go`

**`field list`**
```
Flags:
  --name string    Filter by field name
```

Table columns: `ID | NAME | DISPLAY_NAME | TYPE | OPTIONS`

SDK call: `client.ListFields(ctx, &ListFieldsInput{...})`

#### Step 2.8: `internal/cli/status_page.go`

**`status-page list`**
```
Flags:
  --id []int64    Filter by page IDs (optional, lists all if empty)
```

Table columns: `ID | NAME | SLUG | STATUS | COMPONENTS`

SDK call: `client.ListStatusPages(ctx, pageIDs)`

Empty slice lists all pages (verified SDK behavior).

**`status-page changes`**
```
Flags:
  --page-id int64     Page ID (required)
  --type    string    Change type: incident or maintenance (required)
```

Table columns: `ID | TITLE | TYPE | STATUS | CREATED | UPDATED`

SDK call: `client.ListStatusChanges(ctx, &ListStatusChangesInput{...})`

**`status-page create-incident`**
```
Flags:
  --page-id    int64     Page ID (required)
  --title      string    Title (required, max 255 chars)
  --message    string    Initial update message (optional)
  --impact     string    Impact level (optional)
  --components string    Affected components: "id1:degraded,id2:partial_outage" (optional)
  --notify               Notify subscribers (optional, default false)
```

SDK call: `client.CreateStatusIncident(ctx, &CreateStatusIncidentInput{...})`

**`status-page create-timeline`**
```
Flags:
  --page-id int64     Page ID (required)
  --change  int64     Change ID (required)
  --message string    Message (required)
  --status  string    Status (optional)
```

SDK call: `client.CreateChangeTimeline(ctx, &CreateChangeTimelineInput{...})`

#### Step 2.9: `internal/cli/template.go`

**`template get-preset`**
```
Flags:
  --channel string    Notification channel (required). Values: dingtalk, feishu, slack, email, etc.
```

Output: print the template code directly (not table formatted).

SDK call: `client.GetPresetTemplate(ctx, &GetPresetTemplateInput{Channel: channel})`

**`template validate`**
```
Flags:
  --channel  string    Notification channel (required)
  --file     string    Path to template file (required)
  --incident string    Real incident ID for preview (optional, uses mock data if empty)
```

Reads template code from file, calls SDK, prints:
- Success/failure
- Rendered preview
- Rendered size vs size limit
- Any errors or warnings

SDK call: `client.ValidateTemplate(ctx, &ValidateTemplateInput{...})`

**`template variables`**
```
Flags:
  --category string    Filter by category: core, time, people, alerts, labels, context, notification, post_incident
```

Table columns: `NAME | TYPE | DESCRIPTION | EXAMPLE`

Data source: `flashduty.TemplateVariables()` (static, no API call needed)

**`template functions`**
```
Flags:
  --type string    Filter: custom, sprig, or all (default: all)
```

Table columns: `NAME | SYNTAX | DESCRIPTION`

Data source: `flashduty.TemplateCustomFunctions()` + `flashduty.TemplateSprigFunctions()` (static)

#### Step 2.10: Verify Phase 2

```bash
go build ./...
go vet ./...
# Test with a real app key:
export FLASHDUTY_APP_KEY=fd_xxx
./bin/flashduty incident list --since 168h
./bin/flashduty incident get <some-id>
./bin/flashduty member list
./bin/flashduty channel list
./bin/flashduty template variables
```

---

### Phase 3: Build & Distribution

#### Step 3.1: Makefile

```makefile
BINARY=flashduty
VERSION=$(shell git describe --tags --always --dirty)
COMMIT=$(shell git rev-parse --short HEAD)
DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build check lint test

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/flashduty

check: lint test build

lint:
	golangci-lint run ./...

test:
	go test -race -cover ./...

clean:
	rm -rf bin/
```

#### Step 3.2: `.goreleaser.yml`

Standard Go binary release config:
- Targets: linux/darwin/windows x amd64/arm64
- Archive format: tar.gz (unix), zip (windows)
- Homebrew tap: `flashcatcloud/homebrew-tap`
- Changelog: auto-generated from conventional commits

#### Step 3.3: CI workflows

`.github/workflows/ci.yml`:
- Trigger: push + PR to main
- Jobs: lint (golangci-lint), test (go test -race), build (go build)

`.github/workflows/release.yml`:
- Trigger: tag push `v*`
- Job: GoReleaser

---

### Phase 4: Polish (Post-MVP)

These are not needed for v0.1.0 but should be tracked:

1. **Shell completions**: `flashduty completion bash/zsh/fish/powershell`
2. **Man pages**: generated from Cobra
3. **`--output wide`**: extended table with more columns
4. **Color output**: severity-based coloring (red=Critical, yellow=Warning, blue=Info)
5. **`--raw` support**: requires SDK changes to add `SkipEnrich` to input structs
6. **Email-based `--assign`**: resolve emails to person IDs via `ListMembers`
7. **Typed SDK responses**: fix `CreateIncident` and `CreateStatusIncident` to return typed structs

---

## Dependency Summary

| Dependency | Purpose |
|------------|---------|
| `github.com/flashcatcloud/flashduty-sdk` | API client (v0.3.1) |
| `github.com/spf13/cobra` | CLI framework |
| `gopkg.in/yaml.v3` | Config file read/write |
| `golang.org/x/term` | Masked password input for login |

Note: Viper is NOT used. Config resolution is handled directly with yaml.v3 + os.Getenv() + Cobra flags.

---

## File Count

| Phase | Files | Scope |
|-------|-------|-------|
| Phase 1: Skeleton | 8 files | main, root, version, login, config, output (3), timeutil |
| Phase 2: Commands | 9 files | incident, change, member, team, channel, escalation_rule, field, status_page, template |
| Phase 3: Build | 4 files | Makefile, .goreleaser.yml, 2 CI workflows |
| **Total** | **21 files** | |

---

## Success Criteria

1. `go build ./...` and `go vet ./...` clean
2. All 26 commands functional against live FlashDuty API
3. Table output aligned, readable, and truncated by default
4. `--no-trunc` shows full table content
5. `--json` output valid and complete (always untruncated)
6. Login flow works with masked input
7. Config persistence works across sessions
8. Interactive prompting for `incident create` when required flags missing
9. Time parsing works for Go durations, dates, datetimes, and unix timestamps
10. Pagination works for incident, change, member, and team list commands
11. `flashduty --help` and all subcommand help text is clear
12. `flashduty version` shows build info
