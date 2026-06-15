package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

// stdinReader is the source read when --data is exactly "-". A package var so
// tests can substitute a buffer (mirrors newClientFn); production reads os.Stdin.
var stdinReader io.Reader = os.Stdin

// resolveDataSource turns a --data flag value into the raw JSON body string,
// supporting two source forms across EVERY --data-bearing command:
//
//	--data '<inline json>'  → returned verbatim
//	--data -                → contents of STDIN
//
// STDIN is read ONLY when the flag is exactly "-"; an empty/absent --data is
// never treated as a stdin request, so commands driven purely by typed flags
// don't block on an empty pipe. Reading from STDIN lets callers pipe a quoted
// heredoc, avoiding shell-quoting hell for JSON bodies that contain commas or
// quotes (e.g. SQL in params).
func resolveDataSource(dataFlag string) (string, error) {
	if dataFlag == "-" {
		b, err := io.ReadAll(stdinReader)
		if err != nil {
			return "", fmt.Errorf("failed to read --data from stdin: %w", err)
		}
		return string(b), nil
	}
	return dataFlag, nil
}

// This file is the hand-written runtime support for the generated commands in
// zz_generated_*.go (produced by internal/cmd/cligen). Generated files stay
// pure data + wiring; all shared behavior lives here so it can be reviewed and
// tested like normal code.

// genAssembleBody builds a request body map from an optional --data JSON blob
// overlaid with explicitly-set typed flags. Flags win over --data so an agent
// can pass a JSON skeleton and override one field. setFlags is called after the
// --data merge to stamp the changed scalar flags; it may return an error (e.g.
// int-parse failure from a positional argument).
//
// The --data value accepts two source forms (see resolveDataSource): inline
// JSON, or - to read STDIN.
func genAssembleBody(dataFlag string, setFlags func(body map[string]any) error) (map[string]any, error) {
	dataJSON, err := resolveDataSource(dataFlag)
	if err != nil {
		return nil, err
	}
	body := map[string]any{}
	if dataJSON != "" {
		if err := json.Unmarshal([]byte(dataJSON), &body); err != nil {
			return nil, fmt.Errorf("invalid --data JSON: %w", err)
		}
	}
	if err := setFlags(body); err != nil {
		return nil, err
	}
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

// genFoldPositional folds a generated command's positional argument(s) into the
// request body under wire, BEFORE the typed flags are stamped. The flag for the
// same field is kept; folding the positional first lets an explicitly-set flag
// still override it (matching genAssembleBody's --data-then-flags overlay order).
//
// kind selects how args map onto the body, matching the emitted flag type:
//
//	"string"   — string scalar     → body[wire] = args[0]
//	"int"      — int64 scalar       → body[wire] = ParseInt(args[0]) (schedule_id)
//	"slice"    — []string variadic  → body[wire] = args (string ids)
//	"intslice" — []int64 variadic   → body[wire] = [ParseInt(a) for a in args]
//	             (channel_ids, team_ids, … whose SDK field is []uint64)
//
// args is the validated positional slice; the cobra Args validator (requireArgs)
// guarantees the arity below before RunE runs, but the bounds checks keep this
// safe if it is ever called directly. A no-positional command never calls this.
func genFoldPositional(args []string, body map[string]any, wire, kind string) error {
	switch kind {
	case "slice":
		if len(args) > 0 {
			body[wire] = args
		}
	case "intslice":
		if len(args) > 0 {
			ids := make([]int64, len(args))
			for i, a := range args {
				n, err := strconv.ParseInt(a, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid %s %q: must be an integer", wire, a)
				}
				ids[i] = n
			}
			body[wire] = ids
		}
	case "int":
		if len(args) > 0 {
			n, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid %s %q: must be an integer", wire, args[0])
			}
			body[wire] = n
		}
	default: // "string"
		if len(args) > 0 {
			body[wire] = args[0]
		}
	}
	return nil
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

// printGenericResult renders a generated command's typed response. In
// machine-readable mode (TOON/JSON) it marshals the whole value — which is what
// the agent reads. In human (table) mode it derives an aligned table by
// reflection (renderGenericTable), since generated commands carry no hand-written
// column set; anything that isn't a list or object falls back to indented JSON.
func printGenericResult(ctx *RunContext, data any) error {
	if ctx.Structured() {
		return ctx.Printer.Print(data, nil)
	}
	return renderGenericTable(ctx, data)
}

// genParseTimeFlag parses a relative-or-absolute time flag into unix seconds,
// mirroring the curated incident-list --since/--until handling: a Go duration
// ("7d", "24h") is "now minus duration", "+7d" is the future, "now" is now, and
// a date/datetime/RFC3339/Unix-seconds value passes through. ok is false when the
// flag was not set, so the caller omits the field from the request body.
func genParseTimeFlag(cmd *cobra.Command, name, raw string) (val int64, ok bool, err error) {
	if !cmd.Flags().Changed(name) {
		return 0, false, nil
	}
	v, err := timeutil.Parse(raw)
	if err != nil {
		return 0, false, fmt.Errorf("invalid --%s: %w", name, err)
	}
	return v, true, nil
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
