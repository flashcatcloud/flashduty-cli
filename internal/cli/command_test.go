package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
)

// mockClient provides default "not implemented" stubs for all flashdutyClient
// methods. Embed it in per-test mocks and override only the methods under test.
type mockClient struct{}

func (m *mockClient) ListIncidents(context.Context, *flashduty.ListIncidentsInput) (*flashduty.ListIncidentsOutput, error) {
	return nil, fmt.Errorf("mockClient: ListIncidents not implemented")
}

func (m *mockClient) GetIncidentTimelines(context.Context, []string) ([]flashduty.IncidentTimelineOutput, error) {
	return nil, fmt.Errorf("mockClient: GetIncidentTimelines not implemented")
}

func (m *mockClient) ListIncidentAlerts(context.Context, []string, int) ([]flashduty.IncidentAlertsOutput, error) {
	return nil, fmt.Errorf("mockClient: ListIncidentAlerts not implemented")
}

func (m *mockClient) ListSimilarIncidents(context.Context, string, int) (*flashduty.ListIncidentsOutput, error) {
	return nil, fmt.Errorf("mockClient: ListSimilarIncidents not implemented")
}

func (m *mockClient) CreateIncident(context.Context, *flashduty.CreateIncidentInput) (any, error) {
	return nil, fmt.Errorf("mockClient: CreateIncident not implemented")
}

func (m *mockClient) UpdateIncident(context.Context, *flashduty.UpdateIncidentInput) ([]string, error) {
	return nil, fmt.Errorf("mockClient: UpdateIncident not implemented")
}

func (m *mockClient) AckIncidents(context.Context, []string) error {
	return fmt.Errorf("mockClient: AckIncidents not implemented")
}

func (m *mockClient) CloseIncidents(context.Context, []string) error {
	return fmt.Errorf("mockClient: CloseIncidents not implemented")
}

func (m *mockClient) ListChannels(context.Context, *flashduty.ListChannelsInput) (*flashduty.ListChannelsOutput, error) {
	return nil, fmt.Errorf("mockClient: ListChannels not implemented")
}

func (m *mockClient) ListTeams(context.Context, *flashduty.ListTeamsInput) (*flashduty.ListTeamsOutput, error) {
	return nil, fmt.Errorf("mockClient: ListTeams not implemented")
}

func (m *mockClient) ListMembers(context.Context, *flashduty.ListMembersInput) (*flashduty.ListMembersOutput, error) {
	return nil, fmt.Errorf("mockClient: ListMembers not implemented")
}

func (m *mockClient) ListEscalationRules(context.Context, int64) (*flashduty.ListEscalationRulesOutput, error) {
	return nil, fmt.Errorf("mockClient: ListEscalationRules not implemented")
}

func (m *mockClient) ListFields(context.Context, *flashduty.ListFieldsInput) (*flashduty.ListFieldsOutput, error) {
	return nil, fmt.Errorf("mockClient: ListFields not implemented")
}

func (m *mockClient) ListChanges(context.Context, *flashduty.ListChangesInput) (*flashduty.ListChangesOutput, error) {
	return nil, fmt.Errorf("mockClient: ListChanges not implemented")
}

func (m *mockClient) GetPresetTemplate(context.Context, *flashduty.GetPresetTemplateInput) (*flashduty.GetPresetTemplateOutput, error) {
	return nil, fmt.Errorf("mockClient: GetPresetTemplate not implemented")
}

func (m *mockClient) ValidateTemplate(context.Context, *flashduty.ValidateTemplateInput) (*flashduty.ValidateTemplateOutput, error) {
	return nil, fmt.Errorf("mockClient: ValidateTemplate not implemented")
}

func (m *mockClient) ListStatusPages(context.Context, []int64) ([]flashduty.StatusPage, error) {
	return nil, fmt.Errorf("mockClient: ListStatusPages not implemented")
}

func (m *mockClient) ListStatusChanges(context.Context, *flashduty.ListStatusChangesInput) (*flashduty.ListStatusChangesOutput, error) {
	return nil, fmt.Errorf("mockClient: ListStatusChanges not implemented")
}

func (m *mockClient) CreateStatusIncident(context.Context, *flashduty.CreateStatusIncidentInput) (any, error) {
	return nil, fmt.Errorf("mockClient: CreateStatusIncident not implemented")
}

func (m *mockClient) CreateChangeTimeline(context.Context, *flashduty.CreateChangeTimelineInput) error {
	return fmt.Errorf("mockClient: CreateChangeTimeline not implemented")
}

// saveAndResetGlobals saves the current state of all global vars that commands
// mutate, resets them to safe defaults, and returns a restore function for
// t.Cleanup.
func saveAndResetGlobals(t *testing.T) {
	t.Helper()

	origNewClientFn := newClientFn
	origFlagJSON := flagJSON
	origFlagNoTrunc := flagNoTrunc
	origFlagAppKey := flagAppKey
	origFlagBaseURL := flagBaseURL

	// Reset to defaults so tests start clean.
	flagJSON = false
	flagNoTrunc = false
	flagAppKey = ""
	flagBaseURL = ""

	t.Cleanup(func() {
		newClientFn = origNewClientFn
		flagJSON = origFlagJSON
		flagNoTrunc = origFlagNoTrunc
		flagAppKey = origFlagAppKey
		flagBaseURL = origFlagBaseURL
	})
}

// execCommand sets args on rootCmd, captures stdout to a buffer, runs Execute,
// and returns (stdout string, error). It also resets cobra flag state after
// execution.
func execCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()

	// Reset the persistent flags cobra parsed so subsequent calls within the
	// same test process do not carry stale values.
	rootCmd.SetArgs(nil)
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)

	return buf.String(), err
}

// ---------------------------------------------------------------------------
// Test 191: incident get returns empty results
// ---------------------------------------------------------------------------

type mockGetEmpty struct{ mockClient }

func (m *mockGetEmpty) ListIncidents(_ context.Context, _ *flashduty.ListIncidentsInput) (*flashduty.ListIncidentsOutput, error) {
	return &flashduty.ListIncidentsOutput{Incidents: nil, Total: 0}, nil
}

func TestCommandIncidentGetEmptyResults(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (flashdutyClient, error) { return &mockGetEmpty{}, nil }

	out, err := execCommand("incident", "get", "nonexistent-id")
	if err != nil {
		t.Fatalf("[#191] unexpected error: %v", err)
	}

	// The table printer always emits the header row even when there are no data
	// rows. Verify that the header is present and no data rows follow.
	if !strings.Contains(out, "ID") {
		t.Errorf("[#191] expected table header containing 'ID', got:\n%s", out)
	}
	if !strings.Contains(out, "TITLE") {
		t.Errorf("[#191] expected table header containing 'TITLE', got:\n%s", out)
	}

	// The table should contain only the header line (no data rows).
	// Split on newlines, ignoring trailing empty lines.
	lines := trimmedLines(out)
	// The first line is the table header; there may be an additional status line
	// such as "Showing 0 results...". There should be no incident data rows.
	for _, line := range lines[1:] {
		// If a line looks like incident data (starts with a UUID-like string), fail.
		if strings.HasPrefix(line, "nonexistent-id") {
			t.Errorf("[#191] unexpected data row in table output:\n%s", out)
		}
	}
}

// ---------------------------------------------------------------------------
// Test 199: incident create result without incident_id
// ---------------------------------------------------------------------------

type mockCreateNoID struct{ mockClient }

func (m *mockCreateNoID) CreateIncident(_ context.Context, _ *flashduty.CreateIncidentInput) (any, error) {
	// Return a plain string instead of a map with "incident_id".
	return "ok", nil
}

func TestCommandIncidentCreateWithoutIncidentID(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (flashdutyClient, error) { return &mockCreateNoID{}, nil }

	out, err := execCommand("incident", "create", "--title", "Test incident", "--severity", "Warning")
	if err != nil {
		t.Fatalf("[#199] unexpected error: %v", err)
	}

	expected := "Incident created successfully."
	if !strings.Contains(out, expected) {
		t.Errorf("[#199] expected output containing %q, got:\n%s", expected, out)
	}
}

func TestCommandIncidentCreateWithoutIncidentID_JSON(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (flashdutyClient, error) { return &mockCreateNoID{}, nil }

	out, err := execCommand("incident", "create", "--title", "Test incident", "--severity", "Warning", "--json")
	if err != nil {
		t.Fatalf("[#199/json] unexpected error: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		t.Fatalf("[#199/json] failed to parse JSON output: %v\nraw output:\n%s", err, out)
	}
	if parsed["message"] != "Incident created successfully." {
		t.Errorf("[#199/json] expected message %q, got %q", "Incident created successfully.", parsed["message"])
	}
}

// ---------------------------------------------------------------------------
// Test 223: incident timeline empty
// ---------------------------------------------------------------------------

type mockTimelineEmpty struct{ mockClient }

func (m *mockTimelineEmpty) GetIncidentTimelines(_ context.Context, _ []string) ([]flashduty.IncidentTimelineOutput, error) {
	return []flashduty.IncidentTimelineOutput{
		{IncidentID: "test", Timeline: nil},
	}, nil
}

func TestCommandIncidentTimelineEmpty(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (flashdutyClient, error) { return &mockTimelineEmpty{}, nil }

	out, err := execCommand("incident", "timeline", "test")
	if err != nil {
		t.Fatalf("[#223] unexpected error: %v", err)
	}

	expected := "No timeline events."
	if !strings.Contains(out, expected) {
		t.Errorf("[#223] expected output containing %q, got:\n%s", expected, out)
	}
}

// ---------------------------------------------------------------------------
// Test 263: statuspage create-incident result with change_id
// ---------------------------------------------------------------------------

type mockStatusCreateWithID struct{ mockClient }

func (m *mockStatusCreateWithID) CreateStatusIncident(_ context.Context, _ *flashduty.CreateStatusIncidentInput) (any, error) {
	return map[string]any{"change_id": float64(12345)}, nil
}

func TestCommandStatusPageCreateIncidentWithChangeID(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (flashdutyClient, error) { return &mockStatusCreateWithID{}, nil }

	out, err := execCommand("statuspage", "create-incident", "--page-id", "1", "--title", "Outage")
	if err != nil {
		t.Fatalf("[#263] unexpected error: %v", err)
	}

	expected := "Status incident created: 12345"
	if !strings.Contains(out, expected) {
		t.Errorf("[#263] expected output containing %q, got:\n%s", expected, out)
	}
}

func TestCommandStatusPageCreateIncidentWithChangeID_JSON(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (flashdutyClient, error) { return &mockStatusCreateWithID{}, nil }

	out, err := execCommand("statuspage", "create-incident", "--page-id", "1", "--title", "Outage", "--json")
	if err != nil {
		t.Fatalf("[#263/json] unexpected error: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		t.Fatalf("[#263/json] failed to parse JSON output: %v\nraw output:\n%s", err, out)
	}
	if !strings.Contains(parsed["message"], "12345") {
		t.Errorf("[#263/json] expected message containing '12345', got %q", parsed["message"])
	}
}

// ---------------------------------------------------------------------------
// Test 264: statuspage create-incident result without change_id
// ---------------------------------------------------------------------------

type mockStatusCreateNoID struct{ mockClient }

func (m *mockStatusCreateNoID) CreateStatusIncident(_ context.Context, _ *flashduty.CreateStatusIncidentInput) (any, error) {
	return "ok", nil
}

func TestCommandStatusPageCreateIncidentWithoutChangeID(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (flashdutyClient, error) { return &mockStatusCreateNoID{}, nil }

	out, err := execCommand("statuspage", "create-incident", "--page-id", "1", "--title", "Outage")
	if err != nil {
		t.Fatalf("[#264] unexpected error: %v", err)
	}

	expected := "Status incident created successfully."
	if !strings.Contains(out, expected) {
		t.Errorf("[#264] expected output containing %q, got:\n%s", expected, out)
	}
}

func TestCommandStatusPageCreateIncidentWithoutChangeID_JSON(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (flashdutyClient, error) { return &mockStatusCreateNoID{}, nil }

	out, err := execCommand("statuspage", "create-incident", "--page-id", "1", "--title", "Outage", "--json")
	if err != nil {
		t.Fatalf("[#264/json] unexpected error: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		t.Fatalf("[#264/json] failed to parse JSON output: %v\nraw output:\n%s", err, out)
	}
	if parsed["message"] != "Status incident created successfully." {
		t.Errorf("[#264/json] expected message %q, got %q", "Status incident created successfully.", parsed["message"])
	}
}

// ---------------------------------------------------------------------------
// Test 321: member list with PersonInfos
// ---------------------------------------------------------------------------

type mockMemberPersonInfos struct{ mockClient }

func (m *mockMemberPersonInfos) ListMembers(_ context.Context, _ *flashduty.ListMembersInput) (*flashduty.ListMembersOutput, error) {
	return &flashduty.ListMembersOutput{
		PersonInfos: []flashduty.PersonInfo{
			{PersonID: 100, PersonName: "Alice", Email: "alice@example.com"},
			{PersonID: 200, PersonName: "Bob", Email: "bob@example.com"},
		},
		Members: nil,
		Total:   2,
	}, nil
}

func TestCommandMemberListPersonInfos(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (flashdutyClient, error) { return &mockMemberPersonInfos{}, nil }

	out, err := execCommand("member", "list")
	if err != nil {
		t.Fatalf("[#321] unexpected error: %v", err)
	}

	// PersonInfo columns: ID, NAME, EMAIL (not MemberItem's STATUS, TIMEZONE).
	if !strings.Contains(out, "ID") {
		t.Errorf("[#321] expected header 'ID' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "NAME") {
		t.Errorf("[#321] expected header 'NAME' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "EMAIL") {
		t.Errorf("[#321] expected header 'EMAIL' in output, got:\n%s", out)
	}

	// PersonInfo table must NOT contain the MemberItem-specific columns.
	if strings.Contains(out, "STATUS") {
		t.Errorf("[#321] output should not contain 'STATUS' column for PersonInfo view, got:\n%s", out)
	}
	if strings.Contains(out, "TIMEZONE") {
		t.Errorf("[#321] output should not contain 'TIMEZONE' column for PersonInfo view, got:\n%s", out)
	}

	// Verify both persons appear in the output.
	if !strings.Contains(out, "Alice") {
		t.Errorf("[#321] expected 'Alice' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Bob") {
		t.Errorf("[#321] expected 'Bob' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "alice@example.com") {
		t.Errorf("[#321] expected 'alice@example.com' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "bob@example.com") {
		t.Errorf("[#321] expected 'bob@example.com' in output, got:\n%s", out)
	}

	// Verify the total line.
	if !strings.Contains(out, "Total: 2") {
		t.Errorf("[#321] expected 'Total: 2' in output, got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// trimmedLines splits s by newline and drops trailing empty lines.
func trimmedLines(s string) []string {
	raw := strings.Split(s, "\n")
	// Remove trailing empty lines.
	for len(raw) > 0 && strings.TrimSpace(raw[len(raw)-1]) == "" {
		raw = raw[:len(raw)-1]
	}
	return raw
}
