package skilldoc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFlashdutySkillGuidesLargeListScans(t *testing.T) {
	body := readSkillFile(t, "SKILL.md")
	incident := readSkillFile(t, "reference", "incident.md")
	insight := readSkillFile(t, "reference", "insight.md")

	for _, want := range []string{
		"Large list scans",
		"redirect JSON to a temp file",
		"Do not dump page-sized JSON",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("SKILL.md missing large-list guidance %q", want)
		}
	}
	for _, want := range []string{
		"Do not re-run `incident alerts`, `incident similar`, `incident timeline`, or `change list`",
		"Page-sized scans go to files",
	} {
		if !strings.Contains(incident, want) {
			t.Errorf("incident card missing large-list/summary guidance %q", want)
		}
	}
	for _, want := range []string{
		"For multi-dimensional reports, fetch once to a temp file",
		"Do not run the same account-wide list again for each `jq` bucket",
	} {
		if !strings.Contains(insight, want) {
			t.Errorf("insight card missing reusable JSON guidance %q", want)
		}
	}
}

func readSkillFile(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{"..", "..", "skills", "flashduty"}, parts...)...)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return string(b)
}
