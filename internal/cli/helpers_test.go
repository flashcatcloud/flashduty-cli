package cli

import (
	"strings"
	"testing"
)

func TestParseIntSlice(t *testing.T) {
	tests := []struct {
		id      int
		name    string
		input   string
		want    []int64
		wantErr string // substring expected in error message; empty means no error
	}{
		{
			id:    68,
			name:  "empty string returns nil",
			input: "",
			want:  nil,
		},
		{
			id:    69,
			name:  "single value",
			input: "42",
			want:  []int64{42},
		},
		{
			id:    70,
			name:  "multiple values",
			input: "1,2,3",
			want:  []int64{1, 2, 3},
		},
		{
			id:    71,
			name:  "values with surrounding spaces",
			input: "1, 2, 3",
			want:  []int64{1, 2, 3},
		},
		{
			id:    72,
			name:  "trailing comma is ignored",
			input: "1,2,",
			want:  []int64{1, 2},
		},
		{
			id:      73,
			name:    "invalid value returns error",
			input:   "1,abc,3",
			wantErr: "invalid ID",
		},
		{
			id:    74,
			name:  "negative value",
			input: "-1",
			want:  []int64{-1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseIntSlice(tc.input)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("[#%d] expected error containing %q, got nil", tc.id, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("[#%d] error %q does not contain %q", tc.id, err.Error(), tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("[#%d] unexpected error: %v", tc.id, err)
			}

			if len(got) != len(tc.want) {
				t.Fatalf("[#%d] length mismatch: got %d, want %d", tc.id, len(got), len(tc.want))
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("[#%d] index %d: got %d, want %d", tc.id, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestOrDash(t *testing.T) {
	tests := []struct {
		id    int
		name  string
		input string
		want  string
	}{
		{
			id:    75,
			name:  "empty string returns dash",
			input: "",
			want:  "-",
		},
		{
			id:    76,
			name:  "non-empty string returned as is",
			input: "hello",
			want:  "hello",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := orDash(tc.input)
			if got != tc.want {
				t.Errorf("[#%d] orDash(%q) = %q, want %q", tc.id, tc.input, got, tc.want)
			}
		})
	}
}

// TestMemberPersonInfosDisplay is a placeholder for test #321.
// The member list command constructs the API client internally, so we cannot
// inject a fake client to test display logic in isolation. This will be
// addressed in Phase 3 when an injection seam is introduced.
func TestMemberPersonInfosDisplay(t *testing.T) {
	t.Skip("requires injection seam for fake client (Phase 3)")
}
