package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// testItem is a minimal struct used to exercise Print through the factory.
type testItem struct {
	Name string `json:"name"`
}

func testColumns() []Column {
	return []Column{
		{Header: "NAME", Field: func(v any) string { return v.(testItem).Name }},
	}
}

// Test 43: NewPrinter with jsonMode=true returns a JSONPrinter and produces valid JSON output.
func TestNewPrinter_JSONMode(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(true, false, &buf)

	// Verify the concrete type is *JSONPrinter.
	if _, ok := p.(*JSONPrinter); !ok {
		t.Fatalf("expected *JSONPrinter, got %T", p)
	}

	data := []testItem{{Name: "alert-1"}}
	if err := p.Print(data, testColumns()); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if !json.Valid([]byte(output)) {
		t.Errorf("output is not valid JSON:\n%s", output)
	}

	// Sanity-check the value round-trips.
	var got []testItem
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	if len(got) != 1 || got[0].Name != "alert-1" {
		t.Errorf("unexpected unmarshalled data: %+v", got)
	}
}

// Test 44: NewPrinter with jsonMode=false returns a TablePrinter and produces tab-separated header output.
func TestNewPrinter_TableMode(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(false, false, &buf)

	// Verify the concrete type is *TablePrinter.
	if _, ok := p.(*TablePrinter); !ok {
		t.Fatalf("expected *TablePrinter, got %T", p)
	}

	data := []testItem{{Name: "svc-web"}}
	if err := p.Print(data, testColumns()); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	output := buf.String()

	// The table printer must emit the header row.
	if !strings.Contains(output, "NAME") {
		t.Errorf("table output missing header NAME:\n%s", output)
	}

	// The data row must be present.
	if !strings.Contains(output, "svc-web") {
		t.Errorf("table output missing data value svc-web:\n%s", output)
	}

	// The output must NOT be valid JSON (it is a table, not JSON).
	trimmed := strings.TrimSpace(output)
	if json.Valid([]byte(trimmed)) {
		t.Errorf("table output should not be valid JSON:\n%s", trimmed)
	}
}

// Test 45: NewPrinter with a nil writer does not panic; it defaults to os.Stdout.
func TestNewPrinter_NilWriterDefaults(t *testing.T) {
	// The call itself must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewPrinter panicked with nil writer: %v", r)
		}
	}()

	p := NewPrinter(false, false, nil)
	if p == nil {
		t.Fatal("expected non-nil Printer, got nil")
	}

	// Also verify the JSON path with nil writer.
	pJSON := NewPrinter(true, false, nil)
	if pJSON == nil {
		t.Fatal("expected non-nil JSON Printer, got nil")
	}
}

// Test 46: NewPrinter with jsonMode=true and noTrunc=true still produces JSON output
// (noTrunc is irrelevant for JSON mode).
func TestNewPrinter_NoTruncIrrelevantForJSON(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(true, true, &buf)

	// Verify it is still a JSONPrinter regardless of noTrunc.
	if _, ok := p.(*JSONPrinter); !ok {
		t.Fatalf("expected *JSONPrinter when jsonMode=true and noTrunc=true, got %T", p)
	}

	data := []testItem{{Name: "long-service-name-that-might-be-truncated"}}
	if err := p.Print(data, testColumns()); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if !json.Valid([]byte(output)) {
		t.Errorf("output is not valid JSON:\n%s", output)
	}

	// The full name must appear without truncation.
	if !strings.Contains(output, "long-service-name-that-might-be-truncated") {
		t.Errorf("JSON output should contain the full untruncated name:\n%s", output)
	}
}
