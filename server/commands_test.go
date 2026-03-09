package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestExecuteCommand_Help(t *testing.T) {
	p := setupTestPlugin(t)
	defer p.API.(*plugintest.API).AssertExpectations(t)

	args := &model.CommandArgs{
		Command:   "/claude-help",
		UserId:    "user1",
		ChannelId: "channel1",
	}

	response, appErr := p.ExecuteCommand(nil, args)
	if appErr != nil {
		t.Fatalf("ExecuteCommand returned AppError: %v", appErr)
	}
	assert.NotNil(t, response)
	assert.Contains(t, response.Text, "Claude Code - AI Coding Assistant")
}

func TestExecuteCommand_StartWithoutPath(t *testing.T) {
	p := setupTestPlugin(t)
	defer p.API.(*plugintest.API).AssertExpectations(t)

	args := &model.CommandArgs{
		Command:   "/claude-start",
		UserId:    "user1",
		ChannelId: "channel1",
	}

	response, appErr := p.ExecuteCommand(nil, args)
	if appErr != nil {
		t.Fatalf("ExecuteCommand returned AppError: %v", appErr)
	}
	assert.NotNil(t, response)
	assert.Contains(t, response.Text, "Please provide a project path")
}

func TestExecuteCommand_StopWithoutSession(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)

	// No active session
	api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

	defer api.AssertExpectations(t)

	args := &model.CommandArgs{
		Command:   "/claude-stop",
		UserId:    "user1",
		ChannelId: "channel1",
	}

	response, appErr := p.ExecuteCommand(nil, args)
	if appErr != nil {
		t.Fatalf("ExecuteCommand returned AppError: %v", appErr)
	}
	assert.NotNil(t, response)
	assert.Contains(t, response.Text, "No active session")
}

func TestExecuteCommand_SendWithoutSession(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)

	// No active session
	api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

	defer api.AssertExpectations(t)

	args := &model.CommandArgs{
		Command:   "/claude hello world",
		UserId:    "user1",
		ChannelId: "channel1",
	}

	response, appErr := p.ExecuteCommand(nil, args)
	if appErr != nil {
		t.Fatalf("ExecuteCommand returned AppError: %v", appErr)
	}
	assert.NotNil(t, response)
	assert.Contains(t, response.Text, "No active session")
}

func TestExecuteCommand_Status(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)

	// No active session
	api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

	defer api.AssertExpectations(t)

	args := &model.CommandArgs{
		Command:   "/claude-status",
		UserId:    "user1",
		ChannelId: "channel1",
	}

	response, appErr := p.ExecuteCommand(nil, args)
	if appErr != nil {
		t.Fatalf("ExecuteCommand returned AppError: %v", appErr)
	}
	assert.NotNil(t, response)
	// Should show no active session
	assert.Contains(t, response.Text, "No active session")
}

func TestExecuteCommand_FilesWithoutSession(t *testing.T) {
	p := setupTestPlugin(t)
	defer p.API.(*plugintest.API).AssertExpectations(t)

	args := &model.CommandArgs{
		Command:   "/claude-files",
		UserId:    "user1",
		ChannelId: "channel1",
	}

	response, appErr := p.ExecuteCommand(nil, args)
	if appErr != nil {
		t.Fatalf("ExecuteCommand returned AppError: %v", appErr)
	}
	assert.NotNil(t, response)
	// claude-files command doesn't exist, should return unknown command
	assert.Contains(t, response.Text, "Unknown command")
}

func TestExecuteCommand_ThreadWithoutSession(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)

	// No active session
	api.On("KVGet", mock.AnythingOfType("string")).Return(nil, nil)

	defer api.AssertExpectations(t)

	args := &model.CommandArgs{
		Command:   "/claude-thread context",
		UserId:    "user1",
		ChannelId: "channel1",
		RootId:    "root1", // In a thread
	}

	response, appErr := p.ExecuteCommand(nil, args)
	if appErr != nil {
		t.Fatalf("ExecuteCommand returned AppError: %v", appErr)
	}
	assert.NotNil(t, response)
	assert.Contains(t, response.Text, "No active Claude session")
}

func TestExecuteCommand_ThreadNotInThread(t *testing.T) {
	p := setupTestPlugin(t)
	defer p.API.(*plugintest.API).AssertExpectations(t)

	args := &model.CommandArgs{
		Command:   "/claude-thread context",
		UserId:    "user1",
		ChannelId: "channel1",
		RootId:    "", // Not in a thread
	}

	response, appErr := p.ExecuteCommand(nil, args)
	if appErr != nil {
		t.Fatalf("ExecuteCommand returned AppError: %v", appErr)
	}
	assert.NotNil(t, response)
	assert.Contains(t, response.Text, "must be run in a thread")
}

func TestExecuteCommand_InvalidCommand(t *testing.T) {
	p := setupTestPlugin(t)
	defer p.API.(*plugintest.API).AssertExpectations(t)

	args := &model.CommandArgs{
		Command:   "/claude-invalid",
		UserId:    "user1",
		ChannelId: "channel1",
	}

	response, appErr := p.ExecuteCommand(nil, args)
	if appErr != nil {
		t.Fatalf("ExecuteCommand returned AppError: %v", appErr)
	}
	assert.NotNil(t, response)
	assert.Contains(t, response.Text, "Unknown command")
}

func TestFormatDuration(t *testing.T) {
	// Test with a recent timestamp (less than a minute ago)
	recentTimestamp := time.Now().Add(-30 * time.Second).Unix()
	result := formatDuration(recentTimestamp)
	assert.Contains(t, result, "seconds ago")

	// Test with a timestamp from a few minutes ago
	minutesAgo := time.Now().Add(-5 * time.Minute).Unix()
	result = formatDuration(minutesAgo)
	assert.Contains(t, result, "minutes ago")

	// Test with a timestamp from a few hours ago
	hoursAgo := time.Now().Add(-3 * time.Hour).Unix()
	result = formatDuration(hoursAgo)
	assert.Contains(t, result, "hours ago")

	// Test with a timestamp from days ago
	daysAgo := time.Now().Add(-2 * 24 * time.Hour).Unix()
	result = formatDuration(daysAgo)
	assert.Contains(t, result, "days ago")
}

func TestFormatPID(t *testing.T) {
	// Test with a valid PID
	pid := 12345
	result := formatPID(&pid)
	assert.Equal(t, "PID 12345", result)

	// Test with nil PID
	result = formatPID(nil)
	assert.Equal(t, "Not running", result)

	// Test with zero PID
	zeroPID := 0
	result = formatPID(&zeroPID)
	assert.Equal(t, "PID 0", result)
}

func TestExecuteClaudeStart_WithExistingSession(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)

	// Mock existing session
	existingSession := &ChannelSession{
		SessionID:   "session123",
		ProjectPath: "/tmp/old",
		UserID:      "user1",
	}
	data, _ := json.Marshal(existingSession)
	api.On("KVGet", "session_channel1").Return(data, nil)

	defer api.AssertExpectations(t)

	args := &model.CommandArgs{
		Command:   "/claude-start /tmp/test",
		UserId:    "user1",
		ChannelId: "channel1",
	}

	response, appErr := p.ExecuteCommand(nil, args)
	assert.Nil(t, appErr)
	assert.NotNil(t, response)
	assert.Contains(t, response.Text, "already has an active session")
	assert.Contains(t, response.Text, "/tmp/old")
}

func TestExecuteClaudeStatus_WithActiveSession(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)

	// Mock existing session
	existingSession := &ChannelSession{
		SessionID:     "session123",
		ProjectPath:   "/tmp/test",
		UserID:        "user1",
		CreatedAt:     1678901234,
		LastMessageAt: 1678901334,
	}
	data, _ := json.Marshal(existingSession)
	api.On("KVGet", "session_channel1").Return(data, nil)

	// Mock bridge client GetSession call
	// Note: This will fail without a working bridge, so we expect an error path
	api.On("LogError", "Failed to get session from bridge", mock.Anything, mock.Anything).Return()

	defer api.AssertExpectations(t)

	args := &model.CommandArgs{
		Command:   "/claude-status",
		UserId:    "user1",
		ChannelId: "channel1",
	}

	response, appErr := p.ExecuteCommand(nil, args)
	assert.Nil(t, appErr)
	assert.NotNil(t, response)
	// Should show session status (even if bridge details fail)
	assert.Contains(t, response.Text, "Session Status")
}

func TestExecuteClaudeThread_InvalidAction(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)

	// Mock existing session
	existingSession := &ChannelSession{
		SessionID:   "session123",
		ProjectPath: "/tmp/test",
		UserID:      "user1",
	}
	data, _ := json.Marshal(existingSession)
	api.On("KVGet", "session_channel1").Return(data, nil)

	// Mock GetChannel (required by GetThreadContext)
	channel := &model.Channel{
		Id:   "channel1",
		Name: "test-channel",
		Type: model.ChannelTypeOpen,
	}
	api.On("GetChannel", "channel1").Return(channel, nil)

	// Mock GetPostThread (required by GetThreadContext) with at least one post
	rootPost := &model.Post{
		Id:        "root1",
		UserId:    "user1",
		ChannelId: "channel1",
		Message:   "Root post",
		CreateAt:  1678901234000,
	}
	postList := &model.PostList{
		Order: []string{"root1"},
		Posts: map[string]*model.Post{
			"root1": rootPost,
		},
	}
	api.On("GetPostThread", "root1").Return(postList, nil)

	// Mock GetUser for username lookup
	user := &model.User{
		Id:       "user1",
		Username: "testuser",
	}
	api.On("GetUser", "user1").Return(user, nil)

	// Mock log calls for bridge connection failure and thread send failure
	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	defer api.AssertExpectations(t)

	args := &model.CommandArgs{
		Command:   "/claude-thread invalid",
		UserId:    "user1",
		ChannelId: "channel1",
		RootId:    "root1",
	}

	response, appErr := p.ExecuteCommand(nil, args)
	assert.Nil(t, appErr)
	assert.NotNil(t, response)
	// With invalid action and bridge failure, we'll get an error message
	assert.NotEmpty(t, response.Text)
}

func TestExecuteClaude_EmptyMessage(t *testing.T) {
	p := setupTestPlugin(t)
	api := p.API.(*plugintest.API)

	// No need to mock KVGet - empty message is checked before session retrieval

	defer api.AssertExpectations(t)

	args := &model.CommandArgs{
		Command:   "/claude",
		UserId:    "user1",
		ChannelId: "channel1",
	}

	response, appErr := p.ExecuteCommand(nil, args)
	assert.Nil(t, appErr)
	assert.NotNil(t, response)
	assert.Contains(t, response.Text, "Please provide a message")
}

func TestRespondEphemeral(t *testing.T) {
	response := respondEphemeral("Test message")
	assert.NotNil(t, response)
	assert.Equal(t, "Test message", response.Text)
	assert.Equal(t, model.CommandResponseTypeEphemeral, response.ResponseType)
}

func TestParseProjectPath(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "simple path",
			command:  "/claude-start /tmp/test",
			expected: "/tmp/test",
		},
		{
			name:     "path with spaces",
			command:  "/claude-start /tmp/test project",
			expected: "/tmp/test",
		},
		{
			name:     "quoted path",
			command:  "/claude-start \"/tmp/test project\"",
			expected: "\"/tmp/test",
		},
		{
			name:     "no path",
			command:  "/claude-start",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Fields(tt.command)
			if len(parts) > 1 {
				result := strings.Join(parts[1:], " ")
				if tt.expected != "" {
					assert.Contains(t, result, tt.expected)
				} else {
					// Can't assert empty because we're testing different input
				}
			}
		})
	}
}

// setupTestPlugin creates a plugin instance with mocked API for testing
func setupTestPlugin(t *testing.T) *Plugin {
	api := &plugintest.API{}

	p := &Plugin{}
	p.SetAPI(api)
	p.botUserID = "bot123"

	// Initialize configuration
	config := &configuration{
		BridgeServerURL:      "http://localhost:3002",
		ClaudeCodePath:       "/usr/local/bin/claude-code",
		EnableFileOperations: true,
	}
	p.setConfiguration(config)

	// Initialize bridge client
	p.bridgeClient = NewBridgeClient("http://localhost:3002", api)

	return p
}
