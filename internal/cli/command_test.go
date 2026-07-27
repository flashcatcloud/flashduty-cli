package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/flashcatcloud/go-flashduty"
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

// newIncidentCommentEchoStub wires a gfStub so /incident/feed echoes back
// whatever text was most recently posted to /incident/comment, as a single
// i_comm entry — simulating a backend that faithfully stored the comment, so
// the command's own read-after-write verification passes. Tests that need a
// different feed response (a mismatch, or no comment entry at all) build
// their own gfStub with a custom dataForPath instead of starting from this
// one, since overriding the field after construction would replace this
// whole echo behavior anyway.
func newIncidentCommentEchoStub(t *testing.T) *gfStub {
	t.Helper()
	stub := newGFStub(t)
	var posted string
	stub.dataForPath = func(path string, body map[string]any) any {
		switch path {
		case "/incident/comment":
			posted, _ = body["comment"].(string)
			return map[string]any{}
		case "/incident/feed":
			return map[string]any{
				"items": []any{
					map[string]any{
						"type":       "i_comm",
						"created_at": 1,
						"detail":     map[string]any{"comment": posted},
					},
				},
			}
		default:
			return map[string]any{}
		}
	}
	return stub
}

// writeCommentFile writes content to a fresh file under t.TempDir() and
// returns its path.
func writeCommentFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "comment.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeCommentFile: %v", err)
	}
	return path
}

func TestCommandIncidentComment(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newIncidentCommentEchoStub(t)
	commentFile := writeCommentFile(t, "rollback started")

	out, err := execCommand("incident", "comment", "inc-1", "inc-2", "--comment-file", commentFile, "--mute-reply")
	if err != nil {
		t.Fatalf("[incident-comment] unexpected error: %v", err)
	}
	if len(stub.bodies) == 0 || stub.bodies[0]["comment"] != "rollback started" || stub.bodies[0]["mute_reply"] != true {
		t.Fatalf("[incident-comment] unexpected first request: %#v", stub.bodies)
	}
	if got, want := strings.Join(stringsField(stub.bodies[0], "incident_ids"), ","), "inc-1,inc-2"; got != want {
		t.Fatalf("[incident-comment] expected ids %q, got %q", want, got)
	}
	if !strings.Contains(out, "Commented on 2 incident(s).") {
		t.Fatalf("[incident-comment] unexpected output:\n%s", out)
	}
}

func TestCommandIncidentCommentAllows1024UnicodeRunes(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newIncidentCommentEchoStub(t)

	comment := strings.Repeat("界", 1024)
	commentFile := writeCommentFile(t, comment)

	_, err := execCommand("incident", "comment", "inc-1", "--comment-file", commentFile)
	if err != nil {
		t.Fatalf("[incident-comment-unicode] unexpected error: %v", err)
	}
	if stub.bodies[0]["comment"] != comment {
		t.Fatalf("[incident-comment-unicode] unexpected input: %#v", stub.bodies[0])
	}
}

func TestCommandIncidentCommentRejectsOver1024Runes(t *testing.T) {
	saveAndResetGlobals(t)
	// The length check in RunE rejects the comment before any HTTP call is
	// made, so a plain stub (never a network echo) is all that's needed —
	// mirrors TestCommandIncidentLifecycleRejectsMoreThan100IDs.
	newGFStub(t)

	commentFile := writeCommentFile(t, strings.Repeat("界", 1025))

	_, err := execCommand("incident", "comment", "inc-1", "--comment-file", commentFile)
	if err == nil {
		t.Fatal("[incident-comment-too-long] expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "1024 characters") {
		t.Fatalf("[incident-comment-too-long] unexpected error: %v", err)
	}
}

// TestCommandIncidentCommentPreservesShellMetacharactersByteForByte is the
// regression test for the incident that motivated dropping the inline
// --comment flag: an LLM-authored comment containing backticks, $(...),
// unbalanced quotes, and a line that looks like a heredoc terminator must
// reach the API exactly as written (modulo the leading/trailing-whitespace
// trim applied to match server storage — see
// TestCommandIncidentCommentSurvivesServerSideTrim), whether it arrives via
// --comment-file or via stdin. Neither path ever hands the text to a shell.
func TestCommandIncidentCommentPreservesShellMetacharactersByteForByte(t *testing.T) {
	malicious := "Root cause: restart via `kubectl rollout restart deploy/api`.\n" +
		"Then ran $(rm -rf /tmp/scratch) to clean up staging state.\n" +
		"Quotes: \"double\" and 'single' and it's mine.\n" +
		"EOF\n" +
		"The line above looks like a heredoc terminator but is just comment text.\n"
	want := strings.TrimSpace(malicious)

	t.Run("comment-file", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newIncidentCommentEchoStub(t)
		commentFile := writeCommentFile(t, malicious)

		out, err := execCommand("incident", "comment", "inc-1", "--comment-file", commentFile)
		if err != nil {
			t.Fatalf("[comment-file] unexpected error: %v", err)
		}
		if stub.bodies[0]["comment"] != want {
			t.Fatalf("[comment-file] comment reached the API mangled:\nwant: %q\n got: %q", want, stub.bodies[0]["comment"])
		}
		if !strings.Contains(out, "Commented on 1 incident(s).") {
			t.Fatalf("[comment-file] unexpected output:\n%s", out)
		}
	})

	t.Run("stdin", func(t *testing.T) {
		saveAndResetGlobals(t)
		stub := newIncidentCommentEchoStub(t)
		stdinReader = strings.NewReader(malicious)

		out, err := execCommand("incident", "comment", "inc-1", "--comment-file", "-")
		if err != nil {
			t.Fatalf("[stdin] unexpected error: %v", err)
		}
		if stub.bodies[0]["comment"] != want {
			t.Fatalf("[stdin] comment reached the API mangled:\nwant: %q\n got: %q", want, stub.bodies[0]["comment"])
		}
		if !strings.Contains(out, "Commented on 1 incident(s).") {
			t.Fatalf("[stdin] unexpected output:\n%s", out)
		}
	})
}

// newIncidentCommentTrimmingEchoStub wires a gfStub so /incident/feed echoes
// back whatever text was most recently posted to /incident/comment, but with
// leading/trailing whitespace stripped first — reproducing fc-event's actual
// storage behavior (cmd/server/controller/incident/action.go's Comment
// handler calls strings.TrimSpace on the comment after validating length but
// before writing the feed entry). Unlike newIncidentCommentEchoStub, which
// echoes back byte-for-byte and is therefore self-consistent with whatever
// the client happened to send, this stub is backend-consistent: it can
// actually catch a client that fails to apply the same normalization before
// comparing.
func newIncidentCommentTrimmingEchoStub(t *testing.T) *gfStub {
	t.Helper()
	stub := newGFStub(t)
	var posted string
	stub.dataForPath = func(path string, body map[string]any) any {
		switch path {
		case "/incident/comment":
			posted, _ = body["comment"].(string)
			return map[string]any{}
		case "/incident/feed":
			return map[string]any{
				"items": []any{
					map[string]any{
						"type":       "i_comm",
						"created_at": 1,
						"detail":     map[string]any{"comment": strings.TrimSpace(posted)},
					},
				},
			}
		default:
			return map[string]any{}
		}
	}
	return stub
}

// TestCommandIncidentCommentSurvivesServerSideTrim is the regression test for
// the critical bug found in review: fc-event trims incident comments
// server-side before storing them (see newIncidentCommentTrimmingEchoStub's
// doc comment for the exact call site), but verification used to compare the
// server's trimmed, stored text against the raw, untrimmed bytes read from
// --comment-file. This CLI's own skill card (incident.md) tells agents to
// author the comment via a bash heredoc, which always leaves a trailing
// newline — so every comment written by that documented workflow would fail
// verification and be reported as "not found," and a naive retry on that
// false signal would append one more (trimmed) duplicate comment to the
// incident every time, forever. This test reproduces that exact input shape.
func TestCommandIncidentCommentSurvivesServerSideTrim(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newIncidentCommentTrimmingEchoStub(t)

	// Mirrors a bash heredoc's output: content followed by a trailing newline,
	// the pattern skills/flashduty/reference/incident.md's hot flow uses.
	heredocStyle := "Root cause identified: DB failover.\nFix deploying.\n"
	commentFile := writeCommentFile(t, heredocStyle)

	out, err := execCommand("incident", "comment", "inc-1", "--comment-file", commentFile)
	if err != nil {
		t.Fatalf("[server-trim] unexpected error: %v", err)
	}
	want := strings.TrimSpace(heredocStyle)
	if stub.bodies[0]["comment"] != want {
		t.Fatalf("[server-trim] expected the wire payload pre-trimmed to match what the server stores:\nwant: %q\n got: %q", want, stub.bodies[0]["comment"])
	}
	if !strings.Contains(out, "Commented on 1 incident(s).") {
		t.Fatalf("[server-trim] unexpected output:\n%s", out)
	}
}

// TestCommandIncidentCommentSucceedsDespiteNewerConcurrentComment is the
// regression test for the false-corruption bug: verification used to trust
// whichever i_comm entry had the highest CreatedAt, so a concurrent, unrelated
// comment from another human or a webhook-reply bot (--mute-reply off) landing
// after ours made the command report a perfectly good write as "corrupted."
// This stub puts our own comment on the feed at an older CreatedAt and an
// unrelated comment at a newer one; the command must still succeed because
// verification now searches for an exact text match instead of trusting
// recency.
func TestCommandIncidentCommentSucceedsDespiteNewerConcurrentComment(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	var posted string
	stub.dataForPath = func(path string, body map[string]any) any {
		switch path {
		case "/incident/comment":
			posted, _ = body["comment"].(string)
			return map[string]any{}
		case "/incident/feed":
			return map[string]any{
				"items": []any{
					map[string]any{"type": "i_comm", "created_at": 1, "detail": map[string]any{"comment": posted}},
					map[string]any{"type": "i_comm", "created_at": 2, "detail": map[string]any{"comment": "a reply bot's unrelated concurrent comment"}},
				},
			}
		default:
			return map[string]any{}
		}
	}
	commentFile := writeCommentFile(t, "the real comment")

	out, err := execCommand("incident", "comment", "inc-1", "--comment-file", commentFile)
	if err != nil {
		t.Fatalf("[newer-concurrent-comment] unexpected error: %v", err)
	}
	if !strings.Contains(out, "Commented on 1 incident(s).") {
		t.Fatalf("[newer-concurrent-comment] unexpected output:\n%s", out)
	}
}

// TestCommandIncidentCommentFailsWhenTextNotFoundWithinBudget guards the
// read-after-write verification: when no i_comm entry on the timeline (within
// the page budget) has text matching what was written, the command must exit
// non-zero instead of printing the "Commented on ..." success line. The feed
// here has an entry, just not one with matching text, so this also covers
// what used to be the "mismatch" case — under the new exact-match search that
// entry is simply not a match, not a confirmed corruption.
func TestCommandIncidentCommentFailsWhenTextNotFoundWithinBudget(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.dataForPath = func(path string, body map[string]any) any {
		switch path {
		case "/incident/comment":
			return map[string]any{}
		case "/incident/feed":
			return map[string]any{
				"items": []any{
					map[string]any{
						"type":       "i_comm",
						"created_at": 1,
						"detail":     map[string]any{"comment": "an unrelated comment"},
					},
				},
			}
		default:
			return map[string]any{}
		}
	}
	commentFile := writeCommentFile(t, "the real comment")

	out, err := execCommand("incident", "comment", "inc-1", "--comment-file", commentFile)
	if err == nil {
		t.Fatal("[not-found] expected a non-zero exit, got nil error")
	}
	if !strings.Contains(err.Error(), "no timeline entry matches the written text") {
		t.Fatalf("[not-found] unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "corrupted") {
		t.Fatalf("[not-found] must not claim corruption it has not established: %v", err)
	}
	if !strings.Contains(err.Error(), "Do not write the comment again") {
		t.Fatalf("[not-found] must warn against rewriting, at any granularity: %v", err)
	}
	if strings.Contains(out, "Commented on") {
		t.Fatalf("[not-found] must not report success:\n%s", out)
	}
}

// TestCommandIncidentCommentFailsWhenNoCommentEntryFound guards the other
// genuinely-absent-within-budget case: the feed has no i_comm entry at all
// after the write (e.g. it was dropped), which must also be a hard failure.
func TestCommandIncidentCommentFailsWhenNoCommentEntryFound(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{}}

	commentFile := writeCommentFile(t, "the real comment")

	out, err := execCommand("incident", "comment", "inc-1", "--comment-file", commentFile)
	if err == nil {
		t.Fatal("[readback-missing] expected a non-zero exit, got nil error")
	}
	if !strings.Contains(err.Error(), "no timeline entry matches the written text") {
		t.Fatalf("[readback-missing] unexpected error: %v", err)
	}
	if strings.Contains(out, "Commented on") {
		t.Fatalf("[readback-missing] must not report success:\n%s", out)
	}
}

// TestCommandIncidentCommentChecksEntireBatchNotJustFirstFailure guards
// against stopping at the first failing incident: the error text asserts
// "the write already succeeded for the whole batch, don't retry it" — a claim
// only earned by having actually examined every incident. This stub fails
// verification for inc-1 (feed empty) and inc-3 (feed has an unrelated
// comment), with inc-2 verifying successfully in between. inc-3's feed must
// still be queried, and the error must name both failing incidents, proving
// the walk doesn't stop after inc-1.
func TestCommandIncidentCommentChecksEntireBatchNotJustFirstFailure(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	feedCallsByIncident := map[string]int{}
	stub.dataForPath = func(path string, body map[string]any) any {
		switch path {
		case "/incident/comment":
			return map[string]any{}
		case "/incident/feed":
			incID, _ := body["incident_id"].(string)
			feedCallsByIncident[incID]++
			switch incID {
			case "inc-2":
				return map[string]any{"items": []any{
					map[string]any{"type": "i_comm", "created_at": 1, "detail": map[string]any{"comment": "the real comment"}},
				}}
			case "inc-3":
				return map[string]any{"items": []any{
					map[string]any{"type": "i_comm", "created_at": 1, "detail": map[string]any{"comment": "an unrelated comment"}},
				}}
			default:
				return map[string]any{"items": []any{}}
			}
		default:
			return map[string]any{}
		}
	}
	commentFile := writeCommentFile(t, "the real comment")

	out, err := execCommand("incident", "comment", "inc-1", "inc-2", "inc-3", "--comment-file", commentFile)
	if err == nil {
		t.Fatal("[whole-batch] expected a non-zero exit, got nil error")
	}
	if feedCallsByIncident["inc-3"] == 0 {
		t.Fatalf("[whole-batch] inc-3's feed was never queried — verification stopped at the first failure: %v", err)
	}
	if !strings.Contains(err.Error(), "inc-1") || !strings.Contains(err.Error(), "inc-3") {
		t.Fatalf("[whole-batch] expected both failing incidents named in the error, got: %v", err)
	}
	if strings.Contains(err.Error(), "inc-2") {
		t.Fatalf("[whole-batch] inc-2 verified fine and must not be named as a problem: %v", err)
	}
	if strings.Contains(out, "Commented on") {
		t.Fatalf("[whole-batch] must not report success:\n%s", out)
	}
}

// TestCommandIncidentCommentTransportErrorCarriesAntiRetryWarning guards that
// a transient failure while re-fetching the feed (rate limit, timeout — a
// large batch's own read-back traffic, up to maxIncidentFeedVerifyPages
// requests per incident, is itself a plausible source) gets the identical
// anti-retry framing as a "not found" result. The write already succeeded in
// both cases; a bare transport error here would trigger the same naive
// full-batch retry reflex the "not found" message exists to defuse.
func TestCommandIncidentCommentTransportErrorCarriesAntiRetryWarning(t *testing.T) {
	saveAndResetGlobals(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		_ = json.Unmarshal(raw, &decoded)
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/incident/feed" {
			if incID, _ := decoded["incident_id"].(string); incID == "inc-1" {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"request_id": "req-err",
					"error":      map[string]any{"code": "internal_error", "message": "temporarily unavailable"},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"request_id": "req-feed",
				"error":      map[string]any{"code": "OK", "message": ""},
				"data": map[string]any{"items": []any{
					map[string]any{"type": "i_comm", "created_at": 1, "detail": map[string]any{"comment": "the real comment"}},
				}},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req-ok",
			"error":      map[string]any{"code": "OK", "message": ""},
			"data":       map[string]any{},
		})
	}))
	t.Cleanup(srv.Close)

	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test-key", flashduty.WithBaseURL(srv.URL))
	}

	commentFile := writeCommentFile(t, "the real comment")
	out, err := execCommand("incident", "comment", "inc-1", "inc-2", "--comment-file", commentFile)
	if err == nil {
		t.Fatal("[transport-error] expected a non-zero exit, got nil error")
	}
	if !strings.Contains(err.Error(), "inc-1") {
		t.Fatalf("[transport-error] expected inc-1 named in the error: %v", err)
	}
	if !strings.Contains(err.Error(), "Do not write the comment again") {
		t.Fatalf("[transport-error] must carry the same anti-retry framing as the not-found case: %v", err)
	}
	if strings.Contains(err.Error(), "corrupted") {
		t.Fatalf("[transport-error] must not claim corruption: %v", err)
	}
	if strings.Contains(out, "Commented on") {
		t.Fatalf("[transport-error] must not report success:\n%s", out)
	}
}

// TestCommandIncidentCommentVerifiesAcrossFeedPages guards the read-after-write
// verification against /incident/feed's undocumented default sort order: it
// must not assume the just-written comment is on page 1. This stub puts an
// older, unrelated comment on page 1 (with has_next_page true) and the
// freshly written comment only on page 2 (has_next_page false), simulating a
// feed sorted in whichever direction would bury the new entry past the first
// page. Verification must keep walking pages and find the exact-text match on
// page 2 rather than giving up after page 1.
func TestCommandIncidentCommentVerifiesAcrossFeedPages(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	var posted string
	stub.dataForPath = func(path string, body map[string]any) any {
		switch path {
		case "/incident/comment":
			posted, _ = body["comment"].(string)
			return map[string]any{}
		case "/incident/feed":
			page, _ := body["p"].(float64)
			if page <= 1 {
				return map[string]any{
					"has_next_page": true,
					"items": []any{
						map[string]any{"type": "i_comm", "created_at": 1, "detail": map[string]any{"comment": "an older, unrelated comment"}},
					},
				}
			}
			return map[string]any{
				"has_next_page": false,
				"items": []any{
					map[string]any{"type": "i_comm", "created_at": 2, "detail": map[string]any{"comment": posted}},
				},
			}
		default:
			return map[string]any{}
		}
	}
	commentFile := writeCommentFile(t, "spread across feed pages")

	out, err := execCommand("incident", "comment", "inc-1", "--comment-file", commentFile)
	if err != nil {
		t.Fatalf("[feed-pagination] unexpected error: %v", err)
	}
	if !strings.Contains(out, "Commented on 1 incident(s).") {
		t.Fatalf("[feed-pagination] unexpected output:\n%s", out)
	}
}

// TestCommandIncidentLifecycleRejectsMoreThan100IDs covers the curated
// commands that still enforce the 100-id batch cap client-side. unack and wake
// were dropped in favor of their generated twins, which carry no client-side
// cap (the backend enforces the limit), so they are intentionally absent here.
func TestCommandIncidentLifecycleRejectsMoreThan100IDs(t *testing.T) {
	commentFile := writeCommentFile(t, "too many")

	commands := []struct {
		name string
		args []string
	}{
		{name: "comment", args: []string{"incident", "comment", "--comment-file", commentFile}},
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
