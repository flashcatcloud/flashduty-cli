package cli

import (
	"testing"
)

// TestCommandAlertListActiveRecoveredReachWire is the regression guard for the
// nullable-pointer bug: is_active and ever_muted are *bool in the SDK, so the
// false value must reach the wire. Before the fix they were value+omitempty and
// --recovered (is_active=false) was silently dropped, turning the filter into a
// no-op that returned active alerts too.
func TestCommandAlertListActiveRecoveredReachWire(t *testing.T) {
	cases := []struct {
		name     string
		flag     string
		field    string
		wantBool bool
	}{
		{"active sends is_active=true", "--active", "is_active", true},
		{"recovered sends is_active=false", "--recovered", "is_active", false},
		{"muted sends ever_muted=true", "--muted", "ever_muted", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			if _, err := execCommand("alert", "list", tc.flag); err != nil {
				t.Fatalf("execCommand: %v", err)
			}
			got, ok := stub.lastBody[tc.field]
			if !ok {
				t.Fatalf("%s missing from wire body %#v", tc.field, stub.lastBody)
			}
			gotBool, isBool := got.(bool)
			if !isBool {
				t.Fatalf("%s = %#v (%T), want a JSON bool", tc.field, got, got)
			}
			if gotBool != tc.wantBool {
				t.Errorf("%s = %v, want %v", tc.field, gotBool, tc.wantBool)
			}
		})
	}
}

// TestCommandAlertListNoStatusFilterOmitsIsActive: with neither --active nor
// --recovered, is_active is a nil *bool and omitempty keeps it off the wire, so
// the server applies no status filter.
func TestCommandAlertListNoStatusFilterOmitsIsActive(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("alert", "list"); err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if _, ok := stub.lastBody["is_active"]; ok {
		t.Errorf("is_active should be omitted with no status filter, got %#v", stub.lastBody["is_active"])
	}
	if _, ok := stub.lastBody["ever_muted"]; ok {
		t.Errorf("ever_muted should be omitted without --muted, got %#v", stub.lastBody["ever_muted"])
	}
}
