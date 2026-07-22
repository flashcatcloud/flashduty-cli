package skilldoc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIncidentCardAvoidsUnboundedStructuredHotFlows(t *testing.T) {
	card, err := os.ReadFile(filepath.Join("..", "..", "skills", "flashduty", "reference", "incident.md"))
	if err != nil {
		t.Fatal(err)
	}

	body := string(card)
	for description, command := range map[string]string{
		"triage list":    "incident list --severity Critical --progress Triggered --since 4h --fields incident_id,title,incident_severity,progress,start_time,channel_id --output-format toon",
		"triage detail":  "incident detail <incident-id> --fields incident_id,title,incident_severity,progress,ai_summary,root_cause,resolution,alert_cnt,start_time,channel_id --output-format toon",
		"summary detail": `incident detail   "$ID" --fields incident_id,title,incident_severity,progress,ai_summary,root_cause,resolution,alert_cnt,start_time,channel_id --output-format toon`,
	} {
		if !strings.Contains(body, command) {
			t.Errorf("incident %s must project structured output to the fields needed by the workflow", description)
		}
	}

	for _, command := range []string{
		"incident alerts <incident-id> --output-format toon",
		`incident alerts   "$ID" --output-format toon`,
		"incident post-mortem-list --channel-ids <channel-id> --output-format toon",
		"change list --since 24h --output-format toon",
		"incident timeline <primary-incident-id> --output-format toon",
	} {
		if strings.Contains(body, command) {
			t.Errorf("incident hot flow must use the compact default instead of %q", command)
		}
	}

	_, summary, found := strings.Cut(body, "## Hot flow — full fault analysis (read-only summary)")
	if !found {
		t.Fatal("incident card is missing the full fault analysis section")
	}
	summary, _, found = strings.Cut(summary, "## Hot flow — resolve, document, and merge duplicates")
	if !found {
		t.Fatal("incident card is missing the section after full fault analysis")
	}
	if strings.Contains(summary, `incident timeline "$ID" --output-format toon`) {
		t.Error("full fault analysis must use timeline's compact default; structured timeline is reserved for comment read-back")
	}
}
