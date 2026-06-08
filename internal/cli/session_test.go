package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flashcatcloud/go-flashduty"
)

// TestSessionListFlags asserts the curated flag surface: --app defaults to
// ai-sre and --limit to 200. These defaults are what makes `session list` pipe
// cleanly into the /insight skill without extra flags. Output format is the
// inherited global --output-format flag (default jsonl for session commands),
// so there is no bespoke local --format flag; resolution is covered by
// TestResolveSessionFormat.
func TestSessionListFlags(t *testing.T) {
	cmd := newSessionListCmd()
	flags := cmd.Flags()

	cases := []struct {
		name string
		def  string
	}{
		{"app", "ai-sre"},
		{"limit", "200"},
	}
	for _, c := range cases {
		f := flags.Lookup(c.name)
		if f == nil {
			t.Fatalf("flag --%s not registered", c.name)
		}
		if f.DefValue != c.def {
			t.Errorf("--%s default = %q, want %q", c.name, f.DefValue, c.def)
		}
	}

	// The bespoke --format flag is gone; output format rides the global flag.
	if f := flags.Lookup("format"); f != nil {
		t.Errorf("--format must not be registered; use the global --output-format")
	}

	if f := flags.Lookup("team-id"); f == nil || f.Value.Type() != "int64" {
		t.Errorf("--team-id must be an int64 flag, got %v", f)
	}
}

// TestCommandSessionListJSONL drives `session list` against the stub: it must hit
// /safari/session/list, forward app_name + team_ids + scope, and emit one JSON
// object per session line (jsonl) so downstream tooling can stream the rows.
func TestCommandSessionListJSONL(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"sessions": []map[string]any{
			{"session_id": "sess-1", "session_name": "disk full", "app_name": "ai-sre", "updated_at": 1779432894000},
			{"session_id": "sess-2", "session_name": "oom kill", "app_name": "ai-sre", "updated_at": 1779432895000},
		},
		"total": 2,
	}

	out, err := execCommand("session", "list", "--app", "ai-sre", "--team-id", "42", "--scope", "all", "--limit", "50")
	if err != nil {
		t.Fatalf("[session-list] unexpected error: %v", err)
	}
	if stub.lastPath != "/safari/session/list" {
		t.Fatalf("[session-list] expected /safari/session/list, got %q", stub.lastPath)
	}
	if stub.lastBody["app_name"] != "ai-sre" {
		t.Errorf("[session-list] app_name = %v, want ai-sre", stub.lastBody["app_name"])
	}
	if stub.lastBody["scope"] != "all" {
		t.Errorf("[session-list] scope = %v, want all", stub.lastBody["scope"])
	}
	teamIDs, ok := stub.lastBody["team_ids"].([]any)
	if !ok || len(teamIDs) != 1 || fmt.Sprintf("%v", teamIDs[0]) != "42" {
		t.Errorf("[session-list] team_ids = %v, want [42]", stub.lastBody["team_ids"])
	}

	// jsonl: exactly one JSON object per session, no envelope.
	lines := nonEmptyLines(out)
	if len(lines) != 2 {
		t.Fatalf("[session-list] expected 2 jsonl lines, got %d:\n%s", len(lines), out)
	}
	var first flashduty.SessionItem
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("[session-list] line 0 is not a SessionItem: %v", err)
	}
	if first.SessionID != "sess-1" {
		t.Errorf("[session-list] first session = %q, want sess-1", first.SessionID)
	}
}

// TestResolveSessionFormat covers the session output-format resolver: it reads
// the global --output-format / --json flags, defaults to jsonl (not table),
// accepts jsonl/json/toon, and errors on anything else so a typo fails fast.
func TestResolveSessionFormat(t *testing.T) {
	cases := []struct {
		name    string
		format  string
		json    bool
		want    string
		wantErr bool
	}{
		{"default is jsonl", "", false, sessionFormatJSONL, false},
		{"json bool alias", "", true, sessionFormatJSON, false},
		{"explicit jsonl", "jsonl", false, sessionFormatJSONL, false},
		{"explicit json", "json", false, sessionFormatJSON, false},
		{"explicit toon", "toon", false, sessionFormatTOON, false},
		{"explicit wins over json bool", "jsonl", true, sessionFormatJSONL, false},
		{"case-insensitive", "TOON", false, sessionFormatTOON, false},
		{"table is invalid here", "table", false, "", true},
		{"unknown errors", "yaml", false, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origFormat, origJSON := flagOutputFormat, flagJSON
			defer func() { flagOutputFormat, flagJSON = origFormat, origJSON }()
			flagOutputFormat, flagJSON = tc.format, tc.json

			got, err := resolveSessionFormat()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.format)
				}
				if !strings.Contains(err.Error(), "invalid --output-format") {
					t.Fatalf("expected an invalid --output-format error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("resolveSessionFormat(%q, json=%v) = %q, want %q", tc.format, tc.json, got, tc.want)
			}
		})
	}
}

// TestCommandSessionListOutputFormatJSON proves --output-format json emits the
// whole SessionListResponse envelope (the value that was silently ignored
// before the unification), not jsonl rows.
func TestCommandSessionListOutputFormatJSON(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"sessions": []map[string]any{
			{"session_id": "sess-1", "app_name": "ai-sre", "updated_at": 1779432894000},
			{"session_id": "sess-2", "app_name": "ai-sre", "updated_at": 1779432895000},
		},
		"total": 2,
	}

	out, err := execCommand("session", "list", "--output-format", "json")
	if err != nil {
		t.Fatalf("[session-list-json] unexpected error: %v", err)
	}
	var env flashduty.SessionListResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); err != nil {
		t.Fatalf("[session-list-json] output is not a SessionListResponse envelope: %v\n%s", err, out)
	}
	if env.Total != 2 || len(env.Sessions) != 2 {
		t.Fatalf("[session-list-json] envelope = %+v, want total 2 / 2 sessions", env)
	}
}

// TestCommandSessionListOutputFormatTOON proves --output-format toon routes
// through the TOON encoder (not jsonl, not JSON).
func TestCommandSessionListOutputFormatTOON(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"sessions": []map[string]any{
			{"session_id": "sess-1", "app_name": "ai-sre", "updated_at": 1779432894000},
		},
		"total": 1,
	}

	out, err := execCommand("session", "list", "--output-format", "toon")
	if err != nil {
		t.Fatalf("[session-list-toon] unexpected error: %v", err)
	}
	// TOON list output carries the [N] row-count marker JSON never emits.
	if !strings.Contains(out, "sessions[1]") {
		t.Fatalf("[session-list-toon] expected TOON encoding (sessions[1] marker), got:\n%s", out)
	}
}

// TestCommandSessionExportOutputFormatJSON proves --output-format json buffers
// the NDJSON stream into one JSON array of the event objects.
func TestCommandSessionExportOutputFormatJSON(t *testing.T) {
	saveAndResetGlobals(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = fmt.Fprintln(w, `{"type":"session_meta","session_id":"sess-1"}`)
		_, _ = fmt.Fprintln(w, `{"type":"user_message","seq":1}`)
		_, _ = fmt.Fprintln(w, `{"type":"final_answer","seq":2}`)
	}))
	t.Cleanup(srv.Close)

	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test-key", flashduty.WithBaseURL(srv.URL))
	}

	out, err := execCommand("session", "export", "sess-1", "--output-format", "json")
	if err != nil {
		t.Fatalf("[session-export-json] unexpected error: %v", err)
	}
	var events []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &events); err != nil {
		t.Fatalf("[session-export-json] output is not a JSON array: %v\n%s", err, out)
	}
	if len(events) != 3 {
		t.Fatalf("[session-export-json] expected 3 events in the array, got %d:\n%s", len(events), out)
	}
	if events[0]["type"] != "session_meta" {
		t.Errorf("[session-export-json] first event = %v, want session_meta", events[0]["type"])
	}
}

// TestCommandSessionExportOutputFormatTOON proves --output-format toon encodes
// the buffered events through the TOON encoder.
func TestCommandSessionExportOutputFormatTOON(t *testing.T) {
	saveAndResetGlobals(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = fmt.Fprintln(w, `{"type":"session_meta","session_id":"sess-1"}`)
		_, _ = fmt.Fprintln(w, `{"type":"final_answer","seq":1}`)
	}))
	t.Cleanup(srv.Close)

	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test-key", flashduty.WithBaseURL(srv.URL))
	}

	out, err := execCommand("session", "export", "sess-1", "--output-format", "toon")
	if err != nil {
		t.Fatalf("[session-export-toon] unexpected error: %v", err)
	}
	// A non-empty TOON document mentioning a known field proves it encoded.
	if strings.TrimSpace(out) == "" || !strings.Contains(out, "session_meta") {
		t.Fatalf("[session-export-toon] expected a TOON-encoded transcript, got:\n%s", out)
	}
}

// TestCommandSessionListPaginatesBeyond100 is the regression guard for the
// limit>100 bug: the /safari/session/list handler binds limit with "lte=100",
// so a single request with limit 200 is a hard 400 bind failure, not a clamp.
// `session list --limit 200` must therefore satisfy the request by paginating —
// issuing MULTIPLE page requests each with limit<=100 and advancing p — then
// concatenating the rows. This test serves 250 matching sessions in pages of at
// most 100 and asserts the command (a) never asks for more than 100 in any page,
// (b) advances p across pages, and (c) returns exactly the requested 200 rows.
func TestCommandSessionListPaginatesBeyond100(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	const totalAvailable = 250
	// Serve a page computed from the request's p/limit so we exercise the real
	// loop: each page returns min(limit, remaining) sessions, never more than
	// the server-accepted ceiling.
	stub.dataFor = func(body map[string]any) any {
		p := int(asFloat(body["p"]))
		limit := int(asFloat(body["limit"]))
		if p < 1 {
			p = 1
		}
		if limit > 100 {
			// Mirror the real handler: limit>100 is a bind FAILURE, never a
			// clamp. If the CLI ever sends this, the test must fail loudly.
			t.Fatalf("page request used limit=%d (>100) — server would 400, CLI must paginate", limit)
		}
		offset := (p - 1) * limit
		sessions := make([]map[string]any, 0, limit)
		for i := offset; i < offset+limit && i < totalAvailable; i++ {
			sessions = append(sessions, map[string]any{
				"session_id":   fmt.Sprintf("sess-%03d", i),
				"app_name":     "ai-sre",
				"updated_at":   1779432894000,
				"session_name": fmt.Sprintf("row %d", i),
			})
		}
		return map[string]any{"sessions": sessions, "total": totalAvailable}
	}

	out, err := execCommand("session", "list", "--app", "ai-sre", "--limit", "200", "--output-format", "jsonl")
	if err != nil {
		t.Fatalf("[session-paginate] unexpected error: %v", err)
	}

	// (a) Multiple page requests were issued, and (b) p advanced across them.
	if stub.requests < 2 {
		t.Fatalf("[session-paginate] expected >=2 page requests for limit 200, got %d", stub.requests)
	}
	seenPages := make(map[int]bool)
	for i, b := range stub.bodies {
		limit := int(asFloat(b["limit"]))
		if limit > 100 {
			t.Errorf("[session-paginate] request %d used limit=%d, want <=100", i, limit)
		}
		seenPages[int(asFloat(b["p"]))] = true
	}
	if !seenPages[1] || !seenPages[2] {
		t.Errorf("[session-paginate] expected requests for p=1 and p=2, saw pages %v", seenPages)
	}

	// (c) Exactly 200 rows came back, concatenated and in order across pages.
	lines := nonEmptyLines(out)
	if len(lines) != 200 {
		t.Fatalf("[session-paginate] expected 200 concatenated rows, got %d", len(lines))
	}
	var first, last flashduty.SessionItem
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("[session-paginate] line 0 not a SessionItem: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[199]), &last); err != nil {
		t.Fatalf("[session-paginate] line 199 not a SessionItem: %v", err)
	}
	if first.SessionID != "sess-000" {
		t.Errorf("[session-paginate] first row = %q, want sess-000", first.SessionID)
	}
	if last.SessionID != "sess-199" {
		t.Errorf("[session-paginate] last row = %q, want sess-199", last.SessionID)
	}
}

// TestCommandSessionListStopsWhenServerExhausted proves the loop terminates when
// the server returns fewer rows than requested (a short page) even though
// --limit asks for more, rather than spinning forever.
func TestCommandSessionListStopsWhenServerExhausted(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	const totalAvailable = 130 // exhausts mid-way through page 2
	stub.dataFor = func(body map[string]any) any {
		p := int(asFloat(body["p"]))
		limit := int(asFloat(body["limit"]))
		if limit > 100 {
			t.Fatalf("page request used limit=%d (>100)", limit)
		}
		offset := (p - 1) * limit
		sessions := make([]map[string]any, 0, limit)
		for i := offset; i < offset+limit && i < totalAvailable; i++ {
			sessions = append(sessions, map[string]any{
				"session_id": fmt.Sprintf("sess-%03d", i),
				"app_name":   "ai-sre",
				"updated_at": 1779432894000,
			})
		}
		return map[string]any{"sessions": sessions, "total": totalAvailable}
	}

	out, err := execCommand("session", "list", "--app", "ai-sre", "--limit", "200", "--output-format", "jsonl")
	if err != nil {
		t.Fatalf("[session-exhaust] unexpected error: %v", err)
	}
	lines := nonEmptyLines(out)
	if len(lines) != totalAvailable {
		t.Fatalf("[session-exhaust] expected %d rows (server exhausted), got %d", totalAvailable, len(lines))
	}
	// Page 1 (100) + page 2 (30, short) → exactly 2 requests, no extra spin.
	if stub.requests != 2 {
		t.Errorf("[session-exhaust] expected exactly 2 requests, got %d", stub.requests)
	}
}

// asFloat coerces a decoded JSON number (always float64) to float64, tolerating
// a missing key (returns 0).
func asFloat(v any) float64 {
	f, _ := v.(float64)
	return f
}

// TestCommandSessionListSinceFiltersClientSide proves --since drops rows older
// than the window using the response's updated_at (the API has no time filter).
func TestCommandSessionListSinceFiltersClientSide(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	recent := time.Now().Add(-1 * time.Hour).UnixMilli()
	old := time.Now().Add(-72 * time.Hour).UnixMilli()
	stub.data = map[string]any{
		"sessions": []map[string]any{
			{"session_id": "fresh", "app_name": "ai-sre", "updated_at": recent},
			{"session_id": "stale", "app_name": "ai-sre", "updated_at": old},
		},
		"total": 2,
	}

	out, err := execCommand("session", "list", "--since", "24h")
	if err != nil {
		t.Fatalf("[session-since] unexpected error: %v", err)
	}
	lines := nonEmptyLines(out)
	if len(lines) != 1 {
		t.Fatalf("[session-since] expected 1 row after --since 24h, got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "fresh") {
		t.Errorf("[session-since] expected the fresh session, got: %s", lines[0])
	}
}

// TestCommandSessionListRejectsBadFormat fails fast on an unknown
// --output-format value, with the session-specific message (which lists jsonl
// as a valid value). The global resolver would have rejected it first with a
// different message, so this also proves session commands are exempted from the
// global strict check and validate their own set.
func TestCommandSessionListRejectsBadFormat(t *testing.T) {
	saveAndResetGlobals(t)
	newGFStub(t)

	_, err := execCommand("session", "list", "--output-format", "yaml")
	if err == nil || !strings.Contains(err.Error(), "invalid --output-format") {
		t.Fatalf("expected an invalid --output-format error, got %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "jsonl") {
		t.Fatalf("expected the session-format error (listing jsonl), got %v", err)
	}
}

// TestCommandSessionExportStreamsNDJSON drives `session export` against a stub
// that serves application/x-ndjson. The command must pass session_id +
// include_subagents and write the stream verbatim to stdout, line 1 being a
// session_meta envelope (jq-parseable).
func TestCommandSessionExportStreamsNDJSON(t *testing.T) {
	saveAndResetGlobals(t)

	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Body != nil {
			sc := bufio.NewScanner(r.Body)
			if sc.Scan() {
				gotBody = sc.Text()
			}
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = fmt.Fprintln(w, `{"type":"session_meta","session_id":"sess-1","app_name":"ai-sre"}`)
		_, _ = fmt.Fprintln(w, `{"type":"user_message","seq":1,"content":"disk is full"}`)
		_, _ = fmt.Fprintln(w, `{"type":"tool_call","seq":2,"name":"bash","status":"ok"}`)
		_, _ = fmt.Fprintln(w, `{"type":"final_answer","seq":3,"content":"freed 20G"}`)
	}))
	t.Cleanup(srv.Close)

	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test-key", flashduty.WithBaseURL(srv.URL))
	}

	out, err := execCommand("session", "export", "sess-1", "--include-subagents")
	if err != nil {
		t.Fatalf("[session-export] unexpected error: %v", err)
	}
	if gotPath != "/safari/session/export" {
		t.Fatalf("[session-export] path = %q, want /safari/session/export", gotPath)
	}
	if !strings.Contains(gotBody, `"session_id":"sess-1"`) || !strings.Contains(gotBody, `"include_subagents":true`) {
		t.Errorf("[session-export] request body = %s, want session_id + include_subagents", gotBody)
	}

	lines := nonEmptyLines(out)
	if len(lines) != 4 {
		t.Fatalf("[session-export] expected 4 NDJSON lines, got %d:\n%s", len(lines), out)
	}
	// Line 1 must be a parseable session_meta envelope.
	var meta struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &meta); err != nil {
		t.Fatalf("[session-export] line 1 is not valid JSON: %v", err)
	}
	if meta.Type != "session_meta" || meta.SessionID != "sess-1" {
		t.Errorf("[session-export] line 1 = %+v, want session_meta/sess-1", meta)
	}
}

// TestCommandSessionExportMapsErrorEnvelope confirms a non-2xx JSON error
// envelope on the streaming endpoint surfaces as a CLI error, not a partial
// stream.
func TestCommandSessionExportMapsErrorEnvelope(t *testing.T) {
	saveAndResetGlobals(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req-1",
			"error":      map[string]any{"code": "access_denied", "message": "not your session"},
		})
	}))
	t.Cleanup(srv.Close)

	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test-key", flashduty.WithBaseURL(srv.URL))
	}

	out, err := execCommand("session", "export", "sess-x")
	if err == nil {
		t.Fatalf("[session-export-err] expected an error, got output: %s", out)
	}
	if !strings.Contains(err.Error(), "access_denied") && !strings.Contains(err.Error(), "not your session") {
		t.Errorf("[session-export-err] error = %v, want the access_denied envelope", err)
	}
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
