package cli

import (
	"cmp"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/flashcatcloud/go-flashduty"
	"github.com/spf13/cobra"
)

// artifactRefRe finds an artifact ID (art_) or presented-file ID (pf_) on its
// own or inside a console link such as /ai-sre/artifacts/art_… .
var artifactRefRe = regexp.MustCompile(`\b(art|pf)_[0-9A-Za-z]{16,}\b`)

func attachSafariArtifactDownload(root *cobra.Command) {
	g := genGroup(root, "safari", "AI SRE API")
	genAddLeaf(g, newArtifactDownloadCmd())
	genAddLeaf(g, newArtifactStreamCmd())
}

// newArtifactStreamCmd is the path-name command for GET /safari/artifact/stream:
// it takes the signed token itself. artifact-download is the one to use; this
// serves callers that already hold a download_url from artifact-sign.
func newArtifactStreamCmd() *cobra.Command {
	var token, output string

	cmd := &cobra.Command{
		Use:   "artifact-stream --t <token>",
		Short: "Download a file by signed token",
		Long: `Download a file using the token of a download_url from artifact-sign (the t
query parameter; valid for 5 minutes). To download by artifact ID, file ID or
console link, use artifact-download instead — it does the signing itself.

The file is saved like artifact-download does: under the server's file name, or
at --output (- for stdout).

API: GET /safari/artifact/stream (artifact-read-stream)`,
		Example: `  flashduty safari artifact-stream --t "$TOKEN" -o report.html`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if token == "" {
				return fmt.Errorf("--t is required")
			}
			return runCommand(cmd, args, func(ctx *RunContext) error {
				return downloadSigned(ctx, token, "", output)
			})
		},
	}
	cmd.Flags().StringVar(&token, "t", "", "Token from the download_url of artifact-sign (required)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Save to this path (- for stdout); default: the file's own name in the current directory")
	return cmd
}

func newArtifactDownloadCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "artifact-download <artifact-id | file-id | link>",
		Short: "Download an artifact or session file",
		Long: `Download a published artifact or a file a session presented, and save it.

The argument is an artifact ID (art_…), a presented-file ID (pf_… — the
file_id of a present_files result), or a console link containing either, such
as a link to an artifact page.

The file is saved under the name the server gives it in the current directory,
or at --output; --output - writes the bytes to stdout. The command prints where
it saved the file.

The transfer has no total time limit; it is cancelled only after 60s with no
bytes moved.

API: POST /safari/artifact/gallery/get, POST /safari/artifact/sign, GET /safari/artifact/stream`,
		Example: `  flashduty safari artifact-download art_VnfrWic8UbB9q3EfYR4Gmg
  flashduty safari artifact-download pf_SdhEA5fbZJGnHzwrNJMMSB --output report.html
  flashduty safari artifact-download "https://console.flashcat.cloud/ai-sre/artifacts/art_VnfrWic8UbB9q3EfYR4Gmg" -o - | head`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommand(cmd, args, func(ctx *RunContext) error {
				return runArtifactDownload(ctx, args[0], output)
			})
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Save to this path (- for stdout); default: the file's own name in the current directory")
	return cmd
}

type artifactDownloadResult struct {
	Path        string `json:"path" toon:"path"`
	Bytes       int64  `json:"bytes" toon:"bytes"`
	ContentType string `json:"content_type" toon:"content_type"`
	FileID      string `json:"file_id,omitempty" toon:"file_id,omitempty"`
}

func runArtifactDownload(ctx *RunContext, ref, output string) error {
	m := artifactRefRe.FindStringSubmatch(ref)
	if m == nil {
		return fmt.Errorf("%q has no artifact ID (art_…) or file ID (pf_…)", ref)
	}
	fileID := m[0]
	if m[1] == "art" {
		item, _, err := ctx.Client.Artifacts.ReadGet(cmdContext(ctx.Cmd), &flashduty.ArtifactIDRequest{ArtifactID: m[0]})
		if err != nil {
			return err
		}
		if item.FileID == "" {
			return fmt.Errorf("artifact %s has no file to download", m[0])
		}
		fileID = item.FileID
	}

	signed, _, err := ctx.Client.Artifacts.ReadSign(cmdContext(ctx.Cmd), &flashduty.ArtifactSignRequest{FileID: fileID})
	if err != nil {
		return err
	}
	token, err := signedToken(signed.DownloadURL)
	if err != nil {
		return err
	}
	return downloadSigned(ctx, token, fileID, output)
}

// downloadSigned streams the file behind a signed token to output (see
// artifact-download for the output rules). fileID only labels the result and
// names the file when the server gives no name.
func downloadSigned(ctx *RunContext, token, fileID, output string) error {
	tr := startTransfer(cmdContext(ctx.Cmd))
	rc, resp, err := ctx.Client.Artifacts.Download(tr.ctx, token)
	if err != nil {
		return tr.done(err)
	}
	defer func() { _ = rc.Close() }()
	body := tr.track(rc)

	if output == "-" {
		_, err = io.Copy(ctx.Writer, body)
		return tr.done(err)
	}
	if output == "" {
		output = attachmentName(resp.Header.Get("Content-Disposition"), cmp.Or(fileID, "download"))
	}
	n, err := saveFile(output, body)
	if err = tr.done(err); err != nil {
		return err
	}

	res := artifactDownloadResult{Path: output, Bytes: n, ContentType: resp.Header.Get("Content-Type"), FileID: fileID}
	if ctx.Structured() {
		return ctx.Printer.Print(res, nil)
	}
	ctx.WriteResult(fmt.Sprintf("Saved %s (%d bytes, %s)", res.Path, res.Bytes, res.ContentType))
	return nil
}

// signedToken extracts the t parameter of a sign response's download_url.
func signedToken(downloadURL string) (string, error) {
	u, err := url.Parse(downloadURL)
	if err != nil {
		return "", fmt.Errorf("unexpected download_url %q: %w", downloadURL, err)
	}
	t := u.Query().Get("t")
	if t == "" {
		return "", fmt.Errorf("download_url %q carries no token", downloadURL)
	}
	return t, nil
}

// attachmentName returns the base name of a Content-Disposition filename, or
// fallback when there is none usable. Only the base name is kept so a server
// name can never place the file outside the current directory.
func attachmentName(disposition, fallback string) string {
	_, params, err := mime.ParseMediaType(disposition)
	if err != nil {
		return fallback
	}
	name := filepath.Base(strings.ReplaceAll(params["filename"], `\`, "/"))
	if name == "." || name == "/" || name == ".." || name == "" {
		return fallback
	}
	return name
}

// saveFile writes r to path. A file is written to a temp file beside it and
// renamed into place, so a failed transfer leaves neither a partial file nor a
// damaged earlier one. A destination that exists and is not a regular file (a
// device or FIFO such as /dev/null) is written directly.
func saveFile(path string, r io.Reader) (int64, error) {
	if fi, err := os.Stat(path); err == nil && !fi.Mode().IsRegular() {
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return 0, err
		}
		n, err := io.Copy(f, r)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		return n, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".part-*")
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(tmp, r)
	if err == nil {
		err = tmp.Chmod(0o644)
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		return 0, err
	}
	return n, nil
}
