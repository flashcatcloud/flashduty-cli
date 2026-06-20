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

func TestValidate_GlobalFlagsAllowed(t *testing.T) {
	d := validatorDump()
	docs := []Doc{
		{Path: "G", Body: "```bash\nfduty status-page change-create --type incident --title x --output-format toon\n```\n"},
	}
	if issues := Validate(d, docs); len(issues) != 0 {
		t.Errorf("global flag --output-format should be allowed: %+v", issues)
	}
}
