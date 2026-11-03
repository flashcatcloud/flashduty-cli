package skilldoc

import "testing"

// validatorDump is a minimal dump fixture: one status-page leaf with flags
// {type, title} plus the data flag, under a root with the global
// --output-format and --help and the root-only --version.
func validatorDump() Dump {
	return Dump{
		Commands: []Command{
			{
				Path:  "status-page change-create",
				Group: "status-page",
				Flags: []Flag{
					{Name: "type"}, {Name: "title"}, {Name: "data"},
				},
			},
		},
		Root: []Flag{
			{Name: "output-format", Persistent: true},
			{Name: "json", Persistent: true},
			{Name: "help", Persistent: true},
			{Name: "version"},
		},
	}
}

func TestValidate_UnknownCommandAndFlag(t *testing.T) {
	d := validatorDump()
	docs := []Doc{
		{Path: "A", Body: "```bash\nfduty status-page change-create --type incident --title x\n```\n"},
		{Path: "B", Body: "```bash\nfduty status-page change-create --bogus 1\n```\n"},
		{Path: "C", Body: "```bash\nfduty status-page bogus-verb\n```\n"},
		{Path: "D", Body: "```bash\nfduty status-page change-create --title <ph>\n```\n"},
	}
	issues := Validate(d, docs)

	byDoc := map[string][]Issue{}
	for _, is := range issues {
		byDoc[is.Doc] = append(byDoc[is.Doc], is)
	}

	if n := len(byDoc["A"]); n != 0 {
		t.Errorf("doc A: want 0 issues, got %d: %+v", n, byDoc["A"])
	}
	if n := len(byDoc["B"]); n != 1 || byDoc["B"][0].Kind != "unknown-flag" {
		t.Errorf("doc B: want 1 unknown-flag, got %+v", byDoc["B"])
	}
	if n := len(byDoc["C"]); n != 1 || byDoc["C"][0].Kind != "unknown-command" {
		t.Errorf("doc C: want 1 unknown-command, got %+v", byDoc["C"])
	}
	if n := len(byDoc["D"]); n != 0 {
		t.Errorf("doc D: want 0 issues (placeholder value, known flag), got %+v", byDoc["D"])
	}
}

// A field cligen folds into a required positional keeps a same-named flag
// registered as a genuine alternative source: every generated command with
// such a fold uses requireBodyFieldOrExactArg/requireBodyFieldOrArgs
// (internal/cli/args.go), which explicitly accepts the flag alone. So passing
// that flag — with or without the positional also present — must validate
// clean; only a flag that is not registered on the command at all is an
// actual defect.
func TestValidate_FoldedFlagIsValidAlternative(t *testing.T) {
	d := Dump{Commands: []Command{
		{ // single required id → cligen folds page-id into a positional
			Path:  "status-page change-active-list",
			Group: "status-page",
			Use:   "change-active-list <page-id>",
			Flags: []Flag{{Name: "page-id"}, {Name: "type"}, {Name: "data"}},
		},
		{ // two required ids → no fold; page-id stays a real flag
			Path:  "status-page change-timeline-create",
			Group: "status-page",
			Use:   "change-timeline-create",
			Flags: []Flag{{Name: "page-id"}, {Name: "change-id"}, {Name: "data"}},
		},
	}}
	docs := []Doc{
		{Path: "flag-alone", Body: "```bash\nfduty status-page change-active-list --page-id 5 --type incident\n```\n"},
		{Path: "positional", Body: "```bash\nfduty status-page change-active-list 5 --type incident\n```\n"},
		{Path: "twoid", Body: "```bash\nfduty status-page change-timeline-create --page-id 5 --change-id 9\n```\n"},
		{Path: "unknown-on-folder", Body: "```bash\nfduty status-page change-active-list --page-id 5 --bogus x\n```\n"},
	}
	byDoc := map[string][]Issue{}
	for _, is := range Validate(d, docs) {
		byDoc[is.Doc] = append(byDoc[is.Doc], is)
	}
	if n := len(byDoc["flag-alone"]); n != 0 {
		t.Errorf("flag-alone: folded flag used without the positional want 0 issues, got %+v", byDoc["flag-alone"])
	}
	if n := len(byDoc["positional"]); n != 0 {
		t.Errorf("positional: positional usage want 0 issues, got %+v", byDoc["positional"])
	}
	if n := len(byDoc["twoid"]); n != 0 {
		t.Errorf("twoid: --page-id on non-folding command want 0 issues, got %+v", byDoc["twoid"])
	}
	if n := len(byDoc["unknown-on-folder"]); n != 1 || byDoc["unknown-on-folder"][0].Kind != "unknown-flag" {
		t.Errorf("unknown-on-folder: want 1 unknown-flag, got %+v", byDoc["unknown-on-folder"])
	}
}

func TestValidate_GlobalFlagsAllowed(t *testing.T) {
	d := validatorDump()
	docs := []Doc{
		{Path: "G", Body: "```bash\nfduty status-page change-create --type incident --title x --output-format toon\n```\n"},
	}
	if issues := Validate(d, docs); len(issues) != 0 {
		t.Errorf("global flag --output-format should be allowed: %+v", issues)
	}
}

// A bare `fduty` that carries flags is a root invocation, not prose: its flags
// are checked against the flags the root accepts. Root-only flags (--version)
// are valid there but not on a subcommand.
func TestValidate_RootInvocationFlags(t *testing.T) {
	d := validatorDump()
	docs := []Doc{
		{Path: "version", Body: "Check `fduty --version` first.\n"},
		{Path: "help", Body: "Run `fduty --help`.\n"},
		{Path: "bogus", Body: "Run `fduty --release-notes`.\n"},
		{Path: "leaf-version", Body: "```bash\nfduty status-page change-create --version\n```\n"},
	}
	byDoc := map[string][]Issue{}
	for _, is := range Validate(d, docs) {
		byDoc[is.Doc] = append(byDoc[is.Doc], is)
	}
	for _, ok := range []string{"version", "help"} {
		if n := len(byDoc[ok]); n != 0 {
			t.Errorf("%s: want 0 issues, got %+v", ok, byDoc[ok])
		}
	}
	for _, bad := range []string{"bogus", "leaf-version"} {
		if n := len(byDoc[bad]); n != 1 || byDoc[bad][0].Kind != "unknown-flag" {
			t.Errorf("%s: want 1 unknown-flag, got %+v", bad, byDoc[bad])
		}
	}
}

// Prose mentions of the binary — a bare `fduty` word or a templated
// `fduty <group> <verb>` — are documentation references, not runnable examples,
// and must not be flagged. A genuine wrong command name (no placeholder) must
// STILL be caught, since catching command-name drift is the validator's job.
func TestValidate_SkipsBareAndTemplatedMentions(t *testing.T) {
	d := validatorDump()
	docs := []Doc{
		{Path: "bare", Body: "The `fduty` CLI is the interface. Each `fduty` subprocess gets auth.\n"},
		{Path: "tmpl", Body: "Derive it then run `fduty <group> <verb> --help`.\n"},
		{Path: "drift", Body: "```bash\nfduty statuspage list\n```\n"},
		{Path: "root-bogus", Body: "Run `fduty --bogus` first.\n"},
		{Path: "root-json", Body: "Run `fduty --json`.\n"},
	}
	byDoc := map[string][]Issue{}
	for _, is := range Validate(d, docs) {
		byDoc[is.Doc] = append(byDoc[is.Doc], is)
	}
	if n := len(byDoc["bare"]); n != 0 {
		t.Errorf("bare `fduty` prose mention: want 0 issues, got %+v", byDoc["bare"])
	}
	if n := len(byDoc["tmpl"]); n != 0 {
		t.Errorf("templated `fduty <group> <verb>`: want 0 issues, got %+v", byDoc["tmpl"])
	}
	if n := len(byDoc["drift"]); n != 1 || byDoc["drift"][0].Kind != "unknown-command" {
		t.Errorf("drift `statuspage`: want 1 unknown-command, got %+v", byDoc["drift"])
	}
	// A bare `fduty` followed by flags is a root invocation, not prose.
	if n := len(byDoc["root-bogus"]); n != 1 || byDoc["root-bogus"][0].Kind != "unknown-flag" {
		t.Errorf("`fduty --bogus`: want 1 unknown-flag, got %+v", byDoc["root-bogus"])
	}
	if n := len(byDoc["root-json"]); n != 0 {
		t.Errorf("`fduty --json`: want 0 issues, got %+v", byDoc["root-json"])
	}
}

// verbTableCard wraps rows in the shape every command card shares: an intent
// table, then the GENERATED fence that names the card's command group.
func verbTableCard(fenceID string, rows ...string) string {
	body := "## Intent → verb\n\n| want | verb |\n|---|---|\n"
	for _, r := range rows {
		body += "| x | " + r + " |\n"
	}
	return body + "\n" + FenceStart(fenceID) + "\n" + FenceEnd(fenceID) + "\n"
}

func verbTableDump() Dump {
	return Dump{
		Commands: []Command{
			{Path: "rum field-list", Group: "rum", Flags: []Flag{{Name: "is-facet"}}},
			{Path: "rum issue-info", Group: "rum", Flags: []Flag{{Name: "issue-id"}}},
			{Path: "incident war-room list", Group: "incident"},
			{Path: "safari automation-rule-run", Group: "safari"},
		},
		Root: []Flag{{Name: "json", Persistent: true}},
	}
}

// A verb cell names a command relative to the card's group, so it never reaches
// the `fduty …` harvester: the validator must resolve it from the table itself.
func TestValidate_VerbTableReportsUnknownVerb(t *testing.T) {
	body := "# card\n\n" + verbTableCard("rum", "`field-list`", "`facet-list`")
	issues := Validate(verbTableDump(), []Doc{{Path: "rum.md", Body: body}})
	if len(issues) != 1 {
		t.Fatalf("want 1 issue, got %+v", issues)
	}
	got := issues[0]
	if got.Kind != "unknown-command" || got.Detail != "rum facet-list" || got.Line != 8 {
		t.Errorf("want unknown-command `rum facet-list` at line 8, got %+v", got)
	}
}

func TestValidate_VerbTableCellShapes(t *testing.T) {
	cases := []struct {
		name string
		cell string
		kind string // empty → the cell must validate clean
	}{
		{"bare verb", "`field-list`", ""},
		{"verb with positional", "`issue-info <issue-id>`", ""},
		{"verb with flag", "`field-list --is-facet true`", ""},
		{"verb with global flag", "`field-list --json`", ""},
		{"verb then response field", "`field-list` (save the returned `issue_id`)", ""},
		{"verb then field path", "`field-list` → `.days`", ""},
		{"flag in its own span", "`field-list` with `--is-facet`", ""},
		{"stale flag in its own span", "`field-list` with `--no-such-flag`", "unknown-flag"},
		{"stale flag in the verb span", "`field-list --no-such-flag`", "unknown-flag"},
		{"second verb stale", "`field-list` / `facet-list`", "unknown-command"},
		{"verb in trailing prose stale", "`field-list` (`facet-list` is equivalent)", "unknown-command"},
		{"stale verb with positional", "`facet-list <app-id>`", "unknown-command"},
		{"verb from another group", "`safari automation-rule-run <rule-id>`", ""},
		{"stale verb in another group", "`safari automation-rule-kick <rule-id>`", "unknown-command"},
		{"pipe into jq", "`field-list --json \\| jq '.items[]' --not-a-cli-flag`", ""},
		{"binary-prefixed invocation", "`flashduty rum field-list`", ""},
		{"wildcard family", "`field-*`", ""},
		{"card reference", "`reference/noise.md`", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issues := Validate(verbTableDump(), []Doc{{Path: "c.md", Body: verbTableCard("rum", tc.cell)}})
			if tc.kind == "" {
				if len(issues) != 0 {
					t.Fatalf("want clean, got %+v", issues)
				}
				return
			}
			if len(issues) != 1 || issues[0].Kind != tc.kind {
				t.Fatalf("want 1 %s, got %+v", tc.kind, issues)
			}
		})
	}
}

// A card documenting part of a group declares that group through a subset
// fence, so the group need not match the card's file name.
func TestValidate_VerbTableGroupComesFromFence(t *testing.T) {
	body := verbTableCard("incident[war-room]", "`war-room list <id>`", "`war-room lst <id>`")
	issues := Validate(verbTableDump(), []Doc{{Path: "war-room.md", Body: body}})
	if len(issues) != 1 || issues[0].Detail != "incident war-room lst" {
		t.Fatalf("want 1 issue on `incident war-room lst`, got %+v", issues)
	}
}

// Without a fence no group can be derived, and skipping would quietly switch
// the check off for that card.
func TestValidate_VerbTableWithoutFenceIsReported(t *testing.T) {
	body := "| want | verb |\n|---|---|\n| x | `field-list` |\n"
	issues := Validate(verbTableDump(), []Doc{{Path: "c.md", Body: body}})
	if len(issues) != 1 || issues[0].Kind != "unknown-command" || issues[0].Line != 3 {
		t.Fatalf("want 1 unknown-command at line 3, got %+v", issues)
	}
}

// Only `verb` columns carry verbs: other tables hold flags and field names, and
// a table inside a code fence is sample output.
func TestValidate_VerbTableIgnoresOtherTables(t *testing.T) {
	fence := FenceStart("rum") + "\n" + FenceEnd("rum") + "\n"
	body := "| field | meaning |\n|---|---|\n| `facet-list` | x |\n\n" +
		"```\n| want | verb |\n|---|---|\n| x | `facet-list` |\n```\n\n" + fence
	if issues := Validate(verbTableDump(), []Doc{{Path: "c.md", Body: body}}); len(issues) != 0 {
		t.Fatalf("want clean, got %+v", issues)
	}
}

// Prose after a verb table closes it, so a later line is not read as a row.
func TestValidate_VerbTableEndsAtFirstNonRow(t *testing.T) {
	body := verbTableCard("rum", "`field-list`") + "\n`facet-list` in prose.\n\n| other | col |\n|---|---|\n| `facet-list` | x |\n"
	if issues := Validate(verbTableDump(), []Doc{{Path: "c.md", Body: body}}); len(issues) != 0 {
		t.Fatalf("want clean, got %+v", issues)
	}
}
