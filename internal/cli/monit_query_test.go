package cli

import (
	"fmt"
	"strings"
	"testing"
)

func TestMonitQueryDiagnoseFlags(t *testing.T) {
	cmd := newMonitQueryDiagnoseCmd()
	for _, name := range []string{
		"ds-type", "ds-name", "time-start", "time-end",
		"input-query", "operation",
		"max-logs", "max-patterns", "timeout-seconds",
	} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s missing", name)
		}
	}
}

func TestMonitQueryRowsFlags(t *testing.T) {
	cmd := newMonitQueryRowsCmd()
	for _, name := range []string{"ds-type", "ds-name", "expr", "args"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s missing", name)
		}
	}
}

// --- monit-query diagnose -------------------------------------------------

func TestMonitQueryDiagnoseHappyPath(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"operation": "log_patterns"}

	_, err := execCommand(
		"monit-query", "diagnose",
		"--ds-type", "victorialogs",
		"--ds-name", "vl-prod",
		"--input-query", `{app="api"}`,
		"--operation", "log_patterns",
		"--max-logs", "5000",
		"--max-patterns", "10",
		"--timeout-seconds", "20",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastPath != "/monit/query/diagnose" {
		t.Fatalf("expected /monit/query/diagnose, got %q", stub.lastPath)
	}
	body := stub.lastBody
	if body["ds_type"] != "victorialogs" || body["ds_name"] != "vl-prod" {
		t.Errorf("unexpected ds fields: %#v", body)
	}
	input, _ := body["input"].(map[string]any)
	if input["query"] != `{app="api"}` {
		t.Errorf("expected input query %q, got %v", `{app="api"}`, input["query"])
	}
	if body["operation"] != "log_patterns" {
		t.Errorf("expected operation log_patterns, got %v", body["operation"])
	}
	options, _ := body["options"].(map[string]any)
	if fmt.Sprint(options["max_logs_scanned"]) != "5000" ||
		fmt.Sprint(options["max_patterns"]) != "10" ||
		fmt.Sprint(options["timeout_seconds"]) != "20" {
		t.Errorf("unexpected caps: %#v", options)
	}
	timeRange, _ := body["time_range"].(map[string]any)
	if fmt.Sprint(timeRange["start"]) == "0" || fmt.Sprint(timeRange["start"]) == "<nil>" ||
		fmt.Sprint(timeRange["end"]) == "0" || fmt.Sprint(timeRange["end"]) == "<nil>" {
		t.Errorf("expected non-zero default time range, got %#v", timeRange)
	}
}

func TestMonitQueryDiagnoseRequiredFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{
			name: "missing ds-type",
			args: []string{
				"monit-query", "diagnose",
				"--ds-name", "vl-prod",
				"--input-query", `{app="api"}`,
			},
		},
		{
			name: "missing ds-name",
			args: []string{
				"monit-query", "diagnose",
				"--ds-type", "victorialogs",
				"--input-query", `{app="api"}`,
			},
		},
		{
			name: "missing input-query",
			args: []string{
				"monit-query", "diagnose",
				"--ds-type", "victorialogs",
				"--ds-name", "vl-prod",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			_, err := execCommand(tc.args...)
			if err == nil {
				t.Fatal("expected required-flag error, got nil")
			}
			if !strings.Contains(err.Error(), "required") {
				t.Errorf("expected error to mention 'required', got %q", err.Error())
			}
			if stub.requests != 0 {
				t.Errorf("diagnose should not have been called: %d request(s)", stub.requests)
			}
		})
	}
}

func TestMonitQueryDiagnoseInvalidTimeStart(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-query", "diagnose",
		"--ds-type", "victorialogs",
		"--ds-name", "vl-prod",
		"--input-query", `{app="api"}`,
		"--time-start", "not-a-time",
	)
	if err == nil {
		t.Fatal("expected error for invalid --time-start, got nil")
	}
	if !strings.Contains(err.Error(), "--time-start") {
		t.Errorf("expected error to mention --time-start, got %q", err.Error())
	}
	if stub.requests != 0 {
		t.Errorf("diagnose should not have been called: %d request(s)", stub.requests)
	}
}

// --- monit-query rows -----------------------------------------------------

func TestMonitQueryRowsHappyPath(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	// rows is a raw datasource passthrough: the response envelope "data" is a
	// JSON array of QueryRow ({fields,values}) objects, decoded into
	// QueryRowsResponse ([]QueryRow) and re-marshalled verbatim to the writer.
	stub.data = []any{
		map[string]any{
			"fields": map[string]any{"instance": "node-1"},
			"values": map[string]any{"__value__": 1},
		},
	}

	out, err := execCommand(
		"monit-query", "rows",
		"--ds-type", "prometheus",
		"--ds-name", "prom-prod",
		"--expr", "up",
		"--args", "step=15s",
		"--args", "tenant=acme",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastPath != "/monit/query/rows" {
		t.Fatalf("expected /monit/query/rows, got %q", stub.lastPath)
	}
	body := stub.lastBody
	if body["ds_type"] != "prometheus" || body["ds_name"] != "prom-prod" || body["expr"] != "up" {
		t.Errorf("unexpected rows input: %#v", body)
	}
	args, _ := body["args"].(map[string]any)
	if args["step"] != "15s" || args["tenant"] != "acme" {
		t.Errorf("expected args step=15s tenant=acme, got %#v", args)
	}
	// The rendered output is the re-marshalled row array (passthrough shape).
	if !strings.Contains(out, "node-1") || !strings.Contains(out, "__value__") {
		t.Errorf("expected rendered rows to carry the datasource payload, got:\n%s", out)
	}
}

func TestMonitQueryRowsRequiredFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{
			name: "missing ds-type",
			args: []string{
				"monit-query", "rows",
				"--ds-name", "prom-prod",
				"--expr", "up",
			},
		},
		{
			name: "missing ds-name",
			args: []string{
				"monit-query", "rows",
				"--ds-type", "prometheus",
				"--expr", "up",
			},
		},
		{
			name: "missing expr",
			args: []string{
				"monit-query", "rows",
				"--ds-type", "prometheus",
				"--ds-name", "prom-prod",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			_, err := execCommand(tc.args...)
			if err == nil {
				t.Fatal("expected required-flag error, got nil")
			}
			if !strings.Contains(err.Error(), "required") {
				t.Errorf("expected error to mention 'required', got %q", err.Error())
			}
			if stub.requests != 0 {
				t.Errorf("rows should not have been called: %d request(s)", stub.requests)
			}
		})
	}
}

func TestMonitQueryRowsInvalidArgs(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-query", "rows",
		"--ds-type", "prometheus",
		"--ds-name", "prom-prod",
		"--expr", "up",
		"--args", "no-equals-sign",
	)
	if err == nil {
		t.Fatal("expected error for malformed --args, got nil")
	}
	if !strings.Contains(err.Error(), "--args") {
		t.Errorf("expected error to mention --args, got %q", err.Error())
	}
	if stub.requests != 0 {
		t.Errorf("rows should not have been called: %d request(s)", stub.requests)
	}
}
