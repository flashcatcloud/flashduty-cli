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

// A field cligen folded into a required positional is still a registered flag,
// but passing it as `--flag` fails the binary's Args check. The validator must
// catch this misuse (kind "positional-as-flag") — the exact error that only a
// live run surfaced before Use was threaded into the oracle. Passing the field
// positionally must stay clean, and the same flag name on a command where it is
// NOT folded (two required ids) must remain valid.
func TestValidate_FoldedPositionalAsFlag(t *testing.T) {
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
		{Path: "bad", Body: "```bash\nfduty status-page change-active-list --page-id 5\n```\n"},
		{Path: "good", Body: "```bash\nfduty status-page change-active-list 5 --type incident\n```\n"},
		{Path: "twoid", Body: "```bash\nfduty status-page change-timeline-create --page-id 5 --change-id 9\n```\n"},
	}
	byDoc := map[string][]Issue{}
	for _, is := range Validate(d, docs) {
		byDoc[is.Doc] = append(byDoc[is.Doc], is)
	}
	if n := len(byDoc["bad"]); n != 1 || byDoc["bad"][0].Kind != "positional-as-flag" {
		t.Errorf("bad: want 1 positional-as-flag, got %+v", byDoc["bad"])
	}
	if n := len(byDoc["good"]); n != 0 {
		t.Errorf("good: positional usage want 0 issues, got %+v", byDoc["good"])
	}
	if n := len(byDoc["twoid"]); n != 0 {
		t.Errorf("twoid: --page-id on non-folding command want 0 issues, got %+v", byDoc["twoid"])
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
