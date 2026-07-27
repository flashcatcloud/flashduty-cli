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
