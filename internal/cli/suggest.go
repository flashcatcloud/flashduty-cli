package cli

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Unknown-flag and unknown-command errors list the valid candidates, so a
// caller who guessed a name (a human from memory, an agent or script by
// analogy with a sibling verb) corrects it from the error alone instead of a
// separate --help round trip.

// maxTreeSuggestions caps the whole-tree command matches listed in one error.
const maxTreeSuggestions = 8

// singleRecordVerbs are the interchangeable spellings of "show one record"
// across command groups: curated verbs say get/detail, path-derived generated
// verbs say info/infos. A guessed spelling is swapped for the others when
// looking for a sibling.
var singleRecordVerbs = []string{"get", "info", "infos", "detail", "show", "describe"}

// flagErrorWithCandidates is the root FlagErrorFunc (cobra hands it to every
// descendant). It extends pflag's unknown-flag error with the closest flag
// names and the command's full flag list; every other flag error (bad value,
// missing value) passes through unchanged.
func flagErrorWithCandidates(cmd *cobra.Command, err error) error {
	var notExist *pflag.NotExistError
	if !errors.As(err, &notExist) {
		return err
	}
	// A flag is global when it is the root's persistent flag itself; a
	// command's own flag of the same name (e.g. a --fields with its own
	// meaning) shadows it and is listed as the command's.
	rootFlags := cmd.Root().PersistentFlags()
	var local, global []string
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if !f.Hidden && f.Name != "help" && rootFlags.Lookup(f.Name) != f {
			local = append(local, f.Name)
		}
	})
	rootFlags.VisitAll(func(f *pflag.Flag) {
		if !f.Hidden && cmd.Flags().Lookup(f.Name) == f {
			global = append(global, f.Name)
		}
	})
	sort.Strings(local)
	sort.Strings(global)

	var b strings.Builder
	fmt.Fprintf(&b, "%s for %q\n", err, cmd.CommandPath())
	if close := closeNames(notExist.GetSpecifiedName(), append(append([]string{}, local...), global...)); len(close) > 0 {
		fmt.Fprintf(&b, "\nDid you mean this?\n\t--%s\n", strings.Join(close, "\n\t--"))
	}
	if len(local) > 0 {
		fmt.Fprintf(&b, "Flags: --%s\n", strings.Join(local, ", --"))
	}
	fmt.Fprintf(&b, "Global flags: --%s", strings.Join(global, ", --"))
	return errors.New(b.String())
}

// closeNames returns the candidates that look like a misspelling or a
// near-synonym of name: within edit distance 2 (1 for names under 6
// characters, where 2 edits reach unrelated short flags), a prefix either way,
// or the same first hyphen-token (--channel-id vs --channel).
func closeNames(name string, candidates []string) []string {
	if len(name) < 2 {
		return nil
	}
	maxDist := 2
	if len(name) < 6 {
		maxDist = 1
	}
	seen := map[string]bool{}
	var out []string
	first := strings.SplitN(name, "-", 2)[0]
	for _, c := range candidates {
		if seen[c] || c == name {
			continue
		}
		if levenshtein(name, c) <= maxDist || strings.HasPrefix(c, name) || strings.HasPrefix(name, c) ||
			strings.SplitN(c, "-", 2)[0] == first {
			seen[c] = true
			out = append(out, c)
		}
	}
	sort.Strings(out)
	return out
}

// unknownCommandError reports token as an unknown subcommand of the group cmd.
// Candidates come from, in order: cobra's sibling suggestions plus
// single-record verb synonyms; failing those, every leaf in the whole tree
// whose path contains token on hyphen-token boundaries (so `silence` finds
// `channel silence-rule-list` and `monit-datasource` finds
// `monit datasource-list`). The group's own subcommands are always listed.
func unknownCommandError(cmd *cobra.Command, token string) error {
	// Mirrors cobra's own default (see (*Command).findSuggestions): a group
	// never sets this itself, so the zero value would otherwise disable the
	// Levenshtein half of SuggestionsFor.
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}
	var paths []string
	for _, name := range append(cmd.SuggestionsFor(token), synonymSiblings(cmd, token)...) {
		paths = appendUnique(paths, cmd.CommandPath()+" "+name)
	}
	more := 0
	if len(paths) == 0 {
		paths, more = treeMatches(cmd.Root(), token)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "unknown command %q for %q", token, cmd.CommandPath())
	if len(paths) > 0 {
		b.WriteString("\n\nDid you mean this?\n")
		for _, p := range paths {
			fmt.Fprintf(&b, "\t%s\n", p)
		}
		if more > 0 {
			fmt.Fprintf(&b, "\t(+%d more)\n", more)
		}
	} else {
		b.WriteString("\n")
	}
	var names []string
	for _, c := range cmd.Commands() {
		if c.IsAvailableCommand() {
			names = append(names, c.Name())
		}
	}
	fmt.Fprintf(&b, "Available commands: %s", strings.Join(names, ", "))
	return errors.New(b.String())
}

// synonymSiblings swaps a single-record verb token in token (get, info,
// detail, …) for each other spelling and returns the swaps that name an
// existing subcommand of cmd: `schedule get` → info,
// `channel escalate-rule-get` → escalate-rule-info.
func synonymSiblings(cmd *cobra.Command, token string) []string {
	parts := strings.Split(token, "-")
	var out []string
	for i, p := range parts {
		if !slices.Contains(singleRecordVerbs, p) {
			continue
		}
		for _, alt := range singleRecordVerbs {
			if alt == p {
				continue
			}
			swapped := append(append(append([]string{}, parts[:i]...), alt), parts[i+1:]...)
			name := strings.Join(swapped, "-")
			for _, c := range cmd.Commands() {
				if c.IsAvailableCommand() && (c.Name() == name || c.HasAlias(name)) {
					out = appendUnique(out, c.Name())
				}
			}
		}
	}
	return out
}

// treeMatches returns the runnable leaves under root whose hyphen-joined path
// contains token as a whole run of hyphen-tokens, shortest path first, capped
// at maxTreeSuggestions; more is the number left out.
func treeMatches(root *cobra.Command, token string) (paths []string, more int) {
	needle := "-" + strings.ToLower(token) + "-"
	var walk func(c *cobra.Command, rel []string)
	walk = func(c *cobra.Command, rel []string) {
		for _, child := range c.Commands() {
			if !child.IsAvailableCommand() {
				continue
			}
			p := append(append([]string{}, rel...), child.Name())
			if child.HasSubCommands() {
				walk(child, p)
				continue
			}
			if strings.Contains("-"+strings.Join(p, "-")+"-", needle) {
				paths = append(paths, root.CommandPath()+" "+strings.Join(p, " "))
			}
		}
	}
	walk(root, nil)
	sort.SliceStable(paths, func(i, j int) bool {
		if len(paths[i]) != len(paths[j]) {
			return len(paths[i]) < len(paths[j])
		}
		return paths[i] < paths[j]
	})
	if len(paths) > maxTreeSuggestions {
		more = len(paths) - maxTreeSuggestions
		paths = paths[:maxTreeSuggestions]
	}
	return paths, more
}

// rejectUnknownSubcommand fails when args resolve to a command group (root
// included) followed by a positional token that names none of its
// subcommands. It runs before cobra dispatch because cobra answers --help
// before any Args validator: `fduty alert lsit --help` would otherwise print
// the alert help and exit 0, and the root would get cobra's own
// sibling-only suggestion. A token after an unknown flag is left to cobra so
// the flag error is the one reported.
func rejectUnknownSubcommand(root *cobra.Command, args []string) error {
	// cobra adds these lazily inside Execute; add them now so they resolve.
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd(args...)
	if len(args) > 0 && (args[0] == cobra.ShellCompRequestCmd || args[0] == cobra.ShellCompNoDescRequestCmd) {
		return nil
	}
	// Find's own error is cobra's root-level unknown-command check; the
	// resolved command and leftover args are still returned with it.
	cmd, rest, _ := root.Find(args)
	if cmd == nil || !cmd.HasSubCommands() {
		return nil
	}
	token, ok := firstPositional(cmd, rest)
	if !ok {
		return nil
	}
	return unknownCommandError(cmd, token)
}

// firstPositional returns the first non-flag token of args as parsed by cmd's
// flags (a non-bool flag consumes the next token as its value). ok is false
// when there is none, or when an unknown flag comes first.
func firstPositional(cmd *cobra.Command, args []string) (string, bool) {
	lookup := func(name string, short bool) *pflag.Flag {
		for _, fs := range []*pflag.FlagSet{cmd.Flags(), cmd.InheritedFlags()} {
			if short {
				if f := fs.ShorthandLookup(name); f != nil {
					return f
				}
			} else if f := fs.Lookup(name); f != nil {
				return f
			}
		}
		if !short && (name == "help" || (name == "version" && cmd.Root().Version != "")) {
			return &pflag.Flag{Name: name, NoOptDefVal: "true"}
		}
		if short && name == "h" {
			return &pflag.Flag{Name: "help", NoOptDefVal: "true"}
		}
		return nil
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			return "", false
		case strings.HasPrefix(a, "-") && len(a) > 1:
			if strings.Contains(a, "=") {
				continue
			}
			name, short := strings.TrimPrefix(a, "--"), !strings.HasPrefix(a, "--")
			if short {
				name = a[len(a)-1:]
			}
			f := lookup(name, short)
			if f == nil {
				return "", false
			}
			if f.NoOptDefVal == "" {
				i++
			}
		default:
			return a, true
		}
	}
	return "", false
}

func levenshtein(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

func appendUnique(list []string, s string) []string {
	if slices.Contains(list, s) {
		return list
	}
	return append(list, s)
}
