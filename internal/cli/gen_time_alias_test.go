package cli

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// newTimeAliasFlagCmd builds a command carrying the flag pair cligen emits for
// a relative-time --start-time: the canonical flag plus its --since alias.
// Values are read back through the flag set (rawOf) so the test drives the same
// Changed()/value surface the generated RunE does.
func newTimeAliasFlagCmd(t *testing.T, sets map[string]string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "x"}
	var canonical, alias string
	cmd.Flags().StringVar(&canonical, "start-time", "", "")
	cmd.Flags().StringVar(&alias, "since", "", "")
	for name, raw := range sets {
		if err := cmd.Flags().Set(name, raw); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
	return cmd
}

func rawOf(cmd *cobra.Command, name string) string {
	return cmd.Flags().Lookup(name).Value.String()
}

func TestGenParseTimeFlagAlias(t *testing.T) {
	// Neither spelling set → ok=false so the caller omits the field.
	cmd := newTimeAliasFlagCmd(t, nil)
	if v, ok, err := genParseTimeFlagAlias(cmd, "start-time", "since", rawOf(cmd, "start-time"), rawOf(cmd, "since")); err != nil || ok || v != 0 {
		t.Errorf("unset: got (%d,%v,%v), want (0,false,nil)", v, ok, err)
	}

	// Only the alias set → parsed with the same duration forms.
	cmd = newTimeAliasFlagCmd(t, map[string]string{"since": "24h"})
	v, ok, err := genParseTimeFlagAlias(cmd, "start-time", "since", rawOf(cmd, "start-time"), rawOf(cmd, "since"))
	if err != nil || !ok {
		t.Fatalf("alias 24h: ok=%v err=%v", ok, err)
	}
	if delta := time.Now().Unix() - 24*3600 - v; delta < -5 || delta > 5 {
		t.Errorf("alias 24h: parsed %d not ~24h ago (delta %ds)", v, delta)
	}

	// Only the canonical flag set → unchanged behavior.
	cmd = newTimeAliasFlagCmd(t, map[string]string{"start-time": "1700000000"})
	if v, ok, err := genParseTimeFlagAlias(cmd, "start-time", "since", rawOf(cmd, "start-time"), rawOf(cmd, "since")); err != nil || !ok || v != 1700000000 {
		t.Errorf("canonical only: got (%d,%v,%v), want (1700000000,true,nil)", v, ok, err)
	}

	// Both spellings set to the SAME raw value → accepted silently.
	cmd = newTimeAliasFlagCmd(t, map[string]string{"start-time": "7d", "since": "7d"})
	if _, ok, err := genParseTimeFlagAlias(cmd, "start-time", "since", rawOf(cmd, "start-time"), rawOf(cmd, "since")); err != nil || !ok {
		t.Errorf("same value: got ok=%v err=%v, want (true,nil)", ok, err)
	}

	// Both spellings set to DIFFERENT values → loud conflict error.
	cmd = newTimeAliasFlagCmd(t, map[string]string{"start-time": "30d", "since": "7d"})
	_, _, err = genParseTimeFlagAlias(cmd, "start-time", "since", rawOf(cmd, "start-time"), rawOf(cmd, "since"))
	if err == nil || !strings.Contains(err.Error(), "--since and --start-time disagree") {
		t.Errorf("conflict: expected disagree error, got %v", err)
	}

	// An invalid ALIAS value names the alias spelling in the error.
	cmd = newTimeAliasFlagCmd(t, map[string]string{"since": "not-a-time"})
	if _, _, err := genParseTimeFlagAlias(cmd, "start-time", "since", rawOf(cmd, "start-time"), rawOf(cmd, "since")); err == nil || !strings.Contains(err.Error(), "invalid --since") {
		t.Errorf("invalid alias: expected 'invalid --since' error, got %v", err)
	}
}

// TestCommandGeneratedTimeAliasEquivalence proves --since/--until produce the
// identical wire body as --start-time/--end-time on a generated command.
func TestCommandGeneratedTimeAliasEquivalence(t *testing.T) {
	bodyFor := func(args ...string) map[string]any {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		stub.data = map[string]any{"items": []any{}, "total": 0}
		full := append([]string{"insight", "account"}, args...)
		if _, err := execCommand(full...); err != nil {
			t.Fatalf("[time-alias] insight account %v: %v", args, err)
		}
		if stub.lastPath != "/insight/account" {
			t.Fatalf("[time-alias] expected /insight/account, got %q", stub.lastPath)
		}
		return stub.lastBody
	}

	canonical := bodyFor("--start-time", "1700000000", "--end-time", "1700086400")
	aliased := bodyFor("--since", "1700000000", "--until", "1700086400")
	if !reflect.DeepEqual(canonical, aliased) {
		t.Fatalf("[time-alias] wire bodies differ:\ncanonical: %#v\naliased:   %#v", canonical, aliased)
	}
	if canonical["start_time"] != float64(1700000000) || canonical["end_time"] != float64(1700086400) {
		t.Fatalf("[time-alias] unexpected body: %#v", canonical)
	}
}

// TestCommandGeneratedTimeAliasDurationForms proves the alias spellings accept
// the duration/now forms, parsed to unix seconds on the wire.
func TestCommandGeneratedTimeAliasDurationForms(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{}, "total": 0}

	if _, err := execCommand("insight", "account", "--since", "7d", "--until", "now"); err != nil {
		t.Fatalf("[time-alias-duration] unexpected error: %v", err)
	}
	start, okStart := stub.lastBody["start_time"].(float64)
	end, okEnd := stub.lastBody["end_time"].(float64)
	if !okStart || !okEnd {
		t.Fatalf("[time-alias-duration] body missing parsed times: %#v", stub.lastBody)
	}
	now := float64(time.Now().Unix())
	if d := now - 7*24*3600 - start; d < -10 || d > 10 {
		t.Errorf("[time-alias-duration] start_time %v not ~7d ago (delta %vs)", start, d)
	}
	if d := now - end; d < -10 || d > 10 {
		t.Errorf("[time-alias-duration] end_time %v not ~now (delta %vs)", end, d)
	}
}

// TestCommandGeneratedTimeAliasConflict proves disagreeing spellings fail
// loudly before any request is made, while agreeing spellings are accepted.
func TestCommandGeneratedTimeAliasConflict(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{}, "total": 0}

	_, err := execCommand("insight", "account", "--since", "7d", "--start-time", "30d", "--until", "now")
	if err == nil || !strings.Contains(err.Error(), "--since and --start-time disagree") {
		t.Fatalf("[time-alias-conflict] expected disagree error, got %v", err)
	}
	if stub.requests != 0 {
		t.Fatalf("[time-alias-conflict] conflict must fail before any request, got %d", stub.requests)
	}

	// Same value via both spellings is accepted and reaches the wire once.
	_, err = execCommand("insight", "account", "--since", "1700000000", "--start-time", "1700000000", "--until", "1700086400")
	if err != nil {
		t.Fatalf("[time-alias-same] unexpected error: %v", err)
	}
	if stub.lastBody["start_time"] != float64(1700000000) {
		t.Fatalf("[time-alias-same] unexpected body: %#v", stub.lastBody)
	}
}

// TestCommandGeneratedTimeAliasHelp proves --help teaches both spellings: the
// canonical flag carries the alias note and the alias flag renders its own row.
func TestCommandGeneratedTimeAliasHelp(t *testing.T) {
	saveAndResetGlobals(t)

	out, err := execCommand("insight", "account", "--help")
	if err != nil {
		t.Fatalf("[time-alias-help] unexpected error: %v", err)
	}
	for _, want := range []string{"(alias: --since)", "(alias: --until)", "--since string", "--until string"} {
		if !strings.Contains(out, want) {
			t.Errorf("[time-alias-help] help output missing %q", want)
		}
	}
}

// TestGeneratedTimeAliasTreeConsistency walks the full command tree and asserts
// the alias invariant the generator emits: every relative-time (string-typed)
// --start-time/--end-time flag is paired with a --since/--until flag whose
// usage teaches the pairing. Millisecond-window int flags are deliberately out
// of scope (no duration parsing exists to alias identically).
func TestGeneratedTimeAliasTreeConsistency(t *testing.T) {
	pairs := [][2]string{{"start-time", "since"}, {"end-time", "until"}}
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, pair := range pairs {
			canonical, alias := pair[0], pair[1]
			f := cmd.Flags().Lookup(canonical)
			if f == nil || f.Value.Type() != "string" {
				continue
			}
			if cmd.Flags().Lookup(alias) == nil {
				t.Errorf("%s registers --%s but not --%s", cmd.CommandPath(), canonical, alias)
			}
			if !strings.Contains(f.Usage, "(alias: --"+alias+")") {
				t.Errorf("%s --%s usage lacks the alias note: %q", cmd.CommandPath(), canonical, f.Usage)
			}
		}
		for _, sub := range cmd.Commands() {
			walk(sub)
		}
	}
	walk(rootCmd)
}
