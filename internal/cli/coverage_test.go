package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// specOpMeta is the slice of an operation the coverage tests reason about.
type specOpMeta struct {
	id        string
	path      string
	streaming bool // 200 body is not application/json (e.g. application/x-ndjson)
}

// loadSpecOps reads every public GET/POST operation from the openapi spec
// shipped in the linked go-flashduty module — the same spec cligen generates
// against — recording each op's id, path, and whether its 200 response is a
// non-JSON streaming body. Streaming ops are served by curated commands (the
// generated typed-response template cannot model an io.ReadCloser), so the
// generator-coverage check excludes them.
func loadSpecOps(t *testing.T) []specOpMeta {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/flashcatcloud/go-flashduty").Output()
	if err != nil {
		t.Fatalf("locate go-flashduty module: %v", err)
	}
	specPath := filepath.Join(strings.TrimSpace(string(out)), "openapi", "openapi.en.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var spec struct {
		Paths map[string]map[string]struct {
			OperationID string   `json:"operationId"`
			Tags        []string `json:"tags"`
			Responses   map[string]struct {
				Content map[string]json.RawMessage `json:"content"`
			} `json:"responses"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	var ops []specOpMeta
	for path, methods := range spec.Paths {
		for verb, op := range methods {
			v := strings.ToUpper(verb)
			if v != "GET" && v != "POST" {
				continue
			}
			if op.OperationID == "" || len(op.Tags) == 0 {
				continue
			}
			streaming := false
			if resp, ok := op.Responses["200"]; ok && len(resp.Content) > 0 {
				if _, hasJSON := resp.Content["application/json"]; !hasJSON {
					streaming = true
				}
			}
			ops = append(ops, specOpMeta{id: op.OperationID, path: path, streaming: streaming})
		}
	}
	return ops
}

// loadSpecPaths returns operationId -> path for every public GET/POST operation.
func loadSpecPaths(t *testing.T) map[string]string {
	t.Helper()
	ids := map[string]string{}
	for _, op := range loadSpecOps(t) {
		ids[op.id] = op.path
	}
	return ids
}

// pathCommand derives the path-is-king command for an API path: the first
// segment is the group, the remaining segments hyphen-join into the verb. This
// mirrors cligen's cliGroup/cliVerb (spec path segments are already kebab-case,
// so a plain join matches).
func pathCommand(apiPath string) string {
	var segs []string
	for _, s := range strings.Split(apiPath, "/") {
		if s != "" {
			segs = append(segs, s)
		}
	}
	if len(segs) == 0 {
		return ""
	}
	if len(segs) == 1 {
		return segs[0]
	}
	return segs[0] + " " + strings.Join(segs[1:], "-")
}

// leafCommandPaths walks the real rootCmd (built by init()) and returns the set
// of every command path a user can invoke, minus the "flashduty " root prefix.
func leafCommandPaths() map[string]bool {
	set := map[string]bool{}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			path := strings.TrimPrefix(sub.CommandPath(), "flashduty ")
			set[path] = true
			walk(sub)
		}
	}
	walk(rootCmd)
	return set
}

// TestEveryOperationHasPathCommand is the core invariant of the path-is-king
// command tree: every public operation in the spec is reachable at the command
// derived mechanically from its API path (group = first segment, verb = the
// rest hyphen-joined). An agent that knows only the API path can always invoke
// the operation without guessing — generated commands provide the path-name and
// curated commands win the exact name when they already own it.
func TestEveryOperationHasPathCommand(t *testing.T) {
	specPaths := loadSpecPaths(t)
	leaves := leafCommandPaths()
	var missing []string
	for _, apiPath := range specPaths {
		cmd := pathCommand(apiPath)
		if !leaves[cmd] {
			missing = append(missing, cmd+"  (<= "+apiPath+")")
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d operations have no command at their path-name:\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
	t.Logf("path-is-king: all %d operations reachable at their path-name", len(specPaths))
}

// TestGeneratorTargetsFullSpec asserts the generator emitted a command for every
// non-streaming spec operation (no gaps, no phantom manifest entries from a
// stale run). Streaming ops (200 body is not application/json) are deliberately
// excluded from generation — they cannot be modeled by the typed-response
// template and are served by curated commands instead — so the manifest must NOT
// contain them and they are not required to be generated.
func TestGeneratorTargetsFullSpec(t *testing.T) {
	ops := loadSpecOps(t)
	streaming := map[string]bool{}
	wantGenerated := map[string]bool{}
	for _, op := range ops {
		if op.streaming {
			streaming[op.id] = true
			continue
		}
		wantGenerated[op.id] = true
	}

	gen := map[string]bool{}
	for _, id := range generatedOpIDs {
		gen[id] = true
		if streaming[id] {
			t.Errorf("manifest op %q is streaming and must not be generated (curated only)", id)
		}
		if !wantGenerated[id] && !streaming[id] {
			t.Errorf("manifest op %q is not in the current spec (regenerate cligen)", id)
		}
	}
	for id := range wantGenerated {
		if !gen[id] {
			t.Errorf("op %q has no generated command (regenerate cligen)", id)
		}
	}
	t.Logf("generator targets %d/%d non-streaming spec operations (%d streaming, curated)",
		len(gen), len(wantGenerated), len(streaming))
}
