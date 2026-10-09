package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommandAlertListActiveRecoveredReachWire is the regression guard for the
// nullable-pointer bug: is_active and ever_muted are *bool in the SDK, so the
// false value must reach the wire. Before the fix they were value+omitempty and
// --recovered (is_active=false) was silently dropped, turning the filter into a
// no-op that returned active alerts too.
func TestCommandAlertListActiveRecoveredReachWire(t *testing.T) {
	cases := []struct {
		name     string
		flag     string
		field    string
		wantBool bool
	}{
		{"active sends is_active=true", "--active", "is_active", true},
		{"recovered sends is_active=false", "--recovered", "is_active", false},
		{"muted sends ever_muted=true", "--muted", "ever_muted", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			if _, err := execCommand("alert", "list", tc.flag); err != nil {
				t.Fatalf("execCommand: %v", err)
			}
			got, ok := stub.lastBody[tc.field]
			if !ok {
				t.Fatalf("%s missing from wire body %#v", tc.field, stub.lastBody)
			}
			gotBool, isBool := got.(bool)
			if !isBool {
				t.Fatalf("%s = %#v (%T), want a JSON bool", tc.field, got, got)
			}
			if gotBool != tc.wantBool {
				t.Errorf("%s = %v, want %v", tc.field, gotBool, tc.wantBool)
			}
		})
	}
}

// TestCommandAlertListNoStatusFilterOmitsIsActive: with neither --active nor
// --recovered, is_active is a nil *bool and omitempty keeps it off the wire, so
// the server applies no status filter.
func TestCommandAlertListNoStatusFilterOmitsIsActive(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("alert", "list"); err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if _, ok := stub.lastBody["is_active"]; ok {
		t.Errorf("is_active should be omitted with no status filter, got %#v", stub.lastBody["is_active"])
	}
	if _, ok := stub.lastBody["ever_muted"]; ok {
		t.Errorf("ever_muted should be omitted without --muted, got %#v", stub.lastBody["ever_muted"])
	}
}

// TestCommandAlertListIntegrationFlagReachesWire guards --integration on
// `alert list`: a comma-separated value must parse via the shared
// parseIntSlice helper (same as --channel) and forward as integration_ids on
// /alert/list.
func TestCommandAlertListIntegrationFlagReachesWire(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("alert", "list", "--integration", "10001,10002"); err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if stub.lastPath != "/alert/list" {
		t.Fatalf("expected /alert/list, got %q", stub.lastPath)
	}
	if got, want := fmt.Sprint(stub.lastBody["integration_ids"]), "[10001 10002]"; got != want {
		t.Fatalf("expected integration_ids %q, got %q", want, got)
	}
}

// TestCommandAlertListIntegrationFlagInvalidValue guards the error path: a
// non-numeric --integration value must fail with a named error instead of
// silently dropping the filter.
func TestCommandAlertListIntegrationFlagInvalidValue(t *testing.T) {
	saveAndResetGlobals(t)
	newGFStub(t)

	_, err := execCommand("alert", "list", "--integration", "abc")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid --integration") {
		t.Fatalf("expected an \"invalid --integration\" error, got: %v", err)
	}
}

// TestCommandAlertEventListIntegrationFlagReachesWire guards --integration on
// `alert-event list`: it must forward as integration_ids on /alert-event/list,
// distinct from --integration-type which forwards as integration_types.
func TestCommandAlertEventListIntegrationFlagReachesWire(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("alert-event", "list",
		"--integration", "10001,10002", "--integration-type", "AliCloud"); err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if stub.lastPath != "/alert-event/list" {
		t.Fatalf("expected /alert-event/list, got %q", stub.lastPath)
	}
	if got, want := fmt.Sprint(stub.lastBody["integration_ids"]), "[10001 10002]"; got != want {
		t.Fatalf("expected integration_ids %q, got %q", want, got)
	}
	if got, want := fmt.Sprint(stub.lastBody["integration_types"]), "[AliCloud]"; got != want {
		t.Fatalf("expected integration_types %q, got %q", want, got)
	}
}

// TestCommandAlertMergeCommentFileReachesWireByteForByte guards the same
// shell-interpolation fix applied to incident comment (see
// TestCommandIncidentCommentPreservesShellMetacharactersByteForByte): alert
// merge's comment must come from --comment-file, never an inline shell
// argument, so backticks, $(...), and quotes inside an LLM-authored comment
// reach the API exactly as written.
func TestCommandAlertMergeCommentFileReachesWireByteForByte(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	malicious := "Root cause: restart via `kubectl rollout restart deploy/api`.\n" +
		"Then ran $(rm -rf /tmp/scratch) to clean up staging state.\n" +
		"Quotes: \"double\" and 'single' and it's mine.\n"
	commentFile := writeCommentFile(t, malicious)

	out, err := execCommand("alert", "merge", "alert-1", "alert-2",
		"--incident-id", "inc-1", "--comment-file", commentFile)
	if err != nil {
		t.Fatalf("[alert-merge-comment-file] unexpected error: %v", err)
	}
	if stub.lastPath != "/alert/merge" {
		t.Fatalf("[alert-merge-comment-file] expected /alert/merge, got %q", stub.lastPath)
	}
	if stub.lastBody["comment"] != malicious {
		t.Fatalf("[alert-merge-comment-file] comment reached the API mangled:\nwant: %q\n got: %q", malicious, stub.lastBody["comment"])
	}
	if got, want := strings.Join(stringsField(stub.lastBody, "alert_ids"), ","), "alert-1,alert-2"; got != want {
		t.Fatalf("[alert-merge-comment-file] expected alert_ids %q, got %q", want, got)
	}
	if stub.lastBody["incident_id"] != "inc-1" {
		t.Fatalf("[alert-merge-comment-file] expected incident_id %q, got %#v", "inc-1", stub.lastBody["incident_id"])
	}
	if !strings.Contains(out, "OK: POST /alert/merge") {
		t.Fatalf("[alert-merge-comment-file] unexpected output:\n%s", out)
	}
}

// TestCommandAlertMergeWithoutCommentFileOmitsComment guards that the merge
// comment stays optional now that it is sourced from a file: not passing
// --comment-file must not send an empty "comment" field.
func TestCommandAlertMergeWithoutCommentFileOmitsComment(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("alert", "merge", "alert-1", "--incident-id", "inc-1"); err != nil {
		t.Fatalf("[alert-merge-no-comment] unexpected error: %v", err)
	}
	if _, ok := stub.lastBody["comment"]; ok {
		t.Fatalf("[alert-merge-no-comment] comment should be omitted, got %#v", stub.lastBody["comment"])
	}
}

// TestCommandAlertMergeEmptyCommentFileGivesCleanError guards routing alert
// merge's optional --comment-file through the same resolveCommentFile helper
// incident comment uses: an explicit but empty --comment-file value must fail
// with resolveCommentFile's clean "must not be empty" message, not the raw
// os.ReadFile("") error.
func TestCommandAlertMergeEmptyCommentFileGivesCleanError(t *testing.T) {
	saveAndResetGlobals(t)
	newGFStub(t)

	_, err := execCommand("alert", "merge", "alert-1", "--incident-id", "inc-1", "--comment-file", "")
	if err == nil {
		t.Fatal("[alert-merge-empty-comment-file] expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "--comment-file must not be empty") {
		t.Fatalf("[alert-merge-empty-comment-file] expected the clean resolveCommentFile message, got: %v", err)
	}
}

// TestCommandAlertMergeDataAndCommentFileBothStdinErrors guards against a
// silent-data-loss regression: --data and --comment-file can each read "-"
// for stdin, but stdin is a single stream that only one of them can actually
// drain. genAssembleBody resolves --data before --comment-file, so --data
// claims stdin first; --comment-file must then get a clear, named error
// instead of silently reading EOF and sending an empty comment (which the
// SDK's omitempty tag would then drop from the wire entirely, making the
// command falsely report success).
func TestCommandAlertMergeDataAndCommentFileBothStdinErrors(t *testing.T) {
	saveAndResetGlobals(t)
	newGFStub(t)

	stdinReader = strings.NewReader("{}")

	_, err := execCommand("alert", "merge", "alert-1", "--incident-id", "inc-1",
		"--data", "-", "--comment-file", "-")
	if err == nil {
		t.Fatal("[alert-merge-double-stdin] expected an error, got nil")
	}
	const want = `only one flag can read from stdin: --data and --comment-file were both set to "-"`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("[alert-merge-double-stdin] expected the double-stdin-read error naming both flags, got: %v", err)
	}
}

func TestCommandAlertListFilterFlagsReachWire(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("alert", "list", "--alert-ids", "a1,a2", "--alert-keys", "k1, k2", "--asc", "--by-updated-at"); err != nil {
		t.Fatalf("[alert-list-filters] unexpected error: %v", err)
	}
	if stub.lastPath != "/alert/list" {
		t.Fatalf("[alert-list-filters] expected /alert/list, got %q", stub.lastPath)
	}
	for _, f := range []string{"asc", "by_updated_at"} {
		if got := stub.lastBody[f]; got != true {
			t.Errorf("[alert-list-filters] %s: want true, got %#v", f, got)
		}
	}
	for field, want := range map[string]string{"alert_ids": "a1,a2", "alert_keys": "k1,k2"} {
		got, _ := stub.lastBody[field].([]any)
		var parts []string
		for _, v := range got {
			parts = append(parts, fmt.Sprint(v))
		}
		if strings.Join(parts, ",") != want {
			t.Errorf("[alert-list-filters] %s: want %q, got %#v", field, want, stub.lastBody[field])
		}
	}
}

func TestCommandAlertListFilterFlagsDefaultOmitted(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("alert", "list"); err != nil {
		t.Fatalf("[alert-list-filters-default] unexpected error: %v", err)
	}
	for _, f := range []string{"alert_ids", "alert_keys", "asc", "by_updated_at"} {
		if _, ok := stub.lastBody[f]; ok {
			t.Errorf("[alert-list-filters-default] %s should be omitted by default, got %#v", f, stub.lastBody[f])
		}
	}
}

func TestCommandAlertEventListAsc(t *testing.T) {
	for _, tc := range []struct {
		name  string
		extra []string
		want  any
	}{
		{name: "default omits asc", want: nil},
		{name: "flag sends asc", extra: []string{"--asc"}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			if _, err := execCommand(append([]string{"alert-event", "list"}, tc.extra...)...); err != nil {
				t.Fatalf("[alert-event-list-asc] unexpected error: %v", err)
			}
			if stub.lastPath != "/alert-event/list" {
				t.Fatalf("[alert-event-list-asc] expected /alert-event/list, got %q", stub.lastPath)
			}
			if got := stub.lastBody["asc"]; got != tc.want {
				t.Fatalf("[alert-event-list-asc] asc: want %#v, got %#v", tc.want, got)
			}
		})
	}
}

func TestCommandAlertMergeCommentFileReachesWire(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	path := filepath.Join(t.TempDir(), "comment.txt")
	if err := os.WriteFile(path, []byte("merge `reason` $(x)"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := execCommand("alert", "merge", "a1", "--incident-id", "i1", "--comment-file", path); err != nil {
		t.Fatalf("[alert-merge-comment] unexpected error: %v", err)
	}
	if got := stub.lastBody["comment"]; got != "merge `reason` $(x)" {
		t.Fatalf("[alert-merge-comment] comment: got %#v", got)
	}
}
