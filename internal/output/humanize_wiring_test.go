package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// These tests lock the wiring: each structured printer must route data through
// HumanizeTimestamps so a raw Unix integer never reaches the agent. They guard
// against a future printer being added without the conversion.

func TestJSONPrinter_HumanizesTimestamps(t *testing.T) {
	const ts = 1748419200
	var buf bytes.Buffer
	if err := (&JSONPrinter{w: &buf}).Print(map[string]any{"start_time": ts}, nil); err != nil {
		t.Fatalf("Print: %v", err)
	}
	if strings.Contains(buf.String(), "1748419200") {
		t.Fatalf("raw unix timestamp leaked into JSON output: %s", buf.String())
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}
	if inst := asInstant(t, got["start_time"]); inst != ts {
		t.Fatalf("start_time instant = %d, want %d", inst, ts)
	}
}

func TestTOONPrinter_HumanizesTimestamps(t *testing.T) {
	var buf bytes.Buffer
	if err := (&TOONPrinter{w: &buf}).Print(map[string]any{"start_time": 1748419200}, nil); err != nil {
		t.Fatalf("Print: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "1748419200") {
		t.Fatalf("raw unix timestamp leaked into TOON output: %s", out)
	}
	if !strings.Contains(out, "start_time") {
		t.Fatalf("expected start_time key in TOON output: %s", out)
	}
}
