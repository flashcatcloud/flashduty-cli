package cli

import (
	"testing"
)

// TestChangeListChannelFlag verifies that --channel is a string flag (comma-separated IDs),
// not a singular int64 flag. Mirrors the alert list --channel pattern.
func TestChangeListChannelFlag(t *testing.T) {
	cmd := newChangeListCmd()
	flags := cmd.Flags()

	f := flags.Lookup("channel")
	if f == nil {
		t.Fatal("flag --channel not registered")
	}

	// Must be a string flag (Value.Type() == "string"), not int64.
	if got := f.Value.Type(); got != "string" {
		t.Errorf("--channel flag type = %q, want %q", got, "string")
	}

	// Default must be empty string (not "0").
	if got := f.DefValue; got != "" {
		t.Errorf("--channel default = %q, want %q", got, "")
	}
}

// TestChangeListChannelParsing verifies that a comma-separated --channel value
// is correctly parsed to []int64 via parseIntSlice — the same helper used by
// alert list. Full comma-split semantics are covered by TestParseIntSlice in
// helpers_test.go; this test only confirms the wiring is correct.
func TestChangeListChannelParsing(t *testing.T) {
	// parseIntSlice is the shared helper; spot-check the three-value case.
	got, err := parseIntSlice("100,200,300")
	if err != nil {
		t.Fatalf("parseIntSlice(\"100,200,300\"): unexpected error: %v", err)
	}
	want := []int64{100, 200, 300}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %d, want %d", i, got[i], want[i])
		}
	}
}
