package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Curated chat commands for the AI SRE agent, attached to the generated
// `safari` group: the message, subscribe and attachment endpoints are not part
// of the generated surface.

const (
	safariAppName       = "ai-sre"
	safariEntryKind     = "api"
	safariSurface       = "cli"
	safariReplyOnlyMode = "reply_only"
	// maxMessageRunes is the server limit on a message's text.
	maxMessageRunes = 32000
	// maxAttachmentBytes is the server limit on one attachment. File types are
	// not checked here: unknown extensions are accepted by the server only when
	// the content is UTF-8 text, and its error is passed through.
	maxAttachmentBytes = 50 << 20
)

func attachSafariChat(root *cobra.Command) {
	g := genGroup(root, "safari", "AI SRE API")
	genAddLeaf(g, newSafariSendCmd())
	genAddLeaf(g, newSafariWatchCmd())
}

// readSendMessage resolves the message from the positional argument,
// --message-file (a path, or - for stdin) or piped stdin, accepting exactly one
// source.
func readSendMessage(args []string, messageFile string) (string, error) {
	sources := 0
	if len(args) > 0 {
		sources++
	}
	if messageFile != "" {
		sources++
	}
	if sources > 1 {
		return "", fmt.Errorf("give the message as an argument or --message-file, not both")
	}
	switch {
	case len(args) > 0:
		return args[0], nil
	case messageFile == "-":
		return readAllStdin("--message-file")
	case messageFile != "":
		b, err := os.ReadFile(messageFile)
		if err != nil {
			return "", fmt.Errorf("cannot read --message-file: %w", err)
		}
		return string(b), nil
	case !term.IsTerminal(int(os.Stdin.Fd())):
		return readAllStdin("stdin")
	}
	return "", nil
}

func readAllStdin(flag string) (string, error) {
	b, err := readStdin(flag)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func newSafariSendCmd() *cobra.Command {
	var (
		sessionID, messageFile, envKind, envID string
		files                                  []string
		teamID                                 int64
	)

	cmd := &cobra.Command{
		Use:   "send [message]",
		Short: "Send a message to an AI SRE session",
		Long: `Send one message to an AI SRE session and print the acknowledgement as a JSON
object: {"session_id","invocation_id","root_invocation_id","delivery_mode"}.

The message is the positional argument, --message-file (a path, or - for
stdin), or piped stdin. Without --session-id a new session is started;
--team-id binds it to a team (0 = account scope; the caller must belong to the
team) and --env-kind/--env-id pick the execution environment. Each --file is
uploaded as an attachment to the session and attached to the message.

The message is sent on the CLI channel (reply-only delivery): the agent answers
through its reply tool and the answers are read with 'safari watch'. If the
server does not accept that channel, the command fails instead of falling back.

Follow the answer with:
  flashduty safari watch --session-id <session_id> --root-invocation-id <root_invocation_id>

API: POST /safari/session/message, POST /safari/attachment/upload`,
		Example: `  flashduty safari send "why is checkout latency up?"
  flashduty safari send --session-id sess_01 --file trace.json "does this match?"
  cat question.md | flashduty safari send --team-id 42 --env-kind byoc`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				if envID != "" && envKind == "" {
					return fmt.Errorf("--env-id requires --env-kind")
				}
				text, err := readSendMessage(args, messageFile)
				if err != nil {
					return err
				}
				text = strings.TrimSpace(text)
				if text == "" && len(files) == 0 {
					return fmt.Errorf("a message (argument, --message-file or stdin) or at least one --file is required")
				}
				if n := utf8.RuneCountInString(text); n > maxMessageRunes {
					return fmt.Errorf("message is %d characters; the limit is %d (attach longer content with --file)", n, maxMessageRunes)
				}

				cctx := cmdContext(ctx.Cmd)
				req := &flashduty.SessionMessageRequest{
					SessionID:       sessionID,
					Query:           text,
					Surface:         safariSurface,
					DeliveryMode:    safariReplyOnlyMode,
					TeamID:          teamID,
					EnvironmentKind: envKind,
					EnvironmentID:   envID,
				}

				for _, path := range files {
					// Fail before a session is created for a file that cannot be read.
					if path == "-" {
						continue
					}
					info, err := os.Stat(path)
					if err != nil {
						return fmt.Errorf("--file %s: %w", path, err)
					}
					if info.Size() > maxAttachmentBytes {
						return fmt.Errorf("--file %s is %d MB; the attachment limit is %d MB", path, info.Size()>>20, maxAttachmentBytes>>20)
					}
				}

				if len(files) > 0 {
					// Attachments are stored under an existing session, so a new
					// conversation is created before its files go up.
					if req.SessionID == "" {
						created, _, err := ctx.Client.Sessions.WriteCreate(cctx, &flashduty.SessionCreateRequest{
							AppName: safariAppName, EntryKind: safariEntryKind, TeamID: teamID,
						})
						if err != nil {
							return err
						}
						req.SessionID = created.SessionID
					}
					for _, path := range files {
						part, err := uploadSessionFile(ctx, req.SessionID, path)
						if err != nil {
							return fmt.Errorf("--file %s: %w", path, err)
						}
						req.Parts = append(req.Parts, part)
					}
				}
				if len(req.Parts) > 0 && text != "" {
					// The server ignores query when parts are present, so the
					// text rides as the leading part (as in the web composer).
					req.Parts = append([]flashduty.SessionMessagePart{{Type: "text", Text: text}}, req.Parts...)
					req.Query = ""
				}
				if req.SessionID == "" {
					req.AppName = safariAppName
					req.EntryKind = safariEntryKind
				}

				ack, _, err := ctx.Client.Sessions.WriteMessage(cctx, req)
				if err != nil {
					return err
				}
				if ack.DeliveryMode != safariReplyOnlyMode {
					return fmt.Errorf("the server did not accept the CLI reply channel (delivery_mode=%q, want %q); the message was queued as session_id=%s invocation_id=%s but its answers will not reach 'safari watch'",
						ack.DeliveryMode, safariReplyOnlyMode, ack.SessionID, ack.InvocationID)
				}
				return json.NewEncoder(ctx.Writer).Encode(ack)
			})
		},
	}
	cmd.Flags().StringVar(&sessionID, "session-id", "", "Continue this session (default: start a new one)")
	cmd.Flags().StringVar(&messageFile, "message-file", "", "Read the message from this file, or - for stdin")
	cmd.Flags().StringArrayVar(&files, "file", nil, "Attach a file (repeatable); uploaded as a session attachment")
	cmd.Flags().Int64Var(&teamID, "team-id", 0, "Team scope for a new session (0 = account scope)")
	cmd.Flags().StringVar(&envKind, "env-kind", "", "Execution environment kind: cloud or byoc")
	registerEnumFlag(cmd, "env-kind", "cloud", "byoc")
	cmd.Flags().StringVar(&envID, "env-id", "", "Specific environment of --env-kind")
	return cmd
}

// uploadSessionFile uploads one --file and returns its message part.
func uploadSessionFile(ctx *RunContext, sessionID, path string) (flashduty.SessionMessagePart, error) {
	f, name, err := openUploadFile(path, "stdin")
	if err != nil {
		return flashduty.SessionMessagePart{}, err
	}
	defer func() { _ = f.Close() }()
	att, _, err := ctx.Client.Sessions.UploadAttachment(cmdContext(ctx.Cmd), &flashduty.SessionAttachmentUploadRequest{
		SessionID: sessionID, Filename: name, File: f,
	})
	if err != nil {
		return flashduty.SessionMessagePart{}, err
	}
	return att.FilePart(), nil
}
