package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func decodeJSON(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("stdout is not pure JSON: %v\n%s", err, s)
	}
	return v
}

func keysOf(m map[string]any) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestGlobalFieldsGeneratedList: the global --fields projects every row of a
// generated list envelope and keeps the envelope's own keys.
func TestGlobalFieldsGeneratedList(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{
		map[string]any{"member_id": 100, "member_name": "Alice", "email": "a@example.com"},
		map[string]any{"member_id": 200, "member_name": "Bob"},
	}, "total": 2}

	stdout, stderr, err := execCommandSplit("member", "list", "--fields", "member_id,email", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	env, ok := decodeJSON(t, stdout).(map[string]any)
	if !ok || env["total"] != float64(2) {
		t.Fatalf("envelope keys lost: %s", stdout)
	}
	items := env["items"].([]any)
	for _, it := range items {
		row := it.(map[string]any)
		if len(row) != 2 || row["member_name"] != nil {
			t.Errorf("row not projected to member_id,email: %v", row)
		}
	}
	if _, ok := items[1].(map[string]any)["email"]; !ok {
		t.Errorf("every row should carry every projected key: %v", items[1])
	}
	if stderr != "" {
		t.Errorf("no note expected, got %q", stderr)
	}
}

// TestGlobalFieldsCuratedDetail: a curated single-record verb that had no
// --fields of its own now projects its record.
func TestGlobalFieldsCuratedDetail(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = alertRow()

	stdout, _, err := execCommandSplit("alert", "get", "alert-1", "--fields", "alert_id,title", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	rec := decodeJSON(t, stdout).(map[string]any)
	if len(rec) != 2 || rec["alert_id"] == nil || rec["title"] == nil {
		t.Fatalf("want exactly alert_id,title, got keys %v", keysOf(rec))
	}
}

// TestGlobalFieldsUnmatchedNamesNoted: a name matching no key is reported on
// stderr with the keys that exist; stdout stays the projection of the rest.
func TestGlobalFieldsUnmatchedNamesNoted(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = alertRow()

	stdout, stderr, err := execCommandSplit("alert", "get", "alert-1", "--fields", "alert_id,alert_titel", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if !strings.Contains(stderr, "note: --fields alert_titel matched no key in the output (keys present: ") || !strings.Contains(stderr, "title") {
		t.Errorf("want unmatched-field note listing present keys, got %q", stderr)
	}
	if rec := decodeJSON(t, stdout).(map[string]any); len(rec) != 1 {
		t.Errorf("want only alert_id, got %v", keysOf(rec))
	}
}

// TestGlobalFieldsTableModeIgnored: table output is unchanged by --fields.
func TestGlobalFieldsTableModeIgnored(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{map[string]any{"member_id": 100, "member_name": "Alice"}}, "total": 1}

	withFields, _, err := execCommandSplit("member", "list", "--fields", "member_id")
	if err != nil {
		t.Fatal(err)
	}
	saveAndResetGlobals(t)
	stub = newGFStub(t)
	stub.data = map[string]any{"items": []any{map[string]any{"member_id": 100, "member_name": "Alice"}}, "total": 1}
	without, _, err := execCommandSplit("member", "list")
	if err != nil {
		t.Fatal(err)
	}
	if withFields != without {
		t.Fatalf("table output changed by --fields:\n%s\nvs\n%s", withFields, without)
	}
}

// TestGlobalFieldsOnAcknowledgement: a write whose output is an
// acknowledgement prints it unchanged and says --fields did not apply.
func TestGlobalFieldsOnAcknowledgement(t *testing.T) {
	saveAndResetGlobals(t)
	newGFStub(t)

	stdout, stderr, err := execCommandSplit("channel", "disable", "1", "--fields", "x", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if !strings.Contains(stdout, `"message"`) {
		t.Errorf("acknowledgement should print unchanged, got %q", stdout)
	}
	if !strings.Contains(stderr, `note: --fields does not apply to the output of "flashduty channel disable"; printed unchanged`) {
		t.Errorf("want not-applied note, got %q", stderr)
	}
}

// TestLocalFieldsFlagShadowsGlobal: a command whose own --fields is a request
// field sends it and does not project its output.
func TestLocalFieldsFlagShadowsGlobal(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = []any{map[string]any{"message": "ok"}}

	stdout, _, err := execCommandSplit("monit", "rule-update-fields", "--data", `{"ids":[1],"enabled":false}`, "--fields", "enabled", "--output-format", "json")
	if err != nil {
		t.Fatalf("execCommand: %v", err)
	}
	if got := stub.bodyStrings("fields"); len(got) != 1 || got[0] != "enabled" {
		t.Errorf("request field fields = %v, want [enabled]", got)
	}
	if !strings.Contains(stdout, `"message": "ok"`) {
		t.Errorf("output must not be projected by the request-field flag: %s", stdout)
	}
}

// TestProjectOutputShapes covers the projection directly, including the shape
// that cannot be projected.
func TestProjectOutputShapes(t *testing.T) {
	saveAndResetGlobals(t)
	cmd := rootCmd
	p, err := projectOutput(cmd, []map[string]any{{"a": 1, "b": 2}, {"a": 3}}, []string{"a", "z"})
	if err != nil || len(p.missing) != 1 || p.missing[0] != "z" || strings.Join(p.keys, ",") != "a,b" {
		t.Fatalf("rows: %+v err=%v", p, err)
	}
	if rows := p.out.([]any); len(rows) != 2 || len(rows[0].(map[string]any)) != 1 {
		t.Fatalf("rows not projected: %v", p.out)
	}

	// A record whose own field is named items is a record, not an envelope.
	p, err = projectOutput(cmd, map[string]any{"id": 1, "items": []any{map[string]any{"x": 1}}}, []string{"id"})
	if err != nil || p.out.(map[string]any)["id"] == nil {
		t.Fatalf("record with an items field: %+v err=%v", p, err)
	}

	// An empty page reports nothing missing.
	p, err = projectOutput(cmd, map[string]any{"items": []any{}, "total": 0}, []string{"a"})
	if err != nil || len(p.missing) != 0 {
		t.Fatalf("empty page: %+v err=%v", p, err)
	}

	if _, err := projectOutput(cmd, []string{"x", "y"}, []string{"a"}); err == nil || !strings.Contains(err.Error(), "drop --fields") {
		t.Fatalf("list of scalars should be a shape error, got %v", err)
	}
}

// TestGlobalFieldsGeneratedNoteNamesOriginalKeys: on the generated-command
// path the unmatched-field note is printed once and lists the keys of the
// unprojected rows.
func TestGlobalFieldsGeneratedNoteNamesOriginalKeys(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{map[string]any{"member_id": 100, "member_name": "Alice"}}, "total": 1}

	_, stderr, err := execCommandSplit("member", "list", "--fields", "member_id,membr_name", "--output-format", "json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(stderr, "matched no key") != 1 || !strings.Contains(stderr, "member_name") {
		t.Fatalf("want one note listing member_name, got %q", stderr)
	}
}
