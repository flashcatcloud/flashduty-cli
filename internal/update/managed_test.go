package update

import "testing"

func TestIsManagedByRunner(t *testing.T) {
	t.Setenv("FLASHDUTY_MANAGED_BY_RUNNER", "1")
	if !IsManagedByRunner() {
		t.Fatal("runner-managed CLI was not detected")
	}

	t.Setenv("FLASHDUTY_MANAGED_BY_RUNNER", "true")
	if IsManagedByRunner() {
		t.Fatal("only the runner's explicit marker value should enable managed mode")
	}
}
