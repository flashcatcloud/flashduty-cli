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
// ai-sre, --limit to 200, and --format to jsonl. These defaults are what makes
// `session list` pipe cleanly into the /insight skill without extra flags.
func TestSessionListFlags(t *testing.T) {
	cmd := newSessionListCmd()
	flags := cmd.Flags()

	cases := []struct {
		name string
		def  string
	}{
		{"app", "ai-sre"},
		{"limit", "200"},
		{"format", sessionFormatJSONL},
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

// TestCommandSessionListRejectsBadFormat fails fast on an unknown --format.
func TestCommandSessionListRejectsBadFormat(t *testing.T) {
	saveAndResetGlobals(t)
	newGFStub(t)

	_, err := execCommand("session", "list", "--format", "yaml")
	if err == nil || !strings.Contains(err.Error(), "invalid --format") {
		t.Fatalf("expected an invalid --format error, got %v", err)
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
