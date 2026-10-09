package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestAutomationCreateDailyDefaultsEnabled(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"automation", "create",
		"--name", "Daily SRE brief",
		"--team-id", "123",
		"--schedule", "daily",
		"--at", "08:30",
		"--prompt", "Summarize yesterday's incidents",
		"--json",
	)
	if err != nil {
		t.Fatalf("[automation-create-daily] unexpected error: %v", err)
	}
	if stub.lastPath != "/safari/automation/rule/create" {
		t.Fatalf("[automation-create-daily] path = %q", stub.lastPath)
	}
	assertBody(t, stub.lastBody, "name", "Daily SRE brief")
	assertBody(t, stub.lastBody, "team_id", float64(123))
	assertBody(t, stub.lastBody, "cron_expr", "30 8 * * *")
	assertBody(t, stub.lastBody, "enabled", true)
	assertBody(t, stub.lastBody, "schedule_trigger_enabled", true)
	assertBody(t, stub.lastBody, "prompt", "Summarize yesterday's incidents")
}

func TestAutomationScheduleHelpDocumentsTimezone(t *testing.T) {
	saveAndResetGlobals(t)

	for _, args := range [][]string{
		{"automation", "create", "--help"},
		{"automation", "update", "auto_123", "--help"},
	} {
		out, err := execCommand(args...)
		if err != nil {
			t.Fatalf("%v unexpected error: %v", args, err)
		}
		for _, want := range []string{
			"do not convert it to UTC",
			"--timezone",
			automationTimezoneNote,
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("%v help missing %q\n%s", args, want, out)
			}
		}
		if strings.Contains(out, "has no --timezone flag") {
			t.Fatalf("%v help still says the command has no --timezone flag\n%s", args, out)
		}
	}
}

func TestAutomationCreateTimezone(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"automation", "create",
		"--name", "Daily SRE brief",
		"--schedule", "daily",
		"--at", "09:00",
		"--prompt", "Summarize yesterday's incidents",
		"--timezone", "Asia/Shanghai",
		"--json",
	)
	if err != nil {
		t.Fatalf("[automation-create-timezone] unexpected error: %v", err)
	}
	assertBody(t, stub.lastBody, "timezone", "Asia/Shanghai")
}

func TestAutomationCreateExplicitEmptyTimezoneIsOmitted(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"automation", "create",
		"--name", "Daily SRE brief",
		"--prompt", "Summarize yesterday's incidents",
		"--timezone", "",
		"--json",
	)
	if err != nil {
		t.Fatalf("[automation-create-empty-timezone] unexpected error: %v", err)
	}
	// CreateRequest.Timezone is a string with omitempty, so an explicit ""
	// is dropped. Update's pointer is what sends an empty string.
	if _, ok := stub.lastBody["timezone"]; ok {
		t.Fatalf("[automation-create-empty-timezone] empty string must not be on the wire, body=%#v", stub.lastBody)
	}
}

func TestAutomationCreateOmitsTimezone(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"automation", "create",
		"--name", "Daily SRE brief",
		"--prompt", "Summarize yesterday's incidents",
		"--json",
	)
	if err != nil {
		t.Fatalf("[automation-create-omit-timezone] unexpected error: %v", err)
	}
	if _, ok := stub.lastBody["timezone"]; ok {
		t.Fatalf("[automation-create-omit-timezone] timezone must be omitted, body=%#v", stub.lastBody)
	}
}

func TestAutomationUpdateTimezone(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"automation", "update", "auto_123",
		"--timezone", "Asia/Tokyo",
		"--json",
	)
	if err != nil {
		t.Fatalf("[automation-update-timezone] unexpected error: %v", err)
	}
	assertBody(t, stub.lastBody, "rule_id", "auto_123")
	assertBody(t, stub.lastBody, "timezone", "Asia/Tokyo")
	if _, ok := stub.lastBody["cron_expr"]; ok {
		t.Fatalf("[automation-update-timezone] cron_expr must stay omitted, body=%#v", stub.lastBody)
	}
}

func TestAutomationUpdateEmptyTimezoneIsSent(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"automation", "update", "auto_123",
		"--timezone", "",
		"--json",
	)
	if err != nil {
		t.Fatalf("[automation-update-empty-timezone] unexpected error: %v", err)
	}
	assertBody(t, stub.lastBody, "timezone", "")
}

func TestAutomationUpdateOmitsTimezone(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"automation", "update", "auto_123",
		"--name", "Daily brief v2",
		"--json",
	)
	if err != nil {
		t.Fatalf("[automation-update-omit-timezone] unexpected error: %v", err)
	}
	assertBody(t, stub.lastBody, "name", "Daily brief v2")
	if _, ok := stub.lastBody["timezone"]; ok {
		t.Fatalf("[automation-update-omit-timezone] timezone must stay nil, body=%#v", stub.lastBody)
	}
}

func TestSafariAutomationRuleUpdateTimezone(t *testing.T) {
	saveAndResetGlobals(t)

	help, err := execCommand("safari", "automation-rule-update", "--help")
	if err != nil {
		t.Fatalf("[safari-update-timezone] help: %v", err)
	}
	if !strings.Contains(help, "--timezone") {
		t.Fatalf("[safari-update-timezone] help missing --timezone\n%s", help)
	}

	stub := newGFStub(t)
	_, err = execCommand(
		"safari", "automation-rule-update", "arule_1",
		"--name", "kept",
		"--json",
	)
	if err != nil {
		t.Fatalf("[safari-update-timezone] omit: %v", err)
	}
	if _, ok := stub.lastBody["timezone"]; ok {
		t.Fatalf("[safari-update-timezone] omitted flag must not send timezone, body=%#v", stub.lastBody)
	}

	_, err = execCommand(
		"safari", "automation-rule-update", "arule_1",
		"--timezone", "",
		"--json",
	)
	if err != nil {
		t.Fatalf("[safari-update-timezone] empty: %v", err)
	}
	assertBody(t, stub.lastBody, "timezone", "")

	_, err = execCommand(
		"safari", "automation-rule-update", "arule_1",
		"--timezone", "Europe/London",
		"--json",
	)
	if err != nil {
		t.Fatalf("[safari-update-timezone] named: %v", err)
	}
	assertBody(t, stub.lastBody, "timezone", "Europe/London")
}

func TestAutomationCreateHTTPPostOnly(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"automation", "create",
		"--name", "Webhook triage",
		"--http-post-trigger",
		"--prompt", "Handle the posted payload",
		"--json",
	)
	if err != nil {
		t.Fatalf("[automation-create-post-only] unexpected error: %v", err)
	}
	assertBody(t, stub.lastBody, "cron_expr", automationHTTPPostOnlyCron)
	assertBody(t, stub.lastBody, "enabled", true)
	assertBody(t, stub.lastBody, "schedule_trigger_enabled", false)
	assertBody(t, stub.lastBody, "http_post_trigger_enabled", true)
}

func TestAutomationUpdateMutableFields(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stdinReader = strings.NewReader("updated prompt\n")

	_, err := execCommand(
		"automation", "update", "auto_123",
		"--disable",
		"--disable-schedule",
		"--enable-http-post-trigger",
		"--rotate-http-post-token",
		"--prompt-file", "-",
		"--json",
	)
	if err != nil {
		t.Fatalf("[automation-update] unexpected error: %v", err)
	}
	if stub.lastPath != "/safari/automation/rule/update" {
		t.Fatalf("[automation-update] path = %q", stub.lastPath)
	}
	assertBody(t, stub.lastBody, "rule_id", "auto_123")
	assertBody(t, stub.lastBody, "enabled", false)
	assertBody(t, stub.lastBody, "schedule_trigger_enabled", false)
	assertBody(t, stub.lastBody, "http_post_trigger_enabled", true)
	assertBody(t, stub.lastBody, "rotate_http_post_trigger_token", true)
	assertBody(t, stub.lastBody, "prompt", "updated prompt")
	if _, ok := stub.lastBody["team_id"]; ok {
		t.Fatalf("[automation-update] team_id must not be sent by the friendly update command: %#v", stub.lastBody)
	}
}

func TestAutomationUpdateDoesNotExposeScopeFlag(t *testing.T) {
	saveAndResetGlobals(t)
	newGFStub(t)

	_, err := execCommand("automation", "update", "auto_123", "--team-id", "456")
	if err == nil {
		t.Fatal("[automation-update-scope] expected unknown flag error")
	}
	if !strings.Contains(err.Error(), "unknown flag: --team-id") {
		t.Fatalf("[automation-update-scope] err = %v", err)
	}
}

func TestAutomationFireSendsBearerToken(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"type":        "routine_fire",
		"session_id":  "sess_123",
		"session_url": "/safari/session/sess_123",
	}

	_, err := execCommand(
		"automation", "fire", "auttrig_123",
		"--token", "token-123",
		"--text", "manual test",
		"--json",
	)
	if err != nil {
		t.Fatalf("[automation-fire] unexpected error: %v", err)
	}
	if stub.lastPath != "/safari/automation/triggers/auttrig_123/fire" {
		t.Fatalf("[automation-fire] path = %q", stub.lastPath)
	}
	if stub.lastAuthorization != "Bearer token-123" {
		t.Fatalf("[automation-fire] authorization = %q", stub.lastAuthorization)
	}
	assertBody(t, stub.lastBody, "text", "manual test")
}

func TestAutomationFireDoesNotExposeIgnoredDedupKey(t *testing.T) {
	saveAndResetGlobals(t)
	newGFStub(t)

	_, err := execCommand(
		"automation", "fire", "auttrig_123",
		"--token", "token-123",
		"--dedup-key", "once",
	)
	if err == nil || !strings.Contains(err.Error(), "unknown flag: --dedup-key") {
		t.Fatalf("expected dedup-key to be rejected, got %v", err)
	}
}

func TestSafariAutomationTriggerFirePathCommand(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"safari", "automation-triggers-{trigger_id}-fire", "auttrig_123",
		"--token", "token-123",
		"--data", `{"text":"from data"}`,
		"--json",
	)
	if err != nil {
		t.Fatalf("[safari-automation-trigger-fire] unexpected error: %v", err)
	}
	if stub.lastPath != "/safari/automation/triggers/auttrig_123/fire" {
		t.Fatalf("[safari-automation-trigger-fire] path = %q", stub.lastPath)
	}
	if stub.lastAuthorization != "Bearer token-123" {
		t.Fatalf("[safari-automation-trigger-fire] authorization = %q", stub.lastAuthorization)
	}
	assertBody(t, stub.lastBody, "text", "from data")
}

func TestAutomationCronHelpers(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		at       string
		weekday  string
		cron     string
		want     string
	}{
		{name: "hourly", schedule: "hourly", at: "00:15", want: "15 * * * *"},
		{name: "daily default", schedule: "daily", want: "0 9 * * *"},
		{name: "weekly", schedule: "weekly", at: "10:05", weekday: "fri", want: "5 10 * * 5"},
		{name: "cron", cron: "7 8 * * 1", want: "7 8 * * 1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveAutomationCron(tc.schedule, tc.at, tc.weekday, tc.cron)
			if err != nil {
				t.Fatalf("resolveAutomationCron() unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("resolveAutomationCron() = %q, want %q", got, tc.want)
			}
		})
	}
}

func assertBody(t *testing.T, body map[string]any, key string, want any) {
	t.Helper()
	got, ok := body[key]
	if !ok {
		t.Fatalf("missing body[%q] in %#v", key, body)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "body[%q] = %#v, want %#v\nfull body: %#v", key, got, want, body)
		t.Fatal(buf.String())
	}
}
