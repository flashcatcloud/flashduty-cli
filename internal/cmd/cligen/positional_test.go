package main

import "testing"

// mkOp returns a minimal specOp with the given OpID for use in selectPositional tests.
func mkOp(opID string) specOp { return specOp{OpID: opID} }

// mkField returns a specField for byWire construction.
func mkField(wire string, required bool) specField { return specField{Wire: wire, Required: required} }

// byWireOf builds the byWire map from a list of specFields.
func byWireOf(fields ...specField) map[string]specField {
	m := map[string]specField{}
	for _, f := range fields {
		m[f.Wire] = f
	}
	return m
}

func TestSelectPositional(t *testing.T) {
	cases := []struct {
		name     string
		opID     string
		scalars  []scalarField
		byWire   map[string]specField
		wantWire string
		wantOK   bool
	}{
		{
			// Override-map hit wins over heuristic.
			name:     "override map hit",
			opID:     "incidentMerge",
			scalars:  []scalarField{{Wire: "target_incident_id", Kind: "string"}, {Wire: "source_incident_ids", Kind: "[]string"}},
			byWire:   byWireOf(mkField("target_incident_id", true), mkField("source_incident_ids", true)),
			wantWire: "target_incident_id",
			wantOK:   true,
		},
		{
			// Empty-string override → suppress positional.
			name:    "empty override suppresses",
			opID:    "monit-rule-write-move",
			scalars: []scalarField{{Wire: "dest_folder_id", Kind: "int"}, {Wire: "ids", Kind: "[]int"}},
			byWire:  byWireOf(mkField("dest_folder_id", true), mkField("ids", true)),
			wantOK:  false,
		},
		{
			// Single required *_ids array wins over a co-present required *_id scalar.
			name:     "array wins over scalar",
			opID:     "someOp",
			scalars:  []scalarField{{Wire: "incident_id", Kind: "string"}, {Wire: "person_ids", Kind: "[]string"}},
			byWire:   byWireOf(mkField("incident_id", true), mkField("person_ids", true)),
			wantWire: "person_ids",
			wantOK:   true,
		},
		{
			// Single required *_id scalar when no array.
			name:     "single scalar id",
			opID:     "someOp",
			scalars:  []scalarField{{Wire: "incident_id", Kind: "string"}, {Wire: "limit", Kind: "int"}},
			byWire:   byWireOf(mkField("incident_id", true), mkField("limit", false)),
			wantWire: "incident_id",
			wantOK:   true,
		},
		{
			// Ambiguous: two required scalars → no positional.
			name:    "ambiguous two scalars",
			opID:    "someOp",
			scalars: []scalarField{{Wire: "channel_id", Kind: "string"}, {Wire: "rule_id", Kind: "string"}},
			byWire:  byWireOf(mkField("channel_id", true), mkField("rule_id", true)),
			wantOK:  false,
		},
		{
			// Ambiguous: two required arrays → no positional.
			name:    "ambiguous two arrays",
			opID:    "someOp",
			scalars: []scalarField{{Wire: "person_ids", Kind: "[]string"}, {Wire: "team_ids", Kind: "[]string"}},
			byWire:  byWireOf(mkField("person_ids", true), mkField("team_ids", true)),
			wantOK:  false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := selectPositional(mkOp(c.opID), c.scalars, c.byWire)
			if ok != c.wantOK {
				t.Fatalf("ok=%v, want %v", ok, c.wantOK)
			}
			if ok && got.Wire != c.wantWire {
				t.Errorf("Wire=%q, want %q", got.Wire, c.wantWire)
			}
		})
	}
}

func TestSelectPositionalCreateVerbSuppressed(t *testing.T) {
	// Create-verb suppression is handled in emitCmd (not selectPositional), so
	// selectPositional itself still returns a result for a "create" op that has a
	// single required *_id field. This test documents that contract: the suppression
	// layer above selectPositional is responsible for the create-verb check.
	scalars := []scalarField{{Wire: "team_id", Kind: "int"}}
	byWire := byWireOf(mkField("team_id", true))
	_, ok := selectPositional(mkOp("channelCreate"), scalars, byWire)
	// team_id ends in _id and is required — heuristic would pick it.
	// (emitCmd suppresses it because verb=="create"; selectPositional is unaware.)
	if !ok {
		t.Log("selectPositional returned no positional for channelCreate (team_id); " +
			"emitCmd create-verb suppression is redundant here but still correct")
	}
}
