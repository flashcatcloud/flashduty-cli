package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// mockClient provides default "not implemented" stubs for all flashdutyClient
// methods. Embed it in per-test mocks and override only the methods under test.
type mockClient struct{}

func (m *mockClient) GetAccountInfo(context.Context) (*flashduty.AccountInfo, error) {
	return nil, fmt.Errorf("mockClient: GetAccountInfo not implemented")
}

func (m *mockClient) GetMemberInfo(context.Context) (*flashduty.MemberInfo, error) {
	return nil, fmt.Errorf("mockClient: GetMemberInfo not implemented")
}

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

func (m *mockClient) UnackIncidents(context.Context, []string) error {
	return fmt.Errorf("mockClient: UnackIncidents not implemented")
}

func (m *mockClient) CloseIncidents(context.Context, []string) error {
	return fmt.Errorf("mockClient: CloseIncidents not implemented")
}

func (m *mockClient) WakeIncidents(context.Context, []string) error {
	return fmt.Errorf("mockClient: WakeIncidents not implemented")
}

func (m *mockClient) RemoveIncidents(context.Context, []string) error {
	return fmt.Errorf("mockClient: RemoveIncidents not implemented")
}

func (m *mockClient) DisableIncidentMerge(context.Context, []string) error {
	return fmt.Errorf("mockClient: DisableIncidentMerge not implemented")
}

func (m *mockClient) CommentIncidents(context.Context, *flashduty.IncidentCommentInput) error {
	return fmt.Errorf("mockClient: CommentIncidents not implemented")
}

func (m *mockClient) AddIncidentResponders(context.Context, *flashduty.IncidentAddResponderInput) error {
	return fmt.Errorf("mockClient: AddIncidentResponders not implemented")
}

func (m *mockClient) CreateIncidentWarRoom(context.Context, *flashduty.IncidentWarRoomCreateInput) (*flashduty.IncidentWarRoom, error) {
	return nil, fmt.Errorf("mockClient: CreateIncidentWarRoom not implemented")
}

func (m *mockClient) ListIncidentWarRooms(context.Context, *flashduty.IncidentWarRoomListInput) (*flashduty.IncidentWarRoomListOutput, error) {
	return nil, fmt.Errorf("mockClient: ListIncidentWarRooms not implemented")
}

func (m *mockClient) GetIncidentWarRoom(context.Context, *flashduty.IncidentWarRoomDetailInput) (*flashduty.IncidentWarRoom, error) {
	return nil, fmt.Errorf("mockClient: GetIncidentWarRoom not implemented")
}

func (m *mockClient) DeleteIncidentWarRoom(context.Context, *flashduty.IncidentWarRoomDeleteInput) error {
	return fmt.Errorf("mockClient: DeleteIncidentWarRoom not implemented")
}

func (m *mockClient) AddIncidentWarRoomMembers(context.Context, *flashduty.IncidentWarRoomAddMemberInput) error {
	return fmt.Errorf("mockClient: AddIncidentWarRoomMembers not implemented")
}

func (m *mockClient) GetIncidentWarRoomDefaultObservers(context.Context, string) ([]flashduty.IncidentWarRoomObserver, error) {
	return nil, fmt.Errorf("mockClient: GetIncidentWarRoomDefaultObservers not implemented")
}

func (m *mockClient) ListWarRoomEnabledDataSources(context.Context) (*flashduty.ListWarRoomEnabledDataSourcesOutput, error) {
	return nil, fmt.Errorf("mockClient: ListWarRoomEnabledDataSources not implemented")
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

// Phase 1: Incident additions
func (m *mockClient) GetIncidentDetail(context.Context, *flashduty.GetIncidentDetailInput) (*flashduty.GetIncidentDetailOutput, error) {
	return nil, fmt.Errorf("mockClient: GetIncidentDetail not implemented")
}

func (m *mockClient) GetIncidentFeed(context.Context, *flashduty.GetIncidentFeedInput) (*flashduty.GetIncidentFeedOutput, error) {
	return nil, fmt.Errorf("mockClient: GetIncidentFeed not implemented")
}

func (m *mockClient) ListPostMortems(context.Context, *flashduty.ListPostMortemsInput) (*flashduty.ListPostMortemsOutput, error) {
	return nil, fmt.Errorf("mockClient: ListPostMortems not implemented")
}

func (m *mockClient) MergeIncidents(context.Context, *flashduty.MergeIncidentsInput) error {
	return fmt.Errorf("mockClient: MergeIncidents not implemented")
}

func (m *mockClient) SnoozeIncidents(context.Context, *flashduty.SnoozeIncidentsInput) error {
	return fmt.Errorf("mockClient: SnoozeIncidents not implemented")
}

func (m *mockClient) ReopenIncidents(context.Context, []string) error {
	return fmt.Errorf("mockClient: ReopenIncidents not implemented")
}

func (m *mockClient) ReassignIncidents(context.Context, *flashduty.ReassignIncidentsInput) error {
	return fmt.Errorf("mockClient: ReassignIncidents not implemented")
}

// Phase 1: Alert additions
func (m *mockClient) ListAlerts(context.Context, *flashduty.ListAlertsInput) (*flashduty.ListAlertsOutput, error) {
	return nil, fmt.Errorf("mockClient: ListAlerts not implemented")
}

func (m *mockClient) GetAlertDetail(context.Context, *flashduty.GetAlertDetailInput) (*flashduty.GetAlertDetailOutput, error) {
	return nil, fmt.Errorf("mockClient: GetAlertDetail not implemented")
}

func (m *mockClient) ListAlertEvents(context.Context, *flashduty.ListAlertEventsInput) (*flashduty.ListAlertEventsOutput, error) {
	return nil, fmt.Errorf("mockClient: ListAlertEvents not implemented")
}

func (m *mockClient) MergeAlertsToIncident(context.Context, *flashduty.MergeAlertsInput) error {
	return fmt.Errorf("mockClient: MergeAlertsToIncident not implemented")
}

func (m *mockClient) GetAlertFeed(context.Context, *flashduty.GetAlertFeedInput) (*flashduty.GetAlertFeedOutput, error) {
	return nil, fmt.Errorf("mockClient: GetAlertFeed not implemented")
}

func (m *mockClient) ListAlertEventsGlobal(context.Context, *flashduty.ListAlertEventsGlobalInput) (*flashduty.ListAlertEventsGlobalOutput, error) {
	return nil, fmt.Errorf("mockClient: ListAlertEventsGlobal not implemented")
}

// Phase 2: OnCall + Change
func (m *mockClient) ListSchedulesWithSlots(context.Context, *flashduty.ListSchedulesWithSlotsInput) (*flashduty.ListSchedulesWithSlotsOutput, error) {
	return nil, fmt.Errorf("mockClient: ListSchedulesWithSlots not implemented")
}

func (m *mockClient) GetScheduleDetail(context.Context, *flashduty.GetScheduleDetailInput) (*flashduty.GetScheduleDetailOutput, error) {
	return nil, fmt.Errorf("mockClient: GetScheduleDetail not implemented")
}

func (m *mockClient) QueryChangeTrend(context.Context, *flashduty.QueryChangeTrendInput) (*flashduty.QueryChangeTrendOutput, error) {
	return nil, fmt.Errorf("mockClient: QueryChangeTrend not implemented")
}

// Phase 3: Insight + Admin
func (m *mockClient) QueryInsightByTeam(context.Context, *flashduty.InsightQueryInput) (*flashduty.QueryInsightByTeamOutput, error) {
	return nil, fmt.Errorf("mockClient: QueryInsightByTeam not implemented")
}

func (m *mockClient) QueryInsightByChannel(context.Context, *flashduty.InsightQueryInput) (*flashduty.QueryInsightByChannelOutput, error) {
	return nil, fmt.Errorf("mockClient: QueryInsightByChannel not implemented")
}

func (m *mockClient) QueryInsightByResponder(context.Context, *flashduty.InsightQueryInput) (*flashduty.QueryInsightByResponderOutput, error) {
	return nil, fmt.Errorf("mockClient: QueryInsightByResponder not implemented")
}

func (m *mockClient) QueryInsightAlertTopK(context.Context, *flashduty.QueryInsightAlertTopKInput) (*flashduty.QueryInsightAlertTopKOutput, error) {
	return nil, fmt.Errorf("mockClient: QueryInsightAlertTopK not implemented")
}

func (m *mockClient) QueryInsightIncidentList(context.Context, *flashduty.QueryInsightIncidentListInput) (*flashduty.QueryInsightIncidentListOutput, error) {
	return nil, fmt.Errorf("mockClient: QueryInsightIncidentList not implemented")
}

func (m *mockClient) QueryNotificationTrend(context.Context, *flashduty.QueryNotificationTrendInput) (*flashduty.QueryNotificationTrendOutput, error) {
	return nil, fmt.Errorf("mockClient: QueryNotificationTrend not implemented")
}

func (m *mockClient) SearchAuditLogs(context.Context, *flashduty.SearchAuditLogsInput) (*flashduty.SearchAuditLogsOutput, error) {
	return nil, fmt.Errorf("mockClient: SearchAuditLogs not implemented")
}

func (m *mockClient) StartStatusPageMigration(context.Context, *flashduty.StartStatusPageMigrationInput) (*flashduty.StartStatusPageMigrationOutput, error) {
	return nil, fmt.Errorf("mockClient: StartStatusPageMigration not implemented")
}

func (m *mockClient) StartStatusPageEmailSubscriberMigration(context.Context, *flashduty.StartStatusPageEmailSubscriberMigrationInput) (*flashduty.StartStatusPageMigrationOutput, error) {
	return nil, fmt.Errorf("mockClient: StartStatusPageEmailSubscriberMigration not implemented")
}

func (m *mockClient) GetStatusPageMigrationStatus(context.Context, string) (*flashduty.StatusPageMigrationJob, error) {
	return nil, fmt.Errorf("mockClient: GetStatusPageMigrationStatus not implemented")
}

func (m *mockClient) CancelStatusPageMigration(context.Context, string) error {
	return fmt.Errorf("mockClient: CancelStatusPageMigration not implemented")
}

// Phase 5: Team Management
func (m *mockClient) GetTeamInfo(context.Context, *flashduty.TeamGetInput) (*flashduty.TeamItem, error) {
	return nil, fmt.Errorf("mockClient: GetTeamInfo not implemented")
}

func (m *mockClient) UpsertTeam(context.Context, *flashduty.TeamUpsertInput) (*flashduty.TeamUpsertOutput, error) {
	return nil, fmt.Errorf("mockClient: UpsertTeam not implemented")
}

func (m *mockClient) DeleteTeam(context.Context, *flashduty.TeamDeleteInput) error {
	return fmt.Errorf("mockClient: DeleteTeam not implemented")
}

func (m *mockClient) CreateMCPServer(context.Context, *flashduty.CreateMCPServerInput) (*flashduty.CreateMCPServerOutput, error) {
	return nil, fmt.Errorf("mockClient: CreateMCPServer not implemented")
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
	resetCommandFlags(rootCmd)

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
	resetCommandFlags(rootCmd)

	return buf.String(), err
}

func resetCommandFlags(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	resetFlagSet(cmd.Flags())
	resetFlagSet(cmd.PersistentFlags())
	for _, child := range cmd.Commands() {
		resetCommandFlags(child)
	}
}

func resetFlagSet(flags *pflag.FlagSet) {
	if flags == nil {
		return
	}
	flags.VisitAll(func(flag *pflag.Flag) {
		switch flag.Value.Type() {
		case "bool", "int", "int64", "string":
			_ = flag.Value.Set(flag.DefValue)
			flag.Changed = false
		}
	})
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
// Regression tests for new command batch review findings
// ---------------------------------------------------------------------------

type mockIncidentFeedEmpty struct{ mockClient }

func (m *mockIncidentFeedEmpty) GetIncidentFeed(_ context.Context, _ *flashduty.GetIncidentFeedInput) (*flashduty.GetIncidentFeedOutput, error) {
	return &flashduty.GetIncidentFeedOutput{Items: nil, HasNextPage: false}, nil
}

func TestCommandIncidentFeedEmpty_JSON(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (flashdutyClient, error) { return &mockIncidentFeedEmpty{}, nil }

	out, err := execCommand("incident", "feed", "inc-1", "--json")
	if err != nil {
		t.Fatalf("[incident-feed-empty/json] unexpected error: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		t.Fatalf("[incident-feed-empty/json] failed to parse JSON output: %v\nraw output:\n%s", err, out)
	}
	if parsed["message"] != "No feed events." {
		t.Errorf("[incident-feed-empty/json] expected message %q, got %q", "No feed events.", parsed["message"])
	}
}

func TestCommandIncidentSnoozeRejectsSubMinuteDuration(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (flashdutyClient, error) { return &mockClient{}, nil }

	_, err := execCommand("incident", "snooze", "inc-1", "--duration", "90s")
	if err == nil {
		t.Fatal("[incident-snooze-sub-minute] expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "whole minutes") {
		t.Fatalf("[incident-snooze-sub-minute] expected error containing %q, got %q", "whole minutes", err.Error())
	}
}

func TestCommandIncidentSnoozeRejectsDurationOver24Hours(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (flashdutyClient, error) { return &mockClient{}, nil }

	_, err := execCommand("incident", "snooze", "inc-1", "--duration", "25h")
	if err == nil {
		t.Fatal("[incident-snooze-max] expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "24h") {
		t.Fatalf("[incident-snooze-max] expected error containing %q, got %q", "24h", err.Error())
	}
}

func TestCommandIncidentMergeRejectsMoreThan100Sources(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (flashdutyClient, error) { return &mockClient{}, nil }

	sourceIDs := make([]string, 101)
	for i := range sourceIDs {
		sourceIDs[i] = fmt.Sprintf("inc-%d", i+1)
	}

	_, err := execCommand("incident", "merge", "target-1", "--source", strings.Join(sourceIDs, ","))
	if err == nil {
		t.Fatal("[incident-merge-max-sources] expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "at most 100") {
		t.Fatalf("[incident-merge-max-sources] expected error containing %q, got %q", "at most 100", err.Error())
	}
}

func TestCommandIncidentLifecycleHelpDocumentsSafetyAndLookupHints(t *testing.T) {
	saveAndResetGlobals(t)

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "war-room create integration discovery",
			args: []string{"incident", "war-room", "create", "--help"},
			want: []string{
				"If --integration is omitted",
				"first war-room-enabled IM integration",
				"Use 'flashduty member list'",
			},
		},
		{
			name: "war-room get required integration",
			args: []string{"incident", "war-room", "get", "--help"},
			want: []string{
				"requires --integration",
				"Use 'flashduty incident war-room list'",
			},
		},
		{
			name: "remove destructive behavior",
			args: []string{"incident", "remove", "--help"},
			want: []string{
				"Permanently removes incidents",
				"Prompts for confirmation",
				"--force",
			},
		},
		{
			name: "comment limit",
			args: []string{"incident", "comment", "--help"},
			want: []string{
				"up to 100 incidents",
				"1024 characters",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := execCommand(tt.args...)
			if err != nil {
				t.Fatalf("help command returned error: %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Fatalf("help output missing %q:\n%s", want, out)
				}
			}
		})
	}
}

type mockIncidentLifecycle struct {
	mockClient

	unackIDs        []string
	wakeIDs         []string
	removeIDs       []string
	disableMergeIDs []string
	commentInput    *flashduty.IncidentCommentInput
	responderInput  *flashduty.IncidentAddResponderInput
}

func (m *mockIncidentLifecycle) UnackIncidents(_ context.Context, incidentIDs []string) error {
	m.unackIDs = append([]string(nil), incidentIDs...)
	return nil
}

func (m *mockIncidentLifecycle) WakeIncidents(_ context.Context, incidentIDs []string) error {
	m.wakeIDs = append([]string(nil), incidentIDs...)
	return nil
}

func (m *mockIncidentLifecycle) RemoveIncidents(_ context.Context, incidentIDs []string) error {
	m.removeIDs = append([]string(nil), incidentIDs...)
	return nil
}

func (m *mockIncidentLifecycle) DisableIncidentMerge(_ context.Context, incidentIDs []string) error {
	m.disableMergeIDs = append([]string(nil), incidentIDs...)
	return nil
}

func (m *mockIncidentLifecycle) CommentIncidents(_ context.Context, input *flashduty.IncidentCommentInput) error {
	copied := *input
	copied.IncidentIDs = append([]string(nil), input.IncidentIDs...)
	m.commentInput = &copied
	return nil
}

func (m *mockIncidentLifecycle) AddIncidentResponders(_ context.Context, input *flashduty.IncidentAddResponderInput) error {
	copied := *input
	copied.PersonIDs = append([]int64(nil), input.PersonIDs...)
	if input.Notify != nil {
		notify := *input.Notify
		notify.PersonalChannels = append([]string(nil), input.Notify.PersonalChannels...)
		copied.Notify = &notify
	}
	m.responderInput = &copied
	return nil
}

func TestCommandIncidentUnack(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentLifecycle{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("incident", "unack", "inc-1", "inc-2")
	if err != nil {
		t.Fatalf("[incident-unack] unexpected error: %v", err)
	}
	if got, want := strings.Join(mock.unackIDs, ","), "inc-1,inc-2"; got != want {
		t.Fatalf("[incident-unack] expected ids %q, got %q", want, got)
	}
	if !strings.Contains(out, "Unacknowledged 2 incident(s).") {
		t.Fatalf("[incident-unack] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWake(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentLifecycle{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("incident", "wake", "inc-1")
	if err != nil {
		t.Fatalf("[incident-wake] unexpected error: %v", err)
	}
	if got, want := strings.Join(mock.wakeIDs, ","), "inc-1"; got != want {
		t.Fatalf("[incident-wake] expected ids %q, got %q", want, got)
	}
	if !strings.Contains(out, "Restored notifications for 1 incident(s).") {
		t.Fatalf("[incident-wake] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentComment(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentLifecycle{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("incident", "comment", "inc-1", "inc-2", "--comment", "rollback started", "--mute-reply")
	if err != nil {
		t.Fatalf("[incident-comment] unexpected error: %v", err)
	}
	if mock.commentInput == nil {
		t.Fatal("[incident-comment] expected CommentIncidents to be called")
	}
	if got, want := strings.Join(mock.commentInput.IncidentIDs, ","), "inc-1,inc-2"; got != want {
		t.Fatalf("[incident-comment] expected ids %q, got %q", want, got)
	}
	if mock.commentInput.Comment != "rollback started" || !mock.commentInput.MuteReply {
		t.Fatalf("[incident-comment] unexpected input: %#v", mock.commentInput)
	}
	if !strings.Contains(out, "Commented on 2 incident(s).") {
		t.Fatalf("[incident-comment] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentCommentAllows1024UnicodeRunes(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentLifecycle{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	comment := strings.Repeat("界", 1024)
	_, err := execCommand("incident", "comment", "inc-1", "--comment", comment)
	if err != nil {
		t.Fatalf("[incident-comment-unicode] unexpected error: %v", err)
	}
	if mock.commentInput == nil || mock.commentInput.Comment != comment {
		t.Fatalf("[incident-comment-unicode] unexpected input: %#v", mock.commentInput)
	}
}

func TestCommandIncidentLifecycleRejectsMoreThan100IDs(t *testing.T) {
	commands := []struct {
		name string
		args []string
	}{
		{name: "unack", args: []string{"incident", "unack"}},
		{name: "wake", args: []string{"incident", "wake"}},
		{name: "comment", args: []string{"incident", "comment", "--comment", "too many"}},
		{name: "remove", args: []string{"incident", "remove"}},
	}

	incidentIDs := make([]string, 101)
	for i := range incidentIDs {
		incidentIDs[i] = fmt.Sprintf("inc-%d", i+1)
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			mock := &mockIncidentLifecycle{}
			newClientFn = func() (flashdutyClient, error) { return mock, nil }

			args := append([]string(nil), tc.args...)
			args = append(args, incidentIDs...)
			_, err := execCommand(args...)
			if err == nil {
				t.Fatal("expected too-many-ids error, got nil")
			}
			if !strings.Contains(err.Error(), "at most 100 incident IDs") {
				t.Fatalf("expected max-id error, got %q", err.Error())
			}
		})
	}
}

func TestCommandIncidentAddResponder(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentLifecycle{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand(
		"incident", "add-responder", "inc-1",
		"--person", "101,202",
		"--follow-preference",
		"--notify-channel", "voice,sms",
		"--template-id", "6321aad26c12104586a88916",
	)
	if err != nil {
		t.Fatalf("[incident-add-responder] unexpected error: %v", err)
	}
	if mock.responderInput == nil {
		t.Fatal("[incident-add-responder] expected AddIncidentResponders to be called")
	}
	if mock.responderInput.IncidentID != "inc-1" {
		t.Fatalf("[incident-add-responder] expected incident inc-1, got %q", mock.responderInput.IncidentID)
	}
	if got, want := fmt.Sprint(mock.responderInput.PersonIDs), "[101 202]"; got != want {
		t.Fatalf("[incident-add-responder] expected people %q, got %q", want, got)
	}
	if mock.responderInput.Notify == nil || !mock.responderInput.Notify.FollowPreference {
		t.Fatalf("[incident-add-responder] expected follow preference notify, got %#v", mock.responderInput.Notify)
	}
	if got, want := strings.Join(mock.responderInput.Notify.PersonalChannels, ","), "voice,sms"; got != want {
		t.Fatalf("[incident-add-responder] expected channels %q, got %q", want, got)
	}
	if mock.responderInput.Notify.TemplateID != "6321aad26c12104586a88916" {
		t.Fatalf("[incident-add-responder] unexpected template id: %#v", mock.responderInput.Notify)
	}
	if !strings.Contains(out, "Added 2 responder(s) to incident inc-1.") {
		t.Fatalf("[incident-add-responder] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentRemoveRequiresForceWhenNonInteractive(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentLifecycle{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("incident", "remove", "inc-1")
	if err != nil {
		t.Fatalf("[incident-remove-abort] unexpected error: %v", err)
	}
	if len(mock.removeIDs) != 0 {
		t.Fatalf("[incident-remove-abort] remove should not be called, got ids %#v", mock.removeIDs)
	}
	if !strings.Contains(out, "Aborted.") {
		t.Fatalf("[incident-remove-abort] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentRemoveWithForce(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentLifecycle{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("incident", "remove", "inc-1", "inc-2", "--force")
	if err != nil {
		t.Fatalf("[incident-remove-force] unexpected error: %v", err)
	}
	if got, want := strings.Join(mock.removeIDs, ","), "inc-1,inc-2"; got != want {
		t.Fatalf("[incident-remove-force] expected ids %q, got %q", want, got)
	}
	if !strings.Contains(out, "Removed 2 incident(s).") {
		t.Fatalf("[incident-remove-force] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentDisableMerge(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentLifecycle{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("incident", "disable-merge", "inc-1", "inc-2")
	if err != nil {
		t.Fatalf("[incident-disable-merge] unexpected error: %v", err)
	}
	if got, want := strings.Join(mock.disableMergeIDs, ","), "inc-1,inc-2"; got != want {
		t.Fatalf("[incident-disable-merge] expected ids %q, got %q", want, got)
	}
	if !strings.Contains(out, "Disabled auto-merge for 2 incident(s).") {
		t.Fatalf("[incident-disable-merge] unexpected output:\n%s", out)
	}
}

type mockIncidentWarRoom struct {
	mockClient

	createInput           *flashduty.IncidentWarRoomCreateInput
	listInput             *flashduty.IncidentWarRoomListInput
	getInput              *flashduty.IncidentWarRoomDetailInput
	deleteInput           *flashduty.IncidentWarRoomDeleteInput
	addMemberInput        *flashduty.IncidentWarRoomAddMemberInput
	defaultObserverIncID  string
	defaultObserverOutput []flashduty.IncidentWarRoomObserver
	enabledDataSources    []flashduty.DataSourceIntegration
}

func (m *mockIncidentWarRoom) CreateIncidentWarRoom(_ context.Context, input *flashduty.IncidentWarRoomCreateInput) (*flashduty.IncidentWarRoom, error) {
	copied := *input
	copied.MemberIDs = append([]int64(nil), input.MemberIDs...)
	m.createInput = &copied
	return &flashduty.IncidentWarRoom{ChatID: "chat-1", ChatName: "INC outage", ShareLink: "https://chat.example/1"}, nil
}

func (m *mockIncidentWarRoom) ListIncidentWarRooms(_ context.Context, input *flashduty.IncidentWarRoomListInput) (*flashduty.IncidentWarRoomListOutput, error) {
	copied := *input
	m.listInput = &copied
	return &flashduty.IncidentWarRoomListOutput{
		Items: []flashduty.IncidentWarRoomItem{
			{IntegrationID: 42, ChatID: "chat-1", IncidentID: "inc-1", Status: "enabled", PluginType: "feishu"},
		},
	}, nil
}

func (m *mockIncidentWarRoom) GetIncidentWarRoom(_ context.Context, input *flashduty.IncidentWarRoomDetailInput) (*flashduty.IncidentWarRoom, error) {
	copied := *input
	m.getInput = &copied
	return &flashduty.IncidentWarRoom{ChatID: "chat-1", ChatName: "INC outage", ShareLink: "https://chat.example/1"}, nil
}

func (m *mockIncidentWarRoom) DeleteIncidentWarRoom(_ context.Context, input *flashduty.IncidentWarRoomDeleteInput) error {
	copied := *input
	m.deleteInput = &copied
	return nil
}

func (m *mockIncidentWarRoom) AddIncidentWarRoomMembers(_ context.Context, input *flashduty.IncidentWarRoomAddMemberInput) error {
	copied := *input
	copied.MemberIDs = append([]int64(nil), input.MemberIDs...)
	m.addMemberInput = &copied
	return nil
}

func (m *mockIncidentWarRoom) GetIncidentWarRoomDefaultObservers(_ context.Context, incidentID string) ([]flashduty.IncidentWarRoomObserver, error) {
	m.defaultObserverIncID = incidentID
	return m.defaultObserverOutput, nil
}

func (m *mockIncidentWarRoom) ListWarRoomEnabledDataSources(context.Context) (*flashduty.ListWarRoomEnabledDataSourcesOutput, error) {
	return &flashduty.ListWarRoomEnabledDataSourcesOutput{Items: m.enabledDataSources}, nil
}

func TestCommandIncidentWarRoomCreateWithObservers(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentWarRoom{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("incident", "war-room", "create", "inc-1", "--integration", "42", "--member", "101,202", "--add-observers")
	if err != nil {
		t.Fatalf("[incident-war-room-create] unexpected error: %v", err)
	}
	if mock.createInput == nil {
		t.Fatal("[incident-war-room-create] expected CreateIncidentWarRoom to be called")
	}
	if mock.createInput.IncidentID != "inc-1" || mock.createInput.IntegrationID != 42 || !mock.createInput.AddObservers {
		t.Fatalf("[incident-war-room-create] unexpected input: %#v", mock.createInput)
	}
	if got, want := fmt.Sprint(mock.createInput.MemberIDs), "[101 202]"; got != want {
		t.Fatalf("[incident-war-room-create] expected member ids %q, got %q", want, got)
	}
	if !strings.Contains(out, "War room created: chat-1") {
		t.Fatalf("[incident-war-room-create] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWarRoomCreateAutoDiscoversIntegration(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentWarRoom{
		enabledDataSources: []flashduty.DataSourceIntegration{
			{DataSourceID: 42, Name: "Feishu", PluginType: "feishu_app"},
		},
	}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("incident", "war-room", "create", "inc-1", "--member", "101")
	if err != nil {
		t.Fatalf("[incident-war-room-create-autodiscover] unexpected error: %v", err)
	}
	if mock.createInput == nil {
		t.Fatal("[incident-war-room-create-autodiscover] expected CreateIncidentWarRoom to be called")
	}
	if mock.createInput.IntegrationID != 42 {
		t.Fatalf("[incident-war-room-create-autodiscover] expected integration 42, got %#v", mock.createInput)
	}
	if !strings.Contains(out, "War room created: chat-1") {
		t.Fatalf("[incident-war-room-create-autodiscover] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWarRoomCreateRequiresEnabledIntegration(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentWarRoom{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	_, err := execCommand("incident", "war-room", "create", "inc-1")
	if err == nil || !strings.Contains(err.Error(), "no IM integration has war-room enabled") {
		t.Fatalf("[incident-war-room-create-no-enabled-integration] expected enabled integration error, got %v", err)
	}
	if mock.createInput != nil {
		t.Fatalf("[incident-war-room-create-no-enabled-integration] did not expect create call: %#v", mock.createInput)
	}
}

func TestCommandIncidentWarRoomDefaultObservers(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentWarRoom{
		defaultObserverOutput: []flashduty.IncidentWarRoomObserver{
			{PersonID: 101, PersonName: "Alice", Email: "alice@example.com"},
		},
	}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("incident", "war-room", "default-observers", "inc-1")
	if err != nil {
		t.Fatalf("[incident-war-room-default-observers] unexpected error: %v", err)
	}
	if mock.defaultObserverIncID != "inc-1" {
		t.Fatalf("[incident-war-room-default-observers] expected incident inc-1, got %q", mock.defaultObserverIncID)
	}
	if !strings.Contains(out, "Alice") || !strings.Contains(out, "alice@example.com") || !strings.Contains(out, "Total: 1") {
		t.Fatalf("[incident-war-room-default-observers] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWarRoomList(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentWarRoom{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("incident", "war-room", "list", "inc-1", "--integration", "42")
	if err != nil {
		t.Fatalf("[incident-war-room-list] unexpected error: %v", err)
	}
	if mock.listInput == nil || mock.listInput.IncidentID != "inc-1" || mock.listInput.IntegrationID != 42 {
		t.Fatalf("[incident-war-room-list] unexpected input: %#v", mock.listInput)
	}
	if !strings.Contains(out, "chat-1") || !strings.Contains(out, "Total: 1") {
		t.Fatalf("[incident-war-room-list] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWarRoomGet(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentWarRoom{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("incident", "war-room", "get", "chat-1", "--integration", "42")
	if err != nil {
		t.Fatalf("[incident-war-room-get] unexpected error: %v", err)
	}
	if mock.getInput == nil || mock.getInput.ChatID != "chat-1" || mock.getInput.IntegrationID != 42 {
		t.Fatalf("[incident-war-room-get] unexpected input: %#v", mock.getInput)
	}
	if !strings.Contains(out, "Chat ID:") || !strings.Contains(out, "chat-1") {
		t.Fatalf("[incident-war-room-get] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWarRoomAddMember(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentWarRoom{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("incident", "war-room", "add-member", "chat-1", "--integration", "42", "--member", "101,202")
	if err != nil {
		t.Fatalf("[incident-war-room-add-member] unexpected error: %v", err)
	}
	if mock.addMemberInput == nil || mock.addMemberInput.ChatID != "chat-1" || mock.addMemberInput.IntegrationID != 42 {
		t.Fatalf("[incident-war-room-add-member] unexpected input: %#v", mock.addMemberInput)
	}
	if got, want := fmt.Sprint(mock.addMemberInput.MemberIDs), "[101 202]"; got != want {
		t.Fatalf("[incident-war-room-add-member] expected members %q, got %q", want, got)
	}
	if !strings.Contains(out, "Added 2 member(s) to war room chat-1.") {
		t.Fatalf("[incident-war-room-add-member] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWarRoomDeleteWithForce(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockIncidentWarRoom{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("incident", "war-room", "delete", "inc-1", "--integration", "42", "--force")
	if err != nil {
		t.Fatalf("[incident-war-room-delete] unexpected error: %v", err)
	}
	if mock.deleteInput == nil || mock.deleteInput.IncidentID != "inc-1" || mock.deleteInput.IntegrationID != 42 {
		t.Fatalf("[incident-war-room-delete] unexpected input: %#v", mock.deleteInput)
	}
	if !strings.Contains(out, "Deleted war room for incident inc-1.") {
		t.Fatalf("[incident-war-room-delete] unexpected output:\n%s", out)
	}
}

type mockAuditSearchPagination struct {
	mockClient
	calls []*flashduty.SearchAuditLogsInput
}

func (m *mockAuditSearchPagination) SearchAuditLogs(_ context.Context, input *flashduty.SearchAuditLogsInput) (*flashduty.SearchAuditLogsOutput, error) {
	copied := *input
	m.calls = append(m.calls, &copied)

	if input.SearchAfterCtx == "" {
		return &flashduty.SearchAuditLogsOutput{
			AuditLogs: []flashduty.AuditLogRecord{
				{CreatedAt: 1712000000, MemberName: "Alice", Operation: "incident.create", Body: "page-1"},
			},
			Total:          2,
			SearchAfterCtx: "cursor-1",
		}, nil
	}

	if input.SearchAfterCtx == "cursor-1" {
		return &flashduty.SearchAuditLogsOutput{
			AuditLogs: []flashduty.AuditLogRecord{
				{CreatedAt: 1712003600, MemberName: "Bob", Operation: "incident.close", Body: "page-2"},
			},
			Total:          2,
			SearchAfterCtx: "",
		}, nil
	}

	return &flashduty.SearchAuditLogsOutput{
		AuditLogs:      nil,
		Total:          2,
		SearchAfterCtx: "",
	}, nil
}

func TestCommandAuditSearchPageUsesCursorPagination(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockAuditSearchPagination{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	out, err := execCommand("audit", "search", "--limit", "1", "--page", "2")
	if err != nil {
		t.Fatalf("[audit-search-page] unexpected error: %v", err)
	}

	if !strings.Contains(out, "Bob") || !strings.Contains(out, "page-2") {
		t.Fatalf("[audit-search-page] expected second page output, got:\n%s", out)
	}
	if strings.Contains(out, "Alice") || strings.Contains(out, "page-1") {
		t.Fatalf("[audit-search-page] output should not contain first page rows, got:\n%s", out)
	}
	if !strings.Contains(out, "Showing 1 results (page 2, total 2).") {
		t.Fatalf("[audit-search-page] expected paginated footer, got:\n%s", out)
	}
	if len(mock.calls) != 2 {
		t.Fatalf("[audit-search-page] expected 2 API calls, got %d", len(mock.calls))
	}
	if mock.calls[0].SearchAfterCtx != "" {
		t.Fatalf("[audit-search-page] expected first call cursor to be empty, got %q", mock.calls[0].SearchAfterCtx)
	}
	if mock.calls[1].SearchAfterCtx != "cursor-1" {
		t.Fatalf("[audit-search-page] expected second call cursor %q, got %q", "cursor-1", mock.calls[1].SearchAfterCtx)
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
