package cli

import (
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
