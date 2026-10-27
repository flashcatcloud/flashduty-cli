package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/flashcatcloud/go-flashduty"
)

// sendServer records the calls `safari send` makes and answers each endpoint.
type sendServer struct {
	mu           sync.Mutex
	paths        []string
	message      map[string]any
	uploadSess   string
	uploadName   string
	uploadBody   string
	ackMode      string
	createCalled map[string]any
}

func newSendServer(t *testing.T, ackMode string) *sendServer {
	t.Helper()
	s := &sendServer{ackMode: ackMode}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.paths = append(s.paths, r.URL.Path)
		reply := func(data any) {
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "r", "data": data})
		}
		switch r.URL.Path {
		case "/safari/session/create":
			_ = json.NewDecoder(r.Body).Decode(&s.createCalled)
			reply(map[string]any{"session_id": "sess-new", "app_name": "ai-sre"})
		case "/safari/attachment/upload":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("multipart: %v", err)
			}
			s.uploadSess = r.FormValue("session_id")
			f, hdr, _ := r.FormFile("file")
			b, _ := io.ReadAll(f)
			s.uploadName, s.uploadBody = hdr.Filename, string(b)
			reply(map[string]any{"object_key": "attachments/a1/" + hdr.Filename, "mime": "text/plain", "filename": hdr.Filename, "size": len(b), "attachment_id": "a1"})
		case "/safari/session/message":
			_ = json.NewDecoder(r.Body).Decode(&s.message)
			sid, _ := s.message["session_id"].(string)
			if sid == "" {
				sid = "sess-auto"
			}
			w.WriteHeader(http.StatusAccepted)
			reply(map[string]any{"session_id": sid, "invocation_id": "inv-1", "root_invocation_id": "inv-1", "delivery_mode": s.ackMode})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test-key", flashduty.WithBaseURL(srv.URL))
	}
	return s
}

func TestSafariSendNewSessionAck(t *testing.T) {
	saveAndResetGlobals(t)
	s := newSendServer(t, "reply_only")

	out, err := execCommand("safari", "send", "why is latency up?", "--team-id", "42", "--env-kind", "byoc", "--env-id", "env-9")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	var ack map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &ack); err != nil {
		t.Fatalf("stdout is not one JSON object: %q", out)
	}
	if ack["session_id"] != "sess-auto" || ack["invocation_id"] != "inv-1" || ack["root_invocation_id"] != "inv-1" || ack["delivery_mode"] != "reply_only" {
		t.Errorf("ack = %v", ack)
	}
	if len(s.paths) != 1 || s.paths[0] != "/safari/session/message" {
		t.Errorf("calls = %v, want only the message call", s.paths)
	}
	m := s.message
	if m["query"] != "why is latency up?" || m["surface"] != "cli" || m["delivery_mode"] != "reply_only" ||
		m["app_name"] != "ai-sre" || m["entry_kind"] != "api" || m["team_id"] != float64(42) ||
		m["environment_kind"] != "byoc" || m["environment_id"] != "env-9" {
		t.Errorf("message body = %v", m)
	}
	if _, ok := m["session_id"]; ok {
		t.Errorf("new session must omit session_id: %v", m)
	}
}

func TestSafariSendExistingSessionOmitsNewSessionFields(t *testing.T) {
	saveAndResetGlobals(t)
	s := newSendServer(t, "reply_only")
	if _, err := execCommand("safari", "send", "--session-id", "sess-7", "follow up"); err != nil {
		t.Fatalf("send: %v", err)
	}
	m := s.message
	if m["session_id"] != "sess-7" {
		t.Errorf("session_id = %v", m["session_id"])
	}
	if _, ok := m["app_name"]; ok {
		t.Errorf("app_name must be omitted for an existing session: %v", m)
	}
	if _, ok := m["entry_kind"]; ok {
		t.Errorf("entry_kind must be omitted for an existing session: %v", m)
	}
}

func TestSafariSendFilesCreateSessionThenUploadThenMessage(t *testing.T) {
	saveAndResetGlobals(t)
	s := newSendServer(t, "reply_only")
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.txt")
	if err := os.WriteFile(path, []byte("stack trace"), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := execCommand("safari", "send", "--file", path, "--team-id", "5", "see attached")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if strings.Join(s.paths, ",") != "/safari/session/create,/safari/attachment/upload,/safari/session/message" {
		t.Errorf("call order = %v", s.paths)
	}
	if s.createCalled["app_name"] != "ai-sre" || s.createCalled["entry_kind"] != "api" || s.createCalled["team_id"] != float64(5) {
		t.Errorf("create body = %v", s.createCalled)
	}
	if s.uploadSess != "sess-new" || s.uploadName != "trace.txt" || s.uploadBody != "stack trace" {
		t.Errorf("upload = %q %q %q", s.uploadSess, s.uploadName, s.uploadBody)
	}
	m := s.message
	if m["session_id"] != "sess-new" {
		t.Errorf("message session_id = %v", m["session_id"])
	}
	if _, ok := m["app_name"]; ok {
		t.Errorf("app_name must be omitted once the session exists: %v", m)
	}
	parts, _ := m["parts"].([]any)
	if len(parts) != 2 {
		t.Fatalf("parts = %v", m["parts"])
	}
	file := parts[1].(map[string]any)["file"].(map[string]any)
	if parts[1].(map[string]any)["type"] != "file" || file["object_key"] != "attachments/a1/trace.txt" || file["mime"] != "text/plain" || file["filename"] != "trace.txt" {
		t.Errorf("file part = %v", parts[1])
	}
	if _, ok := m["query"]; ok {
		t.Errorf("query must be omitted when parts are present: %v", m["query"])
	}
	lead := parts[0].(map[string]any)
	if lead["type"] != "text" || lead["text"] != "see attached" {
		t.Errorf("leading part = %v, want the message text", lead)
	}
	if !strings.Contains(out, `"session_id":"sess-new"`) {
		t.Errorf("stdout = %q", out)
	}
}

func TestSafariSendFailsWhenReplyChannelNotAccepted(t *testing.T) {
	saveAndResetGlobals(t)
	newSendServer(t, "")
	out, err := execCommand("safari", "send", "hello")
	if err == nil {
		t.Fatal("expected an error when the ack does not grant reply_only")
	}
	for _, want := range []string{"reply channel", "sess-auto", "inv-1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("stdout must stay empty on failure, got %q", out)
	}
}

func TestSafariSendMessageSources(t *testing.T) {
	saveAndResetGlobals(t)
	s := newSendServer(t, "reply_only")
	dir := t.TempDir()
	path := filepath.Join(dir, "q.md")
	if err := os.WriteFile(path, []byte("  from file \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := execCommand("safari", "send", "--message-file", path); err != nil {
		t.Fatalf("message-file: %v", err)
	}
	if s.message["query"] != "from file" {
		t.Errorf("query = %q", s.message["query"])
	}

	stdinReader = strings.NewReader("from stdin")
	if _, err := execCommand("safari", "send", "--message-file", "-"); err != nil {
		t.Fatalf("message-file -: %v", err)
	}
	if s.message["query"] != "from stdin" {
		t.Errorf("query = %q", s.message["query"])
	}
}

func TestSafariSendValidation(t *testing.T) {
	saveAndResetGlobals(t)
	s := newSendServer(t, "reply_only")
	dir := t.TempDir()
	path := filepath.Join(dir, "q.md")
	_ = os.WriteFile(path, []byte("x"), 0o600)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"both sources", []string{"safari", "send", "a", "--message-file", path}, "not both"},
		{"env-id without kind", []string{"safari", "send", "a", "--env-id", "e1"}, "--env-id requires --env-kind"},
		{"too long", []string{"safari", "send", strings.Repeat("字", 32001)}, "limit is 32000"},
		{"missing file", []string{"safari", "send", "a", "--file", filepath.Join(dir, "nope")}, "--file"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := execCommand(c.args...)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want it to contain %q", err, c.want)
			}
		})
	}
	for _, p := range s.paths {
		if p == "/safari/session/message" {
			t.Errorf("a rejected command must not reach the message endpoint: %v", s.paths)
		}
	}
}
