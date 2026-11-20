package cli

import (
	"fmt"
	"strings"
	"testing"
)

// TestCommandChangeList exercises the go-flashduty-backed `change list` command:
// the request hits /change/list, --channel-ids is forwarded as channel_ids, and the
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

	out, err := execCommand("change", "list", "--channel-ids", "100,200", "--limit", "10", "--page", "1")
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
