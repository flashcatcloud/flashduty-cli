package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"

	"github.com/flashcatcloud/flashduty-cli/internal/timeutil"
)

func newSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Inspect AI agent sessions",
		Long: "Inspect AI agent sessions (AI SRE and other Flashduty agents).\n\n" +
			"'session list' enumerates sessions visible to the caller; 'session export' streams\n" +
			"one session's full event transcript as newline-delimited JSON for offline analysis.",
	}
	cmd.AddCommand(newSessionListCmd())
	cmd.AddCommand(newSessionExportCmd())
	return cmd
}

// session commands accept these output shapes via the global --output-format
// flag (and its --json alias). jsonl (one SessionItem JSON object per line) is
// the default because the rows feed line-oriented downstream tooling (the
// /insight skill streams them through jq); json emits the whole
// SessionListResponse envelope; toon is the compact, fewer-tokens encoding.
//
// jsonl is NOT a value the global table|json|toon resolver accepts, so session
// commands carry the ownsOutputFormat annotation and resolve the flag here via
// resolveSessionFormat instead of through resolveOutputFormat.
const (
	sessionFormatJSONL = "jsonl"
	sessionFormatJSON  = "json"
	sessionFormatTOON  = "toon"
)

// resolveSessionFormat maps the global --output-format / --json flags to a
// session output shape, defaulting to jsonl. Unlike the account-wide resolver
// it accepts jsonl (and rejects table, which is meaningless for the bulk
// streaming rows these commands emit). An unrecognized value errors so a typo
// fails fast rather than silently falling back.
func resolveSessionFormat() (string, error) {
	switch f := strings.ToLower(strings.TrimSpace(flagOutputFormat)); f {
	case sessionFormatJSONL, sessionFormatJSON, sessionFormatTOON:
		return f, nil
	case "":
		if flagJSON {
			return sessionFormatJSON, nil
		}
		return sessionFormatJSONL, nil
	default:
		return "", fmt.Errorf("invalid --output-format %q (want jsonl, json, or toon)", flagOutputFormat)
	}
}

// sessionPageLimit is the largest per-page Limit the /safari/session/list
// handler accepts. The server validates limit with binding "lte=100": a
// limit > 100 is a hard 400 bind failure, NOT a clamp, so every page request
// must carry Limit <= 100. To honor a --limit above this, `session list`
// paginates server-side (see fetchSessionsPaged).
const sessionPageLimit = 100

func newSessionListCmd() *cobra.Command {
	var (
		app    string
		scope  string
		status string
		since  string
		teamID int64
		limit  int
		page   int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List agent sessions",
		// Resolve --output-format ourselves: jsonl is the default and is not a
		// value the global table|json|toon resolver accepts.
		Annotations: map[string]string{ownsOutputFormat: "true"},
		Long: curatedLong(
			"List agent sessions visible to the caller, newest first. Reads are scoped to the "+
				"person the app_key resolves to within its account.\n\n"+
				"--app selects the agent app (default ai-sre). The API has no time-window filter, so "+
				"--since (e.g. 30d, 24h, 2026-05-01) is applied CLIENT-SIDE against each session's "+
				"updated_at after fetching. --team-id restricts to one team (sets team_ids); --scope "+
				"chooses the visibility bucket (all = own + member-teams, the default). Output is "+
				"newline-delimited JSON (jsonl) by default so rows pipe straight into jq; use "+
				"--output-format json for the full envelope or --output-format toon for the compact "+
				"encoding.",
			"Sessions", "List"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				format, err := resolveSessionFormat()
				if err != nil {
					return err
				}

				var sinceUnix int64
				if since != "" {
					ts, err := timeutil.Parse(since)
					if err != nil {
						return fmt.Errorf("invalid --since: %w", err)
					}
					sinceUnix = ts
				}

				req := &flashduty.SessionListRequest{
					AppName: app,
					Scope:   scope,
					Status:  status,
					Orderby: "updated_at",
				}
				if teamID > 0 {
					req.TeamIDs = []int64{teamID}
				}

				sessions, total, err := fetchSessionsPaged(cmdContext(ctx.Cmd), ctx.Client, req, page, limit)
				if err != nil {
					return err
				}

				if sinceUnix > 0 {
					sessions = filterSessionsSince(sessions, sinceUnix)
				}

				return writeSessionList(ctx.Writer, format, sessions, total)
			})
		},
	}

	cmd.Flags().StringVar(&app, "app", "ai-sre", "Agent app to list sessions for")
	cmd.Flags().StringVar(&scope, "scope", "", "Visibility scope: all (own + member-teams, default), personal, or team")
	registerEnumFlag(cmd, "scope", "all", "personal", "team")
	cmd.Flags().StringVar(&status, "status", "", "Archive bucket: active (default), archived, or all")
	registerEnumFlag(cmd, "status", "active", "archived", "all")
	cmd.Flags().StringVar(&since, "since", "", "Keep only sessions updated within this window (client-side), e.g. 30d, 24h, 2026-05-01")
	cmd.Flags().Int64Var(&teamID, "team-id", 0, "Restrict to one team ID")
	cmd.Flags().IntVar(&limit, "limit", 200, "Max sessions to fetch; fetched across multiple 100-row server pages as needed")
	cmd.Flags().IntVar(&page, "page", 1, "1-based page to start paginating from")
	// --output-format is the inherited global flag; session commands accept
	// jsonl (default), json, or toon. Override its completion so it advertises
	// the session set, not the global table|json|toon.
	registerEnumFlag(cmd, "output-format", sessionFormatJSONL, sessionFormatJSON, sessionFormatTOON)

	return cmd
}

// fetchSessionsPaged collects up to `limit` sessions across as many server pages
// as needed, starting at page `startPage`. The /safari/session/list handler
// rejects any single request with Limit > 100 (binding "lte=100" → HTTP 400, not
// a clamp), so a --limit above 100 must be satisfied by paginating: each page
// requests min(remaining, 100) rows and advances the 1-based page number P. The
// loop stops once it has `limit` rows, the server reports it has returned every
// matching row (accumulated >= Total), or a page comes back short (fewer rows
// than requested means the server is exhausted). The Total from the last
// response is returned so the caller can report the full match count even when
// the rows were truncated to --limit.
func fetchSessionsPaged(
	ctx context.Context,
	client *flashduty.Client,
	base *flashduty.SessionListRequest,
	startPage, limit int,
) ([]flashduty.SessionItem, int64, error) {
	if startPage < 1 {
		startPage = 1
	}
	if limit < 1 {
		limit = 1
	}

	// Hint the slice at one page; it grows naturally across pages. Sizing it to
	// `limit` would over-allocate when a huge --limit far exceeds what the server
	// actually has (e.g. --limit 1000000 on an account with a few hundred rows).
	capHint := limit
	if capHint > sessionPageLimit {
		capHint = sessionPageLimit
	}
	collected := make([]flashduty.SessionItem, 0, capHint)
	var total int64
	for page := startPage; len(collected) < limit; page++ {
		pageLimit := limit - len(collected)
		if pageLimit > sessionPageLimit {
			pageLimit = sessionPageLimit
		}

		// Copy the filter so each page reuses the same scope/app/team but
		// carries its own pagination cursor.
		req := *base
		req.Page = page
		req.Limit = pageLimit

		resp, _, err := client.Sessions.List(ctx, &req)
		if err != nil {
			return nil, 0, err
		}
		total = resp.Total
		collected = append(collected, resp.Sessions...)

		// Server exhausted: a short page (fewer rows than asked for) or we have
		// already gathered every matching row. Either ends the loop.
		if len(resp.Sessions) < pageLimit || int64(len(collected)) >= total {
			break
		}
	}

	if len(collected) > limit {
		collected = collected[:limit]
	}
	return collected, total, nil
}

// filterSessionsSince keeps sessions whose updated_at is at or after sinceUnix
// (unix seconds). The API exposes no time-window filter, so this is the only
// place a --since window is honored.
func filterSessionsSince(sessions []flashduty.SessionItem, sinceUnix int64) []flashduty.SessionItem {
	kept := make([]flashduty.SessionItem, 0, len(sessions))
	for _, s := range sessions {
		if s.UpdatedAt.Time().Unix() >= sinceUnix {
			kept = append(kept, s)
		}
	}
	return kept
}

// writeSessionList renders the session rows in the requested format. jsonl emits
// one SessionItem per line; json emits the whole SessionListResponse envelope;
// toon emits the compact encoding of that envelope.
func writeSessionList(w io.Writer, format string, sessions []flashduty.SessionItem, total int64) error {
	switch format {
	case sessionFormatJSONL:
		enc := json.NewEncoder(w)
		for i := range sessions {
			if err := enc.Encode(sessions[i]); err != nil {
				return fmt.Errorf("failed to encode session: %w", err)
			}
		}
		return nil
	default:
		envelope := flashduty.SessionListResponse{Sessions: sessions, Total: total}
		var (
			out []byte
			err error
		)
		if format == sessionFormatTOON {
			out, err = toon.Marshal(envelope)
		} else {
			out, err = json.MarshalIndent(envelope, "", "  ")
		}
		if err != nil {
			return fmt.Errorf("failed to marshal sessions: %w", err)
		}
		_, _ = fmt.Fprintln(w, string(out))
		return nil
	}
}

// newSessionExportCmd builds the friendly `session export <id>` command.
func newSessionExportCmd() *cobra.Command {
	return buildSessionExportCmd("export <session_id>")
}

// newSafariSessionExportCmd builds the path-is-king `safari session-export <id>`
// command. session/export is a streaming op, so it is excluded from the
// generated tree (which cannot model an io.ReadCloser response); this curated
// leaf keeps the operation reachable at its mechanical path-name alongside the
// generated safari session-get / session-list.
func newSafariSessionExportCmd() *cobra.Command {
	return buildSessionExportCmd("session-export <session_id>")
}

// buildSessionExportCmd constructs an export command with the given Use line.
// Both the friendly and path-is-king commands share this one implementation so
// the streaming behavior is defined once.
func buildSessionExportCmd(use string) *cobra.Command {
	var includeSubagents bool

	cmd := &cobra.Command{
		Use:   use,
		Short: "Stream a session's full event transcript as NDJSON",
		// Resolve --output-format ourselves: jsonl is the default and is not a
		// value the global table|json|toon resolver accepts.
		Annotations: map[string]string{ownsOutputFormat: "true"},
		Long: "Stream one session's full event transcript as newline-delimited JSON (NDJSON) to stdout.\n\n" +
			"The first line is always a session_meta envelope; each subsequent line is one event\n" +
			"(user_message, llm_call, tool_call, subagent_dispatch, final_answer, agent_text, error).\n" +
			"With --include-subagents, each subagent_dispatch line is followed by the child session's\n" +
			"own stream.\n\n" +
			"The default (jsonl) streams line-by-line so a huge transcript never lands in memory;\n" +
			"redirect it to a file rather than reading it into a terminal. --output-format json\n" +
			"buffers the whole transcript into a single JSON array and --output-format toon into the\n" +
			"compact encoding (both materialize the full transcript, so prefer jsonl for large ones):\n\n" +
			"  flashduty session export <id> > session.ndjson\n",
		Args: requireArgs("session_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				format, err := resolveSessionFormat()
				if err != nil {
					return err
				}

				rc, _, err := ctx.Client.Sessions.Export(cmdContext(ctx.Cmd), &flashduty.SessionExportRequest{
					SessionID:        ctx.Args[0],
					IncludeSubagents: includeSubagents,
				})
				if err != nil {
					return err
				}
				defer func() { _ = rc.Close() }()

				return writeSessionExport(ctx.Writer, format, rc)
			})
		},
	}

	cmd.Flags().BoolVar(&includeSubagents, "include-subagents", false, "Inline each dispatched subagent's own event stream")
	// --output-format is the inherited global flag; session export accepts
	// jsonl (default, streamed), json, or toon. Override its completion so it
	// advertises the session set, not the global table|json|toon.
	registerEnumFlag(cmd, "output-format", sessionFormatJSONL, sessionFormatJSON, sessionFormatTOON)

	return cmd
}

// writeSessionExport renders the export NDJSON stream in the requested format.
// jsonl streams each line straight through without buffering, so a huge
// transcript never lands in memory; json and toon necessarily materialize the
// whole transcript (those encodings need every line) — json emits one indented
// JSON array of the event objects, toon emits the compact encoding.
func writeSessionExport(w io.Writer, format string, rc io.Reader) error {
	sc := flashduty.NewExportScanner(rc)

	if format == sessionFormatJSONL {
		for sc.Scan() {
			if _, err := fmt.Fprintln(w, sc.Text()); err != nil {
				return err
			}
		}
		return sc.Err()
	}

	// json/toon: collect every event line, then encode the whole array.
	events := make([]json.RawMessage, 0, 256)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		events = append(events, json.RawMessage(line))
	}
	if err := sc.Err(); err != nil {
		return err
	}

	var (
		out []byte
		err error
	)
	if format == sessionFormatTOON {
		// TOON marshals Go values, not raw JSON, so decode the events first.
		decoded := make([]any, 0, len(events))
		for _, raw := range events {
			var v any
			if err := json.Unmarshal(raw, &v); err != nil {
				return fmt.Errorf("failed to decode export event: %w", err)
			}
			decoded = append(decoded, v)
		}
		out, err = toon.Marshal(decoded)
	} else {
		out, err = json.MarshalIndent(events, "", "  ")
	}
	if err != nil {
		return fmt.Errorf("failed to marshal export: %w", err)
	}
	_, _ = fmt.Fprintln(w, string(out))
	return nil
}

// attachSafariSessionExport adds the path-is-king `safari session-export` leaf to
// the generated `safari` group. It must run AFTER registerGenerated so the group
// exists; genGroup find-or-creates it and genAddLeaf is a no-op if a same-named
// command is already present.
func attachSafariSessionExport(root *cobra.Command) {
	safari := genGroup(root, "safari", "AI SRE API")
	genAddLeaf(safari, newSafariSessionExportCmd())
}
