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
	for _, name := range []string{"target-kind", "target-locator", "data"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s missing", name)
		}
	}
	// The bespoke --tool-spec mini-DSL is gone; tools come via --data.
	if cmd.Flags().Lookup("tool-spec") != nil {
		t.Errorf("flag --tool-spec should have been removed")
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
		"--data", `{"tools":[{"tool":"ps_top","params":{"limit":5}},{"tool":"uptime"}]}`,
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
	// A no-arg tool defaults to params {} client-side; the SDK's `omitempty`
	// then drops the empty map on the wire, so no "params" key is sent — the
	// same shape the old --tool-spec path produced.
	if _, ok := tool1["params"]; ok {
		t.Errorf("expected uptime to omit params on the wire, got %#v", tool1["params"])
	}
}

// Regression for the original bug: a params JSON value containing an internal
// comma (the SQL case) used to shatter under the comma-split --tool-spec DSL.
// Via the --data body it round-trips intact.
func TestMonitAgentInvokeParamsWithInternalComma(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	const sql = "SELECT a, b FROM t WHERE s='RUNNING'"
	_, err := execCommand(
		"monit-agent", "invoke",
		"--target-locator", "db-1",
		"--data", `{"tools":[{"tool":"mysql.query","params":{"sql":"`+sql+`","max_rows":50}}]}`,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tools, _ := stub.lastBody["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	tool0, _ := tools[0].(map[string]any)
	if tool0["tool"] != "mysql.query" {
		t.Errorf("expected mysql.query, got %v", tool0["tool"])
	}
	params0, _ := tool0["params"].(map[string]any)
	if params0["sql"] != sql {
		t.Errorf("expected sql %q to survive intact, got %#v", sql, params0["sql"])
	}
	if fmt.Sprint(params0["max_rows"]) != "50" {
		t.Errorf("expected max_rows=50, got %#v", params0["max_rows"])
	}
}

// --data - reads the JSON body from stdin, the canonical heredoc form for
// quoted/comma SQL.
func TestMonitAgentInvokeDataFromStdin(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	const sql = "SELECT a, b FROM t WHERE s='RUNNING'"
	stdinReader = strings.NewReader(`{"tools":[{"tool":"mysql.query","params":{"sql":"` + sql + `","max_rows":50}}]}`)

	_, err := execCommand(
		"monit-agent", "invoke",
		"--target-locator", "db-1",
		"--data", "-",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tools, _ := stub.lastBody["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	tool0, _ := tools[0].(map[string]any)
	params0, _ := tool0["params"].(map[string]any)
	if params0["sql"] != sql {
		t.Errorf("expected sql %q from stdin, got %#v", sql, params0["sql"])
	}
}

func TestMonitAgentInvokeOmitsKind(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-agent", "invoke",
		"--target-locator", "10.0.1.5",
		"--data", `{"tools":[{"tool":"uptime"}]}`,
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

// Typed --target-* flags override the matching keys in --data.
func TestMonitAgentInvokeFlagsOverrideData(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-agent", "invoke",
		"--target-kind", "host",
		"--target-locator", "10.0.1.5",
		"--data", `{"target_kind":"mysql","target_locator":"ignored","tools":[{"tool":"uptime"}]}`,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastBody["target_kind"] != "host" {
		t.Errorf("expected typed --target-kind to win, got %v", stub.lastBody["target_kind"])
	}
	if stub.lastBody["target_locator"] != "10.0.1.5" {
		t.Errorf("expected typed --target-locator to win, got %v", stub.lastBody["target_locator"])
	}
}

func TestMonitAgentInvokeRequiresLocator(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-agent", "invoke",
		"--data", `{"tools":[{"tool":"ps_top"}]}`,
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

func TestMonitAgentInvokeRequiresTools(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-agent", "invoke",
		"--target-locator", "10.0.1.5",
	)
	if err == nil {
		t.Fatal("expected missing-tools error, got nil")
	}
	if !strings.Contains(err.Error(), "tools") {
		t.Errorf("expected error to mention tools, got %q", err.Error())
	}
	if stub.requests != 0 {
		t.Errorf("invoke should not have been called: %d request(s)", stub.requests)
	}
}

func TestMonitAgentInvokeRejectsMoreThan8Tools(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	specs := make([]string, 9)
	for i := range specs {
		specs[i] = fmt.Sprintf(`{"tool":"t%d"}`, i)
	}
	data := `{"tools":[` + strings.Join(specs, ",") + `]}`

	_, err := execCommand(
		"monit-agent", "invoke",
		"--target-locator", "10.0.1.5",
		"--data", data,
	)
	if err == nil {
		t.Fatal("expected too-many-tools error, got nil")
	}
	if !strings.Contains(err.Error(), "at most 8") {
		t.Errorf("expected error to mention 'at most 8', got %q", err.Error())
	}
	if stub.requests != 0 {
		t.Errorf("invoke should not have been called: %d request(s)", stub.requests)
	}
}

func TestMonitAgentInvokeMalformedData(t *testing.T) {
	cases := []struct {
		name     string
		data     string
		wantText string
	}{
		{"invalid json", `{"tools":[`, "invalid --data JSON"},
		{"tools not array", `{"tools":{"tool":"x"}}`, "must be a JSON array"},
		{"tool entry not object", `{"tools":["x"]}`, "must be an object"},
		{"missing tool name", `{"tools":[{"params":{}}]}`, "missing a non-empty"},
		{"params not object", `{"tools":[{"tool":"x","params":[]}]}`, "params must be a JSON object"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			_, err := execCommand(
				"monit-agent", "invoke",
				"--target-locator", "10.0.1.5",
				"--data", tc.data,
			)
			if err == nil {
				t.Fatal("expected parse error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantText) {
				t.Errorf("expected error to mention %q, got %q", tc.wantText, err.Error())
			}
			if stub.requests != 0 {
				t.Errorf("invoke should not have been called: %d request(s)", stub.requests)
			}
		})
	}
}
