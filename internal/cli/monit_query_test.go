package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flashcatcloud/go-flashduty"
)

// invokeRawStub captures the raw request body sent to the unified tool
// entry and replies with a canned tool envelope. Byte-level assertions need
// the raw body: decoding to map[string]any first would hide float64 rounding.
func invokeRawStub(t *testing.T) chan string {
	t.Helper()
	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/monit/datasource/tools/invoke" {
			t.Errorf("unexpected endpoint: %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		requests <- string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"request_id":"monit-query-test","data":{"datasource_id":42,"tool":"prometheus.query","data":{"format":"explore_result.v1","result":{"kind":"samples","samples":[{"labels":{"job":"api"},"value":1.25}]}}}}`)
	}))
	t.Cleanup(server.Close)
	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test", flashduty.WithBaseURL(server.URL))
	}
	return requests
}

func TestMonitQueryToolInvokePreservesParams(t *testing.T) {
	saveAndResetGlobals(t)
	requests := invokeRawStub(t)
	out, err := execCommand("monit-query", "42", "--tool", "prometheus.query",
		"--params", `{"expr":"sum by (job) (rate(http_requests_total[5m]))","execution":{"kind":"instant","to_ms":9007199254740993}}`,
		"--output-format", "json")
	if err != nil {
		t.Fatal(err)
	}
	raw := <-requests
	var sent struct {
		DatasourceID uint64          `json:"datasource_id"`
		Tool         string          `json:"tool"`
		Params       json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal([]byte(raw), &sent); err != nil {
		t.Fatal(err)
	}
	if sent.DatasourceID != 42 || sent.Tool != "prometheus.query" {
		t.Fatalf("request lost identity: %s", raw)
	}
	if !strings.Contains(string(sent.Params), `"to_ms":9007199254740993`) || strings.Contains(string(sent.Params), "9007199254740992") {
		t.Fatalf("params lost numeric precision: %s", sent.Params)
	}
	if !strings.Contains(out, "explore_result.v1") {
		t.Fatalf("response lost query evidence: %s", out)
	}
}

func TestMonitQueryToolInvokeParamsFromStdin(t *testing.T) {
	saveAndResetGlobals(t)
	requests := invokeRawStub(t)
	stdinReader = strings.NewReader(`{"expr":"up","execution":{"kind":"range","from_ms":9007199254740993,"to_ms":9007199254740995,"max_data_points":100}}`)
	_, err := execCommand("monit-query", "42", "--tool", "prometheus.query", "--params", "-", "--json")
	if err != nil {
		t.Fatal(err)
	}
	raw := <-requests
	if !strings.Contains(raw, `"from_ms":9007199254740993`) || strings.Contains(raw, "9007199254740992") {
		t.Fatalf("stdin params lost numeric precision: %s", raw)
	}
}

func TestMonitQueryToolInvokeOmitsParams(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	if _, err := execCommand("monit-query", "42", "--tool", "redis_node.overview"); err != nil {
		t.Fatal(err)
	}
	if stub.lastPath != "/monit/datasource/tools/invoke" {
		t.Fatalf("unexpected path: %s", stub.lastPath)
	}
	if _, present := stub.lastBody["params"]; present {
		t.Fatalf("params should be omitted, got %v", stub.lastBody["params"])
	}
	if stub.lastBody["tool"] != "redis_node.overview" {
		t.Fatalf("unexpected tool: %v", stub.lastBody["tool"])
	}
}

func TestMonitQueryToolInvokeAccountID(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	if _, err := execCommand("monit-query", "42", "--tool", "redis_node.overview", "--account-id", "7"); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(stub.lastBody["account_id"]) != "7" {
		t.Fatalf("account_id not stamped: %v", stub.lastBody)
	}
}

func TestMonitQueryToolInvokeRejectsBadInputBeforeRequest(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"missing tool", []string{"monit-query", "42"}, "required"},
		{"bad datasource id", []string{"monit-query", "abc", "--tool", "prometheus.query"}, "datasource-id"},
		{"zero datasource id", []string{"monit-query", "0", "--tool", "prometheus.query"}, "datasource-id"},
		{"malformed params", []string{"monit-query", "42", "--tool", "prometheus.query", "--params", "{oops"}, "--params"},
		{"null params", []string{"monit-query", "42", "--tool", "prometheus.query", "--params", "null"}, "--params"},
		{"array params", []string{"monit-query", "42", "--tool", "prometheus.query", "--params", "[1,2]"}, "--params must be a JSON object"},
		{"string params", []string{"monit-query", "42", "--tool", "prometheus.query", "--params", `"up"`}, "--params must be a JSON object"},
		{"trailing params value", []string{"monit-query", "42", "--tool", "prometheus.query", "--params", "{} {}"}, "--params"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			_, err := execCommand(tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want substring %q", err, tc.want)
			}
			if stub.requests != 0 {
				t.Fatalf("request sent despite invalid input: %d", stub.requests)
			}
		})
	}
}

func TestMonitQueryToolErrorsPassThrough(t *testing.T) {
	saveAndResetGlobals(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"request_id":"trace-query","error":{"code":"BadRequest","reason":"edge_upgrade_required","message":"query tools require Explore-capable Edge"}}`)
	}))
	t.Cleanup(server.Close)
	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test", flashduty.WithBaseURL(server.URL))
	}
	_, err := execCommand("monit-query", "42", "--tool", "prometheus.query",
		"--params", `{"expr":"up","execution":{"kind":"instant","to_ms":1757462400000}}`)
	var apiErr *flashduty.ErrorResponse
	if !errors.As(err, &apiErr) || apiErr.Reason != "edge_upgrade_required" || calls.Load() != 1 {
		t.Fatalf("error lost or replayed: %v, calls=%d", err, calls.Load())
	}
	if !strings.Contains(err.Error(), "edge_upgrade_required") || !strings.Contains(err.Error(), "trace-query") {
		t.Fatalf("CLI error omitted reason/request ID: %v", err)
	}
}

// The retired `monit-query data` subcommand (and the long-gone `diagnose`)
// must fail before any request: monit-query is now a leaf tool-invoke command.
func TestRetiredMonitQuerySubcommandsRejectBeforeRequest(t *testing.T) {
	for _, args := range [][]string{
		{"monit-query", "data", "--ds-type", "prometheus", "--ds-name", "prom-prod", "--expr", "up"},
		{"monit-query", "data"},
		{"monit-query", "diagnose"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			_, err := execCommand(args...)
			if err == nil {
				t.Fatal("retired command form succeeded")
			}
			if stub.requests != 0 {
				t.Fatalf("retired command form sent %d requests", stub.requests)
			}
		})
	}
}

func TestRetiredMonitCommandsRejectBeforeRequest(t *testing.T) {
	for _, args := range [][]string{
		{"monit", "query-diagnose"},
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

// sentToolParams runs monit-query (default format) against invokeRawStub and
// decodes the params the request carried (numbers kept as json.Number); it
// also returns stdout, which leads with the resolved window line.
func sentToolParams(t *testing.T, args ...string) (map[string]any, string) {
	t.Helper()
	requests := invokeRawStub(t)
	stdout, stderr, err := execCommandSplit(append([]string{"monit-query", "42", "--tool", "prometheus.query"}, args...)...)
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Fatalf("stderr not empty: %q", stderr)
	}
	raw := <-requests
	var sent struct {
		Params map[string]any `json:"params"`
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&sent); err != nil {
		t.Fatal(err)
	}
	return sent.Params, stdout
}

func TestMonitQueryWindowFlags(t *testing.T) {
	const (
		from = "2026-01-02T14:00:00+08:00" // 1767333600000
		to   = "2026-01-02T15:00:00+08:00" // 1767337200000
	)
	for _, tc := range []struct {
		name      string
		args      []string
		execution map[string]string // want execution fields (json text); "" = absent
		note      string
	}{
		{"since and until into existing execution",
			[]string{"--params", `{"expr":"rate(http_requests_total{code=~\"5..\"}[5m]) > 0 < 10","execution":{"kind":"range","max_data_points":240}}`, "--since", from, "--until", to},
			map[string]string{"kind": "range", "max_data_points": "240", "from_ms": "1767333600000", "to_ms": "1767337200000"},
			"window 2026-01-02T14:00:00+08:00..2026-01-02T15:00:00+08:00 (1h, "},
		{"until only is an instant",
			[]string{"--params", `{"expr":"up","execution":{"kind":"instant"}}`, "--until", to},
			map[string]string{"kind": "instant", "to_ms": "1767337200000", "from_ms": ""},
			"instant 2026-01-02T15:00:00+08:00"},
		{"since keeps a to_ms from params",
			[]string{"--params", `{"expr":"up","execution":{"kind":"window","to_ms":1767337200000}}`, "--since", from},
			map[string]string{"kind": "window", "from_ms": "1767333600000", "to_ms": "1767337200000"},
			"window 2026-01-02T14:00:00+08:00..2026-01-02T15:00:00+08:00"},
		{"until keeps a from_ms from params",
			[]string{"--params", `{"expr":"up","execution":{"kind":"window","from_ms":1767333600000}}`, "--until", to},
			map[string]string{"kind": "window", "from_ms": "1767333600000", "to_ms": "1767337200000"},
			"(1h, "},
		{"params without execution get one without kind",
			[]string{"--params", `{"expr":"up"}`, "--since", from, "--until", to},
			map[string]string{"kind": "", "from_ms": "1767333600000", "to_ms": "1767337200000"},
			"window "},
		{"no params at all",
			[]string{"--since", from, "--until", to},
			map[string]string{"kind": "", "from_ms": "1767333600000", "to_ms": "1767337200000"},
			"window "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			pinClock(t, time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC))
			params, stdout := sentToolParams(t, tc.args...)
			execution, _ := params["execution"].(map[string]any)
			for key, want := range tc.execution {
				got, present := execution[key]
				if want == "" {
					if present {
						t.Errorf("execution.%s = %v, want absent", key, got)
					}
					continue
				}
				if fmt.Sprint(got) != want {
					t.Errorf("execution.%s = %v, want %s", key, got, want)
				}
			}
			first := strings.SplitN(stdout, "\n", 2)[0]
			isWindow := strings.HasPrefix(first, "window ") || strings.HasPrefix(first, "instant ")
			if !isWindow || !strings.Contains(first, tc.note) {
				t.Errorf("first stdout line %q lacks %q", first, tc.note)
			}
		})
	}
}

func TestMonitQuerySinceDefaultsUntilToNow(t *testing.T) {
	saveAndResetGlobals(t)
	before := time.Now().UnixMilli()
	params, _ := sentToolParams(t, "--params", `{"expr":"up","execution":{"kind":"range","max_data_points":10}}`, "--since", "30m")
	after := time.Now().UnixMilli()
	execution := params["execution"].(map[string]any)
	fromMS, _ := execution["from_ms"].(json.Number).Int64()
	toMS, _ := execution["to_ms"].(json.Number).Int64()
	if toMS < before-1000 || toMS > after {
		t.Fatalf("to_ms %d not now (%d..%d)", toMS, before, after)
	}
	if span := toMS - fromMS; span < 30*60*1000-1000 || span > 30*60*1000+1000 {
		t.Fatalf("window span %dms, want 30m", span)
	}
}

// Re-encoding for the window flags keeps other large integers exact and the
// query text unescaped.
func TestApplyToolWindowFidelity(t *testing.T) {
	in := json.RawMessage(`{"expr":"a < b && c > d","limit":9007199254740993,"execution":{"kind":"range","max_data_points":18446744073709551615}}`)
	out, err := applyToolWindow(in, "2026-01-02T14:00:00+08:00", "2026-01-02T15:00:00+08:00", true, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"execution":{"from_ms":1767333600000,"kind":"range","max_data_points":18446744073709551615,"to_ms":1767337200000},"expr":"a < b && c > d","limit":9007199254740993}`
	if string(out) != want {
		t.Fatalf("got  %s\nwant %s", out, want)
	}
}

func TestMonitQueryWindowFlagsRejectBeforeRequest(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{"since conflicts with from_ms", []string{"--params", `{"expr":"up","execution":{"kind":"range","from_ms":1767333600000}}`, "--since", "1h"},
			[]string{"--since", "execution.from_ms"}},
		{"until conflicts with to_ms", []string{"--params", `{"expr":"up","execution":{"kind":"instant","to_ms":1767337200000}}`, "--until", "1h"},
			[]string{"--until", "execution.to_ms"}},
		{"inverted flags", []string{"--params", `{"expr":"up"}`, "--since", "2026-01-02T15:00:00+08:00", "--until", "2026-01-02T14:00:00+08:00"},
			[]string{"inverted window"}},
		{"equal bounds", []string{"--since", "2026-01-02T15:00:00+08:00", "--until", "2026-01-02T15:00:00+08:00"},
			[]string{"inverted window"}},
		{"until before params from_ms", []string{"--params", `{"execution":{"kind":"window","from_ms":1767337200000}}`, "--until", "2026-01-02T14:00:00+08:00"},
			[]string{"inverted window"}},
		{"bad since", []string{"--since", "yesterday-ish"}, []string{"invalid --since"}},
		{"execution not an object", []string{"--params", `{"execution":"range"}`, "--since", "1h"}, []string{"execution must be an object"}},
		{"empty since", []string{"--params", `{"expr":"up"}`, "--since", ""}, []string{"invalid --since: empty value"}},
		{"empty until", []string{"--params", `{"expr":"up"}`, "--until", " "}, []string{"invalid --until: empty value"}},
		{"array params with window", []string{"--params", `[1]`, "--since", "1h"}, []string{"--params must be a JSON object"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			_, err := execCommand(append([]string{"monit-query", "42", "--tool", "prometheus.query"}, tc.args...)...)
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q lacks %q", err, w)
				}
			}
			if stub.requests != 0 {
				t.Fatalf("request sent despite invalid window: %d", stub.requests)
			}
		})
	}
}

// Without --since/--until the params text is forwarded untouched: key order
// and number spelling survive.
func TestMonitQueryNoWindowFlagsForwardsParamsVerbatim(t *testing.T) {
	saveAndResetGlobals(t)
	requests := invokeRawStub(t)
	const params = `{"expr":"up","execution":{"to_ms":9007199254740993,"kind":"instant"},"ratio":1.50}`
	if _, err := execCommand("monit-query", "42", "--tool", "prometheus.query", "--params", params, "--json"); err != nil {
		t.Fatal(err)
	}
	if raw := <-requests; !strings.Contains(raw, `"params":`+params) {
		t.Fatalf("params not forwarded verbatim: %s", raw)
	}
}

func TestApplyToolWindowRejectsNonObjectParams(t *testing.T) {
	_, err := applyToolWindow(json.RawMessage(`[1]`), "1h", "", true, false)
	if err == nil || err.Error() != "--params must be a JSON object to apply --since/--until" {
		t.Fatalf("error = %v", err)
	}
}
