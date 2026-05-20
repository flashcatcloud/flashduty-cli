package cli

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type capturedIncidentRequest struct {
	path   string
	appKey string
	body   map[string]any
}

func newIncidentLifecycleTestClient(t *testing.T, capture *capturedIncidentRequest) *incidentAPIClient {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capture.path = r.URL.Path
		capture.appKey = r.URL.Query().Get("app_key")
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&capture.body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	t.Cleanup(ts.Close)

	return newIncidentAPIClient("test-key", ts.URL, "flashduty-cli/test")
}

func TestIncidentAPIClientSimpleLifecycleRequests(t *testing.T) {
	tests := []struct {
		name string
		call func(context.Context, *incidentAPIClient) error
		path string
	}{
		{
			name: "unack",
			call: func(ctx context.Context, c *incidentAPIClient) error {
				return c.UnackIncidents(ctx, []string{"inc-1", "inc-2"})
			},
			path: "/incident/unack",
		},
		{
			name: "wake",
			call: func(ctx context.Context, c *incidentAPIClient) error {
				return c.WakeIncidents(ctx, []string{"inc-1", "inc-2"})
			},
			path: "/incident/wake",
		},
		{
			name: "remove",
			call: func(ctx context.Context, c *incidentAPIClient) error {
				return c.RemoveIncidents(ctx, []string{"inc-1", "inc-2"})
			},
			path: "/incident/remove",
		},
		{
			name: "disable merge",
			call: func(ctx context.Context, c *incidentAPIClient) error {
				return c.DisableIncidentMerge(ctx, []string{"inc-1", "inc-2"})
			},
			path: "/incident/disable-merge",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			capture := &capturedIncidentRequest{}
			client := newIncidentLifecycleTestClient(t, capture)

			if err := tc.call(context.Background(), client); err != nil {
				t.Fatalf("call returned error: %v", err)
			}

			if capture.path != tc.path {
				t.Fatalf("expected path %q, got %q", tc.path, capture.path)
			}
			if capture.appKey != "test-key" {
				t.Fatalf("expected app_key test-key, got %q", capture.appKey)
			}
			gotIDs, ok := capture.body["incident_ids"].([]any)
			if !ok || len(gotIDs) != 2 || gotIDs[0] != "inc-1" || gotIDs[1] != "inc-2" {
				t.Fatalf("unexpected body: %#v", capture.body)
			}
		})
	}
}

func TestIncidentAPIClientCommentRequest(t *testing.T) {
	capture := &capturedIncidentRequest{}
	client := newIncidentLifecycleTestClient(t, capture)

	err := client.CommentIncidents(context.Background(), &IncidentCommentInput{
		IncidentIDs: []string{"inc-1"},
		Comment:     "rollback started",
		MuteReply:   true,
	})
	if err != nil {
		t.Fatalf("CommentIncidents returned error: %v", err)
	}

	if capture.path != "/incident/comment" {
		t.Fatalf("expected comment path, got %q", capture.path)
	}
	if capture.body["comment"] != "rollback started" || capture.body["mute_reply"] != true {
		t.Fatalf("unexpected body: %#v", capture.body)
	}
}

func TestIncidentAPIClientAddResponderRequest(t *testing.T) {
	capture := &capturedIncidentRequest{}
	client := newIncidentLifecycleTestClient(t, capture)

	err := client.AddIncidentResponders(context.Background(), &IncidentAddResponderInput{
		IncidentID: "inc-1",
		PersonIDs:  []int64{101, 202},
		Notify: &IncidentNotifyInput{
			FollowPreference: true,
			PersonalChannels: []string{"voice", "sms"},
			TemplateID:       "6321aad26c12104586a88916",
		},
	})
	if err != nil {
		t.Fatalf("AddIncidentResponders returned error: %v", err)
	}

	if capture.path != "/incident/responder/add" {
		t.Fatalf("expected responder add path, got %q", capture.path)
	}
	if capture.body["incident_id"] != "inc-1" {
		t.Fatalf("unexpected body: %#v", capture.body)
	}
	notify, ok := capture.body["notify"].(map[string]any)
	if !ok {
		t.Fatalf("expected notify body, got %#v", capture.body)
	}
	if notify["follow_preference"] != true || notify["template_id"] != "6321aad26c12104586a88916" {
		t.Fatalf("unexpected notify body: %#v", notify)
	}
}

func TestIncidentAPIClientWarRoomCreateRequest(t *testing.T) {
	capture := &capturedIncidentRequest{}
	client := newIncidentLifecycleTestClient(t, capture)

	warRoom, err := client.CreateIncidentWarRoom(context.Background(), &IncidentWarRoomCreateInput{
		IncidentID:    "inc-1",
		IntegrationID: 42,
		MemberIDs:     []int64{101, 202},
		AddObservers:  true,
	})
	if err != nil {
		t.Fatalf("CreateIncidentWarRoom returned error: %v", err)
	}

	if capture.path != "/incident/war-room/create" {
		t.Fatalf("expected war-room create path, got %q", capture.path)
	}
	if capture.body["incident_id"] != "inc-1" || capture.body["add_observers"] != true {
		t.Fatalf("unexpected body: %#v", capture.body)
	}
	if warRoom == nil {
		t.Fatal("expected war room output")
	}
}

func TestIncidentAPIClientWarRoomListDecodesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/incident/war-room/list" {
			t.Fatalf("expected list path, got %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"items": []map[string]any{
					{
						"integration_id": float64(42),
						"chat_id":        "chat-1",
						"incident_id":    "inc-1",
						"status":         "enabled",
						"plugin_type":    "feishu",
					},
				},
			},
		})
	}))
	t.Cleanup(ts.Close)

	client := newIncidentAPIClient("test-key", ts.URL, "flashduty-cli/test")
	result, err := client.ListIncidentWarRooms(context.Background(), &IncidentWarRoomListInput{IncidentID: "inc-1"})
	if err != nil {
		t.Fatalf("ListIncidentWarRooms returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ChatID != "chat-1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestIncidentAPIClientWarRoomDefaultObserversDecodesObservers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/incident/war-room/default-observers" {
			t.Fatalf("expected default observers path, got %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"observers": []map[string]any{
					{"person_id": float64(101), "person_name": "Alice", "email": "alice@example.com"},
				},
			},
		})
	}))
	t.Cleanup(ts.Close)

	client := newIncidentAPIClient("test-key", ts.URL, "flashduty-cli/test")
	observers, err := client.GetIncidentWarRoomDefaultObservers(context.Background(), "inc-1")
	if err != nil {
		t.Fatalf("GetIncidentWarRoomDefaultObservers returned error: %v", err)
	}
	if len(observers) != 1 || observers[0].DisplayName() != "Alice" {
		t.Fatalf("unexpected observers: %#v", observers)
	}
}

type failingRoundTripper struct{}

func (f failingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("dial tcp " + req.URL.String())
}

func TestIncidentAPIClientRedactsAppKeyOnTransportError(t *testing.T) {
	client := newIncidentAPIClient("secret-app-key", "https://api.flashcat.cloud", "flashduty-cli/test")
	client.httpClient = &http.Client{Transport: failingRoundTripper{}}

	err := client.UnackIncidents(context.Background(), []string{"inc-1"})
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	if strings.Contains(err.Error(), "secret-app-key") {
		t.Fatalf("transport error leaked app key: %v", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("expected redacted marker in error, got: %v", err)
	}
}
