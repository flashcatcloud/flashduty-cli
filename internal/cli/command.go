package cli

import (
	"fmt"
	"io"

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
func (ctx *RunContext) PrintList(items any, cols []output.Column, count, page, total int) error {
	if err := ctx.Printer.Print(items, cols); err != nil {
		return err
	}
	if !ctx.Structured() {
		_, _ = fmt.Fprintf(ctx.Writer, "Showing %d results (page %d, total %d).\n", count, page, total)
	}
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
