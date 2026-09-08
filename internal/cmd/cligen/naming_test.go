package main

import "testing"

func TestDiagnosticsNamesSurviveAgentRetirement(t *testing.T) {
	names := methodNames("Monitors/Diagnostics", []string{"monit-read-query-data", "monit-read-query-diagnose"})
	if names["monit-read-query-data"] != "QueryData" || names["monit-read-query-diagnose"] != "QueryDiagnose" {
		t.Fatalf("supported method names changed: %v", names)
	}
}
