package cli

import (
	"reflect"
	"testing"
)

// TestCuratedListFilterFlagsReachWireBody checks that each filter/sort flag on
// the audit, change, channel and field list commands lands on its wire field,
// and that omitting every new flag leaves those fields out of the body.
func TestCuratedListFilterFlagsReachWireBody(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		path    string
		want    map[string]any
		absent  []string
		wantErr bool
	}{
		{
			name:   "audit defaults omit new filters",
			args:   []string{"audit", "search"},
			path:   "/audit/search",
			absent: []string{"is_write", "is_dangerous", "request_id"},
		},
		{
			name: "audit filters",
			args: []string{"audit", "search", "--is-write=false", "--is-dangerous", "--request-id", "req-1"},
			path: "/audit/search",
			want: map[string]any{"is_write": false, "is_dangerous": true, "request_id": "req-1"},
		},
		{
			name:   "change defaults omit new fields",
			args:   []string{"change", "list"},
			path:   "/change/list",
			absent: []string{"asc", "orderby", "include_events", "filters"},
		},
		{
			name: "change sort, events and filters",
			args: []string{"change", "list", "--orderby", "last_time", "--asc", "--include-events", "--filters", `[{"key":"labels.env","oper":"IN","vals":["prod"]}]`},
			path: "/change/list",
			want: map[string]any{
				"orderby":        "last_time",
				"asc":            true,
				"include_events": true,
				"filters":        []any{map[string]any{"key": "labels.env", "oper": "IN", "vals": []any{"prod"}}},
			},
		},
		{
			name:    "change filters rejects bad JSON",
			args:    []string{"change", "list", "--filters", "not-json"},
			wantErr: true,
		},
		{
			name:   "channel defaults omit new fields",
			args:   []string{"channel", "list"},
			path:   "/channel/list",
			absent: []string{"p", "limit", "query", "channel_name", "channel_ids", "is_brief", "is_my_team", "is_my_starred", "is_my_managed", "orderby", "asc"},
		},
		{
			name: "channel filters and paging",
			args: []string{"channel", "list", "--name", "pay.ments", "--channel-name", "payments", "--channel-ids", "1,2", "--is-my-team", "--is-my-managed", "--is-brief", "--orderby", "channel_name", "--asc", "--page", "2", "--limit", "10"},
			path: "/channel/list",
			want: map[string]any{
				"query": `pay\.ments`, "channel_name": "payments", "channel_ids": []any{float64(1), float64(2)},
				"is_my_team": true, "is_my_managed": true, "is_brief": true,
				"orderby": "channel_name", "asc": true, "p": float64(2), "limit": float64(10),
			},
		},
		{
			name: "channel starred",
			args: []string{"channel", "list", "--is-my-starred"},
			path: "/channel/list",
			want: map[string]any{"is_my_starred": true},
		},
		{
			name:   "field defaults omit new fields",
			args:   []string{"field", "list"},
			path:   "/field/list",
			absent: []string{"query", "creator_id", "orderby", "asc"},
		},
		{
			name: "field filters and sort",
			args: []string{"field", "list", "--query", "^env", "--creator-id", "42", "--orderby", "field_name", "--asc"},
			path: "/field/list",
			want: map[string]any{"query": "^env", "creator_id": float64(42), "orderby": "field_name", "asc": true},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			_, err := execCommand(tc.args...)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("[misc-list-flags] expected an error for %v", tc.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("[misc-list-flags] unexpected error: %v", err)
			}
			if stub.lastPath != tc.path {
				t.Fatalf("[misc-list-flags] expected %s, got %q", tc.path, stub.lastPath)
			}
			for k, want := range tc.want {
				if got := stub.lastBody[k]; !reflect.DeepEqual(got, want) {
					t.Errorf("[misc-list-flags] %s: want %#v, got %#v", k, want, got)
				}
			}
			for _, k := range tc.absent {
				if v, ok := stub.lastBody[k]; ok {
					t.Errorf("[misc-list-flags] %s should be absent by default, got %#v", k, v)
				}
			}
		})
	}
}
