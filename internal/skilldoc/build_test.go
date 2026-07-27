package skilldoc

import (
	"testing"

	"github.com/spf13/cobra"
)

func testTree() *cobra.Command {
	root := &cobra.Command{Use: "fduty"}
	sp := &cobra.Command{Use: "status-page"}
	create := &cobra.Command{Use: "change-create <page-id>", Short: "Create status page event",
		Long: "Create status page event.\n\nRequest fields:\n  --type string (required) — Event type. [incident, maintenance]\n",
		Run:  func(*cobra.Command, []string) {}}
	create.Flags().String("type", "", "Event type.")
	_ = create.MarkFlagRequired("type")
	create.Flags().String("title", "", "Event title.")
	sp.AddCommand(create)
	root.AddCommand(sp)
	return root
}

func TestBuild_CapturesLeafWithFlagsAndRequired(t *testing.T) {
	d := Build(testTree())
	var got *Command
	for i := range d.Commands {
		if d.Commands[i].Path == "status-page change-create" {
			got = &d.Commands[i]
		}
	}
	if got == nil {
		t.Fatalf("missing status-page change-create; got %+v", d.Commands)
	}
	if got.Group != "status-page" {
		t.Errorf("group = %q", got.Group)
	}
	// Use must be captured verbatim — it carries the positional placeholder that
	// Path strips, and is the only runtime signal of cligen's positional fold.
	if got.Use != "change-create <page-id>" {
		t.Errorf("Use = %q, want %q", got.Use, "change-create <page-id>")
	}
	var typeFlag *Flag
	for i := range got.Flags {
		if got.Flags[i].Name == "type" {
			typeFlag = &got.Flags[i]
		}
	}
	if typeFlag == nil || !typeFlag.Required {
		t.Errorf("--type should be present and required: %+v", got.Flags)
	}
}

// runnableGroupTree mirrors internal/cli.newGroupCmd: a container command that
// is Runnable (RunE just prints help, same as every group in the real tree —
// alert, incident, oncall schedule, ...) purely so a mistyped subcommand fails
// loudly instead of cobra discarding the leftover arg silently. Runnable()
// alone therefore cannot distinguish a group from a leaf; a command with its
// own subcommands must be excluded regardless of whether it happens to be
// Runnable.
func runnableGroupTree() *cobra.Command {
	root := &cobra.Command{Use: "fduty"}
	incident := &cobra.Command{
		Use: "incident", Short: "Manage incidents",
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	list := &cobra.Command{Use: "list", Short: "List incidents", Run: func(*cobra.Command, []string) {}}
	incident.AddCommand(list)
	root.AddCommand(incident)
	return root
}

func TestBuild_ExcludesRunnableGroupWithSubcommands(t *testing.T) {
	d := Build(runnableGroupTree())

	for _, c := range d.Commands {
		if c.Path == "incident" {
			t.Fatalf("group %q has subcommands and no behavior of its own beyond dispatching to them; it must not get a card entry (it would falsely document it as an invocable command): %+v", c.Path, c)
		}
	}

	var gotLeaf bool
	for _, c := range d.Commands {
		if c.Path == "incident list" {
			gotLeaf = true
		}
	}
	if !gotLeaf {
		t.Fatalf("leaf %q under a runnable group must still get a card entry; got %+v", "incident list", d.Commands)
	}
}
