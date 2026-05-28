package cli

import (
	"context"
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
	mock := &mockMonitQuery{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

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
	if mock.diagnoseInput == nil {
		t.Fatal("expected MonitQueryDiagnose to be called")
	}
	got := mock.diagnoseInput
	if got.DsType != "victorialogs" || got.DsName != "vl-prod" {
		t.Errorf("unexpected ds fields: %+v", got)
	}
	if got.Input.Query != `{app="api"}` {
		t.Errorf("expected input query %q, got %q", `{app="api"}`, got.Input.Query)
	}
	if got.Operation != "log_patterns" {
		t.Errorf("expected operation log_patterns, got %q", got.Operation)
	}
	if got.MaxLogsScanned != 5000 || got.MaxPatterns != 10 || got.TimeoutSeconds != 20 {
		t.Errorf("unexpected caps: logs=%d patterns=%d timeout=%d",
			got.MaxLogsScanned, got.MaxPatterns, got.TimeoutSeconds)
	}
	if got.TimeStart == 0 || got.TimeEnd == 0 {
		t.Errorf("expected non-zero default time range, got start=%d end=%d",
			got.TimeStart, got.TimeEnd)
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
			mock := &mockMonitQuery{}
			newClientFn = func() (flashdutyClient, error) { return mock, nil }

			_, err := execCommand(tc.args...)
			if err == nil {
				t.Fatal("expected required-flag error, got nil")
			}
			if !strings.Contains(err.Error(), "required") {
				t.Errorf("expected error to mention 'required', got %q", err.Error())
			}
			if mock.diagnoseInput != nil {
				t.Errorf("MonitQueryDiagnose should not have been called: %#v", mock.diagnoseInput)
			}
		})
	}
}

func TestMonitQueryDiagnoseInvalidTimeStart(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockMonitQuery{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

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
	if mock.diagnoseInput != nil {
		t.Errorf("MonitQueryDiagnose should not have been called: %#v", mock.diagnoseInput)
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
