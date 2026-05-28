package output

import (
	"bytes"
	"strings"
	"testing"
)

// NewPrinter(FormatTOON) returns a *TOONPrinter, and its output is compact —
// for a uniform array it must NOT repeat the field keys on every row the way
// JSON does. That key-deduplication is the whole point of TOON.
func TestNewPrinter_TOONMode(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(FormatTOON, false, &buf)

	if _, ok := p.(*TOONPrinter); !ok {
		t.Fatalf("expected *TOONPrinter, got %T", p)
	}

	data := []testItem{{Name: "alert-1"}, {Name: "alert-2"}}
	if err := p.Print(data, testColumns()); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "alert-1") || !strings.Contains(out, "alert-2") {
		t.Errorf("TOON output missing data values:\n%s", out)
	}
	// TOON encodes a uniform array as a header + rows, so the field name
	// appears far fewer times than the row count. JSON would repeat it per row.
	if n := strings.Count(out, "name"); n >= len(data) {
		t.Errorf("TOON output repeats key %d times for %d rows; expected key dedup:\n%s", n, len(data), out)
	}
}
