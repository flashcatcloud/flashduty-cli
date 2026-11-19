package cli

import (
	"strings"
	"sync"
	"testing"
	"time"
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

// TestIncidentGetShortIDResolves: `get <6-hex>` resolves the short id via
// /incident/list, then fetches by the resolved full id via /incident/list-by-ids.
func TestIncidentGetShortIDResolves(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	var paths []string
	stub.dataForPath = func(path string, body map[string]any) any {
		paths = append(paths, path)
		return incidentListData(incidentItem(testFullID, testShortID, "kafka backlog"))
	}

	if _, err := execCommand("incident", "get", testShortID); err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if want := []string{"/incident/list", "/incident/list-by-ids"}; !equalStrings(paths, want) {
		t.Fatalf("paths = %v, want %v (resolve + fetch)", paths, want)
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

// TestIncidentGetFullIDUsesListByIDs: a full id skips the resolve and fetches
// through /incident/list-by-ids. /incident/list rejects a request without a
// start_time/end_time window, so the id lookup must not go through it.
func TestIncidentGetFullIDUsesListByIDs(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	var paths []string
	stub.dataForPath = func(path string, body map[string]any) any {
		paths = append(paths, path)
		return incidentListData(incidentItem(testFullID, testShortID, "kafka backlog"))
	}

	if _, err := execCommand("incident", "get", testFullID); err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if want := []string{"/incident/list-by-ids"}; !equalStrings(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
	ids, _ := stub.bodies[0]["incident_ids"].([]any)
	if len(ids) != 1 || ids[0] != testFullID {
		t.Errorf("incident_ids = %#v, want [%q]", stub.bodies[0]["incident_ids"], testFullID)
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

// shortIDStub answers /incident/list with matches (the short-id lookup) and
// every other path with an empty list, recording each request in order.
func shortIDStub(t *testing.T, matches ...map[string]any) (paths *[]string, bodies *[]map[string]any) {
	t.Helper()
	stub := newGFStub(t)
	var mu sync.Mutex
	var p []string
	var b []map[string]any
	stub.dataForPath = func(path string, body map[string]any) any {
		mu.Lock()
		p = append(p, path)
		b = append(b, body)
		mu.Unlock()
		if path == "/incident/list" {
			return incidentListData(matches...)
		}
		return map[string]any{"items": []any{}, "total": 0}
	}
	return &p, &b
}

// incidentIDCommands are incident subcommands taking one incident id, with
// the endpoint and body field the id reaches.
var incidentIDCommands = []struct {
	args  []string
	path  string
	field string
}{
	{[]string{"incident", "alerts"}, "/incident/alert/list", "incident_id"},
	{[]string{"incident", "timeline"}, "/incident/feed", "incident_id"},
	{[]string{"incident", "similar"}, "/incident/past/list", "incident_id"},
	{[]string{"incident", "feed"}, "/incident/feed", "incident_id"},
	{[]string{"incident", "war-room", "list"}, "/incident/war-room/list", "incident_id"},
	{[]string{"incident", "alert-list"}, "/incident/alert/list", "incident_id"},
	{[]string{"incident", "past-list"}, "/incident/past/list", "incident_id"},
}

// TestIncidentSubcommandsResolveShortID: each subcommand resolves a short id
// through /incident/list and sends the full id.
func TestIncidentSubcommandsResolveShortID(t *testing.T) {
	for _, c := range incidentIDCommands {
		t.Run(strings.Join(c.args, " "), func(t *testing.T) {
			saveAndResetGlobals(t)
			paths, bodies := shortIDStub(t, incidentItem(testFullID, testShortID, "High error rate"))
			if _, err := execCommand(append(c.args, testShortID)...); err != nil {
				t.Fatalf("execCommand: %v", err)
			}
			if len(*paths) < 2 || (*paths)[0] != "/incident/list" || (*paths)[1] != c.path {
				t.Fatalf("paths = %v, want /incident/list then %s", *paths, c.path)
			}
			if got := (*bodies)[1][c.field]; got != testFullID {
				t.Errorf("%s = %#v, want %q", c.field, got, testFullID)
			}
		})
	}
}

// TestIncidentSubcommandsFullIDNoLookup: a full id is sent as given, with no
// /incident/list request.
func TestIncidentSubcommandsFullIDNoLookup(t *testing.T) {
	for _, c := range incidentIDCommands {
		t.Run(strings.Join(c.args, " "), func(t *testing.T) {
			saveAndResetGlobals(t)
			paths, bodies := shortIDStub(t)
			if _, err := execCommand(append(c.args, testFullID)...); err != nil {
				t.Fatalf("execCommand: %v", err)
			}
			if len(*paths) == 0 || (*paths)[0] != c.path {
				t.Fatalf("paths = %v, want %s first", *paths, c.path)
			}
			for _, p := range *paths {
				if p == "/incident/list" {
					t.Fatalf("paths = %v: a full id must not be looked up", *paths)
				}
			}
			if got := (*bodies)[0][c.field]; got != testFullID {
				t.Errorf("%s = %#v, want %q", c.field, got, testFullID)
			}
		})
	}
}

// TestIncidentMultiIDAmbiguousShortIDSendsNothing: when one of several ids is
// an ambiguous short id, a write command reports the candidates and sends no
// write — for curated and generated commands, ids given positionally or in
// --data.
func TestIncidentMultiIDAmbiguousShortIDSendsNothing(t *testing.T) {
	other := "64f0c0ffee0123456789beef"
	for _, args := range [][]string{
		{"incident", "close", testFullID, testShortID},
		{"incident", "ack", testFullID, testShortID},
		{"incident", "ack", "--data", `{"incident_ids":["` + testFullID + `","` + testShortID + `"]}`},
		{"incident", "merge", testFullID, "--source", testShortID},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			saveAndResetGlobals(t)
			paths, _ := shortIDStub(t,
				incidentItem(testFullID, testShortID, "High error rate"),
				incidentItem(other, testShortID, "Disk almost full"))
			_, err := execCommand(args...)
			if err == nil || !strings.Contains(err.Error(), testFullID) || !strings.Contains(err.Error(), other) ||
				!strings.Contains(err.Error(), "Disk almost full") {
				t.Fatalf("want an error listing both candidates, got %v", err)
			}
			if want := []string{"/incident/list"}; !equalStrings(*paths, want) {
				t.Errorf("paths = %v, want %v (no write on ambiguity)", *paths, want)
			}
		})
	}
}

// TestIncidentMultiIDShortIDsResolve: every short id of a multi-id write is
// resolved before the single write request.
func TestIncidentMultiIDShortIDsResolve(t *testing.T) {
	saveAndResetGlobals(t)
	paths, bodies := shortIDStub(t, incidentItem(testFullID, testShortID, "High error rate"))
	if _, err := execCommand("incident", "snooze", testShortID, "64f0c0ffee0123456789abcd", "--duration", "30m"); err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if want := []string{"/incident/list", "/incident/snooze"}; !equalStrings(*paths, want) {
		t.Fatalf("paths = %v, want %v", *paths, want)
	}
	if got := stringsField((*bodies)[1], "incident_ids"); !equalStrings(got, []string{testFullID, "64f0c0ffee0123456789abcd"}) {
		t.Errorf("incident_ids = %v", got)
	}
}

// TestIncidentFlagErrorsBeforeShortIDLookup: flag validation fails before the
// short-id lookup, so a flag error costs no request and is not masked by a
// lookup error.
func TestIncidentFlagErrorsBeforeShortIDLookup(t *testing.T) {
	many := strings.TrimSuffix(strings.Repeat(testFullID+",", 101), ",")
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"incident", "snooze", testShortID, "--duration", "bogus"}, "invalid --duration"},
		{[]string{"incident", "merge", testShortID, "--source", many}, "--source accepts at most 100"},
		{[]string{"incident", "reassign", testShortID, "--person", "abc"}, "invalid --person"},
		{[]string{"incident", "similar", testShortID, "--fields", ",", "--output-format", "json"}, "--fields must name"},
		{[]string{"incident", "detail", testShortID, "--fields", ",", "--output-format", "json"}, "--fields must name"},
	} {
		t.Run(strings.Join(c.args[:2], " "), func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			_, err := execCommand(c.args...)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want %q", err, c.want)
			}
			if stub.requests != 0 {
				t.Errorf("%d requests, want 0", stub.requests)
			}
		})
	}
}

// TestAlertMergeResolvesShortIncidentID: alert merge resolves a short id in
// --incident-id like the incident commands.
func TestAlertMergeResolvesShortIncidentID(t *testing.T) {
	saveAndResetGlobals(t)
	paths, bodies := shortIDStub(t, incidentItem(testFullID, testShortID, "High error rate"))
	if _, err := execCommand("alert", "merge", "a00000000000000000000001", "--incident-id", testShortID); err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if want := []string{"/incident/list", "/alert/merge"}; !equalStrings(*paths, want) {
		t.Fatalf("paths = %v, want %v", *paths, want)
	}
	if got := (*bodies)[1]["incident_id"]; got != testFullID {
		t.Errorf("incident_id = %#v, want %q", got, testFullID)
	}
}

// TestIncidentWriteNotesShortIDResolution: a write given a short id states on
// stderr which incident it resolved to before acting; stdout is what a full
// id prints.
func TestIncidentWriteNotesShortIDResolution(t *testing.T) {
	pinClock(t, time.Date(2026, 1, 2, 7, 0, 0, 0, time.UTC))
	match := incidentItem(testFullID, testShortID, "High error rate")
	match["start_time"] = time.Date(2026, 1, 2, 6, 0, 0, 0, time.UTC).Unix()
	note := "resolved " + testShortID + " → " + testFullID + ` "High error rate" started 2026-01-02T14:00:00+08:00` + "\n"
	for _, args := range [][]string{
		{"incident", "close"},
		{"incident", "ack"},
		{"incident", "remove", "--force"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			saveAndResetGlobals(t)
			shortIDStub(t, match)
			fullOut, fullErr, err := execCommandSplit(append(args, testFullID)...)
			if err != nil {
				t.Fatalf("full id: %v", err)
			}
			saveAndResetGlobals(t)
			shortIDStub(t, match)
			out, errOut, err := execCommandSplit(append(args, testShortID)...)
			if err != nil {
				t.Fatalf("short id: %v", err)
			}
			if out != fullOut {
				t.Errorf("stdout = %q, want %q (as for the full id)", out, fullOut)
			}
			if errOut != note+fullErr {
				t.Errorf("stderr = %q, want %q", errOut, note+fullErr)
			}
		})
	}
}

// TestIncidentReadNoShortIDNote: a read resolves a short id silently.
func TestIncidentReadNoShortIDNote(t *testing.T) {
	for _, args := range [][]string{{"incident", "timeline"}, {"incident", "alert-list"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			saveAndResetGlobals(t)
			shortIDStub(t, incidentItem(testFullID, testShortID, "High error rate"))
			_, errOut, err := execCommandSplit(append(args, testShortID)...)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(errOut, "resolved ") {
				t.Errorf("stderr = %q, want no resolution note on a read", errOut)
			}
		})
	}
}

// TestResolvedPrompt: a confirmation prompt names the incidents short ids
// resolved to.
func TestResolvedPrompt(t *testing.T) {
	ctx := &RunContext{resolved: []string{`resolved 311510 → 6a12a4502f0a2396b3311510 "High error rate" started 2026-01-02T14:00:00+08:00`}}
	want := `resolved 311510 → 6a12a4502f0a2396b3311510 "High error rate" started 2026-01-02T14:00:00+08:00` + "\nAre you sure?"
	if got := resolvedPrompt(ctx, "Are you sure?"); got != want {
		t.Errorf("prompt = %q, want %q", got, want)
	}
	if got := resolvedPrompt(&RunContext{}, "Are you sure?"); got != "Are you sure?" {
		t.Errorf("prompt without short ids = %q", got)
	}
}
