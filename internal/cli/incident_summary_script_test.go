package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// runIncidentSummaryScript runs skills/flashduty/scripts/incident-summary.sh against a fake
// fduty that logs every invocation. detailJSON is what the fake prints for `--json` calls; every
// other call prints a one-line placeholder.
func runIncidentSummaryScript(t *testing.T, detailJSON string) (output string, calls []string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Bash fixture is unavailable on Windows")
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	script := filepath.Join(root, "skills", "flashduty", "scripts", "incident-summary.sh")
	dir := t.TempDir()
	log := filepath.Join(dir, "fduty.log")
	bin := filepath.Join(dir, "fduty")
	fake := "#!/usr/bin/env bash\n" +
		"printf '%s\\n' \"$*\" >> \"$FDUTY_LOG\"\n" +
		"case \" $* \" in\n" +
		"  *' --json '*) printf '%s\\n' \"$FDUTY_DETAIL_JSON\" ;;\n" +
		"  *) printf 'compact result\\n' ;;\n" +
		"esac\n"
	if err := os.WriteFile(bin, []byte(fake), 0o755); err != nil {
		t.Fatalf("write fake fduty: %v", err)
	}
	t.Setenv("FDUTY_LOG", log)
	t.Setenv("FDUTY_DETAIL_JSON", detailJSON)
	t.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))

	out, err := exec.Command("bash", script, "inc-1").CombinedOutput()
	if err != nil {
		t.Fatalf("run incident summary: %v\n%s", err, out)
	}
	invocations, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("read fake fduty log: %v", err)
	}
	calls = strings.FieldsFunc(strings.TrimSpace(string(invocations)), func(r rune) bool { return r == '\n' })
	return string(out), calls
}

func TestIncidentSummaryScriptCompactOutput(t *testing.T) {
	// 2026-01-01T00:00:00Z = 1767225600; the concurrent-incidents window is ±15 min around it.
	_, calls := runIncidentSummaryScript(t, `{"incident_id":"inc-1","start_time":"2026-01-01T00:00:00Z"}`)
	if len(calls) != 8 {
		t.Fatalf("fduty calls = %d, want 8:\n%s", len(calls), strings.Join(calls, "\n"))
	}
	wantDetail := "incident detail inc-1 --fields incident_id,title,incident_severity,progress,ai_summary,root_cause,resolution,alert_cnt,start_time,channel_id --output-format toon"
	if calls[0] != wantDetail {
		t.Fatalf("detail call = %q, want compact projection %q", calls[0], wantDetail)
	}
	if strings.Contains(strings.Join(calls[1:6], "\n"), "--output-format toon") {
		t.Fatalf("non-detail reads force raw toon instead of their compact defaults:\n%s", strings.Join(calls, "\n"))
	}
	if calls[6] != "incident detail inc-1 --json" {
		t.Fatalf("start_time probe = %q, want %q", calls[6], "incident detail inc-1 --json")
	}
	// ⑦ is a projected read like ①: explicit --fields keeps full titles and the num/channel_id
	// columns the card tells the agent to reason from, so toon is the compact form here.
	wantConcurrent := "incident list --since 1767224700 --until 1767226500 --limit 50 --fields incident_id,num,title,incident_severity,progress,start_time,channel_id --output-format toon"
	if calls[7] != wantConcurrent {
		t.Fatalf("concurrent-incidents call = %q, want %q", calls[7], wantConcurrent)
	}
}

func TestIncidentSummaryScriptSkipsConcurrentWithoutStartTime(t *testing.T) {
	output, calls := runIncidentSummaryScript(t, `{"incident_id":"inc-1"}`)
	if len(calls) != 7 {
		t.Fatalf("fduty calls = %d, want 7 (six reads + the start_time probe, no list):\n%s", len(calls), strings.Join(calls, "\n"))
	}
	if !strings.Contains(output, "⑦ concurrent incidents: SKIPPED") {
		t.Fatalf("output lacks the SKIPPED marker for ⑦:\n%s", output)
	}
	if !strings.Contains(output, "fduty incident list --since <start-15m> --until <start+15m>") {
		t.Fatalf("SKIPPED marker lacks the manual command:\n%s", output)
	}
}
