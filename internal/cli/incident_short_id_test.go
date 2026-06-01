package cli

import (
	"strings"
	"testing"
)

const (
	testShortID = "311510"
	testFullID  = "6a12a4502f0a2396b3311510"
)

func incidentListData(items ...map[string]any) map[string]any {
	return map[string]any{"items": items, "total": len(items)}
}

func incidentItem(id, num, title string) map[string]any {
	return map[string]any{"incident_id": id, "num": num, "title": title}
}

// TestIncidentDetailShortIDResolves: `detail <6-hex>` first resolves the short
// id via /incident/list (nums + a 30-day window), then fetches /incident/info
// with the RESOLVED full id — never the short id.
func TestIncidentDetailShortIDResolves(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	var paths []string
	stub.dataForPath = func(path string, body map[string]any) any {
		paths = append(paths, path)
		switch path {
		case "/incident/list":
			return incidentListData(incidentItem(testFullID, testShortID, "kafka backlog"))
		case "/incident/info":
			return incidentItem(testFullID, testShortID, "kafka backlog")
		default:
			return nil
		}
	}

	if _, err := execCommand("incident", "detail", testShortID); err != nil {
		t.Fatalf("execCommand: %v", err)
	}

	if want := []string{"/incident/list", "/incident/info"}; !equalStrings(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}

	// Resolve call: nums set, 30-day span.
	listBody := stub.bodies[0]
	nums, _ := listBody["nums"].([]any)
	if len(nums) != 1 || nums[0] != testShortID {
		t.Errorf("resolve nums = %#v, want [%q]", listBody["nums"], testShortID)
	}
	st, _ := listBody["start_time"].(float64)
	et, _ := listBody["end_time"].(float64)
	if et <= st {
		t.Errorf("end_time %v must be > start_time %v", et, st)
	}
	if span := et - st; span != float64(shortIDResolveDays*24*60*60) {
		t.Errorf("resolve span = %v s, want %d-day span", span, shortIDResolveDays)
	}

	// Detail call: the resolved full id, not the short id.
	if got := stub.bodies[1]["incident_id"]; got != testFullID {
		t.Errorf("info incident_id = %#v, want resolved full id %q", got, testFullID)
	}
}

// TestIncidentDetailShortIDAmbiguous: a short id that matches >1 incident fails
// with a candidate list and never calls /incident/info (no silent wrong-pick).
func TestIncidentDetailShortIDAmbiguous(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	other := "6a0ea49000000000b3311510"
	var paths []string
	stub.dataForPath = func(path string, body map[string]any) any {
		paths = append(paths, path)
		if path == "/incident/list" {
			return incidentListData(
				incidentItem(testFullID, testShortID, "kafka backlog"),
				incidentItem(other, testShortID, "another incident"),
			)
		}
		return nil
	}

	_, err := execCommand("incident", "detail", testShortID)
	if err == nil {
		t.Fatal("want ambiguous error, got nil")
	}
	if !strings.Contains(err.Error(), testFullID) || !strings.Contains(err.Error(), other) {
		t.Errorf("error should list both candidate ids, got: %v", err)
	}
	if want := []string{"/incident/list"}; !equalStrings(paths, want) {
		t.Errorf("paths = %v, want %v (info must be skipped on ambiguity)", paths, want)
	}
}

// TestIncidentDetailShortIDNotFound: a short id with no match in the window
// fails with a descriptive error pointing at the full-id fallback.
func TestIncidentDetailShortIDNotFound(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.dataForPath = func(path string, body map[string]any) any {
		if path == "/incident/list" {
			return incidentListData()
		}
		return nil
	}

	_, err := execCommand("incident", "detail", testShortID)
	if err == nil || !strings.Contains(err.Error(), "no incident with short id") {
		t.Errorf("want not-found error, got %v", err)
	}
}

// TestIncidentDetailFullIDSkipsResolve: a 24-hex id goes straight to
// /incident/info with no resolve round-trip (existing behavior preserved).
func TestIncidentDetailFullIDSkipsResolve(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	var paths []string
	stub.dataForPath = func(path string, body map[string]any) any {
		paths = append(paths, path)
		if path == "/incident/info" {
			return incidentItem(testFullID, testShortID, "kafka backlog")
		}
		return nil
	}

	if _, err := execCommand("incident", "detail", testFullID); err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if want := []string{"/incident/info"}; !equalStrings(paths, want) {
		t.Fatalf("paths = %v, want %v (full id must not trigger a resolve)", paths, want)
	}
	if got := stub.bodies[0]["incident_id"]; got != testFullID {
		t.Errorf("info incident_id = %#v, want %q", got, testFullID)
	}
}

// TestIncidentGetShortIDResolves: `get <6-hex>` resolves the short id, then
// fetches by the resolved full id via incident_ids.
func TestIncidentGetShortIDResolves(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.dataForPath = func(path string, body map[string]any) any {
		// Both the resolve and the final fetch hit /incident/list; the canned
		// row is fine for either.
		return incidentListData(incidentItem(testFullID, testShortID, "kafka backlog"))
	}

	if _, err := execCommand("incident", "get", testShortID); err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if stub.requests != 2 {
		t.Fatalf("requests = %d, want 2 (resolve + fetch)", stub.requests)
	}

	// Resolve sent nums; final fetch sent the resolved full id via incident_ids.
	if _, ok := stub.bodies[0]["nums"]; !ok {
		t.Errorf("first request should carry nums, got %#v", stub.bodies[0])
	}
	ids, _ := stub.bodies[1]["incident_ids"].([]any)
	if len(ids) != 1 || ids[0] != testFullID {
		t.Errorf("fetch incident_ids = %#v, want [%q]", stub.bodies[1]["incident_ids"], testFullID)
	}
}

// TestIncidentListNumsReachesWire: --nums is split and sent as the nums array.
func TestIncidentListNumsReachesWire(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = incidentListData()

	if _, err := execCommand("incident", "list", "--nums", "311510,ABC123"); err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if stub.lastPath != "/incident/list" {
		t.Fatalf("path = %q, want /incident/list", stub.lastPath)
	}
	nums, _ := stub.lastBody["nums"].([]any)
	if len(nums) != 2 || nums[0] != "311510" || nums[1] != "ABC123" {
		t.Errorf("nums = %#v, want [311510 ABC123]", stub.lastBody["nums"])
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
