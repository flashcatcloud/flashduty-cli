package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flashcatcloud/flashduty-cli/internal/skilldoc"
)

// fixtureDump is a small dump with one status-page leaf, decoupled from the
// real CLI tree so the test stays deterministic.
func fixtureDump() skilldoc.Dump {
	return skilldoc.Dump{Commands: []skilldoc.Command{
		{
			Path:  "status-page change-create",
			Group: "status-page",
			Short: "Create status page event",
			Long: "Create status page event.\n\nRequest fields:\n" +
				"  --type string (required) — Event type. [incident, maintenance]\n",
			Flags: []skilldoc.Flag{{Name: "type", Type: "string"}, {Name: "data", Type: "string"}},
		},
	}}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunCheck_FlagsStaleAndUnknown(t *testing.T) {
	dir := t.TempDir()
	d := fixtureDump()

	// A reference card with a deliberately stale fence + a bad-flag example.
	body := "# status-page\n\n" +
		"```bash\nfduty status-page change-create --type incident --bogus 1\n```\n\n" +
		skilldoc.FenceStart("status-page") + "\n\n### change-create\nSTALE WRONG\n\n" +
		skilldoc.FenceEnd("status-page") + "\n"
	writeFile(t, filepath.Join(dir, "reference", "status-page.md"), body)

	var out bytes.Buffer
	n, err := runCheck(d, dir, &out)
	if err != nil {
		t.Fatalf("runCheck: %v", err)
	}
	if n == 0 {
		t.Fatalf("expected issues, got 0\noutput:\n%s", out.String())
	}
	got := out.String()
	if !strings.Contains(got, "stale-fence") {
		t.Errorf("missing stale-fence in output:\n%s", got)
	}
	if !strings.Contains(got, "unknown-flag") {
		t.Errorf("missing unknown-flag in output:\n%s", got)
	}
	// Issues are reported with a file:line prefix.
	if !strings.Contains(got, "status-page.md:") {
		t.Errorf("issues should carry file:line; output:\n%s", got)
	}
}

func TestRunCheck_CleanDirIsZero(t *testing.T) {
	dir := t.TempDir()
	d := fixtureDump()

	// A clean card: fresh fence + a valid example.
	body := "# status-page\n\n" +
		"```bash\nfduty status-page change-create --type incident\n```\n\n" +
		skilldoc.GenerateFence(d, "status-page") + "\n"
	writeFile(t, filepath.Join(dir, "reference", "status-page.md"), body)

	var out bytes.Buffer
	n, err := runCheck(d, dir, &out)
	if err != nil {
		t.Fatalf("runCheck: %v", err)
	}
	if n != 0 {
		t.Errorf("clean dir: want 0 issues, got %d:\n%s", n, out.String())
	}
}

// A card checked out with Windows CRLF line endings must still validate clean:
// the fence comparison and harvester normalize EOL, so freshness does not depend
// on how git materialized the file (regression test for the Windows CI failure).
func TestRunCheck_CRLFCardIsClean(t *testing.T) {
	dir := t.TempDir()
	d := fixtureDump()
	body := "# status-page\n\n" +
		"```bash\nfduty status-page change-create --type incident\n```\n\n" +
		skilldoc.GenerateFence(d, "status-page") + "\n"
	crlf := strings.ReplaceAll(body, "\n", "\r\n")
	writeFile(t, filepath.Join(dir, "reference", "status-page.md"), crlf)

	var out bytes.Buffer
	n, err := runCheck(d, dir, &out)
	if err != nil {
		t.Fatalf("runCheck: %v", err)
	}
	if n != 0 {
		t.Errorf("CRLF card should validate clean, got %d issue(s):\n%s", n, out.String())
	}
}

func TestRunCheck_MissingDirIsZero(t *testing.T) {
	d := fixtureDump()
	var out bytes.Buffer
	n, err := runCheck(d, filepath.Join(t.TempDir(), "does-not-exist"), &out)
	if err != nil {
		t.Fatalf("runCheck on missing dir should not error: %v", err)
	}
	if n != 0 {
		t.Errorf("missing skills dir: want 0 issues, got %d", n)
	}
}

// TestRunGenAll covers `skilldoc gen` with no group: it must regenerate every
// card that exists, derive the group set from the dump, and silently skip a dump
// group that has no card file (e.g. webhook) rather than erroring.
func TestRunGenAll_FillsEveryCardAndSkipsCardless(t *testing.T) {
	dir := t.TempDir()
	d := skilldoc.Dump{Commands: []skilldoc.Command{
		{Path: "status-page change-create", Group: "status-page", Short: "Create", Flags: []skilldoc.Flag{{Name: "type", Type: "string"}}},
		{Path: "incident list", Group: "incident", Short: "List incidents", Flags: []skilldoc.Flag{{Name: "limit", Type: "int"}}},
		{Path: "webhook list", Group: "webhook", Short: "List webhooks"}, // group with NO card file
	}}
	for _, g := range []string{"status-page", "incident"} {
		writeFile(t, filepath.Join(dir, "reference", g+".md"),
			"# "+g+"\n\nintro\n\n"+skilldoc.FenceStart(g)+"\n"+skilldoc.FenceEnd(g)+"\n")
	}

	if err := runGenAll(d, dir); err != nil {
		t.Fatalf("runGenAll must not error on the cardless webhook group: %v", err)
	}

	for g, verb := range map[string]string{"status-page": "### change-create", "incident": "### list"} {
		raw, err := os.ReadFile(filepath.Join(dir, "reference", g+".md"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), verb) {
			t.Errorf("%s card fence not filled (want %q):\n%s", g, verb, raw)
		}
		if !strings.Contains(string(raw), "intro") {
			t.Errorf("%s: gen-all clobbered hand-written content", g)
		}
	}
	var out bytes.Buffer
	if n, _ := runCheck(d, dir, &out); n != 0 {
		t.Errorf("after gen-all, check should be clean; got %d:\n%s", n, out.String())
	}
}

// independentlyClassifyResponse re-derives a command's response envelope
// shape ("object" | "array" | "wrapped") and its top-level (or, for the
// wrapped shape, per-row) field names straight from its raw Long text, using
// logic deliberately NOT shared with skilldoc's own responseShapeLine
// extractor (internal/skilldoc/generate.go) — a from-scratch re-read of the
// same ground truth, not a call into the code under test. ok is false when
// Long documents no Response fields block, or the block yields zero fields
// at the target depth (skilldoc's own extractor also emits nothing for that
// case — nothing to cross-check).
func independentlyClassifyResponse(long string) (shape string, fields []string, ok bool) {
	lines := strings.Split(long, "\n")
	headerLine := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "Response fields (") {
			headerLine = i
			break
		}
	}
	if headerLine < 0 {
		return "", nil, false
	}

	header := lines[headerLine]
	switch {
	case strings.Contains(header, "nested under items[]"):
		shape = "wrapped"
	case strings.Contains(header, "TOP-LEVEL array"):
		shape = "array"
	default:
		shape = "object"
	}
	prefix := "  - "
	if shape == "wrapped" {
		prefix = "    - " // one level under the sole top-level "items" row
	}
	for _, l := range lines[headerLine+1:] {
		if strings.TrimSpace(l) == "" {
			break
		}
		if !strings.HasPrefix(l, prefix) {
			continue
		}
		name := strings.TrimPrefix(l, prefix)
		if sp := strings.IndexAny(name, " ("); sp >= 0 {
			name = name[:sp]
		}
		fields = append(fields, name)
	}
	return shape, fields, len(fields) > 0
}

// isCligenWrapperWireName mirrors (independently — not by import) the three
// wire names cligen's own listEnvelope (internal/cmd/cligen/main.go) treats
// as a paginated-list envelope field.
func isCligenWrapperWireName(name string) bool {
	return name == "items" || name == "docs" || name == "list"
}

// responseLineOf returns the "- response: ..." bullet inside a rendered fence
// section, and whether one was present.
func responseLineOf(section string) (string, bool) {
	for _, l := range strings.Split(section, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "- response: ") {
			return l, true
		}
	}
	return "", false
}

// TestGenerateFence_ResponseShapeMatchesRealLong_AllCommands is the
// coverage-complete ground-truth cross-check: for every real command whose
// live Long (built from the actual CLI tree, not a fixture) documents a
// Response fields block, independently reclassify its envelope shape and
// field names (independentlyClassifyResponse, above — separate logic from
// the generator) and assert the fence GenerateFence renders for that verb
// agrees. A single hand-picked example (`schedule list`) proved the
// mechanism works but only ever covered one of ~200 documented commands;
// this walks the whole real dump, so a classification drift ANYWHERE in the
// generator fails the build, not just for the one verb someone happened to
// write a test against.
func TestGenerateFence_ResponseShapeMatchesRealLong_AllCommands(t *testing.T) {
	d := dump()

	checked := 0
	for _, c := range d.Commands {
		wantShape, wantFields, hasBlock := independentlyClassifyResponse(c.Long)
		if !hasBlock {
			continue
		}
		checked++

		// Render this ONE command in isolation (a single-command dump filtered
		// to its own group) rather than slicing a section out of the whole
		// group's fence: several real groups (e.g. "incident") flatten
		// same-named leaves from different subgroups — "incident get" and
		// "incident war-room get" both render as a "### get" heading — so a
		// substring/heading search across the full group fence can grab the
		// wrong command's section. A single-command fence has exactly one
		// response line, unambiguously.
		fence := skilldoc.GenerateFence(skilldoc.Dump{Commands: []skilldoc.Command{c}}, c.Group)
		gotLine, hasLine := responseLineOf(fence)

		// Mirrors the wrapper-drift guard in responseShapeLine: a response
		// this classifier reads as a top-level object whose sole field is one
		// of cligen's own list-envelope wire names (items/docs/list, array
		// type) is a case the generator deliberately suppresses rather than
		// assert a possibly-wrong shape. No real command hits this today
		// (cligen's own header would already say "wrapped" for it), but nothing
		// here should hard-fail if drift ever makes one — that is the guard
		// working as designed, not a bug.
		if wantShape == "object" && len(wantFields) == 1 && isCligenWrapperWireName(wantFields[0]) {
			if hasLine {
				t.Errorf("%s: expected the wrapper-drift guard to suppress this line (sole field %q looks like a list envelope), got:\n%s", c.Path, wantFields[0], gotLine)
			}
			continue
		}

		if !hasLine {
			t.Errorf("%s: generated fence has no response line for a documented Response fields block", c.Path)
			continue
		}
		switch wantShape {
		case "wrapped":
			if !strings.Contains(gotLine, "page wrapper") || !strings.Contains(gotLine, "jq '.items[]'") {
				t.Errorf("%s: want items[] page wrapper, got:\n%s", c.Path, gotLine)
			}
		case "array":
			if !strings.Contains(gotLine, "TOP-LEVEL array") {
				t.Errorf("%s: want TOP-LEVEL array, got:\n%s", c.Path, gotLine)
			}
		default: // "object"
			if !strings.Contains(gotLine, "single object") {
				t.Errorf("%s: want single object, got:\n%s", c.Path, gotLine)
			}
		}
		for _, f := range wantFields {
			if !strings.Contains(gotLine, f+" (") {
				t.Errorf("%s: missing real field %q (from live Long) in generated line:\n%s", c.Path, f, gotLine)
			}
		}
	}
	if checked < 100 {
		t.Fatalf("only cross-checked %d commands — expected on the order of 200; did Response-fields detection break, or has the real CLI shrunk?", checked)
	}
	t.Logf("cross-checked response shape/fields for %d real commands", checked)
}

func TestRunGen_FillsFence(t *testing.T) {
	dir := t.TempDir()
	d := fixtureDump()

	// Card with empty fence markers; gen should fill them with a fresh render.
	card := filepath.Join(dir, "reference", "status-page.md")
	body := "# status-page\n\nintro\n\n" +
		skilldoc.FenceStart("status-page") + "\n" + skilldoc.FenceEnd("status-page") + "\n"
	writeFile(t, card, body)

	if err := runGen(d, dir, "status-page"); err != nil {
		t.Fatalf("runGen: %v", err)
	}

	updated, err := os.ReadFile(card)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "### change-create") {
		t.Errorf("gen did not fill fence:\n%s", updated)
	}
	// After gen, the fence must be fresh (check reports 0 stale-fence issues).
	var out bytes.Buffer
	n, _ := runCheck(d, dir, &out)
	if n != 0 {
		t.Errorf("after gen, check should be clean; got %d:\n%s", n, out.String())
	}
	// Hand-written content outside the fence is preserved.
	if !strings.Contains(string(updated), "intro") {
		t.Errorf("gen clobbered hand-written content:\n%s", updated)
	}
}

// TestRunGen_SplitAcrossCards is the split-card path: one group whose subset
// fence and catch-all fence live in different files. gen must fill both from
// one group-wide render, and check must then be clean.
func TestRunGen_SplitAcrossCards(t *testing.T) {
	dir := t.TempDir()
	mk := func(verb string) skilldoc.Command {
		return skilldoc.Command{Path: "svc " + verb, Group: "svc", Short: "S " + verb, Use: verb}
	}
	d := skilldoc.Dump{Commands: []skilldoc.Command{mk("list"), mk("rule-create"), mk("rule-delete")}}

	rules := filepath.Join(dir, "reference", "rules.md")
	svc := filepath.Join(dir, "reference", "svc.md")
	writeFile(t, rules, "# rules\n\n"+skilldoc.FenceStart("svc[rule]")+"\n"+skilldoc.FenceEnd("svc[rule]")+"\n")
	writeFile(t, svc, "# svc\n\nintro\n\n"+skilldoc.FenceStart("svc")+"\n"+skilldoc.FenceEnd("svc")+"\n")

	if err := runGen(d, dir, "svc"); err != nil {
		t.Fatalf("runGen: %v", err)
	}

	rulesBody, err := os.ReadFile(rules)
	if err != nil {
		t.Fatal(err)
	}
	svcBody, err := os.ReadFile(svc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rulesBody), "### rule-create") || strings.Contains(string(rulesBody), "### list") {
		t.Errorf("rules card should carry exactly the claimed verbs:\n%s", rulesBody)
	}
	if !strings.Contains(string(svcBody), "### list") || strings.Contains(string(svcBody), "### rule-create") {
		t.Errorf("svc card should carry exactly the unclaimed remainder:\n%s", svcBody)
	}
	if !strings.Contains(string(svcBody), "intro") {
		t.Errorf("gen clobbered hand-written content:\n%s", svcBody)
	}

	var out bytes.Buffer
	if n, _ := runCheck(d, dir, &out); n != 0 {
		t.Errorf("after gen, check should be clean; got %d:\n%s", n, out.String())
	}
}

// TestRunGen_TopologyViolationFails asserts gen refuses to write anything when
// the group's fences do not partition its verbs.
func TestRunGen_TopologyViolationFails(t *testing.T) {
	dir := t.TempDir()
	mk := func(verb string) skilldoc.Command {
		return skilldoc.Command{Path: "svc " + verb, Group: "svc", Short: "S " + verb, Use: verb}
	}
	d := skilldoc.Dump{Commands: []skilldoc.Command{mk("list"), mk("rule-create")}}

	// Subset fence only — "list" has no home.
	writeFile(t, filepath.Join(dir, "reference", "rules.md"),
		"# rules\n\n"+skilldoc.FenceStart("svc[rule]")+"\n"+skilldoc.FenceEnd("svc[rule]")+"\n")

	err := runGen(d, dir, "svc")
	if err == nil || !strings.Contains(err.Error(), "no catch-all") {
		t.Fatalf("want topology error mentioning the missing catch-all, got %v", err)
	}
}

// TestRunGen_TwoFencesInOneFile pins the sequential splice loop: a single card
// carrying both a subset fence and the catch-all fence of the same group must
// have both rewritten in one pass (offsets are re-resolved by marker text
// after each splice, so the first replacement must not derail the second).
func TestRunGen_TwoFencesInOneFile(t *testing.T) {
	dir := t.TempDir()
	mk := func(verb string) skilldoc.Command {
		return skilldoc.Command{Path: "svc " + verb, Group: "svc", Short: "S " + verb, Use: verb}
	}
	d := skilldoc.Dump{Commands: []skilldoc.Command{mk("list"), mk("rule-create"), mk("rule-delete")}}

	card := filepath.Join(dir, "reference", "svc.md")
	writeFile(t, card, "# svc\n\nrules first\n\n"+
		skilldoc.FenceStart("svc[rule]")+"\n"+skilldoc.FenceEnd("svc[rule]")+"\n\nthen the rest\n\n"+
		skilldoc.FenceStart("svc")+"\n"+skilldoc.FenceEnd("svc")+"\n")

	if err := runGen(d, dir, "svc"); err != nil {
		t.Fatalf("runGen: %v", err)
	}

	body, err := os.ReadFile(card)
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)
	ruleAt := strings.Index(got, "### rule-create")
	listAt := strings.Index(got, "### list")
	if ruleAt < 0 || listAt < 0 || ruleAt > listAt {
		t.Fatalf("both fences must be filled, subset before catch-all:\n%s", got)
	}
	if !strings.Contains(got, "rules first") || !strings.Contains(got, "then the rest") {
		t.Errorf("gen clobbered hand-written content between fences:\n%s", got)
	}

	var out bytes.Buffer
	if n, _ := runCheck(d, dir, &out); n != 0 {
		t.Errorf("after gen, check should be clean; got %d:\n%s", n, out.String())
	}
}
