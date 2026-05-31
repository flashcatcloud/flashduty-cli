package cli

import (
	"fmt"
	"strings"
	"testing"
)

// TestChangeListChannelFlag verifies that --channel is a string flag (comma-separated IDs),
// not a singular int64 flag. Mirrors the alert list --channel pattern.
func TestChangeListChannelFlag(t *testing.T) {
	cmd := newChangeListCmd()
	flags := cmd.Flags()

	f := flags.Lookup("channel")
	if f == nil {
		t.Fatal("flag --channel not registered")
	}

	// Must be a string flag (Value.Type() == "string"), not int64.
	if got := f.Value.Type(); got != "string" {
		t.Errorf("--channel flag type = %q, want %q", got, "string")
	}

	// Default must be empty string (not "0").
	if got := f.DefValue; got != "" {
		t.Errorf("--channel default = %q, want %q", got, "")
	}
}

// TestChangeListChannelParsing verifies that a comma-separated --channel value
// is correctly parsed to []int64 via parseIntSlice — the same helper used by
// alert list. Full comma-split semantics are covered by TestParseIntSlice in
// helpers_test.go; this test only confirms the wiring is correct.
func TestChangeListChannelParsing(t *testing.T) {
	// parseIntSlice is the shared helper; spot-check the three-value case.
	got, err := parseIntSlice("100,200,300")
	if err != nil {
		t.Fatalf("parseIntSlice(\"100,200,300\"): unexpected error: %v", err)
	}
	want := []int64{100, 200, 300}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %d, want %d", i, got[i], want[i])
		}
	}
}

// TestCommandChangeList exercises the go-flashduty-backed `change list` command:
// the request hits /change/list, --channel is forwarded as channel_ids, and the
// table renders change_status/channel_name straight from the response.
func TestCommandChangeList(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"items": []map[string]any{
			{"change_id": "chg-1", "title": "Deploy api v2", "change_status": "Resolved", "channel_name": "prod", "start_time": 1779432894},
		},
		"total": 1,
	}

	out, err := execCommand("change", "list", "--channel", "100,200", "--limit", "10", "--page", "1")
	if err != nil {
		t.Fatalf("[change-list] unexpected error: %v", err)
	}
	if stub.lastPath != "/change/list" {
		t.Fatalf("[change-list] expected /change/list, got %q", stub.lastPath)
	}
	if got, want := fmt.Sprint(stub.lastBody["channel_ids"]), "[100 200]"; got != want {
		t.Fatalf("[change-list] expected channel_ids %q, got %q", want, got)
	}
	if stub.lastBody["limit"] != float64(10) || stub.lastBody["p"] != float64(1) {
		t.Fatalf("[change-list] unexpected pagination: %#v", stub.lastBody)
	}
	if !strings.Contains(out, "chg-1") || !strings.Contains(out, "Resolved") || !strings.Contains(out, "prod") {
		t.Fatalf("[change-list] unexpected output:\n%s", out)
	}
	if !strings.Contains(out, "total 1") {
		t.Fatalf("[change-list] expected footer with total, got:\n%s", out)
	}
}
