package cli

import (
	"runtime"
	"testing"

	"github.com/flashcatcloud/flashduty-cli/internal/update"
)

func TestRootSkipsAutoUpdateWhenManagedByRunner(t *testing.T) {
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
	t.Setenv("FLASHDUTY_MANAGED_BY_RUNNER", "1")

	origIsTerminal := isTerminalFn
	isTerminalFn = func(int) bool { return true }
	t.Cleanup(func() { isTerminalFn = origIsTerminal })

	called := false
	origCheck := checkForUpdateAutoFn
	checkForUpdateAutoFn = func(string) (*update.CheckResult, error) {
		called = true
		return &update.CheckResult{}, nil
	}
	t.Cleanup(func() { checkForUpdateAutoFn = origCheck })

	if _, err := execCommand("version"); err != nil {
		t.Fatalf("version command should run in runner-managed mode: %v", err)
	}
	if called {
		t.Fatal("runner-managed CLI must not check for updates")
	}
}
