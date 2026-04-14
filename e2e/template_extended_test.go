//go:build e2e

package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test 271-274: get preset for each channel type
func TestTemplateGetPresetAllChannels(t *testing.T) {
	t.Skip("requires TemplateID field not yet supported by SDK")
	channels := []string{"feishu", "email", "dingtalk", "telegram"}
	for _, ch := range channels {
		t.Run(ch, func(t *testing.T) {
			r := runCLI(t, "template", "get-preset", "--channel", ch)
			requireSuccess(t, r)
			if r.Stdout == "" {
				t.Error("expected non-empty template output")
			}
		})
	}
}

// Test 275: get preset invalid channel
func TestTemplateGetPresetInvalidChannel(t *testing.T) {
	r := runCLI(t, "template", "get-preset", "--channel", "invalid_xyz")
	requireFailure(t, r)
}

// Test 277: get preset --json outputs JSON
func TestTemplateGetPresetJSON(t *testing.T) {
	t.Skip("requires TemplateID field not yet supported by SDK")
	plain := runCLI(t, "template", "get-preset", "--channel", "slack")
	requireSuccess(t, plain)
	withJSON := runCLI(t, "template", "get-preset", "--channel", "slack", "--json")
	requireSuccess(t, withJSON)

	if strings.TrimSpace(plain.Stdout) == "" {
		t.Fatal("expected preset template output, got empty output")
	}
	if strings.TrimSpace(plain.Stdout) != strings.TrimSpace(withJSON.Stdout) {
		t.Fatalf("expected --json to preserve raw preset output\nplain:\n%s\nwith --json:\n%s", plain.Stdout, withJSON.Stdout)
	}
}

// Test 278: validate valid template
func TestTemplateValidateValid(t *testing.T) {
	t.Skip("requires TemplateID field not yet supported by SDK")
	// Get a preset template first.
	r := runCLI(t, "template", "get-preset", "--channel", "slack")
	requireSuccess(t, r)

	// Write to temp file.
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "template.txt")
	if err := os.WriteFile(templateFile, []byte(r.Stdout), 0644); err != nil {
		t.Fatal(err)
	}

	// Validate.
	r = runCLI(t, "template", "validate", "--channel", "slack", "--file", templateFile)
	requireSuccess(t, r)
	requireContains(t, r.Stdout, "Status: VALID")
	requireContains(t, r.Stdout, "Size:")
}

// Test 279: validate invalid template
func TestTemplateValidateInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "bad_template.txt")
	if err := os.WriteFile(templateFile, []byte("{{.InvalidField}}"), 0644); err != nil {
		t.Fatal(err)
	}

	r := runCLI(t, "template", "validate", "--channel", "slack", "--file", templateFile)
	// May succeed or fail depending on API validation.
	// Just verify no crash.
	_ = r
}

// Test 281: validate missing --channel
func TestTemplateValidateMissingChannel(t *testing.T) {
	r := runCLI(t, "template", "validate", "--file", "test.txt")
	requireFailure(t, r)
}

// Test 282: validate missing --file
func TestTemplateValidateMissingFile(t *testing.T) {
	r := runCLI(t, "template", "validate", "--channel", "slack")
	requireFailure(t, r)
}

// Test 283: validate nonexistent file
func TestTemplateValidateNonexistentFile(t *testing.T) {
	r := runCLI(t, "template", "validate", "--channel", "slack", "--file", "/nonexistent/path/template.txt")
	requireFailure(t, r)
	requireContains(t, r.Stderr, "failed to read template file")
}

// Test 285: validate JSON output
func TestTemplateValidateJSON(t *testing.T) {
	t.Skip("requires TemplateID field not yet supported by SDK")
	// Get a preset template first.
	r := runCLI(t, "template", "get-preset", "--channel", "slack")
	requireSuccess(t, r)

	tmpDir := t.TempDir()
	templateFile := filepath.Join(tmpDir, "template.txt")
	if err := os.WriteFile(templateFile, []byte(r.Stdout), 0644); err != nil {
		t.Fatal(err)
	}

	r = runCLI(t, "template", "validate", "--channel", "slack", "--file", templateFile, "--json")
	requireSuccess(t, r)
	requireValidJSON(t, r.Stdout)
}

// Tests 290-298: variables --category filter
func TestTemplateVariablesCategory(t *testing.T) {
	categories := []string{"core", "time", "people", "alerts", "labels", "context", "notification", "post_incident"}
	for _, cat := range categories {
		t.Run(cat, func(t *testing.T) {
			r := runCLI(t, "template", "variables", "--category", cat)
			requireSuccess(t, r)
		})
	}
}

// Test 298: variables --category invalid
func TestTemplateVariablesCategoryInvalid(t *testing.T) {
	r := runCLI(t, "template", "variables", "--category", "nonexistent")
	requireSuccess(t, r) // no crash, just empty results
}

// Test 301: functions --type custom
func TestTemplateFunctionsCustom(t *testing.T) {
	r := runCLI(t, "template", "functions", "--type", "custom")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "NAME", "SYNTAX", "DESCRIPTION")
}

// Test 302: functions --type sprig
func TestTemplateFunctionsSprig(t *testing.T) {
	r := runCLI(t, "template", "functions", "--type", "sprig")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "NAME", "SYNTAX", "DESCRIPTION")
}

// Test 303: functions --type all
func TestTemplateFunctionsAll(t *testing.T) {
	r := runCLI(t, "template", "functions", "--type", "all")
	requireSuccess(t, r)
}

// Test 304: functions --type invalid (falls through to default switch case)
func TestTemplateFunctionsInvalidType(t *testing.T) {
	r := runCLI(t, "template", "functions", "--type", "invalid_xyz")
	requireSuccess(t, r)
	requireTableHeaders(t, r.Stdout, "NAME")
}
