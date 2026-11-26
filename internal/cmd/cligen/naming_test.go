package main

import "testing"

func TestDiagnosticsNamesSurviveAgentRetirement(t *testing.T) {
	names := methodNames("Monitors/Diagnostics", []string{"monit-read-query-data", "monit-read-query-diagnose"})
	if names["monit-read-query-data"] != "QueryData" || names["monit-read-query-diagnose"] != "QueryDiagnose" {
		t.Fatalf("supported method names changed: %v", names)
	}
}

func TestChangeNamesKeepListAlongsideWrites(t *testing.T) {
	names := methodNames("On-call/Changes", []string{"change-read-list", "change-read-info", "change-write-create", "change-write-update"})
	want := map[string]string{"change-read-list": "List", "change-read-info": "Info", "change-write-create": "Create", "change-write-update": "Update"}
	for id, name := range want {
		if names[id] != name {
			t.Fatalf("%s = %s, want %s (all: %v)", id, names[id], name, names)
		}
	}
}

func TestDashboardTagsGenerateDistinctServices(t *testing.T) {
	if got := serviceName("Monitors/Dashboards"); got != "Dashboards" {
		t.Fatalf("Monitors service = %s", got)
	}
	if got := serviceName("RUM/Dashboards"); got != "RumDashboards" {
		t.Fatalf("RUM service = %s", got)
	}
}
