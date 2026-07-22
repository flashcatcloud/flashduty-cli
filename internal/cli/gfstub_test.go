package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flashcatcloud/go-flashduty"
)

// gfStub is an httptest-backed stand-in for the go-flashduty API. Migrated
// commands build a *flashduty.Client (a concrete type, not an interface), so
// they can't be mocked the way the legacy flashdutyClient interface is — they
// are exercised against this stub server instead. The stub records every
// request's path and decoded JSON body and replies with a canned envelope, so a
// test can assert exactly what payload a command sent.
type gfStub struct {
	server *httptest.Server

	// lastPath is the path of the most recent request (no query string).
	lastPath string
	// lastBody is the decoded JSON body of the most recent request.
	lastBody map[string]any
	// lastAuthorization is the Authorization header of the most recent request.
	lastAuthorization string
	// bodies records the decoded body of every request, in order.
	bodies []map[string]any
	// requests counts how many requests reached the stub.
	requests int

	// data is the JSON object placed under the envelope "data" key. When nil an
	// empty object is returned, which is enough for mutations that only consume
	// the envelope.
	data any

	// dataFor, when set, computes the envelope "data" payload per request from
	// the decoded body. It takes precedence over data and lets a test return a
	// different page on each call (e.g. cursor pagination).
	dataFor func(body map[string]any) any

	// dataForPath, when set, computes the envelope "data" payload from the
	// request path and decoded body. It takes precedence over dataFor and data,
	// and lets a test serve multiple endpoints in one flow (e.g. war-room create,
	// which first lists war-room-enabled integrations and then creates the room).
	dataForPath func(path string, body map[string]any) any
}

// newGFStub starts a stub server and wires newClientFn to a client pointed at
// it. It returns the stub so tests can inspect the captured request. The server
// is torn down via t.Cleanup.
func newGFStub(t *testing.T) *gfStub {
	t.Helper()
	s := &gfStub{}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests++
		s.lastPath = r.URL.Path
		s.lastAuthorization = r.Header.Get("Authorization")
		s.lastBody = nil
		if body, err := io.ReadAll(r.Body); err == nil && len(body) > 0 {
			_ = json.Unmarshal(body, &s.lastBody)
		}
		s.bodies = append(s.bodies, s.lastBody)

		var payload any
		switch {
		case s.dataForPath != nil:
			payload = s.dataForPath(s.lastPath, s.lastBody)
		case s.dataFor != nil:
			payload = s.dataFor(s.lastBody)
		case s.data != nil:
			payload = s.data
		default:
			payload = map[string]any{}
		}
		resp := map[string]any{
			"request_id": "test-request-id",
			"error":      map[string]any{"code": "OK", "message": ""},
			"data":       payload,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(s.server.Close)

	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test-key", flashduty.WithBaseURL(s.server.URL))
	}
	return s
}

// bodyStrings reads a string-slice field from the last decoded request body.
func (s *gfStub) bodyStrings(key string) []string {
	raw, ok := s.lastBody[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if str, ok := v.(string); ok {
			out = append(out, str)
		}
	}
	return out
}
