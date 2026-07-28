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
	quotedHeredoc := "cat > \"$COMMENT_FILE\" <<'FDUTY_COMMENT_7F3A9C2E_EOF'"
	commentCommand := `fduty incident comment "$ID" --comment-file "$COMMENT_FILE"`

	previous := -1
	for _, requirement := range []string{quotedHeredoc, commentCommand} {
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

	if !strings.Contains(body, "choose a fresh delimiter that is absent as a full line in the intended comment") {
		t.Error("incident card must require a collision-free heredoc delimiter")
	}
	if strings.Contains(body, `--comment "`) {
		t.Error("incident card must not demonstrate the removed inline --comment flag")
	}
	if strings.Contains(body, "read back every target and verify the intended comment is present before reporting success") {
		t.Error("incident card must not retain the manual read-back workaround now that the CLI verifies content fidelity itself")
	}
	// Assert the guarantee is documented, not the sentence that documents it:
	// pinning a full sentence turns this check into a checksum that goes red on
	// every reword and teaches the next reader to update the literal reflexively.
	if !strings.Contains(body, "reads back") || !strings.Contains(body, "exits non-zero") {
		t.Error("incident card must document the command's own read-after-write verification — that is what replaced the manual read-back step removed above")
	}
	// The CLI trims leading/trailing whitespace before sending, because the
	// server trims it too, so the stored text is deliberately NOT the file's
	// exact bytes. The card used to promise byte-for-byte fidelity against the
	// file; that claim is false and would set an agent up to expect trailing
	// newlines to survive a heredoc write.
	if strings.Contains(body, "match the file byte-for-byte") {
		t.Error("incident card must not promise byte-for-byte fidelity against the file: leading/trailing whitespace is stripped before the write")
	}
}

func TestIncidentCommentFileHeredocPreservesMarkdown(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Bash fixture is unavailable on Windows")
	}

	dir := t.TempDir()
	script := `fduty() {
  # Mimic --comment-file: print the referenced file's bytes verbatim.
  cat "$5"
}

ID=64b64ca26f84f00000000000
COMMENT_FILE="` + dir + `/comment.txt"
cat > "$COMMENT_FILE" <<'FDUTY_COMMENT_7F3A9C2E_EOF'
## Investigation
Use ` + "`kubectl get pod`" + ` to inspect the restart.
COMMENT_EOF
The follow-up is still pending.
FDUTY_COMMENT_7F3A9C2E_EOF
fduty incident comment "$ID" --comment-file "$COMMENT_FILE"
`

	output, err := exec.Command("bash", "-c", script).CombinedOutput()
	if err != nil {
		t.Fatalf("Bash fixture failed: %v\n%s", err, output)
	}

	want := "## Investigation\nUse `kubectl get pod` to inspect the restart.\nCOMMENT_EOF\nThe follow-up is still pending.\n"
	if got := string(output); got != want {
		t.Errorf("comment content changed:\nwant: %q\n got: %q", want, got)
	}
}
