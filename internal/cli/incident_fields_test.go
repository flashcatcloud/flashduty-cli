package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestCommandIncidentListFilterFlagsReachWire(t *testing.T) {
	tests := []struct {
		name  string
		extra []string
		field string
		want  any
	}{
		{"incident-ids", []string{"--incident-ids", "a1,b2"}, "incident_ids", []any{"a1", "b2"}},
		{"acker-ids", []string{"--acker-ids", "1,2"}, "acker_ids", []any{float64(1), float64(2)}},
		{"closer-ids", []string{"--closer-ids", "0"}, "closer_ids", []any{float64(0)}},
		{"creator-ids", []string{"--creator-ids", "3"}, "creator_ids", []any{float64(3)}},
		{"responder-ids", []string{"--responder-ids", "4,5"}, "responder_ids", []any{float64(4), float64(5)}},
		{"team-ids", []string{"--team-ids", "6"}, "team_ids", []any{float64(6)}},
		{"asc", []string{"--asc"}, "asc", true},
		{"ever-muted", []string{"--ever-muted"}, "ever_muted", true},
		{"is-my-channel", []string{"--is-my-channel"}, "is_my_channel", true},
		{"is-my-team", []string{"--is-my-team"}, "is_my_team", true},
		{"is-rare", []string{"--is-rare"}, "is_rare", true},
		{"is-snoozed", []string{"--is-snoozed"}, "is_snoozed", true},
		{"default omits", nil, "is_snoozed", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			args := append([]string{"incident", "list"}, tc.extra...)
			if _, err := execCommand(args...); err != nil {
				t.Fatalf("[incident-list-filters] unexpected error: %v", err)
			}
			if stub.lastPath != "/incident/list" {
				t.Fatalf("[incident-list-filters] expected /incident/list, got %q", stub.lastPath)
			}
			if got := stub.lastBody[tc.field]; !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("[incident-list-filters] %s: want %#v, got %#v", tc.field, tc.want, got)
			}
		})
	}
}

func TestCommandIncidentListRejectsBadMemberID(t *testing.T) {
	saveAndResetGlobals(t)
	newGFStub(t)

	_, err := execCommand("incident", "list", "--acker-ids", "x")
	if err == nil {
		t.Fatal("[incident-list-bad-id] expected an error, got nil")
	}
}

func TestCommandIncidentFeedFlagsReachWire(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("incident", "feed", "inc-1", "--asc", "--types", "i_comm,i_assign"); err != nil {
		t.Fatalf("[incident-feed-flags] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/feed" {
		t.Fatalf("[incident-feed-flags] expected /incident/feed, got %q", stub.lastPath)
	}
	if stub.lastBody["asc"] != true {
		t.Fatalf("[incident-feed-flags] asc: got %#v", stub.lastBody["asc"])
	}
	if want := []any{"i_comm", "i_assign"}; !reflect.DeepEqual(stub.lastBody["types"], want) {
		t.Fatalf("[incident-feed-flags] types: want %#v, got %#v", want, stub.lastBody["types"])
	}
}

func TestCommandIncidentMergeTitleAndCommentFile(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	commentFile := writeCommentFile(t, "merged: same root cause `db`")
	if _, err := execCommand("incident", "merge", "target-1", "--source", "inc-1",
		"--title", "DB outage", "--comment-file", commentFile); err != nil {
		t.Fatalf("[incident-merge-title-comment] unexpected error: %v", err)
	}
	if stub.lastBody["title"] != "DB outage" {
		t.Fatalf("[incident-merge-title-comment] title: got %#v", stub.lastBody["title"])
	}
	if stub.lastBody["comment"] != "merged: same root cause `db`" {
		t.Fatalf("[incident-merge-title-comment] comment: got %#v", stub.lastBody["comment"])
	}
}

func TestCommandIncidentMergeOmitsTitleAndCommentByDefault(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("incident", "merge", "target-1", "--source", "inc-1"); err != nil {
		t.Fatalf("[incident-merge-default] unexpected error: %v", err)
	}
	for _, k := range []string{"title", "comment"} {
		if _, ok := stub.lastBody[k]; ok {
			t.Fatalf("[incident-merge-default] %s should be omitted, got %#v", k, stub.lastBody[k])
		}
	}
}

func TestCommandIncidentCommentTypeID(t *testing.T) {
	tests := []struct {
		name  string
		extra []string
		want  any
	}{
		{"default omits", nil, nil},
		{"flag sets", []string{"--comment-type-id", "65f0c0ffee0000000000abcd"}, "65f0c0ffee0000000000abcd"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newIncidentCommentEchoStub(t)
			commentFile := writeCommentFile(t, "note")

			args := append([]string{"incident", "comment", "inc-1", "--comment-file", commentFile}, tc.extra...)
			if _, err := execCommand(args...); err != nil {
				t.Fatalf("[incident-comment-type] unexpected error: %v", err)
			}
			if got := stub.bodies[0]["comment_type_id"]; got != tc.want {
				t.Fatalf("[incident-comment-type] comment_type_id: want %#v, got %#v", tc.want, got)
			}
		})
	}
}

func stubCustomFieldDefs(stub *gfStub) {
	stub.dataForPath = func(path string, _ map[string]any) any {
		if path == "/field/list" {
			return map[string]any{"items": []any{
				map[string]any{"field_name": "region", "field_type": "text"},
				map[string]any{"field_name": "urgent", "field_type": "checkbox"},
				map[string]any{"field_name": "tags", "field_type": "multi_select"},
				map[string]any{"field_name": "tier", "field_type": "single_select"},
			}}
		}
		return map[string]any{}
	}
}

func TestCommandIncidentCreateCustomFieldsAndAssignment(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stubCustomFieldDefs(stub)

	_, err := execCommand("incident", "create", "--title", "db down", "--severity", "Critical",
		"--field", "region=123", "--field", "urgent=true", "--field", "tags=db,core", "--field", `tier=["x"]`,
		"--assign-emails", "a@example.com,b@example.com", "--escalate-rule-id", "0123456789abcdef01234567", "--escalate-layer", "1")
	if err != nil {
		t.Fatalf("[incident-create-fields] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/create" {
		t.Fatalf("[incident-create-fields] expected /incident/create, got %q", stub.lastPath)
	}
	wantFields := map[string]any{"region": "123", "urgent": true, "tags": []any{"db", "core"}, "tier": `["x"]`}
	if got := stub.lastBody["fields"]; !reflect.DeepEqual(got, wantFields) {
		t.Fatalf("[incident-create-fields] fields: want %#v, got %#v", wantFields, got)
	}
	wantAssigned := map[string]any{
		"emails":           []any{"a@example.com", "b@example.com"},
		"escalate_rule_id": "0123456789abcdef01234567",
		"layer_idx":        float64(1),
		"type":             "assign",
	}
	if got := stub.lastBody["assigned_to"]; !reflect.DeepEqual(got, wantAssigned) {
		t.Fatalf("[incident-create-fields] assigned_to: want %#v, got %#v", wantAssigned, got)
	}
}

func TestCommandIncidentCreateWithoutNewFlagsKeepsBody(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	if _, err := execCommand("incident", "create", "--title", "db down", "--severity", "Critical"); err != nil {
		t.Fatalf("[incident-create-default] unexpected error: %v", err)
	}
	if stub.requests != 1 {
		t.Fatalf("[incident-create-default] expected 1 request (no field lookup), got %d", stub.requests)
	}
	for _, key := range []string{"fields", "assigned_to"} {
		if _, ok := stub.lastBody[key]; ok {
			t.Fatalf("[incident-create-default] %s must be omitted, got %#v", key, stub.lastBody[key])
		}
	}
}

func TestCommandIncidentUpdateCustomFieldIsTyped(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stubCustomFieldDefs(stub)

	if _, err := execCommand("incident", "update", "inc-1", "--field", "urgent=false"); err != nil {
		t.Fatalf("[incident-update-field-typed] unexpected error: %v", err)
	}
	if stub.lastPath != "/incident/field/reset" {
		t.Fatalf("[incident-update-field-typed] expected /incident/field/reset, got %q", stub.lastPath)
	}
	if got, ok := stub.lastBody["field_value"]; !ok || got != false {
		t.Fatalf("[incident-update-field-typed] field_value: want false, got %#v (present=%t)", got, ok)
	}
}

func TestCommandIncidentCustomFieldErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"layer without rule", []string{"incident", "create", "--title", "db down", "--severity", "Info", "--escalate-layer", "1"}, "--escalate-layer requires --escalate-rule-id"},
		{"unknown field", []string{"incident", "create", "--title", "db down", "--severity", "Info", "--field", "nope=1"}, "unknown custom field"},
		{"bad checkbox", []string{"incident", "update", "inc-1", "--field", "urgent=maybe"}, "true or false"},
		{"bad multi-select json", []string{"incident", "update", "inc-1", "--field", "tags=[oops"}, "invalid JSON array"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			stubCustomFieldDefs(stub)
			_, err := execCommand(tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("[incident-custom-field-errors] want error containing %q, got %v", tc.want, err)
			}
		})
	}
}
