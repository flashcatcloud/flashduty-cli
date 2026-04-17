//go:build e2e

package e2e_test

import (
	"testing"
)

// Test 239: change list --since --until
func TestChangeListSinceUntil(t *testing.T) {
	r := runCLI(t, "change", "list", "--since", "168h", "--until", "now")
	requireSuccess(t, r)
}

// Test 242: change list --limit
func TestChangeListLimit(t *testing.T) {
	r := runCLI(t, "change", "list", "--limit", "5", "--since", "168h")
	requireSuccess(t, r)
}

// Test 243: change list --page
func TestChangeListPage(t *testing.T) {
	r := runCLI(t, "change", "list", "--page", "1", "--since", "168h")
	requireSuccess(t, r)
}

// Test 245: change list pagination footer
func TestChangeListPaginationFooter(t *testing.T) {
	r := runCLI(t, "change", "list", "--since", "168h")
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Showing")
}

// Test 246: change list --since invalid
func TestChangeListSinceInvalid(t *testing.T) {
	r := runCLI(t, "change", "list", "--since", "garbage")
	requireFailure(t, r)
}

// Test 247: change list --until invalid
func TestChangeListUntilInvalid(t *testing.T) {
	r := runCLI(t, "change", "list", "--until", "garbage")
	requireFailure(t, r)
}
