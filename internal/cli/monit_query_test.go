package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
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

// --- shared mock plumbing -------------------------------------------------

type mockMonitQuery struct {
	mockClient

	diagnoseInput *flashduty.MonitQueryDiagnoseInput
	diagnoseOut   *flashduty.MonitQueryDiagnoseOutput
	diagnoseErr   error

	rowsInput *flashduty.MonitQueryRowsInput
	rowsOut   *flashduty.MonitQueryRowsOutput
	rowsErr   error
}

func (m *mockMonitQuery) MonitQueryDiagnose(_ context.Context, input *flashduty.MonitQueryDiagnoseInput) (*flashduty.MonitQueryDiagnoseOutput, error) {
	copied := *input
	m.diagnoseInput = &copied
	if m.diagnoseErr != nil {
		return nil, m.diagnoseErr
	}
	if m.diagnoseOut != nil {
		return m.diagnoseOut, nil
	}
	return &flashduty.MonitQueryDiagnoseOutput{Operation: "log_patterns"}, nil
}

func (m *mockMonitQuery) MonitQueryRows(_ context.Context, input *flashduty.MonitQueryRowsInput) (*flashduty.MonitQueryRowsOutput, error) {
	copied := *input
	m.rowsInput = &copied
	if m.rowsErr != nil {
		return nil, m.rowsErr
	}
	if m.rowsOut != nil {
		return m.rowsOut, nil
	}
	return &flashduty.MonitQueryRowsOutput{}, nil
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
	mock := &mockMonitQuery{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	_, err := execCommand(
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
	if mock.rowsInput == nil {
		t.Fatal("expected MonitQueryRows to be called")
	}
	got := mock.rowsInput
	if got.DsType != "prometheus" || got.DsName != "prom-prod" || got.Expr != "up" {
		t.Errorf("unexpected rows input: %+v", got)
	}
	if got.Args["step"] != "15s" || got.Args["tenant"] != "acme" {
		t.Errorf("expected args step=15s tenant=acme, got %#v", got.Args)
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
			mock := &mockMonitQuery{}
			newClientFn = func() (flashdutyClient, error) { return mock, nil }

			_, err := execCommand(tc.args...)
			if err == nil {
				t.Fatal("expected required-flag error, got nil")
			}
			if !strings.Contains(err.Error(), "required") {
				t.Errorf("expected error to mention 'required', got %q", err.Error())
			}
			if mock.rowsInput != nil {
				t.Errorf("MonitQueryRows should not have been called: %#v", mock.rowsInput)
			}
		})
	}
}

func TestMonitQueryRowsInvalidArgs(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockMonitQuery{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

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
	if mock.rowsInput != nil {
		t.Errorf("MonitQueryRows should not have been called: %#v", mock.rowsInput)
	}
}
