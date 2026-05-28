package cli

import (
	"strings"
	"testing"
)

func TestMCPCreateFlagSurface(t *testing.T) {
	cmd := newMCPCreateCmd()
	flags := cmd.Flags()
	for _, name := range []string{
		"server-name", "description", "transport",
		"command", "args", "env", "url", "headers",
		"connect-timeout", "call-timeout", "team-id",
	} {
		if flags.Lookup(name) == nil {
			t.Errorf("flag --%s not registered", name)
		}
	}
}

func TestMCPCreateRejectsEmptyServerName(t *testing.T) {
	saveAndResetGlobals(t)
	// The empty-name guard fires inside runCommand before CreateMCPServer is
	// ever called, so a no-op stub is sufficient.
	newClientFn = func() (flashdutyClient, error) { return &mockClient{}, nil }

	_, err := execCommand("mcp", "create")
	if err == nil {
		t.Fatal("expected error for empty --server-name, got nil")
	}
	if !strings.Contains(err.Error(), "--server-name is required") {
		t.Fatalf("expected error %q, got %q", "--server-name is required", err.Error())
	}
}
