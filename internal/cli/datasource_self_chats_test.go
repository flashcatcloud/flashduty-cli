package cli

import (
	"strings"
	"testing"
)

func TestDatasourceDingtalkSelfChatList(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"items": []map[string]any{
			{"name": "国际公司告警群", "chat_id": "cid_abc123", "origin": "user"},
		},
	}

	out, err := execCommand("datasource", "dingtalk-self-chat-list", "6100259006783", "--json")
	if err != nil {
		t.Fatalf("dingtalk-self-chat-list: %v", err)
	}
	if stub.lastPath != "/datasource/dingtalk/self-chat/list" {
		t.Fatalf("path = %q, want /datasource/dingtalk/self-chat/list", stub.lastPath)
	}
	if stub.lastBody["data_source_id"] != float64(6100259006783) {
		t.Errorf("data_source_id = %v, want 6100259006783", stub.lastBody["data_source_id"])
	}
	if !strings.Contains(out, "cid_abc123") {
		t.Errorf("output missing chat_id: %s", out)
	}
}

func TestDatasourceFeishuSelfChatListByFlag(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"items": []map[string]any{
			{"name": "SRE Ops", "chat_id": "oc_123", "origin": "user"},
		},
	}

	out, err := execCommand("datasource", "feishu-self-chat-list", "--data-source-id", "42", "--json")
	if err != nil {
		t.Fatalf("feishu-self-chat-list: %v", err)
	}
	if stub.lastPath != "/datasource/feishu/self-chat/list" {
		t.Fatalf("path = %q, want /datasource/feishu/self-chat/list", stub.lastPath)
	}
	if stub.lastBody["data_source_id"] != float64(42) {
		t.Errorf("data_source_id = %v, want 42", stub.lastBody["data_source_id"])
	}
	if !strings.Contains(out, "oc_123") {
		t.Errorf("output missing chat_id: %s", out)
	}
}

func TestDatasourceSlackSelfChatList(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"items": []map[string]any{
			{"id": "C0123", "name": "alerts", "is_private": true, "origin": "user"},
		},
	}

	out, err := execCommand("datasource", "slack-self-chat-list", "77", "--force-refresh", "--json")
	if err != nil {
		t.Fatalf("slack-self-chat-list: %v", err)
	}
	if stub.lastPath != "/datasource/slack/self-chat/list" {
		t.Fatalf("path = %q, want /datasource/slack/self-chat/list", stub.lastPath)
	}
	if stub.lastBody["data_source_id"] != float64(77) || stub.lastBody["force_refresh"] != true {
		t.Errorf("body = %v", stub.lastBody)
	}
	if !strings.Contains(out, "C0123") {
		t.Errorf("output missing id: %s", out)
	}
}

func TestDatasourceTeamsSelfChatList(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	stub.data = map[string]any{
		"items": []map[string]any{
			{
				"team_id":   "t-1",
				"team_name": "Ops",
				"channels": []map[string]any{
					{"channel_id": "19:ch1", "channel_name": "General", "origin": "user"},
				},
			},
		},
	}

	out, err := execCommand("datasource", "teams-self-chat-list", "88", "--team-id", "t-1", "--force-refresh", "--json")
	if err != nil {
		t.Fatalf("teams-self-chat-list: %v", err)
	}
	if stub.lastPath != "/datasource/teams/self-chat/list" {
		t.Fatalf("path = %q, want /datasource/teams/self-chat/list", stub.lastPath)
	}
	if stub.lastBody["data_source_id"] != float64(88) || stub.lastBody["team_id"] != "t-1" || stub.lastBody["force_refresh"] != true {
		t.Errorf("body = %v", stub.lastBody)
	}
	if !strings.Contains(out, "19:ch1") {
		t.Errorf("output missing channel_id: %s", out)
	}
}
