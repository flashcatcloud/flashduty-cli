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
			if format == "json" {
				var envelope map[string]any
				if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &envelope); err != nil {
					t.Fatalf("bounded json is not an object: %v", err)
				}
				if _, ok := envelope["items"].([]any); !ok {
					t.Fatalf("bounded json lost the items array: %v", envelope)
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
}
