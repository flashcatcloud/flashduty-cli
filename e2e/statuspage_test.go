//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// getFirstStatusPageID gets the first status page ID from the API.
func getFirstStatusPageID(t *testing.T) string {
	t.Helper()
	r := runCLI(t, "statuspage", "list", "--json")
	requireSuccess(t, r)
	var pages []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(r.Stdout)), &pages); err != nil {
		t.Skipf("could not parse statuspage list JSON: %v", err)
	}
	if len(pages) == 0 {
		t.Skip("no status pages available")
	}
	id := pages[0]["page_id"]
	switch v := id.(type) {
	case float64:
		return fmt.Sprintf("%.0f", v)
	case string:
		return v
	default:
		t.Skipf("unexpected page_id type: %T", id)
		return ""
	}
}

func stringifyNumericID(t *testing.T, value any, field string) string {
	t.Helper()
	switch v := value.(type) {
	case float64:
		return fmt.Sprintf("%.0f", v)
	case string:
		return v
	default:
		t.Fatalf("unexpected %s type: %T", field, value)
		return ""
	}
}

func getStatusPageChanges(t *testing.T, pageID, changeType string) []map[string]any {
	t.Helper()

	r := runCLI(t, "statuspage", "changes", "--page-id", pageID, "--type", changeType, "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)

	var changes []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(r.Stdout)), &changes); err != nil {
		t.Fatalf("could not parse statuspage changes JSON: %v\n%s", err, r.Stdout)
	}
	if len(changes) == 0 {
		t.Logf("no %s statuspage changes returned for page_id=%s", changeType, pageID)
	}
	return changes
}

// Test 249: statuspage list --id filter
func TestStatusPageListByID(t *testing.T) {
	id := getFirstStatusPageID(t)
	r := runCLI(t, "statuspage", "list", "--id", id, "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)

	var pages []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(r.Stdout)), &pages); err != nil {
		t.Fatalf("could not parse statuspage list JSON: %v\n%s", err, r.Stdout)
	}
	if len(pages) == 0 {
		t.Skipf("statuspage list returned no rows for known page_id=%s", id)
	}
	for _, page := range pages {
		pageID := stringifyNumericID(t, page["page_id"], "page_id")
		if pageID != id {
			t.Fatalf("expected page_id=%s, got %s", id, pageID)
		}
	}
}

// Test 251: statuspage list --id invalid
func TestStatusPageListByIDInvalid(t *testing.T) {
	r := runCLI(t, "statuspage", "list", "--id", "abc")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "invalid --id")
}

// Test 253: statuspage changes --type incident
func TestStatusPageChangesIncident(t *testing.T) {
	id := getFirstStatusPageID(t)
	changes := getStatusPageChanges(t, id, "incident")
	for _, change := range changes {
		if got := stringifyNumericID(t, change["page_id"], "page_id"); got != id {
			t.Fatalf("expected page_id=%s, got %s", id, got)
		}
		if got, ok := change["type"].(string); !ok || got != "incident" {
			t.Fatalf("expected change type incident, got %#v", change["type"])
		}
	}
}

// Test 254: statuspage changes --type maintenance
func TestStatusPageChangesMaintenance(t *testing.T) {
	id := getFirstStatusPageID(t)
	changes := getStatusPageChanges(t, id, "maintenance")
	for _, change := range changes {
		if got := stringifyNumericID(t, change["page_id"], "page_id"); got != id {
			t.Fatalf("expected page_id=%s, got %s", id, got)
		}
		if got, ok := change["type"].(string); !ok || got != "maintenance" {
			t.Fatalf("expected change type maintenance, got %#v", change["type"])
		}
	}
}

// Test 255: statuspage changes missing --page-id
func TestStatusPageChangesMissingPageID(t *testing.T) {
	r := runCLI(t, "statuspage", "changes", "--type", "incident")
	requireFailure(t, r)
}

// Test 256: statuspage changes missing --type
func TestStatusPageChangesMissingType(t *testing.T) {
	id := getFirstStatusPageID(t)
	r := runCLI(t, "statuspage", "changes", "--page-id", id)
	requireFailure(t, r)
}

// Test 257: statuspage changes JSON
func TestStatusPageChangesJSON(t *testing.T) {
	id := getFirstStatusPageID(t)
	changes := getStatusPageChanges(t, id, "incident")
	for _, change := range changes {
		if got := stringifyNumericID(t, change["page_id"], "page_id"); got != id {
			t.Fatalf("expected page_id=%s, got %s", id, got)
		}
		if got, ok := change["type"].(string); !ok || got != "incident" {
			t.Fatalf("expected change type incident, got %#v", change["type"])
		}
	}
}

// Test 261: statuspage create-incident missing --page-id
func TestStatusPageCreateIncidentMissingPageID(t *testing.T) {
	r := runCLI(t, "statuspage", "create-incident", "--title", "test")
	requireFailure(t, r)
}

// Test 262: statuspage create-incident missing --title
func TestStatusPageCreateIncidentMissingTitle(t *testing.T) {
	r := runCLI(t, "statuspage", "create-incident", "--page-id", "1")
	requireFailure(t, r)
}

// Test 267: statuspage create-timeline missing --page-id
func TestStatusPageCreateTimelineMissingPageID(t *testing.T) {
	r := runCLI(t, "statuspage", "create-timeline", "--change", "1", "--message", "test")
	requireFailure(t, r)
}

// Test 268: statuspage create-timeline missing --change
func TestStatusPageCreateTimelineMissingChange(t *testing.T) {
	r := runCLI(t, "statuspage", "create-timeline", "--page-id", "1", "--message", "test")
	requireFailure(t, r)
}

// Test 269: statuspage create-timeline missing --message
func TestStatusPageCreateTimelineMissingMessage(t *testing.T) {
	r := runCLI(t, "statuspage", "create-timeline", "--page-id", "1", "--change", "1")
	requireFailure(t, r)
}
