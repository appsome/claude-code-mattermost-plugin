package main

import (
	"testing"

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
