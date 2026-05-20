package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
)

const incidentAPIResponseLimit = 10 * 1024 * 1024

type flashdutyCLIClient struct {
	*flashduty.Client
	incident *incidentAPIClient
}

func (c *flashdutyCLIClient) UnackIncidents(ctx context.Context, incidentIDs []string) error {
	return c.incident.UnackIncidents(ctx, incidentIDs)
}

func (c *flashdutyCLIClient) WakeIncidents(ctx context.Context, incidentIDs []string) error {
	return c.incident.WakeIncidents(ctx, incidentIDs)
}

func (c *flashdutyCLIClient) RemoveIncidents(ctx context.Context, incidentIDs []string) error {
	return c.incident.RemoveIncidents(ctx, incidentIDs)
}

func (c *flashdutyCLIClient) DisableIncidentMerge(ctx context.Context, incidentIDs []string) error {
	return c.incident.DisableIncidentMerge(ctx, incidentIDs)
}

func (c *flashdutyCLIClient) CommentIncidents(ctx context.Context, input *IncidentCommentInput) error {
	return c.incident.CommentIncidents(ctx, input)
}

func (c *flashdutyCLIClient) AddIncidentResponders(ctx context.Context, input *IncidentAddResponderInput) error {
	return c.incident.AddIncidentResponders(ctx, input)
}

func (c *flashdutyCLIClient) CreateIncidentWarRoom(ctx context.Context, input *IncidentWarRoomCreateInput) (*IncidentWarRoom, error) {
	return c.incident.CreateIncidentWarRoom(ctx, input)
}

func (c *flashdutyCLIClient) ListIncidentWarRooms(ctx context.Context, input *IncidentWarRoomListInput) (*IncidentWarRoomListOutput, error) {
	return c.incident.ListIncidentWarRooms(ctx, input)
}

func (c *flashdutyCLIClient) GetIncidentWarRoom(ctx context.Context, input *IncidentWarRoomDetailInput) (*IncidentWarRoom, error) {
	return c.incident.GetIncidentWarRoom(ctx, input)
}

func (c *flashdutyCLIClient) DeleteIncidentWarRoom(ctx context.Context, input *IncidentWarRoomDeleteInput) error {
	return c.incident.DeleteIncidentWarRoom(ctx, input)
}

func (c *flashdutyCLIClient) AddIncidentWarRoomMembers(ctx context.Context, input *IncidentWarRoomAddMemberInput) error {
	return c.incident.AddIncidentWarRoomMembers(ctx, input)
}

func (c *flashdutyCLIClient) GetIncidentWarRoomDefaultObservers(ctx context.Context, incidentID string) ([]IncidentWarRoomObserver, error) {
	return c.incident.GetIncidentWarRoomDefaultObservers(ctx, incidentID)
}

type IncidentCommentInput struct {
	IncidentIDs []string
	Comment     string
	MuteReply   bool
}

type IncidentNotifyInput struct {
	FollowPreference bool
	PersonalChannels []string
	TemplateID       string
}

type IncidentAddResponderInput struct {
	IncidentID string
	PersonIDs  []int64
	Notify     *IncidentNotifyInput
}

type IncidentWarRoomCreateInput struct {
	IncidentID    string
	IntegrationID int64
	MemberIDs     []int64
	AddObservers  bool
}

type IncidentWarRoomListInput struct {
	IncidentID    string
	IntegrationID int64
}

type IncidentWarRoomDetailInput struct {
	IntegrationID int64
	ChatID        string
}

type IncidentWarRoomDeleteInput struct {
	IncidentID    string
	IntegrationID int64
}

type IncidentWarRoomAddMemberInput struct {
	IntegrationID int64
	ChatID        string
	MemberIDs     []int64
}

type IncidentWarRoom struct {
	ChatID    string `json:"chat_id"`
	ChatName  string `json:"chat_name"`
	ShareLink string `json:"share_link"`
}

type IncidentWarRoomItem struct {
	AccountID     int64  `json:"account_id"`
	IntegrationID int64  `json:"integration_id"`
	CreatedBy     int64  `json:"created_by"`
	ChatID        string `json:"chat_id"`
	IncidentID    string `json:"incident_id"`
	Status        string `json:"status"`
	CreatedAt     int64  `json:"created_at"`
	PluginType    string `json:"plugin_type"`
}

type IncidentWarRoomListOutput struct {
	Items []IncidentWarRoomItem `json:"items"`
}

type IncidentWarRoomObserver struct {
	PersonID   int64  `json:"person_id"`
	PersonName string `json:"person_name"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Status     string `json:"status"`
}

func (o IncidentWarRoomObserver) DisplayName() string {
	if o.PersonName != "" {
		return o.PersonName
	}
	return o.Name
}

type incidentAPIClient struct {
	appKey     string
	baseURL    *url.URL
	userAgent  string
	httpClient *http.Client
}

func newIncidentAPIClient(appKey, baseURL, userAgent string) *incidentAPIClient {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed == nil {
		parsed, _ = url.Parse("https://api.flashcat.cloud")
	}
	return &incidentAPIClient{
		appKey:    appKey,
		baseURL:   parsed,
		userAgent: userAgent,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *incidentAPIClient) UnackIncidents(ctx context.Context, incidentIDs []string) error {
	return c.postEmpty(ctx, "/incident/unack", map[string]any{"incident_ids": incidentIDs})
}

func (c *incidentAPIClient) WakeIncidents(ctx context.Context, incidentIDs []string) error {
	return c.postEmpty(ctx, "/incident/wake", map[string]any{"incident_ids": incidentIDs})
}

func (c *incidentAPIClient) RemoveIncidents(ctx context.Context, incidentIDs []string) error {
	return c.postEmpty(ctx, "/incident/remove", map[string]any{"incident_ids": incidentIDs})
}

func (c *incidentAPIClient) DisableIncidentMerge(ctx context.Context, incidentIDs []string) error {
	return c.postEmpty(ctx, "/incident/disable-merge", map[string]any{"incident_ids": incidentIDs})
}

func (c *incidentAPIClient) CommentIncidents(ctx context.Context, input *IncidentCommentInput) error {
	if input == nil {
		return fmt.Errorf("incident comment input is required")
	}
	body := map[string]any{
		"incident_ids": input.IncidentIDs,
		"comment":      input.Comment,
	}
	if input.MuteReply {
		body["mute_reply"] = true
	}
	return c.postEmpty(ctx, "/incident/comment", body)
}

func (c *incidentAPIClient) AddIncidentResponders(ctx context.Context, input *IncidentAddResponderInput) error {
	if input == nil {
		return fmt.Errorf("incident responder input is required")
	}
	body := map[string]any{
		"incident_id": input.IncidentID,
		"person_ids":  input.PersonIDs,
	}
	if input.Notify != nil {
		notify := map[string]any{}
		if input.Notify.FollowPreference {
			notify["follow_preference"] = true
		}
		if len(input.Notify.PersonalChannels) > 0 {
			notify["personal_channels"] = input.Notify.PersonalChannels
		}
		if input.Notify.TemplateID != "" {
			notify["template_id"] = input.Notify.TemplateID
		}
		if len(notify) > 0 {
			body["notify"] = notify
		}
	}
	return c.postEmpty(ctx, "/incident/responder/add", body)
}

func (c *incidentAPIClient) CreateIncidentWarRoom(ctx context.Context, input *IncidentWarRoomCreateInput) (*IncidentWarRoom, error) {
	if input == nil {
		return nil, fmt.Errorf("incident war-room create input is required")
	}
	body := map[string]any{
		"incident_id":    input.IncidentID,
		"integration_id": input.IntegrationID,
	}
	if len(input.MemberIDs) > 0 {
		body["member_ids"] = input.MemberIDs
	}
	if input.AddObservers {
		body["add_observers"] = true
	}
	var out IncidentWarRoom
	if err := c.postData(ctx, "/incident/war-room/create", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *incidentAPIClient) ListIncidentWarRooms(ctx context.Context, input *IncidentWarRoomListInput) (*IncidentWarRoomListOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("incident war-room list input is required")
	}
	body := map[string]any{"incident_id": input.IncidentID}
	if input.IntegrationID > 0 {
		body["integration_id"] = input.IntegrationID
	}
	var out IncidentWarRoomListOutput
	if err := c.postData(ctx, "/incident/war-room/list", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *incidentAPIClient) GetIncidentWarRoom(ctx context.Context, input *IncidentWarRoomDetailInput) (*IncidentWarRoom, error) {
	if input == nil {
		return nil, fmt.Errorf("incident war-room detail input is required")
	}
	var out IncidentWarRoom
	if err := c.postData(ctx, "/incident/war-room/detail", map[string]any{
		"integration_id": input.IntegrationID,
		"chat_id":        input.ChatID,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *incidentAPIClient) DeleteIncidentWarRoom(ctx context.Context, input *IncidentWarRoomDeleteInput) error {
	if input == nil {
		return fmt.Errorf("incident war-room delete input is required")
	}
	return c.postEmpty(ctx, "/incident/war-room/delete", map[string]any{
		"incident_id":    input.IncidentID,
		"integration_id": input.IntegrationID,
	})
}

func (c *incidentAPIClient) AddIncidentWarRoomMembers(ctx context.Context, input *IncidentWarRoomAddMemberInput) error {
	if input == nil {
		return fmt.Errorf("incident war-room add-member input is required")
	}
	return c.postEmpty(ctx, "/incident/war-room/add-member", map[string]any{
		"integration_id": input.IntegrationID,
		"chat_id":        input.ChatID,
		"member_ids":     input.MemberIDs,
	})
}

func (c *incidentAPIClient) GetIncidentWarRoomDefaultObservers(ctx context.Context, incidentID string) ([]IncidentWarRoomObserver, error) {
	var out struct {
		Observers []IncidentWarRoomObserver `json:"observers"`
	}
	if err := c.postData(ctx, "/incident/war-room/default-observers", map[string]any{
		"incident_id": incidentID,
	}, &out); err != nil {
		return nil, err
	}
	return out.Observers, nil
}

type incidentAPIEnvelope struct {
	Error *flashduty.DutyError `json:"error,omitempty"`
	Data  json.RawMessage      `json:"data,omitempty"`
}

func (c *incidentAPIClient) postEmpty(ctx context.Context, path string, body any) error {
	resp, err := c.post(ctx, path, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	limited := io.LimitReader(resp.Body, incidentAPIResponseLimit)
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(limited)
		return fmt.Errorf("API request failed (HTTP %d): %s", resp.StatusCode, c.redactAppKey(string(data)))
	}

	var result flashduty.FlashdutyResponse
	if err := json.NewDecoder(limited).Decode(&result); err != nil {
		return fmt.Errorf("invalid API response: %w", err)
	}
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (c *incidentAPIClient) postData(ctx context.Context, path string, body any, out any) error {
	resp, err := c.post(ctx, path, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	limited := io.LimitReader(resp.Body, incidentAPIResponseLimit)
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(limited)
		return fmt.Errorf("API request failed (HTTP %d): %s", resp.StatusCode, c.redactAppKey(string(data)))
	}

	var envelope incidentAPIEnvelope
	if err := json.NewDecoder(limited).Decode(&envelope); err != nil {
		return fmt.Errorf("invalid API response: %w", err)
	}
	if envelope.Error != nil {
		return envelope.Error
	}
	if out == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("invalid API data: %w", err)
	}
	return nil
}

func (c *incidentAPIClient) post(ctx context.Context, path string, body any) (*http.Response, error) {
	reqBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}

	parsedPath, err := url.Parse(strings.TrimPrefix(path, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid request path: %w", err)
	}
	fullURL := c.baseURL.ResolveReference(parsedPath)
	query := fullURL.Query()
	query.Set("app_key", c.appKey)
	fullURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL.String(), bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %s", c.redactAppKey(err.Error()))
	}
	return resp, nil
}

func (c *incidentAPIClient) redactAppKey(s string) string {
	if c.appKey == "" || s == "" {
		return s
	}
	redacted := strings.ReplaceAll(s, c.appKey, "[REDACTED]")
	escaped := url.QueryEscape(c.appKey)
	if escaped != c.appKey {
		redacted = strings.ReplaceAll(redacted, escaped, "[REDACTED]")
	}
	return redacted
}
