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
