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
// depends on: `fduty version --json` must emit a broker_egress field that is true
// on this (unix) build. If this regresses, broker-capable runners would stop
// advertising broker mode and silently fall back to the legacy env-key path.
func TestVersionJSONAdvertisesBrokerEgress(t *testing.T) {
	origJSON := flagJSON
	flagJSON = true
	defer func() { flagJSON = origJSON }()

	cmd := newVersionCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.Run(cmd, nil)

	var got struct {
		Version      string `json:"version"`
		BrokerEgress bool   `json:"broker_egress"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("version --json must be valid JSON, got %q: %v", buf.String(), err)
	}
	if !got.BrokerEgress {
		t.Fatalf("broker_egress must be true on a unix build, got: %q", buf.String())
	}
}
