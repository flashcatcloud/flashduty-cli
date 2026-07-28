package skilldoc

import "testing"

// validatorDump is a minimal dump fixture: one status-page leaf with flags
// {type, title} plus the data flag.
func validatorDump() Dump {
	return Dump{Commands: []Command{
		{
			Path:  "status-page change-create",
			Group: "status-page",
			Flags: []Flag{
				{Name: "type"}, {Name: "title"}, {Name: "data"},
			},
		},
	}}
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
}
