# FlashDuty Go SDK Extraction Plan

## Objective

Extract the pure API client logic from `flashduty-mcp-server/pkg/flashduty/` into a standalone Go SDK module at `github.com/flashcatcloud/flashduty-go-sdk`. This SDK will be consumed by both the MCP server and the new FlashDuty CLI.

## Source Codebase

All source code lives at: `~/go/src/github.com/flashcatcloud/flashduty-mcp-server/pkg/flashduty/`

Key source files:
- `client.go` (~8KB) -- HTTP client, request/response handling, zero MCP deps
- `types.go` (~14KB) -- 28+ data structs, zero MCP deps
- `enrichment.go` (~18KB) -- fetch/enrich methods, zero MCP deps
- `format.go` (~4KB) -- JSON/TOON serialization, has MCP dep (CallToolResult return type)
- `incidents.go` (~823 lines) -- incident tool handlers, heavy MCP coupling
- `users.go` (~234 lines) -- user/team tool handlers
- `channels.go` (~454 lines) -- channel/escalation tool handlers
- `statuspage.go` (~392 lines) -- status page tool handlers
- `changes.go` (~173 lines) -- change tool handlers
- `fields.go` (~113 lines) -- field tool handlers
- `tools.go` -- parameter extraction helpers + toolset registration (MCP-only)

## Target Repository

Create `~/go/src/github.com/flashcatcloud/flashduty-go-sdk/` with this structure:

```
flashduty-go-sdk/
├── client.go              # HTTP client (from client.go, decoupled from pkg/trace)
├── client_options.go      # Functional options: WithBaseURL, WithTimeout, WithUserAgent
├── types.go               # All domain types (from types.go, as-is)
├── enrichment.go          # Enrichment logic (from enrichment.go, as-is)
├── format.go              # Serialization (from format.go, return []byte not CallToolResult)
├── incidents.go           # Incident API methods (extracted from incidents.go handlers)
├── changes.go             # Change API methods (extracted from changes.go handlers)
├── statuspage.go          # Status page API methods (extracted from statuspage.go handlers)
├── members.go             # Member API methods (extracted from users.go handlers)
├── teams.go               # Team API methods (extracted from users.go handlers)
├── channels.go            # Channel API methods (extracted from channels.go handlers)
├── escalation_rules.go    # Escalation rule API methods (extracted from channels.go handlers)
├── fields.go              # Field API methods (extracted from fields.go handlers)
├── errors.go              # Error types (from client.go DutyError + error handling)
├── go.mod
├── go.sum
└── README.md
```

## Pre-requisites

1. Create a new empty GitHub repo `flashcatcloud/flashduty-go-sdk`
2. Clone it locally to `~/go/src/github.com/flashcatcloud/flashduty-go-sdk/`
3. Initialize `go mod init github.com/flashcatcloud/flashduty-go-sdk`

## Implementation Steps

### Phase 1: Foundation (client, types, errors)

#### Step 1.1: Initialize the module

```bash
cd ~/go/src/github.com/flashcatcloud/flashduty-go-sdk
go mod init github.com/flashcatcloud/flashduty-go-sdk
```

Go version should match MCP server (1.24.x).

#### Step 1.2: Copy `types.go` as-is

Source: `flashduty-mcp-server/pkg/flashduty/types.go`

This file has zero MCP dependencies. Copy it, change the package declaration from `package flashduty` to `package flashduty` (same name, just verify). All 28+ exported structs move as-is:

- `EnrichedIncident`, `EnrichedResponder`, `TimelineEvent`, `AlertPreview`
- `PersonInfo`, `ChannelInfo`, `TeamInfo`, `FieldInfo`
- `EscalationRule` and its nested types (`EscalationLayer`, `EscalationTarget`, etc.)
- `ScheduleInfo`, `StatusPage`, `StatusSection`, `StatusComponent`
- `StatusChange`, `ChangeTimeline`, `Change`
- Raw types: `RawIncident`, `RawResponder`, `RawTimelineItem`

Verify: no imports from `mark3labs/mcp-go` or internal MCP server packages.

#### Step 1.3: Create `errors.go`

Extract from `client.go`:
- `DutyError` struct
- `FlashdutyResponse` struct
- Any error-related helper functions

Also check `flashduty-mcp-server/pkg/errors/` for any Flashduty-specific error types that should move to the SDK.

#### Step 1.4: Create `client.go` with functional options

Extract from source `client.go`:
- `Client` struct (fields: `httpClient`, `baseURL`, `appKey`, `userAgent`)
- `NewClient()` -- refactor to use functional options pattern
- `makeRequest()` -- private HTTP method
- `parseResponse()` -- private response parsing
- `handleAPIError()` -- private error handling
- `sanitizeURL()`, `sanitizeError()` -- utility functions

**Critical refactoring needed:**
1. **Remove `pkg/trace` dependency**: The source client.go imports `github.com/flashcatcloud/flashduty-mcp-server/pkg/trace` for W3C Trace Context propagation in HTTP headers. In the SDK, make this optional:
   - Remove the direct import
   - Accept trace context via `context.Context` standard mechanisms, or ignore it
   - Alternatively, accept an optional `http.RoundTripper` wrapper so consumers can inject tracing middleware

2. **Remove `pkg/log` dependency**: The source uses `pkg/mcplog.TruncateBodyDefault()` for log truncation. Replace with:
   - Use `log/slog` from stdlib directly
   - Implement a simple truncation function locally, or drop it

3. **Functional options pattern** for `NewClient`:

```go
type Option func(*Client)

func WithBaseURL(url string) Option {
    return func(c *Client) { c.baseURL = url }
}

func WithTimeout(d time.Duration) Option {
    return func(c *Client) { c.httpClient.Timeout = d }
}

func WithUserAgent(ua string) Option {
    return func(c *Client) { c.userAgent = ua }
}

func WithHTTPClient(hc *http.Client) Option {
    return func(c *Client) { c.httpClient = hc }
}

func NewClient(appKey string, opts ...Option) *Client {
    c := &Client{
        appKey:     appKey,
        baseURL:    "https://api.flashcat.cloud",
        userAgent:  "flashduty-go-sdk",
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
    for _, opt := range opts {
        opt(c)
    }
    return c
}
```

4. **Response types** used by `client.go` (`FlashdutyResponse`, `MemberListResponse`, etc.) should already be in `types.go` or `errors.go`.

#### Step 1.5: Copy `enrichment.go` as-is

Source: `flashduty-mcp-server/pkg/flashduty/enrichment.go`

This file has zero MCP dependencies. Copy as-is. It contains:
- Private fetch methods: `fetchPersonInfos()`, `fetchTeamInfos()`, `fetchChannelInfos()`, `fetchScheduleInfos()`, `fetchIncidentTimeline()`, `fetchIncidentAlerts()`
- Private enrichment methods: `enrichIncidents()`, `enrichChannels()`, `enrichTimelineItems()`
- Concurrent batch fetching with `errgroup`

**Dependency**: Requires `golang.org/x/sync` for `errgroup`. Add to `go.mod`:
```bash
go get golang.org/x/sync
```

Verify all methods are on the `*Client` receiver -- they should be, since they call `c.makeRequest()`.

#### Step 1.6: Create `format.go` (refactored)

Source: `flashduty-mcp-server/pkg/flashduty/format.go`

**Refactoring needed**: The source returns `*mcp.CallToolResult`. SDK should return `([]byte, error)`.

Extract and refactor:
- `OutputFormat` type (JSON, TOON enum)
- `ParseOutputFormat()` -- keep as-is
- `MarshalJSON(v any) ([]byte, error)` -- JSON serialization
- `MarshalTOON(v any) ([]byte, error)` -- TOON serialization
- `Marshal(v any, format OutputFormat) ([]byte, error)` -- dispatch by format

Remove: `SetOutputFormat()`, `GetOutputFormat()` (global state -- let consumers manage format choice).

**Dependency**: `github.com/toon-format/toon-go` for TOON support. Add to `go.mod`:
```bash
go get github.com/toon-format/toon-go
```

#### Step 1.7: Verify Phase 1 compiles

```bash
cd ~/go/src/github.com/flashcatcloud/flashduty-go-sdk
go build ./...
go vet ./...
```

Fix any issues. At this point the SDK has a working HTTP client, all types, enrichment, and serialization -- but no high-level API methods yet.

---

### Phase 2: API Methods (extract from tool handlers)

For each resource, extract the pure API logic from the MCP tool handler functions. The pattern is the same for every resource:

1. Read the MCP server's tool handler function
2. Identify the API call logic (HTTP request construction, parameter mapping, response parsing)
3. Extract that logic into a clean `Client` method with a typed input struct
4. Wire enrichment calls where appropriate

#### Step 2.1: `incidents.go` -- Incident API methods

Source: `flashduty-mcp-server/pkg/flashduty/incidents.go` (~823 lines)

This is the largest and most complex file. Extract these methods:

```go
// Query/List
type ListIncidentsInput struct {
    IDs        []string   // Query by specific IDs
    Status     []string   // Filter: triggered, acknowledged, resolved
    Severity   []string   // Filter: critical, warning, info
    ChannelID  int64      // Filter by channel
    StartTime  int64      // Unix timestamp
    EndTime    int64      // Unix timestamp
    Search     string     // Title search
    Limit      int        // Max results (default 20)
    Enrich     bool       // Whether to enrich with names (default true)
}

func (c *Client) ListIncidents(ctx context.Context, input *ListIncidentsInput) ([]EnrichedIncident, error)
```

The implementation should:
1. If `input.IDs` is set, call `fetchIncidentsByIDs()` (already in enrichment.go or incidents.go)
2. Otherwise, call `fetchIncidentsByFilters()` with the filter params
3. If `input.Enrich` is true (default), call `enrichIncidents()` to resolve person/team/channel names
4. Return enriched incidents

```go
// Timeline
func (c *Client) GetIncidentTimeline(ctx context.Context, incidentID string) ([]TimelineEvent, error)

// Alerts
type ListIncidentAlertsInput struct {
    IncidentID string
    Limit      int    // Default 10
}
func (c *Client) ListIncidentAlerts(ctx context.Context, input *ListIncidentAlertsInput) ([]AlertPreview, int, error)

// Similar
type ListSimilarIncidentsInput struct {
    IncidentID string
    Limit      int    // Default 5
}
func (c *Client) ListSimilarIncidents(ctx context.Context, input *ListSimilarIncidentsInput) ([]EnrichedIncident, error)

// Create
type CreateIncidentInput struct {
    Title       string
    Description string
    Severity    string   // critical, warning, info
    ChannelID   int64
    ResponderIDs []string
}
func (c *Client) CreateIncident(ctx context.Context, input *CreateIncidentInput) (*EnrichedIncident, error)

// Update
type UpdateIncidentInput struct {
    IncidentID  string
    Title       *string
    Description *string
    Severity    *string
    CustomFields map[string]interface{}
}
func (c *Client) UpdateIncident(ctx context.Context, input *UpdateIncidentInput) error

// Ack
func (c *Client) AckIncidents(ctx context.Context, ids []string) error

// Close
func (c *Client) CloseIncidents(ctx context.Context, ids []string) error
```

**Extraction approach for each method:**
1. Find the tool handler (e.g., `QueryIncidents()` returns `(mcp.Tool, server.ToolHandlerFunc)`)
2. Inside the handler closure, find the actual API call logic
3. The handler typically: extracts params from `mcp.CallToolRequest` -> builds request body -> calls `c.makeRequest()` -> parses response -> enriches -> marshals
4. Extract steps 3-5 (build request body -> call API -> parse response -> enrich) into the SDK method
5. Steps 1-2 (param extraction from MCP) stay in the MCP server
6. Step 6 (marshal to CallToolResult) stays in the MCP server

#### Step 2.2: `changes.go` -- Change API methods

Source: `flashduty-mcp-server/pkg/flashduty/changes.go` (~173 lines)

```go
type ListChangesInput struct {
    ChannelID  int64
    StartTime  int64
    EndTime    int64
    Search     string
    Limit      int
}
func (c *Client) ListChanges(ctx context.Context, input *ListChangesInput) ([]Change, error)
```

Simple extraction -- one handler, one API call.

#### Step 2.3: `members.go` -- Member API methods

Source: `flashduty-mcp-server/pkg/flashduty/users.go` (first half, ~120 lines)

```go
type ListMembersInput struct {
    IDs    []string
    Email  string
    Name   string
    Limit  int
}
func (c *Client) ListMembers(ctx context.Context, input *ListMembersInput) ([]MemberItem, error)
```

Note: the source file is `users.go` but it handles both members and teams. Split into two SDK files.

#### Step 2.4: `teams.go` -- Team API methods

Source: `flashduty-mcp-server/pkg/flashduty/users.go` (second half, ~114 lines)

```go
type ListTeamsInput struct {
    IDs   []string
    Name  string
    Limit int
}
func (c *Client) ListTeams(ctx context.Context, input *ListTeamsInput) ([]TeamInfo, error)
```

#### Step 2.5: `channels.go` -- Channel API methods

Source: `flashduty-mcp-server/pkg/flashduty/channels.go` (first portion)

```go
type ListChannelsInput struct {
    IDs    []int64
    Name   string
    Limit  int
    Enrich bool  // Resolve team/creator names (default true)
}
func (c *Client) ListChannels(ctx context.Context, input *ListChannelsInput) ([]ChannelInfo, error)
```

#### Step 2.6: `escalation_rules.go` -- Escalation Rule API methods

Source: `flashduty-mcp-server/pkg/flashduty/channels.go` (second portion)

```go
type ListEscalationRulesInput struct {
    ChannelID int64
}
func (c *Client) ListEscalationRules(ctx context.Context, input *ListEscalationRulesInput) ([]EscalationRule, error)
```

#### Step 2.7: `fields.go` -- Field API methods

Source: `flashduty-mcp-server/pkg/flashduty/fields.go` (~113 lines)

```go
type ListFieldsInput struct {
    Limit int
}
func (c *Client) ListFields(ctx context.Context, input *ListFieldsInput) ([]FieldInfo, error)
```

#### Step 2.8: `statuspage.go` -- Status Page API methods

Source: `flashduty-mcp-server/pkg/flashduty/statuspage.go` (~392 lines)

```go
func (c *Client) ListStatusPages(ctx context.Context, pageIDs []int64) ([]StatusPage, error)

type ListStatusChangesInput struct {
    PageID     int64
    ChangeType string  // incident, maintenance
}
func (c *Client) ListStatusChanges(ctx context.Context, input *ListStatusChangesInput) ([]StatusChange, error)

type CreateStatusIncidentInput struct {
    PageID       int64
    Title        string
    Content      string
    Impact       string
    ComponentIDs []int64
    NotifyEmail  bool
}
func (c *Client) CreateStatusIncident(ctx context.Context, input *CreateStatusIncidentInput) (*StatusChange, error)

type CreateStatusTimelineInput struct {
    ChangeID int64
    Content  string
    Status   string
}
func (c *Client) CreateStatusTimeline(ctx context.Context, input *CreateStatusTimelineInput) error
```

#### Step 2.9: Verify Phase 2 compiles

```bash
cd ~/go/src/github.com/flashcatcloud/flashduty-go-sdk
go build ./...
go vet ./...
go test ./...
```

---

### Phase 3: Tests

#### Step 3.1: Unit tests for client

Test `NewClient` with various options, `sanitizeURL`, `sanitizeError`.

#### Step 3.2: Unit tests for format

Test `MarshalJSON`, `MarshalTOON`, `ParseOutputFormat`.

#### Step 3.3: Unit tests for API methods

For each API method, write table-driven tests using `httptest.Server` to mock the FlashDuty API:
- Verify correct request body construction
- Verify correct response parsing
- Verify enrichment behavior (when enabled/disabled)
- Verify error handling (API errors, network errors)

#### Step 3.4: Verify test coverage

```bash
go test -race -cover ./...
```

Target: 80%+ coverage.

---

### Phase 4: Documentation & Release

#### Step 4.1: Write README.md

Include:
- Installation (`go get github.com/flashcatcloud/flashduty-go-sdk`)
- Quick start example
- Authentication
- Available methods (grouped by resource)
- Output format options

#### Step 4.2: Tag initial release

```bash
git tag v0.1.0
git push origin v0.1.0
```

This makes the module available via `go get`.

---

## Post-Extraction: MCP Server Refactor

After the SDK is published, the MCP server should be refactored to consume it. This is a separate task:

1. Add `require github.com/flashcatcloud/flashduty-go-sdk v0.1.0` to MCP server's `go.mod`
2. Replace direct API calls in tool handlers with SDK client methods
3. Remove duplicated types, client code, and enrichment logic from `pkg/flashduty/`
4. Keep only: tool definitions (mcp.NewTool), parameter extraction, result wrapping, toolset registration
5. Verify all MCP server tests still pass
6. Verify e2e tests still pass

## Key Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Private methods in enrichment.go not accessible | They are on the Client receiver -- ensure they remain accessible when package name stays `flashduty` |
| Breaking MCP server during refactor | SDK extraction is additive; MCP server refactor is a separate step done after SDK is stable |
| API response format undocumented | Copy the exact request/response structures from the MCP server; they are the source of truth |
| `pkg/trace` coupling in client.go | Replace with optional `http.RoundTripper` or remove; trace context can be injected via middleware |
| `pkg/log` coupling in client.go | Replace with stdlib `log/slog`; implement simple body truncation locally |

## Success Criteria

1. `go build ./...` passes with zero MCP dependencies
2. `go vet ./...` clean
3. `go test -race ./...` passes with 80%+ coverage
4. All 16 MCP server tool operations are representable via SDK methods
5. SDK has zero dependency on `mark3labs/mcp-go`
6. SDK dependencies are minimal: stdlib + `golang.org/x/sync` + `toon-go` (optional)
