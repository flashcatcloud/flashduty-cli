package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// helper types used across tests
type sampleItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// dummyColumns returns a non-nil column slice that JSONPrinter should ignore.
func dummyColumns() []Column {
	return []Column{
		{Header: "NAME", Field: func(item any) string { return "ignored" }},
	}
}

func TestJSONPrinter_Print(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		data      any
		columns   []Column
		wantErr   bool
		validate  func(t *testing.T, output string)
	}{
		{
			// Test 35: JSON outputs valid JSON
			name:    "outputs valid JSON from struct",
			data:    sampleItem{Name: "alert-rule", Count: 7},
			columns: nil,
			wantErr: false,
			validate: func(t *testing.T, output string) {
				t.Helper()
				if !json.Valid([]byte(output)) {
					t.Errorf("output is not valid JSON: %s", output)
				}
			},
		},
		{
			// Test 36: JSON is indented
			name:    "output is indented with newlines and spaces",
			data:    sampleItem{Name: "incident", Count: 3},
			columns: nil,
			wantErr: false,
			validate: func(t *testing.T, output string) {
				t.Helper()
				if !strings.Contains(output, "\n") {
					t.Error("expected output to contain newlines for indentation")
				}
				if !strings.Contains(output, "  ") {
					t.Error("expected output to contain two-space indentation")
				}
				// Verify specific indented field format
				if !strings.Contains(output, "  \"name\": \"incident\"") {
					t.Errorf("expected indented field, got: %s", output)
				}
			},
		},
		{
			// Test 37: JSON ignores columns -- full data printed regardless of columns
			name:    "columns parameter is ignored and full data is printed",
			data:    sampleItem{Name: "service", Count: 42},
			columns: dummyColumns(),
			wantErr: false,
			validate: func(t *testing.T, output string) {
				t.Helper()
				var got sampleItem
				if err := json.Unmarshal([]byte(output), &got); err != nil {
					t.Fatalf("failed to unmarshal output: %v", err)
				}
				if got.Name != "service" {
					t.Errorf("Name = %q, want %q", got.Name, "service")
				}
				if got.Count != 42 {
					t.Errorf("Count = %d, want %d", got.Count, 42)
				}
			},
		},
		{
			// Test 38: JSON with slice data
			name: "prints slice as JSON array",
			data: []sampleItem{
				{Name: "alpha", Count: 1},
				{Name: "beta", Count: 2},
			},
			columns: nil,
			wantErr: false,
			validate: func(t *testing.T, output string) {
				t.Helper()
				trimmed := strings.TrimSpace(output)
				if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
					t.Errorf("expected JSON array, got: %s", trimmed)
				}
				var got []sampleItem
				if err := json.Unmarshal([]byte(trimmed), &got); err != nil {
					t.Fatalf("failed to unmarshal array: %v", err)
				}
				if len(got) != 2 {
					t.Errorf("len = %d, want 2", len(got))
				}
			},
		},
		{
			// Test 39: JSON with single item (struct -> JSON object)
			name:    "prints single struct as JSON object",
			data:    sampleItem{Name: "on-call", Count: 99},
			columns: nil,
			wantErr: false,
			validate: func(t *testing.T, output string) {
				t.Helper()
				trimmed := strings.TrimSpace(output)
				if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
					t.Errorf("expected JSON object, got: %s", trimmed)
				}
			},
		},
		{
			// Test 40: JSON with nil data
			name:    "prints nil as JSON null",
			data:    nil,
			columns: nil,
			wantErr: false,
			validate: func(t *testing.T, output string) {
				t.Helper()
				trimmed := strings.TrimSpace(output)
				if trimmed != "null" {
					t.Errorf("output = %q, want %q", trimmed, "null")
				}
				if !json.Valid([]byte(trimmed)) {
					t.Error("null output is not valid JSON")
				}
			},
		},
		{
			// Test 41: JSON with empty slice
			name:    "prints empty slice as empty JSON array",
			data:    []sampleItem{},
			columns: nil,
			wantErr: false,
			validate: func(t *testing.T, output string) {
				t.Helper()
				trimmed := strings.TrimSpace(output)
				if trimmed != "[]" {
					t.Errorf("output = %q, want %q", trimmed, "[]")
				}
			},
		},
		{
			// Test 42: JSON with unmarshalable data (func value)
			name:    "returns error for unmarshalable data containing func",
			data:    map[string]any{"callback": func() {}},
			columns: nil,
			wantErr: true,
			validate: func(t *testing.T, output string) {
				t.Helper()
				// No output validation needed; we only care about the error.
			},
		},
		{
			// Test 42 (variant): JSON with unmarshalable data (chan value)
			name:    "returns error for unmarshalable data containing chan",
			data:    map[string]any{"ch": make(chan int)},
			columns: nil,
			wantErr: true,
			validate: func(t *testing.T, output string) {
				t.Helper()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			printer := &JSONPrinter{w: &buf}

			err := printer.Print(tc.data, tc.columns)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tc.validate(t, buf.String())
		})
	}
}

// TestJSONPrinter_ErrorWrapping verifies the error message wraps with context.
func TestJSONPrinter_ErrorWrapping(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printer := &JSONPrinter{w: &buf}

	err := printer.Print(make(chan int), nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	const wantPrefix = "failed to marshal JSON:"
	if !strings.Contains(err.Error(), wantPrefix) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), wantPrefix)
	}
}
