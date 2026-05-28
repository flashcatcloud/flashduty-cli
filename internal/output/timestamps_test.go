package output

import (
	"testing"
	"time"
)

// asInstant parses an RFC3339 string and returns its Unix seconds, failing the
// test if the value isn't a valid RFC3339 timestamp. Keeps assertions
// timezone-independent: we check the rendered instant, not its wall-clock text.
func asInstant(t *testing.T, v any) int64 {
	t.Helper()
	s, ok := v.(string)
	if !ok {
		t.Fatalf("expected RFC3339 string, got %T (%v)", v, v)
	}
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("value %q is not RFC3339: %v", s, err)
	}
	return parsed.Unix()
}

func TestHumanizeTimestamps_ConvertsSeconds(t *testing.T) {
	const ts = 1748419200 // 2025-05-28T08:00:00Z
	got := HumanizeTimestamps(map[string]any{"start_time": ts})
	m := got.(map[string]any)
	if inst := asInstant(t, m["start_time"]); inst != ts {
		t.Fatalf("start_time instant = %d, want %d", inst, ts)
	}
}

func TestHumanizeTimestamps_ConvertsMillis(t *testing.T) {
	const sec = 1748419200
	got := HumanizeTimestamps(map[string]any{"created_at": int64(sec) * 1000})
	m := got.(map[string]any)
	if inst := asInstant(t, m["created_at"]); inst != sec {
		t.Fatalf("created_at instant = %d, want %d", inst, sec)
	}
}

func TestHumanizeTimestamps_DetectsByFieldName(t *testing.T) {
	const ts = 1748419200
	in := map[string]any{
		"start_time":      ts,
		"ack_time":        ts,
		"close_time":      ts,
		"assigned_at":     ts,
		"acknowledged_at": ts,
		"timestamp":       ts,
		"trigger_time":    ts,
		"end_time":        ts,
	}
	m := HumanizeTimestamps(in).(map[string]any)
	for k := range in {
		if inst := asInstant(t, m[k]); inst != ts {
			t.Fatalf("%s instant = %d, want %d", k, inst, ts)
		}
	}
}

func TestHumanizeTimestamps_LeavesIDFieldsAlone(t *testing.T) {
	// All large enough to convert by magnitude — proves the field-name
	// exclusion (not the magnitude guard) is what keeps these numeric.
	in := map[string]any{
		"updated_by":  int64(1748419200),
		"created_by":  int64(1748419200),
		"timeline_id": int64(1748419200),
		"channel_id":  int64(1748419200),
		"channel_ids": []any{int64(1748419200)},
	}
	m := HumanizeTimestamps(in).(map[string]any)
	for k := range in {
		if _, isStr := m[k].(string); isStr {
			t.Fatalf("%s was converted to a string but is an ID field", k)
		}
	}
}

func TestHumanizeTimestamps_NilPassesThrough(t *testing.T) {
	if got := HumanizeTimestamps(nil); got != nil {
		t.Fatalf("HumanizeTimestamps(nil) = %v, want nil", got)
	}
}

func TestHumanizeTimestamps_LeavesSmallDurationsAlone(t *testing.T) {
	// A *_time-named field holding a small value is a duration, not an absolute
	// unix timestamp — must not be rendered as a 1970 date.
	in := map[string]any{"snooze_time": int64(300)}
	m := HumanizeTimestamps(in).(map[string]any)
	if _, isStr := m["snooze_time"].(string); isStr {
		t.Fatalf("snooze_time=300 was converted; small values must stay numeric")
	}
}

func TestHumanizeTimestamps_SkipsZero(t *testing.T) {
	in := map[string]any{"ack_time": 0}
	m := HumanizeTimestamps(in).(map[string]any)
	if _, isStr := m["ack_time"].(string); isStr {
		t.Fatalf("ack_time=0 (omitted) must not be rendered as a date")
	}
}

func TestHumanizeTimestamps_RecursesNestedAndSlices(t *testing.T) {
	const ts = 1748419200
	in := map[string]any{
		"incidents": []any{
			map[string]any{
				"start_time": ts,
				"labels":     map[string]any{"close_time": ts},
			},
		},
	}
	m := HumanizeTimestamps(in).(map[string]any)
	inc := m["incidents"].([]any)[0].(map[string]any)
	if inst := asInstant(t, inc["start_time"]); inst != ts {
		t.Fatalf("nested start_time instant = %d, want %d", inst, ts)
	}
	if inst := asInstant(t, inc["labels"].(map[string]any)["close_time"]); inst != ts {
		t.Fatalf("deeply nested close_time instant = %d, want %d", inst, ts)
	}
}

func TestHumanizeTimestamps_ConvertsTypedStruct(t *testing.T) {
	// Real SDK results are structs, not maps — the helper must humanize them too.
	type incident struct {
		Title     string `json:"title"`
		StartTime int64  `json:"start_time"`
		UpdatedBy int64  `json:"updated_by"`
	}
	const ts = 1748419200
	m := HumanizeTimestamps(incident{Title: "db down", StartTime: ts, UpdatedBy: 7}).(map[string]any)
	if inst := asInstant(t, m["start_time"]); inst != ts {
		t.Fatalf("struct start_time instant = %d, want %d", inst, ts)
	}
	if _, isStr := m["updated_by"].(string); isStr {
		t.Fatalf("struct updated_by must remain numeric")
	}
	if m["title"] != "db down" {
		t.Fatalf("title = %v, want \"db down\"", m["title"])
	}
}
