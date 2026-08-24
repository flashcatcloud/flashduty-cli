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

// TestTemplateCardUpdateSemanticsArePartial pins the one fact this card has now had
// backwards in both directions. POST /template/update binds every channel field as
// *string and writes only the non-nil ones, so a channel absent from the request keeps
// its stored body and only an explicit empty string clears it. The card once promised
// omission preserved (true today, false then), was corrected to warn that omission
// cleared (true then, false today), and this test exists so the correction does not
// outlive the server behaviour that motivated it.
func TestTemplateCardUpdateSemanticsArePartial(t *testing.T) {
	body, err := os.ReadFile("../../skills/flashduty/reference/template.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)

	// Claims that are false against the current server, in any phrasing.
	for _, banned := range []string{
		// the destructive reading, retired when update became a partial update
		"full-object replace",
		"is CLEARED",
		"every channel field you do not pass is written",
		// buildTemplateUpdates covers 14 channel-content fields, not 16.
		"16 channel fields",
		// status is not a field of update's request at all, so it is not something
		// that "survives" omission alongside the pointer-typed inputs.
		"and `status` survive omission",
	} {
		if strings.Contains(text, banned) {
			t.Errorf("template card asserts %q; update writes only the fields the request carries", banned)
		}
	}

	for _, want := range []string{
		// the semantics, stated as a partial update
		"partial update",
		"leaves it alone",
		// clearing is now an explicit act, and the sharp edge is that older builds
		// dropped an empty-string flag before it reached the wire, making a clear a
		// silent no-op. Both halves have to stay on the card.
		"explicit empty string",
		"v1.4.2",
		"14 channel-content fields",
		"feishu_app_card_v2_table_enabled",
		"info --json",
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
