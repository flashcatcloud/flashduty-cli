package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMonitQueryDataFlags(t *testing.T) {
	cmd := newMonitQueryDataCmd()
	for _, name := range []string{"ds-type", "ds-name", "expr", "args", "delay-seconds"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s missing", name)
		}
	}
}

func TestRetiredMonitCommandsRejectBeforeRequest(t *testing.T) {
	for _, args := range [][]string{
		{"monit-query", "diagnose"}, {"monit", "query-diagnose"},
		{"monit", "rule-counter-status"},
		{"monit", "store-ruleset-create"}, {"monit", "store-ruleset-update"},
		{"monit", "store-ruleset-list"}, {"monit", "store-ruleset-info"}, {"monit", "store-ruleset-delete"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			_, err := execCommand(args...)
			if err == nil || !strings.Contains(err.Error(), "unknown command") {
				t.Fatalf("retired command error=%v", err)
			}
			if stub.requests != 0 {
				t.Fatalf("retired command sent %d requests", stub.requests)
			}
		})
	}
}

// --- monit-query data -----------------------------------------------------

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

// TestMonitQueryDataRawModeNormalizesRFC3339 is the regression test for the
// raw-vs-stats time format inconsistency: a raw-mode VictoriaLogs query given
// RFC3339 --args timestamps must reach the server as the unix-seconds form
// the raw query path requires.
func TestMonitQueryDataRawModeNormalizesRFC3339(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"format": "query_result.v1", "result": map[string]any{"kind": "records", "records": []any{}}}

	_, err := execCommand(
		"monit-query", "data",
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
