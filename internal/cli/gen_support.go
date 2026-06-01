package cli

import (
	"encoding/json"
	"fmt"

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

// genBindBody marshals the assembled body map into the typed request struct so
// the call benefits from the SDK's wire encoding (nullable pointers, etc.).
func genBindBody(body map[string]any, req any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}
	if err := json.Unmarshal(b, req); err != nil {
		return fmt.Errorf("failed to bind request: %w", err)
	}
	return nil
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
