package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/flashcatcloud/go-flashduty"
)

// row carries a typed go-flashduty Timestamp field so we can prove the structured
// printers render it as RFC3339 rather than a raw epoch integer.
type row struct {
	StartTime flashduty.Timestamp `json:"start_time" toon:"start_time"`
}

// TestStructuredTimeIsRFC3339 is the regression guard for the typed-timestamp
// SDK adoption: both the JSON and TOON printers must serialize a
// go-flashduty Timestamp as a human-/LLM-readable RFC3339 string, never as the
// opaque Unix epoch integer.
func TestStructuredTimeIsRFC3339(t *testing.T) {
	// 2026-05-28T08:00:00Z — fixed so we can assert the raw epoch is absent.
	const epochSec = 1779955200
	data := row{StartTime: flashduty.Timestamp(epochSec)}

	cases := []struct {
		name   string
		format Format
	}{
		{name: "JSON", format: FormatJSON},
		{name: "TOON", format: FormatTOON},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := NewPrinter(tc.format, false, &buf)
			if err := p.Print(data, nil); err != nil {
				t.Fatalf("%s Print returned error: %v", tc.name, err)
			}
			out := buf.String()

			// RFC3339 marker (date-time separator) and the year must be present.
			if !strings.Contains(out, "T") {
				t.Errorf("%s output missing RFC3339 'T' separator: %q", tc.name, out)
			}
			if !strings.Contains(out, "2026") {
				t.Errorf("%s output missing year 2026: %q", tc.name, out)
			}
			// The raw epoch integer must NOT appear — that's the bug we fixed.
			if strings.Contains(out, "1779955200") {
				t.Errorf("%s output leaked raw epoch integer: %q", tc.name, out)
			}
		})
	}
}
