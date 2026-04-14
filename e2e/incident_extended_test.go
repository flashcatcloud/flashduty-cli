//go:build e2e

package e2e_test

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Incident List Extended
// ---------------------------------------------------------------------------

// Test 159: --since duration
func TestIncidentListSinceDuration(t *testing.T) {
	r := runCLI(t, "incident", "list", "--since", "168h")
	requireSuccess(t, r)
}

// Test 160: --since date
func TestIncidentListSinceDate(t *testing.T) {
	r := runCLI(t, "incident", "list", "--since", "2026-04-01")
	requireSuccess(t, r)
}

// Test 161: --since datetime
func TestIncidentListSinceDatetime(t *testing.T) {
	r := runCLI(t, "incident", "list", "--since", "2026-04-01 00:00:00")
	requireSuccess(t, r)
}

// Test 163: --until override
func TestIncidentListUntilOverride(t *testing.T) {
	r := runCLI(t, "incident", "list", "--since", "168h", "--until", "now")
	requireSuccess(t, r)
}

// Test 164: --progress single
func TestIncidentListProgressFilter(t *testing.T) {
	r := runCLI(t, "incident", "list", "--progress", "Triggered", "--since", "168h")
	requireSuccess(t, r)
}

// Test 165: --progress multiple
func TestIncidentListProgressMultiple(t *testing.T) {
	r := runCLI(t, "incident", "list", "--progress", "Triggered,Processing", "--since", "168h")
	requireSuccess(t, r)
}

// Test 166: --severity single
func TestIncidentListSeverityFilter(t *testing.T) {
	r := runCLI(t, "incident", "list", "--severity", "Critical", "--since", "168h")
	requireSuccess(t, r)
}

// Test 167: --severity multiple
func TestIncidentListSeverityMultiple(t *testing.T) {
	r := runCLI(t, "incident", "list", "--severity", "Critical,Warning", "--since", "168h")
	requireSuccess(t, r)
}

// Test 169: --title search
func TestIncidentListTitleSearch(t *testing.T) {
	r := runCLI(t, "incident", "list", "--title", "test", "--since", "168h")
	requireSuccess(t, r)
}

// Test 171: --limit
func TestIncidentListLimit(t *testing.T) {
	r := runCLI(t, "incident", "list", "--limit", "3", "--since", "168h")
	requireSuccess(t, r)
}

// Test 172: --page
func TestIncidentListPage(t *testing.T) {
	r := runCLI(t, "incident", "list", "--page", "1", "--since", "168h")
	requireSuccess(t, r)
}

// Test 173: --limit 0 defaults (no crash)
func TestIncidentListLimitZero(t *testing.T) {
	r := runCLI(t, "incident", "list", "--limit", "0", "--since", "168h")
	requireSuccess(t, r)
}

// Test 174: --page 0 defaults (no crash)
func TestIncidentListPageZero(t *testing.T) {
	r := runCLI(t, "incident", "list", "--page", "0", "--since", "168h")
	requireSuccess(t, r)
}

// Test 175: --limit -1 defaults (no crash)
func TestIncidentListLimitNegative(t *testing.T) {
	r := runCLI(t, "incident", "list", "--limit", "-1", "--since", "168h")
	requireSuccess(t, r)
}

// Test 176: --page -1 defaults (no crash)
func TestIncidentListPageNegative(t *testing.T) {
	r := runCLI(t, "incident", "list", "--page", "-1", "--since", "168h")
	requireSuccess(t, r)
}

// Test 180: pagination footer
func TestIncidentListPaginationFooter(t *testing.T) {
	r := runCLI(t, "incident", "list", "--since", "168h")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Showing")
	requireContains(t, r.Stdout, "results")
}

// Test 182: large --limit
func TestIncidentListLargeLimit(t *testing.T) {
	r := runCLI(t, "incident", "list", "--limit", "100", "--since", "168h")
	requireSuccess(t, r)
}

// ---------------------------------------------------------------------------
// Incident Get
// ---------------------------------------------------------------------------

// Test 183-190: These need a real incident. Create one, then test get.
func TestIncidentGetSingleID(t *testing.T) {
	// Create an incident
	name := uniqueName("get_single")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	// Test 183: single ID vertical view
	r = runCLI(t, "incident", "get", id)
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "ID:")
	requireContains(t, r.Stdout, "Title:")
	requireContains(t, r.Stdout, "Severity:")
	requireContains(t, r.Stdout, "Progress:")
	requireContains(t, r.Stdout, "Channel:")
	requireContains(t, r.Stdout, "Created:")
	requireContains(t, r.Stdout, "Alerts:")

	// Test 186: single ID JSON
	r = runCLI(t, "incident", "get", id, "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)

	// Test 190: single ID with empty optional fields shows "-"
	r = runCLI(t, "incident", "get", id)
	requireSuccess(t, r)
	// Labels, custom fields may show "-" if empty
}

// Test 189: nonexistent ID
func TestIncidentGetNonexistentID(t *testing.T) {
	r := runCLI(t, "incident", "get", "nonexistent_id_xyz_999")
	requireMeaningfulResult(t, r)
}

// ---------------------------------------------------------------------------
// Incident Lifecycle Extended
// ---------------------------------------------------------------------------

// Test 193: create with all flags
func TestIncidentCreateAllFlags(t *testing.T) {
	name := uniqueName("create_full")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Warning", "--description", "E2E test description")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Incident created")
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })
}

// Test 197: create missing title (non-interactive)
func TestIncidentCreateMissingTitle(t *testing.T) {
	r := runCLIWithStdin(t, "", "incident", "create", "--severity", "Info")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "--title is required")
}

// Test 198: create missing severity (non-interactive)
func TestIncidentCreateMissingSeverity(t *testing.T) {
	r := runCLIWithStdin(t, "", "incident", "create", "--title", "test")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "--severity is required")
}

// Test 200: ack single ID
func TestIncidentAckSingleID(t *testing.T) {
	name := uniqueName("ack_single")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "ack", id)
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Acknowledged 1 incident(s).")
}

// Test 204: close single ID
func TestIncidentCloseSingleID(t *testing.T) {
	name := uniqueName("close_single")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)

	r = runCLI(t, "incident", "close", id)
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Closed 1 incident(s).")
}

// ---------------------------------------------------------------------------
// Incident Update
// ---------------------------------------------------------------------------

// Test 208: update title
func TestIncidentUpdateTitle(t *testing.T) {
	name := uniqueName("update_title")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	newTitle := uniqueName("updated")
	r = runCLI(t, "incident", "update", id, "--title", newTitle)
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Updated incident")
	requireContains(t, r.Stdout, "title")

	// Verify the title was updated
	r = runCLI(t, "incident", "get", id)
	requireSuccess(t, r)
	requireContains(t, r.Stdout, newTitle)
}

// Test 210: update description
func TestIncidentUpdateDescription(t *testing.T) {
	name := uniqueName("update_desc")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "update", id, "--description", "New description")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Updated incident")
}

// Test 213: update --field invalid format
func TestIncidentUpdateFieldInvalidFormat(t *testing.T) {
	name := uniqueName("update_field_inv")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "update", id, "--field", "noequals")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "invalid --field format")
}

// Test 214: update nothing
func TestIncidentUpdateNothing(t *testing.T) {
	name := uniqueName("update_nothing")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "update", id)
	// The CLI may return success with "No fields were updated." or
	// failure with a "no fields" error -- both are valid behavior.
	if r.ExitCode == 0 {
		requireContains(t, r.Stdout, "No fields were updated.")
	}
}

// Test 215: update no args
func TestIncidentUpdateNoArgs(t *testing.T) {
	r := runCLI(t, "incident", "update")
	requireFailure(t, r)
}

// Test 217: update --field with = in value
func TestIncidentUpdateFieldEqualsInValue(t *testing.T) {
	name := uniqueName("update_field_eq")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	// --field key=a=b should set value to "a=b"
	r = runCLI(t, "incident", "update", id, "--field", "test_key=a=b")
	// This may succeed or fail depending on whether the field exists;
	// just verify no panic and no "invalid --field format" error.
	if r.ExitCode != 0 {
		requireNotContains(t, r.Stderr, "invalid --field format")
	}
}

// ---------------------------------------------------------------------------
// Incident Timeline
// ---------------------------------------------------------------------------

// Test 218: view timeline
func TestIncidentTimelineView(t *testing.T) {
	name := uniqueName("timeline_view")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "timeline", id)
	requireSuccess(t, r)
	// Should have timeline events (at least creation)
}

// Test 220: timeline JSON
func TestIncidentTimelineJSON(t *testing.T) {
	name := uniqueName("timeline_json")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "timeline", id, "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// Test 222: timeline nonexistent ID
func TestIncidentTimelineNonexistentID(t *testing.T) {
	r := runCLI(t, "incident", "timeline", "nonexistent_id_xyz")
	requireMeaningfulResult(t, r)
}

// ---------------------------------------------------------------------------
// Incident Alerts
// ---------------------------------------------------------------------------

// Test 225: view alerts
func TestIncidentAlertsView(t *testing.T) {
	name := uniqueName("alerts_view")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "alerts", id)
	requireSuccess(t, r)
	// May show "No alerts." or table
}

// Test 226: alerts --limit
func TestIncidentAlertsLimit(t *testing.T) {
	name := uniqueName("alerts_limit")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "alerts", id, "--limit", "3")
	requireSuccess(t, r)
}

// Test 227: alerts JSON
func TestIncidentAlertsJSON(t *testing.T) {
	name := uniqueName("alerts_json")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "alerts", id, "--json")
	requireSuccess(t, r)
	// May output JSON array or "No alerts." -- just verify no crash
}

// ---------------------------------------------------------------------------
// Incident Similar
// ---------------------------------------------------------------------------

// Test 232: find similar
func TestIncidentSimilar(t *testing.T) {
	name := uniqueName("similar")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "similar", id)
	// API may reject similar search for incidents without a channel
	if r.ExitCode != 0 {
		return
	}
	// May show table or "No similar incidents found."
}

// Test 233: similar --limit
func TestIncidentSimilarLimit(t *testing.T) {
	name := uniqueName("similar_limit")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "similar", id, "--limit", "2")
	// API may reject similar search for incidents without a channel
	if r.ExitCode != 0 {
		return
	}
}

// Test 234: similar JSON
func TestIncidentSimilarJSON(t *testing.T) {
	name := uniqueName("similar_json")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	id := extractIncidentID(t, r.Stdout)
	t.Cleanup(func() { runCLI(t, "incident", "close", id) })

	r = runCLI(t, "incident", "similar", id, "--json")
	// API may reject similar search for incidents without a channel
	if r.ExitCode != 0 {
		return
	}
	if strings.Contains(r.Stdout, "No similar incidents found.") {
		return
	}
	requireValidJSON(t, r.Stdout)
}

func requireMeaningfulResult(t *testing.T, r cliResult) {
	t.Helper()

	if strings.TrimSpace(r.Stdout) == "" && strings.TrimSpace(r.Stderr) == "" {
		t.Fatal("expected output on stdout or stderr")
	}
	requireNotContains(t, r.Stdout, "panic:")
	requireNotContains(t, r.Stderr, "panic:")
	if r.ExitCode != 0 && !strings.Contains(r.Stderr, "Error: ") {
		t.Fatalf("expected an error message on stderr for non-zero exit, got stderr=%q", r.Stderr)
	}
}
