//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testBinary holds the path to the built binary (set once via TestMain).
var testBinary string

// TestMain builds the binary once before all tests, runs every test in the
// package, then cleans up the temporary directory that held the binary.
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "flashduty-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binPath := filepath.Join(tmpDir, "flashduty")
	// Build with race detector.
	cmd := exec.Command("go", "build", "-race", "-o", binPath, "./cmd/flashduty")
	cmd.Dir = findModuleRoot()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}

	testBinary = binPath
	os.Exit(m.Run())
}

// findModuleRoot walks up from the current directory until it finds go.mod.
func findModuleRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			fmt.Fprintln(os.Stderr, "could not find go.mod")
			os.Exit(1)
		}
		dir = parent
	}
}

// ---------------------------------------------------------------------------
// Environment helpers
// ---------------------------------------------------------------------------

// getE2EAppKey returns the app key from the environment or skips the test.
func getE2EAppKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("FLASHDUTY_E2E_APP_KEY")
	if key == "" {
		t.Skip("FLASHDUTY_E2E_APP_KEY not set, skipping E2E test")
	}
	return key
}

// getE2EBaseURL returns the base URL from the environment, defaulting to
// https://api.flashcat.cloud when the variable is unset.
func getE2EBaseURL() string {
	if v := os.Getenv("FLASHDUTY_E2E_BASE_URL"); v != "" {
		return v
	}
	return "https://api.flashcat.cloud"
}

// ---------------------------------------------------------------------------
// CLI execution
// ---------------------------------------------------------------------------

// cliResult holds the captured output of a CLI execution.
type cliResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// runCLI executes the CLI binary with the given args and the E2E app key set.
func runCLI(t *testing.T, args ...string) cliResult {
	t.Helper()
	return runCLIWithEnvAndHome(t, "", map[string]string{
		"FLASHDUTY_APP_KEY":  getE2EAppKey(t),
		"FLASHDUTY_BASE_URL": getE2EBaseURL(),
	}, nil, args...)
}

// runCLIPublic executes the CLI binary without requiring an E2E app key.
// Use this for commands that do not touch the API client, such as help/version.
func runCLIPublic(t *testing.T, args ...string) cliResult {
	t.Helper()
	return runCLIWithEnvAndHome(t, "", nil, nil, args...)
}

// runCLIPublicWithHome executes an auth-free command while reusing the given
// HOME directory across subprocesses. Use this for persistence tests.
func runCLIPublicWithHome(t *testing.T, home string, args ...string) cliResult {
	t.Helper()
	return runCLINoAuthWithHome(t, home, args...)
}

// runCLINoAuthWithHome executes the CLI binary without auth env vars while
// reusing the given HOME directory across subprocesses.
func runCLINoAuthWithHome(t *testing.T, home string, args ...string) cliResult {
	t.Helper()
	return runCLIWithEnvAndHome(t, home, nil, nil, args...)
}

// runCLINoAuth executes the CLI binary without setting the app key env var.
func runCLINoAuth(t *testing.T, args ...string) cliResult {
	t.Helper()
	return runCLIWithEnvAndHome(t, "", nil, nil, args...)
}

// runCLIWithStdin executes the CLI binary with piped stdin.
func runCLIWithStdin(t *testing.T, stdin string, args ...string) cliResult {
	t.Helper()
	return runCLIWithEnvAndHome(t, "", map[string]string{
		"FLASHDUTY_APP_KEY":  getE2EAppKey(t),
		"FLASHDUTY_BASE_URL": getE2EBaseURL(),
	}, &stdin, args...)
}

// runCLIWithEnvAndHome is the core executor. It starts the binary as a
// subprocess with a clean environment (only HOME, PATH, and the explicitly
// supplied variables), captures stdout/stderr, and returns the result.
func runCLIWithEnvAndHome(t *testing.T, home string, env map[string]string, stdin *string, args ...string) cliResult {
	t.Helper()

	cmd := exec.Command(testBinary, args...)

	if home == "" {
		home = t.TempDir()
	}

	// Set a clean environment so no inherited FLASHDUTY_APP_KEY leaks in.
	cmd.Env = []string{
		"HOME=" + home,
		"PATH=" + os.Getenv("PATH"),
	}
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	if stdin != nil {
		cmd.Stdin = strings.NewReader(*stdin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run CLI: %v", err)
		}
	}

	return cliResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// ---------------------------------------------------------------------------
// Assertion helpers
// ---------------------------------------------------------------------------

// requireSuccess fails the test if the CLI exited with a non-zero code.
func requireSuccess(t *testing.T, r cliResult) {
	t.Helper()
	if r.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s\nstderr: %s",
			r.ExitCode, r.Stdout, r.Stderr)
	}
}

// requireFailure fails the test if the CLI exited with code 0.
func requireFailure(t *testing.T, r cliResult) {
	t.Helper()
	if r.ExitCode == 0 {
		t.Fatalf("expected non-zero exit, got 0\nstdout: %s\nstderr: %s",
			r.Stdout, r.Stderr)
	}
}

// requireValidJSON fails the test if the trimmed output is not valid JSON.
func requireValidJSON(t *testing.T, output string) {
	t.Helper()
	output = strings.TrimSpace(output)
	if !json.Valid([]byte(output)) {
		t.Fatalf("output is not valid JSON:\n%s", output)
	}
}

// requireTableHeaders fails the test if any of the given headers are missing
// from the first line of the output.
func requireTableHeaders(t *testing.T, output string, headers ...string) {
	t.Helper()
	firstLine := strings.Split(output, "\n")[0]
	for _, h := range headers {
		if !strings.Contains(firstLine, h) {
			t.Errorf("header %q not found in first line: %s", h, firstLine)
		}
	}
}

// requireContains fails the test if output does not contain substring.
func requireContains(t *testing.T, output, substring string) {
	t.Helper()
	if !strings.Contains(output, substring) {
		t.Errorf("output does not contain %q:\n%s", substring, output)
	}
}

// requireNotContains fails the test if output contains substring.
func requireNotContains(t *testing.T, output, substring string) {
	t.Helper()
	if strings.Contains(output, substring) {
		t.Errorf("output should not contain %q:\n%s", substring, output)
	}
}

// ---------------------------------------------------------------------------
// Extraction helpers
// ---------------------------------------------------------------------------

// extractIncidentID extracts an incident ID from CLI output. It first tries
// to parse the output as JSON looking for a "message" field, then looks for
// the pattern "Incident created: <id>" line by line.
func extractIncidentID(t *testing.T, output string) string {
	t.Helper()

	trimmed := strings.TrimSpace(output)

	// Try JSON format first.
	if json.Valid([]byte(trimmed)) {
		var result map[string]string
		if err := json.Unmarshal([]byte(trimmed), &result); err == nil {
			if msg, ok := result["message"]; ok {
				trimmed = msg
			}
		}
	}

	// Parse "Incident created: <id>".
	const prefix = "Incident created: "
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}

	t.Fatalf("could not extract incident ID from output:\n%s", output)
	return ""
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

// uniqueName generates a unique test name with the given prefix and a
// nanosecond timestamp to avoid collisions across runs.
func uniqueName(prefix string) string {
	return fmt.Sprintf("e2e_%s_%d", prefix, time.Now().UnixNano())
}
