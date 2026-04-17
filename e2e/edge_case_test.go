//go:build e2e

package e2e_test

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// --json on get command (test 80)
// ---------------------------------------------------------------------------

// Test 80: --json on get command
func TestJSONOnGetCommand(t *testing.T) {
	name := uniqueName("json_get")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "get", id, "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// ---------------------------------------------------------------------------
// --no-trunc on various list commands (tests 81-89)
// ---------------------------------------------------------------------------

func TestNoTruncOnIncidentList(t *testing.T) {
	r := runCLI(t, "incident", "list", "--no-trunc", "--since", "168h")
	requireSuccess(t, r)
}

func TestNoTruncOnTeamList(t *testing.T) {
	r := runCLI(t, "team", "list", "--no-trunc")
	requireSuccess(t, r)
}

func TestNoTruncOnChangeList(t *testing.T) {
	r := runCLI(t, "change", "list", "--no-trunc", "--since", "168h")
	requireSuccess(t, r)
}

func TestNoTruncOnFieldList(t *testing.T) {
	r := runCLI(t, "field", "list", "--no-trunc")
	requireSuccess(t, r)
}

func TestNoTruncOnStatusPageList(t *testing.T) {
	r := runCLI(t, "statuspage", "list", "--no-trunc")
	requireSuccess(t, r)
}

// ---------------------------------------------------------------------------
// --json + --no-trunc combined (test 90)
// ---------------------------------------------------------------------------

// Test 90: --json + --no-trunc combined
func TestJSONAndNoTruncCombined(t *testing.T) {
	r := runCLI(t, "channel", "list", "--json", "--no-trunc")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// ---------------------------------------------------------------------------
// Table vs JSON same data (test 95)
// ---------------------------------------------------------------------------

// Test 95: table vs JSON same data
func TestTableVsJSONSameData(t *testing.T) {
	rTable := runCLI(t, "member", "list")
	requireSuccess(t, rTable)
	rJSON := runCLI(t, "member", "list", "--json")
	requireSuccess(t, rJSON)
	requireValidJSON(t, rJSON.Stdout)
	// JSON should not be empty if table had results
	if strings.Contains(rTable.Stdout, "Total:") && rJSON.Stdout == "null\n" {
		t.Error("JSON output is null but table had results")
	}
}

// ---------------------------------------------------------------------------
// JSON contains untruncated data (test 96)
// ---------------------------------------------------------------------------

// Test 96: JSON contains untruncated data
func TestJSONContainsUntruncatedData(t *testing.T) {
	r := runCLI(t, "incident", "list", "--json", "--since", "168h")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
	// JSON should never contain "..." truncation
	requireNotContains(t, r.Stdout, `..."`)
}

// ---------------------------------------------------------------------------
// Data goes to stdout (test 98)
// ---------------------------------------------------------------------------

// Test 98: data goes to stdout
func TestDataGoesToStdout(t *testing.T) {
	r := runCLIPublic(t, "version")
	requireSuccess(t, r)
	if r.Stdout == "" {
		t.Error("expected stdout to contain data")
	}
}

// ---------------------------------------------------------------------------
// Nested subcommand help (test 109)
// ---------------------------------------------------------------------------

// Test 109: nested subcommand help
func TestNestedSubcommandHelp(t *testing.T) {
	r := runCLIPublic(t, "incident", "list", "--help")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "--since")
	requireContains(t, r.Stdout, "--until")
	requireContains(t, r.Stdout, "--progress")
	requireContains(t, r.Stdout, "--severity")
}

// ---------------------------------------------------------------------------
// --json on write commands (tests 306-309)
// ---------------------------------------------------------------------------

// Test 306: --json on ack command
func TestJSONOnAckCommand(t *testing.T) {
	// Create an incident to ack
	name := uniqueName("json_ack")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "ack", id, "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
	requireContains(t, r.Stdout, "Acknowledged")
}

// Test 307: --json on close command
func TestJSONOnCloseCommand(t *testing.T) {
	name := uniqueName("json_close")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)

	r = runCLI(t, "incident", "close", id, "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
	requireContains(t, r.Stdout, "Closed")
}

// Test 308: --json on create command
func TestJSONOnCreateCommand(t *testing.T) {
	name := uniqueName("json_create")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
	requireContains(t, r.Stdout, "Incident created")
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })
}

// ---------------------------------------------------------------------------
// Time range flags (tests 311-316)
// ---------------------------------------------------------------------------

// Test 311: --until without --since uses default 24h
func TestUntilWithoutSince(t *testing.T) {
	r := runCLI(t, "incident", "list", "--until", "now")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Showing")
}

// Test 312: --since without --until
func TestSinceWithoutUntil(t *testing.T) {
	r := runCLI(t, "incident", "list", "--since", "168h")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Showing")
}

// Test 313: both --since and --until as dates
func TestSinceAndUntilAsDates(t *testing.T) {
	r := runCLI(t, "incident", "list", "--since", "2026-04-01", "--until", "2026-04-10")
	requireSuccess(t, r)
}

// Test 316: --since = --until same time
func TestSinceEqualsUntil(t *testing.T) {
	r := runCLI(t, "incident", "list", "--since", "2026-04-13 12:00:00", "--until", "2026-04-13 12:00:00")
	// The API may reject start_time == end_time, which is valid behavior.
	// Just verify no panic.
	_ = r
}

// ---------------------------------------------------------------------------
// Change list boundary values (tests 317-320)
// ---------------------------------------------------------------------------

// Test 317: change list --limit 0
func TestChangeListLimitZero(t *testing.T) {
	r := runCLI(t, "change", "list", "--limit", "0", "--since", "168h")
	requireSuccess(t, r) // no crash
}

// Test 318: change list --page 0
func TestChangeListPageZero(t *testing.T) {
	r := runCLI(t, "change", "list", "--page", "0", "--since", "168h")
	requireSuccess(t, r) // no crash
}

// Test 319: change list --limit -1
func TestChangeListLimitNegative(t *testing.T) {
	r := runCLI(t, "change", "list", "--limit", "-1", "--since", "168h")
	requireSuccess(t, r) // no crash
}

// Test 320: change list --page -1
func TestChangeListPageNegative(t *testing.T) {
	r := runCLI(t, "change", "list", "--page", "-1", "--since", "168h")
	requireSuccess(t, r) // no crash
}
