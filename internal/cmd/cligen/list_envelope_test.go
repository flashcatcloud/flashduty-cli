package main

import "testing"

func TestListEnvelope(t *testing.T) {
	row := []schemaField{{Wire: "event_id", Type: "string"}}
	cases := []struct {
		name string
		resp []schemaField
		want bool
	}{
		{"pagination siblings", []schemaField{
			{Wire: "items", Type: "array<object>", Children: row},
			{Wire: "total", Type: "integer"},
		}, true},
		{"object sibling stays a list", []schemaField{
			{Wire: "days", Type: "object"},
			{Wire: "items", Type: "array<object>", Children: row},
			{Wire: "total", Type: "integer"},
		}, true},
		{"two row arrays", []schemaField{
			{Wire: "items", Type: "array<object>", Children: row},
			{Wire: "list", Type: "array<object>", Children: row},
		}, false},
		{"no row array", []schemaField{
			{Wire: "rules", Type: "array<object>", Children: row},
			{Wire: "total", Type: "integer"},
		}, false},
	}
	for _, c := range cases {
		if _, got := listEnvelope(c.resp); got != c.want {
			t.Errorf("%s: listEnvelope = %v, want %v", c.name, got, c.want)
		}
	}
}
