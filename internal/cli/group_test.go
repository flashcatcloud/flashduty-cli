package cli

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Unknown-verb hard failure
//
// Background: cobra's built-in "unknown command" detection (legacyArgs in
// args.go) only fires for the ROOT command — it returns nil whenever
// cmd.HasParent() is true. A command-group node (e.g. "incident") declared as
// a bare &cobra.Command{Use, Short} with no RunE is therefore judged
// non-Runnable, and cobra's execute() converts that into flag.ErrHelp, which
// ExecuteC always turns into "print help, exit 0" — regardless of whether the
// typed subcommand actually matched anything. A typo'd verb (e.g.
// `incident list-alerts`) looked exactly like a successful run.
//
// newGroupCmd (command.go) fixes this for every group in the tree, curated
// and generated, by giving the group a real Args validator + RunE so cobra
// actually validates the leftover token instead of discarding it.
// ---------------------------------------------------------------------------

func TestCommandUnknownSubcommandFailsLoudly(t *testing.T) {
	saveAndResetGlobals(t)

	// "lsit" is a 1-edit typo of the real "list" subcommand, well within
	// cobra's default SuggestionsMinimumDistance of 2, so a suggestion is
	// expected.
	_, err := execCommand("incident", "lsit")
	if err == nil {
		t.Fatal("expected an error for unknown subcommand \"lsit\", got nil")
	}
	if !strings.Contains(err.Error(), `unknown command "lsit" for "flashduty incident"`) {
		t.Fatalf("expected error to identify the unknown command, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "Did you mean this?") {
		t.Fatalf("expected a suggestion block, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "list") {
		t.Fatalf("expected \"list\" to be suggested, got %q", err.Error())
	}
}

// TestCommandUnknownSubcommandWithNoNearMatchStillFails covers a plausible
// but wrong guess — "incident list-alerts" for the real "incident alerts".
// It is too far from any real subcommand name, by both edit distance and
// prefix, for cobra's SuggestionsFor to match, so this asserts only the core
// guarantee: non-zero exit and a clearly-labeled unknown command, with no
// suggestion. See TestCommandUnknownSubcommandFailsLoudly for the
// suggestion-bearing case.
func TestCommandUnknownSubcommandWithNoNearMatchStillFails(t *testing.T) {
	saveAndResetGlobals(t)

	_, err := execCommand("incident", "list-alerts")
	if err == nil {
		t.Fatal("expected an error for unknown subcommand \"list-alerts\", got nil")
	}
	if !strings.Contains(err.Error(), `unknown command "list-alerts" for "flashduty incident"`) {
		t.Fatalf("expected error to identify the unknown command, got %q", err.Error())
	}
}

func TestCommandUnknownSubcommandUnderNestedGroupFailsLoudly(t *testing.T) {
	saveAndResetGlobals(t)

	// "oncall schedule" is a group nested two levels below root; it must get
	// the same treatment as a top-level group.
	_, err := execCommand("oncall", "schedule", "lsit")
	if err == nil {
		t.Fatal("expected an error for unknown subcommand \"lsit\", got nil")
	}
	if !strings.Contains(err.Error(), `unknown command "lsit" for "flashduty oncall schedule"`) {
		t.Fatalf("expected error to identify the unknown command, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "Did you mean this?") {
		t.Fatalf("expected a suggestion block, got %q", err.Error())
	}
}

func TestCommandGroupWithoutSubcommandPrintsHelp(t *testing.T) {
	saveAndResetGlobals(t)

	// No subcommand at all is the existing, friendly behavior: print help,
	// exit 0. This must NOT regress into an error.
	out, err := execCommand("incident")
	if err != nil {
		t.Fatalf("expected no error when running a group with no subcommand, got %v", err)
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected help output, got %q", out)
	}
	if !strings.Contains(out, "Available Commands:") {
		t.Fatalf("expected help output to list subcommands, got %q", out)
	}
}

func TestCommandLegitSubcommandUnaffected(t *testing.T) {
	saveAndResetGlobals(t)

	// config show needs no API client, making it a clean check that a real
	// subcommand still dispatches normally through a fixed-up group.
	out, err := execCommand("config", "show")
	if err != nil {
		t.Fatalf("unexpected error running a legitimate subcommand: %v", err)
	}
	if !strings.Contains(out, "app_key:") {
		t.Fatalf("expected config show output, got %q", out)
	}
}
