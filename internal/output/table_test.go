package output

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Tests 17-24: Truncate
// ---------------------------------------------------------------------------

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		// 17: empty string
		{name: "empty string", input: "", maxLen: 10, want: ""},
		// 18: shorter than max
		{name: "shorter than max", input: "abc", maxLen: 10, want: "abc"},
		// 19: exact max
		{name: "exact max", input: "abcde", maxLen: 5, want: "abcde"},
		// 20: longer than max
		{name: "longer than max", input: "abcdefghij", maxLen: 7, want: "abcd..."},
		// 21: maxLen=0 (no limit)
		{name: "maxLen zero no limit", input: "abc", maxLen: 0, want: "abc"},
		// 22: maxLen<=3
		{name: "maxLen equals 3", input: "abcdef", maxLen: 3, want: "abc"},
		// 23: maxLen=1
		{name: "maxLen equals 1", input: "abcdef", maxLen: 1, want: "a"},
		// 24: maxLen negative
		{name: "maxLen negative", input: "abc", maxLen: -1, want: "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests 25-26: FormatTime
// ---------------------------------------------------------------------------

func TestFormatTime(t *testing.T) {
	t.Run("zero returns dash", func(t *testing.T) {
		// 25
		got := FormatTime(0)
		if got != "-" {
			t.Errorf("FormatTime(0) = %q, want %q", got, "-")
		}
	})

	t.Run("nonzero returns formatted local time", func(t *testing.T) {
		// 26
		const ts int64 = 1712000000
		want := time.Unix(ts, 0).Local().Format("2006-01-02 15:04")
		got := FormatTime(ts)
		if got != want {
			t.Errorf("FormatTime(%d) = %q, want %q", ts, got, want)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests 27-34: TablePrinter.Print
// ---------------------------------------------------------------------------

// helper type used as row data in printer tests
type testRow struct {
	Name  string
	Value string
}

// nameCol returns a Column that extracts Name from a testRow.
func nameCol(maxWidth int) Column {
	return Column{
		Header:   "NAME",
		MaxWidth: maxWidth,
		Field: func(item any) string {
			r := item.(testRow)
			return r.Name
		},
	}
}

// valueCol returns a Column that extracts Value from a testRow.
func valueCol(maxWidth int) Column {
	return Column{
		Header:   "VALUE",
		MaxWidth: maxWidth,
		Field: func(item any) string {
			r := item.(testRow)
			return r.Value
		},
	}
}

func TestTablePrinter_RendersHeaders(t *testing.T) {
	// 27
	var buf bytes.Buffer
	p := &TablePrinter{w: &buf}

	columns := []Column{nameCol(0), valueCol(0)}
	data := []testRow{}

	if err := p.Print(data, columns); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "NAME") {
		t.Errorf("output missing header NAME: %q", out)
	}
	if !strings.Contains(out, "VALUE") {
		t.Errorf("output missing header VALUE: %q", out)
	}
}

func TestTablePrinter_RendersValues(t *testing.T) {
	// 28
	var buf bytes.Buffer
	p := &TablePrinter{w: &buf}

	columns := []Column{nameCol(0), valueCol(0)}
	data := []testRow{
		{Name: "alpha", Value: "100"},
	}

	if err := p.Print(data, columns); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "alpha") {
		t.Errorf("output missing value 'alpha': %q", out)
	}
	if !strings.Contains(out, "100") {
		t.Errorf("output missing value '100': %q", out)
	}
}

func TestTablePrinter_TruncatesByDefault(t *testing.T) {
	// 29
	var buf bytes.Buffer
	p := &TablePrinter{w: &buf, noTrunc: false}

	columns := []Column{nameCol(5)}
	data := []testRow{
		{Name: "abcdefghij"},
	}

	if err := p.Print(data, columns); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ab...") {
		t.Errorf("expected truncated value 'ab...' in output: %q", out)
	}
	if strings.Contains(out, "abcdefghij") {
		t.Errorf("expected full value to be truncated, but found 'abcdefghij' in output: %q", out)
	}
}

func TestTablePrinter_NoTruncSkipsTruncation(t *testing.T) {
	// 30
	var buf bytes.Buffer
	p := &TablePrinter{w: &buf, noTrunc: true}

	columns := []Column{nameCol(5)}
	data := []testRow{
		{Name: "abcdefghij"},
	}

	if err := p.Print(data, columns); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "abcdefghij") {
		t.Errorf("expected full value 'abcdefghij' in output with noTrunc=true: %q", out)
	}
}

func TestTablePrinter_EmptyData(t *testing.T) {
	// 31
	var buf bytes.Buffer
	p := &TablePrinter{w: &buf}

	columns := []Column{nameCol(0), valueCol(0)}
	data := []testRow{}

	if err := p.Print(data, columns); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line (headers only), got %d lines: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "NAME") {
		t.Errorf("header line missing 'NAME': %q", lines[0])
	}
}

func TestTablePrinter_MultipleRows(t *testing.T) {
	// 32
	var buf bytes.Buffer
	p := &TablePrinter{w: &buf}

	columns := []Column{nameCol(0)}
	data := []testRow{
		{Name: "row1"},
		{Name: "row2"},
		{Name: "row3"},
	}

	if err := p.Print(data, columns); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// 1 header + 3 data rows = 4 lines
	if len(lines) != 4 {
		t.Errorf("expected 4 lines (1 header + 3 rows), got %d: %q", len(lines), out)
	}
	for _, name := range []string{"row1", "row2", "row3"} {
		if !strings.Contains(out, name) {
			t.Errorf("output missing row %q: %q", name, out)
		}
	}
}

func TestTablePrinter_MultipleColumns(t *testing.T) {
	// 33
	var buf bytes.Buffer
	p := &TablePrinter{w: &buf}

	columns := []Column{
		{Header: "COL1", Field: func(item any) string { return item.(testRow).Name }},
		{Header: "COL2", Field: func(item any) string { return item.(testRow).Value }},
		{Header: "COL3", Field: func(item any) string { return "fixed" }},
		{Header: "COL4", Field: func(item any) string { return "static" }},
	}
	data := []testRow{
		{Name: "a", Value: "b"},
	}

	if err := p.Print(data, columns); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	out := buf.String()
	for _, hdr := range []string{"COL1", "COL2", "COL3", "COL4"} {
		if !strings.Contains(out, hdr) {
			t.Errorf("output missing header %q: %q", hdr, out)
		}
	}
	for _, val := range []string{"a", "b", "fixed", "static"} {
		if !strings.Contains(out, val) {
			t.Errorf("output missing value %q: %q", val, out)
		}
	}
}

func TestTablePrinter_NonSliceInput(t *testing.T) {
	// 34
	var buf bytes.Buffer
	p := &TablePrinter{w: &buf}

	columns := []Column{nameCol(0)}
	singleItem := testRow{Name: "solo"}

	if err := p.Print(singleItem, columns); err != nil {
		t.Fatalf("Print returned error: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// 1 header + 1 data row = 2 lines
	if len(lines) != 2 {
		t.Errorf("expected 2 lines (1 header + 1 row), got %d: %q", len(lines), out)
	}
	if !strings.Contains(out, "solo") {
		t.Errorf("output missing value 'solo': %q", out)
	}
}
