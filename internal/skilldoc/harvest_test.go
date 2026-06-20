package skilldoc

import "testing"

func TestHarvest_JoinsContinuationsAndSkipsProse(t *testing.T) {
	md := "text\n```bash\nfduty status-page change-create --type incident \\\n  --title \"x\"\n```\nmore\n`fduty incident detail <id>`\n"
	ex := HarvestExamples(md)
	if len(ex) != 2 {
		t.Fatalf("want 2 examples, got %d: %+v", len(ex), ex)
	}
	if ex[0].Tokens[0] != "status-page" || ex[0].Tokens[1] != "change-create" {
		t.Errorf("tok: %+v", ex[0].Tokens)
	}
	if !HasPlaceholder("<id>") || HasPlaceholder("--type") {
		t.Errorf("placeholder detection wrong")
	}
	// The joined example must carry the continuation's flags too, else the
	// validator would never see --title on a multi-line example.
	if !containsTok(ex[0].Tokens, "--title") || !containsTok(ex[0].Tokens, "--type") {
		t.Errorf("continuation flags lost: %+v", ex[0].Tokens)
	}
}

func TestHasPlaceholder_Variants(t *testing.T) {
	for _, tok := range []string{"<page-id>", "$VAR", "...", "ou_xxx", "inc_xxx"} {
		if !HasPlaceholder(tok) {
			t.Errorf("expected placeholder: %q", tok)
		}
	}
	for _, tok := range []string{"--type", "incident", "5750613685214", "change-create"} {
		if HasPlaceholder(tok) {
			t.Errorf("expected literal, not placeholder: %q", tok)
		}
	}
}

func containsTok(toks []string, want string) bool {
	for _, t := range toks {
		if t == want {
			return true
		}
	}
	return false
}
