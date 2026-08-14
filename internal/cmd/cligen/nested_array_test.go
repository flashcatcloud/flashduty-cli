package main

import "testing"

// TestTreeExpandsNestedArrayOfObject covers the "Or-of-AND" filter tree shape
// (array<array<object>>, e.g. silence-rule-create's `filters`): the object
// item fields two array layers down must render as children exactly like a
// plain array<object> field does one layer down.
func TestTreeExpandsNestedArrayOfObject(t *testing.T) {
	w := &specWalker{schemas: map[string]any{
		"FilterCondition": map[string]any{
			"type":     "object",
			"required": []any{"key", "oper", "vals"},
			"properties": map[string]any{
				"key":  map[string]any{"type": "string", "description": "e.g. `alert_severity`, `labels.service`"},
				"oper": map[string]any{"type": "string", "enum": []any{"IN", "NOTIN"}},
				"vals": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
		},
	}}

	fields := w.tree(map[string]any{"properties": map[string]any{
		"filters": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type":  "array",
				"items": map[string]any{"$ref": "#/components/schemas/FilterCondition"},
			},
		},
	}}, 0, nil)

	if len(fields) != 1 || fields[0].Wire != "filters" {
		t.Fatalf("fields = %#v, want single 'filters' field", fields)
	}
	children := fields[0].Children
	if len(children) != 3 {
		t.Fatalf("filters.Children = %#v, want 3 (key, oper, vals)", children)
	}
	byWire := map[string]schemaField{}
	for _, c := range children {
		byWire[c.Wire] = c
	}
	if byWire["key"].Type != "string" || byWire["key"].Desc == "" {
		t.Fatalf("key child = %#v, want string with description", byWire["key"])
	}
	if len(byWire["oper"].Enum) != 2 {
		t.Fatalf("oper.Enum = %#v, want [IN NOTIN]", byWire["oper"].Enum)
	}
	if byWire["vals"].Type != "array<string>" {
		t.Fatalf("vals.Type = %q, want array<string>", byWire["vals"].Type)
	}
}

// TestTreeLeavesNestedArrayOfScalarWithoutChildren guards the sibling shape
// (array<array<string>>, e.g. the alert-grouping `equals` field): with no
// object at the bottom, tree() must not synthesize children.
func TestTreeLeavesNestedArrayOfScalarWithoutChildren(t *testing.T) {
	w := &specWalker{}

	fields := w.tree(map[string]any{"properties": map[string]any{
		"equals": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
	}}, 0, nil)

	if len(fields) != 1 || fields[0].Wire != "equals" {
		t.Fatalf("fields = %#v, want single 'equals' field", fields)
	}
	if len(fields[0].Children) != 0 {
		t.Fatalf("equals.Children = %#v, want none", fields[0].Children)
	}
}

// TestSchemaTypeLabelsNestedArrayDepth guards the type label alongside the
// child-expansion behavior above: "array<array>" hid the same missing-depth
// information for the flag/summary line that the missing children hid for
// the field list, and both stem from the array case only unwrapping one
// `items` level.
func TestSchemaTypeLabelsNestedArrayDepth(t *testing.T) {
	cases := []struct {
		name string
		s    map[string]any
		want string
	}{
		{"scalar array", map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, "array<string>"},
		{"array of object", map[string]any{"type": "array", "items": map[string]any{"type": "object"}}, "array<object>"},
		{"array of array of object (filters)", map[string]any{"type": "array", "items": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/FilterCondition"}}}, "array<array<object>>"},
		{"array of array of string (equals)", map[string]any{"type": "array", "items": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}, "array<array<string>>"},
	}
	for _, c := range cases {
		if got := schemaType(c.s); got != c.want {
			t.Errorf("%s: schemaType() = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestTreeExpandsPlainArrayOfObjectUnchanged is a regression guard: the
// existing single-level array<object> behavior (e.g. `layers`) must be
// unaffected by generalizing the array case to unwrap nested arrays.
func TestTreeExpandsPlainArrayOfObjectUnchanged(t *testing.T) {
	w := &specWalker{}

	fields := w.tree(map[string]any{"properties": map[string]any{
		"layers": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "object", "properties": map[string]any{"target": map[string]any{"type": "string"}}},
		},
	}}, 0, nil)

	if len(fields) != 1 || len(fields[0].Children) != 1 || fields[0].Children[0].Wire != "target" {
		t.Fatalf("layers field = %#v, want single child 'target'", fields[0])
	}
}
