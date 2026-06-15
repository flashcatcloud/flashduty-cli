package cli

import (
	"fmt"
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

// TestCommandIncidentListChannelIDFlag verifies that `incident list` accepts
// the canonical --channel-id flag (consistent with the sibling channel
// commands, e.g. `channel info --channel-id`) and forwards it to /incident/list
// as channel_ids. An agent that transferred --channel-id from those commands
// previously hit "unknown flag: --channel-id" and wasted a turn.
func TestCommandIncidentListChannelIDFlag(t *testing.T) {
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
