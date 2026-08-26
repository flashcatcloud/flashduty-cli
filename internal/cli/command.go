package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

// RunContext provides helpers for command execution. It is created by
// runCommand and passed to the command's handler function. Client is the
// typed go-flashduty SDK every command calls through.
type RunContext struct {
	Client  *flashduty.Client
	Cmd     *cobra.Command
	Args    []string
	Writer  io.Writer
	Printer output.Printer
	Format  output.Format
}

// Structured reports whether output should be a machine-readable dump (JSON or
// TOON) rather than the human table/detail view. Command handlers branch on
// this to suppress detail views, footers, and interactive prompts.
func (ctx *RunContext) Structured() bool { return ctx.Format.Structured() }

// runCommand creates a go-flashduty client and RunContext, then calls fn. It
// centralises the setup every API-backed command repeats; handlers reach the
// SDK through ctx.Client.
func runCommand(cmd *cobra.Command, args []string, fn func(ctx *RunContext) error) error {
	client, err := newClient()
	if err != nil {
		return err
	}
	ctx := &RunContext{
		Client:  client,
		Cmd:     cmd,
		Args:    args,
		Writer:  cmd.OutOrStdout(),
		Printer: newPrinter(cmd.OutOrStdout()),
		Format:  currentOutputFormat(),
	}
	return fn(ctx)
}

// PrintList prints items as a table and appends a "Showing N results (page P, total T)." footer.
// In structured mode the footer is suppressed to keep stdout byte-pure for
// jq/toon pipelines, so a page with more rows beyond it is announced on
// stderr instead — without it a consumer sees a partial page
// (e.g. the default --limit 20 of a far larger total) as the whole set. The
// judgment accounts for the page offset: on the last page
// ((page-1)*limit+count reaches total) there is no rest, so no note.
func (ctx *RunContext) PrintList(items any, cols []output.Column, count, page, limit, total int) error {
	if err := ctx.Printer.Print(items, cols); err != nil {
		return err
	}
	if ctx.Structured() {
		if (page-1)*limit+count < total {
			_, _ = fmt.Fprintf(ctx.Cmd.ErrOrStderr(), "note: showing %d of %d total results (page %d); raise --limit or use --page for the rest\n", count, total, page)
		}
		return nil
	}
	_, _ = fmt.Fprintf(ctx.Writer, "Showing %d results (page %d, total %d).\n", count, page, total)
	return nil
}

// PrintTotal prints items as a table and appends a "Total: N" footer.
func (ctx *RunContext) PrintTotal(items any, cols []output.Column, total int) error {
	if err := ctx.Printer.Print(items, cols); err != nil {
		return err
	}
	if !ctx.Structured() {
		_, _ = fmt.Fprintf(ctx.Writer, "Total: %d\n", total)
	}
	return nil
}

// WriteResult prints a success message as plain text or JSON.
func (ctx *RunContext) WriteResult(message string) {
	writeResult(ctx.Writer, message)
}

// WriteRaw writes a non-JSON response body (e.g. a CSV/file download surfaced
// on Response.Raw by the *export endpoints) straight to the output writer, so
// shell redirection (`> file.csv`) captures the bytes verbatim instead of the
// canned "OK: POST ..." acknowledgment.
func (ctx *RunContext) WriteRaw(body []byte) error {
	_, err := ctx.Writer.Write(body)
	return err
}

// WriteResultJSON outputs structured data in JSON or TOON mode, or a
// human-readable message in table mode. JSON stays indented (byte-compatible
// with the legacy --json path); TOON routes through the SDK marshaller.
func (ctx *RunContext) WriteResultJSON(data any, humanMessage string) error {
	if !ctx.Structured() {
		_, _ = fmt.Fprintln(ctx.Writer, humanMessage)
		return nil
	}
	out, err := marshalStructured(data)
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}
	_, _ = fmt.Fprintln(ctx.Writer, string(out))
	return nil
}

// groupUnknownSubcommand is the Args validator every command-group node gets
// from newGroupCmd. A group is a command whose only job is to hold
// subcommands (e.g. "incident", "oncall schedule") — it has no handler of its
// own.
//
// Cobra's built-in "unknown command" detection (legacyArgs, in
// github.com/spf13/cobra's args.go) only raises an error for the ROOT
// command: legacyArgs returns nil whenever cmd.HasParent() is true. So when a
// group is left as a bare &cobra.Command{Use, Short} literal, a typo'd
// subcommand (e.g. "incident list-alerts") walks straight through Find()
// with no error, execute() then finds the group has no RunE, decides it
// isn't Runnable, and returns flag.ErrHelp — which ExecuteC always turns into
// "print help, exit 0", no matter what SilenceErrors/SilenceUsage say. The
// wrong verb looks exactly like a successful run.
//
// Pairing this validator with a real RunE (below) makes the group Runnable,
// so cobra actually calls ValidateArgs with the leftover token and this
// function gets a chance to reject it instead of it being silently dropped.
func groupUnknownSubcommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	// Mirrors cobra's own default (see (*Command).findSuggestions): a group
	// built via newGroupCmd never sets this itself, so the zero value would
	// otherwise disable the Levenshtein half of SuggestionsFor.
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}
	suggestion := ""
	if names := cmd.SuggestionsFor(args[0]); len(names) > 0 {
		var b strings.Builder
		b.WriteString("\n\nDid you mean this?\n")
		for _, name := range names {
			fmt.Fprintf(&b, "\t%s\n", name)
		}
		suggestion = b.String()
	}
	return fmt.Errorf("unknown command %q for %q%s", args[0], cmd.CommandPath(), suggestion)
}

// newGroupCmd creates a command-group node: Use/Short (with Long/Example
// optionally set by the caller on the returned value) and no behavior of its
// own beyond dispatching to its subcommands.
//
// Every group in the command tree — curated (alert, incident, oncall
// schedule, ...) and generated (genGroup, in gen_support.go) — must be built
// through this constructor rather than a bare &cobra.Command{Use, Short}
// literal, so a mistyped subcommand fails loudly instead of silently exiting
// 0 (see groupUnknownSubcommand). Running the group with no subcommand at
// all still prints help and exits 0, unchanged.
func newGroupCmd(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  groupUnknownSubcommand,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
}
