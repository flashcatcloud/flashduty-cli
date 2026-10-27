package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/flashcatcloud/go-flashduty"
)

func TestDedicatedIntegrationCommands(t *testing.T) {
	for _, tc := range []struct {
		verb string
		args []string
		path string
		body string
	}{
		{"list", []string{"--channel-id", "20001", "--page", "2", "--limit", "10", "--plugin-type", "standard.alert", "--status", "enabled"}, "list", `{"channel_id":20001,"p":2,"limit":10,"plugin_type":"standard.alert","status":"enabled"}`},
		{"info", []string{"10001"}, "info", `{"integration_id":10001}`},
		{"create", []string{"--channel-id", "20001", "--plugin-type", "standard.alert", "--name", "Primary", "--integration-key", "0123456789abcdef0123456789abcdef100", "--data", `{"settings":{"headers":{}}}`}, "create", `{"channel_id":20001,"plugin_type":"standard.alert","name":"Primary","integration_key":"0123456789abcdef0123456789abcdef100","settings":{"headers":{}}}`},
		{"update", []string{"10001", "--description", ""}, "update", `{"integration_id":10001,"description":""}`},
		{"delete", []string{"10001"}, "delete", `{"integration_id":10001}`},
		{"enable", []string{"10001"}, "enable", `{"integration_id":10001}`},
		{"disable", []string{"10001"}, "disable", `{"integration_id":10001}`},
		{"key-rotate", []string{"10001"}, "key/rotate", `{"integration_id":10001}`},
	} {
		t.Run(tc.verb, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			args := append([]string{"channel", "integration-" + tc.verb, "--json"}, tc.args...)
			_, err := execCommand(args...)
			if err != nil {
				t.Fatal(err)
			}
			if stub.lastPath != "/channel/integration/"+tc.path || stub.requests != 1 {
				t.Fatalf("path = %s, requests = %d", stub.lastPath, stub.requests)
			}
			var want map[string]any
			err = json.Unmarshal([]byte(tc.body), &want)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(stub.lastBody, want) {
				t.Fatalf("body = %#v, want %#v", stub.lastBody, want)
			}
		})
	}
}

func TestDedicatedIntegrationListPreservesOwnershipAndKey(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{"items": []any{map[string]any{"integration_id": 10001, "channel_id": 20001, "name": "Primary", "integration_key": "fixture-key", "settings": map[string]any{"password": "******"}}}, "p": 1, "limit": 100, "total": 1}
	out, err := execCommand("channel", "integration-list", "--channel-id", "20001", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Items []struct {
			IntegrationID  int64          `json:"integration_id"`
			ChannelID      int64          `json:"channel_id"`
			IntegrationKey string         `json:"integration_key"`
			Settings       map[string]any `json:"settings"`
		} `json:"items"`
		Total int `json:"total"`
	}
	err = json.Unmarshal([]byte(out), &got)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 1 || len(got.Items) != 1 {
		t.Fatalf("unexpected page: %s", out)
	}
	item := got.Items[0]
	if item.IntegrationID != 10001 || item.ChannelID != 20001 || item.IntegrationKey != "fixture-key" || item.Settings["password"] != "******" {
		t.Fatalf("response fields changed: %s", out)
	}
}

func TestDedicatedIntegrationUpdatePreservesEmptyAndOmittedSettings(t *testing.T) {
	for _, tc := range []struct {
		body    string
		present bool
	}{
		{`{"integration_id":10001}`, false},
		{`{"integration_id":10001,"settings":null}`, false},
		{`{"integration_id":10001,"settings":{}}`, true},
	} {
		t.Run(tc.body, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			_, err := execCommand("channel", "integration-update", "--data", tc.body, "--json")
			if err != nil {
				t.Fatal(err)
			}
			settings, present := stub.lastBody["settings"]
			if present != tc.present {
				t.Fatalf("settings presence = %v, want %v: %#v", present, tc.present, stub.lastBody)
			}
			if present && !reflect.DeepEqual(settings, map[string]any{}) {
				t.Fatalf("empty settings changed: %#v", settings)
			}
		})
	}
}

func TestDedicatedIntegrationListRequiresChannel(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	_, err := execCommand("channel", "integration-list", "--json")
	if err == nil || stub.requests != 0 {
		t.Fatalf("error = %v, requests = %d", err, stub.requests)
	}
}

func TestDedicatedIntegrationForbiddenDoesNotCallPublicAPI(t *testing.T) {
	saveAndResetGlobals(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/channel/integration/info" {
			t.Errorf("unexpected fallback: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":"AccessDenied","message":"Channel access denied"}}`))
	}))
	t.Cleanup(server.Close)
	newClientFn = func() (*flashduty.Client, error) {
		return flashduty.NewClient("test-key", flashduty.WithBaseURL(server.URL))
	}
	_, err := execCommand("channel", "integration-info", "10001", "--json")
	if flashduty.ErrorCodeOf(err) != flashduty.ErrorCodeAccessDenied || calls != 1 {
		t.Fatalf("error = %v, calls = %d", err, calls)
	}
}
