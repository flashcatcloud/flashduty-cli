package skilldoc

import (
	"os"
	"strings"
	"testing"
)

func TestChannelCardDisambiguatesFlashcatWorkspace(t *testing.T) {
	body, err := os.ReadFile("../../skills/flashduty/reference/channel.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		"协作空间",
		"Flashcat workspace",
		"灭火图",
		"firemap",
		"do not silently switch",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("channel card must disambiguate Flashduty channel vs Flashcat workspace; missing %q", want)
		}
	}
}

// TestTemplateCardUpdateSemanticsAreDestructive pins the one fact the template card
// previously stated backwards. POST /template/update binds every channel field as a plain
// string and writes all of them unconditionally, so a channel absent from the request is
// stored as "" — omitting a flag deletes that channel's body for every escalation rule
// bound to the template. The card used to promise the opposite ("omitted channel flags are
// left unchanged"), which turns a one-field edit into a silent wipe of a live channel.
func TestTemplateCardUpdateSemanticsAreDestructive(t *testing.T) {
	body, err := os.ReadFile("../../skills/flashduty/reference/template.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)

	for _, banned := range []string{
		"omitted channel flags are left unchanged",
		"only supplied fields overwrite",
		// buildTemplateUpdates writes 14 channel-content fields, not 16.
		"16 channel fields",
		// status is not a field of update's request at all, so it is not a
		// pointer-typed input that "survives" omission.
		"and `status` survive omission",
	} {
		if strings.Contains(text, banned) {
			t.Errorf("template card reasserts the retracted patch-semantics claim %q; update writes every channel field unconditionally", banned)
		}
	}

	for _, want := range []string{
		"full-object replace",
		"is CLEARED",
		"survive omission",
		"14 channel-content fields",
		"info --json",
		// Length comparison, not a non-empty field-set check: only the former catches a
		// body that was truncated rather than cleared.
		"Verify LENGTHS",
		// The write path must move bodies with jq, never through command substitution,
		// which strips every trailing newline off a template body.
		"--rawfile",
		"--data -",
		`"$(cat`,
		"strips *all* trailing newlines",
		"--feishu-app-card-v2-table-enabled",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("template card missing %q", want)
		}
	}
}
