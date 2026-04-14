package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Test 77: When Execute returns an error, stderr contains "Error: <message>\n".
//
// main() calls os.Exit(1) on error, so we cannot invoke it directly in-process.
// Instead we re-execute the test binary with a helper environment variable
// that triggers main(). The subprocess stderr is then inspected.
func TestErrorFormatToStderr(t *testing.T) {
	if os.Getenv("TEST_MAIN_ERROR") == "1" {
		// Inside the subprocess: run main() with an invalid subcommand.
		// os.Args is already set by the exec.Command below.
		main()
		return
	}

	// Re-invoke ourselves as a subprocess so that os.Exit does not kill the
	// test runner. Pass an unknown subcommand to force an error from cobra.
	cmd := exec.Command(os.Args[0], "-test.run=^TestErrorFormatToStderr$")
	cmd.Env = append(os.Environ(), "TEST_MAIN_ERROR=1")
	// Override os.Args inside the subprocess by passing the invalid command
	// as a trailing argument after the test flags. We achieve this by setting
	// Args on the Cmd directly -- but we need the subprocess to call main()
	// with the right os.Args. The trick: we build the real binary instead.
	//
	// A simpler approach: build the CLI binary and invoke it.
	// But the cleanest approach for unit tests: just verify the formatting
	// contract and that cli.Execute() returns an error for bad input.

	// We use a different strategy: build the binary, run it with a bad
	// subcommand, and inspect stderr.
	binPath := filepath.Join(t.TempDir(), "flashduty-test")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = testMainDir(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("[#77] failed to build test binary: %v\n%s", err, out)
	}

	// Run the binary with an unknown subcommand.
	run := exec.Command(binPath, "nonexistent-subcommand-xyz")
	var stderr bytes.Buffer
	run.Stderr = &stderr
	err := run.Run()

	// The binary should exit non-zero.
	if err == nil {
		t.Fatalf("[#77] expected non-zero exit code for invalid subcommand, got success")
	}

	// Verify stderr contains the expected error format.
	got := stderr.String()
	if !strings.HasPrefix(got, "Error: ") {
		t.Errorf("[#77] stderr should start with \"Error: \", got %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("[#77] stderr should end with newline, got %q", got)
	}
	// The error message from cobra for unknown subcommands contains "unknown command".
	if !strings.Contains(got, "unknown command") {
		t.Errorf("[#77] stderr should mention \"unknown command\", got %q", got)
	}

	// Also verify the exact format pattern: "Error: <message>\n"
	// by checking it matches fmt.Sprintf("Error: %s\n", <something>).
	trimmed := strings.TrimPrefix(got, "Error: ")
	trimmed = strings.TrimSuffix(trimmed, "\n")
	reconstructed := fmt.Sprintf("Error: %s\n", trimmed)
	if reconstructed != got {
		t.Errorf("[#77] stderr does not match format \"Error: <msg>\\n\":\n  got:    %q\n  expect: %q", got, reconstructed)
	}
}

// Test 78: SetVersionInfo before Execute -- version/commit/date set by main
// are reflected in the `version` subcommand output.
func TestSetVersionInfoBeforeExecute(t *testing.T) {
	binPath := filepath.Join(t.TempDir(), "flashduty-test")

	// Build with custom ldflags to inject known version info, just like the
	// real release build does.
	ldflags := "-X main.version=1.2.3-test -X main.commit=abc1234 -X main.date=2026-04-13"
	build := exec.Command("go", "build", "-ldflags", ldflags, "-o", binPath, ".")
	build.Dir = testMainDir(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("[#78] failed to build test binary: %v\n%s", err, out)
	}

	// Run the binary with the "version" subcommand.
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

	// Verify each component is present individually for clearer diagnostics.
	for _, sub := range []string{"1.2.3-test", "abc1234", "2026-04-13"} {
		if !strings.Contains(got, sub) {
			t.Errorf("[#78] version output missing %q in %q", sub, got)
		}
	}
}

// testMainDir returns the directory of cmd/flashduty/main.go relative to this
// test file, so it works both locally and in CI.
func testMainDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Dir(filename)
}
