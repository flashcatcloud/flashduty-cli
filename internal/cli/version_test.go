package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestVersionPlainOutput keeps the human-readable line stable (the runner's
// install scripts and humans rely on it).
func TestVersionPlainOutput(t *testing.T) {
	cmd := newVersionCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.Run(cmd, nil)
	if got := buf.String(); !strings.HasPrefix(got, "flashduty version ") {
		t.Fatalf("plain version output changed: %q", got)
	}
}

// TestVersionJSONAdvertisesBrokerEgress locks the capability contract the runner
// depends on: `fduty version --json` must emit a broker_egress field whose value
// matches this build's compile-time capability (true on unix, false elsewhere).
// If the field went missing, broker-capable runners would stop advertising
// broker mode and silently fall back to the legacy env-key path.
func TestVersionJSONAdvertisesBrokerEgress(t *testing.T) {
	origJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = origJSON }()

	cmd := newVersionCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.Run(cmd, nil)

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("version --json must be valid JSON, got %q: %v", buf.String(), err)
	}
	v, ok := got["broker_egress"]
	if !ok {
		t.Fatalf("version --json must include the broker_egress field, got: %q", buf.String())
	}
	if v != brokerEgressCapable {
		t.Fatalf("broker_egress = %v, want %v (compile-time capability)", v, brokerEgressCapable)
	}
}
