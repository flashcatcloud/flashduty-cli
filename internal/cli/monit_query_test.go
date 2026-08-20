package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMonitQueryDiagnoseFlags(t *testing.T) {
	cmd := newMonitQueryDiagnoseCmd()
	for _, name := range []string{
		"ds-type", "ds-name", "time-start", "time-end",
		"input-query", "operation",
		"max-logs", "max-patterns", "timeout-seconds",
	} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s missing", name)
		}
	}
}

func TestMonitQueryRowsFlags(t *testing.T) {
	cmd := newMonitQueryRowsCmd()
	for _, name := range []string{"ds-type", "ds-name", "expr", "args"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s missing", name)
		}
	}
}

func TestMonitQueryDataFlags(t *testing.T) {
	cmd := newMonitQueryDataCmd()
	for _, name := range []string{"ds-type", "ds-name", "expr", "args", "delay-seconds"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s missing", name)
		}
	}
}

// --- monit-query diagnose -------------------------------------------------

func TestMonitQueryDiagnoseHappyPath(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"operation": "log_patterns"}

	_, err := execCommand(
		"monit-query", "diagnose",
		"--ds-type", "victorialogs",
		"--ds-name", "vl-prod",
		"--input-query", `{app="api"}`,
		"--operation", "log_patterns",
		"--max-logs", "5000",
		"--max-patterns", "10",
		"--timeout-seconds", "20",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastPath != "/monit/query/diagnose" {
		t.Fatalf("expected /monit/query/diagnose, got %q", stub.lastPath)
	}
	body := stub.lastBody
	if body["ds_type"] != "victorialogs" || body["ds_name"] != "vl-prod" {
		t.Errorf("unexpected ds fields: %#v", body)
	}
	input, _ := body["input"].(map[string]any)
	if input["query"] != `{app="api"}` {
		t.Errorf("expected input query %q, got %v", `{app="api"}`, input["query"])
	}
	if body["operation"] != "log_patterns" {
		t.Errorf("expected operation log_patterns, got %v", body["operation"])
	}
	options, _ := body["options"].(map[string]any)
	if fmt.Sprint(options["max_logs_scanned"]) != "5000" ||
		fmt.Sprint(options["max_patterns"]) != "10" ||
		fmt.Sprint(options["timeout_seconds"]) != "20" {
		t.Errorf("unexpected caps: %#v", options)
	}
	timeRange, _ := body["time_range"].(map[string]any)
	if fmt.Sprint(timeRange["start"]) == "0" || fmt.Sprint(timeRange["start"]) == "<nil>" ||
		fmt.Sprint(timeRange["end"]) == "0" || fmt.Sprint(timeRange["end"]) == "<nil>" {
		t.Errorf("expected non-zero default time range, got %#v", timeRange)
	}
}

func TestMonitQueryDiagnoseRendersMetricEvidence(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"schema_version": "2",
		"operation":      "metric_trends",
		"ds_type":        "prometheus",
		"ds_name":        "prod-prometheus",
		"query":          "up",
		"window":         map[string]any{"start": "2026-07-14T06:00:00Z", "end": "2026-07-14T07:00:00Z"},
		"results": []any{map[string]any{
			"method": "window_compare",
			"window": map[string]any{"start": "2026-07-14T06:00:00Z", "end": "2026-07-14T07:00:00Z"},
			"summary": map[string]any{
				"series_total": 1, "series_analyzed": 1, "selected_series_total": 1, "series_returned": 1,
				"analysis_truncated": false, "evidence_summary": "One series changed.",
			},
			"series_evidence": []any{map[string]any{
				"labels":       map[string]any{"instance": "api-1"},
				"observations": []any{"The current average increased."},
			}},
			"warnings": []any{},
		}},
	}

	out, err := execCommand(
		"monit-query", "diagnose",
		"--ds-type", "prometheus",
		"--ds-name", "prod-prometheus",
		"--input-query", "up",
		"--operation", "metric_trends",
		"--output-format", "json",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var rendered map[string]any
	if err := json.Unmarshal([]byte(out), &rendered); err != nil {
		t.Fatalf("decode CLI JSON: %v\n%s", err, out)
	}
	if _, found := rendered["data_handling"]; found {
		t.Fatalf("metric output fabricated data_handling: %s", out)
	}
	evidence := rendered["results"].([]any)[0].(map[string]any)["series_evidence"].([]any)[0].(map[string]any)
	for _, field := range []string{"comparison_status", "current_window_stats", "baseline_window_stats"} {
		if _, found := evidence[field]; found {
			t.Fatalf("metric evidence fabricated %s: %s", field, out)
		}
	}
}

func TestMonitQueryDiagnoseRequiredFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{
			name: "missing ds-type",
			args: []string{
				"monit-query", "diagnose",
				"--ds-name", "vl-prod",
				"--input-query", `{app="api"}`,
			},
		},
		{
			name: "missing ds-name",
			args: []string{
				"monit-query", "diagnose",
				"--ds-type", "victorialogs",
				"--input-query", `{app="api"}`,
			},
		},
		{
			name: "missing input-query",
			args: []string{
				"monit-query", "diagnose",
				"--ds-type", "victorialogs",
				"--ds-name", "vl-prod",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			_, err := execCommand(tc.args...)
			if err == nil {
				t.Fatal("expected required-flag error, got nil")
			}
			if !strings.Contains(err.Error(), "required") {
				t.Errorf("expected error to mention 'required', got %q", err.Error())
			}
			if stub.requests != 0 {
				t.Errorf("diagnose should not have been called: %d request(s)", stub.requests)
			}
		})
	}
}

func TestMonitQueryDiagnoseInvalidTimeStart(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-query", "diagnose",
		"--ds-type", "victorialogs",
		"--ds-name", "vl-prod",
		"--input-query", `{app="api"}`,
		"--time-start", "not-a-time",
	)
	if err == nil {
		t.Fatal("expected error for invalid --time-start, got nil")
	}
	if !strings.Contains(err.Error(), "--time-start") {
		t.Errorf("expected error to mention --time-start, got %q", err.Error())
	}
	if stub.requests != 0 {
		t.Errorf("diagnose should not have been called: %d request(s)", stub.requests)
	}
}

// --- monit-query rows -----------------------------------------------------

func TestMonitQueryDataHappyPath(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	// data returns the stable query_result.v1 envelope: data.{format,result}.
	stub.data = map[string]any{
		"format": "query_result.v1",
		"result": map[string]any{
			"kind": "samples",
			"samples": []any{
				map[string]any{"labels": map[string]any{"job": "api"}, "value": 1.25},
			},
		},
	}

	out, err := execCommand(
		"monit-query", "data",
		"--ds-type", "prometheus",
		"--ds-name", "prom-prod",
		"--expr", "up",
		"--delay-seconds", "30",
		"--args", "step=15s",
		"--output-format", "json",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastPath != "/monit/query/data" {
		t.Fatalf("expected /monit/query/data, got %q", stub.lastPath)
	}
	body := stub.lastBody
	if body["ds_type"] != "prometheus" || body["ds_name"] != "prom-prod" || body["expr"] != "up" {
		t.Errorf("unexpected data input: %#v", body)
	}
	if fmt.Sprint(body["delay_seconds"]) != "30" {
		t.Errorf("expected delay_seconds 30, got %v", body["delay_seconds"])
	}
	args, _ := body["args"].(map[string]any)
	if args["step"] != "15s" {
		t.Errorf("expected args step=15s, got %#v", args)
	}
	var rendered map[string]any
	if err := json.Unmarshal([]byte(out), &rendered); err != nil {
		t.Fatalf("decode CLI JSON: %v\n%s", err, out)
	}
	if rendered["format"] != "query_result.v1" {
		t.Errorf("expected format query_result.v1, got %v", rendered["format"])
	}
}

func TestMonitQueryDataRequiredFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{
			name: "missing ds-type",
			args: []string{
				"monit-query", "data",
				"--ds-name", "prom-prod",
				"--expr", "up",
			},
		},
		{
			name: "missing ds-name",
			args: []string{
				"monit-query", "data",
				"--ds-type", "prometheus",
				"--expr", "up",
			},
		},
		{
			name: "missing expr",
			args: []string{
				"monit-query", "data",
				"--ds-type", "prometheus",
				"--ds-name", "prom-prod",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			_, err := execCommand(tc.args...)
			if err == nil {
				t.Fatal("expected required-flag error, got nil")
			}
			if !strings.Contains(err.Error(), "required") {
				t.Errorf("expected error to mention 'required', got %q", err.Error())
			}
			if stub.requests != 0 {
				t.Errorf("data should not have been called: %d request(s)", stub.requests)
			}
		})
	}
}

func TestMonitQueryRowsHappyPath(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	// rows is a raw datasource passthrough: the response envelope "data" is a
	// JSON array of QueryRow ({fields,values}) objects, decoded into
	// QueryRowsResponse ([]QueryRow) and re-marshalled verbatim to the writer.
	stub.data = []any{
		map[string]any{
			"fields": map[string]any{"instance": "node-1"},
			"values": map[string]any{"__value__": 1},
		},
	}

	out, err := execCommand(
		"monit-query", "rows",
		"--ds-type", "prometheus",
		"--ds-name", "prom-prod",
		"--expr", "up",
		"--args", "step=15s",
		"--args", "tenant=acme",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastPath != "/monit/query/rows" {
		t.Fatalf("expected /monit/query/rows, got %q", stub.lastPath)
	}
	body := stub.lastBody
	if body["ds_type"] != "prometheus" || body["ds_name"] != "prom-prod" || body["expr"] != "up" {
		t.Errorf("unexpected rows input: %#v", body)
	}
	args, _ := body["args"].(map[string]any)
	if args["step"] != "15s" || args["tenant"] != "acme" {
		t.Errorf("expected args step=15s tenant=acme, got %#v", args)
	}
	// The rendered output is the re-marshalled row array (passthrough shape).
	if !strings.Contains(out, "node-1") || !strings.Contains(out, "__value__") {
		t.Errorf("expected rendered rows to carry the datasource payload, got:\n%s", out)
	}
}

func TestMonitQueryRowsRequiredFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{
			name: "missing ds-type",
			args: []string{
				"monit-query", "rows",
				"--ds-name", "prom-prod",
				"--expr", "up",
			},
		},
		{
			name: "missing ds-name",
			args: []string{
				"monit-query", "rows",
				"--ds-type", "prometheus",
				"--expr", "up",
			},
		},
		{
			name: "missing expr",
			args: []string{
				"monit-query", "rows",
				"--ds-type", "prometheus",
				"--ds-name", "prom-prod",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)

			_, err := execCommand(tc.args...)
			if err == nil {
				t.Fatal("expected required-flag error, got nil")
			}
			if !strings.Contains(err.Error(), "required") {
				t.Errorf("expected error to mention 'required', got %q", err.Error())
			}
			if stub.requests != 0 {
				t.Errorf("rows should not have been called: %d request(s)", stub.requests)
			}
		})
	}
}

// --- normalizeRawTimeArgs --------------------------------------------------

func TestNormalizeRawTimeArgsAcceptedFormats(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"rfc3339 utc", "2026-08-11T09:40:00Z"},
		{"rfc3339 offset", "2026-08-11T09:40:00+08:00"},
		{"unix seconds", "1786497600"},
		{"unix milliseconds", "1786497600000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]string{"victorialogs.start": tc.input, "victorialogs.end": tc.input}
			if err := normalizeRawTimeArgs("victorialogs", args); err != nil {
				t.Fatalf("normalizeRawTimeArgs(%q): unexpected error: %v", tc.input, err)
			}
			for _, key := range []string{"victorialogs.start", "victorialogs.end"} {
				if _, err := strconv.ParseInt(args[key], 10, 64); err != nil {
					t.Errorf("%s: expected normalized unix-seconds string, got %q", key, args[key])
				}
			}
		})
	}
}

func TestNormalizeRawTimeArgsLokiPrefix(t *testing.T) {
	args := map[string]string{"loki.start": "2026-08-11T09:40:00Z", "loki.end": "2026-08-11T10:05:00Z"}
	if err := normalizeRawTimeArgs("loki", args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantStart := strconv.FormatInt(time.Date(2026, 8, 11, 9, 40, 0, 0, time.UTC).Unix(), 10)
	wantEnd := strconv.FormatInt(time.Date(2026, 8, 11, 10, 5, 0, 0, time.UTC).Unix(), 10)
	if args["loki.start"] != wantStart || args["loki.end"] != wantEnd {
		t.Errorf("unexpected normalized loki args: %#v, want start=%s end=%s", args, wantStart, wantEnd)
	}
}

func TestNormalizeRawTimeArgsIgnoresOtherDsTypes(t *testing.T) {
	args := map[string]string{"prometheus.start": "not-a-time"}
	if err := normalizeRawTimeArgs("prometheus", args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args["prometheus.start"] != "not-a-time" {
		t.Errorf("expected prometheus args untouched, got %#v", args)
	}
}

func TestNormalizeRawTimeArgsIgnoresUnrelatedKeys(t *testing.T) {
	args := map[string]string{"victorialogs.type": "raw", "victorialogs.timespan.value": "15"}
	if err := normalizeRawTimeArgs("victorialogs", args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args["victorialogs.type"] != "raw" || args["victorialogs.timespan.value"] != "15" {
		t.Errorf("expected unrelated args untouched, got %#v", args)
	}
}

func TestNormalizeRawTimeArgsInvalidValue(t *testing.T) {
	args := map[string]string{"victorialogs.start": "not-a-time"}
	err := normalizeRawTimeArgs("victorialogs", args)
	if err == nil {
		t.Fatal("expected error for invalid victorialogs.start, got nil")
	}
	if !strings.Contains(err.Error(), "victorialogs.start") {
		t.Errorf("expected error to mention victorialogs.start, got %q", err.Error())
	}
}

// TestMonitQueryRowsRawModeNormalizesRFC3339 is the regression test for the
// raw-vs-stats time format inconsistency: a raw-mode VictoriaLogs query given
// RFC3339 --args timestamps must reach the server as the unix-seconds form
// the raw query path requires.
func TestMonitQueryRowsRawModeNormalizesRFC3339(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = []any{}

	_, err := execCommand(
		"monit-query", "rows",
		"--ds-type", "victorialogs",
		"--ds-name", "vl-prod",
		"--expr", `{app="api"} |= "error"`,
		"--args", "victorialogs.type=raw",
		"--args", "victorialogs.start=2026-08-11T09:40:00Z",
		"--args", "victorialogs.end=2026-08-11T10:05:00Z",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := stub.lastBody
	argsSent, _ := body["args"].(map[string]any)
	start, ok := argsSent["victorialogs.start"].(string)
	if !ok {
		t.Fatalf("expected victorialogs.start in request args, got %#v", argsSent)
	}
	if _, err := strconv.ParseInt(start, 10, 64); err != nil {
		t.Errorf("expected victorialogs.start to be unix-seconds, got %q", start)
	}
	end, ok := argsSent["victorialogs.end"].(string)
	if !ok {
		t.Fatalf("expected victorialogs.end in request args, got %#v", argsSent)
	}
	if _, err := strconv.ParseInt(end, 10, 64); err != nil {
		t.Errorf("expected victorialogs.end to be unix-seconds, got %q", end)
	}
	wantStart := time.Date(2026, 8, 11, 9, 40, 0, 0, time.UTC).Unix()
	wantEnd := time.Date(2026, 8, 11, 10, 5, 0, 0, time.UTC).Unix()
	if start != strconv.FormatInt(wantStart, 10) || end != strconv.FormatInt(wantEnd, 10) {
		t.Errorf("expected start=%d end=%d, got start=%s end=%s", wantStart, wantEnd, start, end)
	}
}

func TestMonitQueryDataInvalidArgs(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-query", "data",
		"--ds-type", "prometheus",
		"--ds-name", "prom-prod",
		"--expr", "up",
		"--args", "no-equals-sign",
	)
	if err == nil {
		t.Fatal("expected error for malformed --args, got nil")
	}
	if !strings.Contains(err.Error(), "--args") {
		t.Errorf("expected error to mention --args, got %q", err.Error())
	}
	if stub.requests != 0 {
		t.Errorf("data should not have been called: %d request(s)", stub.requests)
	}
}

func TestMonitQueryRowsInvalidArgs(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)

	_, err := execCommand(
		"monit-query", "rows",
		"--ds-type", "prometheus",
		"--ds-name", "prom-prod",
		"--expr", "up",
		"--args", "no-equals-sign",
	)
	if err == nil {
		t.Fatal("expected error for malformed --args, got nil")
	}
	if !strings.Contains(err.Error(), "--args") {
		t.Errorf("expected error to mention --args, got %q", err.Error())
	}
	if stub.requests != 0 {
		t.Errorf("rows should not have been called: %d request(s)", stub.requests)
	}
}
