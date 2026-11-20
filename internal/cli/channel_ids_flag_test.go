package cli

import (
	"fmt"
	"strings"
	"testing"
)

// TestChannelIDsFlagSpellings: every channel-filtering curated command takes
// --channel-ids, and each deprecated spelling it still accepts sends the same
// channel_ids, prints a notice naming --channel-ids, and stays out of --help.
func TestChannelIDsFlagSpellings(t *testing.T) {
	cases := []struct {
		cmd        []string
		path       string
		deprecated []string
	}{
		{[]string{"incident", "list"}, "/incident/list", []string{"channel", "channel-id"}},
		{[]string{"alert", "list"}, "/alert/list", []string{"channel"}},
		{[]string{"alert-event", "list"}, "/alert-event/list", []string{"channel"}},
		{[]string{"change", "list"}, "/change/list", []string{"channel"}},
		{[]string{"insight", "top-alerts", "--label", "check"}, "/insight/alert/topk-by-label", nil},
	}
	for _, tc := range cases {
		name := strings.Join(tc.cmd[:2], " ")
		for _, spelling := range append([]string{"channel-ids"}, tc.deprecated...) {
			t.Run(name+" --"+spelling, func(t *testing.T) {
				saveAndResetGlobals(t)
				stub := newGFStub(t)

				args := append(append([]string{}, tc.cmd...), "--"+spelling, "100,200")
				out, err := execCommand(args...)
				if err != nil {
					t.Fatalf("%v: %v", args, err)
				}
				if stub.lastPath != tc.path {
					t.Fatalf("path = %q, want %q", stub.lastPath, tc.path)
				}
				if got := fmt.Sprint(stub.lastBody["channel_ids"]); got != "[100 200]" {
					t.Fatalf("channel_ids = %s, want [100 200]", got)
				}
				notice := "Flag --" + spelling + " has been deprecated, use --channel-ids instead"
				if deprecated := spelling != "channel-ids"; strings.Contains(out, notice) != deprecated {
					t.Fatalf("output = %q; deprecation notice expected: %v", out, deprecated)
				}
			})
		}

		t.Run(name+" help", func(t *testing.T) {
			saveAndResetGlobals(t)
			out, err := execCommand(append(append([]string{}, tc.cmd[:2]...), "--help")...)
			if err != nil {
				t.Fatalf("--help: %v", err)
			}
			if !strings.Contains(out, "--channel-ids") {
				t.Fatalf("help lacks --channel-ids:\n%s", out)
			}
			for _, spelling := range tc.deprecated {
				if strings.Contains(out, "--"+spelling+" ") {
					t.Fatalf("help shows deprecated --%s:\n%s", spelling, out)
				}
			}
		})
	}
}

// TestChannelIDsFlagInputForms: the value takes the forms the string
// --channel flag took (spaces, empty elements, an empty value as no filter,
// base-10 IDs) and accumulates repeated uses.
func TestChannelIDsFlagInputForms(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--channel-ids", "100, 200,"}, "[100 200]"},
		{[]string{"--channel-ids", "100", "--channel-ids", "200"}, "[100 200]"},
		{[]string{"--channel-ids", "010"}, "[10]"},
		{[]string{"--channel", ""}, "<nil>"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			if _, err := execCommand(append([]string{"change", "list"}, tc.args...)...); err != nil {
				t.Fatalf("change list %v: %v", tc.args, err)
			}
			if got := fmt.Sprint(stub.lastBody["channel_ids"]); got != tc.want {
				t.Fatalf("channel_ids = %s, want %s", got, tc.want)
			}
		})
	}
}
