//go:build e2e

package e2e_test

import (
	"testing"
)

// Test 158: incident list default (last 24h)
func TestIncidentListDefault(t *testing.T) {
	r := runCLI(t, "incident", "list")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "ID", "TITLE", "SEVERITY", "PROGRESS", "CHANNEL", "CREATED")
	requireContains(t, r.Stdout, "Showing")
}

// Test 179: incident list JSON
func TestIncidentListJSON(t *testing.T) {
	r := runCLI(t, "incident", "list", "--json", "--since", "168h")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// Test 177: --since invalid
func TestIncidentListSinceInvalid(t *testing.T) {
	r := runCLI(t, "incident", "list", "--since", "garbage")
	requireFailure(t, r)
}

// Test 178: --until invalid
func TestIncidentListUntilInvalid(t *testing.T) {
	r := runCLI(t, "incident", "list", "--until", "garbage")
	requireFailure(t, r)
}

// Test 181: empty results
func TestIncidentListEmpty(t *testing.T) {
	r := runCLI(t, "incident", "list", "--title", "nonexistent_xyz_999_abc", "--since", "1h")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Showing 0 results")
}

// Test 188: incident get no args
func TestIncidentGetNoArgs(t *testing.T) {
	r := runCLI(t, "incident", "get")
	requireFailure(t, r)
}

// Test 202: incident ack no args
func TestIncidentAckNoArgs(t *testing.T) {
	r := runCLI(t, "incident", "ack")
	requireFailure(t, r)
}

// Test 206: incident close no args
func TestIncidentCloseNoArgs(t *testing.T) {
	r := runCLI(t, "incident", "close")
	requireFailure(t, r)
}

// Test 221: incident timeline no args
func TestIncidentTimelineNoArgs(t *testing.T) {
	r := runCLI(t, "incident", "timeline")
	requireFailure(t, r)
}

// Test 228: incident alerts no args
func TestIncidentAlertsNoArgs(t *testing.T) {
	r := runCLI(t, "incident", "alerts")
	requireFailure(t, r)
}

// Test 235: incident similar no args
func TestIncidentSimilarNoArgs(t *testing.T) {
	r := runCLI(t, "incident", "similar")
	requireFailure(t, r)
}

// Test 192: Full lifecycle (create -> get -> ack -> get -> close -> get)
// This test is sequential and creates a real incident, so it runs as a single test.
func TestIncidentLifecycle(t *testing.T) {
	name := uniqueName("lifecycle")

	// Step 1: Create incident
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Incident created")
	id := extractIncidentID(t, r.Stdout)
	if id == "" {
		t.Fatal("no incident ID returned from create")
	}

	// Cleanup: always try to close the incident at the end.
	t.Cleanup(func() {
		runCLI(t, "incident", "close", id)
	})

	// Step 2: Get - should be Triggered
	r = runCLI(t, "incident", "get", id)
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Triggered")
	requireContains(t, r.Stdout, name)

	// Step 3: Ack
	r = runCLI(t, "incident", "ack", id)
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Acknowledged 1 incident(s).")

	// Step 4: Get - should be Processing
	r = runCLI(t, "incident", "get", id)
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Processing")

	// Step 5: Close
	r = runCLI(t, "incident", "close", id)
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Closed 1 incident(s).")

	// Step 6: Get - should be Closed
	r = runCLI(t, "incident", "get", id)
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Closed")
}

// Test 196: Create minimal (title + severity only)
func TestIncidentCreateMinimal(t *testing.T) {
	name := uniqueName("minimal")
	r := runCLI(t, "incident", "create", "--title", name, "--severity", "Info")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Incident created")
	id := extractIncidentID(t, r.Stdout)
	// Cleanup
	t.Cleanup(func() {
		runCLI(t, "incident", "close", id)
	})
}
