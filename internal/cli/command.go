package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

// RunContext provides helpers for command execution. It is created by
// runCommand and passed to the command's handler function.
type RunContext struct {
	Client  flashdutyClient
	Cmd     *cobra.Command
	Args    []string
	Writer  io.Writer
	Printer output.Printer
	JSON    bool
}

// runCommand creates a client and RunContext, then calls fn.
// It centralises setup that every API-backed command repeats.
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
		JSON:    flagJSON,
	}
	return fn(ctx)
}

// PrintList prints items as a table and appends a "Showing N results (page P, total T)." footer.
func (ctx *RunContext) PrintList(items any, cols []output.Column, count, page, total int) error {
	if err := ctx.Printer.Print(items, cols); err != nil {
		return err
	}
	if !ctx.JSON {
		_, _ = fmt.Fprintf(ctx.Writer, "Showing %d results (page %d, total %d).\n", count, page, total)
	}
	return nil
}

// PrintTotal prints items as a table and appends a "Total: N" footer.
func (ctx *RunContext) PrintTotal(items any, cols []output.Column, total int) error {
	if err := ctx.Printer.Print(items, cols); err != nil {
		return err
	}
	if !ctx.JSON {
		_, _ = fmt.Fprintf(ctx.Writer, "Total: %d\n", total)
	}
	return nil
}

// WriteResult prints a success message as plain text or JSON.
func (ctx *RunContext) WriteResult(message string) {
	writeResult(ctx.Writer, message)
}
