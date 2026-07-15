package main

import "testing"

func TestMergedIncludesObjectOneOfBranchFields(t *testing.T) {
	w := &specWalker{schemas: map[string]any{
		"LogResult": map[string]any{
			"type":     "object",
			"required": []any{"method", "pattern_evidence"},
			"properties": map[string]any{
				"method":           map[string]any{"type": "string"},
				"pattern_evidence": map[string]any{"type": "array"},
			},
		},
		"MetricResult": map[string]any{
			"type":     "object",
			"required": []any{"method", "series_evidence"},
			"properties": map[string]any{
				"method":          map[string]any{"type": "string"},
				"series_evidence": map[string]any{"type": "array"},
			},
		},
	}}

	properties, required := w.merged(map[string]any{
		"oneOf": []any{
			map[string]any{"$ref": "#/components/schemas/LogResult"},
			map[string]any{"$ref": "#/components/schemas/MetricResult"},
		},
	})

	if properties["pattern_evidence"] == nil || properties["series_evidence"] == nil {
		t.Fatalf("oneOf branch fields missing: %#v", properties)
	}
	if !required["method"] || required["pattern_evidence"] || required["series_evidence"] {
		t.Fatalf("oneOf required fields = %#v, want only method", required)
	}
}

func TestMergedCombinesObjectOneOfFieldEnums(t *testing.T) {
	w := &specWalker{schemas: map[string]any{
		"LogResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"operation": map[string]any{"type": "string", "enum": []any{"log_patterns"}},
			},
		},
		"MetricResponse": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"operation": map[string]any{"type": "string", "enum": []any{"metric_trends"}},
			},
		},
	}}

	properties, _ := w.merged(map[string]any{
		"oneOf": []any{
			map[string]any{"$ref": "#/components/schemas/LogResponse"},
			map[string]any{"$ref": "#/components/schemas/MetricResponse"},
		},
	})
	got := enumStrings(asMap(properties["operation"]))
	if len(got) != 2 || got[0] != "log_patterns" || got[1] != "metric_trends" {
		t.Fatalf("operation enum = %#v, want [log_patterns metric_trends]", got)
	}
}

func TestTreePreservesPropertyDescriptionOverRef(t *testing.T) {
	w := &specWalker{schemas: map[string]any{
		"Window": map[string]any{
			"type":        "object",
			"description": "Current analysis window.",
			"properties":  map[string]any{"start": map[string]any{"type": "string"}},
		},
	}}

	fields := w.tree(map[string]any{"properties": map[string]any{
		"baseline_window": map[string]any{
			"$ref":        "#/components/schemas/Window",
			"description": "Baseline analysis window.",
		},
	}}, 0)
	if len(fields) != 1 || fields[0].Desc != "Baseline analysis window." {
		t.Fatalf("field descriptions = %#v", fields)
	}
}
