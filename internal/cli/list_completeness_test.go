package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// pagedCmd is a command with the --page/--limit pair most list commands have.
func pagedCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "x"}
	cmd.Flags().Int("page", 1, "")
	cmd.Flags().Int("limit", 20, "")
	return cmd
}

// TestListPageStatement pins the four completeness rules, in priority order:
// a known total wins, then a has-more signal, then a full page that cannot
// tell, then the bare count.
func TestListPageStatement(t *testing.T) {
	limitOnly := &cobra.Command{Use: "x"}
	limitOnly.Flags().Int("limit", 5, "")

	zeroPaged := &cobra.Command{Use: "x"}
	zeroPaged.Flags().Int("page", 0, "")
	zeroPaged.Flags().Int("limit", 20, "")

	tests := []struct {
		name       string
		cmd        *cobra.Command
		pg         listPage
		wantFooter string
		wantNote   string
	}{
		{
			name:       "a: total known, rows remain",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 20, Page: 1, Base: pageOne, Limit: 20, Total: 55},
			wantFooter: "Showing 20 results (page 1, total 55).",
			wantNote:   "note: showing 20 of 55 total results (page 1): raise --limit or use --page 2",
		},
		{
			name:       "a: total known, last page",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 15, Page: 3, Base: pageOne, Limit: 20, Total: 55},
			wantFooter: "Showing 15 results (page 3, total 55).",
		},
		{
			name:       "a: total beats has-more=false",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 2, Page: 1, Base: pageOne, Limit: 20, Total: 9, More: moreNo},
			wantFooter: "Showing 2 results (page 1, total 9).",
			wantNote:   "note: showing 2 of 9 total results (page 1): raise --limit or use --page 2",
		},
		{
			name:       "a: a total below the shown count is an absent field, not a total",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 3, Page: 1, Base: pageOne, Limit: 20, Total: 0},
			wantFooter: "Showing 3 results (page 1).",
		},
		{
			name:       "a: zero-based pages: first page, next is 1",
			cmd:        zeroPaged,
			pg:         listPage{Count: 20, Page: 0, Base: pageZero, Limit: 20, Total: 25},
			wantFooter: "Showing 20 results (page 0, total 25).",
			wantNote:   "note: showing 20 of 25 total results (page 0): raise --limit or use --page 1",
		},
		{
			name:       "a: zero-based pages: last page is complete",
			cmd:        zeroPaged,
			pg:         listPage{Count: 5, Page: 1, Base: pageZero, Limit: 20, Total: 25},
			wantFooter: "Showing 5 results (page 1, total 25).",
		},
		{
			name:       "a: unknown page base, first page: remainder known, no page number offered",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 20, Base: pageUnknown, Limit: 20, Total: 25},
			wantFooter: "Showing 20 results (total 25).",
			wantNote:   "note: showing 20 of 25 total results: raise --limit or use --page",
		},
		{
			name:       "a: unknown page base, later page: no claim of more",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 5, Page: 2, Base: pageUnknown, Limit: 20, Total: 25},
			wantFooter: "Showing 5 results (page 2, total 25).",
		},
		{
			name:       "a: unknown limit, later page: no claim of more",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 5, Page: 3, Base: pageOne, Total: 45},
			wantFooter: "Showing 5 results (page 3, total 45).",
		},
		{
			name:       "a: unknown limit, first page: remainder still known",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 5, Page: 1, Base: pageOne, Total: 45},
			wantFooter: "Showing 5 results (page 1, total 45).",
			wantNote:   "note: showing 5 of 45 total results (page 1): raise --limit or use --page 2",
		},
		{
			name:       "a: capped total on page 1 is a floor",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 100, Page: 1, Base: pageOne, Limit: 100, Total: 1000, More: moreYes, TotalCapped: true},
			wantFooter: "Showing 100 results (page 1, total at least 1000).",
			wantNote:   "note: showing 100 of at least 1000 total results (page 1): raise --limit or use --page 2",
		},
		{
			name:       "a: capped total past the cap, has-more true",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 39, Page: 11, Base: pageOne, Limit: 100, Total: 1000, More: moreYes, TotalCapped: true},
			wantFooter: "Showing 39 results (page 11, total at least 1000).",
			wantNote:   "note: showing 39 of at least 1000 total results (page 11): raise --limit or use --page 12",
		},
		{
			name:       "a: capped total past the cap, has-more false is the end",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 100, Page: 11, Base: pageOne, Limit: 100, Total: 1000, More: moreNo, TotalCapped: true},
			wantFooter: "Showing 100 results (page 11, total at least 1000).",
		},
		{
			name:       "a: capped total, has-more false beats the page offset",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 100, Page: 1, Base: pageOne, Limit: 100, Total: 1000, More: moreNo, TotalCapped: true},
			wantFooter: "Showing 100 results (page 1, total at least 1000).",
		},
		{
			name:       "a: capped endpoint, a total below the cap is exact",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 100, Page: 1, Base: pageOne, Limit: 100, Total: 999, More: moreYes, TotalCapped: true},
			wantFooter: "Showing 100 results (page 1, total 999).",
			wantNote:   "note: showing 100 of 999 total results (page 1): raise --limit or use --page 2",
		},
		{
			name:       "a: uncapped endpoint, a total of 1000 is exact",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 100, Page: 10, Base: pageOne, Limit: 100, Total: 1000},
			wantFooter: "Showing 100 results (page 10, total 1000).",
		},
		{
			name:       "b: has-more true, singular noun",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 1, Page: 1, Base: pageOne, Limit: 1, Total: -1, More: moreYes},
			wantFooter: "Showing 1 result (page 1), more exist: raise --limit or use --page 2.",
			wantNote:   "note: showing 1 result (page 1), more exist: raise --limit or use --page 2",
		},
		{
			name:       "b: has-more false, even on a full page",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 20, Page: 1, Base: pageOne, Limit: 20, Total: -1, More: moreNo},
			wantFooter: "Showing 20 results (page 1).",
		},
		{
			name:       "c: full page with no signal",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 20, Page: 1, Base: pageOne, Limit: 20, Total: -1},
			wantFooter: "Showing 20 results (page 1, = limit 20), more may exist: raise --limit or use --page 2.",
			wantNote:   "note: showing 20 results (page 1, = limit 20), more may exist: raise --limit or use --page 2",
		},
		{
			name:       "c: names only the flag the command has",
			cmd:        limitOnly,
			pg:         listPage{Count: 5, Limit: 5, Total: -1},
			wantFooter: "Showing 5 results (= limit 5), more may exist: raise --limit.",
			wantNote:   "note: showing 5 results (= limit 5), more may exist: raise --limit",
		},
		{
			name:       "d: short page with no signal",
			cmd:        pagedCmd(),
			pg:         listPage{Count: 7, Page: 1, Base: pageOne, Limit: 20, Total: -1},
			wantFooter: "Showing 7 results (page 1).",
		},
		{
			name:       "d: unknown limit",
			cmd:        &cobra.Command{Use: "x"},
			pg:         listPage{Count: 7, Total: -1},
			wantFooter: "Showing 7 results.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			footer, note := tc.pg.statement(tc.cmd)
			if footer != tc.wantFooter {
				t.Errorf("footer = %q, want %q", footer, tc.wantFooter)
			}
			if note != tc.wantNote {
				t.Errorf("note = %q, want %q", note, tc.wantNote)
			}
		})
	}
}

// A generated command whose response is a bare array carries no total and no
// has-more flag. A full page (rows == the limit the request carried) is the
// only signal there is, and it is announced on stderr while stdout stays the
// plain array.
func TestGeneratedBareArrayFullPageNotesOnStderr(t *testing.T) {
	rows := []any{
		map[string]any{"id": 1, "folder_id": 7, "name": "a"},
		map[string]any{"id": 2, "folder_id": 7, "name": "b"},
	}
	args := func(limit string) []string {
		return []string{"monit", "rule-list-basic", "--folder-id", "7", "--include-descendants", "--limit", limit, "--output-format", "json"}
	}

	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = rows
	full, fullErr, err := execCommandSplit(args("2")...)
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	want := "note: showing 2 results (= limit 2), more may exist: raise --limit"
	if !strings.Contains(fullErr, want) {
		t.Errorf("full page should note on stderr %q, got:\n%s", want, fullErr)
	}
	if strings.Contains(full, "note:") || !strings.HasPrefix(strings.TrimSpace(full), "[") {
		t.Errorf("stdout must stay the bare JSON array, got:\n%s", full)
	}

	saveAndResetGlobals(t)
	stub = newGFStub(t)
	stub.data = rows
	short, shortErr, err := execCommandSplit(args("5")...)
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	if strings.Contains(shortErr, "note:") {
		t.Errorf("a short page proves the set is complete; no note wanted, got:\n%s", shortErr)
	}
	if short != full {
		t.Errorf("stdout must not depend on the note:\nfull:  %s\nshort: %s", full, short)
	}
}

// pagedResp is a list envelope shaped like the SDK's has_next_page responses
// that carry no total.
type pagedResp struct {
	Items       []heuristicRow
	HasNextPage bool
}

// cursorResp is a list envelope that pages by an opaque next cursor.
type cursorResp struct {
	Items      []heuristicRow
	NextCursor string
}

// An envelope that reports has_next_page / a next cursor says so in table
// mode, and a total it never returned is not invented from the row count.
func TestGenericEnvelopeMoreSignalInTableMode(t *testing.T) {
	rows := []heuristicRow{{Name: "a"}, {Name: "b"}}
	cursorCmd := &cobra.Command{Use: "x"}
	cursorCmd.Flags().Int("limit", 20, "")
	tests := []struct {
		name     string
		cmd      *cobra.Command
		defaults listDefaults
		data     any
		want     string
	}{
		{"has_next_page true", pagedCmd(), listDefaults{PageBase: pageOne}, &pagedResp{Items: rows, HasNextPage: true}, "Showing 2 results (page 1), more exist: raise --limit or use --page 2."},
		{"has_next_page false", pagedCmd(), listDefaults{PageBase: pageOne}, &pagedResp{Items: rows}, "Showing 2 results (page 1)."},
		{"next cursor set", cursorCmd, listDefaults{}, &cursorResp{Items: rows, NextCursor: "abc"}, "Showing 2 results, more exist: raise --limit."},
		{"next cursor empty", cursorCmd, listDefaults{}, &cursorResp{Items: rows}, "Showing 2 results."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := tableCtx(&buf)
			ctx.Cmd = tc.cmd
			if err := printGenericResult(ctx, tc.data, nil, tc.defaults); err != nil {
				t.Fatalf("printGenericResult: %v", err)
			}
			got := buf.String()
			if !strings.Contains(got, tc.want) {
				t.Errorf("footer %q missing from:\n%s", tc.want, got)
			}
			if strings.Contains(got, "Total:") || strings.Contains(got, "total") {
				t.Errorf("no total was returned; none may be printed:\n%s", got)
			}
		})
	}
}

// alert events used to print the row count as "Total: N" with no way to page.
func TestAlertEventsStatesCompleteness(t *testing.T) {
	event := map[string]any{"event_id": "e1", "event_severity": "Critical", "event_status": "Triggered", "title": "t"}

	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{event, event}, "total": 45, "has_next_page": true}
	out, _, err := execCommandSplit("alert", "events", "a1", "--limit", "2", "--page", "2")
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	if stub.lastBody["limit"] != float64(2) || stub.lastBody["p"] != float64(2) {
		t.Errorf("--limit/--page must reach the request, got %#v", stub.lastBody)
	}
	if !strings.Contains(out, "Showing 2 results (page 2, total 45).") || strings.Contains(out, "Total:") {
		t.Errorf("table footer should state the real total, got:\n%s", out)
	}

	saveAndResetGlobals(t)
	stub = newGFStub(t)
	stub.data = map[string]any{"items": []any{event, event}, "total": 45, "has_next_page": true}
	jsonOut, stderrText, err := execCommandSplit("alert", "events", "a1", "--limit", "2", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(jsonOut), "[") || strings.Contains(jsonOut, "note:") {
		t.Errorf("stdout must stay the bare JSON array, got:\n%s", jsonOut)
	}
	if !strings.Contains(stderrText, "note: showing 2 of 45 total results (page 1): raise --limit or use --page 2") {
		t.Errorf("structured mode should note the remaining rows on stderr, got:\n%s", stderrText)
	}
}

// incident feed / alert timeline return has_next_page but no total.
func TestFeedCommandsSurfaceHasNextPage(t *testing.T) {
	item := map[string]any{"type": "i_comm", "creator_id": 0, "created_at": 1700000000}

	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{item}, "has_next_page": true}
	_, stderrText, err := execCommandSplit("incident", "feed", "inc-1", "--limit", "1", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	if !strings.Contains(stderrText, "note: showing 1 result (page 1), more exist: raise --limit or use --page 2") {
		t.Errorf("has_next_page=true must be announced, got:\n%s", stderrText)
	}

	saveAndResetGlobals(t)
	stub = newGFStub(t)
	stub.data = map[string]any{"items": []any{item}, "has_next_page": false}
	_, stderrText, err = execCommandSplit("alert", "timeline", "a1", "--limit", "1", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	if strings.Contains(stderrText, "note:") {
		t.Errorf("has_next_page=false is a complete list; no note wanted, got:\n%s", stderrText)
	}
}

// incident similar is a top-N answer with neither a total nor a has-more flag.
func TestIncidentSimilarFullListNotes(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	row := map[string]any{"incident_id": "p1", "title": "t", "score": 0.9}
	stub.data = map[string]any{"items": []any{row, row}}
	_, stderrText, err := execCommandSplit("incident", "similar", "inc-1", "--limit", "2", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	if !strings.Contains(stderrText, "note: showing 2 results (= limit 2), more may exist: raise --limit") {
		t.Errorf("a full top-N list should say more may exist, got:\n%s", stderrText)
	}
}

// With no --limit the request is served the schema's declared default page
// size (rule-list-basic: 50), so a full default page still cannot prove the
// list is complete.
func TestGeneratedBareArrayDefaultLimit(t *testing.T) {
	makeRows := func(n int) []any {
		rows := make([]any, n)
		for i := range rows {
			rows[i] = map[string]any{"id": i + 1, "folder_id": 7, "name": "r"}
		}
		return rows
	}
	run := func(t *testing.T, n int) (stdout, stderr string) {
		t.Helper()
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = makeRows(n)
		out, errText, err := execCommandSplit("monit", "rule-list-basic", "--folder-id", "7", "--include-descendants", "--output-format", "json")
		if err != nil {
			t.Fatalf("execCommandSplit: %v", err)
		}
		if _, sent := stub.lastBody["limit"]; sent {
			t.Fatalf("no --limit was given; none may be sent, got %#v", stub.lastBody)
		}
		return out, errText
	}

	fullOut, fullErr := run(t, 50)
	if want := "note: showing 50 results (= limit 50), more may exist: raise --limit"; !strings.Contains(fullErr, want) {
		t.Errorf("a full default page should note %q, got:\n%s", want, fullErr)
	}
	if strings.Contains(fullOut, "note:") || !strings.HasPrefix(strings.TrimSpace(fullOut), "[") {
		t.Errorf("stdout must stay the bare JSON array, got prefix %q", fullOut[:min(len(fullOut), 40)])
	}

	_, shortErr := run(t, 3)
	if strings.Contains(shortErr, "note:") {
		t.Errorf("a short default page proves the set is complete; no note wanted, got:\n%s", shortErr)
	}
}

// A command whose schema declares a zero-based page (default 0) is on page 0
// when it sets none: the next page is 1, and page 1 of 25 rows at limit 20 is
// the last page.
func TestZeroBasedPageHintAndOffset(t *testing.T) {
	rows := func(n int) []any {
		out := make([]any, n)
		for i := range out {
			out[i] = map[string]any{"version": i + 1}
		}
		return out
	}
	run := func(t *testing.T, n int, extra ...string) string {
		t.Helper()
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": rows(n), "total": 25}
		args := append([]string{"rum", "application-remote-config-history-list", "app1", "--output-format", "json"}, extra...)
		_, stderrText, err := execCommandSplit(args...)
		if err != nil {
			t.Fatalf("execCommandSplit: %v", err)
		}
		return stderrText
	}

	if got, want := run(t, 20), "note: showing 20 of 25 total results (page 0): raise --limit or use --page 1"; !strings.Contains(got, want) {
		t.Errorf("first zero-based page should note %q, got:\n%s", want, got)
	}
	if got := run(t, 5, "--page", "1"); strings.Contains(got, "note:") {
		t.Errorf("page 1 of 25 rows at limit 20 is the last page; no note wanted, got:\n%s", got)
	}
}

// alert events sends --limit/--page only when the user set them, and judges the
// page against the schema's default page size.
func TestAlertEventsSendsFlagsOnlyWhenSet(t *testing.T) {
	event := map[string]any{"event_id": "e1", "event_severity": "Critical", "event_status": "Triggered", "title": "t"}

	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{event}, "total": 1}
	if _, _, err := execCommandSplit("alert", "events", "a1"); err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	if _, ok := stub.lastBody["limit"]; ok {
		t.Errorf("unset --limit must not be sent, got %#v", stub.lastBody)
	}
	if _, ok := stub.lastBody["p"]; ok {
		t.Errorf("unset --page must not be sent, got %#v", stub.lastBody)
	}

	// 20 rows of 45 with nothing set: the default page size (20) and first
	// page are known, so the remainder is announced.
	saveAndResetGlobals(t)
	stub = newGFStub(t)
	rows := make([]any, 20)
	for i := range rows {
		rows[i] = event
	}
	stub.data = map[string]any{"items": rows, "total": 45}
	_, stderrText, err := execCommandSplit("alert", "events", "a1", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommandSplit: %v", err)
	}
	if !strings.Contains(stderrText, "note: showing 20 of 45 total results (page 1): raise --limit or use --page 2") {
		t.Errorf("default page of a longer list should be announced, got:\n%s", stderrText)
	}
}
