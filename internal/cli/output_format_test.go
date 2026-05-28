package cli

import (
	"testing"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

// resolveOutputFormat maps --output-format / --json to a format, with
// --output-format winning, --json as the fallback alias, and an unknown
// value erroring so a typo fails fast instead of silently picking table.
func TestResolveOutputFormat(t *testing.T) {
	cases := []struct {
		name    string
		format  string
		json    bool
		want    output.Format
		wantErr bool
	}{
		{"default is table", "", false, output.FormatTable, false},
		{"json bool alias", "", true, output.FormatJSON, false},
		{"explicit table", "table", false, output.FormatTable, false},
		{"explicit json", "json", false, output.FormatJSON, false},
		{"explicit toon", "toon", false, output.FormatTOON, false},
		{"toon wins over json bool", "toon", true, output.FormatTOON, false},
		{"case-insensitive", "TOON", false, output.FormatTOON, false},
		{"invalid errors", "yaml", false, output.FormatTable, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origFormat, origJSON := flagOutputFormat, flagJSON
			defer func() { flagOutputFormat, flagJSON = origFormat, origJSON }()
			flagOutputFormat, flagJSON = tc.format, tc.json

			got, err := resolveOutputFormat()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.format)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("resolveOutputFormat(%q, json=%v) = %v, want %v", tc.format, tc.json, got, tc.want)
			}
		})
	}
}
