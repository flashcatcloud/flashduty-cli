package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flashcatcloud/go-flashduty"
)

// artifactServer fakes gallery-get → sign → stream for one artifact whose file
// is a JSON document (the shape the typed SDK path used to swallow).
func artifactServer(t *testing.T, file string, stall time.Duration) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/safari/artifact/gallery/get":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"artifact_id":"art_AAAAAAAAAAAAAAAAAAAAAA"`) {
				t.Errorf("gallery/get body = %s", body)
			}
			_, _ = io.WriteString(w, `{"data":{"artifact_id":"art_AAAAAAAAAAAAAAAAAAAAAA","file_id":"pf_BBBBBBBBBBBBBBBBBBBBBB"}}`)
		case "/safari/artifact/sign":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"file_id":"pf_BBBBBBBBBBBBBBBBBBBBBB"`) {
				t.Errorf("sign body = %s", body)
			}
			_, _ = io.WriteString(w, `{"data":{"download_url":"/safari/artifact/stream?mode=download&t=TOK%2B1","expires_in":300}}`)
		case "/safari/artifact/stream":
			if got := r.URL.Query().Get("t"); got != "TOK+1" {
				t.Errorf("stream t = %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", `attachment; filename="../../etc/data.json"`)
			w.(http.Flusher).Flush()
			time.Sleep(stall)
			_, _ = io.WriteString(w, file)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test-key", flashduty.WithBaseURL(srv.URL))
	}
	return srv
}

func TestArtifactDownloadFromConsoleLinkSavesServerName(t *testing.T) {
	saveAndResetGlobals(t)
	const file = `{"error":{"code":"x"},"rows":[1,2]}`
	artifactServer(t, file, 0)
	t.Chdir(t.TempDir())

	out, err := execCommand("safari", "artifact-download", "https://console.example/ai-sre/artifacts/art_AAAAAAAAAAAAAAAAAAAAAA")
	if err != nil {
		t.Fatalf("artifact-download: %v", err)
	}
	got, err := os.ReadFile("data.json") // base name only, never ../../etc
	if err != nil || string(got) != file {
		t.Fatalf("saved file = %q, %v; want %q", got, err, file)
	}
	if !strings.Contains(out, "Saved data.json (35 bytes, application/json)") {
		t.Fatalf("output = %q", out)
	}
}

func TestArtifactDownloadOutputFlagAndStructured(t *testing.T) {
	saveAndResetGlobals(t)
	artifactServer(t, "hello", 0)
	dst := filepath.Join(t.TempDir(), "x.txt")

	out, err := execCommand("safari", "artifact-download", "pf_BBBBBBBBBBBBBBBBBBBBBB", "-o", dst, "--json")
	if err != nil {
		t.Fatalf("artifact-download: %v", err)
	}
	var res artifactDownloadResult
	if err := json.Unmarshal([]byte(out), &res); err != nil || res.Path != dst || res.Bytes != 5 || res.FileID != "pf_BBBBBBBBBBBBBBBBBBBBBB" {
		t.Fatalf("result = %+v (%v) from %q", res, err, out)
	}
}

func TestArtifactDownloadStdout(t *testing.T) {
	saveAndResetGlobals(t)
	artifactServer(t, "line1\nline2\n", 0)

	out, err := execCommand("safari", "artifact-download", "pf_BBBBBBBBBBBBBBBBBBBBBB", "-o", "-")
	if err != nil || out != "line1\nline2\n" {
		t.Fatalf("stdout = %q, %v", out, err)
	}
}

func TestArtifactDownloadDefaultNameNeverReplacesLocalFile(t *testing.T) {
	saveAndResetGlobals(t)
	artifactServer(t, "new", 0)
	t.Chdir(t.TempDir())
	if err := os.WriteFile("data.json", []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := execCommand("safari", "artifact-download", "pf_BBBBBBBBBBBBBBBBBBBBBB")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v", err)
	}
	if got, _ := os.ReadFile("data.json"); string(got) != "mine" {
		t.Fatalf("local file changed to %q", got)
	}
}

func TestTransferBodyIgnoresSlowConsumer(t *testing.T) {
	old := transferIdleTimeout
	transferIdleTimeout = 50 * time.Millisecond
	t.Cleanup(func() { transferIdleTimeout = old })

	tr := startTransfer(t.Context())
	body := tr.trackBody(strings.NewReader("abc"))
	buf := make([]byte, 1)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			time.Sleep(80 * time.Millisecond) // consumer slower than the idle timeout
		}
		if err != nil {
			break
		}
	}
	if err := context.Cause(tr.ctx); err != nil {
		t.Fatalf("slow consumer cancelled the transfer: %v", err)
	}
	_ = tr.done(nil)
}

func TestArtifactDownloadRejectsRefWithoutID(t *testing.T) {
	saveAndResetGlobals(t)
	_, err := execCommand("safari", "artifact-download", "https://console.example/ai-sre/artifacts")
	if err == nil || !strings.Contains(err.Error(), "no artifact ID") {
		t.Fatalf("err = %v", err)
	}
}

func TestTransferStallKeepsEarlierFileAndLeavesNoPartial(t *testing.T) {
	saveAndResetGlobals(t)
	old := transferIdleTimeout
	transferIdleTimeout = 50 * time.Millisecond
	t.Cleanup(func() { transferIdleTimeout = old })
	artifactServer(t, "late", 500*time.Millisecond)
	dir := t.TempDir()
	dst := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(dst, []byte("earlier"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := execCommand("safari", "artifact-download", "pf_BBBBBBBBBBBBBBBBBBBBBB", "-o", dst)
	if !errors.Is(err, errTransferStalled) {
		t.Fatalf("err = %v, want errTransferStalled", err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "earlier" {
		t.Fatalf("earlier file changed to %q", got)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Fatalf("temp file left behind: %v", entries)
	}
}

func TestTransferProgressKeepsSlowTransferAlive(t *testing.T) {
	old := transferIdleTimeout
	transferIdleTimeout = 80 * time.Millisecond
	t.Cleanup(func() { transferIdleTimeout = old })

	tr := startTransfer(t.Context())
	pr, pw := io.Pipe()
	go func() {
		for range 5 { // 5×40ms = 200ms total, more than the idle timeout
			time.Sleep(40 * time.Millisecond)
			_, _ = pw.Write([]byte("x"))
		}
		_ = pw.Close()
	}()
	n, err := io.Copy(io.Discard, tr.track(pr))
	if err = tr.done(err); err != nil || n != 5 || tr.ctx.Err() == nil {
		t.Fatalf("n=%d err=%v ctxErr=%v", n, err, tr.ctx.Err())
	}
}
