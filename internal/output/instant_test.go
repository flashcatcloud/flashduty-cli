package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/flashcatcloud/go-flashduty"
)

// instantFixture mimics an SDK response record with timestamp fields in both
// states plus a plain integer that happens to be 0, so the tests prove the
// transform touches only timestamps.
type instantFixture struct {
	ID        int64                    `json:"id"`
	Title     string                   `json:"title"`
	StartTime flashduty.Timestamp      `json:"start_time"`
	EndTime   flashduty.Timestamp      `json:"end_time"`
	SeenAtMs  flashduty.TimestampMilli `json:"seen_at_ms"`
}

func decodeObject(t *testing.T, data any) map[string]any {
	t.Helper()
	out, err := json.Marshal(NullUnsetInstants(data))
	if err != nil {
		t.Fatalf("marshal transformed value: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("transformed value did not marshal to valid JSON: %v", err)
	}
	return got
}

func TestNullUnsetInstants_ZeroBecomesNull(t *testing.T) {
	t.Parallel()

	got := decodeObject(t, instantFixture{ID: 0, Title: "active alert"})

	for _, field := range []string{"start_time", "end_time", "seen_at_ms"} {
		v, ok := got[field]
		if !ok {
			t.Errorf("field %q missing from output", field)
			continue
		}
		if v != nil {
			t.Errorf("field %q = %#v, want nil (JSON null)", field, v)
		}
	}
	// A non-timestamp zero integer must stay the number 0.
	if v, ok := got["id"].(float64); !ok || v != 0 {
		t.Errorf("id = %#v, want numeric 0", got["id"])
	}
}

func TestNullUnsetInstants_SetValueKeepsRFC3339(t *testing.T) {
	t.Parallel()

	const epochSec = 1779955200 // 2026-05-28T08:00:00Z
	got := decodeObject(t, instantFixture{
		StartTime: flashduty.Timestamp(epochSec),
		EndTime:   flashduty.Timestamp(epochSec + 60),
		SeenAtMs:  flashduty.TimestampMilli(epochSec * 1000),
	})

	for _, field := range []string{"start_time", "end_time", "seen_at_ms"} {
		s, ok := got[field].(string)
		if !ok {
			t.Errorf("field %q = %#v, want RFC3339 string", field, got[field])
			continue
		}
		if _, err := time.Parse(time.RFC3339, s); err != nil {
			t.Errorf("field %q = %q, not RFC3339: %v", field, s, err)
		}
	}
}

// TestNullUnsetInstants_FieldOrderPreserved guards against the transform
// silently switching JSON objects to alphabetical key order: struct field
// order is part of the CLI's documented output shape.
func TestNullUnsetInstants_FieldOrderPreserved(t *testing.T) {
	t.Parallel()

	// All instants set: the transform must be byte-invisible.
	fixture := instantFixture{StartTime: 1, EndTime: 2, SeenAtMs: 3}
	plain, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatalf("marshal original: %v", err)
	}
	transformed, err := json.MarshalIndent(NullUnsetInstants(fixture), "", "  ")
	if err != nil {
		t.Fatalf("marshal transformed: %v", err)
	}
	if string(plain) != string(transformed) {
		t.Errorf("byte shape changed for a value without unset instants:\noriginal:\n%s\ntransformed:\n%s", plain, transformed)
	}
}

// TestNullUnsetInstants_ThroughContainers covers the shapes commands actually
// print: slices of SDK rows, pointers to a detail record, and the
// map[string]any rows produced by --fields projection.
func TestNullUnsetInstants_ThroughContainers(t *testing.T) {
	t.Parallel()

	const epochSec = 1779955200
	active := flashduty.AlertItem{AlertID: "a1", Title: "firing"}
	recovered := flashduty.AlertItem{AlertID: "a2", Title: "recovered", EndTime: flashduty.Timestamp(epochSec)}

	t.Run("slice of SDK rows", func(t *testing.T) {
		out, err := json.Marshal(NullUnsetInstants([]flashduty.AlertItem{active, recovered}))
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var rows []map[string]any
		if err := json.Unmarshal(out, &rows); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if rows[0]["end_time"] != nil {
			t.Errorf("active alert end_time = %#v, want nil", rows[0]["end_time"])
		}
		if _, ok := rows[1]["end_time"].(string); !ok {
			t.Errorf("recovered alert end_time = %#v, want RFC3339 string", rows[1]["end_time"])
		}
	})

	t.Run("pointer to detail record", func(t *testing.T) {
		got := decodeObject(t, &active)
		if got["end_time"] != nil {
			t.Errorf("end_time = %#v, want nil", got["end_time"])
		}
	})

	t.Run("projected map rows", func(t *testing.T) {
		rows := []map[string]any{
			{"alert_id": "a1", "end_time": flashduty.Timestamp(0)},
			{"alert_id": "a2", "end_time": flashduty.Timestamp(epochSec)},
		}
		out, err := json.Marshal(NullUnsetInstants(rows))
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got []map[string]any
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if got[0]["end_time"] != nil {
			t.Errorf("projected unset end_time = %#v, want nil", got[0]["end_time"])
		}
		if _, ok := got[1]["end_time"].(string); !ok {
			t.Errorf("projected set end_time = %#v, want RFC3339 string", got[1]["end_time"])
		}
	})
}

// TestJSONPrinter_UnsetInstantIsNull is the end-to-end guard: --json output
// never emits the bare integer 0 for an unset timestamp.
func TestJSONPrinter_UnsetInstantIsNull(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p := &JSONPrinter{w: &buf}
	if err := p.Print(instantFixture{Title: "firing"}, nil); err != nil {
		t.Fatalf("Print: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, `"end_time": 0`) {
		t.Errorf("unset end_time rendered as bare integer 0: %s", out)
	}
	if !strings.Contains(out, `"end_time": null`) {
		t.Errorf("unset end_time not rendered as null: %s", out)
	}
}
