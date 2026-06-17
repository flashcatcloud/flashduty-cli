package cli

import "testing"

func TestTeamGetAcceptsPositionalID(t *testing.T) {
	cmd := newTeamGetCmd()
	// MaximumNArgs(1): two positional args should be rejected
	if cmd.Args != nil {
		if err := cmd.Args(cmd, []string{"123456", "789"}); err == nil {
			t.Fatal("team get should reject two positional args")
		}
	}
}

func TestTeamGetNoArgNoFlagFails(t *testing.T) {
	cmd := newTeamGetCmd()
	err := cmd.PreRunE(cmd, []string{})
	if err == nil {
		t.Fatal("expected error when no positional arg and no flag provided")
	}
}

func TestTeamGetPositionalArgBypassesPreRunE(t *testing.T) {
	cmd := newTeamGetCmd()
	err := cmd.PreRunE(cmd, []string{"123456"})
	if err != nil {
		t.Fatalf("PreRunE should succeed with positional arg, got: %v", err)
	}
}
