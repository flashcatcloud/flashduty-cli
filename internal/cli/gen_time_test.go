package cli

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func newTimeFlagCmd(t *testing.T, set bool, raw string) (*cobra.Command, string) {
	t.Helper()
	cmd := &cobra.Command{Use: "x"}
	var v string
	cmd.Flags().StringVar(&v, "start-time", "", "")
	if set {
		if err := cmd.Flags().Set("start-time", raw); err != nil {
			t.Fatalf("set flag: %v", err)
		}
	}
	return cmd, v
}

func TestGenParseTimeFlag(t *testing.T) {
	// Unset flag → ok=false so the caller omits the field.
	cmd, raw := newTimeFlagCmd(t, false, "")
	if v, ok, err := genParseTimeFlag(cmd, "start-time", raw); err != nil || ok || v != 0 {
		t.Errorf("unset: got (%d,%v,%v), want (0,false,nil)", v, ok, err)
	}

	// Relative duration → roughly now minus the duration.
	cmd, raw = newTimeFlagCmd(t, true, "24h")
	v, ok, err := genParseTimeFlag(cmd, "start-time", raw)
	if err != nil || !ok {
		t.Fatalf("24h: ok=%v err=%v", ok, err)
	}
	if delta := time.Now().Unix() - 24*3600 - v; delta < -5 || delta > 5 {
		t.Errorf("24h: parsed %d not ~24h ago (delta %ds)", v, delta)
	}

	// Raw unix seconds pass through unchanged (back-compat with old int flag).
	cmd, raw = newTimeFlagCmd(t, true, "1700000000")
	if v, ok, err := genParseTimeFlag(cmd, "start-time", raw); err != nil || !ok || v != 1700000000 {
		t.Errorf("unix passthrough: got (%d,%v,%v), want (1700000000,true,nil)", v, ok, err)
	}

	// Invalid value → error mentioning the flag.
	cmd, raw = newTimeFlagCmd(t, true, "not-a-time")
	if _, ok, err := genParseTimeFlag(cmd, "start-time", raw); err == nil || ok {
		t.Errorf("invalid: expected error, got ok=%v err=%v", ok, err)
	}
}
