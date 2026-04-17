//go:build e2e

package e2e_test

import (
	"testing"
)

// Test 270: get preset slack
func TestTemplateGetPresetSlack(t *testing.T) {
	t.Skip("requires TemplateID field not yet supported by SDK")
	r := runCLI(t, "template", "get-preset", "--channel", "slack")
	requireSuccess(t, r)
	// Template code should contain template syntax.
	if r.Stdout == "" {
		t.Error("expected template code output, got empty")
	}
}

// Test 276: get preset missing --channel
func TestTemplateGetPresetMissingChannel(t *testing.T) {
	r := runCLI(t, "template", "get-preset")
	requireFailure(t, r)
}

// Test 289: variables default
func TestTemplateVariables(t *testing.T) {
	r := runCLI(t, "template", "variables")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "NAME", "TYPE", "DESCRIPTION", "EXAMPLE")
}

// Test 299: variables JSON
func TestTemplateVariablesJSON(t *testing.T) {
	r := runCLI(t, "template", "variables", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// Test 300: functions default
func TestTemplateFunctions(t *testing.T) {
	r := runCLI(t, "template", "functions")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "NAME", "SYNTAX", "DESCRIPTION")
}

// Test 305: functions JSON
func TestTemplateFunctionsJSON(t *testing.T) {
	r := runCLI(t, "template", "functions", "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}
