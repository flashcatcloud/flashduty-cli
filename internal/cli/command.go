package cli

import (
	"fmt"
	"io"
	"strconv"

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

// pageBase says how a command numbers its pages. The zero value means the
// command has no page notion at all.
type pageBase int8

const (
	pageNone    pageBase = iota // no page number (top-N lists, one-shot lists)
	pageOne                     // pages start at 1
	pageZero                    // pages start at 0
	pageUnknown                 // pages by number, base not declared by the API schema
)

// listDefaults is what an API's request schema declares for a list: the page
// size served when the request sets none (0 when undeclared) and how pages are
// numbered.
type listDefaults struct {
	Limit    int
	PageBase pageBase
}

// listDefaultsFor returns the schema-declared defaults of an SDK Service.Method.
func listDefaultsFor(service, method string) listDefaults {
	return listDefaultsBySDKMethod[service+"."+method]
}

// listPage is everything a list response tells us about its own completeness.
// It is the single input to the completeness statement every list output makes,
// so a partial page is never mistaken for the whole set.
type listPage struct {
	Count int       // rows shown
	Page  int       // page number as sent to the API (0 when the request set none)
	Base  pageBase  // how Page is numbered
	Limit int       // effective page size: requested, else the schema default; 0 when unknown
	Total int       // total matching rows; negative when the API returned none
	More  moreState // has_next_page / next cursor, when the API returned one
	// TotalCapped marks an endpoint that stops counting at serverCountCap; its
	// callers must also set More, the only "rows remain" signal past the cap.
	TotalCapped bool
	// Siblings names the response fields beside the rows that are not
	// pagination metadata; a table shows only the rows.
	Siblings []string
}

// totalKnown reports whether Total is a usable server total. A total below the
// shown count is an absent field, not a real total.
func (p listPage) totalKnown() bool { return p.Total >= 0 && p.Total >= p.Count }

// serverCountCap is where the incident, alert and alert-event list endpoints
// stop counting: past it they report this value as the total while rows keep
// coming (the alert and alert-event API reference says "capped at 1000").
const serverCountCap = 1000

// totalIsFloor reports whether Total is a capped count, which says only that at
// least that many rows match.
func (p listPage) totalIsFloor() bool { return p.TotalCapped && p.Total >= serverCountCap }

// offset returns how many rows precede this page, when that is knowable: always
// for the first page, and for later pages only with a known page size and base.
func (p listPage) offset() (int, bool) {
	switch {
	case p.Base == pageNone, p.Page == 0 && p.Base != pageOne, p.Base == pageOne && p.Page <= 1:
		return 0, true
	case p.Limit <= 0:
		return 0, false
	case p.Base == pageOne:
		return max(p.Page-1, 0) * p.Limit, true
	case p.Base == pageZero:
		return p.Page * p.Limit, true
	}
	return 0, false
}

// nextPage returns the page number after this one, when the base is known.
func (p listPage) nextPage() (int, bool) {
	if p.Base != pageOne && p.Base != pageZero {
		return 0, false
	}
	if p.Base == pageOne {
		return max(p.Page, 1) + 1, true
	}
	return p.Page + 1, true
}

// results renders a row count with its noun.
func results(n int) string {
	if n == 1 {
		return "1 result"
	}
	return fmt.Sprintf("%d results", n)
}

// statement applies the completeness rules in order and returns the table
// footer and the structured-mode stderr note ("" when no warning applies):
//
//	a. total known: "showing N of T", plus a next-page hint when rows remain;
//	   a capped total reads "at least T", and has_next_page alone decides
//	   whether rows remain;
//	b. else has_next_page/cursor known: true warns, false is complete;
//	c. else a full page (N == limit) cannot tell: warn that more may exist;
//	d. else only the plain count.
//
// A count is never labelled a total, and "rows remain" is claimed from a page
// offset only when that offset is known.
func (p listPage) statement(cmd *cobra.Command) (footer, note string) {
	pageText := ""
	if p.Base != pageNone && (p.Page > 0 || p.Base == pageOne || p.Base == pageZero) {
		pageText = fmt.Sprintf("page %d", p.Page)
	}
	paren := func(extra string) string {
		switch {
		case pageText != "" && extra != "":
			return " (" + pageText + ", " + extra + ")"
		case pageText != "":
			return " (" + pageText + ")"
		case extra != "":
			return " (" + extra + ")"
		}
		return ""
	}
	hint := moreHint(cmd, p)
	// warn builds the footer and note for a partial list: "<shown>, <why>: <how>".
	warn := func(shown, why string) (string, string) {
		body := "showing " + shown
		if why != "" {
			body += ", " + why
		}
		if hint != "" {
			body += ": " + hint
		}
		return "S" + body[1:] + ".", "note: " + body
	}
	switch {
	case p.totalKnown():
		total := strconv.Itoa(p.Total)
		if p.totalIsFloor() {
			total = "at least " + total
		}
		footer = fmt.Sprintf("Showing %s%s.", results(p.Count), paren("total "+total))
		off, offKnown := p.offset()
		remain := offKnown && off+p.Count < p.Total
		if p.totalIsFloor() {
			remain = p.More == moreYes
		}
		if remain {
			_, note = warn(fmt.Sprintf("%d of %s total results%s", p.Count, total, paren("")), "")
		}
		return footer, note
	case p.More == moreYes:
		return warn(results(p.Count)+paren(""), "more exist")
	case p.More == moreNo:
		return fmt.Sprintf("Showing %s%s.", results(p.Count), paren("")), ""
	case p.Limit > 0 && p.Count == p.Limit:
		return warn(results(p.Count)+paren(fmt.Sprintf("= limit %d", p.Limit)), "more may exist")
	}
	return fmt.Sprintf("Showing %s%s.", results(p.Count), paren("")), ""
}

// moreHint names the flags this command actually has for reaching further rows.
// A page number is offered only when the page base is known.
func moreHint(cmd *cobra.Command, p listPage) string {
	has := func(name string) bool { return cmd.Flags().Lookup(name) != nil }
	paged := p.Base != pageNone && has("page")
	next, nextKnown := p.nextPage()
	pageFlag := "--page"
	if nextKnown {
		pageFlag = fmt.Sprintf("--page %d", next)
	}
	switch {
	case paged && has("limit"):
		return "raise --limit or use " + pageFlag
	case paged:
		return "use " + pageFlag
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
	return ctx.PrintPage(items, cols, listPage{Count: count, Page: page, Base: pageOne, Limit: limit, Total: total})
}

// PrintCappedList is PrintList for an endpoint whose total stops at
// serverCountCap; hasNext is the response's has_next_page.
func (ctx *RunContext) PrintCappedList(items any, cols []output.Column, count, page, limit, total int, hasNext bool) error {
	return ctx.PrintPage(items, cols, listPage{
		Count: count, Page: page, Base: pageOne, Limit: limit, Total: total, More: moreOf(hasNext), TotalCapped: true,
	})
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
