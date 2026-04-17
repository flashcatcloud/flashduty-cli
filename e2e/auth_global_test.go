//go:build e2e

package e2e_test

import (
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Version & Help (tests 105-110)
// ---------------------------------------------------------------------------

func TestVersionOutput(t *testing.T) {
	// Test 105: `version` contains "flashduty version"
	r := runCLIPublic(t, "version")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "flashduty version")
}

func TestVersionBuildInfo(t *testing.T) {
	// Test 106: version output contains version, commit, date
	r := runCLIPublic(t, "version")
	requireSuccess(t, r)
	// At minimum it should have the format "flashduty version X (Y) built Z"
	requireContains(t, r.Stdout, "built")
}

func TestRootHelp(t *testing.T) {
	// Test 107: --help contains command names
	r := runCLIPublic(t, "--help")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "incident")
	requireContains(t, r.Stdout, "channel")
	requireContains(t, r.Stdout, "member")
	requireContains(t, r.Stdout, "team")
	requireContains(t, r.Stdout, "config")
	requireContains(t, r.Stdout, "login")
}

func TestSubcommandHelp(t *testing.T) {
	// Test 108: incident --help lists subcommands
	r := runCLIPublic(t, "incident", "--help")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "list")
	requireContains(t, r.Stdout, "get")
	requireContains(t, r.Stdout, "create")
	requireContains(t, r.Stdout, "update")
	requireContains(t, r.Stdout, "ack")
	requireContains(t, r.Stdout, "close")
	requireContains(t, r.Stdout, "timeline")
	requireContains(t, r.Stdout, "alerts")
	requireContains(t, r.Stdout, "similar")
}

func TestHelpForEverySubcommand(t *testing.T) {
	// Test 110: all top-level commands show help without errors
	commands := []string{
		"channel", "member", "team", "field", "escalation-rule",
		"statuspage", "template", "change", "config", "login",
	}
	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			r := runCLIPublic(t, cmd, "--help")
			requireSuccess(t, r)
		})
	}
}

// ---------------------------------------------------------------------------
// Exit Codes (tests 100-104)
// ---------------------------------------------------------------------------

func TestSuccessExitCode(t *testing.T) {
	// Test 100: `version` exits 0
	r := runCLIPublic(t, "version")
	if r.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", r.ExitCode)
	}
}

func TestAuthErrorExitCode(t *testing.T) {
	// Test 101: no app key -> exit 1
	r := runCLINoAuth(t, "channel", "list")
	requireFailure(t, r)
}

func TestInvalidArgsExitCode(t *testing.T) {
	// Test 102: `incident get` with no ID -> exit 1
	r := runCLIPublic(t, "incident", "get")
	requireFailure(t, r)
}

func TestInvalidCommandExitCode(t *testing.T) {
	// Test 103: unknown command -> exit 1
	r := runCLIPublic(t, "nonexistent")
	requireFailure(t, r)
}

func TestMissingRequiredFlagExitCode(t *testing.T) {
	// Test 104: missing --channel for escalation-rule list -> exit 1
	r := runCLI(t, "escalation-rule", "list")
	requireFailure(t, r)
}

// ---------------------------------------------------------------------------
// Auth Error (tests 123-125)
// ---------------------------------------------------------------------------

func TestNoAuthSuggestsLogin(t *testing.T) {
	// Tests 123-125: no auth suggests running login
	commands := [][]string{
		{"channel", "list"},
		{"incident", "list"},
		{"member", "list"},
	}
	for _, args := range commands {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			r := runCLINoAuth(t, args...)
			requireFailure(t, r)
			requireContains(t, r.Stderr, "flashduty login")
		})
	}
}

// ---------------------------------------------------------------------------
// Output Format (tests 95, 97-99, 314)
// ---------------------------------------------------------------------------

func TestErrorsGoToStderr(t *testing.T) {
	// Test 97: failed command -> stderr has error, stdout is empty
	r := runCLINoAuth(t, "channel", "list")
	requireFailure(t, r)
	if r.Stderr == "" {
		t.Error("expected stderr to contain error message")
	}
}

func TestErrorPrefixFormat(t *testing.T) {
	// Test 99: stderr starts with "Error: "
	r := runCLINoAuth(t, "channel", "list")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "Error: ")
}

func TestAppKeyHiddenFromHelp(t *testing.T) {
	// Test 314: --help does NOT show --app-key
	r := runCLIPublic(t, "--help")
	requireSuccess(t, r)
	requireNotContains(t, r.Stdout, "--app-key")
}

// ---------------------------------------------------------------------------
// Global Flags (tests 79, 91-94)
// ---------------------------------------------------------------------------

func TestJSONOnListCommand(t *testing.T) {
	// Test 79: --json on channel list -> valid JSON
	r := runCLI(t, "channel", "list", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

func TestBaseURLValidOverride(t *testing.T) {
	// Test 91: --base-url with valid URL succeeds
	r := runCLI(t, "channel", "list", "--base-url", "https://api.flashcat.cloud")
	requireSuccess(t, r)
}

func TestBaseURLInvalid(t *testing.T) {
	// Test 92: --base-url with invalid URL -> error
	r := runCLI(t, "channel", "list", "--base-url", "https://invalid.example.com")
	requireFailure(t, r)
}

func TestAppKeyOverride(t *testing.T) {
	// Test 93: --app-key with valid key succeeds
	appKey := os.Getenv("FLASHDUTY_E2E_APP_KEY")
	if appKey == "" {
		t.Skip("FLASHDUTY_E2E_APP_KEY not set")
	}
	r := runCLINoAuth(t, "channel", "list", "--app-key", appKey)
	requireSuccess(t, r)
}

func TestAppKeyInvalid(t *testing.T) {
	// Test 94: --app-key with invalid key -> auth error
	r := runCLINoAuth(t, "channel", "list", "--app-key", "invalid_key_xyz")
	requireFailure(t, r)
}
