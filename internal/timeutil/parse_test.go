package timeutil

import (
	"math"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	type testCase struct {
		name      string
		input     string
		wantExact int64  // used when exactMatch is true
		wantApprox int64 // used when approxMatch is true (expected unix timestamp)
		exactMatch bool
		approxMatch bool
		wantErr    bool
		tolerance  int64 // seconds of tolerance for approximate matches
	}

	now := time.Now().Unix()

	tests := []testCase{
		// 1. Relative duration minutes
		{
			name:        "relative duration minutes 5m",
			input:       "5m",
			wantApprox:  now - int64(5*time.Minute/time.Second),
			approxMatch: true,
			tolerance:   2,
		},
		// 2. Relative duration hours
		{
			name:        "relative duration hours 24h",
			input:       "24h",
			wantApprox:  now - int64(24*time.Hour/time.Second),
			approxMatch: true,
			tolerance:   2,
		},
		// 3. Relative duration large
		{
			name:        "relative duration large 168h",
			input:       "168h",
			wantApprox:  now - int64(168*time.Hour/time.Second),
			approxMatch: true,
			tolerance:   2,
		},
		// 4. Relative duration seconds
		{
			name:        "relative duration seconds 30s",
			input:       "30s",
			wantApprox:  now - 30,
			approxMatch: true,
			tolerance:   2,
		},
		// 5. Absolute date
		{
			name:       "absolute date 2026-04-01",
			input:      "2026-04-01",
			wantExact:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local).Unix(),
			exactMatch: true,
		},
		// 6. Absolute datetime
		{
			name:       "absolute datetime 2026-04-01 10:00:00",
			input:      "2026-04-01 10:00:00",
			wantExact:  time.Date(2026, 4, 1, 10, 0, 0, 0, time.Local).Unix(),
			exactMatch: true,
		},
		// 7. Unix timestamp
		{
			name:       "unix timestamp 1712000000",
			input:      "1712000000",
			wantExact:  1712000000,
			exactMatch: true,
		},
		// 8. "now" keyword
		{
			name:        "now keyword",
			input:       "now",
			wantApprox:  now,
			approxMatch: true,
			tolerance:   2,
		},
		// 9. Empty string
		{
			name:        "empty string",
			input:       "",
			wantApprox:  now,
			approxMatch: true,
			tolerance:   2,
		},
		// 10. Invalid string
		{
			name:    "invalid string garbage",
			input:   "garbage",
			wantErr: true,
		},
		// 11. Invalid string 2
		{
			name:    "invalid string abc123",
			input:   "abc123",
			wantErr: true,
		},
		// 12. Small number (not a valid timestamp)
		{
			name:    "small number 999 not timestamp",
			input:   "999",
			wantErr: true,
		},
		// 13. Whitespace trimmed
		{
			name:        "whitespace trimmed around 24h",
			input:       "  24h  ",
			wantApprox:  now - int64(24*time.Hour/time.Second),
			approxMatch: true,
			tolerance:   2,
		},
		// 14. Negative duration rejected
		{
			name:    "negative duration -5m rejected",
			input:   "-5m",
			wantErr: true,
		},
		// 15. Boundary timestamp 1000000000 (code checks ts > 1000000000, not >=)
		{
			name:    "boundary timestamp 1000000000 rejected",
			input:   "1000000000",
			wantErr: true,
		},
		// 16. Boundary timestamp 1000000001
		{
			name:       "boundary timestamp 1000000001 accepted",
			input:      "1000000001",
			wantExact:  1000000001,
			exactMatch: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) expected error, got %d", tc.input, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tc.input, err)
			}

			if tc.exactMatch {
				if got != tc.wantExact {
					t.Errorf("Parse(%q) = %d, want exactly %d", tc.input, got, tc.wantExact)
				}
			}

			if tc.approxMatch {
				diff := int64(math.Abs(float64(got - tc.wantApprox)))
				if diff > tc.tolerance {
					t.Errorf("Parse(%q) = %d, want approximately %d (tolerance %ds, actual diff %ds)",
						tc.input, got, tc.wantApprox, tc.tolerance, diff)
				}
			}
		})
	}
}
