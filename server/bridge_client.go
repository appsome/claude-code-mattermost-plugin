package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin"
)

// BridgeClient handles HTTP communication with the Claude Code bridge server
type BridgeClient struct {
	baseURL    string
	api        plugin.API
	httpClient *http.Client
}

// Session represents a Claude Code session from the bridge server
type Session struct {
	ID                  string `json:"id"`
	ProjectPath         string `json:"projectPath"`
	MattermostUserID    string `json:"mattermostUserId"`
	MattermostChannelID string `json:"mattermostChannelId"`
	CLIPid              *int   `json:"cliPid"`
	Status              string `json:"status"`
	CreatedAt           int64  `json:"createdAt"`
	UpdatedAt           int64  `json:"updatedAt"`
}

// Message represents a message in the session
type Message struct {
	ID        int    `json:"id"`
	SessionID string `json:"sessionId"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// CreateSessionRequest is the request body for creating a session
type CreateSessionRequest struct {
	ProjectPath         string `json:"projectPath"`
	MattermostUserID    string `json:"mattermostUserId"`
	MattermostChannelID string `json:"mattermostChannelId"`
}

// SendMessageRequest is the request body for sending a message
type SendMessageRequest struct {
	Message string `json:"message"`
}

// NewBridgeClient creates a new bridge server HTTP client
func NewBridgeClient(baseURL string, api plugin.API) *BridgeClient {
	return &BridgeClient{
		baseURL: baseURL,
		api:     api,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateSession creates a new session on the bridge server
func (bc *BridgeClient) CreateSession(projectPath, userID, channelID string) (*Session, error) {
	reqBody := CreateSessionRequest{
		ProjectPath:         projectPath,
		MattermostUserID:    userID,
		MattermostChannelID: channelID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := bc.httpClient.Post(
		bc.baseURL+"/api/sessions",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Session Session `json:"session"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result.Session, nil
}

// SendMessage sends a message to a session
func (bc *BridgeClient) SendMessage(sessionID, message string) error {
	reqBody := SendMessageRequest{
		Message: message,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := bc.httpClient.Post(
		fmt.Sprintf("%s/api/sessions/%s/message", bc.baseURL, sessionID),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetMessages retrieves message history for a session
func (bc *BridgeClient) GetMessages(sessionID string, limit int) ([]Message, error) {
	url := fmt.Sprintf("%s/api/sessions/%s/messages", bc.baseURL, sessionID)
	if limit > 0 {
		url = fmt.Sprintf("%s?limit=%d", url, limit)
	}

	resp, err := bc.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Messages []Message `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Messages, nil
}

// GetSession retrieves session details
func (bc *BridgeClient) GetSession(sessionID string) (*Session, error) {
	resp, err := bc.httpClient.Get(fmt.Sprintf("%s/api/sessions/%s", bc.baseURL, sessionID))
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Session Session `json:"session"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result.Session, nil
}

// DeleteSession stops and deletes a session
func (bc *BridgeClient) DeleteSession(sessionID string) error {
	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/api/sessions/%s", bc.baseURL, sessionID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
