package cli

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/flashcatcloud/flashduty-cli/internal/update"
)

func TestRootAutoUpdateCheckTimeoutWarnsAfterCommand(t *testing.T) {
	saveAndResetGlobals(t)
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	}
	t.Setenv("CI", "")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("JENKINS_URL", "")
	t.Setenv("GITLAB_CI", "")
	t.Setenv("FLASHDUTY_NO_UPDATE_CHECK", "")

	origVersion := versionStr
	versionStr = "0.6.0"
	t.Cleanup(func() { versionStr = origVersion })

	origIsTerminal := isTerminalFn
	isTerminalFn = func(int) bool { return true }
	t.Cleanup(func() { isTerminalFn = origIsTerminal })

	called := false
	origCheck := checkForUpdateAutoFn
	checkForUpdateAutoFn = func(string) (*update.CheckResult, error) {
		called = true
		return nil, context.DeadlineExceeded
	}
	t.Cleanup(func() { checkForUpdateAutoFn = origCheck })

	out, err := execCommand("version")
	if err != nil {
		t.Fatalf("version command should still run when auto update check times out: %v", err)
	}
	if !called {
		t.Fatal("auto update check was not called")
	}
	if !strings.Contains(out, "flashduty version 0.6.0") {
		t.Fatalf("version output missing, got:\n%s", out)
	}
	if !strings.Contains(out, "auto update check timeout, please run 'flashduty update --check' manually") {
		t.Fatalf("timeout guidance missing, got:\n%s", out)
	}
}
