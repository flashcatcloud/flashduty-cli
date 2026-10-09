package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// curatedFieldRoutes declares how a curated command that shadows a generated
// twin (see genAddLeaf) reaches each top-level request field of its operation
// when no flag of the same kebab-case name carries it. Values:
//
//	"--flag"       a curated flag with a different name sets the field
//	"args"         positional arguments set the field
//	"omit: <why>"  the field is deliberately not exposed, with the reason
//
// A field with a same-named curated flag needs no entry. Every other field
// must be routed here, so a field the API adds later fails
// TestCuratedCommandsCoverRequestFields instead of silently vanishing behind
// the curated command.
var curatedFieldRoutes = map[string]map[string]string{
	"alert list": {
		"alert_severity":   "--severity",
		"channel_ids":      "--channel",
		"integration_ids":  "--integration",
		"start_time":       "--since",
		"end_time":         "--until",
		"p":                "--page",
		"is_active":        "--active",
		"ever_muted":       "--muted",
		"search_after_ctx": "omit: cursor returned by the previous page; the curated list pages with --page and does not print the next-page cursor",
	},
	"alert merge": {
		"comment": "--comment-file",
	},
	"alert-event list": {
		"severities":        "--severity",
		"channel_ids":       "--channel",
		"integration_ids":   "--integration",
		"integration_types": "--integration-type",
		"start_time":        "--since",
		"end_time":          "--until",
		"p":                 "--page",
		"orderby":           "omit: event_time is the only supported value and the server default",
		"search_after_ctx":  "omit: cursor returned by the previous page; the curated list pages with --page and does not print the next-page cursor",
	},
	"incident merge": {
		"source_incident_ids": "--source",
		"target_incident_id":  "args",
		"owner_id":            "omit: ignored by the server; the merge never changes the target owner",
	},
}

var apiLineRe = regexp.MustCompile(`(?m)^API: [A-Z]+ \S+ \(([^)]+)\)$`)

// shadowedTwins pairs every curated command that took a generated leaf's name
// with the generated twin genAddLeaf dropped, keyed by command path (without
// the root prefix). The twin tree is rebuilt on a bare root so each generated
// leaf is present, then matched by path against the real rootCmd.
func shadowedTwins(t *testing.T) map[string][2]*cobra.Command {
	t.Helper()
	twinRoot := &cobra.Command{Use: rootCmd.Use}
	registerGenerated(twinRoot)

	out := map[string][2]*cobra.Command{}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, twin := range c.Commands() {
			walk(twin)
			if twin.HasSubCommands() {
				continue
			}
			path := strings.TrimPrefix(twin.CommandPath(), twinRoot.Name()+" ")
			real, _, err := rootCmd.Find(strings.Fields(path))
			if err != nil || real == nil {
				t.Fatalf("generated command %q missing from the real tree", path)
			}
			if real.Long != twin.Long {
				out[path] = [2]*cobra.Command{real, twin}
			}
		}
	}
	walk(twinRoot)
	return out
}

// specRequestFields maps operationId to the top-level request-body property
// names, following $ref and allOf/oneOf/anyOf composition.
func specRequestFields(t *testing.T) map[string][]string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/flashcatcloud/go-flashduty").Output()
	if err != nil {
		t.Fatalf("locate go-flashduty module: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(out)), "openapi", "openapi.en.json"))
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	type schema struct {
		Ref        string                     `json:"$ref"`
		Properties map[string]json.RawMessage `json:"properties"`
		AllOf      []schema                   `json:"allOf"`
		OneOf      []schema                   `json:"oneOf"`
		AnyOf      []schema                   `json:"anyOf"`
	}
	var spec struct {
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
			RequestBody struct {
				Content map[string]struct {
					Schema schema `json:"schema"`
				} `json:"content"`
			} `json:"requestBody"`
		} `json:"paths"`
		Components struct {
			Schemas map[string]schema `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parse spec: %v", err)
	}

	var collect func(s schema, into map[string]bool)
	collect = func(s schema, into map[string]bool) {
		if s.Ref != "" {
			collect(spec.Components.Schemas[strings.TrimPrefix(s.Ref, "#/components/schemas/")], into)
		}
		for name := range s.Properties {
			into[name] = true
		}
		for _, group := range [][]schema{s.AllOf, s.OneOf, s.AnyOf} {
			for _, sub := range group {
				collect(sub, into)
			}
		}
	}

	fields := map[string][]string{}
	for _, methods := range spec.Paths {
		for _, op := range methods {
			body, ok := op.RequestBody.Content["application/json"]
			if !ok || op.OperationID == "" {
				continue
			}
			set := map[string]bool{}
			collect(body.Schema, set)
			for name := range set {
				fields[op.OperationID] = append(fields[op.OperationID], name)
			}
			sort.Strings(fields[op.OperationID])
		}
	}
	return fields
}

// TestCuratedCommandsCoverRequestFields guards curated commands against
// silently hiding request fields: every top-level field of a shadowed
// operation must be reachable through a same-named flag or be routed in
// curatedFieldRoutes, and every route must still point at a live flag and a
// field the spec still has.
func TestCuratedCommandsCoverRequestFields(t *testing.T) {
	twins := shadowedTwins(t)
	fields := specRequestFields(t)

	var problems []string
	for path, pair := range twins {
		real, twin := pair[0], pair[1]
		m := apiLineRe.FindStringSubmatch(twin.Long)
		if m == nil {
			t.Fatalf("generated command %q has no API line in its help", path)
		}
		routes := curatedFieldRoutes[path]
		inSpec := map[string]bool{}
		for _, field := range fields[m[1]] {
			inSpec[field] = true
			same := strings.ReplaceAll(field, "_", "-")
			route, routed := routes[field]
			switch {
			case !routed:
				if real.Flags().Lookup(same) == nil {
					problems = append(problems, path+": request field "+field+" is not reachable (add --"+same+" or route it in curatedFieldRoutes)")
				}
			case route == "--"+same:
				problems = append(problems, path+": route for "+field+" is redundant, the same-named flag already covers it")
			case strings.HasPrefix(route, "--"):
				if real.Flags().Lookup(strings.TrimPrefix(route, "--")) == nil {
					problems = append(problems, path+": route "+field+" -> "+route+" names a flag the command does not have")
				}
			case route == "args":
			case strings.HasPrefix(route, "omit: ") && strings.TrimSpace(strings.TrimPrefix(route, "omit: ")) != "":
			default:
				problems = append(problems, path+": route "+field+" -> "+route+` must be "--flag", "args", or "omit: <reason>"`)
			}
		}
		for field := range routes {
			if !inSpec[field] {
				problems = append(problems, path+": route for "+field+" is stale, the spec no longer has that field")
			}
		}
	}
	for path := range curatedFieldRoutes {
		if _, ok := twins[path]; !ok {
			problems = append(problems, path+": curatedFieldRoutes entry for a command that no longer shadows a generated twin")
		}
	}

	sort.Strings(problems)
	if len(problems) > 0 {
		t.Errorf("%d curated request-field gaps:\n  %s", len(problems), strings.Join(problems, "\n  "))
	}
	t.Logf("checked %d curated commands that shadow generated twins", len(twins))
}
