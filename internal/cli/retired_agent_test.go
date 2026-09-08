package cli

import (
	"strings"
	"testing"
)

func TestRetiredAgentCommandsAreUnknown(t *testing.T) {
	for _, args := range [][]string{{"monit-agent", "catalog"}, {"monit-agent", "invoke"}, {"monit", "targets"}, {"monit", "tools-catalog"}, {"monit", "tools-invoke"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			saveAndResetGlobals(t)
			_, err := execCommand(args...)
			if err == nil || !strings.Contains(err.Error(), "unknown command") {
				t.Fatalf("retired command %v: error=%v", args, err)
			}
		})
	}
}
