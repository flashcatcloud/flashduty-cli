package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateRejectsRunnerManagedCLIWithoutChecking(t *testing.T) {
	saveAndResetGlobals(t)
	t.Setenv("FLASHDUTY_MANAGED_BY_RUNNER", "1")

	checked := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		checked = true
		_, _ = w.Write([]byte("v0.0.0\n"))
	}))
	t.Cleanup(server.Close)
	t.Setenv("FLASHDUTY_UPDATE_BASE_URL", server.URL)

	_, err := execCommand("update", "--check")
	if err == nil || !strings.Contains(err.Error(), "managed by flashduty-runner") {
		t.Fatalf("expected runner-managed update rejection, got %v", err)
	}
	if checked {
		t.Fatal("runner-managed CLI must reject update before checking for releases")
	}
}
