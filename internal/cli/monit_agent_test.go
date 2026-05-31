package cli

import (
	"fmt"
	"strings"
	"testing"
)

// --- flag surface ---------------------------------------------------------

func TestMonitAgentCatalogFlags(t *testing.T) {
	cmd := newMonitAgentCatalogCmd()
	for _, name := range []string{"target-kind", "target-locator"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s missing", name)
		}
	}
}

func TestMonitAgentInvokeFlags(t *testing.T) {
	cmd := newMonitAgentInvokeCmd()
	for _, name := range []string{"target-kind", "target-locator", "tool-spec"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s missing", name)
		}
	}
}

// --- monit-agent catalog --------------------------------------------------

func TestMonitAgentCatalogHappyPath(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"tools": []map[string]any{
			{"name": "ps_top", "description": "Top processes by CPU"},
		},
	}

	_, err := execCommand(
		"monit-agent", "catalog",
		"--target-kind", "host",
		"--target-locator", "10.0.1.5",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastPath != "/monit/tools/catalog" {
		t.Fatalf("expected /monit/tools/catalog, got %q", stub.lastPath)
	}
	if stub.lastBody["target_kind"] != "host" || stub.lastBody["target_locator"] != "10.0.1.5" {
		t.Errorf("unexpected catalog input: %#v", stub.lastBody)
	}
}

func TestMonitAgentCatalogOmitsKind(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-agent", "catalog",
		"--target-locator", "web-01",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.requests == 0 {
		t.Fatal("expected catalog request to be sent")
	}
	if _, ok := stub.lastBody["target_kind"]; ok {
		t.Errorf("expected target_kind omitted, got %v", stub.lastBody["target_kind"])
	}
	if stub.lastBody["target_locator"] != "web-01" {
		t.Errorf("expected locator web-01, got %v", stub.lastBody["target_locator"])
	}
}

func TestMonitAgentCatalogRequiresLocator(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand("monit-agent", "catalog", "--target-kind", "host")
	if err == nil {
		t.Fatal("expected required-flag error, got nil")
	}
	if !strings.Contains(err.Error(), "--target-locator") {
		t.Errorf("expected error to mention --target-locator, got %q", err.Error())
	}
	if stub.requests != 0 {
		t.Errorf("catalog should not have been called: %d request(s)", stub.requests)
	}
}

// --- monit-agent invoke ---------------------------------------------------

func TestMonitAgentInvokeHappyPath(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-agent", "invoke",
		"--target-kind", "host",
		"--target-locator", "10.0.1.5",
		"--tool-spec", `name=ps_top,params={"limit":5}`,
		"--tool-spec", "name=uptime",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastPath != "/monit/tools/invoke" {
		t.Fatalf("expected /monit/tools/invoke, got %q", stub.lastPath)
	}
	if stub.lastBody["target_kind"] != "host" || stub.lastBody["target_locator"] != "10.0.1.5" {
		t.Errorf("unexpected invoke target: %#v", stub.lastBody)
	}
	tools, _ := stub.lastBody["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	tool0, _ := tools[0].(map[string]any)
	if tool0["tool"] != "ps_top" {
		t.Errorf("expected first tool ps_top, got %v", tool0["tool"])
	}
	params0, _ := tool0["params"].(map[string]any)
	if fmt.Sprint(params0["limit"]) != "5" {
		t.Errorf("expected ps_top params limit=5, got %#v", tool0["params"])
	}
	tool1, _ := tools[1].(map[string]any)
	if tool1["tool"] != "uptime" {
		t.Errorf("expected second tool uptime, got %v", tool1["tool"])
	}
}

func TestMonitAgentInvokeOmitsKind(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-agent", "invoke",
		"--target-locator", "10.0.1.5",
		"--tool-spec", "name=uptime",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.requests == 0 {
		t.Fatal("expected invoke request to be sent")
	}
	if _, ok := stub.lastBody["target_kind"]; ok {
		t.Errorf("expected target_kind omitted, got %v", stub.lastBody["target_kind"])
	}
}

func TestMonitAgentInvokeRequiresLocator(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-agent", "invoke",
		"--tool-spec", "name=ps_top",
	)
	if err == nil {
		t.Fatal("expected required-flag error, got nil")
	}
	if !strings.Contains(err.Error(), "--target-locator") {
		t.Errorf("expected error to mention --target-locator, got %q", err.Error())
	}
	if stub.requests != 0 {
		t.Errorf("invoke should not have been called: %d request(s)", stub.requests)
	}
}

func TestMonitAgentInvokeRequiresToolSpec(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-agent", "invoke",
		"--target-locator", "10.0.1.5",
	)
	if err == nil {
		t.Fatal("expected required-flag error, got nil")
	}
	if !strings.Contains(err.Error(), "--tool-spec") {
		t.Errorf("expected error to mention --tool-spec, got %q", err.Error())
	}
	if stub.requests != 0 {
		t.Errorf("invoke should not have been called: %d request(s)", stub.requests)
	}
}

func TestMonitAgentInvokeRejectsMoreThan8Specs(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	args := []string{
		"monit-agent", "invoke",
		"--target-locator", "10.0.1.5",
	}
	for i := 0; i < 9; i++ {
		args = append(args, "--tool-spec", "name=t"+string(rune('0'+i)))
	}

	_, err := execCommand(args...)
	if err == nil {
		t.Fatal("expected too-many-tools error, got nil")
	}
	if !strings.Contains(err.Error(), "up to 8") {
		t.Errorf("expected error to mention 'up to 8', got %q", err.Error())
	}
	if stub.requests != 0 {
		t.Errorf("invoke should not have been called: %d request(s)", stub.requests)
	}
}

func TestMonitAgentInvokeMalformedSpec(t *testing.T) {
	cases := []struct {
		name string
		spec string
	}{
		{"missing name=", "params={}"},
		{"missing equals", "no-equals-sign"},
		{"unknown key", "namez=foo,params={}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			_, err := execCommand(
				"monit-agent", "invoke",
				"--target-locator", "10.0.1.5",
				"--tool-spec", tc.spec,
			)
			if err == nil {
				t.Fatal("expected parse error, got nil")
			}
			if !strings.Contains(err.Error(), "--tool-spec") {
				t.Errorf("expected error to mention --tool-spec, got %q", err.Error())
			}
			if stub.requests != 0 {
				t.Errorf("invoke should not have been called: %d request(s)", stub.requests)
			}
		})
	}
}
