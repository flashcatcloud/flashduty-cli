//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// The CLI group is `status-page` (hyphenated) and its verbs come straight from
// the API paths — `change-active-list`, `change-create`,
// `change-timeline-create`, … — see internal/cli/zz_generated_status_pages.go.
// `statuspage`, `create-incident` and `create-timeline` never existed on this
// command tree.
//
// Generated list commands print their response envelope verbatim, so `--json`
// output is {"items":[...]} — an object, NOT a top-level array.

// statusPageItems unmarshals the {"items":[...]} envelope of a status-page list
// response.
func statusPageItems(stdout string) ([]map[string]any, error) {
	var envelope struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &envelope); err != nil {
		return nil, err
	}
	return envelope.Items, nil
}

// getFirstStatusPageID gets the first status page ID from the API.
func getFirstStatusPageID(t *testing.T) string {
	t.Helper()
	r := runCLI(t, "status-page", "list", "--json")
	requireSuccess(t, r)
	pages, err := statusPageItems(r.Stdout)
	if err != nil {
		t.Skipf("could not parse status-page list JSON: %v", err)
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

// getStatusPageChanges lists the in-progress events of one type for a page.
// `page-id` is POSITIONAL on change-active-list; only `--type` is a flag.
func getStatusPageChanges(t *testing.T, pageID, changeType string) []map[string]any {
	t.Helper()

	r := runCLI(t, "status-page", "change-active-list", pageID, "--type", changeType, "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)

	changes, err := statusPageItems(r.Stdout)
	if err != nil {
		t.Fatalf("could not parse status-page change-active-list JSON: %v\n%s", err, r.Stdout)
	}
	if len(changes) == 0 {
		t.Logf("no active %s events returned for page_id=%s", changeType, pageID)
	}
	return changes
}

// Test 249: status-page info <page-id> returns the requested page.
// There is no `--id` filter on `list` (its only flag is `--data`), so a
// single-page lookup goes through `info`, whose response is a bare object.
func TestStatusPageInfoByID(t *testing.T) {
	id := getFirstStatusPageID(t)
	r := runCLI(t, "status-page", "info", id, "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)

	var page map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(r.Stdout)), &page); err != nil {
		t.Fatalf("could not parse status-page info JSON: %v\n%s", err, r.Stdout)
	}
	if got := stringifyNumericID(t, page["page_id"], "page_id"); got != id {
		t.Fatalf("expected page_id=%s, got %s", id, got)
	}
}

// Test 251: a non-numeric positional page id is rejected before any request.
func TestStatusPageChangeActiveListInvalidPageID(t *testing.T) {
	r := runCLI(t, "status-page", "change-active-list", "abc", "--type", "incident")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "invalid page_id")
}

// Test 253: change-active-list --type incident
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

// Test 254: change-active-list --type maintenance
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

// Test 255: change-active-list without the <page-id> positional
func TestStatusPageChangesMissingPageID(t *testing.T) {
	r := runCLI(t, "status-page", "change-active-list", "--type", "incident")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "missing page_id")
}

// Test 256: change-active-list missing --type
func TestStatusPageChangesMissingType(t *testing.T) {
	id := getFirstStatusPageID(t)
	r := runCLI(t, "status-page", "change-active-list", id)
	requireFailure(t, r)
	requireContains(t, r.Stderr, "missing required request fields: type")
}

// Test 257: change-active-list JSON
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

// Test 261: change-create without the <page-id> positional (and without the
// --page-id / --data fallbacks) fails in the argument validator.
func TestStatusPageChangeCreateMissingPageID(t *testing.T) {
	r := runCLI(t, "status-page", "change-create", "--title", "test")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "missing page_id")
}

// Test 262: change-create with only the positional reports every missing
// required body field.
func TestStatusPageChangeCreateMissingTitle(t *testing.T) {
	r := runCLI(t, "status-page", "change-create", "1", "--type", "incident", "--status", "investigating")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "title")
}

// Test 263: `updates` (and the nested `component_changes`) has NO typed flag, so
// a change-create that sets every scalar flag still fails until --data supplies
// it. This is the invariant that makes --data mandatory on real calls.
func TestStatusPageChangeCreateRequiresUpdatesViaData(t *testing.T) {
	r := runCLI(t, "status-page", "change-create", "1",
		"--type", "incident", "--status", "investigating",
		"--title", "e2e", "--description", "e2e")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "missing required request fields: updates")
}

// Test 267: change-timeline-create missing --page-id (both IDs are flags here,
// this command has no positional).
func TestStatusPageChangeTimelineCreateMissingPageID(t *testing.T) {
	r := runCLI(t, "status-page", "change-timeline-create", "--change-id", "1", "--status", "investigating", "--description", "test")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "missing required request fields: page_id")
}

// Test 268: change-timeline-create missing --change-id
func TestStatusPageChangeTimelineCreateMissingChangeID(t *testing.T) {
	r := runCLI(t, "status-page", "change-timeline-create", "--page-id", "1", "--status", "investigating", "--description", "test")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "missing required request fields: change_id")
}

// Test 269: change-timeline-create missing --status.
// (--description is `omitempty` in the SDK request, so omitting it is caught by
// the server, not locally; --status is the locally-validated required field.)
func TestStatusPageChangeTimelineCreateMissingStatus(t *testing.T) {
	r := runCLI(t, "status-page", "change-timeline-create", "--page-id", "1", "--change-id", "1", "--description", "test")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "missing required request fields: status")
}
