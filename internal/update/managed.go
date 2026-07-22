package update

import "os"

// IsManagedByRunner reports whether flashduty-runner owns this CLI binary.
// The marker is intentionally exact so a user environment cannot
// accidentally disable standalone CLI updates with a truthy-looking value.
func IsManagedByRunner() bool {
	return os.Getenv("FLASHDUTY_MANAGED_BY_RUNNER") == "1"
}
