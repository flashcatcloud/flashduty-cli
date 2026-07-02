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

func TestAutomationScheduleHelpDocumentsUTC(t *testing.T) {
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
			"UTC",
			"Convert local wall-clock requests to UTC before passing --at or --cron-expr.",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("%v help missing %q\n%s", args, want, out)
			}
		}
	}
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
		"rule_id":      "auto_123",
		"run_id":       "run_123",
		"status":       "queued",
		"trigger_kind": "http_post",
	}

	_, err := execCommand(
		"automation", "fire", "auttrig_123",
		"--token", "token-123",
		"--text", "manual test",
		"--dedup-key", "once",
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
	assertBody(t, stub.lastBody, "dedup_key", "once")
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
