package cli

import (
	"strings"
	"testing"
)

func TestMCPCreateFlagSurface(t *testing.T) {
	cmd := newMCPCreateCmd()
	flags := cmd.Flags()
	for _, name := range []string{
		"server-name", "description", "transport",
		"command", "args", "env", "url", "headers",
		"connect-timeout", "call-timeout", "team-id",
	} {
		if flags.Lookup(name) == nil {
			t.Errorf("flag --%s not registered", name)
		}
	}
}

func TestMCPCreateRejectsEmptyServerName(t *testing.T) {
	saveAndResetGlobals(t)
	// The empty-name guard fires inside the handler before WriteServerCreate is
	// ever called, so a stub server that records no request is sufficient.
	stub := newGFStub(t)

	_, err := execCommand("mcp", "create")
	if err == nil {
		t.Fatal("expected error for empty --server-name, got nil")
	}
	if !strings.Contains(err.Error(), "--server-name is required") {
		t.Fatalf("expected error %q, got %q", "--server-name is required", err.Error())
	}
	if stub.requests != 0 {
		t.Fatalf("expected no request to reach the server, got %d", stub.requests)
	}
}

func TestCommandMCPCreate(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"server_id": "srv-1", "status": "enabled"}

	out, err := execCommand("mcp", "create",
		"--server-name", "demo",
		"--transport", "streamable-http",
		"--url", "https://mcp.example/sse",
		"--connect-timeout", "15",
		"--call-timeout", "90",
		"--team-id", "7",
	)
	if err != nil {
		t.Fatalf("[mcp-create] unexpected error: %v", err)
	}
	if stub.lastPath != "/safari/mcp/server/create" {
		t.Fatalf("[mcp-create] expected /safari/mcp/server/create, got %q", stub.lastPath)
	}
	if stub.lastBody["server_name"] != "demo" || stub.lastBody["transport"] != "streamable-http" || stub.lastBody["url"] != "https://mcp.example/sse" {
		t.Fatalf("[mcp-create] unexpected input: %#v", stub.lastBody)
	}
	if stub.lastBody["connect_timeout"] != float64(15) || stub.lastBody["call_timeout"] != float64(90) || stub.lastBody["team_id"] != float64(7) {
		t.Fatalf("[mcp-create] unexpected numeric input: %#v", stub.lastBody)
	}
	if !strings.Contains(out, "MCP server registered: srv-1 (status: enabled)") {
		t.Fatalf("[mcp-create] unexpected output:\n%s", out)
	}
}
