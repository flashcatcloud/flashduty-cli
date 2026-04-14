//go:build e2e

package e2e_test

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Channel
// ---------------------------------------------------------------------------

// Test 126: channel list
func TestChannelList(t *testing.T) {
	r := runCLI(t, "channel", "list")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "ID", "NAME", "TEAM", "CREATOR")
}

// Test 128: channel list JSON
func TestChannelListJSON(t *testing.T) {
	r := runCLI(t, "channel", "list", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// Test 129: channel list empty results (no crash)
func TestChannelListNoResults(t *testing.T) {
	r := runCLI(t, "channel", "list", "--name", "nonexistent_xyz_999")
	requireSuccess(t, r)
}

// ---------------------------------------------------------------------------
// Member
// ---------------------------------------------------------------------------

// Test 131: member list
func TestMemberList(t *testing.T) {
	r := runCLI(t, "member", "list")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "ID", "NAME", "EMAIL", "STATUS", "TIMEZONE")
}

// Test 138: member list JSON
func TestMemberListJSON(t *testing.T) {
	r := runCLI(t, "member", "list", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// Test 139: member list empty results
func TestMemberListNoResults(t *testing.T) {
	r := runCLI(t, "member", "list", "--name", "nonexistent_xyz_999", "--email", "nonexistent_xyz")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "No members found.")
}

// ---------------------------------------------------------------------------
// Team
// ---------------------------------------------------------------------------

// Test 140: team list
func TestTeamList(t *testing.T) {
	r := runCLI(t, "team", "list")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "ID", "NAME", "MEMBERS")
}

// Test 143: team list JSON
func TestTeamListJSON(t *testing.T) {
	r := runCLI(t, "team", "list", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// ---------------------------------------------------------------------------
// Field
// ---------------------------------------------------------------------------

// Test 146: field list
func TestFieldList(t *testing.T) {
	r := runCLI(t, "field", "list")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "ID", "NAME", "DISPLAY_NAME", "TYPE", "OPTIONS")
}

// Test 148: field list JSON
func TestFieldListJSON(t *testing.T) {
	r := runCLI(t, "field", "list", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// ---------------------------------------------------------------------------
// Change
// ---------------------------------------------------------------------------

// Test 238: change list
func TestChangeList(t *testing.T) {
	r := runCLI(t, "change", "list", "--since", "168h")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "ID", "TITLE", "TYPE", "STATUS", "CHANNEL", "TIME")
}

// Test 244: change list JSON
func TestChangeListJSON(t *testing.T) {
	r := runCLI(t, "change", "list", "--since", "168h", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// ---------------------------------------------------------------------------
// StatusPage
// ---------------------------------------------------------------------------

// Test 248: statuspage list
func TestStatusPageList(t *testing.T) {
	r := runCLI(t, "statuspage", "list")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "ID", "NAME", "SLUG", "STATUS", "COMPONENTS")
}

// Test 252: statuspage list JSON
func TestStatusPageListJSON(t *testing.T) {
	r := runCLI(t, "statuspage", "list", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}
