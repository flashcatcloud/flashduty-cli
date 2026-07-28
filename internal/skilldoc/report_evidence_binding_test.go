package skilldoc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSkillCardBindsEvidenceToScopeNotJustVerb locks in the generalized
// evidence-binding rule: a claim about a specific time window or entity may
// only be made if a tool call this turn actually covered that window or
// entity, not merely the same verb. This patches the gap behind three
// production failures — reporting per-day/WoW figures from a rolling-window
// call, inventing a baseline window no call touched, and generalizing one
// entity's config from its siblings' — all of which passed the older,
// narrower "did you run the command at all" check.
func TestSkillCardBindsEvidenceToScopeNotJustVerb(t *testing.T) {
	card, err := os.ReadFile(filepath.Join("..", "..", "skills", "flashduty", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(card)

	_, section, found := strings.Cut(body, "## Output — prefer `toon`")
	if !found {
		t.Fatal("SKILL.md is missing the Output section that carries the evidence-binding rule")
	}
	section, _, found = strings.Cut(section, "## Command names")
	if !found {
		t.Fatal("SKILL.md is missing the section after Output — prefer `toon`")
	}

	// The rule must bind a claim to the scope actually queried (time window,
	// entity) — not just to having run the right verb at some point this
	// turn. A call for one window or entity must not license a claim about
	// a different one.
	if !strings.Contains(section, "scope") {
		t.Error("SKILL.md evidence-binding rule must talk about matching the queried scope, not just the verb")
	}
	if !strings.Contains(section, "window") || !strings.Contains(section, "entity") {
		t.Error("SKILL.md evidence-binding rule must name both axes it covers: time window and entity")
	}
	// It must forbid generalizing from adjacent evidence (a wider/different
	// window, a sibling entity) — the specific failure mode this rule exists
	// to stop, not just "don't invent from nothing".
	if !strings.Contains(section, "does not transfer") && !strings.Contains(section, "extrapolat") {
		t.Error("SKILL.md evidence-binding rule must forbid extrapolating a claim from a window or entity you queried differently")
	}
	// It must give a concrete fallback action, mirroring incident.md's
	// established phrasing, not just a prohibition with nothing to do
	// instead.
	if !strings.Contains(section, "未查询") || !strings.Contains(section, "<command>") {
		t.Error("SKILL.md evidence-binding rule must give a concrete fallback action (未查询 — 可运行 <command>), not just a prohibition")
	}
}

// TestInsightCardTiesWindowComparisonsToSkillRule locks in the insight-card
// instance of the same rule: day-over-day / week-over-week claims require a
// single call spanning every window compared, with --aggregate-unit as the
// concrete way to get one. This is the domain-specific reinforcement, not a
// duplicate of the general SKILL.md rule.
func TestInsightCardTiesWindowComparisonsToSkillRule(t *testing.T) {
	card, err := os.ReadFile(filepath.Join("..", "..", "skills", "flashduty", "reference", "insight.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(card)

	_, gotchas, found := strings.Cut(body, "## Gotchas")
	if !found {
		t.Fatal("insight card is missing the Gotchas section")
	}
	gotchas, _, found = strings.Cut(gotchas, "## Worked example")
	if !found {
		t.Fatal("insight card is missing the section after Gotchas")
	}

	if !strings.Contains(gotchas, "window") {
		t.Error("insight Gotchas must warn that a window-over-window claim needs a call spanning every window compared")
	}
	if !strings.Contains(gotchas, "--aggregate-unit") {
		t.Error("insight Gotchas must point to --aggregate-unit as the concrete way to get real per-bucket figures in one call")
	}
	// It should point back to the general rule rather than re-deriving it —
	// SKILL.md and insight.md must not carry two competing versions of the
	// same rule.
	if !strings.Contains(gotchas, "SKILL.md") {
		t.Error("insight Gotchas should reference the general SKILL.md evidence-binding rule instead of restating it")
	}
}
