package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestRenameCLI(t *testing.T) {
	cases := []struct{ in, want string }{
		{"flashduty incident list", "fduty incident list"},
		{"run 'flashduty login' first", "run 'fduty login' first"},
		{"Run `flashduty incident timeline <id>`", "Run `fduty incident timeline <id>`"},
		{"  flashduty team list\n  flashduty team get 1", "  fduty team list\n  fduty team get 1"},
		{"cat rows.csv | flashduty enrichment upload", "cat rows.csv | fduty enrichment upload"},
		{"flashduty is managed by flashduty-runner", "fduty is managed by flashduty-runner"},
		// Not the command word: product name, paths, domains, identifiers.
		{"Flashduty CLI", "Flashduty CLI"},
		{"go-flashduty client", "go-flashduty client"},
		{"~/.flashduty config", "~/.flashduty config"},
		{"skills/flashduty cards", "skills/flashduty cards"},
		{"flashduty.example.com", "flashduty.example.com"},
		{"flashduty", "flashduty"},
	}
	for _, c := range cases {
		if got := renameCLI(c.in, "fduty"); got != c.want {
			t.Errorf("renameCLI(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// "$" in the name is literal text, not a regexp group reference.
	if got, want := renameCLI("run flashduty login", "fd$1"), "run fd$1 login"; got != want {
		t.Errorf("renameCLI with $ in name = %q, want %q", got, want)
	}
}

func TestRenameCommandText(t *testing.T) {
	root := &cobra.Command{Use: "flashduty", Long: "Run 'flashduty login'."}
	child := &cobra.Command{Use: "list", Short: "List", Example: "  flashduty team list"}
	child.Flags().String("id", "", "from 'flashduty team list'")
	root.PersistentFlags().String("x", "", "see 'flashduty login'")
	root.AddCommand(child)

	renameCommandText(root, "fduty")

	for _, c := range []struct{ got, want string }{
		{root.Long, "Run 'fduty login'."},
		{child.Example, "  fduty team list"},
		{child.Flags().Lookup("id").Usage, "from 'fduty team list'"},
		{root.PersistentFlags().Lookup("x").Usage, "see 'fduty login'"},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}

// TestRenameCLIMatchesRegexOnCommandTree pins the shortcut in renameCLI: for
// every help string of the real command tree, skipping text without
// cliWordLiteral must give exactly what the regex alone gives. It fails if
// cliWord is ever changed to match text that lacks the literal.
func TestRenameCLIMatchesRegexOnCommandTree(t *testing.T) {
	regexOnly := func(s, name string) string {
		return cliWord.ReplaceAllString(s, "${1}"+strings.ReplaceAll(name, "$", "$$")+" ")
	}
	var texts []string
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		texts = append(texts, c.Short, c.Long, c.Example)
		add := func(f *pflag.Flag) { texts = append(texts, f.Usage) }
		c.Flags().VisitAll(add)
		c.PersistentFlags().VisitAll(add)
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(rootCmd)
	if len(texts) < 1000 {
		t.Fatalf("walked only %d help strings; the command tree is not populated", len(texts))
	}
	changed := 0
	for _, s := range texts {
		want := regexOnly(s, "fduty")
		if got := renameCLI(s, "fduty"); got != want {
			t.Fatalf("renameCLI differs from the regex on %q:\ngot  %q\nwant %q", s, got, want)
		}
		if want != s {
			changed++
		}
	}
	if changed == 0 {
		t.Fatal("no help string mentions the CLI; the comparison proves nothing")
	}
}
