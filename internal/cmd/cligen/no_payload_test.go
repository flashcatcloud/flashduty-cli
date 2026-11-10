package main

import (
	"strings"
	"testing"
)

// loadRealServices builds the exact service set run() generates from: the spec
// shipped in the linked go-flashduty module, read against the compiled SDK.
func loadRealServices(t *testing.T) ([]service, map[string]map[string]methodInfo) {
	t.Helper()
	path, err := specPath()
	if err != nil {
		t.Fatalf("locate spec: %v", err)
	}
	spec, err := loadSpec(path)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	sdk := reflectSDK()
	return collectServices(asMap(spec["paths"]), asMap(asMap(spec["components"])["schemas"]), sdk), sdk
}

// TestNoPayloadOpsDeclareNoPayload pins the premise the acknowledgement rests
// on. An operation whose SDK method returns no typed data (methodInfo.HasData
// false) is routed to printRawOrNoPayload, which tells the caller the API returned
// nothing. That is only honest while the spec agrees, and today it does: every
// such operation either declares its 200 `data` as an empty object (the
// mutations) or declares no application/json 200 body at all (the CSV/file
// downloads, whose bytes arrive on Response.Raw instead). Either way RespTree,
// the generator's own read of the declared `data` fields, is empty.
//
// If an endpoint later grows a real payload, this fails rather than letting the
// CLI keep reporting "no payload" over a response it silently dropped — the SDK
// method must gain a data return first, which moves the command to
// printGenericResult on its own.
//
// What it does not prove: a spec that grew real fields while still declaring
// `data` as a free-form object would leave RespTree empty and slip past. The
// SDK method signature is the backstop there — a method with no data return has
// nowhere to decode such a payload into, so the CLI could not have printed it
// either way, and the fix is to declare the fields in the spec.
//
// Operations the SDK map does not answer for are skipped, which loses nothing:
// emitService skips those too (with a warning) and emits no command at all, and
// coverage_test in internal/cli fails when any spec operation has no command.
func TestNoPayloadOpsDeclareNoPayload(t *testing.T) {
	services, sdk := loadRealServices(t)

	checked := 0
	for _, s := range services {
		for _, o := range s.Ops {
			mi, ok := sdk[s.Name][o.Method]
			if !ok || mi.HasData {
				continue
			}
			checked++
			if len(o.RespTree) > 0 {
				names := make([]string, 0, len(o.RespTree))
				for _, f := range o.RespTree {
					names = append(names, f.Wire)
				}
				t.Errorf("%s %s (%s) declares a 200 payload (%s) but %s.%s returns no data, so its command would acknowledge a response it discards",
					o.HTTP, o.Path, o.OpID, strings.Join(names, ","), s.Name, o.Method)
			}
		}
	}
	if checked == 0 {
		t.Fatal("found no operation without a data return — the lookup is broken, not the spec clean")
	}
	t.Logf("checked %d operations whose command acknowledges an empty response", checked)
}

// TestEmittedNoDataCommandDelegatesToTheSharedHelper guards the generator side
// of the same contract: the emitted body must hand its response to the one
// helper that decides between streaming a download and reporting no payload,
// and must carry no acknowledgement string of its own. The committed generated
// files are checked separately (TestNoCommandEchoesItsRequestPath, in
// internal/cli), which would not catch a generator edited without regenerating.
func TestEmittedNoDataCommandDelegatesToTheSharedHelper(t *testing.T) {
	services, sdk := loadRealServices(t)

	emitted := 0
	for _, s := range services {
		for _, o := range s.Ops {
			mi, ok := sdk[s.Name][o.Method]
			if !ok || mi.HasData {
				continue
			}
			emitted++
			src := emitCmd("genTestCmd", s, o, mi)
			if !strings.Contains(src, "return printRawOrNoPayload(ctx, resp)") {
				t.Fatalf("%s %s: emitted command does not render its response through printRawOrNoPayload:\n%s", o.HTTP, o.Path, src)
			}
			if strings.Contains(src, `"OK: `) {
				t.Fatalf("%s %s: emitted command still carries a canned acknowledgement string:\n%s", o.HTTP, o.Path, src)
			}
		}
	}
	if emitted == 0 {
		t.Fatal("emitted no command without a data return — the lookup is broken")
	}
	t.Logf("checked the emitted body of %d commands", emitted)
}
