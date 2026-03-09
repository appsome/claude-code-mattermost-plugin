package main

import (
	"encoding/json"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewOutputHandler(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)

	handler := NewOutputHandler(plugin)

	assert.NotNil(t, handler)
	assert.Equal(t, plugin, handler.plugin)
}

func TestOutputHandlerGetOrCreateBuffer(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	handler := NewOutputHandler(plugin)

	// First call should create new buffer
	buf1 := handler.getOrCreateBuffer("session1", "channel1")
	assert.NotNil(t, buf1)
	assert.Equal(t, "session1", buf1.sessionID)
	assert.Equal(t, "channel1", buf1.channelID)

	// Second call should return same buffer
	buf2 := handler.getOrCreateBuffer("session1", "channel1")
	assert.Equal(t, buf1, buf2)

	// Different session should create new buffer
	buf3 := handler.getOrCreateBuffer("session2", "channel2")
	assert.NotNil(t, buf3)
	assert.NotEqual(t, buf1, buf3)
	assert.Equal(t, "session2", buf3.sessionID)
	assert.Equal(t, "channel2", buf3.channelID)
}

func TestOutputHandlerHandleOutputInvalidJSON(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	plugin.botUserID = "bot123"
	handler := NewOutputHandler(plugin)

	// Mock CreatePost for raw text
	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == "channel1" &&
			post.UserId == "bot123" &&
			post.Message == "invalid json text"
	})).Return(&model.Post{}, nil)

	handler.HandleOutput("session1", "channel1", "invalid json text")

	api.AssertExpectations(t)
}

func TestOutputHandlerHandleOutputAssistantMessage(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	plugin.botUserID = "bot123"
	handler := NewOutputHandler(plugin)

	tests := []struct {
		name    string
		message CLIOutputMessage
		wantMsg string
	}{
		{
			name: "simple text message",
			message: CLIOutputMessage{
				Type:    "assistant",
				Message: "Hello, how can I help?",
			},
			wantMsg: "Hello, how can I help?",
		},
		{
			name: "content blocks",
			message: CLIOutputMessage{
				Type: "assistant",
				ContentBlocks: []ContentBlock{
					{Type: "text", Text: "First block"},
					{Type: "text", Text: "Second block"},
				},
			},
			wantMsg: "First block\nSecond block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.message)

			api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
				return post.ChannelId == "channel1" &&
					post.UserId == "bot123" &&
					post.Message == tt.wantMsg
			})).Return(&model.Post{}, nil).Once()

			handler.HandleOutput("session1", "channel1", string(data))

			api.AssertExpectations(t)
		})
	}
}

func TestOutputHandlerHandleOutputSystemMessage(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	plugin.botUserID = "bot123"
	handler := NewOutputHandler(plugin)

	message := CLIOutputMessage{
		Type:    "system",
		Message: "System is ready",
	}
	data, _ := json.Marshal(message)

	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == "channel1" &&
			post.UserId == "bot123" &&
			post.Message == "_System is ready_"
	})).Return(&model.Post{}, nil)

	handler.HandleOutput("session1", "channel1", string(data))

	api.AssertExpectations(t)
}

func TestOutputHandlerHandleOutputResultMessage(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	plugin.botUserID = "bot123"
	handler := NewOutputHandler(plugin)

	tests := []struct {
		name    string
		message CLIOutputMessage
		wantMsg string
	}{
		{
			name: "result only",
			message: CLIOutputMessage{
				Type:   "result",
				Result: "Task completed successfully",
			},
			wantMsg: "Task completed successfully",
		},
		{
			name: "result with cost",
			message: CLIOutputMessage{
				Type:      "result",
				Result:    "Done",
				TotalCost: "$0.50",
			},
			wantMsg: "Done\n\n_(Cost: $0.50)_",
		},
		{
			name: "result with usage",
			message: CLIOutputMessage{
				Type:   "result",
				Result: "Done",
				TotalUsage: &Usage{
					InputTokens:  100,
					OutputTokens: 50,
				},
			},
			wantMsg: "Done\n\n_(Tokens: 100 in / 50 out)_",
		},
		{
			name: "result with cost and usage",
			message: CLIOutputMessage{
				Type:      "result",
				Result:    "Done",
				TotalCost: "$0.50",
				TotalUsage: &Usage{
					InputTokens:  100,
					OutputTokens: 50,
				},
			},
			wantMsg: "Done\n\n_(Cost: $0.50, Tokens: 100 in / 50 out)_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.message)

			api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
				return post.ChannelId == "channel1" &&
					post.UserId == "bot123" &&
					post.Message == tt.wantMsg
			})).Return(&model.Post{}, nil).Once()

			handler.HandleOutput("session1", "channel1", string(data))

			api.AssertExpectations(t)
		})
	}
}

func TestOutputHandlerHandleOutputToolUseMessage(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	plugin.botUserID = "bot123"
	handler := NewOutputHandler(plugin)

	message := CLIOutputMessage{
		Type:     "tool_use",
		ToolName: "file_editor",
	}
	data, _ := json.Marshal(message)

	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == "channel1" &&
			post.UserId == "bot123" &&
			post.Message == ":wrench: Using tool: **file_editor**"
	})).Return(&model.Post{}, nil)

	handler.HandleOutput("session1", "channel1", string(data))

	api.AssertExpectations(t)
}

func TestOutputHandlerHandleOutputErrorMessage(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	plugin.botUserID = "bot123"
	handler := NewOutputHandler(plugin)

	message := CLIOutputMessage{
		Type:  "error",
		Error: "Something went wrong",
	}
	data, _ := json.Marshal(message)

	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == "channel1" &&
			post.UserId == "bot123" &&
			post.Message == ":warning: **Error**: Something went wrong"
	})).Return(&model.Post{}, nil)

	handler.HandleOutput("session1", "channel1", string(data))

	api.AssertExpectations(t)
}

func TestOutputHandlerHandleError(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	plugin.botUserID = "bot123"
	handler := NewOutputHandler(plugin)

	tests := []struct {
		name       string
		errorMsg   string
		shouldPost bool
	}{
		{
			name:       "normal error",
			errorMsg:   "File not found",
			shouldPost: true,
		},
		{
			name:       "empty error",
			errorMsg:   "",
			shouldPost: false,
		},
		{
			name:       "debugger message - should skip",
			errorMsg:   "Debugger listening on port 9229",
			shouldPost: false,
		},
		{
			name:       "debugger help message - should skip",
			errorMsg:   "For help, see: https://nodejs.org/en/docs/inspector",
			shouldPost: false,
		},
		{
			name:       "waiting for debugger - should skip",
			errorMsg:   "Waiting for the debugger to disconnect...",
			shouldPost: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPost {
				api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
					return post.ChannelId == "channel1" &&
						post.UserId == "bot123" &&
						post.Message == ":warning: **Error**: "+tt.errorMsg
				})).Return(&model.Post{}, nil).Once()
			}

			handler.HandleError("session1", "channel1", tt.errorMsg)

			if tt.shouldPost {
				api.AssertExpectations(t)
			}
		})
	}
}

func TestOutputHandlerHandleExit(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		wantMsg  string
	}{
		{
			name:     "successful exit",
			exitCode: 0,
			wantMsg:  ":white_check_mark: Claude Code session completed successfully.",
		},
		{
			name:     "error exit",
			exitCode: 1,
			wantMsg:  ":x: Claude Code session ended with exit code 1.",
		},
		{
			name:     "signal exit",
			exitCode: 130,
			wantMsg:  ":x: Claude Code session ended with exit code 130.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh API mock for each subtest
			api := &plugintest.API{}
			plugin := &Plugin{}
			plugin.SetAPI(api)
			plugin.botUserID = "bot123"
			handler := NewOutputHandler(plugin)

			// Setup mocks
			api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
				return post.ChannelId == "channel1" &&
					post.UserId == "bot123" &&
					post.Message == tt.wantMsg
			})).Return(&model.Post{}, nil).Once()

			// DeleteSession will be called - mock only KVDelete
			api.On("KVDelete", "session_channel1").Return(nil).Once()

			handler.HandleExit("session1", "channel1", tt.exitCode)

			api.AssertExpectations(t)
		})
	}
}

func TestOutputHandlerPostBotMessage(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	plugin.botUserID = "bot123"
	handler := NewOutputHandler(plugin)

	tests := []struct {
		name     string
		content  string
		shouldPost bool
	}{
		{
			name:     "normal message",
			content:  "Hello, world!",
			shouldPost: true,
		},
		{
			name:     "empty message",
			content:  "",
			shouldPost: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPost {
				api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
					return post.ChannelId == "channel1" &&
						post.UserId == "bot123" &&
						post.Message == tt.content
				})).Return(&model.Post{}, nil).Once()
			}

			handler.postBotMessage("channel1", tt.content)

			if tt.shouldPost {
				api.AssertExpectations(t)
			}
		})
	}
}

func TestOutputHandlerPostRawMessage(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	plugin.botUserID = "bot123"
	handler := NewOutputHandler(plugin)

	tests := []struct {
		name     string
		data     string
		wantMsg  string
		shouldPost bool
	}{
		{
			name:     "single line text",
			data:     "simple text",
			wantMsg:  "simple text",
			shouldPost: true,
		},
		{
			name:     "multi-line text",
			data:     "line 1\nline 2\nline 3",
			wantMsg:  "```\nline 1\nline 2\nline 3\n```",
			shouldPost: true,
		},
		{
			name:     "JSON-like text",
			data:     `{"key": "value"}`,
			wantMsg:  "```\n{\"key\": \"value\"}\n```",
			shouldPost: true,
		},
		{
			name:     "empty text",
			data:     "",
			wantMsg:  "",
			shouldPost: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPost {
				api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
					return post.ChannelId == "channel1" &&
						post.UserId == "bot123" &&
						post.Message == tt.wantMsg
				})).Return(&model.Post{}, nil).Once()
			}

			handler.postRawMessage("session1", "channel1", tt.data)

			if tt.shouldPost {
				api.AssertExpectations(t)
			}
		})
	}
}

func TestOutputHandlerHandleFileChange(t *testing.T) {
	api := &plugintest.API{}
	plugin := &Plugin{}
	plugin.SetAPI(api)
	plugin.botUserID = "bot123"
	handler := NewOutputHandler(plugin)

	tests := []struct {
		name       string
		message    CLIOutputMessage
		wantEmoji  string
		wantAction string
	}{
		{
			name: "create file",
			message: CLIOutputMessage{
				Type:       "tool_result",
				FilePath:   "test.go",
				ChangeType: "create",
			},
			wantEmoji:  ":new:",
			wantAction: "create",
		},
		{
			name: "modify file",
			message: CLIOutputMessage{
				Type:       "tool_result",
				FilePath:   "test.go",
				ChangeType: "modify",
			},
			wantEmoji:  ":pencil2:",
			wantAction: "modify",
		},
		{
			name: "edit file",
			message: CLIOutputMessage{
				Type:       "tool_result",
				FilePath:   "test.go",
				ChangeType: "edit",
			},
			wantEmoji:  ":pencil2:",
			wantAction: "edit",
		},
		{
			name: "delete file",
			message: CLIOutputMessage{
				Type:       "tool_result",
				FilePath:   "test.go",
				ChangeType: "delete",
			},
			wantEmoji:  ":wastebasket:",
			wantAction: "delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock GetConfig for getPluginURL only if interactive buttons are added
			// (not for delete operations)
			if tt.wantAction != "delete" {
				api.On("GetConfig").Return(&model.Config{
					ServiceSettings: model.ServiceSettings{
						SiteURL: model.NewString("http://localhost:8065"),
					},
				}).Once()
			}

			api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
				hasEmoji := post.Message[:len(tt.wantEmoji)] == tt.wantEmoji
				hasAction := containsString(post.Message, tt.wantAction)
				hasFilePath := containsString(post.Message, tt.message.FilePath)
				return post.ChannelId == "channel1" &&
					post.UserId == "bot123" &&
					hasEmoji && hasAction && hasFilePath
			})).Return(&model.Post{}, nil).Once()

			handler.handleFileChange("session1", "channel1", &tt.message)

			api.AssertExpectations(t)
		})
	}
}

func TestCLIOutputMessageStructure(t *testing.T) {
	// Test that CLIOutputMessage can be marshaled and unmarshaled
	msg := CLIOutputMessage{
		Type:      "assistant",
		Subtype:   "text",
		Message:   "Hello",
		SessionID: "session1",
		Timestamp: 1234567890,
		ContentBlocks: []ContentBlock{
			{Type: "text", Text: "Content"},
		},
		ToolName:   "tool1",
		ToolResult: "result",
		FilePath:   "/path/to/file",
		ChangeType: "modify",
		Result:     "success",
		TotalCost:  "$1.00",
		TotalUsage: &Usage{InputTokens: 100, OutputTokens: 50},
		Error:      "no error",
	}

	// Marshal
	data, err := json.Marshal(msg)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	// Unmarshal
	var msg2 CLIOutputMessage
	err = json.Unmarshal(data, &msg2)
	assert.NoError(t, err)
	assert.Equal(t, msg.Type, msg2.Type)
	assert.Equal(t, msg.Message, msg2.Message)
	assert.Equal(t, msg.SessionID, msg2.SessionID)
	assert.Equal(t, msg.ToolName, msg2.ToolName)
	assert.Equal(t, msg.FilePath, msg2.FilePath)
	assert.Equal(t, msg.Result, msg2.Result)
	assert.Equal(t, msg.TotalCost, msg2.TotalCost)
	assert.NotNil(t, msg2.TotalUsage)
	assert.Equal(t, msg.TotalUsage.InputTokens, msg2.TotalUsage.InputTokens)
	assert.Equal(t, msg.TotalUsage.OutputTokens, msg2.TotalUsage.OutputTokens)
}

func TestUsageStructure(t *testing.T) {
	usage := Usage{
		InputTokens:  150,
		OutputTokens: 75,
	}

	data, err := json.Marshal(usage)
	assert.NoError(t, err)

	var usage2 Usage
	err = json.Unmarshal(data, &usage2)
	assert.NoError(t, err)
	assert.Equal(t, usage.InputTokens, usage2.InputTokens)
	assert.Equal(t, usage.OutputTokens, usage2.OutputTokens)
}

func TestContentBlockStructure(t *testing.T) {
	block := ContentBlock{
		Type: "text",
		Text: "Sample text",
		Name: "block1",
	}

	data, err := json.Marshal(block)
	assert.NoError(t, err)

	var block2 ContentBlock
	err = json.Unmarshal(data, &block2)
	assert.NoError(t, err)
	assert.Equal(t, block.Type, block2.Type)
	assert.Equal(t, block.Text, block2.Text)
	assert.Equal(t, block.Name, block2.Name)
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
