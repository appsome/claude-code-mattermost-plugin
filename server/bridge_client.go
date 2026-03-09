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

// ContextRequest is the request body for sending context
type ContextRequest struct {
	Source   string           `json:"source"`
	ThreadID string           `json:"threadId,omitempty"`
	Content  string           `json:"content"`
	Action   string           `json:"action,omitempty"`
	Metadata *ContextMetadata `json:"metadata,omitempty"`
}

// ContextMetadata contains metadata about the context source
type ContextMetadata struct {
	ChannelName  string   `json:"channelName,omitempty"`
	RootPostID   string   `json:"rootPostId,omitempty"`
	MessageCount int      `json:"messageCount,omitempty"`
	Participants []string `json:"participants,omitempty"`
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

// SendContext sends context (e.g., thread history) to a session
func (bc *BridgeClient) SendContext(sessionID string, contextReq *ContextRequest) error {
	jsonData, err := json.Marshal(contextReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := bc.httpClient.Post(
		fmt.Sprintf("%s/api/sessions/%s/context", bc.baseURL, sessionID),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send context: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ApproveChange approves a code change
func (bc *BridgeClient) ApproveChange(sessionID, changeID string) error {
	reqBody := map[string]string{
		"changeId": changeID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := bc.httpClient.Post(
		fmt.Sprintf("%s/api/sessions/%s/approve", bc.baseURL, sessionID),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to approve change: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RejectChange rejects a code change
func (bc *BridgeClient) RejectChange(sessionID, changeID string) error {
	reqBody := map[string]string{
		"changeId": changeID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := bc.httpClient.Post(
		fmt.Sprintf("%s/api/sessions/%s/reject", bc.baseURL, sessionID),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to reject change: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ModifyChange requests modifications to a code change
func (bc *BridgeClient) ModifyChange(sessionID, changeID, instructions string) error {
	reqBody := map[string]string{
		"changeId":     changeID,
		"instructions": instructions,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := bc.httpClient.Post(
		fmt.Sprintf("%s/api/sessions/%s/modify", bc.baseURL, sessionID),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to modify change: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetFileContentByName retrieves the full content of a file from the session's project by filename
func (bc *BridgeClient) GetFileContentByName(sessionID, filename string) (string, error) {
	reqBody := map[string]string{
		"filename": filename,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := bc.httpClient.Post(
		fmt.Sprintf("%s/api/sessions/%s/file", bc.baseURL, sessionID),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Content, nil
}

// ListFiles retrieves the file tree for a session
func (bc *BridgeClient) ListFiles(sessionID string) ([]FileNode, error) {
	resp, err := bc.httpClient.Get(fmt.Sprintf("%s/api/sessions/%s/files", bc.baseURL, sessionID))
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Files []FileNode `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Files, nil
}

// GetFileContent retrieves the content of a file
func (bc *BridgeClient) GetFileContent(sessionID, filePath string) (string, error) {
	resp, err := bc.httpClient.Get(fmt.Sprintf("%s/api/sessions/%s/files/%s", bc.baseURL, sessionID, filePath))
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Content, nil
}

// CreateFile creates a new file in the project
func (bc *BridgeClient) CreateFile(sessionID, filePath, content string) error {
	reqBody := map[string]string{
		"path":    filePath,
		"content": content,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := bc.httpClient.Post(
		fmt.Sprintf("%s/api/sessions/%s/files", bc.baseURL, sessionID),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateFile updates the content of an existing file
func (bc *BridgeClient) UpdateFile(sessionID, filePath, content string) error {
	reqBody := map[string]string{
		"content": content,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/api/sessions/%s/files/%s", bc.baseURL, sessionID, filePath),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteFile deletes a file from the project
func (bc *BridgeClient) DeleteFile(sessionID, filePath string) error {
	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/api/sessions/%s/files/%s", bc.baseURL, sessionID, filePath),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bridge server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
