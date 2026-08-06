// Command skilldoc is the dev tool for the flashduty skill cards. It builds the
// command-tree dump in-process (via cli.RootForDump) and either rewrites a
// card's generated fence (`skilldoc gen <group>`) or validates every card under
// skills/flashduty against the dump (`skilldoc check`): unknown commands/flags
// in examples and out-of-date generated fences. Run from the repo root.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/flashcatcloud/flashduty-cli/internal/cli"
	"github.com/flashcatcloud/flashduty-cli/internal/skilldoc"
)

// skillDir is the card root relative to the repo root.
const skillDir = "skills/flashduty"

func main() {
	root := &cobra.Command{
		Use:           "skilldoc",
		Short:         "Generate and validate flashduty skill command-cards",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(genCmd(), checkCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "skilldoc:", err)
		os.Exit(1)
	}
}

func genCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gen [group]",
		Short: "Rewrite every GENERATED:<group> fence across the skills/flashduty cards (every group if none given)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			base, err := cardBase()
			if err != nil {
				return err
			}
			d := dump()
			if len(args) == 1 {
				return runGen(d, base, args[0])
			}
			return runGenAll(d, base)
		},
	}
}

func checkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Validate every card under skills/flashduty against the command oracle",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			base, err := cardBase()
			if err != nil {
				return err
			}
			n, err := runCheck(dump(), base, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			if n > 0 {
				return fmt.Errorf("%d card issue(s) found", n)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "skilldoc: cards OK")
			return err
		},
	}
}

// dump builds the command-tree dump from the live CLI root, in-process.
func dump() skilldoc.Dump { return skilldoc.Build(cli.RootForDump()) }

// runGen regenerates every GENERATED fence of group across the cards under
// <base>, leaving hand-written content outside the fences untouched. A group
// may split its fences across cards (subset fences claiming verb prefixes,
// plus the catch-all for the rest — see skilldoc.RenderGroupFences), so the
// fresh render is computed for the group as a whole, then spliced per card.
func runGen(d skilldoc.Dump, base, group string) error {
	docs, err := loadDocs(base)
	if err != nil {
		return err
	}

	var ids []string
	perDoc := map[string][]string{}
	var docOrder []string
	for _, doc := range docs {
		for _, fl := range skilldoc.FenceLocs(doc.Body) {
			spec, err := skilldoc.ParseFenceID(fl.ID)
			if err != nil {
				return fmt.Errorf("%s: %w", doc.Path, err)
			}
			if spec.Group != group {
				continue
			}
			if len(perDoc[doc.Path]) == 0 {
				docOrder = append(docOrder, doc.Path)
			}
			perDoc[doc.Path] = append(perDoc[doc.Path], fl.ID)
			ids = append(ids, fl.ID)
		}
	}
	if len(ids) == 0 {
		return fmt.Errorf("no GENERATED:%s fence found under %s (add the start/end markers first)", group, base)
	}

	rendered, violations := skilldoc.RenderGroupFences(d, group, ids)
	if len(violations) > 0 {
		return fmt.Errorf("group %s fence topology: %s", group, strings.Join(violations, "; "))
	}

	byPath := map[string]skilldoc.Doc{}
	for _, doc := range docs {
		byPath[doc.Path] = doc
	}
	for _, p := range docOrder {
		doc := byPath[p]
		body := doc.Body
		for _, id := range perDoc[p] {
			start, end := skilldoc.FenceStart(id), skilldoc.FenceEnd(id)
			si := strings.Index(body, start)
			ei := strings.Index(body[si:], end)
			if si < 0 || ei < 0 {
				return fmt.Errorf("%s: unterminated GENERATED:%s fence", p, id)
			}
			body = body[:si] + rendered[id] + body[si+ei+len(end):]
		}
		if body == doc.Body {
			continue // already fresh
		}
		if err := os.WriteFile(filepath.Join(base, p), []byte(body), 0o644); err != nil {
			return fmt.Errorf("write card: %w", err)
		}
	}
	return nil
}

// runGenAll regenerates the fences of every dump group that has at least one
// GENERATED marker in a card under <base>. The group set is derived from the
// dump (intersected with the fences that actually exist), so it stays correct
// as domains are added or renamed — no hardcoded list. Groups without any
// fence (e.g. webhook) are skipped.
func runGenAll(d skilldoc.Dump, base string) error {
	docs, err := loadDocs(base)
	if err != nil {
		return err
	}
	withFence := map[string]bool{}
	for _, doc := range docs {
		for _, fl := range skilldoc.FenceLocs(doc.Body) {
			if spec, err := skilldoc.ParseFenceID(fl.ID); err == nil {
				withFence[spec.Group] = true
			}
		}
	}

	seen := map[string]bool{}
	var groups []string
	for _, c := range d.Commands {
		if c.Group != "" && !seen[c.Group] {
			seen[c.Group] = true
			groups = append(groups, c.Group)
		}
	}
	sort.Strings(groups)
	for _, g := range groups {
		if !withFence[g] {
			continue
		}
		if err := runGen(d, base, g); err != nil {
			return fmt.Errorf("gen %s: %w", g, err)
		}
	}
	return nil
}

// runCheck loads every *.md under base, validates examples and fence freshness
// against the dump, prints each issue as "relpath:line  kind  detail", and
// returns the issue count. A missing base directory is not an error (no cards →
// no issues).
func runCheck(d skilldoc.Dump, base string, w io.Writer) (int, error) {
	docs, err := loadDocs(base)
	if err != nil {
		return 0, err
	}

	issues := append(skilldoc.Validate(d, docs), skilldoc.CheckFences(d, docs)...)
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Doc != issues[j].Doc {
			return issues[i].Doc < issues[j].Doc
		}
		if issues[i].Line != issues[j].Line {
			return issues[i].Line < issues[j].Line
		}
		return issues[i].Kind < issues[j].Kind
	})
	for _, is := range issues {
		if _, err := fmt.Fprintf(w, "%s:%d  %s  %s\n", is.Doc, is.Line, is.Kind, is.Detail); err != nil {
			return 0, err
		}
	}
	return len(issues), nil
}

// loadDocs reads every *.md file under base (recursively) into a Doc with its
// path relative to base. A non-existent base yields no docs.
func loadDocs(base string) ([]skilldoc.Doc, error) {
	info, err := os.Stat(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", base)
	}

	var docs []skilldoc.Doc
	err = filepath.WalkDir(base, func(path string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			rel = path
		}
		docs = append(docs, skilldoc.Doc{Path: rel, Body: normalizeEOL(string(raw))})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return docs, nil
}

// normalizeEOL collapses Windows CRLF to LF so the byte-exact fence comparison
// and the line-based harvester are insensitive to how git checked the cards out
// (Windows autocrlf would otherwise make every fence look stale). The generated
// fence is always LF, so LF is the canonical form to compare against.
func normalizeEOL(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

// cardBase resolves <repoRoot>/skills/flashduty by walking up from the cwd to
// the directory containing go.mod.
func cardBase() (string, error) {
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, skillDir), nil
}

// repoRoot walks up from the working directory until it finds go.mod.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s upward", dir)
		}
		dir = parent
	}
}
