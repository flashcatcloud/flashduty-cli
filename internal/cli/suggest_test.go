package cli

import (
	"strings"
	"testing"
)

// TestUnknownFlagListsCandidates: an unknown flag names its closest matches and
// the command's full flag list, on curated and generated commands alike.
func TestUnknownFlagListsCandidates(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		want    []string
		notWant []string
	}{
		{
			name: "near synonym on curated list",
			args: []string{"alert", "list", "--channel-id", "1"},
			want: []string{
				`unknown flag: --channel-id for "flashduty alert list"`,
				"Did you mean this?\n\t--channel\n",
				"Flags: --active, --alert-ids,",
				"Global flags: --base-url, --json, --no-trunc, --output-format",
			},
		},
		{
			name: "global --fields listed where no local one shadows it",
			args: []string{"alert", "get", "a-1", "--feilds", "x"},
			want: []string{"Global flags: --base-url, --fields, --json, --no-trunc, --output-format", "\t--fields\n"},
		},
		{
			name:    "no close match still lists the flags",
			args:    []string{"incident", "comment", "inc-1", "--content", "x"},
			want:    []string{`unknown flag: --content for "flashduty incident comment"`, "--comment-file"},
			notWant: []string{"Did you mean"},
		},
		{
			name: "generated command",
			args: []string{"member", "list", "--team-ids", "1"},
			want: []string{`unknown flag: --team-ids for "flashduty member list"`, "Flags: --asc, --data,"},
		},
		{
			name:    "hidden flags are not offered",
			args:    []string{"incident", "list", "--channel-idz", "1"},
			want:    []string{"\t--channel\n"},
			notWant: []string{"--channel-id\n", "--app-key"},
		},
		{
			name:    "a command's own --fields is offered as its flag",
			args:    []string{"incident", "list", "--field", "x"},
			want:    []string{"Did you mean this?\n\t--fields\n", "--channel, --closer-ids", "--asc, --channel", "Global flags: --base-url, --json, --no-trunc, --output-format"},
			notWant: []string{"Global flags: --base-url, --fields"},
		},
		{
			name: "root",
			args: []string{"--release-notes"},
			want: []string{`unknown flag: --release-notes for "flashduty"`, "Flags: --version"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			_, err := execCommand(tc.args...)
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error missing %q:\n%s", w, err)
				}
			}
			for _, nw := range tc.notWant {
				if strings.Contains(err.Error(), nw) {
					t.Errorf("error should not contain %q:\n%s", nw, err)
				}
			}
		})
	}
}

// TestFlagValueErrorUnchanged: only unknown-flag errors are extended; a bad
// value for a known flag keeps pflag's message.
func TestFlagValueErrorUnchanged(t *testing.T) {
	saveAndResetGlobals(t)
	_, err := execCommand("alert", "list", "--limit", "abc")
	if err == nil || !strings.HasPrefix(err.Error(), `invalid argument "abc" for "--limit" flag`) || strings.Contains(err.Error(), "Global flags") {
		t.Fatalf("want pflag's invalid-argument error unchanged, got %v", err)
	}
}

// TestUnknownCommandSuggestions: an unknown command is matched against its
// siblings, single-record verb synonyms, and then the whole tree.
func TestUnknownCommandSuggestions(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		want    []string
		notWant []string
	}{
		{
			name: "root token found deeper in the tree",
			args: []string{"silence", "--help"},
			want: []string{`unknown command "silence" for "flashduty"`, "\tflashduty channel silence-rule-list\n", "Available commands: "},
		},
		{
			name: "hyphen-joined group and verb",
			args: []string{"monit-datasource", "list"},
			want: []string{"\tflashduty monit datasource-list\n"},
		},
		{
			name:    "single-record synonym",
			args:    []string{"schedule", "get", "1"},
			want:    []string{`unknown command "get" for "flashduty schedule"`, "\tflashduty schedule info\n", "Available commands: by-person, create,"},
			notWant: []string{"silence"},
		},
		{
			name: "synonym inside a hyphenated verb",
			args: []string{"channel", "escalate-rule-get", "1"},
			want: []string{"\tflashduty channel escalate-rule-info\n"},
		},
		{
			name: "typo with --help is still an error",
			args: []string{"alert", "lsit", "--help"},
			want: []string{`unknown command "lsit" for "flashduty alert"`, "\tflashduty alert list\n"},
		},
		{
			name: "value-taking global flag before the token",
			args: []string{"--output-format", "json", "alert", "lsit"},
			want: []string{`unknown command "lsit" for "flashduty alert"`},
		},
		{
			name:    "no match lists the level's commands",
			args:    []string{"escalationz"},
			want:    []string{`unknown command "escalationz" for "flashduty"`, "Available commands: "},
			notWant: []string{"Did you mean"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			out, err := execCommand(tc.args...)
			if err == nil {
				t.Fatalf("expected an error, got output:\n%s", out)
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error missing %q:\n%s", w, err)
				}
			}
			for _, nw := range tc.notWant {
				if strings.Contains(err.Error(), nw) {
					t.Errorf("error should not contain %q:\n%s", nw, err)
				}
			}
		})
	}
}

// TestKnownInvocationsStillDispatch: the pre-dispatch check leaves real
// commands, group help, cobra's lazily-added commands and an unknown flag
// (which cobra reports) alone.
func TestKnownInvocationsStillDispatch(t *testing.T) {
	for _, args := range [][]string{
		{"alert"},
		{"alert", "--help"},
		{"--json", "alert", "--help"},
		{"help", "alert"},
		{"completion", "bash"},
		{"__complete", "alert", "l"},
		{"version"},
		{"--version"},
	} {
		saveAndResetGlobals(t)
		if out, err := execCommand(args...); err != nil {
			t.Errorf("%v: unexpected error %v (output %q)", args, err, out)
		}
	}

	saveAndResetGlobals(t)
	_, err := execCommand("alert", "--bogus", "lsit")
	if err == nil || !strings.Contains(err.Error(), "unknown flag: --bogus") {
		t.Errorf("an unknown flag before the token should be the reported error, got %v", err)
	}
}

// TestVersionFlagMatchesVersionCommand: --version prints the version
// command's line.
func TestVersionFlagMatchesVersionCommand(t *testing.T) {
	saveAndResetGlobals(t)
	viaCmd, err := execCommand("version")
	if err != nil {
		t.Fatal(err)
	}
	saveAndResetGlobals(t)
	viaFlag, err := execCommand("--version")
	if err != nil {
		t.Fatal(err)
	}
	if viaCmd != viaFlag || !strings.HasPrefix(viaFlag, "flashduty version ") {
		t.Fatalf("--version %q != version %q", viaFlag, viaCmd)
	}
}
