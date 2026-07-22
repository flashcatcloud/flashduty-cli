package skilldoc

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestIncidentCardCommentWorkflow(t *testing.T) {
	card, err := os.ReadFile(filepath.Join("..", "..", "skills", "flashduty", "reference", "incident.md"))
	if err != nil {
		t.Fatal(err)
	}

	body := string(card)
	quotedHeredoc := "COMMENT=$(cat <<'COMMENT_EOF'"
	commentCommand := `fduty incident comment "$ID" --comment "$COMMENT"`
	timelineCommand := `fduty incident timeline "$ID" --output-format toon`

	previous := -1
	for _, requirement := range []string{quotedHeredoc, commentCommand, timelineCommand} {
		position := strings.Index(body, requirement)
		if position == -1 {
			t.Errorf("incident card is missing %q", requirement)
			continue
		}
		if position <= previous {
			t.Errorf("incident card must place %q after the previous workflow step", requirement)
		}
		previous = position
	}

	if !strings.Contains(body, "After every comment write, read back every target and verify the intended comment is present before reporting success.") {
		t.Error("incident card must require content-fidelity verification after every comment write")
	}
	if strings.Contains(body, `fduty incident comment <incident-id> --comment "Root cause identified: DB failover. Fix deploying."`) {
		t.Error("incident card must not retain the unsafe inline comment example")
	}
}

func TestIncidentCommentQuotedHeredocPreservesMarkdown(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Bash fixture is unavailable on Windows")
	}

	script := `fduty() {
  printf '%s' "$5"
}

ID=64b64ca26f84f00000000000
COMMENT=$(cat <<'COMMENT_EOF'
## Investigation
Use ` + "`kubectl get pod`" + ` to inspect the restart.
The follow-up is still pending.
COMMENT_EOF
)
fduty incident comment "$ID" --comment "$COMMENT"
`

	output, err := exec.Command("bash", "-c", script).CombinedOutput()
	if err != nil {
		t.Fatalf("Bash fixture failed: %v\n%s", err, output)
	}

	want := "## Investigation\nUse `kubectl get pod` to inspect the restart.\nThe follow-up is still pending."
	if got := string(output); got != want {
		t.Errorf("comment content changed:\nwant: %q\n got: %q", want, got)
	}
}
