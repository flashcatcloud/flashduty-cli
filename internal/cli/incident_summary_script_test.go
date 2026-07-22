package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestIncidentSummaryScriptCompactOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Bash fixture is unavailable on Windows")
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	script := filepath.Join(root, "skills", "flashduty", "scripts", "incident-summary.sh")
	log := filepath.Join(t.TempDir(), "fduty.log")
	bin := filepath.Join(t.TempDir(), "fduty")
	if err := os.WriteFile(bin, []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$*\" >> \"$FDUTY_LOG\"\nprintf 'compact result\\n'\n"), 0o755); err != nil {
		t.Fatalf("write fake fduty: %v", err)
	}
	t.Setenv("FDUTY_LOG", log)
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))

	output, err := exec.Command("bash", script, "inc-1").CombinedOutput()
	if err != nil {
		t.Fatalf("run incident summary: %v\n%s", err, output)
	}
	invocations, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("read fake fduty log: %v", err)
	}
	lines := strings.FieldsFunc(strings.TrimSpace(string(invocations)), func(r rune) bool { return r == '\n' })
	if len(lines) != 6 {
		t.Fatalf("fduty calls = %d, want 6:\n%s", len(lines), invocations)
	}
	wantDetail := "incident detail inc-1 --fields incident_id,title,incident_severity,progress,ai_summary,root_cause,resolution,alert_cnt,start_time,channel_id --output-format toon"
	if lines[0] != wantDetail {
		t.Fatalf("detail call = %q, want compact projection %q", lines[0], wantDetail)
	}
	if strings.Contains(strings.Join(lines[1:], "\n"), "--output-format toon") {
		t.Fatalf("non-detail reads force raw toon instead of their compact defaults:\n%s", invocations)
	}
}
