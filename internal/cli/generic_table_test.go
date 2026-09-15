package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/flashcatcloud/go-flashduty"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

// tableCtx builds a RunContext that renders a human table into buf.
func tableCtx(buf *bytes.Buffer) *RunContext {
	return &RunContext{
		Writer:  buf,
		Printer: output.NewPrinter(output.FormatTable, false, buf),
		Format:  output.FormatTable,
	}
}

func structuredCtx(buf *bytes.Buffer, f output.Format) *RunContext {
	return &RunContext{
		Writer:  buf,
		Printer: output.NewPrinter(f, false, buf),
		Format:  f,
	}
}

// heuristicRow has no displayColumns entry, so it exercises the reflective
// heuristic. The nested field must never become a column.
type heuristicRow struct {
	Name   string `json:"name"`
	Count  int    `json:"count"`
	Nested struct {
		X int `json:"x"`
	} `json:"nested"`
}

type fakeListResp struct {
	Items []heuristicRow `json:"items"`
	Total int            `json:"total"`
}

func TestRenderGenericTable_DisplayColumns(t *testing.T) {
	var buf bytes.Buffer
	resp := &flashduty.IncidentListResponse{
		Items: []flashduty.IncidentInfo{
			{IncidentID: "abc123", Title: "db down", IncidentSeverity: "Critical", Progress: "Triage", ChannelName: "prod-db"},
		},
		Total: 1,
	}
	if err := renderGenericTable(tableCtx(&buf), resp); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"ID", "TITLE", "SEVERITY", "PROGRESS", "CHANNEL", "CREATED", "abc123", "db down", "Critical", "Triage", "prod-db", "Total: 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n---\n%s", want, got)
		}
	}
	// A zero timestamp renders as "-", proving the instant path is reached.
	if !strings.Contains(got, "-") {
		t.Errorf("expected zero StartTime to render as \"-\"\n%s", got)
	}
}

func TestRenderGenericTable_FormattedColumns(t *testing.T) {
	// insight rows carry ratio/seconds fields the curated tables rendered as
	// percent/duration; the colSpec.Format path must apply that, not print the
	// raw float.
	var buf bytes.Buffer
	rows := []flashduty.DimensionInsightItem{
		{TeamName: "sre", TotalIncidentCnt: 4, AcknowledgementPct: 0.85, MeanSecondsToAck: 150},
	}
	if err := renderGenericTable(tableCtx(&buf), rows); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"TEAM", "ACK%", "MTTA", "sre", "85%"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n---\n%s", want, got)
		}
	}
	// The raw ratio must NOT leak — proves Format ran instead of scalarString.
	if strings.Contains(got, "0.85") {
		t.Errorf("raw ratio leaked; Format not applied\n%s", got)
	}
}

func TestRenderGenericTable_Heuristic(t *testing.T) {
	var buf bytes.Buffer
	resp := &fakeListResp{Items: []heuristicRow{{Name: "alpha", Count: 7}}, Total: 1}
	if err := renderGenericTable(tableCtx(&buf), resp); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"NAME", "COUNT", "alpha", "7", "Total: 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n---\n%s", want, got)
		}
	}
	if strings.Contains(got, "NESTED") {
		t.Errorf("nested struct field must not become a column\n%s", got)
	}
}

func TestRenderGenericTable_TopLevelArray(t *testing.T) {
	var buf bytes.Buffer
	rows := []heuristicRow{{Name: "x", Count: 1}, {Name: "y", Count: 2}}
	if err := renderGenericTable(tableCtx(&buf), rows); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"NAME", "COUNT", "x", "y", "Total: 2"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n---\n%s", want, got)
		}
	}
}

func TestRenderGenericTable_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := renderGenericTable(tableCtx(&buf), &fakeListResp{}); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := strings.TrimSpace(buf.String()); got != "No results." {
		t.Errorf("empty list: got %q, want %q", got, "No results.")
	}
}

func TestRenderGenericTable_DetailVertical(t *testing.T) {
	var buf bytes.Buffer
	row := heuristicRow{Name: "solo", Count: 42}
	if err := renderGenericTable(tableCtx(&buf), &row); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"FIELD", "VALUE", "NAME", "solo", "COUNT", "42"} {
		if !strings.Contains(got, want) {
			t.Errorf("vertical output missing %q\n---\n%s", want, got)
		}
	}
}

type detailOwner struct {
	Email string `json:"email"`
}

type detailRecipient struct {
	PersonID int64  `json:"person_id"`
	Reason   string `json:"reason"`
	Status   string `json:"status"`
}

// nestedDetail is a single-object response carrying the non-scalar shapes SDK
// responses use: a nested object, an array of objects, an array of scalars, a
// map, a free-form (any) value, and an unset nested pointer — plus the byte
// slices that are single encoded values rather than arrays (raw JSON, bytes).
type nestedDetail struct {
	Name       string            `json:"name"`
	Owner      detailOwner       `json:"owner"`
	Recipients []detailRecipient `json:"recipients"`
	Tags       []string          `json:"tags"`
	Labels     map[string]string `json:"labels"`
	Payload    any               `json:"payload"`
	Backup     *detailOwner      `json:"backup"`
	Result     json.RawMessage   `json:"result"`
	Digest     []byte            `json:"digest"`
}

// TestRenderGenericTable_DetailShowsNestedFields pins that a single object's
// table output carries every non-empty value, not only its top-level scalars:
// nested fields print under their path, so nothing json/toon would show is
// missing from the table.
func TestRenderGenericTable_DetailShowsNestedFields(t *testing.T) {
	var buf bytes.Buffer
	resp := &nestedDetail{
		Name:  "db-rollback",
		Owner: detailOwner{Email: "sre@example.com"},
		Recipients: []detailRecipient{
			{PersonID: 1, Status: "accepted"},
			{PersonID: 2, Status: "skipped", Reason: "no_email"},
		},
		Tags:    []string{"prod", "db"},
		Labels:  map[string]string{"service": "api", "env": "prod"},
		Payload: map[string]any{"window": []any{"22:00", "23:00"}},
		Result:  json.RawMessage(`{ "rows": [1, 2] }`),
		Digest:  []byte("hi"),
	}
	if err := renderGenericTable(tableCtx(&buf), resp); err != nil {
		t.Fatalf("render: %v", err)
	}

	rows := [][2]string{
		{"FIELD", "VALUE"},
		{"NAME", "db-rollback"},
		{"OWNER.EMAIL", "sre@example.com"},
		{"RECIPIENTS[0].PERSON_ID", "1"},
		{"RECIPIENTS[0].STATUS", "accepted"},
		{"RECIPIENTS[1].PERSON_ID", "2"},
		{"RECIPIENTS[1].REASON", "no_email"},
		{"RECIPIENTS[1].STATUS", "skipped"},
		{"TAGS[0]", "prod"},
		{"TAGS[1]", "db"},
		{"LABELS.env", "prod"},
		{"LABELS.service", "api"},
		{"PAYLOAD.window[0]", "22:00"},
		{"PAYLOAD.window[1]", "23:00"},
		// Byte slices print as --json encodes them, not one row per byte.
		{"RESULT", `{"rows":[1,2]}`},
		{"DIGEST", "aGk="},
	}
	var want strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&want, "%-25s%s\n", r[0], r[1])
	}
	if got := buf.String(); got != want.String() {
		t.Errorf("detail output mismatch\n got:\n%s\nwant:\n%s", got, want.String())
	}
}

// TestRenderGenericTable_TopLevelScalarArray pins that a bare array of scalars
// (e.g. a list of names) is printed, not treated as a row table it cannot be.
func TestRenderGenericTable_TopLevelScalarArray(t *testing.T) {
	var buf bytes.Buffer
	if err := renderGenericTable(tableCtx(&buf), flashduty.SLSLogstoresResponse{"app-log", "audit-log"}); err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"app-log", "audit-log"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("output missing %q\n---\n%s", want, buf.String())
		}
	}
}

func TestRenderGenericTable_McpServerItemRendersDetail(t *testing.T) {
	var buf bytes.Buffer
	item := &flashduty.McpServerItem{
		ServerID:    "mcp_test",
		ServerName:  "github",
		Description: "GitHub connector",
		AuthMode:    "shared",
		Status:      "enabled",
		Transport:   "streamable-http",
		URL:         "https://mcp.example.com/github",
	}
	if err := renderGenericTable(tableCtx(&buf), item); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := buf.String()
	if strings.Contains(got, "No results.") {
		t.Fatalf("single MCP server item was rendered as an empty list:\n%s", got)
	}
	for _, want := range []string{"FIELD", "VALUE", "SERVER_ID", "mcp_test", "SERVER_NAME", "github"} {
		if !strings.Contains(got, want) {
			t.Errorf("MCP server detail output missing %q\n---\n%s", want, got)
		}
	}
}

func TestRenderGenericTable_McpServerPerUserOAuthNotice(t *testing.T) {
	var buf bytes.Buffer
	item := &flashduty.McpServerItem{
		ServerID:    "mcp_oauth",
		ServerName:  "github",
		Description: "GitHub connector",
		AuthMode:    "per_user_oauth",
		Status:      "enabled",
		Transport:   "streamable-http",
		URL:         "https://mcp.example.com/github",
	}
	if err := renderGenericTable(tableCtx(&buf), item); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := buf.String()
	for _, want := range []string{
		"registered but not usable until OAuth is completed in Flashduty Plugins -> MCP",
		"tools will not appear until authorized",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("per-user OAuth notice missing %q\n---\n%s", want, got)
		}
	}
}

func TestPrintGenericResult_StructuredUnchanged(t *testing.T) {
	resp := &fakeListResp{Items: []heuristicRow{{Name: "alpha", Count: 7}}, Total: 1}

	// In any structured mode, printGenericResult must produce byte-identical
	// output to the direct printer (the agent path is untouched by the renderer).
	for _, f := range []output.Format{output.FormatJSON, output.FormatTOON} {
		var got, want bytes.Buffer
		if err := printGenericResult(structuredCtx(&got, f), resp); err != nil {
			t.Fatalf("%v render: %v", f, err)
		}
		if err := output.NewPrinter(f, false, &want).Print(resp, nil); err != nil {
			t.Fatalf("%v reference: %v", f, err)
		}
		if got.String() != want.String() {
			t.Errorf("structured output changed for %v\n got:\n%s\nwant:\n%s", f, got.String(), want.String())
		}
	}
}

// displayColumnSamples is one zero value per row type that displayColumns keys.
// The validation test cross-checks this list against displayColumns so a new
// entry without a sample (or vice versa) fails loudly.
var displayColumnSamples = []any{
	flashduty.IncidentInfo{},
	flashduty.PastIncidentItem{},
	flashduty.AlertInfo{},
	flashduty.AlertItem{},
	flashduty.AlertEventItem{},
	flashduty.ChangeItem{},
	flashduty.ChannelItem{},
	flashduty.AutomationRuleItem{},
	flashduty.AutomationRunItem{},
	flashduty.AutomationTemplateItem{},
	flashduty.TeamItem{},
	flashduty.MemberItem{},
	flashduty.FieldItem{},
	flashduty.WarRoomItem{},
	flashduty.WarRoomPersonItem{},
	flashduty.DimensionInsightItem{},
	flashduty.ResponderInsightItem{},
}

// TestDisplayColumns_FieldsResolve guards against typos: every displayColumns
// field name must resolve on its row type (the names are reflection strings, so
// the compiler can't catch a typo), and every key must have a sample type.
func TestDisplayColumns_FieldsResolve(t *testing.T) {
	typeByName := make(map[string]reflect.Type, len(displayColumnSamples))
	for _, s := range displayColumnSamples {
		ty := reflect.TypeOf(s)
		typeByName[ty.Name()] = ty
	}

	for name, specs := range displayColumns {
		ty, ok := typeByName[name]
		if !ok {
			t.Errorf("displayColumns[%q] has no sample in displayColumnSamples", name)
			continue
		}
		for _, s := range specs {
			if _, ok := ty.FieldByName(s.Field); !ok {
				t.Errorf("%s: field %q (header %q) does not exist", name, s.Field, s.Header)
			}
		}
	}

	for name := range typeByName {
		if _, ok := displayColumns[name]; !ok {
			t.Errorf("sample type %q is not used by displayColumns (remove it or add columns)", name)
		}
	}
}
