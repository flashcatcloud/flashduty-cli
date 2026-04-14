//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// getFirstChannelID runs channel list --json and extracts the first channel ID.
func getFirstChannelID(t *testing.T) string {
	t.Helper()
	r := runCLI(t, "channel", "list", "--json")
	requireSuccess(t, r)
	var channels []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(r.Stdout)), &channels); err != nil {
		t.Skipf("could not parse channel list JSON: %v", err)
	}
	if len(channels) == 0 {
		t.Skip("no channels available for escalation rule tests")
	}
	// channel_id may be a number or string depending on JSON marshalling.
	id := channels[0]["channel_id"]
	switch v := id.(type) {
	case float64:
		return fmt.Sprintf("%.0f", v)
	case string:
		return v
	default:
		t.Skipf("unexpected channel_id type: %T", id)
		return ""
	}
}

// ---------------------------------------------------------------------------
// Escalation Rule
// ---------------------------------------------------------------------------

// Test 150: escalation-rule list --channel
func TestEscalationRuleListByChannel(t *testing.T) {
	chID := getFirstChannelID(t)
	r := runCLI(t, "escalation-rule", "list", "--channel", chID)
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "ID", "NAME", "CHANNEL", "STATUS", "PRIORITY", "LAYERS")
}

// Test 152: escalation-rule list --channel-name (0 match)
func TestEscalationRuleListByChannelNameNoMatch(t *testing.T) {
	r := runCLI(t, "escalation-rule", "list", "--channel-name", "nonexistent_xyz_999")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "no channel found")
}

// Test 154: missing both flags
func TestEscalationRuleListMissingFlags(t *testing.T) {
	r := runCLI(t, "escalation-rule", "list")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "--channel")
}

// Test 156: escalation-rule list JSON
func TestEscalationRuleListJSON(t *testing.T) {
	chID := getFirstChannelID(t)
	r := runCLI(t, "escalation-rule", "list", "--channel", chID, "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}
