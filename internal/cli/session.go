package cli

import (
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

// sessionListFormats are the output shapes 'session list' supports. jsonl (one
// SessionItem JSON object per line) is the default because the rows feed
// line-oriented downstream tooling (the /insight skill streams them through jq);
// json emits the whole SessionListResponse envelope; toon is the compact,
// fewer-tokens encoding.
const (
	sessionFormatJSONL = "jsonl"
	sessionFormatJSON  = "json"
	sessionFormatTOON  = "toon"
)

func newSessionListCmd() *cobra.Command {
	var (
		app    string
		scope  string
		status string
		since  string
		format string
		teamID int64
		limit  int
		page   int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List agent sessions",
		Long: curatedLong(
			"List agent sessions visible to the caller, newest first. Reads are scoped to the "+
				"person the app_key resolves to within its account.\n\n"+
				"--app selects the agent app (default ai-sre). The API has no time-window filter, so "+
				"--since (e.g. 30d, 24h, 2026-05-01) is applied CLIENT-SIDE against each session's "+
				"updated_at after fetching. --team-id restricts to one team (sets team_ids); --scope "+
				"chooses the visibility bucket (all = own + member-teams, the default). Output is "+
				"newline-delimited JSON (jsonl) by default so rows pipe straight into jq; use "+
				"--format json for the full envelope or --format toon for the compact encoding.",
			"Sessions", "List"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				format = strings.ToLower(strings.TrimSpace(format))
				switch format {
				case sessionFormatJSONL, sessionFormatJSON, sessionFormatTOON:
				default:
					return fmt.Errorf("invalid --format %q (want jsonl, json, or toon)", format)
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
				req.Limit = limit
				req.Page = page
				if teamID > 0 {
					req.TeamIDs = []int64{teamID}
				}

				resp, _, err := ctx.Client.Sessions.List(cmdContext(ctx.Cmd), req)
				if err != nil {
					return err
				}

				sessions := resp.Sessions
				if sinceUnix > 0 {
					sessions = filterSessionsSince(sessions, sinceUnix)
				}

				return writeSessionList(ctx.Writer, format, sessions, resp.Total)
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
	cmd.Flags().IntVar(&limit, "limit", 200, "Max sessions to fetch (server caps at 100/page)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().StringVar(&format, "format", sessionFormatJSONL, "Output format: jsonl (default), json, or toon")
	registerEnumFlag(cmd, "format", sessionFormatJSONL, sessionFormatJSON, sessionFormatTOON)

	return cmd
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
		Long: "Stream one session's full event transcript as newline-delimited JSON (NDJSON) to stdout.\n\n" +
			"The first line is always a session_meta envelope; each subsequent line is one event\n" +
			"(user_message, llm_call, tool_call, subagent_dispatch, final_answer, agent_text, error).\n" +
			"With --include-subagents, each subagent_dispatch line is followed by the child session's\n" +
			"own stream. The transcript can be large, so redirect it to a file rather than reading it\n" +
			"into a terminal:\n\n" +
			"  flashduty session export <id> > session.ndjson\n",
		Args: requireArgs("session_id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				rc, _, err := ctx.Client.Sessions.Export(cmdContext(ctx.Cmd), &flashduty.SessionExportRequest{
					SessionID:        ctx.Args[0],
					IncludeSubagents: includeSubagents,
				})
				if err != nil {
					return err
				}
				defer func() { _ = rc.Close() }()

				// Stream the NDJSON straight through to the writer without
				// buffering the whole transcript: copy line-by-line so a huge
				// export never lands in memory or the agent's context.
				sc := flashduty.NewExportScanner(rc)
				for sc.Scan() {
					if _, err := fmt.Fprintln(ctx.Writer, sc.Text()); err != nil {
						return err
					}
				}
				return sc.Err()
			})
		},
	}

	cmd.Flags().BoolVar(&includeSubagents, "include-subagents", false, "Inline each dispatched subagent's own event stream")

	return cmd
}

// attachSafariSessionExport adds the path-is-king `safari session-export` leaf to
// the generated `safari` group. It must run AFTER registerGenerated so the group
// exists; genGroup find-or-creates it and genAddLeaf is a no-op if a same-named
// command is already present.
func attachSafariSessionExport(root *cobra.Command) {
	safari := genGroup(root, "safari", "AI SRE API")
	genAddLeaf(safari, newSafariSessionExportCmd())
}
