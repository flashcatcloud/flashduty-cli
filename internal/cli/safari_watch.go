package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"
)

// Frame types on the session event stream that matter to a watcher.
const (
	frameTypeSettled     = "session_turn_settled"
	frameTypeEnvFailed   = "environment.failed"
	frameTypeError       = "error"
	replyToolName        = "reply"
	replyPurposeResult   = "result"
	replyPurposeProgress = "progress"
	// settledReasonSteered settles a message the running turn took in.
	settledReasonSteered = "steered"
)

// watchEvent is the one user-facing line `safari watch` prints. Only the fields
// of its Type are set.
type watchEvent struct {
	Type             string `json:"type"`
	Purpose          string `json:"purpose,omitempty"`
	Message          string `json:"message,omitempty"`
	InvocationID     string `json:"invocation_id,omitempty"`
	RootInvocationID string `json:"root_invocation_id,omitempty"`
	EventID          string `json:"event_id,omitempty"`
	CreatedAt        int64  `json:"created_at,omitempty"`
	Reason           string `json:"reason,omitempty"`
	StandingTasks    int    `json:"standing_tasks,omitempty"`
	Code             string `json:"code,omitempty"`
}

// settledPayload is the payload of a session_turn_settled frame (the subset a
// watcher needs).
type settledPayload struct {
	RootInvocationID string            `json:"root_invocation_id"`
	Reason           string            `json:"reason"`
	StandingTasks    []json.RawMessage `json:"standing_tasks"`
	// FinalEventID names the turn's answer. A turn that is not reply-only
	// (one started from the web console, say) answers in that event's text
	// instead of through the reply tool; FinalKind set means it has nothing
	// to deliver.
	FinalEventID string `json:"final_event_id"`
	FinalKind    string `json:"final_kind"`
	DeliveryMode string `json:"delivery_mode"`
}

// frameOutcome is what one frame means for the watch loop.
type frameOutcome struct {
	Events []watchEvent
	// Settled is true for a settle with no standing tasks once the followed
	// request has settled: the request is then a candidate for "fully done"
	// (confirmed against is_running).
	Settled bool
	// Fatal is true for a frame after which the session will produce nothing
	// more (environment provisioning failed before the turn started).
	Fatal bool
}

// watchState turns raw stream frames into events. It owns the request
// tracking, the dedupe set and the resume cursor, none of which depend on the
// transport.
//
// The watch follows the session, not one invocation: a message sent while
// another turn runs may be taken into that turn, and its answer then comes
// from that turn's invocation. rootID (the request being followed) only marks
// where output starts — its first frame — and which settle makes the request
// a done candidate.
type watchState struct {
	rootID string
	// started is set once output begins: at the followed request's first
	// frame, or immediately when no request is followed.
	started bool
	// rootSettled is set once the followed request has settled (answered in a
	// turn of its own, or taken into the running turn).
	rootSettled bool
	seen        map[string]struct{}
	// answers holds, per root, the visible text of its latest persisted
	// assistant frame, so a turn that answers without the reply tool can still
	// be printed from its settle's final_event_id. A finished turn clears it.
	answers map[string]heldAnswer

	// When an earlier watch is re-armed with --root-invocation-id and --since,
	// the stream is replayed from the session start so the request's own
	// frames are seen again. Persisted frames up to quietAt (and, at exactly
	// quietAt, up to and including quietID) were already printed: they update
	// the state but print nothing.
	quietAt     int64
	quietID     string
	quietPassed bool
	// quietDone is set when the last persisted frame was a quiet done
	// candidate. It cannot end the stream — later frames may still follow in
	// the replay — so it is acted on only if the stream ends with nothing new.
	quietDone bool

	cursorAt int64
	cursorID string
}

type heldAnswer struct {
	eventID, invocation, text string
}

func newWatchState(rootID string, since int64) *watchState {
	return &watchState{
		rootID: rootID, started: rootID == "", rootSettled: rootID == "",
		seen: map[string]struct{}{}, answers: map[string]heldAnswer{}, cursorAt: since,
	}
}

// quiet reports whether a persisted frame was already printed by an earlier
// watch. Live-only frames (no event id) are never quiet.
func (w *watchState) quiet(f *flashduty.SessionEventFrame) bool {
	switch {
	case f.EventID == "" || w.quietAt == 0 || f.CreatedAt > w.quietAt:
		return false
	case f.CreatedAt < w.quietAt:
		return true
	case w.quietID == "":
		// --since alone resumes at that millisecond inclusive.
		return false
	case w.quietPassed:
		return false
	}
	if f.EventID == w.quietID {
		w.quietPassed = true
	}
	return true
}

// frameRoot is the root invocation a frame belongs to. A frame that carries no
// root id belongs to the root whose own invocation id it carries.
func frameRoot(f *flashduty.SessionEventFrame) string {
	if f.RootInvocationID != "" {
		return f.RootInvocationID
	}
	return f.InvocationID
}

// handle processes one frame. Persisted frames always advance the resume
// cursor, even when filtered out, so a reconnect never replays them.
func (w *watchState) handle(f *flashduty.SessionEventFrame) frameOutcome {
	quiet := w.quiet(f)
	out := w.classify(f)
	if quiet {
		w.quietDone = out.Settled
		out.Events, out.Settled = nil, false
	} else if f.EventID != "" {
		w.quietDone = false
	}
	for i := range out.Events {
		out.Events[i].CreatedAt = f.CreatedAt
	}
	return out
}

func (w *watchState) classify(f *flashduty.SessionEventFrame) frameOutcome {
	if f.EventID != "" {
		// The server streams persisted frames in its own cursor order (the
		// database collation of (created_at, event_id)), so the last one
		// received is the cursor; comparing ids here would use a different
		// order.
		w.cursorAt, w.cursorID = f.CreatedAt, f.EventID
		if _, dup := w.seen[f.EventID]; dup {
			return frameOutcome{}
		}
		w.seen[f.EventID] = struct{}{}
	}
	if !w.started {
		if frameRoot(f) != w.rootID {
			return frameOutcome{}
		}
		w.started = true
	}

	switch f.Type {
	case frameTypeSettled:
		var p settledPayload
		_ = json.Unmarshal(f.Payload, &p)
		root := p.RootInvocationID
		if root == "" {
			root = frameRoot(f)
		}
		var events []watchEvent
		if a := w.answers[root]; a.eventID != "" && a.eventID == p.FinalEventID && p.DeliveryMode != safariReplyOnlyMode && p.FinalKind == "" {
			// Carries the settle's own event id, so the line's
			// created_at/event_id pair is one real stream position to resume
			// after.
			events = append(events, watchEvent{Type: "reply", Purpose: replyPurposeResult, Message: a.text, InvocationID: a.invocation, EventID: f.EventID})
		}
		if root == w.rootID {
			w.rootSettled = true
		}
		events = append(events, watchEvent{Type: "settled", RootInvocationID: root, Reason: p.Reason, EventID: f.EventID, StandingTasks: len(p.StandingTasks)})
		// A steered settle is written while the absorbing turn still runs and
		// is about to answer, so it is never the session's last frame.
		steered := p.Reason == settledReasonSteered
		if !steered {
			// The session runs one turn at a time: a finished turn leaves no
			// other answer pending.
			clear(w.answers)
		}
		done := w.rootSettled && len(p.StandingTasks) == 0 && !steered
		return frameOutcome{Events: events, Settled: done}
	case frameTypeEnvFailed:
		var p struct {
			ErrorMessage string `json:"error_message"`
		}
		_ = json.Unmarshal(f.Payload, &p)
		msg := strings.TrimSpace(p.ErrorMessage)
		if msg == "" {
			msg = "environment provisioning failed"
		}
		return frameOutcome{Events: []watchEvent{{Type: "error", Message: msg, Code: "environment_failed"}}, Fatal: true}
	case frameTypeError:
		return frameOutcome{Events: []watchEvent{{Type: "error", Message: errorFrameMessage(f), Code: f.ErrorCode}}}
	case "assistant", "model":
		if replies := repliesFromFrame(f); len(replies) > 0 {
			return frameOutcome{Events: replies}
		}
		if f.IsError {
			return frameOutcome{Events: []watchEvent{{Type: "error", Message: errorFrameMessage(f), Code: f.ErrorCode, EventID: f.EventID}}}
		}
		if text := visibleText(f); text != "" && f.EventID != "" {
			w.answers[frameRoot(f)] = heldAnswer{eventID: f.EventID, invocation: f.InvocationID, text: text}
		}
	}
	return frameOutcome{}
}

// visibleText is a frame's answer text: its text parts without the model's
// thinking.
func visibleText(f *flashduty.SessionEventFrame) string {
	var c adkContent
	if len(f.Payload) == 0 || json.Unmarshal(f.Payload, &c) != nil {
		return ""
	}
	var texts []string
	for _, p := range c.Parts {
		if t := strings.TrimSpace(p.Text); t != "" && !p.Thought {
			texts = append(texts, t)
		}
	}
	return strings.Join(texts, "\n")
}

// adkContent is the slice of an ADK content payload the reply tool result
// and the answer text live in.
type adkContent struct {
	Parts []struct {
		Text             string `json:"text"`
		Thought          bool   `json:"thought"`
		FunctionResponse *struct {
			Name     string `json:"name"`
			Response struct {
				Submitted bool   `json:"submitted"`
				Delivered bool   `json:"delivered"`
				Message   string `json:"message"`
				Purpose   string `json:"purpose"`
			} `json:"response"`
		} `json:"function_response"`
	} `json:"parts"`
}

// repliesFromFrame extracts the messages the agent handed over through the
// reply tool. Under the reply-only contract these tool results are the only
// user-facing output; assistant prose, thinking and other tool calls are
// private working text and are never returned. A result counts only when the
// tool accepted it (submitted, or the legacy delivered flag) and the purpose is
// one the protocol knows: an unknown purpose fails closed.
func repliesFromFrame(f *flashduty.SessionEventFrame) []watchEvent {
	if len(f.Payload) == 0 {
		return nil
	}
	var c adkContent
	if err := json.Unmarshal(f.Payload, &c); err != nil {
		return nil
	}
	var out []watchEvent
	for _, part := range c.Parts {
		r := part.FunctionResponse
		if r == nil || strings.TrimSpace(r.Name) != replyToolName {
			continue
		}
		if !r.Response.Submitted && !r.Response.Delivered {
			continue
		}
		purpose := strings.TrimSpace(r.Response.Purpose)
		switch purpose {
		case "":
			purpose = replyPurposeResult
		case replyPurposeResult, replyPurposeProgress:
		default:
			continue
		}
		msg := strings.TrimSpace(r.Response.Message)
		if msg == "" {
			continue
		}
		out = append(out, watchEvent{Type: "reply", Purpose: purpose, Message: msg, InvocationID: f.InvocationID, EventID: f.EventID})
	}
	return out
}

// errorFrameMessage pulls a human-readable message out of an error frame:
// a message-like JSON field, else the text parts, else the error code.
func errorFrameMessage(f *flashduty.SessionEventFrame) string {
	var obj map[string]any
	if json.Unmarshal(f.Payload, &obj) == nil {
		for _, k := range []string{"message", "error", "detail", "text"} {
			if s, ok := obj[k].(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	var c adkContent
	if json.Unmarshal(f.Payload, &c) == nil {
		var texts []string
		for _, p := range c.Parts {
			if t := strings.TrimSpace(p.Text); t != "" {
				texts = append(texts, t)
			}
		}
		if len(texts) > 0 {
			return strings.Join(texts, "\n")
		}
	}
	if f.ErrorCode != "" {
		return f.ErrorCode
	}
	return "session error"
}

// watchRunner drives one watch: subscribe, map frames to events, reconnect on
// stream loss, and decide when the request is fully done.
type watchRunner struct {
	client *flashduty.Client
	state  *watchState
	sessID string
	out    io.Writer // stdout: events only
	errw   io.Writer // stderr: diagnostics

	maxFailures    int
	backoff        func(attempt int) time.Duration
	confirmRetries int
	confirmGap     time.Duration
	// healthyAfter is how long a stream must stay open before its clean close
	// counts as healthy. The server closes a subscription after minutes without
	// a frame; heartbeats are not frames, so a quiet stretch must not count
	// toward maxFailures.
	healthyAfter time.Duration
}

func newWatchRunner(client *flashduty.Client, sessionID, rootID string, since int64, out, errw io.Writer) *watchRunner {
	return &watchRunner{
		client: client, state: newWatchState(rootID, since), sessID: sessionID, out: out, errw: errw,
		maxFailures:    8,
		backoff:        func(attempt int) time.Duration { return time.Second << min(attempt, 5) },
		confirmRetries: 5,
		confirmGap:     2 * time.Second,
		healthyAfter:   30 * time.Second,
	}
}

func (r *watchRunner) emit(ev watchEvent) error {
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(r.out, "%s\n", b)
	return err
}

// errFatalFrame ends the watch after an event that closes the session's output.
var errFatalFrame = errors.New("session produced a terminal error")

// run blocks until the request is fully done (nil), an unrecoverable error, or
// ctx cancellation.
//
// Done rule: once the followed request has settled — in a turn of its own, or
// as "steered" when the running turn took it in — the next settle that is not
// steered and lists no standing tasks (its own, or the absorbing turn's) makes
// the watch ask POST /safari/session/get, and it exits only when is_running is
// false (no in-flight turn, so no queued follow-up turn or notification round
// is about to start). A settle that lists standing tasks keeps the watch open:
// those background tasks post later rounds into the session.
//
// Why such a settle is final: the server persists it only when the whole
// session has no open tasks and no queued notifications, and stamps
// standing_tasks with the running background shell and monitor tasks. With
// both empty nothing can start another round; a new message opens a new root.
// A steered settle is different — it is written while the absorbing turn still
// runs, before that turn answers — so it never ends the watch.
func (r *watchRunner) run(ctx context.Context) error {
	failures := 0
	settledSeen := false
	confirmTries := r.confirmRetries
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if settledSeen {
			done, err := r.confirmIdle(ctx, confirmTries)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
			// Still running: resume listening, and re-check once per later
			// reconnect instead of waiting out the full retry budget again.
			confirmTries = 1
		}

		started := time.Now()
		progressed, settled, err := r.streamOnce(ctx)
		if err == nil && r.state.quietDone {
			// A re-armed watch replayed up to a done candidate the earlier
			// watch already printed, and nothing new followed it.
			settled = true
		}
		if err == nil && time.Since(started) >= r.healthyAfter {
			progressed = true
		}
		if settled {
			settledSeen = true
			confirmTries = r.confirmRetries
		}
		if err != nil {
			if errors.Is(err, errFatalFrame) || ctx.Err() != nil || isPermanent(err) {
				return err
			}
		}
		if progressed {
			failures = 0
		} else {
			failures++
			if failures > r.maxFailures {
				if err == nil {
					err = errors.New("stream closed repeatedly without events")
				}
				return fmt.Errorf("giving up after %d failed attempts: %w", failures-1, err)
			}
		}
		if settled {
			// Go straight to the idle confirmation at the top of the loop.
			continue
		}
		reason := "stream closed"
		if err != nil {
			reason = err.Error()
		}
		_, _ = fmt.Fprintf(r.errw, "watch: %s; resuming from %d\n", reason, r.state.cursorAt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.backoff(failures)):
		}
	}
}

// streamOnce opens one subscription and processes frames until the stream ends.
// settled reports that a done-candidate settle frame was seen; progressed that
// at least one frame arrived.
func (r *watchRunner) streamOnce(ctx context.Context) (progressed, settled bool, err error) {
	rc, _, err := r.client.Sessions.Subscribe(ctx, &flashduty.SessionSubscribeRequest{
		SessionID:   r.sessID,
		LastEventAt: r.state.cursorAt,
		LastEventID: r.state.cursorID,
	})
	if err != nil {
		return false, false, err
	}
	defer func() { _ = rc.Close() }()

	fr := flashduty.NewSessionFrameReader(rc)
	for {
		frame, err := fr.Next()
		if err == io.EOF {
			return progressed, settled, nil
		}
		if err != nil {
			return progressed, settled, err
		}
		progressed = true
		out := r.state.handle(frame)
		for _, ev := range out.Events {
			if werr := r.emit(ev); werr != nil {
				return progressed, settled, fmt.Errorf("%w: %v", errFatalFrame, werr)
			}
		}
		if out.Fatal {
			return progressed, settled, errFatalFrame
		}
		if out.Settled {
			return progressed, true, nil
		}
	}
}

// confirmIdle reports whether the session has no in-flight turn. A running
// session right after a settle is usually a lagging in-flight marker, so it is
// re-checked a few times before the watch goes back to listening.
func (r *watchRunner) confirmIdle(ctx context.Context, tries int) (bool, error) {
	for i := 0; i < tries; i++ {
		resp, _, err := r.client.Sessions.ReadInfo(ctx, &flashduty.SessionGetRequest{SessionID: r.sessID, Limit: 1})
		if err != nil {
			if ctx.Err() != nil {
				return false, ctx.Err()
			}
			if isPermanent(err) {
				return false, err
			}
		} else if !resp.Session.IsRunning {
			return true, nil
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(r.confirmGap):
		}
	}
	return false, nil
}

// isPermanent reports whether retrying cannot help: a 4xx API answer other
// than rate limiting (bad session id, no access, bad credentials).
func isPermanent(err error) bool {
	var rl *flashduty.RateLimitError
	if errors.As(err, &rl) {
		return false
	}
	var apiErr *flashduty.ErrorResponse
	if errors.As(err, &apiErr) && apiErr.Response != nil {
		s := apiErr.Response.StatusCode
		return s >= 400 && s < 500 && s != http.StatusRequestTimeout
	}
	return false
}

func newSafariWatchCmd() *cobra.Command {
	var sessionID, rootID, sinceEventID string
	var since int64

	cmd := &cobra.Command{
		Use:   "watch --session-id <id> [--root-invocation-id <id>] [--since <unix-ms>]",
		Short: "Stream an AI SRE session's replies as one JSON line each",
		Long: `Follow an AI SRE session and print what the agent sends the user, one compact
JSON object per line on stdout (nothing else goes to stdout; diagnostics go to
stderr). Built to run under a tool that turns every stdout line into a
notification, such as Claude Code's Monitor.

Events:
  {"type":"reply","purpose":"progress|result","message":"...","invocation_id":"...","event_id":"...","created_at":1700000000000}
  {"type":"settled","root_invocation_id":"...","reason":"...","event_id":"...","created_at":1700000000000}
  {"type":"error","message":"...","created_at":1700000000000}

Printed: messages the agent sends with its reply tool, and the final answer of
a turn that does not use the reply tool (one started from the web console).
Tool calls, thinking and other assistant text are not. Every event carries
created_at (Unix ms of its frame). Events are de-duplicated by event id and the
stream is resumed from the last seen event after a drop.

The watch follows the whole session. A message sent while another turn is
running is taken into that turn ('safari send' asks for this), so its answer
comes from that turn, and replies to other people's messages in the same
session are printed too.

With --root-invocation-id (printed by 'safari send') output starts at that
request and the watch ends once it is done. Without it, --since (Unix
milliseconds) bounds the replay; the default is "now".

To re-arm a watch later, pass the last line's created_at as --since and its
event_id as --since-event-id: nothing up to and including that event is
printed again. --since alone resumes at that millisecond inclusive, so the
frame at exactly that time (and any sibling in the same millisecond) is
delivered again.

The command exits 0 once the request is fully done: it has settled (answered
in a turn of its own, or taken into the running turn), a turn has then settled
with no background tasks left, and the session reports no running turn. It
exits non-zero on an unrecoverable error.

API: POST /safari/session/subscribe, POST /safari/session/get`,
		Example: `  flashduty safari watch --session-id sess_01 --root-invocation-id inv_01`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if sinceEventID != "" && since == 0 {
					return fmt.Errorf("--since-event-id requires --since (the created_at of the same event)")
				}
				if since == 0 && rootID == "" {
					since = time.Now().UnixMilli()
				}
				r := newWatchRunner(ctx.Client, sessionID, rootID, since, ctx.Writer, ctx.Cmd.ErrOrStderr())
				switch {
				case rootID != "" && since != 0:
					// The request's own frames may predate the cursor; replay
					// from the session start, silently up to the cursor, so
					// its settle is still seen.
					r.state.cursorAt = 0
					r.state.quietAt, r.state.quietID = since, sinceEventID
				case sinceEventID != "":
					r.state.cursorID = sinceEventID
					r.state.seen[sinceEventID] = struct{}{}
				}
				err := r.run(cmdContext(ctx.Cmd))
				if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, errFatalFrame) {
					// Surface the failure on stdout as well: a notification
					// consumer reads nothing else.
					_ = r.emit(watchEvent{Type: "error", Message: err.Error()})
				}
				return err
			})
		},
	}
	cmd.Flags().StringVar(&sessionID, "session-id", "", "Session to follow (required)")
	cmd.Flags().StringVar(&rootID, "root-invocation-id", "", "Start at this request and end once it is done (the root_invocation_id printed by 'safari send')")
	cmd.Flags().Int64Var(&since, "since", 0, "Print only what comes after this Unix-millisecond timestamp (default: now; with --root-invocation-id the session is replayed from its start and everything up to this point is skipped silently)")
	cmd.Flags().StringVar(&sinceEventID, "since-event-id", "", "With --since: resume strictly after the event with this id (the event_id of the last line seen)")
	_ = cmd.MarkFlagRequired("session-id")
	return cmd
}
