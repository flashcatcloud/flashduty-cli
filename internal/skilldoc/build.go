package skilldoc

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Build walks the cobra tree rooted at root and returns a structured dump of
// every runnable, non-hidden leaf command. Group containers (non-runnable
// parents like "status-page") are descended into but not emitted themselves.
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

	if c.Runnable() && !c.Hidden {
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
