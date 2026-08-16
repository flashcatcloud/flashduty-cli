package skilldoc

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Build walks the cobra tree rooted at root and returns a structured dump of
// every runnable, non-hidden leaf command. Group containers (parents with
// their own subcommands, like "status-page") are descended into but not
// emitted themselves — see the predicate comment in walk for why this is
// keyed off subcommands rather than Runnable().
//
// Path is the space-joined chain of cobra command names below the root, using
// c.Name() so a positional placeholder in Use (e.g. "change-create <page-id>")
// is stripped to the bare verb. Required flags are detected via cobra's
// one-required-flag annotation. Enums and nested --data fields are NOT
// re-derived here; they live verbatim in Long, which cligen authored.
func Build(root *cobra.Command) Dump {
	var d Dump
	walk(root, nil, &d)
	sort.Slice(d.Commands, func(i, j int) bool {
		return d.Commands[i].Path < d.Commands[j].Path
	})
	return d
}

func walk(c *cobra.Command, parents []string, d *Dump) {
	// The root itself contributes no path segment; its children start the path.
	var path []string
	if len(parents) > 0 || c.Parent() != nil {
		path = append(append([]string{}, parents...), c.Name())
	}

	// A card entry is only correct for a leaf an agent can actually invoke to
	// get work done. c.Runnable() alone is NOT that signal: every command
	// group in this tree (alert, incident, oncall schedule, incident
	// war-room, the generated genGroup groups, ...) is built through
	// newGroupCmd (internal/cli/command.go), which gives it a real RunE
	// (print help) and Args validator (groupUnknownSubcommand) so a typo'd
	// subcommand fails loudly instead of cobra silently discarding it — see
	// that constructor's doc comment. That makes every group Runnable too,
	// even though its RunE does nothing but dispatch to children. The
	// distinguishing fact is not "does it run" but "does it hold
	// subcommands": every node in this tree with children is a pure
	// container (verified when this predicate was fixed — no command mixes
	// its own business logic with child commands), so HasSubCommands() is
	// the correct group/leaf split. If that ever stops being true — a
	// command gains both real behavior of its own AND subcommands — this
	// predicate must change to emit a card for that command's own behavior
	// while still not treating its children as absent.
	//
	// Deprecated commands are excluded: cards guide NEW usage, and a
	// deprecated verb's replacement is what the card prose should teach.
	// The command stays in the CLI tree (with its runtime deprecation
	// warning) for the migration period — it just earns no card entry.
	if !c.HasSubCommands() && c.Runnable() && !c.Hidden && c.Deprecated == "" {
		d.Commands = append(d.Commands, command(c, path))
	}

	for _, child := range c.Commands() {
		walk(child, path, d)
	}
}

func command(c *cobra.Command, path []string) Command {
	cmd := Command{
		Path:    strings.Join(path, " "),
		Short:   c.Short,
		Use:     c.Use,
		Long:    c.Long,
		Example: c.Example,
	}
	if len(path) > 0 {
		cmd.Group = path[0]
	}
	c.Flags().VisitAll(func(f *pflag.Flag) {
		cmd.Flags = append(cmd.Flags, Flag{
			Name:     f.Name,
			Type:     f.Value.Type(),
			Default:  f.DefValue,
			Usage:    f.Usage,
			Required: f.Annotations[cobra.BashCompOneRequiredFlag] != nil,
		})
	})
	return cmd
}
