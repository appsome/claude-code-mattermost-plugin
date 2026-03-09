package main

import (
	"github.com/mattermost/mattermost/server/public/plugin"
)

// BridgeClient handles HTTP communication with the Claude Code bridge server
type BridgeClient struct {
	baseURL string
	api     plugin.API
}

// NewBridgeClient creates a new bridge server HTTP client
func NewBridgeClient(baseURL string, api plugin.API) *BridgeClient {
	return &BridgeClient{
		baseURL: baseURL,
		api:     api,
	}
}

// TODO: Methods will be implemented in Issue #3 (Bridge Server)
// - CreateSession(projectPath, userID, channelID) (Session, error)
// - SendMessage(sessionID, message string) error
// - GetMessages(sessionID) ([]Message, error)
// - StopSession(sessionID) error
// - ListFiles(sessionID) ([]FileNode, error)
// - GetFile(sessionID, path) (string, error)
// - UpdateFile(sessionID, path, content string) error
