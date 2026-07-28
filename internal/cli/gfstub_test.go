package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
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

	// mu guards every field below. Verified commands (e.g. incident comment's
	// write-back check) now issue concurrent requests against a single stub
	// server, so the handler below runs on more than one goroutine at once;
	// without this lock the plain field writes here would race. It is
	// deliberately released before invoking dataFor/dataForPath/data (see the
	// handler below) so those calls run concurrently rather than being
	// serialized by this lock — a test whose own closure touches shared state
	// is responsible for synchronizing that state itself.
	mu sync.Mutex

	// lastPath is the path of the most recent request (no query string). When
	// a test fans out concurrent requests, "most recent" is whichever
	// goroutine happened to reach this handler last, not any particular
	// logical request — such a test should assert against bodies, or a
	// dataForPath closure that inspects each request as it arrives, instead.
	lastPath string
	// lastBody is the decoded JSON body of the most recent request. Same
	// concurrency caveat as lastPath.
	lastBody map[string]any
	// lastAuthorization is the Authorization header of the most recent
	// request. Same concurrency caveat as lastPath.
	lastAuthorization string
	// bodies records the decoded body of every request, in the order each
	// reached this handler.
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
		var body map[string]any
		if raw, err := io.ReadAll(r.Body); err == nil && len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		path := r.URL.Path
		authorization := r.Header.Get("Authorization")

		s.mu.Lock()
		s.requests++
		s.lastPath = path
		s.lastAuthorization = authorization
		s.lastBody = body
		s.bodies = append(s.bodies, body)
		dataForPath, dataFor, data := s.dataForPath, s.dataFor, s.data
		s.mu.Unlock()

		// dataForPath/dataFor/data run outside the lock: some tests (e.g. ones
		// exercising verifyIncidentCommentsWritten's concurrent fan-out) rely on
		// being able to observe genuine overlap between in-flight requests, which
		// holding s.mu across the call would silently serialize away. A test
		// closure that itself touches shared state is responsible for its own
		// synchronization, same as any other concurrently-invoked callback.
		var payload any
		switch {
		case dataForPath != nil:
			payload = dataForPath(path, body)
		case dataFor != nil:
			payload = dataFor(body)
		case data != nil:
			payload = data
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
	return stringsField(s.lastBody, key)
}

// stringsField reads a string-slice field from a decoded JSON body map. Tests
// that need to inspect a specific historical request (via gfStub.bodies)
// rather than the "most recent" convenience fields call this directly.
func stringsField(body map[string]any, key string) []string {
	raw, ok := body[key].([]any)
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
