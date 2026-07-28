package skilldoc

import (
	"strings"
	"testing"
)

// The three Long fixtures below reproduce cligen's exact header phrasing for
// each of the three envelope shapes it can document (verified against the
// real generated output in internal/cli/zz_generated_*.go and
// zz_generated_response_help.go — see internal/skilldoc/generate.go's
// responseShapeLine doc comment). Only the header wording is load-bearing;
// the field content below it is a synthetic fixture, same convention as
// generatorDump's Request-fields block.

const objectShapeLong = `Get schedule info.

Request fields:
  --schedule-id int (required) — Schedule ID.

Response fields ('data' envelope is unwrapped — these fields are at the top level):
  - schedule_id (integer) (required) — Schedule ID.
  - name (string) — Schedule display name.
  - cur_oncall (object) — Current on-call group, or null when nobody is on-call.
    - group_name (string) — Group display name.
`

const topLevelArrayShapeLong = `List agents.

Response fields (this command's ` + "`--json`" + ` is a TOP-LEVEL array of these row objects — pipe ` + "`jq '.[]'`" + `, NOT ` + "`.items[]`" + `):
  - agent_id (string) (required) — Unique agent ID.
  - status (string) — Agent status. [enabled, disabled]
`

const itemsWrappedShapeLong = `List schedules.

Response fields ('data' envelope is unwrapped — rows are nested under items[]; pipe 'jq '.items[]'', NOT '.data.items[]'):
  - items (array<object>) (required) — Schedules on this page.
    - schedule_id (integer) (required) — Schedule ID.
    - name (string) — Schedule display name.
    - cur_oncall (object) — Current on-call group, or null when nobody is on-call.
      - group_name (string) — Group display name.
`

const noResponseBlockLong = `Delete schedules.

Request fields:
  --schedule-id int (required) — Schedule ID.
`

// itemsWrappedWithPaginationSiblingsShapeLong reproduces the real shape of
// `fduty insight incident-list` (internal/cli/zz_generated_analytics.go): a
// page-wrapper response whose top level carries "items" ALONGSIDE scalar
// pagination metadata — has_next_page, search_after_ctx, total — not just
// "items" alone. cligen's listEnvelope requires every non-"items" top-level
// field here to be a scalar, so these three are always pagination metadata,
// never more row data.
const itemsWrappedWithPaginationSiblingsShapeLong = `List insight incidents.

Response fields ('data' envelope is unwrapped — rows are nested under items[]; pipe 'jq '.items[]'', NOT '.data.items[]'):
  - has_next_page (boolean)
  - items (array<object>)
    - incident_id (string)
    - title (string)
  - search_after_ctx (string) — Cursor token to fetch the next page. Pass it back in the next request's 'search_after_ctx'.
  - total (integer) — Total matching incidents.
`

// driftedWrapperHeaderLong simulates a future cligen wording change to the
// items[]-wrapper header that no longer contains the literal substring
// "nested under items[]" this parser keys on (e.g. cligen's header text was
// reworded from "rows are nested under items[]" to "rows appear inside
// items[]"). The response is STILL genuinely items[]-wrapped — same shape as
// itemsWrappedShapeLong above, just described differently — so this must
// never be reported as "single object".
const driftedWrapperHeaderLong = `List schedules.

Response fields ('data' envelope is unwrapped — rows appear inside items[]):
  - items (array<object>) (required) — Schedules on this page.
    - schedule_id (integer) (required) — Schedule ID.
`

// driftedWrapperHeaderDocsLong is the same drift scenario but with cligen's
// "docs" wire name (listEnvelope in internal/cmd/cligen/main.go recognizes
// items/docs/list interchangeably as a list-envelope field) instead of
// "items", to prove the guard isn't hardcoded to one wire name.
const driftedWrapperHeaderDocsLong = `List reports.

Response fields ('data' envelope is unwrapped — rows appear inside docs[]):
  - docs (array<object>) (required) — Reports on this page.
    - report_id (integer) (required) — Report ID.
`

// singleArrayFieldObjectLong is a GENUINE single-object response (per
// cligen's own header, and correctly so — this mirrors the real
// Automations.RuleReadList shape, whose sole top-level field is "rules", not
// one of cligen's own list-envelope wire names) whose one top-level field
// happens to be an array. The guard must NOT suppress this: only the exact
// wrapper wire names (items/docs/list) are a drift signal, not "any object
// with one array field".
const singleArrayFieldObjectLong = `List automation rules.

Response fields ('data' envelope is unwrapped — these fields are at the top level):
  - rules (array<object>) (required) — Automation rules.
    - rule_id (string) (required) — Rule ID.
`

func TestResponseShapeLine_Object(t *testing.T) {
	got := responseShapeLine(objectShapeLong)
	if !strings.Contains(got, "single object") {
		t.Errorf("object shape not detected:\n%s", got)
	}
	if strings.Contains(got, "TOP-LEVEL array") || strings.Contains(got, "items") {
		t.Errorf("object shape must not mention array/items wrapper phrasing:\n%s", got)
	}
	// Only the TOP-LEVEL fields (2-space indent) are named — not the nested
	// cur_oncall.group_name one level deeper.
	for _, want := range []string{"schedule_id (integer)", "name (string)", "cur_oncall (object)"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing top-level field %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "group_name") {
		t.Errorf("nested field group_name must NOT appear at the object's own level:\n%s", got)
	}
}

func TestResponseShapeLine_TopLevelArray(t *testing.T) {
	got := responseShapeLine(topLevelArrayShapeLong)
	if !strings.Contains(got, "TOP-LEVEL array") {
		t.Errorf("array shape not detected:\n%s", got)
	}
	if !strings.Contains(got, "jq '.[]'") || !strings.Contains(got, "NOT `.items[]`") {
		t.Errorf("array shape must warn against `.items[]`:\n%s", got)
	}
	for _, want := range []string{"agent_id (string)", "status (string)"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing row field %q:\n%s", want, got)
		}
	}
}

func TestResponseShapeLine_ItemsWrapped(t *testing.T) {
	got := responseShapeLine(itemsWrappedShapeLong)
	if !strings.Contains(got, "items: [...]") || !strings.Contains(got, "page wrapper") {
		t.Errorf("items[] wrapper shape not detected:\n%s", got)
	}
	if !strings.Contains(got, "jq '.items[]'") {
		t.Errorf("items[] shape must point at `.items[]`, not top-level `.[]`:\n%s", got)
	}
	// The row fields are the ones nested UNDER items (4-space indent), not the
	// bare "items" wrapper key itself.
	for _, want := range []string{"schedule_id (integer)", "name (string)", "cur_oncall (object)"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing row field %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "- fields: items") || strings.HasPrefix(strings.TrimSpace(strings.SplitN(got, "fields:", 2)[1]), "items (") {
		t.Errorf("wrapper key `items` itself must not be listed as a row field:\n%s", got)
	}
	if strings.Contains(got, "group_name") {
		t.Errorf("field nested two levels deep (cur_oncall.group_name) must NOT appear:\n%s", got)
	}
}

// TestResponseShapeLine_ItemsWrappedNamesPaginationSiblings covers the real
// `insight incident-list` shape: the wrapper's top level carries "items"
// ALONGSIDE scalar pagination metadata (has_next_page, search_after_ctx,
// total). Those siblings must be named in the wrapper descriptor itself —
// an agent told to paginate via `--search-after-ctx` needs to know the
// returned cursor lives at `.search_after_ctx` next to `.items`, not silently
// dropped because the parser only ever looked one level under "items".
func TestResponseShapeLine_ItemsWrappedNamesPaginationSiblings(t *testing.T) {
	got := responseShapeLine(itemsWrappedWithPaginationSiblingsShapeLong)
	if !strings.Contains(got, "items: [...]") || !strings.Contains(got, "page wrapper") {
		t.Errorf("items[] wrapper shape not detected:\n%s", got)
	}
	// The siblings are named inside the wrapper descriptor, in the order
	// cligen documented them (source order, already alphabetical here).
	if !strings.Contains(got, "{items: [...], has_next_page, search_after_ctx, total}") {
		t.Errorf("wrapper descriptor must name the pagination siblings:\n%s", got)
	}
	// The row fields (nested under items) are still named, under a label that
	// disambiguates them from the siblings just named above.
	if !strings.Contains(got, "items fields: incident_id (string); title (string)") {
		t.Errorf("row fields must still be listed under an unambiguous label:\n%s", got)
	}
	// The siblings must not ALSO appear duplicated in the row-field list.
	fieldsPart := strings.SplitN(got, "items fields:", 2)[1]
	for _, sib := range []string{"has_next_page", "search_after_ctx", "total"} {
		if strings.Contains(fieldsPart, sib) {
			t.Errorf("pagination sibling %q must not be duplicated in the row-field list:\n%s", sib, got)
		}
	}
}

// TestResponseShapeLine_ItemsWrappedNoSiblingsUnchanged proves the sibling
// feature is additive: a wrapper with no pagination siblings beyond "items"
// (itemsWrappedShapeLong) renders the exact same wrapper descriptor as
// before — no trailing ", " artifact from an empty sibling list.
func TestResponseShapeLine_ItemsWrappedNoSiblingsUnchanged(t *testing.T) {
	got := responseShapeLine(itemsWrappedShapeLong)
	if !strings.Contains(got, "`{items: [...]}` page wrapper") {
		t.Errorf("wrapper with no siblings must render the bare descriptor, got:\n%s", got)
	}
}

func TestResponseShapeLine_NoBlockIsEmpty(t *testing.T) {
	if got := responseShapeLine(noResponseBlockLong); got != "" {
		t.Errorf("command with no Response fields block must yield no response line, got:\n%s", got)
	}
}

// TestResponseShapeLine_DriftedWrapperHeaderYieldsNothingNotWrongClaim covers
// the failure mode a reviewer traced: if cligen's items[]-wrapper header
// wording ever drifts away from the literal "nested under items[]" substring
// this parser matches on, the shape must NOT silently fall through to
// "single object" — that would be the exact confidently-wrong claim this
// whole feature exists to prevent. It must instead emit nothing. Covers both
// wrapper wire names cligen's listEnvelope recognizes: "items" and "docs".
func TestResponseShapeLine_DriftedWrapperHeaderYieldsNothingNotWrongClaim(t *testing.T) {
	for _, tc := range []struct {
		name string
		long string
	}{
		{"items", driftedWrapperHeaderLong},
		{"docs", driftedWrapperHeaderDocsLong},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := responseShapeLine(tc.long)
			if got != "" {
				t.Errorf("drifted wrapper header must yield no response line (not a false 'single object' claim), got:\n%s", got)
			}
		})
	}
}

// TestResponseShapeLine_SingleArrayFieldObjectIsNotSuppressed proves the
// drift guard is scoped to cligen's own wrapper wire names (items/docs/list),
// not "any object shape with exactly one array field" — a real, correctly
// classified single-object response (mirroring Automations.RuleReadList,
// whose sole field is "rules") must still render normally.
func TestResponseShapeLine_SingleArrayFieldObjectIsNotSuppressed(t *testing.T) {
	got := responseShapeLine(singleArrayFieldObjectLong)
	if !strings.Contains(got, "single object") {
		t.Errorf("a genuine single-object response with one array field must not be suppressed by the wrapper-drift guard:\n%s", got)
	}
	if !strings.Contains(got, "rules (array<object>)") {
		t.Errorf("missing the rules field:\n%s", got)
	}
}

// TestGenerateFence_InjectsResponseShapePerVerb is the fence-level integration
// check: a group with one verb of each shape must surface a "- response: "
// line in that verb's own section, and a verb with no documented response
// must surface none (not a blank placeholder line).
func TestGenerateFence_InjectsResponseShapePerVerb(t *testing.T) {
	d := Dump{Commands: []Command{
		{Path: "widget info", Group: "widget", Short: "Get widget", Use: "info", Long: objectShapeLong},
		{Path: "widget list", Group: "widget", Short: "List widgets", Use: "list", Long: itemsWrappedShapeLong},
		{Path: "widget delete", Group: "widget", Short: "Delete widget", Use: "delete", Long: noResponseBlockLong},
	}}
	out := GenerateFence(d, "widget")

	infoSec := sectionFor(out, "info")
	if !strings.Contains(infoSec, "- response: single object") {
		t.Errorf("info section missing object response line:\n%s", infoSec)
	}
	listSec := sectionFor(out, "list")
	if !strings.Contains(listSec, "- response:") || !strings.Contains(listSec, "page wrapper") {
		t.Errorf("list section missing items[] response line:\n%s", listSec)
	}
	deleteSec := sectionFor(out, "delete")
	if strings.Contains(deleteSec, "- response:") {
		t.Errorf("delete section must not fabricate a response line when Long documents none:\n%s", deleteSec)
	}
}

// TestGenerateFence_DedupesIdenticalResponseShapesWithinGroup covers the fence
// bloat a group of create/get/update-style verbs commonly produces: several
// commands documenting the exact same resource repeat an identical ~700-800
// byte field list once per verb. The first occurrence must keep the full
// list; every later BYTE-IDENTICAL occurrence in the same group must instead
// reference the first by name; a genuinely different shape must never be
// touched.
func TestGenerateFence_DedupesIdenticalResponseShapesWithinGroup(t *testing.T) {
	d := Dump{Commands: []Command{
		{Path: "widget create", Group: "widget", Short: "Create widget", Use: "create", Long: objectShapeLong},
		{Path: "widget get", Group: "widget", Short: "Get widget", Use: "get <widget-id>", Long: objectShapeLong},
		{Path: "widget list", Group: "widget", Short: "List widgets", Use: "list", Long: itemsWrappedShapeLong},
	}}
	out := GenerateFence(d, "widget")

	createSec := sectionFor(out, "create")
	if !strings.Contains(createSec, "- response: single object") || !strings.Contains(createSec, "schedule_id (integer)") {
		t.Errorf("first occurrence must keep the full field list:\n%s", createSec)
	}
	if strings.Contains(createSec, "same shape as") {
		t.Errorf("first occurrence must not reference itself:\n%s", createSec)
	}

	getSec := sectionFor(out, "get")
	if !strings.Contains(getSec, "- response: same shape as `create` above") {
		t.Errorf("later byte-identical shape must reference the first occurrence by name, got:\n%s", getSec)
	}
	if strings.Contains(getSec, "schedule_id (integer)") {
		t.Errorf("deduped occurrence must not repeat the field list:\n%s", getSec)
	}

	listSec := sectionFor(out, "list")
	if !strings.Contains(listSec, "page wrapper") || !strings.Contains(listSec, "schedule_id (integer)") {
		t.Errorf("a genuinely different shape must render in full, not be deduped:\n%s", listSec)
	}
	if strings.Contains(listSec, "same shape as") {
		t.Errorf("a genuinely different shape must not be treated as a duplicate:\n%s", listSec)
	}

	if out != GenerateFence(d, "widget") {
		t.Errorf("GenerateFence dedup must be deterministic")
	}
}

// TestGenerateFence_ResponseShapeDedupIsPerGroupNotGlobal proves the dedup map
// is scoped to one GenerateFence call: two unrelated groups documenting the
// identical response shape must each render their own full field list in
// full — a card must stay self-contained and never reference a command in
// another card.
func TestGenerateFence_ResponseShapeDedupIsPerGroupNotGlobal(t *testing.T) {
	d := Dump{Commands: []Command{
		{Path: "widget create", Group: "widget", Short: "Create widget", Use: "create", Long: objectShapeLong},
		{Path: "gadget create", Group: "gadget", Short: "Create gadget", Use: "create", Long: objectShapeLong},
	}}

	widgetSec := sectionFor(GenerateFence(d, "widget"), "create")
	if !strings.Contains(widgetSec, "schedule_id (integer)") || strings.Contains(widgetSec, "same shape as") {
		t.Errorf("widget's create must render its own full field list:\n%s", widgetSec)
	}
	gadgetSec := sectionFor(GenerateFence(d, "gadget"), "create")
	if !strings.Contains(gadgetSec, "schedule_id (integer)") || strings.Contains(gadgetSec, "same shape as") {
		t.Errorf("gadget's create must render its own full field list, not reference widget's card:\n%s", gadgetSec)
	}
}
