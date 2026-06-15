package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

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
	origFlagOutputFormat := flagOutputFormat
	origUpdateNotice := updateNotice
	origUpdateCheckWarning := updateCheckWarning
	origStdinReader := stdinReader

	// Reset to defaults so tests start clean.
	flagJSON = false
	flagNoTrunc = false
	flagAppKey = ""
	flagBaseURL = ""
	flagOutputFormat = ""
	updateNotice = nil
	updateCheckWarning = ""

	t.Cleanup(func() {
		newClientFn = origNewClientFn
		flagJSON = origFlagJSON
		flagNoTrunc = origFlagNoTrunc
		flagAppKey = origFlagAppKey
		flagBaseURL = origFlagBaseURL
		flagOutputFormat = origFlagOutputFormat
		updateNotice = origUpdateNotice
		updateCheckWarning = origUpdateCheckWarning
		stdinReader = origStdinReader
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
		case "stringSlice", "stringArray":
			// Slice-valued flags accumulate across Parse() calls; clear them
			// explicitly so a later test isn't observing the previous test's
			// repeated --flag entries. pflag's SliceValue / Append interfaces
			// don't expose a "reset to default" — Set("") would append an
			// empty entry, so we use Replace([]) to truly empty the slice.
			if sv, ok := flag.Value.(pflag.SliceValue); ok {
				_ = sv.Replace([]string{})
				flag.Changed = false
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Test 191: incident get returns empty results
// ---------------------------------------------------------------------------

func TestCommandIncidentGetEmptyResults(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{}, "total": 0}

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

func TestCommandIncidentCreateWithoutIncidentID(t *testing.T) {
	saveAndResetGlobals(t)
	// Empty data → no incident_id, so the command falls back to the generic
	// success message.
	newGFStub(t)

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
	newGFStub(t)

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

// These two guard the migration's behavior-preservation: the hand-written SDK
// forced assigned_to.type = "assign" on both create and reassign, and the
// go-flashduty port keeps that exact wire (see incident.go). Without the
// explicit Type the backend would relabel an already-assigned incident as
// "reassign".
func TestCommandIncidentCreateSetsAssignType(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"incident", "create",
		"--title", "Disk full", "--severity", "Warning",
		"--assign", "101,202",
	)
	if err != nil {
		t.Fatalf("[incident-create-assign] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/create" {
		t.Fatalf("[incident-create-assign] expected /incident/create, got %q", stub.lastPath)
	}
	assignedTo, ok := stub.lastBody["assigned_to"].(map[string]any)
	if !ok {
		t.Fatalf("[incident-create-assign] expected assigned_to object, got %#v", stub.lastBody["assigned_to"])
	}
	if assignedTo["type"] != "assign" {
		t.Fatalf("[incident-create-assign] expected assigned_to.type=assign (legacy wire), got %#v", assignedTo["type"])
	}
	if got, want := fmt.Sprint(assignedTo["person_ids"]), "[101 202]"; got != want {
		t.Fatalf("[incident-create-assign] expected person_ids %q, got %q", want, got)
	}
}

func TestCommandIncidentReassignSetsAssignType(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand("incident", "reassign", "inc-1", "--person", "303,404")
	if err != nil {
		t.Fatalf("[incident-reassign-assign] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/assign" {
		t.Fatalf("[incident-reassign-assign] expected /incident/assign, got %q", stub.lastPath)
	}
	if got, want := strings.Join(stub.bodyStrings("incident_ids"), ","), "inc-1"; got != want {
		t.Fatalf("[incident-reassign-assign] expected incident_ids %q, got %q", want, got)
	}
	assignedTo, ok := stub.lastBody["assigned_to"].(map[string]any)
	if !ok {
		t.Fatalf("[incident-reassign-assign] expected assigned_to object, got %#v", stub.lastBody["assigned_to"])
	}
	if assignedTo["type"] != "assign" {
		t.Fatalf("[incident-reassign-assign] expected assigned_to.type=assign (legacy wire), got %#v", assignedTo["type"])
	}
	if got, want := fmt.Sprint(assignedTo["person_ids"]), "[303 404]"; got != want {
		t.Fatalf("[incident-reassign-assign] expected person_ids %q, got %q", want, got)
	}
}

// ---------------------------------------------------------------------------
// Test 223: incident timeline empty
// ---------------------------------------------------------------------------

func TestCommandIncidentTimelineEmpty(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{}}

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
// Test 321: member list with PersonInfos
// ---------------------------------------------------------------------------

func TestCommandMemberListPersonInfos(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"items": []any{
			map[string]any{"member_id": 100, "member_name": "Alice", "email": "alice@example.com", "status": "enabled", "time_zone": "Asia/Shanghai"},
			map[string]any{"member_id": 200, "member_name": "Bob", "email": "bob@example.com", "status": "enabled", "time_zone": "UTC"},
		},
		"total": 2,
	}

	out, err := execCommand("member", "list")
	if err != nil {
		t.Fatalf("[#321] unexpected error: %v", err)
	}

	// The migrated `member list` renders MemberItem rows: ID, NAME, EMAIL,
	// STATUS, TIMEZONE. (The legacy PersonInfos-only view is gone — go-flashduty's
	// /member/list returns member rows directly.)
	for _, h := range []string{"ID", "NAME", "EMAIL", "STATUS", "TIMEZONE"} {
		if !strings.Contains(out, h) {
			t.Errorf("[#321] expected header %q in output, got:\n%s", h, out)
		}
	}

	for _, v := range []string{"Alice", "Bob", "alice@example.com", "bob@example.com"} {
		if !strings.Contains(out, v) {
			t.Errorf("[#321] expected %q in output, got:\n%s", v, out)
		}
	}

	if !strings.Contains(out, "Total: 2") {
		t.Errorf("[#321] expected 'Total: 2' in output, got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// Regression tests for new command batch review findings
// ---------------------------------------------------------------------------

func TestCommandIncidentFeedEmpty_JSON(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{}, "has_next_page": false}

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
	newGFStub(t)

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
	newGFStub(t)

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
	newGFStub(t)

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

func TestCommandIncidentUnack(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	// unack is served by the generated twin (positional ids → incident_ids).
	out, err := execCommand("incident", "unack", "inc-1", "inc-2")
	if err != nil {
		t.Fatalf("[incident-unack] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/unack" {
		t.Fatalf("[incident-unack] expected /incident/unack, got %q", stub.lastPath)
	}
	if got, want := strings.Join(stub.bodyStrings("incident_ids"), ","), "inc-1,inc-2"; got != want {
		t.Fatalf("[incident-unack] expected ids %q, got %q", want, got)
	}
	if !strings.Contains(out, "OK: POST /incident/unack") {
		t.Fatalf("[incident-unack] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWake(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	// wake is served by the generated twin (positional id → incident_ids).
	out, err := execCommand("incident", "wake", "inc-1")
	if err != nil {
		t.Fatalf("[incident-wake] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/wake" {
		t.Fatalf("[incident-wake] expected /incident/wake, got %q", stub.lastPath)
	}
	if got, want := strings.Join(stub.bodyStrings("incident_ids"), ","), "inc-1"; got != want {
		t.Fatalf("[incident-wake] expected ids %q, got %q", want, got)
	}
	if !strings.Contains(out, "OK: POST /incident/wake") {
		t.Fatalf("[incident-wake] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentComment(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	out, err := execCommand("incident", "comment", "inc-1", "inc-2", "--comment", "rollback started", "--mute-reply")
	if err != nil {
		t.Fatalf("[incident-comment] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/comment" {
		t.Fatalf("[incident-comment] expected /incident/comment, got %q", stub.lastPath)
	}
	if got, want := strings.Join(stub.bodyStrings("incident_ids"), ","), "inc-1,inc-2"; got != want {
		t.Fatalf("[incident-comment] expected ids %q, got %q", want, got)
	}
	if stub.lastBody["comment"] != "rollback started" || stub.lastBody["mute_reply"] != true {
		t.Fatalf("[incident-comment] unexpected input: %#v", stub.lastBody)
	}
	if !strings.Contains(out, "Commented on 2 incident(s).") {
		t.Fatalf("[incident-comment] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentCommentAllows1024UnicodeRunes(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	comment := strings.Repeat("界", 1024)
	_, err := execCommand("incident", "comment", "inc-1", "--comment", comment)
	if err != nil {
		t.Fatalf("[incident-comment-unicode] unexpected error: %v", err)
	}
	if stub.lastBody["comment"] != comment {
		t.Fatalf("[incident-comment-unicode] unexpected input: %#v", stub.lastBody)
	}
}

// TestCommandIncidentLifecycleRejectsMoreThan100IDs covers the curated
// commands that still enforce the 100-id batch cap client-side. unack and wake
// were dropped in favor of their generated twins, which carry no client-side
// cap (the backend enforces the limit), so they are intentionally absent here.
func TestCommandIncidentLifecycleRejectsMoreThan100IDs(t *testing.T) {
	commands := []struct {
		name string
		args []string
	}{
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
			newGFStub(t)

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
	stub := newGFStub(t)

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
	if stub.lastPath != "/incident/responder/add" {
		t.Fatalf("[incident-add-responder] expected /incident/responder/add, got %q", stub.lastPath)
	}
	if stub.lastBody["incident_id"] != "inc-1" {
		t.Fatalf("[incident-add-responder] expected incident inc-1, got %v", stub.lastBody["incident_id"])
	}
	if got, want := fmt.Sprint(stub.lastBody["person_ids"]), "[101 202]"; got != want {
		t.Fatalf("[incident-add-responder] expected people %q, got %q", want, got)
	}
	notify, ok := stub.lastBody["notify"].(map[string]any)
	if !ok || notify["follow_preference"] != true {
		t.Fatalf("[incident-add-responder] expected follow preference notify, got %#v", stub.lastBody["notify"])
	}
	channels, _ := notify["personal_channels"].([]any)
	if got, want := fmt.Sprint(channels), "[voice sms]"; got != want {
		t.Fatalf("[incident-add-responder] expected channels %q, got %q", want, got)
	}
	if notify["template_id"] != "6321aad26c12104586a88916" {
		t.Fatalf("[incident-add-responder] unexpected template id: %#v", notify)
	}
	if !strings.Contains(out, "Added 2 responder(s) to incident inc-1.") {
		t.Fatalf("[incident-add-responder] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentRemoveRequiresForceWhenNonInteractive(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	out, err := execCommand("incident", "remove", "inc-1")
	if err != nil {
		t.Fatalf("[incident-remove-abort] unexpected error: %v", err)
	}
	if stub.requests != 0 {
		t.Fatalf("[incident-remove-abort] remove should not be called, got %d request(s)", stub.requests)
	}
	if !strings.Contains(out, "Aborted.") {
		t.Fatalf("[incident-remove-abort] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentRemoveWithForce(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	out, err := execCommand("incident", "remove", "inc-1", "inc-2", "--force")
	if err != nil {
		t.Fatalf("[incident-remove-force] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/remove" {
		t.Fatalf("[incident-remove-force] expected /incident/remove, got %q", stub.lastPath)
	}
	if got, want := strings.Join(stub.bodyStrings("incident_ids"), ","), "inc-1,inc-2"; got != want {
		t.Fatalf("[incident-remove-force] expected ids %q, got %q", want, got)
	}
	if !strings.Contains(out, "Removed 2 incident(s).") {
		t.Fatalf("[incident-remove-force] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentDisableMerge(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	// disable-merge is served by the generated twin (positional ids → incident_ids).
	out, err := execCommand("incident", "disable-merge", "inc-1", "inc-2")
	if err != nil {
		t.Fatalf("[incident-disable-merge] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/disable-merge" {
		t.Fatalf("[incident-disable-merge] expected /incident/disable-merge, got %q", stub.lastPath)
	}
	if got, want := strings.Join(stub.bodyStrings("incident_ids"), ","), "inc-1,inc-2"; got != want {
		t.Fatalf("[incident-disable-merge] expected ids %q, got %q", want, got)
	}
	if !strings.Contains(out, "OK: POST /incident/disable-merge") {
		t.Fatalf("[incident-disable-merge] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWarRoomCreateWithObservers(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"chat_id": "chat-1", "chat_name": "INC outage", "share_link": "https://chat.example/1"}

	out, err := execCommand("incident", "war-room", "create", "inc-1", "--integration", "42", "--member", "101,202", "--add-observers")
	if err != nil {
		t.Fatalf("[incident-war-room-create] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/war-room/create" {
		t.Fatalf("[incident-war-room-create] expected /incident/war-room/create, got %q", stub.lastPath)
	}
	if stub.lastBody["incident_id"] != "inc-1" || stub.lastBody["integration_id"] != float64(42) || stub.lastBody["add_observers"] != true {
		t.Fatalf("[incident-war-room-create] unexpected input: %#v", stub.lastBody)
	}
	if got, want := fmt.Sprint(stub.lastBody["member_ids"]), "[101 202]"; got != want {
		t.Fatalf("[incident-war-room-create] expected member ids %q, got %q", want, got)
	}
	if !strings.Contains(out, "War room created: chat-1") {
		t.Fatalf("[incident-war-room-create] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWarRoomCreateAutoDiscoversIntegration(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	// First call lists war-room-enabled integrations; second call creates the
	// room. Serve a distinct payload per path.
	stub.dataForPath = func(path string, _ map[string]any) any {
		switch path {
		case "/datasource/im/war-room-enabled/list":
			return map[string]any{"items": []map[string]any{{"data_source_id": 42, "integration_id": 42}}}
		default:
			return map[string]any{"chat_id": "chat-1", "chat_name": "INC outage"}
		}
	}

	out, err := execCommand("incident", "war-room", "create", "inc-1", "--member", "101")
	if err != nil {
		t.Fatalf("[incident-war-room-create-autodiscover] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/war-room/create" {
		t.Fatalf("[incident-war-room-create-autodiscover] expected create as last call, got %q", stub.lastPath)
	}
	if stub.lastBody["integration_id"] != float64(42) {
		t.Fatalf("[incident-war-room-create-autodiscover] expected integration 42, got %#v", stub.lastBody)
	}
	if !strings.Contains(out, "War room created: chat-1") {
		t.Fatalf("[incident-war-room-create-autodiscover] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWarRoomCreateRequiresEnabledIntegration(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	// No war-room-enabled integrations: the list returns an empty items slice.
	stub.data = map[string]any{"items": []map[string]any{}}

	_, err := execCommand("incident", "war-room", "create", "inc-1")
	if err == nil || !strings.Contains(err.Error(), "no IM integration has war-room enabled") {
		t.Fatalf("[incident-war-room-create-no-enabled-integration] expected enabled integration error, got %v", err)
	}
	if stub.lastPath != "/datasource/im/war-room-enabled/list" {
		t.Fatalf("[incident-war-room-create-no-enabled-integration] did not expect create call; last path %q", stub.lastPath)
	}
}

func TestCommandIncidentWarRoomDefaultObservers(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"observers": []map[string]any{
			{"person_id": 101, "person_name": "Alice", "email": "alice@example.com"},
		},
	}

	out, err := execCommand("incident", "war-room", "default-observers", "inc-1")
	if err != nil {
		t.Fatalf("[incident-war-room-default-observers] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/war-room/default-observers" {
		t.Fatalf("[incident-war-room-default-observers] expected /incident/war-room/default-observers, got %q", stub.lastPath)
	}
	if stub.lastBody["incident_id"] != "inc-1" {
		t.Fatalf("[incident-war-room-default-observers] expected incident inc-1, got %#v", stub.lastBody)
	}
	if !strings.Contains(out, "Alice") || !strings.Contains(out, "alice@example.com") || !strings.Contains(out, "Total: 1") {
		t.Fatalf("[incident-war-room-default-observers] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWarRoomList(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"items": []map[string]any{
			{"integration_id": 42, "chat_id": "chat-1", "incident_id": "inc-1", "status": "enabled", "plugin_type": "feishu"},
		},
	}

	out, err := execCommand("incident", "war-room", "list", "inc-1", "--integration", "42")
	if err != nil {
		t.Fatalf("[incident-war-room-list] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/war-room/list" {
		t.Fatalf("[incident-war-room-list] expected /incident/war-room/list, got %q", stub.lastPath)
	}
	if stub.lastBody["incident_id"] != "inc-1" || stub.lastBody["integration_id"] != float64(42) {
		t.Fatalf("[incident-war-room-list] unexpected input: %#v", stub.lastBody)
	}
	if !strings.Contains(out, "chat-1") || !strings.Contains(out, "Total: 1") {
		t.Fatalf("[incident-war-room-list] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWarRoomGet(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"chat_id": "chat-1", "chat_name": "INC outage", "share_link": "https://chat.example/1"}

	out, err := execCommand("incident", "war-room", "get", "chat-1", "--integration", "42")
	if err != nil {
		t.Fatalf("[incident-war-room-get] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/war-room/detail" {
		t.Fatalf("[incident-war-room-get] expected /incident/war-room/detail, got %q", stub.lastPath)
	}
	if stub.lastBody["chat_id"] != "chat-1" || stub.lastBody["integration_id"] != float64(42) {
		t.Fatalf("[incident-war-room-get] unexpected input: %#v", stub.lastBody)
	}
	if !strings.Contains(out, "Chat ID:") || !strings.Contains(out, "chat-1") {
		t.Fatalf("[incident-war-room-get] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWarRoomAddMember(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	// WriteAddWarRoomMember decodes the envelope "data" into a *string.
	stub.data = "ok"

	out, err := execCommand("incident", "war-room", "add-member", "chat-1", "--integration", "42", "--member", "101,202")
	if err != nil {
		t.Fatalf("[incident-war-room-add-member] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/war-room/add-member" {
		t.Fatalf("[incident-war-room-add-member] expected /incident/war-room/add-member, got %q", stub.lastPath)
	}
	if stub.lastBody["chat_id"] != "chat-1" || stub.lastBody["integration_id"] != float64(42) {
		t.Fatalf("[incident-war-room-add-member] unexpected input: %#v", stub.lastBody)
	}
	if got, want := fmt.Sprint(stub.lastBody["member_ids"]), "[101 202]"; got != want {
		t.Fatalf("[incident-war-room-add-member] expected members %q, got %q", want, got)
	}
	if !strings.Contains(out, "Added 2 member(s) to war room chat-1.") {
		t.Fatalf("[incident-war-room-add-member] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentWarRoomDeleteWithForce(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	out, err := execCommand("incident", "war-room", "delete", "inc-1", "--integration", "42", "--force")
	if err != nil {
		t.Fatalf("[incident-war-room-delete] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/war-room/delete" {
		t.Fatalf("[incident-war-room-delete] expected /incident/war-room/delete, got %q", stub.lastPath)
	}
	if stub.lastBody["incident_id"] != "inc-1" || stub.lastBody["integration_id"] != float64(42) {
		t.Fatalf("[incident-war-room-delete] unexpected input: %#v", stub.lastBody)
	}
	if !strings.Contains(out, "Deleted war room for incident inc-1.") {
		t.Fatalf("[incident-war-room-delete] unexpected output:\n%s", out)
	}
}

func TestCommandAuditSearchPageUsesCursorPagination(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.dataFor = func(body map[string]any) any {
		cursor, _ := body["search_after_ctx"].(string)
		switch cursor {
		case "":
			return map[string]any{
				"docs": []map[string]any{
					{"created_at": 1712000000000, "member_name": "Alice", "operation": "incident.create", "body": "page-1"},
				},
				"total":            2,
				"search_after_ctx": "cursor-1",
			}
		case "cursor-1":
			return map[string]any{
				"docs": []map[string]any{
					{"created_at": 1712003600000, "member_name": "Bob", "operation": "incident.close", "body": "page-2"},
				},
				"total":            2,
				"search_after_ctx": "",
			}
		default:
			return map[string]any{"docs": []map[string]any{}, "total": 2, "search_after_ctx": ""}
		}
	}

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
	if len(stub.bodies) != 2 {
		t.Fatalf("[audit-search-page] expected 2 API calls, got %d", len(stub.bodies))
	}
	if c, _ := stub.bodies[0]["search_after_ctx"].(string); c != "" {
		t.Fatalf("[audit-search-page] expected first call cursor to be empty, got %q", c)
	}
	if c, _ := stub.bodies[1]["search_after_ctx"].(string); c != "cursor-1" {
		t.Fatalf("[audit-search-page] expected second call cursor %q, got %q", "cursor-1", c)
	}
}

// ---------------------------------------------------------------------------
// CLI-wide --data source forms (inline / stdin), proven on a generated command
// ---------------------------------------------------------------------------

// A generated command reads its body from STDIN when --data is exactly "-".
func TestCommandDataFromStdin(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = []any{} // /monit/datasource/list returns a top-level array

	stdinReader = strings.NewReader(`{"type":"prometheus"}`)

	_, err := execCommand("monit", "datasource-list", "--data", "-")
	if err != nil {
		t.Fatalf("[data-stdin] unexpected error: %v", err)
	}
	if stub.lastPath != "/monit/datasource/list" {
		t.Fatalf("[data-stdin] expected /monit/datasource/list, got %q", stub.lastPath)
	}
	if stub.lastBody["type"] != "prometheus" {
		t.Errorf("[data-stdin] expected type=prometheus from stdin, got %#v", stub.lastBody["type"])
	}
}

// Inline --data still works, and a typed flag overrides a matching --data key.
func TestCommandDataInlineFlagOverride(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = []any{} // /monit/datasource/list returns a top-level array

	_, err := execCommand(
		"monit", "datasource-list",
		"--data", `{"type":"loki"}`,
		"--type", "prometheus",
	)
	if err != nil {
		t.Fatalf("[data-inline] unexpected error: %v", err)
	}
	if stub.lastBody["type"] != "prometheus" {
		t.Errorf("[data-inline] expected typed --type to win over --data, got %#v", stub.lastBody["type"])
	}
}

// With --data absent, stdin is NEVER read (guards against the empty-pipe hang).
// A non-blocking sentinel reader fails the test if it is ever consumed.
func TestCommandNoDataDoesNotReadStdin(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = []any{} // /monit/datasource/list returns a top-level array

	stdinReader = readerFunc(func([]byte) (int, error) {
		t.Fatal("[no-data] stdin was read despite --data being absent")
		return 0, io.EOF
	})

	_, err := execCommand("monit", "datasource-list", "--type", "mysql")
	if err != nil {
		t.Fatalf("[no-data] unexpected error: %v", err)
	}
	if stub.lastBody["type"] != "mysql" {
		t.Errorf("[no-data] expected type=mysql, got %#v", stub.lastBody["type"])
	}
}

// readerFunc adapts a function to io.Reader so a test can assert Read is never
// called.
type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

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
