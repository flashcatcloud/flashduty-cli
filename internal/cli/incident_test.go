package cli

import (
	"fmt"
	"strings"
	"testing"
)

// TestCommandIncidentSimilarLimitReachesWire guards the *int64 Limit field on
// ListPastIncidentsRequest: --limit must reach the wire body (it is wrapped
// with flashduty.Int64). The command's --limit default is 5, never 0, so the
// value is always sent.
func TestCommandIncidentSimilarLimitReachesWire(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("incident", "similar", "inc-1", "--limit", "7"); err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if stub.lastPath != "/incident/past/list" {
		t.Fatalf("path = %q, want /incident/past/list", stub.lastPath)
	}
	// JSON numbers decode to float64 through the stub.
	if got, _ := stub.lastBody["limit"].(float64); got != 7 {
		t.Errorf("limit = %#v, want 7", stub.lastBody["limit"])
	}
	if stub.lastBody["incident_id"] != "inc-1" {
		t.Errorf("incident_id = %#v, want inc-1", stub.lastBody["incident_id"])
	}
}

// TestCommandIncidentListChannelFlag verifies --channel is a string flag
// (comma-separated IDs), matching the sibling list verbs (alert list
// --channel, alert-event list --channel, change list --channel) — not the
// singular int64 --channel-id this command used to require — and that
// --channel-id is still registered but hidden+deprecated.
func TestCommandIncidentListChannelFlag(t *testing.T) {
	cmd := newIncidentListCmd()
	flags := cmd.Flags()

	f := flags.Lookup("channel")
	if f == nil {
		t.Fatal("flag --channel not registered")
	}
	if got := f.Value.Type(); got != "string" {
		t.Errorf("--channel flag type = %q, want %q", got, "string")
	}
	if got := f.DefValue; got != "" {
		t.Errorf("--channel default = %q, want %q", got, "")
	}

	idFlag := flags.Lookup("channel-id")
	if idFlag == nil {
		t.Fatal("flag --channel-id must still be registered (deprecated alias)")
	}
	if !idFlag.Hidden {
		t.Error("--channel-id must be hidden now that --channel is canonical")
	}
	if idFlag.Deprecated == "" {
		t.Error("--channel-id must carry a deprecation message")
	}
}

// TestCommandIncidentListChannelForwardsMultipleIDs verifies a
// comma-separated --channel value reaches /incident/list as channel_ids —
// the same wire shape alert list / change list already use.
func TestCommandIncidentListChannelForwardsMultipleIDs(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("incident", "list", "--channel", "100,200"); err != nil {
		t.Fatalf("execCommand --channel: %v", err)
	}
	if stub.lastPath != "/incident/list" {
		t.Fatalf("path = %q, want /incident/list", stub.lastPath)
	}
	if got, want := fmt.Sprint(stub.lastBody["channel_ids"]), "[100 200]"; got != want {
		t.Fatalf("channel_ids = %q, want %q", got, want)
	}
}

// TestCommandIncidentListChannelIDFlagDeprecatedAlias verifies the
// deprecated --channel-id alias still works and still forwards to
// /incident/list as channel_ids, so scripts written before --channel existed
// keep working. --channel is canonical now; see
// TestCommandIncidentListChannelFlag above.
func TestCommandIncidentListChannelIDFlagDeprecatedAlias(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("incident", "list", "--channel-id", "123"); err != nil {
		t.Fatalf("execCommand --channel-id: %v", err)
	}
	if stub.lastPath != "/incident/list" {
		t.Fatalf("path = %q, want /incident/list", stub.lastPath)
	}
	if got, want := fmt.Sprint(stub.lastBody["channel_ids"]), "[123]"; got != want {
		t.Fatalf("channel_ids = %q, want %q", got, want)
	}
}

func TestCommandIncidentListHelpSurfacesInsightIncidentExport(t *testing.T) {
	saveAndResetGlobals(t)

	out, err := execCommand("incident", "list", "--help")
	if err != nil {
		t.Fatalf("incident list --help: %v", err)
	}
	if !strings.Contains(out, "fduty insight incident-export") {
		t.Fatalf("help output missing incident export discovery hint:\n%s", out)
	}
}
