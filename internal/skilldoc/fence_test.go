package skilldoc

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseFenceID_Shapes(t *testing.T) {
	cases := []struct {
		id   string
		want FenceSpec
		bad  bool
	}{
		{id: "channel", want: FenceSpec{Group: "channel"}},
		{id: "status-page", want: FenceSpec{Group: "status-page"}},
		{id: "channel[silence-rule]", want: FenceSpec{Group: "channel", Prefixes: []string{"silence-rule"}}},
		{id: "channel[silence-rule,inhibit-rule,unsubscribe-rule]", want: FenceSpec{Group: "channel", Prefixes: []string{"silence-rule", "inhibit-rule", "unsubscribe-rule"}}},
		{id: "channel[a, b]", bad: true}, // spaces are malformed, not tolerated
		{id: "channel[]", bad: true},     // empty claim list
		{id: "channel[a,]", bad: true},   // trailing comma
		{id: "Channel[a]", bad: true},    // groups are lowercase
		{id: "channel[a][b]", bad: true}, // one bracket span only
	}
	for _, c := range cases {
		got, err := ParseFenceID(c.id)
		if c.bad {
			if err == nil {
				t.Errorf("ParseFenceID(%q): want error, got %+v", c.id, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseFenceID(%q): %v", c.id, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseFenceID(%q) = %+v, want %+v", c.id, got, c.want)
		}
		if got.ID() != c.id {
			t.Errorf("ParseFenceID(%q).ID() = %q — id must round-trip", c.id, got.ID())
		}
	}
}

func TestFenceLocs_FindsStartsOnly(t *testing.T) {
	body := "intro\n" +
		FenceStart("svc[rule-]") + "\ncontent\n" + FenceEnd("svc[rule-]") + "\n\nmore prose\n" +
		FenceStart("svc") + "\ncontent\n" + FenceEnd("svc") + "\n"
	locs := FenceLocs(body)
	if len(locs) != 2 || locs[0].ID != "svc[rule-]" || locs[1].ID != "svc" {
		t.Fatalf("FenceLocs = %+v, want the two start markers in order", locs)
	}
	if locs[0].Offset >= locs[1].Offset {
		t.Errorf("offsets not in document order: %+v", locs)
	}
}

// partitionDump is a group with enough verbs to split: two rule verbs and two
// plain verbs.
func partitionDump() Dump {
	mk := func(verb string) Command {
		return Command{Path: "svc " + verb, Group: "svc", Short: "S " + verb, Use: verb}
	}
	return Dump{Commands: []Command{mk("list"), mk("create"), mk("rule-create"), mk("rule-delete")}}
}

func TestRenderGroupFences_SubsetPlusCatchAll(t *testing.T) {
	d := partitionDump()
	out, violations := RenderGroupFences(d, "svc", []string{"svc[rule-]", "svc"})
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %v", violations)
	}
	subset, catchAll := out["svc[rule-]"], out["svc"]
	if !strings.Contains(subset, "### rule-create") || !strings.Contains(subset, "### rule-delete") {
		t.Errorf("subset fence should carry the rule verbs:\n%s", subset)
	}
	if strings.Contains(subset, "### list") || strings.Contains(subset, "### create") {
		t.Errorf("subset fence must not carry unclaimed verbs:\n%s", subset)
	}
	if !strings.Contains(catchAll, "### list") || !strings.Contains(catchAll, "### create") {
		t.Errorf("catch-all fence should carry the remaining verbs:\n%s", catchAll)
	}
	if strings.Contains(catchAll, "### rule-create") {
		t.Errorf("catch-all fence must not repeat claimed verbs:\n%s", catchAll)
	}
	if !strings.HasPrefix(subset, FenceStart("svc[rule-]")) || !strings.HasSuffix(subset, FenceEnd("svc[rule-]")) {
		t.Errorf("subset fence markers must carry the full fence id:\n%s", subset)
	}
}

func TestRenderGroupFences_Violations(t *testing.T) {
	d := partitionDump()
	cases := []struct {
		name string
		ids  []string
		want string // substring of some violation
	}{
		{name: "double claim", ids: []string{"svc[rule-]", "svc[rule-create]", "svc"}, want: `verb "rule-create" claimed by`},
		{name: "dead prefix", ids: []string{"svc[nope]", "svc"}, want: `prefix "nope" claims no verb`},
		{name: "no catch-all", ids: []string{"svc[rule-]"}, want: "no catch-all fence for unclaimed verbs: create, list"},
		{name: "duplicate id", ids: []string{"svc", "svc"}, want: `fence "svc" appears more than once`},
		{name: "foreign group", ids: []string{"other", "svc"}, want: `fence "other" does not belong to group "svc"`},
	}
	for _, c := range cases {
		_, violations := RenderGroupFences(d, "svc", c.ids)
		found := false
		for _, v := range violations {
			if strings.Contains(v, c.want) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: want a violation containing %q, got %v", c.name, c.want, violations)
		}
	}
}

func TestCheckFences_SplitAcrossDocs(t *testing.T) {
	d := partitionDump()
	fresh, violations := RenderGroupFences(d, "svc", []string{"svc[rule-]", "svc"})
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %v", violations)
	}

	docs := []Doc{
		{Path: "rules", Body: "# Rules\n\n" + fresh["svc[rule-]"] + "\n"},
		{Path: "svc", Body: "# Svc\n\n" + fresh["svc"] + "\n"},
	}
	if issues := CheckFences(d, docs); len(issues) != 0 {
		t.Errorf("fresh split fences: want 0 issues, got %+v", issues)
	}

	stale := []Doc{
		{Path: "rules", Body: "# Rules\n\n" + FenceStart("svc[rule-]") + "\n\nWRONG\n\n" + FenceEnd("svc[rule-]") + "\n"},
		{Path: "svc", Body: "# Svc\n\n" + fresh["svc"] + "\n"},
	}
	issues := CheckFences(d, stale)
	if len(issues) != 1 || issues[0].Kind != "stale-fence" || issues[0].Doc != "rules" {
		t.Errorf("stale subset fence: want 1 stale-fence on rules, got %+v", issues)
	}
}

func TestCheckFences_TopologyIssues(t *testing.T) {
	d := partitionDump()
	// A subset fence with no catch-all anywhere: the group's remaining verbs
	// have no home.
	fresh, _ := RenderGroupFences(d, "svc", []string{"svc[rule-]", "svc"})
	docs := []Doc{{Path: "rules", Body: "# Rules\n\n" + fresh["svc[rule-]"] + "\n"}}
	issues := CheckFences(d, docs)
	if len(issues) != 1 || issues[0].Kind != "fence-topology" {
		t.Fatalf("want 1 fence-topology issue, got %+v", issues)
	}
	if !strings.Contains(issues[0].Detail, "no catch-all") {
		t.Errorf("detail should explain the missing catch-all: %+v", issues[0])
	}

	// A marker naming a group the CLI does not have.
	ghost := []Doc{{Path: "ghost", Body: FenceStart("nosuch") + "\n" + FenceEnd("nosuch") + "\n"}}
	issues = CheckFences(d, ghost)
	if len(issues) != 1 || issues[0].Kind != "fence-topology" || !strings.Contains(issues[0].Detail, "unknown command group") {
		t.Errorf("unknown group marker should be flagged: %+v", issues)
	}
}
