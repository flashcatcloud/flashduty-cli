package cli

import (
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
	cmd := newMCPCreateCmd()
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for empty --server-name, got nil")
	}
}
