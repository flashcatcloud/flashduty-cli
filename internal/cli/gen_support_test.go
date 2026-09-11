package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

func TestGenBindBodyAllowsNullForRequiredNullableField(t *testing.T) {
	req := new(struct {
		Value *bool `json:"value"`
	})

	if err := genBindBody(map[string]any{"value": nil}, req); err != nil {
		t.Fatalf("genBindBody required nullable field: %v", err)
	}
	if req.Value != nil {
		t.Fatalf("Value = %v, want nil", req.Value)
	}
}

// oversizedInsightRows returns n insight-incident rows whose fat description
// fields push any page of them well past compactListOutputLimit, plus the
// incident IDs in row order so a test can tell emitted rows from dropped ones.
func oversizedInsightRows(n int) ([]any, []string) {
	rows := make([]any, n)
	ids := make([]string, n)
	for i := range rows {
		ids[i] = fmt.Sprintf("inc-%024d", i)
		rows[i] = map[string]any{
			"incident_id":      ids[i],
			"title":            fmt.Sprintf("Database failover on db-%d", i),
			"severity":         "Critical",
			"channel_id":       12345,
			"channel_name":     "db-alerts",
			"description":      strings.Repeat(fmt.Sprintf("row %d root-cause detail ", i), 40),
			"seconds_to_ack":   42,
			"seconds_to_close": 3600,
			"notifications":    3,
		}
	}
	return rows, ids
}

// TestPrintGenericResultBoundsListEnvelope drives a generated list verb whose
// response is an items[] page envelope (insight incident-list): an oversized
// page must come back under the structured-output limit with the reduction
// announced on stderr and the envelope keys intact, in both structured
// formats.
func TestPrintGenericResultBoundsListEnvelope(t *testing.T) {
	for _, format := range []string{"json", "toon"} {
		t.Run(format, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			rows, ids := oversizedInsightRows(40)
			stub.data = map[string]any{
				"items":            rows,
				"total":            40,
				"has_next_page":    true,
				"search_after_ctx": "cursor-1",
			}

			out, stderrText, err := execCommandSplit("insight", "incident-list",
				"--start-time", "7d", "--end-time", "now", "--output-format", format)
			if err != nil {
				t.Fatalf("execCommandSplit: %v", err)
			}
			if len([]byte(out)) >= compactListOutputLimit {
				t.Errorf("bounded %s envelope is %d bytes, want <%d", format, len([]byte(out)), compactListOutputLimit)
			}
			if !strings.Contains(stderrText, "note: emitted") {
				t.Errorf("reduced %s page should announce itself on stderr, got:\n%s", format, stderrText)
			}
			// insight incident-list is a generated verb: it declares no narrowing
			// flag, so even though it carries a --limit, the note names none.
			if !strings.Contains(stderrText, "declares no flag that narrows the output") {
				t.Errorf("a generated verb should get the flagless note, got:\n%s", stderrText)
			}
			for _, flag := range []string{"--limit", "--fields"} {
				if strings.Contains(stderrText, flag) {
					t.Errorf("advice must not name %s on a verb that declares no narrowing flag, got:\n%s", flag, stderrText)
				}
			}
			// The first row survives intact; the last was dropped by the
			// prefix reduction.
			if !strings.Contains(out, ids[0]) {
				t.Errorf("bounded %s output lost leading row %q:\n%s", format, ids[0], out)
			}
			if strings.Contains(out, ids[len(ids)-1]) {
				t.Errorf("bounded %s output still contains trailing row %q", format, ids[len(ids)-1])
			}
			// The pagination envelope rides along with the bounded rows.
			for _, key := range []string{"total", "has_next_page", "search_after_ctx"} {
				if !strings.Contains(out, key) {
					t.Errorf("bounded %s output lost envelope key %q:\n%s", format, key, out)
				}
			}
			// The reduced page says so in the payload, not only on stderr —
			// with the row-withheld shape: emitted_rows present alongside
			// truncated. A values-clipped reduction carries truncated alone.
			if format == "toon" {
				// TOON renders the marker as envelope-level lines beside the
				// items block.
				if !strings.Contains(out, "truncated: true") || !strings.Contains(out, "emitted_rows:") {
					t.Errorf("bounded toon output lost the in-payload truncation marker:\n%s", out)
				}
			}
			if format == "json" {
				var envelope map[string]any
				if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &envelope); err != nil {
					t.Fatalf("bounded json is not an object: %v", err)
				}
				items, ok := envelope["items"].([]any)
				if !ok {
					t.Fatalf("bounded json lost the items array: %v", envelope)
				}
				if envelope["truncated"] != true {
					t.Errorf("bounded json truncated = %v, want true", envelope["truncated"])
				}
				emitted, ok := envelope["emitted_rows"].(float64)
				if !ok {
					t.Fatalf("bounded json lost emitted_rows: %v", envelope)
				}
				if int(emitted) != len(items) || int(emitted) >= len(rows) {
					t.Errorf("bounded json emitted_rows = %v, want len(items)=%d and < %d requested rows",
						envelope["emitted_rows"], len(items), len(rows))
				}
			}
		})
	}
}

// TestPrintGenericResultBoundsTopLevelArray drives a generated verb whose
// response is a bare top-level array (monit rule-list-basic): an oversized
// page must be bounded the same way as an items[] envelope.
func TestPrintGenericResultBoundsTopLevelArray(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	rows := make([]any, 40)
	for i := range rows {
		rows[i] = map[string]any{
			"id":           i + 1,
			"name":         strings.Repeat(fmt.Sprintf("rule %d ", i), 40),
			"folder_id":    100,
			"ds_type":      "prometheus",
			"cron_pattern": "0 * * * * *",
			"enabled":      true,
		}
	}
	stub.data = rows

	out, stderrText, err := execCommandSplit("monit", "rule-list-basic", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	if len([]byte(out)) >= compactListOutputLimit {
		t.Errorf("bounded top-level array is %d bytes, want <%d", len([]byte(out)), compactListOutputLimit)
	}
	if !strings.Contains(stderrText, "note: emitted") {
		t.Errorf("reduced page should announce itself on stderr, got:\n%s", stderrText)
	}
	// monit rule-list-basic is a generated verb whose --limit sizes the response
	// only alongside --include-descendants: it declares no narrowing flag, so the
	// note must not offer --limit.
	if !strings.Contains(stderrText, "declares no flag that narrows the output") {
		t.Errorf("a generated verb should get the flagless note, got:\n%s", stderrText)
	}
	if strings.Contains(stderrText, "--limit") {
		t.Errorf("advice must not name --limit; this verb declares no narrowing flag, got:\n%s", stderrText)
	}
	var decoded []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &decoded); err != nil {
		t.Fatalf("bounded top-level array is not a JSON array: %v\n%s", err, out)
	}
	if len(decoded) == 0 || len(decoded) >= len(rows) {
		t.Errorf("bounded array has %d rows, want a reduced page in [1, %d)", len(decoded), len(rows))
	}
}

// TestPrintGenericResultKeepsLargeIDsExact is the precision guard for the
// typed-slice round trip: an integer ID above 2^53 (channel_id) must reach
// the bounded output with every digit intact, where a float64 decode would
// have rounded it.
func TestPrintGenericResultKeepsLargeIDsExact(t *testing.T) {
	const bigChannelID = "9007199254740993" // 2^53 + 1
	for _, format := range []string{"json", "toon"} {
		t.Run(format, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			rows, _ := oversizedInsightRows(40)
			for _, row := range rows {
				row.(map[string]any)["channel_id"] = json.Number(bigChannelID)
			}
			stub.data = map[string]any{"items": rows, "total": 40}

			out, _, err := execCommandSplit("insight", "incident-list",
				"--start-time", "7d", "--end-time", "now", "--output-format", format)
			if err != nil {
				t.Fatalf("execCommandSplit: %v", err)
			}
			if !strings.Contains(out, bigChannelID) {
				t.Errorf("bounded %s output lost digits of channel_id %s", format, bigChannelID)
			}
			if strings.Contains(out, "9007199254740992") {
				t.Errorf("bounded %s output rounded channel_id to the nearest float64", format)
			}
		})
	}
}

// TestPrintGenericResultDetailNeverBounded: a detail-shaped single object is
// excluded from list bounding no matter its size — never reduced, never
// errored — and prints byte-identical to the direct printer.
func TestPrintGenericResultDetailNeverBounded(t *testing.T) {
	detail := &heuristicRow{Name: strings.Repeat("x", 40*1024), Count: 7}

	for _, f := range []output.Format{output.FormatJSON, output.FormatTOON} {
		var got, want bytes.Buffer
		if err := printGenericResult(structuredCtx(&got, f), detail); err != nil {
			t.Fatalf("%v oversized detail errored: %v", f, err)
		}
		if err := output.NewPrinter(f, false, &want).Print(detail, nil); err != nil {
			t.Fatalf("%v reference: %v", f, err)
		}
		if got.String() != want.String() {
			t.Errorf("oversized detail output changed for %v\n got:\n%s\nwant:\n%s", f, got.String(), want.String())
		}
		if len(got.Bytes()) < compactListOutputLimit {
			t.Errorf("detail payload should pass through unbounded, got %d bytes", len(got.Bytes()))
		}
	}
}

// TestPrintGenericResultShortenedRowStaysUTF8 covers the single-row-overflow
// path through a generated verb: one row too big on its own is shortened with
// a "..." marker, valid UTF-8, and a stderr note — never an unmarked clip.
func TestPrintGenericResultShortenedRowStaysUTF8(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"items": []any{map[string]any{
			"incident_id": "inc-1",
			"title":       strings.Repeat("数据库故障", 5000),
			"severity":    "Critical",
		}},
		"total": 1,
	}

	out, stderrText, err := execCommandSplit("insight", "incident-list",
		"--start-time", "7d", "--end-time", "now", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	if len([]byte(out)) >= compactListOutputLimit {
		t.Fatalf("shortened single row is %d bytes, want <%d", len([]byte(out)), compactListOutputLimit)
	}
	if !utf8.ValidString(out) || !strings.Contains(out, "...") {
		t.Fatalf("shortened row must retain valid UTF-8 and show the truncation marker")
	}
	if !strings.Contains(out, "inc-1") {
		t.Errorf("identifier field must survive shortening intact, got:\n%s", out)
	}
	if !strings.Contains(stderrText, "were shortened to fit") {
		t.Errorf("shortened row should announce the clipped fields on stderr, got:\n%s", stderrText)
	}
	// Clipped values are data loss too: the marker appears even though every
	// row was emitted — and it rides WITHOUT emitted_rows, because paging
	// cannot restore a clipped value (only a narrower --fields can), so the
	// withheld-rows continuation must not be read into this payload.
	var envelope map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &envelope); err != nil {
		t.Fatalf("shortened single row is not a JSON object: %v", err)
	}
	if envelope["truncated"] != true {
		t.Errorf("shortened single row lost the in-payload truncation marker: %v", envelope)
	}
	if _, ok := envelope["emitted_rows"]; ok {
		t.Errorf("a values-clipped page must not carry emitted_rows (it misreads as withheld rows): %v", envelope)
	}
	if items, _ := envelope["items"].([]any); len(items) != 1 {
		t.Errorf("clipping must not drop rows, got %d items", len(items))
	}
}

// TestPrintGenericResultCompleteEnvelopeUnmarked guards the marker's negative
// case: a page that fits carries no truncated/emitted_rows keys — the marker
// means "this page was reduced", not "this command supports reduction".
func TestPrintGenericResultCompleteEnvelopeUnmarked(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"items": []any{
			map[string]any{"incident_id": "inc-1", "title": "small", "severity": "Info"},
			map[string]any{"incident_id": "inc-2", "title": "smaller", "severity": "Info"},
		},
		"total":         2,
		"has_next_page": false,
	}

	out, stderrText, err := execCommandSplit("insight", "incident-list",
		"--start-time", "7d", "--end-time", "now", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	if strings.Contains(out, "truncated") || strings.Contains(out, "emitted_rows") {
		t.Errorf("within-budget envelope must not carry the truncation marker:\n%s", out)
	}
	if strings.Contains(stderrText, "note: emitted") {
		t.Errorf("within-budget envelope must not announce a reduction, got:\n%s", stderrText)
	}
}

// TestBoundedEnvelopeReductionFactsMatchPayload pins the reduction contract a
// --json consumer walks: emitted_rows appears only when rows were WITHHELD
// with every emitted value intact (a prefix reduction paging can repair), and a
// clipped value (it ends in "...") never rides under that marker, because
// paging cannot restore a clipped value — only a narrower --fields can. The
// fixture sweeps the first row across the band where the envelope re-fit
// switches between the two reductions, since that is where the reported facts
// and the printed payload can drift apart.
func TestBoundedEnvelopeReductionFactsMatchPayload(t *testing.T) {
	for _, titleBytes := range []int{15000, 15200, 15400, 15600, 15800, 16000, 16200, 16400, 16600, 16800} {
		t.Run(fmt.Sprintf("title=%d", titleBytes), func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			stub.data = map[string]any{
				"items": []any{
					map[string]any{"incident_id": "inc-1", "title": strings.Repeat("x", titleBytes), "severity": "Critical"},
					map[string]any{"incident_id": "inc-2", "title": "small follower", "severity": "Info"},
				},
				"total":         2,
				"has_next_page": true,
			}

			out, stderrText, err := execCommandSplit("insight", "incident-list",
				"--start-time", "7d", "--end-time", "now", "--output-format", "json")
			if err != nil {
				t.Fatalf("execCommandSplit: %v", err)
			}
			if len([]byte(out)) >= compactListOutputLimit {
				t.Errorf("bounded envelope is %d bytes, want <%d", len([]byte(out)), compactListOutputLimit)
			}
			var envelope map[string]any
			if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &envelope); err != nil {
				t.Fatalf("bounded output is not a JSON object: %v\n%s", err, out)
			}
			items, ok := envelope["items"].([]any)
			if !ok {
				t.Fatalf("bounded output lost the items array: %v", envelope)
			}
			_, rowsWithheld := envelope["emitted_rows"]
			clippedAt := ""
			for _, item := range items {
				row, ok := item.(map[string]any)
				if !ok {
					t.Fatalf("emitted row is not an object: %v", item)
				}
				for key, value := range row {
					if text, ok := value.(string); ok && strings.HasSuffix(text, "...") {
						clippedAt = fmt.Sprintf("%s (%d bytes emitted)", key, len(text))
					}
				}
			}
			mode := fmt.Sprintf("items=%d emitted_rows=%v clipped=%q", len(items), envelope["emitted_rows"], clippedAt)
			clipped := clippedAt != ""
			t.Log(mode)

			if envelope["truncated"] != true {
				t.Errorf("reduced envelope must carry truncated: %s", mode)
			}
			if !rowsWithheld && !clipped {
				t.Errorf("a marked payload was reduced one way or the other: %s", mode)
			}
			if rowsWithheld && clipped {
				t.Errorf("emitted_rows tells a paging walk the withheld rows are recoverable, but the payload also carries a clipped value that paging cannot restore: %s", mode)
			}
			if clipped {
				if strings.Contains(stderrText, "every value intact") {
					t.Errorf("a clipped value cannot be announced as \"every value intact\": %s\ngot: %s", mode, stderrText)
				}
				if !strings.Contains(stderrText, "were shortened to fit") {
					t.Errorf("a clipped value should be announced as shortened on stderr: %s\ngot: %s", mode, stderrText)
				}
				// The re-fit sizes the rows against the limit minus the
				// envelope's framing, but the note names the published cap:
				// an internal remainder tells the reader nothing they can act
				// on, and it contradicts the documented limit.
				if published := fmt.Sprintf("the %d-byte limit", compactListOutputLimit); !strings.Contains(stderrText, published) {
					t.Errorf("the shortened note should name the published limit (%s): %s\ngot: %s", published, mode, stderrText)
				}
			} else if !strings.Contains(stderrText, "every value intact") {
				t.Errorf("a rows-withheld reduction should announce intact values on stderr: %s\ngot: %s", mode, stderrText)
			}
		})
	}
}
