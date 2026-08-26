package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// testBinName returns a platform-appropriate binary name.
func testBinName() string {
	if runtime.GOOS == "windows" {
		return "flashduty-test.exe"
	}
	return "flashduty-test"
}

// testMainDir returns the directory containing main.go, derived from this
// test file's location so it works in CI on any platform.
func testMainDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Dir(filename)
}

// buildTestBinary compiles the CLI binary into a temp directory.
func buildTestBinary(t *testing.T, ldflags string) string {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), testBinName())
	args := []string{"build", "-o", binPath}
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	args = append(args, ".")
	build := exec.Command("go", args...)
	build.Dir = testMainDir(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("failed to build test binary: %v\n%s", err, out)
	}
	return binPath
}

// Test 77: When Execute returns an error, stderr contains "Error: <message>\n".
func TestErrorFormatToStderr(t *testing.T) {
	binPath := buildTestBinary(t, "")

	run := exec.Command(binPath, "nonexistent-subcommand-xyz")
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	err := run.Run()

	if err == nil {
		t.Fatalf("[#77] expected non-zero exit code for invalid subcommand, got success")
	}

	// On most platforms the error goes to stderr. Combine both for robustness.
	got := stderr.String()
	if got == "" {
		got = stdout.String()
	}

	if !strings.HasPrefix(got, "Error: ") {
		t.Errorf("[#77] output should start with \"Error: \", got %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("[#77] output should end with newline, got %q", got)
	}
	if !strings.Contains(got, "unknown command") {
		t.Errorf("[#77] output should mention \"unknown command\", got %q", got)
	}

	trimmed := strings.TrimPrefix(got, "Error: ")
	trimmed = strings.TrimSuffix(trimmed, "\n")
	reconstructed := fmt.Sprintf("Error: %s\n", trimmed)
	if reconstructed != got {
		t.Errorf("[#77] output does not match format \"Error: <msg>\\n\":\n  got:    %q\n  expect: %q", got, reconstructed)
	}
}

// Test 78: SetVersionInfo before Execute -- version/commit/date set by main
// are reflected in the `version` subcommand output.
func TestSetVersionInfoBeforeExecute(t *testing.T) {
	ldflags := "-X main.version=1.2.3-test -X main.commit=abc1234 -X main.date=2026-04-13"
	binPath := buildTestBinary(t, ldflags)

	run := exec.Command(binPath, "version")
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("[#78] version command failed: %v\n%s", err, out)
	}

	got := string(out)
	want := "flashduty version 1.2.3-test (abc1234) built 2026-04-13\n"
	if got != want {
		t.Errorf("[#78] version output mismatch:\n  got:  %q\n  want: %q", got, want)
	}

	for _, sub := range []string{"1.2.3-test", "abc1234", "2026-04-13"} {
		if !strings.Contains(got, sub) {
			t.Errorf("[#78] version output missing %q in %q", sub, got)
		}
	}
}

// Test 79: When a compact list projection overflows its byte budget, the
// binary exits non-zero, writes nothing to stdout, and reports the error on
// stderr — a pipeline reading stdout must see a failed call, never an empty
// page masquerading as "no data".
func TestProjectionOverflowFailsHard(t *testing.T) {
	binPath := buildTestBinary(t, "")

	// Stub the alert-event list endpoint with a page whose projection stays
	// over the 16 KiB budget even after value shortening.
	var body strings.Builder
	body.WriteString(`{"request_id":"r","error":{"code":"OK","message":""},"data":{"total":100,"items":[`)
	for i := 0; i < 100; i++ {
		if i > 0 {
			body.WriteByte(',')
		}
		fmt.Fprintf(&body, `{"event_id":"%024x","alert_id":"%024x","event_severity":"Warning","event_status":"Triggered","event_time":1712000000,"title":%q}`,
			i, i+1_000_000, strings.Repeat("x", 200))
	}
	body.WriteString(`]}}`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body.String()))
	}))
	defer srv.Close()

	run := exec.Command(binPath, "alert-event", "list", "--limit", "100",
		"--output-format", "json", "--app-key", "test-key", "--base-url", srv.URL)
	// Isolate HOME so the test never reads the developer's real CLI config.
	run.Env = append(os.Environ(), "HOME="+t.TempDir())
	var stdout, stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr

	err := run.Run()
	if err == nil {
		t.Fatalf("[#79] expected non-zero exit code for an over-budget projection, got success; stderr:\n%s", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("[#79] a failed projection must write nothing to stdout, got %d bytes:\n%s", stdout.Len(), stdout.String())
	}
	if !strings.Contains(stderr.String(), "Error: projected list is") || !strings.Contains(stderr.String(), "exceeds the 16384-byte limit") {
		t.Errorf("[#79] stderr should report the byte-limit refusal, got:\n%s", stderr.String())
	}
}
