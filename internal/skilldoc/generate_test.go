package skilldoc

import (
	"strings"
	"testing"
)

// generatorDump mirrors the real cligen Long shape: a Request fields block with
// a required enum flag, a non-required flag, and a nested --data body field.
func generatorDump() Dump {
	return Dump{Commands: []Command{
		{
			Path:  "status-page change-create",
			Group: "status-page",
			Short: "Create status page event",
			Use:   "change-create <page-id>",
			Long: `Create status page event.

Create a new incident or maintenance event on a status page.

API: POST /status-page/change/create (statusPageChangeCreate)

Request fields:
  --description string — Event description (Markdown). Required by the validator.
  --page-id int (required) — Status page ID.
  --type string (required) — Event type. [incident, maintenance]
  updates (array<object>, via --data) (required) — Timeline updates.
    - status (string) — Change status after this update. [investigating, identified, monitoring, resolved]

Response fields ('data' envelope is unwrapped — these fields are at the top level):
  - change_id (integer) (required) — Newly created event ID.
`,
			Flags: []Flag{
				{Name: "description", Type: "string"},
				{Name: "page-id", Type: "int"},
				{Name: "type", Type: "string"},
				{Name: "data", Type: "string"},
			},
		},
		{
			Path:  "status-page change-active-list",
			Group: "status-page",
			Short: "List active status page events",
			Use:   "change-active-list <page-id>",
			Long: `List active status page events.

Request fields:
  --page-id int (required) — Status page ID.
  --type string (required) — Event type filter. [incident, maintenance]
`,
			Flags: []Flag{{Name: "page-id", Type: "int"}, {Name: "type", Type: "string"}, {Name: "data", Type: "string"}},
		},
		// A different group must be excluded.
		{Path: "incident detail", Group: "incident", Short: "x", Flags: []Flag{{Name: "data"}}},
	}}
}

// foldedFlagNames must fold the EXACT flag a positional shadows. A scalar
// "<type>" folds only "type" — an unrelated plural flag "--types" must survive.
// An array positional "<incident-id> [<id2>...]" folds the plural "*-ids" wire
// ("incident-ids"), since cligen singularizes the placeholder. Matching on a
// trailing-"s"-stripped key would wrongly collapse "types" onto "<type>".
func TestFoldedFlagNames_ExactScalarAndArrayPlural(t *testing.T) {
	scalar := foldedFlagNames([]string{"<type>"})
	if !scalar["type"] {
		t.Errorf("scalar <type> should fold flag `type`: %v", scalar)
	}
	if scalar["types"] {
		t.Errorf("scalar <type> must NOT fold unrelated plural `types`: %v", scalar)
	}
	array := foldedFlagNames([]string{"<incident-id>", "[<id2>...]"})
	if !array["incident-ids"] {
		t.Errorf("array <incident-id> [<id2>...] should fold plural wire `incident-ids`: %v", array)
	}
	if array["incident-id"] {
		t.Errorf("array positional should fold the plural wire only, not singular: %v", array)
	}
	if n := len(foldedFlagNames([]string{"[<incident-id>]"})); n != 0 {
		t.Errorf("optional [<incident-id>] must fold nothing, got %v", n)
	}
}

func TestGenerateFence_StatusPage(t *testing.T) {
	d := generatorDump()
	out := GenerateFence(d, "status-page")

	// Fence markers, scoped to the group.
	if !strings.Contains(out, "GENERATED:status-page START") {
		t.Errorf("missing start marker:\n%s", out)
	}
	if !strings.Contains(out, "GENERATED:status-page END") {
		t.Errorf("missing end marker:\n%s", out)
	}

	// Each leaf verb of the group is listed; other groups are excluded.
	if !strings.Contains(out, "### change-create") {
		t.Errorf("missing change-create section:\n%s", out)
	}
	if !strings.Contains(out, "### change-active-list") {
		t.Errorf("missing change-active-list section:\n%s", out)
	}
	if strings.Contains(out, "incident detail") {
		t.Errorf("other-group command leaked into fence:\n%s", out)
	}

	// change-create's --type is required and carries its enum.
	if !strings.Contains(out, "--type") {
		t.Errorf("missing --type flag:\n%s", out)
	}
	if !strings.Contains(out, "incident | maintenance") {
		t.Errorf("missing --type enum incident | maintenance:\n%s", out)
	}

	// Deterministic.
	if out != GenerateFence(d, "status-page") {
		t.Errorf("GenerateFence not deterministic")
	}
}

// TestGenerateFence_RequiredMarker checks required flags are visibly marked.
func TestGenerateFence_RequiredMarker(t *testing.T) {
	out := GenerateFence(generatorDump(), "status-page")
	// The change-create section must mark --type and --page-id required but not
	// --description.
	sec := sectionFor(out, "change-create")
	if !strings.Contains(sec, "--type") || !markedRequired(sec, "--type") {
		t.Errorf("--type should be marked required in section:\n%s", sec)
	}
	if markedRequired(sec, "--description") {
		t.Errorf("--description should NOT be marked required:\n%s", sec)
	}
}

// TestGenerateFence_PositionalArg asserts that a field cligen folded into a
// positional argument (recorded in Use as "<page-id>") is rendered as a
// positional — shown in the verb heading and as a `(positional, required)` row —
// and is NOT presented as a `--page-id` flag (passing the flag alone fails the
// binary's Args check). The non-folded --type flag must still render normally.
func TestGenerateFence_PositionalArg(t *testing.T) {
	sec := sectionFor(GenerateFence(generatorDump(), "status-page"), "change-active-list")
	if sec == "" {
		t.Fatal("no change-active-list section")
	}
	// Heading carries the positional signature.
	if !strings.Contains(sec, "### change-active-list <page-id>") {
		t.Errorf("heading should show positional `<page-id>`:\n%s", sec)
	}
	// page-id is documented as a positional, not as a --flag.
	if !strings.Contains(sec, "`<page-id>` (positional, required)") {
		t.Errorf("page-id should render as a positional row:\n%s", sec)
	}
	if strings.Contains(sec, "--page-id") {
		t.Errorf("folded positional must NOT appear as a --page-id flag row:\n%s", sec)
	}
	// A non-folded flag still renders as a flag.
	if !strings.Contains(sec, "--type") {
		t.Errorf("non-folded --type flag should remain:\n%s", sec)
	}
}

// TestSplitTrailingEnum_IgnoresBracketedProseInsideDescription reproduces the
// real `field create --field-name` bug: its description quotes a regex
// character class mid-sentence ("1-40 chars of '[a-zA-Z0-9_]'") followed by
// more prose and a trailing constraint. An unanchored bracket match treats
// that as a one-value enum declaration, fabricating an "enum: a-zA-Z0-9_" tag
// AND deleting the bracket from the sentence (leaving an empty pair of
// quotes where the character class used to be). Since
// the bracket isn't at the tail's end (more text and the constraint follow
// it), it must be left alone entirely: no enum, description word-for-word.
func TestSplitTrailingEnum_IgnoresBracketedProseInsideDescription(t *testing.T) {
	tail := `(required) — Machine name. Must start with a letter or underscore; 1–40 chars of '[a-zA-Z0-9_]'. Immutable after creation. (≤39 chars)`
	if enum := parseEnum(tail); enum != nil {
		t.Errorf("bracketed prose must not be parsed as an enum, got %v", enum)
	}
	got := cleanUsage(tail)
	want := `Machine name. Must start with a letter or underscore; 1–40 chars of '[a-zA-Z0-9_]'. Immutable after creation. (≤39 chars)`
	if got != want {
		t.Errorf("cleanUsage must leave the character class and constraint untouched:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestSplitTrailingEnum_RecognizesGenuineTrailingEnum proves the fix doesn't
// overcorrect: cligen's own trailing enum bracket ("line += " [" + ... +
// "]"", cligen/main.go) -- anchored at the tail's end, nothing else
// following it -- is still recognized and still stripped from the usage
// text.
func TestSplitTrailingEnum_RecognizesGenuineTrailingEnum(t *testing.T) {
	tail := `(required) — Event type. [incident, maintenance]`
	if enum := parseEnum(tail); !equalStrings(enum, []string{"incident", "maintenance"}) {
		t.Errorf("genuine trailing enum not recognized, got %v", enum)
	}
	if got, want := cleanUsage(tail), "Event type."; got != want {
		t.Errorf("cleanUsage should strip the genuine enum bracket:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestSplitTrailingEnum_GenuineEnumWithTrailingConstraint covers a flag that
// has BOTH an enum and a constraint (cligen writes enum then constraint, in
// that order, when both are present): the enum must still be recognized and
// stripped, and the constraint must survive untouched and back in its
// original trailing position -- not swallowed by the enum-bracket removal.
func TestSplitTrailingEnum_GenuineEnumWithTrailingConstraint(t *testing.T) {
	tail := `— Some usage. [a, b] (≤5 chars)`
	if enum := parseEnum(tail); !equalStrings(enum, []string{"a", "b"}) {
		t.Errorf("enum with a trailing constraint not recognized, got %v", enum)
	}
	if got, want := cleanUsage(tail), "Some usage. (≤5 chars)"; got != want {
		t.Errorf("cleanUsage must strip only the enum bracket, keeping the constraint:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestSplitTrailingEnum_SingleValueEnumAtEnd covers a one-member enum (e.g.
// the real `--environment-kind [byoc]`), proving the fix doesn't require a
// comma to recognize a genuine trailing bracket as an enum.
func TestSplitTrailingEnum_SingleValueEnumAtEnd(t *testing.T) {
	tail := `— Pin to a specific runner. [byoc]`
	if enum := parseEnum(tail); !equalStrings(enum, []string{"byoc"}) {
		t.Errorf("single-value trailing enum not recognized, got %v", enum)
	}
}

// TestGenerateFence_BracketedProseSurvivesAlongsideGenuineEnum is the
// fence-level integration check: one command with a description-embedded
// character class (must NOT become an enum) and, on a different flag in the
// same command, a genuine trailing enum (must still render as one) —
// proving the fix distinguishes them within the same Request-fields block,
// not just in isolation.
func TestGenerateFence_BracketedProseSurvivesAlongsideGenuineEnum(t *testing.T) {
	d := Dump{Commands: []Command{{
		Path:  "field create",
		Group: "field",
		Short: "Create field",
		Use:   "create",
		Long: `Create field.

Request fields:
  --field-name string (required) — Machine name. Must start with a letter or underscore; 1–40 chars of '[a-zA-Z0-9_]'. Immutable after creation. (≤39 chars)
  --field-type string (required) — Field input type. [checkbox, text]
`,
		Flags: []Flag{{Name: "field-name", Type: "string"}, {Name: "field-type", Type: "string"}},
	}}}
	sec := sectionFor(GenerateFence(d, "field"), "create")
	if !strings.Contains(sec, "1–40 chars of '[a-zA-Z0-9_]'") {
		t.Errorf("character class embedded in prose must render intact:\n%s", sec)
	}
	if strings.Contains(sec, "chars of ''") {
		t.Errorf("character class must not be deleted from the description:\n%s", sec)
	}
	if strings.Contains(sec, "enum: a-zA-Z0-9_") {
		t.Errorf("character class must not be fabricated into an enum tag:\n%s", sec)
	}
	if !strings.Contains(sec, "enum: checkbox | text") {
		t.Errorf("the genuine enum on the OTHER flag in the same command must still render:\n%s", sec)
	}
}

// equalStrings compares two string slices for the enum-parsing tests above.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// sectionFor returns the slice of out from "### <verb>" to the next "### " or end.
func sectionFor(out, verb string) string {
	start := strings.Index(out, "### "+verb)
	if start < 0 {
		return ""
	}
	rest := out[start+len("### "+verb):]
	if next := strings.Index(rest, "\n### "); next >= 0 {
		return out[start : start+len("### "+verb)+next]
	}
	return out[start:]
}

// markedRequired reports whether the row for flag carries the generator's
// required marker. The marker is the literal "(required)" token emitted right
// after the type (not any "required" prose that may appear in a flag's usage
// text, e.g. "Required by the validator").
func markedRequired(section, flag string) bool {
	for _, line := range strings.Split(section, "\n") {
		// Only inspect the flag's own bullet row (starts with "- `<flag>`").
		if strings.HasPrefix(strings.TrimSpace(line), "- `"+flag+"`") {
			return strings.Contains(line, "(required)")
		}
	}
	return false
}
