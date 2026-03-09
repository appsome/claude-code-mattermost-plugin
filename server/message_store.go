package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin"
)

// MessageStore handles message persistence using Mattermost KV store
type MessageStore struct {
	api plugin.API
}

// StoredMessage represents a message stored in the KV store
type StoredMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"` // "user", "assistant", "system"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// SessionMessages holds all messages for a session
type SessionMessages struct {
	SessionID string          `json:"session_id"`
	Messages  []StoredMessage `json:"messages"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// NewMessageStore creates a new MessageStore
func NewMessageStore(api plugin.API) *MessageStore {
	return &MessageStore{
		api: api,
	}
}

// kvKey returns the KV store key for a session's messages
func (ms *MessageStore) kvKey(sessionID string) string {
	return fmt.Sprintf("messages_%s", sessionID)
}

// GetMessages retrieves all messages for a session
func (ms *MessageStore) GetMessages(sessionID string) ([]StoredMessage, error) {
	data, appErr := ms.api.KVGet(ms.kvKey(sessionID))
	if appErr != nil {
		return nil, fmt.Errorf("failed to get messages: %s", appErr.Error())
	}

	if data == nil {
		return []StoredMessage{}, nil
	}

	var sessionMessages SessionMessages
	if err := json.Unmarshal(data, &sessionMessages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return sessionMessages.Messages, nil
}

// AddMessage adds a new message to a session
func (ms *MessageStore) AddMessage(sessionID, role, content string) (*StoredMessage, error) {
	messages, err := ms.GetMessages(sessionID)
	if err != nil {
		// If error, start with empty messages
		messages = []StoredMessage{}
	}

	msg := StoredMessage{
		ID:        fmt.Sprintf("%s_%d", sessionID, len(messages)),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	messages = append(messages, msg)

	sessionMessages := SessionMessages{
		SessionID: sessionID,
		Messages:  messages,
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(sessionMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal messages: %w", err)
	}

	if appErr := ms.api.KVSet(ms.kvKey(sessionID), data); appErr != nil {
		return nil, fmt.Errorf("failed to save messages: %s", appErr.Error())
	}

	return &msg, nil
}

// DeleteSessionMessages removes all messages for a session
func (ms *MessageStore) DeleteSessionMessages(sessionID string) error {
	if appErr := ms.api.KVDelete(ms.kvKey(sessionID)); appErr != nil {
		return fmt.Errorf("failed to delete messages: %s", appErr.Error())
	}
	return nil
}

// GetMessageCount returns the number of messages for a session
func (ms *MessageStore) GetMessageCount(sessionID string) (int, error) {
	messages, err := ms.GetMessages(sessionID)
	if err != nil {
		return 0, err
	}
	return len(messages), nil
}

// GetLastMessage returns the last message for a session
func (ms *MessageStore) GetLastMessage(sessionID string) (*StoredMessage, error) {
	messages, err := ms.GetMessages(sessionID)
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return nil, nil
	}

	return &messages[len(messages)-1], nil
}

// GetMessagesByRole returns all messages with a specific role
func (ms *MessageStore) GetMessagesByRole(sessionID, role string) ([]StoredMessage, error) {
	messages, err := ms.GetMessages(sessionID)
	if err != nil {
		return nil, err
	}

	var filtered []StoredMessage
	for _, msg := range messages {
		if msg.Role == role {
			filtered = append(filtered, msg)
		}
	}

	return filtered, nil
}
