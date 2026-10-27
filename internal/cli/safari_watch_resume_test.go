package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/flashcatcloud/go-flashduty"
)

func TestWatchEventsCarryCreatedAt(t *testing.T) {
	w := newWatchState("", 0)
	for _, f := range []*flashduty.SessionEventFrame{
		replyFrame("e1", "i", "r", "result", "hi", 1700000000123),
		settleFrame("e2", "r", 1700000000456, 0),
		{EventID: "e3", CreatedAt: 1700000000789, Type: "assistant", IsError: true, Payload: json.RawMessage(`{"parts":[{"text":"boom"}]}`)},
	} {
		out := w.handle(f)
		if len(out.Events) != 1 || out.Events[0].CreatedAt != f.CreatedAt {
			t.Errorf("frame %s: events = %+v, want created_at %d", f.EventID, out.Events, f.CreatedAt)
			continue
		}
		b, _ := json.Marshal(out.Events[0])
		if !strings.Contains(string(b), fmt.Sprintf(`"created_at":%d`, f.CreatedAt)) {
			t.Errorf("serialized event lacks created_at: %s", b)
		}
	}
}

func TestWatchSinceEventIDRequiresSince(t *testing.T) {
	saveAndResetGlobals(t)
	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("k", flashduty.WithBaseURL("http://127.0.0.1:1"))
	}
	_, err := execCommand("safari", "watch", "--session-id", "s1", "--root-invocation-id", "r1", "--since-event-id", "e1")
	if err == nil || !strings.Contains(err.Error(), "--since-event-id requires --since") {
		t.Fatalf("err = %v", err)
	}
}

func TestWatchSinceEventIDResumesStrictlyAfter(t *testing.T) {
	s := &watchServer{
		streams: []string{sse(
			// A server that only knows the millisecond replays the boundary frame.
			replyFrame("e1", "i", "r1", "result", "already seen", 500),
			replyFrame("e2", "i", "r1", "result", "new", 501),
			settleFrame("e3", "r1", 502, 0),
		)},
		running: []bool{false},
	}
	r, out, _ := newTestWatchRunner(t, s, "r1")
	r.state = newWatchState("r1", 500)
	r.state.cursorID = "e1"
	r.state.seen["e1"] = struct{}{}
	if err := r.run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(s.cursors[0], `"last_event_at":500`) || !strings.Contains(s.cursors[0], `"last_event_id":"e1"`) {
		t.Errorf("subscribe body = %s", s.cursors[0])
	}
	if strings.Contains(out.String(), "already seen") || !strings.Contains(out.String(), "new") {
		t.Errorf("stdout:\n%s", out.String())
	}
}
