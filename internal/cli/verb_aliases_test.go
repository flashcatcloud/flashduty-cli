package cli

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// get/detail verb aliases
//
// The single-record read verb is spelled inconsistently across resources:
// incident answers both "get" and "detail", alert answered only "get", and
// channel answered only the path-derived "info". Aliases make the other
// spellings resolve to the same command instead of failing with
// "unknown command".
// ---------------------------------------------------------------------------

// TestCommandAlertDetailAliasResolvesToGet pins that `alert detail <id>` runs
// the same request as `alert get <id>` (POST /alert/info with the alert_id).
func TestCommandAlertDetailAliasResolvesToGet(t *testing.T) {
	for _, verb := range []string{"get", "detail"} {
		t.Run(verb, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			if _, err := execCommand("alert", verb, "alert-1"); err != nil {
				t.Fatalf("execCommand: %v", err)
			}
			if stub.lastPath != "/alert/info" {
				t.Fatalf("expected /alert/info, got %q", stub.lastPath)
			}
			if stub.lastBody["alert_id"] != "alert-1" {
				t.Fatalf("expected alert_id %q, got %#v", "alert-1", stub.lastBody["alert_id"])
			}
		})
	}
}

// TestCommandChannelGetDetailAliasesResolveToInfo pins that `channel get <id>`
// and `channel detail <id>` run the same request as the canonical
// `channel info <id>` (POST /channel/info with the channel_id).
func TestCommandChannelGetDetailAliasesResolveToInfo(t *testing.T) {
	for _, verb := range []string{"info", "get", "detail"} {
		t.Run(verb, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			if _, err := execCommand("channel", verb, "1001"); err != nil {
				t.Fatalf("execCommand: %v", err)
			}
			if stub.lastPath != "/channel/info" {
				t.Fatalf("expected /channel/info, got %q", stub.lastPath)
			}
			if stub.lastBody["channel_id"] != float64(1001) {
				t.Fatalf("expected channel_id 1001, got %#v", stub.lastBody["channel_id"])
			}
		})
	}
}

// TestCommandAliasHelpShowsCanonicalCommand verifies that invoking an alias
// with --help renders the aliased command's help (its canonical Use line and
// flags), not an error.
func TestCommandAliasHelpShowsCanonicalCommand(t *testing.T) {
	saveAndResetGlobals(t)

	out, err := execCommand("channel", "get", "--help")
	if err != nil {
		t.Fatalf("unexpected error running alias with --help: %v", err)
	}
	if !strings.Contains(out, "Get channel detail") {
		t.Fatalf("expected the channel info command's help, got %q", out)
	}
	if !strings.Contains(out, "--channel-id") {
		t.Fatalf("expected the channel info flags in help output, got %q", out)
	}
}

// TestCommandUnknownVerbStillFailsLoudly guards the aliases against swallowing
// genuinely unknown verbs: they must keep the hard failure (and the
// did-you-mean suggestion for near misses) from newGroupCmd.
func TestCommandUnknownVerbStillFailsLoudly(t *testing.T) {
	saveAndResetGlobals(t)

	if _, err := execCommand("alert", "show", "alert-1"); err == nil {
		t.Fatal("expected an error for unknown subcommand \"show\", got nil")
	} else if !strings.Contains(err.Error(), `unknown command "show" for "flashduty alert"`) {
		t.Fatalf("expected error to identify the unknown command, got %q", err.Error())
	}

	if _, err := execCommand("channel", "show", "1001"); err == nil {
		t.Fatal("expected an error for unknown subcommand \"show\", got nil")
	} else if !strings.Contains(err.Error(), `unknown command "show" for "flashduty channel"`) {
		t.Fatalf("expected error to identify the unknown command, got %q", err.Error())
	}

	// A near-miss typo of a real verb still gets a suggestion. (Suggestions
	// match command names; "gt" is a 1-edit typo of "get".)
	_, err := execCommand("alert", "gt", "alert-1")
	if err == nil {
		t.Fatal("expected an error for unknown subcommand \"gt\", got nil")
	}
	if !strings.Contains(err.Error(), "Did you mean this?") {
		t.Fatalf("expected a suggestion block, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "get") {
		t.Fatalf("expected \"get\" to be suggested, got %q", err.Error())
	}
}
