package cli

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
)

// This file is the hand-written runtime support for the generated commands in
// zz_generated_*.go (produced by internal/cmd/cligen). Generated files stay
// pure data + wiring; all shared behavior lives here so it can be reviewed and
// tested like normal code.

// genAssembleBody builds a request body map from an optional --data JSON blob
// overlaid with explicitly-set typed flags. Flags win over --data so an agent
// can pass a JSON skeleton and override one field. setFlags is called after the
// --data merge to stamp the changed scalar flags.
func genAssembleBody(dataJSON string, setFlags func(body map[string]any)) (map[string]any, error) {
	body := map[string]any{}
	if dataJSON != "" {
		if err := json.Unmarshal([]byte(dataJSON), &body); err != nil {
			return nil, fmt.Errorf("invalid --data JSON: %w", err)
		}
	}
	setFlags(body)
	return body, nil
}

// responseHelp returns the rendered "Response fields" block for an SDK
// Service.Method (from the generated responseHelpBySDKMethod table), or "" when
// the response has no documented schema. Curated commands append it to their
// Long so they show the same output shape as the generated commands.
func responseHelp(service, method string) string {
	return responseHelpBySDKMethod[service+"."+method]
}

// curatedLong composes a curated command's Long help from an intro paragraph
// plus the shared spec-derived Response-fields block for the SDK method it
// calls, so agents read output field names instead of guessing them with jq.
func curatedLong(intro, service, method string) string {
	if rh := responseHelp(service, method); rh != "" {
		return intro + "\n\n" + rh
	}
	return intro
}

// genBindBody marshals the assembled body map into the typed request struct so
// the call benefits from the SDK's wire encoding (nullable pointers, etc.).
//
// POST request structs tag fields with `json`, so json.Unmarshal binds them.
// GET query structs tag fields with `url` and carry NO json tag, so the
// Unmarshal pass silently skips every field, leaving the request empty and the
// command un-driveable. After the json pass, additively bind any url-tagged
// field from the body by its url wire-name. For POST structs the url pass is a
// no-op, so existing behavior is unchanged.
func genBindBody(body map[string]any, req any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}
	if err := json.Unmarshal(b, req); err != nil {
		return fmt.Errorf("failed to bind request: %w", err)
	}
	bindURLTagged(body, reflect.ValueOf(req))
	return nil
}

// bindURLTagged fills fields carrying a `url` struct tag (GET query params) from
// the body map keyed by the url wire-name. json.Unmarshal cannot reach these
// because they lack a json tag. The 6 GET request types use only int64/string
// fields; values arrive as Go ints/strings (typed flags) or float64/string
// (--data JSON), all coerced here.
func bindURLTagged(body map[string]any, rv reflect.Value) {
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Anonymous {
			bindURLTagged(body, rv.Field(i))
			continue
		}
		if f.Tag.Get("json") != "" { // json-tagged: already bound by Unmarshal
			continue
		}
		wire := strings.Split(f.Tag.Get("url"), ",")[0]
		if wire == "" || wire == "-" {
			continue
		}
		raw, ok := body[wire]
		if !ok {
			continue
		}
		fv := rv.Field(i)
		if !fv.CanSet() {
			continue
		}
		switch fv.Kind() {
		case reflect.String:
			if s, ok := raw.(string); ok {
				fv.SetString(s)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			switch n := raw.(type) {
			case float64:
				fv.SetInt(int64(n))
			case int64:
				fv.SetInt(n)
			case int:
				fv.SetInt(int64(n))
			case json.Number:
				if iv, err := n.Int64(); err == nil {
					fv.SetInt(iv)
				}
			}
		}
	}
}

// printGenericResult renders a generated command's typed response. Generated
// commands have no curated column set, so in machine-readable mode (TOON/JSON)
// it marshals the whole value — which is what the agent reads — and in human
// table mode it falls back to pretty JSON rather than a blank table.
func printGenericResult(ctx *RunContext, data any) error {
	if ctx.Structured() {
		return ctx.Printer.Print(data, nil)
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}
	_, err = fmt.Fprintln(ctx.Writer, string(out))
	return err
}

// genGroup finds an existing subcommand named `name` under parent, or creates a
// group command with that name. This lets generated commands attach to the same
// group a curated command already owns (partial-coverage services) and lets a
// multi-segment API path build its intermediate group chain idempotently.
func genGroup(parent *cobra.Command, name, short string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	g := &cobra.Command{Use: name, Short: short}
	parent.AddCommand(g)
	return g
}

// genAddLeaf attaches a generated leaf command under parent unless a command
// with the same name already exists there. A curated command always wins the
// exact path-name (it registers first, in init()), so its richer implementation
// keeps the canonical command while the generated twin is harmlessly dropped;
// the operation remains reachable at its path-name either way.
func genAddLeaf(parent *cobra.Command, leaf *cobra.Command) {
	for _, c := range parent.Commands() {
		if c.Name() == leaf.Name() {
			return
		}
	}
	parent.AddCommand(leaf)
}
