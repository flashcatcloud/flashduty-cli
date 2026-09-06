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
	"github.com/toon-format/toon-go"
)

func TestDatasourceToolInvokeStdinPreservesJSON(t *testing.T) {
	saveAndResetGlobals(t)
	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/monit/datasource/tools/invoke" {
			t.Errorf("unexpected endpoint: %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		requests <- string(raw)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"request_id":"tools-test","data":{"datasource_id":42,"tool":"mongodb_mongod.command","data":{"counter":9007199254740993,"nested":[null,false,0]},"summary":"evidence"}}`)
	}))
	t.Cleanup(server.Close)
	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test", flashduty.WithBaseURL(server.URL))
	}
	stdinReader = strings.NewReader(`{"datasource_id":42,"tool":"mongodb_mongod.command","params":{"command":{"count":"events","query":{"counter":9007199254740993}},"database":"app"}}`)
	out, err := execCommand("monit", "datasource-tools-invoke", "--data", "-", "--output-format", "json")
	if err != nil {
		t.Fatal(err)
	}
	if raw := <-requests; !strings.Contains(raw, `"counter":9007199254740993`) || strings.Contains(raw, "9007199254740992") {
		t.Fatalf("request lost numeric precision: %s", raw)
	}
	var result struct {
		DatasourceID uint64          `json:"datasource_id"`
		Tool         string          `json:"tool"`
		Data         json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if result.DatasourceID != 42 || result.Tool != "mongodb_mongod.command" || !strings.Contains(string(result.Data), "9007199254740993") {
		t.Fatalf("response lost identity or evidence: %s", out)
	}
}

func TestDatasourceToolErrorsAreNotReplayed(t *testing.T) {
	for _, status := range []int{400, 429, 503, 504} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			saveAndResetGlobals(t)
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				io.WriteString(w, `{"request_id":"trace-tools","error":{"code":"ServiceUnavailable","reason":"edge_upgrade_required","message":"upgrade Edge to v0.71.0"}}`)
			}))
			t.Cleanup(server.Close)
			newClientFn = func() (*flashduty.Client, error) {
				return flashduty.NewClient("test", flashduty.WithBaseURL(server.URL))
			}
			_, err := execCommand("monit", "datasource-tools-invoke", "42", "--tool", "redis_node.overview", "--json")
			var apiErr *flashduty.ErrorResponse
			if !errors.As(err, &apiErr) || apiErr.Reason != "edge_upgrade_required" || calls.Load() != 1 {
				t.Fatalf("error lost or replayed: %v, calls=%d", err, calls.Load())
			}
			if !strings.Contains(err.Error(), "edge_upgrade_required") || !strings.Contains(err.Error(), "trace-tools") {
				t.Fatalf("CLI error omitted reason/request ID: %v", err)
			}
		})
	}
}

func TestDatasourceWritePreservesFalseFlagsAndOmission(t *testing.T) {
	for _, command := range []string{"datasource-create", "datasource-update"} {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/explicit=%v", command, explicit), func(t *testing.T) {
				saveAndResetGlobals(t)
				stub := newGFStub(t)
				args := []string{"monit", command, "--data", `{"id":42,"name":"cache","type_ident":"redis_node","address":"redis:6379","edge_cluster_name":"edge","payload":{"redis_node":{"database":0}}}`}
				if explicit {
					args = append(args, "--enabled=false", "--alerting-enabled=false")
				}
				if _, err := execCommand(args...); err != nil {
					t.Fatal(err)
				}
				for _, field := range []string{"enabled", "alerting_enabled"} {
					value, present := stub.lastBody[field]
					if present != explicit || (explicit && value != false) {
						t.Fatalf("%s changed presence/value: %+v", field, stub.lastBody)
					}
				}
			})
		}
	}
}

func TestDataBodyRejectsMultipleValues(t *testing.T) {
	for _, raw := range []string{`{} {}`, `{} garbage`, `null`} {
		if _, err := genAssembleBody(raw, func(map[string]any) error { return nil }); err == nil {
			t.Fatalf("invalid body accepted: %s", raw)
		}
	}
}

func TestDatasourceWriteRejectsExplicitNull(t *testing.T) {
	for _, command := range []string{"datasource-create", "datasource-update"} {
		for _, field := range []string{"enabled", "alerting_enabled"} {
			t.Run(command+"/"+field, func(t *testing.T) {
				saveAndResetGlobals(t)
				stub := newGFStub(t)
				body := fmt.Sprintf(`{"id":42,"name":"cache","type_ident":"redis_node","address":"redis:6379","edge_cluster_name":"edge","payload":{"redis_node":{}},%q:null}`, field)
				_, err := execCommand("monit", command, "--data", body)
				if err == nil || !strings.Contains(err.Error(), field+" must not be null") || stub.requests != 0 {
					t.Fatalf("null became omission: err=%v requests=%d", err, stub.requests)
				}
			})
		}
	}
}

func TestDatasourceToolTOONPreservesEvidence(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = json.RawMessage(`{"datasource_id":42,"tool":"redis_node.overview","data":{"counter":9007199254740993,"unsigned":18446744073709551615,"ratio":0.1234567890123456789,"rate":1.25,"nested":[null,false,{"ready":true}]}}`)
	out, err := execCommand("monit", "datasource-tools-invoke", "42", "--tool", "redis_node.overview", "--output-format", "toon")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := toon.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid TOON: %v: %s", err, out)
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("tool data is not an object: %s", out)
	}
	for key, want := range map[string]string{"counter": "9007199254740993", "unsigned": "18446744073709551615", "ratio": "0.1234567890123456789"} {
		if data[key] != want {
			t.Errorf("%s lost precision: %v; output=%s", key, data[key], out)
		}
	}
	nested, ok := data["nested"].([]any)
	if !ok || len(nested) != 3 || nested[0] != nil || nested[1] != false || nested[2].(map[string]any)["ready"] != true {
		t.Fatalf("nested evidence changed: %s", out)
	}
	if fmt.Sprint(data["rate"]) != "1.25" {
		t.Fatalf("ordinary rate changed: %s", out)
	}
}
