package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestGenBindBodyAllowsNullForRequiredNullableField(t *testing.T) {
	req := new(struct {
		Value *bool `json:"value"`
	})

	if err := genBindBody(io.Discard, map[string]any{"value": nil}, req); err != nil {
		t.Fatalf("genBindBody required nullable field: %v", err)
	}
	if req.Value != nil {
		t.Fatalf("Value = %v, want nil", req.Value)
	}
}

func TestGenBindBodyWarnsOnFieldsTheRequestCannotCarry(t *testing.T) {
	type target struct {
		Kind  string          `json:"kind"`
		Query json.RawMessage `json:"query,omitempty"`
	}
	type embedded struct {
		Team int64 `json:"team_id,omitempty"`
	}
	req := new(struct {
		embedded
		Name    string            `json:"name"`
		Targets []target          `json:"targets,omitempty"`
		Labels  map[string]target `json:"labels,omitempty"`
		Page    int64             `url:"p,omitempty"`
	})
	body := map[string]any{
		"name":    "r",
		"NAME":    "folded",
		"team_id": 1,
		"p":       2,
		"extra":   true,
		"targets": []any{
			map[string]any{"kind": "x", "purpose": "a", "query": map[string]any{"anything": 1}},
			map[string]any{"kind": "y", "purpose": "b"},
		},
		"labels": map[string]any{"k": map[string]any{"kind": "z", "note": "c"}},
	}

	var warn bytes.Buffer
	if err := genBindBody(&warn, body, req); err != nil {
		t.Fatalf("genBindBody: %v", err)
	}
	want := "warning: --data fields not sent, unknown to this CLI version: extra, labels.k.note, targets[].purpose\n"
	if warn.String() != want {
		t.Fatalf("warning = %q, want %q", warn.String(), want)
	}

	warn.Reset()
	if err := genBindBody(&warn, map[string]any{"name": "r", "targets": []any{map[string]any{"kind": "x"}}}, req); err != nil {
		t.Fatalf("genBindBody: %v", err)
	}
	if warn.Len() != 0 {
		t.Fatalf("warning on a fully bound body: %q", warn.String())
	}
}

// oversizedInsightRows returns n insight-incident rows whose fat description
// fields push a page of 40 well past 16 KiB, plus the incident IDs in row
// order so a test can tell emitted rows from dropped ones.
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

// TestPrintGenericResultListEnvelopeComplete drives a generated list verb whose
// response is an items[] page envelope (insight incident-list): a page far
// past 16 KiB comes back whole in both structured formats — every row, the
// envelope keys, no in-payload marker, no stderr note about a reduction.
func TestPrintGenericResultListEnvelopeComplete(t *testing.T) {
	for _, format := range []string{"json", "toon"} {
		t.Run(format, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			rows, ids := oversizedInsightRows(40)
			stub.data = map[string]any{
				"items":            rows,
				"total":            40,
				"has_next_page":    false,
				"search_after_ctx": "cursor-1",
			}

			out, stderrText, err := execCommandSplit("insight", "incident-list",
				"--start-time", "7d", "--end-time", "now", "--output-format", format)
			if err != nil {
				t.Fatalf("execCommandSplit: %v", err)
			}
			if len(out) < 16*1024*2 {
				t.Fatalf("fixture too small to exercise a large page: %d bytes", len(out))
			}
			for _, id := range ids {
				if !strings.Contains(out, id) {
					t.Errorf("%s output lost row %q", format, id)
				}
			}
			for _, key := range []string{"total", "has_next_page", "search_after_ctx"} {
				if !strings.Contains(out, key) {
					t.Errorf("%s output lost envelope key %q", format, key)
				}
			}
			for _, marker := range []string{"truncated", "emitted_rows", "..."} {
				if strings.Contains(out, marker) {
					t.Errorf("%s output carries reduction marker %q", format, marker)
				}
			}
			assertNoReductionNote(t, stderrText)
		})
	}
}

// TestPrintGenericResultTopLevelArrayComplete: a generated verb whose response
// is a bare top-level array (monit rule-list-basic) prints every row of a page
// far past 16 KiB, with no stderr note about a reduction.
func TestPrintGenericResultTopLevelArrayComplete(t *testing.T) {
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
	var decoded []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &decoded); err != nil {
		t.Fatalf("top-level array is not a JSON array: %v\n%s", err, out)
	}
	if len(decoded) != len(rows) {
		t.Errorf("got %d rows, want all %d", len(decoded), len(rows))
	}
	for i, row := range decoded {
		if want := rows[i].(map[string]any)["name"]; row["name"] != want {
			t.Errorf("row %d name altered", i)
		}
	}
	assertNoReductionNote(t, stderrText)
}

// TestPrintGenericResultKeepsLargeIDsExact is the precision guard for the
// --fields round trip through decoded JSON: an integer ID above 2^53
// (channel_id) must reach the projected output with every digit intact, where
// a float64 decode would have rounded it.
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
				"--start-time", "7d", "--end-time", "now", "--fields", "incident_id,channel_id", "--output-format", format)
			if err != nil {
				t.Fatalf("execCommandSplit: %v", err)
			}
			if !strings.Contains(out, bigChannelID) {
				t.Errorf("projected %s output lost digits of channel_id %s", format, bigChannelID)
			}
			if strings.Contains(out, "9007199254740992") {
				t.Errorf("projected %s output rounded channel_id to the nearest float64", format)
			}
		})
	}
}
