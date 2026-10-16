package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestFolderCommands(t *testing.T) {
	for _, tt := range []struct {
		action string
		args   []string
		body   map[string]any
	}{
		{"list", nil, nil},
		{"health", nil, nil},
		{"create", []string{"--name", "Checkout", "--parent-id", "10"}, map[string]any{"name": "Checkout", "parent_id": float64(10)}},
		{"update", []string{"--id", "12", "--name", "Checkout", "--note", "", "--team-id", "0"}, map[string]any{"id": float64(12), "name": "Checkout"}},
		{"move", []string{"--id", "12", "--dest-parent-id", "0"}, map[string]any{"id": float64(12), "dest_parent_id": float64(0)}},
		{"delete", []string{"--id", "12"}, map[string]any{"id": float64(12)}},
	} {
		t.Run(tt.action, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			stub.dataForPath = func(_ string, _ map[string]any) any {
				if tt.action == "list" {
					return []any{}
				}
				return map[string]any{}
			}
			args := append([]string{"monit", "folder-" + tt.action}, tt.args...)
			if _, err := execCommand(args...); err != nil {
				t.Fatal(err)
			}
			if stub.lastPath != "/monit/folder/"+tt.action {
				t.Errorf("path = %q", stub.lastPath)
			}
			if tt.body != nil && !reflect.DeepEqual(stub.lastBody, tt.body) {
				t.Errorf("body = %#v, want %#v", stub.lastBody, tt.body)
			}
		})
	}
}

func TestFolderMoveRequiresDestination(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	_, err := execCommand("monit", "folder-move", "--id", "12")
	if err == nil || !strings.Contains(err.Error(), "dest-parent-id") {
		t.Fatalf("expected missing destination error, got %v", err)
	}
	if stub.lastPath != "" {
		t.Fatalf("unexpected request to %q", stub.lastPath)
	}
}
