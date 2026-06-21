package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flashcatcloud/flashduty-cli/internal/skilldoc"
)

// fixtureDump is a small dump with one status-page leaf, decoupled from the
// real CLI tree so the test stays deterministic.
func fixtureDump() skilldoc.Dump {
	return skilldoc.Dump{Commands: []skilldoc.Command{
		{
			Path:  "status-page change-create",
			Group: "status-page",
			Short: "Create status page event",
			Long: "Create status page event.\n\nRequest fields:\n" +
				"  --type string (required) — Event type. [incident, maintenance]\n",
			Flags: []skilldoc.Flag{{Name: "type", Type: "string"}, {Name: "data", Type: "string"}},
		},
	}}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunCheck_FlagsStaleAndUnknown(t *testing.T) {
	dir := t.TempDir()
	d := fixtureDump()

	// A reference card with a deliberately stale fence + a bad-flag example.
	body := "# status-page\n\n" +
		"```bash\nfduty status-page change-create --type incident --bogus 1\n```\n\n" +
		skilldoc.FenceStart("status-page") + "\n\n### change-create\nSTALE WRONG\n\n" +
		skilldoc.FenceEnd("status-page") + "\n"
	writeFile(t, filepath.Join(dir, "reference", "status-page.md"), body)

	var out bytes.Buffer
	n, err := runCheck(d, dir, &out)
	if err != nil {
		t.Fatalf("runCheck: %v", err)
	}
	if n == 0 {
		t.Fatalf("expected issues, got 0\noutput:\n%s", out.String())
	}
	got := out.String()
	if !strings.Contains(got, "stale-fence") {
		t.Errorf("missing stale-fence in output:\n%s", got)
	}
	if !strings.Contains(got, "unknown-flag") {
		t.Errorf("missing unknown-flag in output:\n%s", got)
	}
	// Issues are reported with a file:line prefix.
	if !strings.Contains(got, "status-page.md:") {
		t.Errorf("issues should carry file:line; output:\n%s", got)
	}
}

func TestRunCheck_CleanDirIsZero(t *testing.T) {
	dir := t.TempDir()
	d := fixtureDump()

	// A clean card: fresh fence + a valid example.
	body := "# status-page\n\n" +
		"```bash\nfduty status-page change-create --type incident\n```\n\n" +
		skilldoc.GenerateFence(d, "status-page") + "\n"
	writeFile(t, filepath.Join(dir, "reference", "status-page.md"), body)

	var out bytes.Buffer
	n, err := runCheck(d, dir, &out)
	if err != nil {
		t.Fatalf("runCheck: %v", err)
	}
	if n != 0 {
		t.Errorf("clean dir: want 0 issues, got %d:\n%s", n, out.String())
	}
}

// A card checked out with Windows CRLF line endings must still validate clean:
// the fence comparison and harvester normalize EOL, so freshness does not depend
// on how git materialized the file (regression test for the Windows CI failure).
func TestRunCheck_CRLFCardIsClean(t *testing.T) {
	dir := t.TempDir()
	d := fixtureDump()
	body := "# status-page\n\n" +
		"```bash\nfduty status-page change-create --type incident\n```\n\n" +
		skilldoc.GenerateFence(d, "status-page") + "\n"
	crlf := strings.ReplaceAll(body, "\n", "\r\n")
	writeFile(t, filepath.Join(dir, "reference", "status-page.md"), crlf)

	var out bytes.Buffer
	n, err := runCheck(d, dir, &out)
	if err != nil {
		t.Fatalf("runCheck: %v", err)
	}
	if n != 0 {
		t.Errorf("CRLF card should validate clean, got %d issue(s):\n%s", n, out.String())
	}
}

func TestRunCheck_MissingDirIsZero(t *testing.T) {
	d := fixtureDump()
	var out bytes.Buffer
	n, err := runCheck(d, filepath.Join(t.TempDir(), "does-not-exist"), &out)
	if err != nil {
		t.Fatalf("runCheck on missing dir should not error: %v", err)
	}
	if n != 0 {
		t.Errorf("missing skills dir: want 0 issues, got %d", n)
	}
}

// TestRunGenAll covers `skilldoc gen` with no group: it must regenerate every
// card that exists, derive the group set from the dump, and silently skip a dump
// group that has no card file (e.g. webhook) rather than erroring.
func TestRunGenAll_FillsEveryCardAndSkipsCardless(t *testing.T) {
	dir := t.TempDir()
	d := skilldoc.Dump{Commands: []skilldoc.Command{
		{Path: "status-page change-create", Group: "status-page", Short: "Create", Flags: []skilldoc.Flag{{Name: "type", Type: "string"}}},
		{Path: "incident list", Group: "incident", Short: "List incidents", Flags: []skilldoc.Flag{{Name: "limit", Type: "int"}}},
		{Path: "webhook list", Group: "webhook", Short: "List webhooks"}, // group with NO card file
	}}
	for _, g := range []string{"status-page", "incident"} {
		writeFile(t, filepath.Join(dir, "reference", g+".md"),
			"# "+g+"\n\nintro\n\n"+skilldoc.FenceStart(g)+"\n"+skilldoc.FenceEnd(g)+"\n")
	}

	if err := runGenAll(d, dir); err != nil {
		t.Fatalf("runGenAll must not error on the cardless webhook group: %v", err)
	}

	for g, verb := range map[string]string{"status-page": "### change-create", "incident": "### list"} {
		raw, err := os.ReadFile(filepath.Join(dir, "reference", g+".md"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), verb) {
			t.Errorf("%s card fence not filled (want %q):\n%s", g, verb, raw)
		}
		if !strings.Contains(string(raw), "intro") {
			t.Errorf("%s: gen-all clobbered hand-written content", g)
		}
	}
	var out bytes.Buffer
	if n, _ := runCheck(d, dir, &out); n != 0 {
		t.Errorf("after gen-all, check should be clean; got %d:\n%s", n, out.String())
	}
}

func TestRunGen_FillsFence(t *testing.T) {
	dir := t.TempDir()
	d := fixtureDump()

	// Card with empty fence markers; gen should fill them with a fresh render.
	card := filepath.Join(dir, "reference", "status-page.md")
	body := "# status-page\n\nintro\n\n" +
		skilldoc.FenceStart("status-page") + "\n" + skilldoc.FenceEnd("status-page") + "\n"
	writeFile(t, card, body)

	if err := runGen(d, dir, "status-page"); err != nil {
		t.Fatalf("runGen: %v", err)
	}

	updated, err := os.ReadFile(card)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "### change-create") {
		t.Errorf("gen did not fill fence:\n%s", updated)
	}
	// After gen, the fence must be fresh (check reports 0 stale-fence issues).
	var out bytes.Buffer
	n, _ := runCheck(d, dir, &out)
	if n != 0 {
		t.Errorf("after gen, check should be clean; got %d:\n%s", n, out.String())
	}
	// Hand-written content outside the fence is preserved.
	if !strings.Contains(string(updated), "intro") {
		t.Errorf("gen clobbered hand-written content:\n%s", updated)
	}
}
