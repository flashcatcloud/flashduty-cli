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
		Printer: newPrinter(cmd),
		Format:  currentOutputFormat(),
	}
	return fn(ctx)
}

// moreState is what a list response says about whether rows exist beyond the
// ones returned.
type moreState int8

const (
	moreUnknown moreState = iota // the API gave no signal
	moreNo                       // has_next_page=false / empty next cursor
	moreYes                      // has_next_page=true / non-empty next cursor
)

// moreOf maps a response's has_next_page flag to a moreState.
func moreOf(hasNext bool) moreState {
	if hasNext {
		return moreYes
	}
	return moreNo
}

// listPage is everything a list response tells us about its own completeness.
// It is the single input to the completeness statement every list output makes,
// so a partial page is never mistaken for the whole set.
type listPage struct {
	Count int       // rows shown
	Page  int       // 1-based page; 0 when the command has no page notion
	Limit int       // page size requested or the known default; 0 when unknown
	Total int       // total matching rows; negative when the API returned none
	More  moreState // has_next_page / next cursor, when the API returned one
}

// totalKnown reports whether Total is a usable server total. A total below the
// shown count is an absent field decoded as zero, not a real total.
func (p listPage) totalKnown() bool { return p.Total >= 0 && p.Total >= p.Count }

// statement applies the completeness rules in order and returns the table
// footer and the structured-mode stderr note ("" when no warning applies):
//
//	a. total known: "showing N of T", plus a next-page hint when rows remain;
//	b. else has_next_page/cursor known: true warns, false is complete;
//	c. else a full page (N == limit) cannot tell: warn that more may exist;
//	d. else only the plain count.
//
// A count is never labelled a total.
func (p listPage) statement(cmd *cobra.Command) (footer, note string) {
	pageSuffix := ""
	if p.Page > 0 {
		pageSuffix = fmt.Sprintf(" (page %d)", p.Page)
	}
	hint := moreHint(cmd, p.Page)
	warn := func(body string) (string, string) {
		if hint != "" {
			body += "; " + hint
		}
		return "S" + body[1:] + ".", "note: " + body
	}
	switch {
	case p.totalKnown():
		if p.Page > 0 {
			footer = fmt.Sprintf("Showing %d results (page %d, total %d).", p.Count, p.Page, p.Total)
		} else {
			footer = fmt.Sprintf("Showing %d results (total %d).", p.Count, p.Total)
		}
		offset := (max(p.Page, 1) - 1) * p.Limit
		if p.Limit == 0 {
			offset = (max(p.Page, 1) - 1) * p.Count
		}
		if offset+p.Count < p.Total {
			_, note = warn(fmt.Sprintf("showing %d of %d total results%s", p.Count, p.Total, pageSuffix))
		}
		return footer, note
	case p.More == moreYes:
		return warn(fmt.Sprintf("showing %d results%s; more results exist", p.Count, pageSuffix))
	case p.More == moreNo:
		return fmt.Sprintf("Showing %d results%s.", p.Count, pageSuffix), ""
	case p.Limit > 0 && p.Count == p.Limit:
		return warn(fmt.Sprintf("showing %d results%s (= limit %d); more may exist", p.Count, pageSuffix, p.Limit))
	}
	return fmt.Sprintf("Showing %d results%s.", p.Count, pageSuffix), ""
}

// moreHint names the flags this command actually has for reaching further rows.
// page is the page just shown; 0 means the command has no page notion, so --page
// is not offered.
func moreHint(cmd *cobra.Command, page int) string {
	has := func(name string) bool { return cmd.Flags().Lookup(name) != nil }
	switch {
	case page > 0 && has("page") && has("limit"):
		return fmt.Sprintf("raise --limit or use --page %d", page+1)
	case page > 0 && has("page"):
		return fmt.Sprintf("use --page %d", page+1)
	case has("search-after-ctx"):
		return "pass the response's search_after_ctx as --search-after-ctx"
	case has("cursor"):
		return "pass the response's next_cursor as --cursor"
	case has("limit"):
		return "raise --limit"
	}
	return ""
}

// announce states how complete the list just printed is: a footer on stdout in
// table mode, a stderr note (only when rows may be missing) in structured mode
// so stdout stays byte-pure for jq/toon pipelines.
func (ctx *RunContext) announce(pg listPage) {
	if ctx.Structured() {
		ctx.noteIncomplete(pg)
		return
	}
	footer, _ := pg.statement(ctx.Cmd)
	_, _ = fmt.Fprintln(ctx.Writer, footer)
}

// noteIncomplete writes the stderr note when rows may be missing. Commands that
// own their output format (session list) call it directly.
func (ctx *RunContext) noteIncomplete(pg listPage) {
	if _, note := pg.statement(ctx.Cmd); note != "" {
		_, _ = fmt.Fprintln(ctx.Cmd.ErrOrStderr(), note)
	}
}

// PrintPage prints items and then states how complete they are (see listPage).
func (ctx *RunContext) PrintPage(items any, cols []output.Column, pg listPage) error {
	if err := ctx.Printer.Print(items, cols); err != nil {
		return err
	}
	ctx.announce(pg)
	return nil
}

// PrintList is PrintPage for a response that carried a total.
func (ctx *RunContext) PrintList(items any, cols []output.Column, count, page, limit, total int) error {
	return ctx.PrintPage(items, cols, listPage{Count: count, Page: page, Limit: limit, Total: total})
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
// Cobra still answers --help before ValidateArgs, which is why Execute also
// runs rejectUnknownSubcommand ahead of dispatch.
func groupUnknownSubcommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return unknownCommandError(cmd, args[0])
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
