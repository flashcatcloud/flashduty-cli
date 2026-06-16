package cli

import (
	"fmt"
	"strings"
	"testing"
)

// TestGenPositionalUseLine asserts the generated --help Use line carries the
// positional placeholder: an *_ids array op renders the singular id followed by
// the variadic marker; an *_id scalar op renders the single id; the override and
// int cases render their pinned field.
func TestGenPositionalUseLine(t *testing.T) {
	// (a) and (b) call the generated constructors directly so the Use-line
	// assertions stay independent of registration order. `incident ack` now
	// surfaces the generated twin (curated shadow dropped); `incident info` was
	// always generated-only (the curated leaf is named `detail`).
	ack := genIncidentsAckCmd()
	if got := ack.Use; got != "ack <incident-id> [<id2>...]" {
		t.Errorf("ack twin Use = %q, want %q", got, "ack <incident-id> [<id2>...]")
	}
	if ack.Args == nil {
		t.Errorf("ack twin has no Args validator")
	}
	if err := ack.Args(ack, nil); err == nil {
		t.Errorf("ack twin Args accepted zero args (want >=1)")
	}
	if err := ack.Args(ack, []string{"id1"}); err != nil {
		t.Errorf("ack twin Args rejected one arg: %v", err)
	}

	// `incident info` pins incident_id as an OPTIONAL positional: the backend
	// relaxed incident_id (a lookup may instead pass the 6-char num via --num), so
	// the positional is 0-or-1. `info <id>` stays for back-compat; `info` alone
	// (with --num) is valid; `info id1 id2` is still rejected.
	info := genIncidentsInfoCmd()
	if got := info.Use; got != "info [<incident-id>]" {
		t.Errorf("info twin Use = %q, want %q", got, "info [<incident-id>]")
	}
	if info.Args == nil {
		t.Errorf("info twin has no Args validator")
	}
	if err := info.Args(info, nil); err != nil {
		t.Errorf("info twin Args rejected zero args (want 0-or-1; --num path): %v", err)
	}
	if err := info.Args(info, []string{"id1"}); err != nil {
		t.Errorf("info twin Args rejected one arg: %v", err)
	}
	if err := info.Args(info, []string{"id1", "id2"}); err == nil {
		t.Errorf("info twin Args accepted two args (want at most one)")
	}

	// Override cases: merge pins target_incident_id (NOT source_incident_ids);
	// war-room detail pins chat_id.
	merge := genIncidentsMergeCmd()
	if got, want := merge.Use, "merge <target-incident-id>"; got != want {
		t.Errorf("merge override Use = %q, want %q", got, want)
	}
	if strings.Contains(merge.Use, "source-incident") {
		t.Errorf("merge override Use leaked source_incident_ids: %q", merge.Use)
	}
	detail := genIncidentsWarRoomDetailCmd()
	if got, want := detail.Use, "war-room-detail <chat-id>"; got != want {
		t.Errorf("war-room detail override Use = %q, want %q", got, want)
	}
}

// TestGenPositionalScalarStringRuntime invokes a GENERATED-ONLY string-scalar
// command (`field info <field-id>`) and asserts the positional folds into the
// request body under the wire key.
func TestGenPositionalScalarStringRuntime(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("field", "info", "fld-123"); err != nil {
		t.Fatalf("execCommand field info: %v", err)
	}
	if stub.lastPath != "/field/info" {
		t.Fatalf("path = %q, want /field/info", stub.lastPath)
	}
	if got := stub.lastBody["field_id"]; got != "fld-123" {
		t.Errorf("field_id = %#v, want fld-123", got)
	}
}

// TestGenPositionalSliceRuntime invokes a GENERATED-ONLY string-slice command
// (`alert list-by-ids <alert-id> [<id2>...]`, whose alert_ids field is []string)
// and asserts every positional arg folds into the wire array verbatim.
func TestGenPositionalSliceRuntime(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("alert", "list-by-ids", "a1", "a2", "a3"); err != nil {
		t.Fatalf("execCommand alert list-by-ids: %v", err)
	}
	if stub.lastPath != "/alert/list-by-ids" {
		t.Fatalf("path = %q, want /alert/list-by-ids", stub.lastPath)
	}
	if got, want := fmt.Sprint(stub.bodyStrings("alert_ids")), "[a1 a2 a3]"; got != want {
		t.Errorf("alert_ids = %q, want %q", got, want)
	}
}

// TestGenPositionalIntSliceRuntime invokes a GENERATED-ONLY int-slice command
// (`team infos <team-id> [<id2>...]`, whose team_ids field is []uint64) and
// asserts each positional arg is PARSED to an int in the wire array. A raw
// []string fold (the wrong kind) would fail SDK binding, so this guards the
// intslice path specifically.
func TestGenPositionalIntSliceRuntime(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("team", "infos", "11", "22", "33"); err != nil {
		t.Fatalf("execCommand team infos: %v", err)
	}
	if stub.lastPath != "/team/infos" {
		t.Fatalf("path = %q, want /team/infos", stub.lastPath)
	}
	raw, ok := stub.lastBody["team_ids"].([]any)
	if !ok || len(raw) != 3 {
		t.Fatalf("team_ids = %#v, want a 3-element array", stub.lastBody["team_ids"])
	}
	// JSON numbers decode to float64 through the stub.
	for i, want := range []float64{11, 22, 33} {
		if got, _ := raw[i].(float64); got != want {
			t.Errorf("team_ids[%d] = %#v, want %v", i, raw[i], want)
		}
	}
}

// TestGenPositionalIntRuntime invokes a GENERATED-ONLY int-scalar command
// (`schedule info <schedule-id>`) and asserts the positional parses to an int
// in the wire body (schedule_id is Int64Var, so genFoldPositional uses ParseInt).
func TestGenPositionalIntRuntime(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	// schedule info also needs --start/--end (relative-time required flags); supply
	// them so the command reaches the wire. The positional is the assertion target.
	if _, err := execCommand("schedule", "info", "4242", "--start", "now", "--end", "now"); err != nil {
		t.Fatalf("execCommand schedule info: %v", err)
	}
	if stub.lastPath != "/schedule/info" {
		t.Fatalf("path = %q, want /schedule/info", stub.lastPath)
	}
	// JSON numbers decode to float64 through the stub.
	if got, _ := stub.lastBody["schedule_id"].(float64); got != 4242 {
		t.Errorf("schedule_id = %#v, want 4242", stub.lastBody["schedule_id"])
	}
}

// TestGenPositionalFlagOverridesPositional asserts the overlay order: an
// explicitly-set typed flag for the same field wins over the positional arg
// (positional folds first, the changed flag stamps after).
func TestGenPositionalFlagOverridesPositional(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("field", "info", "fromArg", "--field-id", "fromFlag"); err != nil {
		t.Fatalf("execCommand field info with flag: %v", err)
	}
	if got := stub.lastBody["field_id"]; got != "fromFlag" {
		t.Errorf("field_id = %#v, want fromFlag (explicit flag must override positional)", got)
	}
}

// TestGenFoldPositional unit-tests the runtime helper across all three kinds.
func TestGenFoldPositional(t *testing.T) {
	// string scalar → args[0]
	b := map[string]any{}
	if err := genFoldPositional([]string{"abc"}, b, "x_id", "string"); err != nil {
		t.Fatalf("string: %v", err)
	}
	if b["x_id"] != "abc" {
		t.Errorf("string: x_id = %#v, want abc", b["x_id"])
	}

	// string slice → all args
	b = map[string]any{}
	if err := genFoldPositional([]string{"a", "b"}, b, "x_ids", "slice"); err != nil {
		t.Fatalf("slice: %v", err)
	}
	if got, want := fmt.Sprint(b["x_ids"]), "[a b]"; got != want {
		t.Errorf("slice: x_ids = %q, want %q", got, want)
	}

	// int slice → each arg parsed to int64
	b = map[string]any{}
	if err := genFoldPositional([]string{"1", "2"}, b, "x_ids", "intslice"); err != nil {
		t.Fatalf("intslice: %v", err)
	}
	if got, want := fmt.Sprint(b["x_ids"]), "[1 2]"; got != want {
		t.Errorf("intslice: x_ids = %q, want %q", got, want)
	}
	if _, ok := b["x_ids"].([]int64); !ok {
		t.Errorf("intslice: x_ids type = %T, want []int64", b["x_ids"])
	}

	// int slice with non-numeric arg → clean error
	b = map[string]any{}
	if err := genFoldPositional([]string{"1", "x"}, b, "x_ids", "intslice"); err == nil {
		t.Errorf("intslice: expected error on non-numeric arg, got nil")
	}

	// int scalar → ParseInt
	b = map[string]any{}
	if err := genFoldPositional([]string{"77"}, b, "x_id", "int"); err != nil {
		t.Fatalf("int: %v", err)
	}
	if b["x_id"] != int64(77) {
		t.Errorf("int: x_id = %#v, want int64(77)", b["x_id"])
	}

	// int scalar with non-numeric arg → clean error
	b = map[string]any{}
	if err := genFoldPositional([]string{"nope"}, b, "x_id", "int"); err == nil {
		t.Errorf("int: expected error on non-numeric arg, got nil")
	}
}
