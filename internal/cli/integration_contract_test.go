package cli

import (
	"encoding/json"
	"reflect"
	"testing"
)

func integrationContractFixture() map[string]any {
	return map[string]any{
		"integration_id": 42, "data_source_id": 42, "account_id": 100, "exclusive_data_source_id": 0,
		"name": "Contract fixture", "description": "", "team_id": 7,
		"category": "im", "plugin_type": "slack_app", "plugin_type_name": "Slack", "plugin_id": 9,
		"ref_id": "a_100", "status": "enabled", "created_at": 1700000000, "updated_at": 1700000001, "last_time": 0,
		"creator_id": 11, "updated_by": 12, "no_editable": false, "editable": true,
		"settings":        map[string]any{"headers": map[string]any{"Authorization": "fixture-not-a-real-secret"}},
		"integration_key": "fixture-not-a-real-key",
	}
}

func TestIntegrationListPreservesConfigurationAndPermissions(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	fixture := integrationContractFixture()
	stub.data = map[string]any{"p": 1, "limit": 100, "total": 1, "items": []any{fixture}}
	out, err := execCommand("integration", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	items, ok := result["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected list output: %s", out)
	}
	item := items[0].(map[string]any)
	for field := range fixture {
		if _, ok := item[field]; !ok {
			t.Errorf("list dropped backend field %s", field)
		}
	}
	for _, field := range []string{"settings", "integration_key", "editable", "no_editable"} {
		if !reflect.DeepEqual(item[field], fixture[field]) {
			t.Errorf("%s changed: got %#v, want %#v", field, item[field], fixture[field])
		}
	}
	if stub.lastPath != "/integration/list" {
		t.Fatalf("request path = %s", stub.lastPath)
	}
}

func TestIntegrationInfoPreservesConfigurationAndTestResult(t *testing.T) {
	for _, withTest := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent test", true: "failed test"}[withTest], func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			fixture := integrationContractFixture()
			for _, field := range []string{"account_id", "data_source_id", "exclusive_data_source_id"} {
				delete(fixture, field)
			}
			if withTest {
				fixture["test"] = map[string]any{"success": false, "message": "fixture plugin test failed"}
			}
			stub.data = fixture
			out, err := execCommand("integration", "info", "42", "--json")
			if err != nil {
				t.Fatal(err)
			}
			var result map[string]any
			if err := json.Unmarshal([]byte(out), &result); err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"settings", "integration_key", "plugin_id", "editable", "creator_id", "updated_by", "no_editable"} {
				if _, ok := result[field]; !ok {
					t.Errorf("info dropped backend field %s", field)
				}
			}
			if !reflect.DeepEqual(result["settings"], fixture["settings"]) || result["integration_key"] != fixture["integration_key"] {
				t.Errorf("info changed configuration or key: %s", out)
			}
			if !reflect.DeepEqual(result["test"], fixture["test"]) {
				t.Errorf("test result changed: %s", out)
			}
			if !withTest {
				if _, ok := result["test"]; ok {
					t.Errorf("absent test was fabricated: %s", out)
				}
			}
		})
	}
}

func TestIntegrationCreatePreservesKeyAndPluginTest(t *testing.T) {
	for _, mode := range []string{"data", "flags"} {
		t.Run(mode, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			stub.data = map[string]any{"integration_id": 42, "integration_key": "fixture-key", "test": map[string]any{"success": false, "message": "fixture plugin test failed"}}
			args := []string{"integration", "create", "--json", "--data", `{"plugin_type":"standard.alert","integration_key":"00000000000000000000000000000000100","is_test":true}`}
			if mode == "flags" {
				args = []string{"integration", "create", "--json", "--plugin-type", "standard.alert", "--integration-key", "00000000000000000000000000000000100", "--is-test"}
			}
			out, err := execCommand(args...)
			if err != nil {
				t.Fatal(err)
			}
			if stub.lastBody["integration_key"] != "00000000000000000000000000000000100" || stub.lastBody["is_test"] != true {
				t.Errorf("create dropped key or test flag: %#v", stub.lastBody)
			}
			var result map[string]any
			if err := json.Unmarshal([]byte(out), &result); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result["test"], stub.data.(map[string]any)["test"]) {
				t.Errorf("create lost failed plugin test: %s", out)
			}
		})
	}
}

func TestIntegrationUpdatePreservesTestAndExplicitEmptyHeaders(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	_, err := execCommand("integration", "update", "42", "--team-id", "0", "--is-test", "--data", `{"settings":{"headers":{}}}`)
	if err != nil {
		t.Fatal(err)
	}
	if stub.lastBody["is_test"] != true || stub.lastBody["team_id"] != float64(0) {
		t.Errorf("update lost test flag or explicit team clear: %#v", stub.lastBody)
	}
	want := map[string]any{"headers": map[string]any{}}
	if !reflect.DeepEqual(stub.lastBody["settings"], want) {
		t.Errorf("update changed explicit empty headers: %#v", stub.lastBody)
	}
}
