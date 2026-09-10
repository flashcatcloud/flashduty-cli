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
		{"array params", []string{"monit-query", "42", "--tool", "prometheus.query", "--params", "[1,2]"}, "--params"},
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
