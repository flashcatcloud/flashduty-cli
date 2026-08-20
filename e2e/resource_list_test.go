//go:build e2e

package e2e_test

import (
	"strings"
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
	r := runCLI(t, "member", "list", "--query", "nonexistent_xyz_999")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "No results.")
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

// Test 248: status-page list
//
// `status-page list` is a generated command with no displayColumns entry, so the
// table columns are the reflective heuristic: the first 8 scalar fields of
// StatusPageItem, headed by their upper-cased JSON tag. Components/sections are
// nested arrays and are skipped by that heuristic, so there is no COMPONENTS
// column, and PAGE_ID/NAME/URL_NAME fall past the 8-column cut.
func TestStatusPageList(t *testing.T) {
	r := runCLI(t, "status-page", "list")
	requireSuccess(t, r)
	if strings.HasPrefix(strings.TrimSpace(r.Stdout), "No results.") {
		t.Skip("no status pages available")
	}
	requireTableHeaders(t, r.Stdout, "CONTACT_INFO", "CUSTOM_DOMAIN", "DATE_VIEW", "DISPLAY_UPTIME_MODE")
}

// Test 252: status-page list JSON
func TestStatusPageListJSON(t *testing.T) {
	r := runCLI(t, "status-page", "list", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}
