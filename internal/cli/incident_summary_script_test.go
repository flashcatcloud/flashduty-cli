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
	if strings.Contains(string(invocations), "--output-format toon") {
		t.Fatalf("summary forces toon instead of each command's compact default:\n%s", invocations)
	}
}
