package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDumpCommands_EmitsJSONWithStatusPageChangeCreate(t *testing.T) {
	cmd := newDumpCommandsCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	got := out.String()
	if !json.Valid(out.Bytes()) {
		t.Fatalf("output is not valid JSON:\n%s", got)
	}
	if !strings.Contains(got, `"status-page change-create"`) {
		head := got
		if len(head) > 400 {
			head = head[:400]
		}
		t.Errorf("dump missing status-page change-create path; output head:\n%s", head)
	}
}
