//go:build e2e

package e2e_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func decodeObjectList(t *testing.T, output string) []map[string]any {
	t.Helper()
	var items []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &items); err != nil {
		t.Fatalf("failed to decode JSON output: %v\n%s", err, output)
	}
	return items
}

func mustStringField(t *testing.T, item map[string]any, key string) string {
	t.Helper()
	raw, ok := item[key]
	if !ok {
		t.Fatalf("missing field %q in item: %#v", key, item)
	}
	value, ok := raw.(string)
	if !ok {
		t.Fatalf("field %q has unexpected type %T", key, raw)
	}
	if value == "" {
		t.Fatalf("field %q is empty in item: %#v", key, item)
	}
	return value
}

func requireAllMatchSubstring(t *testing.T, items []map[string]any, key, filter string) {
	t.Helper()
	if len(items) == 0 {
		t.Fatalf("expected at least one result for filter %q", filter)
	}
	lowerFilter := strings.ToLower(filter)
	for _, item := range items {
		value := mustStringField(t, item, key)
		if !strings.Contains(strings.ToLower(value), lowerFilter) {
			t.Fatalf("field %q=%q does not contain filter %q (case-insensitive)", key, value, filter)
		}
	}
}

// ---------------------------------------------------------------------------
// Channel filters
// ---------------------------------------------------------------------------

// Test 127: channel list --name filter
func TestChannelListNameFilter(t *testing.T) {
	r := runCLI(t, "channel", "list", "--json")
	requireSuccess(t, r)
	channels := decodeObjectList(t, r.Stdout)
	if len(channels) == 0 {
		t.Skip("no channels available")
	}

	filter := mustStringField(t, channels[0], "channel_name")
	r = runCLI(t, "channel", "list", "--name", filter, "--json")
	requireSuccess(t, r)
	requireAllMatchSubstring(t, decodeObjectList(t, r.Stdout), "channel_name", filter)
}

// Test 130: channel list --no-trunc
func TestChannelListNoTrunc(t *testing.T) {
	defaultOutput := runCLI(t, "channel", "list")
	requireSuccess(t, defaultOutput)
	noTruncOutput := runCLI(t, "channel", "list", "--no-trunc")
	requireSuccess(t, noTruncOutput)
	if defaultOutput.Stdout != noTruncOutput.Stdout {
		t.Fatal("channel list output changed under --no-trunc even though the command has no truncation columns")
	}
}

// ---------------------------------------------------------------------------
// Member filters
// ---------------------------------------------------------------------------

// Test 132: member list --name
func TestMemberListNameFilter(t *testing.T) {
	r := runCLI(t, "member", "list", "--json")
	requireSuccess(t, r)
	members := decodeObjectList(t, r.Stdout)
	if len(members) == 0 {
		t.Skip("no members available")
	}

	filter := mustStringField(t, members[0], "member_name")
	r = runCLI(t, "member", "list", "--name", filter, "--json")
	requireSuccess(t, r)
	requireAllMatchSubstring(t, decodeObjectList(t, r.Stdout), "member_name", filter)
}

// Test 133: member list --email
func TestMemberListEmailFilter(t *testing.T) {
	r := runCLI(t, "member", "list", "--json")
	requireSuccess(t, r)
	members := decodeObjectList(t, r.Stdout)
	if len(members) == 0 {
		t.Skip("no members available")
	}

	filter := mustStringField(t, members[0], "email")
	r = runCLI(t, "member", "list", "--email", filter, "--json")
	requireSuccess(t, r)
	requireAllMatchSubstring(t, decodeObjectList(t, r.Stdout), "email", filter)
}

// Test 134: member list --name + --email combined
func TestMemberListNameAndEmailFilter(t *testing.T) {
	r := runCLI(t, "member", "list", "--json")
	requireSuccess(t, r)
	members := decodeObjectList(t, r.Stdout)
	if len(members) == 0 {
		t.Skip("no members available")
	}

	nameFilter := mustStringField(t, members[0], "member_name")
	emailFilter := mustStringField(t, members[0], "email")
	r = runCLI(t, "member", "list", "--name", nameFilter, "--email", emailFilter, "--json")
	requireSuccess(t, r)
	results := decodeObjectList(t, r.Stdout)
	if len(results) == 0 {
		t.Fatalf("expected at least one result for name=%q email=%q", nameFilter, emailFilter)
	}
	for _, item := range results {
		if !strings.Contains(mustStringField(t, item, "member_name"), nameFilter) {
			t.Fatalf("member_name did not match combined filter %q", nameFilter)
		}
		if !strings.Contains(mustStringField(t, item, "email"), emailFilter) {
			t.Fatalf("email did not match combined filter %q", emailFilter)
		}
	}
}

// Test 135: member list --page 1
func TestMemberListPage1(t *testing.T) {
	r := runCLI(t, "member", "list", "--page", "1", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// Test 136: member list --page 2
func TestMemberListPage2(t *testing.T) {
	r := runCLI(t, "member", "list", "--page", "2", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// Test 137: pagination different pages
func TestMemberListPaginationDiffers(t *testing.T) {
	r1 := runCLI(t, "member", "list", "--page", "1", "--json")
	requireSuccess(t, r1)
	page1 := decodeObjectList(t, r1.Stdout)

	r2 := runCLI(t, "member", "list", "--page", "2", "--json")
	requireSuccess(t, r2)
	page2 := decodeObjectList(t, r2.Stdout)

	if len(page1) > 0 && len(page2) > 0 && strings.TrimSpace(r1.Stdout) == strings.TrimSpace(r2.Stdout) {
		t.Fatal("page 1 and page 2 returned identical non-empty result sets")
	}
}

// ---------------------------------------------------------------------------
// Team filters
// ---------------------------------------------------------------------------

// Test 141: team list --name
func TestTeamListNameFilter(t *testing.T) {
	r := runCLI(t, "team", "list", "--json")
	requireSuccess(t, r)
	teams := decodeObjectList(t, r.Stdout)
	if len(teams) == 0 {
		t.Skip("no teams available")
	}

	filter := mustStringField(t, teams[0], "team_name")
	r = runCLI(t, "team", "list", "--name", filter, "--json")
	requireSuccess(t, r)
	requireAllMatchSubstring(t, decodeObjectList(t, r.Stdout), "team_name", filter)
}

// Test 142: team list --page
func TestTeamListPage(t *testing.T) {
	r := runCLI(t, "team", "list", "--page", "1", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// Test 144: team members column truncation
func TestTeamListMembersColumnTruncation(t *testing.T) {
	r := runCLI(t, "team", "list")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "MEMBERS")
}

// Test 145: team list --no-trunc
func TestTeamListNoTruncMembers(t *testing.T) {
	defaultOutput := runCLI(t, "team", "list")
	requireSuccess(t, defaultOutput)
	noTruncOutput := runCLI(t, "team", "list", "--no-trunc")
	requireSuccess(t, noTruncOutput)

	requireTableHeaders(t, noTruncOutput.Stdout, "MEMBERS")
	if strings.Contains(defaultOutput.Stdout, "...") {
		requireNotContains(t, noTruncOutput.Stdout, "...")
	}
}

// ---------------------------------------------------------------------------
// Field filters
// ---------------------------------------------------------------------------

// Test 147: field list --name
func TestFieldListNameFilter(t *testing.T) {
	r := runCLI(t, "field", "list", "--json")
	requireSuccess(t, r)
	fields := decodeObjectList(t, r.Stdout)
	if len(fields) == 0 {
		t.Skip("no fields available")
	}

	filter := mustStringField(t, fields[0], "field_name")
	r = runCLI(t, "field", "list", "--name", filter, "--json")
	requireSuccess(t, r)
	requireAllMatchSubstring(t, decodeObjectList(t, r.Stdout), "field_name", filter)
}

// Test 149: field empty OPTIONS column
func TestFieldListEmptyOptions(t *testing.T) {
	r := runCLI(t, "field", "list")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "OPTIONS")
}
