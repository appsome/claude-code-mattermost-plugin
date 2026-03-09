package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

// ChannelSession represents an active Claude Code session for a channel
type ChannelSession struct {
	SessionID     string `json:"session_id"`
	ProjectPath   string `json:"project_path"`
	UserID        string `json:"user_id"`
	ChannelID     string `json:"channel_id"`
	CreatedAt     int64  `json:"created_at"`
	LastMessageAt int64  `json:"last_message_at"`
}

const (
	sessionKeyPrefix = "session_"
)

// getSessionKey returns the KV store key for a channel's session
func getSessionKey(channelID string) string {
	return sessionKeyPrefix + channelID
}

// GetActiveSession retrieves the active session for a channel
func (p *Plugin) GetActiveSession(channelID string) (*ChannelSession, error) {
	key := getSessionKey(channelID)
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		return nil, fmt.Errorf("failed to get session from KV store: %w", appErr)
	}

	if data == nil {
		return nil, nil // No active session
	}

	var session ChannelSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// SaveSession stores a session in the KV store
func (p *Plugin) SaveSession(channelID string, session *ChannelSession) error {
	key := getSessionKey(channelID)
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if appErr := p.API.KVSet(key, data); appErr != nil {
		return fmt.Errorf("failed to save session to KV store: %w", appErr)
	}

	return nil
}

// DeleteSession removes a session from the KV store
func (p *Plugin) DeleteSession(channelID string) error {
	key := getSessionKey(channelID)
	if appErr := p.API.KVDelete(key); appErr != nil {
		return fmt.Errorf("failed to delete session from KV store: %w", appErr)
	}

	return nil
}

// UpdateSessionLastMessage updates the last message timestamp
func (p *Plugin) UpdateSessionLastMessage(channelID string) error {
	session, err := p.GetActiveSession(channelID)
	if err != nil {
		return err
	}

	if session == nil {
		return fmt.Errorf("no active session for channel")
	}

	session.LastMessageAt = time.Now().Unix()
	return p.SaveSession(channelID, session)
}

// CreateSession creates a new session and spawns a CLI process
func (p *Plugin) CreateSession(channelID, projectPath, userID string) (*ChannelSession, error) {
	// Check if there's already an active session
	existing, err := p.GetActiveSession(channelID)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		return nil, fmt.Errorf("channel already has an active session")
	}

	// Generate a new session ID
	sessionID := model.NewId()

	// Spawn CLI process
	if err := p.processManager.Spawn(sessionID, projectPath, channelID, userID); err != nil {
		return nil, fmt.Errorf("failed to spawn CLI process: %w", err)
	}

	// Store session locally
	now := time.Now().Unix()
	session := &ChannelSession{
		SessionID:     sessionID,
		ProjectPath:   projectPath,
		UserID:        userID,
		ChannelID:     channelID,
		CreatedAt:     now,
		LastMessageAt: now,
	}

	if err := p.SaveSession(channelID, session); err != nil {
		// Clean up CLI process
		_ = p.processManager.Kill(sessionID)
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// StopSession stops a session and removes it from storage
func (p *Plugin) StopSession(channelID string) error {
	session, err := p.GetActiveSession(channelID)
	if err != nil {
		return err
	}

	if session == nil {
		return fmt.Errorf("no active session for channel")
	}

	// Kill the CLI process
	if err := p.processManager.Kill(session.SessionID); err != nil {
		p.API.LogWarn("Failed to kill CLI process", "error", err.Error())
		// Continue with local cleanup
	}

	// Remove from local storage
	if err := p.DeleteSession(channelID); err != nil {
		return fmt.Errorf("failed to delete local session: %w", err)
	}

	// Clean up message history
	if err := p.messageStore.DeleteSessionMessages(session.SessionID); err != nil {
		p.API.LogWarn("Failed to delete message history", "error", err.Error())
	}

	return nil
}

// GetSessionForChannel returns the session ID for a channel (helper for actions)
func (p *Plugin) GetSessionForChannel(channelID string) string {
	session, err := p.GetActiveSession(channelID)
	if err != nil || session == nil {
		return ""
	}
	return session.SessionID
}

// IsSessionActive checks if a session is active and has a running process
func (p *Plugin) IsSessionActive(channelID string) bool {
	session, err := p.GetActiveSession(channelID)
	if err != nil || session == nil {
		return false
	}
	return p.processManager.IsRunning(session.SessionID)
}
