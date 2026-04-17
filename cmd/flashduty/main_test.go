package main

import (
	"bytes"
	"fmt"
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
