package main

import "testing"

func TestListEnvelope(t *testing.T) {
	row := []schemaField{{Wire: "event_id", Type: "string"}}
	cases := []struct {
		name string
		resp []schemaField
		want string
	}{
		{"pagination siblings", []schemaField{
			{Wire: "items", Type: "array<object>", Children: row},
			{Wire: "total", Type: "integer"},
		}, "items"},
		{"object sibling stays a list", []schemaField{
			{Wire: "days", Type: "object"},
			{Wire: "items", Type: "array<object>", Children: row},
			{Wire: "total", Type: "integer"},
		}, "items"},
		{"two row arrays", []schemaField{
			{Wire: "items", Type: "array<object>", Children: row},
			{Wire: "list", Type: "array<object>", Children: row},
		}, ""},
		{"environments rows with a scalar sibling", []schemaField{
			{Wire: "environments", Type: "array<object>", Children: row},
			{Wire: "total", Type: "integer"},
			{Wire: "latest_version", Type: "string"},
		}, "environments"},
		{"no row array", []schemaField{
			{Wire: "rules", Type: "array<object>", Children: row},
			{Wire: "total", Type: "integer"},
		}, ""},
	}
	for _, c := range cases {
		if _, got := listEnvelope(c.resp); got != c.want {
			t.Errorf("%s: listEnvelope key = %q, want %q", c.name, got, c.want)
		}
	}
}
