package main

import (
	"encoding/json"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewWebSocketClient(t *testing.T) {
	p := setupPlugin()
	baseURL := "http://localhost:3001"

	ws := NewWebSocketClient(baseURL, p)

	assert.NotNil(t, ws)
	assert.Equal(t, baseURL, ws.baseURL)
	assert.Equal(t, p, ws.plugin)
	assert.NotNil(t, ws.subscriptions)
	assert.NotNil(t, ws.stopChan)
	assert.Equal(t, 0, len(ws.subscriptions))
	assert.False(t, ws.reconnecting)
	assert.Nil(t, ws.conn)
}

func TestWebSocketClient_Subscribe(t *testing.T) {
	p := setupPlugin()
	ws := NewWebSocketClient("http://localhost:3001", p)

	sessionID := "session_123"
	channelID := "channel_456"

	ws.Subscribe(sessionID, channelID)

	ws.mu.RLock()
	defer ws.mu.RUnlock()

	assert.Equal(t, channelID, ws.subscriptions[sessionID])
	assert.Equal(t, 1, len(ws.subscriptions))
}

func TestWebSocketClient_Unsubscribe(t *testing.T) {
	p := setupPlugin()
	ws := NewWebSocketClient("http://localhost:3001", p)

	sessionID := "session_123"
	channelID := "channel_456"

	// Subscribe first
	ws.Subscribe(sessionID, channelID)
	assert.Equal(t, 1, len(ws.subscriptions))

	// Then unsubscribe
	ws.Unsubscribe(sessionID)

	ws.mu.RLock()
	defer ws.mu.RUnlock()

	assert.Equal(t, 0, len(ws.subscriptions))
}

func TestWebSocketClient_MultipleSubscriptions(t *testing.T) {
	p := setupPlugin()
	ws := NewWebSocketClient("http://localhost:3001", p)

	subscriptions := map[string]string{
		"session_1": "channel_1",
		"session_2": "channel_2",
		"session_3": "channel_3",
	}

	for sessionID, channelID := range subscriptions {
		ws.Subscribe(sessionID, channelID)
	}

	ws.mu.RLock()
	defer ws.mu.RUnlock()

	assert.Equal(t, 3, len(ws.subscriptions))
	for sessionID, channelID := range subscriptions {
		assert.Equal(t, channelID, ws.subscriptions[sessionID])
	}
}

func TestWebSocketClient_ProcessMessage_NotSubscribed(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	ws := NewWebSocketClient("http://localhost:3001", p)

	msg := &WebSocketMessage{
		Type:      "output",
		SessionID: "unsubscribed_session",
		Data:      json.RawMessage(`{"output": "test output"}`),
	}

	// Should not call any API methods for unsubscribed session
	ws.processMessage(msg)
}

func TestWebSocketClient_HandleOutput(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	ws := NewWebSocketClient("http://localhost:3001", p)

	channelID := "channel_123"
	sessionID := "session_123"
	ws.Subscribe(sessionID, channelID)

	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == channelID &&
			post.UserId == p.botUserID &&
			post.Message == "test output"
	})).Return(&model.Post{}, nil)

	msg := &WebSocketMessage{
		Type:      "output",
		SessionID: sessionID,
		Data:      json.RawMessage(`{"output": "test output"}`),
	}

	ws.processMessage(msg)
}

func TestWebSocketClient_HandleError(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	ws := NewWebSocketClient("http://localhost:3001", p)

	channelID := "channel_123"
	sessionID := "session_123"
	ws.Subscribe(sessionID, channelID)

	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == channelID &&
			post.UserId == p.botUserID &&
			post.Message == "⚠️ Error: test error"
	})).Return(&model.Post{}, nil)

	msg := &WebSocketMessage{
		Type:      "error",
		SessionID: sessionID,
		Data:      json.RawMessage(`{"error": "test error"}`),
	}

	ws.processMessage(msg)
}

func TestWebSocketClient_HandleStatus_Stopped(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	setupKVMocks(api)
	ws := NewWebSocketClient("http://localhost:3001", p)

	channelID := "channel_123"
	sessionID := "session_123"
	ws.Subscribe(sessionID, channelID)
	p.SaveSession(channelID, &ChannelSession{
		SessionID: sessionID,
		UserID:    "user_id",
	})

	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == channelID &&
			post.UserId == p.botUserID &&
			post.Message == "⏹️ Claude Code session stopped."
	})).Return(&model.Post{}, nil)

	// Mock session deletion
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Maybe().Return()

	msg := &WebSocketMessage{
		Type:      "status",
		SessionID: sessionID,
		Data:      json.RawMessage(`{"status": "stopped"}`),
	}

	ws.processMessage(msg)

	// Verify unsubscribed
	ws.mu.RLock()
	subscriptionCount := len(ws.subscriptions)
	ws.mu.RUnlock()
	assert.Equal(t, 0, subscriptionCount)
}

func TestWebSocketClient_HandleFileChange(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	ws := NewWebSocketClient("http://localhost:3001", p)

	channelID := "channel_123"
	sessionID := "session_123"
	ws.Subscribe(sessionID, channelID)

	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == channelID &&
			post.UserId == p.botUserID &&
			post.Message == "📝 File modified: `test.go`"
	})).Return(&model.Post{}, nil)

	msg := &WebSocketMessage{
		Type:      "file_change",
		SessionID: sessionID,
		Data:      json.RawMessage(`{"path": "test.go", "action": "modified"}`),
	}

	ws.processMessage(msg)
}

func TestWebSocketClient_HandleOutput_InvalidData(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	ws := NewWebSocketClient("http://localhost:3001", p)

	channelID := "channel_123"
	sessionID := "session_123"
	ws.Subscribe(sessionID, channelID)

	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return()

	msg := &WebSocketMessage{
		Type:      "output",
		SessionID: sessionID,
		Data:      json.RawMessage(`invalid json`),
	}

	ws.processMessage(msg)
}

func TestWebSocketClient_Close(t *testing.T) {
	p := setupPlugin()
	ws := NewWebSocketClient("http://localhost:3001", p)

	err := ws.Close()

	assert.NoError(t, err)
	assert.Nil(t, ws.conn)
}

func TestPostBotMessage_Success(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	channelID := "channel_123"
	message := "test message"

	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == channelID &&
			post.UserId == p.botUserID &&
			post.Message == message
	})).Return(&model.Post{}, nil)

	p.postBotMessage(channelID, message)
}

func TestPostBotMessage_Error(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)

	channelID := "channel_123"
	message := "test message"

	api.On("CreatePost", mock.Anything).Return(nil, model.NewAppError("CreatePost", "error", nil, "", 500))
	api.On("LogError", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	p.postBotMessage(channelID, message)
}

func TestWebSocketMessage_JSONSerialization(t *testing.T) {
	msg := &WebSocketMessage{
		Type:      "output",
		SessionID: "session_123",
		Data:      json.RawMessage(`{"output": "test"}`),
		Timestamp: 1234567890,
	}

	data, err := json.Marshal(msg)
	assert.NoError(t, err)

	var decoded WebSocketMessage
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.SessionID, decoded.SessionID)
	assert.Equal(t, msg.Timestamp, decoded.Timestamp)
}

func TestSubscribeMessage_JSONSerialization(t *testing.T) {
	msg := &SubscribeMessage{
		Type:      "subscribe",
		SessionID: "session_123",
	}

	data, err := json.Marshal(msg)
	assert.NoError(t, err)

	var decoded SubscribeMessage
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.SessionID, decoded.SessionID)
}

func TestWebSocketClient_HandleStatus_WithMessage(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	ws := NewWebSocketClient("http://localhost:3001", p)

	channelID := "channel_123"
	sessionID := "session_123"
	ws.Subscribe(sessionID, channelID)

	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == channelID &&
			post.UserId == p.botUserID &&
			post.Message == "Running tests..."
	})).Return(&model.Post{}, nil)

	msg := &WebSocketMessage{
		Type:      "status",
		SessionID: sessionID,
		Data:      json.RawMessage(`{"status": "running", "message": "Running tests..."}`),
	}

	ws.processMessage(msg)
}

func TestWebSocketClient_ProcessMessage_UnknownType(t *testing.T) {
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	p := setupPlugin()
	p.SetAPI(api)
	ws := NewWebSocketClient("http://localhost:3001", p)

	channelID := "channel_123"
	sessionID := "session_123"
	ws.Subscribe(sessionID, channelID)

	api.On("LogDebug", "Unknown message type", "type", "unknown_type").Return()

	msg := &WebSocketMessage{
		Type:      "unknown_type",
		SessionID: sessionID,
		Data:      json.RawMessage(`{}`),
	}

	ws.processMessage(msg)
}
