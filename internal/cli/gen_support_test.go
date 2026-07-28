package cli

import "testing"

func TestGenBindBodyAllowsNullForRequiredNullableField(t *testing.T) {
	req := new(struct {
		Value *bool `json:"value"`
	})

	if err := genBindBody(map[string]any{"value": nil}, req); err != nil {
		t.Fatalf("genBindBody required nullable field: %v", err)
	}
	if req.Value != nil {
		t.Fatalf("Value = %v, want nil", req.Value)
	}
}
