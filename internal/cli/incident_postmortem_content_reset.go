package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"
)

const maxPostMortemContentIdempotencyKeyRunes = 128

func newIncidentPostMortemContentResetCmd() *cobra.Command {
	var (
		markdownFile     string
		expectedRevision int64
		idempotencyKey   string
	)

	cmd := &cobra.Command{
		Use:   "post-mortem-content-reset <post-mortem-id>",
		Short: "Reset post-mortem Markdown content",
		Long: curatedLong(
			"Replace the collaborative Markdown body of a drafting post-mortem report in one shot.\n\n"+
				"Read Markdown from --markdown-file (or \"-\" for stdin). The entire file is sent as-is; leading and trailing content is preserved. Empty Markdown is rejected.\n\n"+
				"--expected-revision guards against overwriting a concurrent edit: the reset only succeeds when the document's current revision equals it, and 0 is valid (first write / empty document). Negative values are rejected. When omitted, the CLI first fetches the report's current revision via the post-mortem info endpoint and uses that; pass it explicitly for strict concurrency control, when the caller already holds a revision and must fail on any intervening write.\n\n"+
				"--idempotency-key is required, must be non-empty, and at most 128 Unicode characters.",
			"Incidents",
			"PostMortemWriteResetContent",
		),
		Args: requireExactArg("post-mortem-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				markdown, err := readPostMortemMarkdownFile(markdownFile)
				if err != nil {
					return err
				}
				if err := validatePostMortemContentResetFlags(idempotencyKey); err != nil {
					return err
				}

				revision := expectedRevision
				if cmd.Flags().Changed("expected-revision") {
					if revision < 0 {
						return fmt.Errorf("--expected-revision must be >= 0")
					}
				} else {
					revision, err = currentPostMortemRevisionFn(ctx, ctx.Args[0])
					if err != nil {
						return err
					}
				}

				out, _, err := ctx.Client.Incidents.PostMortemWriteResetContent(cmdContext(ctx.Cmd), &flashduty.ResetPostMortemContentRequest{
					PostMortemID:     ctx.Args[0],
					Markdown:         markdown,
					ExpectedRevision: flashduty.Int64(revision),
					IdempotencyKey:   idempotencyKey,
				})
				if err != nil {
					return err
				}

				human := fmt.Sprintf(
					"Reset post-mortem content for %s: generation %d→%d, revision %d→%d",
					out.PostMortemID,
					out.PreviousGeneration,
					out.Generation,
					out.PreviousRevision,
					out.Revision,
				)
				return ctx.WriteResultJSON(out, human)
			})
		},
	}

	cmd.Flags().StringVar(&markdownFile, "markdown-file", "", "Path to Markdown content, or \"-\" to read stdin (required)")
	cmd.Flags().Int64Var(&expectedRevision, "expected-revision", 0, "Expected document revision; 0 is valid (optional: when omitted, the current revision is fetched via post-mortem info first)")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key for safe retries; max 128 Unicode characters (required)")
	_ = cmd.MarkFlagRequired("markdown-file")
	_ = cmd.MarkFlagRequired("idempotency-key")

	return cmd
}

func validatePostMortemContentResetFlags(idempotencyKey string) error {
	if idempotencyKey == "" {
		return fmt.Errorf("--idempotency-key must not be empty")
	}
	if utf8.RuneCountInString(idempotencyKey) > maxPostMortemContentIdempotencyKeyRunes {
		return fmt.Errorf("--idempotency-key must be at most %d Unicode characters", maxPostMortemContentIdempotencyKeyRunes)
	}
	return nil
}

// currentPostMortemRevisionFn resolves the current collaboration revision of a
// post-mortem report. It is a package variable so tests can stub it.
var currentPostMortemRevisionFn = fetchCurrentPostMortemRevision

// fetchCurrentPostMortemRevision GETs /incident/post-mortem/info and reads
// data.meta.revision. go-flashduty v0.5.11's typed PostMortemMeta does not
// expose the revision field yet (the server returns it), so this request goes
// out directly, reusing the SDK client's base URL and the same credential
// resolution as defaultNewClient — including broker mode, where the sentinel
// app_key is overwritten by the broker as the request egresses.
//
// TODO: switch to ctx.Client.Incidents.PostMortemInfo once go-flashduty
// exposes meta.revision on PostMortemMeta.
func fetchCurrentPostMortemRevision(ctx *RunContext, postMortemID string) (int64, error) {
	cfg, err := loadResolvedConfig()
	if err != nil {
		return 0, err
	}

	appKey := cfg.AppKey
	hc := &http.Client{Timeout: 30 * time.Second}
	if fdStr := os.Getenv("FLASHDUTY_CRED_FD"); fdStr != "" {
		fd, perr := strconv.Atoi(fdStr)
		// fds 0/1/2 are stdio; see defaultNewClient.
		if perr != nil || fd < 3 {
			return 0, fmt.Errorf("invalid FLASHDUTY_CRED_FD=%q", fdStr)
		}
		bc := newBrokerHTTPClient(fd)
		if bc == nil {
			return 0, errBrokerUnsupported
		}
		hc = bc
		appKey = "broker-sentinel"
	} else if appKey == "" {
		return 0, fmt.Errorf("no app key configured. Run 'flashduty login' or set FLASHDUTY_APP_KEY")
	}

	rel, err := url.Parse("incident/post-mortem/info")
	if err != nil {
		return 0, fmt.Errorf("failed to build post-mortem info URL: %w", err)
	}
	u := ctx.Client.BaseURL.ResolveReference(rel)
	q := u.Query()
	q.Set("post_mortem_id", postMortemID)
	q.Set("app_key", appKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(cmdContext(ctx.Cmd), http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to build post-mortem info request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := hc.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch post-mortem info for %s: %w", postMortemID, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, fmt.Errorf("failed to read post-mortem info response: %w", err)
	}

	var env struct {
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Data struct {
			Meta struct {
				Revision *int64 `json:"revision"`
			} `json:"meta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return 0, fmt.Errorf("failed to decode post-mortem info response: %w", err)
	}
	if resp.StatusCode != http.StatusOK || (env.Error != nil && env.Error.Code != "" && env.Error.Code != "OK") {
		msg := http.StatusText(resp.StatusCode)
		if env.Error != nil && env.Error.Message != "" {
			msg = env.Error.Message
		}
		return 0, fmt.Errorf("failed to fetch post-mortem info for %s: %s", postMortemID, msg)
	}
	if env.Data.Meta.Revision == nil {
		return 0, fmt.Errorf("post-mortem info for %s did not report a revision; pass --expected-revision explicitly", postMortemID)
	}
	return *env.Data.Meta.Revision, nil
}

// readPostMortemMarkdownFile loads Markdown bytes without trimming. Path "-"
// reads the injectable stdinReader so tests never touch the real stdin and
// absent flags never block on an empty pipe.
func readPostMortemMarkdownFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("--markdown-file is required")
	}

	var (
		b   []byte
		err error
	)
	if path == "-" {
		b, err = io.ReadAll(stdinReader)
		if err != nil {
			return "", fmt.Errorf("failed to read markdown from stdin: %w", err)
		}
	} else {
		b, err = os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read markdown file: %w", err)
		}
	}
	if len(b) == 0 {
		return "", fmt.Errorf("markdown content must not be empty")
	}
	return string(b), nil
}
