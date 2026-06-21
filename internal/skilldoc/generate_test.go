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
