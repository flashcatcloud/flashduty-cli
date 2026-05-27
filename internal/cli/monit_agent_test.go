package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
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

// --- shared mock plumbing -------------------------------------------------

type mockMonitAgent struct {
	mockClient

	catalogInput *flashduty.MonitAgentCatalogInput
	catalogOut   *flashduty.MonitAgentCatalogOutput
	catalogErr   error

	invokeInput *flashduty.MonitAgentInvokeInput
	invokeOut   *flashduty.MonitAgentInvokeOutput
	invokeErr   error
}

func (m *mockMonitAgent) MonitAgentCatalog(_ context.Context, input *flashduty.MonitAgentCatalogInput) (*flashduty.MonitAgentCatalogOutput, error) {
	copied := *input
	m.catalogInput = &copied
	if m.catalogErr != nil {
		return nil, m.catalogErr
	}
	if m.catalogOut != nil {
		return m.catalogOut, nil
	}
	return &flashduty.MonitAgentCatalogOutput{}, nil
}

func (m *mockMonitAgent) MonitAgentInvoke(_ context.Context, input *flashduty.MonitAgentInvokeInput) (*flashduty.MonitAgentInvokeOutput, error) {
	copied := *input
	copied.Tools = append([]flashduty.MonitAgentInvokeTool(nil), input.Tools...)
	m.invokeInput = &copied
	if m.invokeErr != nil {
		return nil, m.invokeErr
	}
	if m.invokeOut != nil {
		return m.invokeOut, nil
	}
	return &flashduty.MonitAgentInvokeOutput{}, nil
}

// --- monit-agent catalog --------------------------------------------------

func TestMonitAgentCatalogHappyPath(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockMonitAgent{
		catalogOut: &flashduty.MonitAgentCatalogOutput{
			Tools: []flashduty.MonitAgentTool{
				{Name: "ps_top", Description: "Top processes by CPU"},
			},
		},
	}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	_, err := execCommand(
		"monit-agent", "catalog",
		"--target-kind", "host",
		"--target-locator", "10.0.1.5",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.catalogInput == nil {
		t.Fatal("expected MonitAgentCatalog to be called")
	}
	if mock.catalogInput.TargetKind != "host" || mock.catalogInput.TargetLocator != "10.0.1.5" {
		t.Errorf("unexpected catalog input: %+v", mock.catalogInput)
	}
}

func TestMonitAgentCatalogOmitsKind(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockMonitAgent{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	_, err := execCommand(
		"monit-agent", "catalog",
		"--target-locator", "web-01",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.catalogInput == nil {
		t.Fatal("expected MonitAgentCatalog to be called")
	}
	if mock.catalogInput.TargetKind != "" {
		t.Errorf("expected empty target-kind, got %q", mock.catalogInput.TargetKind)
	}
	if mock.catalogInput.TargetLocator != "web-01" {
		t.Errorf("expected locator web-01, got %q", mock.catalogInput.TargetLocator)
	}
}

func TestMonitAgentCatalogRequiresLocator(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockMonitAgent{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	_, err := execCommand("monit-agent", "catalog", "--target-kind", "host")
	if err == nil {
		t.Fatal("expected required-flag error, got nil")
	}
	if !strings.Contains(err.Error(), "--target-locator") {
		t.Errorf("expected error to mention --target-locator, got %q", err.Error())
	}
	if mock.catalogInput != nil {
		t.Errorf("MonitAgentCatalog should not have been called: %#v", mock.catalogInput)
	}
}

// --- monit-agent invoke ---------------------------------------------------

func TestMonitAgentInvokeHappyPath(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockMonitAgent{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

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
	if mock.invokeInput == nil {
		t.Fatal("expected MonitAgentInvoke to be called")
	}
	got := mock.invokeInput
	if got.TargetKind != "host" || got.TargetLocator != "10.0.1.5" {
		t.Errorf("unexpected invoke target: %+v", got)
	}
	if len(got.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(got.Tools))
	}
	if got.Tools[0].Tool != "ps_top" {
		t.Errorf("expected first tool ps_top, got %q", got.Tools[0].Tool)
	}
	if string(got.Tools[0].Params) != `{"limit":5}` {
		t.Errorf("expected ps_top params %q, got %q", `{"limit":5}`, string(got.Tools[0].Params))
	}
	if got.Tools[1].Tool != "uptime" {
		t.Errorf("expected second tool uptime, got %q", got.Tools[1].Tool)
	}
	// default params for a name-only spec must be valid JSON `{}`, so the
	// server-side decoder accepts it.
	if !json.Valid(got.Tools[1].Params) {
		t.Errorf("uptime params not valid JSON: %q", string(got.Tools[1].Params))
	}
}

func TestMonitAgentInvokeOmitsKind(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockMonitAgent{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

	_, err := execCommand(
		"monit-agent", "invoke",
		"--target-locator", "10.0.1.5",
		"--tool-spec", "name=uptime",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.invokeInput == nil {
		t.Fatal("expected MonitAgentInvoke to be called")
	}
	if mock.invokeInput.TargetKind != "" {
		t.Errorf("expected empty target-kind, got %q", mock.invokeInput.TargetKind)
	}
}

func TestMonitAgentInvokeRequiresLocator(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockMonitAgent{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

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
	if mock.invokeInput != nil {
		t.Errorf("MonitAgentInvoke should not have been called: %#v", mock.invokeInput)
	}
}

func TestMonitAgentInvokeRequiresToolSpec(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockMonitAgent{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

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
	if mock.invokeInput != nil {
		t.Errorf("MonitAgentInvoke should not have been called: %#v", mock.invokeInput)
	}
}

func TestMonitAgentInvokeRejectsMoreThan8Specs(t *testing.T) {
	saveAndResetGlobals(t)
	mock := &mockMonitAgent{}
	newClientFn = func() (flashdutyClient, error) { return mock, nil }

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
	if mock.invokeInput != nil {
		t.Errorf("MonitAgentInvoke should not have been called: %#v", mock.invokeInput)
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
			mock := &mockMonitAgent{}
			newClientFn = func() (flashdutyClient, error) { return mock, nil }

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
			if mock.invokeInput != nil {
				t.Errorf("MonitAgentInvoke should not have been called: %#v", mock.invokeInput)
			}
		})
	}
}
