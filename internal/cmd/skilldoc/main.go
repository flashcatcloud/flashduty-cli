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
		Short: "Rewrite the generated fence in skills/flashduty/reference/<group>.md (every card if no group given)",
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
			fmt.Fprintln(cmd.OutOrStdout(), "skilldoc: cards OK")
			return nil
		},
	}
}

// dump builds the command-tree dump from the live CLI root, in-process.
func dump() skilldoc.Dump { return skilldoc.Build(cli.RootForDump()) }

// runGen rewrites the GENERATED:<group> fence inside <base>/reference/<group>.md
// with a fresh render, leaving all hand-written content outside the fence
// untouched.
func runGen(d skilldoc.Dump, base, group string) error {
	card := filepath.Join(base, "reference", group+".md")
	raw, err := os.ReadFile(card)
	if err != nil {
		return fmt.Errorf("read card: %w", err)
	}
	body := string(raw)

	start, end := skilldoc.FenceStart(group), skilldoc.FenceEnd(group)
	si := strings.Index(body, start)
	ei := strings.Index(body, end)
	if si < 0 || ei < 0 || ei < si {
		return fmt.Errorf("%s: no GENERATED:%s fence to fill (add the start/end markers first)", card, group)
	}

	fresh := skilldoc.GenerateFence(d, group)
	updated := body[:si] + fresh + body[ei+len(end):]
	if updated == body {
		return nil // already fresh
	}
	if err := os.WriteFile(card, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write card: %w", err)
	}
	return nil
}

// runGenAll regenerates the fence of every dump group that has a card file under
// <base>/reference. The group set is derived from the dump (intersected with the
// cards that actually exist), so it stays correct as domains are added or
// renamed — no hardcoded list. Groups without a card (e.g. webhook) are skipped.
func runGenAll(d skilldoc.Dump, base string) error {
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
		if _, err := os.Stat(filepath.Join(base, "reference", g+".md")); err != nil {
			continue // no card for this group
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
		fmt.Fprintf(w, "%s:%d  %s  %s\n", is.Doc, is.Line, is.Kind, is.Detail)
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
		docs = append(docs, skilldoc.Doc{Path: rel, Body: string(raw)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return docs, nil
}

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
