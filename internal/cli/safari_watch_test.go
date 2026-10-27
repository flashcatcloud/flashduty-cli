package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flashcatcloud/go-flashduty"
)

func replyFrame(eventID, invocation, root, purpose, message string, createdAt int64) *flashduty.SessionEventFrame {
	payload := fmt.Sprintf(`{"parts":[{"function_response":{"name":"reply","response":{"submitted":true,"purpose":%q,"message":%q}}}]}`, purpose, message)
	return &flashduty.SessionEventFrame{
		EventID: eventID, CreatedAt: createdAt, Type: "assistant",
		InvocationID: invocation, RootInvocationID: root, Payload: json.RawMessage(payload),
	}
}

func settleFrame(eventID, root string, createdAt int64, standing int) *flashduty.SessionEventFrame {
	tasks := make([]string, standing)
	for i := range tasks {
		tasks[i] = fmt.Sprintf(`{"task_id":"t%d","kind":"process"}`, i)
	}
	payload := fmt.Sprintf(`{"root_invocation_id":%q,"reason":"completed","standing_tasks":[%s]}`, root, strings.Join(tasks, ","))
	return &flashduty.SessionEventFrame{
		EventID: eventID, CreatedAt: createdAt, Type: "session_turn_settled",
		RootInvocationID: root, Payload: json.RawMessage(payload),
	}
}

func TestWatchStateReplyMapping(t *testing.T) {
	w := newWatchState("", 0)

	out := w.handle(replyFrame("e1", "inv1", "root1", "progress", " looking at logs ", 10))
	if len(out.Events) != 1 {
		t.Fatalf("events = %+v", out.Events)
	}
	ev := out.Events[0]
	if ev.Type != "reply" || ev.Purpose != "progress" || ev.Message != "looking at logs" || ev.InvocationID != "inv1" || ev.EventID != "e1" {
		t.Errorf("event = %+v", ev)
	}

	// An empty purpose is a result (frames written before purpose existed).
	out = w.handle(replyFrame("e2", "inv1", "root1", "", "done", 11))
	if len(out.Events) != 1 || out.Events[0].Purpose != "result" {
		t.Errorf("empty purpose events = %+v", out.Events)
	}

	// The legacy receipt flag still counts.
	legacy := &flashduty.SessionEventFrame{EventID: "e3", Type: "assistant", Payload: json.RawMessage(
		`{"parts":[{"function_response":{"name":"reply","response":{"delivered":true,"message":"old"}}}]}`)}
	if out = w.handle(legacy); len(out.Events) != 1 || out.Events[0].Message != "old" {
		t.Errorf("legacy events = %+v", out.Events)
	}

	// Two replies in one frame are both emitted, in order.
	multi := &flashduty.SessionEventFrame{EventID: "e4", Type: "assistant", Payload: json.RawMessage(
		`{"parts":[{"function_response":{"name":"reply","response":{"submitted":true,"message":"a"}}},{"function_response":{"name":"reply","response":{"submitted":true,"purpose":"progress","message":"b"}}}]}`)}
	if out = w.handle(multi); len(out.Events) != 2 || out.Events[0].Message != "a" || out.Events[1].Purpose != "progress" {
		t.Errorf("multi events = %+v", out.Events)
	}
}

func TestWatchStateDropsEverythingButReplies(t *testing.T) {
	cases := map[string]string{
		"assistant prose":        `{"parts":[{"text":"private working text"}]}`,
		"tool call":              `{"parts":[{"function_call":{"name":"reply","args":{"message":"x"}}}]}`,
		"other tool result":      `{"parts":[{"function_response":{"name":"shell","response":{"submitted":true,"message":"x"}}}]}`,
		"refused reply":          `{"parts":[{"function_response":{"name":"reply","response":{"submitted":false,"message":"x"}}}]}`,
		"unknown purpose":        `{"parts":[{"function_response":{"name":"reply","response":{"submitted":true,"purpose":"weird","message":"x"}}}]}`,
		"empty message":          `{"parts":[{"function_response":{"name":"reply","response":{"submitted":true,"message":"  "}}}]}`,
		"thinking text":          `{"parts":[{"text":"hmm","thought":true}]}`,
		"malformed payload":      `not json`,
		"ephemeral token chunk":  `{"parts":[{"text":"par"}]}`,
		"user echo of own query": `{"parts":[{"text":"question"}]}`,
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			w := newWatchState("", 0)
			frame := &flashduty.SessionEventFrame{EventID: "e", Type: "assistant", Payload: json.RawMessage(payload)}
			if name == "ephemeral token chunk" {
				frame = &flashduty.SessionEventFrame{Type: "assistant_token", Payload: json.RawMessage(payload)}
			}
			if name == "user echo of own query" {
				frame = &flashduty.SessionEventFrame{EventID: "e", Type: "user", Payload: json.RawMessage(payload)}
			}
			if out := w.handle(frame); len(out.Events) != 0 || out.Settled || out.Fatal {
				t.Errorf("outcome = %+v, want nothing", out)
			}
		})
	}
}

func TestWatchStateDedupeAndCursor(t *testing.T) {
	w := newWatchState("", 5)
	f := replyFrame("e1", "i", "r", "result", "once", 100)
	if len(w.handle(f).Events) != 1 {
		t.Fatal("first delivery must emit")
	}
	if len(w.handle(f).Events) != 0 {
		t.Error("a replayed event id must not emit twice")
	}
	if w.cursorAt != 100 || w.cursorID != "e1" {
		t.Errorf("cursor = %d/%s", w.cursorAt, w.cursorID)
	}
	// The cursor is the last persisted frame received, in the server's order,
	// not a client-side maximum: an id that sorts lower bytewise still wins
	// when it arrives later.
	w.handle(replyFrame("B1", "i", "r", "result", "same ms, collation-later", 100))
	if w.cursorAt != 100 || w.cursorID != "B1" {
		t.Errorf("cursor = %d/%s, want 100/B1", w.cursorAt, w.cursorID)
	}
}

func TestWatchStateRootFilter(t *testing.T) {
	w := newWatchState("rootA", 0)
	if out := w.handle(replyFrame("e1", "invB", "rootB", "result", "other", 1)); len(out.Events) != 0 {
		t.Errorf("other root emitted: %+v", out.Events)
	}
	// Filtered frames still advance the resume cursor.
	if w.cursorID != "e1" {
		t.Errorf("cursor id = %q", w.cursorID)
	}
	if out := w.handle(replyFrame("e2", "invA", "rootA", "result", "mine", 2)); len(out.Events) != 1 {
		t.Errorf("own root dropped: %+v", out.Events)
	}
	// A frame without a root id belongs to the root whose own invocation it is.
	bare := replyFrame("e3", "rootA", "", "result", "bare", 3)
	if out := w.handle(bare); len(out.Events) != 1 {
		t.Errorf("bare invocation frame dropped: %+v", out.Events)
	}
	if out := w.handle(settleFrame("e4", "rootB", 4, 0)); len(out.Events) != 0 || out.Settled {
		t.Errorf("other root's settle leaked: %+v", out)
	}
}

func TestWatchStateSettle(t *testing.T) {
	w := newWatchState("r", 0)
	out := w.handle(settleFrame("s1", "r", 1, 0))
	if !out.Settled || len(out.Events) != 1 || out.Events[0].Type != "settled" || out.Events[0].Reason != "completed" || out.Events[0].RootInvocationID != "r" {
		t.Errorf("settle outcome = %+v", out)
	}
	// Standing tasks keep the root open: reported, but not a done candidate.
	out = w.handle(settleFrame("s2", "r", 2, 2))
	if out.Settled || len(out.Events) != 1 || out.Events[0].StandingTasks != 2 {
		t.Errorf("standing settle outcome = %+v", out)
	}
}

func TestWatchStateErrors(t *testing.T) {
	w := newWatchState("", 0)
	out := w.handle(&flashduty.SessionEventFrame{Type: "environment.failed", Payload: json.RawMessage(`{"error_message":"no sandbox capacity"}`)})
	if !out.Fatal || len(out.Events) != 1 || out.Events[0].Type != "error" || out.Events[0].Message != "no sandbox capacity" {
		t.Errorf("env failed outcome = %+v", out)
	}
	out = w.handle(&flashduty.SessionEventFrame{Type: "error", Payload: json.RawMessage(`{"message":"model unavailable"}`)})
	if out.Fatal || len(out.Events) != 1 || out.Events[0].Message != "model unavailable" {
		t.Errorf("error frame outcome = %+v", out)
	}
	out = w.handle(&flashduty.SessionEventFrame{EventID: "e9", Type: "assistant", IsError: true, ErrorCode: "llm_failed", Payload: json.RawMessage(`{"parts":[{"text":"provider error"}]}`)})
	if len(out.Events) != 1 || out.Events[0].Message != "provider error" || out.Events[0].Code != "llm_failed" {
		t.Errorf("is_error outcome = %+v", out)
	}
}

// watchServer scripts /subscribe (one SSE body per connection) and /get.
type watchServer struct {
	t         *testing.T
	streams   []string
	conns     atomic.Int32
	cursors   []string
	mu        sync.Mutex
	running   []bool // is_running answers, consumed in order; the last repeats
	getCalls  atomic.Int32
	streamErr int // when >0, the first N subscribe calls answer HTTP 500
}

func (s *watchServer) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	switch r.URL.Path {
	case "/safari/session/subscribe":
		n := int(s.conns.Add(1)) - 1
		s.mu.Lock()
		s.cursors = append(s.cursors, string(body))
		s.mu.Unlock()
		if n < s.streamErr {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error":{"code":"InternalError","message":"boom"}}`)
			return
		}
		i := n - s.streamErr
		if i >= len(s.streams) {
			i = len(s.streams) - 1
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, s.streams[i])
	case "/safari/session/get":
		k := int(s.getCalls.Add(1)) - 1
		s.mu.Lock()
		if k >= len(s.running) {
			k = len(s.running) - 1
		}
		running := s.running[k]
		s.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "r", "data": map[string]any{
			"events": []any{}, "pending_messages": []any{}, "session": map[string]any{"session_id": "s1", "is_running": running},
		}})
	default:
		s.t.Errorf("unexpected path %s", r.URL.Path)
	}
}

func sse(frames ...*flashduty.SessionEventFrame) string {
	var b strings.Builder
	b.WriteString(": ping\n\n")
	for _, f := range frames {
		raw, _ := json.Marshal(f)
		fmt.Fprintf(&b, "data: %s\n\n", raw)
	}
	return b.String()
}

func newTestWatchRunner(t *testing.T, s *watchServer, root string) (*watchRunner, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	s.t = t
	srv := httptest.NewServer(http.HandlerFunc(s.handler))
	t.Cleanup(srv.Close)
	c, err := flashduty.NewClient("k", flashduty.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	var out, errw bytes.Buffer
	r := newWatchRunner(c, "s1", root, 0, &out, &errw)
	r.backoff = func(int) time.Duration { return time.Millisecond }
	r.confirmGap = time.Millisecond
	return r, &out, &errw
}

func outputLines(b *bytes.Buffer) []string {
	s := strings.TrimSpace(b.String())
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func TestWatchRunExitsWhenSettledAndIdle(t *testing.T) {
	s := &watchServer{
		streams: []string{sse(
			replyFrame("e1", "i1", "r1", "progress", "checking", 1),
			replyFrame("e2", "i1", "r1", "result", "root cause: disk full", 2),
			settleFrame("e3", "r1", 3, 0),
		)},
		running: []bool{false},
	}
	r, out, _ := newTestWatchRunner(t, s, "r1")
	if err := r.run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	lines := outputLines(out)
	if len(lines) != 3 {
		t.Fatalf("stdout lines = %q", lines)
	}
	for _, l := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Errorf("stdout line is not a JSON object: %q", l)
		}
	}
	if !strings.Contains(lines[1], `"purpose":"result"`) || !strings.Contains(lines[2], `"type":"settled"`) {
		t.Errorf("lines = %q", lines)
	}
}

func TestWatchRunKeepsListeningWhileSessionRunning(t *testing.T) {
	// The first settle is followed by a still-running session (a second round is
	// starting); the watcher must keep listening and finish after round two.
	s := &watchServer{
		streams: []string{
			sse(replyFrame("e1", "i1", "r1", "result", "round one", 1), settleFrame("e2", "r1", 2, 0)),
			sse(replyFrame("e1", "i1", "r1", "result", "round one", 1), replyFrame("e3", "i2", "r1", "result", "round two", 3), settleFrame("e4", "r1", 4, 0)),
		},
		running: []bool{true, true, false},
	}
	r, out, _ := newTestWatchRunner(t, s, "r1")
	r.confirmRetries = 2
	if err := r.run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	text := out.String()
	if strings.Count(text, "round one") != 1 || strings.Count(text, "round two") != 1 {
		t.Errorf("replies not delivered exactly once:\n%s", text)
	}
	if strings.Count(text, `"type":"settled"`) != 2 {
		t.Errorf("want two settled events:\n%s", text)
	}
}

func TestWatchRunStandingTasksKeepWatchOpen(t *testing.T) {
	s := &watchServer{
		streams: []string{
			sse(settleFrame("e1", "r1", 1, 1)),
			sse(replyFrame("e2", "i2", "r1", "result", "task finished", 2), settleFrame("e3", "r1", 3, 0)),
		},
		running: []bool{false},
	}
	r, out, _ := newTestWatchRunner(t, s, "r1")
	if err := r.run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if s.conns.Load() != 2 {
		t.Errorf("connections = %d, want 2 (a settle with standing tasks must not end the watch)", s.conns.Load())
	}
	if !strings.Contains(out.String(), "task finished") {
		t.Errorf("stdout:\n%s", out.String())
	}
}

func TestWatchRunResumesFromCursorAfterDrop(t *testing.T) {
	// The stream drops after one reply without [DONE]; the reconnect must carry
	// the cursor, and the replayed frame must not print twice.
	s := &watchServer{
		streams: []string{
			sse(replyFrame("e1", "i1", "r1", "progress", "step one", 7)),
			sse(replyFrame("e1", "i1", "r1", "progress", "step one", 7), settleFrame("e2", "r1", 8, 0)),
		},
		running: []bool{false},
	}
	r, out, errw := newTestWatchRunner(t, s, "r1")
	if err := r.run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if strings.Count(out.String(), "step one") != 1 {
		t.Errorf("duplicate reply:\n%s", out.String())
	}
	if len(s.cursors) != 2 || !strings.Contains(s.cursors[1], `"last_event_at":7`) || !strings.Contains(s.cursors[1], `"last_event_id":"e1"`) {
		t.Errorf("subscribe bodies = %q", s.cursors)
	}
	if !strings.Contains(errw.String(), "resuming") {
		t.Errorf("diagnostics must go to stderr, got %q", errw.String())
	}
	if strings.Contains(out.String(), "resuming") {
		t.Errorf("diagnostics leaked to stdout:\n%s", out.String())
	}
}

func TestWatchRunRetriesTransientHTTPErrors(t *testing.T) {
	s := &watchServer{
		streamErr: 2,
		streams:   []string{sse(settleFrame("e1", "r1", 1, 0))},
		running:   []bool{false},
	}
	r, _, _ := newTestWatchRunner(t, s, "r1")
	if err := r.run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if s.conns.Load() != 3 {
		t.Errorf("connections = %d, want 3", s.conns.Load())
	}
}

func TestWatchRunGivesUpAfterRepeatedFailures(t *testing.T) {
	s := &watchServer{streamErr: 100, streams: []string{""}, running: []bool{false}}
	r, _, _ := newTestWatchRunner(t, s, "r1")
	r.maxFailures = 2
	if err := r.run(context.Background()); err == nil || !strings.Contains(err.Error(), "giving up") {
		t.Fatalf("err = %v", err)
	}
}

func TestWatchRunFailsFastOnClientErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"error":{"code":"AccessDenied","message":"session not accessible"}}`)
	}))
	t.Cleanup(srv.Close)
	c, _ := flashduty.NewClient("k", flashduty.WithBaseURL(srv.URL))
	var out, errw bytes.Buffer
	r := newWatchRunner(c, "s1", "", 0, &out, &errw)
	r.backoff = func(int) time.Duration { return time.Millisecond }
	err := r.run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not accessible") {
		t.Fatalf("err = %v", err)
	}
}

func TestWatchRunEnvironmentFailureIsFatal(t *testing.T) {
	s := &watchServer{
		streams: []string{sse(&flashduty.SessionEventFrame{Type: "environment.failed", RootInvocationID: "r1", Payload: json.RawMessage(`{"error_message":"no capacity"}`)})},
		running: []bool{false},
	}
	r, out, _ := newTestWatchRunner(t, s, "r1")
	if err := r.run(context.Background()); err == nil {
		t.Fatal("expected a non-nil error")
	}
	if !strings.Contains(out.String(), `"type":"error"`) || !strings.Contains(out.String(), "no capacity") {
		t.Errorf("stdout:\n%s", out.String())
	}
}

func TestWatchRunStopsOnContextCancel(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, ": ping\n\n")
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(func() { close(release); srv.Close() })
	c, _ := flashduty.NewClient("k", flashduty.WithBaseURL(srv.URL))
	var out, errw bytes.Buffer
	r := newWatchRunner(c, "s1", "r1", 0, &out, &errw)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.run(ctx) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Error("expected a cancellation error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run did not return after cancel")
	}
	if out.Len() != 0 {
		t.Errorf("stdout must stay empty, got %q", out.String())
	}
}

func TestWatchRunQuietCleanClosesAreNotFailures(t *testing.T) {
	// Every stream closes cleanly with no frames (a quiet stretch). With a zero
	// healthy threshold each counts as healthy, so more reconnects than
	// maxFailures still reach the settle that follows.
	quiet := sse()
	streams := make([]string, 0, 6)
	for i := 0; i < 5; i++ {
		streams = append(streams, quiet)
	}
	streams = append(streams, sse(settleFrame("e1", "r1", 1, 0)))
	s := &watchServer{streams: streams, running: []bool{false}}
	r, _, _ := newTestWatchRunner(t, s, "r1")
	r.maxFailures = 2
	r.healthyAfter = 0
	if err := r.run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if s.conns.Load() != 6 {
		t.Errorf("connections = %d, want 6", s.conns.Load())
	}
}

func TestWatchRunQuickCleanClosesStillCountAsFailures(t *testing.T) {
	s := &watchServer{streams: []string{sse()}, running: []bool{false}}
	r, _, _ := newTestWatchRunner(t, s, "r1")
	r.maxFailures = 2
	if err := r.run(context.Background()); err == nil || !strings.Contains(err.Error(), "giving up") {
		t.Fatalf("err = %v", err)
	}
}
