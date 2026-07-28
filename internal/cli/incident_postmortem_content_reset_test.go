package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"
)

func TestIncidentPostMortemContentResetFromFile(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"post_mortem_id":      "pm_abc",
		"generation":          3,
		"revision":            3,
		"previous_generation": 2,
		"previous_revision":   2,
		"markdown_bytes":      42,
		"markdown_sha256":     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	// Preserve leading/trailing content; do not trim.
	markdown := "\n## impact\n中文\n\n"
	path := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(path, []byte(markdown), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := execCommand(
		"incident", "post-mortem-content-reset", "pm_abc",
		"--markdown-file", path,
		"--expected-revision", "0",
		"--idempotency-key", "aisre-run-1",
		"--json",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.requests != 1 {
		t.Fatalf("requests = %d, want 1", stub.requests)
	}
	if stub.lastPath != "/incident/post-mortem/content/reset" {
		t.Fatalf("path = %q", stub.lastPath)
	}
	assertBody(t, stub.lastBody, "post_mortem_id", "pm_abc")
	assertBody(t, stub.lastBody, "markdown", markdown)
	assertBody(t, stub.lastBody, "expected_revision", float64(0))
	assertBody(t, stub.lastBody, "idempotency_key", "aisre-run-1")
	if !strings.Contains(out, `"generation": 3`) {
		t.Fatalf("structured output missing generation:\n%s", out)
	}
	if !strings.Contains(out, `"revision": 3`) {
		t.Fatalf("structured output missing revision:\n%s", out)
	}
	if !strings.Contains(out, `"previous_generation": 2`) {
		t.Fatalf("structured output missing previous_generation:\n%s", out)
	}
}

func TestIncidentPostMortemContentResetFromStdin(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"post_mortem_id":      "pm_stdin",
		"generation":          1,
		"revision":            1,
		"previous_generation": 0,
		"previous_revision":   0,
		"markdown_bytes":      11,
		"markdown_sha256":     "ab",
	}

	markdown := "  keep spaces  \n"
	stdinReader = strings.NewReader(markdown)

	_, err := execCommand(
		"incident", "post-mortem-content-reset", "pm_stdin",
		"--markdown-file", "-",
		"--expected-revision", "7",
		"--idempotency-key", "key-stdin",
		"--json",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/post-mortem/content/reset" {
		t.Fatalf("path = %q", stub.lastPath)
	}
	assertBody(t, stub.lastBody, "markdown", markdown)
	assertBody(t, stub.lastBody, "expected_revision", float64(7))
}

func TestIncidentPostMortemContentResetHumanSummary(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"post_mortem_id":      "pm_human",
		"generation":          5,
		"revision":            1,
		"previous_generation": 4,
		"previous_revision":   12,
		"markdown_bytes":      3,
		"markdown_sha256":     "ab",
	}

	path := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(path, []byte("hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := execCommand(
		"incident", "post-mortem-content-reset", "pm_human",
		"--markdown-file", path,
		"--expected-revision", "12",
		"--idempotency-key", "key-human",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "Reset post-mortem content for pm_human: generation 4→5, revision 12→1"
	if !strings.Contains(out, want) {
		t.Fatalf("human output missing summary %q\n%s", want, out)
	}
}

func TestIncidentPostMortemContentResetValidationFailsBeforeRequest(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	path := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(path, []byte("ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	emptyPath := filepath.Join(t.TempDir(), "empty.md")
	if err := os.WriteFile(emptyPath, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}
	missingPath := filepath.Join(t.TempDir(), "missing.md")

	oversizedKey := strings.Repeat("字", maxPostMortemContentIdempotencyKeyRunes+1)
	if utf8.RuneCountInString(oversizedKey) <= maxPostMortemContentIdempotencyKeyRunes {
		t.Fatal("test setup: oversized key not long enough")
	}

	cases := []struct {
		name string
		args []string
		want string
		// setStdin, when non-empty, injects stdin for --markdown-file -
		setStdin string
	}{
		{
			name: "missing markdown-file",
			args: []string{
				"incident", "post-mortem-content-reset", "pm_x",
				"--expected-revision", "1",
				"--idempotency-key", "k",
			},
			want: "required flag",
		},
		{
			name: "missing idempotency-key",
			args: []string{
				"incident", "post-mortem-content-reset", "pm_x",
				"--markdown-file", path,
				"--expected-revision", "1",
			},
			want: "required flag",
		},
		{
			name: "negative expected-revision",
			args: []string{
				"incident", "post-mortem-content-reset", "pm_x",
				"--markdown-file", path,
				"--expected-revision", "-1",
				"--idempotency-key", "k",
			},
			want: "--expected-revision must be >= 0",
		},
		{
			name: "empty markdown file",
			args: []string{
				"incident", "post-mortem-content-reset", "pm_x",
				"--markdown-file", emptyPath,
				"--expected-revision", "0",
				"--idempotency-key", "k",
			},
			want: "markdown content must not be empty",
		},
		{
			name: "empty stdin",
			args: []string{
				"incident", "post-mortem-content-reset", "pm_x",
				"--markdown-file", "-",
				"--expected-revision", "0",
				"--idempotency-key", "k",
			},
			setStdin: "",
			want:     "markdown content must not be empty",
		},
		{
			name: "oversized idempotency key",
			args: []string{
				"incident", "post-mortem-content-reset", "pm_x",
				"--markdown-file", path,
				"--expected-revision", "0",
				"--idempotency-key", oversizedKey,
			},
			want: "at most 128 Unicode characters",
		},
		{
			name: "unreadable file",
			args: []string{
				"incident", "post-mortem-content-reset", "pm_x",
				"--markdown-file", missingPath,
				"--expected-revision", "0",
				"--idempotency-key", "k",
			},
			want: "failed to read markdown file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub.requests = 0
			stub.lastPath = ""
			stub.lastBody = nil
			if tc.name == "empty stdin" || tc.setStdin != "" || (len(tc.args) > 0 && containsArg(tc.args, "-")) {
				stdinReader = strings.NewReader(tc.setStdin)
			}
			_, err := execCommand(tc.args...)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want substring %q", err, tc.want)
			}
			if stub.requests != 0 {
				t.Fatalf("request was sent (path=%q) despite validation failure", stub.lastPath)
			}
		})
	}
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

// When --expected-revision is omitted, the CLI resolves the current revision
// through currentPostMortemRevisionFn and sends that.
func TestIncidentPostMortemContentResetFetchesCurrentRevision(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"post_mortem_id":      "pm_auto",
		"generation":          5,
		"revision":            1,
		"previous_generation": 4,
		"previous_revision":   16,
		"markdown_bytes":      3,
		"markdown_sha256":     "ab",
	}

	var gotID string
	currentPostMortemRevisionFn = func(_ *RunContext, postMortemID string) (int64, error) {
		gotID = postMortemID
		return 16, nil
	}

	path := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(path, []byte("hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := execCommand(
		"incident", "post-mortem-content-reset", "pm_auto",
		"--markdown-file", path,
		"--idempotency-key", "key-auto",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != "pm_auto" {
		t.Fatalf("revision lookup called with %q, want pm_auto", gotID)
	}
	if stub.lastPath != "/incident/post-mortem/content/reset" {
		t.Fatalf("path = %q", stub.lastPath)
	}
	assertBody(t, stub.lastBody, "expected_revision", float64(16))
}

// A revision-lookup failure aborts before the reset request is sent.
func TestIncidentPostMortemContentResetRevisionLookupFailure(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	currentPostMortemRevisionFn = func(_ *RunContext, _ string) (int64, error) {
		return 0, fmt.Errorf("boom")
	}

	path := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(path, []byte("hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := execCommand(
		"incident", "post-mortem-content-reset", "pm_x",
		"--markdown-file", path,
		"--idempotency-key", "k",
	)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected lookup error, got %v", err)
	}
	if stub.requests != 0 {
		t.Fatalf("reset was sent despite revision lookup failure (path=%q)", stub.lastPath)
	}
}

// fetchCurrentPostMortemRevision reads data.meta.revision from the info
// envelope served at the SDK client's base URL.
func TestFetchCurrentPostMortemRevision(t *testing.T) {
	saveAndResetGlobals(t)
	t.Setenv("FLASHDUTY_CRED_FD", "")
	flagAppKey = "test-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/incident/post-mortem/info" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("post_mortem_id") != "pm_1" {
			t.Fatalf("unexpected query %q", r.URL.RawQuery)
		}
		if r.URL.Query().Get("app_key") != "test-key" {
			t.Fatalf("app_key not propagated: %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id": "r1",
			"data":       map[string]any{"meta": map[string]any{"revision": 42}},
		})
	}))
	t.Cleanup(srv.Close)

	client, err := flashduty.NewClient("test-key", flashduty.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	ctx := &RunContext{Client: client, Cmd: cmd}

	rev, err := fetchCurrentPostMortemRevision(ctx, "pm_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rev != 42 {
		t.Fatalf("revision = %d, want 42", rev)
	}
}

// An API error envelope from the info endpoint surfaces as an error.
func TestFetchCurrentPostMortemRevisionAPIError(t *testing.T) {
	saveAndResetGlobals(t)
	t.Setenv("FLASHDUTY_CRED_FD", "")
	flagAppKey = "test-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id": "r1",
			"error":      map[string]any{"code": "NOT_FOUND", "message": "post-mortem not found"},
		})
	}))
	t.Cleanup(srv.Close)

	client, err := flashduty.NewClient("test-key", flashduty.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	ctx := &RunContext{Client: client, Cmd: cmd}

	_, err = fetchCurrentPostMortemRevision(ctx, "pm_missing")
	if err == nil || !strings.Contains(err.Error(), "post-mortem not found") {
		t.Fatalf("expected info error, got %v", err)
	}
}
